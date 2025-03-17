package replication

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
)

// LeaderNodeImpl 实现了 LeaderNode 接口
type LeaderNodeImpl struct {
	// 节点标识和角色
	nodeID string
	role   ReplicationRole

	// 本地缓存实例
	cache cache.Cache

	// 复制日志，用于记录和同步操作
	log ReplicationLog

	// 当前领导者ID
	leaderID string

	// 配置选项
	config *ReplicationConfig

	// 数据同步器实例
	syncer Syncer

	// 从节点信息映射(nodeID -> address)
	followers      map[string]string
	followerStatus map[string]ReplicationState

	// 互斥锁保护并发访问
	mu sync.RWMutex

	// 心跳定时器
	heartbeatTimer *time.Timer

	// 停止信号通道
	stopCh chan struct{}

	// 运行状态
	running bool

	// 当前任期
	currentTerm uint64
}

// NewLeaderNode 创建一个新的主节点
func NewLeaderNode(nodeID string, cache cache.Cache, log ReplicationLog, syncer Syncer, options ...ReplicationOption) *LeaderNodeImpl {
	// 使用默认配置
	config := DefaultReplicationConfig()

	// 应用选项
	for _, opt := range options {
		opt(config)
	}

	// 创建节点
	node := &LeaderNodeImpl{
		nodeID:         nodeID,
		role:           RoleLeader,
		cache:          cache,
		log:            log,
		config:         config,
		syncer:         syncer,
		followers:      make(map[string]string),
		followerStatus: make(map[string]ReplicationState),
		stopCh:         make(chan struct{}),
		leaderID:       nodeID, // 作为Leader，leaderID是自己的ID
		currentTerm:    1,      // 初始任期为1
	}

	return node
}

// Start 启动Leader节点服务
func (l *LeaderNodeImpl) Start() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.running {
		return nil
	}

	l.running = true

	// 启动数据同步器
	if err := l.syncer.Start(); err != nil {
		l.running = false
		return fmt.Errorf("failed to start syncer: %w", err)
	}

	// 启动心跳定时任务
	l.startHeartbeat()

	return nil
}

// Stop 停止Leader节点服务
func (l *LeaderNodeImpl) Stop() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.running {
		return nil
	}

	// 停止心跳
	if l.heartbeatTimer != nil {
		l.heartbeatTimer.Stop()
	}

	// 发送停止信号
	close(l.stopCh)

	// 停止同步器
	if err := l.syncer.Stop(); err != nil {
		return fmt.Errorf("failed to stop syncer: %w", err)
	}

	l.running = false
	return nil
}

// GetRole 获取节点角色
func (l *LeaderNodeImpl) GetRole() ReplicationRole {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.role
}

// SetRole 设置节点角色
func (l *LeaderNodeImpl) SetRole(role ReplicationRole) {
	l.mu.Lock()
	defer l.mu.Unlock()

	prevRole := l.role
	l.role = role

	// 如果角色发生变化，更新相关状态
	if prevRole != role {
		if role == RoleLeader {
			l.leaderID = l.nodeID
			l.startHeartbeat()
		} else {
			// 如果不再是Leader，停止心跳
			if l.heartbeatTimer != nil {
				l.heartbeatTimer.Stop()
			}
		}
	}
}

// GetLeader 获取当前领导者ID
func (l *LeaderNodeImpl) GetLeader() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.leaderID
}

// AddFollower 添加从节点
func (l *LeaderNodeImpl) AddFollower(nodeID string, address string) error {
	l.mu.Lock()

	// 检查是否已存在
	if _, exists := l.followers[nodeID]; exists {
		l.mu.Unlock()
		return fmt.Errorf("follower '%s' already exists", nodeID)
	}

	// 添加到从节点列表
	l.followers[nodeID] = address
	l.followerStatus[nodeID] = StateOutOfSync

	// 复制地址用于goroutine
	targetAddress := address
	l.mu.Unlock()

	// 触发全量同步
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), l.config.ReplicationTimeout)
		defer cancel()

		err := l.syncer.FullSync(ctx, targetAddress)

		// 更新状态
		l.mu.Lock()
		defer l.mu.Unlock()

		if err == nil {
			l.followerStatus[nodeID] = StateNormal
		}
	}()

	return nil
}

// RemoveFollower 移除从节点
func (l *LeaderNodeImpl) RemoveFollower(nodeID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 检查是否存在
	if _, exists := l.followers[nodeID]; !exists {
		return fmt.Errorf("follower '%s' does not exist", nodeID)
	}

	// 从列表中移除
	delete(l.followers, nodeID)
	delete(l.followerStatus, nodeID)

	return nil
}

// GetFollowers 获取所有从节点
func (l *LeaderNodeImpl) GetFollowers() map[string]string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 返回副本而不是直接引用
	result := make(map[string]string, len(l.followers))
	for id, addr := range l.followers {
		result[id] = addr
	}

	return result
}

// ReplicateEntries 复制日志条目到所有从节点
func (l *LeaderNodeImpl) ReplicateEntries(ctx context.Context) error {
	// 检查是否为Leader
	l.mu.RLock()
	if l.role != RoleLeader {
		l.mu.RUnlock()
		return errors.New("not the leader")
	}

	// 复制从节点信息以避免在整个操作期间持有锁
	followers := make(map[string]string, len(l.followers))
	for nodeID, addr := range l.followers {
		followers[nodeID] = addr
	}
	l.mu.RUnlock()

	// 检查是否有从节点
	if len(followers) == 0 {
		return nil // 没有从节点，视为成功
	}

	var wg sync.WaitGroup
	var syncErrors []error
	var syncErrorsMu sync.Mutex

	// 对每个从节点执行增量同步
	for nodeID, addr := range followers {
		wg.Add(1)
		go func(id string, address string) {
			defer wg.Done()

			// 获取最后同步的索引
			lastIndex := uint64(0) // 这里简化处理，实际可能需要跟踪每个节点的同步进度

			// 执行增量同步
			err := l.syncer.IncrementalSync(ctx, address, lastIndex)
			if err != nil {
				syncErrorsMu.Lock()
				syncErrors = append(syncErrors, fmt.Errorf("sync to %s failed: %w", id, err))
				syncErrorsMu.Unlock()

				// 更新状态 - 使用单独的加锁区域
				l.mu.Lock()
				l.followerStatus[id] = StateOutOfSync
				l.mu.Unlock()
			} else {
				// 更新状态 - 使用单独的加锁区域
				l.mu.Lock()
				l.followerStatus[id] = StateNormal
				l.mu.Unlock()
			}
		}(nodeID, addr)
	}

	// 等待所有同步操作完成
	wg.Wait()

	// 如果有错误，返回第一个错误
	if len(syncErrors) > 0 {
		return syncErrors[0]
	}

	return nil
}

// SendHeartbeat 发送心跳到所有从节点
func (l *LeaderNodeImpl) SendHeartbeat(ctx context.Context) error {
	l.mu.RLock()
	followers := make(map[string]string, len(l.followers))
	for id, addr := range l.followers {
		followers[id] = addr
	}
	term := l.currentTerm
	l.mu.RUnlock()

	// 检查是否有从节点
	if len(followers) == 0 {
		return nil
	}

	var wg sync.WaitGroup

	// 向每个从节点发送心跳
	for nodeID, addr := range followers {
		wg.Add(1)
		go func(id string, address string) {
			defer wg.Done()

			// 创建心跳请求
			req := &SyncRequest{
				LeaderID:     l.nodeID,
				LastLogIndex: 0,
				LastLogTerm:  term,
				FullSync:     false, // 心跳不需要同步数据
			}

			// 发送请求
			_, err := l.RequestSync(ctx, address, req)

			// 如果心跳失败，更新从节点状态
			if err != nil {
				l.mu.Lock()
				l.followerStatus[id] = StateOutOfSync
				l.mu.Unlock()
			}
		}(nodeID, addr)
	}

	// 等待所有心跳发送完成
	wg.Wait()

	return nil
}

// RequestSync 请求同步数据
func (l *LeaderNodeImpl) RequestSync(ctx context.Context, target string, req *SyncRequest) (*SyncResponse, error) {
	// 这里简单实现，实际项目中可能需要基于HTTP或其他RPC机制
	// 该方法在实际项目中通常会调用底层网络传输层来发送请求

	// 在这个简化示例中，我们只是简单返回一个空响应
	return &SyncResponse{
		Success: true,
		Entries: []*ReplicationEntry{},
	}, nil
}

// ApplyEntry 应用日志条目到本地缓存
func (l *LeaderNodeImpl) ApplyEntry(ctx context.Context, entry *ReplicationEntry) error {
	if entry == nil {
		return errors.New("cannot apply nil entry")
	}

	// 根据条目类型执行不同操作
	switch entry.Command {
	case "Set":
		return l.cache.Set(ctx, entry.Key, entry.Value, entry.Expiration)
	case "Del":
		return l.cache.Del(ctx, entry.Key)
	default:
		return fmt.Errorf("unknown command: %s", entry.Command)
	}
}

// GetLog 获取复制日志
func (l *LeaderNodeImpl) GetLog() ReplicationLog {
	return l.log
}

// startHeartbeat 启动心跳定时器
func (l *LeaderNodeImpl) startHeartbeat() {
	// 停止现有的心跳定时器
	if l.heartbeatTimer != nil {
		l.heartbeatTimer.Stop()
	}

	// 创建新的定时器
	l.heartbeatTimer = time.NewTimer(l.config.HeartbeatInterval)

	// 启动心跳协程
	go func() {
		for {
			select {
			case <-l.stopCh:
				return
			case <-l.heartbeatTimer.C:
				func() {
					ctx, cancel := context.WithTimeout(context.Background(), l.config.HeartbeatInterval/2)
					defer cancel()

					// 发送心跳
					_ = l.SendHeartbeat(ctx)

					// 重置定时器
					l.mu.Lock()
					if l.running && l.role == RoleLeader {
						l.heartbeatTimer.Reset(l.config.HeartbeatInterval)
					}
					l.mu.Unlock()
				}()
			}
		}
	}()
}

// UpdateFollowerStatus 更新从节点状态
func (l *LeaderNodeImpl) UpdateFollowerStatus(nodeID string, state ReplicationState) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.followers[nodeID]; exists {
		l.followerStatus[nodeID] = state
	}
}

// GetFollowerStatus 获取从节点状态
func (l *LeaderNodeImpl) GetFollowerStatus() map[string]ReplicationState {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make(map[string]ReplicationState, len(l.followerStatus))
	for id, state := range l.followerStatus {
		result[id] = state
	}

	return result
}
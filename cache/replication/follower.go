package replication

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
)

// FollowerNodeImpl 实现了 FollowerNode 接口
type FollowerNodeImpl struct {
	// 节点标识和角色
	nodeID string
	role   ReplicationRole

	// 本地缓存实例
	cache cache.Cache

	// 复制日志，用于记录和同步操作
	log ReplicationLog

	// 当前领导者ID
	leaderID string

	// 领导者地址
	leaderAddr string

	// 配置选项
	config *ReplicationConfig

	// 数据同步器实例
	syncer Syncer

	// 互斥锁保护并发访问
	mu sync.RWMutex

	// 心跳检测定时器
	heartbeatTimer *time.Timer

	// 心跳超时时间
	heartbeatTimeout time.Duration

	// 最后一次收到心跳的时间
	lastHeartbeat time.Time

	// 停止信号通道
	stopCh chan struct{}

	// 运行状态
	running bool

	// 当前同步状态
	syncState ReplicationState

	// 当前任期
	currentTerm uint64

	// 选举回调函数
	onLeaderFail func()
}

// NewFollowerNode 创建一个新的从节点
func NewFollowerNode(nodeID string, cache cache.Cache, log ReplicationLog, syncer Syncer, options ...ReplicationOption) *FollowerNodeImpl {
	// 使用默认配置
	config := DefaultReplicationConfig()

	// 应用选项
	for _, opt := range options {
		opt(config)
	}

	return &FollowerNodeImpl{
		nodeID:           nodeID,
		cache:            cache,
		log:              log,
		config:           config,
		syncer:           syncer,
		role:             RoleFollower,
		syncState:        StateOutOfSync,
		heartbeatTimeout: config.ElectionTimeout,
		lastHeartbeat:    time.Now(),
		stopCh:           make(chan struct{}),
		currentTerm:      0,
	}
}

// Start 启动从节点服务
func (f *FollowerNodeImpl) Start() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.running {
		return nil
	}

	f.running = true

	// 启动心跳检测
	f.startHeartbeatMonitor()

	return nil
}

// Stop 停止从节点服务
func (f *FollowerNodeImpl) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.running {
		return nil
	}

	// 停止心跳检测定时器
	if f.heartbeatTimer != nil {
		f.heartbeatTimer.Stop()
	}

	close(f.stopCh)
	f.running = false
	return nil
}

// GetRole 获取节点角色
func (f *FollowerNodeImpl) GetRole() ReplicationRole {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.role
}

// SetRole 设置节点角色
func (f *FollowerNodeImpl) SetRole(role ReplicationRole) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.role = role
}

// GetLeader 获取当前领导者ID
func (f *FollowerNodeImpl) GetLeader() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.leaderID
}

// SetLeader 设置领导者ID和地址
func (f *FollowerNodeImpl) SetLeader(leaderID string, leaderAddr string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.leaderID = leaderID
	f.leaderAddr = leaderAddr

	// 重置最后心跳时间
	f.lastHeartbeat = time.Now()
}

// SyncFromLeader 从主节点同步数据
func (f *FollowerNodeImpl) SyncFromLeader(ctx context.Context) error {
	f.mu.Lock()
	leaderAddr := f.leaderAddr
	f.mu.Unlock()

	if leaderAddr == "" {
		return errors.New("no leader available for sync")
	}

	f.mu.Lock()
	f.syncState = StateSyncing
	f.mu.Unlock()

	// 执行增量同步
	lastIndex := f.log.GetLastIndex()
	err := f.syncer.IncrementalSync(ctx, leaderAddr, lastIndex)

	f.mu.Lock()
	defer f.mu.Unlock()

	if err != nil {
		f.syncState = StateOutOfSync
		return fmt.Errorf("failed to sync from leader: %w", err)
	}

	f.syncState = StateNormal
	return nil
}

// RequestFullSync 请求全量同步
func (f *FollowerNodeImpl) RequestFullSync(ctx context.Context) error {
	f.mu.RLock()
	leaderAddr := f.leaderAddr
	f.mu.RUnlock()

	if leaderAddr == "" {
		return errors.New("no leader available for full sync")
	}

	f.mu.Lock()
	f.syncState = StateSyncing
	f.mu.Unlock()

	// 执行全量同步
	err := f.syncer.FullSync(ctx, leaderAddr)

	f.mu.Lock()
	defer f.mu.Unlock()

	if err != nil {
		f.syncState = StateOutOfSync
		return fmt.Errorf("failed to perform full sync: %w", err)
	}

	f.syncState = StateNormal
	return nil
}

// HandleHeartbeat 处理心跳消息
func (f *FollowerNodeImpl) HandleHeartbeat(ctx context.Context, leaderID string, term uint64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// 如果收到更高任期的心跳，更新任期
	if term > f.currentTerm {
		f.currentTerm = term
	}

	// 更新最后心跳时间
	f.lastHeartbeat = time.Now()

	// 如果当前没有领导者，设置领导者ID
	if f.leaderID == "" {
		f.leaderID = leaderID
	}

	return nil
}

// PromoteToLeader 提升为领导者
func (f *FollowerNodeImpl) PromoteToLeader(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// 增加任期
	f.currentTerm++

	// 更改角色
	f.role = RoleLeader

	// 记录自己为领导者
	f.leaderID = f.nodeID

	// 停止心跳监控
	if f.heartbeatTimer != nil {
		f.heartbeatTimer.Stop()
	}

	return nil
}

// RequestSync 请求同步数据
func (f *FollowerNodeImpl) RequestSync(ctx context.Context, target string, req *SyncRequest) (*SyncResponse, error) {
	// 这里简单实现，实际项目中可能需要基于HTTP或其他RPC机制
	// 该方法在实际项目中通常会调用底层网络传输层来发送请求

	// 在这个简化示例中，我们只是简单返回一个空响应
	return &SyncResponse{
		Success: true,
		Entries: []*ReplicationEntry{},
	}, nil
}

// ApplyEntry 应用一条日志条目到本地缓存
func (f *FollowerNodeImpl) ApplyEntry(ctx context.Context, entry *ReplicationEntry) error {
	if entry == nil {
		return errors.New("cannot apply nil entry")
	}

	// 根据条目类型执行不同操作
	switch entry.Command {
	case "Set":
		return f.cache.Set(ctx, entry.Key, entry.Value, entry.Expiration)
	case "Del":
		return f.cache.Del(ctx, entry.Key)
	default:
		return fmt.Errorf("unknown command: %s", entry.Command)
	}
}

// GetLog 获取复制日志
func (f *FollowerNodeImpl) GetLog() ReplicationLog {
	return f.log
}

// startHeartbeatMonitor 启动心跳监控定时器
func (f *FollowerNodeImpl) startHeartbeatMonitor() {
	// 停止现有的心跳定时器
	if f.heartbeatTimer != nil {
		f.heartbeatTimer.Stop()
	}

	// 创建新的定时器
	f.heartbeatTimer = time.NewTimer(f.heartbeatTimeout)

	// 启动监控协程
	go func() {
		for {
			select {
			case <-f.stopCh:
				return
			case <-f.heartbeatTimer.C:
				func() {
					f.mu.Lock()
					defer f.mu.Unlock()

					// 检查是否超时
					elapsed := time.Since(f.lastHeartbeat)
					if elapsed > f.heartbeatTimeout {
						// 领导者可能已失效
						if f.onLeaderFail != nil {
							// 执行选举回调
							go f.onLeaderFail()
						}
					}

					// 无论是否超时，重置定时器继续监控
					f.heartbeatTimer.Reset(f.heartbeatTimeout)
				}()
			}
		}
	}()
}

// SetElectionCallback 设置选举回调函数
// 当检测到领导者可能失效时会调用此函数
func (f *FollowerNodeImpl) SetElectionCallback(callback func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onLeaderFail = callback
}

// GetSyncState 获取当前同步状态
func (f *FollowerNodeImpl) GetSyncState() ReplicationState {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.syncState
}

// GetCurrentTerm 获取当前任期
func (f *FollowerNodeImpl) GetCurrentTerm() uint64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.currentTerm
}

// UpdateLastHeartbeat 更新最后心跳时间
func (f *FollowerNodeImpl) UpdateLastHeartbeat() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastHeartbeat = time.Now()
}
package integration

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/fyerfyer/fyer-cache/cache/cluster"
	"github.com/fyerfyer/fyer-cache/cache/replication"
)

// LeaderFollowerCluster 实现了主从架构的缓存集群
type LeaderFollowerCluster struct {
	// 节点标识
	nodeID string

	// 节点地址
	address string

	// 本地缓存实例
	localCache cache.Cache

	// 集群节点实例
	clusterNode *cluster.Node

	// 复制日志
	log replication.ReplicationLog

	// 数据同步器
	syncer replication.Syncer

	// 领导者节点实现
	leaderNode replication.LeaderNode

	// 跟随者节点实现
	followerNode replication.FollowerNode

	// 当前节点角色
	currentRole replication.ReplicationRole

	// 复制配置
	config *replication.ReplicationConfig

	// 互斥锁保护并发访问
	mu sync.RWMutex

	// 运行状态
	running bool

	// 停止信号通道
	stopCh chan struct{}

	// 故障转移监控器停止信号通道
	failoverStopCh chan struct{}

	// 事件处理完成等待组
	wg sync.WaitGroup
}

// LeaderFollowerClusterOption 配置选项函数
type LeaderFollowerClusterOption func(*LeaderFollowerCluster)

// WithSyncerAddress 设置同步器地址
func WithSyncerAddress(syncerAddr string) LeaderFollowerClusterOption {
	return func(lfc *LeaderFollowerCluster) {
		if lfc.config != nil {
			lfc.config.SyncerAddress = syncerAddr
		}
	}
}

// WithInitialRole 设置初始角色
func WithInitialRole(role replication.ReplicationRole) LeaderFollowerClusterOption {
	return func(lfc *LeaderFollowerCluster) {
		lfc.currentRole = role
	}
}

// WithElectionTimeout 设置选举超时
func WithElectionTimeout(timeout time.Duration) LeaderFollowerClusterOption {
	return func(lfc *LeaderFollowerCluster) {
		if timeout > 0 {
			lfc.config.ElectionTimeout = timeout
		}
	}
}

// WithHeartbeatInterval 设置心跳间隔
func WithHeartbeatInterval(interval time.Duration) LeaderFollowerClusterOption {
	return func(lfc *LeaderFollowerCluster) {
		if interval > 0 {
			lfc.config.HeartbeatInterval = interval
		}
	}
}

// NewLeaderFollowerCluster 创建新的主从集群
func NewLeaderFollowerCluster(
	nodeID string,
	address string,
	localCache cache.Cache,
	options ...LeaderFollowerClusterOption,
) (*LeaderFollowerCluster, error) {
	if nodeID == "" {
		return nil, errors.New("node ID cannot be empty")
	}

	if localCache == nil {
		return nil, errors.New("local cache cannot be nil")
	}

	// 使用默认复制配置
	config := replication.DefaultReplicationConfig()
	config.NodeID = nodeID
	config.Address = address
	config.SyncerAddress = "" // 默认为空，会由选项函数设置

	// 创建复制日志
	log := replication.NewMemoryReplicationLog()

	// 创建数据同步器
	syncer := replication.NewMemorySyncer(nodeID, localCache, log, replication.WithAddress(address))

	// 创建集群实例
	lfc := &LeaderFollowerCluster{
		nodeID:      nodeID,
		address:     address,
		localCache:  localCache,
		log:         log,
		currentRole: replication.RoleFollower, // 默认为Follower
		config:      config,
		stopCh:      make(chan struct{}),
		failoverStopCh: make(chan struct{}),
	}

	// 应用选项
	for _, opt := range options {
		opt(lfc)
	}

	// 如果没有设置同步器地址，使用集群地址
    if lfc.config.SyncerAddress == "" {
        lfc.config.SyncerAddress = address
    }

	// 创建数据同步器
    lfc.syncer = replication.NewMemorySyncer(nodeID, localCache, lfc.log,
        replication.WithNodeID(nodeID),
        replication.WithAddress(lfc.config.SyncerAddress))

	// 根据初始角色创建相应的节点
	if lfc.currentRole == replication.RoleLeader {
		lfc.leaderNode = replication.NewLeaderNode(nodeID, localCache, log, syncer,
			replication.WithNodeID(nodeID),
			replication.WithAddress(address),
			replication.WithRole(replication.RoleLeader),
			replication.WithHeartbeatInterval(config.HeartbeatInterval),
			replication.WithElectionTimeout(config.ElectionTimeout),
		)
	} else {
		lfc.followerNode = replication.NewFollowerNode(nodeID, localCache, log, syncer,
			replication.WithNodeID(nodeID),
			replication.WithAddress(address),
			replication.WithRole(replication.RoleFollower),
			replication.WithHeartbeatInterval(config.HeartbeatInterval),
			replication.WithElectionTimeout(config.ElectionTimeout),
		)
	}

	// 创建集群节点
	clusterNode, err := cluster.NewNode(nodeID, localCache,
		cluster.WithNodeID(nodeID),
		cluster.WithBindAddr(address),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster node: %w", err)
	}
	lfc.clusterNode = clusterNode

	return lfc, nil
}

// Start 启动主从集群
func (lfc *LeaderFollowerCluster) Start() error {
	lfc.mu.Lock()
	defer lfc.mu.Unlock()

	if lfc.running {
		return nil
	}

	// 先启动集群节点
	if err := lfc.clusterNode.Start(); err != nil {
		return fmt.Errorf("failed to start cluster node: %w", err)
	}

	// 启动同步器
	if err := lfc.syncer.Start(); err != nil {
		// 启动失败，停止集群节点
		_ = lfc.clusterNode.Stop()
		return fmt.Errorf("failed to start syncer: %w", err)
	}

	// 根据角色启动相应的节点
	if lfc.currentRole == replication.RoleLeader {
		if err := lfc.leaderNode.Start(); err != nil {
			// 启动失败，停止之前启动的组件
			_ = lfc.syncer.Stop()
			_ = lfc.clusterNode.Stop()
			return fmt.Errorf("failed to start leader node: %w", err)
		}
	} else {
		if err := lfc.followerNode.Start(); err != nil {
			// 启动失败，停止之前启动的组件
			_ = lfc.syncer.Stop()
			_ = lfc.clusterNode.Stop()
			return fmt.Errorf("failed to start follower node: %w", err)
		}

		// 设置选举回调
		lfc.setElectionCallback()
	}

	// 开始监听集群事件
	lfc.wg.Add(1)
	go lfc.handleClusterEvents()

	lfc.running = true
	return nil
}

// Stop 停止主从集群
func (lfc *LeaderFollowerCluster) Stop() error {
	lfc.mu.Lock()
	defer lfc.mu.Unlock()

	if !lfc.running {
		return nil
	}

	// 停止故障转移监控
	select {
	case <-lfc.failoverStopCh:
		// 已关闭
	default:
		close(lfc.failoverStopCh)
	}

	// 停止事件监听
	close(lfc.stopCh)
	lfc.wg.Wait()

	// 根据角色停止相应的节点
	var firstErr error
	if lfc.currentRole == replication.RoleLeader && lfc.leaderNode != nil {
		if leaderImpl, ok := lfc.leaderNode.(*replication.LeaderNodeImpl); ok {
			if err := leaderImpl.Stop(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	} else if lfc.followerNode != nil {
		if followerImpl, ok := lfc.followerNode.(*replication.FollowerNodeImpl); ok {
			if err := followerImpl.Stop(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}

	// 停止同步器
	if err := lfc.syncer.Stop(); err != nil && firstErr == nil {
		firstErr = err
	}

	// 停止集群节点
	if err := lfc.clusterNode.Stop(); err != nil && firstErr == nil {
		firstErr = err
	}

	lfc.running = false
	return firstErr
}

// Join 加入集群
func (lfc *LeaderFollowerCluster) Join(seedAddr string) error {
	// 先通过集群节点加入集群
	if err := lfc.clusterNode.Join(seedAddr); err != nil {
		return fmt.Errorf("failed to join cluster: %w", err)
	}

	// 如果是跟随者，需要找到领导者并设置
	if lfc.currentRole == replication.RoleFollower && lfc.followerNode != nil {
		members := lfc.clusterNode.Members()

		// 查找领导者节点
		for _, member := range members {
			// 检查元数据中是否标记了领导者
			if member.Metadata != nil && member.Metadata["role"] == "leader" {
				// 找到领导者，设置跟随者的领导者
				if followerImpl, ok := lfc.followerNode.(*replication.FollowerNodeImpl); ok {
					followerImpl.SetLeader(member.ID, member.Addr)

					// 请求全量同步
					ctx, cancel := context.WithTimeout(context.Background(), lfc.config.ReplicationTimeout)
					defer cancel()
					_ = followerImpl.RequestFullSync(ctx)

					break
				}
			}
		}
	}

	return nil
}

// Leave 离开集群
func (lfc *LeaderFollowerCluster) Leave() error {
	// 通过集群节点离开集群
	return lfc.clusterNode.Leave()
}

// IsLeader 检查当前节点是否为领导者
func (lfc *LeaderFollowerCluster) IsLeader() bool {
	lfc.mu.RLock()
	defer lfc.mu.RUnlock()
	return lfc.currentRole == replication.RoleLeader
}

// GetLeader 获取当前领导者ID
func (lfc *LeaderFollowerCluster) GetLeader() string {
	lfc.mu.RLock()
	defer lfc.mu.RUnlock()

	if lfc.currentRole == replication.RoleLeader {
		return lfc.nodeID
	}

	if lfc.followerNode != nil {
		return lfc.followerNode.GetLeader()
	}

	return ""
}

// PromoteToLeader 提升为领导者
func (lfc *LeaderFollowerCluster) PromoteToLeader() error {
	lfc.mu.Lock()
	defer lfc.mu.Unlock()

	if lfc.currentRole == replication.RoleLeader {
		return nil // 已经是领导者
	}

	// 停止跟随者节点
	if lfc.followerNode != nil {
		if followerImpl, ok := lfc.followerNode.(*replication.FollowerNodeImpl); ok {
			if err := followerImpl.Stop(); err != nil {
				return fmt.Errorf("failed to stop follower node: %w", err)
			}
		}

		// 创建领导者节点
		lfc.leaderNode = replication.NewLeaderNode(lfc.nodeID, lfc.localCache, lfc.log, lfc.syncer,
			replication.WithNodeID(lfc.nodeID),
			replication.WithAddress(lfc.address),
			replication.WithRole(replication.RoleLeader),
			replication.WithHeartbeatInterval(lfc.config.HeartbeatInterval),
			replication.WithElectionTimeout(lfc.config.ElectionTimeout),
		)

		// 启动领导者节点
		if leaderImpl, ok := lfc.leaderNode.(*replication.LeaderNodeImpl); ok {
			if err := leaderImpl.Start(); err != nil {
				return fmt.Errorf("failed to start leader node: %w", err)
			}
		}

		// 更新角色
		lfc.currentRole = replication.RoleLeader
		lfc.followerNode = nil

		// 更新集群元数据，通知其他节点自己是领导者
		metadata := map[string]string{"role": "leader"}
		lfc.clusterNode.Membership().SetLocalMetadata(metadata)
	}

	return nil
}

// DemoteToFollower 降级为跟随者
func (lfc *LeaderFollowerCluster) DemoteToFollower(leaderID, leaderAddr string) error {
	lfc.mu.Lock()
	defer lfc.mu.Unlock()

	if lfc.currentRole == replication.RoleFollower {
		return nil // 已经是跟随者
	}

	// 停止领导者节点
	if lfc.leaderNode != nil {
		if leaderImpl, ok := lfc.leaderNode.(*replication.LeaderNodeImpl); ok {
			if err := leaderImpl.Stop(); err != nil {
				return fmt.Errorf("failed to stop leader node: %w", err)
			}
		}

		// 创建跟随者节点
		lfc.followerNode = replication.NewFollowerNode(lfc.nodeID, lfc.localCache, lfc.log, lfc.syncer,
			replication.WithNodeID(lfc.nodeID),
			replication.WithAddress(lfc.address),
			replication.WithRole(replication.RoleFollower),
			replication.WithHeartbeatInterval(lfc.config.HeartbeatInterval),
			replication.WithElectionTimeout(lfc.config.ElectionTimeout),
		)

		// 启动跟随者节点
		if followerImpl, ok := lfc.followerNode.(*replication.FollowerNodeImpl); ok {
			followerImpl.SetLeader(leaderID, leaderAddr)
			if err := followerImpl.Start(); err != nil {
				return fmt.Errorf("failed to start follower node: %w", err)
			}

			// 设置选举回调
			lfc.setElectionCallback()

			// 请求全量同步
			ctx, cancel := context.WithTimeout(context.Background(), lfc.config.ReplicationTimeout)
			defer cancel()
			_ = followerImpl.RequestFullSync(ctx)
		}

		// 更新角色
		lfc.currentRole = replication.RoleFollower
		lfc.leaderNode = nil

		// 更新集群元数据
		metadata := map[string]string{"role": "follower"}
		lfc.clusterNode.Membership().SetLocalMetadata(metadata)
	}

	return nil
}

// AddFollower 添加从节点（仅Leader可调用）
func (lfc *LeaderFollowerCluster) AddFollower(nodeID, address string) error {
	lfc.mu.RLock()
	defer lfc.mu.RUnlock()

	if lfc.currentRole != replication.RoleLeader || lfc.leaderNode == nil {
		return errors.New("only leader can add followers")
	}

	return lfc.leaderNode.AddFollower(nodeID, address)
}

// RemoveFollower 移除从节点（仅Leader可调用）
func (lfc *LeaderFollowerCluster) RemoveFollower(nodeID string) error {
	lfc.mu.RLock()
	defer lfc.mu.RUnlock()

	if lfc.currentRole != replication.RoleLeader || lfc.leaderNode == nil {
		return errors.New("only leader can remove followers")
	}

	return lfc.leaderNode.RemoveFollower(nodeID)
}

// SyncFromLeader 从领导者同步数据（仅Follower可调用）
func (lfc *LeaderFollowerCluster) SyncFromLeader(ctx context.Context) error {
	lfc.mu.RLock()
	defer lfc.mu.RUnlock()

	if lfc.currentRole != replication.RoleFollower || lfc.followerNode == nil {
		return errors.New("only follower can sync from leader")
	}

	return lfc.followerNode.SyncFromLeader(ctx)
}

// handleClusterEvents 处理集群事件
func (lfc *LeaderFollowerCluster) handleClusterEvents() {
	defer lfc.wg.Done()

	eventCh := lfc.clusterNode.Events()

	for {
		select {
		case <-lfc.stopCh:
			return
		case event, ok := <-eventCh:
			if !ok {
				return // 通道已关闭
			}

			// 处理集群事件
			switch event.Type {
			case cache.EventNodeJoin:
				// 处理节点加入事件
				lfc.handleNodeJoin(event)

			case cache.EventNodeLeave, cache.EventNodeFailed:
				// 处理节点离开或失败事件
				lfc.handleNodeFailure(event)
			}
		}
	}
}

// handleNodeJoin 处理节点加入事件
func (lfc *LeaderFollowerCluster) handleNodeJoin(event cache.ClusterEvent) {
	// 如果当前节点是领导者，需要考虑将新节点添加为从节点
	lfc.mu.RLock()
	defer lfc.mu.RUnlock()

	if lfc.currentRole == replication.RoleLeader && lfc.leaderNode != nil {
		// 获取节点地址
		addr, ok := event.Details.(string)
		if !ok {
			return
		}

		// 添加从节点
		_ = lfc.leaderNode.AddFollower(event.NodeID, addr)
	}
}

// handleNodeFailure 处理节点失败事件
func (lfc *LeaderFollowerCluster) handleNodeFailure(event cache.ClusterEvent) {
	// 如果检测到领导者节点失败，可能需要触发选举
	if lfc.currentRole == replication.RoleFollower && lfc.followerNode != nil {
		leaderID := lfc.followerNode.GetLeader()
		if event.NodeID == leaderID {
			// 领导者失效，可能需要进行故障转移
			lfc.handleFailover()
		}
	} else if lfc.currentRole == replication.RoleLeader && lfc.leaderNode != nil {
		// 如果是从节点失败，从列表中移除
		_ = lfc.leaderNode.RemoveFollower(event.NodeID)
	}
}

// handleFailover 处理故障转移
func (lfc *LeaderFollowerCluster) handleFailover() {
	lfc.mu.Lock()

	// 重置故障转移监控通道
	select {
	case <-lfc.failoverStopCh:
		lfc.failoverStopCh = make(chan struct{})
	default:
	}

	// 在尝试选举之前等待一段时间，防止多个节点同时尝试成为领导者
	waitTime := time.Duration(100+lfc.config.Priority*50) * time.Millisecond
	failoverStopCh := lfc.failoverStopCh
	lfc.mu.Unlock()

	// 等待一段时间后尝试选举
	select {
	case <-time.After(waitTime):
		// 尝试提升为领导者
		_ = lfc.PromoteToLeader()
	case <-failoverStopCh:
		// 故障转移被取消
		return
	}
}

// setElectionCallback 设置选举回调
func (lfc *LeaderFollowerCluster) setElectionCallback() {
	if followerImpl, ok := lfc.followerNode.(*replication.FollowerNodeImpl); ok {
		followerImpl.SetElectionCallback(func() {
			// 选举回调触发时尝试故障转移
			go lfc.handleFailover()
		})
	}
}

// Get 从缓存获取值
func (lfc *LeaderFollowerCluster) Get(ctx context.Context, key string) (any, error) {
	return lfc.localCache.Get(ctx, key)
}

// Set 设置缓存值
func (lfc *LeaderFollowerCluster) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
	// 记录操作到同步日志
	if lfc.currentRole == replication.RoleLeader {
		if msync, ok := lfc.syncer.(*replication.MemorySyncer); ok {
			value, ok := val.([]byte)
			if !ok {
				// 尝试转换为字节数组
				if str, ok := val.(string); ok {
					value = []byte(str)
				} else {
					// 其他类型暂不支持
					return errors.New("only []byte and string values are supported for replication")
				}
			}

			if err := msync.RecordSetOperation(key, value, expiration, 0); err != nil {
				return fmt.Errorf("failed to record set operation: %w", err)
			}
		}
	}

	// 本地执行
	return lfc.localCache.Set(ctx, key, val, expiration)
}

// Del 删除缓存值
func (lfc *LeaderFollowerCluster) Del(ctx context.Context, key string) error {
	// 记录操作到同步日志
	if lfc.currentRole == replication.RoleLeader {
		if msync, ok := lfc.syncer.(*replication.MemorySyncer); ok {
			if err := msync.RecordDelOperation(key, 0); err != nil {
				return fmt.Errorf("failed to record delete operation: %w", err)
			}
		}
	}

	// 本地执行
	return lfc.localCache.Del(ctx, key)
}

// GetClusterInfo 获取集群信息
func (lfc *LeaderFollowerCluster) GetClusterInfo() map[string]interface{} {
	lfc.mu.RLock()
	defer lfc.mu.RUnlock()

	info := make(map[string]interface{})
	info["nodeID"] = lfc.nodeID
	info["address"] = lfc.address
	info["role"] = lfc.currentRole.String()

	if lfc.currentRole == replication.RoleLeader && lfc.leaderNode != nil {
		followers := lfc.leaderNode.GetFollowers()
		info["followers"] = followers
		info["followerCount"] = len(followers)
	} else {
		info["leader"] = lfc.GetLeader()
	}

	// 添加集群成员信息
	members := lfc.clusterNode.Members()
	memberInfo := make(map[string]string)
	for _, member := range members {
		memberInfo[member.ID] = member.Addr
	}
	info["members"] = memberInfo
	info["memberCount"] = len(members)

	return info
}

// ReplicateNow 立即触发复制
func (lfc *LeaderFollowerCluster) ReplicateNow(ctx context.Context) error {
	lfc.mu.RLock()
	defer lfc.mu.RUnlock()

	if lfc.currentRole != replication.RoleLeader || lfc.leaderNode == nil {
		return errors.New("only leader can trigger replication")
	}

	return lfc.leaderNode.ReplicateEntries(ctx)
}
package integration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/fyerfyer/fyer-cache/cache/cluster"
)

// ShardedClusterCache 整合一致性哈希分片与集群功能的缓存实现
type ShardedClusterCache struct {
	// 本地缓存实例
	localCache cache.Cache

	// 分片缓存实例，用于路由请求到正确的节点
	shardedCache *cache.NodeShardedCache

	// 集群节点
	node *cluster.Node

	// 节点分发器，处理节点加入/离开事件
	nodeDistributor *NodeDistributor

	// 本地节点元数据
	localNodeID string
	address     string

	// 保护状态的互斥锁
	mu sync.RWMutex

	// 是否已经启动
	running bool
}

// ShardedClusterCacheOption 配置选项函数
type ShardedClusterCacheOption func(*ShardedClusterCache)

// WithVirtualNodeCount 设置一致性哈希环的虚拟节点数量
func WithVirtualNodeCount(count int) ShardedClusterCacheOption {
	return func(scc *ShardedClusterCache) {
		if count > 0 && scc.shardedCache != nil {
			// 更新虚拟节点数量
			cache.WithNodeHashReplicas(count)(scc.shardedCache)
		}
	}
}

// WithReplicaFactor 设置副本因子（每个键存储在多少个节点上）
func WithReplicaFactor(factor int) ShardedClusterCacheOption {
	return func(scc *ShardedClusterCache) {
		if factor > 0 && scc.shardedCache != nil {
			// 更新复制因子
			cache.WithNodeReplicaFactor(factor)(scc.shardedCache)
		}
	}
}

// NewShardedClusterCache 创建新的分片集群缓存
func NewShardedClusterCache(localCache cache.Cache, nodeID string, bindAddr string, clusterOptions []cluster.NodeOption, cacheOptions []ShardedClusterCacheOption) (*ShardedClusterCache, error) {
	if localCache == nil {
		return nil, fmt.Errorf("local cache cannot be nil")
	}

	if nodeID == "" {
		return nil, fmt.Errorf("node ID cannot be empty")
	}

	if bindAddr == "" {
		return nil, fmt.Errorf("bind address cannot be empty")
	}

	// 创建分片缓存
	shardedCache := cache.NewNodeShardedCache()

	// 将本地缓存添加到分片缓存
	err := shardedCache.AddNode(nodeID, localCache, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to add local cache to sharded cache: %w", err)
	}

	// 创建集群节点 - 确保绑定地址被正确传入
	nodeOptions := append([]cluster.NodeOption{
		cluster.WithBindAddr(bindAddr), // 确保这个值被正确使用
		cluster.WithNodeID(nodeID),
	}, clusterOptions...)

	node, err := cluster.NewNode(nodeID, localCache, nodeOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster node: %w", err)
	}

	// 创建节点分发器，管理分片与集群的整合
	distributor := NewNodeDistributor(nodeID, shardedCache)

	scc := &ShardedClusterCache{
		localCache:      localCache,
		shardedCache:    shardedCache,
		node:            node,
		nodeDistributor: distributor,
		localNodeID:     nodeID,
		address:         bindAddr,
	}

	// 应用配置选项
	for _, opt := range cacheOptions {
		opt(scc)
	}

	return scc, nil
}

// Start 启动分片集群缓存
func (scc *ShardedClusterCache) Start() error {
	scc.mu.Lock()
	defer scc.mu.Unlock()

	if scc.running {
		return nil
	}

	// 启动集群节点
	err := scc.node.Start()
	if err != nil {
		return fmt.Errorf("failed to start cluster node: %w", err)
	}

	// 设置事件处理器以更新分片
	go scc.handleClusterEvents()

	scc.running = true
	return nil
}

// Stop 停止分片集群缓存
func (scc *ShardedClusterCache) Stop() error {
	scc.mu.Lock()
	defer scc.mu.Unlock()

	if !scc.running {
		return nil
	}

	// 停止集群节点
	err := scc.node.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop cluster node: %w", err)
	}

	// 清理节点分发器
	scc.nodeDistributor.Clear()

	scc.running = false
	return nil
}

// Join 加入集群
func (scc *ShardedClusterCache) Join(seedNodeAddr string) error {
	scc.mu.RLock()
	running := scc.running
	scc.mu.RUnlock()

	if !running {
		return fmt.Errorf("cache must be started before joining a cluster")
	}

	// 加入集群
	err := scc.node.Join(seedNodeAddr)
	if err != nil {
		return fmt.Errorf("failed to join cluster: %w", err)
	}

	// 初始化节点分发器，从现有成员构建分片
	members := scc.node.Members()
	err = scc.nodeDistributor.InitFromMembers(members)
	if err != nil {
		return fmt.Errorf("failed to initialize from members: %w", err)
	}

	return nil
}

// Leave 离开集群
func (scc *ShardedClusterCache) Leave() error {
	scc.mu.RLock()
	running := scc.running
	scc.mu.RUnlock()

	if !running {
		return nil
	}

	// 离开集群
	return scc.node.Leave()
}

// Members 获取集群成员列表
func (scc *ShardedClusterCache) Members() []cache.NodeInfo {
	return scc.node.Members()
}

// IsCoordinator 判断当前节点是否为协调者
func (scc *ShardedClusterCache) IsCoordinator() bool {
	return scc.node.IsCoordinator()
}

// handleClusterEvents 处理集群事件
func (scc *ShardedClusterCache) handleClusterEvents() {
	// 获取集群事件通道
	eventCh := scc.node.Events()

	for event := range eventCh {
		// 将集群事件传递给节点分发器
		scc.nodeDistributor.HandleClusterEvent(event)
	}
}

// Cache 接口实现

// Get 从缓存获取值
func (scc *ShardedClusterCache) Get(ctx context.Context, key string) (any, error) {
	// 添加日志，跟踪获取的键
	fmt.Printf("[ShardedClusterCache.Get] Getting key=%s on node=%s\n", key, scc.localNodeID)

	// 使用分片缓存的路由逻辑来确定正确的节点
	value, err := scc.shardedCache.Get(ctx, key)
	fmt.Printf("[ShardedClusterCache.Get] Result: value=%v, err=%v\n", value, err)
	return value, err
}

// Set 设置缓存值
func (scc *ShardedClusterCache) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	// 添加日志，跟踪设置的键和值
	fmt.Printf("[ShardedClusterCache.Set] Setting key=%s, value=%v on node=%s\n", key, value, scc.localNodeID)

	// 使用分片缓存的路由逻辑来确定正确的节点
	err := scc.shardedCache.Set(ctx, key, value, expiration)
	fmt.Printf("[ShardedClusterCache.Set] Result: err=%v\n", err)
	return err
}

// Del 删除缓存值
func (scc *ShardedClusterCache) Del(ctx context.Context, key string) error {
	// 添加日志，跟踪删除的键
	fmt.Printf("[ShardedClusterCache.Del] Deleting key=%s on node=%s\n", key, scc.localNodeID)

	// 使用分片缓存的路由逻辑来确定正确的节点
	err := scc.shardedCache.Del(ctx, key)
	fmt.Printf("[ShardedClusterCache.Del] Result: err=%v\n", err)
	return err
}

// GetNodeCount 获取集群中的节点数量
func (scc *ShardedClusterCache) GetNodeCount() int {
	return len(scc.shardedCache.GetNodeIDs())
}

// GetClusterInfo 获取集群信息
func (scc *ShardedClusterCache) GetClusterInfo() map[string]interface{} {
	members := scc.Members()

	// 计算各种状态的节点数
	var upCount, suspectCount, downCount, leftCount int
	for _, member := range members {
		switch member.Status {
		case cache.NodeStatusUp:
			upCount++
		case cache.NodeStatusSuspect:
			suspectCount++
		case cache.NodeStatusDown:
			downCount++
		case cache.NodeStatusLeft:
			leftCount++
		}
	}

	return map[string]interface{}{
		"node_id":        scc.localNodeID,
		"address":        scc.address,
		"is_coordinator": scc.IsCoordinator(),
		"member_count":   len(members),
		"node_count":     scc.GetNodeCount(),
		"replica_factor": scc.shardedCache.ReplicaFactor(),
		"status_counts": map[string]int{
			"up":      upCount,
			"suspect": suspectCount,
			"down":    downCount,
			"left":    leftCount,
		},
	}
}

// SetLocalMetadata 设置本地节点的元数据
func (scc *ShardedClusterCache) SetLocalMetadata(metadata map[string]string) {
	// 需要通过节点的内部结构访问并更新本地节点的元数据
	if scc.node != nil && scc.node.Membership() != nil {
		scc.node.Membership().SetLocalMetadata(metadata)
	}
}

// Events 获取事件通道
func (scc *ShardedClusterCache) Events() <-chan cache.ClusterEvent {
	return scc.node.Events()
}

// ReplicaFactor 获取当前的副本因子
func (scc *ShardedClusterCache) ReplicaFactor() int {
	return scc.shardedCache.ReplicaFactor()
}

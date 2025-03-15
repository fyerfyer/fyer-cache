package cluster

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
)

// ClusterCache 将本地缓存与集群功能结合
type ClusterCache struct {
	// 内部缓存实现
	cache cache.Cache

	// 集群节点
	node *Node

	// 用于测试时等待操作传播的同步机制
	syncDelay time.Duration
}

// ClusterCacheOption 配置选项
type ClusterCacheOption func(*ClusterCache)

// WithSyncDelay 设置同步延迟，主要用于测试
func WithSyncDelay(delay time.Duration) ClusterCacheOption {
	return func(cc *ClusterCache) {
		cc.syncDelay = delay
	}
}

// NewClusterCache 创建新的集群缓存
func NewClusterCache(localCache cache.Cache, nodeID string, bindAddr string, options ...NodeOption) (*ClusterCache, error) {
	// 确保nodeID不为空
	if nodeID == "" {
		return nil, fmt.Errorf("node ID cannot be empty")
	}

	if localCache == nil {
		return nil, fmt.Errorf("cache cannot be nil")
	}

	// 创建节点选项
	nodeOptions := append([]NodeOption{
		WithNodeID(nodeID),
		WithBindAddr(bindAddr),
	}, options...)

	// 创建集群节点
	node, err := NewNode(nodeID, localCache, nodeOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster node: %w", err)
	}

	return &ClusterCache{
		cache:     localCache,
		node:      node,
		syncDelay: 100 * time.Millisecond, // 默认值，便于测试
	}, nil
}

// Start 启动集群缓存
func (cc *ClusterCache) Start() error {
	return cc.node.Start()
}

// Stop 停止集群缓存
func (cc *ClusterCache) Stop() error {
	return cc.node.Stop()
}

// Join 加入集群
func (cc *ClusterCache) Join(seedNodeAddr string) error {
	return cc.node.Join(seedNodeAddr)
}

// Leave 离开集群
func (cc *ClusterCache) Leave() error {
	return cc.node.Leave()
}

// Members 获取集群成员
func (cc *ClusterCache) Members() []cache.NodeInfo {
	return cc.node.Members()
}

// Events 获取事件通道
func (cc *ClusterCache) Events() <-chan cache.ClusterEvent {
	return cc.node.Events()
}

// IsCoordinator 判断当前节点是否为协调者节点
func (cc *ClusterCache) IsCoordinator() bool {
	return cc.node.IsCoordinator()
}

// SetLocalMetadata 设置本地节点的元数据信息
// 元数据可以包含节点的额外信息，如区域、数据中心、机器规格等
func (cc *ClusterCache) SetLocalMetadata(metadata map[string]string) {
	// 访问内部节点的成员管理器并更新元数据
	if cc.node != nil && cc.node.Membership() != nil {
		cc.node.Membership().SetLocalMetadata(metadata)
	}
}

// 广播操作到所有节点
func (cc *ClusterCache) broadcastToNodes(ctx context.Context, key string, value any, expiration time.Duration, opType string) error {
	members := cc.Members()

	// 对于每个节点，创建一个goroutine进行数据同步
	var wg sync.WaitGroup
	var errs []error
	//var errLock sync.Mutex

	for _, member := range members {
		// 跳过当前节点
		if member.ID == cc.node.id {
			continue
		}

		wg.Add(1)
		go func(memberID string, memberAddr string) {
			defer wg.Done()

			// 在实际实现中，这里应该通过集群通信同步数据
			// 为简化，这里我们仅记录操作类型
			if opType == "SET" {
				// 这里应该实现向远程节点发送SET请求
			} else if opType == "DEL" {
				// 这里应该实现向远程节点发送DEL请求
			}

			// 如果出错，添加到错误列表
			// 注意：在实际实现中，你可能想要更优雅地处理错误
			// 例如，忽略一些临时错误或实现重试机制
		}(member.ID, member.Addr)
	}

	wg.Wait()

	// 如果有错误，返回第一个错误
	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

// Cache接口实现

// Get 从缓存获取值
func (cc *ClusterCache) Get(ctx context.Context, key string) (any, error) {
	// 先尝试从本地缓存获取
	val, err := cc.cache.Get(ctx, key)
	if err == nil {
		return val, nil
	}

	// 在测试模式下，模拟同步延迟
	if cc.syncDelay > 0 {
		time.Sleep(cc.syncDelay)
	}

	// 如果本地缓存未找到，返回错误
	// 实际集群实现可能会尝试从其他节点获取
	return nil, err
}

// Set 设置缓存值
func (cc *ClusterCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
	// 首先在本地缓存设置
	err := cc.cache.Set(ctx, key, val, expiration)
	if err != nil {
		return err
	}

	// 在测试模式下，模拟同步延迟
	if cc.syncDelay > 0 {
		time.Sleep(cc.syncDelay)
	}

	// 广播到其他节点
	// 注意：在实际实现中，这可能是异步的并且有更复杂的错误处理
	go cc.broadcastToNodes(ctx, key, val, expiration, "SET")

	return nil
}

// Del 删除缓存值
func (cc *ClusterCache) Del(ctx context.Context, key string) error {
	// 在本地执行删除
	err := cc.cache.Del(ctx, key)
	if err != nil {
		return err
	}

	// 在测试模式下，模拟同步延迟
	if cc.syncDelay > 0 {
		time.Sleep(cc.syncDelay)
	}

	// 广播到其他节点
	go cc.broadcastToNodes(ctx, key, nil, 0, "DEL")

	return nil
}

// WithEventHandler 设置事件处理函数
// 这是一个辅助方法，使用户能够方便地处理集群事件
func (cc *ClusterCache) WithEventHandler(handler func(event cache.ClusterEvent)) {
	if handler == nil {
		return
	}

	// 获取事件通道
	eventCh := cc.Events()

	// 启动异步事件处理
	go func() {
		for event := range eventCh {
			handler(event)
		}
	}()
}

// GetClusterInfo 获取集群信息摘要
func (cc *ClusterCache) GetClusterInfo() map[string]interface{} {
	members := cc.Members()

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
		"node_id":        cc.node.id,
		"address":        cc.node.config.BindAddr,
		"is_coordinator": cc.IsCoordinator(),
		"member_count":   len(members),
		"status_counts": map[string]int{
			"up":      upCount,
			"suspect": suspectCount,
			"down":    downCount,
			"left":    leftCount,
		},
	}
}

// Membership 返回节点的成员管理器
func (cc *ClusterCache) Membership() *Membership {
	if cc.node != nil {
		return cc.node.Membership()
	}
	return nil
}

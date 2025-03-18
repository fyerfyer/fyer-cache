package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-cache/internal/ferr"
)

// NodeShardedCache 基于节点的分片缓存实现
type NodeShardedCache struct {
	// 一致性哈希环
	ring Ring

	// 节点ID到缓存的映射
	nodes map[string]Cache

	// 保护节点映射的并发访问
	mu sync.RWMutex

	// 备份因子，每个键存储在多少个节点上（用于容错）
	replicaFactor int
}

// NodeShardedOption 节点分片缓存选项
type NodeShardedOption func(*NodeShardedCache)

// WithReplicaFactor 设置备份因子
// 每个键将被复制到多少个节点（默认为1，表示没有备份）
func WithReplicaFactor(factor int) NodeShardedOption {
	return func(nsc *NodeShardedCache) {
		if factor > 0 {
			nsc.replicaFactor = factor
		}
	}
}

// WithHashReplicas 设置每个物理节点的虚拟节点数量
func WithHashReplicas(replicas int) NodeShardedOption {
	return func(nsc *NodeShardedCache) {
		ring, ok := nsc.ring.(*ConsistentHash)
		if ok && ring != nil {
			ring.replicas = replicas
		}
	}
}

// WithCustomHashFunc 设置自定义哈希函数
func WithCustomHashFunc(fn HashFunc) NodeShardedOption {
	return func(nsc *NodeShardedCache) {
		// 重新创建一致性哈希环，应用新的哈希函数
		oldRing := nsc.ring
		newRing := NewConsistentHash(defaultReplicas, fn)

		// 迁移所有节点
		for _, nodeID := range oldRing.GetNodes() {
			// 获取原节点权重
			weight := 1
			if ch, ok := oldRing.(*ConsistentHash); ok {
				if w, exists := ch.weights[nodeID]; exists {
					weight = w
				}
			}
			newRing.Add(nodeID, weight)
		}

		nsc.ring = newRing
	}
}

// NewNodeShardedCache 创建新的基于节点的分片缓存
func NewNodeShardedCache(options ...NodeShardedOption) *NodeShardedCache {
	nsc := &NodeShardedCache{
		ring:          NewConsistentHash(defaultReplicas, nil),
		nodes:         make(map[string]Cache),
		replicaFactor: 1, // 默认不备份
	}

	// 应用选项
	for _, opt := range options {
		opt(nsc)
	}

	return nsc
}

// AddNode 添加缓存节点
func (nsc *NodeShardedCache) AddNode(nodeID string, cache Cache, weight int) error {
	if cache == nil {
		return fmt.Errorf("cache cannot be nil")
	}

	nsc.mu.Lock()
	defer nsc.mu.Unlock()

	// 检查节点是否已存在
	if _, exists := nsc.nodes[nodeID]; exists {
		return fmt.Errorf("node already exists: %s", nodeID)
	}

	// 添加节点到一致性哈希环
	nsc.ring.Add(nodeID, weight)

	// 添加节点到节点映射
	nsc.nodes[nodeID] = cache

	return nil
}

// RemoveNode 移除缓存节点
func (nsc *NodeShardedCache) RemoveNode(nodeID string) error {
	nsc.mu.Lock()
	defer nsc.mu.Unlock()

	// 检查节点是否存在
	if _, exists := nsc.nodes[nodeID]; !exists {
		return fmt.Errorf("node does not exist: %s", nodeID)
	}

	// 从一致性哈希环中移除节点
	nsc.ring.Remove(nodeID)

	// 从节点映射中移除节点
	delete(nsc.nodes, nodeID)

	return nil
}

// Get 从正确的节点获取缓存项
func (nsc *NodeShardedCache) Get(ctx context.Context, key string) (any, error) {
	nsc.mu.RLock()

	// 检查是否有节点
	if len(nsc.nodes) == 0 {
		fmt.Printf("[NodeShardedCache.Get] No nodes available\n")
		nsc.mu.RUnlock()
		return nil, fmt.Errorf("no nodes available")
	}

	// 获取包含该键的所有节点
	var nodeIDs []string
	if nsc.replicaFactor > 1 {
		nodeIDs = nsc.ring.GetN(key, nsc.replicaFactor)
		fmt.Printf("[NodeShardedCache.Get] Getting key=%s from %d nodes: %v\n", key, nsc.replicaFactor, nodeIDs)
	} else {
		nodeIDs = []string{nsc.ring.Get(key)}
		fmt.Printf("[NodeShardedCache.Get] Getting key=%s from node: %s\n", key, nodeIDs[0])
	}

	// 复制必要的节点信息
	nodeCaches := make([]Cache, 0, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		if cache, exists := nsc.nodes[nodeID]; exists {
			nodeCaches = append(nodeCaches, cache)
			fmt.Printf("[NodeShardedCache.Get] Found cache for node=%s\n", nodeID)
		} else {
			fmt.Printf("[NodeShardedCache.Get] No cache found for node=%s\n", nodeID)
		}
	}

	nsc.mu.RUnlock()

	// 首先检查主节点
	if len(nodeCaches) == 0 {
		fmt.Printf("[NodeShardedCache.Get] No cache nodes found for key=%s\n", key)
		return nil, ferr.ErrKeyNotFound
	}

	fmt.Printf("[NodeShardedCache.Get] Trying primary node for key=%s\n", key)
	primaryValue, err := nodeCaches[0].Get(ctx, key)
	if err == nil {
		fmt.Printf("[NodeShardedCache.Get] Primary node returned value=%v\n", primaryValue)
		return primaryValue, nil
	} else {
		fmt.Printf("[NodeShardedCache.Get] Primary node failed with err=%v\n", err)
	}

	// 如果有备份节点，尝试从备份节点获取
	for i := 1; i < len(nodeCaches); i++ {
		fmt.Printf("[NodeShardedCache.Get] Trying backup node %d for key=%s\n", i, key)
		backupValue, err := nodeCaches[i].Get(ctx, key)
		if err == nil {
			fmt.Printf("[NodeShardedCache.Get] Backup node %d returned value=%v\n", i, backupValue)
			return backupValue, nil
		} else {
			fmt.Printf("[NodeShardedCache.Get] Backup node %d failed with err=%v\n", i, err)
		}
	}

	fmt.Printf("[NodeShardedCache.Get] All nodes failed for key=%s\n", key)
	return nil, ferr.ErrKeyNotFound
}

// Set 将缓存项设置到正确的节点
func (nsc *NodeShardedCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
	nsc.mu.RLock()
	defer nsc.mu.RUnlock()

	// 检查是否有节点
	if len(nsc.nodes) == 0 {
		fmt.Printf("[NodeShardedCache.Set] No nodes available\n")
		return fmt.Errorf("no nodes available")
	}

	// 获取应存储键的所有节点
	var nodeIDs []string
	if nsc.replicaFactor > 1 {
		nodeIDs = nsc.ring.GetN(key, nsc.replicaFactor)
		fmt.Printf("[NodeShardedCache.Set] Setting key=%s on %d nodes: %v\n", key, nsc.replicaFactor, nodeIDs)
	} else {
		nodeIDs = []string{nsc.ring.Get(key)}
		fmt.Printf("[NodeShardedCache.Set] Setting key=%s on node: %s\n", key, nodeIDs[0])
	}

	// 复制必要的节点信息，并跟踪每个节点
	nodeCaches := make([]struct {
		id    string
		cache Cache
	}, 0, len(nodeIDs))

	for _, nodeID := range nodeIDs {
		if cache, exists := nsc.nodes[nodeID]; exists {
			nodeCaches = append(nodeCaches, struct {
				id    string
				cache Cache
			}{id: nodeID, cache: cache})
			fmt.Printf("[NodeShardedCache.Set] Found cache for node=%s\n", nodeID)
		} else {
			fmt.Printf("[NodeShardedCache.Set] No cache found for node=%s\n", nodeID)
		}
	}

	if len(nodeCaches) == 0 {
		fmt.Printf("[NodeShardedCache.Set] No nodes available to set key=%s\n", key)
		return fmt.Errorf("no nodes available to set key %s", key)
	}

	// 在主节点设置值
	fmt.Printf("[NodeShardedCache.Set] Setting on primary node=%s, key=%s\n", nodeCaches[0].id, key)
	primaryErr := nodeCaches[0].cache.Set(ctx, key, val, expiration)
	if primaryErr != nil {
		fmt.Printf("[NodeShardedCache.Set] Primary node set failed: %v\n", primaryErr)
	} else {
		fmt.Printf("[NodeShardedCache.Set] Primary node set succeeded\n")
	}

	// 在备份节点设置值，并记录日志
	var backupErrors []error
	for i := 1; i < len(nodeCaches); i++ {
		fmt.Printf("[NodeShardedCache.Set] Setting on backup node=%s, key=%s\n", nodeCaches[i].id, key)
		err := nodeCaches[i].cache.Set(ctx, key, val, expiration)
		if err != nil {
			fmt.Printf("[NodeShardedCache.Set] Backup node set failed: %v\n", err)
			backupErrors = append(backupErrors, err)
		} else {
			fmt.Printf("[NodeShardedCache.Set] Backup node set succeeded\n")
		}
	}

	// 如果主节点失败，但有备份成功，认为操作成功
	if primaryErr != nil && len(backupErrors) < len(nodeCaches)-1 {
		fmt.Printf("[NodeShardedCache.Set] Primary failed but at least one backup succeeded\n")
		return nil
	}

	// 如果主节点失败，返回错误
	if primaryErr != nil {
		fmt.Printf("[NodeShardedCache.Set] Returning primary error\n")
		return primaryErr
	}

	// 即使备份失败，只要主节点成功，也认为操作成功
	return nil
}

// Del 从正确的节点删除缓存项
func (nsc *NodeShardedCache) Del(ctx context.Context, key string) error {
	nsc.mu.RLock()

	// 检查是否有节点
	if len(nsc.nodes) == 0 {
		nsc.mu.RUnlock()
		return nil // 没有节点，视为操作成功
	}

	// 获取可能包含该键的所有节点
	var nodeIDs []string
	if nsc.replicaFactor > 1 {
		nodeIDs = nsc.ring.GetN(key, nsc.replicaFactor)
	} else {
		nodeIDs = []string{nsc.ring.Get(key)}
	}

	// 复制必要的节点信息
	nodeCaches := make([]Cache, 0, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		if cache, exists := nsc.nodes[nodeID]; exists {
			nodeCaches = append(nodeCaches, cache)
		}
	}

	nsc.mu.RUnlock()

	// 从所有可能的节点删除
	var errs []error
	for _, cache := range nodeCaches {
		if err := cache.Del(ctx, key); err != nil {
			errs = append(errs, err)
		}
	}

	// 如果所有节点都失败，返回错误
	if len(errs) == len(nodeCaches) && len(nodeCaches) > 0 {
		return fmt.Errorf("failed to delete from all nodes")
	}

	// 否则视为操作成功
	return nil
}

// GetNodeIDs 获取所有节点ID
func (nsc *NodeShardedCache) GetNodeIDs() []string {
	nsc.mu.RLock()
	defer nsc.mu.RUnlock()

	return nsc.ring.GetNodes()
}

// GetNodeCount 获取节点数量
func (nsc *NodeShardedCache) GetNodeCount() int {
	nsc.mu.RLock()
	defer nsc.mu.RUnlock()

	return len(nsc.nodes)
}

// ReplicaFactor 获取当前的副本因子
func (nsc *NodeShardedCache) ReplicaFactor() int {
	return nsc.replicaFactor
}

// GetClusterInfo 获取集群信息
func (nsc *NodeShardedCache) GetClusterInfo() map[string]interface{} {
	nsc.mu.RLock()
	defer nsc.mu.RUnlock()

	info := make(map[string]interface{})

	// 基本信息
	info["node_count"] = len(nsc.nodes)
	info["replica_factor"] = nsc.replicaFactor

	// 节点列表
	nodeList := make([]string, 0, len(nsc.nodes))
	for nodeID := range nsc.nodes {
		nodeList = append(nodeList, nodeID)
	}
	info["nodes"] = nodeList

	return info
}
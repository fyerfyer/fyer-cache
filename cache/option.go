package cache

import (
	"time"
)

// Option 定义配置选项的函数类型
type Option func(*MemoryCache)

// WithEvictionCallback 设置驱逐回调函数
func WithEvictionCallback(callback func(key string, value any)) Option {
	return func(c *MemoryCache) {
		c.onEvict = callback
	}
}

// WithCleanupInterval 设置清理间隔
// 如果传入零值，则使用默认的清理间隔
func WithCleanupInterval(interval time.Duration) Option {
	return func(c *MemoryCache) {
		if interval > 0 {
			c.cleanupInterval = interval
		}
	}
}

// WithShardCount 设置分片数量
// 如果传入的值不是2的幂，将向上取整为2的幂
func WithShardCount(count int) Option {
	return func(c *MemoryCache) {
		c.shardCount = count
	}
}

// WithAsyncCleanup 启用异步清理
// 如果为true，将使用基于工作池的异步清理机制替代默认的同步清理
func WithAsyncCleanup(enable bool) Option {
	return func(c *MemoryCache) {
		c.useAsyncCleanup = enable
	}
}

// CacheWithWorkerCount 设置清理工作协程数量
// 只有在启用AsyncCleanup时生效
func CacheWithWorkerCount(count int) Option {
	return func(c *MemoryCache) {
		c.workerCount = count
	}
}

// CacheWithQueueSize 设置清理任务队列大小
// 只有在启用AsyncCleanup时生效
func CacheWithQueueSize(size int) Option {
	return func(c *MemoryCache) {
		c.queueSize = size
	}
}

// CleanerOption 清理器选项
type CleanerOption func(*Cleaner)

// WithCleanerWorkerCount 设置清理器的工作协程数量
func WithCleanerWorkerCount(count int) CleanerOption {
	return func(c *Cleaner) {
		if count > 0 {
			c.workerCount = count
		}
	}
}

// WithCleanerInterval 设置清理器的清理间隔
func WithCleanerInterval(interval time.Duration) CleanerOption {
	return func(c *Cleaner) {
		if interval > 0 {
			c.interval = interval
		}
	}
}

// WithCleanerQueueSize 设置清理器的任务队列大小
func WithCleanerQueueSize(size int) CleanerOption {
	return func(c *Cleaner) {
		if size > 0 {
			c.taskQueue = make(chan CleanTask, size)
		}
	}
}

// WithCleanerEvictionCallback 设置清理器的淘汰回调
func WithCleanerEvictionCallback(callback func(key string, value any)) CleanerOption {
	return func(c *Cleaner) {
		c.onEvict = callback
	}
}

// NodeShardedCacheOption 节点分片缓存选项
type NodeShardedCacheOption func(*NodeShardedCache)

// WithNodeHashReplicas 设置每个物理节点的虚拟节点数量
func WithNodeHashReplicas(replicas int) NodeShardedCacheOption {
	return func(nsc *NodeShardedCache) {
		if ring, ok := nsc.ring.(*ConsistentHash); ok && ring != nil {
			ring.replicas = replicas
		}
	}
}

// WithNodeCustomHashFunc 设置自定义哈希函数
func WithNodeCustomHashFunc(fn HashFunc) NodeShardedCacheOption {
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

// WithNodeReplicaFactor 设置备份因子
// 每个键将被复制到多少个节点（默认为1，表示没有备份）
func WithNodeReplicaFactor(factor int) NodeShardedCacheOption {
	return func(nsc *NodeShardedCache) {
		if factor > 0 {
			nsc.replicaFactor = factor
		}
	}
}

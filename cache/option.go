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

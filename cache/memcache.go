package cache

import (
	"context"
	"runtime"
	"time"

	"github.com/fyerfyer/fyer-cache/internal/ferr"
)

var cleanupInterval = 2 * time.Minute

// MemoryCache 内存缓存结构
type MemoryCache struct {
	data        *ShardedMap                 // 存储缓存数据，使用分片映射代替sync.Map
	onEvict     func(key string, value any) // 当缓存项被移除时的回调函数
	stopCleanUp chan struct{}               // 停止清理任务
	cleaner     *Cleaner                    // 异步清理器

	// 新增的配置字段
	shardCount      int           // 分片数量
	useAsyncCleanup bool          // 是否使用异步清理
	workerCount     int           // 工作协程数量
	queueSize       int           // 任务队列大小
	cleanupInterval time.Duration // 清理间隔
}

// cacheItem 缓存项结构
type cacheItem struct {
	value      any       // 实际存储的值
	expiration time.Time // 过期时间
}

// NewMemoryCache 创建新的内存缓存实例
func NewMemoryCache(options ...Option) *MemoryCache {
	cache := &MemoryCache{
		// 设置默认值
		shardCount:      DefaultShardCount,
		workerCount:     runtime.NumCPU(),
		queueSize:       DefaultQueueSize,
		useAsyncCleanup: false,
		cleanupInterval: cleanupInterval, // 添加默认清理间隔字段
	}

	// 应用选项模式配置（但不立即启动清理器）
	for _, opt := range options {
		opt(cache)
	}

	// 创建分片映射
	cache.data = NewShardedMap(cache.shardCount)

	// 初始化其他必要的资源
	cache.stopCleanUp = make(chan struct{})

	// 在所有必要的字段初始化后，再启动清理器
	if cache.useAsyncCleanup {
		cleanerOptions := []CleanerOption{
			WithCleanerWorkerCount(cache.workerCount),
			WithCleanerQueueSize(cache.queueSize),
			WithCleanerEvictionCallback(cache.onEvict),
		}

		// 如果设置了自定义清理间隔
		if cache.cleanupInterval != cleanupInterval {
			cleanerOptions = append(cleanerOptions,
				WithCleanerInterval(cache.cleanupInterval))
		}

		cache.cleaner = NewCleaner(cache.data, cleanerOptions...)
		cache.cleaner.Start()
	} else if cache.cleanupInterval > 0 {
		// 使用传统定时器清理
		cache.startSyncCleanupTimer(cache.cleanupInterval)
	}

	return cache
}

// Set 实现缓存接口的Set方法
func (c *MemoryCache) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	item := &cacheItem{
		value:      value,
		expiration: time.Now().Add(expiration),
	}
	c.data.Store(key, item)
	return nil
}

// Get 实现缓存接口的Get方法
func (c *MemoryCache) Get(ctx context.Context, key string) (any, error) {
	value, ok := c.data.Load(key)
	if !ok {
		return nil, ferr.ErrKeyNotFound
	}

	item := value
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		c.data.Delete(key)
		if c.onEvict != nil {
			c.onEvict(key, item.value)
		}
		return nil, ferr.ErrKeyNotFound
	}

	return item.value, nil
}

// Del 实现缓存接口的Delete方法
func (c *MemoryCache) Del(ctx context.Context, key string) error {
	if value, loaded := c.data.LoadAndDelete(key); loaded {
		if c.onEvict != nil {
			c.onEvict(key, value.value)
		}
	}
	return nil
}

// startCleanUpTimer 启动定时清理任务
func (c *MemoryCache) startCleanUpTimer(interval time.Duration) {
	if interval <= 0 {
		interval = cleanupInterval
	}

	// 如果已经使用异步清理器，则更新其配置
	if c.useAsyncCleanup && c.cleaner != nil {
		c.cleaner.interval = interval
		return
	}

	// 如果配置使用异步清理但尚未创建清理器
	if c.useAsyncCleanup {
		cleanerOptions := []CleanerOption{
			WithCleanerWorkerCount(c.workerCount),
			WithCleanerInterval(interval),
			WithCleanerQueueSize(c.queueSize),
			WithCleanerEvictionCallback(c.onEvict),
		}

		c.cleaner = NewCleaner(c.data, cleanerOptions...)
		c.cleaner.Start()
		return
	}

	// 否则使用传统的定时器清理方式
	ticker := time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-ticker.C:
				c.cleanUp()
			case <-c.stopCleanUp:
				ticker.Stop()
				return
			}
		}
	}()
}

// startSyncCleanupTimer 启动同步定时清理任务
func (c *MemoryCache) startSyncCleanupTimer(interval time.Duration) {
	ticker := time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-ticker.C:
				c.cleanUp()
			case <-c.stopCleanUp:
				ticker.Stop()
				return
			}
		}
	}()
}

// cleanUp 清理过期缓存项
func (c *MemoryCache) cleanUp() {
	now := time.Now()

	// 使用ShardedMap的Range方法遍历所有项
	c.data.Range(func(key string, value *cacheItem) bool {
		if !value.expiration.IsZero() && now.After(value.expiration) {
			c.data.Delete(key)
			if c.onEvict != nil {
				c.onEvict(key, value.value)
			}
		}
		return true
	})
}

// Close 关闭内存缓存使用的通道
func (c *MemoryCache) Close() error {
	// 停止传统清理
	if c.stopCleanUp != nil {
		close(c.stopCleanUp)
	}

	// 停止异步清理器
	if c.cleaner != nil {
		c.cleaner.Stop()
	}

	return nil
}

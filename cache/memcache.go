package cache

import (
	"context"
	"github.com/fyerfyer/fyer-cache/internal/ferr"
	"sync"
	"time"
)

var cleanupInterval = 2 * time.Minute

// MemoryCache 内存缓存结构
type MemoryCache struct {
	data        sync.Map                    // 存储缓存数据
	onEvict     func(key string, value any) // 当缓存项被移除时的回调函数
	ticker      *time.Ticker                // 定时清理任务
	stopCleanUp chan struct{}               // 停止清理任务
}

// cacheItem 缓存项结构
type cacheItem struct {
	value      any       // 实际存储的值
	expiration time.Time // 过期时间
}

// NewMemoryCache 创建新的内存缓存实例
func NewMemoryCache(options ...Option) *MemoryCache {
	cache := &MemoryCache{}
	// 应用选项模式配置
	for _, opt := range options {
		opt(cache)
	}
	return cache
}

// Option 定义配置选项的函数类型
type Option func(*MemoryCache)

// WithEvictionCallback 设置驱逐回调函数
func WithEvictionCallback(callback func(key string, value any)) Option {
	return func(c *MemoryCache) {
		c.onEvict = callback
	}
}

// WithCleanupInterval 设置清理间隔并调用自动清理函数
// 如果传入零值，则使用默认的清理间隔
func WithCleanupInterval(interval time.Duration) Option {
	return func(c *MemoryCache) {
		c.startCleanUpTimer(interval)
	}
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

	item := value.(*cacheItem)
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
			item := value.(*cacheItem)
			c.onEvict(key, item.value)
		}
	}
	return nil
}

// startCleanUpTimer 启动定时清理任务
func (c *MemoryCache) startCleanUpTimer(internal time.Duration) {
	if internal <= 0 {
		internal = cleanupInterval
	}

	// 初始化清理相关
	c.stopCleanUp = make(chan struct{})
	c.ticker = time.NewTicker(internal)

	go func() {
		for {
			select {
			case <-c.ticker.C:
				c.cleanUp()
			case <-c.stopCleanUp:
				c.ticker.Stop()
				return
			}
		}
	}()
}

// cleanUp 清理过期缓存项
func (c *MemoryCache) cleanUp() {
	now := time.Now()
	c.data.Range(func(key, value any) bool {
		item := value.(*cacheItem)
		if !item.expiration.IsZero() && now.After(item.expiration) {
			if _, loaded := c.data.LoadAndDelete(key); loaded {
				if c.onEvict != nil {
					c.onEvict(key.(string), item.value)
				}
			}
		}

		return true
	})
}

// Close 关闭内存缓存使用的通道
func (c *MemoryCache) Close() error {
	if c.stopCleanUp != nil {
		close(c.stopCleanUp)
	}

	return nil
}

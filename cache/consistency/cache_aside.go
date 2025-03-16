package consistency

import (
	"context"
	"fmt"
	"github.com/fyerfyer/fyer-cache/cache"
	"time"

	"github.com/fyerfyer/fyer-cache/internal/ferr"
)

// CacheAsideStrategy 实现 Cache Aside 一致性模式
// 先查询缓存，缓存不命中时从数据源获取，并写回缓存
type CacheAsideStrategy struct {
	cache      cache.Cache // 缓存实例
	dataSource DataSource  // 后端数据源
	options    *CacheAsideOptions
}

// NewCacheAsideStrategy 创建Cache Aside策略实例
func NewCacheAsideStrategy(cache cache.Cache, dataSource DataSource, opts ...Option) *CacheAsideStrategy {
	options := defaultCacheAsideOptions()

	// 应用选项
	for _, opt := range opts {
		opt(options)
	}

	return &CacheAsideStrategy{
		cache:      cache,
		dataSource: dataSource,
		options:    options,
	}
}

// Get 从缓存获取数据，缓存未命中时从数据源获取
func (c *CacheAsideStrategy) Get(ctx context.Context, key string) (any, error) {
	// 首先尝试从缓存获取
	value, err := c.cache.Get(ctx, key)
	if err == nil {
		// 缓存命中，直接返回
		return value, nil
	}

	// 如果错误不是"键不存在"，则返回错误
	if err != ferr.ErrKeyNotFound {
		return nil, err
	}

	// 从数据源获取
	value, err = c.dataSource.Load(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to load from data source: %w", err)
	}

	// 如果配置了缓存未命中时写入，将数据写入缓存
	if c.options.WriteOnMiss {
		// 使用默认过期时间写入缓存
		// 这里不处理写入缓存的错误，即使写入失败也返回查询结果
		_ = c.cache.Set(ctx, key, value, c.options.DefaultTTL)
	}

	return value, nil
}

// Set 设置缓存数据并同步到数据源
// 在Cache Aside模式中，先更新数据源再更新缓存
func (c *CacheAsideStrategy) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	// 先更新数据源
	if err := c.dataSource.Store(ctx, key, value); err != nil {
		return fmt.Errorf("failed to store in data source: %w", err)
	}

	// 再更新缓存
	if err := c.cache.Set(ctx, key, value, expiration); err != nil {
		// 即使缓存更新失败，数据源已更新成功，我们仍认为操作成功
		// 但记录错误
		return fmt.Errorf("data source updated but cache set failed: %w", err)
	}

	return nil
}

// Del 删除缓存和数据源中的数据
// 在Cache Aside模式中，先删除缓存再删除数据源
func (c *CacheAsideStrategy) Del(ctx context.Context, key string) error {
	// 先删除缓存
	_ = c.cache.Del(ctx, key)

	// 再删除数据源
	if err := c.dataSource.Remove(ctx, key); err != nil {
		return fmt.Errorf("failed to remove from data source: %w", err)
	}

	return nil
}

// Invalidate 使缓存失效但不影响数据源
func (c *CacheAsideStrategy) Invalidate(ctx context.Context, key string) error {
	return c.cache.Del(ctx, key)
}

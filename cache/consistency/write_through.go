package consistency

import (
	"context"
	"fmt"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/fyerfyer/fyer-cache/internal/ferr"
)

// WriteThroughStrategy 实现 Write Through 一致性模式
// 写操作同时更新缓存和数据源，确保缓存与数据源保持一致
type WriteThroughStrategy struct {
	cache      cache.Cache // 缓存实例
	dataSource DataSource  // 后端数据源
	options    *WriteThroughOptions
}

// NewWriteThroughStrategy 创建 Write Through 策略实例
func NewWriteThroughStrategy(cache cache.Cache, dataSource DataSource, opts ...Option) *WriteThroughStrategy {
	options := defaultWriteThroughOptions()

	// 应用选项
	for _, opt := range opts {
		opt(options)
	}

	return &WriteThroughStrategy{
		cache:      cache,
		dataSource: dataSource,
		options:    options,
	}
}

// Get 从缓存获取数据，缓存未命中时从数据源获取
func (w *WriteThroughStrategy) Get(ctx context.Context, key string) (any, error) {
	// 尝试从缓存获取
	value, err := w.cache.Get(ctx, key)
	if err == nil {
		// 缓存命中
		return value, nil
	}

	// 如果错误不是"键不存在"，返回错误
	if err != ferr.ErrKeyNotFound {
		return nil, err
	}

	// 从数据源获取
	value, err = w.dataSource.Load(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to load from data source: %w", err)
	}

	// 写入缓存
	// 使用默认过期时间
	// 这里不处理写入缓存的错误，即使写入失败也返回查询结果
	_ = w.cache.Set(ctx, key, value, w.options.DefaultTTL)

	return value, nil
}

// Set 设置缓存数据并同步到数据源
// 在Write Through模式中，同时更新数据源和缓存
func (w *WriteThroughStrategy) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	// 如果启用了延迟写入
	if w.options.DelayedWrite {
		// 先更新缓存
		if err := w.cache.Set(ctx, key, value, expiration); err != nil {
			return fmt.Errorf("failed to set cache: %w", err)
		}

		// 异步更新数据源
		go func() {
			// 创建新的上下文，避免使用可能已取消的上下文
			newCtx, cancel := context.WithTimeout(context.Background(), w.options.WriteDelay*2)
			defer cancel()

			// 延迟一段时间再写入数据源
			time.Sleep(w.options.WriteDelay)

			// 更新数据源
			if err := w.dataSource.Store(newCtx, key, value); err != nil {
				// 这里只能记录错误，无法返回
				// 实际应用中应该有日志或监控机制
				// log.Printf("Error writing to data source: %v", err)
			}
		}()

		return nil
	}

	// 标准Write Through：同时更新数据源和缓存
	// 先更新数据源
	if err := w.dataSource.Store(ctx, key, value); err != nil {
		return fmt.Errorf("failed to store in data source: %w", err)
	}

	// 再更新缓存
	if err := w.cache.Set(ctx, key, value, expiration); err != nil {
		// 数据源已更新，但缓存更新失败
		// 此时需要移除缓存项以避免不一致
		_ = w.cache.Del(ctx, key)
		return fmt.Errorf("data source updated but cache set failed: %w", err)
	}

	return nil
}

// Del 删除缓存和数据源中的数据
func (w *WriteThroughStrategy) Del(ctx context.Context, key string) error {
	// 在Write Through模式中，先删除数据源再删除缓存

	// 先删除数据源
	if err := w.dataSource.Remove(ctx, key); err != nil {
		return fmt.Errorf("failed to remove from data source: %w", err)
	}

	// 再删除缓存
	// 即使缓存删除失败也不影响操作结果
	_ = w.cache.Del(ctx, key)

	return nil
}

// Invalidate 使缓存失效但不影响数据源
func (w *WriteThroughStrategy) Invalidate(ctx context.Context, key string) error {
	return w.cache.Del(ctx, key)
}

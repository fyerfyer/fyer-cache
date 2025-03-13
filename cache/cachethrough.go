package cache

import (
	"context"
	"errors"
	"github.com/fyerfyer/fyer-cache/internal/ferr"
	"log"
	"time"
)

var defaultTTL = 5 * time.Minute

// ThroughCache 实现读写穿透的缓存装饰器
type ThroughCache struct {
	Cache      Cache
	dataSource DataSource
	defaultTTL time.Duration
}

type ThroughCacheOpt func(*ThroughCache)

func ThroughCacheWithTTL(ttl time.Duration) ThroughCacheOpt {
	return func(tc *ThroughCache) {
		tc.defaultTTL = ttl
	}
}

// NewThroughCache 创建一个读写穿透的缓存装饰器
func NewThroughCache(cache Cache, dataSource DataSource, opts ...ThroughCacheOpt) *ThroughCache {
	tc := &ThroughCache{
		Cache:      cache,
		dataSource: dataSource,
		defaultTTL: defaultTTL,
	}

	for _, opt := range opts {
		opt(tc)
	}

	return tc
}

// Get 从缓存中获取数据
// 如果缓存中不存在，则从数据源中获取，读取后写回缓存
func (tc *ThroughCache) Get(ctx context.Context, key string) (any, error) {
	val, err := tc.Cache.Get(ctx, key)
	if err != nil {
		if errors.Is(err, ferr.ErrKeyNotFound) {
			val, err = tc.dataSource.Load(ctx, key)
			if err != nil {
				return nil, err
			}

			go func() {
				err = tc.Cache.Set(ctx, key, val, tc.defaultTTL)
				// 这里只记录错误日志，不返回错误
				if err != nil {
					log.Fatalf("failed to write from data source to cache: %v", err)
				}
			}()
		} else {
			return nil, err
		}
	}

	return val, nil
}

// Set 写入缓存
// 首先写入数据源，然后写入缓存
func (tc *ThroughCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
	if err := tc.dataSource.Store(ctx, key, val); err != nil {
		return err
	}

	return tc.Cache.Set(ctx, key, val, expiration)
}

// Del 删除缓存
// 同样为了保证一致性，在数据源和缓存层都执行删除
func (tc *ThroughCache) Del(ctx context.Context, key string) error {
	if err := tc.dataSource.Remove(ctx, key); err != nil {
		return err
	}

	return tc.Cache.Del(ctx, key)
}

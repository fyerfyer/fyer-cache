package distributelock

import (
	"context"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// SingleFlightRedisLock 基于SingleFlight的分布式锁
type SingleFlightRedisLock struct {
	*RedisLock
	sg *singleflight.Group
}

// NewSingleFlightRedisLock 创建SingleFlightRedisLock实例
func NewSingleFlightRedisLock(client *redis.Client, key string, opts ...Option) *SingleFlightRedisLock {
	return &SingleFlightRedisLock{
		RedisLock: NewRedisLock(client, key, opts...),
		sg:        new(singleflight.Group),
	}
}

// Lock 重写RedisLock Lock方法
func (l *SingleFlightRedisLock) Lock(ctx context.Context) error {
	_, err, _ := l.sg.Do(l.key, func() (interface{}, error) {
		return nil, l.RedisLock.Lock(ctx)
	})
	return err
}

// TryLock 重写RedisLock TryLock方法
func (l *SingleFlightRedisLock) TryLock(ctx context.Context) error {
	_, err, _ := l.sg.Do(l.key, func() (interface{}, error) {
		return nil, l.RedisLock.TryLock(ctx)
	})
	return err
}

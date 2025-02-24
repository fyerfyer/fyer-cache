package distributelock

import (
	"context"
	"time"

	"github.com/fyerfyer/fyer-cache/internal/ferr"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RedisLock 基于Redis的分布式锁
type RedisLock struct {
	client   redis.Cmdable
	key      string
	val      string
	options  *LockOption
	watchdog *WatchDog
	// isLocker 标记锁的持有状态
	// 防止重复加锁或未加锁就解锁，以及在续约失败时能正确更新状态
	isLocked bool
}

// NewRedisLock 创建RedisLock实例
func NewRedisLock(client redis.Cmdable, key string, opts ...Option) *RedisLock {
	options := DefaultOption()
	for _, opt := range opts {
		opt(options)
	}

	return &RedisLock{
		client:  client,
		key:     key,
		val:     uuid.New().String(),
		options: options,
	}
}

// Lock 获取锁
func (l *RedisLock) Lock(ctx context.Context) error {
	// 如果已经持有锁，直接返回错误
	if l.isLocked {
		return ferr.ErrLockAlreadyHeld
	}

	retryCnt := 0
	for {
		// 尝试获取锁
		success, err := l.tryAcquire(ctx)
		if err != nil {
			return err
		}

		if success {
			// 如果开启了watchdog，则用watchdog续约
			l.isLocked = true
			if l.options.EnableWatchdog {
				l.watchdog = NewWatchDog(l)
				l.watchdog.Start()
			}
		}

		// 检查重试次数
		if retryCnt > l.options.RetryTimes {
			return ferr.ErrLockAcquireFailed
		}

		// 计算下一次重试时间
		var waitTime time.Duration
		if l.options.BackoffStrategy != nil {
			waitTime = l.options.BackoffStrategy.NextBackoff(retryCnt)
		} else {
			waitTime = l.options.RetryInterval
		}

		// 如果不需要阻塞等待，直接返回error
		if !l.options.BlockWaiting {
			return ferr.ErrLockAcquireFailed
		}

		// 等待重试，直到ctx超时
		select {
		case <-ctx.Done():
			return ctx.Err()

		// 当到了重试的时间间隔，重新重试
		case <-time.After(waitTime):
			retryCnt++
			continue
		}
	}
}

// TryLock 尝试获取锁
// 获取不到锁的话立即返回错误
func (l *RedisLock) TryLock(ctx context.Context) error {
	if l.isLocked {
		return ferr.ErrLockAlreadyHeld
	}

	success, err := l.tryAcquire(ctx)
	if err != nil {
		return err
	}

	if !success {
		return ferr.ErrLockAcquireFailed
	}

	l.isLocked = true
	if l.options.EnableWatchdog {
		l.watchdog = NewWatchDog(l, WithInterval(500*time.Millisecond))
		l.watchdog.Start()
	}

	return nil
}

// tryAcquire 尝试获取锁
// 使用lua脚本保证redis操作的原子性
func (l *RedisLock) tryAcquire(ctx context.Context) (bool, error) {
	expireMS := int64(l.options.Expiration / time.Millisecond)
	// 修改Result()为Int64()
	res, err := l.client.Eval(ctx, lockScript, []string{l.key}, l.val, expireMS).Int64()
	if err != nil {
		return false, err
	}

	return res == 1, nil
}

// Unlock 释放锁
func (l *RedisLock) Unlock(ctx context.Context) error {
	if !l.isLocked {
		return ferr.ErrLockNotHeld
	}

	// 如果使用watchdog的话，停止使用
	if l.watchdog != nil {
		l.watchdog.Stop()
		l.watchdog = nil
	}

	// 释放锁
	// 修改Result()为Int64()
	res, err := l.client.Eval(ctx, unlockScript, []string{l.key}, l.val).Int64()
	if err != nil {
		return err
	}

	l.isLocked = false
	if res == 0 {
		return ferr.ErrLockNotHeld
	}

	return nil
}

// Refresh 手动续约
func (l *RedisLock) Refresh(ctx context.Context) error {
	if !l.isLocked {
		return ferr.ErrLockNotHeld
	}

	expireMS := int64(l.options.Expiration / time.Millisecond)
	// 修改Result()为Int64()
	res, err := l.client.Eval(ctx, refreshScript, []string{l.key}, l.val, expireMS).Int64()
	if err != nil {
		return err
	}

	if res == 0 {
		l.isLocked = false // 续约失败，释放锁
		return ferr.ErrLockNotHeld
	}

	return nil
}

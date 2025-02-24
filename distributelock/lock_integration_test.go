package distributelock

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRedisConnection 测试Redis连接
func TestRedisConnection(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	ctx := context.Background()

	// 测试连接
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}

	// 测试基本操作
	err := rdb.Set(ctx, "test_key", "test_value", 5*time.Second).Err()
	if err != nil {
		t.Fatalf("Failed to set key: %v", err)
	}

	val, err := rdb.Get(ctx, "test_key").Result()
	if err != nil {
		t.Fatalf("Failed to get key: %v", err)
	}

	t.Logf("Successfully connected to Redis and performed test operations. Value: %s", val)
}

// TestRedisLock_Concurrent 测试RedisLock的并发场景
func TestRedisLock_Concurrent(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	// 验证Redis连接
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}

	testCases := []struct {
		name          string
		goroutines    int
		successCount  int
		lockDuration  time.Duration
		testDuration  time.Duration
		enableRefresh bool
	}{
		{
			name:          "basic concurrent test",
			goroutines:    10,
			successCount:  1,
			lockDuration:  1 * time.Second,
			testDuration:  3 * time.Second,
			enableRefresh: false,
		},
		{
			name:          "concurrent with refresh",
			goroutines:    5,
			successCount:  1,
			lockDuration:  500 * time.Millisecond,
			testDuration:  2 * time.Second,
			enableRefresh: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 清理测试key
			rdb.Del(ctx, "test-concurrent")

			var wg sync.WaitGroup
			successCount := 0
			var mu sync.Mutex

			// 启动多个goroutine竞争锁
			for i := 0; i < tc.goroutines; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()

					lock := NewRedisLock(rdb, "test-concurrent",
						WithExpiration(tc.lockDuration),
						WithWatchdog(tc.enableRefresh),
						WithBlockWaiting(false)) // 使用TryLock，不阻塞等待

					// 尝试获取锁
					err := lock.TryLock(ctx)
					if err != nil {
						t.Logf("Goroutine %d failed to acquire lock: %v", id, err)
						return
					}

					// 成功获取到锁
					mu.Lock()
					successCount++
					mu.Unlock()

					t.Logf("Goroutine %d acquired lock successfully", id)

					// 模拟业务处理
					time.Sleep(tc.lockDuration / 2)

					// 释放锁
					if err := lock.Unlock(ctx); err != nil {
						t.Errorf("Goroutine %d failed to release lock: %v", id, err)
					} else {
						t.Logf("Goroutine %d released lock successfully", id)
					}
				}(i)
			}

			// 等待所有goroutine完成
			wg.Wait()

			// 验证结果
			mu.Lock()
			actualCount := successCount
			mu.Unlock()

			if actualCount != tc.successCount {
				t.Errorf("Expected success count: %d, got: %d", tc.successCount, actualCount)
			}
		})
	}
}

// TestRedisLock_Scenarios 测试RedisLock的各种场景
// 注意：如果watchdog的查询间隔太长的话test会不通过
func TestRedisLock_Scenarios(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	// 清理测试key
	defer rdb.Del(context.Background(), "test-scenario")

	t.Run("lock ownership transfer", func(t *testing.T) {
		ctx := context.Background()

		// 第一个客户端获取锁
		lock1 := NewRedisLock(rdb, "test-scenario",
			WithExpiration(5*time.Second),
			WithWatchdog(false))

		err := lock1.TryLock(ctx)
		require.NoError(t, err)

		// 第二个客户端尝试获取同一个锁
		lock2 := NewRedisLock(rdb, "test-scenario",
			WithExpiration(5*time.Second))

		err = lock2.TryLock(ctx)
		assert.Error(t, err) // 应该获取失败

		// 第一个客户端释放锁
		err = lock1.Unlock(ctx)
		require.NoError(t, err)

		// 第二个客户端现在应该能获取到锁
		err = lock2.TryLock(ctx)
		assert.NoError(t, err)

		// 清理
		_ = lock2.Unlock(ctx)
	})

	t.Run("watchdog auto refresh", func(t *testing.T) {
		ctx := context.Background()

		// 使用更短的过期时间
		lock := NewRedisLock(rdb, "test-scenario",
			WithExpiration(1*time.Second), // 改为1秒
			WithWatchdog(true))

		err := lock.TryLock(ctx)
		require.NoError(t, err)

		// 验证初始TTL
		initialTTL, err := rdb.TTL(ctx, "test-scenario").Result()
		require.NoError(t, err)
		assert.True(t, initialTTL > 0, "Initial TTL should be positive")
		t.Logf("Initial TTL: %v", initialTTL)

		// 等待略长于过期时间
		time.Sleep(1500 * time.Millisecond) // 等待1.5秒

		// 验证watchdog是否成功续约
		currentTTL, err := rdb.TTL(ctx, "test-scenario").Result()
		require.NoError(t, err)
		assert.True(t, currentTTL > 0, "Lock should still be valid")
		t.Logf("Current TTL after wait: %v", currentTTL)

		// 清理
		err = lock.Unlock(ctx)
		require.NoError(t, err)

		// 验证锁已被释放
		finalTTL, err := rdb.TTL(ctx, "test-scenario").Result()
		require.NoError(t, err)
		assert.True(t, finalTTL < 0, "Lock should be released")
		t.Logf("Final TTL after unlock: %v", finalTTL)
	})

	t.Run("lock reentrant prevention", func(t *testing.T) {
		ctx := context.Background()

		lock := NewRedisLock(rdb, "test-scenario",
			WithExpiration(5*time.Second))

		// 第一次加锁
		err := lock.TryLock(ctx)
		require.NoError(t, err)

		// 同一个实例重复加锁
		err = lock.TryLock(ctx)
		assert.Error(t, err) // 应该返回错误

		// 清理
		_ = lock.Unlock(ctx)
	})
}

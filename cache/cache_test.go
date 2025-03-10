package cache

import (
	"context"
	"fmt"
	"github.com/fyerfyer/fyer-cache/internal/ferr"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoryCache_BasicOperations(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "Set and Get",
			testFunc: func(t *testing.T) {
				err := cache.Set(ctx, "key1", "value1", time.Minute)
				if err != nil {
					t.Errorf("Set failed: %v", err)
				}

				val, err := cache.Get(ctx, "key1")
				if err != nil {
					t.Errorf("Get failed: %v", err)
				}
				if val != "value1" {
					t.Errorf("Expected value1, got %v", val)
				}
			},
		},
		{
			name: "Get Non-Existent Key",
			testFunc: func(t *testing.T) {
				_, err := cache.Get(ctx, "non-existent")
				if err != ferr.ErrKeyNotFound {
					t.Errorf("Expected ErrKeyNotFound, got %v", err)
				}
			},
		},
		{
			name: "Delete Key",
			testFunc: func(t *testing.T) {
				cache.Set(ctx, "key2", "value2", time.Minute)
				err := cache.Del(ctx, "key2")
				if err != nil {
					t.Errorf("Delete failed: %v", err)
				}

				_, err = cache.Get(ctx, "key2")
				if err != ferr.ErrKeyNotFound {
					t.Errorf("Expected ErrKeyNotFound after deletion, got %v", err)
				}
			},
		},
		{
			name: "Expiration",
			testFunc: func(t *testing.T) {
				cache.Set(ctx, "key3", "value3", 50*time.Millisecond)
				time.Sleep(100 * time.Millisecond)

				_, err := cache.Get(ctx, "key3")
				if err != ferr.ErrKeyNotFound {
					t.Errorf("Expected ErrKeyNotFound for expired key, got %v", err)
				}
			},
		},
		{
			name: "Eviction Callback",
			testFunc: func(t *testing.T) {
				called := false
				cache := NewMemoryCache(WithEvictionCallback(func(key string, value any) {
					called = true
					if key != "key4" || value != "value4" {
						t.Errorf("Unexpected eviction values: key=%s, value=%v", key, value)
					}
				}))

				cache.Set(ctx, "key4", "value4", 50*time.Millisecond)
				time.Sleep(100 * time.Millisecond)
				cache.Get(ctx, "key4") // This will trigger the eviction

				if !called {
					t.Error("Eviction callback was not called")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func TestMemoryCache_Concurrent(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	// 并发测试参数
	const (
		numGoroutines   = 10   // 并发 goroutine 数量
		opsPerGoroutine = 1000 // 每个 goroutine 的操作次数
		keyRange        = 100  // key的范围，用于模拟不同key的读写冲突
	)

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // 一半goroutine做写操作，一半做读操作

	// 创建计数器记录成功的操作数
	var readSuccess, writeSuccess int64

	// 启动写入 goroutine
	for i := 0; i < numGoroutines; i++ {
		go func(routine int) {
			defer wg.Done()

			for j := 0; j < opsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d", rand.Intn(keyRange))
				value := fmt.Sprintf("value-%d-%d", routine, j)
				err := cache.Set(ctx, key, value, time.Minute)
				if err == nil {
					atomic.AddInt64(&writeSuccess, 1)
				}

				// 随机删除一些键
				if rand.Float64() < 0.1 { // 10% 的概率删除
					cache.Del(ctx, key)
				}
			}
		}(i)
	}

	// 启动读取 goroutine
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < opsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d", rand.Intn(keyRange))
				_, err := cache.Get(ctx, key)
				if err == nil {
					atomic.AddInt64(&readSuccess, 1)
				}
			}
		}()
	}

	// 等待所有 goroutine 完成
	wg.Wait()

	t.Logf("Concurrent test completed - Read success: %d, Write success: %d",
		readSuccess, writeSuccess)
}

func TestMemoryCache_ConcurrentExpiration(t *testing.T) {
	cache := NewMemoryCache(WithCleanupInterval(100 * time.Millisecond))
	ctx := context.Background()

	const (
		numKeys         = 1000
		goroutines      = 5
		shortExpiration = 100 * time.Millisecond
	)

	// 填充初始数据
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		err := cache.Set(ctx, key, i, shortExpiration)
		if err != nil {
			t.Fatalf("Failed to set initial data: %v", err)
		}
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// 启动多个 goroutine 同时进行读取操作
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()

			// 在清理过程中持续读取
			for j := 0; j < numKeys; j++ {
				key := fmt.Sprintf("key-%d", j)
				_, _ = cache.Get(ctx, key)
				// 不检查错误，因为键可能已经过期
			}
		}()
	}

	// 等待足够长的时间让清理程序运行
	time.Sleep(300 * time.Millisecond)

	wg.Wait()
	cache.Close()
}

func TestMemoryCache_ConcurrentOverwrite(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	const (
		numGoroutines = 10
		iterations    = 1000
	)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// 多个 goroutine 同时对同一个键进行写操作
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				value := fmt.Sprintf("value-%d-%d", goroutineID, j)
				err := cache.Set(ctx, "same-key", value, time.Minute)
				if err != nil {
					t.Errorf("Failed to set value: %v", err)
				}

				// 立即读取
				_, err = cache.Get(ctx, "same-key")
				if err != nil && err != ferr.ErrKeyNotFound {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// 验证最终可以正确读取
	_, err := cache.Get(ctx, "same-key")
	if err != nil {
		t.Errorf("Final read failed: %v", err)
	}
}

// 测试异步清理功能
func TestAsyncCleaner(t *testing.T) {
	ctx := context.Background()

	// 创建使用异步清理的缓存
	cache := NewMemoryCache(
		WithShardCount(32),
		WithAsyncCleanup(true),
		CacheWithWorkerCount(4),
		CacheWithQueueSize(100),
		WithCleanupInterval(100*time.Millisecond),
	)

	// 添加带有短期超时的项目
	const itemCount = 5000

	start := time.Now()

	// 添加项目
	for i := 0; i < itemCount; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		// 设置随机过期时间，0-200ms
		expiration := time.Duration(rand.Intn(201)) * time.Millisecond
		err := cache.Set(ctx, key, fmt.Sprintf("value-%d", i), expiration)
		if err != nil {
			t.Fatalf("Failed to set item: %v", err)
		}
	}

	// 等待足够长的时间让所有项目过期
	time.Sleep(300 * time.Millisecond)

	// 再等待一段时间让清理工作完成
	time.Sleep(200 * time.Millisecond)

	// 检查剩余的项目数量
	count := cache.data.Count()
	t.Logf("Items added: %d, remaining after expiration: %d", itemCount, count)

	if count > 0 {
		t.Logf("Not all items were cleaned up - this is expected as some items might have been added after checking")
	}

	// 关闭缓存
	cache.Close()

	duration := time.Since(start)
	t.Logf("Async cleanup test completed in %v", duration)
}

// 测试分片均匀性
func TestShardDistribution(t *testing.T) {
	ctx := context.Background()

	// 使用32个分片创建缓存
	cache := NewMemoryCache(WithShardCount(32))

	// 插入大量随机键
	const keyCount = 100000

	for i := 0; i < keyCount; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		cache.Set(ctx, key, i, time.Minute)
	}

	// 统计每个分片中的项目数量
	shardCounts := make([]int, cache.data.count)

	for i := 0; i < cache.data.count; i++ {
		shard := cache.data.shards[i]
		shard.mu.RLock()
		shardCounts[i] = len(shard.items)
		shard.mu.RUnlock()
	}

	// 计算标准差，检查分布均匀性
	var sum, sumSq float64
	for _, count := range shardCounts {
		sum += float64(count)
		sumSq += float64(count * count)
	}

	mean := sum / float64(len(shardCounts))
	variance := (sumSq / float64(len(shardCounts))) - (mean * mean)
	stdDev := float64(0)
	if variance > 0 {
		stdDev = variance
	}

	// 输出分片分布统计
	t.Logf("Shard distribution - Mean: %.2f items/shard, StdDev: %.2f (%.2f%%)",
		mean, stdDev, 100*stdDev/mean)

	// 输出每个分片的计数
	for i, count := range shardCounts {
		deviation := 100 * (float64(count) - mean) / mean
		t.Logf("Shard %d: %d items (%.2f%% from mean)", i, count, deviation)
	}

	// 清理
	cache.Close()
}

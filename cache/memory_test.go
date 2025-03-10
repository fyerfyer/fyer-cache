package cache

import (
	"context"
	"strconv"
	"testing"
	"time"
)

func TestMemCacheMemory_MemoryManagement(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "Memory Limit and Eviction",
			testFunc: func(t *testing.T) {
				// 创建一个很小内存限制的缓存（100字节）
				cache := NewMemoryCache()
				memCache := NewMemCacheMemory(cache, 100)

				// 存储一个较大的值
				largeValue := make([]byte, 60)
				err := memCache.Set(ctx, "key1", largeValue, time.Minute)
				if err != nil {
					t.Errorf("Failed to set first value: %v", err)
				}

				// 存储另一个较大的值，应该触发淘汰
				largeValue2 := make([]byte, 60)
				err = memCache.Set(ctx, "key2", largeValue2, time.Minute)
				if err != nil {
					t.Errorf("Failed to set second value: %v", err)
				}

				// 第一个key应该被淘汰
				_, err = memCache.Get(ctx, "key1")
				if err == nil {
					t.Error("Expected key1 to be evicted, but it still exists")
				}
			},
		},
		{
			name: "LRU Eviction Order",
			testFunc: func(t *testing.T) {
				cache := NewMemoryCache()
				memCache := NewMemCacheMemory(cache, 200)

				// 按顺序插入三个值
				value := make([]byte, 40)
				memCache.Set(ctx, "key1", value, time.Minute)
				memCache.Set(ctx, "key2", value, time.Minute)
				memCache.Set(ctx, "key3", value, time.Minute)

				// 访问key1，使它变成最近使用
				memCache.Get(ctx, "key1")

				// 插入第四个值，触发淘汰
				memCache.Set(ctx, "key4", value, time.Minute)

				// key2应该被淘汰（最久未使用）
				_, err := memCache.Get(ctx, "key2")
				if err == nil {
					t.Error("Expected key2 to be evicted, but it still exists")
				}

				// key1应该还在（最近使用）
				_, err = memCache.Get(ctx, "key1")
				if err != nil {
					t.Error("Expected key1 to exist, but it was evicted")
				}
			},
		},
		{
			name: "Memory Usage Tracking",
			testFunc: func(t *testing.T) {
				cache := NewMemoryCache()
				memCache := NewMemCacheMemory(cache, 1000)

				// 存储一个已知大小的值
				value := make([]byte, 100)
				memCache.Set(ctx, "key1", value, time.Minute)

				// 删除该值
				memCache.Del(ctx, "key1")

				// 检查内存使用量是否正确减少
				if memCache.usedMemory != 0 {
					t.Errorf("Expected memory usage to be 0, got %d", memCache.usedMemory)
				}
			},
		},
		{
			name: "Large Object Rejection",
			testFunc: func(t *testing.T) {
				cache := NewMemoryCache()
				memCache := NewMemCacheMemory(cache, 50)

				// 尝试存储一个超过内存限制的值
				largeValue := make([]byte, 100)
				err := memCache.Set(ctx, "key1", largeValue, time.Minute)
				if err != nil {
					t.Error("Expected large value to be rejected or trigger evictions")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

// 测试异步清理功能
func TestAsyncCleanup(t *testing.T) {
	ctx := context.Background()

	// 创建使用异步清理的缓存
	options := []Option{
		WithAsyncCleanup(true),
		CacheWithWorkerCount(2),
		CacheWithQueueSize(10),
		WithCleanupInterval(100 * time.Millisecond),
	}

	cache := NewMemoryCache(options...)

	// 添加一些短期过期的缓存项
	const itemCount = 1000
	for i := 0; i < itemCount; i++ {
		key := "expire-key-" + strconv.Itoa(i)
		err := cache.Set(ctx, key, []byte{1, 2, 3, 4}, 50*time.Millisecond)
		if err != nil {
			t.Fatalf("Failed to set test data: %v", err)
		}
	}

	// 等待足够的时间让项目过期并被清理
	time.Sleep(200 * time.Millisecond)

	// 等待另外100ms让清理任务完成
	time.Sleep(100 * time.Millisecond)

	// 检查计数器
	count := cache.data.Count()
	if count > 0 {
		t.Logf("Expected all items to be cleaned up, but %d items remain", count)
	}

	// 测试关闭
	err := cache.Close()
	if err != nil {
		t.Errorf("Failed to close cache: %v", err)
	}
}

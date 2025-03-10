package cache

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"
)

var (
	benchCtx  = context.Background()
	benchRand = rand.New(rand.NewSource(42))
)

func generateRandomString(length int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, length)
	for i := range b {
		b[i] = letterBytes[benchRand.Intn(len(letterBytes))]
	}
	return string(b)
}

func generateRandomValue(size int) []byte {
	val := make([]byte, size)
	benchRand.Read(val)
	return val
}

// 添加新的辅助函数，接受随机数生成器作为参数
func generateRandomStringWithRand(r *rand.Rand, length int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, length)
	for i := range b {
		b[i] = letterBytes[r.Intn(len(letterBytes))]
	}
	return string(b)
}

func generateRandomValueWithRand(r *rand.Rand, size int) []byte {
	val := make([]byte, size)
	r.Read(val)
	return val
}

func BenchmarkMemoryCache_Set(b *testing.B) {
	cache := NewMemoryCache()
	defer cache.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(benchCtx, key, "value", time.Minute)
	}
}

func BenchmarkMemoryCache_Get_Hit(b *testing.B) {
	cache := NewMemoryCache()
	defer cache.Close()

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(benchCtx, key, fmt.Sprintf("value-%d", i), time.Minute)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		cache.Get(benchCtx, key)
	}
}

func BenchmarkMemoryCache_Get_Miss(b *testing.B) {
	cache := NewMemoryCache()
	defer cache.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("missing-key-%d", i)
		cache.Get(benchCtx, key)
	}
}

func BenchmarkMemoryCache_Del(b *testing.B) {
	cache := NewMemoryCache()
	defer cache.Close()

	// 预填充缓存
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(benchCtx, key, "value", time.Minute)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Del(benchCtx, key)
	}
}

// 并行操作基准测试

func BenchmarkMemoryCache_Parallel_Get(b *testing.B) {
	cache := NewMemoryCache()
	defer cache.Close()

	// 预填充10k个数据
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(benchCtx, key, fmt.Sprintf("value-%d", i), time.Minute)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%10000)
			cache.Get(benchCtx, key)
			i++
		}
	})
}

func BenchmarkMemoryCache_Parallel_Set(b *testing.B) {
	cache := NewMemoryCache()
	defer cache.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i)
			cache.Set(benchCtx, key, fmt.Sprintf("value-%d", i), time.Minute)
			i++
		}
	})
}

func BenchmarkMemoryCache_Parallel_Mixed(b *testing.B) {
	cache := NewMemoryCache()
	defer cache.Close()

	for i := 0; i < 5000; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(benchCtx, key, fmt.Sprintf("value-%d", i), time.Minute)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// 为每个goroutine创建独立的随机数生成器
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		i := 0
		for pb.Next() {
			if i%3 == 0 {
				// 33%写操作
				key := fmt.Sprintf("key-%d", localRand.Intn(10000))
				cache.Set(benchCtx, key, fmt.Sprintf("value-%d", i), time.Minute)
			} else {
				// 67%读操作
				key := fmt.Sprintf("key-%d", localRand.Intn(10000))
				cache.Get(benchCtx, key)
			}
			i++
		}
	})
}

// Shard count benchmarks

func benchmarkWithShardCount(b *testing.B, shardCount int) {
	cache := NewMemoryCache(WithShardCount(shardCount))
	defer cache.Close()

	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(benchCtx, key, fmt.Sprintf("value-%d", i), time.Minute)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// 为每个goroutine创建单独的随机数生成器
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		i := 0
		for pb.Next() {
			if i%4 == 0 {
				// 25%写操作
				key := fmt.Sprintf("key-%d", localRand.Intn(20000))
				cache.Set(benchCtx, key, fmt.Sprintf("value-%d", i), time.Minute)
			} else {
				// 75%读操作
				key := fmt.Sprintf("key-%d", localRand.Intn(10000))
				cache.Get(benchCtx, key)
			}
			i++
		}
	})
}

func BenchmarkShardCount_4(b *testing.B)   { benchmarkWithShardCount(b, 4) }
func BenchmarkShardCount_8(b *testing.B)   { benchmarkWithShardCount(b, 8) }
func BenchmarkShardCount_16(b *testing.B)  { benchmarkWithShardCount(b, 16) }
func BenchmarkShardCount_32(b *testing.B)  { benchmarkWithShardCount(b, 32) }
func BenchmarkShardCount_64(b *testing.B)  { benchmarkWithShardCount(b, 64) }
func BenchmarkShardCount_128(b *testing.B) { benchmarkWithShardCount(b, 128) }

// 清理策略基准测试

func BenchmarkCleanupStrategy_Sync(b *testing.B) {
	b.StopTimer()
	cache := NewMemoryCache(WithCleanupInterval(time.Millisecond * 100))
	defer cache.Close()

	// 添加多个将要过期的数据
	for i := 0; i < 10000; i++ {
		cache.Set(benchCtx, fmt.Sprintf("key-%d", i), "value", time.Millisecond*50)
	}

	time.Sleep(time.Millisecond * 200)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		cache.Set(benchCtx, fmt.Sprintf("new-key-%d", i), "value", time.Minute)
		cache.Get(benchCtx, fmt.Sprintf("new-key-%d", i))
	}
}

func BenchmarkCleanupStrategy_Async(b *testing.B) {
	b.StopTimer()
	cache := NewMemoryCache(
		WithAsyncCleanup(true),
		CacheWithWorkerCount(4),
		WithCleanupInterval(time.Millisecond*100),
	)
	defer cache.Close()

	for i := 0; i < 10000; i++ {
		cache.Set(benchCtx, fmt.Sprintf("key-%d", i), "value", time.Millisecond*50)
	}

	time.Sleep(time.Millisecond * 200)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		cache.Set(benchCtx, fmt.Sprintf("new-key-%d", i), "value", time.Minute)
		cache.Get(benchCtx, fmt.Sprintf("new-key-%d", i))
	}
}

// 异步清理基准测试

func benchmarkWithWorkerCount(b *testing.B, workerCount int) {
	cache := NewMemoryCache(
		WithAsyncCleanup(true),
		CacheWithWorkerCount(workerCount),
		WithCleanupInterval(time.Millisecond*100),
	)
	defer cache.Close()

	for i := 0; i < 10000; i++ {
		expiration := time.Duration(50+benchRand.Intn(100)) * time.Millisecond
		cache.Set(benchCtx, fmt.Sprintf("key-%d", i), "value", expiration)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// 为每个goroutine创建单独的随机数生成器
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		i := 0
		for pb.Next() {
			if i%4 == 0 {
				// 25%写操作
				cache.Set(benchCtx, fmt.Sprintf("bench-key-%d", i), "value", time.Minute)
			} else {
				// 75%读操作
				// 使用localRand代替benchRand
				cache.Get(benchCtx, fmt.Sprintf("key-%d", localRand.Intn(10000)))
			}
			i++
		}
	})
}

func BenchmarkWorkerCount_1(b *testing.B)  { benchmarkWithWorkerCount(b, 1) }
func BenchmarkWorkerCount_2(b *testing.B)  { benchmarkWithWorkerCount(b, 2) }
func BenchmarkWorkerCount_4(b *testing.B)  { benchmarkWithWorkerCount(b, 4) }
func BenchmarkWorkerCount_8(b *testing.B)  { benchmarkWithWorkerCount(b, 8) }
func BenchmarkWorkerCount_16(b *testing.B) { benchmarkWithWorkerCount(b, 16) }

// 带内存限制的缓存基准测试

func BenchmarkMemCacheMemory_WithEviction(b *testing.B) {
	baseCache := NewMemoryCache()
	memCache := NewMemCacheMemory(baseCache, 1024*1024)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// 为每个goroutine创建单独的随机数生成器
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		i := 0
		for pb.Next() {
			valueSize := 256 + localRand.Intn(512) // 使用localRand

			// 内联generateRandomValue的逻辑，使用localRand
			value := make([]byte, valueSize)
			localRand.Read(value)

			key := fmt.Sprintf("key-%d", i%5000)
			memCache.Set(benchCtx, key, value, time.Minute)

			if i%3 != 0 {
				memCache.Get(benchCtx, key)
			}
			i++
		}
	})
}

// 测试不同数据类型性能

func benchmarkWithValueSize(b *testing.B, valueSize int) {
	cache := NewMemoryCache()
	defer cache.Close()

	value := generateRandomValue(valueSize)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := strconv.Itoa(i)
		cache.Set(benchCtx, key, value, time.Minute)
	}
}

func BenchmarkValueSize_32B(b *testing.B)   { benchmarkWithValueSize(b, 32) }
func BenchmarkValueSize_256B(b *testing.B)  { benchmarkWithValueSize(b, 256) }
func BenchmarkValueSize_1KB(b *testing.B)   { benchmarkWithValueSize(b, 1024) }
func BenchmarkValueSize_10KB(b *testing.B)  { benchmarkWithValueSize(b, 10*1024) }
func BenchmarkValueSize_100KB(b *testing.B) { benchmarkWithValueSize(b, 100*1024) }

// 集成场景性能测试

// 修改BenchmarkScenario_SessionCache
func BenchmarkScenario_SessionCache(b *testing.B) {
	// 模拟有多写入的session场景
	cache := NewMemoryCache(WithShardCount(32))
	defer cache.Close()

	// 创建一个专用的随机数生成器用于初始化
	initRand := rand.New(rand.NewSource(42))

	// 预填充会话数据
	sessionIDs := make([]string, 10000)
	for i := 0; i < 10000; i++ {
		sessionID := generateRandomStringWithRand(initRand, 32)
		sessionIDs[i] = sessionID
		sessionData := map[string]interface{}{
			"user_id":       i,
			"username":      generateRandomStringWithRand(initRand, 16),
			"email":         generateRandomStringWithRand(initRand, 20) + "@example.com",
			"permissions":   []string{"read", "write", "admin"},
			"last_access":   time.Now().Unix(),
			"created_at":    time.Now().Add(-time.Hour * 24 * time.Duration(initRand.Intn(30))).Unix(),
			"user_settings": generateRandomValueWithRand(initRand, 512),
		}
		cache.Set(benchCtx, "session:"+sessionID, sessionData, time.Hour)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// 为每个goroutine创建单独的随机数生成器
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		for pb.Next() {
			// 90%读操作
			if localRand.Float64() < 0.9 {
				// 随机读取session
				sessionIdx := localRand.Intn(len(sessionIDs))
				cache.Get(benchCtx, "session:"+sessionIDs[sessionIdx])
			} else {
				// 10%写操作
				sessionIdx := localRand.Intn(len(sessionIDs))
				sessionData := map[string]interface{}{
					"user_id":       sessionIdx,
					"last_access":   time.Now().Unix(),
					"permissions":   []string{"read", "write"},
					"user_settings": generateRandomValueWithRand(localRand, 512),
				}
				cache.Set(benchCtx, "session:"+sessionIDs[sessionIdx], sessionData, time.Hour)
			}
		}
	})
}

// 修改BenchmarkScenario_APIRateLimit (虽然看起来已经使用了localRand, 但为了确保完整性)
func BenchmarkScenario_APIRateLimit(b *testing.B) {
	// 模拟api限流
	cache := NewMemoryCache(
		WithShardCount(64),
		WithAsyncCleanup(true),
		CacheWithWorkerCount(4),
	)
	defer cache.Close()

	numClients := 1000

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// 确保使用本地随机数生成器
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		counter := 0
		for pb.Next() {
			clientID := localRand.Intn(numClients)
			key := fmt.Sprintf("ratelimit:client:%d", clientID)

			val, err := cache.Get(benchCtx, key)

			var count int
			if err == nil {
				count = val.(int) + 1
			} else {
				count = 1
			}

			cache.Set(benchCtx, key, count, time.Second*60)
			counter++
		}
	})
}

// 键分布基准测试

func BenchmarkKeyDistribution_Sequential(b *testing.B) {
	cache := NewMemoryCache()
	defer cache.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(benchCtx, key, "value", time.Minute)
	}
}

// 修改BenchmarkKeyDistribution_Random
func BenchmarkKeyDistribution_Random(b *testing.B) {
	cache := NewMemoryCache()
	defer cache.Close()

	// 创建一个局部随机数生成器（非并行测试也最好不用全局）
	localRand := rand.New(rand.NewSource(42))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 使用随机键以产生不同的哈希
		key := fmt.Sprintf("key-%d", localRand.Intn(1000000))
		cache.Set(benchCtx, key, "value", time.Minute)
	}
}

// 内容访问基准测试

// 修改BenchmarkContention_HighContention - 虽然没有使用随机生成器，但最好为每个goroutine添加
func BenchmarkContention_HighContention(b *testing.B) {
	cache := NewMemoryCache()
	defer cache.Close()

	const keyCount = 10

	var wg sync.WaitGroup
	b.ResetTimer()

	for g := 0; g < 100; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			// 为每个goroutine创建独立的随机数生成器（虽然这里没有使用，但为了代码一致性）
			localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
			_ = localRand // 避免未使用变量警告

			for i := 0; i < b.N/100; i++ {
				keyIdx := i % keyCount
				key := fmt.Sprintf("key-%d", keyIdx)

				if i%5 == 0 {
					// 20%写操作
					cache.Set(benchCtx, key, fmt.Sprintf("value-%d-%d", goroutineID, i), time.Minute)
				} else {
					// 80%读操作
					cache.Get(benchCtx, key)
				}
			}
		}(g)
	}

	wg.Wait()
}

// 修改BenchmarkContention_LowContention - 同样为每个goroutine添加随机数生成器
func BenchmarkContention_LowContention(b *testing.B) {
	cache := NewMemoryCache()
	defer cache.Close()

	const keyCount = 100000

	var wg sync.WaitGroup
	b.ResetTimer()

	for g := 0; g < 100; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			// 为每个goroutine创建独立的随机数生成器
			localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
			_ = localRand // 避免未使用变量警告

			keyOffset := goroutineID * keyCount / 100

			for i := 0; i < b.N/100; i++ {
				keyIdx := keyOffset + (i % (keyCount / 100))
				key := fmt.Sprintf("key-%d", keyIdx)

				if i%5 == 0 {
					// 20%写操作
					cache.Set(benchCtx, key, fmt.Sprintf("value-%d-%d", goroutineID, i), time.Minute)
				} else {
					// 80%读操作
					cache.Get(benchCtx, key)
				}
			}
		}(g)
	}

	wg.Wait()
}

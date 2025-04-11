package cache

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/internal/ferr"
)

var (
	benchCtx  = context.Background()
	benchRand = rand.New(rand.NewSource(42))
)

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[benchRand.Intn(len(charset))]
	}
	return string(b)
}

func generateRandomValue(size int) []byte {
	b := make([]byte, size)
	benchRand.Read(b)
	return b
}

func generateRandomStringWithRand(r *rand.Rand, length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}

func generateRandomValueWithRand(r *rand.Rand, size int) []byte {
	b := make([]byte, size)
	r.Read(b)
	return b
}

func BenchmarkMemoryCache_Set(b *testing.B) {
	cache := NewMemoryCache()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := generateRandomValue(100)
		err := cache.Set(benchCtx, key, value, 5*time.Minute)
		if err != nil {
			b.Fatalf("Failed to set: %v", err)
		}
	}
}

func BenchmarkMemoryCache_Get_Hit(b *testing.B) {
	cache := NewMemoryCache()

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := generateRandomValue(100)
		cache.Set(benchCtx, key, value, 5*time.Minute)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		_, err := cache.Get(benchCtx, key)
		if err != nil {
			b.Fatalf("Failed to get key %s: %v", key, err)
		}
	}
}

func BenchmarkMemoryCache_Get_Miss(b *testing.B) {
	cache := NewMemoryCache()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("nonexistent-key-%d", i)
		_, err := cache.Get(benchCtx, key)
		if err != ferr.ErrKeyNotFound {
			b.Fatalf("Expected ErrKeyNotFound for key %s, got: %v", key, err)
		}
	}
}

func BenchmarkMemoryCache_Del(b *testing.B) {
	cache := NewMemoryCache()

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := generateRandomValue(100)
		cache.Set(benchCtx, key, value, 5*time.Minute)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		err := cache.Del(benchCtx, key)
		if err != nil {
			b.Fatalf("Failed to delete key %s: %v", key, err)
		}
	}
}

func BenchmarkMemoryCache_Parallel_Get(b *testing.B) {
	cache := NewMemoryCache()

	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := generateRandomValue(100)
		cache.Set(benchCtx, key, value, 5*time.Minute)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		for pb.Next() {
			key := fmt.Sprintf("key-%d", localRand.Intn(10000))
			_, err := cache.Get(benchCtx, key)
			if err != nil && err != ferr.ErrKeyNotFound {
				b.Fatalf("Unexpected error on get: %v", err)
			}
		}
	})
}

func BenchmarkMemoryCache_Parallel_Set(b *testing.B) {
	cache := NewMemoryCache()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d-%d", localRand.Intn(10000), i)
			value := generateRandomValueWithRand(localRand, 100)
			err := cache.Set(benchCtx, key, value, 5*time.Minute)
			if err != nil {
				b.Fatalf("Failed to set: %v", err)
			}
			i++
		}
	})
}

func BenchmarkMemoryCache_Parallel_Mixed(b *testing.B) {
	cache := NewMemoryCache()

	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := generateRandomValue(100)
		cache.Set(benchCtx, key, value, 5*time.Minute)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		i := 0
		for pb.Next() {
			op := localRand.Intn(3)
			key := fmt.Sprintf("key-%d", localRand.Intn(10000))

			switch op {
			case 0: // Get
				_, _ = cache.Get(benchCtx, key)
			case 1: // Set
				value := generateRandomValueWithRand(localRand, 100)
				_ = cache.Set(benchCtx, key, value, 5*time.Minute)
			case 2: // Del
				_ = cache.Del(benchCtx, key)
			}
			i++
		}
	})
}

func benchmarkWithShardCount(b *testing.B, shardCount int) {
	cache := NewMemoryCache(WithShardCount(shardCount))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d-%d", localRand.Intn(10000), i)
			value := generateRandomValueWithRand(localRand, 100)
			err := cache.Set(benchCtx, key, value, 5*time.Minute)
			if err != nil {
				b.Fatalf("Failed to set: %v", err)
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

func BenchmarkCleanupStrategy_Sync(b *testing.B) {
	cache := NewMemoryCache(WithAsyncCleanup(false))

	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := generateRandomValue(100)
		cache.Set(benchCtx, key, value, 100*time.Millisecond)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("new-key-%d", i)
		value := generateRandomValue(100)
		err := cache.Set(benchCtx, key, value, 5*time.Minute)
		if err != nil {
			b.Fatalf("Failed to set: %v", err)
		}
	}
}

func BenchmarkCleanupStrategy_Async(b *testing.B) {
	cache := NewMemoryCache(
		WithAsyncCleanup(true),
		CacheWithWorkerCount(4),
		CacheWithQueueSize(1000),
	)

	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := generateRandomValue(100)
		cache.Set(benchCtx, key, value, 100*time.Millisecond)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("new-key-%d", i)
		value := generateRandomValue(100)
		err := cache.Set(benchCtx, key, value, 5*time.Minute)
		if err != nil {
			b.Fatalf("Failed to set: %v", err)
		}
	}
}

func benchmarkWithWorkerCount(b *testing.B, workerCount int) {
	cache := NewMemoryCache(
		WithAsyncCleanup(true),
		CacheWithWorkerCount(workerCount),
		CacheWithQueueSize(1000),
	)

	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := generateRandomValue(100)
		cache.Set(benchCtx, key, value, 100*time.Millisecond)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("new-key-%d", i)
		value := generateRandomValue(100)
		err := cache.Set(benchCtx, key, value, 5*time.Minute)
		if err != nil {
			b.Fatalf("Failed to set: %v", err)
		}
	}
}

func BenchmarkWorkerCount_1(b *testing.B)  { benchmarkWithWorkerCount(b, 1) }
func BenchmarkWorkerCount_2(b *testing.B)  { benchmarkWithWorkerCount(b, 2) }
func BenchmarkWorkerCount_4(b *testing.B)  { benchmarkWithWorkerCount(b, 4) }
func BenchmarkWorkerCount_8(b *testing.B)  { benchmarkWithWorkerCount(b, 8) }
func BenchmarkWorkerCount_16(b *testing.B) { benchmarkWithWorkerCount(b, 16) }

func BenchmarkMemCacheMemory_WithEviction(b *testing.B) {
	base := NewMemoryCache()
	cache := NewMemCacheMemory(base, 1024*1024) // 1MB maximum

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := generateRandomValue(1000) // 1KB values
		err := cache.Set(benchCtx, key, value, 5*time.Minute)
		if err != nil {
			b.Fatalf("Failed to set: %v", err)
		}
	}
}

func benchmarkWithValueSize(b *testing.B, valueSize int) {
	cache := NewMemoryCache()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := generateRandomValue(valueSize)
		err := cache.Set(benchCtx, key, value, 5*time.Minute)
		if err != nil {
			b.Fatalf("Failed to set: %v", err)
		}
	}
}

func BenchmarkValueSize_32B(b *testing.B)   { benchmarkWithValueSize(b, 32) }
func BenchmarkValueSize_256B(b *testing.B)  { benchmarkWithValueSize(b, 256) }

func BenchmarkScenario_SessionCache(b *testing.B) {
	cache := NewMemoryCache(
		WithCleanupInterval(5 * time.Second),
		WithShardCount(64),
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		for pb.Next() {
			sessionID := generateRandomStringWithRand(localRand, 32)
			userData := map[string]interface{}{
				"user_id":    localRand.Intn(10000),
				"username":   generateRandomStringWithRand(localRand, 10),
				"email":      generateRandomStringWithRand(localRand, 15) + "@example.com",
				"last_login": time.Now(),
				"role":       "user",
			}

			err := cache.Set(benchCtx, "session:"+sessionID, userData, 30*time.Minute)
			if err != nil {
				b.Fatalf("Failed to set session: %v", err)
			}

			_, err = cache.Get(benchCtx, "session:"+sessionID)
			if err != nil {
				b.Fatalf("Failed to get session: %v", err)
			}
		}
	})
}

func BenchmarkScenario_APIRateLimit(b *testing.B) {
	cache := NewMemoryCache(
		WithCleanupInterval(100 * time.Millisecond),
		WithShardCount(32),
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))

		for pb.Next() {
			userID := localRand.Intn(1000)
			endpoint := fmt.Sprintf("/api/resource-%d", localRand.Intn(10))
			key := fmt.Sprintf("ratelimit:%d:%s", userID, endpoint)

			val, err := cache.Get(benchCtx, key)

			var count int
			if err == nil {
				count = val.(int) + 1
			} else {
				count = 1
			}

			err = cache.Set(benchCtx, key, count, 1*time.Minute)
			if err != nil {
				b.Fatalf("Failed to update rate limit: %v", err)
			}
		}
	})
}

func BenchmarkKeyDistribution_Random(b *testing.B) {
	cache := NewMemoryCache(WithShardCount(64))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		for pb.Next() {
			key := generateRandomStringWithRand(localRand, 16)
			value := generateRandomValueWithRand(localRand, 100)
			_ = cache.Set(benchCtx, key, value, 5*time.Minute)
		}
	})
}

func BenchmarkContention_HighContention(b *testing.B) {
	cache := NewMemoryCache()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// Create a local random generator for each goroutine
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))

		for pb.Next() {
			const hotKey = "hot-key"
			// Use the local random generator instead of the global one
			err := cache.Set(benchCtx, hotKey, generateRandomValueWithRand(localRand, 100), 5*time.Minute)
			if err != nil {
				b.Fatalf("Failed to set: %v", err)
			}

			_, err = cache.Get(benchCtx, hotKey)
			if err != nil && err != ferr.ErrKeyNotFound {
				b.Fatalf("Failed to get: %v", err)
			}
		}
	})
}

func BenchmarkContention_LowContention(b *testing.B) {
	cache := NewMemoryCache()
	keyCount := 10000
	keys := make([]string, keyCount)

	for i := 0; i < keyCount; i++ {
		keys[i] = "key-" + strconv.Itoa(i)
		cache.Set(benchCtx, keys[i], generateRandomValue(100), 5*time.Minute)
	}

	var mu sync.Mutex
	counter := 0

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		for pb.Next() {
			mu.Lock()
			i := counter % keyCount
			counter++
			mu.Unlock()

			key := keys[i]
			if localRand.Intn(10) < 8 { // 80% reads, 20% writes
				_, _ = cache.Get(benchCtx, key)
			} else {
				_ = cache.Set(benchCtx, key, generateRandomValueWithRand(localRand, 100), 5*time.Minute)
			}
		}
	})
}
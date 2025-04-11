package consistency

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/internal/ferr"
)

type benchmarkMockCache struct {
	mu   sync.RWMutex
	data map[string]any
}

func newBenchmarkMockCache() *benchmarkMockCache {
	return &benchmarkMockCache{
		data: make(map[string]any),
	}
}

func (m *benchmarkMockCache) Get(ctx context.Context, key string) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return nil, ferr.ErrKeyNotFound
}

func (m *benchmarkMockCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = val
	return nil
}

func (m *benchmarkMockCache) Del(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

type benchmarkMockDataSource struct {
	mu   sync.RWMutex
	data map[string]any
}

func newBenchmarkMockDataSource() *benchmarkMockDataSource {
	return &benchmarkMockDataSource{
		data: make(map[string]any),
	}
}

func (ds *benchmarkMockDataSource) Load(ctx context.Context, key string) (any, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	if val, ok := ds.data[key]; ok {
		return val, nil
	}
	return nil, ferr.ErrKeyNotFound
}

func (ds *benchmarkMockDataSource) Store(ctx context.Context, key string, value any) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.data[key] = value
	return nil
}

func (ds *benchmarkMockDataSource) Remove(ctx context.Context, key string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	delete(ds.data, key)
	return nil
}

// Simple MessagePublisher implementation for benchmarks
type benchmarkMockPublisher struct{}

func newBenchmarkMockPublisher() *benchmarkMockPublisher {
	return &benchmarkMockPublisher{}
}

func (p *benchmarkMockPublisher) Publish(ctx context.Context, topic string, msg []byte) error {
	return nil
}

func (p *benchmarkMockPublisher) Subscribe(topic string, handler func(msg []byte)) error {
	return nil
}

func (p *benchmarkMockPublisher) Close() error {
	return nil
}

func BenchmarkCacheAsideStrategy_Get_CacheHit(b *testing.B) {
	mockCache := newBenchmarkMockCache()
	mockDataSource := newBenchmarkMockDataSource()

	strategy := NewCacheAsideStrategy(mockCache, mockDataSource)

	ctx := context.Background()
	mockCache.Set(ctx, "test-key", "test-value", 5*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = strategy.Get(ctx, "test-key")
	}
}

func BenchmarkCacheAsideStrategy_Get_CacheMiss(b *testing.B) {
	mockCache := newBenchmarkMockCache()
	mockDataSource := newBenchmarkMockDataSource()

	strategy := NewCacheAsideStrategy(mockCache, mockDataSource)

	ctx := context.Background()
	mockDataSource.Store(ctx, "test-key", "test-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = strategy.Get(ctx, "test-key")
	}
}

func BenchmarkCacheAsideStrategy_Set(b *testing.B) {
	mockCache := newBenchmarkMockCache()
	mockDataSource := newBenchmarkMockDataSource()

	strategy := NewCacheAsideStrategy(mockCache, mockDataSource)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "test-key" + strconv.Itoa(i)
		_ = strategy.Set(ctx, key, "test-value", 5*time.Minute)
	}
}

func BenchmarkCacheAsideStrategy_Del(b *testing.B) {
	mockCache := newBenchmarkMockCache()
	mockDataSource := newBenchmarkMockDataSource()

	strategy := NewCacheAsideStrategy(mockCache, mockDataSource)

	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		key := "test-key" + strconv.Itoa(i)
		mockCache.Set(ctx, key, "test-value", 5*time.Minute)
		mockDataSource.Store(ctx, key, "test-value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "test-key" + strconv.Itoa(i)
		_ = strategy.Del(ctx, key)
	}
}

func BenchmarkWriteThroughStrategy_Get_CacheHit(b *testing.B) {
	mockCache := newBenchmarkMockCache()
	mockDataSource := newBenchmarkMockDataSource()

	strategy := NewWriteThroughStrategy(mockCache, mockDataSource)

	ctx := context.Background()
	mockCache.Set(ctx, "test-key", "test-value", 5*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = strategy.Get(ctx, "test-key")
	}
}

func BenchmarkWriteThroughStrategy_Get_CacheMiss(b *testing.B) {
	mockCache := newBenchmarkMockCache()
	mockDataSource := newBenchmarkMockDataSource()

	strategy := NewWriteThroughStrategy(mockCache, mockDataSource)

	ctx := context.Background()
	mockDataSource.Store(ctx, "test-key", "test-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = strategy.Get(ctx, "test-key")
	}
}

func BenchmarkWriteThroughStrategy_Set(b *testing.B) {
	mockCache := newBenchmarkMockCache()
	mockDataSource := newBenchmarkMockDataSource()

	strategy := NewWriteThroughStrategy(mockCache, mockDataSource)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "test-key" + strconv.Itoa(i)
		_ = strategy.Set(ctx, key, "test-value", 5*time.Minute)
	}
}

func BenchmarkWriteThroughStrategy_Del(b *testing.B) {
	mockCache := newBenchmarkMockCache()
	mockDataSource := newBenchmarkMockDataSource()

	strategy := NewWriteThroughStrategy(mockCache, mockDataSource)

	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		key := "test-key" + strconv.Itoa(i)
		mockCache.Set(ctx, key, "test-value", 5*time.Minute)
		mockDataSource.Store(ctx, key, "test-value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "test-key" + strconv.Itoa(i)
		_ = strategy.Del(ctx, key)
	}
}

func BenchmarkMQNotifierStrategy_Get_CacheHit(b *testing.B) {
	mockCache := newBenchmarkMockCache()
	mockDataSource := newBenchmarkMockDataSource()
	mockPublisher := newBenchmarkMockPublisher()

	strategy := NewMQNotifierStrategy(mockCache, mockDataSource, mockPublisher)

	ctx := context.Background()
	mockCache.Set(ctx, "test-key", "test-value", 5*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = strategy.Get(ctx, "test-key")
	}
}

func BenchmarkMQNotifierStrategy_Get_CacheMiss(b *testing.B) {
	mockCache := newBenchmarkMockCache()
	mockDataSource := newBenchmarkMockDataSource()
	mockPublisher := newBenchmarkMockPublisher()

	strategy := NewMQNotifierStrategy(mockCache, mockDataSource, mockPublisher)

	ctx := context.Background()
	mockDataSource.Store(ctx, "test-key", "test-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = strategy.Get(ctx, "test-key")
	}
}

func BenchmarkMQNotifierStrategy_Set(b *testing.B) {
	mockCache := newBenchmarkMockCache()
	mockDataSource := newBenchmarkMockDataSource()
	mockPublisher := newBenchmarkMockPublisher()

	strategy := NewMQNotifierStrategy(mockCache, mockDataSource, mockPublisher)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "test-key" + strconv.Itoa(i)
		_ = strategy.Set(ctx, key, "test-value", 5*time.Minute)
	}
}

func BenchmarkMQNotifierStrategy_Del(b *testing.B) {
	mockCache := newBenchmarkMockCache()
	mockDataSource := newBenchmarkMockDataSource()
	mockPublisher := newBenchmarkMockPublisher()

	strategy := NewMQNotifierStrategy(mockCache, mockDataSource, mockPublisher)

	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		key := "test-key" + strconv.Itoa(i)
		mockCache.Set(ctx, key, "test-value", 5*time.Minute)
		mockDataSource.Store(ctx, key, "test-value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "test-key" + strconv.Itoa(i)
		_ = strategy.Del(ctx, key)
	}
}

func BenchmarkCompareStrategies_Get_CacheHit(b *testing.B) {
	b.Run("CacheAside", func(b *testing.B) {
		mockCache := newBenchmarkMockCache()
		mockDataSource := newBenchmarkMockDataSource()
		strategy := NewCacheAsideStrategy(mockCache, mockDataSource)

		ctx := context.Background()
		mockCache.Set(ctx, "test-key", "test-value", 5*time.Minute)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = strategy.Get(ctx, "test-key")
		}
	})

	b.Run("WriteThrough", func(b *testing.B) {
		mockCache := newBenchmarkMockCache()
		mockDataSource := newBenchmarkMockDataSource()
		strategy := NewWriteThroughStrategy(mockCache, mockDataSource)

		ctx := context.Background()
		mockCache.Set(ctx, "test-key", "test-value", 5*time.Minute)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = strategy.Get(ctx, "test-key")
		}
	})

	b.Run("MQNotifier", func(b *testing.B) {
		mockCache := newBenchmarkMockCache()
		mockDataSource := newBenchmarkMockDataSource()
		mockPublisher := newBenchmarkMockPublisher()
		strategy := NewMQNotifierStrategy(mockCache, mockDataSource, mockPublisher)

		ctx := context.Background()
		mockCache.Set(ctx, "test-key", "test-value", 5*time.Minute)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = strategy.Get(ctx, "test-key")
		}
	})
}

func BenchmarkCompareStrategies_Set(b *testing.B) {
	b.Run("CacheAside", func(b *testing.B) {
		mockCache := newBenchmarkMockCache()
		mockDataSource := newBenchmarkMockDataSource()
		strategy := NewCacheAsideStrategy(mockCache, mockDataSource)

		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := "test-key" + strconv.Itoa(i)
			_ = strategy.Set(ctx, key, "test-value", 5*time.Minute)
		}
	})

	b.Run("WriteThrough", func(b *testing.B) {
		mockCache := newBenchmarkMockCache()
		mockDataSource := newBenchmarkMockDataSource()
		strategy := NewWriteThroughStrategy(mockCache, mockDataSource)

		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := "test-key" + strconv.Itoa(i)
			_ = strategy.Set(ctx, key, "test-value", 5*time.Minute)
		}
	})

	b.Run("MQNotifier", func(b *testing.B) {
		mockCache := newBenchmarkMockCache()
		mockDataSource := newBenchmarkMockDataSource()
		mockPublisher := newBenchmarkMockPublisher()
		strategy := NewMQNotifierStrategy(mockCache, mockDataSource, mockPublisher)

		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := "test-key" + strconv.Itoa(i)
			_ = strategy.Set(ctx, key, "test-value", 5*time.Minute)
		}
	})
}

func BenchmarkCompareStrategies_Del(b *testing.B) {
	b.Run("CacheAside", func(b *testing.B) {
		mockCache := newBenchmarkMockCache()
		mockDataSource := newBenchmarkMockDataSource()
		strategy := NewCacheAsideStrategy(mockCache, mockDataSource)

		ctx := context.Background()
		for i := 0; i < b.N; i++ {
			key := "test-key" + strconv.Itoa(i)
			mockCache.Set(ctx, key, "test-value", 5*time.Minute)
			mockDataSource.Store(ctx, key, "test-value")
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := "test-key" + strconv.Itoa(i)
			_ = strategy.Del(ctx, key)
		}
	})

	b.Run("WriteThrough", func(b *testing.B) {
		mockCache := newBenchmarkMockCache()
		mockDataSource := newBenchmarkMockDataSource()
		strategy := NewWriteThroughStrategy(mockCache, mockDataSource)

		ctx := context.Background()
		for i := 0; i < b.N; i++ {
			key := "test-key" + strconv.Itoa(i)
			mockCache.Set(ctx, key, "test-value", 5*time.Minute)
			mockDataSource.Store(ctx, key, "test-value")
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := "test-key" + strconv.Itoa(i)
			_ = strategy.Del(ctx, key)
		}
	})

	b.Run("MQNotifier", func(b *testing.B) {
		mockCache := newBenchmarkMockCache()
		mockDataSource := newBenchmarkMockDataSource()
		mockPublisher := newBenchmarkMockPublisher()
		strategy := NewMQNotifierStrategy(mockCache, mockDataSource, mockPublisher)

		ctx := context.Background()
		for i := 0; i < b.N; i++ {
			key := "test-key" + strconv.Itoa(i)
			mockCache.Set(ctx, key, "test-value", 5*time.Minute)
			mockDataSource.Store(ctx, key, "test-value")
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := "test-key" + strconv.Itoa(i)
			_ = strategy.Del(ctx, key)
		}
	})
}
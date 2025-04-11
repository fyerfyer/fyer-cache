package metrics

import (
	"context"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/internal/ferr"
)

type benchmarkCache struct {
	mu   sync.Mutex // Add mutex for thread safety
	data map[string]any
}

func newBenchmarkCache() *benchmarkCache {
	return &benchmarkCache{
		data: make(map[string]any),
	}
}

func (m *benchmarkCache) Get(ctx context.Context, key string) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	val, ok := m.data[key]
	if !ok {
		return nil, ferr.ErrKeyNotFound
	}
	return val, nil
}

func (m *benchmarkCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = val
	return nil
}

func (m *benchmarkCache) Del(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func BenchmarkPrometheusCollector_IncHits(b *testing.B) {
	config := &MetricsConfig{
		Namespace: "benchmark",
		Subsystem: "cache",
	}
	collector := NewPrometheusCollector(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.IncHits()
	}
}

func BenchmarkPrometheusCollector_IncMisses(b *testing.B) {
	config := &MetricsConfig{
		Namespace: "benchmark",
		Subsystem: "cache",
	}
	collector := NewPrometheusCollector(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.IncMisses()
	}
}

func BenchmarkPrometheusCollector_IncRequests(b *testing.B) {
	config := &MetricsConfig{
		Namespace: "benchmark",
		Subsystem: "cache",
	}
	collector := NewPrometheusCollector(config)

	operations := []string{"get", "set", "del"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.IncRequests(operations[i%3])
	}
}

func BenchmarkPrometheusCollector_ObserveLatency(b *testing.B) {
	config := &MetricsConfig{
		Namespace: "benchmark",
		Subsystem: "cache",
	}
	collector := NewPrometheusCollector(config)
	duration := 100 * time.Microsecond

	operations := []string{"get", "set", "del"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.ObserveLatency(operations[i%3], duration)
	}
}

func BenchmarkPrometheusExporter_Export(b *testing.B) {
	config := &MetricsConfig{
		Namespace: "benchmark",
		Subsystem: "cache",
	}
	collector := NewPrometheusCollector(config)
	exporter := NewPrometheusExporter(collector)

	collector.IncHits()
	collector.IncMisses()
	collector.IncRequests("get")
	collector.ObserveLatency("get", 100*time.Microsecond)
	collector.SetMemoryUsage(1024 * 1024)
	collector.SetItemCount(100)

	w := httptest.NewRecorder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		exporter.Export(w)
		w.Body.Reset()
	}
}

func BenchmarkMonitoredCache_Get(b *testing.B) {
	ctx := context.Background()
	originalCache := newBenchmarkCache()
	monitoredCache := NewMonitoredCache(originalCache)

	originalCache.Set(ctx, "key", "value", 0)

	b.Run("Original", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = originalCache.Get(ctx, "key")
		}
	})

	b.Run("Monitored", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = monitoredCache.Get(ctx, "key")
		}
	})
}

func BenchmarkMonitoredCache_Set(b *testing.B) {
	ctx := context.Background()
	originalCache := newBenchmarkCache()
	monitoredCache := NewMonitoredCache(originalCache)

	b.Run("Original", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = originalCache.Set(ctx, "key", "value", 0)
		}
	})

	b.Run("Monitored", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = monitoredCache.Set(ctx, "key", "value", 0)
		}
	})
}

func BenchmarkMonitoredCache_Del(b *testing.B) {
	ctx := context.Background()
	originalCache := newBenchmarkCache()
	monitoredCache := NewMonitoredCache(originalCache)

	b.Run("Original", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			originalCache.Set(ctx, "key", "value", 0)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = originalCache.Del(ctx, "key")
		}
	})

	b.Run("Monitored", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			monitoredCache.Set(ctx, "key", "value", 0)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = monitoredCache.Del(ctx, "key")
		}
	})
}

func BenchmarkMonitoredCache_WithServer(b *testing.B) {
	ctx := context.Background()
	originalCache := newBenchmarkCache()
	monitoredCache := NewMonitoredCache(originalCache, WithMetricsServer(":0"))

	err := monitoredCache.server.Start()
	if err != nil {
		b.Fatal(err)
	}
	defer monitoredCache.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = monitoredCache.Set(ctx, "key", "value", 0)
		_, _ = monitoredCache.Get(ctx, "key")
		_ = monitoredCache.Del(ctx, "key")
	}
}

func BenchmarkMonitoredCache_HighLoad(b *testing.B) {
	ctx := context.Background()
	originalCache := newBenchmarkCache()
	monitoredCache := NewMonitoredCache(originalCache)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		keyCounter := 0
		for pb.Next() {
			key := "key" + strconv.Itoa(keyCounter%100)
			_ = monitoredCache.Set(ctx, key, "value", 0)
			_, _ = monitoredCache.Get(ctx, key)
			_ = monitoredCache.Del(ctx, key)
			keyCounter++
		}
	})
}

func BenchmarkCollectMemStats(b *testing.B) {
	originalCache := newBenchmarkCache()
	monitoredCache := NewMonitoredCache(originalCache,
		WithMemoryUpdateInterval(100*time.Millisecond),
		WithEnableMemStats(true))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitoredCache.collectMemStatsOnce() // 使用非阻塞版本
	}
}

func BenchmarkMonitoredCache_HitRateCalculation(b *testing.B) {
	originalCache := newBenchmarkCache()
	monitoredCache := NewMonitoredCache(originalCache)

	for i := 0; i < 1000; i++ {
		if i%2 == 0 {
			monitoredCache.collector.IncHits()
		} else {
			monitoredCache.collector.IncMisses()
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = monitoredCache.HitRate()
	}
}
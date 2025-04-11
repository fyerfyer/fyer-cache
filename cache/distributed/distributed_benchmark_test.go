package distributed

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
)

var (
	benchCtx = context.Background()
	letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
)

func generateRandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func generateRandomBytes(size int) []byte {
	randBytes := make([]byte, size)
	for i := range randBytes {
		randBytes[i] = byte(rand.Intn(256))
	}
	return randBytes
}

type benchHTTPClient struct {
	latency time.Duration
	responses map[string]interface{}
	mu sync.RWMutex
	requestCount int
}

func newBenchHTTPClient(latency time.Duration) *benchHTTPClient {
	return &benchHTTPClient{
		latency:    latency,
		responses:  make(map[string]interface{}),
		mu:         sync.RWMutex{},
	}
}

func (c *benchHTTPClient) Do(method, url string, body, result interface{}) error {
	c.mu.Lock()
	c.requestCount++
	c.mu.Unlock()

	if c.latency > 0 {
		time.Sleep(c.latency)
	}

	if method == "GET" {
		c.mu.RLock()
		val, exists := c.responses[url]
		c.mu.RUnlock()

		if !exists {
			return fmt.Errorf("key not found")
		}

		if result != nil {
			switch v := result.(type) {
			case *map[string]interface{}:
				*v = val.(map[string]interface{})
			case *string:
				*v = val.(string)
			case *[]byte:
				*v = val.([]byte)
			}
		}
		return nil
	} else if method == "POST" {
		c.mu.Lock()
		c.responses[url] = body
		c.mu.Unlock()
		return nil
	} else if method == "DELETE" {
		c.mu.Lock()
		delete(c.responses, url)
		c.mu.Unlock()
		return nil
	}

	return fmt.Errorf("unsupported method: %s", method)
}

func setupRemoteCache(b *testing.B, latency time.Duration) *RemoteCache {
	b.Helper()
	nodeID := "node1"
	address := "localhost:8080"

	client := newBenchHTTPClient(latency)
	rc := NewRemoteCache(nodeID, address)
	rc.client = client

	return rc
}

func setupMockServerAndCache(b *testing.B) (*httptest.Server, *RemoteCache) {
	b.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true, "value": "test-value"}`))
	}))

	rc := NewRemoteCache("test-node", server.URL)
	return server, rc
}

func BenchmarkRemoteCache_Get(b *testing.B) {
	rc := setupRemoteCache(b, 0)

	client := rc.client.(*benchHTTPClient)
	client.responses["/cache/get"] = "test-value"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rc.Get(benchCtx, "test-key")
	}
}

func BenchmarkRemoteCache_Set(b *testing.B) {
	rc := setupRemoteCache(b, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rc.Set(benchCtx, fmt.Sprintf("key-%d", i), "value", time.Minute)
	}
}

func BenchmarkRemoteCache_Del(b *testing.B) {
	rc := setupRemoteCache(b, 0)

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		_ = rc.Set(benchCtx, key, "value", time.Minute)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rc.Del(benchCtx, fmt.Sprintf("key-%d", i))
	}
}

func BenchmarkRemoteCache_WithLatency(b *testing.B) {
	latencies := []time.Duration{
		1 * time.Millisecond,
		5 * time.Millisecond,
		10 * time.Millisecond,
		50 * time.Millisecond,
	}

	for _, latency := range latencies {
		b.Run(fmt.Sprintf("Latency-%s", latency), func(b *testing.B) {
			rc := setupRemoteCache(b, latency)

			client := rc.client.(*benchHTTPClient)
			client.responses["/cache/get"] = "test-value"

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = rc.Get(benchCtx, "test-key")
			}
		})
	}
}

func BenchmarkRemoteCache_Parallel(b *testing.B) {
	rc := setupRemoteCache(b, 1*time.Millisecond)
	client := rc.client.(*benchHTTPClient)

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("/cache/get?key=%d", i)
		client.responses[key] = fmt.Sprintf("value-%d", i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		for pb.Next() {
			key := strconv.Itoa(localRand.Intn(1000))
			_, _ = rc.Get(benchCtx, key)
		}
	})
}

func BenchmarkRemoteCache_ParallelMixed(b *testing.B) {
	rc := setupRemoteCache(b, 1*time.Millisecond)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		for pb.Next() {
			op := localRand.Intn(3)
			key := strconv.Itoa(localRand.Intn(1000))

			switch op {
			case 0:
				_, _ = rc.Get(benchCtx, key)
			case 1:
				_ = rc.Set(benchCtx, key, generateRandomString(localRand.Intn(100)+1), time.Minute)
			case 2:
				_ = rc.Del(benchCtx, key)
			}
		}
	})
}

func BenchmarkRemoteCache_WithRetry(b *testing.B) {
	retries := []int{0, 1, 3, 5}

	for _, retry := range retries {
		b.Run(fmt.Sprintf("Retries-%d", retry), func(b *testing.B) {
			nodeID := "node1"
			address := "localhost:8080"

			rc := NewRemoteCache(nodeID, address, WithRetry(retry, 1*time.Millisecond))
			client := newBenchHTTPClient(1 * time.Millisecond)
			rc.client = client

			client.responses["/cache/get"] = "test-value"

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = rc.Get(benchCtx, "test-key")
			}
		})
	}
}

func BenchmarkRemoteCache_WithTimeout(b *testing.B) {
	timeouts := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
	}

	for _, timeout := range timeouts {
		b.Run(fmt.Sprintf("Timeout-%s", timeout), func(b *testing.B) {
			nodeID := "node1"
			address := "localhost:8080"

			rc := NewRemoteCache(nodeID, address, WithTimeout(timeout))
			client := newBenchHTTPClient(1 * time.Millisecond)
			rc.client = client

			client.responses["/cache/get"] = "test-value"

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = rc.Get(benchCtx, "test-key")
			}
		})
	}
}

func BenchmarkRemoteCache_DifferentPayloadSizes(b *testing.B) {
	sizes := []int{
		100,     // 100 bytes
		1000,    // 1 KB
		10000,   // 10 KB
		100000,  // 100 KB
	}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size-%d", size), func(b *testing.B) {
			rc := setupRemoteCache(b, 1*time.Millisecond)

			value := generateRandomBytes(size)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = rc.Set(benchCtx, fmt.Sprintf("key-%d", i), value, time.Minute)
			}
		})
	}
}

func BenchmarkRemoteCache_RealHTTPServer(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	server, rc := setupMockServerAndCache(b)
	defer server.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rc.Get(benchCtx, "test-key")
	}
}
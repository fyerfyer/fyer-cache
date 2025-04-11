package cache

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func setupShardedCache(b *testing.B, nodeCount int, virtualNodes int, replicaFactor int) *NodeShardedCache {
	b.Helper()

	var options []NodeShardedOption
	if virtualNodes > 0 {
		options = append(options, WithHashReplicas(virtualNodes))
	}
	if replicaFactor > 1 {
		options = append(options, WithReplicaFactor(replicaFactor))
	}

	cache := NewNodeShardedCache(options...)

	for i := 0; i < nodeCount; i++ {
		nodeID := fmt.Sprintf("node-%d", i)
		err := cache.AddNode(nodeID, NewMemoryCache(), 1)
		if err != nil {
			b.Fatalf("Failed to add node %s: %v", nodeID, err)
		}
	}

	return cache
}

func populateCache(b *testing.B, cache *NodeShardedCache, itemCount int) {
	b.Helper()
	ctx := context.Background()

	for i := 0; i < itemCount; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := generateRandomValue(100)
		err := cache.Set(ctx, key, value, 10*time.Minute)
		if err != nil {
			b.Fatalf("Failed to populate cache: %v", err)
		}
	}
}

func BenchmarkNodeShardedCache_Get(b *testing.B) {
	ctx := context.Background()
	cache := setupShardedCache(b, 3, 100, 1)
	populateCache(b, cache, 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%10000)
		_, _ = cache.Get(ctx, key)
	}
}

func BenchmarkNodeShardedCache_Set(b *testing.B) {
	ctx := context.Background()
	cache := setupShardedCache(b, 3, 100, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench-key-%d", i)
		value := generateRandomValue(100)
		_ = cache.Set(ctx, key, value, 10*time.Minute)
	}
}

func BenchmarkNodeShardedCache_Del(b *testing.B) {
	ctx := context.Background()
	cache := setupShardedCache(b, 3, 100, 1)

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("del-key-%d", i)
		_ = cache.Set(ctx, key, []byte("value"), 10*time.Minute)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("del-key-%d", i)
		_ = cache.Del(ctx, key)
	}
}

func BenchmarkNodeShardedCache_Parallel(b *testing.B) {
	ctx := context.Background()
	cache := setupShardedCache(b, 3, 100, 1)

	populateCache(b, cache, 10000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", localRand.Intn(10000))
			op := localRand.Intn(3)
			switch op {
			case 0:
				_, _ = cache.Get(ctx, key)
			case 1:
				value := generateRandomValueWithRand(localRand, 100)
				_ = cache.Set(ctx, key, value, 10*time.Minute)
			case 2:
				_ = cache.Del(ctx, key)
			}
			i++
		}
	})
}

func BenchmarkNodeShardedCache_NodeCount(b *testing.B) {
	nodeCounts := []int{1, 2, 4, 8, 16}

	for _, nodeCount := range nodeCounts {
		b.Run(fmt.Sprintf("Nodes-%d", nodeCount), func(b *testing.B) {
			ctx := context.Background()
			cache := setupShardedCache(b, nodeCount, 100, 1)

			populateCache(b, cache, 10000)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("key-%d", i%10000)
				_, _ = cache.Get(ctx, key)
			}
		})
	}
}

func BenchmarkNodeShardedCache_VirtualNodes(b *testing.B) {
	virtualNodeCounts := []int{10, 50, 100, 500, 1000}

	for _, vnodes := range virtualNodeCounts {
		b.Run(fmt.Sprintf("VNodes-%d", vnodes), func(b *testing.B) {
			ctx := context.Background()
			cache := setupShardedCache(b, 3, vnodes, 1)

			populateCache(b, cache, 10000)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("key-%d", i%10000)
				_, _ = cache.Get(ctx, key)
			}
		})
	}
}

func BenchmarkNodeShardedCache_ReplicaFactor(b *testing.B) {
	replicaFactors := []int{1, 2, 3}

	for _, rf := range replicaFactors {
		b.Run(fmt.Sprintf("RF-%d", rf), func(b *testing.B) {
			ctx := context.Background()
			cache := setupShardedCache(b, 5, 100, rf)

			populateCache(b, cache, 10000)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("key-%d", i%10000)
				_, _ = cache.Get(ctx, key)
			}
		})
	}
}

func BenchmarkNodeShardedCache_KeyDistribution(b *testing.B) {
	patterns := []struct {
		name      string
		keyPrefix string
	}{
		{"Sequential", "seq-"},
		{"UserIDs", "user-"},
		{"Sessions", "sess-"},
		{"MD5Hashes", "hash-"},
		{"UUIDs", "uuid-"},
	}

	for _, pattern := range patterns {
		b.Run(pattern.name, func(b *testing.B) {
			ctx := context.Background()
			cache := setupShardedCache(b, 3, 100, 1)

			for i := 0; i < 10000; i++ {
				key := fmt.Sprintf("%s%d", pattern.keyPrefix, i)
				_ = cache.Set(ctx, key, []byte("value"), 10*time.Minute)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("%s%d", pattern.keyPrefix, i%10000)
				_, _ = cache.Get(ctx, key)
			}
		})
	}
}

func BenchmarkNodeShardedCache_HotSpots(b *testing.B) {
	hotKeyPercentages := []int{10, 50, 90}

	for _, hotPct := range hotKeyPercentages {
		b.Run(fmt.Sprintf("HotKeys-%d%%", hotPct), func(b *testing.B) {
			ctx := context.Background()
			cache := setupShardedCache(b, 3, 100, 1)

			populateCache(b, cache, 10000)

			for i := 0; i < 10; i++ {
				key := fmt.Sprintf("hot-key-%d", i)
				_ = cache.Set(ctx, key, []byte("hot-value"), 10*time.Minute)
			}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
				for pb.Next() {
					useHotKey := localRand.Intn(100) < hotPct

					var key string
					if useHotKey {
						hotKeyIndex := localRand.Intn(10)
						key = fmt.Sprintf("hot-key-%d", hotKeyIndex)
					} else {
						coldKeyIndex := localRand.Intn(10000)
						key = fmt.Sprintf("cold-key-%d", coldKeyIndex)
					}

					_, _ = cache.Get(ctx, key)
				}
			})
		})
	}
}

func BenchmarkNodeShardedCache_WeightedNodes(b *testing.B) {
	ctx := context.Background()

	cache := NewNodeShardedCache()

	err := cache.AddNode("node1", NewMemoryCache(), 1)
	if err != nil {
		b.Fatalf("Failed to add node1: %v", err)
	}
	err = cache.AddNode("node2", NewMemoryCache(), 2)
	if err != nil {
		b.Fatalf("Failed to add node2: %v", err)
	}
	err = cache.AddNode("node3", NewMemoryCache(), 4)
	if err != nil {
		b.Fatalf("Failed to add node3: %v", err)
	}

	populateCache(b, cache, 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%10000)
		_, _ = cache.Get(ctx, key)
	}
}
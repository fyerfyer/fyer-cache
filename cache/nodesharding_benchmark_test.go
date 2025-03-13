package cache

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// 基准测试部分

func BenchmarkNodeShardedCache_Get(b *testing.B) {
	ctx := context.Background()

	shardedCache := NewNodeShardedCache(WithHashReplicas(100))

	// 添加5个节点
	for i := 1; i <= 5; i++ {
		nodeName := fmt.Sprintf("node%d", i)
		err := shardedCache.AddNode(nodeName, NewMemoryCache(), 1)
		if err != nil {
			b.Fatalf("Failed to add %s: %v", nodeName, err)
		}
	}

	// 预填充10K个键
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("bench-key-%d", i)
		err := shardedCache.Set(ctx, key, fmt.Sprintf("value-%d", i), time.Hour)
		if err != nil {
			b.Fatalf("Failed to set key: %v", err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// 为每个goroutine创建随机源
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		for pb.Next() {
			key := fmt.Sprintf("bench-key-%d", r.Intn(10000))
			_, _ = shardedCache.Get(ctx, key)
		}
	})
}

func BenchmarkNodeShardedCache_Set(b *testing.B) {
	ctx := context.Background()

	shardedCache := NewNodeShardedCache(WithHashReplicas(100))

	// 添加5个节点
	for i := 1; i <= 5; i++ {
		nodeName := fmt.Sprintf("node%d", i)
		err := shardedCache.AddNode(nodeName, NewMemoryCache(), 1)
		if err != nil {
			b.Fatalf("Failed to add %s: %v", nodeName, err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// 为每个goroutine创建随机源和计数器
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		counter := 0
		for pb.Next() {
			key := fmt.Sprintf("bench-key-%d-%d", r.Intn(10000), counter)
			value := fmt.Sprintf("value-%d", counter)
			_ = shardedCache.Set(ctx, key, value, time.Hour)
			counter++
		}
	})
}

func BenchmarkNodeShardedCache_ReplicaFactor(b *testing.B) {
	ctx := context.Background()

	// 测试不同复制因子下的性能
	for _, factor := range []int{1, 2, 3} {
		b.Run(fmt.Sprintf("ReplicaFactor-%d", factor), func(b *testing.B) {
			shardedCache := NewNodeShardedCache(
				WithHashReplicas(100),
				WithReplicaFactor(factor),
			)

			// 添加5个节点
			for i := 1; i <= 5; i++ {
				nodeName := fmt.Sprintf("node%d", i)
				err := shardedCache.AddNode(nodeName, NewMemoryCache(), 1)
				if err != nil {
					b.Fatalf("Failed to add %s: %v", nodeName, err)
				}
			}

			// 预填充数据
			for i := 0; i < 1000; i++ {
				key := fmt.Sprintf("bench-key-%d", i)
				err := shardedCache.Set(ctx, key, fmt.Sprintf("value-%d", i), time.Hour)
				if err != nil {
					b.Fatalf("Failed to set key: %v", err)
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("bench-key-%d", i%1000)
				if i%10 < 8 { // 80% 读, 20% 写
					_, _ = shardedCache.Get(ctx, key)
				} else {
					_ = shardedCache.Set(ctx, key, fmt.Sprintf("new-value-%d", i), time.Hour)
				}
			}
		})
	}
}

func BenchmarkNodeShardedCache_NodeCount(b *testing.B) {
	ctx := context.Background()

	// 测试不同节点数量下的性能
	for _, nodeCount := range []int{3, 5, 10, 20} {
		b.Run(fmt.Sprintf("NodeCount-%d", nodeCount), func(b *testing.B) {
			shardedCache := NewNodeShardedCache(WithHashReplicas(100))

			// 添加指定数量的节点
			for i := 1; i <= nodeCount; i++ {
				nodeName := fmt.Sprintf("node%d", i)
				err := shardedCache.AddNode(nodeName, NewMemoryCache(), 1)
				if err != nil {
					b.Fatalf("Failed to add %s: %v", nodeName, err)
				}
			}

			// 预填充数据
			for i := 0; i < 1000; i++ {
				key := fmt.Sprintf("bench-key-%d", i)
				err := shardedCache.Set(ctx, key, fmt.Sprintf("value-%d", i), time.Hour)
				if err != nil {
					b.Fatalf("Failed to set key: %v", err)
				}
			}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				r := rand.New(rand.NewSource(time.Now().UnixNano()))
				counter := 0
				for pb.Next() {
					key := fmt.Sprintf("bench-key-%d", r.Intn(1000))
					if r.Float32() < 0.8 { // 80% 读
						_, _ = shardedCache.Get(ctx, key)
					} else { // 20% 写
						_ = shardedCache.Set(ctx, key, fmt.Sprintf("new-value-%d", counter), time.Hour)
						counter++
					}
				}
			})
		})
	}
}

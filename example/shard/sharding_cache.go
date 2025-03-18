package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
)

func main() {
	fmt.Println("Sharded Cache Example")
	fmt.Println("=====================")

	// 创建默认分片数的缓存
	defaultCache := cache.NewMemoryCache()
	fmt.Printf("Created cache with default shard count (%d shards)\n\n", cache.DefaultShardCount)

	// 基本操作示例
	demoBasicOperations(defaultCache)

	// 演示分片在高并发下的优势
	demoConcurrencyBenefits()
}

func demoBasicOperations(c cache.Cache) {
	ctx := context.Background()

	fmt.Println("Basic Cache Operations:")
	fmt.Println("----------------------")

	// 设置一些值
	fmt.Println("Setting values...")
	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		err := c.Set(ctx, key, value, 5*time.Minute)
		if err != nil {
			fmt.Printf("Failed to set %s: %v\n", key, err)
		}
	}

	// 获取值
	fmt.Println("Getting values...")
	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("key%d", i)
		value, err := c.Get(ctx, key)
		if err != nil {
			fmt.Printf("Failed to get %s: %v\n", key, err)
			continue
		}
		fmt.Printf("  %s = %s\n", key, value)
	}

	// 删除一个值
	fmt.Println("Deleting key3...")
	err := c.Del(ctx, "key3")
	if err != nil {
		fmt.Printf("Failed to delete key3: %v\n", err)
	}

	// 再次获取，确认删除成功
	_, err = c.Get(ctx, "key3")
	if err != nil {
		fmt.Printf("  key3 was successfully deleted: %v\n", err)
	}

	fmt.Println()
}

func demoConcurrencyBenefits() {
	fmt.Println("Demonstrating Sharding Benefits:")
	fmt.Println("-------------------------------")

	// 对比单分片和多分片缓存在高并发下的性能
	singleShardCache := cache.NewMemoryCache(cache.WithShardCount(1))
	multiShardCache := cache.NewMemoryCache() // 默认分片数

	// 测试参数
	itemCount := 10000
	workerCount := 50

	fmt.Printf("Running concurrent test with %d items and %d workers\n", itemCount, workerCount)

	// 测试单分片缓存
	fmt.Println("\nTesting single shard cache...")
	singleTime := runConcurrentTest(singleShardCache, itemCount, workerCount)

	// 测试多分片缓存
	fmt.Println("\nTesting multi-shard cache...")
	multiTime := runConcurrentTest(multiShardCache, itemCount, workerCount)

	// 显示性能比较
	fmt.Println("\nResults:")
	fmt.Printf("  Single shard time: %v\n", singleTime)
	fmt.Printf("  Multi shard time:  %v\n", multiTime)
	fmt.Printf("  Performance improvement: %.2fx faster with sharding\n",
		float64(singleTime)/float64(multiTime))
}

func runConcurrentTest(c cache.Cache, itemCount, workerCount int) time.Duration {
	ctx := context.Background()
	var wg sync.WaitGroup

	itemsPerWorker := itemCount / workerCount

	startTime := time.Now()

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// 计算此worker的项目范围
			startIdx := workerID * itemsPerWorker
			endIdx := startIdx + itemsPerWorker

			for i := startIdx; i < endIdx; i++ {
				key := fmt.Sprintf("test-key-%d", i)
				value := fmt.Sprintf("value-%d", i)

				// 执行Set和Get操作
				_ = c.Set(ctx, key, value, time.Minute)
				_, _ = c.Get(ctx, key)
				_ = c.Set(ctx, key, "updated-"+value, time.Minute)
				_, _ = c.Get(ctx, key)
			}
		}(w)
	}

	wg.Wait()
	return time.Since(startTime)
}
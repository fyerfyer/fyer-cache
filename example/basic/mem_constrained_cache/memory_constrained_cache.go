package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
)

func main() {
	// 创建一个上下文对象
	ctx := context.Background()

	// 追踪被淘汰的键
	var evictedKeys []string

	// 创建一个基础内存缓存，并设置淘汰回调
	memCache := cache.NewMemoryCache(
		cache.WithEvictionCallback(func(key string, value any) {
			evictedKeys = append(evictedKeys, key)
			fmt.Printf("Item evicted: key=%s\n", key)
		}),
	)

	// 创建一个内存限制为 5KB 的缓存包装器
	// MemCacheMemory 会在内存用量超过限制时自动淘汰最久未使用的项目
	memoryLimitBytes := int64(5 * 1024) // 5KB
	constrainedCache := cache.NewMemCacheMemory(memCache, memoryLimitBytes)

	fmt.Printf("Created memory constrained cache with %d bytes limit\n", memoryLimitBytes)

	// 添加一些数据，并打印当前使用的内存
	fmt.Println("\n--- Adding items to cache ---")
	for i := 1; i <= 10; i++ {
		// 创建不同大小的数据（约1KB每项）
		data := make([]byte, 1000+i)
		for j := range data {
			data[j] = byte(i)
		}

		key := fmt.Sprintf("key-%d", i)
		err := constrainedCache.Set(ctx, key, data, 10*time.Minute)
		if err != nil {
			log.Printf("Failed to set item %s: %v", key, err)
			continue
		}

		// 打印当前内存使用情况
		memUsage := constrainedCache.MemoryUsage()
		fmt.Printf("Added %s: size=%d bytes, total memory usage: %d/%d bytes (%.1f%%)\n",
			key, len(data), memUsage, memoryLimitBytes, float64(memUsage*100)/float64(memoryLimitBytes))
	}

	// 展示被淘汰的键
	fmt.Printf("\n--- LRU Eviction Results ---\n")
	fmt.Printf("Evicted keys: %v\n", evictedKeys)

	// 尝试访问一些键，验证有些已被淘汰
	fmt.Println("\n--- Checking cached items ---")
	for i := 1; i <= 10; i++ {
		key := fmt.Sprintf("key-%d", i)
		_, err := constrainedCache.Get(ctx, key)
		if err != nil {
			fmt.Printf("Item %s: not found (evicted)\n", key)
		} else {
			fmt.Printf("Item %s: still in cache\n", key)
		}
	}

	// 展示LRU机制 - 访问某个项使其变为"最近使用"
	fmt.Println("\n--- Demonstrating LRU behavior ---")

	// 获取当前在缓存中的第一个键
	var existingKey string
	for i := 1; i <= 10; i++ {
		key := fmt.Sprintf("key-%d", i)
		if _, err := constrainedCache.Get(ctx, key); err == nil {
			existingKey = key
			break
		}
	}

	if existingKey != "" {
		// 频繁访问这个键，使其变为"最近使用"
		fmt.Printf("Accessing %s multiple times to make it recently used\n", existingKey)
		for i := 0; i < 5; i++ {
			constrainedCache.Get(ctx, existingKey)
		}

		// 添加一个大值触发淘汰
		largeValue := make([]byte, 2000)
		fmt.Println("Adding large item to trigger eviction...")
		err := constrainedCache.Set(ctx, "large-item", largeValue, 10*time.Minute)
		if err != nil {
			log.Printf("Failed to set large item: %v", err)
		}

		// 检查我们频繁访问的键是否仍在缓存中
		_, err = constrainedCache.Get(ctx, existingKey)
		if err != nil {
			fmt.Printf("%s was evicted despite being recently used\n", existingKey)
		} else {
			fmt.Printf("%s was kept in cache because it was recently used\n", existingKey)
		}

		// 显示最终内存使用情况
		fmt.Printf("\nFinal memory usage: %d/%d bytes (%.1f%%)\n",
			constrainedCache.MemoryUsage(), memoryLimitBytes,
			float64(constrainedCache.MemoryUsage()*100)/float64(memoryLimitBytes))
	}

	// 测试对象大小估计
	fmt.Println("\n--- Testing object size estimation ---")

	// 字符串
	stringVal := "This is a test string"
	constrainedCache.Set(ctx, "string-key", stringVal, time.Minute)

	// 结构体
	type TestStruct struct {
		ID   int
		Name string
		Data []byte
	}
	structVal := TestStruct{
		ID:   123,
		Name: "Test Object",
		Data: make([]byte, 100),
	}
	constrainedCache.Set(ctx, "struct-key", structVal, time.Minute)

	// 显示最终内存使用情况
	fmt.Printf("Final memory usage after adding different types: %d bytes\n",
		constrainedCache.MemoryUsage())
}
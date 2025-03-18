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

	// 创建一个新的内存缓存实例，使用默认配置
	memCache := cache.NewMemoryCache()
	fmt.Println("Memory cache created successfully")

	// 设置缓存项，带有1分钟的过期时间
	err := memCache.Set(ctx, "user:1001", "John Doe", 1*time.Minute)
	if err != nil {
		log.Fatalf("Failed to set cache item: %v", err)
	}
	fmt.Println("Added string item to cache with 1 minute expiration")

	// 设置结构体值到缓存
	type User struct {
		ID    int
		Name  string
		Email string
	}
	user := User{
		ID:    1002,
		Name:  "Jane Smith",
		Email: "jane@example.com",
	}
	err = memCache.Set(ctx, "user:1002", user, 2*time.Minute)
	if err != nil {
		log.Fatalf("Failed to set struct cache item: %v", err)
	}
	fmt.Println("Added struct item to cache with 2 minutes expiration")

	// 从缓存获取字符串值
	val, err := memCache.Get(ctx, "user:1001")
	if err != nil {
		log.Fatalf("Failed to get cache item: %v", err)
	}
	name, ok := val.(string)
	if !ok {
		log.Fatalf("Unexpected type for user:1001")
	}
	fmt.Printf("Retrieved string value: %s\n", name)

	// 从缓存获取结构体值
	val, err = memCache.Get(ctx, "user:1002")
	if err != nil {
		log.Fatalf("Failed to get cache item: %v", err)
	}
	cachedUser, ok := val.(User)
	if !ok {
		log.Fatalf("Unexpected type for user:1002")
	}
	fmt.Printf("Retrieved user: ID=%d, Name=%s, Email=%s\n",
		cachedUser.ID, cachedUser.Name, cachedUser.Email)

	// 尝试获取不存在的键
	val, err = memCache.Get(ctx, "non-existent-key")
	if err != nil {
		fmt.Printf("Expected error for non-existent key: %v\n", err)
	}

	// 删除缓存项
	err = memCache.Del(ctx, "user:1001")
	if err != nil {
		log.Fatalf("Failed to delete cache item: %v", err)
	}
	fmt.Println("Deleted cache item user:1001")

	// 验证删除是否成功
	_, err = memCache.Get(ctx, "user:1001")
	if err != nil {
		fmt.Printf("Item was successfully deleted: %v\n", err)
	}

	// 设置较短过期时间的缓存项，然后等待过期
	err = memCache.Set(ctx, "short-lived", "This will expire soon", 2*time.Second)
	if err != nil {
		log.Fatalf("Failed to set short-lived item: %v", err)
	}
	fmt.Println("Added item with 2 seconds expiration")

	// 等待超过过期时间
	fmt.Println("Waiting for cache item to expire...")
	time.Sleep(3 * time.Second)

	// 尝试获取过期项
	_, err = memCache.Get(ctx, "short-lived")
	if err != nil {
		fmt.Printf("Item has expired as expected: %v\n", err)
	}

	// 使用自定义淘汰回调创建新的缓存
	evictionCache := cache.NewMemoryCache(
		cache.WithEvictionCallback(func(key string, value any) {
			fmt.Printf("Eviction callback: item with key '%s' was evicted\n", key)
		}),
		cache.WithCleanupInterval(500 * time.Millisecond),
	)

	// 设置快速过期的项目来测试回调
	err = evictionCache.Set(ctx, "callback-test", "Testing callbacks", 1*time.Second)
	if err != nil {
		log.Fatalf("Failed to set callback test item: %v", err)
	}
	fmt.Println("Waiting for eviction callback...")
	time.Sleep(2 * time.Second)

	// 关闭缓存，清理资源
	err = memCache.Close()
	if err != nil {
		log.Fatalf("Error closing cache: %v", err)
	}
	fmt.Println("Cache closed successfully")
}
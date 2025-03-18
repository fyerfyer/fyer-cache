package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/fyerfyer/fyer-cache/cache/consistency"
)

// MockDatabase 模拟一个简单的数据库，作为缓存的后端数据源
type MockDatabase struct {
	data  map[string]interface{}
	mutex sync.RWMutex
	// 用于演示目的的访问延迟
	accessDelay time.Duration
}

// NewMockDatabase 创建一个新的模拟数据库实例
func NewMockDatabase(accessDelay time.Duration) *MockDatabase {
	return &MockDatabase{
		data:        make(map[string]interface{}),
		accessDelay: accessDelay,
	}
}

// 实现 DataSource 接口的方法

// Load 从数据库加载数据
func (db *MockDatabase) Load(ctx context.Context, key string) (interface{}, error) {
	// 模拟数据库访问延迟
	time.Sleep(db.accessDelay)

	db.mutex.RLock()
	defer db.mutex.RUnlock()

	value, exists := db.data[key]
	if !exists {
		return nil, fmt.Errorf("key %s not found in database", key)
	}
	fmt.Printf("Database: Loaded %s from database\n", key)
	return value, nil
}

// Store 将数据存入数据库
func (db *MockDatabase) Store(ctx context.Context, key string, value interface{}) error {
	// 模拟数据库访问延迟
	time.Sleep(db.accessDelay)

	db.mutex.Lock()
	defer db.mutex.Unlock()

	db.data[key] = value
	fmt.Printf("Database: Stored %s in database\n", key)
	return nil
}

// Remove 从数据库删除数据
func (db *MockDatabase) Remove(ctx context.Context, key string) error {
	// 模拟数据库访问延迟
	time.Sleep(db.accessDelay)

	db.mutex.Lock()
	defer db.mutex.Unlock()

	delete(db.data, key)
	fmt.Printf("Database: Removed %s from database\n", key)
	return nil
}

func main() {
	ctx := context.Background()

	// 创建一个内存缓存实例
	memCache := cache.NewMemoryCache()
	fmt.Println("Created memory cache")

	// 创建一个模拟数据库实例，访问延迟设为500毫秒
	db := NewMockDatabase(500 * time.Millisecond)
	fmt.Println("Created mock database with 500ms access delay")

	// 预先在数据库中存储一些数据
	db.Store(ctx, "user:1", "Alice")
	db.Store(ctx, "user:2", "Bob")
	db.Store(ctx, "user:3", "Charlie")

	// 创建 Cache-Aside 策略实例
	cacheAside := consistency.NewCacheAsideStrategy(
		memCache,
		db,
		consistency.WithCacheAsideTTL(5*time.Minute),
		consistency.WithCacheAsideWriteOnMiss(true),
	)
	fmt.Println("Created Cache-Aside strategy")

	// 场景1：第一次读取数据（缓存未命中）
	fmt.Println("\n--- Scenario 1: First Read (Cache Miss) ---")
	startTime := time.Now()
	user1, err := cacheAside.Get(ctx, "user:1")
	if err != nil {
		log.Fatalf("Failed to get user:1: %v", err)
	}
	elapsed := time.Since(startTime)
	fmt.Printf("Retrieved user:1 = %v (took %v)\n", user1, elapsed)

	// 场景2：再次读取相同数据（缓存命中）
	fmt.Println("\n--- Scenario 2: Second Read (Cache Hit) ---")
	startTime = time.Now()
	user1Again, err := cacheAside.Get(ctx, "user:1")
	if err != nil {
		log.Fatalf("Failed to get user:1 again: %v", err)
	}
	elapsed = time.Since(startTime)
	fmt.Printf("Retrieved user:1 again = %v (took %v)\n", user1Again, elapsed)

	// 场景3：写入新数据
	fmt.Println("\n--- Scenario 3: Writing New Data ---")
	err = cacheAside.Set(ctx, "user:4", "Dave", 5*time.Minute)
	if err != nil {
		log.Fatalf("Failed to set user:4: %v", err)
	}
	fmt.Println("Added user:4 = Dave to cache and database")

	// 验证数据已写入缓存和数据库
	startTime = time.Now()
	user4, err := cacheAside.Get(ctx, "user:4")
	if err != nil {
		log.Fatalf("Failed to get user:4: %v", err)
	}
	elapsed = time.Since(startTime)
	fmt.Printf("Retrieved user:4 = %v (took %v)\n", user4, elapsed)

	// 场景4：删除数据
	fmt.Println("\n--- Scenario 4: Deleting Data ---")
	err = cacheAside.Del(ctx, "user:2")
	if err != nil {
		log.Fatalf("Failed to delete user:2: %v", err)
	}
	fmt.Println("Deleted user:2 from cache and database")

	// 尝试获取已删除的数据
	fmt.Println("\n--- Scenario 5: Retrieving Deleted Data ---")
	_, err = cacheAside.Get(ctx, "user:2")
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	} else {
		fmt.Println("Unexpectedly found user:2, which should have been deleted")
	}

	// 场景6：读取不存在的数据
	fmt.Println("\n--- Scenario 6: Reading Non-existent Data ---")
	_, err = cacheAside.Get(ctx, "user:999")
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	} else {
		fmt.Println("Unexpectedly found user:999, which should not exist")
	}

	// 场景7：使用Invalidate方法使缓存失效但不影响数据源
	fmt.Println("\n--- Scenario 7: Cache Invalidation ---")
	err = cacheAside.Invalidate(ctx, "user:3")
	if err != nil {
		log.Fatalf("Failed to invalidate user:3: %v", err)
	}
	fmt.Println("Invalidated user:3 in cache (still in database)")

	// 再次获取user:3，应该会从数据库重新加载
	startTime = time.Now()
	user3, err := cacheAside.Get(ctx, "user:3")
	if err != nil {
		log.Fatalf("Failed to get user:3: %v", err)
	}
	elapsed = time.Since(startTime)
	fmt.Printf("Retrieved user:3 = %v after invalidation (took %v)\n", user3, elapsed)

	fmt.Println("\nCache-Aside pattern demonstration completed")
}
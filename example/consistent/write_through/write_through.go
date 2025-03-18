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

// MockDatabase 模拟一个数据库实现DataSource接口
type MockDatabase struct {
	data  map[string]interface{}
	mutex sync.RWMutex
	// 为了演示目的，跟踪数据库操作
	operationCount map[string]int
	// 模拟数据库访问延迟
	accessDelay time.Duration
}

// NewMockDatabase 创建一个新的模拟数据库
func NewMockDatabase(accessDelay time.Duration) *MockDatabase {
	return &MockDatabase{
		data:           make(map[string]interface{}),
		operationCount: map[string]int{"load": 0, "store": 0, "remove": 0},
		accessDelay:    accessDelay,
	}
}

// Load 从数据库加载数据
func (db *MockDatabase) Load(ctx context.Context, key string) (interface{}, error) {
	// 模拟数据库访问延迟
	time.Sleep(db.accessDelay)

	db.mutex.RLock()
	defer db.mutex.RUnlock()

	db.operationCount["load"]++
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

	db.operationCount["store"]++
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

	db.operationCount["remove"]++
	delete(db.data, key)
	fmt.Printf("Database: Removed %s from database\n", key)
	return nil
}

// GetOperationCounts 返回操作统计
func (db *MockDatabase) GetOperationCounts() map[string]int {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	// 创建一个副本避免并发问题
	counts := make(map[string]int)
	for k, v := range db.operationCount {
		counts[k] = v
	}
	return counts
}

func main() {
	ctx := context.Background()

	// 创建一个内存缓存实例
	memCache := cache.NewMemoryCache()
	fmt.Println("Created memory cache")

	// 创建一个模拟数据库实例，设置500毫秒访问延迟
	db := NewMockDatabase(500 * time.Millisecond)
	fmt.Println("Created mock database with 500ms access delay")

	// 预先在数据库中存储一些数据
	db.Store(ctx, "user:1", "Alice")
	db.Store(ctx, "user:2", "Bob")
	db.Store(ctx, "user:3", "Charlie")
	fmt.Println("Preloaded 3 users into the database")

	// 创建Write-Through策略实例(默认同步写入)
	writeThrough := consistency.NewWriteThroughStrategy(
		memCache,
		db,
		consistency.WithWriteThroughTTL(5*time.Minute),
	)
	fmt.Println("Created Write-Through strategy with synchronous writes")

	// 场景1：第一次读取一个存在于数据库中的键
	fmt.Println("\n--- Scenario 1: First Read (Cache Miss) ---")
	startTime := time.Now()
	user1, err := writeThrough.Get(ctx, "user:1")
	if err != nil {
		log.Fatalf("Failed to get user:1: %v", err)
	}
	elapsed := time.Since(startTime)
	fmt.Printf("Retrieved user:1 = %v (took %v)\n", user1, elapsed)
	fmt.Printf("Database operation counts: %v\n", db.GetOperationCounts())

	// 场景2：再次读取相同的键(应该从缓存命中)
	fmt.Println("\n--- Scenario 2: Second Read (Cache Hit) ---")
	startTime = time.Now()
	user1Again, err := writeThrough.Get(ctx, "user:1")
	if err != nil {
		log.Fatalf("Failed to get user:1 again: %v", err)
	}
	elapsed = time.Since(startTime)
	fmt.Printf("Retrieved user:1 again = %v (took %v)\n", user1Again, elapsed)
	fmt.Printf("Database operation counts: %v\n", db.GetOperationCounts())

	// 场景3：使用Write-Through写入新数据
	fmt.Println("\n--- Scenario 3: Writing New Data (Synchronous) ---")
	startTime = time.Now()
	err = writeThrough.Set(ctx, "user:4", "Dave", 5*time.Minute)
	if err != nil {
		log.Fatalf("Failed to set user:4: %v", err)
	}
	elapsed = time.Since(startTime)
	fmt.Printf("Added user:4 = Dave (took %v)\n", elapsed)
	fmt.Printf("Database operation counts: %v\n", db.GetOperationCounts())

	// 验证数据同时存在于缓存和数据库中
	startTime = time.Now()
	user4, err := writeThrough.Get(ctx, "user:4")
	if err != nil {
		log.Fatalf("Failed to get user:4: %v", err)
	}
	elapsed = time.Since(startTime)
	fmt.Printf("Retrieved user:4 = %v (took %v, should be fast from cache)\n", user4, elapsed)

	// 场景4：创建带延迟写入的Write-Through实例
	fmt.Println("\n--- Scenario 4: Delayed Write Configuration ---")
	delayedWriteThrough := consistency.NewWriteThroughStrategy(
		memCache,
		db,
		consistency.WithWriteThroughTTL(5*time.Minute),
		consistency.WithDelayedWrite(true, 200*time.Millisecond),
	)
	fmt.Println("Created Write-Through strategy with delayed writes (200ms)")

	// 场景5：使用延迟写入策略
	fmt.Println("\n--- Scenario 5: Writing with Delayed Write ---")
	startTime = time.Now()
	err = delayedWriteThrough.Set(ctx, "user:5", "Eve", 5*time.Minute)
	if err != nil {
		log.Fatalf("Failed to set user:5: %v", err)
	}
	elapsed = time.Since(startTime)
	fmt.Printf("Added user:5 = Eve (took %v, should be fast, only updated cache)\n", elapsed)

	// 验证数据立即存在于缓存中
	user5, err := delayedWriteThrough.Get(ctx, "user:5")
	if err != nil {
		log.Fatalf("Failed to get user:5: %v", err)
	}
	fmt.Printf("Retrieved user:5 from cache = %v\n", user5)

	// 等待一段时间，确保异步写入完成
	fmt.Println("Waiting for delayed write to complete...")
	time.Sleep(300 * time.Millisecond)
	fmt.Printf("Database operation counts: %v\n", db.GetOperationCounts())

	// 场景6：删除数据
	fmt.Println("\n--- Scenario 6: Deleting Data ---")
	startTime = time.Now()
	err = writeThrough.Del(ctx, "user:2")
	if err != nil {
		log.Fatalf("Failed to delete user:2: %v", err)
	}
	elapsed = time.Since(startTime)
	fmt.Printf("Deleted user:2 (took %v)\n", elapsed)
	fmt.Printf("Database operation counts: %v\n", db.GetOperationCounts())

	// 验证数据已从缓存和数据库中删除
	_, err = writeThrough.Get(ctx, "user:2")
	if err != nil {
		fmt.Printf("Verification: user:2 was deleted, error: %v\n", err)
	} else {
		fmt.Println("Unexpected: user:2 still exists")
	}

	// 场景7：使用Invalidate()只从缓存删除而不影响数据源
	fmt.Println("\n--- Scenario 7: Cache Invalidation ---")
	err = writeThrough.Invalidate(ctx, "user:3")
	if err != nil {
		log.Fatalf("Failed to invalidate user:3: %v", err)
	}
	fmt.Println("Invalidated user:3 in cache (still in database)")

	// 读取会触发从数据库重新加载
	startTime = time.Now()
	user3, err := writeThrough.Get(ctx, "user:3")
	if err != nil {
		log.Fatalf("Failed to get user:3: %v", err)
	}
	elapsed = time.Since(startTime)
	fmt.Printf("Retrieved user:3 = %v after invalidation (took %v)\n", user3, elapsed)
	fmt.Printf("Database operation counts: %v\n", db.GetOperationCounts())

	fmt.Println("\nWrite-Through cache strategy demonstration completed")
}
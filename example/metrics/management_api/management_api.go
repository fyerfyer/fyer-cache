package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/fyerfyer/fyer-cache/cache/api"
	"github.com/fyerfyer/fyer-cache/cache/metrics"
)

func main() {
	fmt.Println("Cache Management API Example")
	fmt.Println("===========================")

	ctx := context.Background()

	// 创建一个基本内存缓存
	memCache := cache.NewMemoryCache()

	// 添加一些测试数据
	for i := 1; i <= 10; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		err := memCache.Set(ctx, key, value, 1*time.Hour)
		if err != nil {
			log.Fatalf("Failed to set cache item: %v", err)
		}
	}

	fmt.Println("Added 10 items to cache")

	// 创建带监控的缓存
	monitoredCache := metrics.NewMonitoredCache(
		memCache,
		metrics.WithMetricsServer(":8081"),
		metrics.WithCollectInterval(5*time.Second),
	)

	fmt.Printf("Started metrics server on http://localhost:8081/metrics\n")

	// 创建管理API服务器
	apiServer := api.NewAPIServer(
		monitoredCache,
		api.WithBindAddress(":8080"),
		api.WithBasePath("/api/cache"),
	)

	// 启动API服务器
	err := apiServer.Start()
	if err != nil {
		log.Fatalf("Failed to start API server: %v", err)
	}
	defer apiServer.Stop()

	fmt.Printf("Started management API server on http://localhost:8080\n")
	fmt.Println("\nAvailable endpoints:")
	fmt.Println("  GET  /api/cache/stats      - Get cache statistics")
	fmt.Println("  POST /api/cache/invalidate?key={key} - Invalidate specific cache key")

	// 演示访问管理API的例子
	fmt.Println("\nDemonstrating API usage...")

	// 等待API服务器启动
	time.Sleep(1 * time.Second)

	// 展示获取缓存统计信息
	fmt.Println("\n1. Fetch cache statistics:")
	statsData := fetchStats()
	printJSON(statsData)

	// 展示缓存失效
	fmt.Println("\n2. Invalidate a cache key:")
	key := "key5"
	invalidateKey(key)

	// 验证缓存项已被删除
	value, err := monitoredCache.Get(ctx, key)
	if err != nil {
		fmt.Printf("Verified: key5 was successfully invalidated (error: %v)\n", err)
	} else {
		fmt.Printf("Unexpected: key5 still exists with value: %v\n", value)
	}

	// 再次获取统计信息
	fmt.Println("\n3. Fetch updated statistics after invalidation:")
	statsData = fetchStats()
	printJSON(statsData)

	// 模拟一些缓存操作，以生成更多指标
	fmt.Println("\nGenerating more cache operations to demonstrate metrics...")

	// 执行一些 Get 操作，包括一些缓存命中和未命中
	for i := 1; i <= 15; i++ {
		key := fmt.Sprintf("key%d", i) // 注意只添加了10个，所以有些会未命中
		_, _ = monitoredCache.Get(ctx, key)
	}

	// 添加一些新数据
	for i := 11; i <= 20; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		_ = monitoredCache.Set(ctx, key, value, 1*time.Hour)
	}

	// 等待指标收集
	time.Sleep(6 * time.Second)

	// 再次获取统计信息，查看命中/未命中率
	fmt.Println("\n4. Fetch statistics showing hit/miss rates:")
	statsData = fetchStats()
	printJSON(statsData)

	fmt.Println("\nManagement API example is running.")
	fmt.Println("Press Ctrl+C to exit.")

	// 等待中断信号
	waitForSignal()
}

// fetchStats 从API获取缓存统计信息
func fetchStats() map[string]interface{} {
	resp, err := http.Get("http://localhost:8080/api/cache/stats")
	if err != nil {
		log.Printf("Failed to fetch stats: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error response from stats endpoint: %d", resp.StatusCode)
		return nil
	}

	var data map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		log.Printf("Failed to decode response: %v", err)
		return nil
	}

	return data
}

// invalidateKey 使缓存键失效
func invalidateKey(key string) {
	url := fmt.Sprintf("http://localhost:8080/api/cache/invalidate?key=%s", key)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		log.Printf("Failed to invalidate key: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error response from invalidate endpoint: %d", resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Response body: %s", body)
		return
	}

	var data map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		log.Printf("Failed to decode response: %v", err)
		return
	}

	fmt.Printf("Successfully invalidated key '%s'\n", key)
	fmt.Printf("Response: %v\n", data)
}

// printJSON 打印JSON数据
func printJSON(data interface{}) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("Failed to format JSON: %v", err)
		return
	}
	fmt.Println(string(jsonData))
}

// waitForSignal 等待中断信号
func waitForSignal() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	fmt.Println("\nShutting down...")
}
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/fyerfyer/fyer-cache/cache/metrics"
)

func main() {
	fmt.Println("Cache Metrics Collection Example")
	fmt.Println("===============================")

	// 创建一个基本的内存缓存实例
	memCache := cache.NewMemoryCache()

	// 创建一个带监控功能的缓存装饰器
	// 我们使用默认选项，但也可以自定义配置
	monitoredCache := metrics.NewMonitoredCache(
		memCache,
		// 自定义指标服务器地址
		metrics.WithMetricsServer(":8080"),
		// 设置指标收集间隔
		metrics.WithCollectInterval(5*time.Second),
		// 设置内存使用更新间隔
		metrics.WithMemoryUpdateInterval(10*time.Second),
		// 添加自定义标签
		metrics.WithLabels(map[string]string{
			"environment": "demo",
			"region":      "example",
		}),
	)

	// 创建上下文对象
	ctx := context.Background()

	// 设置一些数据，生成指标
	fmt.Println("\nAdding data to cache to generate metrics...")
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		err := monitoredCache.Set(ctx, key, value, 10*time.Minute)
		if err != nil {
			log.Printf("Failed to set %s: %v", key, err)
		}
	}

	// 读取一些数据，触发命中和未命中指标
	fmt.Println("Reading data to generate hit/miss metrics...")
	for i := 0; i < 150; i++ { // 读取150次，保证有一些未命中
		key := fmt.Sprintf("key-%d", i)
		_, err := monitoredCache.Get(ctx, key)
		if err != nil {
			// 不记录错误，因为我们预期会有一些未命中
		}
	}

	// 删除一些数据，触发删除指标
	fmt.Println("Deleting some data to generate delete operation metrics...")
	for i := 0; i < 30; i++ {
		key := fmt.Sprintf("key-%d", i)
		err := monitoredCache.Del(ctx, key)
		if err != nil {
			log.Printf("Failed to delete %s: %v", key, err)
		}
	}

	// 展示如何获取当前命中率
	hitRate := monitoredCache.HitRate()
	fmt.Printf("\nCurrent cache hit rate: %.2f%%\n", hitRate*100)

	// 提供关于如何访问指标的信息
	fmt.Println("\nMetrics server started on :8080")
	fmt.Println("You can access metrics at: http://localhost:8080/metrics")
	fmt.Println("Health check endpoint: http://localhost:8080/health")

	// 生成一些背景流量以便持续更新指标
	go generateBackgroundTraffic(ctx, monitoredCache)

	// 等待中断信号退出
	waitForSignal()

	// 关闭缓存资源
	fmt.Println("\nShutting down...")

	// 给监控缓存一个关闭的方法调用
	// 注意这里应该实际关闭监控功能，但MonitoredCache可能没有Stop方法
	// 如果有的话，应该调用monitoredCache.Stop()

	// 关闭底层的内存缓存
	if err := memCache.Close(); err != nil {
		log.Printf("Error closing cache: %v", err)
	}
}

// generateBackgroundTraffic 生成连续的缓存操作以更新指标
func generateBackgroundTraffic(ctx context.Context, c cache.Cache) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// 计数器，用于生成不同的键
	counter := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 周期性地执行各种缓存操作

			// 写操作
			key := fmt.Sprintf("background-key-%d", counter)
			value := fmt.Sprintf("background-value-%d", counter)
			err := c.Set(ctx, key, value, 5*time.Minute)
			if err != nil {
				log.Printf("Background set error: %v", err)
			}

			// 读操作 - 一些命中，一些未命中
			readKey := fmt.Sprintf("background-key-%d", counter-10)
			_, _ = c.Get(ctx, readKey)  // 忽略错误，因为有些键可能不存在

			// 删除操作 - 偶尔删除一些较旧的键
			if counter%5 == 0 {
				deleteKey := fmt.Sprintf("background-key-%d", counter-20)
				_ = c.Del(ctx, deleteKey)  // 忽略错误
			}

			// 增加计数器
			counter++
		}
	}
}

// waitForSignal 等待中断信号
func waitForSignal() {
	fmt.Println("\nServer is running. Press Ctrl+C to stop.")

	// 创建一个通道来接收信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// 等待信号
	<-sigCh
}
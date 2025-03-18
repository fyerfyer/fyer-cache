package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fyerfyer/fyer-cache/cache/distributed"
)

func main() {
	// 创建上下文对象
	ctx := context.Background()

	// 设置远程缓存节点信息
	nodeID := "remote-node-1"
	nodeAddress := "localhost:8080" // 实际应用中应替换为真实的远程节点地址

	fmt.Println("Remote Cache Client Example")
	fmt.Println("===========================")

	// 基本用法示例
	fmt.Println("\n1. Basic Usage")
	fmt.Println("-------------")

	// 创建远程缓存客户端 - 使用默认配置
	remoteCache := distributed.NewRemoteCache(nodeID, nodeAddress)

	fmt.Printf("Created remote cache client for node '%s' at %s\n", nodeID, nodeAddress)

	// 设置缓存项，设置1分钟过期
	err := remoteCache.Set(ctx, "user:1001", "John Doe", 1*time.Minute)
	if err != nil {
		log.Printf("Failed to set cache item: %v", err)
	} else {
		fmt.Println("Successfully set 'user:1001' on remote node")
	}

	// 获取缓存项
	val, err := remoteCache.Get(ctx, "user:1001")
	if err != nil {
		log.Printf("Failed to get cache item: %v", err)
	} else {
		fmt.Printf("Retrieved value for 'user:1001': %v\n", val)
	}

	// 删除缓存项
	err = remoteCache.Del(ctx, "user:1001")
	if err != nil {
		log.Printf("Failed to delete cache item: %v", err)
	} else {
		fmt.Println("Successfully deleted 'user:1001' from remote node")
	}

	// 使用自定义配置的示例
	fmt.Println("\n2. Custom Configuration")
	fmt.Println("----------------------")

	// 创建带自定义选项的远程缓存客户端
	customCache := distributed.NewRemoteCache(
		"remote-node-2",
		"localhost:8081",
		distributed.WithTimeout(500*time.Millisecond),   // 较短的超时时间
		distributed.WithRetry(5, 200*time.Millisecond),  // 更多重试次数和间隔
	)

	fmt.Println("Created remote cache client with custom configuration:")
	fmt.Println(" - Shorter timeout (500ms)")
	fmt.Println(" - More retries (5 attempts, 200ms interval)")

	// 尝试使用自定义配置的客户端
	err = customCache.Set(ctx, "config:app", map[string]string{"env": "production"}, 10*time.Minute)
	if err != nil {
		log.Printf("Failed to set structured data: %v", err)
	} else {
		fmt.Println("Successfully set structured data on remote node")
	}

	// 错误处理示例
	fmt.Println("\n3. Error Handling")
	fmt.Println("----------------")

	// 创建指向不存在节点的客户端，用于演示错误处理
	unavailableCache := distributed.NewRemoteCache(
		"unavailable-node",
		"non-existent-host:9999",
		distributed.WithTimeout(300*time.Millisecond),  // 短超时使错误快速返回
		distributed.WithRetry(2, 100*time.Millisecond), // 少量重试以加快示例
		distributed.WithDebugLog(true),
	)

	fmt.Println("Attempting operations on an unavailable node...")

	// 尝试设置值到不可用节点
	err = unavailableCache.Set(ctx, "test:key", "test-value", time.Minute)
	if err != nil {
		fmt.Println("Error handled gracefully:")
		fmt.Printf(" - %v\n", err)
	}

	// 尝试从不可用节点获取值
	_, err = unavailableCache.Get(ctx, "test:key")
	if err != nil {
		fmt.Println("Get error handled gracefully:")
		fmt.Printf(" - %v\n", err)
	}

	// 演示本地操作的限制
	fmt.Println("\n4. Local Operations Limitations")
	fmt.Println("------------------------------")

	_, err = remoteCache.GetLocal(ctx, "local:key")
	if err != nil {
		fmt.Printf("GetLocal not supported: %v\n", err)
	}

	err = remoteCache.SetLocal(ctx, "local:key", "value", time.Minute)
	if err != nil {
		fmt.Printf("SetLocal not supported: %v\n", err)
	}

	err = remoteCache.DelLocal(ctx, "local:key")
	if err != nil {
		fmt.Printf("DelLocal not supported: %v\n", err)
	}

	fmt.Println("\nRemote Cache Client Example Completed")
	fmt.Println("Note: Some operations above may have failed because no actual remote cache server was running.")
	fmt.Println("In a real application, you would need to have the cache servers running at the specified addresses.")
}
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
)

func main() {
	// 创建上下文对象
	ctx := context.Background()

	// 创建一个基于节点的分片缓存
	// 默认配置使用100个虚拟节点复制和CRC32哈希函数
	shardedCache := cache.NewNodeShardedCache()
	fmt.Println("Created node sharded cache with default configuration")

	// 创建3个内存缓存作为后端存储节点
	node1 := cache.NewMemoryCache()
	node2 := cache.NewMemoryCache()
	node3 := cache.NewMemoryCache()

	// 添加节点到分片缓存
	// 参数: nodeID, cache实例, 权重(用于决定分配到该节点的键的比例)
	err := shardedCache.AddNode("node1", node1, 1)
	if err != nil {
		log.Fatalf("Failed to add node1: %v", err)
	}

	err = shardedCache.AddNode("node2", node2, 1)
	if err != nil {
		log.Fatalf("Failed to add node2: %v", err)
	}

	err = shardedCache.AddNode("node3", node3, 1)
	if err != nil {
		log.Fatalf("Failed to add node3: %v", err)
	}

	fmt.Printf("Added 3 nodes to the sharded cache\n")
	fmt.Printf("Current node count: %d\n", shardedCache.GetNodeCount())

	// 设置一些数据，分片缓存会自动将数据分发到正确的节点
	for i := 1; i <= 10; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		err := shardedCache.Set(ctx, key, value, 10*time.Minute)
		if err != nil {
			log.Printf("Failed to set %s: %v", key, err)
		}
	}
	fmt.Println("Added 10 items to the sharded cache")

	// 获取数据，分片缓存会自动从正确的节点获取数据
	for i := 1; i <= 10; i++ {
		key := fmt.Sprintf("key%d", i)
		value, err := shardedCache.Get(ctx, key)
		if err != nil {
			fmt.Printf("Failed to get %s: %v\n", key, err)
			continue
		}
		fmt.Printf("Retrieved: %s = %v\n", key, value)
	}

	// 演示分布式删除
	fmt.Println("\nDemonstrating distributed deletion...")
	err = shardedCache.Del(ctx, "key5")
	if err != nil {
		log.Fatalf("Failed to delete key5: %v", err)
	}

	// 验证删除
	_, err = shardedCache.Get(ctx, "key5")
	if err != nil {
		fmt.Printf("Confirmed deletion of key5: %v\n", err)
	}

	// 演示添加和移除节点时的一致性哈希特性
	fmt.Println("\nDemonstrating consistent hashing with node changes...")

	// 创建1000个随机键，用于测试分布
	keyCount := 1000
	testKeys := make([]string, keyCount)
	for i := 0; i < keyCount; i++ {
		testKeys[i] = fmt.Sprintf("test-key-%d", i)
	}

	// 在当前3节点配置下，记录键到节点的映射
	initialMapping := make(map[string]string)
	for _, key := range testKeys {
		// 为了演示，我们只是找到键应该存储在哪个节点，但不实际存储
		nodes := shardedCache.GetNodeIDs()
		nodeIndex := int(fnvHash(key)) % len(nodes)
		initialMapping[key] = nodes[nodeIndex]
	}

	// 添加第四个节点
	node4 := cache.NewMemoryCache()
	err = shardedCache.AddNode("node4", node4, 1)
	if err != nil {
		log.Fatalf("Failed to add node4: %v", err)
	}
	fmt.Printf("Added node4 - New node count: %d\n", shardedCache.GetNodeCount())

	// 检查有多少键被重新映射
	remappedCount := 0
	for _, key := range testKeys {
		nodes := shardedCache.GetNodeIDs()
		nodeIndex := int(fnvHash(key)) % len(nodes)
		newNode := nodes[nodeIndex]
		if initialMapping[key] != newNode {
			remappedCount++
		}
	}

	// 理论上，添加第4个节点应该导致大约1/4的键被重新映射
	fmt.Printf("Keys remapped after adding node4: %d/%d (%.1f%%)\n",
		remappedCount, keyCount, float64(remappedCount)*100/float64(keyCount))

	// 现在移除一个节点，再次检查映射变化
	oldMapping := make(map[string]string)
	for _, key := range testKeys {
		nodes := shardedCache.GetNodeIDs()
		nodeIndex := int(fnvHash(key)) % len(nodes)
		oldMapping[key] = nodes[nodeIndex]
	}

	err = shardedCache.RemoveNode("node1")
	if err != nil {
		log.Fatalf("Failed to remove node1: %v", err)
	}
	fmt.Printf("Removed node1 - New node count: %d\n", shardedCache.GetNodeCount())

	// 检查有多少键被重新映射
	remappedCount = 0
	for _, key := range testKeys {
		nodes := shardedCache.GetNodeIDs()
		nodeIndex := int(fnvHash(key)) % len(nodes)
		newNode := nodes[nodeIndex]
		if oldMapping[key] != newNode {
			remappedCount++
		}
	}

	// 理论上，移除一个节点(从4个中)应该导致大约1/3的键被重新映射
	fmt.Printf("Keys remapped after removing node1: %d/%d (%.1f%%)\n",
		remappedCount, keyCount, float64(remappedCount)*100/float64(keyCount))

	// 并发操作示例
	fmt.Println("\nDemonstrating concurrent operations...")
	var wg sync.WaitGroup
	operationCount := 100
	wg.Add(operationCount)

	for i := 0; i < operationCount; i++ {
		go func(index int) {
			defer wg.Done()
			key := fmt.Sprintf("concurrent-key-%d", index)
			value := fmt.Sprintf("concurrent-value-%d", index)

			// 并发设置值
			err := shardedCache.Set(ctx, key, value, 5*time.Minute)
			if err != nil {
				log.Printf("Failed to set %s: %v", key, err)
				return
			}

			// 并发获取值
			val, err := shardedCache.Get(ctx, key)
			if err != nil {
				log.Printf("Failed to get %s: %v", key, err)
				return
			}

			if val != value {
				log.Printf("Value mismatch for %s: expected %s, got %v", key, value, val)
			}
		}(i)
	}

	wg.Wait()
	fmt.Printf("Successfully performed %d concurrent operations\n", operationCount)

	// 以不同权重添加节点的示例
	fmt.Println("\nDemonstrating weighted nodes...")
	weightedCache := cache.NewNodeShardedCache()

	// 添加权重不同的节点
	weightedCache.AddNode("light-node", cache.NewMemoryCache(), 1)    // 权重 1
	weightedCache.AddNode("medium-node", cache.NewMemoryCache(), 2)   // 权重 2
	weightedCache.AddNode("heavy-node", cache.NewMemoryCache(), 3)    // 权重 3

	// 使用随机键测试分布
	nodeCounts := make(map[string]int)
	testCount := 10000

	for i := 0; i < testCount; i++ {
		key := strconv.Itoa(rand.Intn(1000000))
		// 模拟Get操作但不实际执行，只检查键会被路由到哪个节点
		nodes := weightedCache.GetNodeIDs()
		nodeIndex := int(fnvHash(key)) % len(nodes)
		selectedNode := nodes[nodeIndex]
		nodeCounts[selectedNode]++
	}

	// 显示分布结果
	fmt.Println("Key distribution across weighted nodes:")
	total := float64(testCount)
	for node, count := range nodeCounts {
		fmt.Printf("  %s: %d keys (%.1f%%)\n", node, count, float64(count)*100/total)
	}
	fmt.Println("Note: With weighted nodes, 'heavy-node' should receive more keys than 'light-node'")
}

// 简化的FNV哈希函数用于演示目的
// 实际的NodeSharedCache使用更复杂的一致性哈希算法
func fnvHash(key string) uint32 {
	hash := uint32(2166136261)
	for i := 0; i < len(key); i++ {
		hash *= 16777619
		hash ^= uint32(key[i])
	}
	return hash
}
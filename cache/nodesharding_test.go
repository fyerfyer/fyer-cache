package cache

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/internal/ferr"
)

func TestNodeShardedCache_BasicOperations(t *testing.T) {
	ctx := context.Background()

	// 创建一个简单的分片缓存，包含3个节点
	shardedCache := NewNodeShardedCache()

	// 添加3个缓存节点
	node1 := NewMemoryCache()
	node2 := NewMemoryCache()
	node3 := NewMemoryCache()

	err := shardedCache.AddNode("node1", node1, 1)
	if err != nil {
		t.Fatalf("Failed to add node1: %v", err)
	}

	err = shardedCache.AddNode("node2", node2, 1)
	if err != nil {
		t.Fatalf("Failed to add node2: %v", err)
	}

	err = shardedCache.AddNode("node3", node3, 1)
	if err != nil {
		t.Fatalf("Failed to add node3: %v", err)
	}

	// 测试基本的Set和Get操作
	err = shardedCache.Set(ctx, "key1", "value1", time.Minute)
	if err != nil {
		t.Errorf("Failed to set value: %v", err)
	}

	val, err := shardedCache.Get(ctx, "key1")
	if err != nil {
		t.Errorf("Failed to get value: %v", err)
	}

	if val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}

	// 测试删除操作
	err = shardedCache.Del(ctx, "key1")
	if err != nil {
		t.Errorf("Failed to delete key: %v", err)
	}

	_, err = shardedCache.Get(ctx, "key1")
	if err != ferr.ErrKeyNotFound {
		t.Errorf("Expected key not found after deletion, got: %v", err)
	}
}

func TestNodeShardedCache_NodeManagement(t *testing.T) {
	//ctx := context.Background()
	shardedCache := NewNodeShardedCache()

	// 测试添加节点
	node1 := NewMemoryCache()
	err := shardedCache.AddNode("node1", node1, 1)
	if err != nil {
		t.Errorf("Failed to add node: %v", err)
	}

	// 测试重复添加相同ID的节点
	err = shardedCache.AddNode("node1", NewMemoryCache(), 1)
	if err == nil {
		t.Error("Expected error when adding duplicate node ID, but got nil")
	}

	// 测试获取节点列表
	nodes := shardedCache.GetNodeIDs()
	if len(nodes) != 1 || nodes[0] != "node1" {
		t.Errorf("Expected [node1], got %v", nodes)
	}

	// 测试删除节点
	err = shardedCache.RemoveNode("node1")
	if err != nil {
		t.Errorf("Failed to remove node: %v", err)
	}

	// 删除后节点列表应为空
	nodes = shardedCache.GetNodeIDs()
	if len(nodes) != 0 {
		t.Errorf("Expected empty node list after removal, got %v", nodes)
	}

	// 测试删除不存在的节点
	err = shardedCache.RemoveNode("non-existent")
	if err == nil {
		t.Error("Expected error when removing non-existent node, but got nil")
	}
}

func TestNodeShardedCache_Distribution(t *testing.T) {
	ctx := context.Background()

	// 创建具有3个节点的分片缓存
	shardedCache := NewNodeShardedCache(WithHashReplicas(100))

	node1 := NewMemoryCache()
	node2 := NewMemoryCache()
	node3 := NewMemoryCache()

	shardedCache.AddNode("node1", node1, 1)
	shardedCache.AddNode("node2", node2, 1)
	shardedCache.AddNode("node3", node3, 1)

	// 生成大量键值对并计算分布
	const keyCount = 10000

	for i := 0; i < keyCount; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		err := shardedCache.Set(ctx, key, i, time.Minute)
		if err != nil {
			t.Errorf("Failed to set key %s: %v", key, err)
		}
	}

	// 检查每个节点的项目数量
	checkDistribution := func() {
		// 计数器，记录每个节点在哈希环上的分布数量
		distribution := make(map[string]int)

		ringImpl, ok := shardedCache.ring.(*ConsistentHash)
		if !ok {
			t.Fatal("Failed to get ConsistentHash implementation")
		}

		for i := 0; i < keyCount; i++ {
			key := fmt.Sprintf("test-key-%d", i)
			node := ringImpl.Get(key)
			distribution[node]++
		}

		t.Logf("Key distribution: %v", distribution)

		// 计算标准差，以评估分布均匀性
		mean := keyCount / 3
		var variance float64
		for _, count := range distribution {
			diff := float64(count - mean)
			variance += diff * diff
		}
		variance /= 3
		stdDev := float64(0)
		if variance > 0 {
			stdDev = variance
		}

		t.Logf("Standard Deviation: %.2f (%.2f%% of mean)",
			stdDev, 100*stdDev/float64(mean))

		// 验证每个节点都有键
		for _, node := range []string{"node1", "node2", "node3"} {
			if distribution[node] == 0 {
				t.Errorf("Node %s has no keys assigned", node)
			}
		}
	}

	checkDistribution()
}

func TestNodeShardedCache_NodeAddRemove(t *testing.T) {
	ctx := context.Background()

	// 创建分片缓存，初始有2个节点
	shardedCache := NewNodeShardedCache(WithHashReplicas(100))

	node1 := NewMemoryCache()
	node2 := NewMemoryCache()

	shardedCache.AddNode("node1", node1, 1)
	shardedCache.AddNode("node2", node2, 1)

	// 加入大量数据
	const keyCount = 1000
	for i := 0; i < keyCount; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		err := shardedCache.Set(ctx, key, i, time.Minute)
		if err != nil {
			t.Errorf("Failed to set key %s: %v", key, err)
		}
	}

	// 记录键到节点的初始映射
	ringImpl, ok := shardedCache.ring.(*ConsistentHash)
	if !ok {
		t.Fatal("Failed to get ConsistentHash implementation")
	}

	initialMapping := make(map[string]string)
	for i := 0; i < keyCount; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		node := ringImpl.Get(key)
		initialMapping[key] = node
	}

	// 添加第三个节点
	node3 := NewMemoryCache()
	err := shardedCache.AddNode("node3", node3, 1)
	if err != nil {
		t.Fatalf("Failed to add node3: %v", err)
	}

	// 计算有多少键被重新映射
	remapped := 0
	for i := 0; i < keyCount; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		newNode := ringImpl.Get(key)
		if initialMapping[key] != newNode {
			remapped++
		}
	}

	remappedPercent := float64(remapped) * 100 / keyCount
	t.Logf("After adding a node, %.2f%% of keys were remapped", remappedPercent)

	// 理论上应该大约有1/3的键被重新映射
	if remappedPercent < 20 || remappedPercent > 50 {
		t.Logf("Warning: Unexpected remapping percentage: %.2f%% (expected around 33%%)",
			remappedPercent)
	}

	// 现在移除一个节点
	currentMapping := make(map[string]string)
	for i := 0; i < keyCount; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		node := ringImpl.Get(key)
		currentMapping[key] = node
	}

	err = shardedCache.RemoveNode("node2")
	if err != nil {
		t.Fatalf("Failed to remove node2: %v", err)
	}

	// 计算移除节点后重新映射的键数
	remapped = 0
	for i := 0; i < keyCount; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		newNode := ringImpl.Get(key)
		if currentMapping[key] != newNode {
			remapped++
			// 如果之前映射到被移除的节点，现在应该映射到其他节点
			if currentMapping[key] == "node2" && (newNode == "node1" || newNode == "node3") {
				// 正确的重映射
			} else if currentMapping[key] != "node2" && newNode != currentMapping[key] {
				t.Errorf("Key %s incorrectly remapped from %s to %s",
					key, currentMapping[key], newNode)
			}
		}
	}

	remappedPercent = float64(remapped) * 100 / keyCount
	t.Logf("After removing a node, %.2f%% of keys were remapped", remappedPercent)

	// 验证被删除节点的键已正确重映射
	nodeKeys := make(map[string]int)
	for i := 0; i < keyCount; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		node := ringImpl.Get(key)
		nodeKeys[node]++

		if node == "node2" {
			t.Errorf("Key %s still mapped to removed node2", key)
		}
	}

	t.Logf("Final key distribution: %v", nodeKeys)
}

func TestNodeShardedCache_ReplicaFactor(t *testing.T) {
	ctx := context.Background()

	// 创建一个具有3个备份的分片缓存
	shardedCache := NewNodeShardedCache(
		WithReplicaFactor(3),
		WithHashReplicas(100),
	)

	// 添加4个节点以支持3个备份
	for i := 1; i <= 4; i++ {
		nodeName := fmt.Sprintf("node%d", i)
		err := shardedCache.AddNode(nodeName, NewMemoryCache(), 1)
		if err != nil {
			t.Fatalf("Failed to add %s: %v", nodeName, err)
		}
	}

	// 设置一个测试值
	err := shardedCache.Set(ctx, "replicated-key", "replicated-value", time.Minute)
	if err != nil {
		t.Fatalf("Failed to set replicated key: %v", err)
	}

	// 确认可以正常获取
	val, err := shardedCache.Get(ctx, "replicated-key")
	if err != nil {
		t.Errorf("Failed to get replicated key: %v", err)
	}
	if val != "replicated-value" {
		t.Errorf("Expected 'replicated-value', got %v", val)
	}

	// 获取主节点
	ringImpl, ok := shardedCache.ring.(*ConsistentHash)
	if !ok {
		t.Fatal("Failed to get ConsistentHash implementation")
	}

	primaryNode := ringImpl.Get("replicated-key")

	// 从分片缓存中删除主节点
	err = shardedCache.RemoveNode(primaryNode)
	if err != nil {
		t.Fatalf("Failed to remove primary node: %v", err)
	}

	// 测试备份机制 - 主节点删除后，应该仍能从备份节点获取值
	val, err = shardedCache.Get(ctx, "replicated-key")
	if err != nil {
		t.Errorf("Failed to get key from backup node: %v", err)
	}
	if val != "replicated-value" {
		t.Errorf("Expected 'replicated-value' from backup, got %v", val)
	}

	t.Logf("Successfully retrieved value from backup after primary node %s was removed", primaryNode)
}

func TestNodeShardedCache_CustomHashFunction(t *testing.T) {
	ctx := context.Background()

	// 使用FNV哈希函数创建分片缓存
	fnvHash := func(data []byte) uint32 {
		h := fnv.New32a()
		h.Write(data)
		return h.Sum32()
	}

	shardedCache := NewNodeShardedCache(WithCustomHashFunc(fnvHash))

	// 添加3个节点
	for i := 1; i <= 3; i++ {
		nodeName := fmt.Sprintf("node%d", i)
		err := shardedCache.AddNode(nodeName, NewMemoryCache(), 1)
		if err != nil {
			t.Fatalf("Failed to add %s: %v", nodeName, err)
		}
	}

	// 测试基本功能
	err := shardedCache.Set(ctx, "test-key", "test-value", time.Minute)
	if err != nil {
		t.Fatalf("Failed to set key: %v", err)
	}

	val, err := shardedCache.Get(ctx, "test-key")
	if err != nil {
		t.Errorf("Failed to get key: %v", err)
	}
	if val != "test-value" {
		t.Errorf("Expected 'test-value', got %v", val)
	}
}

func TestNodeShardedCache_WeightedNodes(t *testing.T) {
	ctx := context.Background()

	shardedCache := NewNodeShardedCache(WithHashReplicas(100))

	// 添加具有不同权重的3个节点
	node1 := NewMemoryCache()
	node2 := NewMemoryCache()
	node3 := NewMemoryCache()

	shardedCache.AddNode("node1", node1, 1) // 权重1
	shardedCache.AddNode("node2", node2, 2) // 权重2
	shardedCache.AddNode("node3", node3, 3) // 权重3

	// 生成大量键值对
	const keyCount = 10000

	for i := 0; i < keyCount; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		err := shardedCache.Set(ctx, key, i, time.Minute)
		if err != nil {
			t.Errorf("Failed to set key %s: %v", key, err)
		}
	}

	// 检查键分布
	ringImpl, ok := shardedCache.ring.(*ConsistentHash)
	if !ok {
		t.Fatal("Failed to get ConsistentHash implementation")
	}

	distribution := make(map[string]int)
	for i := 0; i < keyCount; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		node := ringImpl.Get(key)
		distribution[node]++
	}

	t.Logf("Weighted key distribution: %v", distribution)

	// 验证权重较高的节点获得更多的键
	if distribution["node1"] >= distribution["node2"] || distribution["node2"] >= distribution["node3"] {
		t.Errorf("Expected node3 > node2 > node1 in key distribution, got: node1=%d, node2=%d, node3=%d",
			distribution["node1"], distribution["node2"], distribution["node3"])
	}
}

func TestNodeShardedCache_Concurrent(t *testing.T) {
	ctx := context.Background()

	shardedCache := NewNodeShardedCache(
		WithHashReplicas(100),
		WithReplicaFactor(2),
	)

	// 添加3个节点
	for i := 1; i <= 3; i++ {
		nodeName := fmt.Sprintf("node%d", i)
		err := shardedCache.AddNode(nodeName, NewMemoryCache(), 1)
		if err != nil {
			t.Fatalf("Failed to add %s: %v", nodeName, err)
		}
	}

	// 并发参数
	const (
		goroutines       = 10
		keysPerGoroutine = 1000
	)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// 启动多个goroutine进行并发操作
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()

			// 为每个goroutine创建独立的随机数生成器
			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)))

			for i := 0; i < keysPerGoroutine; i++ {
				key := fmt.Sprintf("key-%d-%d", id, i)
				value := fmt.Sprintf("value-%d-%d", id, i)

				// 75%的写操作，25%的读操作
				if r.Float32() < 0.75 {
					err := shardedCache.Set(ctx, key, value, time.Minute)
					if err != nil {
						t.Errorf("Concurrent Set failed: %v", err)
					}
				} else {
					_, _ = shardedCache.Get(ctx, key)
					// 忽略错误，因为键可能尚未设置
				}

				// 10%的概率删除键
				if r.Float32() < 0.1 {
					_ = shardedCache.Del(ctx, key)
				}
			}
		}(g)
	}

	wg.Wait()

	// 验证节点数量不变
	if shardedCache.GetNodeCount() != 3 {
		t.Errorf("Expected 3 nodes after concurrent operations, got %d",
			shardedCache.GetNodeCount())
	}
}

func TestNodeShardedCache_EmptyNodes(t *testing.T) {
	ctx := context.Background()

	// 创建空的分片缓存（没有节点）
	shardedCache := NewNodeShardedCache()

	// 测试在没有节点的情况下操作
	err := shardedCache.Set(ctx, "key", "value", time.Minute)
	if err == nil {
		t.Error("Expected error when setting to empty cache")
	}

	_, err = shardedCache.Get(ctx, "key")
	if err == nil {
		t.Error("Expected error when getting from empty cache")
	}

	// 删除应该不会报错
	err = shardedCache.Del(ctx, "key")
	if err != nil {
		t.Errorf("Del on empty cache should not error, got: %v", err)
	}
}

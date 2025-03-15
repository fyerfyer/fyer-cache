package integration

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/fyerfyer/fyer-cache/cache/cluster"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShardedClusterCache_Integration 分片集群缓存的集成测试
func TestShardedClusterCache_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 创建3个节点的集群进行测试
	ctx := context.Background()
	numNodes := 3
	clusterNodes := make([]*ShardedClusterCache, numNodes)

	// 获取可用的地址
	addresses := make([]string, numNodes)
	for i := 0; i < numNodes; i++ {
		addresses[i] = getAvailableAddr(t)
	}

	// 创建和启动所有节点
	for i := 0; i < numNodes; i++ {
		nodeID := fmt.Sprintf("node%d", i+1)
		cache := cache.NewMemoryCache()

		// 创建节点选项：短间隔以加速测试
		clusterOptions := []cluster.NodeOption{
			cluster.WithGossipInterval(500 * time.Millisecond),
			cluster.WithProbeInterval(500 * time.Millisecond),
			cluster.WithSyncInterval(1 * time.Second),
		}

		// 创建分片缓存选项
		cacheOptions := []ShardedClusterCacheOption{
			WithVirtualNodeCount(100),
			WithReplicaFactor(2), // 使用2个副本以实现冗余
		}

		// 创建分片集群缓存
		scc, err := NewShardedClusterCache(cache, nodeID, addresses[i], clusterOptions, cacheOptions)
		require.NoError(t, err, "Failed to create ShardedClusterCache for node %s", nodeID)

		// 启动节点
		err = scc.Start()
		require.NoError(t, err, "Failed to start ShardedClusterCache for node %s", nodeID)

		clusterNodes[i] = scc
		t.Logf("Started node: %s at %s", nodeID, addresses[i])

		// 添加延迟以避免启动过于集中
		time.Sleep(100 * time.Millisecond)
	}

	// 确保在测试结束时关闭所有节点
	defer func() {
		for i, node := range clusterNodes {
			if node != nil {
				if err := node.Stop(); err != nil {
					t.Logf("Error stopping node %d: %v", i+1, err)
				}
			}
		}
	}()

	t.Log("All nodes started successfully")

	// 形成集群：将所有节点连接到第一个节点
	err := clusterNodes[0].Join(addresses[0]) // 第一个节点加入自己，创建集群
	require.NoError(t, err, "Failed to bootstrap first node")

	// 其他节点加入第一个节点
	for i := 1; i < numNodes; i++ {
		err = clusterNodes[i].Join(addresses[0])
		require.NoError(t, err, "Node %d failed to join cluster", i+1)
		// 添加短暂延迟，让加入事件传播
		time.Sleep(300 * time.Millisecond)
	}

	// 等待集群稳定
	t.Log("Waiting for cluster to stabilize...")
	time.Sleep(3 * time.Second)

	// 验证所有节点都能看到彼此
	for i, node := range clusterNodes {
		members := node.Members()
		t.Logf("Node %d sees %d members", i+1, len(members))
		assert.Equal(t, numNodes, len(members), "Node %d should see all %d members", i+1, numNodes)
	}

	// 测试基本的缓存操作
	t.Log("Testing basic cache operations across the cluster...")

	// 在第一个节点上设置一个值
	key1 := "test-key-1"
	value1 := "test-value-1"
	err = clusterNodes[0].Set(ctx, key1, value1, 5*time.Minute)
	require.NoError(t, err, "Failed to set value on node 1")

	// 在第二个节点上设置一个值
	key2 := "test-key-2"
	value2 := "test-value-2"
	err = clusterNodes[1].Set(ctx, key2, value2, 5*time.Minute)
	require.NoError(t, err, "Failed to set value on node 2")

	// 等待值同步和传播
	t.Log("Waiting for values to propagate...")
	time.Sleep(2 * time.Second)

	// 验证所有节点可以检索这些值
	for i, node := range clusterNodes {
		// 读取第一个键
		val1, err := node.Get(ctx, key1)
		assert.NoError(t, err, "Node %d failed to get key1", i+1)
		if err == nil {
			assert.Equal(t, value1, val1, "Node %d got incorrect value for key1", i+1)
		}

		// 读取第二个键
		val2, err := node.Get(ctx, key2)
		assert.NoError(t, err, "Node %d failed to get key2", i+1)
		if err == nil {
			assert.Equal(t, value2, val2, "Node %d got incorrect value for key2", i+1)
		}
	}

	// 测试删除操作
	t.Log("Testing delete operation...")
	err = clusterNodes[2].Del(ctx, key1)
	require.NoError(t, err, "Failed to delete key1 from node 3")

	// 等待删除操作传播
	time.Sleep(2 * time.Second)

	// 验证所有节点都已删除该键
	for i, node := range clusterNodes {
		_, err := node.Get(ctx, key1)
		assert.Error(t, err, "Node %d should report key1 as deleted", i+1)
	}

	// 测试节点故障和恢复
	t.Log("Testing node failure and recovery...")

	// 停止第二个节点
	t.Log("Stopping node 2...")
	err = clusterNodes[1].Stop()
	require.NoError(t, err, "Failed to stop node 2")

	// 等待集群检测到节点故障
	t.Log("Waiting for failure detection...")
	time.Sleep(3 * time.Second)

	// 验证其他节点已检测到故障
	members := clusterNodes[0].Members()
	nodeFound := false
	nodeStatus := cache.NodeStatusUp

	for _, member := range members {
		if member.ID == "node2" {
			nodeFound = true
			nodeStatus = member.Status
			break
		}
	}

	if nodeFound {
		// 节点可能被标记为可疑、故障或已离开
		t.Logf("Node 2 status as seen by node 1: %v", nodeStatus)
		assert.NotEqual(t, cache.NodeStatusUp, nodeStatus, "Node 2 should not be reported as UP")
	} else {
		// 或者节点可能已经从成员列表中完全移除
		t.Log("Node 2 has been completely removed from the member list")
	}

	// 在节点恢复之前，验证剩余节点仍然可以正常工作
	err = clusterNodes[0].Set(ctx, "recovery-test", "value", time.Minute)
	assert.NoError(t, err, "Failed to set value after node failure")

	val, err := clusterNodes[2].Get(ctx, "recovery-test")
	assert.NoError(t, err, "Failed to get value after node failure")
	assert.Equal(t, "value", val)

	// 重新启动第二个节点
	t.Log("Restarting node 2...")
	clusterNodes[1], err = NewShardedClusterCache(
		cache.NewMemoryCache(),
		"node2",
		addresses[1],
		[]cluster.NodeOption{
			cluster.WithGossipInterval(500 * time.Millisecond),
			cluster.WithProbeInterval(500 * time.Millisecond),
		},
		[]ShardedClusterCacheOption{
			WithVirtualNodeCount(100),
			WithReplicaFactor(2),
		},
	)
	require.NoError(t, err, "Failed to recreate node 2")

	err = clusterNodes[1].Start()
	require.NoError(t, err, "Failed to start recreated node 2")

	// 重新加入集群
	err = clusterNodes[1].Join(addresses[0])
	require.NoError(t, err, "Failed to rejoin node 2 to the cluster")

	// 等待节点重新加入集群并同步数据
	t.Log("Waiting for node 2 to rejoin and synchronize...")
	time.Sleep(5 * time.Second)

	// 验证重新加入的节点可以访问数据
	val, err = clusterNodes[1].Get(ctx, "recovery-test")
	assert.NoError(t, err, "Rejoined node should be able to get data")
	assert.Equal(t, "value", val)

	// 验证所有节点再次看到完整的集群
	for i, node := range clusterNodes {
		members := node.Members()
		t.Logf("Node %d sees %d members after recovery", i+1, len(members))
		assert.Equal(t, numNodes, len(members), "All nodes should see complete cluster after recovery")
	}

	t.Log("Integration test completed successfully")
}

// TestShardedClusterCache_Coordinator 测试协调者选举
func TestShardedClusterCache_Coordinator(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping coordinator test in short mode")
	}

	// 创建三个节点，ID设置为按字母顺序排列
	// 假设字典序最小的节点为协调者
	ids := []string{"a-node", "b-node", "c-node"}
	addrs := make([]string, 3)
	nodes := make([]*ShardedClusterCache, 3)

	// 获取可用地址
	for i := 0; i < 3; i++ {
		addrs[i] = getAvailableAddr(t)
	}

	// 创建和启动节点
	for i := 0; i < 3; i++ {
		cache := cache.NewMemoryCache()

		scc, err := NewShardedClusterCache(cache, ids[i], addrs[i], nil, nil)
		require.NoError(t, err, "Failed to create node %s", ids[i])

		err = scc.Start()
		require.NoError(t, err, "Failed to start node %s", ids[i])

		nodes[i] = scc

		// 添加延迟以避免启动过于集中
		time.Sleep(100 * time.Millisecond)
	}

	// 确保在测试结束时关闭所有节点
	defer func() {
		for _, node := range nodes {
			if node != nil {
				node.Stop()
			}
		}
	}()

	// 形成集群
	err := nodes[0].Join(addrs[0])
	require.NoError(t, err, "Failed to bootstrap first node")

	for i := 1; i < 3; i++ {
		err = nodes[i].Join(addrs[0])
		require.NoError(t, err, "Node %s failed to join cluster", ids[i])
		time.Sleep(300 * time.Millisecond) // 延迟以让节点加入传播
	}

	// 等待集群稳定
	time.Sleep(2 * time.Second)

	// 验证所有节点都认为a-node是协调者
	for i, node := range nodes {
		isCoord := node.IsCoordinator()
		if i == 0 {
			// a-node 应该是协调者
			assert.True(t, isCoord, "Node %s should be the coordinator", ids[i])
		} else {
			assert.False(t, isCoord, "Node %s should not be the coordinator", ids[i])
		}
	}

	// 停止当前协调者
	t.Log("Stopping the coordinator node...")
	err = nodes[0].Stop()
	require.NoError(t, err, "Failed to stop coordinator node")

	// 等待新协调者被选出
	t.Log("Waiting for new coordinator election...")
	time.Sleep(3 * time.Second)

	// b-node 应该成为新的协调者
	assert.True(t, nodes[1].IsCoordinator(), "Node %s should become the new coordinator", ids[1])
	assert.False(t, nodes[2].IsCoordinator(), "Node %s should not be the coordinator", ids[2])

	t.Log("Coordinator election test completed successfully")
}

// 负载测试 - 测试在大量读写操作下的表现
func TestShardedClusterCache_LoadTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	// 创建一个包含2个节点的集群
	ctx := context.Background()
	numNodes := 2
	clusterNodes := make([]*ShardedClusterCache, numNodes)
	addresses := make([]string, numNodes)

	// 获取可用地址
	for i := 0; i < numNodes; i++ {
		addresses[i] = getAvailableAddr(t)
	}

	// 创建和启动所有节点
	for i := 0; i < numNodes; i++ {
		nodeID := fmt.Sprintf("node%d", i+1)
		localCache := cache.NewMemoryCache()

		// 创建节点
		scc, err := NewShardedClusterCache(
			localCache,
			nodeID,
			addresses[i],
			[]cluster.NodeOption{
				cluster.WithGossipInterval(500 * time.Millisecond),
			},
			[]ShardedClusterCacheOption{
				WithVirtualNodeCount(100),
			},
		)
		require.NoError(t, err)

		err = scc.Start()
		require.NoError(t, err)

		clusterNodes[i] = scc
	}

	// 确保清理
	defer func() {
		for _, node := range clusterNodes {
			if node != nil {
				node.Stop()
			}
		}
	}()

	// 形成集群
	err := clusterNodes[0].Join(addresses[0])
	require.NoError(t, err)

	err = clusterNodes[1].Join(addresses[0])
	require.NoError(t, err)

	// 等待集群稳定
	time.Sleep(2 * time.Second)

	// 执行负载测试 - 插入一些数据，然后进行读取
	const keyCount = 100

	t.Log("Inserting test data...")
	// 在两个节点间分配写入操作
	for i := 0; i < keyCount; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)

		// 选择节点进行写入 (轮询)
		nodeIdx := i % numNodes
		err := clusterNodes[nodeIdx].Set(ctx, key, value, 5*time.Minute)
		if err != nil {
			t.Errorf("Failed to set %s: %v", key, err)
		}

		// 每10个键添加一个小延迟，避免过快写入
		if i > 0 && i%10 == 0 {
			time.Sleep(50 * time.Millisecond)
		}
	}

	// 等待数据同步
	t.Log("Waiting for data to propagate...")
	time.Sleep(2 * time.Second)

	// 验证数据可以从任意节点读取
	t.Log("Verifying data can be read from any node...")
	missingKeys := 0

	for i := 0; i < keyCount; i++ {
		key := fmt.Sprintf("key-%d", i)
		expectedValue := fmt.Sprintf("value-%d", i)

		// 从随机节点读取
		nodeIdx := i % numNodes
		value, err := clusterNodes[nodeIdx].Get(ctx, key)

		if err != nil {
			t.Logf("Could not find key %s: %v", key, err)
			missingKeys++
		} else if value != expectedValue {
			t.Errorf("Incorrect value for key %s: expected %s, got %v", key, expectedValue, value)
		}
	}

	// 容许少量键未找到（由于分布式系统的最终一致性）
	allowedMissingKeys := keyCount * 5 / 100 // 允许5%的键未找到
	assert.LessOrEqual(t, missingKeys, allowedMissingKeys,
		"Too many missing keys (%d), threshold is %d", missingKeys, allowedMissingKeys)

	t.Logf("Load test completed: %d of %d keys accessible (%.1f%%)",
		keyCount-missingKeys, keyCount, float64(keyCount-missingKeys)*100/float64(keyCount))
}

// 获取可用的本地地址
func getAvailableAddr(t *testing.T) string {
	// 使用端口0让系统分配可用端口
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "Failed to find available port")

	// 获取实际分配的地址
	addr := listener.Addr().String()

	// 关闭监听器以释放端口
	listener.Close()

	return addr
}

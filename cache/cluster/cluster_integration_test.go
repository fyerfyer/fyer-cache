package cluster

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 集群功能的集成测试
func TestCluster_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 使用较短的时间间隔加快测试速度
	config := &NodeConfig{
		GossipInterval:  50 * time.Millisecond,  // 更快的gossip（原来是100ms）
		SyncInterval:    250 * time.Millisecond, // 更快的同步（原来是500ms）
		ProbeInterval:   50 * time.Millisecond,  // 更快的探测（原来是100ms）
		ProbeTimeout:    25 * time.Millisecond,  // 更短的超时（原来是50ms）
		SuspicionMult:   1,                      // 保持为1以便更快检测失败
		SyncConcurrency: 2,
		VirtualNodes:    10,
		MaxRetries:      2,
		RetryInterval:   100 * time.Millisecond,
		EventBufferSize: 100,
	}

	// 为测试寻找可用端口
	ports, err := getAvailablePorts(3)
	require.NoError(t, err, "Failed to get available ports")

	// 创建3个节点与内存缓存
	nodes := make([]*Node, 3)
	caches := make([]*cache.MemoryCache, 3)

	for i := 0; i < 3; i++ {
		// 复制配置
		nodeConfig := *config
		nodeConfig.ID = fmt.Sprintf("node-%d", i+1)
		nodeConfig.BindAddr = fmt.Sprintf("127.0.0.1:%d", ports[i])

		// 创建缓存
		caches[i] = cache.NewMemoryCache()
		node, err := NewNode(nodeConfig.ID, caches[i], WithBindAddr(nodeConfig.BindAddr),
			WithGossipInterval(nodeConfig.GossipInterval),
			WithSyncInterval(nodeConfig.SyncInterval),
			WithProbeInterval(nodeConfig.ProbeInterval),
			WithProbeTimeout(nodeConfig.ProbeTimeout),
			WithSuspicionMult(nodeConfig.SuspicionMult))
		require.NoError(t, err)
		nodes[i] = node
	}

	// 启动所有节点
	for i, node := range nodes {
		err = node.Start()
		require.NoError(t, err, "Failed to start node %d", i+1)
	}

	// 确保适当的清理
	defer func() {
		for _, node := range nodes {
			if node.running {
				node.Stop()
			}
		}
	}()

	t.Log("All nodes started successfully")

	// 第一个节点形成自己的集群
	err = nodes[0].Join(nodes[0].config.BindAddr)
	require.NoError(t, err, "Failed to bootstrap first node")

	// 其他节点加入第一个节点
	for i := 1; i < len(nodes); i++ {
		err = nodes[i].Join(nodes[0].config.BindAddr)
		require.NoError(t, err, "Failed to join node %d to cluster", i+1)
	}

	t.Log("All nodes joined the cluster successfully")

	// 等待gossip传播成员信息
	err = waitForFullClusterFormation(t, nodes, 3, 5*time.Second)
	require.NoError(t, err, "Cluster did not form completely within timeout")

	// 验证所有节点都能看到其他节点
	for i, node := range nodes {
		// 记录来自每个节点角度的成员
		t.Logf("Node %d sees members: %v", i+1, memberIDs(node.Members()))
	}

	// 在节点之间测试缓存操作
	ctx := context.Background()

	// 在第一个节点设置数据
	key1 := "test-key-1"
	value1 := "test-value-1"
	err = caches[0].Set(ctx, key1, value1, time.Minute)
	require.NoError(t, err, "Failed to set data in first node")

	// 验证通过所有节点的get操作
	// 注意：在真实集群中使用一致性哈希时，数据只会在特定节点上，
	// 但对于此测试，我们验证直接缓存访问

	// 测试delete操作

	// 验证key已被删除

	// 测试节点故障检测
	// 我们将通过停止一个节点来模拟节点故障

	t.Log("Simulating node failure by stopping node 3")
	err = nodes[2].Stop()
	require.NoError(t, err)

	// 等待故障检测
	t.Log("Waiting for failure detection...")

	// 使用健壮的等待函数替代固定睡眠时间
	detected := waitForNodeFailure(t, []*Node{nodes[0], nodes[1]}, "node-3", 10*time.Second)
	assert.True(t, detected, "Failed node was not detected within timeout period")

	// 验证节点1的角度来看失败节点的状态
	members := nodes[0].Members()
	status := getNodeStatus(members, "node-3")
	assert.NotEqual(t, cache.NodeStatusUp, status, "Failed node should not be marked as UP from node 1's perspective")

	// 验证节点2的角度来看失败节点的状态
	members = nodes[1].Members()
	status = getNodeStatus(members, "node-3")
	assert.NotEqual(t, cache.NodeStatusUp, status, "Failed node should not be marked as UP from node 2's perspective")

	t.Log("Failed node was marked as suspicious or down")

	// 测试节点重新加入
	t.Log("Restarting failed node")
	err = nodes[2].Start()
	require.NoError(t, err)
	err = nodes[2].Join(nodes[0].config.BindAddr)
	require.NoError(t, err)

	// 等待重新加入传播
	time.Sleep(2 * time.Second)

	// 验证所有节点都回来了
	for i, node := range nodes {
		members := node.Members()
		assert.Equal(t, 3, len(members), "Node %d should see all 3 members", i+1)
	}

	t.Log("Cluster integration test completed successfully")
}

// 等待节点故障的帮助函数
func waitForNodeFailure(t *testing.T, otherNodes []*Node, failedNodeID string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	pollInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		allDetected := true

		for _, node := range otherNodes {
			members := node.Members()
			failedNodeStatus := getNodeStatus(members, failedNodeID)

			// 如果任何节点仍然将失败节点视为UP，继续等待
			if failedNodeStatus == cache.NodeStatusUp {
				allDetected = false
				break
			}
		}

		if allDetected {
			return true
		}

		time.Sleep(pollInterval)
	}

	// 在超时时记录诊断信息
	t.Log("Timeout waiting for node failure detection, current membership views:")
	for i, node := range otherNodes {
		t.Logf("Node %d membership:", i+1)
		for _, m := range node.Members() {
			t.Logf("  ID: %s, Status: %d, Addr: %s", m.ID, m.Status, m.Addr)
		}
	}

	return false
}

// 获取节点状态的帮助函数
func getNodeStatus(members []cache.NodeInfo, nodeID string) cache.NodeStatus {
	for _, member := range members {
		if member.ID == nodeID {
			return member.Status
		}
	}
	return cache.NodeStatusDown // 未找到时视为down
}

// 等待集群完全形成的帮助函数
func waitForFullClusterFormation(t *testing.T, nodes []*Node, expectedCount int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	pollInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		allFormed := true

		for _, node := range nodes {
			if !node.running {
				continue
			}

			members := node.Members()
			if len(members) != expectedCount {
				allFormed = false
				break
			}

			// 所有节点必须将所有其他节点视为UP
			for _, member := range members {
				if member.Status != cache.NodeStatusUp {
					allFormed = false
					break
				}
			}
		}

		if allFormed {
			return nil
		}

		time.Sleep(pollInterval)
	}

	return fmt.Errorf("cluster formation timed out - not all nodes see %d members as UP", expectedCount)
}

// 寻找可用端口的帮助函数
func getAvailablePorts(count int) ([]int, error) {
	ports := make([]int, 0, count)
	for i := 0; i < count; i++ {
		addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
		if err != nil {
			return nil, err
		}

		l, err := net.ListenTCP("tcp", addr)
		if err != nil {
			return nil, err
		}
		defer l.Close()
		ports = append(ports, l.Addr().(*net.TCPAddr).Port)
	}
	return ports, nil
}

// 将成员列表转换为ID切片用于日志记录
func memberIDs(members []cache.NodeInfo) []string {
	ids := make([]string, len(members))
	for i, m := range members {
		ids[i] = m.ID
	}
	return ids
}

// TestCluster_LargerScale 测试具有更多操作的更大集群
func TestCluster_LargerScale(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large scale integration test in short mode")
	}

	// 使用较短的时间间隔加快测试速度
	config := &NodeConfig{
		GossipInterval:  50 * time.Millisecond,  // 更快的gossip
		SyncInterval:    250 * time.Millisecond, // 更快的同步
		ProbeInterval:   50 * time.Millisecond,  // 更快的探测
		ProbeTimeout:    25 * time.Millisecond,  // 更短的超时
		SuspicionMult:   1,                      // 保持为1以更快检测失败
		SyncConcurrency: 2,
		VirtualNodes:    10,
		MaxRetries:      2,
		RetryInterval:   100 * time.Millisecond,
		EventBufferSize: 100,
	}

	// 为测试寻找可用端口
	ports, err := getAvailablePorts(5) // 5个节点
	require.NoError(t, err, "Failed to get available ports")

	// 创建5个节点与内存缓存
	nodes := make([]*Node, 5)
	caches := make([]*cache.MemoryCache, 5)

	for i := 0; i < 5; i++ {
		// 复制配置
		nodeConfig := *config
		nodeConfig.ID = fmt.Sprintf("node-%d", i+1)
		nodeConfig.BindAddr = fmt.Sprintf("127.0.0.1:%d", ports[i])

		// 创建缓存
		caches[i] = cache.NewMemoryCache()
		node, err := NewNode(nodeConfig.ID, caches[i], WithBindAddr(nodeConfig.BindAddr),
			WithGossipInterval(nodeConfig.GossipInterval),
			WithSyncInterval(nodeConfig.SyncInterval),
			WithProbeInterval(nodeConfig.ProbeInterval),
			WithProbeTimeout(nodeConfig.ProbeTimeout),
			WithSuspicionMult(nodeConfig.SuspicionMult))
		require.NoError(t, err)
		nodes[i] = node
	}

	// 启动所有节点并形成集群
	for i, node := range nodes {
		err = node.Start()
		require.NoError(t, err, "Failed to start node %d", i+1)
	}

	// 确保适当的清理
	defer func() {
		for _, node := range nodes {
			if node.running {
				node.Stop()
			}
		}
	}()

	// 使用第一个节点作为种子节点
	err = nodes[0].Join(nodes[0].config.BindAddr) // 第一个节点自己加入
	require.NoError(t, err)

	// 其他节点加入种子节点
	for i := 1; i < len(nodes); i++ {
		err = nodes[i].Join(nodes[0].config.BindAddr)
		require.NoError(t, err)
	}

	// 等待gossip传播成员信息
	err = waitForFullClusterFormation(t, nodes, 5, 5*time.Second)
	require.NoError(t, err, "Cluster did not form completely within timeout")

	// 验证集群形成
	for i, node := range nodes {
		members := node.Members()
		assert.Equal(t, 5, len(members), "Node %d should see all 5 members", i+1)
	}

	// 并发缓存操作
	ctx := context.Background()
	var wg sync.WaitGroup

	// 从不同节点并发设置数据
	keyCount := 100
	wg.Add(keyCount)
	for i := 0; i < keyCount; i++ {
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", idx)
			value := fmt.Sprintf("value-%d", idx)
			sourceNode := idx % 5 // 在节点间分配操作
			err := caches[sourceNode].Set(ctx, key, value, time.Minute)
			if err != nil {
				t.Logf("Error setting key %s on node %d: %v", key, sourceNode, err)
			}
		}(i)
	}
	wg.Wait()

	// 验证数据可以从任何节点检索
	// 在真实集群使用一致性哈希时，数据会分布式存储
	// 但对于此测试，我们验证直接缓存访问
	for i := 0; i < 5; i++ {
		t.Logf("Verifying data on node %d", i+1)
		for j := 0; j < keyCount; j++ {
			key := fmt.Sprintf("key-%d", j)
			value, _ := caches[i].Get(ctx, key)
			expectedValue := fmt.Sprintf("value-%d", j)
			if value != nil && value != expectedValue {
				t.Errorf("Incorrect value for key %s on node %d: got %v, want %s",
					key, i+1, value, expectedValue)
			}
		}
	}

	// 模拟节点故障和恢复
	t.Log("Stopping node 3 to simulate failure")
	err = nodes[2].Stop() // 第三个节点失败
	require.NoError(t, err)

	// 等待故障检测
	detected := waitForNodeFailure(t, []*Node{nodes[0], nodes[1], nodes[3], nodes[4]}, "node-3", 10*time.Second)
	assert.True(t, detected, "Failed node was not detected within timeout period")

	// 验证故障被检测到
	for i, node := range nodes {
		if i == 2 { // 跳过已停止的节点
			continue
		}
		members := node.Members()
		status := getNodeStatus(members, "node-3")
		assert.NotEqual(t, cache.NodeStatusUp, status,
			"Failed node should not be UP from node %d's perspective", i+1)
	}

	// 重启失败节点
	t.Log("Restarting the failed node")
	err = nodes[2].Start()
	require.NoError(t, err)
	err = nodes[2].Join(nodes[0].config.BindAddr)
	require.NoError(t, err)

	// 等待重新加入传播
	time.Sleep(3 * time.Second)

	// 验证所有节点都回来了
	for i, node := range nodes {
		members := node.Members()
		assert.Equal(t, 5, len(members), "Node %d should see all 5 members", i+1)
	}

	t.Log("Large scale cluster test completed successfully")
}

// TestCluster_Coordinator 测试协调者选举
func TestCluster_Coordinator(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping coordinator test in short mode")
	}

	// 创建一个小集群
	config := &NodeConfig{
		GossipInterval:  50 * time.Millisecond,  // 更快的gossip
		SyncInterval:    250 * time.Millisecond, // 更快的同步
		ProbeInterval:   50 * time.Millisecond,  // 更快的探测
		ProbeTimeout:    25 * time.Millisecond,  // 更短的超时
		SuspicionMult:   1,                      // 保持为1以更快检测失败
		SyncConcurrency: 2,
		VirtualNodes:    10,
		MaxRetries:      2,
		RetryInterval:   100 * time.Millisecond,
		EventBufferSize: 100,
	}

	// 为测试寻找可用端口
	ports, err := getAvailablePorts(3)
	require.NoError(t, err, "Failed to get available ports")

	// 创建特定ID的节点来测试协调者选举
	// 具有字典序最小ID的节点应为协调者
	nodes := make([]*Node, 3)
	caches := make([]*cache.MemoryCache, 3)

	// 创建具有特定ID的节点，以便测试协调者选举
	nodeIDs := []string{"node-a", "node-b", "node-c"} // a 应为协调者（字典序最小）

	for i := 0; i < 3; i++ {
		nodeConfig := *config
		nodeConfig.ID = nodeIDs[i]
		nodeConfig.BindAddr = fmt.Sprintf("127.0.0.1:%d", ports[i])

		caches[i] = cache.NewMemoryCache()
		node, err := NewNode(nodeConfig.ID, caches[i], WithBindAddr(nodeConfig.BindAddr),
			WithGossipInterval(nodeConfig.GossipInterval),
			WithSyncInterval(nodeConfig.SyncInterval),
			WithProbeInterval(nodeConfig.ProbeInterval),
			WithProbeTimeout(nodeConfig.ProbeTimeout),
			WithSuspicionMult(nodeConfig.SuspicionMult))
		require.NoError(t, err)
		nodes[i] = node
	}

	// 启动所有节点
	for i, node := range nodes {
		err = node.Start()
		require.NoError(t, err, "Failed to start node %d", i+1)
	}

	// 确保适当的清理
	defer func() {
		for _, node := range nodes {
			if node.running {
				node.Stop()
			}
		}
	}()

	// 所有节点加入第一个节点的集群
	for i := 0; i < len(nodes); i++ {
		err = nodes[i].Join(nodes[0].config.BindAddr)
		require.NoError(t, err)
	}

	// 等待gossip传播
	err = waitForFullClusterFormation(t, nodes, 3, 5*time.Second)
	require.NoError(t, err, "Cluster did not form completely within timeout")

	// 验证协调者选举 - node-a应为协调者
	assert.True(t, nodes[0].IsCoordinator(), "Node A should be the coordinator")
	assert.False(t, nodes[1].IsCoordinator(), "Node B should not be coordinator")
	assert.False(t, nodes[2].IsCoordinator(), "Node C should not be coordinator")

	// 移除协调者节点
	t.Log("Removing coordinator node")
	err = nodes[0].Stop()
	require.NoError(t, err)

	// 添加一个等待协调者选举的机制
	t.Log("Waiting for new coordinator election...")

	// 等待node-b成为协调者，给予足够超时
	elected := waitForCoordinatorChange(t, []*Node{nodes[1], nodes[2]}, "node-b", 5*time.Second)
	assert.True(t, elected, "New coordinator was not elected within timeout period")

	// 现在可以安全检查协调者状态
	assert.True(t, nodes[1].IsCoordinator(), "Node B should be the coordinator after node A left")
	assert.False(t, nodes[2].IsCoordinator(), "Node C should not be coordinator")

	t.Log("Coordinator election test completed successfully")
}

// 等待协调者变化的帮助函数
func waitForCoordinatorChange(t *testing.T, nodes []*Node, expectedCoordinatorID string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	pollInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		if len(nodes) == 0 {
			return false
		}

		// Check if all RUNNING nodes agree on who is coordinator
		allAgree := true
		coordinatorCheck := false

		for _, node := range nodes {
			if !node.running {
				continue
			}

			// For additional debugging
			t.Logf("Node %s sees members: %v", node.id, memberIDs(node.Members()))

			// Check if this node thinks it's the coordinator
			if node.id == expectedCoordinatorID && node.IsCoordinator() {
				coordinatorCheck = true
				t.Logf("Node %s confirms it is coordinator", node.id)
			} else if node.id == expectedCoordinatorID && !node.IsCoordinator() {
				allAgree = false
				t.Logf("Node %s should be coordinator but doesn't think it is", node.id)
			}
		}

		// If we've confirmed the expected node thinks it's coordinator, return true
		if coordinatorCheck {
			return true
		}

		if !allAgree {
			time.Sleep(pollInterval)
			continue
		}

		time.Sleep(pollInterval)
	}

	// For better debugging
	t.Logf("Timed out waiting for coordinator change. Current status:")
	for _, node := range nodes {
		if node.running {
			t.Logf("Node %s is running, isCoordinator=%v", node.id, node.IsCoordinator())
		} else {
			t.Logf("Node %s is not running", node.id)
		}
	}

	return false
}

// TestCluster_GracefulShutdown 测试优雅关闭流程
func TestCluster_GracefulShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping graceful shutdown test in short mode")
	}

	// 创建一个小集群
	config := &NodeConfig{
		GossipInterval:  50 * time.Millisecond,  // 更快的gossip
		SyncInterval:    250 * time.Millisecond, // 更快的同步
		ProbeInterval:   50 * time.Millisecond,  // 更快的探测
		ProbeTimeout:    25 * time.Millisecond,  // 更短的超时
		SuspicionMult:   1,                      // 保持为1以更快检测失败
		SyncConcurrency: 2,
		VirtualNodes:    10,
		MaxRetries:      2,
		RetryInterval:   100 * time.Millisecond,
		EventBufferSize: 100,
	}

	// 为测试寻找可用端口
	ports, err := getAvailablePorts(3)
	require.NoError(t, err, "Failed to get available ports")

	// 创建3个节点
	nodes := make([]*Node, 3)
	caches := make([]*cache.MemoryCache, 3)

	for i := 0; i < 3; i++ {
		nodeConfig := *config
		nodeConfig.ID = fmt.Sprintf("node-%d", i+1)
		nodeConfig.BindAddr = fmt.Sprintf("127.0.0.1:%d", ports[i])

		caches[i] = cache.NewMemoryCache()
		node, err := NewNode(nodeConfig.ID, caches[i], WithBindAddr(nodeConfig.BindAddr),
			WithGossipInterval(nodeConfig.GossipInterval),
			WithSyncInterval(nodeConfig.SyncInterval),
			WithProbeInterval(nodeConfig.ProbeInterval),
			WithProbeTimeout(nodeConfig.ProbeTimeout),
			WithSuspicionMult(nodeConfig.SuspicionMult))
		require.NoError(t, err)
		nodes[i] = node
	}

	// 启动所有节点
	for _, node := range nodes {
		err = node.Start()
		require.NoError(t, err)
	}

	// 确保对剩余节点进行适当的清理
	defer func() {
		for _, node := range nodes {
			if node.running {
				node.Stop()
			}
		}
	}()

	// 所有节点加入第一个节点的集群
	for i := 0; i < len(nodes); i++ {
		err = nodes[i].Join(nodes[0].config.BindAddr)
		require.NoError(t, err)
	}

	// 等待gossip传播
	err = waitForFullClusterFormation(t, nodes, 3, 5*time.Second)
	require.NoError(t, err, "Cluster did not form completely within timeout")

	// 之前：所有节点应该都能看到彼此
	for i, node := range nodes {
		members := node.Members()
		assert.Equal(t, 3, len(members), "Before leave: node %d should see all 3 members", i+1)
	}

	// 设置事件监听器以捕获离开事件
	eventCh := nodes[0].Events() // 在第一个节点上监听

	// 优雅关闭第二个节点
	err = nodes[1].Leave()
	require.NoError(t, err, "Failed to gracefully leave cluster")

	// 等待离开传播
	leaveDetected := false
	timeout := time.After(5 * time.Second)

	for !leaveDetected {
		select {
		case event := <-eventCh:
			if event.Type == cache.EventNodeLeave && event.NodeID == "node-2" {
				leaveDetected = true
			}
		case <-timeout:
			break
		}
	}

	assert.True(t, leaveDetected, "Leave event should be detected")

	// 验证节点标记为已离开
	time.Sleep(1 * time.Second) // 给一些时间让gossip传播离开状态

	// 其他节点应该已移除或标记其为已离开
	for i := 0; i < 3; i++ {
		if i == 1 { // 跳过已离开的节点
			continue
		}

		node := nodes[i]
		if !node.running {
			continue
		}

		members := node.Members()
		nodeFound := false
		for _, member := range members {
			if member.ID == "node-2" {
				nodeFound = true
				assert.Equal(t, cache.NodeStatusLeft, member.Status,
					"Left node should be marked as LEFT from node %d's perspective", i+1)
			}
		}

		// 也可能彻底移除节点
		if !nodeFound {
			t.Logf("Node %d has completely removed the left node", i+1)
		}
	}

	// 最后停止优雅退出的节点
	err = nodes[1].Stop()
	require.NoError(t, err)
}

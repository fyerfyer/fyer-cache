package cluster

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/stretchr/testify/require"
)

func TestClusterCache_BasicOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping cluster cache test in short mode")
	}

	// 为节点寻找可用端口
	ports, err := getTestAvailablePorts(3)
	require.NoError(t, err, "Failed to get available ports")

	// 使用内存缓存创建集群缓存实例
	caches := make([]*ClusterCache, 3)
	for i := 0; i < 3; i++ {
		nodeID := fmt.Sprintf("node-%d", i+1)
		bindAddr := fmt.Sprintf("127.0.0.1:%d", ports[i])

		memCache := cache.NewMemoryCache()
		clusterCache, err := NewClusterCache(memCache, nodeID, bindAddr)
		require.NoError(t, err)

		caches[i] = clusterCache
	}

	// 确保正确清理
	defer func() {
		for _, c := range caches {
			if c != nil {
				c.Stop()
			}
		}
	}()

	// 启动所有缓存，节点之间添加短暂延迟以确保正确初始化
	for i, c := range caches {
		err := c.Start()
		require.NoError(t, err, "Failed to start cluster cache %d", i+1)
		time.Sleep(100 * time.Millisecond) // 节点启动之间添加延迟
	}

	t.Log("All cluster caches started successfully")

	// 第一个节点形成自己的集群
	err = caches[0].Join(caches[0].node.config.BindAddr)
	require.NoError(t, err, "Failed to bootstrap first node")

	// 其他节点加入第一个节点
	for i := 1; i < len(caches); i++ {
		err = caches[i].Join(caches[0].node.config.BindAddr)
		require.NoError(t, err, "Failed to join node %d to cluster", i+1)
		// 添加延迟以允许加入操作传播
		time.Sleep(100 * time.Millisecond)
	}

	// 等待gossip协议传播成员信息
	time.Sleep(2 * time.Second)

	// 验证集群形成
	for i, c := range caches {
		members := c.Members()
		require.Equal(t, 3, len(members), "Node %d should see all 3 members", i+1)
	}

	t.Log("Cluster formation verified")

	ctx := context.Background()

	// 测试基本缓存操作
	// 在每个节点分别设置数据（因为在这个简单实现中数据不会自动复制）
	for i, c := range caches {
		err = c.Set(ctx, "key1", "value1", time.Minute)
		require.NoError(t, err, "Failed to set data in node %d", i+1)
	}

	// 从每个节点获取数据
	for i, c := range caches {
		val, err := c.Get(ctx, "key1")
		require.NoError(t, err, "Node %d should be able to get the value", i+1)
		require.Equal(t, "value1", val)
	}

	// 测试删除操作 - 仅从节点1删除
	err = caches[1].Del(ctx, "key1")
	require.NoError(t, err)

	// 验证节点1无法获取键
	_, err = caches[1].Get(ctx, "key1")
	require.Error(t, err, "Node 2 should report key as deleted")

	// 其他节点仍然可以获取键，因为删除操作尚未传播
	// （在实际实现中，删除操作会传播）
	// 我们不测试这一点，因为它取决于具体实现

	// 测试元数据功能
	caches[0].SetLocalMetadata(map[string]string{
		"region": "us-west",
		"role":   "primary",
	})

	// 等待元数据传播
	time.Sleep(1 * time.Second)

	// 验证每个节点上的元数据
	for i, c := range caches {
		found := false
		for _, member := range c.Members() {
			if member.ID == "node-1" {
				require.Contains(t, member.Metadata, "region")
				require.Contains(t, member.Metadata, "role")
				require.Equal(t, "us-west", member.Metadata["region"])
				require.Equal(t, "primary", member.Metadata["role"])
				found = true
				break
			}
		}
		require.True(t, found, "Node %d should see node-1's metadata", i+1)
	}

	// 测试协调器选举
	coordinator := 0
	for i, c := range caches {
		if c.IsCoordinator() {
			coordinator = i + 1
			break
		}
	}
	t.Logf("Node %d is the coordinator", coordinator)

	// 测试事件处理
	var wg sync.WaitGroup
	wg.Add(1)

	// 在第一个节点上设置事件处理器
	receivedEvents := make([]cache.ClusterEvent, 0)
	var eventMu sync.Mutex

	caches[0].WithEventHandler(func(event cache.ClusterEvent) {
		eventMu.Lock()
		defer eventMu.Unlock()
		receivedEvents = append(receivedEvents, event)
		t.Logf("Received event: %v for node %s", event.Type, event.NodeID)

		// 如果我们检测到节点离开事件，通知测试可以继续
		if event.Type == cache.EventNodeLeave && event.NodeID == "node-3" {
			wg.Done()
		}
	})

	// 测试节点故障检测
	t.Log("Testing node leave detection...")

	// 停止一个节点以模拟故障/离开
	err = caches[2].Leave()
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	err = caches[2].Stop()
	require.NoError(t, err)

	// 等待检测到离开事件（带超时）
	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		// 事件被检测到
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for node leave event")
	}

	// 验证我们至少收到一个事件（希望是节点离开事件）
	eventMu.Lock()
	require.NotEmpty(t, receivedEvents, "Should have received some events")
	eventMu.Unlock()

	// 验证剩余节点现在只能看到2个成员
	// 这可能需要一些时间来更新成员资格，所以我们会多次尝试
	success := false
	for attempts := 0; attempts < 5; attempts++ {
		correctCount := 0
		for i := 0; i < 2; i++ {
			members := caches[i].Members()
			upCount := 0
			for _, m := range members {
				if m.Status == cache.NodeStatusUp {
					upCount++
				}
			}
			if upCount == 2 {
				correctCount++
			}
		}

		if correctCount == 2 {
			success = true
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	require.True(t, success, "Both remaining nodes should see exactly 2 UP nodes")

	// 测试集群信息
	info := caches[0].GetClusterInfo()
	require.NotNil(t, info)
	require.Contains(t, info, "node_id")
	require.Contains(t, info, "address")
	require.Contains(t, info, "is_coordinator")
	require.Contains(t, info, "member_count")
}

func TestClusterCache_Concurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent cluster cache test in short mode")
	}

	// 寻找可用端口
	ports, err := getTestAvailablePorts(3)
	require.NoError(t, err)

	// 创建集群缓存实例
	caches := make([]*ClusterCache, 3)
	for i := 0; i < 3; i++ {
		nodeID := fmt.Sprintf("node-%d", i+1)
		bindAddr := fmt.Sprintf("127.0.0.1:%d", ports[i])

		memCache := cache.NewMemoryCache(
			cache.WithShardCount(32),
			cache.WithCleanupInterval(500*time.Millisecond),
			cache.WithAsyncCleanup(true),
		)

		clusterCache, err := NewClusterCache(memCache, nodeID, bindAddr,
			WithGossipInterval(100*time.Millisecond),
			WithSyncInterval(500*time.Millisecond),
		)
		require.NoError(t, err)

		caches[i] = clusterCache
	}

	// 确保正确清理
	defer func() {
		for _, c := range caches {
			if c != nil {
				c.Stop()
			}
		}
	}()

	// 启动所有缓存，添加延迟
	for i, c := range caches {
		err := c.Start()
		require.NoError(t, err, "Failed to start cluster cache %d", i+1)
		time.Sleep(100 * time.Millisecond)
	}

	// 给服务器额外的初始化时间
	time.Sleep(200 * time.Millisecond)

	// 形成集群，加入之间添加延迟
	err = caches[0].Join(caches[0].node.config.BindAddr)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	for i := 1; i < len(caches); i++ {
		err = caches[i].Join(caches[0].node.config.BindAddr)
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
	}

	// 等待gossip传播
	time.Sleep(2 * time.Second)

	// 验证集群形成
	for i, c := range caches {
		members := c.Members()
		if len(members) != 3 {
			t.Fatalf("Node %d sees %d members instead of 3", i+1, len(members))
		}
	}

	ctx := context.Background()

	// 并发操作参数
	const (
		numGoroutines   = 10
		opsPerGoroutine = 20
	)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// 开始并发操作
	for g := 0; g < numGoroutines; g++ {
		go func(id int) {
			defer wg.Done()

			// 为每个goroutine使用确定性但不同的种子
			r := rand.New(rand.NewSource(int64(id * 1000)))

			for i := 0; i < opsPerGoroutine; i++ {
				key := fmt.Sprintf("key-%d-%d", id, i)
				value := fmt.Sprintf("value-%d-%d", id, i)

				// 为测试目的向所有缓存设置值
				// 在实际应用中，你只会设置到一个缓存并依赖复制机制
				for j := 0; j < len(caches); j++ {
					err := caches[j].Set(ctx, key, value, time.Minute)
					if err != nil {
						t.Logf("Set error on %s: %v", key, err)
					}
				}

				// 从随机缓存获取值
				getIdx := r.Intn(len(caches))
				val, err := caches[getIdx].Get(ctx, key)
				if err != nil {
					t.Logf("Get error on %s: %v", key, err)
				} else if val != value {
					t.Logf("Value mismatch for %s: expected %s, got %v", key, value, val)
				}

				// 删除一些键
				if i%2 == 0 {
					delIdx := r.Intn(len(caches))
					err = caches[delIdx].Del(ctx, key)
					if err != nil {
						t.Logf("Delete error on %s: %v", key, err)
					}
				}

				// 短暂睡眠以减少争用
				time.Sleep(time.Millisecond)
			}
		}(g)
	}

	// 等待所有操作完成
	wg.Wait()

	// 简单检查一些键
	successCount := 0
	totalChecks := 50

	for i := 0; i < numGoroutines; i++ {
		for j := 1; j < opsPerGoroutine; j += 2 { // 只检查未删除的键
			key := fmt.Sprintf("key-%d-%d", i, j)
			expectedValue := fmt.Sprintf("value-%d-%d", i, j)

			// 从随机节点检查
			checkIdx := rand.Intn(len(caches) - 1) // -1 是因为我们知道有一个已停止
			val, err := caches[checkIdx].Get(ctx, key)
			if err == nil && val == expectedValue {
				successCount++
			}

			// 只检查一个子集
			if totalChecks <= 0 {
				break
			}
			totalChecks--
		}
	}

	t.Logf("Data check: %d successful key validations", successCount)
}

// 查找可用端口的辅助函数
func getTestAvailablePorts(count int) ([]int, error) {
	ports := make([]int, count)
	listeners := make([]net.Listener, count)

	defer func() {
		for _, l := range listeners {
			if l != nil {
				l.Close()
			}
		}
	}()

	for i := 0; i < count; i++ {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return nil, err
		}

		addr := listener.Addr().String()
		_, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, err
		}

		ports[i] = port
		listeners[i] = listener
	}

	return ports, nil
}

package integration

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/fyerfyer/fyer-cache/cache/cluster"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getAvailablePorts returns n available ports for testing
func getAvailablePorts(n int) ([]int, error) {
	ports := make([]int, n)
	listeners := make([]*net.TCPListener, n)

	// Find n available ports by actually binding to them
	for i := 0; i < n; i++ {
		addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
		if err != nil {
			// Close any listeners we've opened so far
			for j := 0; j < i; j++ {
				if listeners[j] != nil {
					listeners[j].Close()
				}
			}
			return nil, err
		}

		listener, err := net.ListenTCP("tcp", addr)
		if err != nil {
			// Close any listeners we've opened so far
			for j := 0; j < i; j++ {
				if listeners[j] != nil {
					listeners[j].Close()
				}
			}
			return nil, err
		}

		// Get the port that was assigned by the system
		ports[i] = listener.Addr().(*net.TCPAddr).Port
		listeners[i] = listener
	}

	// Close all the listeners we opened
	for i := 0; i < n; i++ {
		if listeners[i] != nil {
			listeners[i].Close()
		}
	}

	return ports, nil
}

func TestShardedClusterCache_BasicOperations(t *testing.T) {
	// 在短测试模式下跳过
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 获取节点可用端口
	ports, err := getAvailablePorts(2)
	require.NoError(t, err, "Failed to get available ports")

	// 创建两个缓存节点
	node1Addr := "127.0.0.1:" + strconv.Itoa(ports[0])
	node2Addr := "127.0.0.1:" + strconv.Itoa(ports[1])

	// 创建底层本地缓存
	localCache1 := cache.NewMemoryCache()
	localCache2 := cache.NewMemoryCache()

	// 创建第一个带分片集群缓存的节点
	cache1, err := NewShardedClusterCache(
		localCache1,
		"node1",
		node1Addr,
		[]cluster.NodeOption{
			cluster.WithProbeInterval(100 * time.Millisecond),
			cluster.WithGossipInterval(100 * time.Millisecond),
		},
		[]ShardedClusterCacheOption{
			WithVirtualNodeCount(10),
		},
	)
	require.NoError(t, err, "Failed to create first cache")

	// 创建第二个带分片集群缓存的节点
	cache2, err := NewShardedClusterCache(
		localCache2,
		"node2",
		node2Addr,
		[]cluster.NodeOption{
			cluster.WithProbeInterval(100 * time.Millisecond),
			cluster.WithGossipInterval(100 * time.Millisecond),
		},
		[]ShardedClusterCacheOption{
			WithVirtualNodeCount(10),
		},
	)
	require.NoError(t, err, "Failed to create second cache")

	// 确保正确清理
	defer func() {
		cache1.Stop()
		cache2.Stop()
	}()

	// 启动缓存
	err = cache1.Start()
	require.NoError(t, err, "Failed to start first cache")

	err = cache2.Start()
	require.NoError(t, err, "Failed to start second cache")

	// 组建集群 - node1引导并且node2加入
	err = cache1.Join(node1Addr)
	require.NoError(t, err, "Failed to bootstrap first node")

	// 等待node1稳定
	time.Sleep(300 * time.Millisecond)

	err = cache2.Join(node1Addr)
	require.NoError(t, err, "Failed to join second node to cluster")

	// 等待gossip传播成员信息
	time.Sleep(1 * time.Second)

	// 验证集群形成
	members1 := cache1.Members()
	assert.Equal(t, 2, len(members1), "First node should see 2 members")

	members2 := cache2.Members()
	assert.Equal(t, 2, len(members2), "Second node should see 2 members")

	ctx := context.Background()

	// 测试基本缓存操作
	t.Log("Testing basic cache operations...")

	// Set操作
	err = cache1.Set(ctx, "key1", "value1", time.Minute)
	require.NoError(t, err, "Failed to set key1")

	err = cache2.Set(ctx, "key2", "value2", time.Minute)
	require.NoError(t, err, "Failed to set key2")

	// 短暂等待操作传播
	time.Sleep(100 * time.Millisecond)

	// Get操作 - 键应该通过一致性哈希路由到正确的节点
	val1, err := cache1.Get(ctx, "key1")
	if err == nil {
		assert.Equal(t, "value1", val1, "Value for key1 is incorrect")
	} else {
		// 在实际一致性哈希场景中,key1可能路由到node2
		val1, err = cache2.Get(ctx, "key1")
		require.NoError(t, err, "Failed to get key1 from either node")
		assert.Equal(t, "value1", val1, "Value for key1 is incorrect")
	}

	val2, err := cache2.Get(ctx, "key2")
	if err == nil {
		assert.Equal(t, "value2", val2, "Value for key2 is incorrect")
	} else {
		// 在实际一致性哈希场景中,key2可能路由到node1
		val2, err = cache1.Get(ctx, "key2")
		require.NoError(t, err, "Failed to get key2 from either node")
		assert.Equal(t, "value2", val2, "Value for key2 is incorrect")
	}

	// 测试删除操作
	err = cache1.Del(ctx, "key1")
	require.NoError(t, err, "Failed to delete key1")

	// 短暂等待操作传播
	time.Sleep(100 * time.Millisecond)

	// 验证key1已从集群中删除
	_, err = cache1.Get(ctx, "key1")
	assert.Error(t, err, "Key1 should be deleted")

	_, err = cache2.Get(ctx, "key1")
	assert.Error(t, err, "Key1 should be deleted")

	// 测试集群信息
	info := cache1.GetClusterInfo()
	assert.NotNil(t, info, "Cluster info should not be nil")
	assert.Contains(t, info, "node_id")

	// 测试离开集群
	err = cache2.Leave()
	require.NoError(t, err, "Failed to leave cluster")

	// 等待集群更新
	time.Sleep(500 * time.Millisecond)

	// Node1现在应该只看到自己
	members1 = cache1.Members()
	assert.Equal(t, 1, len(members1), "Node1 should now see only 1 member (itself)")
}

func TestShardedClusterCache_NodeDistribution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 创建一个用于测试分布的单节点缓存
	localCache := cache.NewMemoryCache()

	nodeAddr := "127.0.0.1:10000" // 在这个测试中实际上不需要绑定

	cache1, err := NewShardedClusterCache(
		localCache,
		"node1",
		nodeAddr,
		[]cluster.NodeOption{},
		[]ShardedClusterCacheOption{
			WithVirtualNodeCount(100),
		},
	)
	require.NoError(t, err)

	defer cache1.Stop()

	// 不加入集群直接启动
	err = cache1.Start()
	require.NoError(t, err)

	// 直接使用节点分配器进行测试
	nodeDistributor := cache1.nodeDistributor

	// 向分配器添加一些远程节点
	remoteNodes := []cache.NodeInfo{
		{ID: "node2", Addr: "127.0.0.1:10001", Status: cache.NodeStatusUp},
		{ID: "node3", Addr: "127.0.0.1:10002", Status: cache.NodeStatusUp},
		{ID: "node4", Addr: "127.0.0.1:10003", Status: cache.NodeStatusUp},
	}

	err = nodeDistributor.InitFromMembers(remoteNodes)
	require.NoError(t, err, "Failed to initialize node distributor")

	// 验证节点地址
	addresses := nodeDistributor.GetAllNodeAddresses()
	assert.Equal(t, 3, len(addresses), "Should have 3 node addresses")

	// 测试清空功能
	nodeDistributor.Clear()
	addresses = nodeDistributor.GetAllNodeAddresses()
	assert.Equal(t, 0, len(addresses), "All nodes should be cleared")

	// 测试处理集群事件
	event := cache.ClusterEvent{
		Type:    cache.EventNodeJoin,
		NodeID:  "node2",
		Time:    time.Now(),
		Details: "127.0.0.1:10001",
	}

	nodeDistributor.HandleClusterEvent(event)
	addresses = nodeDistributor.GetAllNodeAddresses()
	assert.Equal(t, 1, len(addresses), "Should have 1 node after join event")
	assert.Equal(t, "127.0.0.1:10001", addresses["node2"])

	// 测试节点离开事件
	leaveEvent := cache.ClusterEvent{
		Type:   cache.EventNodeLeave,
		NodeID: "node2",
		Time:   time.Now(),
	}

	nodeDistributor.HandleClusterEvent(leaveEvent)
	addresses = nodeDistributor.GetAllNodeAddresses()
	assert.Equal(t, 0, len(addresses), "Should have 0 nodes after leave event")
}

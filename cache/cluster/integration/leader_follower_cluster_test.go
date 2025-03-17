package integration

import (
	"context"
	"fmt"
	"github.com/fyerfyer/fyer-cache/cache"
	"net"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/cache/replication"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLeaderFollowerCluster_Creation 测试集群创建
func TestLeaderFollowerCluster_Creation(t *testing.T) {
	// 创建一个主节点集群
	leaderCluster, err := createTestCluster(t, "leader1", replication.RoleLeader)
	require.NoError(t, err, "Failed to create leader cluster")
	defer leaderCluster.Stop()

	assert.Equal(t, "leader1", leaderCluster.nodeID)
	assert.Equal(t, replication.RoleLeader, leaderCluster.currentRole)
	assert.NotNil(t, leaderCluster.localCache)
	assert.NotNil(t, leaderCluster.log)
	assert.NotNil(t, leaderCluster.syncer)
	assert.NotNil(t, leaderCluster.leaderNode)
	assert.Nil(t, leaderCluster.followerNode)

	// 创建一个从节点集群
	followerCluster, err := createTestCluster(t, "follower1", replication.RoleFollower)
	require.NoError(t, err, "Failed to create follower cluster")
	defer followerCluster.Stop()

	assert.Equal(t, "follower1", followerCluster.nodeID)
	assert.Equal(t, replication.RoleFollower, followerCluster.currentRole)
	assert.NotNil(t, followerCluster.followerNode)
	assert.Nil(t, followerCluster.leaderNode)
}

// TestLeaderFollowerCluster_StartStop 测试集群启动和停止
func TestLeaderFollowerCluster_StartStop(t *testing.T) {
	// 创建集群
	testCluster, err := createTestCluster(t, "node1", replication.RoleLeader)
	require.NoError(t, err, "Failed to create test cluster")

	// 测试启动
	err = testCluster.Start()
	require.NoError(t, err, "Failed to start cluster")
	assert.True(t, testCluster.running)

	// 测试重复启动
	err = testCluster.Start()
	require.NoError(t, err, "Starting an already running cluster should not error")

	// 测试停止
	err = testCluster.Stop()
	require.NoError(t, err, "Failed to stop cluster")
	assert.False(t, testCluster.running)

	// 测试重复停止
	err = testCluster.Stop()
	require.NoError(t, err, "Stopping an already stopped cluster should not error")
}

// TestLeaderFollowerCluster_JoinLeave 测试集群加入和离开
func TestLeaderFollowerCluster_JoinLeave(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// 创建主节点集群
	leader, err := createTestCluster(t, "leader1", replication.RoleLeader)
	require.NoError(t, err)

	// 启动主节点
	err = leader.Start()
	require.NoError(t, err)
	defer leader.Stop()

	// 创建从节点集群
	follower, err := createTestCluster(t, "follower1", replication.RoleFollower)
	require.NoError(t, err)

	// 启动从节点
	err = follower.Start()
	require.NoError(t, err)
	defer follower.Stop()

	// 主节点自引导形成集群
	err = leader.Join(leader.address)
	require.NoError(t, err)

	// 等待集群稳定
	time.Sleep(500 * time.Millisecond)

	// 从节点加入集群
	err = follower.Join(leader.address)
	require.NoError(t, err)

	// 等待从节点加入
	time.Sleep(500 * time.Millisecond)

	// 验证集群成员
	leaderMembers := leader.clusterNode.Members()
	assert.Len(t, leaderMembers, 2, "Leader should see 2 nodes")

	followerMembers := follower.clusterNode.Members()
	assert.Len(t, followerMembers, 2, "Follower should see 2 nodes")

	// 测试离开集群
	err = follower.Leave()
	require.NoError(t, err)

	// 等待节点离开传播
	time.Sleep(500 * time.Millisecond)

	// 验证集群成员
	leaderMembers = leader.clusterNode.Members()
	assert.Len(t, leaderMembers, 1, "Leader should see only itself now")
}

// TestLeaderFollowerCluster_DataReplication 测试数据复制
func TestLeaderFollowerCluster_DataReplication(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	// 创建主节点和从节点
	leader, err := createTestCluster(t, "leader1", replication.RoleLeader)
	require.NoError(t, err, "Failed to create leader node")

	follower, err := createTestCluster(t, "follower1", replication.RoleFollower)
	require.NoError(t, err, "Failed to create follower node")

	// 启动节点
	err = leader.Start()
	require.NoError(t, err, "Failed to start leader node")
	defer leader.Stop()

	err = follower.Start()
	require.NoError(t, err, "Failed to start follower node")
	defer follower.Stop()

	// 给服务器一些时间启动
	time.Sleep(200 * time.Millisecond)

	// 形成集群
	err = leader.Join(leader.address)
	require.NoError(t, err, "Failed to bootstrap leader node")

	// 等待集群稳定
	time.Sleep(500 * time.Millisecond)

	err = follower.Join(leader.address)
	require.NoError(t, err, "Failed to join follower to cluster")

	// 等待集群稳定
	time.Sleep(1000 * time.Millisecond)

	// 添加从节点
	err = leader.AddFollower("follower1", follower.address)
	if err != nil {
		t.Logf("Warning: AddFollower returned error: %v (may be already added)", err)
	}

	// 等待同步设置完成
	time.Sleep(1 * time.Second)

	// 设置数据到主节点
	ctx := context.Background()
	err = leader.Set(ctx, "test-key", "test-value", time.Minute)
	require.NoError(t, err, "Failed to set test key")

	// 触发立即复制
	err = leader.ReplicateNow(ctx)
	require.NoError(t, err, "Failed to trigger immediate replication")

	// 给复制一些时间完成
	time.Sleep(1 * time.Second)

	// 从从节点查询数据
	val, err := follower.Get(ctx, "test-key")
	require.NoError(t, err, "Failed to get test key from follower")
	assert.Equal(t, "test-value", val, "Replicated value does not match")
}

// TestLeaderFollowerCluster_RoleChange 测试角色变更
func TestLeaderFollowerCluster_RoleChange(t *testing.T) {
	// 创建一个初始为领导者的节点
	node, err := createTestCluster(t, "node1", replication.RoleLeader)
	require.NoError(t, err)

	// 启动节点
	err = node.Start()
	require.NoError(t, err)
	defer node.Stop()

	// 验证初始角色
	assert.True(t, node.IsLeader())

	// 测试降级为跟随者
	err = node.DemoteToFollower("node2", "127.0.0.1:8000")
	require.NoError(t, err)
	assert.False(t, node.IsLeader())
	assert.Equal(t, "node2", node.GetLeader())

	// 测试提升为领导者
	err = node.PromoteToLeader()
	require.NoError(t, err)
	assert.True(t, node.IsLeader())
	assert.Equal(t, "node1", node.GetLeader())
}

// TestLeaderFollowerCluster_ClusterInfo 测试获取集群信息
func TestLeaderFollowerCluster_ClusterInfo(t *testing.T) {
	// 创建测试节点
	node, err := createTestCluster(t, "node1", replication.RoleLeader)
	require.NoError(t, err)

	// 启动节点
	err = node.Start()
	require.NoError(t, err)
	defer node.Stop()

	// 获取集群信息
	info := node.GetClusterInfo()

	// 验证基本信息
	assert.Equal(t, "node1", info["nodeID"])
	assert.Equal(t, "Leader", info["role"])
	assert.Equal(t, node.address, info["address"])

	// 验证从节点信息
	assert.NotNil(t, info["followers"])
	assert.Equal(t, 0, info["followerCount"])
}

// TestLeaderFollowerCluster_SyncFromLeader 测试从领导者同步数据
func TestLeaderFollowerCluster_SyncFromLeader(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// 创建主从节点
	leader, err := createTestCluster(t, "leader1", replication.RoleLeader)
	require.NoError(t, err)

	follower, err := createTestCluster(t, "follower1", replication.RoleFollower)
	require.NoError(t, err)

	// 启动节点
	err = leader.Start()
	require.NoError(t, err)
	defer leader.Stop()

	err = follower.Start()
	require.NoError(t, err)
	defer follower.Stop()

	// 形成集群
	err = leader.Join(leader.address)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	err = follower.Join(leader.address)
	require.NoError(t, err)

	// 等待集群稳定
	time.Sleep(500 * time.Millisecond)

	// 添加从节点
	err = leader.AddFollower("follower1", follower.address)
	require.NoError(t, err)

	// 在主节点设置一些数据
	ctx := context.Background()
	err = leader.Set(ctx, "key1", []byte("value1"), time.Minute)
	require.NoError(t, err)

	err = leader.Set(ctx, "key2", []byte("value2"), time.Minute)
	require.NoError(t, err)

	// 触发从节点主动同步
	err = follower.SyncFromLeader(ctx)
	require.NoError(t, err)

	// 给同步一些时间完成
	time.Sleep(200 * time.Millisecond)

	// 验证数据已同步
	val1, err := follower.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value1"), val1)

	val2, err := follower.Get(ctx, "key2")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value2"), val2)
}

// createTestCluster 创建一个测试集群
func createTestCluster(t *testing.T, nodeID string, role replication.ReplicationRole) (*LeaderFollowerCluster, error) {
	// 获取可用端口
	port, err := getAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to get available port: %w", err)
	}

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	cache := cache.NewMemoryCache()

	// 创建集群
	cluster, err := NewLeaderFollowerCluster(
		nodeID,
		addr,
		cache,
		WithInitialRole(role),
	)
	if err != nil {
		return nil, err
	}

	t.Logf("HTTP server for %s listening on %s", nodeID, addr)
	return cluster, nil
}

// getAvailablePort 获取可用端口
func getAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port, nil
}
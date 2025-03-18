package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fyerfyer/fyer-cache/cache"
	"net"
	"net/http"
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

// TestLeaderFollowerCluster_SyncFromLeader 测试从Leader同步数据
func TestLeaderFollowerCluster_SyncFromLeader(t *testing.T) {
	// 创建一个主节点集群
	leaderCluster, err := createTestCluster(t, "leader1", replication.RoleLeader)
	require.NoError(t, err, "Failed to create leader cluster")

	// 启动主节点
	err = leaderCluster.Start()
	require.NoError(t, err, "Failed to start leader cluster")
	defer leaderCluster.Stop()

	// 创建一个从节点集群
	followerCluster, err := createTestCluster(t, "follower1", replication.RoleFollower)
	require.NoError(t, err, "Failed to create follower cluster")

	// 启动从节点
	err = followerCluster.Start()
	require.NoError(t, err, "Failed to start follower cluster")
	defer followerCluster.Stop()

	// 主节点自引导形成集群
	err = leaderCluster.Join(leaderCluster.address)
	require.NoError(t, err, "Failed to bootstrap leader")

	// 设置从节点的领导者 - 使用同步器地址而不是集群地址
	followerCluster.followerNode.SetLeader(leaderCluster.nodeID, leaderCluster.config.SyncerAddress)


	// 从节点加入集群
	err = followerCluster.Join(leaderCluster.address)
	require.NoError(t, err, "Failed to join follower to cluster")

	// 等待集群稳定
	time.Sleep(500 * time.Millisecond)

	// 在主节点写入数据
	ctx := context.Background()
	testKey := "sync-test-key"
	testValue := "sync-test-value"

	err = leaderCluster.Set(ctx, testKey, testValue, time.Minute)
	require.NoError(t, err, "Failed to set data in leader")

	// 创建一个有超时的上下文，避免无限同步
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// 从从节点同步数据 - 使用带超时的上下文
	err = followerCluster.SyncFromLeader(timeoutCtx)
	// 即使超时也不视为错误，因为数据可能已经同步
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		require.NoError(t, err, "Failed to sync from leader")
	}

	// 给同步一些时间完成
	time.Sleep(500 * time.Millisecond)

	// 验证从节点是否已同步数据
	val, err := followerCluster.Get(ctx, testKey)
	assert.NoError(t, err, "Failed to get synced data from follower")
	assert.Equal(t, testValue, val, "Follower data does not match leader data")
}

func TestLeaderFollowerCluster_HTTPServerSetup(t *testing.T) {
	// 测试使用动态端口分配进行正确的HTTP服务器初始化
	leader, err := createTestCluster(t, "leader-http-test", replication.RoleLeader)
	require.NoError(t, err)
	defer leader.Stop()

	// 启动leader节点
	err = leader.Start()
	require.NoError(t, err)

	// 验证HTTP服务器是否在预期地址上监听
	// 这可以通过发送简单的健康检查请求来完成
	syncerAddr := leader.config.SyncerAddress // 使用同步器地址，而不是集群地址
	resp, err := http.Get(fmt.Sprintf("http://%s/replication/health", syncerAddr))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestLeaderFollowerCluster_FollowerManagement(t *testing.T) {
	// 创建leader和follower节点
	leader, err := createTestCluster(t, "leader-mgmt-test", replication.RoleLeader)
	require.NoError(t, err)
	defer leader.Stop()

	follower, err := createTestCluster(t, "follower-mgmt-test", replication.RoleFollower)
	require.NoError(t, err)
	defer follower.Stop()

	// 启动两个节点
	err = leader.Start()
	require.NoError(t, err)

	err = follower.Start()
	require.NoError(t, err)

	// 测试添加follower
	err = leader.AddFollower(follower.nodeID, follower.address)
	require.NoError(t, err)

	// 测试重复添加follower（应该返回错误）
	err = leader.AddFollower(follower.nodeID, follower.address)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")

	// 测试移除follower
	err = leader.RemoveFollower(follower.nodeID)
	require.NoError(t, err)

	// 测试移除不存在的follower（应该返回错误）
	err = leader.RemoveFollower("non-existent")
	require.Error(t, err)
}

func TestLeaderFollowerCluster_ReplicationEndpoints(t *testing.T) {
	// 测试同步端点的可用性和通信
	leader, err := createTestCluster(t, "leader-sync-test", replication.RoleLeader)
	require.NoError(t, err)
	defer leader.Stop()

	follower, err := createTestCluster(t, "follower-sync-test", replication.RoleFollower)
	require.NoError(t, err)
	defer follower.Stop()

	// 启动两个节点
	err = leader.Start()
	require.NoError(t, err)

	err = follower.Start()
	require.NoError(t, err)

	// 验证follower上的同步端点是否可用
	// 使用同步器地址而不是集群地址
	syncURL := fmt.Sprintf("http://%s/replication/sync", follower.config.SyncerAddress)
	req := &replication.SyncRequest{
		LeaderID: leader.nodeID,
		FullSync: true,
	}

	reqBody, err := json.Marshal(req)
	require.NoError(t, err)

	resp, err := http.Post(syncURL, "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestLeaderFollowerCluster_SimpleDataReplication(t *testing.T) {
	// 创建leader和follower节点
	leader, err := createTestCluster(t, "leader-data-test", replication.RoleLeader)
	require.NoError(t, err)
	defer leader.Stop()

	follower, err := createTestCluster(t, "follower-data-test", replication.RoleFollower)
	require.NoError(t, err)
	defer follower.Stop()

	// 启动两个节点
	err = leader.Start()
	require.NoError(t, err)

	err = follower.Start()
	require.NoError(t, err)

	// 形成集群
	err = leader.Join(leader.address)
	require.NoError(t, err)

	// 让集群稳定下来
	time.Sleep(500 * time.Millisecond)

	// 显式添加follower（这应该在Join期间发生，但我们这里显式处理）
	err = leader.AddFollower(follower.nodeID, follower.config.SyncerAddress)
	require.NoError(t, err)

	// 等待同步设置完成
	time.Sleep(500 * time.Millisecond)

	// 在leader上设置数据
	ctx := context.Background()
	testKey := "test-replication-key"
	testValue := "test-replication-value"

	err = leader.Set(ctx, testKey, testValue, time.Minute)
	require.NoError(t, err)

	// 触发即时复制
	err = leader.ReplicateNow(ctx)
	require.NoError(t, err)

	// 给复制预留时间
	time.Sleep(500 * time.Millisecond)

	// 直接通过follower的缓存查询数据
	// 注意：在实际的分布式系统中，你应该通过缓存接口进行查询
	val, err := follower.localCache.Get(ctx, testKey)
	require.NoError(t, err)
	assert.Equal(t, testValue, val)
}

func TestReplicationSyncer_PortConfiguration(t *testing.T) {
	cache := cache.NewMemoryCache()
	log := replication.NewMemoryReplicationLog()

	// 使用指定地址创建同步器
	specificPort := 8088 // 选择一个可用端口
	addr := fmt.Sprintf("127.0.0.1:%d", specificPort)
	syncer := replication.NewMemorySyncer("test-node", cache, log,
		replication.WithAddress(addr))

	err := syncer.Start()
	require.NoError(t, err)
	defer syncer.Stop()

	// 验证服务器是否在指定端口上监听
	resp, err := http.Get(fmt.Sprintf("http://%s/replication/health", addr))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// createTestCluster 创建一个测试集群
func createTestCluster(t *testing.T, nodeID string, role replication.ReplicationRole) (*LeaderFollowerCluster, error) {
	// 获取两个可用端口 - 一个用于集群，一个用于复制
	clusterPort, err := getAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster port: %w", err)
	}

	syncerPort, err := getAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to get syncer port: %w", err)
	}

	clusterAddr := fmt.Sprintf("127.0.0.1:%d", clusterPort)
	syncerAddr := fmt.Sprintf("127.0.0.1:%d", syncerPort)

	cache := cache.NewMemoryCache()

	// 创建集群
	cluster, err := NewLeaderFollowerCluster(
		nodeID,
		clusterAddr,
		cache,
		WithInitialRole(role),
		WithSyncerAddress(syncerAddr), // 新选项，用于设置同步器地址
	)
	if err != nil {
		return nil, err
	}

	t.Logf("HTTP server for %s - Cluster: %s, Syncer: %s", nodeID, clusterAddr, syncerAddr)
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
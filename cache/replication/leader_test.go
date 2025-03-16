package replication

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLeaderNodeImpl_NewLeaderNode 测试创建新的 Leader 节点
func TestLeaderNodeImpl_NewLeaderNode(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	leader := NewLeaderNode("leader1", cache, log, syncer)
	require.NotNil(t, leader)

	// 验证基本属性
	assert.Equal(t, "leader1", leader.nodeID)
	assert.Equal(t, RoleLeader, leader.role)
	assert.Equal(t, "leader1", leader.leaderID) // 自己是领导者
	assert.NotNil(t, leader.followers)
	assert.NotNil(t, leader.followerStatus)
}

// TestLeaderNodeImpl_StartStop 测试启动和停止 Leader 节点
func TestLeaderNodeImpl_StartStop(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	leader := NewLeaderNode("leader1", cache, log, syncer)

	// 测试启动
	err := leader.Start()
	assert.NoError(t, err)
	assert.True(t, leader.running)
	assert.True(t, syncer.startCalled)

	// 测试重复启动
	err = leader.Start()
	assert.NoError(t, err) // 不应产生错误

	// 测试停止
	err = leader.Stop()
	assert.NoError(t, err)
	assert.False(t, leader.running)
	assert.True(t, syncer.stopCalled)

	// 测试重复停止
	err = leader.Stop()
	assert.NoError(t, err) // 不应产生错误
}

// TestLeaderNodeImpl_FollowerManagement 测试从节点管理功能
func TestLeaderNodeImpl_FollowerManagement(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	leader := NewLeaderNode("leader1", cache, log, syncer)
	require.NotNil(t, leader)

	// 测试添加从节点
	err := leader.AddFollower("follower1", "127.0.0.1:8001")
	require.NoError(t, err)

	// 验证从节点状态
	followers := leader.GetFollowers()
	assert.Len(t, followers, 1)
	assert.Equal(t, "127.0.0.1:8001", followers["follower1"])

	status := leader.followerStatus["follower1"]
	assert.Equal(t, StateOutOfSync, status)

	// 测试添加已存在的从节点
	err = leader.AddFollower("follower1", "127.0.0.1:8002")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// 测试更新从节点状态
	leader.UpdateFollowerStatus("follower1", StateNormal)
	status = leader.GetFollowerStatus()["follower1"]
	assert.Equal(t, StateNormal, status)

	// 测试更新不存在的从节点状态
	leader.UpdateFollowerStatus("non-existent", StateNormal)
	_, exists := leader.GetFollowerStatus()["non-existent"]
	assert.False(t, exists)

	// 测试移除从节点
	err = leader.RemoveFollower("follower1")
	require.NoError(t, err)

	// 验证从节点已移除
	followers = leader.GetFollowers()
	assert.Len(t, followers, 0)

	// 测试移除不存在的从节点
	err = leader.RemoveFollower("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

// TestLeaderNodeImpl_RoleManagement 测试角色管理功能
func TestLeaderNodeImpl_RoleManagement(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	leader := NewLeaderNode("leader1", cache, log, syncer)
	require.NotNil(t, leader)

	// 验证初始角色
	assert.Equal(t, RoleLeader, leader.GetRole())
	assert.Equal(t, "leader1", leader.GetLeader())

	// 更改角色到从节点
	leader.SetRole(RoleFollower)
	assert.Equal(t, RoleFollower, leader.GetRole())

	// 更改回领导者
	leader.SetRole(RoleLeader)
	assert.Equal(t, RoleLeader, leader.GetRole())

	// 更改到候选者
	leader.SetRole(RoleCandidate)
	assert.Equal(t, RoleCandidate, leader.GetRole())
}

// TestLeaderNodeImpl_ReplicateEntries 测试复制日志条目
func TestLeaderNodeImpl_ReplicateEntries(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	leader := NewLeaderNode("leader1", cache, log, syncer)
	require.NotNil(t, leader)

	// 添加从节点
	err := leader.AddFollower("follower1", "127.0.0.1:8001")
	require.NoError(t, err)

	err = leader.AddFollower("follower2", "127.0.0.1:8002")
	require.NoError(t, err)

	// 测试条目复制
	ctx := context.Background()
	err = leader.ReplicateEntries(ctx)
	assert.NoError(t, err)

	// 测试在非领导者角色下复制
	leader.SetRole(RoleFollower)
	err = leader.ReplicateEntries(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not the leader")
}

// TestLeaderNodeImpl_SendHeartbeat 测试发送心跳
func TestLeaderNodeImpl_SendHeartbeat(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	leader := NewLeaderNode("leader1", cache, log, syncer)
	require.NotNil(t, leader)

	// 空节点情况下不应报错
	ctx := context.Background()
	err := leader.SendHeartbeat(ctx)
	assert.NoError(t, err)

	// 添加从节点
	err = leader.AddFollower("follower1", "127.0.0.1:8001")
	require.NoError(t, err)

	// 测试心跳发送
	err = leader.SendHeartbeat(ctx)
	assert.NoError(t, err)
}

// TestLeaderNodeImpl_ApplyEntry 测试应用日志条目
func TestLeaderNodeImpl_ApplyEntry(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	leader := NewLeaderNode("leader1", cache, log, syncer)
	require.NotNil(t, leader)

	ctx := context.Background()

	// 测试应用Set条目
	setEntry := &ReplicationEntry{
		Command:    "Set",
		Key:        "testkey",
		Value:      []byte("testvalue"),
		Expiration: time.Minute,
	}

	err := leader.ApplyEntry(ctx, setEntry)
	assert.NoError(t, err)

	// 验证键已设置
	val, err := cache.Get(ctx, "testkey")
	assert.NoError(t, err)
	assert.Equal(t, []byte("testvalue"), val)

	// 测试应用Del条目
	delEntry := &ReplicationEntry{
		Command: "Del",
		Key:     "testkey",
	}

	err = leader.ApplyEntry(ctx, delEntry)
	assert.NoError(t, err)

	// 验证键已删除
	_, err = cache.Get(ctx, "testkey")
	assert.Error(t, err)

	// 测试应用空条目
	err = leader.ApplyEntry(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil entry")

	// 测试未知命令
	unknownEntry := &ReplicationEntry{
		Command: "Unknown",
		Key:     "testkey",
	}

	err = leader.ApplyEntry(ctx, unknownEntry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

// TestLeaderNodeImpl_GetLog 测试获取复制日志
func TestLeaderNodeImpl_GetLog(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	leader := NewLeaderNode("leader1", cache, log, syncer)
	require.NotNil(t, leader)

	// 测试获取日志
	assert.Equal(t, log, leader.GetLog())
}

// stubResponseSyncer 可以定制响应的模拟同步器
type stubResponseSyncer struct {
	mockSyncer
	syncError error
}

func (s *stubResponseSyncer) IncrementalSync(ctx context.Context, target string, startIndex uint64) error {
	return s.syncError
}

// TestLeaderNodeImpl_ReplicateFail 测试复制失败场景
func TestLeaderNodeImpl_ReplicateFail(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &stubResponseSyncer{
		syncError: errors.New("sync failed"),
	}

	leader := NewLeaderNode("leader1", cache, log, syncer)
	require.NotNil(t, leader)

	// 添加从节点
	err := leader.AddFollower("follower1", "127.0.0.1:8001")
	require.NoError(t, err)

	// 测试复制失败
	ctx := context.Background()
	err = leader.ReplicateEntries(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sync failed")

	// 验证从节点状态已更新为不同步
	status := leader.GetFollowerStatus()["follower1"]
	assert.Equal(t, StateOutOfSync, status)
}
package replication

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFollowerNodeImpl_NewFollowerNode 测试创建新的 Follower 节点
func TestFollowerNodeImpl_NewFollowerNode(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	follower := NewFollowerNode("follower1", cache, log, syncer)
	require.NotNil(t, follower)

	// 验证基本属性
	assert.Equal(t, "follower1", follower.nodeID)
	assert.Equal(t, RoleFollower, follower.role)
	assert.Equal(t, "", follower.leaderID) // 初始无领导者
	assert.NotNil(t, follower.stopCh)
	assert.False(t, follower.running)
	assert.Equal(t, StateOutOfSync, follower.syncState)
}

// TestFollowerNodeImpl_StartStop 测试启动和停止 Follower 节点
func TestFollowerNodeImpl_StartStop(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	follower := NewFollowerNode("follower1", cache, log, syncer)

	// 测试启动
	err := follower.Start()
	assert.NoError(t, err)
	assert.True(t, follower.running)

	// 测试重复启动
	err = follower.Start()
	assert.NoError(t, err) // 不应产生错误

	// 测试停止
	err = follower.Stop()
	assert.NoError(t, err)
	assert.False(t, follower.running)

	// 测试重复停止
	err = follower.Stop()
	assert.NoError(t, err) // 不应产生错误
}

// TestFollowerNodeImpl_LeaderManagement 测试领导者管理功能
func TestFollowerNodeImpl_LeaderManagement(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	follower := NewFollowerNode("follower1", cache, log, syncer)

	// 测试初始状态
	assert.Equal(t, "", follower.GetLeader())

	// 测试设置领导者
	follower.SetLeader("leader1", "127.0.0.1:8001")
	assert.Equal(t, "leader1", follower.GetLeader())
	assert.Equal(t, "127.0.0.1:8001", follower.leaderAddr)

	// 验证最后心跳时间被更新
	assert.False(t, follower.lastHeartbeat.IsZero())

	// 测试再次更改领导者
	oldHeartbeat := follower.lastHeartbeat
	time.Sleep(10 * time.Millisecond) // 确保时间戳变化

	follower.SetLeader("leader2", "127.0.0.1:8002")
	assert.Equal(t, "leader2", follower.GetLeader())
	assert.Equal(t, "127.0.0.1:8002", follower.leaderAddr)
	assert.True(t, follower.lastHeartbeat.After(oldHeartbeat)) // 心跳时间应更新
}

// TestFollowerNodeImpl_RoleManagement 测试角色管理功能
func TestFollowerNodeImpl_RoleManagement(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	follower := NewFollowerNode("follower1", cache, log, syncer)

	// 验证初始角色
	assert.Equal(t, RoleFollower, follower.GetRole())

	// 更改角色到候选者
	follower.SetRole(RoleCandidate)
	assert.Equal(t, RoleCandidate, follower.GetRole())

	// 更改回从节点
	follower.SetRole(RoleFollower)
	assert.Equal(t, RoleFollower, follower.GetRole())

	// 更改到领导者
	follower.SetRole(RoleLeader)
	assert.Equal(t, RoleLeader, follower.GetRole())
}

// TestFollowerNodeImpl_SyncFromLeader 测试从领导者同步数据
func TestFollowerNodeImpl_SyncFromLeader(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &stubResponseSyncer{
		syncError: nil, // 成功同步
	}

	follower := NewFollowerNode("follower1", cache, log, syncer)

	// 没有领导者时同步应该失败
	ctx := context.Background()
	err := follower.SyncFromLeader(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no leader available")

	// 设置领导者后同步
	follower.SetLeader("leader1", "127.0.0.1:8001")

	// 测试成功同步
	err = follower.SyncFromLeader(ctx)
	assert.NoError(t, err)
	assert.Equal(t, StateNormal, follower.GetSyncState())

	// 测试同步失败
	syncer.syncError = errors.New("sync failed")
	err = follower.SyncFromLeader(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sync failed")
	assert.Equal(t, StateOutOfSync, follower.GetSyncState())
}

// TestFollowerNodeImpl_RequestFullSync 测试请求全量同步
func TestFollowerNodeImpl_RequestFullSync(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &stubResponseSyncer{
		syncError: nil, // 成功同步
	}

	follower := NewFollowerNode("follower1", cache, log, syncer)

	// 没有领导者时同步应该失败
	ctx := context.Background()
	err := follower.RequestFullSync(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no leader available")

	// 设置领导者后同步
	follower.SetLeader("leader1", "127.0.0.1:8001")

	// 测试成功同步
	err = follower.RequestFullSync(ctx)
	assert.NoError(t, err)
	assert.Equal(t, StateNormal, follower.GetSyncState())

	// 测试同步失败
	syncer.syncError = errors.New("full sync failed")
	err = follower.RequestFullSync(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "full sync failed")
	assert.Equal(t, StateOutOfSync, follower.GetSyncState())
}

// TestFollowerNodeImpl_HandleHeartbeat 测试处理心跳消息
func TestFollowerNodeImpl_HandleHeartbeat(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	follower := NewFollowerNode("follower1", cache, log, syncer)
	ctx := context.Background()

	// 设置初始状态
	follower.currentTerm = 1

	// 测试接收到更高任期的心跳
	oldHeartbeat := follower.lastHeartbeat
	time.Sleep(10 * time.Millisecond) // 确保时间戳变化

	err := follower.HandleHeartbeat(ctx, "leader1", 2)
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), follower.GetCurrentTerm()) // 任期应该更新
	assert.True(t, follower.lastHeartbeat.After(oldHeartbeat)) // 心跳时间应更新
	assert.Equal(t, "leader1", follower.GetLeader()) // 领导者应更新

	// 测试接收到低任期的心跳 - 应该被接受，但不更新任期
	oldTerm := follower.currentTerm
	oldHeartbeat = follower.lastHeartbeat
	time.Sleep(10 * time.Millisecond)

	err = follower.HandleHeartbeat(ctx, "leader2", 1)
	assert.NoError(t, err)
	assert.Equal(t, oldTerm, follower.currentTerm) // 任期不应变化
	assert.True(t, follower.lastHeartbeat.After(oldHeartbeat)) // 心跳时间仍应更新
	assert.Equal(t, "leader1", follower.GetLeader()) // 领导者不应变化
}

// TestFollowerNodeImpl_PromoteToLeader 测试提升为领导者
func TestFollowerNodeImpl_PromoteToLeader(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	follower := NewFollowerNode("follower1", cache, log, syncer)
	ctx := context.Background()

	// 设置初始状态
	follower.currentTerm = 1
	follower.SetLeader("leader1", "127.0.0.1:8001")

	// 启动节点
	err := follower.Start()
	require.NoError(t, err)

	// 提升为领导者
	err = follower.PromoteToLeader(ctx)
	assert.NoError(t, err)

	// 验证状态变化
	assert.Equal(t, uint64(2), follower.currentTerm) // 任期应增加
	assert.Equal(t, RoleLeader, follower.GetRole()) // 角色应变为领导者
	assert.Equal(t, "follower1", follower.GetLeader()) // 领导者应为自己

	// 清理
	follower.Stop()
}

// TestFollowerNodeImpl_ApplyEntry 测试应用日志条目
func TestFollowerNodeImpl_ApplyEntry(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	follower := NewFollowerNode("follower1", cache, log, syncer)
	ctx := context.Background()

	// 测试应用Set条目
	setEntry := &ReplicationEntry{
		Command:    "Set",
		Key:        "testkey",
		Value:      []byte("testvalue"),
		Expiration: time.Minute,
	}

	err := follower.ApplyEntry(ctx, setEntry)
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

	err = follower.ApplyEntry(ctx, delEntry)
	assert.NoError(t, err)

	// 验证键已删除
	val, err = cache.Get(ctx, "testkey")
	assert.NoError(t, err)
	assert.Nil(t, val)

	// 测试应用空条目
	err = follower.ApplyEntry(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil entry")

	// 测试未知命令
	unknownEntry := &ReplicationEntry{
		Command: "Unknown",
		Key:     "testkey",
	}

	err = follower.ApplyEntry(ctx, unknownEntry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

// TestFollowerNodeImpl_ElectionCallback 测试选举回调
func TestFollowerNodeImpl_ElectionCallback(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	follower := NewFollowerNode("follower1", cache, log, syncer)

	// 测试设置选举回调
	callbackCalled := false
	follower.SetElectionCallback(func() {
		callbackCalled = true
	})

	// 调用选举回调（通常由心跳监控器调用）
	if follower.onLeaderFail != nil {
		follower.onLeaderFail()
	}

	// 验证回调被调用
	assert.True(t, callbackCalled)
}

// TestFollowerNodeImpl_GetLog 测试获取复制日志
func TestFollowerNodeImpl_GetLog(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	follower := NewFollowerNode("follower1", cache, log, syncer)

	// 测试获取日志
	retrievedLog := follower.GetLog()
	assert.Equal(t, log, retrievedLog)
}

// TestFollowerNodeImpl_GetSyncState 测试获取同步状态
func TestFollowerNodeImpl_GetSyncState(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	follower := NewFollowerNode("follower1", cache, log, syncer)

	// 验证初始状态
	assert.Equal(t, StateOutOfSync, follower.GetSyncState())

	// 更新状态
	follower.syncState = StateSyncing
	assert.Equal(t, StateSyncing, follower.GetSyncState())
}

// TestFollowerNodeImpl_UpdateLastHeartbeat 测试更新最后心跳时间
func TestFollowerNodeImpl_UpdateLastHeartbeat(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	follower := NewFollowerNode("follower1", cache, log, syncer)

	initialHeartbeat := follower.lastHeartbeat
	time.Sleep(10 * time.Millisecond) // 确保时间戳变化

	follower.UpdateLastHeartbeat()
	assert.True(t, follower.lastHeartbeat.After(initialHeartbeat))
}

// stubResponseSyncer 可以定制响应的模拟同步器
type stubSyncerForFollower struct {
	mockSyncer
	syncError error
}

func (s *stubSyncerForFollower) FullSync(ctx context.Context, target string) error {
	return s.syncError
}

func (s *stubSyncerForFollower) IncrementalSync(ctx context.Context, target string, startIndex uint64) error {
	return s.syncError
}
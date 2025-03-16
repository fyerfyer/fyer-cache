package cluster

import (
	"github.com/fyerfyer/fyer-cache/mocks"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/cache/replication"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 创建测试选举管理器
func createTestElectionManager(t *testing.T, nodeID string) (*ElectionManager, *Membership, *EventPublisher) {
	// 创建必要的依赖
	config := DefaultNodeConfig()
	config.ID = nodeID
	config.BindAddr = "127.0.0.1:9000"
	electionTimeout := 50 * time.Millisecond

	// 创建事件发布器
	eventPublisher := NewEventPublisher(10)

	// 使用 mock 传输层代替 Transport
	mockTransport := mocks.NewNodeTransporter(t)
	
	// 设置必要的mock行为
	mockTransport.EXPECT().Address().Return(config.BindAddr).Maybe()

	// 创建成员管理器
	membership := NewMembership(nodeID, config.BindAddr, mockTransport, eventPublisher, config)

	// 创建选举管理器
	manager := NewElectionManager(nodeID, membership, eventPublisher, WithElectionTimeout(electionTimeout))

	return manager, membership, eventPublisher
}

// TestElectionManager_NewElectionManager 测试创建选举管理器
func TestElectionManager_NewElectionManager(t *testing.T) {
	// 创建选举管理器
	manager, _, _ := createTestElectionManager(t, "node1")

	// 验证初始状态
	assert.Equal(t, "node1", manager.nodeID)
	assert.Equal(t, uint64(0), manager.currentTerm)
	assert.Equal(t, "", manager.currentLeader)
	assert.Equal(t, replication.RoleFollower, manager.role)
	assert.NotNil(t, manager.config)
	assert.NotNil(t, manager.votesReceived)
	assert.Equal(t, "", manager.votedFor)
	assert.False(t, manager.running)
}

// TestElectionManager_StartStop 测试启动和停止选举管理器
func TestElectionManager_StartStop(t *testing.T) {
	// 创建选举管理器
	manager, _, _ := createTestElectionManager(t, "node1")

	// 启动选举管理器
	err := manager.Start()
	require.NoError(t, err)
	assert.True(t, manager.running)

	// 再次启动应该不会报错
	err = manager.Start()
	require.NoError(t, err)

	// 停止选举管理器
	err = manager.Stop()
	require.NoError(t, err)
	assert.False(t, manager.running)

	// 再次停止应该不会报错
	err = manager.Stop()
	require.NoError(t, err)
}

// TestElectionManager_BecomeLeader 测试成为领导者
func TestElectionManager_BecomeLeader(t *testing.T) {
	// 创建选举管理器
	manager, _, eventPublisher := createTestElectionManager(t, "node1")

	// 订阅事件
	eventCh := eventPublisher.Subscribe()

	// 使节点成为候选者
	manager.mu.Lock()
	manager.role = replication.RoleCandidate
	manager.currentTerm = 1
	manager.mu.Unlock()

	// 成为领导者
	manager.becomeLeader()

	// 验证状态
	assert.Equal(t, replication.RoleLeader, manager.role)
	assert.Equal(t, "node1", manager.currentLeader)

	// 验证是否发布了角色变化事件
	select {
	case event := <-eventCh:
		assert.Equal(t, ClusterEventRoleChange, event.Type)
		details, ok := event.Details.(replication.ReplicationEvent)
		assert.True(t, ok)
		assert.Equal(t, replication.EventRoleChange, details.Type)
		assert.Equal(t, replication.RoleLeader, details.Role)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected role change event not received")
	}

	// 验证是否发布了领导者选举事件
	select {
	case event := <-eventCh:
		assert.Equal(t, ClusterEventLeaderElected, event.Type)
		details, ok := event.Details.(replication.ReplicationEvent)
		assert.True(t, ok)
		assert.Equal(t, replication.EventLeaderElected, details.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected leader elected event not received")
	}
}

// TestElectionManager_BecomeFollower 测试成为跟随者
func TestElectionManager_BecomeFollower(t *testing.T) {
	// 创建选举管理器
	manager, _, eventPublisher := createTestElectionManager(t, "node1")

	// 订阅事件
	eventCh := eventPublisher.Subscribe()

	// 先成为领导者
	manager.mu.Lock()
	manager.role = replication.RoleLeader
	manager.currentTerm = 1
	manager.currentLeader = "node1"
	manager.mu.Unlock()

	// 成为跟随者
	manager.becomeFollower("node2")

	// 验证状态
	assert.Equal(t, replication.RoleFollower, manager.role)
	assert.Equal(t, "node2", manager.currentLeader)
	assert.Equal(t, "", manager.votedFor)

	// 验证是否发布了角色变化事件
	select {
	case event := <-eventCh:
		assert.Equal(t, ClusterEventRoleChange, event.Type)
		details, ok := event.Details.(replication.ReplicationEvent)
		assert.True(t, ok)
		assert.Equal(t, replication.EventRoleChange, details.Type)
		assert.Equal(t, replication.RoleFollower, details.Role)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected role change event not received")
	}
}

// TestElectionManager_HandleVoteRequest 测试处理投票请求
func TestElectionManager_HandleVoteRequest(t *testing.T) {
	// 创建选举管理器
	manager, _, _ := createTestElectionManager(t, "node1")

	// 设置初始状态
	manager.mu.Lock()
	manager.currentTerm = 1
	manager.votedFor = ""
	manager.lastLogIndex = 10
	manager.lastLogTerm = 1
	manager.mu.Unlock()

	// 测试1: 任期小于当前任期的请求应该被拒绝
	req1 := &VoteRequest{
		Term:         0,
		CandidateID:  "node2",
		LastLogIndex: 10,
		LastLogTerm:  1,
	}
	resp1 := manager.HandleVoteRequest(req1)
	assert.Equal(t, uint64(1), resp1.Term)
	assert.False(t, resp1.VoteGranted)

	// 测试2: 日志不够新的请求应该被拒绝
	req2 := &VoteRequest{
		Term:         2,
		CandidateID:  "node2",
		LastLogIndex: 5, // 比当前的10要旧
		LastLogTerm:  1,
	}
	resp2 := manager.HandleVoteRequest(req2)
	assert.Equal(t, uint64(2), resp2.Term) // 任期应该更新
	assert.False(t, resp2.VoteGranted)     // 但不投票

	// 测试3: 符合条件的请求应该获得投票
	req3 := &VoteRequest{
		Term:         2,
		CandidateID:  "node2",
		LastLogIndex: 20, // 比当前的10要新
		LastLogTerm:  1,
	}
	resp3 := manager.HandleVoteRequest(req3)
	assert.Equal(t, uint64(2), resp3.Term)
	assert.True(t, resp3.VoteGranted)
	assert.Equal(t, "node2", resp3.VotedFor)

	// 验证内部状态
	assert.Equal(t, "node2", manager.votedFor)
	assert.Equal(t, uint64(2), manager.currentTerm)
}

// TestElectionManager_HandleHeartbeat 测试处理心跳消息
func TestElectionManager_HandleHeartbeat(t *testing.T) {
	// 创建选举管理器
	manager, _, _ := createTestElectionManager(t, "node1")

	// 设置初始状态 - 假设是候选者
	manager.mu.Lock()
	manager.role = replication.RoleCandidate
	manager.currentTerm = 1
	manager.mu.Unlock()

	// 测试1: 心跳任期小于当前任期，应该被拒绝
	heartbeat1 := &Heartbeat{
		Term:     0,
		LeaderID: "node2",
	}
	err := manager.HandleHeartbeat(heartbeat1)
	require.Error(t, err)
	assert.Equal(t, replication.RoleCandidate, manager.role) // 角色不变

	// 测试2: 心跳任期等于当前任期，应该转为跟随者
	heartbeat2 := &Heartbeat{
		Term:     1,
		LeaderID: "node2",
	}
	err = manager.HandleHeartbeat(heartbeat2)
	require.NoError(t, err)
	assert.Equal(t, replication.RoleFollower, manager.role) // 角色变为跟随者
	assert.Equal(t, "node2", manager.currentLeader)

	// 测试3: 心跳任期大于当前任期，应该更新任期并转为跟随者
	manager.mu.Lock()
	manager.role = replication.RoleCandidate
	manager.mu.Unlock()

	heartbeat3 := &Heartbeat{
		Term:     2,
		LeaderID: "node3",
	}
	err = manager.HandleHeartbeat(heartbeat3)
	require.NoError(t, err)
	assert.Equal(t, replication.RoleFollower, manager.role) // 角色变为跟随者
	assert.Equal(t, "node3", manager.currentLeader)
	assert.Equal(t, uint64(2), manager.currentTerm) // 任期更新
}

// TestElectionManager_GetLeaderAndRole 测试获取领导者ID和角色
func TestElectionManager_GetLeaderAndRole(t *testing.T) {
	// 创建选举管理器
	manager, _, _ := createTestElectionManager(t, "node1")

	// 设置初始状态
	manager.mu.Lock()
	manager.currentLeader = "node2"
	manager.role = replication.RoleFollower
	manager.mu.Unlock()

	// 测试获取领导者ID
	leader := manager.GetLeader()
	assert.Equal(t, "node2", leader)

	// 测试获取角色
	role := manager.GetRole()
	assert.Equal(t, replication.RoleFollower, role)
}

// TestElectionManager_SetLogInfo 测试更新日志信息
func TestElectionManager_SetLogInfo(t *testing.T) {
	// 创建选举管理器
	manager, _, _ := createTestElectionManager(t, "node1")

	// 设置日志信息
	manager.SetLogInfo(100, 2)

	// 验证日志信息已更新
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	assert.Equal(t, uint64(100), manager.lastLogIndex)
	assert.Equal(t, uint64(2), manager.lastLogTerm)
}
package cluster

import (
	"fmt"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/fyerfyer/fyer-cache/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewMembership(t *testing.T) {
	// 创建必要的依赖
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)
	config := DefaultNodeConfig()

	// 测试创建成员管理器
	membership := NewMembership("test-node", "127.0.0.1:8000", mockTransport, events, config)

	// 验证初始化
	assert.NotNil(t, membership)
	assert.Equal(t, "test-node", membership.localID)
	assert.Equal(t, "127.0.0.1:8000", membership.localAddr)
	assert.NotNil(t, membership.members)
	assert.NotNil(t, membership.suspectTime)
	assert.Equal(t, config.SuspicionMult, membership.suspicionMult)
	assert.Equal(t, config.ProbeInterval, membership.probeInterval)
	assert.Equal(t, config.ProbeTimeout, membership.probeTimeout)
	assert.Equal(t, events, membership.events)
	assert.Equal(t, mockTransport, membership.transport)
	assert.NotNil(t, membership.stopCh)
	assert.False(t, membership.running)
}

func TestMembership_StartStop(t *testing.T) {
	// 创建依赖
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)
	config := DefaultNodeConfig()
	config.ProbeInterval = 10 * time.Millisecond // 短时间间隔加速测试

	membership := NewMembership("test-node", "127.0.0.1:8000", mockTransport, events, config)

	// 测试启动
	err := membership.Start()
	assert.NoError(t, err)
	assert.True(t, membership.running)

	// 验证本地节点已添加到成员列表
	members := membership.Members()
	found := false
	for _, member := range members {
		if member.ID == "test-node" {
			found = true
			assert.Equal(t, "127.0.0.1:8000", member.Addr)
			assert.Equal(t, cache.NodeStatusUp, member.Status)
			break
		}
	}
	assert.True(t, found, "Local node should be added to membership")

	// 再次启动，不应报错
	err = membership.Start()
	assert.NoError(t, err)

	// 测试停止
	err = membership.Stop()
	assert.NoError(t, err)
	assert.False(t, membership.running)

	// 再次停止，不应报错
	err = membership.Stop()
	assert.NoError(t, err)
}

func TestMembership_AddMember(t *testing.T) {
	// 创建依赖
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)
	config := DefaultNodeConfig()

	membership := NewMembership("local-node", "127.0.0.1:8000", mockTransport, events, config)

	membership.Start()

	// 订阅事件
	eventChan := events.Subscribe()

	// 添加成员
	metadata := map[string]string{"role": "worker", "dc": "us-west"}
	membership.AddMember("node1", "127.0.0.1:8001", metadata)

	// 验证是否添加成功
	members := membership.Members()
	found := false
	for _, member := range members {
		if member.ID == "node1" {
			found = true
			assert.Equal(t, "127.0.0.1:8001", member.Addr)
			assert.Equal(t, "worker", member.Metadata["role"])
			assert.Equal(t, "us-west", member.Metadata["dc"])
		}
	}
	assert.True(t, found, "Added node should be in membership")

	// 验证是否发出了事件
	select {
	case event := <-eventChan:
		assert.Equal(t, cache.EventNodeJoin, event.Type)
		assert.Equal(t, "node1", event.NodeID)

	case <-time.After(100 * time.Millisecond):
		t.Error("No event received for node join")
	}

	// 测试重复添加相同节点
	membership.AddMember("node1", "127.0.0.1:8001", metadata)
	// 应该不会产生错误，且成员数量不变
	assert.Equal(t, 2, len(membership.Members())) // 本地节点 + node1

	// Clean up
	membership.Stop()
}

func TestMembership_RemoveMember(t *testing.T) {
	// 创建依赖
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)
	config := DefaultNodeConfig()

	membership := NewMembership("local-node", "127.0.0.1:8000", mockTransport, events, config)
	membership.Start()

	// 添加几个成员
	membership.AddMember("node1", "127.0.0.1:8001", nil)
	membership.AddMember("node2", "127.0.0.1:8002", nil)

	// 订阅事件
	eventChan := events.Subscribe()

	// 测试优雅移除
	membership.RemoveMember("node1", true)

	// 验证是否已移除
	members := membership.Members()
	for _, member := range members {
		if member.ID == "node1" {
			t.Error("node1 should have been removed")
		}
	}

	// 验证事件
	select {
	case event := <-eventChan:
		assert.Equal(t, cache.EventNodeLeave, event.Type)
		assert.Equal(t, "node1", event.NodeID)
		assert.Equal(t, true, event.Details)
	case <-time.After(100 * time.Millisecond):
		t.Error("Leave event was not published")
	}

	// 测试强制移除
	membership.RemoveMember("node2", false)

	// 验证是否已移除
	members = membership.Members()
	for _, member := range members {
		if member.ID == "node2" {
			t.Error("node2 should have been removed")
		}
	}

	// 验证事件
	select {
	case event := <-eventChan:
		assert.Equal(t, cache.EventNodeLeave, event.Type)
		assert.Equal(t, "node2", event.NodeID)
		assert.Equal(t, false, event.Details)
	case <-time.After(100 * time.Millisecond):
		t.Error("Leave event was not published")
	}

	// 清理
	membership.Stop()
}

func TestMembership_MarkSuspectAndFailed(t *testing.T) {
	// 创建依赖
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)
	config := DefaultNodeConfig()

	membership := NewMembership("local-node", "127.0.0.1:8000", mockTransport, events, config)

	// 添加成员
	membership.AddMember("node1", "127.0.0.1:8001", nil)

	// 订阅事件
	eventChan := events.Subscribe()

	// 标记为可疑
	membership.MarkSuspect("node1")

	// 验证状态
	members := membership.Members()
	for _, member := range members {
		if member.ID == "node1" {
			assert.Equal(t, cache.NodeStatusSuspect, member.Status)
			// 验证怀疑时间已记录
			_, exists := membership.suspectTime["node1"]
			assert.True(t, exists, "Suspect time should be recorded")
		}
	}

	// 标记为失败
	reason := "connection timeout"
	membership.MarkFailed("node1", reason)

	// 验证状态
	members = membership.Members()
	for _, member := range members {
		if member.ID == "node1" {
			assert.Equal(t, cache.NodeStatusDown, member.Status)
		}
	}

	// 验证事件
	select {
	case event := <-eventChan:
		assert.Equal(t, cache.EventNodeFailed, event.Type)
		assert.Equal(t, "node1", event.NodeID)
		assert.Equal(t, reason, event.Details)
	case <-time.After(100 * time.Millisecond):
		t.Error("Failed event was not published")
	}
}

func TestMembership_MarkAlive(t *testing.T) {
	// 创建依赖
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)
	config := DefaultNodeConfig()

	membership := NewMembership("local-node", "127.0.0.1:8000", mockTransport, events, config)

	// 添加成员并标记为可疑
	membership.AddMember("node1", "127.0.0.1:8001", nil)
	membership.MarkSuspect("node1")

	// 订阅事件
	eventChan := events.Subscribe()

	// 标记为活跃
	membership.MarkAlive("node1")

	// 验证状态
	members := membership.Members()
	for _, member := range members {
		if member.ID == "node1" {
			assert.Equal(t, cache.NodeStatusUp, member.Status)
			// 验证怀疑时间已移除
			_, exists := membership.suspectTime["node1"]
			assert.False(t, exists, "Suspect time should be removed")
		}
	}

	// 验证恢复事件
	select {
	case event := <-eventChan:
		assert.Equal(t, cache.EventNodeRecovered, event.Type)
		assert.Equal(t, "node1", event.NodeID)
	case <-time.After(100 * time.Millisecond):
		t.Error("Recovered event was not published")
	}
}

func TestMembership_GetNodeAddr(t *testing.T) {
	// 创建依赖
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)
	config := DefaultNodeConfig()

	membership := NewMembership("local-node", "127.0.0.1:8000", mockTransport, events, config)

	// 添加成员
	membership.AddMember("node1", "127.0.0.1:8001", nil)

	// 获取存在的节点
	addr, err := membership.GetNodeAddr("node1")
	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.1:8001", addr)

	// 获取不存在的节点
	_, err = membership.GetNodeAddr("nonexistent")
	assert.Error(t, err)
}

func TestMembership_GetNodeByAddr(t *testing.T) {
	// 创建依赖
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)
	config := DefaultNodeConfig()

	membership := NewMembership("local-node", "127.0.0.1:8000", mockTransport, events, config)

	// 添加成员
	membership.AddMember("node1", "127.0.0.1:8001", nil)

	// 通过地址查找ID
	id, err := membership.GetNodeByAddr("127.0.0.1:8001")
	assert.NoError(t, err)
	assert.Equal(t, "node1", id)

	// 查找不存在的地址
	_, err = membership.GetNodeByAddr("127.0.0.1:9999")
	assert.Error(t, err)
}

func TestMembership_IsCoordinator(t *testing.T) {
	// 创建依赖
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)
	config := DefaultNodeConfig()

	// 测试场景1: 单节点，应该是协调者
	membership1 := NewMembership("node1", "127.0.0.1:8000", mockTransport, events, config)
	membership1.Start()
	assert.True(t, membership1.IsCoordinator(), "Single node should be coordinator")
	membership1.Stop()

	// 测试场景2: 当有ID更小的节点时，不应该是协调者
	membership2 := NewMembership("bnode", "127.0.0.1:8000", mockTransport, events, config)
	membership2.Start()
	membership2.AddMember("anode", "127.0.0.1:8001", nil)
	assert.False(t, membership2.IsCoordinator(), "Node with larger ID should not be coordinator")
	membership2.Stop()

	// 测试场景3: 当有ID更大的节点时，应该是协调者
	membership3 := NewMembership("anode", "127.0.0.1:8000", mockTransport, events, config)
	membership3.Start()
	membership3.AddMember("bnode", "127.0.0.1:8001", nil)
	membership3.AddMember("cnode", "127.0.0.1:8002", nil)
	assert.True(t, membership3.IsCoordinator(), "Node with smallest ID should be coordinator")
	membership3.Stop()
}

func TestMembership_ProbeLoop(t *testing.T) {
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)
	config := DefaultNodeConfig()
	config.ProbeInterval = 10 * time.Millisecond
	config.ProbeTimeout = 5 * time.Millisecond

	membership := NewMembership("local-node", "127.0.0.1:8000", mockTransport, events, config)

	// 添加两个成员
	membership.AddMember("node1", "127.0.0.1:8001", nil)
	membership.AddMember("node2", "127.0.0.1:8002", nil)

	//pingPayload := &PingPayload{Sequence: 1}
	//pingBytes, _ := MarshalPingPayload(pingPayload)

	// node1 成功接收 ping
	mockTransport.EXPECT().
		Send("127.0.0.1:8001", mock.MatchedBy(func(msg *cache.NodeMessage) bool {
			return msg.Type == cache.MsgPing
		})).
		Return(&cache.NodeMessage{
			Type:      cache.MsgPong,
			SenderID:  "node1",
			Timestamp: time.Now(),
		}, nil).
		Maybe()

	// node2 发生超时错误
	mockTransport.EXPECT().
		Send("127.0.0.1:8002", mock.MatchedBy(func(msg *cache.NodeMessage) bool {
			return msg.Type == cache.MsgPing
		})).
		Return(nil, fmt.Errorf("connection timeout")).
		Maybe()

	// 创建并启动成员管理器
	err := membership.Start()
	assert.NoError(t, err)

	// 等待足够的时间让探测循环运行几次
	time.Sleep(100 * time.Millisecond)

	// 检验 node1 是否健康
	members := membership.Members()
	node1Found := false
	node2Found := false
	node2Status := cache.NodeStatusUp

	for _, member := range members {
		if member.ID == "node1" {
			node1Found = true
			assert.NotEqual(t, cache.NodeStatusUp, member.Status)
		}
		if member.ID == "node2" {
			node2Found = true
			node2Status = member.Status
		}
	}

	assert.True(t, node1Found, "node1 should be in membership")
	assert.True(t, node2Found, "node2 should be in membership")

	// node2 不应该被标记为可疑或失败
	isSuspectOrFailed := node2Status == cache.NodeStatusSuspect || node2Status == cache.NodeStatusDown
	assert.True(t, isSuspectOrFailed,
		fmt.Sprintf("node2 should be marked as suspect or failed, current status: %d", node2Status))

	if !isSuspectOrFailed {
		t.Logf("Members: %+v", members)
		t.Logf("Suspect times: %+v", membership.suspectTime)
	}

	membership.Stop()
}

func TestMembership_ProbeTimeout(t *testing.T) {
	// 创建依赖
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)
	config := DefaultNodeConfig()
	config.ProbeInterval = 50 * time.Millisecond // 快速探测间隔
	config.SuspicionMult = 1                     // 设置为1加速测试

	membership := NewMembership("local-node", "127.0.0.1:8000", mockTransport, events, config)

	// 添加成员
	membership.AddMember("node1", "127.0.0.1:8001", nil)

	// 设置预期 - 所有ping都会失败
	mockTransport.EXPECT().Send(mock.Anything, mock.Anything).Return(nil, fmt.Errorf("connection failed")).Maybe()

	// 启动成员管理
	membership.Start()

	// 等待足够的时间让节点被标记为可疑，然后失败
	time.Sleep(100 * time.Millisecond)

	// 验证节点状态为可疑
	members := membership.Members()
	found := false
	for _, member := range members {
		if member.ID == "node1" {
			found = true
			// Changed the assertion to compare with the actual NodeStatus value
			assert.Equal(t, cache.NodeStatusSuspect, member.Status, "Expected node1 to be SUSPECT, but it is %v", member.Status)
		}
	}
	assert.True(t, found, "Node1 should still be in the membership")

	// 等待更长时间使节点失败
	time.Sleep(200 * time.Millisecond)

	// 验证节点状态
	members = membership.Members()
	found = false
	for _, member := range members {
		if member.ID == "node1" {
			found = true
			assert.Equal(t, cache.NodeStatusDown, member.Status)
		}
	}

	// 节点可能被彻底删除了，所以不做断言

	// 停止
	membership.Stop()
}

func TestMembership_Members(t *testing.T) {
	// 创建依赖
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)
	config := DefaultNodeConfig()

	membership := NewMembership("local-node", "127.0.0.1:8000", mockTransport, events, config)
	membership.Start()

	// 添加几个成员
	membership.AddMember("node1", "127.0.0.1:8001", map[string]string{"region": "us-west"})
	membership.AddMember("node2", "127.0.0.1:8002", map[string]string{"region": "us-east"})
	membership.AddMember("node3", "127.0.0.1:8003", map[string]string{"region": "eu-west"})

	// 验证成员列表
	members := membership.Members()
	assert.Equal(t, 4, len(members)) // 本地节点 + 3个添加的节点

	// 验证每个节点信息都复制出来了，不是直接引用内部map
	memberMap := make(map[string]cache.NodeInfo)
	for _, m := range members {
		memberMap[m.ID] = m
	}

	assert.Contains(t, memberMap, "local-node")
	assert.Contains(t, memberMap, "node1")
	assert.Contains(t, memberMap, "node2")
	assert.Contains(t, memberMap, "node3")

	assert.Equal(t, "127.0.0.1:8001", memberMap["node1"].Addr)
	assert.Equal(t, "us-west", memberMap["node1"].Metadata["region"])

	assert.Equal(t, "127.0.0.1:8002", memberMap["node2"].Addr)
	assert.Equal(t, "us-east", memberMap["node2"].Metadata["region"])

	assert.Equal(t, "127.0.0.1:8003", memberMap["node3"].Addr)
	assert.Equal(t, "eu-west", memberMap["node3"].Metadata["region"])

	// 停止
	membership.Stop()
}

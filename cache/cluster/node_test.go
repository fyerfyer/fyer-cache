package cluster

import (
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/fyerfyer/fyer-cache/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewNode(t *testing.T) {
	// 测试所有参数有效的情况
	t.Run("Valid Parameters", func(t *testing.T) {
		mockCache := mocks.NewCache(t)
		node, err := NewNode("node1", mockCache, WithBindAddr(":8001"))

		assert.NoError(t, err)
		assert.NotNil(t, node)
		assert.Equal(t, "node1", node.id)
		assert.Equal(t, mockCache, node.cache)
		assert.NotNil(t, node.transport)
		assert.NotNil(t, node.membership)
		assert.NotNil(t, node.gossiper)
		assert.NotNil(t, node.syncManager)
		assert.NotNil(t, node.eventPublisher)
		assert.Equal(t, ":8001", node.config.BindAddr)
		assert.False(t, node.running)
	})

	// 测试节点ID为空的情况
	t.Run("Empty NodeID", func(t *testing.T) {
		mockCache := mocks.NewCache(t)
		node, err := NewNode("", mockCache)

		assert.Error(t, err)
		assert.Nil(t, node)
	})

	// 测试缓存为空的情况
	t.Run("Nil Cache", func(t *testing.T) {
		node, err := NewNode("node1", nil)

		assert.Error(t, err)
		assert.Nil(t, node)
	})

	// 测试应用多个配置选项
	t.Run("Multiple Options", func(t *testing.T) {
		mockCache := mocks.NewCache(t)
		node, err := NewNode("node1", mockCache,
			WithBindAddr(":8002"),
			WithGossipInterval(2*time.Second),
			WithProbeInterval(3*time.Second),
			WithSyncInterval(30*time.Second),
			WithEventBufferSize(200),
		)

		assert.NoError(t, err)
		assert.NotNil(t, node)
		assert.Equal(t, ":8002", node.config.BindAddr)
		assert.Equal(t, 2*time.Second, node.config.GossipInterval)
		assert.Equal(t, 3*time.Second, node.config.ProbeInterval)
		assert.Equal(t, 30*time.Second, node.config.SyncInterval)
		assert.Equal(t, 200, node.config.EventBufferSize)
	})

	// 测试设置ID选项
	t.Run("WithNodeID Option", func(t *testing.T) {
		mockCache := mocks.NewCache(t)
		node, err := NewNode("default-id", mockCache, WithNodeID("custom-id"))

		assert.NoError(t, err)
		assert.NotNil(t, node)
		assert.Equal(t, "custom-id", node.id)
	})
}

func TestNode_StartStop(t *testing.T) {
	mockCache := mocks.NewCache(t)
	mockTransport := mocks.NewNodeTransporter(t)

	// 注入mock传输层的Node实例
	node, err := createTestNodeWithMockTransport(t, "test-node", mockCache, mockTransport)
	assert.NoError(t, err)

	// 设置传输层Start和Stop期望
	mockTransport.EXPECT().Start().Return(nil)
	mockTransport.EXPECT().Stop().Return(nil)

	// 启动节点
	err = node.Start()
	assert.NoError(t, err)
	assert.True(t, node.running)

	// 再次启动，不应该有错误
	err = node.Start()
	assert.NoError(t, err)

	// 停止节点
	err = node.Stop()
	assert.NoError(t, err)
	assert.False(t, node.running)

	// 再次停止，不应该有错误
	err = node.Stop()
	assert.NoError(t, err)
}

func TestNode_Join(t *testing.T) {
	mockCache := mocks.NewCache(t)
	mockTransport := mocks.NewNodeTransporter(t)

	// 创建测试节点
	node, err := createTestNodeWithMockTransport(t, "test-node", mockCache, mockTransport)
	assert.NoError(t, err)

	// 启动节点
	mockTransport.EXPECT().Start().Return(nil)
	err = node.Start()
	assert.NoError(t, err)

	// 测试尝试加入空地址
	err = node.Join("")
	assert.Error(t, err)

	// 测试尝试加入自身
	err = node.Join(node.config.BindAddr)
	assert.NoError(t, err) // 加入自身应该成功，表示创建新集群

	// 测试加入有效种子节点
	seedAddr := "127.0.0.1:9000"

	// 准备加入响应
	joinPayload := &JoinPayload{
		NodeID:   "seed-node",
		Address:  seedAddr,
		Metadata: map[string]string{"role": "seed"},
	}
	respBytes, _ := MarshalJoinPayload(joinPayload)

	mockTransport.EXPECT().Send(seedAddr, mock.Anything).Run(func(nodeAddr string, msg *cache.NodeMessage) {
		// 验证发送的加入消息
		assert.Equal(t, cache.MsgJoin, msg.Type)
		assert.Equal(t, "test-node", msg.SenderID)

		// 解析消息载荷
		payload, _ := UnmarshalJoinPayload(msg.Payload)
		assert.Equal(t, "test-node", payload.NodeID)
	}).Return(&cache.NodeMessage{
		Type:      cache.MsgJoin,
		SenderID:  "seed-node",
		Timestamp: time.Now(),
		Payload:   respBytes,
	}, nil)

	err = node.Join(seedAddr)
	assert.NoError(t, err)

	// 验证种子节点是否已加入成员列表
	members := node.Members()
	found := false
	for _, member := range members {
		if member.ID == "seed-node" {
			found = true
			assert.Equal(t, seedAddr, member.Addr)
			break
		}
	}
	assert.True(t, found, "Seed node should be added to membership")

	// 清理
	mockTransport.EXPECT().Stop().Return(nil)
	node.Stop()
}

func TestNode_Leave(t *testing.T) {
	mockCache := mocks.NewCache(t)
	mockTransport := mocks.NewNodeTransporter(t)

	// 创建测试节点
	node, err := createTestNodeWithMockTransport(t, "test-node", mockCache, mockTransport)
	assert.NoError(t, err)

	// 启动节点
	mockTransport.EXPECT().Start().Return(nil)
	err = node.Start()
	assert.NoError(t, err)

	// 添加一些成员
	node.membership.AddMember("node1", "127.0.0.1:9001", nil)
	node.membership.AddMember("node2", "127.0.0.1:9002", nil)

	// 期望发送离开消息给所有成员
	mockTransport.EXPECT().Send("127.0.0.1:9001", mock.Anything).Return(&cache.NodeMessage{Type: MsgAck}, nil).Maybe()
	mockTransport.EXPECT().Send("127.0.0.1:9002", mock.Anything).Return(&cache.NodeMessage{Type: MsgAck}, nil).Maybe()

	// 执行离开操作
	err = node.Leave()
	assert.NoError(t, err)

	// 检查本地节点是否被标记为已离开
	members := node.Members()
	found := false
	for _, member := range members {
		if member.ID == "test-node" {
			found = true
			break
		}
	}
	assert.False(t, found, "Local node should be removed from membership")

	// 清理
	mockTransport.EXPECT().Stop().Return(nil)
	node.Stop()
}

func TestNode_Members(t *testing.T) {
	mockCache := mocks.NewCache(t)
	mockTransport := mocks.NewNodeTransporter(t)

	// 创建测试节点
	node, err := createTestNodeWithMockTransport(t, "test-node", mockCache, mockTransport)
	assert.NoError(t, err)

	// 启动节点
	mockTransport.EXPECT().Start().Return(nil)
	err = node.Start()
	assert.NoError(t, err)

	// 添加一些成员
	node.membership.AddMember("node1", "127.0.0.1:9001", map[string]string{"role": "worker"})
	node.membership.AddMember("node2", "127.0.0.1:9002", map[string]string{"role": "worker"})

	// 获取成员列表
	members := node.Members()
	assert.Equal(t, 3, len(members)) // 本地节点 + 两个添加的节点

	// 验证成员信息
	memberMap := make(map[string]cache.NodeInfo)
	for _, member := range members {
		memberMap[member.ID] = member
	}

	// 验证本地节点
	assert.Contains(t, memberMap, "test-node")

	// 验证添加的节点
	assert.Contains(t, memberMap, "node1")
	assert.Equal(t, "127.0.0.1:9001", memberMap["node1"].Addr)
	assert.Equal(t, "worker", memberMap["node1"].Metadata["role"])

	assert.Contains(t, memberMap, "node2")
	assert.Equal(t, "127.0.0.1:9002", memberMap["node2"].Addr)
	assert.Equal(t, "worker", memberMap["node2"].Metadata["role"])

	// 清理
	mockTransport.EXPECT().Stop().Return(nil)
	node.Stop()
}

func TestNode_Events(t *testing.T) {
	mockCache := mocks.NewCache(t)
	mockTransport := mocks.NewNodeTransporter(t)

	// 创建测试节点
	node, err := createTestNodeWithMockTransport(t, "test-node", mockCache, mockTransport)
	assert.NoError(t, err)

	// 启动节点
	mockTransport.EXPECT().Start().Return(nil)
	err = node.Start()
	assert.NoError(t, err)

	// 获取事件通道
	eventCh := node.Events()
	assert.NotNil(t, eventCh)

	// 模拟一个事件
	testEvent := cache.ClusterEvent{
		Type:    cache.EventNodeJoin,
		NodeID:  "new-node",
		Time:    time.Now(),
		Details: "127.0.0.1:9003",
	}

	// 发布事件
	node.eventPublisher.Publish(testEvent)

	// 验证事件是否可以接收
	select {
	case event := <-eventCh:
		assert.Equal(t, testEvent.Type, event.Type)
		assert.Equal(t, testEvent.NodeID, event.NodeID)
		assert.Equal(t, testEvent.Details, event.Details)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for event")
	}

	// 清理
	mockTransport.EXPECT().Stop().Return(nil)
	node.Stop()
}

func TestNode_IsCoordinator(t *testing.T) {
	mockCache := mocks.NewCache(t)
	mockTransport := mocks.NewNodeTransporter(t)

	// 创建测试节点
	node, err := createTestNodeWithMockTransport(t, "test-node", mockCache, mockTransport)
	assert.NoError(t, err)

	// 启动节点
	mockTransport.EXPECT().Start().Return(nil)
	err = node.Start()
	assert.NoError(t, err)

	// 单节点应该是协调者
	assert.True(t, node.IsCoordinator())

	// 添加一个ID更小的节点，使当前节点不再是协调者
	node.membership.AddMember("aaa-node", "127.0.0.1:9001", nil)
	assert.False(t, node.IsCoordinator())

	// 添加一个ID更大的节点，不应影响当前节点的协调者状态
	node.membership.AddMember("zzz-node", "127.0.0.1:9002", nil)
	assert.False(t, node.IsCoordinator())

	// 清理
	mockTransport.EXPECT().Stop().Return(nil)
	node.Stop()
}

func TestNode_HandleMessage(t *testing.T) {
	mockCache := mocks.NewCache(t)
	mockTransport := mocks.NewNodeTransporter(t)

	// 创建测试节点
	node, err := createTestNodeWithMockTransport(t, "test-node", mockCache, mockTransport)
	assert.NoError(t, err)

	// 启动节点
	mockTransport.EXPECT().Start().Return(nil)
	err = node.Start()
	assert.NoError(t, err)

	// 测试处理加入消息
	joinPayload := &JoinPayload{
		NodeID:   "joining-node",
		Address:  "127.0.0.1:9003",
		Metadata: map[string]string{"version": "1.0"},
	}
	joinBytes, _ := MarshalJoinPayload(joinPayload)

	joinMsg := &cache.NodeMessage{
		Type:      cache.MsgJoin,
		SenderID:  "joining-node",
		Timestamp: time.Now(),
		Payload:   joinBytes,
	}

	// 处理消息
	response, err := node.handleMessage(joinMsg)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, cache.MsgJoin, response.Type)

	// 验证成员是否已添加
	members := node.Members()
	found := false
	for _, member := range members {
		if member.ID == "joining-node" {
			found = true
			assert.Equal(t, "127.0.0.1:9003", member.Addr)
			assert.Equal(t, "1.0", member.Metadata["version"])
			break
		}
	}
	assert.True(t, found, "Joining node should be added to membership")

	// 清理
	mockTransport.EXPECT().Stop().Return(nil)
	node.Stop()
}

// 创建一个测试节点，注入mock传输层
func createTestNodeWithMockTransport(t *testing.T, nodeID string, mockCache *mocks.Cache, mockTransport *mocks.NodeTransporter) (*Node, error) {
	// 使用配置选项创建节点
	node, err := NewNode(nodeID, mockCache, WithBindAddr(":8001"))
	if err != nil {
		return nil, err
	}

	// 替换传输层为mock
	node.transport = mockTransport

	// 注入mock传输层到其他组件
	node.membership = NewMembership(nodeID, ":8001", mockTransport, node.eventPublisher, node.config)
	node.gossiper = NewGossiper(nodeID, node.membership, mockTransport, node.config)

	syncOptions := []SyncManagerOption{
		WithSyncerInterval(node.config.SyncInterval),
		WithSyncerConcurrency(node.config.SyncConcurrency),
		WithMaxSyncItems(1000),
	}
	node.syncManager = NewSyncManager(nodeID, mockCache, mockTransport, node.eventPublisher, syncOptions...)

	mockTransport.EXPECT().Address().Return(":8001").Maybe()

	return node, nil
}

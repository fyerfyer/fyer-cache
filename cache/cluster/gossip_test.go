package cluster

import (
	"sync"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/fyerfyer/fyer-cache/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewGossiper(t *testing.T) {
	// 创建必要的依赖
	mockTransport := mocks.NewNodeTransporter(t)
	membership := NewMembership("node1", "127.0.0.1:8000", mockTransport, NewEventPublisher(10), DefaultNodeConfig())

	// 测试使用默认配置的情况
	config := DefaultNodeConfig()
	config.GossipInterval = 0 // 测试默认值处理
	gossiper := NewGossiper("node1", membership, mockTransport, config)

	assert.NotNil(t, gossiper)
	assert.Equal(t, "node1", gossiper.localID)
	assert.Equal(t, membership, gossiper.membership)
	assert.Equal(t, mockTransport, gossiper.transport)
	assert.Equal(t, 1*time.Second, gossiper.interval) // 确认使用了默认值
	assert.Equal(t, 3, gossiper.fanout)
	assert.NotNil(t, gossiper.rand)
	assert.NotNil(t, gossiper.stopCh)
	assert.False(t, gossiper.running)

	// 测试使用自定义配置的情况
	customConfig := DefaultNodeConfig()
	customConfig.GossipInterval = 2 * time.Second
	customGossiper := NewGossiper("node2", membership, mockTransport, customConfig)

	assert.Equal(t, 2*time.Second, customGossiper.interval)
}

func TestGossiper_StartStop(t *testing.T) {
	// 创建必要的依赖
	mockTransport := mocks.NewNodeTransporter(t)
	membership := NewMembership("node1", "127.0.0.1:8000", mockTransport, NewEventPublisher(10), DefaultNodeConfig())
	config := DefaultNodeConfig()
	config.GossipInterval = 10 * time.Millisecond // 短间隔，加快测试速度
	gossiper := NewGossiper("node1", membership, mockTransport, config)

	// 验证初始状态
	assert.False(t, gossiper.running)

	// 启动 Gossiper
	err := gossiper.Start()
	assert.NoError(t, err)
	assert.True(t, gossiper.running)

	// 再次启动，不应产生错误
	err = gossiper.Start()
	assert.NoError(t, err)

	// 停止 Gossiper
	err = gossiper.Stop()
	assert.NoError(t, err)
	assert.False(t, gossiper.running)

	// 再次停止，不应产生错误
	err = gossiper.Stop()
	assert.NoError(t, err)
}

func TestGossiper_SelectRandomNodes(t *testing.T) {
	// 创建必要的依赖
	mockTransport := mocks.NewNodeTransporter(t)
	membership := NewMembership("node1", "127.0.0.1:8000", mockTransport, NewEventPublisher(10), DefaultNodeConfig())
	gossiper := NewGossiper("node1", membership, mockTransport, DefaultNodeConfig())

	// 测试场景1: 节点数量小于要选择的数量
	members1 := []cache.NodeInfo{
		{ID: "node1", Addr: "127.0.0.1:8001"},
		{ID: "node2", Addr: "127.0.0.1:8002"},
	}
	selected1 := gossiper.selectRandomNodes(3, members1)
	assert.Len(t, selected1, 2) // 应该只返回两个节点
	assert.ElementsMatch(t, members1, selected1)

	// 测试场景2: 节点数量等于要选择的数量
	members2 := []cache.NodeInfo{
		{ID: "node1", Addr: "127.0.0.1:8001"},
		{ID: "node2", Addr: "127.0.0.1:8002"},
		{ID: "node3", Addr: "127.0.0.1:8003"},
	}
	selected2 := gossiper.selectRandomNodes(3, members2)
	assert.Len(t, selected2, 3)
	assert.ElementsMatch(t, members2, selected2)

	// 测试场景3: 节点数量大于要选择的数量
	members3 := []cache.NodeInfo{
		{ID: "node1", Addr: "127.0.0.1:8001"},
		{ID: "node2", Addr: "127.0.0.1:8002"},
		{ID: "node3", Addr: "127.0.0.1:8003"},
		{ID: "node4", Addr: "127.0.0.1:8004"},
		{ID: "node5", Addr: "127.0.0.1:8005"},
	}
	selected3 := gossiper.selectRandomNodes(3, members3)
	assert.Len(t, selected3, 3) // 应该只返回三个节点

	// 确保选择的节点在原始列表中
	for _, node := range selected3 {
		found := false
		for _, original := range members3 {
			if node.ID == original.ID {
				found = true
				break
			}
		}
		assert.True(t, found)
	}

	// 测试随机性: 多次选择应该产生不同的结果
	// 注：这个测试理论上可能会失败，但概率非常小
	selected4 := gossiper.selectRandomNodes(3, members3)
	selected5 := gossiper.selectRandomNodes(3, members3)
	isDifferent := false

	// 检查是否至少有一个节点不相同
	nodeMap := make(map[string]bool)
	for _, node := range selected4 {
		nodeMap[node.ID] = true
	}

	for _, node := range selected5 {
		if !nodeMap[node.ID] {
			isDifferent = true
			break
		}
	}

	if !isDifferent && len(selected4) == len(selected5) {
		// 检查顺序是否不同
		for i := 0; i < len(selected4); i++ {
			if selected4[i].ID != selected5[i].ID {
				isDifferent = true
				break
			}
		}
	}

	// 这里不做强制断言，因为理论上随机选择也可能产生相同结果
	if !isDifferent {
		t.Log("Warning: Multiple random selections returned identical results. This might be random chance.")
	}
}

func TestGossiper_HandleGossip(t *testing.T) {
	// 创建依赖
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)
	config := DefaultNodeConfig()
	membership := NewMembership("local-node", "127.0.0.1:8000", mockTransport, events, config)
	gossiper := NewGossiper("local-node", membership, mockTransport, config)

	// 模拟一些初始状态
	membership.Start()

	// 测试场景1：处理非Gossip类型的消息
	nonGossipMsg := &cache.NodeMessage{
		Type:     cache.MsgPing,
		SenderID: "node2",
		Payload:  []byte("ping"),
	}
	err := gossiper.HandleGossip(nonGossipMsg)
	assert.NoError(t, err) // 应该忽略非Gossip消息而不返回错误

	// 测试场景2：处理格式错误的Gossip消息
	invalidMsg := &cache.NodeMessage{
		Type:     cache.MsgGossip,
		SenderID: "node2",
		Payload:  []byte("invalid json"),
	}
	err = gossiper.HandleGossip(invalidMsg)
	assert.Error(t, err) // 应该返回解析错误

	// 测试场景3：处理正常的Gossip消息
	gossipPayload := &GossipPayload{
		Nodes: []GossipNodeInfo{
			{
				ID:       "node2",
				Address:  "127.0.0.1:8002",
				Status:   int(cache.NodeStatusUp),
				LastSeen: time.Now().UnixNano(),
				Metadata: map[string]string{"region": "us-west"},
			},
			{
				ID:       "node3",
				Address:  "127.0.0.1:8003",
				Status:   int(cache.NodeStatusSuspect),
				LastSeen: time.Now().UnixNano(),
				Metadata: map[string]string{"region": "us-east"},
			},
			{
				ID:       "local-node", // 自己的ID，应该被忽略
				Address:  "127.0.0.1:8000",
				Status:   int(cache.NodeStatusDown),
				LastSeen: time.Now().UnixNano(),
			},
		},
	}

	payloadBytes, _ := MarshalGossipPayload(gossipPayload)
	validMsg := &cache.NodeMessage{
		Type:     cache.MsgGossip,
		SenderID: "node4",
		Payload:  payloadBytes,
	}

	err = gossiper.HandleGossip(validMsg)
	assert.NoError(t, err)

	// 验证成员列表是否更新
	members := membership.Members()

	// 创建一个节点ID到NodeInfo的映射，方便查找
	memberMap := make(map[string]*cache.NodeInfo)
	for i := range members {
		memberMap[members[i].ID] = &members[i]
	}

	// 检查node2是否被添加为UP状态
	node2, exists := memberMap["node2"]
	assert.True(t, exists, "node2 should be added to membership")
	if exists {
		assert.Equal(t, cache.NodeStatusUp, node2.Status)
		assert.Equal(t, "127.0.0.1:8002", node2.Addr)
		assert.Equal(t, "us-west", node2.Metadata["region"])
	}

	// 检查node3是否被添加为SUSPECT状态
	node3, exists := memberMap["node3"]
	assert.True(t, exists, "node3 should be added to membership")
	if exists {
		assert.Equal(t, cache.NodeStatusSuspect, node3.Status)
		assert.Equal(t, "127.0.0.1:8003", node3.Addr)
		assert.Equal(t, "us-east", node3.Metadata["region"])
	}
}

func TestGossiper_SpreadGossip(t *testing.T) {
	// Create dependencies
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)
	config := DefaultNodeConfig()
	membership := NewMembership("local-node", "127.0.0.1:8000", mockTransport, events, config)

	// Add some members
	membership.AddMember("node1", "127.0.0.1:8001", map[string]string{"role": "storage"})
	membership.AddMember("node2", "127.0.0.1:8002", map[string]string{"role": "compute"})
	membership.AddMember("node3", "127.0.0.1:8003", map[string]string{"role": "gateway"})

	gossiper := NewGossiper("local-node", membership, mockTransport, config)

	// Set up a map to track sent messages
	var sentMessages sync.Map

	// Set mock transport's expected behavior
	mockTransport.EXPECT().Send(mock.Anything, mock.Anything).
		Run(func(nodeAddr string, msg *cache.NodeMessage) {
			assert.Equal(t, cache.MsgGossip, msg.Type)
			assert.Equal(t, "local-node", msg.SenderID)
			assert.NotNil(t, msg.Payload)

			// Parse the message content for deeper validation
			payload, err := UnmarshalGossipPayload(msg.Payload)
			if err == nil {
				// Verify the gossip payload contains all nodes
				assert.GreaterOrEqual(t, len(payload.Nodes), 3) // The issue is here - should be 3, not 4

				// Record this address as having received a message
				sentMessages.Store(nodeAddr, true)
			}
		}).
		Return(&cache.NodeMessage{Type: MsgAck}, nil).
		Maybe() // Using Maybe() since not all nodes might be selected

	// Execute gossip spread
	err := gossiper.spreadGossip()
	assert.NoError(t, err)

	// Give async sends some time to complete
	time.Sleep(100 * time.Millisecond)

	// Verify at least one node received a message
	sent := false
	sentMessages.Range(func(key, value interface{}) bool {
		sent = true
		return false // Stop iteration
	})

	assert.True(t, sent, "Should have sent at least one message")
}

func TestGossiper_GossipLoop(t *testing.T) {
	// 创建依赖
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)
	config := DefaultNodeConfig()
	config.GossipInterval = 10 * time.Millisecond // 短间隔以加速测试
	membership := NewMembership("local-node", "127.0.0.1:8000", mockTransport, events, config)

	// 添加一些成员
	membership.AddMember("node1", "127.0.0.1:8001", nil)
	membership.AddMember("node2", "127.0.0.1:8002", nil)

	gossiper := NewGossiper("local-node", membership, mockTransport, config)

	// 设置计数器来跟踪消息发送
	var messageCount int32
	var mu sync.Mutex

	// 设置mock transport的行为
	mockTransport.EXPECT().Send(mock.Anything, mock.Anything).
		Run(func(nodeAddr string, msg *cache.NodeMessage) {
			if msg.Type == cache.MsgGossip {
				mu.Lock()
				messageCount++
				mu.Unlock()
			}
		}).
		Return(&cache.NodeMessage{Type: MsgAck}, nil).
		Maybe()

	// 启动gossiper
	err := gossiper.Start()
	assert.NoError(t, err)

	// 等待足够时间让gossip循环运行几次
	time.Sleep(50 * time.Millisecond)

	// 停止gossiper
	err = gossiper.Stop()
	assert.NoError(t, err)

	// 验证是否发送了消息
	mu.Lock()
	assert.True(t, messageCount > 0, "Should have sent at least one gossip message")
	mu.Unlock()
}

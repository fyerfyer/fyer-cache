package cluster

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
)

// Node 集群节点实现
type Node struct {
	id             string
	cache          cache.Cache
	transport      NodeTransporter
	membership     *Membership
	gossiper       *Gossiper
	syncManager    *SyncManager
	eventPublisher *EventPublisher
	config         *NodeConfig
	running        bool
	mu             sync.RWMutex
	stopCh         chan struct{}
}

// NewNode 创建新的集群节点
func NewNode(id string, cache cache.Cache, options ...NodeOption) (*Node, error) {
	if id == "" {
		return nil, errors.New("node ID cannot be empty")
	}

	if cache == nil {
		return nil, errors.New("cache cannot be nil")
	}

	// 使用默认配置
	config := DefaultNodeConfig()

	// 应用自定义选项
	for _, option := range options {
		option(config)
	}

	// 如果没有设置ID，则使用传入的ID
	if config.ID == "" {
		config.ID = id
	}

	// 创建事件发布器
	eventPublisher := NewEventPublisher(config.EventBufferSize)

	// 创建HTTP传输层
	transportConfig := &TransportConfig{
		BindAddr:        config.BindAddr,
		PathPrefix:      "/cluster",
		Timeout:         5 * time.Second,
		MaxIdleConns:    10,
		MaxConnsPerHost: 2,
	}
	transport := NewHTTPTransport(transportConfig)

	node := &Node{
		id:             config.ID,
		cache:          cache,
		transport:      transport,
		eventPublisher: eventPublisher,
		config:         config,
		stopCh:         make(chan struct{}),
	}

	// 设置消息处理函数
	transport.SetHandler(node.handleMessage)

	// 创建成员管理器
	node.membership = NewMembership(config.ID, config.BindAddr, transport, eventPublisher, config)

	// 创建Gossip协议实现
	node.gossiper = NewGossiper(config.ID, node.membership, transport, config)

	// 创建同步管理器
	syncOptions := []SyncManagerOption{
		WithSyncerInterval(config.SyncInterval),
		WithSyncerConcurrency(config.SyncConcurrency),
		WithMaxSyncItems(1000), // 默认一次同步1000个项
	}
	node.syncManager = NewSyncManager(config.ID, cache, transport, eventPublisher, syncOptions...)

	return node, nil
}

// Start 启动节点服务
func (n *Node) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.running {
		return nil
	}

	// 启动传输层
	if err := n.transport.Start(); err != nil {
		return fmt.Errorf("failed to start transport: %w", err)
	}

	// 启动成员管理
	if err := n.membership.Start(); err != nil {
		n.transport.Stop()
		return fmt.Errorf("failed to start membership: %w", err)
	}

	// 启动Gossip协议
	if err := n.gossiper.Start(); err != nil {
		n.membership.Stop()
		n.transport.Stop()
		return fmt.Errorf("failed to start gossiper: %w", err)
	}

	// 启动同步管理器
	if err := n.syncManager.Start(); err != nil {
		n.gossiper.Stop()
		n.membership.Stop()
		n.transport.Stop()
		return fmt.Errorf("failed to start sync manager: %w", err)
	}

	n.running = true
	return nil
}

// Stop 停止节点服务
func (n *Node) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.running {
		return nil
	}

	// 关闭所有组件，顺序相反
	n.syncManager.Stop()
	n.gossiper.Stop()
	n.membership.Stop()
	n.transport.Stop()
	n.eventPublisher.Close()

	// close(n.stopCh)
	select {
	case <-n.stopCh:
	default:
		close(n.stopCh)
	}
	n.running = false
	return nil
}

// Join 加入集群
func (n *Node) Join(seedNodeAddr string) error {
	n.mu.RLock()
	if !n.running {
		n.mu.RUnlock()
		return errors.New("node is not running")
	}
	n.mu.RUnlock()

	if seedNodeAddr == "" {
		return errors.New("seed node address cannot be empty")
	}

	if seedNodeAddr == n.config.BindAddr {
		// 尝试加入自己，这是成为第一个节点
		return nil
	}

	// 创建加入消息
	payload := &JoinPayload{
		NodeID:  n.id,
		Address: n.config.BindAddr,
		Metadata: map[string]string{
			"started": time.Now().Format(time.RFC3339),
		},
	}

	// 序列化消息
	payloadBytes, err := MarshalJoinPayload(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal join payload: %w", err)
	}

	// 创建节点消息
	msg := &cache.NodeMessage{
		Type:      cache.MsgJoin,
		SenderID:  n.id,
		Timestamp: time.Now(),
		Payload:   payloadBytes,
	}

	// 发送加入消息到种子节点
	resp, err := n.transport.Send(seedNodeAddr, msg)
	if err != nil {
		return fmt.Errorf("failed to send join message: %w", err)
	}

	if resp.Type != cache.MsgJoin {
		return fmt.Errorf("unexpected response type: %d", resp.Type)
	}

	// 处理响应
	respPayload, err := UnmarshalJoinPayload(resp.Payload)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 添加种子节点到成员列表
	n.membership.AddMember(respPayload.NodeID, seedNodeAddr, respPayload.Metadata)

	// 触发与种子节点的同步
	n.syncManager.TriggerSync(seedNodeAddr)

	return nil
}

// Leave 离开集群
func (n *Node) Leave() error {
	n.mu.RLock()
	if !n.running {
		n.mu.RUnlock()
		return errors.New("node is not running")
	}
	n.mu.RUnlock()

	// 创建离开消息
	payload := &LeavePayload{
		NodeID:   n.id,
		Graceful: true,
	}

	// 序列化消息
	payloadBytes, err := MarshalLeavePayload(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal leave payload: %w", err)
	}

	// 创建节点消息
	msg := &cache.NodeMessage{
		Type:      cache.MsgLeave,
		SenderID:  n.id,
		Timestamp: time.Now(),
		Payload:   payloadBytes,
	}

	// 获取所有当前成员
	members := n.membership.Members()

	// 向所有成员发送离开消息
	for _, member := range members {
		// 不发给自己
		if member.ID == n.id {
			continue
		}

		// 异步发送消息
		go func(addr string) {
			_, err := n.transport.Send(addr, msg)
			if err != nil {
				// 只记录错误，不返回，因为这是尽力而为的通知
				fmt.Printf("Failed to send leave message to %s: %v\n", addr, err)
			}
		}(member.Addr)
	}

	// 标记自己为离开状态
	n.membership.RemoveMember(n.id, true)

	return nil
}

// Members 获取集群成员列表
func (n *Node) Members() []cache.NodeInfo {
	return n.membership.Members()
}

// IsCoordinator 判断当前节点是否为协调节点
func (n *Node) IsCoordinator() bool {
	return n.membership.IsCoordinator()
}

// Events 返回事件通知通道
func (n *Node) Events() <-chan cache.ClusterEvent {
	return n.eventPublisher.Events()
}

// handleMessage 处理接收到的消息
func (n *Node) handleMessage(msg *cache.NodeMessage) (*cache.NodeMessage, error) {
	switch msg.Type {
	case cache.MsgJoin:
		return n.handleJoinMessage(msg)
	case cache.MsgLeave:
		return n.handleLeaveMessage(msg)
	case cache.MsgPing:
		return n.handlePingMessage(msg)
	case cache.MsgPong:
		// Pong消息一般只是响应，不需要特别处理
		return msg, nil
	case cache.MsgGossip:
		return msg, n.gossiper.HandleGossip(msg)
	case cache.MsgSyncRequest:
		return n.syncManager.HandleSyncRequest(msg)
	default:
		return msg, fmt.Errorf("unknown message type: %d", msg.Type)
	}
}

// handleJoinMessage 处理加入消息
func (n *Node) handleJoinMessage(msg *cache.NodeMessage) (*cache.NodeMessage, error) {
	// 解析消息
	payload, err := UnmarshalJoinPayload(msg.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal join payload: %w", err)
	}

	// 添加新成员
	n.membership.AddMember(payload.NodeID, payload.Address, payload.Metadata)

	// 创建响应消息
	respPayload := &JoinPayload{
		NodeID:   n.id,
		Address:  n.config.BindAddr,
		Metadata: map[string]string{"role": "seed"},
	}

	// 序列化响应
	respBytes, err := MarshalJoinPayload(respPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	// 返回响应
	return &cache.NodeMessage{
		Type:      cache.MsgJoin,
		SenderID:  n.id,
		Timestamp: time.Now(),
		Payload:   respBytes,
	}, nil
}

// handleLeaveMessage 处理离开消息
func (n *Node) handleLeaveMessage(msg *cache.NodeMessage) (*cache.NodeMessage, error) {
	// 解析消息
	payload, err := UnmarshalLeavePayload(msg.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal leave payload: %w", err)
	}

	// 从成员列表移除
	n.membership.RemoveMember(payload.NodeID, payload.Graceful)

	// 返回简单确认
	return &cache.NodeMessage{
		Type:      cache.MsgLeave,
		SenderID:  n.id,
		Timestamp: time.Now(),
		Payload:   []byte("ok"),
	}, nil
}

// handlePingMessage 处理ping消息
func (n *Node) handlePingMessage(msg *cache.NodeMessage) (*cache.NodeMessage, error) {
	// 解析ping消息
	pingPayload, err := UnmarshalPingPayload(msg.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal ping payload: %w", err)
	}

	// 标记发送者为活跃状态
	n.membership.MarkAlive(msg.SenderID)

	// 创建pong响应
	pongPayload := &PongPayload{
		Sequence:     pingPayload.Sequence,
		LoadFactor:   0.0, // 简化实现，实际可以计算负载
		MemoryUsage:  0.0, // 简化实现
		ItemCount:    0,   // 简化实现
		ResponseTime: 0,
	}

	// 序列化响应
	respBytes, err := MarshalPongPayload(pongPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pong response: %w", err)
	}

	// 返回pong响应
	return &cache.NodeMessage{
		Type:      cache.MsgPong,
		SenderID:  n.id,
		Timestamp: time.Now(),
		Payload:   respBytes,
	}, nil
}

// Membership 获取节点的成员管理器
func (n *Node) Membership() *Membership {
	return n.membership
}

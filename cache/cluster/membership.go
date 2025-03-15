package cluster

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
)

// Membership 集群成员管理
// 负责追踪集群中的所有节点及其状态
type Membership struct {
	// 本地节点ID
	localID string

	// 本地节点地址
	localAddr string

	// 所有节点信息，键为节点ID
	members map[string]*cache.NodeInfo

	// 可疑状态节点的怀疑开始时间
	suspectTime map[string]time.Time

	// 可疑状态的超时倍数
	suspicionMult int

	// 节点健康检查间隔
	probeInterval time.Duration

	// 健康检查超时时间
	probeTimeout time.Duration

	// 当前正在probe的索引
	probeIndex int

	// 事件发布器
	events *EventPublisher

	// 传输层，用于发送消息
	transport NodeTransporter

	// 互斥锁保护成员列表
	mu sync.RWMutex

	// 停止信号通道
	stopCh chan struct{}

	// 是否已运行
	running bool
}

// NodeTransporter 是节点传输器接口，用于解决循环依赖问题
type NodeTransporter interface {
	// Send 发送消息到指定节点
	Send(nodeAddr string, msg *cache.NodeMessage) (*cache.NodeMessage, error)

	// Address 返回当前节点地址
	Address() string

	// Start 启动传输层服务
	Start() error

	// Stop 停止传输层服务
	Stop() error
}

// NewMembership 创建新的成员管理器
func NewMembership(localID, localAddr string, transport NodeTransporter, events *EventPublisher, config *NodeConfig) *Membership {
	return &Membership{
		localID:       localID,
		localAddr:     localAddr,
		members:       make(map[string]*cache.NodeInfo),
		suspectTime:   make(map[string]time.Time),
		suspicionMult: config.SuspicionMult,
		probeInterval: config.ProbeInterval,
		probeTimeout:  config.ProbeTimeout,
		events:        events,
		transport:     transport,
		stopCh:        make(chan struct{}),
	}
}

// Start 启动成员管理器
func (m *Membership) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	m.running = true

	// 添加自身为成员
	m.addLocalNode()

	// 启动健康检查
	go m.probeLoop()

	return nil
}

// Stop 停止成员管理器
func (m *Membership) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	// close(m.stopCh)
	select {
	case <-m.stopCh:
	default:
		close(m.stopCh)
	}
	m.running = false
	return nil
}

// Members 获取所有成员
func (m *Membership) Members() []cache.NodeInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]cache.NodeInfo, 0, len(m.members))
	for _, member := range m.members {
		// 创建完整的深拷贝，包括元数据
		nodeCopy := *member

		// 深拷贝元数据map
		if member.Metadata != nil {
			nodeCopy.Metadata = make(map[string]string)
			for k, v := range member.Metadata {
				nodeCopy.Metadata[k] = v
			}
		}

		result = append(result, nodeCopy)
	}
	return result
}

// AddMember 添加新成员
func (m *Membership) AddMember(nodeID, addr string, metadata map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查成员是否已存在
	if existing, ok := m.members[nodeID]; ok {
		// 如果地址变化，则更新地址
		if existing.Addr != addr {
			existing.Addr = addr
		}
		// 更新元数据
		if metadata != nil {
			if existing.Metadata == nil {
				existing.Metadata = make(map[string]string)
			}
			for k, v := range metadata {
				existing.Metadata[k] = v
			}
		}
		// 更新状态
		existing.Status = cache.NodeStatusUp
		existing.LastSeen = time.Now()
		return
	}

	// 添加新成员
	m.members[nodeID] = &cache.NodeInfo{
		ID:       nodeID,
		Addr:     addr,
		Status:   cache.NodeStatusUp,
		LastSeen: time.Now(),
		Metadata: metadata,
	}

	// 发布节点加入事件
	if m.events != nil {
		m.events.Publish(NewNodeJoinEvent(nodeID, addr))
	}
}

// RemoveMember 移除成员
func (m *Membership) RemoveMember(nodeID string, graceful bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.members[nodeID]; !exists {
		return
	}

	// 发布离开事件
	if m.events != nil {
		m.events.Publish(NewNodeLeaveEvent(nodeID, graceful))
	}

	// 从怀疑列表移除
	delete(m.suspectTime, nodeID)

	// 从成员列表移除
	delete(m.members, nodeID)
}

// MarkSuspect 将节点标记为可疑状态
func (m *Membership) MarkSuspect(nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	node, ok := m.members[nodeID]
	if !ok || node.Status != cache.NodeStatusUp {
		return
	}

	// 更新节点状态
	node.Status = cache.NodeStatusSuspect
	m.suspectTime[nodeID] = time.Now()
}

// MarkAlive 将节点标记为在线状态
func (m *Membership) MarkAlive(nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	node, ok := m.members[nodeID]
	if !ok {
		return
	}

	// 如果节点之前是可疑状态，发布恢复事件
	if node.Status == cache.NodeStatusSuspect && m.events != nil {
		m.events.Publish(NewNodeRecoveredEvent(nodeID))
	}

	// 更新节点状态
	node.Status = cache.NodeStatusUp
	node.LastSeen = time.Now()
	delete(m.suspectTime, nodeID)
}

// MarkFailed 将节点标记为失败状态
func (m *Membership) MarkFailed(nodeID string, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	node, ok := m.members[nodeID]
	if !ok {
		return
	}

	// 更新节点状态
	node.Status = cache.NodeStatusDown

	// 发布故障事件
	if m.events != nil {
		m.events.Publish(NewNodeFailedEvent(nodeID, reason))
	}
}

// GetNodeAddr 获取节点地址
func (m *Membership) GetNodeAddr(nodeID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	node, ok := m.members[nodeID]
	if !ok {
		return "", fmt.Errorf("node not found: %s", nodeID)
	}

	return node.Addr, nil
}

// GetNodeByAddr 通过地址查找节点ID
func (m *Membership) GetNodeByAddr(addr string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for id, node := range m.members {
		if node.Addr == addr {
			return id, nil
		}
	}

	return "", fmt.Errorf("no node found with address: %s", addr)
}

// IsCoordinator 检查当前节点是否为协调者
// 这里采用简单策略：节点ID字母顺序最小的为协调者
func (m *Membership) IsCoordinator() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for id := range m.members {
		// 如果有比本地ID小的节点，则本节点不是协调者
		if id < m.localID && m.members[id].Status == cache.NodeStatusUp {
			return false
		}
	}

	return true
}

// addLocalNode 添加本地节点到成员列表
func (m *Membership) addLocalNode() {
	m.members[m.localID] = &cache.NodeInfo{
		ID:       m.localID,
		Addr:     m.localAddr,
		Status:   cache.NodeStatusUp,
		LastSeen: time.Now(),
		Metadata: map[string]string{"self": "true"},
	}
}

// probeLoop 定期进行节点健康检查
func (m *Membership) probeLoop() {
	ticker := time.NewTicker(m.probeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			// 检查可疑节点是否应该标记为失败
			m.checkSuspectedNodes()

			// 探测下一个节点
			m.probeNextNode()
		}
	}
}

func (m *Membership) checkSuspectedNodes() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for nodeID, suspectTime := range m.suspectTime {
		// 如果节点被怀疑的时间长于 suspicionMult * probeInterval
		if now.Sub(suspectTime) > time.Duration(m.suspicionMult)*m.probeInterval {
			node, exists := m.members[nodeID]
			if !exists || node.Status != cache.NodeStatusSuspect {
				delete(m.suspectTime, nodeID)
				continue
			}

			// 标记为失败状态
			node.Status = cache.NodeStatusDown

			// 发布故障事件
			if m.events != nil {
				m.events.Publish(NewNodeFailedEvent(nodeID, "probe timeout exceeded"))
			}
		}
	}
}

// probeNextNode 探测下一个节点是否健康
func (m *Membership) probeNextNode() {
	m.mu.Lock()

	var nodesToProbe []struct {
		id   string
		addr string
	}

	// 获取所有非本地节点
	for id, node := range m.members {
		if id != m.localID && node.Status == cache.NodeStatusUp {
			nodesToProbe = append(nodesToProbe, struct {
				id   string
				addr string
			}{id: id, addr: node.Addr})
		}
	}

	// 如果没有节点需要探测，直接返回
	if len(nodesToProbe) == 0 {
		m.mu.Unlock()
		return
	}

	// 获取下一个要探测的节点
	if m.probeIndex >= len(nodesToProbe) {
		m.probeIndex = 0
	}

	nodeToProbe := nodesToProbe[m.probeIndex]
	m.probeIndex = (m.probeIndex + 1) % len(nodesToProbe)

	m.mu.Unlock()

	// 异步发送 ping 消息
	go func(id, addr string) {
		// 创建Ping消息
		pingPayload := &PingPayload{
			Sequence: time.Now().UnixNano(),
		}

		payloadBytes, err := MarshalPingPayload(pingPayload)
		if err != nil {
			return
		}

		msg := &cache.NodeMessage{
			Type:      cache.MsgPing,
			SenderID:  m.localID,
			Timestamp: time.Now(),
			Payload:   payloadBytes,
		}

		// 创建一个带超时的上下文
		ctx, cancel := context.WithTimeout(context.Background(), m.probeTimeout)
		defer cancel()

		// 创建一个通道用于接收响应
		respChan := make(chan *cache.NodeMessage, 1)
		errChan := make(chan error, 1)

		// 异步发送请求
		go func() {
			resp, err := m.transport.Send(addr, msg)
			if err != nil {
				errChan <- err
				return
			}
			respChan <- resp
		}()

		// 等待回复或超时
		select {
		case <-respChan:
			// 收到响应，标记节点为活跃
			m.MarkAlive(id)
		case <-errChan:
			// 发生错误，标记节点为可疑
			m.MarkSuspect(id)
		case <-ctx.Done():
			// 超时，标记节点为可疑
			m.MarkSuspect(id)
		}
	}(nodeToProbe.id, nodeToProbe.addr)
}

// SetLocalMetadata 设置本地节点的元数据
func (m *Membership) SetLocalMetadata(metadata map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	localNode, exists := m.members[m.localID]
	if !exists {
		return
	}

	// 创建元数据的副本
	localNode.Metadata = make(map[string]string)
	for k, v := range metadata {
		localNode.Metadata[k] = v
	}
}

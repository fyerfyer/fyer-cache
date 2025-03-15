package cluster

import (
	"math/rand"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
)

// Gossiper 实现Gossip协议
// 负责定期交换集群成员信息
type Gossiper struct {
	// 本地节点ID
	localID string

	// 成员管理器
	membership *Membership

	// 传输层
	transport NodeTransporter

	// gossip间隔
	interval time.Duration

	// 每次选择的节点数量
	fanout int

	// 最大消息大小
	maxPacketSize int

	// 随机数生成器
	rand *rand.Rand

	// 互斥锁保护随机数生成器
	randMu sync.Mutex

	// 停止信号通道
	stopCh chan struct{}

	// 是否已启动
	running bool

	// 互斥锁保护运行状态
	mu sync.Mutex
}

// NewGossiper 创建新的Gossip协议实现
func NewGossiper(localID string, membership *Membership, transport NodeTransporter, config *NodeConfig) *Gossiper {
	// 默认值
	interval := config.GossipInterval
	if interval <= 0 {
		interval = 1 * time.Second
	}

	fanout := 3                  // 默认每次选择3个节点
	maxPacketSize := 1024 * 1024 // 默认最大包大小1MB

	return &Gossiper{
		localID:       localID,
		membership:    membership,
		transport:     transport,
		interval:      interval,
		fanout:        fanout,
		maxPacketSize: maxPacketSize,
		rand:          rand.New(rand.NewSource(time.Now().UnixNano())),
		stopCh:        make(chan struct{}),
	}
}

// Start 启动Gossip协议
func (g *Gossiper) Start() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.running {
		return nil
	}

	g.running = true
	go g.gossipLoop()

	return nil
}

// Stop 停止Gossip协议
func (g *Gossiper) Stop() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.running {
		return nil
	}

	// close(g.stopCh)
	select {
	case <-g.stopCh:
	default:
		close(g.stopCh)
	}
	g.running = false
	return nil
}

// gossipLoop 定期执行Gossip传播
func (g *Gossiper) gossipLoop() {
	ticker := time.NewTicker(g.interval)
	defer ticker.Stop()

	for {
		select {
		case <-g.stopCh:
			return
		case <-ticker.C:
			if err := g.spreadGossip(); err != nil {
				// 简单记录错误，继续执行
				// 在实际应用中可以添加日志
				// log.Printf("Error spreading gossip: %v", err)
			}
		}
	}
}

// spreadGossip 传播成员信息给随机节点
func (g *Gossiper) spreadGossip() error {
	// 获取当前成员
	members := g.membership.Members()

	var otherNodes []cache.NodeInfo
	for _, member := range members {
		if member.ID != g.localID {
			otherNodes = append(otherNodes, member)
		}
	}

	if len(otherNodes) == 0 {
		return nil
	}

	// 为所有已知节点准备Gossip负载
	// 这时我们已经有了成员的深拷贝，不会受到并发修改的影响
	gossipNodes := make([]GossipNodeInfo, 0, len(members))
	for _, member := range members {
		// 创建元数据的深拷贝，防止并发修改
		metadataCopy := make(map[string]string)
		for k, v := range member.Metadata {
			metadataCopy[k] = v
		}

		gossipNodes = append(gossipNodes, GossipNodeInfo{
			ID:       member.ID,
			Address:  member.Addr,
			Status:   int(member.Status),
			LastSeen: member.LastSeen.UnixNano(),
			Metadata: metadataCopy, // 使用复制后的元数据
		})
	}

	payload := &GossipPayload{
		Nodes: gossipNodes,
	}

	// 序列化消息
	payloadBytes, err := MarshalGossipPayload(payload)
	if err != nil {
		return err
	}

	// 创建gossip消息
	msg := &cache.NodeMessage{
		Type:      cache.MsgGossip,
		SenderID:  g.localID,
		Timestamp: time.Now(),
		Payload:   payloadBytes,
	}

	// 选择目标
	// 使用fanout或所有节点，如果少于fanout
	targetCount := g.fanout
	if len(otherNodes) < targetCount {
		targetCount = len(otherNodes)
	}

	targets := g.selectRandomNodes(targetCount, otherNodes)

	// 发送Gossip给目标
	for _, target := range targets {
		go func(addr string) {
			_, err := g.transport.Send(addr, msg)
			if err != nil {
				// log.Fatalf("Error sending gossip to %s: %v\n", addr, err)
			}
		}(target.Addr)
	}

	return nil
}

// selectRandomNodes 随机选择n个节点
func (g *Gossiper) selectRandomNodes(n int, members []cache.NodeInfo) []cache.NodeInfo {
	if len(members) <= n {
		return members
	}

	g.randMu.Lock()
	defer g.randMu.Unlock()

	// 打乱成员列表
	perm := g.rand.Perm(len(members))

	// 选择前n个
	result := make([]cache.NodeInfo, n)
	for i := 0; i < n; i++ {
		result[i] = members[perm[i]]
	}

	return result
}

// HandleGossip 处理接收到的Gossip消息
func (g *Gossiper) HandleGossip(msg *cache.NodeMessage) error {
	// 验证消息类型
	if msg.Type != cache.MsgGossip {
		return nil
	}

	// 反序列化消息
	payload, err := UnmarshalGossipPayload(msg.Payload)
	if err != nil {
		return err
	}

	// 处理每个节点信息
	for _, node := range payload.Nodes {
		// 忽略关于自己的信息
		if node.ID == g.localID {
			continue
		}

		// 转换节点状态
		status := cache.NodeStatus(node.Status)

		// 恢复LastSeen时间
		//lastSeen := time.Unix(0, node.LastSeen)

		// 更新成员信息
		if status == cache.NodeStatusUp {
			g.membership.AddMember(node.ID, node.Address, node.Metadata)
			g.membership.MarkAlive(node.ID)
		} else if status == cache.NodeStatusSuspect {
			g.membership.AddMember(node.ID, node.Address, node.Metadata)
			g.membership.MarkSuspect(node.ID)
		} else if status == cache.NodeStatusDown {
			g.membership.MarkFailed(node.ID, "reported down by "+msg.SenderID)
		} else if status == cache.NodeStatusLeft {
			g.membership.RemoveMember(node.ID, true)
		}
	}

	return nil
}

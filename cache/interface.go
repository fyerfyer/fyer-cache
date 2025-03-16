package cache

import (
	"context"
	"time"
)

// Cache 基本缓存接口
type Cache interface {
	Get(ctx context.Context, key string) (any, error)
	Set(ctx context.Context, key string, val any, expiration time.Duration) error
	Del(ctx context.Context, key string) error
}

//=================== 一致性哈希相关接口 ===================

// Ring 一致性哈希环接口
type Ring interface {
	// Add 添加节点
	Add(node string, weight int)

	// Remove 移除节点
	Remove(node string)

	// Get 获取键对应的节点
	Get(key string) string

	// GetN 获取键对应的N个节点（用于备份）
	GetN(key string, count int) []string

	// GetNodes 获取所有节点
	GetNodes() []string
}

// ShardedCache 分片缓存接口
type ShardedCache interface {
	Cache

	// AddNode 添加缓存节点
	AddNode(nodeID string, cache Cache, weight int) error

	// RemoveNode 移除缓存节点
	RemoveNode(nodeID string) error

	// GetNodeIDs 获取所有节点ID
	GetNodeIDs() []string

	// GetNodeCount 获取节点数量
	GetNodeCount() int
}

//=================== 节点通信相关接口 ===================

// NodeMessage 节点间通信消息
type NodeMessage struct {
	Type      MessageType
	SenderID  string
	Timestamp time.Time
	Payload   []byte
}

// MessageType 消息类型
type MessageType int

const (
	// 集群管理消息
	MsgJoin MessageType = iota
	MsgLeave
	MsgPing
	MsgPong

	// 数据同步消息
	MsgSync
	MsgSyncRequest
	MsgSyncResponse

	// Gossip消息
	MsgGossip
)

// NodeTransport 节点通信传输层接口
type NodeTransport interface {
	// Send 发送消息到指定节点
	Send(nodeAddr string, msg *NodeMessage) (*NodeMessage, error)

	// Start 启动传输层服务
	Start() error

	// Stop 停止传输层服务
	Stop() error

	// Address 返回当前节点地址
	Address() string
}

// ClusterNode 集群节点接口
type ClusterNode interface {
	// Join 加入集群
	Join(seedNodeAddr string) error

	// Leave 离开集群
	Leave() error

	// Start 启动节点服务
	Start() error

	// Stop 停止节点服务
	Stop() error

	// Members 获取集群成员列表
	Members() []NodeInfo

	// IsCoordinator 判断当前节点是否为协调节点
	IsCoordinator() bool

	// Events 返回事件通知通道
	Events() <-chan ClusterEvent
}

// NodeInfo 节点信息
type NodeInfo struct {
	ID       string
	Addr     string
	Status   NodeStatus
	LastSeen time.Time
	Metadata map[string]string
}

// NodeStatus 节点状态
type NodeStatus int

const (
	NodeStatusUp NodeStatus = iota
	NodeStatusSuspect
	NodeStatusDown
	NodeStatusLeft
)

// ClusterEvent 集群事件
type ClusterEvent struct {
	Type    EventType
	NodeID  string
	Time    time.Time
	Details interface{}
}

// EventType 事件类型
type EventType int

const (
	EventNodeJoin EventType = iota
	EventNodeLeave
	EventNodeFailed
	EventNodeRecovered
	EventSyncCompleted
)

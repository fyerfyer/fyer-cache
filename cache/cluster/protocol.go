package cluster

import (
	"encoding/json"
	"fmt"
	"time"
)

// 消息类型常量已在 interface.go 中定义，这里不需要重复定义

// 协议版本常量
const (
	ProtocolVersion = "v1.0"
	MaxPayloadSize  = 1024 * 1024 // 1MB
)

// 各种消息载荷结构定义

// JoinPayload 节点加入消息载荷
type JoinPayload struct {
	NodeID   string            `json:"node_id"`
	Address  string            `json:"address"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// LeavePayload 节点离开消息载荷
type LeavePayload struct {
	NodeID   string `json:"node_id"`
	Graceful bool   `json:"graceful"` // 是否优雅退出
}

// PingPayload 心跳消息载荷
type PingPayload struct {
	Sequence int64             `json:"seq"` // 序列号，用于检测丢包
	Metadata map[string]string `json:"metadata,omitempty"`
}

// PongPayload 心跳响应载荷
type PongPayload struct {
	Sequence     int64             `json:"seq"`       // 对应Ping的序列号
	LoadFactor   float64           `json:"load"`      // 节点负载因子
	MemoryUsage  float64           `json:"memory"`    // 内存使用率
	ItemCount    int               `json:"items"`     // 缓存项数量
	ResponseTime time.Duration     `json:"resp_time"` // 响应时间
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// SyncRequestPayload 同步请求载荷
type SyncRequestPayload struct {
	FromNodeID string `json:"from_node_id"`
	StartKey   string `json:"start_key,omitempty"` // 起始键，空表示从头开始
	EndKey     string `json:"end_key,omitempty"`   // 结束键，空表示到末尾
	MaxItems   int    `json:"max_items,omitempty"` // 最大项目数，0表示无限制
}

// SyncItem 同步数据项
type SyncItem struct {
	Key        string        `json:"key"`
	Value      []byte        `json:"value"`
	Expiration time.Duration `json:"expiration"`
}

// SyncResponsePayload 同步响应载荷
type SyncResponsePayload struct {
	FromNodeID string     `json:"from_node_id"`
	Items      []SyncItem `json:"items"`
	HasMore    bool       `json:"has_more"`           // 是否还有更多数据
	NextKey    string     `json:"next_key,omitempty"` // 下一个键，用于分页
}

// GossipNodeInfo Gossip传播的节点信息
type GossipNodeInfo struct {
	ID       string            `json:"id"`
	Address  string            `json:"address"`
	Status   int               `json:"status"`
	LastSeen int64             `json:"last_seen"` // Unix时间戳
	Metadata map[string]string `json:"metadata,omitempty"`
}

// GossipPayload Gossip消息载荷
type GossipPayload struct {
	Nodes []GossipNodeInfo `json:"nodes"`
}

// 消息序列化与反序列化函数

// MarshalJoinPayload 序列化节点加入消息
func MarshalJoinPayload(payload *JoinPayload) ([]byte, error) {
	return json.Marshal(payload)
}

// UnmarshalJoinPayload 反序列化节点加入消息
func UnmarshalJoinPayload(data []byte) (*JoinPayload, error) {
	var payload JoinPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// MarshalLeavePayload 序列化节点离开消息
func MarshalLeavePayload(payload *LeavePayload) ([]byte, error) {
	return json.Marshal(payload)
}

// UnmarshalLeavePayload 反序列化节点离开消息
func UnmarshalLeavePayload(data []byte) (*LeavePayload, error) {
	var payload LeavePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// MarshalPingPayload 序列化心跳消息
func MarshalPingPayload(payload *PingPayload) ([]byte, error) {
	return json.Marshal(payload)
}

// UnmarshalPingPayload 反序列化心跳消息
func UnmarshalPingPayload(data []byte) (*PingPayload, error) {
	var payload PingPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// MarshalPongPayload 序列化心跳响应
func MarshalPongPayload(payload *PongPayload) ([]byte, error) {
	return json.Marshal(payload)
}

// UnmarshalPongPayload 反序列化心跳响应
func UnmarshalPongPayload(data []byte) (*PongPayload, error) {
	var payload PongPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// MarshalSyncRequestPayload 序列化同步请求
func MarshalSyncRequestPayload(payload *SyncRequestPayload) ([]byte, error) {
	return json.Marshal(payload)
}

// UnmarshalSyncRequestPayload 反序列化同步请求
func UnmarshalSyncRequestPayload(data []byte) (*SyncRequestPayload, error) {
	var payload SyncRequestPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// MarshalSyncResponsePayload 序列化同步响应
func MarshalSyncResponsePayload(payload *SyncResponsePayload) ([]byte, error) {
	return json.Marshal(payload)
}

// UnmarshalSyncResponsePayload 反序列化同步响应
func UnmarshalSyncResponsePayload(data []byte) (*SyncResponsePayload, error) {
	var payload SyncResponsePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// MarshalGossipPayload 序列化Gossip消息
func MarshalGossipPayload(payload *GossipPayload) ([]byte, error) {
	return json.Marshal(payload)
}

// UnmarshalGossipPayload 反序列化Gossip消息
func UnmarshalGossipPayload(data []byte) (*GossipPayload, error) {
	var payload GossipPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// MessageToBytes 将NodeMessage序列化为字节数组
func MessageToBytes(msg *NodeMessage) ([]byte, error) {
	return json.Marshal(msg)
}

// BytesToMessage 将字节数组反序列化为NodeMessage
func BytesToMessage(data []byte) (*NodeMessage, error) {
	var msg NodeMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}
	return &msg, nil
}

// NodeMessage 结构已在interface.go中定义，这里使用导入
type NodeMessage struct {
	Type      int       `json:"type"`
	SenderID  string    `json:"sender_id"`
	Timestamp time.Time `json:"timestamp"`
	Payload   []byte    `json:"payload"`
}

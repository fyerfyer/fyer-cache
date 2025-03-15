package cluster

import (
	"sync"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
)

// EventPublisher 事件发布器
// 管理事件通道并提供事件发布功能
type EventPublisher struct {
	eventCh     chan cache.ClusterEvent
	subscribers []chan<- cache.ClusterEvent
	mu          sync.RWMutex
	bufferSize  int
}

// NewEventPublisher 创建事件发布器
func NewEventPublisher(bufferSize int) *EventPublisher {
	if bufferSize <= 0 {
		bufferSize = 100 // 默认缓冲区大小
	}

	return &EventPublisher{
		eventCh:    make(chan cache.ClusterEvent, bufferSize),
		bufferSize: bufferSize,
	}
}

// Subscribe 订阅事件
// 返回一个只读事件通道
func (p *EventPublisher) Subscribe() <-chan cache.ClusterEvent {
	p.mu.Lock()
	defer p.mu.Unlock()

	ch := make(chan cache.ClusterEvent, p.bufferSize)
	p.subscribers = append(p.subscribers, ch)

	return ch
}

// Publish 发布事件
func (p *EventPublisher) Publish(event cache.ClusterEvent) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 发送到所有订阅者
	for _, sub := range p.subscribers {
		// 非阻塞发送，防止因订阅者不处理导致堵塞
		select {
		case sub <- event:
			// 成功发送
		default:
			// 通道已满，跳过
		}
	}

	// 同时发送到主事件通道
	select {
	case p.eventCh <- event:
		// 成功发送
	default:
		// 通道已满，跳过
	}
}

// Close 关闭事件发布器
func (p *EventPublisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 关闭所有订阅通道
	for _, sub := range p.subscribers {
		close(sub)
	}

	// 清空订阅者列表
	p.subscribers = nil

	// 关闭主事件通道
	if p.eventCh != nil {
		close(p.eventCh)
		p.eventCh = nil
	}
}

// Events 返回主事件通道
func (p *EventPublisher) Events() <-chan cache.ClusterEvent {
	return p.eventCh
}

// 以下是事件构建工具函数

// NewNodeJoinEvent 创建节点加入事件
func NewNodeJoinEvent(nodeID string, addr string) cache.ClusterEvent {
	return cache.ClusterEvent{
		Type:    cache.EventNodeJoin,
		NodeID:  nodeID,
		Time:    time.Now(),
		Details: addr,
	}
}

// NewNodeLeaveEvent 创建节点离开事件
func NewNodeLeaveEvent(nodeID string, graceful bool) cache.ClusterEvent {
	return cache.ClusterEvent{
		Type:    cache.EventNodeLeave,
		NodeID:  nodeID,
		Time:    time.Now(),
		Details: graceful,
	}
}

// NewNodeFailedEvent 创建节点故障事件
func NewNodeFailedEvent(nodeID string, reason string) cache.ClusterEvent {
	return cache.ClusterEvent{
		Type:    cache.EventNodeFailed,
		NodeID:  nodeID,
		Time:    time.Now(),
		Details: reason,
	}
}

// NewNodeRecoveredEvent 创建节点恢复事件
func NewNodeRecoveredEvent(nodeID string) cache.ClusterEvent {
	return cache.ClusterEvent{
		Type:   cache.EventNodeRecovered,
		NodeID: nodeID,
		Time:   time.Now(),
	}
}

// NewSyncCompletedEvent 创建同步完成事件
func NewSyncCompletedEvent(nodeID string, keyCount int) cache.ClusterEvent {
	return cache.ClusterEvent{
		Type:    cache.EventSyncCompleted,
		NodeID:  nodeID,
		Time:    time.Now(),
		Details: keyCount,
	}
}

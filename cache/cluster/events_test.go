package cluster

import (
	"sync"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/stretchr/testify/assert"
)

func TestNewEventPublisher(t *testing.T) {
	tests := []struct {
		name       string
		bufferSize int
		expected   int
	}{
		{
			name:       "With positive buffer size",
			bufferSize: 200,
			expected:   200,
		},
		{
			name:       "With zero buffer size",
			bufferSize: 0,
			expected:   100, // 默认值
		},
		{
			name:       "With negative buffer size",
			bufferSize: -10,
			expected:   100, // 默认值
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publisher := NewEventPublisher(tt.bufferSize)
			assert.NotNil(t, publisher)
			assert.Equal(t, tt.expected, publisher.bufferSize)
			assert.NotNil(t, publisher.eventCh)
			assert.Len(t, publisher.subscribers, 0)
		})
	}
}

func TestEventPublisher_Subscribe(t *testing.T) {
	publisher := NewEventPublisher(10)
	defer publisher.Close()

	// 订阅事件
	ch1 := publisher.Subscribe()
	assert.NotNil(t, ch1)
	assert.Len(t, publisher.subscribers, 1)

	// 再次订阅
	ch2 := publisher.Subscribe()
	assert.NotNil(t, ch2)
	assert.Len(t, publisher.subscribers, 2)

	// 验证两个通道不同
	assert.NotEqual(t, ch1, ch2)
}

func TestEventPublisher_Publish(t *testing.T) {
	publisher := NewEventPublisher(10)
	defer publisher.Close()

	// 创建两个订阅者
	subscriber1 := publisher.Subscribe()
	subscriber2 := publisher.Subscribe()
	mainChan := publisher.Events()

	// 创建测试事件
	event := cache.ClusterEvent{
		Type:   cache.EventNodeJoin,
		NodeID: "test-node",
		Time:   time.Now(),
	}

	// 发布事件
	publisher.Publish(event)

	// 验证所有订阅者都收到了事件
	select {
	case receivedEvent := <-subscriber1:
		assert.Equal(t, event.Type, receivedEvent.Type)
		assert.Equal(t, event.NodeID, receivedEvent.NodeID)
	case <-time.After(100 * time.Millisecond):
		t.Error("Subscriber 1 didn't receive the event")
	}

	select {
	case receivedEvent := <-subscriber2:
		assert.Equal(t, event.Type, receivedEvent.Type)
		assert.Equal(t, event.NodeID, receivedEvent.NodeID)
	case <-time.After(100 * time.Millisecond):
		t.Error("Subscriber 2 didn't receive the event")
	}

	select {
	case receivedEvent := <-mainChan:
		assert.Equal(t, event.Type, receivedEvent.Type)
		assert.Equal(t, event.NodeID, receivedEvent.NodeID)
	case <-time.After(100 * time.Millisecond):
		t.Error("Main channel didn't receive the event")
	}
}

func TestEventPublisher_PublishChannelFull(t *testing.T) {
	// 创建一个只有1个缓冲的发布器
	publisher := NewEventPublisher(1)
	defer publisher.Close()

	// 创建一个订阅者，但不消费事件
	subscriber := publisher.Subscribe()

	// 发布两个事件，第二个应该被丢弃而不是阻塞
	event1 := cache.ClusterEvent{Type: cache.EventNodeJoin, NodeID: "node1"}
	event2 := cache.ClusterEvent{Type: cache.EventNodeLeave, NodeID: "node2"}

	publisher.Publish(event1)
	publisher.Publish(event2) // 这不应该阻塞

	// 验证只收到第一个事件
	select {
	case receivedEvent := <-subscriber:
		assert.Equal(t, event1.Type, receivedEvent.Type)
		assert.Equal(t, "node1", receivedEvent.NodeID)

		// 检查是否有第二个事件（不应该有）
		select {
		case receivedEvent := <-subscriber:
			assert.Equal(t, event2.Type, receivedEvent.Type)
		default:
			// 期望进入这里，因为第二个事件应该被丢弃
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Subscriber didn't receive any event")
	}
}

func TestEventPublisher_Close(t *testing.T) {
	publisher := NewEventPublisher(10)

	// 订阅一些事件通道
	_ = publisher.Subscribe()
	_ = publisher.Subscribe()
	mainChan := publisher.Events()

	// 关闭发布器
	publisher.Close()

	// 验证主通道已关闭
	_, ok := <-mainChan
	assert.False(t, ok, "Main event channel should be closed")

	// 验证订阅者列表已清空
	assert.Nil(t, publisher.eventCh)
	assert.Nil(t, publisher.subscribers)
}

func TestEventPublisher_ConcurrentOperations(t *testing.T) {
	publisher := NewEventPublisher(100)
	defer publisher.Close()

	const numSubscribers = 10
	const numEvents = 50

	var wg sync.WaitGroup
	wg.Add(numSubscribers)

	// 并发订阅
	subscribers := make([]<-chan cache.ClusterEvent, numSubscribers)
	for i := 0; i < numSubscribers; i++ {
		go func(index int) {
			defer wg.Done()
			subscribers[index] = publisher.Subscribe()
		}(i)
	}

	wg.Wait()

	// 验证所有订阅者都被正确创建
	publisher.mu.RLock()
	assert.Len(t, publisher.subscribers, numSubscribers)
	publisher.mu.RUnlock()

	// 并发发布事件
	wg.Add(numEvents)
	for i := 0; i < numEvents; i++ {
		go func(index int) {
			defer wg.Done()
			event := cache.ClusterEvent{
				Type:   cache.EventNodeJoin,
				NodeID: "node-" + string(rune('A'+index%26)),
				Time:   time.Now(),
			}
			publisher.Publish(event)
		}(i)
	}

	wg.Wait()
}

func TestEventCreationHelpers(t *testing.T) {
	t.Run("NewNodeJoinEvent", func(t *testing.T) {
		event := NewNodeJoinEvent("node1", "192.168.1.1:8080")
		assert.Equal(t, cache.EventNodeJoin, event.Type)
		assert.Equal(t, "node1", event.NodeID)
		assert.Equal(t, "192.168.1.1:8080", event.Details)
	})

	t.Run("NewNodeLeaveEvent", func(t *testing.T) {
		event := NewNodeLeaveEvent("node1", true)
		assert.Equal(t, cache.EventNodeLeave, event.Type)
		assert.Equal(t, "node1", event.NodeID)
		assert.Equal(t, true, event.Details)
	})

	t.Run("NewNodeFailedEvent", func(t *testing.T) {
		event := NewNodeFailedEvent("node1", "connection timeout")
		assert.Equal(t, cache.EventNodeFailed, event.Type)
		assert.Equal(t, "node1", event.NodeID)
		assert.Equal(t, "connection timeout", event.Details)
	})

	t.Run("NewNodeRecoveredEvent", func(t *testing.T) {
		event := NewNodeRecoveredEvent("node1")
		assert.Equal(t, cache.EventNodeRecovered, event.Type)
		assert.Equal(t, "node1", event.NodeID)
		assert.Nil(t, event.Details)
	})

	t.Run("NewSyncCompletedEvent", func(t *testing.T) {
		event := NewSyncCompletedEvent("node1", 100)
		assert.Equal(t, cache.EventSyncCompleted, event.Type)
		assert.Equal(t, "node1", event.NodeID)
		assert.Equal(t, 100, event.Details)
	})
}

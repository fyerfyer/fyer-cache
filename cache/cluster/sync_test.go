package cluster

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/fyerfyer/fyer-cache/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewSyncManager(t *testing.T) {
	// 创建依赖
	mockCache := mocks.NewCache(t)
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)

	// 测试使用默认配置
	sm := NewSyncManager("node1", mockCache, mockTransport, events)
	assert.NotNil(t, sm)
	assert.Equal(t, "node1", sm.nodeID)
	assert.Equal(t, mockCache, sm.cache)
	assert.Equal(t, mockTransport, sm.transport)
	assert.Equal(t, events, sm.events)
	assert.Equal(t, time.Minute, sm.interval)
	assert.Equal(t, 5, sm.concurrency)
	assert.Equal(t, 100, sm.maxItems)
	assert.NotNil(t, sm.queue)
	assert.NotNil(t, sm.processingMap)
	assert.NotNil(t, sm.stopCh)
	assert.False(t, sm.running)

	// 测试使用自定义配置
	customSM := NewSyncManager("node1", mockCache, mockTransport, events,
		WithSyncerInterval(30*time.Second),
		WithSyncerConcurrency(10),
		WithMaxSyncItems(500),
	)
	assert.Equal(t, 30*time.Second, customSM.interval)
	assert.Equal(t, 10, customSM.concurrency)
	assert.Equal(t, 500, customSM.maxItems)
}

func TestSyncManager_StartStop(t *testing.T) {
	// 创建依赖
	mockCache := mocks.NewCache(t)
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)

	sm := NewSyncManager("node1", mockCache, mockTransport, events,
		WithSyncerInterval(10*time.Millisecond), // 短间隔加快测试速度
	)

	// 验证初始状态
	assert.False(t, sm.running)

	// 启动同步管理器
	err := sm.Start()
	assert.NoError(t, err)
	assert.True(t, sm.running)

	// 再次启动，应该不报错
	err = sm.Start()
	assert.NoError(t, err)

	// 停止同步管理器
	err = sm.Stop()
	assert.NoError(t, err)
	assert.False(t, sm.running)

	// 再次停止，应该不报错
	err = sm.Stop()
	assert.NoError(t, err)
}

func TestSyncManager_QueueSyncTask(t *testing.T) {
	// 创建依赖
	mockCache := mocks.NewCache(t)
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)

	sm := NewSyncManager("node1", mockCache, mockTransport, events)

	// 添加几个任务
	sm.QueueSyncTask("node2", KeyRange{Start: "a", End: "b"}, 1)
	sm.QueueSyncTask("node3", KeyRange{Start: "c", End: "d"}, 3)
	sm.QueueSyncTask("node4", KeyRange{Start: "e", End: "f"}, 2)

	// 验证任务数量
	sm.mu.Lock()
	assert.Equal(t, 3, len(sm.queue))

	// 验证任务优先级排序，高优先级应该排在前面
	// 注意：由于实际的QueueSyncTask实现可能不会立即排序，这个检查可能需要调整
	// 如果它使用了特定的排序算法，可能需要先处理一下再验证
	// 这里假设实现会按priority降序排序
	if len(sm.queue) >= 3 {
		assert.Equal(t, "node3", sm.queue[0].targetNode)
		assert.Equal(t, "node4", sm.queue[1].targetNode)
		assert.Equal(t, "node2", sm.queue[2].targetNode)
	}
	sm.mu.Unlock()
}

func TestSyncManager_TriggerSync(t *testing.T) {
	// 创建依赖
	mockCache := mocks.NewCache(t)
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)

	sm := NewSyncManager("node1", mockCache, mockTransport, events)

	// 触发同步
	sm.TriggerSync("node2")

	// 验证任务
	sm.mu.Lock()
	assert.Equal(t, 1, len(sm.queue))
	if len(sm.queue) > 0 {
		assert.Equal(t, "node2", sm.queue[0].targetNode)
		assert.Equal(t, "", sm.queue[0].keyRange.Start)
		assert.Equal(t, "", sm.queue[0].keyRange.End)
		assert.Equal(t, 2, sm.queue[0].priority) // 默认优先级应该是2
	}
	sm.mu.Unlock()
}

func TestSyncManager_TriggerFullSync(t *testing.T) {
	// 创建依赖
	mockCache := mocks.NewCache(t)
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)

	sm := NewSyncManager("node1", mockCache, mockTransport, events)

	// 触发完整同步
	nodes := []string{"node2", "node3", "node4"}
	sm.TriggerFullSync(nodes)

	// 验证为每个节点创建了同步任务
	sm.mu.Lock()
	assert.Equal(t, len(nodes), len(sm.queue))
	sm.mu.Unlock()
}

func TestSyncManager_HandleSyncRequest(t *testing.T) {
	// 创建依赖
	mockCache := mocks.NewCache(t)
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)

	sm := NewSyncManager("node1", mockCache, mockTransport, events)

	// 创建一些测试数据
	//ctx := context.Background()
	mockCache.EXPECT().Get(mock.Anything, "key1").Return("value1", nil).Maybe()
	mockCache.EXPECT().Get(mock.Anything, "key2").Return("value2", nil).Maybe()
	mockCache.EXPECT().Get(mock.Anything, "key3").Return("value2", nil).Maybe()

	// 创建同步请求消息
	syncReq := &SyncRequestPayload{
		FromNodeID: "node2",
		StartKey:   "a",
		EndKey:     "z",
		MaxItems:   10,
	}

	payload, err := MarshalSyncRequestPayload(syncReq)
	assert.NoError(t, err)

	msg := &cache.NodeMessage{
		Type:      cache.MsgSyncRequest,
		SenderID:  "node2",
		Timestamp: time.Now(),
		Payload:   payload,
	}

	// 处理同步请求
	resp, err := sm.HandleSyncRequest(msg)

	// 即使没有任何键，我们也应该得到一个正确的响应
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, cache.MsgSyncResponse, resp.Type)
	assert.Equal(t, "node1", resp.SenderID)

	// 解析响应
	var respPayload SyncResponsePayload
	err = json.Unmarshal(resp.Payload, &respPayload)
	assert.NoError(t, err)

	// 验证响应内容
	assert.Equal(t, "node1", respPayload.FromNodeID)
}

func TestSyncManager_SyncToNode(t *testing.T) {
	// 创建依赖
	mockCache := mocks.NewCache(t)
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)

	sm := NewSyncManager("node1", mockCache, mockTransport, events)

	// 设置mock transport的预期行为
	syncResp := &SyncResponsePayload{
		FromNodeID: "node2",
		Items: []SyncItem{
			{
				Key:        "key1",
				Value:      []byte("value1"),
				Expiration: time.Minute,
			},
		},
		HasMore: false,
	}
	respBytes, _ := MarshalSyncResponsePayload(syncResp)

	mockTransport.EXPECT().Send(mock.Anything, mock.Anything).Run(func(nodeAddr string, msg *cache.NodeMessage) {
		// 验证发送的是同步请求消息
		assert.Equal(t, cache.MsgSyncRequest, msg.Type)
		assert.Equal(t, "node1", msg.SenderID)

		// 验证请求负载
		payload, _ := UnmarshalSyncRequestPayload(msg.Payload)
		assert.Equal(t, "node1", payload.FromNodeID)
		assert.Equal(t, "a", payload.StartKey)
		assert.Equal(t, "z", payload.EndKey)
		assert.Equal(t, sm.maxItems, payload.MaxItems)
	}).Return(&cache.NodeMessage{
		Type:     cache.MsgSyncResponse,
		SenderID: "node2",
		Payload:  respBytes,
	}, nil)

	// 设置cache的预期行为
	mockCache.EXPECT().Set(mock.Anything, "key1", mock.Anything, time.Minute).Return(nil)

	// 添加一个任务
	sm.QueueSyncTask("node2", KeyRange{Start: "a", End: "z"}, 1)

	// 执行同步任务处理
	go sm.processSyncTasks()

	// 等待任务被处理
	time.Sleep(100 * time.Millisecond)
}

func TestSyncManager_ProcessSyncTasks(t *testing.T) {
	// 创建依赖
	mockCache := mocks.NewCache(t)
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)

	sm := NewSyncManager("node1", mockCache, mockTransport, events)

	// 添加几个任务
	sm.QueueSyncTask("node2", KeyRange{Start: "a", End: "b"}, 1)
	sm.QueueSyncTask("node3", KeyRange{Start: "c", End: "d"}, 2)

	// 设置模拟响应
	syncResp := &SyncResponsePayload{
		FromNodeID: "node2",
		Items:      []SyncItem{},
		HasMore:    false,
	}
	respBytes, _ := MarshalSyncResponsePayload(syncResp)

	// 设置mock transport的预期行为
	mockTransport.EXPECT().Send(mock.Anything, mock.Anything).Return(&cache.NodeMessage{
		Type:     cache.MsgSyncResponse,
		SenderID: "node2",
		Payload:  respBytes,
	}, nil).Maybe()

	// 处理同步任务
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		sm.processSyncTasks()
	}()

	// 等待处理完成
	wg.Wait()

	// 验证队列应该已经处理
	sm.mu.Lock()
	assert.Equal(t, 0, len(sm.queue))
	assert.Equal(t, 0, len(sm.processingMap))
	sm.mu.Unlock()
}

func TestSyncManager_SyncLoop(t *testing.T) {
	// 创建依赖
	mockCache := mocks.NewCache(t)
	mockTransport := mocks.NewNodeTransporter(t)
	events := NewEventPublisher(10)

	sm := NewSyncManager("node1", mockCache, mockTransport, events,
		WithSyncerInterval(10*time.Millisecond), // 短间隔加快测试
	)

	// 设置mock transport的预期行为
	syncResp := &SyncResponsePayload{
		FromNodeID: "node2",
		Items:      []SyncItem{},
		HasMore:    false,
	}
	respBytes, _ := MarshalSyncResponsePayload(syncResp)

	mockTransport.EXPECT().Send(mock.Anything, mock.Anything).Return(&cache.NodeMessage{
		Type:     cache.MsgSyncResponse,
		SenderID: "node2",
		Payload:  respBytes,
	}, nil).Maybe()

	// 添加任务
	sm.QueueSyncTask("node2", KeyRange{Start: "a", End: "z"}, 1)

	// 启动同步管理器
	err := sm.Start()
	assert.NoError(t, err)

	// 等待同步循环运行几次
	time.Sleep(30 * time.Millisecond)

	// 停止同步管理器
	err = sm.Stop()
	assert.NoError(t, err)
}

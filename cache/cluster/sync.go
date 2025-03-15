package cluster

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
)

// KeyRange 表示键的范围，用于分批同步
type KeyRange struct {
	Start string // 开始键
	End   string // 结束键
}

// syncTask 表示一个同步任务
type syncTask struct {
	targetNode string   // 目标节点
	keyRange   KeyRange // 键范围
	priority   int      // 优先级
}

// SyncManager 数据同步管理器
// 负责节点间的数据同步
type SyncManager struct {
	nodeID        string
	cache         cache.Cache         // 本地缓存
	transport     NodeTransporter     // 通信层
	interval      time.Duration       // 同步间隔
	concurrency   int                 // 并发同步任务数
	queue         []*syncTask         // 同步任务队列
	processingMap map[string]struct{} // 正在处理的任务
	mu            sync.Mutex          // 互斥锁保护队列和处理映射
	events        *EventPublisher     // 事件发布器
	stopCh        chan struct{}       // 停止信号
	running       bool                // 运行状态标记
	maxItems      int                 // 单次同步的最大项目数
}

// NodeTransporter 已在membership.go中定义，是节点传输器接口

// SyncManagerOption 同步管理器配置选项
type SyncManagerOption func(*SyncManager)

// WithSyncerInterval 设置同步间隔
func WithSyncerInterval(interval time.Duration) SyncManagerOption {
	return func(s *SyncManager) {
		if interval > 0 {
			s.interval = interval
		}
	}
}

// WithSyncerConcurrency 设置同步并发数
func WithSyncerConcurrency(concurrency int) SyncManagerOption {
	return func(s *SyncManager) {
		if concurrency > 0 {
			s.concurrency = concurrency
		}
	}
}

// WithMaxSyncItems 设置单次同步最大项数
func WithMaxSyncItems(maxItems int) SyncManagerOption {
	return func(s *SyncManager) {
		if maxItems > 0 {
			s.maxItems = maxItems
		}
	}
}

// NewSyncManager 创建新的同步管理器
func NewSyncManager(nodeID string, cache cache.Cache, transport NodeTransporter, events *EventPublisher, options ...SyncManagerOption) *SyncManager {
	sm := &SyncManager{
		nodeID:        nodeID,
		cache:         cache,
		transport:     transport,
		interval:      time.Minute, // 默认同步间隔1分钟
		concurrency:   5,           // 默认5个并发同步任务
		queue:         make([]*syncTask, 0),
		processingMap: make(map[string]struct{}),
		events:        events,
		stopCh:        make(chan struct{}),
		maxItems:      100, // 默认单次同步100个项目
	}

	// 应用选项
	for _, option := range options {
		option(sm)
	}

	return sm
}

// Start 启动同步管理器
func (sm *SyncManager) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.running {
		return nil
	}

	sm.running = true

	// 启动定期同步
	go sm.syncLoop()

	return nil
}

// Stop 停止同步管理器
func (sm *SyncManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.running {
		return nil
	}

	// 检查是否已经关闭通道
	select {
	case <-sm.stopCh:
	default:
		// 通道未关闭
		close(sm.stopCh)
	}

	sm.running = false
	return nil
}

// syncLoop 定期执行同步任务
func (sm *SyncManager) syncLoop() {
	ticker := time.NewTicker(sm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-sm.stopCh:
			return
		case <-ticker.C:
			sm.processSyncTasks()
		}
	}
}

// processSyncTasks 处理同步任务队列
func (sm *SyncManager) processSyncTasks() {
	sm.mu.Lock()

	// 如果没有任务，退出
	if len(sm.queue) == 0 {
		sm.mu.Unlock()
		return
	}

	// 有多少任务可以执行
	taskCount := sm.concurrency
	if taskCount > len(sm.queue) {
		taskCount = len(sm.queue)
	}

	// 选择要处理的任务
	tasks := sm.queue[:taskCount]
	sm.queue = sm.queue[taskCount:]

	sm.mu.Unlock()

	// 并发处理任务
	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Add(1)
		go func(t *syncTask) {
			defer wg.Done()
			sm.syncToNode(t.targetNode, t.keyRange)
		}(task)
	}

	// 等待所有任务完成
	wg.Wait()
}

// syncToNode 向特定节点发送同步请求
func (sm *SyncManager) syncToNode(targetNode string, keyRange KeyRange) {
	// 防止重复同步同一个节点
	taskKey := fmt.Sprintf("%s-%s-%s", targetNode, keyRange.Start, keyRange.End)

	sm.mu.Lock()
	if _, exists := sm.processingMap[taskKey]; exists {
		sm.mu.Unlock()
		return
	}
	sm.processingMap[taskKey] = struct{}{}
	sm.mu.Unlock()

	// 函数结束时清理
	defer func() {
		sm.mu.Lock()
		delete(sm.processingMap, taskKey)
		sm.mu.Unlock()
	}()

	// 创建同步请求
	syncReq := &SyncRequestPayload{
		FromNodeID: sm.nodeID,
		StartKey:   keyRange.Start,
		EndKey:     keyRange.End,
		MaxItems:   sm.maxItems,
	}

	// 序列化请求
	payload, err := MarshalSyncRequestPayload(syncReq)
	if err != nil {
		fmt.Printf("Failed to marshal sync request: %v\n", err)
		return
	}

	// 创建消息
	msg := &cache.NodeMessage{
		Type:      cache.MsgSyncRequest,
		SenderID:  sm.nodeID,
		Timestamp: time.Now(),
		Payload:   payload,
	}

	// 查找目标节点地址
	targetAddr := targetNode // 假定targetNode就是地址，实际上可能需要查找

	// 发送请求
	resp, err := sm.transport.Send(targetAddr, msg)
	if err != nil {
		fmt.Printf("Failed to send sync request to %s: %v\n", targetAddr, err)
		return
	}

	// 处理响应
	if resp.Type != cache.MsgSyncResponse {
		fmt.Printf("Unexpected response type from %s: %d\n", targetAddr, resp.Type)
		return
	}

	// 解析响应
	syncResp, err := UnmarshalSyncResponsePayload(resp.Payload)
	if err != nil {
		fmt.Printf("Failed to unmarshal sync response: %v\n", err)
		return
	}

	// 处理同步项
	appliedItems := 0

	ctx := context.Background()
	for _, item := range syncResp.Items {
		// 写入本地缓存
		err := sm.cache.Set(ctx, item.Key, item.Value, item.Expiration)
		if err != nil {
			fmt.Printf("Failed to apply sync item %s: %v\n", item.Key, err)
			continue
		}
		appliedItems++
	}

	// 如果还有更多数据，继续同步
	if syncResp.HasMore && syncResp.NextKey != "" {
		// 创建新的任务
		nextRange := KeyRange{
			Start: syncResp.NextKey,
			End:   keyRange.End,
		}

		sm.QueueSyncTask(targetNode, nextRange, 1)
	}

	// 发布同步完成事件
	if sm.events != nil {
		sm.events.Publish(NewSyncCompletedEvent(targetNode, appliedItems))
	}
}

// QueueSyncTask 将同步任务加入队列
func (sm *SyncManager) QueueSyncTask(targetNode string, keyRange KeyRange, priority int) {
	task := &syncTask{
		targetNode: targetNode,
		keyRange:   keyRange,
		priority:   priority,
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.queue = append(sm.queue, task)

	// 根据优先级排序，高优先级排在前面
	if len(sm.queue) > 1 {
		// 简单的冒泡排序，实际中可能需要更高效的排序算法
		for i := len(sm.queue) - 1; i > 0; i-- {
			if sm.queue[i].priority > sm.queue[i-1].priority {
				sm.queue[i], sm.queue[i-1] = sm.queue[i-1], sm.queue[i]
			} else {
				break
			}
		}
	}
}

// TriggerSync 触发对特定节点的同步
func (sm *SyncManager) TriggerSync(targetNode string) {
	sm.QueueSyncTask(targetNode, KeyRange{Start: "", End: ""}, 2) // 空范围表示全部
}

// TriggerFullSync 触发对所有节点的完整同步
func (sm *SyncManager) TriggerFullSync(nodes []string) {
	// 随机化节点顺序，避免所有节点同时开始同步造成网络风暴
	randomNodes := make([]string, len(nodes))
	copy(randomNodes, nodes)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := len(randomNodes) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		randomNodes[i], randomNodes[j] = randomNodes[j], randomNodes[i]
	}

	// 为每个节点创建同步任务
	for _, node := range randomNodes {
		if node != sm.nodeID { // 不同步自己
			sm.QueueSyncTask(node, KeyRange{Start: "", End: ""}, 1)
		}
	}
}

// HandleSyncRequest 处理收到的同步请求
func (sm *SyncManager) HandleSyncRequest(msg *cache.NodeMessage) (*cache.NodeMessage, error) {
	// 解析请求
	syncReq, err := UnmarshalSyncRequestPayload(msg.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal sync request: %w", err)
	}

	// 提取请求参数
	startKey := syncReq.StartKey
	endKey := syncReq.EndKey
	maxItems := syncReq.MaxItems

	// 准备响应
	response := &SyncResponsePayload{
		FromNodeID: sm.nodeID,
		Items:      make([]SyncItem, 0, maxItems),
		HasMore:    false,
		NextKey:    "",
	}

	// 收集缓存数据
	// 注意：这里是一个简化实现，实际上我们需要一种方式来有效地遍历键范围
	// 这里我们假设有一个方法可以列出所有键，然后我们过滤它们

	ctx := context.Background()

	// 这里是一个虚构的例子，实际实现会依赖于Cache接口提供的遍历功能
	// 或者直接访问底层存储
	keys := []string{"key1", "key2", "key3"} // 假设这些是所有键

	collectedCount := 0
	var lastKey string

	for _, key := range keys {
		// 检查是否在请求的范围内
		if (startKey == "" || key >= startKey) && (endKey == "" || key < endKey) {
			// 获取值
			value, err := sm.cache.Get(ctx, key)
			if err != nil {
				continue // 跳过获取失败的项
			}

			// 转换为字节数组
			// 注意：这里假设值可以直接转为[]byte，实际实现可能需要序列化
			valueBytes, ok := value.([]byte)
			if !ok {
				// 简单处理：跳过不是字节数组的值
				continue
			}

			// 添加到响应
			item := SyncItem{
				Key:        key,
				Value:      valueBytes,
				Expiration: 5 * time.Minute, // 这里使用默认过期时间，实际实现应获取真实过期时间
			}

			response.Items = append(response.Items, item)
			collectedCount++
			lastKey = key

			// 检查是否达到最大项目数
			if maxItems > 0 && collectedCount >= maxItems {
				response.HasMore = true
				response.NextKey = lastKey
				break
			}
		}
	}

	// 序列化响应
	respPayload, err := MarshalSyncResponsePayload(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sync response: %w", err)
	}

	// 创建响应消息
	respMsg := &cache.NodeMessage{
		Type:      cache.MsgSyncResponse,
		SenderID:  sm.nodeID,
		Timestamp: time.Now(),
		Payload:   respPayload,
	}

	return respMsg, nil
}

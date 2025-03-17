package replication

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
)

// MemorySyncer 是内存实现的数据同步器
type MemorySyncer struct {
	// 本节点的缓存
	localCache cache.Cache

	// 本节点ID
	nodeID string

	// 复制日志
	log ReplicationLog

	// 配置选项
	config *ReplicationConfig

	// HTTP客户端，用于与其他节点通信
	client *http.Client

	// 日志通道，用于接收新的操作日志
	logCh chan *ReplicationEntry

	// 状态锁
	mu sync.RWMutex

	// 运行状态
	running bool

	// 停止信号通道
	stopCh chan struct{}

	// HTTP服务器相关
	httpServer *http.Server
	mux        *http.ServeMux
	serverPort int32 // 使用atomic操作访问
}

// NewMemorySyncer 创建新的内存同步器
func NewMemorySyncer(nodeID string, localCache cache.Cache, log ReplicationLog, options ...ReplicationOption) *MemorySyncer {
	// 使用默认配置
	config := DefaultReplicationConfig()

	// 应用选项
	for _, opt := range options {
		opt(config)
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: config.ReplicationTimeout,
	}

	return &MemorySyncer{
		nodeID:     nodeID,
		localCache: localCache,
		log:        log,
		config:     config,
		client:     client,
		logCh:      make(chan *ReplicationEntry, 1000), // 缓冲区大小1000
		stopCh:     make(chan struct{}),
	}
}

// Start 启动同步器
func (s *MemorySyncer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	s.running = true

	// 启动后台处理协程
	go s.processLogEntries()

	// 启动 HTTP 服务器
	if err := s.StartHTTPServer(); err != nil {
		s.running = false
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	return nil
}

// Stop 停止同步器
func (ms *MemorySyncer) Stop() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if !ms.running {
		return nil
	}

	// 关闭HTTP服务器
	if ms.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := ms.httpServer.Shutdown(ctx); err != nil {
			fmt.Printf("HTTP server close error: %v\n", err)
		}
	}

	// 安全关闭通道
	select {
	case <-ms.stopCh:
		// 已关闭
	default:
		close(ms.stopCh)
	}

	ms.running = false
	return nil
}

// processLogEntries 处理日志条目
func (s *MemorySyncer) processLogEntries() {
	for {
		select {
		case <-s.stopCh:
			return
		case entry := <-s.logCh:
			s.applyEntryToLocal(context.Background(), entry)
		}
	}
}

// FullSync 执行全量同步
func (s *MemorySyncer) FullSync(ctx context.Context, target string) error {
	// 发送全量同步请求
	req := &SyncRequest{
		LeaderID:     s.nodeID,
		LastLogIndex: 0,
		LastLogTerm:  0,
		FullSync:     true,
	}

	// 进行同步
	return s.syncWithRetries(ctx, target, req)
}

// IncrementalSync 执行增量同步
func (s *MemorySyncer) IncrementalSync(ctx context.Context, target string, startIndex uint64) error {
	// 获取最后一条日志的任期
	lastTerm := s.log.GetLastTerm()

	// 发送增量同步请求
	req := &SyncRequest{
		LeaderID:     s.nodeID,
		LastLogIndex: startIndex,
		LastLogTerm:  lastTerm,
		FullSync:     false,
	}

	fmt.Printf("[%s] IncrementalSync - Starting from index: %d\n", s.nodeID, startIndex)

	// 进行同步
	return s.syncWithRetries(ctx, target, req)
}

// syncWithRetries 带重试的同步
func (s *MemorySyncer) syncWithRetries(ctx context.Context, target string, req *SyncRequest) error {
	var lastErr error

	// 重试循环
	for i := 0; i < s.config.MaxRetries; i++ {
		// 检查上下文是否已取消
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// 执行同步
		err := s.sendSyncRequest(ctx, target, req)
		if err == nil {
			return nil // 成功则直接返回
		}

		lastErr = err

		// 最后一次尝试失败，不再等待
		if i == s.config.MaxRetries-1 {
			break
		}

		// 等待后重试
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.config.RetryInterval):
			// 继续重试
		}
	}

	return fmt.Errorf("sync failed after %d retries: %w", s.config.MaxRetries, lastErr)
}

// sendSyncRequest 发送同步请求
func (s *MemorySyncer) sendSyncRequest(ctx context.Context, target string, req *SyncRequest) error {
	// 序列化请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal sync request: %w", err)
	}

	// 构建URL - 添加调试日志
	url := fmt.Sprintf("http://%s/replication/sync", target)
	fmt.Printf("Sending sync request to: %s\n", url)

	// 创建请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Node-ID", s.nodeID)

	// 发送请求
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send sync request: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK response: %s", resp.Status)
	}

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// 解析响应
	var syncResp SyncResponse
	if err := json.Unmarshal(respBody, &syncResp); err != nil {
		return fmt.Errorf("failed to unmarshal sync response: %w", err)
	}

	// 处理响应
	if !syncResp.Success {
		return fmt.Errorf("sync request failed on remote node")
	}

	// 应用获取的日志条目
	if err := s.ApplySync(ctx, syncResp.Entries); err != nil {
		return fmt.Errorf("failed to apply sync entries: %w", err)
	}

	// 如果有更多数据，继续同步
	if syncResp.NextIndex > 0 {
		nextReq := &SyncRequest{
			LeaderID:     s.nodeID,
			LastLogIndex: syncResp.NextIndex,
			LastLogTerm:  req.LastLogTerm,
			FullSync:     false,
		}
		return s.sendSyncRequest(ctx, target, nextReq)
	}

	return nil
}

// ApplySync 应用同步数据
func (s *MemorySyncer) ApplySync(ctx context.Context, entries []*ReplicationEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// 应用每个日志条目
	for _, entry := range entries {
		if err := s.applyEntryToLocal(ctx, entry); err != nil {
			return fmt.Errorf("failed to apply entry: %w", err)
		}

		// 将条目添加到本地日志
		if err := s.log.Append(entry); err != nil {
			return fmt.Errorf("failed to append entry to local log: %w", err)
		}
	}

	return nil
}

// applyEntryToLocal 应用日志条目到本地缓存
// applyEntryToLocal 应用日志条目到本地缓存
func (s *MemorySyncer) applyEntryToLocal(ctx context.Context, entry *ReplicationEntry) error {
	if entry == nil {
		return errors.New("nil entry")
	}

	fmt.Printf("[%s] Applying entry: cmd=%s, key=%s\n", s.nodeID, entry.Command, entry.Key)

	// 根据命令类型执行不同操作
	switch entry.Command {
	case "Set":
		var value interface{} = entry.Value // default to byte array

		// Convert based on the stored type information
		if entry.ValueType == "string" {
			value = string(entry.Value)
		}

		return s.localCache.Set(ctx, entry.Key, value, entry.Expiration)
	case "Del":
		return s.localCache.Del(ctx, entry.Key)
	default:
		return fmt.Errorf("unknown command: %s", entry.Command)
	}
}

// RecordSetOperation 记录Set操作到日志
func (s *MemorySyncer) RecordSetOperation(key string, value []byte, expiration time.Duration, term uint64) error {
	entry := &ReplicationEntry{
		Term:       term,
		Command:    "Set",
		Key:        key,
		Value:      value,
		ValueType:  "string",
		Expiration: expiration,
		Timestamp:  time.Now(),
	}

	// 添加到复制日志
	if err := s.log.Append(entry); err != nil {
		return fmt.Errorf("failed to append Set operation to log: %w", err)
	}

	// 如果正在运行，将条目发送到处理通道
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.running {
		select {
		case s.logCh <- entry:
			// 成功发送到通道
		default:
			// 通道已满，不阻塞，但记录错误
			return fmt.Errorf("log channel is full, entry discarded")
		}
	}

	return nil
}

// RecordDelOperation 记录Del操作到日志
func (s *MemorySyncer) RecordDelOperation(key string, term uint64) error {
	entry := &ReplicationEntry{
		Term:      term,
		Command:   "Del",
		Key:       key,
		Timestamp: time.Now(),
	}

	// 添加到复制日志
	if err := s.log.Append(entry); err != nil {
		return fmt.Errorf("failed to append Del operation to log: %w", err)
	}

	// 如果正在运行，将条目发送到处理通道
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.running {
		select {
		case s.logCh <- entry:
			// 成功发送到通道
		default:
			// 通道已满，不阻塞，但记录错误
			return fmt.Errorf("log channel is full, entry discarded")
		}
	}

	return nil
}

// CleanupExpiredEntries 清理过期的日志条目
func (s *MemorySyncer) CleanupExpiredEntries(ctx context.Context, maxAge time.Duration) error {
	// 获取所有日志条目
	entries, err := s.log.GetFrom(1, int(s.log.GetLastIndex()))
	if err != nil {
		return fmt.Errorf("failed to get log entries: %w", err)
	}

	// 计算截断点
	cutoffTime := time.Now().Add(-maxAge)
	var truncateIndex uint64 = 0

	// 查找应保留的第一个条目索引
	for _, entry := range entries {
		if entry.Timestamp.After(cutoffTime) {
			break
		}
		truncateIndex = entry.Index
	}

	// 如果找到了截断点，截断日志
	if truncateIndex > 0 {
		if err := s.log.Truncate(truncateIndex); err != nil {
			return fmt.Errorf("failed to truncate log: %w", err)
		}
	}

	return nil
}

// StartHTTPServer 启动同步 HTTP 服务器
func (s *MemorySyncer) StartHTTPServer() error {
	// 先检查是否已经有服务运行
	if s.httpServer != nil {
		return nil // Server already running
	}

	// 创建新的 mux 或复用现有的
	if s.mux == nil {
		s.mux = http.NewServeMux()
	}

	// 清除旧的处理程序并重新注册
	s.mux = http.NewServeMux()

	// 注册同步处理程序
	s.mux.HandleFunc("/replication/sync", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		var req SyncRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		resp, err := s.HandleSyncRequest(ctx, &req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to handle sync request: %v", err), http.StatusInternalServerError)
			return
		}

		respBody, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(respBody)
	})

	s.mux.HandleFunc("/replication/receive", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		var resp SyncResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			http.Error(w, "Failed to unmarshal sync response", http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		// 记录接收到的条目
		fmt.Printf("[%s] Received %d entries from leader\n", s.nodeID, len(resp.Entries))
		for i, entry := range resp.Entries {
			fmt.Printf("[%s] Entry %d: cmd=%s, key=%s\n", s.nodeID, i, entry.Command, entry.Key)
		}

		// 应用同步数据
		if err := s.ApplySync(ctx, resp.Entries); err != nil {
			http.Error(w, "Failed to apply sync entries", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 添加健康检查端点
	s.mux.HandleFunc("/replication/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 创建新的 HTTP 服务器
	addr := s.config.Address
	fmt.Printf("Starting replicate HTTP server at %s (node: %s)\n", addr, s.nodeID)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.mux,
	}

	// 启动 HTTP 服务器 (使用goroutine)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	// 等待服务器启动 (添加短暂延迟让服务器有时间启动)
	time.Sleep(100 * time.Millisecond)

	return nil
}

// HandleSyncRequest 处理来自从节点的同步请求
func (s *MemorySyncer) HandleSyncRequest(ctx context.Context, req *SyncRequest) (*SyncResponse, error) {
	// 准备响应
	resp := &SyncResponse{
		Success:   true,
		Entries:   make([]*ReplicationEntry, 0),
		NextIndex: req.LastLogIndex,
	}

	// 判断是全量同步还是增量同步
	startIndex := req.LastLogIndex
	if req.FullSync {
		startIndex = 0 // 全量同步从0开始
	}

	// 获取日志条目
	entries, err := s.log.GetFrom(startIndex+1, s.config.MaxLogEntries)
	if err != nil && startIndex > 0 { // 如果不是因为没有条目导致的错误
		return nil, fmt.Errorf("failed to get log entries: %w", err)
	}

	fmt.Printf("[%s] Sending %d entries in sync response\n", s.nodeID, len(entries))
	for i, entry := range entries {
		fmt.Printf("[%s] Entry %d: cmd=%s, key=%s\n", s.nodeID, i, entry.Command, entry.Key)
	}

	if len(entries) > 0 {
		resp.Entries = entries
		resp.NextIndex = entries[len(entries)-1].Index
	} else {
		// 没有新条目
		resp.NextIndex = s.log.GetLastIndex()
	}

	return resp, nil
}

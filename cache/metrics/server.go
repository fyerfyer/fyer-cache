package metrics

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// Server 提供HTTP服务器，暴露指标端点
type Server struct {
	exporter Exporter
	config   *MetricsConfig
	server   *http.Server
	running  bool
	mu       sync.RWMutex
}

// NewServer 创建一个新的指标服务器
func NewServer(exporter Exporter, config *MetricsConfig) *Server {
	if config == nil {
		config = DefaultMetricsConfig()
	}

	return &Server{
		exporter: exporter,
		config:   config,
		running:  false,
	}
}

// Start 启动HTTP服务器
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil // 已经运行中
	}

	mux := http.NewServeMux()
	mux.HandleFunc(s.config.MetricsPath, s.handleMetrics)
	// 添加健康检查端点
	mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Addr:    s.config.Address,
		Handler: mux,
	}

	// 在后台启动HTTP服务器
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// 实际应用中应该有适当的日志记录
		}
	}()

	s.running = true
	return nil
}

// Stop 停止HTTP服务器
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || s.server == nil {
		return nil // 已经停止或从未启动
	}

	// 创建一个带超时的上下文用于优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 关闭服务器
	if err := s.server.Shutdown(ctx); err != nil {
		return err
	}

	s.running = false
	return nil
}

// IsRunning 返回服务器是否正在运行
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// handleMetrics 处理/metrics端点的HTTP请求
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// 调用导出器将指标导出到响应
	s.exporter.Export(w)
}

// handleHealth 处理/health端点的HTTP请求
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
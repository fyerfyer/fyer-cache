package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
)

// APIServer 提供缓存管理 HTTP API
type APIServer struct {
	// 基础缓存实例
	cache cache.Cache

	// 集群缓存提供者(可选)，用于扩缩容操作
	clusterCache ClusterCacheProvider

	// HTTP 服务器
	server *http.Server

	// API 配置
	config *APIConfig

	// 服务器状态
	mu      sync.RWMutex
	running bool

	// 实际监听地址（用于随机端口）
	actualAddr string
}

// NewAPIServer 创建新的 API 服务器
func NewAPIServer(cache cache.Cache, options ...Option) *APIServer {
	// 使用默认配置
	config := DefaultAPIConfig()

	// 应用选项
	for _, option := range options {
		option(config)
	}

	return &APIServer{
		cache:  cache,
		config: config,
	}
}

// WithClusterCache 配置集群缓存提供者，用于扩缩容操作
func (s *APIServer) WithClusterCache(clusterCache ClusterCacheProvider) *APIServer {
	s.clusterCache = clusterCache
	return s
}

// Start 启动 API 服务器
func (s *APIServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil // 已经运行中，不做任何操作
	}

	mux := http.NewServeMux()
	s.setupRoutes(mux)

	// 先创建监听器，以便获取实际端口
	listener, err := net.Listen("tcp", s.config.BindAddress)
	if err != nil {
		return err
	}

	// 获取实际监听地址
	s.actualAddr = listener.Addr().String()

	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	// 启动 HTTP 服务器，非阻塞方式
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			// 实际应用中应当有日志记录
			fmt.Printf("API server error: %v\n", err)
		}
	}()

	s.running = true
	return nil
}

// Stop 停止 API 服务器
func (s *APIServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil // 已经停止，不做任何操作
	}

	// 创建上下文，设置 5 秒超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 优雅关闭服务器
	err := s.server.Shutdown(ctx)
	s.running = false
	return err
}

// setupRoutes 设置 HTTP 路由
func (s *APIServer) setupRoutes(mux *http.ServeMux) {
	basePath := s.config.BasePath

	// GET /stats - 获取缓存状态
	mux.HandleFunc(fmt.Sprintf("%s/stats", basePath), func(w http.ResponseWriter, r *http.Request) {
		HandleStats(w, r, s.cache, s.clusterCache)
	})

	// POST /invalidate?key=X - 手动删除某个缓存
	mux.HandleFunc(fmt.Sprintf("%s/invalidate", basePath), func(w http.ResponseWriter, r *http.Request) {
		HandleInvalidate(w, r, s.cache)
	})

	// POST /scale?nodes=N - 动态扩容/缩容
	mux.HandleFunc(fmt.Sprintf("%s/scale", basePath), func(w http.ResponseWriter, r *http.Request) {
		if s.clusterCache != nil {
			HandleScale(w, r, s.clusterCache)
		} else {
			writeJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: "Scaling is only available in cluster mode",
			})
		}
	})
}

// IsRunning 检查 API 服务器是否正在运行
func (s *APIServer) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// Address 返回 API 服务器绑定地址
func (s *APIServer) Address() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 如果有实际地址，返回实际地址，否则返回配置地址
	if s.actualAddr != "" {
		return s.actualAddr
	}
	return s.config.BindAddress
}

// writeServerJSONResponse 辅助方法，写入 JSON 响应
func writeServerJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
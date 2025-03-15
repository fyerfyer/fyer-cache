package cluster

import (
	"sync"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
)

// TransportConfig 传输层配置
type TransportConfig struct {
	// 传输层监听地址
	BindAddr string

	// HTTP路径前缀
	PathPrefix string

	// 请求超时时间
	Timeout time.Duration

	// 最大闲置连接数
	MaxIdleConns int

	// 每个主机最大连接数
	MaxConnsPerHost int

	// 消息处理器
	Handler func(msg *cache.NodeMessage) (*cache.NodeMessage, error)
}

// DefaultTransportConfig 返回默认传输层配置
func DefaultTransportConfig() *TransportConfig {
	return &TransportConfig{
		BindAddr:        ":7946", // 默认端口
		PathPrefix:      "/cluster",
		Timeout:         5 * time.Second,
		MaxIdleConns:    10,
		MaxConnsPerHost: 2,
	}
}

// Transport 基本传输层实现，是NodeTransport的抽象基类
type Transport struct {
	config    *TransportConfig
	handler   func(msg *cache.NodeMessage) (*cache.NodeMessage, error)
	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
	started   bool
	mu        sync.RWMutex
}

// NewTransport 创建基本传输层
func NewTransport(config *TransportConfig) *Transport {
	if config == nil {
		config = DefaultTransportConfig()
	}

	t := &Transport{
		config:  config,
		handler: config.Handler,
		stopCh:  make(chan struct{}),
	}

	return t
}

// SetHandler 设置消息处理器
func (t *Transport) SetHandler(handler func(msg *cache.NodeMessage) (*cache.NodeMessage, error)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.handler = handler
}

// HandleMessage 处理接收到的消息
func (t *Transport) HandleMessage(msg *cache.NodeMessage) (*cache.NodeMessage, error) {
	t.mu.RLock()
	handler := t.handler
	t.mu.RUnlock()

	if handler != nil {
		return handler(msg)
	}

	// 默认实现：简单地返回相同的消息作为回显
	return msg, nil
}

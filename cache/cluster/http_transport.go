package cluster

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
)

// HTTPTransport 基于HTTP的节点传输实现
type HTTPTransport struct {
	*Transport
	server *http.Server
	client *http.Client
}

// NewHTTPTransport 创建新的基于HTTP的传输层
func NewHTTPTransport(config *TransportConfig) *HTTPTransport {
	if config == nil {
		config = DefaultTransportConfig()
	}

	baseTransport := NewTransport(config)

	client := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        config.MaxIdleConns,
			MaxConnsPerHost:     config.MaxConnsPerHost,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}

	return &HTTPTransport{
		Transport: baseTransport,
		client:    client,
	}
}

// Start 启动HTTP传输层服务
func (t *HTTPTransport) Start() error {
	var startErr error

	t.startOnce.Do(func() {
		t.mu.Lock()
		defer t.mu.Unlock()

		mux := http.NewServeMux()
		mux.HandleFunc(t.config.PathPrefix+"/message", t.handleHTTPMessage)

		t.server = &http.Server{
			Addr:    t.config.BindAddr,
			Handler: mux,
		}

		go func() {
			if err := t.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				// 只记录错误，不阻止启动
				fmt.Printf("HTTP transport server error: %v\n", err)
			}
		}()

		t.started = true
	})

	return startErr
}

// Stop 停止HTTP传输层服务
func (t *HTTPTransport) Stop() error {
	var stopErr error

	t.stopOnce.Do(func() {
		t.mu.Lock()
		defer t.mu.Unlock()

		if !t.started {
			return
		}

		// 创建超时上下文
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 关闭HTTP服务器
		stopErr = t.server.Shutdown(ctx)
		close(t.stopCh)
		t.started = false
	})

	return stopErr
}

// Send 发送消息到指定节点
func (t *HTTPTransport) Send(nodeAddr string, msg *cache.NodeMessage) (*cache.NodeMessage, error) {
	if nodeAddr == "" {
		return nil, fmt.Errorf("empty node address")
	}

	// 序列化消息
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	// 构造URL
	url := fmt.Sprintf("http://%s%s/message", nodeAddr, t.config.PathPrefix)

	// 发送HTTP请求
	resp, err := t.client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK response: %s", resp.Status)
	}

	// 读取响应内容
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 反序列化响应
	var response cache.NodeMessage
	if err := json.Unmarshal(respData, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// Address 返回当前节点地址
func (t *HTTPTransport) Address() string {
	return t.config.BindAddr
}

// handleHTTPMessage 处理HTTP消息请求
func (t *HTTPTransport) handleHTTPMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// 反序列化消息
	var msg cache.NodeMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		http.Error(w, "Failed to unmarshal message", http.StatusBadRequest)
		return
	}

	// 处理消息
	response, err := t.HandleMessage(&msg)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to handle message: %v", err), http.StatusInternalServerError)
		return
	}

	// 序列化响应
	respData, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	// 返回响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respData)
}

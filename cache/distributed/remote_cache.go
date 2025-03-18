package distributed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/fyerfyer/fyer-cache/internal/ferr"
)

// RemoteCache 实现对远程缓存节点的操作
type RemoteCache struct {
	nodeID   string
	address  string
	client   HTTPClient // 改为接口类型，不要使用指针
	timeout  time.Duration
	retries  int
	interval time.Duration
	debugLog bool
}

// RemoteCacheOption 配置选项函数
type RemoteCacheOption func(*RemoteCache)

// WithTimeout 设置操作超时
func WithTimeout(timeout time.Duration) RemoteCacheOption {
	return func(rc *RemoteCache) {
		if timeout > 0 {
			rc.timeout = timeout
		}
	}
}

// WithRetry 设置重试参数
func WithRetry(retries int, interval time.Duration) RemoteCacheOption {
	return func(rc *RemoteCache) {
		if retries > 0 {
			rc.retries = retries
		}
		if interval > 0 {
			rc.interval = interval
		}
	}
}

// WithDebugLog 启用调试日志
func WithDebugLog(enabled bool) RemoteCacheOption {
	return func(rc *RemoteCache) {
		rc.debugLog = enabled
	}
}

// HTTPClient 定义简单的HTTP客户端接口
// 这里定义一个简单接口而不是直接依赖具体实现，方便测试和扩展
type HTTPClient interface {
	Do(method, url string, body, result interface{}) error
}

// NewRemoteCache 创建新的远程缓存客户端
func NewRemoteCache(nodeID string, address string, options ...RemoteCacheOption) *RemoteCache {
	rc := &RemoteCache{
		nodeID:   nodeID,
		address:  address,
		timeout:  2 * time.Second, // 默认2秒超时
		retries:  3,               // 默认3次重试
		interval: 100 * time.Millisecond,
	}

	// 应用配置选项
	for _, opt := range options {
		opt(rc)
	}

	// 创建HTTP客户端
	rc.client = &defaultHTTPClient{ // 这里没变，因为defaultHTTPClient实现了HTTPClient接口
		timeout: rc.timeout,
	}

	return rc
}

// Get 从远程缓存获取数据
func (rc *RemoteCache) Get(ctx context.Context, key string) (any, error) {
	var result struct {
		Value interface{} `json:"value"`
	}

	req := struct {
		Key string `json:"key"`
	}{
		Key: key,
	}

	// 构造URL
	url := fmt.Sprintf("http://%s/cache/get", rc.address)

	// 执行带重试的HTTP请求
	err := rc.doWithRetry("POST", url, req, &result)

	if err != nil {
		// 判断是否为"键不存在"错误
		if err.Error() == "key not found" {
			return nil, ferr.ErrKeyNotFound
		}
		return nil, fmt.Errorf("remote get failed: %w", err)
	}

	return result.Value, nil
}

// Set 在远程缓存中设置数据
func (rc *RemoteCache) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	req := struct {
		Key        string      `json:"key"`
		Value      interface{} `json:"value"`
		Expiration int64       `json:"expiration"` // 毫秒
	}{
		Key:        key,
		Value:      value,
		Expiration: expiration.Milliseconds(),
	}

	// 构造URL
	url := fmt.Sprintf("http://%s/cache/set", rc.address)

	// 执行带重试的HTTP请求
	err := rc.doWithRetry("POST", url, req, nil)
	if err != nil {
		return fmt.Errorf("remote set failed: %w", err)
	}

	return nil
}

// Del 从远程缓存删除数据
func (rc *RemoteCache) Del(ctx context.Context, key string) error {
	req := struct {
		Key string `json:"key"`
	}{
		Key: key,
	}

	// 构造URL
	url := fmt.Sprintf("http://%s/cache/delete", rc.address)

	// 执行带重试的HTTP请求
	err := rc.doWithRetry("POST", url, req, nil)
	if err != nil {
		return fmt.Errorf("remote delete failed: %w", err)
	}

	return nil
}

// GetLocal 获取本地节点数据，不适用于RemoteCache，总是返回错误
func (rc *RemoteCache) GetLocal(ctx context.Context, key string) (any, error) {
	return nil, fmt.Errorf("GetLocal not supported on remote cache")
}

// SetLocal 在本地节点设置数据，不适用于RemoteCache，总是返回错误
func (rc *RemoteCache) SetLocal(ctx context.Context, key string, value any, expiration time.Duration) error {
	return fmt.Errorf("SetLocal not supported on remote cache")
}

// DelLocal 从本地节点删除数据，不适用于RemoteCache，总是返回错误
func (rc *RemoteCache) DelLocal(ctx context.Context, key string) error {
	return fmt.Errorf("DelLocal not supported on remote cache")
}

// NodeID 返回节点ID
func (rc *RemoteCache) NodeID() string {
	return rc.nodeID
}

// NodeAddress 返回节点地址
func (rc *RemoteCache) NodeAddress() string {
	return rc.address
}

// doWithRetry 执行带重试逻辑的HTTP请求
func (rc *RemoteCache) doWithRetry(method, url string, body, result interface{}) error {
	if rc.debugLog {
		log.Printf("DEBUG RemoteCache: Sending %s request to %s", method, url)
		if body != nil {
			bodyJSON, _ := json.Marshal(body)
			log.Printf("DEBUG RemoteCache: Request body: %s", string(bodyJSON))
		}
	}

	var lastErr error
	for i := 0; i <= rc.retries; i++ {
		err := rc.client.Do(method, url, body, result)
		if err == nil {
			return nil
		}

		lastErr = err

		// 如果是最后一次尝试，不再等待
		if i == rc.retries {
			break
		}

		// 等待一段时间后重试
		time.Sleep(rc.interval)
	}

	if lastErr != nil && rc.debugLog {
		log.Printf("DEBUG RemoteCache: Response received successfully")
	}

	return lastErr
}

// defaultHTTPClient 是HTTPClient接口的默认实现
type defaultHTTPClient struct {
	timeout time.Duration
}

// Do 执行HTTP请求
func (c *defaultHTTPClient) Do(method, url string, body, result interface{}) error {
	// 检查URL是否为空
	if url == "" {
		return fmt.Errorf("empty URL")
	}

	if method == "" {
		return fmt.Errorf("empty method")
	}

	// 检查请求方法和主体
	if method == "POST" && body == nil {
		return fmt.Errorf("body required for POST method")
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: c.timeout,
	}

	var req *http.Request
	var err error

	// 准备请求体
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}

		// 创建请求
		req, err = http.NewRequest(method, url, bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		// 无请求体的请求
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned error status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// 如果需要解析结果且提供了result参数
	if result != nil {
		// 读取响应体
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		// 解析响应
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// RemoteCacheError 表示远程缓存操作错误
type RemoteCacheError struct {
	StatusCode int
	Message    string
}

func (e *RemoteCacheError) Error() string {
	return fmt.Sprintf("remote cache error (code=%d): %s", e.StatusCode, e.Message)
}
package distributed

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/internal/ferr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistributedCacheConfig(t *testing.T) {
	// 测试默认配置
	defaultConfig := NewDistributedCacheConfig()
	assert.Equal(t, 1, defaultConfig.Weight)
	assert.Equal(t, 1, defaultConfig.ReplicaFactor)
	assert.Equal(t, 100, defaultConfig.VirtualNodes)
	assert.Equal(t, 2*time.Second, defaultConfig.OperationTimeout)
	assert.Equal(t, 3, defaultConfig.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, defaultConfig.RetryInterval)

	// 测试配置选项
	tests := []struct {
		name     string
		option   Option
		expected func(*DistributedCacheConfig) bool
	}{
		{
			name:   "WithNodeID",
			option: WithNodeID("test-node"),
			expected: func(cfg *DistributedCacheConfig) bool {
				return cfg.NodeID == "test-node"
			},
		},
		{
			name:   "WithAddress",
			option: WithAddress("localhost:8080"),
			expected: func(cfg *DistributedCacheConfig) bool {
				return cfg.Address == "localhost:8080"
			},
		},
		{
			name:   "WithWeight",
			option: WithWeight(5),
			expected: func(cfg *DistributedCacheConfig) bool {
				return cfg.Weight == 5
			},
		},
		{
			name:   "WithWeight negative",
			option: WithWeight(-1),
			expected: func(cfg *DistributedCacheConfig) bool {
				return cfg.Weight == 1 // 应该保持不变
			},
		},
		{
			name:   "WithReplicaFactor",
			option: WithReplicaFactor(3),
			expected: func(cfg *DistributedCacheConfig) bool {
				return cfg.ReplicaFactor == 3
			},
		},
		{
			name:   "WithVirtualNodes",
			option: WithVirtualNodes(200),
			expected: func(cfg *DistributedCacheConfig) bool {
				return cfg.VirtualNodes == 200
			},
		},
		{
			name:   "WithOperationTimeout",
			option: WithOperationTimeout(5 * time.Second),
			expected: func(cfg *DistributedCacheConfig) bool {
				return cfg.OperationTimeout == 5*time.Second
			},
		},
		{
			name:   "WithRetryOptions",
			option: WithRetryOptions(5, 200*time.Millisecond),
			expected: func(cfg *DistributedCacheConfig) bool {
				return cfg.MaxRetries == 5 && cfg.RetryInterval == 200*time.Millisecond
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewDistributedCacheConfig()
			tt.option(cfg)
			if !tt.expected(cfg) {
				t.Errorf("Option %s did not set expected value", tt.name)
			}
		})
	}
}

func TestRemoteCache(t *testing.T) {
	// 简单创建和方法测试
	rc := NewRemoteCache("node1", "localhost:8080")
	require.NotNil(t, rc)
	assert.Equal(t, "node1", rc.NodeID())
	assert.Equal(t, "localhost:8080", rc.NodeAddress())

	// 测试配置选项
	rcWithOptions := NewRemoteCache("node2", "localhost:8081",
		WithTimeout(5*time.Second),
		WithRetry(5, 200*time.Millisecond),
	)
	assert.Equal(t, 5*time.Second, rcWithOptions.timeout)
	assert.Equal(t, 5, rcWithOptions.retries)
	assert.Equal(t, 200*time.Millisecond, rcWithOptions.interval)

	// 测试本地操作不支持
	ctx := context.Background()
	_, err := rc.GetLocal(ctx, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported on remote cache")

	err = rc.SetLocal(ctx, "key", "value", time.Minute)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported on remote cache")

	err = rc.DelLocal(ctx, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported on remote cache")
}

// TestHTTPClient 测试自定义HTTP客户端
func TestHTTPClient(t *testing.T) {
	client := &defaultHTTPClient{
		timeout: 2 * time.Second,
	}

	// 测试无效URL
	err := client.Do("POST", "", nil, nil)
	assert.Error(t, err)

	// 测试请求方法和主体验证
	err = client.Do("", "http://example.com", nil, nil)
	assert.Error(t, err)

	// 测试正常请求
	// 仅模拟返回结果，不实际发送请求
	result := struct {
		Value string `json:"value"`
	}{}
	err = client.Do("GET", "http://example.com", nil, &result)
	assert.NoError(t, err)

	// 测试不支持的结果类型
	unsupported := "string result"
	err = client.Do("GET", "http://example.com", nil, &unsupported)
	assert.Error(t, err)

	// 测试支持的结果类型
	mapResult := make(map[string]interface{})
	err = client.Do("GET", "http://example.com", nil, &mapResult)
	assert.NoError(t, err)

	// 测试另一种支持的结果类型
	structResult := struct{ Value interface{} }{}
	err = client.Do("GET", "http://example.com", nil, &structResult)
	assert.NoError(t, err)
}

func TestRemoteCacheOperations(t *testing.T) {
	ctx := context.Background()

	// 创建一个测试HTTP客户端
	mockClient := &mockHTTPClient{
		responses: map[string]mockResponse{
			"cache/get": { // 移除前导斜杠
				result: map[string]interface{}{"value": "test-value"},
				err:    nil,
			},
			"cache/set": { // 移除前导斜杠
				result: nil,
				err:    nil,
			},
			"cache/delete": { // 移除前导斜杠
				result: nil,
				err:    nil,
			},
		},
	}

	// 创建RemoteCache并注入mock客户端
	rc := NewRemoteCache("node1", "localhost:8080")
	rc.client = mockClient

	// 测试Get
	fmt.Println("[TestRemoteCacheOperations] Testing Get...")
	value, err := rc.Get(ctx, "test-key")
	fmt.Printf("[TestRemoteCacheOperations] Get result: value=%v, err=%v\n", value, err)
	assert.NoError(t, err)
	assert.Equal(t, "test-value", value)

	// 测试Set
	fmt.Println("[TestRemoteCacheOperations] Testing Set...")
	err = rc.Set(ctx, "test-key", "test-value", time.Minute)
	fmt.Printf("[TestRemoteCacheOperations] Set result: err=%v\n", err)
	assert.NoError(t, err)

	// 测试Del
	fmt.Println("[TestRemoteCacheOperations] Testing Del...")
	err = rc.Del(ctx, "test-key")
	fmt.Printf("[TestRemoteCacheOperations] Del result: err=%v\n", err)
	assert.NoError(t, err)

	// 测试错误处理
	mockClient.responses["cache/get"] = mockResponse{nil, ferr.ErrKeyNotFound}
	fmt.Println("[TestRemoteCacheOperations] Testing error handling...")
	_, err = rc.Get(ctx, "missing-key")
	fmt.Printf("[TestRemoteCacheOperations] Error handling result: err=%v\n", err)
	assert.Equal(t, ferr.ErrKeyNotFound, err)

	// 测试doWithRetry重试逻辑
	retryClient := &mockRetryHTTPClient{
		failCount: 2, // 前两次失败，第三次成功
		result:    map[string]interface{}{"value": "retry-value"},
	}

	rc.client = retryClient
	rc.retries = 3

	fmt.Println("[TestRemoteCacheOperations] Testing retry logic...")
	value, err = rc.Get(ctx, "retry-key")
	fmt.Printf("[TestRemoteCacheOperations] Retry logic result: value=%v, err=%v\n", value, err)
	assert.NoError(t, err)
	assert.Equal(t, "retry-value", value)
}

// mockHTTPClient 是一个模拟HTTP客户端
type mockHTTPClient struct {
	responses map[string]mockResponse
}

type mockResponse struct {
	result interface{}
	err    error
}

func (m *mockHTTPClient) Do(method, url string, body, result interface{}) error {
	// 从URL中提取路径
	path := url[len("http://"):]
	path = path[strings.Index(path, "/")+1:]

	fmt.Printf("[mockHTTPClient.Do] URL: %s, path: %s\n", url, path)

	response, ok := m.responses[path]
	if !ok {
		fmt.Printf("[mockHTTPClient.Do] No mock response for path: %s\n", path)
		return fmt.Errorf("no mock response for %s", path)
	}

	if response.err != nil {
		fmt.Printf("[mockHTTPClient.Do] Returning error: %v\n", response.err)
		return response.err
	}

	// 如果需要填充结果且提供了result参数
	if result != nil && response.result != nil {
		fmt.Printf("[mockHTTPClient.Do] Result type: %T, response result type: %T\n", result, response.result)

		// 使用反射获取Value字段
		resultVal := reflect.ValueOf(result)
		if resultVal.Kind() == reflect.Ptr && resultVal.Elem().Kind() == reflect.Struct {
			// 尝试找到名为Value的字段
			valueField := resultVal.Elem().FieldByName("Value")
			if valueField.IsValid() && valueField.CanSet() {
				// 如果response.result是map且包含"value"键
				if valueMap, ok := response.result.(map[string]interface{}); ok {
					if value, exists := valueMap["value"]; exists {
						fmt.Printf("[mockHTTPClient.Do] Setting struct Value: %v\n", value)
						valueField.Set(reflect.ValueOf(value))
					}
				}
			}
		}
	}

	return nil
}

// mockRetryHTTPClient是一个模拟重试逻辑的HTTP客户端
type mockRetryHTTPClient struct {
	failCount int
	callCount int
	result    interface{}
}

func (m *mockRetryHTTPClient) Do(method, url string, body, result interface{}) error {
	m.callCount++
	fmt.Printf("[mockRetryHTTPClient.Do] Call %d, URL: %s\n", m.callCount, url)

	// 模拟前几次调用失败
	if m.callCount <= m.failCount {
		fmt.Printf("[mockRetryHTTPClient.Do] Simulated failure %d\n", m.callCount)
		return fmt.Errorf("simulated failure %d", m.callCount)
	}

	// 最后一次成功
	if result != nil {
		fmt.Printf("[mockRetryHTTPClient.Do] Result type: %T, m.result type: %T\n", result, m.result)

		// 使用反射获取Value字段
		resultVal := reflect.ValueOf(result)
		if resultVal.Kind() == reflect.Ptr && resultVal.Elem().Kind() == reflect.Struct {
			// 尝试找到名为Value的字段
			valueField := resultVal.Elem().FieldByName("Value")
			if valueField.IsValid() && valueField.CanSet() {
				// 如果m.result是map且包含"value"键
				if valueMap, ok := m.result.(map[string]interface{}); ok {
					if value, exists := valueMap["value"]; exists {
						fmt.Printf("[mockRetryHTTPClient.Do] Setting struct Value: %v\n", value)
						valueField.Set(reflect.ValueOf(value))
					}
				}
			} else {
				fmt.Printf("[mockRetryHTTPClient.Do] Cannot find or set Value field\n")
			}
		} else {
			fmt.Printf("[mockRetryHTTPClient.Do] Result is not a pointer to struct\n")
		}
	}

	return nil
}

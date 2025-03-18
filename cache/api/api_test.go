package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockMemoryStatsProvider 是内存统计提供者的模拟实现
type MockMemoryStatsProvider struct {
	mock.Mock
	data map[string]interface{}
}

func NewMockMemoryStatsProvider() *MockMemoryStatsProvider {
	return &MockMemoryStatsProvider{
		data: make(map[string]interface{}),
	}
}

func (m *MockMemoryStatsProvider) Get(ctx context.Context, key string) (any, error) {
	args := m.Called(ctx, key)
	if args.Get(0) != nil {
		return args.Get(0), args.Error(1)
	}

	// 默认实现
	val, ok := m.data[key]
	if !ok {
		return nil, fmt.Errorf("key not found")
	}
	return val, nil
}

func (m *MockMemoryStatsProvider) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
	args := m.Called(ctx, key, val, expiration)
	if args.Error(0) != nil {
		return args.Error(0)
	}

	// 默认实现
	m.data[key] = val
	return nil
}

func (m *MockMemoryStatsProvider) Del(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	if args.Error(0) != nil {
		return args.Error(0)
	}

	// 默认实现
	delete(m.data, key)
	return nil
}

func (m *MockMemoryStatsProvider) MemoryUsage() int64 {
	args := m.Called()
	return args.Get(0).(int64)
}

func (m *MockMemoryStatsProvider) ItemCount() int64 {
	args := m.Called()
	return args.Get(0).(int64)
}

func (m *MockMemoryStatsProvider) HitRate() float64 {
	args := m.Called()
	return args.Get(0).(float64)
}

func (m *MockMemoryStatsProvider) MissRate() float64 {
	args := m.Called()
	return args.Get(0).(float64)
}

// MockClusterProvider 是集群提供者的模拟实现
type MockClusterProvider struct {
	mock.Mock
}

func (m *MockClusterProvider) GetNodeCount() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockClusterProvider) GetClusterInfo() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

func (m *MockClusterProvider) AddNode(nodeID string, address string) error {
	args := m.Called(nodeID, address)
	return args.Error(0)
}

func (m *MockClusterProvider) RemoveNode(nodeID string) error {
	args := m.Called(nodeID)
	return args.Error(0)
}

// TestAPIServer_Creation 测试API服务器的创建
func TestAPIServer_Creation(t *testing.T) {
	mockCache := &mocks.Cache{}

	// 测试默认选项
	server := NewAPIServer(mockCache)
	assert.NotNil(t, server)
	assert.Equal(t, mockCache, server.cache)
	assert.Equal(t, ":8080", server.config.BindAddress)
	assert.Equal(t, "/api", server.config.BasePath)

	// 测试自定义选项
	customServer := NewAPIServer(
		mockCache,
		WithBindAddress(":9090"),
		WithBasePath("/admin"),
		WithTimeouts(20*time.Second, 30*time.Second, 60*time.Second),
	)
	assert.Equal(t, ":9090", customServer.config.BindAddress)
	assert.Equal(t, "/admin", customServer.config.BasePath)
	assert.Equal(t, 20*time.Second, customServer.config.ReadTimeout)
	assert.Equal(t, 30*time.Second, customServer.config.WriteTimeout)
}

// TestAPIServer_StartStop 测试API服务器的启动和停止
func TestAPIServer_StartStop(t *testing.T) {
	mockCache := &mocks.Cache{}

	// 使用随机端口避免端口冲突
	server := NewAPIServer(mockCache, WithBindAddress(":0"))

	// 启动服务器
	err := server.Start()
	require.NoError(t, err)
	assert.True(t, server.IsRunning())

	// 再次启动应该不会出错
	err = server.Start()
	assert.NoError(t, err)

	// 停止服务器
	err = server.Stop()
	require.NoError(t, err)
	assert.False(t, server.IsRunning())

	// 再次停止应该不会出错
	err = server.Stop()
	assert.NoError(t, err)
}

// TestHandleStats 测试获取统计信息
func TestHandleStats(t *testing.T) {
	// 设置 mock
	mockStats := new(MockMemoryStatsProvider)
	mockCluster := new(MockClusterProvider)

	// 设置期望行为
	mockStats.On("ItemCount").Return(int64(100))
	mockStats.On("MemoryUsage").Return(int64(1024 * 1024)) // 1MB
	mockStats.On("HitRate").Return(0.75)
	mockStats.On("MissRate").Return(0.25)

	clusterInfo := map[string]interface{}{
		"nodes": []interface{}{"node1", "node2", "node3"},
		"replica_factor": float64(2),
	}
	mockCluster.On("GetNodeCount").Return(3)
	mockCluster.On("GetClusterInfo").Return(clusterInfo)

	// 创建测试请求
	req, err := http.NewRequest("GET", "/api/stats", nil)
	require.NoError(t, err)

	// 创建 ResponseRecorder 记录响应
	rr := httptest.NewRecorder()

	// 调用被测试的处理函数
	HandleStats(rr, req, mockStats, mockCluster)

	// 检查状态码
	assert.Equal(t, http.StatusOK, rr.Code)

	// 解析响应体
	var stats StatsResponse
	err = json.Unmarshal(rr.Body.Bytes(), &stats)
	require.NoError(t, err)

	// 验证响应内容
	assert.Equal(t, int64(100), stats.ItemCount)
	assert.Equal(t, int64(1024*1024), stats.MemoryUsage)
	assert.Equal(t, 0.75, stats.HitRate)
	assert.Equal(t, 0.25, stats.MissRate)
	assert.Equal(t, 3, stats.NodeCount)
	assert.Equal(t, clusterInfo, stats.ClusterInfo)

	// 验证预期调用
	mockStats.AssertExpectations(t)
	mockCluster.AssertExpectations(t)
}

// TestHandleInvalidate 测试缓存失效
func TestHandleInvalidate(t *testing.T) {
	// 设置 mock
	mockCache := &mocks.Cache{}

	// 设置期望行为 - Del 应该被调用一次
	mockCache.On("Del", mock.Anything, "testkey").Return(nil)

	// 创建测试请求
	req, err := http.NewRequest("POST", "/api/invalidate?key=testkey", nil)
	require.NoError(t, err)

	// 创建 ResponseRecorder 记录响应
	rr := httptest.NewRecorder()

	// 调用被测试的处理函数
	HandleInvalidate(rr, req, mockCache)

	// 检查状态码
	assert.Equal(t, http.StatusOK, rr.Code)

	// 解析响应体
	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	// 验证响应内容
	assert.True(t, response.Success)
	assert.Contains(t, response.Message, "testkey")

	// 验证预期调用
	mockCache.AssertExpectations(t)

	// 测试错误情况 - 缺少key参数
	req, _ = http.NewRequest("POST", "/api/invalidate", nil)
	rr = httptest.NewRecorder()

	HandleInvalidate(rr, req, mockCache)

	// 检查状态码
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// TestHandleScale 测试扩缩容
func TestHandleScale(t *testing.T) {
	// 设置 mock
	mockCluster := new(MockClusterProvider)

	// 设置期望行为
	mockCluster.On("GetNodeCount").Return(2)
	mockCluster.On("AddNode", "node-3", mock.Anything).Return(nil)

	// 创建测试请求 - 扩容到3个节点
	req, err := http.NewRequest("POST", "/api/scale?nodes=3", nil)
	require.NoError(t, err)

	// 创建 ResponseRecorder 记录响应
	rr := httptest.NewRecorder()

	// 调用被测试的处理函数
	HandleScale(rr, req, mockCluster)

	// 检查状态码
	assert.Equal(t, http.StatusOK, rr.Code)

	// 解析响应体
	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	// 验证响应内容
	assert.True(t, response.Success)
	assert.Contains(t, response.Message, "scaled from 2 to 3")

	// 验证预期调用
	mockCluster.AssertExpectations(t)

	// 测试非集群模式
	req, _ = http.NewRequest("POST", "/api/scale?nodes=2", nil)
	rr = httptest.NewRecorder()

	HandleScale(rr, req, nil)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// TestAPIServerIntegration 简单的API服务器集成测试
func TestAPIServerIntegration(t *testing.T) {
	// 设置 mock
	mockCache := &mocks.Cache{}
	mockStats := NewMockMemoryStatsProvider()
	mockCluster := new(MockClusterProvider)

	// 设置期望行为
	mockStats.On("ItemCount").Return(int64(100))
	mockStats.On("MemoryUsage").Return(int64(1024 * 1024))
	mockStats.On("HitRate").Return(0.75)
	mockStats.On("MissRate").Return(0.25)
	mockStats.On("Del", mock.Anything, "testkey").Return(nil)
	mockCluster.On("GetNodeCount").Return(3)
	mockCluster.On("GetClusterInfo").Return(map[string]interface{}{})

	// 创建 API 服务器，使用随机端口避免冲突
	server := NewAPIServer(mockCache, WithBindAddress(":0"))
	server.cache = mockStats  // 替换为支持统计的缓存
	server.WithClusterCache(mockCluster)

	// 启动服务器
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// 给服务器一点时间启动
	time.Sleep(100 * time.Millisecond)

	// 获取实际绑定的地址
	address := server.Address()
	baseURL := fmt.Sprintf("http://%s%s", address, server.config.BasePath)

	// 测试 GET /stats
	resp, err := http.Get(baseURL + "/stats")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// 测试 POST /invalidate
	invalidateURL := baseURL + "/invalidate?key=testkey"
	invalidateResp, err := http.Post(invalidateURL, "application/json", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, invalidateResp.StatusCode)

	// 读取并验证响应
	body, err := io.ReadAll(invalidateResp.Body)
	require.NoError(t, err)
	invalidateResp.Body.Close()

	var response APIResponse
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)
	assert.True(t, response.Success)

	// 验证预期调用
	mockCache.AssertExpectations(t)
	mockStats.AssertExpectations(t)
	mockCluster.AssertExpectations(t)
}
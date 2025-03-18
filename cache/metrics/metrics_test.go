package metrics

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/internal/ferr"
	"github.com/fyerfyer/fyer-cache/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockCollector 是一个简单的Collector实现，用于测试
type mockCollector struct {
	mock.Mock
}

func (m *mockCollector) IncHits() {
	m.Called()
}

func (m *mockCollector) IncMisses() {
	m.Called()
}

func (m *mockCollector) IncRequests(operation string) {
	m.Called(operation)
}

func (m *mockCollector) ObserveLatency(operation string, duration time.Duration) {
	m.Called(operation, duration)
}

func (m *mockCollector) SetMemoryUsage(bytes int64) {
	m.Called(bytes)
}

func (m *mockCollector) SetItemCount(count int64) {
	m.Called(count)
}

func (m *mockCollector) HitRate() float64 {
	args := m.Called()
	return args.Get(0).(float64)
}

// mockExporter 是一个简单的Exporter实现，用于测试
type mockExporter struct {
	mock.Mock
}

func (m *mockExporter) Export(w http.ResponseWriter) {
	m.Called(w)
}

func (m *mockExporter) RegisterMetric(name, help string, typ MetricType, labels ...string) {
	m.Called(name, help, typ, labels)
}

// TestPrometheusCollector_NewPrometheusCollector 测试创建新的收集器
func TestPrometheusCollector_NewPrometheusCollector(t *testing.T) {
	// 测试使用默认配置
	collector := NewPrometheusCollector(nil)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.registry)
	assert.NotNil(t, collector.requestCounter)
	assert.NotNil(t, collector.latencyHistogram)
	assert.Equal(t, DefaultNamespace, collector.namespace)
	assert.Equal(t, DefaultSubsystem, collector.subsystem)

	// 测试使用自定义配置
	config := &MetricsConfig{
		Namespace: "test",
		Subsystem: "cache_test",
	}
	collector = NewPrometheusCollector(config)
	assert.Equal(t, "test", collector.namespace)
	assert.Equal(t, "cache_test", collector.subsystem)
}

// TestPrometheusCollector_BasicOperations 测试收集器的基本操作
func TestPrometheusCollector_BasicOperations(t *testing.T) {
	collector := NewPrometheusCollector(nil)

	// 测试hits和misses计数
	collector.IncHits()
	collector.IncHits()
	collector.IncMisses()

	assert.Equal(t, uint64(2), collector.hitCount)
	assert.Equal(t, uint64(1), collector.missCount)
	assert.InDelta(t, 0.666, collector.HitRate(), 0.001)

	// 测试请求计数器
	collector.IncRequests("get")
	collector.IncRequests("set")
	collector.IncRequests("get")

	// 测试延迟观测
	collector.ObserveLatency("get", 10*time.Millisecond)
	collector.ObserveLatency("set", 20*time.Millisecond)

	// 测试设置内存和项目数
	collector.SetMemoryUsage(1024)
	collector.SetItemCount(100)
}

// TestPrometheusExporter_Export 测试导出器的导出功能
func TestPrometheusExporter_Export(t *testing.T) {
	// 使用明确配置创建收集器
	config := DefaultMetricsConfig()
	collector := NewPrometheusCollector(config)
	require.NotNil(t, collector)
	require.NotNil(t, collector.registry, "Registry should not be nil")

	// 创建导出器
	exporter := NewPrometheusExporter(collector)
	require.NotNil(t, exporter)
	require.NotNil(t, exporter.registry, "Exporter registry should not be nil")

	// 添加一些样本数据
	collector.IncHits()
	collector.IncRequests("get")
	collector.ObserveLatency("get", 10*time.Millisecond)

	// 创建一个测试HTTP响应记录器
	recorder := httptest.NewRecorder()

	// 导出指标
	exporter.Export(recorder)

	// 验证响应
	assert.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	assert.Contains(t, body, "fyercache_cache_hit_count")
	assert.Contains(t, body, "fyercache_cache_request_count")
	assert.Contains(t, body, "fyercache_cache_operation_latency_seconds")
}

// TestPrometheusExporter_RegisterMetric 测试注册新的指标
func TestPrometheusExporter_RegisterMetric(t *testing.T) {
	collector := NewPrometheusCollector(nil)
	exporter := NewPrometheusExporter(collector)

	// 注册不同类型的指标
	exporter.RegisterMetric("test_counter", "Test counter", TypeCounter, "label1")
	exporter.RegisterMetric("test_gauge", "Test gauge", TypeGauge, "label1")
	exporter.RegisterMetric("test_histogram", "Test histogram", TypeHistogram, "label1")
	exporter.RegisterMetric("test_summary", "Test summary", TypeSummary, "label1")

	// 没有直接的方法来检查注册，但我们可以验证没有panic
}

// TestServer_StartStop 测试服务器的启动和停止功能
func TestServer_StartStop(t *testing.T) {
	collector := NewPrometheusCollector(nil)
	exporter := NewPrometheusExporter(collector)

	config := DefaultMetricsConfig()
	config.Address = "127.0.0.1:0" // 使用系统分配的端口
	server := NewServer(exporter, config)

	// 启动服务器
	err := server.Start()
	require.NoError(t, err)
	assert.True(t, server.IsRunning())

	// 再次启动，不应报错
	err = server.Start()
	assert.NoError(t, err)

	// 停止服务器
	err = server.Stop()
	require.NoError(t, err)
	assert.False(t, server.IsRunning())

	// 再次停止，不应报错
	err = server.Stop()
	assert.NoError(t, err)
}

// TestMonitoredCache_Operations 测试监控缓存的基本操作
func TestMonitoredCache_Operations(t *testing.T) {
	// 创建mock缓存
	mockCache := new(mocks.Cache)

	// 设置预期行为
	mockCache.EXPECT().Get(mock.Anything, "hit-key").Return("value", nil)
	mockCache.EXPECT().Get(mock.Anything, "miss-key").Return(nil, ferr.ErrKeyNotFound)
	mockCache.EXPECT().Get(mock.Anything, "error-key").Return(nil, errors.New("unexpected error"))
	mockCache.EXPECT().Set(mock.Anything, "test-key", "test-value", 1*time.Minute).Return(nil)
	mockCache.EXPECT().Del(mock.Anything, "test-key").Return(nil)

	// 创建mock收集器
	mockCollector := new(mockCollector)
	mockCollector.On("IncHits").Return()
	mockCollector.On("IncMisses").Return()
	mockCollector.On("IncRequests", mock.Anything).Return()
	mockCollector.On("ObserveLatency", mock.Anything, mock.Anything).Return()
	mockCollector.On("HitRate").Return(0.75)

	// 创建mock导出器
	mockExporter := new(mockExporter)

	// 创建受监控的缓存
	monitoredCache := &MonitoredCache{
		cache:      mockCache,
		collector:  mockCollector,
		exporter:   mockExporter,
		stopCh:     make(chan struct{}),
		labels:     make(map[string]string),
	}

	ctx := context.Background()

	// 测试Get - 命中
	value, err := monitoredCache.Get(ctx, "hit-key")
	assert.NoError(t, err)
	assert.Equal(t, "value", value)
	mockCollector.AssertCalled(t, "IncHits")
	mockCollector.AssertCalled(t, "IncRequests", "get")
	mockCollector.AssertCalled(t, "ObserveLatency", "get", mock.Anything)

	// 测试Get - 未命中
	_, err = monitoredCache.Get(ctx, "miss-key")
	assert.Equal(t, ferr.ErrKeyNotFound, err)
	mockCollector.AssertCalled(t, "IncMisses")

	// 测试Get - 其他错误
	_, err = monitoredCache.Get(ctx, "error-key")
	assert.Error(t, err)
	assert.NotEqual(t, ferr.ErrKeyNotFound, err)

	// 测试Set
	err = monitoredCache.Set(ctx, "test-key", "test-value", 1*time.Minute)
	assert.NoError(t, err)
	mockCollector.AssertCalled(t, "IncRequests", "set")
	mockCollector.AssertCalled(t, "ObserveLatency", "set", mock.Anything)

	// 测试Del
	err = monitoredCache.Del(ctx, "test-key")
	assert.NoError(t, err)
	mockCollector.AssertCalled(t, "IncRequests", "del")
	mockCollector.AssertCalled(t, "ObserveLatency", "del", mock.Anything)

	// 测试HitRate
	rate := monitoredCache.HitRate()
	assert.Equal(t, 0.75, rate)
}

// TestMetricsConfig 测试指标配置
func TestMetricsConfig(t *testing.T) {
	config := DefaultMetricsConfig()
	assert.Equal(t, ":8080", config.Address)
	assert.Equal(t, "/metrics", config.MetricsPath)
	assert.Equal(t, 15*time.Second, config.CollectInterval)
	assert.Equal(t, true, config.EnableMemStats)
	assert.Equal(t, 5*time.Second, config.MemStatsInterval)
	assert.Equal(t, DefaultNamespace, config.Namespace)
	assert.Equal(t, DefaultSubsystem, config.Subsystem)
}

// TestMonitorOptions 测试监控选项
func TestMonitorOptions(t *testing.T) {
	mc := &MonitoredCache{
		namespace:          DefaultNamespace,
		subsystem:          DefaultSubsystem,
		collectInterval:    15 * time.Second,
		memoryUpdateInterval: 5 * time.Second,
		enableMemStats:     true,
		labels:             make(map[string]string),
	}

	// 测试WithMetricsServer选项
	WithMetricsServer(":9090")(mc)
	assert.NotNil(t, mc.server)
	assert.Equal(t, ":9090", mc.server.config.Address)

	// 测试WithMetricsPath选项
	WithMetricsPath("/custom-metrics")(mc)
	assert.Equal(t, "/custom-metrics", mc.server.config.MetricsPath)

	// 测试WithCollectInterval选项
	WithCollectInterval(30 * time.Second)(mc)
	assert.Equal(t, 30*time.Second, mc.collectInterval)

	// 测试WithMemoryUpdateInterval选项
	WithMemoryUpdateInterval(10 * time.Second)(mc)
	assert.Equal(t, 10*time.Second, mc.memoryUpdateInterval)

	// 测试WithNamespace选项
	WithNamespace("custom")(mc)
	assert.Equal(t, "custom", mc.namespace)

	// 测试WithSubsystem选项
	WithSubsystem("custom_cache")(mc)
	assert.Equal(t, "custom_cache", mc.subsystem)

	// 测试WithEnableMemStats选项
	WithEnableMemStats(false)(mc)
	assert.False(t, mc.enableMemStats)

	// 测试WithLabels选项
	WithLabels(map[string]string{"region": "us-west", "env": "prod"})(mc)
	assert.Equal(t, "us-west", mc.labels["region"])
	assert.Equal(t, "prod", mc.labels["env"])
}
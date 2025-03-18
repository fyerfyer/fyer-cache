package metrics

import (
	"net/http"
	"time"
)

// MetricType 表示指标类型
type MetricType string

const (
	// 指标类型常量
	TypeCounter   MetricType = "counter"
	TypeGauge     MetricType = "gauge"
	TypeHistogram MetricType = "histogram"
	TypeSummary   MetricType = "summary"
)

// 默认指标名称常量
const (
	// 命名空间和子系统
	DefaultNamespace = "fyercache"
	DefaultSubsystem = "cache"

	// 指标名称
	MetricHitCount     = "hit_count"
	MetricMissCount    = "miss_count"
	MetricRequestCount = "request_count"
	MetricLatency      = "operation_latency_seconds"
	MetricMemoryUsage  = "memory_usage_bytes"
	MetricItemCount    = "item_count"

	// 指标帮助文本
	HelpHitCount     = "Total number of cache hits"
	HelpMissCount    = "Total number of cache misses"
	HelpRequestCount = "Total number of cache requests by operation type"
	HelpLatency      = "Latency distribution of cache operations in seconds"
	HelpMemoryUsage  = "Current memory usage of the cache in bytes"
	HelpItemCount    = "Current number of items in the cache"
)

// Collector 定义了指标收集器的接口
type Collector interface {
	// 更新缓存命中/未命中计数
	IncHits()
	IncMisses()

	// 记录请求计数
	IncRequests(operation string)

	// 记录操作延迟
	ObserveLatency(operation string, duration time.Duration)

	// 设置内存用量和条目数
	SetMemoryUsage(bytes int64)
	SetItemCount(count int64)

	// 获取当前命中率
	HitRate() float64
}

// Exporter 定义了指标导出器的接口
type Exporter interface {
	// 导出收集到的指标
	Export(w http.ResponseWriter)

	// 注册新的指标
	RegisterMetric(name, help string, typ MetricType, labels ...string)
}

// Provider 定义了指标提供者的接口，结合收集器和导出器
type Provider interface {
	Collector
	Exporter

	// 启动指标暴露服务（如HTTP服务器）
	Start() error

	// 停止指标暴露服务
	Stop() error
}

// MetricsConfig 包含指标系统配置
type MetricsConfig struct {
	// HTTP服务器地址
	Address string

	// 指标路径
	MetricsPath string

	// 指标采集间隔
	CollectInterval time.Duration

	// 是否启用内存统计
	EnableMemStats bool

	// 内存统计更新间隔
	MemStatsInterval time.Duration

	// 命名空间
	Namespace string

	// 子系统
	Subsystem string
}

// DefaultMetricsConfig 返回默认指标配置
func DefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Address:          ":8080",
		MetricsPath:      "/metrics",
		CollectInterval:  15 * time.Second,
		EnableMemStats:   true,
		MemStatsInterval: 5 * time.Second,
		Namespace:        DefaultNamespace,
		Subsystem:        DefaultSubsystem,
	}
}
package metrics

import (
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusCollector 实现了基于 Prometheus 的指标收集器
type PrometheusCollector struct {
	// 命中/未命中计数
	hitCount  uint64
	missCount uint64

	// Prometheus 指标
	hitCounter       prometheus.Counter
	missCounter      prometheus.Counter
	requestCounter   *prometheus.CounterVec
	latencyHistogram *prometheus.HistogramVec
	memoryGauge      prometheus.Gauge
	itemsGauge       prometheus.Gauge

	// 注册表
	registry *prometheus.Registry

	// 命名空间和子系统
	namespace string
	subsystem string
}

// NewPrometheusCollector 创建一个新的 Prometheus 指标收集器
func NewPrometheusCollector(config *MetricsConfig) *PrometheusCollector {
	if config == nil {
		config = DefaultMetricsConfig()
	}

	namespace := config.Namespace
	subsystem := config.Subsystem

	// 创建注册表
	registry := prometheus.NewRegistry()

	// 创建命中计数器
	hitCounter := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      MetricHitCount,
			Help:      HelpHitCount,
		},
	)
	registry.MustRegister(hitCounter)

	// 创建未命中计数器
	missCounter := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      MetricMissCount,
			Help:      HelpMissCount,
		},
	)
	registry.MustRegister(missCounter)

	// 创建请求计数器
	requestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      MetricRequestCount,
			Help:      HelpRequestCount,
		},
		[]string{"operation"},
	)
	registry.MustRegister(requestCounter)

	// 创建延迟直方图
	latencyHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      MetricLatency,
			Help:      HelpLatency,
			Buckets:   []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
		[]string{"operation"},
	)
	registry.MustRegister(latencyHistogram)

	// 创建内存使用量仪表盘
	memoryGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      MetricMemoryUsage,
			Help:      HelpMemoryUsage,
		},
	)
	registry.MustRegister(memoryGauge)

	// 创建条目数仪表盘
	itemsGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      MetricItemCount,
			Help:      HelpItemCount,
		},
	)
	registry.MustRegister(itemsGauge)

	return &PrometheusCollector{
		hitCount:         0,
		missCount:        0,
		hitCounter:       hitCounter,
		missCounter:      missCounter,
		requestCounter:   requestCounter,
		latencyHistogram: latencyHistogram,
		memoryGauge:      memoryGauge,
		itemsGauge:       itemsGauge,
		registry:         registry,
		namespace:        namespace,
		subsystem:        subsystem,
	}
}

// IncHits 增加命中计数
func (pc *PrometheusCollector) IncHits() {
	atomic.AddUint64(&pc.hitCount, 1)
	// 使用已注册的计数器
	pc.hitCounter.Inc()
}

// IncMisses 增加未命中计数
func (pc *PrometheusCollector) IncMisses() {
	atomic.AddUint64(&pc.missCount, 1)
	// 使用已注册的计数器
	pc.missCounter.Inc()
}

// IncRequests 增加请求计数
func (pc *PrometheusCollector) IncRequests(operation string) {
	pc.requestCounter.WithLabelValues(operation).Inc()
}

// ObserveLatency 记录操作延迟
func (pc *PrometheusCollector) ObserveLatency(operation string, duration time.Duration) {
	pc.latencyHistogram.WithLabelValues(operation).Observe(duration.Seconds())
}

// SetMemoryUsage 设置内存使用量
func (pc *PrometheusCollector) SetMemoryUsage(bytes int64) {
	pc.memoryGauge.Set(float64(bytes))
}

// SetItemCount 设置条目数量
func (pc *PrometheusCollector) SetItemCount(count int64) {
	pc.itemsGauge.Set(float64(count))
}

// HitRate 获取命中率
func (pc *PrometheusCollector) HitRate() float64 {
	hits := atomic.LoadUint64(&pc.hitCount)
	misses := atomic.LoadUint64(&pc.missCount)

	total := hits + misses
	if total == 0 {
		return 0.0
	}

	return float64(hits) / float64(total)
}

// GetRegistry 获取 Prometheus 注册表
func (pc *PrometheusCollector) GetRegistry() *prometheus.Registry {
	return pc.registry
}

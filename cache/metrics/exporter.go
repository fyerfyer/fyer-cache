package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusExporter 将收集的指标导出为 Prometheus 格式
type PrometheusExporter struct {
	collector *PrometheusCollector
	registry  *prometheus.Registry
}

// NewPrometheusExporter 创建 Prometheus 指标导出器
func NewPrometheusExporter(collector *PrometheusCollector) *PrometheusExporter {
	return &PrometheusExporter{
		collector: collector,
		registry:  collector.GetRegistry(),
	}
}

// Export 将指标导出到 HTTP 响应
func (pe *PrometheusExporter) Export(w http.ResponseWriter) {
	// 创建一个有效的HTTP请求对象
	req, _ := http.NewRequest("GET", "/metrics", nil)
	h := promhttp.HandlerFor(pe.registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, req)
}


// RegisterMetric 注册新的指标
func (pe *PrometheusExporter) RegisterMetric(name, help string, typ MetricType, labels ...string) {
	switch typ {
	case TypeCounter:
		counter := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: pe.collector.namespace,
				Subsystem: pe.collector.subsystem,
				Name:      name,
				Help:      help,
			},
			labels,
		)
		pe.registry.MustRegister(counter)
	case TypeGauge:
		gauge := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: pe.collector.namespace,
				Subsystem: pe.collector.subsystem,
				Name:      name,
				Help:      help,
			},
			labels,
		)
		pe.registry.MustRegister(gauge)
	case TypeHistogram:
		histogram := prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: pe.collector.namespace,
				Subsystem: pe.collector.subsystem,
				Name:      name,
				Help:      help,
			},
			labels,
		)
		pe.registry.MustRegister(histogram)
	case TypeSummary:
		summary := prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace: pe.collector.namespace,
				Subsystem: pe.collector.subsystem,
				Name:      name,
				Help:      help,
			},
			labels,
		)
		pe.registry.MustRegister(summary)
	}
}
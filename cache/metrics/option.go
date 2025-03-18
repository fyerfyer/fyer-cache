package metrics

import (
	"time"
)

// MonitorOption 定义监控缓存的配置选项
type MonitorOption func(*MonitoredCache)

// WithMetricsServer 配置指标HTTP服务器
// addr 格式为 "host:port", 如 ":8080" 表示监听所有接口的8080端口
func WithMetricsServer(addr string) MonitorOption {
	return func(mc *MonitoredCache) {
		config := DefaultMetricsConfig()
		config.Address = addr
		mc.server = NewServer(mc.exporter, config)
	}
}

// WithMetricsPath 设置指标路径
// 默认为 "/metrics"
func WithMetricsPath(path string) MonitorOption {
	return func(mc *MonitoredCache) {
		if mc.server != nil {
			mc.server.config.MetricsPath = path
		} else {
			config := DefaultMetricsConfig()
			config.MetricsPath = path
			mc.server = NewServer(mc.exporter, config)
		}
	}
}

// WithCollectInterval 设置指标收集间隔
func WithCollectInterval(interval time.Duration) MonitorOption {
	return func(mc *MonitoredCache) {
		if interval > 0 {
			mc.collectInterval = interval
		}
	}
}

// WithMemoryUpdateInterval 设置内存使用量统计更新间隔
func WithMemoryUpdateInterval(interval time.Duration) MonitorOption {
	return func(mc *MonitoredCache) {
		if interval > 0 {
			mc.memoryUpdateInterval = interval
		}
	}
}

// WithNamespace 设置指标命名空间
func WithNamespace(namespace string) MonitorOption {
	return func(mc *MonitoredCache) {
		if namespace != "" {
			mc.namespace = namespace
		}
	}
}

// WithSubsystem 设置指标子系统名称
func WithSubsystem(subsystem string) MonitorOption {
	return func(mc *MonitoredCache) {
		if subsystem != "" {
			mc.subsystem = subsystem
		}
	}
}

// WithEnableMemStats 启用或禁用内存统计
func WithEnableMemStats(enable bool) MonitorOption {
	return func(mc *MonitoredCache) {
		mc.enableMemStats = enable
	}
}

// WithLabels 添加自定义标签
func WithLabels(labels map[string]string) MonitorOption {
	return func(mc *MonitoredCache) {
		if mc.labels == nil {
			mc.labels = make(map[string]string)
		}

		// 复制标签以避免外部引用修改
		for k, v := range labels {
			mc.labels[k] = v
		}
	}
}
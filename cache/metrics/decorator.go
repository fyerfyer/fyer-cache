package metrics

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/fyerfyer/fyer-cache/internal/ferr"
)

// MonitoredCache 用于包装缓存实例，添加监控功能
type MonitoredCache struct {
	cache             cache.Cache  // 原始缓存
	collector         Collector    // 指标收集器
	exporter          Exporter     // 指标导出器
	server            *Server      // HTTP服务器
	namespace         string       // 指标命名空间
	subsystem         string       // 指标子系统
	collectInterval   time.Duration // 指标收集间隔
	memoryUpdateInterval time.Duration // 内存统计更新间隔
	enableMemStats    bool         // 是否启用内存统计
	stopCh            chan struct{} // 停止信号通道
	labels            map[string]string // 自定义标签
	running           bool         // 是否运行中
	mu                sync.Mutex   // 互斥锁
}

// NewMonitoredCache 创建一个带监控功能的缓存装饰器
func NewMonitoredCache(originalCache cache.Cache, options ...MonitorOption) *MonitoredCache {
	// 创建默认配置
	config := DefaultMetricsConfig()

	mc := &MonitoredCache{
		cache:               originalCache,
		namespace:           config.Namespace,
		subsystem:           config.Subsystem,
		collectInterval:     config.CollectInterval,
		memoryUpdateInterval: config.MemStatsInterval,
		enableMemStats:      config.EnableMemStats,
		stopCh:              make(chan struct{}),
		labels:              make(map[string]string),
	}

	// 创建收集器
	collector := NewPrometheusCollector(config)
	mc.collector = collector

	// 创建导出器
	exporter := NewPrometheusExporter(collector)
	mc.exporter = exporter

	// 创建HTTP服务器
	mc.server = NewServer(exporter, config)

	// 应用选项
	for _, opt := range options {
		opt(mc)
	}

	// 启动后台指标收集
	go mc.collectMetrics()

	// 如果启用了内存监控，启动内存统计收集
	if mc.enableMemStats {
		go mc.collectMemStats()
	}

	// 启动服务器
	if mc.server != nil {
		err := mc.server.Start()
		if err != nil {
			// 实际应用中应该使用日志记录错误
		}
	}

	mc.running = true
	return mc
}

// Stop 停止监控
func (mc *MonitoredCache) Stop() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.running {
		return nil
	}

	// 停止HTTP服务器
	if mc.server != nil && mc.server.IsRunning() {
		if err := mc.server.Stop(); err != nil {
			return err
		}
	}

	// 发送停止信号
	close(mc.stopCh)
	mc.running = false
	return nil
}

// collectMetrics 周期性收集指标
func (mc *MonitoredCache) collectMetrics() {
	ticker := time.NewTicker(mc.collectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 更新指标
			count := mc.getItemCount()
			mc.collector.SetItemCount(count)
		case <-mc.stopCh:
			return
		}
	}
}

// collectMemStats 收集内存统计信息
func (mc *MonitoredCache) collectMemStats() {
	ticker := time.NewTicker(mc.memoryUpdateInterval)
	defer ticker.Stop()

	var memStats runtime.MemStats

	for {
		select {
		case <-ticker.C:
			// 获取内存统计
			runtime.ReadMemStats(&memStats)
			mc.collector.SetMemoryUsage(int64(memStats.Alloc))
		case <-mc.stopCh:
			return
		}
	}
}

// getItemCount 尝试获取缓存条目数量
func (mc *MonitoredCache) getItemCount() int64 {
	// 检查缓存是否实现了统计接口
	type counter interface {
		Count() int64
	}

	if c, ok := mc.cache.(counter); ok {
		return c.Count()
	}

	// 无法获取条目数量时返回0
	return 0
}

// Get 获取缓存键值，同时收集指标
func (mc *MonitoredCache) Get(ctx context.Context, key string) (any, error) {
	startTime := time.Now()
	value, err := mc.cache.Get(ctx, key)
	duration := time.Since(startTime)

	// 记录请求计数
	mc.collector.IncRequests("get")

	// 记录延迟
	mc.collector.ObserveLatency("get", duration)

	// 记录命中/未命中
	if err == nil {
		mc.collector.IncHits()
	} else if errors.Is(err, ferr.ErrKeyNotFound) {
		mc.collector.IncMisses()
	}

	return value, err
}

// Set 设置缓存键值，同时收集指标
func (mc *MonitoredCache) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	startTime := time.Now()
	err := mc.cache.Set(ctx, key, value, expiration)
	duration := time.Since(startTime)

	// 记录请求计数
	mc.collector.IncRequests("set")

	// 记录延迟
	mc.collector.ObserveLatency("set", duration)

	return err
}

// Del 删除缓存键值，同时收集指标
func (mc *MonitoredCache) Del(ctx context.Context, key string) error {
	startTime := time.Now()
	err := mc.cache.Del(ctx, key)
	duration := time.Since(startTime)

	// 记录请求计数
	mc.collector.IncRequests("del")

	// 记录延迟
	mc.collector.ObserveLatency("del", duration)

	return err
}

// HitRate 获取当前缓存命中率
func (mc *MonitoredCache) HitRate() float64 {
	return mc.collector.HitRate()
}
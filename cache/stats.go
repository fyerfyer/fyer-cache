package cache

import (
	"sync/atomic"
)

// StatsCollector 缓存统计信息收集器
type StatsCollector struct {
	hits      int64
	misses    int64
	evictions int64
}

// NewStatsCollector 创建统计收集器
func NewStatsCollector() *StatsCollector {
	return &StatsCollector{}
}

// RecordHit 记录缓存命中
func (sc *StatsCollector) RecordHit() {
	atomic.AddInt64(&sc.hits, 1)
}

// RecordMiss 记录缓存未命中
func (sc *StatsCollector) RecordMiss() {
	atomic.AddInt64(&sc.misses, 1)
}

// RecordEviction 记录缓存淘汰
func (sc *StatsCollector) RecordEviction() {
	atomic.AddInt64(&sc.evictions, 1)
}

// Hits 获取命中次数
func (sc *StatsCollector) Hits() int64 {
	return atomic.LoadInt64(&sc.hits)
}

// Misses 获取未命中次数
func (sc *StatsCollector) Misses() int64 {
	return atomic.LoadInt64(&sc.misses)
}

// Evictions 获取淘汰次数
func (sc *StatsCollector) Evictions() int64 {
	return atomic.LoadInt64(&sc.evictions)
}

// HitRate 获取命中率
func (sc *StatsCollector) HitRate() float64 {
	hits := atomic.LoadInt64(&sc.hits)
	misses := atomic.LoadInt64(&sc.misses)
	total := hits + misses

	if total == 0 {
		return 0
	}

	return float64(hits) / float64(total)
}

// MissRate 获取未命中率
func (sc *StatsCollector) MissRate() float64 {
	hits := atomic.LoadInt64(&sc.hits)
	misses := atomic.LoadInt64(&sc.misses)
	total := hits + misses

	if total == 0 {
		return 0
	}

	return float64(misses) / float64(total)
}
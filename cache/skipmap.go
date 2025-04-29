package cache

import (
	"hash/fnv"
	"math"
	"sync"
	"sync/atomic"
)

// SkipShard 单个跳表分片
type SkipShard struct {
	items *SkipList
	mu    sync.RWMutex  // 每个分片的读写锁
}

// SkipShardedMap 基于跳表的分片映射
type SkipShardedMap struct {
	shards    []*SkipShard
	count     int     // 分片数量
	mask      uint32  // 用于位运算快速定位分片
	itemCount int64   // 总项目数量
}

// NewSkipShardedMap 创建一个新的基于跳表的分片映射
func NewSkipShardedMap(shardCount int) *SkipShardedMap {
	if shardCount <= 0 {
		shardCount = DefaultShardCount
	}

	// 确保分片数量是2的幂
	if !isPowerOfTwo(shardCount) {
		// 向上取整为2的幂
		shardCount = int(math.Pow(2, math.Ceil(math.Log2(float64(shardCount)))))
	}

	sm := &SkipShardedMap{
		shards: make([]*SkipShard, shardCount),
		count:  shardCount,
		mask:   uint32(shardCount - 1), // 用于位运算
	}

	// 初始化每个分片
	for i := 0; i < shardCount; i++ {
		sm.shards[i] = &SkipShard{
			items: newSkipList(),
		}
	}

	return sm
}

// getShard 根据键获取对应的分片
func (sm *SkipShardedMap) getShard(key string) *SkipShard {
	hasher := fnv.New32()
	hasher.Write([]byte(key))
	hash := hasher.Sum32()

	// 使用位掩码快速计算分片索引
	index := hash & sm.mask
	return sm.shards[index]
}

// Store 存储键值对
func (sm *SkipShardedMap) Store(key string, value *cacheItem) {
	shard := sm.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	// 检查键是否已存在
	_, exists := shard.items.Get(key)

	// 存储数据
	shard.items.Set(key, value)

	// 如果键不存在，计数增加
	if !exists {
		atomic.AddInt64(&sm.itemCount, 1)
	}
}

// Load 加载指定键的值
func (sm *SkipShardedMap) Load(key string) (*cacheItem, bool) {
	shard := sm.getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	return shard.items.Get(key)
}

// Delete 删除指定键
func (sm *SkipShardedMap) Delete(key string) {
	shard := sm.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	// 如果成功删除，减少计数
	if shard.items.Delete(key) {
		atomic.AddInt64(&sm.itemCount, -1)
	}
}

// LoadAndDelete 加载并删除键值对
func (sm *SkipShardedMap) LoadAndDelete(key string) (*cacheItem, bool) {
	shard := sm.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	// 先获取值
	value, exists := shard.items.Get(key)
	if !exists {
		return nil, false
	}

	// 如果存在，删除并减少计数
	shard.items.Delete(key)
	atomic.AddInt64(&sm.itemCount, -1)

	return value, true
}

// Range 遍历所有键值对
func (sm *SkipShardedMap) Range(f func(key string, value *cacheItem) bool) {
	// 遍历每个分片
	for i := 0; i < sm.count; i++ {
		shard := sm.shards[i]

		// 创建快照
		var entries []struct {
			key   string
			value *cacheItem
		}
		
		// 获取读锁并复制数据
		shard.mu.RLock()
		shard.items.ForEach(func(key string, value *cacheItem) bool {
			entries = append(entries, struct {
				key   string
				value *cacheItem
			}{key: key, value: value})
			return true
		})
		shard.mu.RUnlock()

		// 处理快照中的数据（不持有锁）
		for _, entry := range entries {
			if !f(entry.key, entry.value) {
				return
			}
		}
	}
}

// RangeShard 遍历特定分片的键值对
func (sm *SkipShardedMap) RangeShard(shardIndex int, f func(key string, value *cacheItem) bool) {
	if shardIndex < 0 || shardIndex >= sm.count {
		return
	}

	shard := sm.shards[shardIndex]
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	shard.items.ForEach(func(key string, value *cacheItem) bool {
		return f(key, value)
	})
}

// RangeKeys 范围查询，获取指定范围内的键值对
func (sm *SkipShardedMap) RangeKeys(startKey, endKey string, limit int, f func(key string, value *cacheItem) bool) {
	// 如果是空范围，则返回
	if startKey > endKey && endKey != "" {
		return
	}

	// 遍历所有分片
	count := 0
	for i := 0; i < sm.count && (limit <= 0 || count < limit); i++ {
		shard := sm.shards[i]

		shard.mu.RLock()
		// 在分片内使用跳表的范围查询
		shard.items.Range(startKey, endKey, limit - count, func(key string, value *cacheItem) bool {
			// 释放锁调用回调
			shard.mu.RUnlock()

			result := f(key, value)

			shard.mu.RLock()
			count++
			return result && (limit <= 0 || count < limit)
		})
		shard.mu.RUnlock()

		// 如果已经达到限制或回调返回false，停止遍历
		if limit > 0 && count >= limit {
			break
		}
	}
}

// Count 返回所有分片中的项目总数
func (sm *SkipShardedMap) Count() int64 {
	return atomic.LoadInt64(&sm.itemCount)
}

// ShardCount 返回分片数量
func (sm *SkipShardedMap) ShardCount() int {
	return sm.count
}
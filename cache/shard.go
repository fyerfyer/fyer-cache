package cache

import (
	"hash/fnv"
	"math"
	"sync"
	"sync/atomic"
)

// DefaultShardCount 默认分片数量
// 设置为 32 作为默认值，是 CPU 核心数的倍数，且为 2 的幂以便哈希计算
const DefaultShardCount = 32

// ShardedMap 分片映射结构
// 将数据分散到多个分片中，每个分片有独立的锁，减少锁竞争
type ShardedMap struct {
	shards    []*Shard
	count     int    // 分片数量
	mask      uint32 // 用于位运算快速定位分片
	itemCount int64  // 总项目数量
}

// Shard 单个分片
type Shard struct {
	items map[string]*cacheItem
	mu    sync.RWMutex // 每个分片有独立的读写锁
}

// NewShardedMap 创建一个新的分片映射
// shardCount 必须是 2 的幂，便于使用位掩码快速计算分片索引
func NewShardedMap(shardCount int) *ShardedMap {
	if shardCount <= 0 {
		shardCount = DefaultShardCount
	}

	// 确保分片数量是 2 的幂
	if !isPowerOfTwo(shardCount) {
		// 向上取整为 2 的幂
		shardCount = int(math.Pow(2, math.Ceil(math.Log2(float64(shardCount)))))
	}

	sm := &ShardedMap{
		shards: make([]*Shard, shardCount),
		count:  shardCount,
		mask:   uint32(shardCount - 1), // 用于位运算
	}

	// 初始化每个分片
	for i := 0; i < shardCount; i++ {
		sm.shards[i] = &Shard{
			items: make(map[string]*cacheItem),
		}
	}

	return sm
}

// isPowerOfTwo 检查一个数是否为2的幂
func isPowerOfTwo(n int) bool {
	return n > 0 && (n&(n-1)) == 0
}

// getShard 根据键获取对应的分片
func (sm *ShardedMap) getShard(key string) *Shard {
	hasher := fnv.New32()
	hasher.Write([]byte(key))
	hash := hasher.Sum32()

	// 使用位掩码快速计算分片索引
	index := hash & sm.mask
	return sm.shards[index]
}

// Store 存储键值对
func (sm *ShardedMap) Store(key string, value *cacheItem) {
	shard := sm.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	// 如果键不存在，计数增加
	if _, exists := shard.items[key]; !exists {
		atomic.AddInt64(&sm.itemCount, 1)
	}

	shard.items[key] = value
}

// Load 加载指定键的值
func (sm *ShardedMap) Load(key string) (*cacheItem, bool) {
	shard := sm.getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	value, ok := shard.items[key]
	return value, ok
}

// LoadAndDelete 加载键值并删除
func (sm *ShardedMap) LoadAndDelete(key string) (*cacheItem, bool) {
	shard := sm.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	value, ok := shard.items[key]
	if ok {
		delete(shard.items, key)
		atomic.AddInt64(&sm.itemCount, -1)
	}

	return value, ok
}

// Delete 删除指定键
func (sm *ShardedMap) Delete(key string) {
	shard := sm.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if _, ok := shard.items[key]; ok {
		delete(shard.items, key)
		atomic.AddInt64(&sm.itemCount, -1)
	}
}

// Range 遍历所有键值对
// 注意: 由于分片锁的性质，Range 方法在遍历过程中不能保证整体一致性
func (sm *ShardedMap) Range(f func(key string, value *cacheItem) bool) {
	// 遍历每个分片
	for i := 0; i < sm.count; i++ {
		shard := sm.shards[i]

		// 为每个分片创建一个临时快照，避免长时间持有锁
		var tempMap map[string]*cacheItem

		// 获取读锁并复制分片内容
		shard.mu.RLock()
		tempMap = make(map[string]*cacheItem, len(shard.items))
		for k, v := range shard.items {
			tempMap[k] = v
		}
		shard.mu.RUnlock()

		// 遍历临时快照
		for k, v := range tempMap {
			if !f(k, v) {
				return
			}
		}
	}
}

// RangeShard 遍历特定分片的键值对
// 这个方法对于异步清理非常有用，可以只处理单个分片
func (sm *ShardedMap) RangeShard(shardIndex int, f func(key string, value *cacheItem) bool) {
	if shardIndex < 0 || shardIndex >= sm.count {
		return
	}

	shard := sm.shards[shardIndex]

	// 创建临时快照
	var tempMap map[string]*cacheItem

	shard.mu.RLock()
	tempMap = make(map[string]*cacheItem, len(shard.items))
	for k, v := range shard.items {
		tempMap[k] = v
	}
	shard.mu.RUnlock()

	// 遍历临时快照
	for k, v := range tempMap {
		if !f(k, v) {
			return
		}
	}
}

// Count 返回所有分片中的项目总数
func (sm *ShardedMap) Count() int64 {
	return atomic.LoadInt64(&sm.itemCount)
}

// ShardCount 返回分片数量
func (sm *ShardedMap) ShardCount() int {
	return sm.count
}

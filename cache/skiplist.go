package cache

import (
	"math/rand"
	"sync"
	"time"
)

const (
	maxLevel    = 32  // 跳表最大层数
	probability = 0.25 // 提升到上一层的概率
)

// skipNode 跳表节点
type skipNode struct {
	key        string       // 键
	value      *cacheItem   // 值
	forward    []*skipNode  // 前向指针
}

// SkipList 跳表实现
type SkipList struct {
	head       *skipNode     // 头节点
	length     int           // 长度
	level      int           // 当前层数
	random     *rand.Rand    // 随机数生成器
	mu         sync.RWMutex  // 读写锁
}

// newSkipList 创建新的跳表
func newSkipList() *SkipList {
	return &SkipList{
		head:   &skipNode{forward: make([]*skipNode, maxLevel)},
		level:  1,
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// randomLevel 随机生成层数
func (sl *SkipList) randomLevel() int {
	level := 1
	for level < maxLevel && sl.random.Float64() < probability {
		level++
	}
	return level
}

// Set 设置键值对
func (sl *SkipList) Set(key string, value *cacheItem) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	// 查找待插入的位置
	update := make([]*skipNode, maxLevel)
	current := sl.head

	// 从最高层开始向下查找
	for i := sl.level - 1; i >= 0; i-- {
		for current.forward[i] != nil && current.forward[i].key < key {
			current = current.forward[i]
		}
		update[i] = current
	}

	// 移至第0层，判断是否已存在相同key的节点
	current = current.forward[0]

	// 如果是更新已存在的节点
	if current != nil && current.key == key {
		current.value = value
		return
	}

	// 生成随机层数
	level := sl.randomLevel()

	// 如果新层数大于当前层数
	if level > sl.level {
		for i := sl.level; i < level; i++ {
			update[i] = sl.head
		}
		sl.level = level
	}

	// 创建新节点
	newNode := &skipNode{
		key:     key,
		value:   value,
		forward: make([]*skipNode, level),
	}

	// 更新指针
	for i := 0; i < level; i++ {
		newNode.forward[i] = update[i].forward[i]
		update[i].forward[i] = newNode
	}

	sl.length++
}

// Get 获取键对应的值
func (sl *SkipList) Get(key string) (*cacheItem, bool) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	current := sl.head

	// 从最高层开始查找
	for i := sl.level - 1; i >= 0; i-- {
		for current.forward[i] != nil && current.forward[i].key < key {
			current = current.forward[i]
		}
	}

	// 移至第0层的下一个节点
	current = current.forward[0]

	// 判断是否找到
	if current != nil && current.key == key {
		return current.value, true
	}

	return nil, false
}

// Delete 删除键值对
func (sl *SkipList) Delete(key string) bool {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	// 查找要删除的节点位置
	update := make([]*skipNode, maxLevel)
	current := sl.head

	// 从最高层开始查找
	for i := sl.level - 1; i >= 0; i-- {
		for current.forward[i] != nil && current.forward[i].key < key {
			current = current.forward[i]
		}
		update[i] = current
	}

	// 移至第0层的下一个节点
	current = current.forward[0]

	// 如果没找到要删除的节点
	if current == nil || current.key != key {
		return false
	}

	// 更新前向指针，移除当前节点
	for i := 0; i < sl.level && update[i].forward[i] == current; i++ {
		update[i].forward[i] = current.forward[i]
	}

	// 如果删除的是最高层节点，可能需要降低level
	for sl.level > 1 && sl.head.forward[sl.level-1] == nil {
		sl.level--
	}

	sl.length--
	return true
}

// Range 范围查询从startKey到endKey之间的元素
func (sl *SkipList) Range(startKey, endKey string, limit int, visit func(key string, value *cacheItem) bool) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	// 从head开始
	current := sl.head

	// 找到>=startKey的第一个节点
	for i := sl.level - 1; i >= 0; i-- {
		for current.forward[i] != nil && current.forward[i].key < startKey {
			current = current.forward[i]
		}
	}

	// 移至第0层的下一个节点(>=startKey的第一个节点)
	current = current.forward[0]

	// 遍历范围内的节点
	count := 0
	for current != nil && (endKey == "" || current.key <= endKey) && (limit <= 0 || count < limit) {
		// 如果访问函数返回false，停止遍历
		if !visit(current.key, current.value) {
			break
		}
		current = current.forward[0]
		count++
	}
}

// Len 返回跳表长度
func (sl *SkipList) Len() int {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.length
}

// ForEach 遍历所有元素
func (sl *SkipList) ForEach(visit func(key string, value *cacheItem) bool) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	current := sl.head.forward[0]
	for current != nil {
		if !visit(current.key, current.value) {
			break
		}
		current = current.forward[0]
	}
}
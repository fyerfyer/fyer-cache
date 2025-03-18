package cache

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

// EvictAlgorithm 淘汰算法接口
type EvictAlgorithm interface {
	// Evict 淘汰算法
	Evict(c *MemCacheMemory) string
}

// LRUEviction LRU淘汰算法
type LRUEviction struct {
	accessList *doubleLinkedList
	mu         sync.RWMutex // 添加锁保护访问列表
}

// MemCacheMemory 缓存内存结构
type MemCacheMemory struct {
	*MemoryCache
	maxMemroy  int64
	usedMemory int64
	evict      EvictAlgorithm
	accessList *doubleLinkedList
	mu         sync.RWMutex // 添加锁保护内存管理相关操作
}

// doubleLinkedList 双向链表
type doubleLinkedList struct {
	head, tail *listNode
	nodeMap    map[string]*listNode // 添加映射以提高查找效率
	mu         sync.RWMutex         // 添加锁保护链表操作
}

// listNode 链表节点
type listNode struct {
	key        string
	accesstime time.Time
	prev, next *listNode
}

// NewMemCacheMemory 创建带内存淘汰的内存缓存实例
func NewMemCacheMemory(c *MemoryCache, maxMemory int64) *MemCacheMemory {
	// 先创建共享的访问列表
	sharedList := newDoubleLinkedList()

	mcm := &MemCacheMemory{
		MemoryCache: c,
		maxMemroy:   maxMemory,
		evict: &LRUEviction{
			accessList: sharedList, // 使用共享列表
		},
		accessList: sharedList, // 使用相同的共享列表
	}

	// 包装原有evict回调
	origin := c.onEvict
	c.onEvict = func(key string, value any) {
		size := mcm.estimateObjectSize(value)
		atomic.AddInt64(&mcm.usedMemory, -size)
		mcm.accessList.del(key)
		if origin != nil {
			origin(key, value)
		}
	}

	return mcm
}

// estimateObjectSize 估算对象大小
func (c *MemCacheMemory) estimateObjectSize(obj any) int64 {
	return sizeOf(reflect.ValueOf(obj))
}

// sizeOf 递归计算对象大小 - 保持原有逻辑
func sizeOf(v reflect.Value) int64 {
	switch v.Kind() {
	case reflect.Bool:
		return 1
	case reflect.Int8, reflect.Uint8:
		return 1
	case reflect.Int16, reflect.Uint16:
		return 2
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		return 4
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		return 8
	case reflect.Complex64:
		return 8
	case reflect.Complex128:
		return 16

	case reflect.Ptr:
		if v.IsNil() {
			return 8
		}
		return 8 + sizeOf(v.Elem())

	case reflect.String:
		// string 头部占用 16 字节(指针+长度)，再加上字符串实际数据
		return int64(16 + len(v.String()))

	case reflect.Slice:
		// slice 头部占用 24 字节(指针+长度+容量)
		size := int64(24)
		if v.Len() > 0 {
			// 对于基础类型的切片，直接计算总大小
			if v.Type().Elem().Kind() == reflect.Uint8 {
				size += int64(v.Len())
			} else {
				elementSize := sizeOf(v.Index(0))
				size += elementSize * int64(v.Len())
			}
		}
		return size

	case reflect.Array:
		if v.Len() == 0 {
			return 0
		}
		elementSize := sizeOf(v.Index(0))
		return elementSize * int64(v.Len())

	case reflect.Map:
		// map 头部估算为 8 字节
		size := int64(8)
		keys := v.MapKeys()
		for _, key := range keys {
			size += sizeOf(key) + sizeOf(v.MapIndex(key))
		}
		return size

	case reflect.Struct:
		size := int64(0)
		for i := 0; i < v.NumField(); i++ {
			size += sizeOf(v.Field(i))
		}
		// 考虑内存对齐，向上取整到 8 的倍数
		if size%8 != 0 {
			size = (size/8 + 1) * 8
		}
		return size

	default:
		// 其他类型保守估计为 8 字节
		return 8
	}
}

// Set 实现缓存Set方法，添加内存管理
func (c *MemCacheMemory) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
	size := c.estimateObjectSize(val)

	// 获取互斥锁进行内存管理操作
	c.mu.Lock()
	// 检查是否需要末位淘汰
	for atomic.LoadInt64(&c.usedMemory)+size > c.maxMemroy {
		evictKey := c.evict.Evict(c)
		if evictKey == "" {
			break
		}

		// 释放锁，以免在Del操作时造成死锁
		c.mu.Unlock()
		c.Del(ctx, evictKey)
		c.mu.Lock()
	}

	// 更新内存用量
	atomic.AddInt64(&c.usedMemory, size)
	c.mu.Unlock()

	// 更新访问记录
	c.accessList.add(key)

	// 调用底层Set方法
	return c.MemoryCache.Set(ctx, key, val, expiration)
}

// Get 实现缓存Get方法，更新访问记录
func (c *MemCacheMemory) Get(ctx context.Context, key string) (any, error) {
	val, err := c.MemoryCache.Get(ctx, key)
	if err == nil {
		c.accessList.access(key)
	}

	return val, err
}

// Del 实现缓存Del方法，更新访问记录
func (c *MemCacheMemory) Del(ctx context.Context, key string) error {
	err := c.MemoryCache.Del(ctx, key)
	if err == nil {
		c.accessList.del(key)
	}

	return err
}

// newDoubleLinkedList 创建新的双向链表
func newDoubleLinkedList() *doubleLinkedList {
	dl := &doubleLinkedList{
		head:    &listNode{},
		tail:    &listNode{},
		nodeMap: make(map[string]*listNode), // 初始化映射
	}

	// 确保头尾节点正确连接
	dl.head.next = dl.tail
	dl.tail.prev = dl.head

	return dl
}

// add 添加节点到链表头部
func (dl *doubleLinkedList) add(key string) {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	// 如果节点已存在，先删除
	if node, exists := dl.nodeMap[key]; exists {
		node.prev.next = node.next
		node.next.prev = node.prev
	}

	// 创建新节点
	node := &listNode{
		key:        key,
		accesstime: time.Now(),
	}

	// 添加到链表头部
	if dl.head.next == nil {
		dl.head.next = node
		dl.tail.prev = node
		node.prev = dl.head
		node.next = dl.tail
	} else {
		node.next = dl.head.next
		node.prev = dl.head
		dl.head.next.prev = node
		dl.head.next = node
	}

	// 更新映射
	dl.nodeMap[key] = node
}

// access 更新节点访问时间并移动到头部
func (dl *doubleLinkedList) access(key string) {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	// 使用映射直接查找节点
	if node, exists := dl.nodeMap[key]; exists {
		// 从当前位置移除
		node.prev.next = node.next
		node.next.prev = node.prev

		// 添加到头部
		node.next = dl.head.next
		node.prev = dl.head
		dl.head.next.prev = node
		dl.head.next = node
		node.accesstime = time.Now()
	}
}

// del 从链表中删除节点
func (dl *doubleLinkedList) del(key string) {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	// 使用映射直接查找节点
	if node, exists := dl.nodeMap[key]; exists {
		node.prev.next = node.next
		node.next.prev = node.prev
		delete(dl.nodeMap, key) // 从映射中删除
	}
}

// Evict LRU算法实现
func (lru *LRUEviction) Evict(c *MemCacheMemory) string {
	lru.mu.RLock()
	defer lru.mu.RUnlock()

	if lru.accessList.tail.prev == lru.accessList.head {
		return ""
	}

	lruKey := lru.accessList.tail.prev.key
	return lruKey
}

// MemoryUsage 获取内存使用情况（字节）
func (c *MemCacheMemory) MemoryUsage() int64 {
	return atomic.LoadInt64(&c.usedMemory)
}

// ItemCount 获取缓存项目数量
func (c *MemCacheMemory) ItemCount() int64 {
	return int64(c.MemoryCache.data.Count())
}

// HitRate 获取缓存命中率
func (c *MemCacheMemory) HitRate() float64 {
	return c.MemoryCache.HitRate()
}

// MissRate 获取缓存未命中率
func (c *MemCacheMemory) MissRate() float64 {
	return c.MemoryCache.MissRate()
}
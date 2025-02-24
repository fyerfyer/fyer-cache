package cache

import (
	"context"
	"reflect"
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
}

// MemCacheMemory 缓存内存结构
type MemCacheMemory struct {
	*MemoryCache
	maxMemroy  int64
	usedMemory int64
	evict      EvictAlgorithm
	accessList *doubleLinkedList
}

// doubleLinkedList 双向链表
type doubleLinkedList struct {
	head, tail *listNode
}

// listNode 链表节点
type listNode struct {
	key        string
	accesstime time.Time
	prev, next *listNode
}

// NewMemCacheMemory 创建带内存淘汰的内存缓存实例
func NewMemCacheMemory(c *MemoryCache, maxMemory int64) *MemCacheMemory {
	mcm := &MemCacheMemory{
		MemoryCache: c,
		maxMemroy:   maxMemory,
		evict: &LRUEviction{
			accessList: newDoubleLinkedList(),
		},
		accessList: newDoubleLinkedList(),
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

// sizeOf 递归计算对象大小
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

func (c *MemCacheMemory) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
	size := c.estimateObjectSize(val)

	// 检查是否需要末位淘汰
	for atomic.LoadInt64(&c.usedMemory)+size > c.maxMemroy {
		evictKey := c.evict.Evict(c)
		if evictKey == "" {
			break
		}

		c.Del(ctx, evictKey)
	}

	// 更新内存用量
	atomic.AddInt64(&c.usedMemory, size)
	c.accessList.add(key)
	return c.MemoryCache.Set(ctx, key, val, expiration)
}

func (c *MemCacheMemory) Get(ctx context.Context, key string) (any, error) {
	val, err := c.MemoryCache.Get(ctx, key)
	if err == nil {
		c.accessList.access(key)
	}

	return val, err
}

func (c *MemCacheMemory) Del(ctx context.Context, key string) error {
	err := c.MemoryCache.Del(ctx, key)
	if err == nil {
		c.accessList.del(key)
	}

	return err
}

// newLRUEviction 创建LRU淘汰算法实例
func newDoubleLinkedList() *doubleLinkedList {
	return &doubleLinkedList{
		head: &listNode{},
		tail: &listNode{},
	}
}

func (dl *doubleLinkedList) add(key string) {
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
}

func (dl *doubleLinkedList) access(key string) {
	// 查找并移动到头部
	current := dl.head.next
	for current != dl.tail {
		if current.key == key {
			// 从当前位置移除
			current.prev.next = current.next
			current.next.prev = current.prev

			// 添加到头部
			current.next = dl.head.next
			current.prev = dl.head
			dl.head.next.prev = current
			dl.head.next = current
			current.accesstime = time.Now()
			break
		}
		current = current.next
	}
}

func (dl *doubleLinkedList) del(key string) {
	current := dl.head.next
	for current != dl.tail {
		if current.key == key {
			current.prev.next = current.next
			current.next.prev = current.prev
			break
		}
		current = current.next
	}
}

// Evict LRU算法实现
func (lru *LRUEviction) Evict(c *MemCacheMemory) string {
	if c.accessList.tail.prev == c.accessList.head {
		return ""
	}

	lruKey := c.accessList.tail.prev.key
	return lruKey
}

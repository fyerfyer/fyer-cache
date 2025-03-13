package cache

import (
	"hash/crc32"
	"sort"
	"strconv"
	"sync"
)

// HashFunc 定义哈希函数类型
type HashFunc func(data []byte) uint32

// ConsistentHash 一致性哈希环实现
type ConsistentHash struct {
	// 哈希函数
	hashFunc HashFunc

	// 每个物理节点对应的虚拟节点数量
	replicas int

	// 已排序的哈希值
	sortedHashes []uint32

	// 哈希值到节点的映射
	hashMap map[uint32]string

	// 节点权重
	weights map[string]int

	// 节点到其哈希值的映射，用于高效删除
	nodeHashes map[string][]uint32

	// 保护环的并发访问
	mu sync.RWMutex
}

// defaultReplicas 默认虚拟节点数量
const defaultReplicas = 100

// NewConsistentHash 创建一个新的一致性哈希环
func NewConsistentHash(replicas int, fn HashFunc) *ConsistentHash {
	if replicas <= 0 {
		replicas = defaultReplicas
	}

	if fn == nil {
		// 默认使用 crc32.ChecksumIEEE
		fn = crc32.ChecksumIEEE
	}

	return &ConsistentHash{
		hashFunc:     fn,
		replicas:     replicas,
		sortedHashes: make([]uint32, 0),
		hashMap:      make(map[uint32]string),
		weights:      make(map[string]int),
		nodeHashes:   make(map[string][]uint32),
	}
}

// Add 添加节点到哈希环
// weight 是节点的权重，影响该节点的虚拟节点数量
func (c *ConsistentHash) Add(node string, weight int) {
	if weight <= 0 {
		weight = 1
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 如果节点已存在，先删除它
	if _, ok := c.weights[node]; ok {
		c.removeWithoutLock(node)
	}

	// 记录节点权重
	c.weights[node] = weight

	// 为节点创建虚拟节点
	nodeHashes := make([]uint32, 0, c.replicas*weight)
	for i := 0; i < c.replicas*weight; i++ {
		// 为虚拟节点创建唯一标识
		virtualNode := node + ":" + strconv.Itoa(i)
		hash := c.hashFunc([]byte(virtualNode))

		// 添加哈希到排序列表
		c.sortedHashes = append(c.sortedHashes, hash)
		// 记录哈希到节点的映射
		c.hashMap[hash] = node
		// 记录节点的哈希值，用于删除
		nodeHashes = append(nodeHashes, hash)
	}

	// 存储节点的所有哈希值
	c.nodeHashes[node] = nodeHashes

	// 重新排序哈希值
	sort.Slice(c.sortedHashes, func(i, j int) bool {
		return c.sortedHashes[i] < c.sortedHashes[j]
	})
}

// Remove 从哈希环中移除节点
func (c *ConsistentHash) Remove(node string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.removeWithoutLock(node)
}

// removeWithoutLock 不加锁地移除节点（内部使用）
func (c *ConsistentHash) removeWithoutLock(node string) {
	// 查找节点的哈希值
	hashes, ok := c.nodeHashes[node]
	if !ok {
		return
	}

	// 从哈希表中删除
	for _, hash := range hashes {
		delete(c.hashMap, hash)
	}

	// 使用一个临时集合快速检查哈希是否需要被删除
	hashSet := make(map[uint32]struct{}, len(hashes))
	for _, hash := range hashes {
		hashSet[hash] = struct{}{}
	}

	// 创建新的排序列表，移除要删除的哈希
	newSortedHashes := make([]uint32, 0, len(c.sortedHashes)-len(hashSet))
	for _, hash := range c.sortedHashes {
		if _, exists := hashSet[hash]; !exists {
			newSortedHashes = append(newSortedHashes, hash)
		}
	}
	c.sortedHashes = newSortedHashes

	// 删除节点的权重记录
	delete(c.weights, node)
	// 删除节点的哈希记录
	delete(c.nodeHashes, node)
}

// Get 根据键获取对应的节点
func (c *ConsistentHash) Get(key string) string {
	if len(c.sortedHashes) == 0 {
		return ""
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// 计算键的哈希值
	hash := c.hashFunc([]byte(key))

	// 在排序的哈希环上查找第一个大于等于键哈希的位置
	idx := c.search(hash)

	// 返回对应的节点
	return c.hashMap[c.sortedHashes[idx]]
}

// GetN 获取键映射到的N个节点，用于备份
func (c *ConsistentHash) GetN(key string, count int) []string {
	if len(c.sortedHashes) == 0 {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.weights) <= 0 {
		return []string{}
	}

	// 计算键的哈希值
	hash := c.hashFunc([]byte(key))

	// 在排序的哈希环上查找第一个大于等于键哈希的位置
	idx := c.search(hash)

	// 收集唯一的节点
	uniqueNodes := make(map[string]struct{})
	result := make([]string, 0, count)

	// 从找到的位置开始，沿环顺时针收集节点
	for len(uniqueNodes) < count && len(uniqueNodes) < len(c.weights) {
		if idx >= len(c.sortedHashes) {
			idx = 0 // 防止索引越界
		}

		node := c.hashMap[c.sortedHashes[idx]]
		if _, exists := uniqueNodes[node]; !exists {
			uniqueNodes[node] = struct{}{}
			result = append(result, node)
		}
		idx = (idx + 1) % len(c.sortedHashes)
	}

	return result
}

// GetNodes 获取所有节点
func (c *ConsistentHash) GetNodes() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	nodes := make([]string, 0, len(c.weights))
	for node := range c.weights {
		nodes = append(nodes, node)
	}

	return nodes
}

// search 二分查找大于等于给定哈希值的第一个索引
// 如果所有哈希值都小于给定哈希，则返回第一个索引（环形）
func (c *ConsistentHash) search(hash uint32) int {
	// 二分查找
	n := len(c.sortedHashes)
	if n == 0 {
		return 0
	}

	idx := sort.Search(n, func(i int) bool {
		return c.sortedHashes[i] >= hash
	})

	// 如果没有找到大于等于哈希的值，回到环的起点
	if idx == n {
		idx = 0
	}

	return idx
}

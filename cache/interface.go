package cache

import (
	"context"
	"time"
)

// Cache 基本缓存接口
type Cache interface {
	Get(ctx context.Context, key string) (any, error)
	Set(ctx context.Context, key string, val any, expiration time.Duration) error
	Del(ctx context.Context, key string) error
}

//=================== 一致性哈希相关接口 ===================

// Ring 一致性哈希环接口
type Ring interface {
	// Add 添加节点
	Add(node string, weight int)

	// Remove 移除节点
	Remove(node string)

	// Get 获取键对应的节点
	Get(key string) string

	// GetN 获取键对应的N个节点（用于备份）
	GetN(key string, count int) []string

	// GetNodes 获取所有节点
	GetNodes() []string
}

// ShardedCache 分片缓存接口
type ShardedCache interface {
	Cache

	// AddNode 添加缓存节点
	AddNode(nodeID string, cache Cache, weight int) error

	// RemoveNode 移除缓存节点
	RemoveNode(nodeID string) error

	// GetNodeIDs 获取所有节点ID
	GetNodeIDs() []string

	// GetNodeCount 获取节点数量
	GetNodeCount() int
}

// DataSource 数据源接口
// 表示read through和write through中的数据来源
type DataSource interface {
	Load(ctx context.Context, key string) (any, error)
	Store(ctx context.Context, key string, val any) error
	Remove(ctx context.Context, key string) error
}

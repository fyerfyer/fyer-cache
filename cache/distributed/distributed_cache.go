package distributed

import (
	"context"
	"time"
)

// DistributedCache 是分布式缓存操作的接口
// 它扩展了基本的缓存操作，增加了对分布式环境的感知
type DistributedCache interface {
	// 核心缓存操作

	Get(ctx context.Context, key string) (any, error)
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
	Del(ctx context.Context, key string) error

	// 本地缓存操作，绕过分布式路由直接访问当前节点的数据

	GetLocal(ctx context.Context, key string) (any, error)
	SetLocal(ctx context.Context, key string, value any, expiration time.Duration) error
	DelLocal(ctx context.Context, key string) error

	// 节点管理

	NodeID() string
	NodeAddress() string
}

// CacheNode 表示分布式缓存中的一个节点
type CacheNode struct {
	ID      string
	Address string
	Weight  int
}

// DistributedCacheConfig 分布式缓存配置
type DistributedCacheConfig struct {
	// 当前节点信息
	NodeID  string
	Address string
	Weight  int

	// 复制因子 - 数据在几个节点上保存副本
	ReplicaFactor int

	// 一致性哈希环的虚拟节点数量
	VirtualNodes int

	// 操作超时时间
	OperationTimeout time.Duration

	// 重试配置
	MaxRetries    int
	RetryInterval time.Duration
}

// NewDistributedCacheConfig 返回默认的分布式缓存配置
func NewDistributedCacheConfig() *DistributedCacheConfig {
	return &DistributedCacheConfig{
		Weight:           1,
		ReplicaFactor:    1,
		VirtualNodes:     100,
		OperationTimeout: 2 * time.Second,
		MaxRetries:       3,
		RetryInterval:    100 * time.Millisecond,
	}
}

// Option 是分布式缓存的配置选项
type Option func(*DistributedCacheConfig)

// WithNodeID 设置节点ID
func WithNodeID(id string) Option {
	return func(cfg *DistributedCacheConfig) {
		cfg.NodeID = id
	}
}

// WithAddress 设置节点地址
func WithAddress(addr string) Option {
	return func(cfg *DistributedCacheConfig) {
		cfg.Address = addr
	}
}

// WithWeight 设置节点权重
func WithWeight(weight int) Option {
	return func(cfg *DistributedCacheConfig) {
		if weight > 0 {
			cfg.Weight = weight
		}
	}
}

// WithReplicaFactor 设置复制因子
func WithReplicaFactor(factor int) Option {
	return func(cfg *DistributedCacheConfig) {
		if factor > 0 {
			cfg.ReplicaFactor = factor
		}
	}
}

// WithVirtualNodes 设置虚拟节点数量
func WithVirtualNodes(count int) Option {
	return func(cfg *DistributedCacheConfig) {
		if count > 0 {
			cfg.VirtualNodes = count
		}
	}
}

// WithOperationTimeout 设置操作超时时间
func WithOperationTimeout(timeout time.Duration) Option {
	return func(cfg *DistributedCacheConfig) {
		if timeout > 0 {
			cfg.OperationTimeout = timeout
		}
	}
}

// WithRetryOptions 设置重试选项
func WithRetryOptions(maxRetries int, interval time.Duration) Option {
	return func(cfg *DistributedCacheConfig) {
		if maxRetries > 0 {
			cfg.MaxRetries = maxRetries
		}
		if interval > 0 {
			cfg.RetryInterval = interval
		}
	}
}

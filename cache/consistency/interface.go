package consistency

import (
	"context"
	"time"
)

// DataSource 定义数据源接口
// 它是缓存一致性策略的后备存储
type DataSource interface {
	// Load 从数据源加载数据
	Load(ctx context.Context, key string) (any, error)

	// Store 将数据保存到数据源
	Store(ctx context.Context, key string, value any) error

	// Remove 从数据源删除数据
	Remove(ctx context.Context, key string) error
}

// ConsistencyStrategy 定义缓存一致性策略接口
// 它处理缓存和数据源之间的一致性问题
type ConsistencyStrategy interface {
	// Get 从缓存获取数据，缓存不存在时从数据源获取
	Get(ctx context.Context, key string) (any, error)

	// Set 设置缓存数据并根据策略同步到数据源
	Set(ctx context.Context, key string, value any, expiration time.Duration) error

	// Del 删除缓存和数据源中的数据
	Del(ctx context.Context, key string) error

	// Invalidate 使缓存失效但不影响数据源
	Invalidate(ctx context.Context, key string) error
}

// MessagePublisher 消息发布接口
// 用于实现基于消息的缓存一致性通知
type MessagePublisher interface {
	// Publish 发布缓存更改消息
	Publish(ctx context.Context, topic string, msg []byte) error

	// Subscribe 订阅缓存更改消息
	Subscribe(topic string, handler func(msg []byte)) error

	// Close 关闭发布者
	Close() error
}

// 消息类型常量
const (
	// MsgTypeInvalidate 表示缓存失效消息
	MsgTypeInvalidate = "invalidate"

	// MsgTypeUpdate 表示缓存更新消息
	MsgTypeUpdate = "update"
)

// CacheMessage 缓存消息结构
// 用于在缓存节点间通信
type CacheMessage struct {
	// Type 消息类型
	Type string `json:"type"`

	// Key 缓存键
	Key string `json:"key"`

	// Data 可选的数据负载(对于更新消息)
	Data []byte `json:"data,omitempty"`

	// Timestamp 消息时间戳
	Timestamp int64 `json:"timestamp"`
}

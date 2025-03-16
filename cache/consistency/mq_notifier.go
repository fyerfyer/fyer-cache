package consistency

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/fyerfyer/fyer-cache/internal/ferr"
)

// MQNotifierStrategy 实现基于消息队列的缓存一致性策略
// 该策略在数据源变更时发布消息，使缓存失效
type MQNotifierStrategy struct {
	cache      cache.Cache      // 缓存实例
	dataSource DataSource       // 后端数据源
	publisher  MessagePublisher // 消息发布器
	options    *MQNotifierOptions
}

// NewMQNotifierStrategy 创建基于消息队列通知的一致性策略
func NewMQNotifierStrategy(cache cache.Cache, dataSource DataSource, publisher MessagePublisher, opts ...Option) *MQNotifierStrategy {
	options := defaultMQNotifierOptions()

	// 应用选项
	for _, opt := range opts {
		opt(options)
	}

	strategy := &MQNotifierStrategy{
		cache:      cache,
		dataSource: dataSource,
		publisher:  publisher,
		options:    options,
	}

	// 自动订阅消息处理
	err := publisher.Subscribe(options.Topic, strategy.handleMessage)
	if err != nil {
		// 实际应用应当有日志记录
		// log.Printf("Failed to subscribe to topic %s: %v", options.Topic, err)
	}

	return strategy
}

// Get 从缓存获取数据，缓存未命中时从数据源获取
func (m *MQNotifierStrategy) Get(ctx context.Context, key string) (any, error) {
	// 首先尝试从缓存获取
	value, err := m.cache.Get(ctx, key)
	if err == nil {
		// 缓存命中，直接返回
		return value, nil
	}

	// 如果错误不是"键不存在"，则返回错误
	if err != ferr.ErrKeyNotFound {
		return nil, err
	}

	// 从数据源获取
	value, err = m.dataSource.Load(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to load from data source: %w", err)
	}

	// 将数据写入缓存
	_ = m.cache.Set(ctx, key, value, m.options.DefaultTTL)

	return value, nil
}

// Set 设置缓存数据并同步到数据源
// 在MQ策略中，先更新数据源，然后发布消息
func (m *MQNotifierStrategy) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	// 先更新数据源
	if err := m.dataSource.Store(ctx, key, value); err != nil {
		return fmt.Errorf("failed to store in data source: %w", err)
	}

	// 更新缓存
	if err := m.cache.Set(ctx, key, value, expiration); err != nil {
		// 如果缓存更新失败，但数据源已经更新，记录错误但认为操作成功
		// 实际应用中应该有日志记录
		// log.Printf("Cache set failed but data source updated: %v", err)
	}

	// 发布缓存更新消息
	message := CacheMessage{
		Type:      MsgTypeUpdate,
		Key:       key,
		Timestamp: time.Now().UnixNano(),
	}

	// 尝试发布消息，支持重试
	return m.publishWithRetry(ctx, message)
}

// Del 删除缓存和数据源中的数据
func (m *MQNotifierStrategy) Del(ctx context.Context, key string) error {
	// 先从数据源删除
	if err := m.dataSource.Remove(ctx, key); err != nil {
		return fmt.Errorf("failed to remove from data source: %w", err)
	}

	// 删除缓存
	_ = m.cache.Del(ctx, key)

	// 发布缓存失效消息
	message := CacheMessage{
		Type:      MsgTypeInvalidate,
		Key:       key,
		Timestamp: time.Now().UnixNano(),
	}

	// 尝试发布消息，支持重试
	return m.publishWithRetry(ctx, message)
}

// Invalidate 使缓存失效但不影响数据源
func (m *MQNotifierStrategy) Invalidate(ctx context.Context, key string) error {
	// 删除缓存
	if err := m.cache.Del(ctx, key); err != nil {
		return err
	}

	// 发布缓存失效消息
	message := CacheMessage{
		Type:      MsgTypeInvalidate,
		Key:       key,
		Timestamp: time.Now().UnixNano(),
	}

	// 尝试发布消息，支持重试
	return m.publishWithRetry(ctx, message)
}

// publishWithRetry 尝试发布消息，支持重试
func (m *MQNotifierStrategy) publishWithRetry(ctx context.Context, message CacheMessage) error {
	// 序列化消息
	msgBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal cache message: %w", err)
	}

	// 第一次尝试
	err = m.publisher.Publish(ctx, m.options.Topic, msgBytes)
	if err == nil {
		return nil // 成功，直接返回
	}

	// 失败后进行重试
	retries := 0
	for retries < m.options.MaxRetries {
		// 等待重试间隔
		time.Sleep(m.options.RetryDelay)

		// 重试发布
		err = m.publisher.Publish(ctx, m.options.Topic, msgBytes)
		if err == nil {
			return nil // 成功，直接返回
		}

		retries++
	}

	return fmt.Errorf("failed to publish message after %d retries: %w", m.options.MaxRetries, err)
}

// handleMessage 处理接收到的缓存消息
func (m *MQNotifierStrategy) handleMessage(msg []byte) {
	var message CacheMessage
	if err := json.Unmarshal(msg, &message); err != nil {
		// 解析失败，记录错误并返回
		// 实际应用中应该有日志记录
		log.Fatalf("Failed to unmarshal cache message: %v", err)
		return
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch message.Type {
	case MsgTypeInvalidate:
		// 失效消息只需要删除缓存
		_ = m.cache.Del(ctx, message.Key)

	case MsgTypeUpdate:
		// 更新消息可能包含数据，但我们这里简单处理为失效缓存
		// 在实际实现中，可以根据需求选择更新缓存或使缓存失效
		_ = m.cache.Del(ctx, message.Key)

	default:
		// 未知消息类型，忽略
	}
}

// Close 关闭资源
func (m *MQNotifierStrategy) Close() error {
	if m.publisher != nil {
		return m.publisher.Close()
	}
	return nil
}

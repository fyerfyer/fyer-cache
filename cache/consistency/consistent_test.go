package consistency

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/internal/ferr"
	"github.com/fyerfyer/fyer-cache/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestCacheAsideStrategy 测试 Cache Aside 策略
func TestCacheAsideStrategy(t *testing.T) {
	t.Run("Get_CacheHit", func(t *testing.T) {
		mockCache := mocks.NewCache(t)
		mockDataSource := mocks.NewDataSource(t)
		// 设置 cache.Get 期望行为 - 缓存命中
		mockCache.EXPECT().Get(mock.Anything, "key1").Return("value1", nil)

		// 创建 Cache Aside 策略
		strategy := NewCacheAsideStrategy(mockCache, mockDataSource)

		// 执行测试
		value, err := strategy.Get(context.Background(), "key1")

		// 验证结果
		assert.NoError(t, err)
		assert.Equal(t, "value1", value)

		// 验证不会调用数据源
		mockDataSource.AssertNotCalled(t, "Load")
	})

	t.Run("Get_CacheMiss", func(t *testing.T) {
		// 重置 mock
		mockCache := mocks.NewCache(t)
		mockDataSource := mocks.NewDataSource(t)

		// 设置 cache.Get 期望行为 - 缓存未命中
		mockCache.EXPECT().Get(mock.Anything, "key1").Return(nil, ferr.ErrKeyNotFound)

		// 设置 dataSource.Load 期望行为
		mockDataSource.EXPECT().Load(mock.Anything, "key1").Return("value1", nil)

		// 设置 cache.Set 期望行为
		mockCache.EXPECT().Set(mock.Anything, "key1", "value1", mock.Anything).Return(nil)

		// 创建 Cache Aside 策略
		strategy := NewCacheAsideStrategy(mockCache, mockDataSource)

		// 执行测试
		value, err := strategy.Get(context.Background(), "key1")

		// 验证结果
		assert.NoError(t, err)
		assert.Equal(t, "value1", value)
	})

	t.Run("Get_DataSourceError", func(t *testing.T) {
		// 重置 mock
		mockCache := mocks.NewCache(t)
		mockDataSource := mocks.NewDataSource(t)

		// 设置 cache.Get 期望行为 - 缓存未命中
		mockCache.EXPECT().Get(mock.Anything, "key1").Return(nil, ferr.ErrKeyNotFound)

		// 设置 dataSource.Load 期望行为 - 数据源错误
		expectedErr := errors.New("data source error")
		mockDataSource.EXPECT().Load(mock.Anything, "key1").Return(nil, expectedErr)

		// 创建 Cache Aside 策略
		strategy := NewCacheAsideStrategy(mockCache, mockDataSource)

		// 执行测试
		value, err := strategy.Get(context.Background(), "key1")

		// 验证结果
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Contains(t, err.Error(), "data source error")
	})

	t.Run("Set", func(t *testing.T) {
		// 重置 mock
		mockCache := mocks.NewCache(t)
		mockDataSource := mocks.NewDataSource(t)

		// 设置 dataSource.Store 期望行为
		mockDataSource.EXPECT().Store(mock.Anything, "key1", "value1").Return(nil)

		// 设置 cache.Set 期望行为
		mockCache.EXPECT().Set(mock.Anything, "key1", "value1", 5*time.Minute).Return(nil)

		// 创建 Cache Aside 策略
		strategy := NewCacheAsideStrategy(mockCache, mockDataSource)

		// 执行测试
		err := strategy.Set(context.Background(), "key1", "value1", 5*time.Minute)

		// 验证结果
		assert.NoError(t, err)
	})

	t.Run("Del", func(t *testing.T) {
		// 重置 mock
		mockCache := mocks.NewCache(t)
		mockDataSource := mocks.NewDataSource(t)

		// 设置 cache.Del 期望行为
		mockCache.EXPECT().Del(mock.Anything, "key1").Return(nil)

		// 设置 dataSource.Remove 期望行为
		mockDataSource.EXPECT().Remove(mock.Anything, "key1").Return(nil)

		// 创建 Cache Aside 策略
		strategy := NewCacheAsideStrategy(mockCache, mockDataSource)

		// 执行测试
		err := strategy.Del(context.Background(), "key1")

		// 验证结果
		assert.NoError(t, err)
	})
}

// TestWriteThroughStrategy 测试 Write Through 策略
func TestWriteThroughStrategy(t *testing.T) {
	t.Run("Get_CacheHit", func(t *testing.T) {
		mockCache := mocks.NewCache(t)
		mockDataSource := mocks.NewDataSource(t)
		// 设置 cache.Get 期望行为 - 缓存命中
		mockCache.EXPECT().Get(mock.Anything, "key1").Return("value1", nil)

		// 创建 Write Through 策略
		strategy := NewWriteThroughStrategy(mockCache, mockDataSource)

		// 执行测试
		value, err := strategy.Get(context.Background(), "key1")

		// 验证结果
		assert.NoError(t, err)
		assert.Equal(t, "value1", value)

		// 验证不会调用数据源
		mockDataSource.AssertNotCalled(t, "Load")
	})

	t.Run("Get_CacheMiss", func(t *testing.T) {
		mockCache := mocks.NewCache(t)
		mockDataSource := mocks.NewDataSource(t)

		// 设置 cache.Get 期望行为 - 缓存未命中
		mockCache.EXPECT().Get(mock.Anything, "key1").Return(nil, ferr.ErrKeyNotFound)

		// 设置 dataSource.Load 期望行为
		mockDataSource.EXPECT().Load(mock.Anything, "key1").Return("value1", nil)

		// 设置 cache.Set 期望行为
		mockCache.EXPECT().Set(mock.Anything, "key1", "value1", mock.Anything).Return(nil)

		// 创建 Write Through 策略
		strategy := NewWriteThroughStrategy(mockCache, mockDataSource)

		// 执行测试
		value, err := strategy.Get(context.Background(), "key1")

		// 验证结果
		assert.NoError(t, err)
		assert.Equal(t, "value1", value)
	})

	t.Run("Set_Normal", func(t *testing.T) {
		// 重置 mock
		mockCache := mocks.NewCache(t)
		mockDataSource := mocks.NewDataSource(t)

		// 设置 dataSource.Store 期望行为
		mockDataSource.EXPECT().Store(mock.Anything, "key1", "value1").Return(nil)

		// 设置 cache.Set 期望行为
		mockCache.EXPECT().Set(mock.Anything, "key1", "value1", 5*time.Minute).Return(nil)

		// 创建 Write Through 策略
		strategy := NewWriteThroughStrategy(mockCache, mockDataSource)

		// 执行测试
		err := strategy.Set(context.Background(), "key1", "value1", 5*time.Minute)

		// 验证结果
		assert.NoError(t, err)
	})

	t.Run("Set_WithDelayedWrite", func(t *testing.T) {
		// 重置 mock
		mockCache := new(mocks.Cache)
		mockDS := new(mocks.DataSource)

		// 设置 cache.Set 期望行为
		mockCache.EXPECT().Set(mock.Anything, "key1", "value1", time.Minute).Return(nil)

		// 关键修复: 设置异步调用的 dataSource.Store 期望行为
		// 注意: 因为是异步调用，我们使用 mock.Anything 来匹配 context
		mockDS.EXPECT().Store(mock.Anything, "key1", "value1").Return(nil)

		// 创建带延迟写入的 Write Through 策略
		strategy := NewWriteThroughStrategy(mockCache, mockDS,
			WithDelayedWrite(true, 10*time.Millisecond)) // 使用更短的延迟简化测试

		// 执行测试
		ctx := context.Background()
		err := strategy.Set(ctx, "key1", "value1", time.Minute)
		assert.NoError(t, err)

		// 等待异步写入完成 - 这是修复的关键
		// 使用比延迟写入更长的时间，确保异步操作有足够时间完成
		time.Sleep(50 * time.Millisecond)

		// 验证数据源也被写入了
		mockDS.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("Del", func(t *testing.T) {
		// 重置 mock
		mockCache := mocks.NewCache(t)
		mockDataSource := mocks.NewDataSource(t)

		// 设置 dataSource.Remove 期望行为
		mockDataSource.EXPECT().Remove(mock.Anything, "key1").Return(nil)

		// 设置 cache.Del 期望行为
		mockCache.EXPECT().Del(mock.Anything, "key1").Return(nil)

		// 创建 Write Through 策略
		strategy := NewWriteThroughStrategy(mockCache, mockDataSource)

		// 执行测试
		err := strategy.Del(context.Background(), "key1")

		// 验证结果
		assert.NoError(t, err)
	})
}

// TestMQNotifierStrategy 测试 MQ 通知器策略
func TestMQNotifierStrategy(t *testing.T) {
	t.Run("Get_CacheHit", func(t *testing.T) {
		mockCache := mocks.NewCache(t)
		mockDataSource := mocks.NewDataSource(t)
		mockPublisher := mocks.NewMessagePublisher(t)

		// 设置 cache.Get 期望行为 - 缓存命中
		mockCache.EXPECT().Get(mock.Anything, "key1").Return("value1", nil)

		// 设置 publisher.Subscribe 期望
		mockPublisher.EXPECT().Subscribe(mock.Anything, mock.Anything).Return(nil)

		// 创建 MQ Notifier 策略
		strategy := NewMQNotifierStrategy(mockCache, mockDataSource, mockPublisher)

		// 执行测试
		value, err := strategy.Get(context.Background(), "key1")

		// 验证结果
		assert.NoError(t, err)
		assert.Equal(t, "value1", value)
	})

	t.Run("Set", func(t *testing.T) {
		// 重置 mock
		mockCache := mocks.NewCache(t)
		mockDataSource := mocks.NewDataSource(t)
		mockPublisher := mocks.NewMessagePublisher(t)

		// 设置 dataSource.Store 期望行为
		mockDataSource.EXPECT().Store(mock.Anything, "key1", "value1").Return(nil)

		// 设置 cache.Set 期望行为
		mockCache.EXPECT().Set(mock.Anything, "key1", "value1", 5*time.Minute).Return(nil)

		// 设置 publisher.Subscribe 期望
		mockPublisher.EXPECT().Subscribe(mock.Anything, mock.Anything).Return(nil)

		// 设置 publisher.Publish 期望 - 对于任何消息都返回成功
		mockPublisher.EXPECT().Publish(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// 创建 MQ Notifier 策略
		strategy := NewMQNotifierStrategy(mockCache, mockDataSource, mockPublisher)

		// 执行测试
		err := strategy.Set(context.Background(), "key1", "value1", 5*time.Minute)

		// 验证结果
		assert.NoError(t, err)
	})

	t.Run("Del", func(t *testing.T) {
		// 重置 mock
		mockCache := mocks.NewCache(t)
		mockDataSource := mocks.NewDataSource(t)
		mockPublisher := mocks.NewMessagePublisher(t)

		// 设置 dataSource.Remove 期望行为
		mockDataSource.EXPECT().Remove(mock.Anything, "key1").Return(nil)

		// 设置 cache.Del 期望行为
		mockCache.EXPECT().Del(mock.Anything, "key1").Return(nil)

		// 设置 publisher.Subscribe 期望
		mockPublisher.EXPECT().Subscribe(mock.Anything, mock.Anything).Return(nil)

		// 设置 publisher.Publish 期望 - 对于任何消息都返回成功
		mockPublisher.EXPECT().Publish(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// 创建 MQ Notifier 策略
		strategy := NewMQNotifierStrategy(mockCache, mockDataSource, mockPublisher)

		// 执行测试
		err := strategy.Del(context.Background(), "key1")

		// 验证结果
		assert.NoError(t, err)
	})

	t.Run("Invalidate", func(t *testing.T) {
		// 重置 mock
		mockCache := mocks.NewCache(t)
		mockDataSource := mocks.NewDataSource(t)
		mockPublisher := mocks.NewMessagePublisher(t)

		// 设置 cache.Del 期望行为
		mockCache.EXPECT().Del(mock.Anything, "key1").Return(nil)

		// 设置 publisher.Subscribe 期望
		mockPublisher.EXPECT().Subscribe(mock.Anything, mock.Anything).Return(nil)

		// 设置 publisher.Publish 期望 - 对于任何消息都返回成功
		mockPublisher.EXPECT().Publish(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// 创建 MQ Notifier 策略
		strategy := NewMQNotifierStrategy(mockCache, mockDataSource, mockPublisher)

		// 执行测试
		err := strategy.Invalidate(context.Background(), "key1")

		// 验证结果
		assert.NoError(t, err)
	})

	t.Run("Close", func(t *testing.T) {
		// 重置 mock
		mockCache := mocks.NewCache(t)
		mockDataSource := mocks.NewDataSource(t)
		mockPublisher := mocks.NewMessagePublisher(t)

		// 设置 publisher.Subscribe 期望
		mockPublisher.EXPECT().Subscribe(mock.Anything, mock.Anything).Return(nil)

		// 设置 publisher.Close 期望
		mockPublisher.EXPECT().Close().Return(nil)

		// 创建 MQ Notifier 策略
		strategy := NewMQNotifierStrategy(mockCache, mockDataSource, mockPublisher)

		// 执行测试
		err := strategy.Close()

		// 验证结果
		assert.NoError(t, err)
	})
}

// TestOptions 测试配置选项
func TestOptions(t *testing.T) {
	t.Run("CacheAsideOptions", func(t *testing.T) {
		options := defaultCacheAsideOptions()
		assert.Equal(t, DefaultTTL, options.DefaultTTL)
		assert.True(t, options.WriteOnMiss)

		// 测试 WithCacheAsideTTL
		WithCacheAsideTTL(10 * time.Second)(options)
		assert.Equal(t, 10*time.Second, options.DefaultTTL)

		// 测试 WithCacheAsideWriteOnMiss
		WithCacheAsideWriteOnMiss(false)(options)
		assert.False(t, options.WriteOnMiss)
	})

	t.Run("WriteThroughOptions", func(t *testing.T) {
		options := defaultWriteThroughOptions()
		assert.Equal(t, DefaultTTL, options.DefaultTTL)
		assert.False(t, options.DelayedWrite)
		assert.Equal(t, 100*time.Millisecond, options.WriteDelay)

		// 测试 WithWriteThroughTTL
		WithWriteThroughTTL(10 * time.Second)(options)
		assert.Equal(t, 10*time.Second, options.DefaultTTL)

		// 测试 WithDelayedWrite
		WithDelayedWrite(true, 200*time.Millisecond)(options)
		assert.True(t, options.DelayedWrite)
		assert.Equal(t, 200*time.Millisecond, options.WriteDelay)
	})

	t.Run("MQNotifierOptions", func(t *testing.T) {
		options := defaultMQNotifierOptions()
		assert.Equal(t, DefaultTTL, options.DefaultTTL)
		assert.Equal(t, DefaultTopic, options.Topic)
		assert.Equal(t, DefaultMaxRetry, options.MaxRetries)
		assert.Equal(t, 500*time.Millisecond, options.RetryDelay)

		// 测试 WithMQTopic
		WithMQTopic("custom-topic")(options)
		assert.Equal(t, "custom-topic", options.Topic)

		// 测试 WithMQRetry
		WithMQRetry(5, time.Second)(options)
		assert.Equal(t, 5, options.MaxRetries)
		assert.Equal(t, time.Second, options.RetryDelay)

		// 测试 WithMQNotifierTTL
		WithMQNotifierTTL(10 * time.Second)(options)
		assert.Equal(t, 10*time.Second, options.DefaultTTL)
	})
}

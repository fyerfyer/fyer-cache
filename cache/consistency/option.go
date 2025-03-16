package consistency

import "time"

// 默认选项值
const (
	DefaultTTL      = 5 * time.Minute
	DefaultTopic    = "cache:consistency"
	DefaultMaxRetry = 3
)

// Option 是一个函数类型，用于设置策略选项
type Option func(interface{})

// CacheAsideOptions 定义 Cache Aside 策略的选项
type CacheAsideOptions struct {
	// DefaultTTL 缓存默认过期时间
	DefaultTTL time.Duration

	// WriteOnMiss 是否在缓存未命中时写入缓存
	WriteOnMiss bool
}

// defaultCacheAsideOptions 返回默认的 Cache Aside 选项
func defaultCacheAsideOptions() *CacheAsideOptions {
	return &CacheAsideOptions{
		DefaultTTL:  DefaultTTL,
		WriteOnMiss: true,
	}
}

// WithCacheAsideTTL 设置 Cache Aside 策略的默认TTL
func WithCacheAsideTTL(ttl time.Duration) Option {
	return func(o interface{}) {
		if opts, ok := o.(*CacheAsideOptions); ok {
			opts.DefaultTTL = ttl
		}
	}
}

// WithCacheAsideWriteOnMiss 设置是否在未命中时写入缓存
func WithCacheAsideWriteOnMiss(writeOnMiss bool) Option {
	return func(o interface{}) {
		if opts, ok := o.(*CacheAsideOptions); ok {
			opts.WriteOnMiss = writeOnMiss
		}
	}
}

// WriteThroughOptions 定义 Write Through 策略的选项
type WriteThroughOptions struct {
	// DefaultTTL 缓存默认过期时间
	DefaultTTL time.Duration

	// DelayedWrite 是否使用延迟写入
	DelayedWrite bool

	// WriteDelay 延迟写入时间
	WriteDelay time.Duration
}

// defaultWriteThroughOptions 返回默认的 Write Through 选项
func defaultWriteThroughOptions() *WriteThroughOptions {
	return &WriteThroughOptions{
		DefaultTTL:   DefaultTTL,
		DelayedWrite: false,
		WriteDelay:   100 * time.Millisecond,
	}
}

// WithWriteThroughTTL 设置 Write Through 策略的默认TTL
func WithWriteThroughTTL(ttl time.Duration) Option {
	return func(o interface{}) {
		if opts, ok := o.(*WriteThroughOptions); ok {
			opts.DefaultTTL = ttl
		}
	}
}

// WithDelayedWrite 设置是否使用延迟写入及延迟时间
func WithDelayedWrite(delayed bool, delay time.Duration) Option {
	return func(o interface{}) {
		if opts, ok := o.(*WriteThroughOptions); ok {
			opts.DelayedWrite = delayed
			if delay > 0 {
				opts.WriteDelay = delay
			}
		}
	}
}

// MQNotifierOptions 定义基于消息队列的通知策略选项
type MQNotifierOptions struct {
	// DefaultTTL 缓存默认过期时间
	DefaultTTL time.Duration

	// Topic 消息主题
	Topic string

	// MaxRetries 最大重试次数
	MaxRetries int

	// RetryDelay 重试延迟
	RetryDelay time.Duration
}

// defaultMQNotifierOptions 返回默认的 MQ 通知器选项
func defaultMQNotifierOptions() *MQNotifierOptions {
	return &MQNotifierOptions{
		DefaultTTL: DefaultTTL,
		Topic:      DefaultTopic,
		MaxRetries: DefaultMaxRetry,
		RetryDelay: 500 * time.Millisecond,
	}
}

// WithMQTopic 设置消息主题
func WithMQTopic(topic string) Option {
	return func(o interface{}) {
		if opts, ok := o.(*MQNotifierOptions); ok && topic != "" {
			opts.Topic = topic
		}
	}
}

// WithMQRetry 设置重试策略
func WithMQRetry(maxRetries int, retryDelay time.Duration) Option {
	return func(o interface{}) {
		if opts, ok := o.(*MQNotifierOptions); ok {
			if maxRetries >= 0 {
				opts.MaxRetries = maxRetries
			}
			if retryDelay > 0 {
				opts.RetryDelay = retryDelay
			}
		}
	}
}

// WithMQNotifierTTL 设置 MQ 通知器的默认TTL
func WithMQNotifierTTL(ttl time.Duration) Option {
	return func(o interface{}) {
		if opts, ok := o.(*MQNotifierOptions); ok {
			opts.DefaultTTL = ttl
		}
	}
}

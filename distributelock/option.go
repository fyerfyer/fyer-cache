package distributelock

import "time"

// LockOption 锁配置选项
type LockOption struct {
	// 锁过期时间
	Expiration time.Duration
	// 是否启用自动续约
	EnableWatchdog bool
	// 重试间隔
	RetryInterval time.Duration
	// 重试次数
	RetryTimes int
	// 是否阻塞等待
	BlockWaiting bool
	// 最大重试次数
	MaxRetries int
	// 重试等待时间增长策略
	BackoffStrategy BackoffStrategy
}

// BackoffStrategy 重试等待时间增长策略接口
type BackoffStrategy interface {
	NextBackoff(retryCount int) time.Duration
}

// ExponentialBackoff 指数重试时间增长策略
type ExponentialBackoff struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
}

// NextBackoff 指数计算下一次重试等待时间
func (e *ExponentialBackoff) NextBackoff(retryCount int) time.Duration {
	interval := float64(e.InitialInterval)
	for i := 0; i < retryCount; i++ {
		interval = interval * e.Multiplier
		if interval > float64(e.MaxInterval) {
			return e.MaxInterval
		}
	}

	return time.Duration(interval)
}

// Option 定义选项函数类型
type Option func(*LockOption)

// WithExpiration 设置锁过期时间
func WithExpiration(expiration time.Duration) Option {
	return func(o *LockOption) {
		o.Expiration = expiration
	}
}

// WithWatchdog 设置是否启用自动续约
func WithWatchdog(enable bool) Option {
	return func(o *LockOption) {
		o.EnableWatchdog = enable
	}
}

// WithRetryInterval 设置重试间隔
func WithRetryInterval(interval time.Duration) Option {
	return func(o *LockOption) {
		o.RetryInterval = interval
	}
}

// WithRetryTimes 设置重试次数
func WithRetryTimes(times int) Option {
	return func(o *LockOption) {
		o.RetryTimes = times
	}
}

// WithBlockWaiting 设置是否阻塞等待
func WithBlockWaiting(block bool) Option {
	return func(o *LockOption) {
		o.BlockWaiting = block
	}
}

// DefaultOption 默认选项
func DefaultOption() *LockOption {
	return &LockOption{
		Expiration:     30 * time.Second,
		EnableWatchdog: true,
		RetryInterval:  100 * time.Millisecond,
		RetryTimes:     0, // 0表示无限重试
		BlockWaiting:   true,
	}
}

package api

import (
	"time"
)

// APIConfig API 服务器配置
type APIConfig struct {
	// 绑定地址，例如 ":8080"
	BindAddress string

	// API 基础路径，如 "/api"
	BasePath string

	// 读取超时
	ReadTimeout time.Duration

	// 写入超时
	WriteTimeout time.Duration

	// 空闲连接超时
	IdleTimeout time.Duration
}

// DefaultAPIConfig 返回默认的 API 配置
func DefaultAPIConfig() *APIConfig {
	return &APIConfig{
		BindAddress:  ":8080",
		BasePath:     "/api",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// Option API 服务器配置选项
type Option func(*APIConfig)

// WithBindAddress 设置绑定地址
func WithBindAddress(addr string) Option {
	return func(c *APIConfig) {
		if addr != "" {
			c.BindAddress = addr
		}
	}
}

// WithBasePath 设置 API 基础路径
func WithBasePath(path string) Option {
	return func(c *APIConfig) {
		if path != "" {
			c.BasePath = path
		}
	}
}

// WithTimeouts 设置所有超时参数
func WithTimeouts(readTimeout, writeTimeout, idleTimeout time.Duration) Option {
	return func(c *APIConfig) {
		if readTimeout > 0 {
			c.ReadTimeout = readTimeout
		}
		if writeTimeout > 0 {
			c.WriteTimeout = writeTimeout
		}
		if idleTimeout > 0 {
			c.IdleTimeout = idleTimeout
		}
	}
}

// WithReadTimeout 设置读取超时
func WithReadTimeout(timeout time.Duration) Option {
	return func(c *APIConfig) {
		if timeout > 0 {
			c.ReadTimeout = timeout
		}
	}
}

// WithWriteTimeout 设置写入超时
func WithWriteTimeout(timeout time.Duration) Option {
	return func(c *APIConfig) {
		if timeout > 0 {
			c.WriteTimeout = timeout
		}
	}
}
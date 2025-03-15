package cluster

import (
	"time"
)

// NodeConfig 节点基础配置
type NodeConfig struct {
	// 节点唯一标识符
	ID string

	// 节点绑定地址，格式为 "ip:port"
	BindAddr string

	// Gossip通信间隔
	GossipInterval time.Duration

	// 数据同步间隔
	SyncInterval time.Duration

	// 节点健康检查间隔
	ProbeInterval time.Duration

	// 健康检查超时时间
	ProbeTimeout time.Duration

	// 可疑状态乘数
	// 节点被标记为可疑状态后，等待这个倍数的ProbeInterval才会认为节点失效
	SuspicionMult int

	// 数据同步并发数
	SyncConcurrency int

	// 虚拟节点数量，用于一致性哈希
	VirtualNodes int

	// 重试次数
	MaxRetries int

	// 重试间隔
	RetryInterval time.Duration

	// 事件通道缓冲区大小
	EventBufferSize int
}

// NodeOption 节点配置选项
type NodeOption func(*NodeConfig)

// WithNodeID 设置节点ID
func WithNodeID(id string) NodeOption {
	return func(c *NodeConfig) {
		c.ID = id
	}
}

// WithBindAddr 设置绑定地址
func WithBindAddr(addr string) NodeOption {
	return func(c *NodeConfig) {
		c.BindAddr = addr
	}
}

// WithGossipInterval 设置Gossip通信间隔
func WithGossipInterval(interval time.Duration) NodeOption {
	return func(c *NodeConfig) {
		if interval > 0 {
			c.GossipInterval = interval
		}
	}
}

// WithSyncInterval 设置数据同步间隔
func WithSyncInterval(interval time.Duration) NodeOption {
	return func(c *NodeConfig) {
		if interval > 0 {
			c.SyncInterval = interval
		}
	}
}

// WithProbeInterval 设置健康检查间隔
func WithProbeInterval(interval time.Duration) NodeOption {
	return func(c *NodeConfig) {
		if interval > 0 {
			c.ProbeInterval = interval
		}
	}
}

// WithProbeTimeout 设置健康检查超时
func WithProbeTimeout(timeout time.Duration) NodeOption {
	return func(c *NodeConfig) {
		if timeout > 0 {
			c.ProbeTimeout = timeout
		}
	}
}

// WithSuspicionMult 设置可疑状态乘数
func WithSuspicionMult(mult int) NodeOption {
	return func(c *NodeConfig) {
		if mult > 0 {
			c.SuspicionMult = mult
		}
	}
}

// WithSyncConcurrency 设置数据同步并发数
func WithSyncConcurrency(concurrency int) NodeOption {
	return func(c *NodeConfig) {
		if concurrency > 0 {
			c.SyncConcurrency = concurrency
		}
	}
}

// WithVirtualNodes 设置虚拟节点数量
func WithVirtualNodes(count int) NodeOption {
	return func(c *NodeConfig) {
		if count > 0 {
			c.VirtualNodes = count
		}
	}
}

// WithMaxRetries 设置最大重试次数
func WithMaxRetries(retries int) NodeOption {
	return func(c *NodeConfig) {
		if retries > 0 {
			c.MaxRetries = retries
		}
	}
}

// WithRetryInterval 设置重试间隔
func WithRetryInterval(interval time.Duration) NodeOption {
	return func(c *NodeConfig) {
		if interval > 0 {
			c.RetryInterval = interval
		}
	}
}

// WithEventBufferSize 设置事件缓冲区大小
func WithEventBufferSize(size int) NodeOption {
	return func(c *NodeConfig) {
		if size > 0 {
			c.EventBufferSize = size
		}
	}
}

// DefaultNodeConfig 返回默认节点配置
func DefaultNodeConfig() *NodeConfig {
	return &NodeConfig{
		GossipInterval:  1 * time.Second,
		SyncInterval:    30 * time.Second,
		ProbeInterval:   1 * time.Second,
		ProbeTimeout:    500 * time.Millisecond,
		SuspicionMult:   3,
		SyncConcurrency: 5,
		VirtualNodes:    100,
		MaxRetries:      3,
		RetryInterval:   1 * time.Second,
		EventBufferSize: 100,
	}
}

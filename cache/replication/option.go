package replication

import (
	"time"
)

// 默认配置常量
const (
	// 默认心跳间隔
	DefaultHeartbeatInterval = 500 * time.Millisecond

	// 默认选举超时
	DefaultElectionTimeout = 1 * time.Second

	// 默认最大日志条目数
	DefaultMaxLogEntries = 1000

	// 默认触发快照的日志条目阈值
	DefaultSnapshotThreshold = 10000

	// 默认重试次数
	DefaultMaxRetries = 3

	// 默认重试间隔
	DefaultRetryInterval = 200 * time.Millisecond

	// 默认复制超时
	DefaultReplicationTimeout = 5 * time.Second
)

// ReplicationConfig 主从复制配置
type ReplicationConfig struct {
	// 节点ID
	NodeID string

	// 节点地址
	Address string

	// 节点角色
	Role ReplicationRole

	// 心跳间隔
	HeartbeatInterval time.Duration

	// 选举超时
	ElectionTimeout time.Duration

	// 单次同步最大日志条目数
	MaxLogEntries int

	// 触发快照的日志条目阈值
	SnapshotThreshold int

	// 最大重试次数
	MaxRetries int

	// 重试间隔
	RetryInterval time.Duration

	// 复制超时
	ReplicationTimeout time.Duration

	// 选举优先级，值越大优先级越高
	Priority int

	// 同步器地址
	SyncerAddress string
}

// DefaultReplicationConfig 返回默认复制配置
func DefaultReplicationConfig() *ReplicationConfig {
	return &ReplicationConfig{
		Role:               RoleFollower,
		HeartbeatInterval:  DefaultHeartbeatInterval,
		ElectionTimeout:    DefaultElectionTimeout,
		MaxLogEntries:      DefaultMaxLogEntries,
		SnapshotThreshold:  DefaultSnapshotThreshold,
		MaxRetries:         DefaultMaxRetries,
		RetryInterval:      DefaultRetryInterval,
		ReplicationTimeout: DefaultReplicationTimeout,
		Priority:           1, // 默认优先级为1
	}
}

// ReplicationOption 复制配置选项函数
type ReplicationOption func(*ReplicationConfig)

// WithNodeID 设置节点ID
func WithNodeID(id string) ReplicationOption {
	return func(config *ReplicationConfig) {
		if id != "" {
			config.NodeID = id
		}
	}
}

// WithAddress 设置节点地址
func WithAddress(addr string) ReplicationOption {
	return func(config *ReplicationConfig) {
		if addr != "" {
			config.Address = addr
		}
	}
}

// WithRole 设置节点角色
func WithRole(role ReplicationRole) ReplicationOption {
	return func(config *ReplicationConfig) {
		config.Role = role
	}
}

// WithHeartbeatInterval 设置心跳间隔
func WithHeartbeatInterval(interval time.Duration) ReplicationOption {
	return func(config *ReplicationConfig) {
		if interval > 0 {
			config.HeartbeatInterval = interval
		}
	}
}

// WithElectionTimeout 设置选举超时
func WithElectionTimeout(timeout time.Duration) ReplicationOption {
	return func(config *ReplicationConfig) {
		if timeout > 0 {
			config.ElectionTimeout = timeout
		}
	}
}

// WithMaxLogEntries 设置单次同步最大日志条目数
func WithMaxLogEntries(count int) ReplicationOption {
	return func(config *ReplicationConfig) {
		if count > 0 {
			config.MaxLogEntries = count
		}
	}
}

// WithSnapshotThreshold 设置触发快照的日志条目阈值
func WithSnapshotThreshold(threshold int) ReplicationOption {
	return func(config *ReplicationConfig) {
		if threshold > 0 {
			config.SnapshotThreshold = threshold
		}
	}
}

// WithMaxRetries 设置最大重试次数
func WithMaxRetries(retries int) ReplicationOption {
	return func(config *ReplicationConfig) {
		if retries > 0 {
			config.MaxRetries = retries
		}
	}
}

// WithRetryInterval 设置重试间隔
func WithRetryInterval(interval time.Duration) ReplicationOption {
	return func(config *ReplicationConfig) {
		if interval > 0 {
			config.RetryInterval = interval
		}
	}
}

// WithReplicationTimeout 设置复制超时
func WithReplicationTimeout(timeout time.Duration) ReplicationOption {
	return func(config *ReplicationConfig) {
		if timeout > 0 {
			config.ReplicationTimeout = timeout
		}
	}
}

// WithPriority 设置选举优先级
func WithPriority(priority int) ReplicationOption {
	return func(config *ReplicationConfig) {
		if priority > 0 {
			config.Priority = priority
		}
	}
}

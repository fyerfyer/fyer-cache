package replication

import (
	"context"
	"time"
)

// ReplicationRole 定义节点在复制中的角色
type ReplicationRole int

const (
	// RoleLeader 表示该节点是主节点
	RoleLeader ReplicationRole = iota
	// RoleFollower 表示该节点是从节点
	RoleFollower
	// RoleCandidate 表示节点正在参与选举
	RoleCandidate
)

// String 返回角色的字符串表示
func (r ReplicationRole) String() string {
	switch r {
	case RoleLeader:
		return "Leader"
	case RoleFollower:
		return "Follower"
	case RoleCandidate:
		return "Candidate"
	default:
		return "Unknown"
	}
}

// ReplicationState 节点复制状态
type ReplicationState int

const (
	// StateNormal 正常状态
	StateNormal ReplicationState = iota
	// StateSyncing 同步中状态
	StateSyncing
	// StateOutOfSync 不同步状态
	StateOutOfSync
)

// ReplicationEntry 复制日志条目
type ReplicationEntry struct {
	// Index 日志索引
	Index uint64
	// Term 任期号
	Term uint64
	// Command 命令类型 (Set, Del)
	Command string
	// Key 缓存键
	Key string
	// Value 缓存值
	Value []byte
	// Expiration 过期时间
	Expiration time.Duration
	// Timestamp 操作时间戳
	Timestamp time.Time
}

// ReplicationLog 复制日志接口
type ReplicationLog interface {
	// Append 追加日志条目
	Append(entry *ReplicationEntry) error

	// Get 获取指定索引的日志
	Get(index uint64) (*ReplicationEntry, error)

	// GetFrom 获取从指定索引开始的所有日志
	GetFrom(startIndex uint64, maxCount int) ([]*ReplicationEntry, error)

	// GetLastIndex 获取最后一条日志的索引
	GetLastIndex() uint64

	// GetLastTerm 获取最后一条日志的任期
	GetLastTerm() uint64

	// Truncate 截断日志到指定索引
	Truncate(index uint64) error
}

// ReplicationOptions 复制选项
type ReplicationOptions struct {
	// HeartbeatInterval 心跳间隔
	HeartbeatInterval time.Duration

	// ElectionTimeout 选举超时
	ElectionTimeout time.Duration

	// MaxLogEntries 单次同步最大日志条目数
	MaxLogEntries int

	// SnapshotThreshold 触发快照的日志条目阈值
	SnapshotThreshold int
}

// SyncRequest 同步请求
type SyncRequest struct {
	// LeaderID 领导者ID
	LeaderID string

	// LastLogIndex 请求者最后日志索引
	LastLogIndex uint64

	// LastLogTerm 请求者最后日志任期
	LastLogTerm uint64

	// FullSync 是否请求全量同步
	FullSync bool
}

// SyncResponse 同步响应
type SyncResponse struct {
	// Success 是否成功
	Success bool

	// Entries 日志条目
	Entries []*ReplicationEntry

	// NextIndex 下一个索引
	NextIndex uint64
}

// ReplicationNode 支持复制功能的节点接口
type ReplicationNode interface {
	// GetRole 获取当前节点角色
	GetRole() ReplicationRole

	// SetRole 设置节点角色
	SetRole(role ReplicationRole)

	// GetLeader 获取当前领导者ID
	GetLeader() string

	// RequestSync 请求同步数据
	RequestSync(ctx context.Context, target string, req *SyncRequest) (*SyncResponse, error)

	// ApplyEntry 应用一条日志条目到本地缓存
	ApplyEntry(ctx context.Context, entry *ReplicationEntry) error

	// GetLog 获取复制日志
	GetLog() ReplicationLog
}

// LeaderNode 领导者节点接口
type LeaderNode interface {
	ReplicationNode
	
	// Start 启动领导者节点服务
	Start() error
	
	// Stop 停止领导者节点服务
	Stop() error
	
	// AddFollower 添加一个从节点
	AddFollower(nodeID string, address string) error
	
	// RemoveFollower 移除一个从节点
	RemoveFollower(nodeID string) error
	
	// GetFollowers 获取所有从节点
	GetFollowers() map[string]string
	
	// ReplicateEntries 复制日志条目到所有从节点
	ReplicateEntries(ctx context.Context) error
	
	// SendHeartbeat 发送心跳到所有从节点
	SendHeartbeat(ctx context.Context) error
}

// FollowerNode 从节点接口
type FollowerNode interface {
	ReplicationNode
	
	// Start 启动从节点服务
	Start() error
	
	// Stop 停止从节点服务
	Stop() error
	
	// SyncFromLeader 从主节点同步数据
	SyncFromLeader(ctx context.Context) error
	
	// HandleHeartbeat 处理心跳消息
	HandleHeartbeat(ctx context.Context, leaderID string, term uint64) error
	
	// PromoteToLeader 提升为领导者
	PromoteToLeader(ctx context.Context) error
}

// Syncer 数据同步接口
type Syncer interface {
	// FullSync 执行全量同步
	FullSync(ctx context.Context, target string) error

	// IncrementalSync 执行增量同步
	IncrementalSync(ctx context.Context, target string, startIndex uint64) error

	// ApplySync 应用同步数据
	ApplySync(ctx context.Context, entries []*ReplicationEntry) error

	// Start 启动同步服务
	Start() error

	// Stop 停止同步服务
	Stop() error
}

// ReplicationEvent 复制事件
type ReplicationEvent struct {
	// Type 事件类型
	Type ReplicationEventType

	// NodeID 相关节点ID
	NodeID string

	// Role 节点角色
	Role ReplicationRole

	// Term 任期
	Term uint64

	// Timestamp 时间戳
	Timestamp time.Time

	// Details 额外详情
	Details interface{}
}

// ReplicationEventType 事件类型
type ReplicationEventType int

const (
	// EventRoleChange 角色变更
	EventRoleChange ReplicationEventType = iota

	// EventLeaderElected 领导者选举成功
	EventLeaderElected

	// EventSyncCompleted 同步完成
	EventSyncCompleted

	// EventSyncFailed 同步失败
	EventSyncFailed

	// EventFollowerAdded 新增从节点
	EventFollowerAdded

	// EventFollowerRemoved 移除从节点
	EventFollowerRemoved

	// EventLeaderLost 领导者丢失
	EventLeaderLost
)

package replication

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCache 是一个简单的缓存mock，用于测试
type mockCache struct {
	data map[string][]byte
}

func newMockCache() *mockCache {
	return &mockCache{
		data: make(map[string][]byte),
	}
}

func (m *mockCache) Get(ctx context.Context, key string) (any, error) {
	val, ok := m.data[key]
	if !ok {
		return nil, nil
	}
	return val, nil
}

func (m *mockCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
	if val == nil {
		return nil
	}

	var data []byte
	switch v := val.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return nil
	}

	m.data[key] = data
	return nil
}

func (m *mockCache) Del(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

// TestMemoryReplicationLog_BasicOperations 测试内存复制日志的基本操作
func TestMemoryReplicationLog_BasicOperations(t *testing.T) {
	log := NewMemoryReplicationLog()

	// 测试初始状态
	assert.Equal(t, uint64(0), log.GetLastIndex())
	assert.Equal(t, uint64(0), log.GetLastTerm())
	assert.Equal(t, 0, log.Count())

	// 测试添加日志条目
	entry1 := &ReplicationEntry{
		Term:      1,
		Command:   "Set",
		Key:       "key1",
		Value:     []byte("value1"),
		Timestamp: time.Now(),
	}

	err := log.Append(entry1)
	require.NoError(t, err)

	// 验证日志状态
	assert.Equal(t, uint64(1), log.GetLastIndex())
	assert.Equal(t, uint64(1), log.GetLastTerm())
	assert.Equal(t, 1, log.Count())

	// 获取日志条目
	entry, err := log.Get(1)
	require.NoError(t, err)
	assert.Equal(t, entry1.Key, entry.Key)
	assert.Equal(t, entry1.Term, entry.Term)

	// 追加第二个条目
	entry2 := &ReplicationEntry{
		Term:      2,
		Command:   "Set",
		Key:       "key2",
		Value:     []byte("value2"),
		Timestamp: time.Now(),
	}

	err = log.Append(entry2)
	require.NoError(t, err)

	// 测试 GetFrom
	entries, err := log.GetFrom(1, 10)
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	// 测试截断
	err = log.Truncate(1)
	require.NoError(t, err)
	assert.Equal(t, 1, log.Count())

	// 测试清空
	log.Clear()
	assert.Equal(t, 0, log.Count())
}

// TestMemorySyncer_BasicOperations 测试内存同步器的基本操作
func TestMemorySyncer_BasicOperations(t *testing.T) {
	//ctx := context.Background()
	cache := newMockCache()
	log := NewMemoryReplicationLog()

	syncer := NewMemorySyncer("node1", cache, log)
	require.NotNil(t, syncer)

	// 启动同步器
	err := syncer.Start()
	require.NoError(t, err)
	defer syncer.Stop()

	// 记录Set操作
	err = syncer.RecordSetOperation("test-key", []byte("test-value"), time.Minute, 1)
	require.NoError(t, err)

	// 验证日志中有记录
	assert.Equal(t, 1, log.Count())

	entry, err := log.Get(1)
	require.NoError(t, err)
	assert.Equal(t, "Set", entry.Command)
	assert.Equal(t, "test-key", entry.Key)

	// 记录Del操作
	err = syncer.RecordDelOperation("test-key", 1)
	require.NoError(t, err)

	// 验证日志中有记录
	assert.Equal(t, 2, log.Count())

	entry, err = log.Get(2)
	require.NoError(t, err)
	assert.Equal(t, "Del", entry.Command)
	assert.Equal(t, "test-key", entry.Key)
}

// mockSyncer 模拟同步器
type mockSyncer struct {
	startCalled bool
	stopCalled  bool
}

func (m *mockSyncer) Start() error {
	m.startCalled = true
	return nil
}

func (m *mockSyncer) Stop() error {
	m.stopCalled = true
	return nil
}

func (m *mockSyncer) FullSync(ctx context.Context, target string) error {
	return nil
}

func (m *mockSyncer) IncrementalSync(ctx context.Context, target string, startIndex uint64) error {
	return nil
}

func (m *mockSyncer) ApplySync(ctx context.Context, entries []*ReplicationEntry) error {
	return nil
}

// TestLeaderNode_BasicOperations 测试Leader节点的基本操作
func TestLeaderNode_BasicOperations(t *testing.T) {
	//ctx := context.Background()
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	// 创建Leader节点
	leader := NewLeaderNode("leader1", cache, log, syncer, WithRole(RoleLeader))
	require.NotNil(t, leader)

	// 测试添加从节点
	err := leader.AddFollower("follower1", "127.0.0.1:8001")
	require.NoError(t, err)

	// 验证从节点状态
	followers := leader.GetFollowers()
	assert.Len(t, followers, 1)
	assert.Equal(t, "127.0.0.1:8001", followers["follower1"])

	// 测试移除从节点
	err = leader.RemoveFollower("follower1")
	require.NoError(t, err)

	// 验证从节点已移除
	followers = leader.GetFollowers()
	assert.Len(t, followers, 0)
}

// TestFollowerNode_BasicOperations 测试Follower节点的基本操作
func TestFollowerNode_BasicOperations(t *testing.T) {
	//ctx := context.Background()
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := &mockSyncer{}

	// 创建Follower节点
	follower := NewFollowerNode("follower1", cache, log, syncer, WithRole(RoleFollower))
	require.NotNil(t, follower)

	// 设置Leader
	follower.SetLeader("leader1", "127.0.0.1:8000")

	// 验证Leader信息
	assert.Equal(t, "leader1", follower.GetLeader())

	// 验证节点角色
	assert.Equal(t, RoleFollower, follower.GetRole())

	// 更改角色
	follower.SetRole(RoleCandidate)
	assert.Equal(t, RoleCandidate, follower.GetRole())
}

// TestReplicationConfig 测试复制配置选项
func TestReplicationConfig(t *testing.T) {
	// 测试默认配置
	config := DefaultReplicationConfig()
	assert.Equal(t, RoleFollower, config.Role)
	assert.Equal(t, DefaultHeartbeatInterval, config.HeartbeatInterval)
	assert.Equal(t, DefaultElectionTimeout, config.ElectionTimeout)
	assert.Equal(t, DefaultMaxLogEntries, config.MaxLogEntries)

	// 测试配置选项
	nodeID := "test-node"
	address := "127.0.0.1:8000"
	role := RoleLeader
	heartbeatInterval := 100 * time.Millisecond
	electionTimeout := 500 * time.Millisecond
	maxLogEntries := 500
	priority := 5

	options := []ReplicationOption{
		WithNodeID(nodeID),
		WithAddress(address),
		WithRole(role),
		WithHeartbeatInterval(heartbeatInterval),
		WithElectionTimeout(electionTimeout),
		WithMaxLogEntries(maxLogEntries),
		WithPriority(priority),
	}

	// 应用选项到配置
	config = DefaultReplicationConfig()
	for _, opt := range options {
		opt(config)
	}

	// 验证配置
	assert.Equal(t, nodeID, config.NodeID)
	assert.Equal(t, address, config.Address)
	assert.Equal(t, role, config.Role)
	assert.Equal(t, heartbeatInterval, config.HeartbeatInterval)
	assert.Equal(t, electionTimeout, config.ElectionTimeout)
	assert.Equal(t, maxLogEntries, config.MaxLogEntries)
	assert.Equal(t, priority, config.Priority)
}

// TestReplicationEntry_Serialization 测试复制日志条目的序列化
func TestReplicationEntry_Serialization(t *testing.T) {
	// 创建日志实例
	log := NewMemoryReplicationLog()

	// 添加日志条目
	entry := &ReplicationEntry{
		Term:       1,
		Command:    "Set",
		Key:        "testkey",
		Value:      []byte("testvalue"),
		Expiration: time.Minute,
		Timestamp:  time.Now().Truncate(time.Millisecond),
	}

	err := log.Append(entry)
	require.NoError(t, err)

	// 序列化
	data, err := log.Serialize()
	require.NoError(t, err)

	// 创建新日志实例并反序列化
	newLog := NewMemoryReplicationLog()
	err = newLog.Deserialize(data)
	require.NoError(t, err)

	// 验证反序列化结果
	assert.Equal(t, 1, newLog.Count())

	newEntry, err := newLog.Get(1)
	require.NoError(t, err)
	assert.Equal(t, entry.Term, newEntry.Term)
	assert.Equal(t, entry.Command, newEntry.Command)
	assert.Equal(t, entry.Key, newEntry.Key)
	assert.Equal(t, entry.Expiration, newEntry.Expiration)
}

// TestReplicationRole_String 测试角色字符串表示
func TestReplicationRole_String(t *testing.T) {
	assert.Equal(t, "Leader", RoleLeader.String())
	assert.Equal(t, "Follower", RoleFollower.String())
	assert.Equal(t, "Candidate", RoleCandidate.String())

	// 测试无效角色
	var invalidRole ReplicationRole = 99
	assert.Equal(t, "Unknown", invalidRole.String())
}
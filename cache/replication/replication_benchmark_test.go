package replication

import (
	"context"
	"strconv"
	"testing"
	"time"
)

func setupReplicationLog(b *testing.B) *MemoryReplicationLog {
	b.Helper()
	return NewMemoryReplicationLog()
}

func prepareLogEntries(b *testing.B, count int) []*ReplicationEntry {
	b.Helper()
	entries := make([]*ReplicationEntry, count)
	for i := 0; i < count; i++ {
		entries[i] = &ReplicationEntry{
			Term:       1,
			Command:    "Set",
			Key:        "key-" + strconv.Itoa(i),
			Value:      []byte("value-" + strconv.Itoa(i)),
			Expiration: time.Minute,
			Timestamp:  time.Now(),
		}
	}
	return entries
}

type benchSyncer struct {
	startCalled bool
	stopCalled  bool
}

func (m *benchSyncer) RecordSetOperation(key string, value []byte, expiration time.Duration, term uint64) error {
	return nil
}

func (m *benchSyncer) RecordDelOperation(key string, term uint64) error {
	return nil
}

func (m *benchSyncer) Start() error {
	m.startCalled = true
	return nil
}

func (m *benchSyncer) Stop() error {
	m.stopCalled = true
	return nil
}

func (m *benchSyncer) FullSync(ctx context.Context, target string) error {
	return nil
}

func (m *benchSyncer) IncrementalSync(ctx context.Context, target string, startIndex uint64) error {
	return nil
}

func (m *benchSyncer) ApplySync(ctx context.Context, entries []*ReplicationEntry) error {
	return nil
}

func setupLeaderFollower(b *testing.B) (*LeaderNodeImpl, *FollowerNodeImpl, *benchSyncer) {
	b.Helper()

	cache1 := newMockCache()
	cache2 := newMockCache()
	log1 := NewMemoryReplicationLog()
	log2 := NewMemoryReplicationLog()
	syncer := &benchSyncer{}

	leaderNode := NewLeaderNode("leader", cache1, log1, syncer,
		WithRole(RoleLeader),
		WithNodeID("leader"),
		WithAddress("127.0.0.1:8001"))

	followerNode := NewFollowerNode("follower", cache2, log2, syncer,
		WithRole(RoleFollower),
		WithNodeID("follower"),
		WithAddress("127.0.0.1:8002"))

	return leaderNode, followerNode, syncer
}

func BenchmarkMemoryReplicationLog_Append(b *testing.B) {
	log := setupReplicationLog(b)
	entries := prepareLogEntries(b, b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := log.Append(entries[i])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemoryReplicationLog_Get(b *testing.B) {
	log := setupReplicationLog(b)
	entries := prepareLogEntries(b, 1000)

	for _, entry := range entries {
		err := log.Append(entry)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := uint64((i % 1000) + 1)
		_, err := log.Get(index)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemoryReplicationLog_GetFrom(b *testing.B) {
	log := setupReplicationLog(b)
	entries := prepareLogEntries(b, 1000)

	for _, entry := range entries {
		err := log.Append(entry)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		startIndex := uint64((i % 500) + 1)
		_, err := log.GetFrom(startIndex, 50)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReplicationLog_Serialize(b *testing.B) {
	entries := prepareLogEntries(b, 100)
	log := setupReplicationLog(b)

	for _, entry := range entries {
		log.Append(entry)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := log.Serialize()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReplicationLog_Deserialize(b *testing.B) {
	entries := prepareLogEntries(b, 100)
	log := setupReplicationLog(b)

	for _, entry := range entries {
		log.Append(entry)
	}

	serialized, err := log.Serialize()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		newLog := NewMemoryReplicationLog()
		err := newLog.Deserialize(serialized)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLeaderNode_AddFollower(b *testing.B) {
	leaderNode, _, _ := setupLeaderFollower(b)

	err := leaderNode.Start()
	if err != nil {
		b.Fatal(err)
	}
	defer leaderNode.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		followerID := "follower-" + strconv.Itoa(i)
		followerAddr := "127.0.0.1:" + strconv.Itoa(8000+i)
		err := leaderNode.AddFollower(followerID, followerAddr)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLeaderNode_ApplyEntry(b *testing.B) {
	leaderNode, _, _ := setupLeaderFollower(b)
	entries := prepareLogEntries(b, b.N)

	err := leaderNode.Start()
	if err != nil {
		b.Fatal(err)
	}
	defer leaderNode.Stop()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := leaderNode.ApplyEntry(ctx, entries[i])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLeaderNode_SendHeartbeat(b *testing.B) {
	leaderNode, _, _ := setupLeaderFollower(b)

	err := leaderNode.Start()
	if err != nil {
		b.Fatal(err)
	}
	defer leaderNode.Stop()

	for i := 0; i < 5; i++ {
		followerID := "follower-" + strconv.Itoa(i)
		followerAddr := "127.0.0.1:" + strconv.Itoa(8100+i)
		leaderNode.AddFollower(followerID, followerAddr)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		leaderNode.SendHeartbeat(ctx)
	}
}

func BenchmarkFollowerNode_ApplyEntry(b *testing.B) {
	_, followerNode, _ := setupLeaderFollower(b)
	entries := prepareLogEntries(b, b.N)

	err := followerNode.Start()
	if err != nil {
		b.Fatal(err)
	}
	defer followerNode.Stop()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := followerNode.ApplyEntry(ctx, entries[i])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFollowerNode_HandleHeartbeat(b *testing.B) {
	_, followerNode, _ := setupLeaderFollower(b)

	err := followerNode.Start()
	if err != nil {
		b.Fatal(err)
	}
	defer followerNode.Stop()

	//heartbeat := &Heartbeat{
	//	LeaderID:  "leader-1",
	//	Term:      1,
	//	Timestamp: time.Now().UnixNano(),
	//}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		followerNode.HandleHeartbeat(ctx, "leader-1", 1)
	}
}

func BenchmarkMemorySyncer_RecordSetOperation(b *testing.B) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := NewMemorySyncer("test-node", cache, log, WithNodeID("test-node"))

	key := "test-key"
	value := []byte("test-value")
	expiration := time.Minute

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := syncer.RecordSetOperation(key, value, expiration, 1)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemorySyncer_RecordDelOperation(b *testing.B) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()
	syncer := NewMemorySyncer("test-node", cache, log, WithNodeID("test-node"))

	key := "test-key"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := syncer.RecordDelOperation(key, 1)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParallelLogAppend(b *testing.B) {
	log := setupReplicationLog(b)
	entries := prepareLogEntries(b, b.N)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			entry := entries[i%len(entries)]
			entry.Key = entry.Key + "-" + strconv.Itoa(i)
			err := log.Append(entry)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}
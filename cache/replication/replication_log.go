package replication

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// MemoryReplicationLog 内存实现的复制日志
type MemoryReplicationLog struct {
	entries  []*ReplicationEntry
	mu       sync.RWMutex
	lastTerm uint64
}

// NewMemoryReplicationLog 创建新的内存复制日志
func NewMemoryReplicationLog() *MemoryReplicationLog {
	return &MemoryReplicationLog{
		entries: make([]*ReplicationEntry, 0),
	}
}

// Append 追加日志条目
func (l *MemoryReplicationLog) Append(entry *ReplicationEntry) error {
	if entry == nil {
		return errors.New("cannot apply nil entry")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 如果是空日志，索引从1开始
	if len(l.entries) == 0 {
		entry.Index = 1
	} else {
		// 否则索引为最后一条日志的索引加1
		entry.Index = l.entries[len(l.entries)-1].Index + 1
	}

	// 更新最后任期
	if entry.Term > l.lastTerm {
		l.lastTerm = entry.Term
	}

	// 如果时间戳未设置，添加当前时间
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// 追加日志
	l.entries = append(l.entries, entry)

	return nil
}

// Get 获取指定索引的日志
func (l *MemoryReplicationLog) Get(index uint64) (*ReplicationEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 检查日志是否为空
	if len(l.entries) == 0 {
		return nil, errors.New("log is empty")
	}

	// 检查索引是否有效
	firstIndex := l.entries[0].Index
	if index < firstIndex {
		return nil, errors.New("requested index is compacted")
	}

	adjustedIndex := index - firstIndex
	if int(adjustedIndex) >= len(l.entries) {
		return nil, errors.New("index out of range")
	}

	return l.entries[adjustedIndex], nil
}

// GetFrom 获取从指定索引开始的所有日志
func (l *MemoryReplicationLog) GetFrom(startIndex uint64, maxCount int) ([]*ReplicationEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 检查日志是否为空
	if len(l.entries) == 0 {
		return nil, errors.New("log is empty")
	}

	// 检查起始索引是否有效
	firstIndex := l.entries[0].Index
	if startIndex < firstIndex {
		return nil, errors.New("requested starting index is compacted")
	}

	adjustedStartIndex := startIndex - firstIndex
	if int(adjustedStartIndex) >= len(l.entries) {
		return []*ReplicationEntry{}, nil // 没有匹配的条目，返回空数组
	}

	// 计算结束索引
	endPos := int(adjustedStartIndex) + maxCount
	if endPos > len(l.entries) || maxCount <= 0 {
		endPos = len(l.entries)
	}

	// 返回结果副本
	result := make([]*ReplicationEntry, endPos-int(adjustedStartIndex))
	copy(result, l.entries[adjustedStartIndex:endPos])

	return result, nil
}

// GetLastIndex 获取最后一条日志的索引
func (l *MemoryReplicationLog) GetLastIndex() uint64 {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if len(l.entries) == 0 {
		return 0
	}

	return l.entries[len(l.entries)-1].Index
}

// GetLastTerm 获取最后一条日志的任期
func (l *MemoryReplicationLog) GetLastTerm() uint64 {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.lastTerm
}

// Truncate 截断日志到指定索引
func (l *MemoryReplicationLog) Truncate(index uint64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 检查日志是否为空
	if len(l.entries) == 0 {
		return nil // 空日志无需截断
	}

	// 检查索引是否有效
	firstIndex := l.entries[0].Index
	if index < firstIndex {
		return errors.New("cannot truncate compacted entries")
	}

	// 如果索引超出范围，不做任何操作
	adjustedIndex := index - firstIndex
	if int(adjustedIndex) >= len(l.entries) {
		return nil
	}

	// 截断后索引之后的条目
	l.entries = l.entries[:int(adjustedIndex)+1]

	// 更新lastTerm
	if len(l.entries) > 0 {
		l.lastTerm = l.entries[len(l.entries)-1].Term
	} else {
		l.lastTerm = 0
	}

	return nil
}

// Serialize 序列化日志
func (l *MemoryReplicationLog) Serialize() ([]byte, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return json.Marshal(l.entries)
}

// Deserialize 反序列化日志
func (l *MemoryReplicationLog) Deserialize(data []byte) error {
	var entries []*ReplicationEntry
	err := json.Unmarshal(data, &entries)
	if err != nil {
		return fmt.Errorf("failed to deserialize log: %w", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.entries = entries

	// 更新lastTerm
	if len(entries) > 0 {
		l.lastTerm = entries[len(entries)-1].Term
	} else {
		l.lastTerm = 0
	}

	return nil
}

// Count 返回日志条目数量
func (l *MemoryReplicationLog) Count() int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return len(l.entries)
}

// Clear 清空日志
func (l *MemoryReplicationLog) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.entries = make([]*ReplicationEntry, 0)
	l.lastTerm = 0
}

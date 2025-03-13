package cache

import (
	"fmt"
	"hash/crc32"
	"hash/fnv"
	"strconv"
	"testing"
)

// 基准测试部分

func BenchmarkConsistentHash_Get(b *testing.B) {
	hash := NewConsistentHash(100, nil)

	// 添加100个节点
	for i := 0; i < 100; i++ {
		hash.Add(fmt.Sprintf("node-%d", i), 1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash.Get(strconv.Itoa(i))
	}
}

func BenchmarkConsistentHash_Add(b *testing.B) {
	hash := NewConsistentHash(100, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nodeName := fmt.Sprintf("bench-node-%d", i)
		hash.Add(nodeName, 1)
	}
}

func BenchmarkConsistentHash_GetN(b *testing.B) {
	hash := NewConsistentHash(100, nil)

	// 添加100个节点
	for i := 0; i < 100; i++ {
		hash.Add(fmt.Sprintf("node-%d", i), 1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash.GetN(strconv.Itoa(i), 3)
	}
}

func BenchmarkDifferentHashFunctions(b *testing.B) {
	// 测试不同哈希函数的性能
	benchmarks := []struct {
		name     string
		hashFunc HashFunc
	}{
		{"CRC32", crc32.ChecksumIEEE},
		{"FNV32", func(data []byte) uint32 {
			h := fnv.New32a()
			h.Write(data)
			return h.Sum32()
		}},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			hash := NewConsistentHash(100, bm.hashFunc)

			// 添加100个节点
			for i := 0; i < 100; i++ {
				hash.Add(fmt.Sprintf("node-%d", i), 1)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				hash.Get(strconv.Itoa(i))
			}
		})
	}
}

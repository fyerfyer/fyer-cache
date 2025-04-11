package cache

import (
	"fmt"
	"hash/crc32"
	"hash/fnv"
	"strconv"
	"testing"
)

func BenchmarkConsistentHash_Get(b *testing.B) {
	hash := NewConsistentHash(100, nil)

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

func BenchmarkConsistentHash_Remove(b *testing.B) {
	const nodeCount = 1000

	hash := NewConsistentHash(100, nil)
	nodeNames := make([]string, nodeCount)

	for i := 0; i < nodeCount; i++ {
		nodeName := fmt.Sprintf("bench-node-%d", i)
		nodeNames[i] = nodeName
		hash.Add(nodeName, 1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := i % nodeCount

		hash.Remove(nodeNames[idx])

		hash.Add(nodeNames[idx], 1)
	}
}

func BenchmarkConsistentHash_GetN(b *testing.B) {
	hash := NewConsistentHash(100, nil)

	for i := 0; i < 100; i++ {
		hash.Add(fmt.Sprintf("node-%d", i), 1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash.GetN(strconv.Itoa(i), 3)
	}
}

func BenchmarkConsistentHash_GetNodes(b *testing.B) {
	hash := NewConsistentHash(100, nil)

	for i := 0; i < 100; i++ {
		hash.Add(fmt.Sprintf("node-%d", i), 1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash.GetNodes()
	}
}

func BenchmarkConsistentHash_LargeCluster(b *testing.B) {
	hash := NewConsistentHash(100, nil)

	for i := 0; i < 1000; i++ {
		hash.Add(fmt.Sprintf("node-%d", i), 1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash.Get(strconv.Itoa(i))
	}
}

func BenchmarkConsistentHash_WeightedNodes(b *testing.B) {
	hash := NewConsistentHash(100, nil)

	hash.Add("node-1", 1)
	hash.Add("node-2", 2)
	hash.Add("node-3", 3)
	hash.Add("node-4", 4)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash.Get(strconv.Itoa(i))
	}
}

func BenchmarkDifferentHashFunctions(b *testing.B) {
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

func BenchmarkConsistentHash_ReplicaCount(b *testing.B) {
	benchmarks := []struct {
		replicas int
	}{
		{10},
		{50},
		{100},
		{500},
		{1000},
	}

	for _, bm := range benchmarks {
		b.Run(fmt.Sprintf("Replicas-%d", bm.replicas), func(b *testing.B) {
			hash := NewConsistentHash(bm.replicas, nil)

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

func BenchmarkConsistentHash_NodeCount(b *testing.B) {
	benchmarks := []struct {
		nodes int
	}{
		{10},
		{50},
		{100},
		{500},
		{1000},
	}

	for _, bm := range benchmarks {
		b.Run(fmt.Sprintf("Nodes-%d", bm.nodes), func(b *testing.B) {
			hash := NewConsistentHash(100, nil)

			for i := 0; i < bm.nodes; i++ {
				hash.Add(fmt.Sprintf("node-%d", i), 1)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				hash.Get(strconv.Itoa(i))
			}
		})
	}
}
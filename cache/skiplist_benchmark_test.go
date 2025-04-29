package cache

import (
	"fmt"
	"testing"
)

func BenchmarkSkipList_Set(b *testing.B) {
	sl := newSkipList()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench-key-%d", i)
		sl.Set(key, &cacheItem{value: i})
	}
}

func BenchmarkSkipList_Get(b *testing.B) {
	sl := newSkipList()

	// 预填充数据
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("bench-key-%d", i)
		sl.Set(key, &cacheItem{value: i})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench-key-%d", i%10000)
		sl.Get(key)
	}
}

func BenchmarkSkipList_Range(b *testing.B) {
	sl := newSkipList()

	// 预填充有序数据
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("bench-key-%05d", i)
		sl.Set(key, &cacheItem{value: i})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := fmt.Sprintf("bench-key-%05d", i%5000)
		end := fmt.Sprintf("bench-key-%05d", (i%5000)+100)

		sl.Range(start, end, -1, func(key string, value *cacheItem) bool {
			return true
		})
	}
}
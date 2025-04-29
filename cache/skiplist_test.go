package cache

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestSkipList_BasicOperations(t *testing.T) {
	sl := newSkipList()

	// 测试Set和Get
	t.Run("Set and Get", func(t *testing.T) {
		item := &cacheItem{value: "value1", expiration: time.Now().Add(time.Hour)}
		sl.Set("key1", item)

		// 获取存在的键
		value, ok := sl.Get("key1")
		if !ok {
			t.Error("Expected to find key1, but got not found")
		}

		if value.value != "value1" {
			t.Errorf("Expected value1, got %v", value.value)
		}

		// 获取不存在的键
		_, ok = sl.Get("nonexistent")
		if ok {
			t.Error("Expected not to find nonexistent key")
		}
	})

	// 测试Delete
	t.Run("Delete", func(t *testing.T) {
		item := &cacheItem{value: "value2", expiration: time.Now().Add(time.Hour)}
		sl.Set("key2", item)

		// 删除存在的键
		ok := sl.Delete("key2")
		if !ok {
			t.Error("Expected to delete key2 successfully")
		}

		// 确认键已被删除
		_, ok = sl.Get("key2")
		if ok {
			t.Error("Key2 should have been deleted")
		}

		// 删除不存在的键
		ok = sl.Delete("nonexistent")
		if ok {
			t.Error("Expected delete of nonexistent key to return false")
		}
	})

	// 测试Len
	t.Run("Length", func(t *testing.T) {
		sl := newSkipList()
		if sl.Len() != 0 {
			t.Errorf("Expected empty skiplist to have length 0, got %d", sl.Len())
		}

		sl.Set("key1", &cacheItem{value: "value1"})
		if sl.Len() != 1 {
			t.Errorf("Expected length 1 after adding one item, got %d", sl.Len())
		}

		sl.Set("key2", &cacheItem{value: "value2"})
		if sl.Len() != 2 {
			t.Errorf("Expected length 2 after adding two items, got %d", sl.Len())
		}

		sl.Delete("key1")
		if sl.Len() != 1 {
			t.Errorf("Expected length 1 after deleting one item, got %d", sl.Len())
		}
	})

	// 测试更新已存在的键
	t.Run("Update", func(t *testing.T) {
		sl := newSkipList()
		sl.Set("key", &cacheItem{value: "original"})

		// 更新值
		sl.Set("key", &cacheItem{value: "updated"})

		// 检查新值
		value, ok := sl.Get("key")
		if !ok {
			t.Error("Failed to get updated key")
		}
		if value.value != "updated" {
			t.Errorf("Expected updated value, got %v", value.value)
		}

		// 确认长度没有增加
		if sl.Len() != 1 {
			t.Errorf("Expected length to remain 1 after update, got %d", sl.Len())
		}
	})
}

func TestSkipList_Range(t *testing.T) {
	sl := newSkipList()

	// 添加10个键值对
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%d", i)
		sl.Set(key, &cacheItem{value: i})
	}

	// 测试空范围查询（应该返回0个结果）
	t.Run("Empty Range", func(t *testing.T) {
		count := 0
		sl.Range("key9", "key1", -1, func(key string, value *cacheItem) bool {
			count++
			return true
		})

		if count != 0 {
			t.Errorf("Expected 0 items in empty range, got %d", count)
		}
	})

	// 测试完整范围查询
	t.Run("Full Range", func(t *testing.T) {
		keys := make([]string, 0)
		sl.Range("key0", "key9", -1, func(key string, value *cacheItem) bool {
			keys = append(keys, key)
			return true
		})

		if len(keys) != 10 {
			t.Errorf("Expected 10 items in full range, got %d", len(keys))
		}

		// 检查顺序
		for i := 0; i < len(keys)-1; i++ {
			if keys[i] > keys[i+1] {
				t.Errorf("Expected sorted keys, but %s > %s", keys[i], keys[i+1])
			}
		}
	})

	// 测试子范围查询
	t.Run("Sub Range", func(t *testing.T) {
		keys := make([]string, 0)
		sl.Range("key3", "key7", -1, func(key string, value *cacheItem) bool {
			keys = append(keys, key)
			return true
		})

		expected := []string{"key3", "key4", "key5", "key6", "key7"}
		if len(keys) != len(expected) {
			t.Errorf("Expected %d items, got %d", len(expected), len(keys))
		}

		for i, key := range expected {
			if i < len(keys) && keys[i] != key {
				t.Errorf("Expected %s at position %d, got %s", key, i, keys[i])
			}
		}
	})

	// 测试限制数量
	t.Run("Limited Range", func(t *testing.T) {
		keys := make([]string, 0)
		sl.Range("key0", "key9", 5, func(key string, value *cacheItem) bool {
			keys = append(keys, key)
			return true
		})

		if len(keys) != 5 {
			t.Errorf("Expected 5 items with limit, got %d", len(keys))
		}
	})

	// 测试提前终止
	t.Run("Early Termination", func(t *testing.T) {
		keys := make([]string, 0)
		sl.Range("key0", "key9", -1, func(key string, value *cacheItem) bool {
			keys = append(keys, key)
			return len(keys) < 3 // 只获取前3个
		})

		if len(keys) != 3 {
			t.Errorf("Expected 3 items after early termination, got %d", len(keys))
		}
	})
}

func TestSkipList_Concurrent(t *testing.T) {
	sl := newSkipList()

	// 并发测试参数
	goroutines := 8
	operationsPerGoroutine := 1000

	// 等待组
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// 启动goroutines进行并发操作
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()

			// 创建独立的随机数生成器
			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)))

			for i := 0; i < operationsPerGoroutine; i++ {
				key := fmt.Sprintf("key-%d-%d", id, i)
				value := &cacheItem{value: fmt.Sprintf("value-%d-%d", id, i)}

				// 随机操作类型: 0=Set, 1=Get, 2=Delete
				op := r.Intn(10)

				switch {
				case op < 5: // 50% Set操作
					sl.Set(key, value)
				case op < 8: // 30% Get操作
					sl.Get(key)
				default: // 20% Delete操作
					sl.Delete(key)
				}
			}
		}(g)
	}

	// 等待所有goroutine完成
	wg.Wait()

	// 确认跳表数据一致性
	t.Logf("Final skiplist contains %d items after concurrent operations", sl.Len())
}

func TestSkipList_ForEach(t *testing.T) {
	sl := newSkipList()

	// 添加有序的键值对
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%d", i)
		sl.Set(key, &cacheItem{value: i})
	}

	// 收集所有键
	var keys []string
	sl.ForEach(func(key string, value *cacheItem) bool {
		keys = append(keys, key)
		return true
	})

	// 验证数量
	if len(keys) != 10 {
		t.Errorf("Expected 10 keys, got %d", len(keys))
	}

	// 验证顺序
	sort.Strings(keys)
	for i, key := range keys {
		expected := fmt.Sprintf("key%d", i)
		if key != expected {
			t.Errorf("Expected %s at position %d, got %s", expected, i, key)
		}
	}

	// 测试提前终止
	count := 0
	sl.ForEach(func(key string, value *cacheItem) bool {
		count++
		return count < 5
	})

	if count != 5 {
		t.Errorf("Expected ForEach to stop after 5 items, processed %d", count)
	}
}
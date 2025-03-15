package cluster

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/stretchr/testify/require"
)

// TestCluster_ConcurrentOperations 测试集群的并发操作
func TestCluster_ConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent cluster test in short mode")
	}

	// 使用较短的时间间隔加快测试速度
	config := &NodeConfig{
		GossipInterval:  50 * time.Millisecond,
		SyncInterval:    250 * time.Millisecond,
		ProbeInterval:   50 * time.Millisecond,
		ProbeTimeout:    25 * time.Millisecond,
		SuspicionMult:   1,
		SyncConcurrency: 2,
		VirtualNodes:    10,
		MaxRetries:      2,
		RetryInterval:   100 * time.Millisecond,
		EventBufferSize: 100,
	}

	// 为测试寻找可用端口
	ports, err := getAvailablePorts(3)
	require.NoError(t, err, "Failed to get available ports")

	// 创建3个节点与内存缓存
	nodes := make([]*Node, 3)
	caches := make([]*cache.MemoryCache, 3)

	for i := 0; i < 3; i++ {
		// 创建缓存
		caches[i] = cache.NewMemoryCache(
			cache.WithShardCount(32),
			cache.WithCleanupInterval(500*time.Millisecond),
			cache.WithAsyncCleanup(true),
		)

		// 复制配置
		nodeConfig := *config
		nodeConfig.ID = fmt.Sprintf("node-%d", i+1)
		nodeConfig.BindAddr = fmt.Sprintf("127.0.0.1:%d", ports[i])

		// 创建节点
		node, err := NewNode(nodeConfig.ID, caches[i], WithBindAddr(nodeConfig.BindAddr))
		require.NoError(t, err, "Failed to create node %d", i+1)
		nodes[i] = node
	}

	// 启动所有节点
	for i, node := range nodes {
		err := node.Start()
		require.NoError(t, err, "Failed to start node %d", i+1)
	}

	// 确保适当的清理
	defer func() {
		for _, node := range nodes {
			_ = node.Stop()
		}
		for _, c := range caches {
			_ = c.Close()
		}
	}()

	t.Log("All nodes started successfully")

	// 第一个节点形成自己的集群
	err = nodes[0].Join(nodes[0].config.BindAddr)
	require.NoError(t, err, "Failed to bootstrap first node")

	// 其他节点加入第一个节点
	for i := 1; i < len(nodes); i++ {
		err = nodes[i].Join(nodes[0].config.BindAddr)
		require.NoError(t, err, "Failed to join node %d to cluster", i+1)
	}

	// 等待gossip传播成员信息
	err = waitForFullClusterFormation(t, nodes, 3, 5*time.Second)
	require.NoError(t, err, "Cluster did not form completely within timeout")

	ctx := context.Background()

	// 并发操作参数
	const (
		numGoroutines   = 20  // 并发操作的goroutine数量
		opsPerGoroutine = 100 // 每个goroutine执行的操作次数
		keyRange        = 500 // 键的范围
	)

	// 启动并发写入goroutines
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// 记录成功和失败的操作
	var (
		successOps int32
		failedOps  int32
		mu         sync.Mutex
		failures   = make(map[string]int)
	)

	// 开始计时
	startTime := time.Now()

	// 启动并发goroutines
	for g := 0; g < numGoroutines; g++ {
		go func(id int) {
			defer wg.Done()

			// 为每个goroutine创建一个新的随机源
			rnd := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)))
			nodeIndex := id % len(nodes)
			cache := caches[nodeIndex]

			localSuccess := 0
			localFailures := make(map[string]int)

			for i := 0; i < opsPerGoroutine; i++ {
				// 选择操作: 0=Set, 1=Get, 2=Delete
				op := rnd.Intn(3)
				keyNum := rnd.Intn(keyRange)
				key := fmt.Sprintf("key-%d", keyNum)
				value := fmt.Sprintf("value-%d-%d", id, i)

				var err error
				switch op {
				case 0: // Set
					err = cache.Set(ctx, key, value, 10*time.Second)
					if err == nil {
						// 验证写入是否成功
						time.Sleep(time.Millisecond) // 给一点时间让缓存操作完成
						_, verifyErr := cache.Get(ctx, key)
						if verifyErr != nil {
							err = fmt.Errorf("verification failed: %v", verifyErr)
						}
					}
				case 1: // Get
					_, err = cache.Get(ctx, key)
					// 忽略键不存在的错误
					if err != nil && err.Error() == "key not found" {
						err = nil
					}
				case 2: // Delete
					err = cache.Del(ctx, key)
				}

				if err == nil {
					localSuccess++
				} else {
					errMsg := fmt.Sprintf("%s-%d", err.Error(), op)
					localFailures[errMsg]++
				}
			}

			// 更新全局计数器
			mu.Lock()
			successOps += int32(localSuccess)
			failedOps += int32(opsPerGoroutine - localSuccess)
			for k, v := range localFailures {
				failures[k] += v
			}
			mu.Unlock()
		}(g)
	}

	// 等待所有goroutines完成
	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("Concurrent test completed in %v", duration)
	t.Logf("Total operations: %d, Successful: %d, Failed: %d",
		numGoroutines*opsPerGoroutine, successOps, failedOps)

	if len(failures) > 0 {
		t.Log("Failure breakdown:")
		for errMsg, count := range failures {
			t.Logf("  - %s: %d", errMsg, count)
		}
	}

	// 尝试读取数据，验证集群的一致性
	consistencyOps := 0
	consistencySuccess := 0

	for i := 0; i < keyRange; i += 10 { // 抽样检查
		key := fmt.Sprintf("key-%d", i)
		for _, cache := range caches {
			_, err := cache.Get(ctx, key)
			consistencyOps++
			if err == nil {
				consistencySuccess++
				// 一旦在一个节点上找到，退出此key的检查
				break
			}
		}
	}

	t.Logf("Consistency check: %d/%d keys found (%d%%)",
		consistencySuccess, consistencyOps, consistencySuccess*100/consistencyOps)

	// 测试节点故障恢复
	t.Log("Testing node failure and recovery...")

	// 停止一个节点
	t.Log("Stopping node 2")
	err = nodes[1].Stop()
	require.NoError(t, err)

	// 等待故障检测
	t.Log("Waiting for failure detection...")
	time.Sleep(1 * time.Second)

	// 启动更多的操作
	var wg2 sync.WaitGroup
	wg2.Add(numGoroutines / 2)

	for g := 0; g < numGoroutines/2; g++ {
		go func(id int) {
			defer wg2.Done()
			rnd := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)))
			nodeIndex := id % 2 * 2 // 使用节点0和节点2 (跳过节点1)
			cache := caches[nodeIndex]

			for i := 0; i < opsPerGoroutine/2; i++ {
				keyNum := rnd.Intn(keyRange)
				key := fmt.Sprintf("key-%d", keyNum)
				value := fmt.Sprintf("value-recovery-%d-%d", id, i)

				// 只做写入操作
				_ = cache.Set(ctx, key, value, 10*time.Second)
				time.Sleep(time.Millisecond) // 控制速率
			}
		}(g)
	}

	wg2.Wait()

	// 重启停止的节点
	t.Log("Restarting node 2")
	err = nodes[1].Start()
	require.NoError(t, err)
	err = nodes[1].Join(nodes[0].config.BindAddr)
	require.NoError(t, err)

	// 等待节点重新加入
	time.Sleep(2 * time.Second)

	// 最终一致性检查
	finalConsistencyOps := 0
	finalConsistencySuccess := 0
	nodeHits := make([]int, len(nodes))

	// 对每个键，在每个节点上尝试获取
	for i := 0; i < keyRange; i += 5 { // 抽样检查
		key := fmt.Sprintf("key-%d", i)
		found := false

		for j, cache := range caches {
			_, err := cache.Get(ctx, key)
			finalConsistencyOps++

			if err == nil {
				nodeHits[j]++
				found = true
			}
		}

		if found {
			finalConsistencySuccess++
		}
	}

	t.Logf("Final consistency check: %d/%d keys found across any node",
		finalConsistencySuccess, keyRange/5)

	t.Logf("Node hit distribution: node1=%d, node2=%d, node3=%d",
		nodeHits[0], nodeHits[1], nodeHits[2])

	// 检查重新加入的节点是否能正常工作
	if nodeHits[1] == 0 {
		t.Errorf("Node 2 (rejoined node) didn't serve any keys")
	}

	t.Log("Concurrent cluster test completed successfully")
}

// 等待特定时间直到集群完全形成
func waitForConcurrentClusterFormation(t *testing.T, nodes []*Node, expectedCount int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		allFormed := true
		for _, node := range nodes {
			members := node.Members()
			if len(members) < expectedCount {
				allFormed = false
				break
			}
		}

		if allFormed {
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}

	// 打印最终状态用于诊断
	for i, node := range nodes {
		members := node.Members()
		t.Logf("Node %d sees %d members: %v", i+1, len(members), memberIDs(members))
	}

	return fmt.Errorf("cluster did not completely form within %v, expected %d members", timeout, expectedCount)
}

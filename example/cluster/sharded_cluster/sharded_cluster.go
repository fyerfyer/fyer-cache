package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/fyerfyer/fyer-cache/cache/cluster"
	"github.com/fyerfyer/fyer-cache/cache/cluster/integration"
)

func main() {
	fmt.Println("Sharded Cluster Cache Example")
	fmt.Println("============================")

	// 获取可用端口以避免冲突
	ports, err := getAvailablePorts(3)
	if err != nil {
		log.Fatalf("Failed to get available ports: %v", err)
	}

	// 创建3个分片集群缓存节点
	nodes := make([]*integration.ShardedClusterCache, 3)

	fmt.Println("Creating three sharded cluster nodes...")
	for i := 0; i < 3; i++ {
		nodeID := fmt.Sprintf("node-%d", i+1)
		bindAddr := fmt.Sprintf("127.0.0.1:%d", ports[i])

		// 为每个节点创建本地缓存
		localCache := cache.NewMemoryCache()

		// 创建集群节点配置选项
		clusterOptions := []cluster.NodeOption{
			cluster.WithNodeID(nodeID),
			cluster.WithBindAddr(bindAddr),
			// 为了演示目的，使用较短的时间间隔
			cluster.WithProbeInterval(1 * time.Second),
			cluster.WithGossipInterval(500 * time.Millisecond),
		}

		// 创建分片缓存选项
		cacheOptions := []integration.ShardedClusterCacheOption{
			integration.WithVirtualNodeCount(100), // 每个节点100个虚拟节点
			integration.WithReplicaFactor(2),      // 每个键存储2份副本
		}

		// 创建分片集群缓存节点
		shardedCache, err := integration.NewShardedClusterCache(
			localCache, nodeID, bindAddr, clusterOptions, cacheOptions,
		)
		if err != nil {
			log.Fatalf("Failed to create node %s: %v", nodeID, err)
		}

		nodes[i] = shardedCache
		fmt.Printf("Created node %s at %s\n", nodeID, bindAddr)
	}

	// 启动所有节点
	fmt.Println("\nStarting all nodes...")
	for _, node := range nodes {
		if err := node.Start(); err != nil {
			log.Fatalf("Failed to start node %s: %v", node.GetNodeID(), err)
		}
		fmt.Printf("Started node %s\n", node.GetNodeID())
	}

	// 确保优雅退出，停止所有节点
	defer func() {
		fmt.Println("\nStopping all nodes...")
		for _, node := range nodes {
			if err := node.Stop(); err != nil {
				log.Printf("Error stopping node %s: %v", node.GetNodeID(), err)
			}
			fmt.Printf("Stopped node %s\n", node.GetNodeID())
		}
	}()

	// 让第一个节点形成自己的集群
	fmt.Println("\nNode 1 is forming its own cluster...")
	err = nodes[0].Join(nodes[0].GetAddress())
	if err != nil {
		log.Fatalf("Failed to bootstrap first node: %v", err)
	}
	fmt.Printf("Node %s formed its own cluster\n", nodes[0].GetNodeID())

	// 等待一小段时间让第一个节点稳定
	time.Sleep(1 * time.Second)

	// 其他节点加入第一个节点的集群
	fmt.Println("\nOther nodes joining the cluster...")
	for i := 1; i < len(nodes); i++ {
		err = nodes[i].Join(nodes[0].GetAddress())
		if err != nil {
			log.Fatalf("Failed to join node %s to cluster: %v", nodes[i].GetNodeID(), err)
		}
		fmt.Printf("Node %s joined the cluster\n", nodes[i].GetNodeID())

		// 给一点时间让节点加入传播
		time.Sleep(500 * time.Millisecond)
	}

	// 等待集群稳定
	fmt.Println("\nWaiting for cluster to stabilize...")
	time.Sleep(2 * time.Second)

	// 显示集群成员信息
	fmt.Println("\nCluster membership from each node's perspective:")
	for i, node := range nodes {
		members := node.Members()
		fmt.Printf("Node %d sees %d members: %v\n", i+1, len(members), memberIDs(members))
	}

	// 演示分片数据分布
	fmt.Println("\nDemonstrating sharded data distribution across the cluster...")
	ctx := context.Background()

	// 存储一些数据
	fmt.Println("\nStoring data in the cluster...")
	keyCount := 10
	for i := 1; i <= keyCount; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)

		// 随机选择一个节点进行写入
		nodeIndex := i % len(nodes)
		err := nodes[nodeIndex].Set(ctx, key, value, 10*time.Minute)
		if err != nil {
			fmt.Printf("Failed to set %s: %v\n", key, err)
			continue
		}

		fmt.Printf("Set %s = %s through node %d\n", key, value, nodeIndex+1)
	}

	// 从不同节点读取数据
	fmt.Println("\nRetrieving data from the cluster (using different nodes)...")
	for i := 1; i <= keyCount; i++ {
		key := fmt.Sprintf("key-%d", i)

		// 通过与写入不同的节点读取数据
		nodeIndex := (i + 1) % len(nodes)
		value, err := nodes[nodeIndex].Get(ctx, key)
		if err != nil {
			fmt.Printf("Failed to get %s through node %d: %v\n", key, nodeIndex+1, err)
			continue
		}

		fmt.Printf("Got %s = %s through node %d\n", key, value, nodeIndex+1)
	}

	// 显示节点的副本因子
	fmt.Println("\nReplica factor configuration:")
	for i, node := range nodes {
		fmt.Printf("Node %d replica factor: %d\n", i+1, node.ReplicaFactor())
	}

	// 测试删除操作
	fmt.Println("\nDemonstrating deletion...")
	deleteKey := "key-3"

	// 通过任意节点删除数据
	err = nodes[0].Del(ctx, deleteKey)
	if err != nil {
		fmt.Printf("Failed to delete %s: %v\n", deleteKey, err)
	} else {
		fmt.Printf("Deleted %s through node 1\n", deleteKey)
	}

	// 验证数据已被删除
	time.Sleep(500 * time.Millisecond) // 等待删除操作传播

	fmt.Println("Verifying deletion across all nodes...")
	for i, node := range nodes {
		_, err := node.Get(ctx, deleteKey)
		if err != nil {
			fmt.Printf("Node %d confirms %s is deleted\n", i+1, deleteKey)
		} else {
			fmt.Printf("Node %d still has %s (may be replication delay)\n", i+1, deleteKey)
		}
	}

	// 演示节点离开集群
	fmt.Println("\nDemonstrating node leaving the cluster...")
	err = nodes[2].Leave()
	if err != nil {
		log.Printf("Error when node 3 leaves: %v", err)
	} else {
		fmt.Println("Node 3 has left the cluster")
	}

	// 等待集群成员信息更新
	fmt.Println("Waiting for membership changes to propagate...")
	time.Sleep(2 * time.Second)

	// 再次检查成员信息
	fmt.Println("\nCluster membership after node left:")
	for i := 0; i < 2; i++ { // Only check the first two nodes
		members := nodes[i].Members()
		fmt.Printf("Node %d now sees %d members: %v\n", i+1, len(members), memberIDs(members))
	}

	// 测试节点数据重新分布
	fmt.Println("\nTesting data access after node departure...")
	for i := 1; i <= keyCount; i++ {
		if i == 3 { // Skip the deleted key
			continue
		}

		key := fmt.Sprintf("key-%d", i)
		value, err := nodes[0].Get(ctx, key)
		if err != nil {
			fmt.Printf("Failed to get %s after node departure: %v\n", key, err)
		} else {
			fmt.Printf("Successfully retrieved %s = %s after node departure\n", key, value)
		}
	}

	fmt.Println("\nSharded cluster cache example completed.")
	fmt.Println("Press Ctrl+C to exit.")

	// 让程序继续运行直到手动终止
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait() // 这里不会真的等待，只是让main不退出
}

// getAvailablePorts 获取n个可用端口
func getAvailablePorts(n int) ([]int, error) {
	ports := make([]int, n)
	listeners := make([]*net.TCPListener, n)

	// 找到n个可用端口
	for i := 0; i < n; i++ {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			// 关闭已创建的监听器
			for j := 0; j < i; j++ {
				listeners[j].Close()
			}
			return nil, fmt.Errorf("failed to find available port: %w", err)
		}

		listeners[i] = listener.(*net.TCPListener)
		ports[i] = listeners[i].Addr().(*net.TCPAddr).Port
	}

	// 关闭所有监听器，释放端口
	for i := 0; i < n; i++ {
		listeners[i].Close()
	}

	return ports, nil
}

// memberIDs 提取成员ID列表，用于打印
func memberIDs(members []cache.NodeInfo) []string {
	ids := make([]string, len(members))
	for i, member := range members {
		ids[i] = member.ID
	}
	return ids
}

// getNodeInfo 获取ShardedClusterCache的NodeInfo的辅助函数
func getNodeInfo(s *integration.ShardedClusterCache) cache.NodeInfo {
	addr := s.GetAddress()

	return cache.NodeInfo{
		ID:   s.GetNodeID(),
		Addr: addr,
		Metadata: map[string]string{
			"local": "true",
		},
	}
}
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/fyerfyer/fyer-cache/cache/cluster/integration"
	"github.com/fyerfyer/fyer-cache/cache/replication"
)

func main() {
	fmt.Println("Leader-Follower Replication Example")
	fmt.Println("==================================")

	// 获取可用端口以避免冲突
	ports, err := getAvailablePorts(3) // 2个节点 + 1个同步服务器端口
	if err != nil {
		log.Fatalf("Failed to get available ports: %v", err)
	}

	// 构建节点地址
	node1Addr := fmt.Sprintf("127.0.0.1:%d", ports[0])
	node2Addr := fmt.Sprintf("127.0.0.1:%d", ports[1])
	syncerAddr := fmt.Sprintf("127.0.0.1:%d", ports[2])

	fmt.Printf("Setting up leader node at %s\n", node1Addr)
	fmt.Printf("Setting up follower node at %s\n", node2Addr)
	fmt.Printf("Setting up syncer at %s\n", syncerAddr)

	// 创建Leader节点
	leaderCache := cache.NewMemoryCache()
	leaderCluster, err := integration.NewLeaderFollowerCluster(
		"leader-node",
		node1Addr,
		leaderCache,
		integration.WithInitialRole(replication.RoleLeader),
		integration.WithSyncerAddress(syncerAddr),
		integration.WithHeartbeatInterval(1*time.Second),
	)
	if err != nil {
		log.Fatalf("Failed to create leader node: %v", err)
	}

	// 创建Follower节点
	followerCache := cache.NewMemoryCache()
	followerCluster, err := integration.NewLeaderFollowerCluster(
		"follower-node",
		node2Addr,
		followerCache,
		integration.WithInitialRole(replication.RoleFollower),
		integration.WithSyncerAddress(syncerAddr),
		integration.WithHeartbeatInterval(1*time.Second),
	)
	if err != nil {
		log.Fatalf("Failed to create follower node: %v", err)
	}

	// 确保优雅退出
	defer func() {
		fmt.Println("\nStopping cluster nodes...")
		if err := leaderCluster.Stop(); err != nil {
			log.Printf("Error stopping leader node: %v", err)
		}
		if err := followerCluster.Stop(); err != nil {
			log.Printf("Error stopping follower node: %v", err)
		}
		fmt.Println("All nodes stopped")
	}()

	// 启动Leader节点
	if err := leaderCluster.Start(); err != nil {
		log.Fatalf("Failed to start leader node: %v", err)
	}
	fmt.Println("Leader node started successfully")

	// 让Leader节点形成集群
	if err := leaderCluster.Join(node1Addr); err != nil {
		log.Fatalf("Failed to bootstrap leader node: %v", err)
	}
	fmt.Println("Leader node formed its own cluster")

	// 稍等一下让Leader稳定
	time.Sleep(1 * time.Second)

	// 启动Follower节点
	if err := followerCluster.Start(); err != nil {
		log.Fatalf("Failed to start follower node: %v", err)
	}
	fmt.Println("Follower node started successfully")

	// Follower加入集群
	if err := followerCluster.Join(node1Addr); err != nil {
		log.Fatalf("Failed to join follower to cluster: %v", err)
	}
	fmt.Println("Follower node joined the cluster")

	// 稍等一下让集群稳定
	fmt.Println("Waiting for cluster to stabilize...")
	time.Sleep(2 * time.Second)

	// 验证Leader和Follower角色
	leaderIsLeader := leaderCluster.IsLeader()
	followerIsLeader := followerCluster.IsLeader()

	fmt.Printf("\nRole verification:\n")
	fmt.Printf("Leader node is leader: %t\n", leaderIsLeader)
	fmt.Printf("Follower node is leader: %t\n", followerIsLeader)

	if !leaderIsLeader {
		fmt.Println("Warning: Leader node is not in leader role!")
	}
	if followerIsLeader {
		fmt.Println("Warning: Follower node is unexpectedly in leader role!")
	}

	// 在Leader节点上设置一些数据
	ctx := context.Background()
	fmt.Println("\nSetting data on leader node...")

	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)

		err := leaderCluster.Set(ctx, key, value, 10*time.Minute)
		if err != nil {
			log.Printf("Failed to set %s: %v", key, err)
			continue
		}
		fmt.Printf("Set %s = %s on leader node\n", key, value)
	}

	// 触发同步（在实际应用中这可能是自动的）
	fmt.Println("\nTriggering replication to follower...")
	if err := followerCluster.SyncFromLeader(ctx); err != nil {
		log.Printf("Warning: Sync from leader failed: %v", err)
	}

	// 等待同步完成
	fmt.Println("Waiting for replication to complete...")
	time.Sleep(2 * time.Second)

	// 从Follower节点读取数据验证同步
	fmt.Println("\nReading data from follower node to verify replication:")
	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("key%d", i)
		value, err := followerCluster.Get(ctx, key)

		if err != nil {
			fmt.Printf("Failed to get %s from follower: %v\n", key, err)
		} else {
			fmt.Printf("Retrieved %s = %v from follower\n", key, value)
		}
	}

	// 在Leader上删除一个键
	fmt.Println("\nDeleting key3 on leader...")
	if err := leaderCluster.Del(ctx, "key3"); err != nil {
		log.Printf("Failed to delete key3: %v", err)
	}

	// 触发同步
	fmt.Println("Triggering replication again...")
	if err := followerCluster.SyncFromLeader(ctx); err != nil {
		log.Printf("Warning: Sync from leader failed: %v", err)
	}

	// 等待同步完成
	time.Sleep(2 * time.Second)

	// 验证删除操作已同步
	fmt.Println("\nVerifying deletion was replicated:")
	_, err = followerCluster.Get(ctx, "key3")
	if err != nil {
		fmt.Println("Verified: key3 was successfully deleted on follower")
	} else {
		fmt.Println("Warning: key3 still exists on follower, replication may have failed")
	}

	// 演示故障转移 - 停止Leader节点
	fmt.Println("\nDemonstrating failover scenario:")
	fmt.Println("Stopping leader node to simulate failure...")
	if err := leaderCluster.Stop(); err != nil {
		log.Printf("Error stopping leader node: %v", err)
	}

	// 给一些时间让Follower检测到Leader失败
	fmt.Println("Waiting for follower to detect leader failure...")
	time.Sleep(5 * time.Second)

	// 检查Follower是否已提升为Leader
	if followerCluster.IsLeader() {
		fmt.Println("Failover successful: Follower was promoted to leader")
	} else {
		fmt.Println("Failover did not occur: Follower is still in follower role")
		// 在实际场景中，可能需要手动提升
		fmt.Println("Manually promoting follower to leader...")
		if err := followerCluster.PromoteToLeader(); err != nil {
			log.Printf("Failed to promote follower: %v", err)
		} else {
			fmt.Println("Follower manually promoted to leader")
		}
	}

	// 在新Leader上设置新数据
	fmt.Println("\nSetting new data on new leader (former follower)...")
	newKey := "new-key"
	newValue := "new-value-after-failover"

	err = followerCluster.Set(ctx, newKey, newValue, 10*time.Minute)
	if err != nil {
		log.Printf("Failed to set %s: %v", newKey, err)
	} else {
		fmt.Printf("Successfully set %s = %s on new leader\n", newKey, newValue)
	}

	// 设置信号处理，优雅退出
	setupSignalHandler(followerCluster)
	fmt.Println("\nExample running. Press Ctrl+C to exit.")

	// 保持程序运行，直到收到信号
	select {}
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

// setupSignalHandler 设置信号处理函数，优雅退出
func setupSignalHandler(node *integration.LeaderFollowerCluster) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\nShutting down...")
		if err := node.Stop(); err != nil {
			log.Printf("Error stopping node: %v", err)
		} else {
			fmt.Println("Node stopped successfully")
		}
		os.Exit(0)
	}()
}
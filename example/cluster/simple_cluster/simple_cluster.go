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
	"github.com/fyerfyer/fyer-cache/cache/cluster"
)

func main() {
	fmt.Println("Simple Cluster Example")
	fmt.Println("=====================")

	// 获取可用端口以避免冲突
	ports, err := getAvailablePorts(3)
	if err != nil {
		log.Fatalf("Failed to get available ports: %v", err)
	}

	// 创建3个节点的集群
	nodes := make([]*ClusterNode, 3)
	for i := 0; i < 3; i++ {
		nodeID := fmt.Sprintf("node-%d", i+1)
		bindAddr := fmt.Sprintf("127.0.0.1:%d", ports[i])

		// 为每个节点创建本地缓存
		localCache := cache.NewMemoryCache()

		// 创建集群节点
		node, err := createClusterNode(nodeID, bindAddr, localCache)
		if err != nil {
			log.Fatalf("Failed to create node %s: %v", nodeID, err)
		}

		nodes[i] = node
		fmt.Printf("Created node %s at %s\n", nodeID, bindAddr)
	}

	// 启动所有节点
	for _, node := range nodes {
		if err := node.Start(); err != nil {
			log.Fatalf("Failed to start node %s: %v", node.ID, err)
		}
		fmt.Printf("Started node %s\n", node.ID)
	}

	// 确保优雅退出，停止所有节点
	defer func() {
		for _, node := range nodes {
			if err := node.Stop(); err != nil {
				log.Printf("Error stopping node %s: %v", node.ID, err)
			}
		}
	}()

	// 让第一个节点形成自己的集群
	if err := nodes[0].Join(nodes[0].Address); err != nil {
		log.Fatalf("Failed to bootstrap first node: %v", err)
	}
	fmt.Printf("Node %s formed its own cluster\n", nodes[0].ID)

	// 其他节点加入第一个节点的集群
	for i := 1; i < len(nodes); i++ {
		if err := nodes[i].Join(nodes[0].Address); err != nil {
			log.Fatalf("Failed to join node %s to cluster: %v", nodes[i].ID, err)
		}
		fmt.Printf("Node %s joined the cluster\n", nodes[i].ID)
	}

	// 等待集群稳定
	fmt.Println("Waiting for cluster to stabilize...")
	time.Sleep(2 * time.Second)

	// 在集群中存储一些数据
	ctx := context.Background()

	fmt.Println("\nStoring data in the cluster...")
	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)

		// 通过不同节点存储数据
		nodeIndex := i % len(nodes)
		err := nodes[nodeIndex].SetData(ctx, key, value)
		if err != nil {
			log.Printf("Failed to set %s: %v", key, err)
		} else {
			fmt.Printf("Set %s = %s through node %s\n", key, value, nodes[nodeIndex].ID)
		}
	}

	// 从不同节点读取数据
	fmt.Println("\nRetrieving data from the cluster...")
	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("key%d", i)

		// 通过随机节点读取数据 (不一定是存储该数据的同一节点)
		nodeIndex := (i + 2) % len(nodes)
		value, err := nodes[nodeIndex].GetData(ctx, key)
		if err != nil {
			fmt.Printf("Failed to get %s through node %s: %v\n", key, nodes[nodeIndex].ID, err)
		} else {
			fmt.Printf("Got %s = %s through node %s\n", key, value, nodes[nodeIndex].ID)
		}
	}

	// 展示集群成员信息
	fmt.Println("\nCluster membership from each node's perspective:")
	for _, node := range nodes {
		members := node.GetMembers()
		fmt.Printf("Node %s sees %d members: %v\n", node.ID, len(members), memberIDs(members))
	}

	// 演示节点离开集群
	fmt.Println("\nDemonstrating node leaving the cluster...")
	if err := nodes[2].Leave(); err != nil {
		log.Printf("Error when node %s leaves: %v", nodes[2].ID, err)
	} else {
		fmt.Printf("Node %s has left the cluster\n", nodes[2].ID)
	}

	// 等待集群成员信息更新
	fmt.Println("Waiting for membership changes to propagate...")
	time.Sleep(2 * time.Second)

	// 再次检查成员信息
	fmt.Println("\nCluster membership after node left:")
	for i := 0; i < 2; i++ {
		members := nodes[i].GetMembers()
		fmt.Printf("Node %s now sees %d members: %v\n", nodes[i].ID, len(members), memberIDs(members))
	}

	// 设置信号处理，优雅退出
	setupSignalHandler(nodes)

	fmt.Println("\nExample completed.")
	fmt.Println("Press Ctrl+C to exit.")

	// 保持程序运行，直到收到信号
	select {}
}

// ClusterNode 封装集群节点
type ClusterNode struct {
	ID        string
	Address   string
	cache     cache.Cache
	node      *cluster.Node
	running   bool
}

// createClusterNode 创建新的集群节点
func createClusterNode(nodeID, bindAddr string, localCache cache.Cache) (*ClusterNode, error) {
	// 设置集群配置选项
	options := []cluster.NodeOption{
		cluster.WithNodeID(nodeID),
		cluster.WithBindAddr(bindAddr),
		cluster.WithProbeInterval(1 * time.Second),     // 更频繁的健康检查
		cluster.WithGossipInterval(500 * time.Millisecond), // 更频繁的gossip通信
	}

	// 创建集群节点
	node, err := cluster.NewNode(nodeID, localCache, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster node: %w", err)
	}

	return &ClusterNode{
		ID:      nodeID,
		Address: bindAddr,
		cache:   localCache,
		node:    node,
	}, nil
}

// Start 启动节点
func (cn *ClusterNode) Start() error {
	if cn.running {
		return nil
	}

	err := cn.node.Start()
	if err == nil {
		cn.running = true
	}
	return err
}

// Stop 停止节点
func (cn *ClusterNode) Stop() error {
	if !cn.running {
		return nil
	}

	err := cn.node.Stop()
	if err == nil {
		cn.running = false
	}
	return err
}

// Join 加入集群
func (cn *ClusterNode) Join(seedAddr string) error {
	return cn.node.Join(seedAddr)
}

// Leave 离开集群
func (cn *ClusterNode) Leave() error {
	return cn.node.Leave()
}

// GetMembers 获取集群成员列表
func (cn *ClusterNode) GetMembers() []cache.NodeInfo {
	return cn.node.Members()
}

// SetData 设置数据到节点缓存
func (cn *ClusterNode) SetData(ctx context.Context, key string, value string) error {
	return cn.cache.Set(ctx, key, value, 10*time.Minute)
}

// GetData 从节点缓存获取数据
func (cn *ClusterNode) GetData(ctx context.Context, key string) (string, error) {
	value, err := cn.cache.Get(ctx, key)
	if err != nil {
		return "", err
	}

	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("value is not a string")
	}

	return strValue, nil
}

// memberIDs 提取成员ID列表，用于打印
func memberIDs(members []cache.NodeInfo) []string {
	ids := make([]string, len(members))
	for i, member := range members {
		ids[i] = member.ID
	}
	return ids
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
func setupSignalHandler(nodes []*ClusterNode) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\nShutting down cluster nodes...")
		for _, node := range nodes {
			if err := node.Stop(); err != nil {
				log.Printf("Error stopping node %s: %v", node.ID, err)
			} else {
				fmt.Printf("Node %s stopped successfully\n", node.ID)
			}
		}
		os.Exit(0)
	}()
}
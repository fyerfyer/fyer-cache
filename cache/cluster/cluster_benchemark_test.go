package cluster

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/fyerfyer/fyer-cache/mocks"
)

func setupBenchmarkNode(b *testing.B, nodeID string) (*Node, *mocks.Cache, string) {
	b.Helper()

	// Get an available port for this node
	port, err := getAvailablePort()
	if err != nil {
		b.Fatalf("Failed to get port: %v", err)
	}
	addr := "127.0.0.1:" + strconv.Itoa(port)

	mockCache := mocks.NewCache(b)
	node, err := NewNode(nodeID, mockCache, WithNodeID(nodeID), WithBindAddr(addr))
	if err != nil {
		b.Fatalf("Failed to create node: %v", err)
	}
	return node, mockCache, addr
}

func getAvailablePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func BenchmarkNodeCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		mockCache := mocks.NewCache(b)
		_, err := NewNode("test-node", mockCache)
		if err != nil {
			b.Fatalf("Failed to create node: %v", err)
		}
	}
}

func BenchmarkNodeStartStop(b *testing.B) {
	node, _, _ := setupBenchmarkNode(b, "test-node")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := node.Start(); err != nil {
			b.Fatalf("Failed to start node: %v", err)
		}
		if err := node.Stop(); err != nil {
			b.Fatalf("Failed to stop node: %v", err)
		}
	}
}

func BenchmarkNodeJoin(b *testing.B) {
	// Setup two nodes with specific ports
	port1, err := getAvailablePort()
	if err != nil {
		b.Fatalf("Failed to get port: %v", err)
	}
	port2, err := getAvailablePort()
	if err != nil {
		b.Fatalf("Failed to get port: %v", err)
	}

	addr1 := "127.0.0.1:" + strconv.Itoa(port1)
	addr2 := "127.0.0.1:" + strconv.Itoa(port2)

	// Create nodes with explicit addresses
	memCache1 := cache.NewMemoryCache()
	memCache2 := cache.NewMemoryCache()

	node1, err := NewNode("node1", memCache1, WithBindAddr(addr1))
	if err != nil {
		b.Fatalf("Failed to create node1: %v", err)
	}

	node2, err := NewNode("node2", memCache2, WithBindAddr(addr2))
	if err != nil {
		b.Fatalf("Failed to create node2: %v", err)
	}

	// Start both nodes
	err = node1.Start()
	if err != nil {
		b.Fatalf("Failed to start node1: %v", err)
	}
	defer node1.Stop()

	// Give node1 time to start
	time.Sleep(100 * time.Millisecond)

	err = node2.Start()
	if err != nil {
		b.Fatalf("Failed to start node2: %v", err)
	}
	defer node2.Stop()

	// Bootstrap node1
	err = node1.Join(addr1)
	if err != nil {
		b.Fatalf("Failed to bootstrap node1: %v", err)
	}

	// Give node1 time to stabilize
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = node2.Join(addr1)
		if err != nil {
			b.Fatalf("Failed to join cluster: %v", err)
		}

		// Give join operation time to complete
		time.Sleep(50 * time.Millisecond)

		err = node2.Leave()
		if err != nil {
			b.Fatalf("Failed to leave cluster: %v", err)
		}

		// Give leave operation time to complete
		time.Sleep(50 * time.Millisecond)
	}
}

func BenchmarkNodeMembers(b *testing.B) {
	node, _, addr := setupBenchmarkNode(b, "test-node")

	err := node.Start()
	if err != nil {
		b.Fatalf("Failed to start node: %v", err)
	}
	defer node.Stop()

	// Allow node to start up fully
	time.Sleep(100 * time.Millisecond)

	// Join using the actual address we generated
	err = node.Join(addr)
	if err != nil {
		b.Fatalf("Failed to bootstrap node: %v", err)
	}

	// Allow join operation to complete
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = node.Members()
	}
}

func BenchmarkClusterCache_Get(b *testing.B) {
	port, err := getAvailablePort()
	if err != nil {
		b.Fatalf("Failed to get port: %v", err)
	}
	addr := "127.0.0.1:" + strconv.Itoa(port)

	localCache := cache.NewMemoryCache()
	clusterCache, err := NewClusterCache(localCache, "test-node", addr)
	if err != nil {
		b.Fatalf("Failed to create cluster cache: %v", err)
	}

	err = clusterCache.Start()
	if err != nil {
		b.Fatalf("Failed to start cluster cache: %v", err)
	}
	defer clusterCache.Stop()

	// Allow node to start fully
	time.Sleep(100 * time.Millisecond)

	err = clusterCache.Join(addr)
	if err != nil {
		b.Fatalf("Failed to bootstrap cluster: %v", err)
	}

	// Allow join operation to complete
	time.Sleep(100 * time.Millisecond)

	ctx := context.Background()

	// Populate cache with test data
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key-%d", i)
		err := clusterCache.Set(ctx, key, fmt.Sprintf("value-%d", i), 5*time.Minute)
		if err != nil {
			b.Fatalf("Failed to set value: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%100)
		_, _ = clusterCache.Get(ctx, key)
	}
}

func BenchmarkClusterCache_Set(b *testing.B) {
	port, err := getAvailablePort()
	if err != nil {
		b.Fatalf("Failed to get port: %v", err)
	}
	addr := "127.0.0.1:" + strconv.Itoa(port)

	localCache := cache.NewMemoryCache()
	clusterCache, err := NewClusterCache(localCache, "test-node", addr)
	if err != nil {
		b.Fatalf("Failed to create cluster cache: %v", err)
	}

	err = clusterCache.Start()
	if err != nil {
		b.Fatalf("Failed to start cluster cache: %v", err)
	}
	defer clusterCache.Stop()

	// Allow node to start fully
	time.Sleep(100 * time.Millisecond)

	err = clusterCache.Join(addr)
	if err != nil {
		b.Fatalf("Failed to bootstrap cluster: %v", err)
	}

	// Allow join operation to complete
	time.Sleep(100 * time.Millisecond)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%100)
		_ = clusterCache.Set(ctx, key, fmt.Sprintf("value-%d", i), 5*time.Minute)
	}
}

func BenchmarkClusterCache_Del(b *testing.B) {
	port, err := getAvailablePort()
	if err != nil {
		b.Fatalf("Failed to get port: %v", err)
	}
	addr := "127.0.0.1:" + strconv.Itoa(port)

	localCache := cache.NewMemoryCache()
	clusterCache, err := NewClusterCache(localCache, "test-node", addr)
	if err != nil {
		b.Fatalf("Failed to create cluster cache: %v", err)
	}

	err = clusterCache.Start()
	if err != nil {
		b.Fatalf("Failed to start cluster cache: %v", err)
	}
	defer clusterCache.Stop()

	// Allow node to start fully
	time.Sleep(100 * time.Millisecond)

	err = clusterCache.Join(addr)
	if err != nil {
		b.Fatalf("Failed to bootstrap cluster: %v", err)
	}

	// Allow join operation to complete
	time.Sleep(100 * time.Millisecond)

	ctx := context.Background()

	// Populate the cache with test data
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key-%d", i)
		err := clusterCache.Set(ctx, key, fmt.Sprintf("value-%d", i), 5*time.Minute)
		if err != nil {
			b.Fatalf("Failed to set value: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%100)
		_ = clusterCache.Del(ctx, key)
	}
}

func BenchmarkClusterCache_Parallel(b *testing.B) {
	port, err := getAvailablePort()
	if err != nil {
		b.Fatalf("Failed to get port: %v", err)
	}
	addr := "127.0.0.1:" + strconv.Itoa(port)

	localCache := cache.NewMemoryCache()
	clusterCache, err := NewClusterCache(localCache, "test-node", addr)
	if err != nil {
		b.Fatalf("Failed to create cluster cache: %v", err)
	}

	err = clusterCache.Start()
	if err != nil {
		b.Fatalf("Failed to start cluster cache: %v", err)
	}
	defer clusterCache.Stop()

	// Allow node to start fully
	time.Sleep(100 * time.Millisecond)

	err = clusterCache.Join(addr)
	if err != nil {
		b.Fatalf("Failed to bootstrap cluster: %v", err)
	}

	// Allow join operation to complete
	time.Sleep(100 * time.Millisecond)

	ctx := context.Background()

	// Populate the cache with test data
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key-%d", i)
		err := clusterCache.Set(ctx, key, fmt.Sprintf("value-%d", i), 5*time.Minute)
		if err != nil {
			b.Fatalf("Failed to set value: %v", err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		localCtx := context.Background()
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%100)
			_, _ = clusterCache.Get(localCtx, key)
			i++
		}
	})
}

func BenchmarkClusterCache_MixedOperations(b *testing.B) {
	port, err := getAvailablePort()
	if err != nil {
		b.Fatalf("Failed to get port: %v", err)
	}
	addr := "127.0.0.1:" + strconv.Itoa(port)

	localCache := cache.NewMemoryCache()
	clusterCache, err := NewClusterCache(localCache, "test-node", addr)
	if err != nil {
		b.Fatalf("Failed to create cluster cache: %v", err)
	}

	err = clusterCache.Start()
	if err != nil {
		b.Fatalf("Failed to start cluster cache: %v", err)
	}
	defer clusterCache.Stop()

	// Allow node to start fully
	time.Sleep(100 * time.Millisecond)

	err = clusterCache.Join(addr)
	if err != nil {
		b.Fatalf("Failed to bootstrap cluster: %v", err)
	}

	// Allow join operation to complete
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	var wg sync.WaitGroup

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			r := 0
			for j := 0; j < b.N/4; j++ {
				key := fmt.Sprintf("key-%d", r%100)
				switch r % 3 {
				case 0:
					_ = clusterCache.Set(ctx, key, fmt.Sprintf("value-%d", r), 5*time.Minute)
				case 1:
					_, _ = clusterCache.Get(ctx, key)
				case 2:
					_ = clusterCache.Del(ctx, key)
				}
				r++
			}
		}()
	}

	wg.Wait()
}

func BenchmarkClusterNodeScaling(b *testing.B) {
	nodeCounts := []int{2, 5, 10}

	for _, count := range nodeCounts {
		b.Run(fmt.Sprintf("NodeCount_%d", count), func(b *testing.B) {
			ports, err := getMultiplePorts(count)
			if err != nil {
				b.Fatalf("Failed to get ports: %v", err)
			}

			nodes := make([]*Node, count)
			for i := 0; i < count; i++ {
				nodeID := fmt.Sprintf("node-%d", i)
				addr := "127.0.0.1:" + strconv.Itoa(ports[i])
				nodes[i], err = NewNode(nodeID, cache.NewMemoryCache(), WithNodeID(nodeID), WithBindAddr(addr))
				if err != nil {
					b.Fatalf("Failed to create node: %v", err)
				}
			}

			// Start nodes with delay between each start
			for i, node := range nodes {
				err := node.Start()
				if err != nil {
					b.Fatalf("Failed to start node %d: %v", i, err)
				}
				defer node.Stop()

				// Give each node time to start
				time.Sleep(100 * time.Millisecond)
			}

			seedAddr := nodes[0].config.BindAddr
			if seedAddr == "" {
				b.Fatalf("First node has empty bind address")
			}

			err = nodes[0].Join(seedAddr)
			if err != nil {
				b.Fatalf("Failed to bootstrap first node: %v", err)
			}

			time.Sleep(200 * time.Millisecond)

			for i := 1; i < count; i++ {
				err := nodes[i].Join(seedAddr)
				if err != nil {
					b.Fatalf("Failed to join node %d to cluster: %v", i, err)
				}
				time.Sleep(100 * time.Millisecond)
			}

			time.Sleep(300 * time.Millisecond)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = nodes[0].Members()
			}
		})
	}
}

func getMultiplePorts(count int) ([]int, error) {
	ports := make([]int, count)
	listeners := make([]net.Listener, count)

	defer func() {
		for _, l := range listeners {
			if l != nil {
				l.Close()
			}
		}
	}()

	for i := 0; i < count; i++ {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return nil, err
		}

		ports[i] = listener.Addr().(*net.TCPAddr).Port
		listeners[i] = listener
	}

	return ports, nil
}
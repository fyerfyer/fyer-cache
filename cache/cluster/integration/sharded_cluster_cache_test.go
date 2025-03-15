package integration

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/fyerfyer/fyer-cache/cache/cluster"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewShardedClusterCache(t *testing.T) {
	localCache := cache.NewMemoryCache()

	tests := []struct {
		name          string
		localCache    cache.Cache
		nodeID        string
		bindAddr      string
		clusterOpts   []cluster.NodeOption
		cacheOpts     []ShardedClusterCacheOption
		expectError   bool
		errorContains string
	}{
		{
			name:          "Valid configuration",
			localCache:    localCache,
			nodeID:        "node1",
			bindAddr:      "127.0.0.1:8001",
			clusterOpts:   []cluster.NodeOption{},
			cacheOpts:     []ShardedClusterCacheOption{},
			expectError:   false,
			errorContains: "",
		},
		{
			name:          "Nil local cache",
			localCache:    nil,
			nodeID:        "node1",
			bindAddr:      "127.0.0.1:8001",
			clusterOpts:   []cluster.NodeOption{},
			cacheOpts:     []ShardedClusterCacheOption{},
			expectError:   true,
			errorContains: "local cache cannot be nil",
		},
		{
			name:          "Empty node ID",
			localCache:    localCache,
			nodeID:        "",
			bindAddr:      "127.0.0.1:8001",
			clusterOpts:   []cluster.NodeOption{},
			cacheOpts:     []ShardedClusterCacheOption{},
			expectError:   true,
			errorContains: "node ID cannot be empty",
		},
		{
			name:          "Empty bind address",
			localCache:    localCache,
			nodeID:        "node1",
			bindAddr:      "",
			clusterOpts:   []cluster.NodeOption{},
			cacheOpts:     []ShardedClusterCacheOption{},
			expectError:   true,
			errorContains: "bind address cannot be empty",
		},
		{
			name:       "With cache options",
			localCache: localCache,
			nodeID:     "node1",
			bindAddr:   "127.0.0.1:8001",
			clusterOpts: []cluster.NodeOption{
				cluster.WithGossipInterval(500 * time.Millisecond),
			},
			cacheOpts: []ShardedClusterCacheOption{
				WithVirtualNodeCount(200),
				WithReplicaFactor(3),
			},
			expectError:   false,
			errorContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scc, err := NewShardedClusterCache(tt.localCache, tt.nodeID, tt.bindAddr, tt.clusterOpts, tt.cacheOpts)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, scc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, scc)
				assert.Equal(t, tt.nodeID, scc.localNodeID)
				assert.Equal(t, tt.bindAddr, scc.address)
				assert.Equal(t, tt.localCache, scc.localCache)
				assert.NotNil(t, scc.shardedCache)
				assert.NotNil(t, scc.node)
				assert.NotNil(t, scc.nodeDistributor)
				assert.False(t, scc.running)
			}
		})
	}
}

func TestShardedClusterCache_StartStop(t *testing.T) {
	localCache := cache.NewMemoryCache()
	nodeID := "test-node"
	bindAddr := getAvailableAddr(t)

	scc, err := NewShardedClusterCache(localCache, nodeID, bindAddr, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, scc)

	// Test Start
	err = scc.Start()
	assert.NoError(t, err)
	assert.True(t, scc.running)

	// Test double Start (should be no-op)
	err = scc.Start()
	assert.NoError(t, err)
	assert.True(t, scc.running)

	// Test Stop
	err = scc.Stop()
	assert.NoError(t, err)
	assert.False(t, scc.running)

	// Test double Stop (should be no-op)
	err = scc.Stop()
	assert.NoError(t, err)
	assert.False(t, scc.running)
}

func TestShardedClusterCache_Join(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create two cache instances with memory caches
	localCache1 := cache.NewMemoryCache()
	localCache2 := cache.NewMemoryCache()

	addr1 := getAvailableAddr(t)
	addr2 := getAvailableAddr(t)

	scc1, err := NewShardedClusterCache(localCache1, "node1", addr1, nil, nil)
	require.NoError(t, err)

	scc2, err := NewShardedClusterCache(localCache2, "node2", addr2, nil, nil)
	require.NoError(t, err)

	// Start both nodes
	err = scc1.Start()
	require.NoError(t, err)
	defer scc1.Stop()

	err = scc2.Start()
	require.NoError(t, err)
	defer scc2.Stop()

	// Node 1 joins itself (forms new cluster)
	err = scc1.Join(addr1)
	assert.NoError(t, err)

	// Wait for cluster to stabilize
	time.Sleep(500 * time.Millisecond)

	// Node 2 joins node 1
	err = scc2.Join(addr1)
	assert.NoError(t, err)

	// Wait for cluster to stabilize
	time.Sleep(1 * time.Second)

	// Verify both nodes see each other
	members1 := scc1.Members()
	assert.GreaterOrEqual(t, len(members1), 2)

	members2 := scc2.Members()
	assert.GreaterOrEqual(t, len(members2), 2)

	// Verify node IDs
	ids1 := make(map[string]bool)
	for _, m := range members1 {
		ids1[m.ID] = true
	}

	ids2 := make(map[string]bool)
	for _, m := range members2 {
		ids2[m.ID] = true
	}

	assert.True(t, ids1["node1"])
	assert.True(t, ids1["node2"])
	assert.True(t, ids2["node1"])
	assert.True(t, ids2["node2"])
}

func TestShardedClusterCache_Leave(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create two cache instances
	localCache1 := cache.NewMemoryCache()
	localCache2 := cache.NewMemoryCache()

	addr1 := getAvailableAddr(t)
	addr2 := getAvailableAddr(t)

	scc1, err := NewShardedClusterCache(localCache1, "node1", addr1, nil, nil)
	require.NoError(t, err)

	scc2, err := NewShardedClusterCache(localCache2, "node2", addr2, nil, nil)
	require.NoError(t, err)

	// Start both nodes
	err = scc1.Start()
	require.NoError(t, err)
	defer scc1.Stop()

	err = scc2.Start()
	require.NoError(t, err)
	defer scc2.Stop()

	// Form cluster
	err = scc1.Join(addr1)
	require.NoError(t, err)

	err = scc2.Join(addr1)
	require.NoError(t, err)

	// Wait for cluster to stabilize
	time.Sleep(1 * time.Second)

	// Verify both nodes see each other
	assert.GreaterOrEqual(t, len(scc1.Members()), 2)
	assert.GreaterOrEqual(t, len(scc2.Members()), 2)

	// Node 2 leaves the cluster
	err = scc2.Leave()
	assert.NoError(t, err)

	// Wait for leave to propagate
	time.Sleep(1 * time.Second)

	// Node 1 should see the leave
	members1 := scc1.Members()
	node2Status := cache.NodeStatusUp
	for _, m := range members1 {
		if m.ID == "node2" {
			node2Status = m.Status
			break
		}
	}

	// Node 2 should be marked as left or completely removed
	if len(members1) == 2 {
		assert.Equal(t, cache.NodeStatusLeft, node2Status)
	} else {
		assert.Equal(t, 1, len(members1))
	}
}

func TestShardedClusterCache_CacheOperations(t *testing.T) {
	// Create two cache instances with memory caches
	localCache1 := cache.NewMemoryCache()
	localCache2 := cache.NewMemoryCache()

	nodeID1 := "node1"
	nodeID2 := "node2"
	bindAddr1 := getAvailableAddr(t)
	bindAddr2 := getAvailableAddr(t)

	// Start both nodes
	scc1, err := NewShardedClusterCache(localCache1, nodeID1, bindAddr1, nil, nil)
	require.NoError(t, err)
	err = scc1.Start()
	require.NoError(t, err)
	defer scc1.Stop()

	scc2, err := NewShardedClusterCache(localCache2, nodeID2, bindAddr2, nil, nil)
	require.NoError(t, err)
	err = scc2.Start()
	require.NoError(t, err)
	defer scc2.Stop()

	// Node 1 forms new cluster
	err = scc1.Join(bindAddr1)
	require.NoError(t, err)

	// Wait for cluster to stabilize
	time.Sleep(200 * time.Millisecond)

	// Node 2 joins node 1
	err = scc2.Join(bindAddr1)
	require.NoError(t, err)

	// Wait for cluster to stabilize
	time.Sleep(500 * time.Millisecond)

	ctx := context.Background()

	// Set values from node1
	err = scc1.Set(ctx, "key1", "value1", time.Minute)
	require.NoError(t, err)
	err = scc1.Set(ctx, "key2", "value2", time.Minute)
	require.NoError(t, err)

	// Wait for values to propagate (in case there's async replication)
	time.Sleep(200 * time.Millisecond)

	// Get values from both nodes
	val1, err := scc1.Get(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, "value1", val1)

	val2, err := scc2.Get(ctx, "key2")
	require.NoError(t, err)
	assert.Equal(t, "value2", val2)

	// Delete a key from node2
	err = scc2.Del(ctx, "key1")
	require.NoError(t, err)

	// Wait for delete to propagate
	time.Sleep(200 * time.Millisecond)

	// Key should be deleted on both nodes
	_, err = scc1.Get(ctx, "key1")
	assert.Error(t, err, "Key1 should be deleted")

	// The other key should still be accessible
	val2Again, err := scc1.Get(ctx, "key2")
	require.NoError(t, err)
	assert.Equal(t, "value2", val2Again)
}

func TestShardedClusterCache_GetClusterInfo(t *testing.T) {
	localCache := cache.NewMemoryCache()
	nodeID := "test-node"
	bindAddr := getAvailableAddr(t)

	scc, err := NewShardedClusterCache(localCache, nodeID, bindAddr, nil, nil)
	require.NoError(t, err)

	err = scc.Start()
	require.NoError(t, err)
	defer scc.Stop()

	err = scc.Join(bindAddr)
	require.NoError(t, err)

	// Wait for cluster to initialize
	time.Sleep(500 * time.Millisecond)

	// Get cluster info
	info := scc.GetClusterInfo()
	assert.NotNil(t, info)

	// Check essential fields
	assert.Equal(t, nodeID, info["node_id"])
	assert.Equal(t, bindAddr, info["address"])
	assert.True(t, info["is_coordinator"].(bool))
	assert.Equal(t, 1, info["member_count"])
	assert.Equal(t, 1, info["node_count"])

	// Check status counts
	statusCounts, ok := info["status_counts"].(map[string]int)
	assert.True(t, ok)
	assert.Equal(t, 1, statusCounts["up"])
	assert.Equal(t, 0, statusCounts["suspect"])
	assert.Equal(t, 0, statusCounts["down"])
	assert.Equal(t, 0, statusCounts["left"])
}

func TestShardedClusterCache_PartialMock(t *testing.T) {
	// Create a simplified ShardedClusterCache for testing
	localCache := cache.NewMemoryCache()

	// Create a direct instance of ShardedClusterCache
	// with minimal dependencies to avoid the need for mocking structs
	scc := &ShardedClusterCache{
		localCache:  localCache,
		localNodeID: "test-node",
		address:     "test-addr:8080",
		running:     false,
	}

	// Test basic operations without dependencies on ClusterNode or NodeShardedCache
	assert.Equal(t, "test-node", scc.localNodeID)
	assert.Equal(t, "test-addr:8080", scc.address)
	assert.False(t, scc.running)

	// Set the running flag directly (simulating Start)
	scc.running = true
	assert.True(t, scc.running)

	// Set the running flag back to false (simulating Stop)
	scc.running = false
	assert.False(t, scc.running)
}

// Helper function to find available ports
func getTestAvailablePorts(t *testing.T, count int) []int {
	ports := make([]int, 0, count)

	for i := 0; i < count; i++ {
		addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
		require.NoError(t, err)

		l, err := net.ListenTCP("tcp", addr)
		require.NoError(t, err)

		port := l.Addr().(*net.TCPAddr).Port
		l.Close()
		ports = append(ports, port)
	}

	return ports
}

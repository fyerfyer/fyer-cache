package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/fyerfyer/fyer-cache/cache/replication"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

func TestLeaderFollowerCluster_HTTPServerSetup(t *testing.T) {
	// Test proper HTTP server initialization with dynamic port assignment
	leader, err := createTestCluster(t, "leader-http-test", replication.RoleLeader)
	require.NoError(t, err)
	defer leader.Stop()

	// Start the leader
	err = leader.Start()
	require.NoError(t, err)

	// Verify HTTP server is listening on the expected address
	// This can be done by making a simple health check request
	syncerAddr := leader.config.SyncerAddress // Use the syncer address, not cluster address
	resp, err := http.Get(fmt.Sprintf("http://%s/replication/health", syncerAddr))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestLeaderFollowerCluster_FollowerManagement(t *testing.T) {
	// Create leader and follower
	leader, err := createTestCluster(t, "leader-mgmt-test", replication.RoleLeader)
	require.NoError(t, err)
	defer leader.Stop()

	follower, err := createTestCluster(t, "follower-mgmt-test", replication.RoleFollower)
	require.NoError(t, err)
	defer follower.Stop()

	// Start both nodes
	err = leader.Start()
	require.NoError(t, err)

	err = follower.Start()
	require.NoError(t, err)

	// Test adding follower
	err = leader.AddFollower(follower.nodeID, follower.address)
	require.NoError(t, err)

	// Test duplicate follower addition (should return error)
	err = leader.AddFollower(follower.nodeID, follower.address)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")

	// Test removing follower
	err = leader.RemoveFollower(follower.nodeID)
	require.NoError(t, err)

	// Test removing non-existent follower (should return error)
	err = leader.RemoveFollower("non-existent")
	require.Error(t, err)
}

func TestLeaderFollowerCluster_ReplicationEndpoints(t *testing.T) {
	// Test sync endpoint availability and communication
	leader, err := createTestCluster(t, "leader-sync-test", replication.RoleLeader)
	require.NoError(t, err)
	defer leader.Stop()

	follower, err := createTestCluster(t, "follower-sync-test", replication.RoleFollower)
	require.NoError(t, err)
	defer follower.Stop()

	// Start both nodes
	err = leader.Start()
	require.NoError(t, err)

	err = follower.Start()
	require.NoError(t, err)

	// Verify sync endpoint is available on follower
	// Use the syncerAddress instead of cluster address
	syncURL := fmt.Sprintf("http://%s/replication/sync", follower.config.SyncerAddress)
	req := &replication.SyncRequest{
		LeaderID: leader.nodeID,
		FullSync: true,
	}

	reqBody, err := json.Marshal(req)
	require.NoError(t, err)

	resp, err := http.Post(syncURL, "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestLeaderFollowerCluster_SimpleDataReplication(t *testing.T) {
	// Create leader and follower
	leader, err := createTestCluster(t, "leader-data-test", replication.RoleLeader)
	require.NoError(t, err)
	defer leader.Stop()

	follower, err := createTestCluster(t, "follower-data-test", replication.RoleFollower)
	require.NoError(t, err)
	defer follower.Stop()

	// Start both nodes
	err = leader.Start()
	require.NoError(t, err)

	err = follower.Start()
	require.NoError(t, err)

	// Form cluster
	err = leader.Join(leader.address)
	require.NoError(t, err)

	// Let cluster stabilize
	time.Sleep(500 * time.Millisecond)

	// Add follower explicitly (this should happen during Join but we're being explicit)
	err = leader.AddFollower(follower.nodeID, follower.config.SyncerAddress)
	require.NoError(t, err)

	// Let sync setup complete
	time.Sleep(500 * time.Millisecond)

	// Set data on leader
	ctx := context.Background()
	testKey := "test-replication-key"
	testValue := "test-replication-value"

	err = leader.Set(ctx, testKey, testValue, time.Minute)
	require.NoError(t, err)

	// Trigger immediate replication
	err = leader.ReplicateNow(ctx)
	require.NoError(t, err)

	// Give time for replication
	time.Sleep(500 * time.Millisecond)

	// Query data from follower directly via its cache
	// Note: In actual distributed systems you'd query through the cache interface
	val, err := follower.localCache.Get(ctx, testKey)
	require.NoError(t, err)
	assert.Equal(t, testValue, val)
}

func TestReplicationSyncer_PortConfiguration(t *testing.T) {
	cache := cache.NewMemoryCache()
	log := replication.NewMemoryReplicationLog()

	// Create syncer with explicit address
	specificPort := 8088 // Choose an available port
	addr := fmt.Sprintf("127.0.0.1:%d", specificPort)
	syncer := replication.NewMemorySyncer("test-node", cache, log,
		replication.WithAddress(addr))

	err := syncer.Start()
	require.NoError(t, err)
	defer syncer.Stop()

	// Verify the server is listening on the specified port
	resp, err := http.Get(fmt.Sprintf("http://%s/replication/health", addr))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

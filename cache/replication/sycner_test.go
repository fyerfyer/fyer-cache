package replication

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

// TestMemorySyncer_HTTPEndpoints 测试同步器的HTTP端点
func TestMemorySyncer_HTTPEndpoints(t *testing.T) {
	cache := newMockCache()
	log := NewMemoryReplicationLog()

	// 创建同步器
	nodeID := "test-node"
	addr := "127.0.0.1:8080"
	syncer := NewMemorySyncer(nodeID, cache, log, WithNodeID(nodeID), WithAddress(addr))

	// 启动同步器
	err := syncer.Start()
	require.NoError(t, err)
	defer syncer.Stop()

	// 获取实际绑定的地址
	actualAddr := syncer.config.Address
	fmt.Printf("Syncer bound to address: %s\n", actualAddr)

	// 1. 测试健康检查端点
	healthURL := fmt.Sprintf("http://%s/replication/health", actualAddr)
	resp, err := http.Get(healthURL)
	require.NoError(t, err, "Health check request failed")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Health check should return 200 OK")
	resp.Body.Close()

	// 2. 测试同步端点
	syncURL := fmt.Sprintf("http://%s/replication/sync", actualAddr)

	// 创建一个测试同步请求
	req := &SyncRequest{
		LeaderID:     "leader-1",
		LastLogIndex: 0,
		LastLogTerm:  0,
		FullSync:     true,
	}

	reqBody, err := json.Marshal(req)
	require.NoError(t, err)

	// 发送POST请求到同步端点
	httpResp, err := http.Post(syncURL, "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "Sync request failed")
	defer httpResp.Body.Close()

	// 验证响应状态码
	assert.Equal(t, http.StatusOK, httpResp.StatusCode, "Sync endpoint should return 200 OK")

	// 解析响应
	var syncResp SyncResponse
	err = json.NewDecoder(httpResp.Body).Decode(&syncResp)
	require.NoError(t, err, "Failed to decode sync response")

	// 验证响应内容
	assert.True(t, syncResp.Success, "Sync response should indicate success")
}
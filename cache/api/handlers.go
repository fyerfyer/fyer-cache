package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/fyerfyer/fyer-cache/cache"
)

// MemoryStatsProvider 内存缓存统计信息提供者接口
type MemoryStatsProvider interface {
	// MemoryUsage 返回缓存使用的内存字节数
	MemoryUsage() int64

	// ItemCount 返回缓存项数量
	ItemCount() int64

	// HitRate 返回缓存命中率
	HitRate() float64

	// MissRate 返回缓存未命中率
	MissRate() float64
}

// ClusterCacheProvider 集群缓存提供者接口
type ClusterCacheProvider interface {
	// GetNodeCount 获取集群节点数量
	GetNodeCount() int

	// GetClusterInfo 获取集群信息
	GetClusterInfo() map[string]interface{}

	// AddNode 添加节点到集群
	AddNode(nodeID string, address string) error

	// RemoveNode 从集群移除节点
	RemoveNode(nodeID string) error
}

// HandleStats 处理 GET /stats 请求
func HandleStats(w http.ResponseWriter, r *http.Request, cache cache.Cache, clusterCache ClusterCacheProvider) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 构建基本响应
	stats := &StatsResponse{}

	// 尝试获取内存使用情况
	if memCache, ok := cache.(MemoryStatsProvider); ok {
		stats.ItemCount = memCache.ItemCount()
		stats.MemoryUsage = memCache.MemoryUsage()
		stats.HitRate = memCache.HitRate()
		stats.MissRate = memCache.MissRate()
	}

	// 如果有集群信息，添加到响应中
	if clusterCache != nil {
		stats.NodeCount = clusterCache.GetNodeCount()
		stats.ClusterInfo = clusterCache.GetClusterInfo()
	}

	// 返回JSON响应
	writeJSONResponse(w, http.StatusOK, stats)
}

// HandleInvalidate 处理 POST /invalidate?key=X 请求
func HandleInvalidate(w http.ResponseWriter, r *http.Request, cache cache.Cache) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		writeJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Missing key parameter",
		})
		return
	}

	// 删除缓存
	err := cache.Del(context.Background(), key)
	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "Failed to invalidate cache: " + err.Error(),
		})
		return
	}

	// 成功响应
	writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Cache key '%s' invalidated successfully", key),
		Data: InvalidateResponse{
			Key: key,
		},
	})
}

// HandleScale 处理 POST /scale?nodes=N 请求
func HandleScale(w http.ResponseWriter, r *http.Request, clusterCache ClusterCacheProvider) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 检查是否是集群模式
	if clusterCache == nil {
		writeJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Scaling is only supported in cluster mode",
		})
		return
	}

	// 解析nodes参数
	nodesStr := r.URL.Query().Get("nodes")
	targetNodes, err := strconv.Atoi(nodesStr)
	if err != nil || targetNodes < 1 {
		writeJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid nodes parameter: must be a positive integer",
		})
		return
	}

	// 获取当前节点数
	currentNodes := clusterCache.GetNodeCount()

	// 如果目标节点数与当前节点数相同，无需操作
	if targetNodes == currentNodes {
		writeJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "No scaling needed, node count already matches target",
			Data: ScaleInfo{
				PreviousNodeCount: currentNodes,
				CurrentNodeCount:  currentNodes,
			},
		})
		return
	}

	addedNodes := []string{}
	removedNodes := []string{}

	// 执行扩/缩容逻辑
	if targetNodes > currentNodes {
		// 扩容
		for i := currentNodes + 1; i <= targetNodes; i++ {
			nodeID := fmt.Sprintf("node-%d", i)
			address := fmt.Sprintf("127.0.0.1:%d", 8000+i) // 示例地址

			err := clusterCache.AddNode(nodeID, address)
			if err != nil {
				writeJSONResponse(w, http.StatusInternalServerError, APIResponse{
					Success: false,
					Message: "Failed to scale cluster: " + err.Error(),
					Data: ScaleInfo{
						PreviousNodeCount: currentNodes,
						CurrentNodeCount:  clusterCache.GetNodeCount(),
						AddedNodes:        addedNodes,
					},
				})
				return
			}
			addedNodes = append(addedNodes, nodeID)
		}
	} else {
		// 缩容
		for i := currentNodes; i > targetNodes; i-- {
			nodeID := fmt.Sprintf("node-%d", i)

			err := clusterCache.RemoveNode(nodeID)
			if err != nil {
				writeJSONResponse(w, http.StatusInternalServerError, APIResponse{
					Success: false,
					Message: "Failed to scale cluster: " + err.Error(),
					Data: ScaleInfo{
						PreviousNodeCount: currentNodes,
						CurrentNodeCount:  clusterCache.GetNodeCount(),
						RemovedNodes:      removedNodes,
					},
				})
				return
			}
			removedNodes = append(removedNodes, nodeID)
		}
	}

	// 返回成功响应
	writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Cluster scaled from %d to %d nodes", currentNodes, targetNodes),
		Data: ScaleInfo{
			PreviousNodeCount: currentNodes,
			CurrentNodeCount:  targetNodes,
			AddedNodes:        addedNodes,
			RemovedNodes:      removedNodes,
		},
	})
}

// writeJSONResponse 辅助方法，写入JSON响应
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// 如果编码失败，记录错误并返回通用错误
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
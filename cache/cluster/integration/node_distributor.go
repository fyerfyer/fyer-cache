package integration

import (
	"fmt"
	"sync"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/fyerfyer/fyer-cache/cache/distributed"
)

// NodeDistributor 管理集群节点和一致性哈希环之间的映射关系
// 它监听集群事件并更新一致性哈希环，实现集群节点变更和缓存分片的协调
type NodeDistributor struct {
	localNodeID  string                  // 本地节点ID
	shardedCache *cache.NodeShardedCache // 一致性哈希分片缓存
	mu           sync.RWMutex            // 保护并发访问
	nodeMap      map[string]string       // 节点ID到地址的映射表
}

// NewNodeDistributor 创建新的节点分配器
func NewNodeDistributor(localNodeID string, shardedCache *cache.NodeShardedCache) *NodeDistributor {
	return &NodeDistributor{
		localNodeID:  localNodeID,
		shardedCache: shardedCache,
		nodeMap:      make(map[string]string),
	}
}

// HandleClusterEvent 处理集群事件，更新一致性哈希环
// 这是连接集群成员管理与一致性哈希的关键方法
func (nd *NodeDistributor) HandleClusterEvent(event cache.ClusterEvent) {
	nd.mu.Lock()
	defer nd.mu.Unlock()

	switch event.Type {
	case cache.EventNodeJoin:
		// 节点加入事件
		nodeID := event.NodeID
		nodeAddr, ok := event.Details.(string)
		if !ok {
			// 如果详情不是地址，则无法处理
			return
		}

		// 记录节点映射
		nd.nodeMap[nodeID] = nodeAddr

		// 为远程节点创建RemoteCache实例
		if nodeID != nd.localNodeID {
			remoteCache := distributed.NewRemoteCache(nodeID, nodeAddr)
			// 将远程节点添加到一致性哈希环
			nd.shardedCache.AddNode(nodeID, remoteCache, 1)
		}

	case cache.EventNodeLeave, cache.EventNodeFailed:
		// 节点离开或失败事件
		nodeID := event.NodeID
		if nodeID == nd.localNodeID {
			// 忽略本地节点的离开/失败事件
			return
		}

		// 从一致性哈希环中移除节点
		nd.shardedCache.RemoveNode(nodeID)
		// 从映射表中删除
		delete(nd.nodeMap, nodeID)

	case cache.EventNodeRecovered:
		// 节点恢复事件
		nodeID := event.NodeID
		if nodeID == nd.localNodeID {
			// 忽略本地节点的恢复事件
			return
		}

		// 获取节点地址
		nodeAddr, exists := nd.nodeMap[nodeID]
		if !exists {
			// 如果没有地址记录，无法处理
			return
		}

		// 重新添加到一致性哈希环
		remoteCache := distributed.NewRemoteCache(nodeID, nodeAddr)
		nd.shardedCache.AddNode(nodeID, remoteCache, 1)
	}
}

// InitFromMembers 从现有成员初始化哈希环
// 用于节点启动或重启时重建一致性哈希环
func (nd *NodeDistributor) InitFromMembers(members []cache.NodeInfo) error {
	nd.mu.Lock()
	defer nd.mu.Unlock()

	for _, member := range members {
		if member.Status != cache.NodeStatusUp && member.Status != cache.NodeStatusSuspect {
			// 只添加状态为Up或Suspect的节点
			continue
		}

		if member.ID == nd.localNodeID {
			// 本地节点由调用者负责添加
			continue
		}

		// 保存节点映射
		nd.nodeMap[member.ID] = member.Addr

		// 创建远程缓存并添加到哈希环
		remoteCache := distributed.NewRemoteCache(member.ID, member.Addr)
		err := nd.shardedCache.AddNode(member.ID, remoteCache, 1)
		if err != nil {
			return fmt.Errorf("failed to add node %s to hash ring: %w", member.ID, err)
		}
	}

	return nil
}

// GetAllNodeAddresses 获取所有节点的地址
// 用于调试和监控
func (nd *NodeDistributor) GetAllNodeAddresses() map[string]string {
	nd.mu.RLock()
	defer nd.mu.RUnlock()

	// 创建副本以避免并发问题
	result := make(map[string]string)
	for id, addr := range nd.nodeMap {
		result[id] = addr
	}
	return result
}

// Clear 清空节点映射和哈希环
// 用于重置或清理资源
func (nd *NodeDistributor) Clear() {
	nd.mu.Lock()
	defer nd.mu.Unlock()

	// 从哈希环中移除所有远程节点
	for nodeID := range nd.nodeMap {
		if nodeID != nd.localNodeID {
			nd.shardedCache.RemoveNode(nodeID)
		}
	}

	// 清空节点映射表
	nd.nodeMap = make(map[string]string)
}

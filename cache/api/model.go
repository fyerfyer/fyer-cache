package api

// StatsResponse 表示 GET /stats 的响应结构
type StatsResponse struct {
	// 缓存中的项目数量
	ItemCount int64 `json:"item_count"`

	// 内存使用情况（字节）
	MemoryUsage int64 `json:"memory_usage,omitempty"`

	// 缓存命中率
	HitRate float64 `json:"hit_rate,omitempty"`

	// 缓存未命中率
	MissRate float64 `json:"miss_rate,omitempty"`

	// 节点数量
	NodeCount int `json:"node_count,omitempty"`

	// 分片数量
	ShardCount int `json:"shard_count,omitempty"`

	// 集群信息（如果是集群模式）
	ClusterInfo map[string]interface{} `json:"cluster_info,omitempty"`
}

// APIResponse 通用 API 响应格式
type APIResponse struct {
	// 操作是否成功
	Success bool `json:"success"`

	// 响应消息
	Message string `json:"message,omitempty"`

	// 响应数据
	Data interface{} `json:"data,omitempty"`
}

// InvalidateRequest 缓存失效请求
type InvalidateRequest struct {
	// 要失效的缓存键
	Key string `json:"key"`
}

// InvalidateResponse 缓存失效响应
type InvalidateResponse struct {
	// 被删除的键
	Key string `json:"key"`
}

// ScaleRequest 扩缩容请求
type ScaleRequest struct {
	// 目标节点数量
	Nodes int `json:"nodes"`
}

// ScaleInfo 集群扩缩容信息
type ScaleInfo struct {
	// 扩缩容前节点数量
	PreviousNodeCount int `json:"previous_node_count"`

	// 扩缩容后节点数量
	CurrentNodeCount int `json:"current_node_count"`

	// 新增的节点ID
	AddedNodes []string `json:"added_nodes,omitempty"`

	// 移除的节点ID
	RemovedNodes []string `json:"removed_nodes,omitempty"`
}
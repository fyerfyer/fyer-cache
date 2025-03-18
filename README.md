# FyerCache

FyerCache 是一个高性能、分布式、可扩展的 Go 语言缓存系统，提供从单机内存缓存到复杂的分布式缓存集群的完整解决方案。

**文档：** [https://fyerfyer.github.io/fyer-cache](https://fyerfyer.github.io/fyer-cache)

## 功能特性

- **灵活的缓存类型**
    - 内存缓存，支持TTL过期和自动清理
    - 分片缓存，减少锁竞争，提升并发性能
    - 分布式缓存，跨多个节点扩展

- **强大的分布式功能**
    - 基于一致性哈希的数据分片
    - 节点自动发现和集群管理
    - 支持主从架构和分片集群架构
    - 数据自动迁移和负载均衡

- **多种一致性策略**
    - Cache-Aside 模式
    - Write-Through 模式
    - 基于消息队列的缓存同步

- **高级功能**
    - 优雅的故障处理和自修复能力
    - 丰富的监控指标与Prometheus集成
    - 可扩展的API服务器
    - 完整的HTTP管理接口

## 快速开始

### 安装

```bash
go get -u github.com/fyerfyer/fyer-cache
```

### 基本用法

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/fyerfyer/fyer-cache/cache"
)

func main() {
    // 创建一个内存缓存
    memCache := cache.NewMemoryCache()
    
    ctx := context.Background()
    
    // 设置缓存项，1分钟过期
    err := memCache.Set(ctx, "user:1001", "John Doe", 1*time.Minute)
    if err != nil {
        fmt.Printf("Failed to set cache: %v\n", err)
    }
    
    // 获取缓存项
    val, err := memCache.Get(ctx, "user:1001")
    if err != nil {
        fmt.Printf("Failed to get from cache: %v\n", err)
    } else {
        fmt.Printf("User: %v\n", val)
    }
    
    // 删除缓存项
    err = memCache.Del(ctx, "user:1001")
    if err != nil {
        fmt.Printf("Failed to delete from cache: %v\n", err)
    }
}
```

### 分片缓存

```go
// 创建一个分片缓存，提高并发性能
shardedCache := cache.NewMemoryCache(
    cache.WithShardCount(32), // 32个分片
    cache.WithCleanupInterval(5*time.Minute),
    cache.WithAsyncCleanup(true),
)
```

### 分布式缓存

```go
// 创建分布式缓存节点
shardedCluster, err := integration.NewShardedClusterCache(
    localCache,         // 本地缓存实例
    "node1",            // 节点ID
    "localhost:7000",   // 节点地址
    []cluster.NodeOption{
        cluster.WithGossipInterval(200 * time.Millisecond),
    },
    []integration.ShardedClusterCacheOption{
        integration.WithVirtualNodeCount(100),
    },
)

// 启动节点
err = shardedCluster.Start()

// 加入集群
err = shardedCluster.Join("localhost:7001") // 加入已有集群
// 或形成新集群
err = shardedCluster.Join(shardedCluster.GetAddress())
```

## 架构概述

FyerCache 采用模块化设计，主要组件包括：

- **核心缓存引擎** - 提供基础缓存功能和内存管理
- **分片机制** - 使用一致性哈希实现数据分片
- **集群管理** - 处理节点发现、成员管理和故障检测
- **数据同步** - 确保多节点间数据一致性
- **监控系统** - 收集性能指标和运行状态

## 高级用例

### 主从架构缓存集群

```go
// 创建主节点
leaderCluster, err := integration.NewLeaderFollowerCluster(
    localCache,
    "leader-1",
    "localhost:8000",
    []cluster.NodeOption{...},
    []integration.LeaderFollowerClusterOption{
        integration.WithInitialRole(replication.RoleLeader),
    },
)

// 启动并初始化集群
leaderCluster.Start()
leaderCluster.Join(leaderCluster.GetAddress())

// 创建并添加从节点
followerCluster, err := integration.NewLeaderFollowerCluster(...)
followerCluster.Start()
followerCluster.Join(leaderCluster.GetAddress())
```

### 监控与指标

```go
// 创建带监控的缓存
monitoredCache := metrics.NewMonitoredCache(
    memCache,
    metrics.WithMetricsServer(":8080"),
    metrics.WithNamespace("myapp"),
    metrics.WithCollectInterval(10*time.Second),
)

// 启动监控服务
monitoredCache.Start()
```

### API 服务器

```go
// 创建API服务器
apiServer := api.NewAPIServer(
    cache,
    api.WithBindAddress(":8081"),
    api.WithBasePath("/api/cache"),
)

// 启动服务器
apiServer.Start()
```

## 许可证

FyerCache 在 MIT 许可证下发布。详情请参阅 LICENSE 文件。

---

详细文档请访问：[https://fyerfyer.github.io/fyer-cache](https://fyerfyer.github.io/fyer-cache)
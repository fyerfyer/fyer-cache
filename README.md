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

## 性能测试

### 基础缓存性能

| 操作类型 | 性能数据 | 说明 |
|---------|---------|------|
| Get (命中) | ~348 ns/op | 基础缓存读取能力 |
| Get (未命中) | ~1074 ns/op | 键不存在时的处理性能 |
| Set | ~2275 ns/op | 基础写入操作性能 |
| Del | ~1054 ns/op | 删除操作性能 |
| 并行 Get | ~266 ns/op | 提升约24% |
| 并行 Set | ~1851 ns/op | 提升约19% |
| 并行混合操作 | ~695 ns/op | 实际场景模拟 |

### 分片策略性能对比

| 分片数量 | 性能数据 | 相对提升 |
|---------|---------|---------|
| 4 分片 | ~1025 ns/op | 基准参考 |
| 16 分片 | ~978 ns/op | 提升约5% |
| 32 分片 | ~1098 ns/op | 轻微下降 |
| 64 分片 | ~929 ns/op | 提升约9% |
| 128 分片 | ~767 ns/op | 提升约25% |

分片数量为128时性能最佳，表明该程度的分片在减少锁竞争方面取得了良好的平衡。

### 清理策略性能

| 策略类型 | 性能数据 | 说明 |
|---------|---------|------|
| 同步清理 | ~5401 ns/op | 传统清理方式 |
| 异步清理 | ~2047 ns/op | 提升约62% |

### 一致性哈希性能

| 操作类型 | 性能数据 | 说明 |
|---------|---------|------|
| 键查找 | ~473 ns/op | 单个查找速度 |
| 节点添加 | ~20.1 ms/op | 重新平衡哈希环 |
| 节点移除 | ~37.5 ms/op | 重新分配所有键 |
| 大集群查找(1000节点) | ~588 ns/op | 扩展性良好 |
| FNV32哈希 | ~353 ns/op | 较好的哈希函数选择 |
| CRC32哈希 | ~447 ns/op | 标准哈希函数 |

FNV32哈希函数比标准CRC32快约21%，适合对性能要求较高的场景。

### 节点分片缓存性能

| 配置项 | 性能数据 | 说明 |
|-------|---------|------|
| 基本Get操作 | ~781 ns/op | 分片路由开销 |
| 基本Set操作 | ~2294 ns/op | 包含分片路由 |
| 并行操作 | ~916 ns/op | 高并发环境 |
| 副本因子=1 | ~804 ns/op | 无数据冗余 |
| 副本因子=2 | ~1301 ns/op | 性能下降约38% |
| 副本因子=3 | ~1473 ns/op | 性能下降约45% |

提高副本因子增加了数据可靠性，但会导致写入性能下降。

### 一致性策略性能对比

| 策略类型 | Get性能 | Set性能 | 说明 |
|---------|--------|--------|------|
| Cache-Aside | ~37 ns/op | ~1398 ns/op | 最佳读取性能 |
| Write-Through | ~121 ns/op | ~3119 ns/op | 最强一致性保证 |
| MQ通知 | ~48 ns/op | ~2121 ns/op | 平衡的读写性能 |

不同一致性策略适用于不同场景，Cache-Aside适合读多写少场景，而Write-Through则适合对一致性要求高的应用。

### 分布式缓存网络性能

| 网络延迟 | 操作性能 | 影响因素 |
|---------|---------|---------|
| 本地操作 | ~427-696 ns/op | 无网络延迟 |
| 1ms网络延迟 | ~1.78 ms/op | 主要受网络影响 |
| 5ms网络延迟 | ~5.58 ms/op | 典型数据中心间延迟 |
| 10ms网络延迟 | ~10.66 ms/op | 跨区域延迟 |

### 监控性能开销

| 操作类型 | 无监控 | 有监控 | 监控开销 |
|---------|-------|-------|---------|
| Get | ~31 ns/op | ~271 ns/op | 增加约240ns |
| Set | ~38 ns/op | ~278 ns/op | 增加约240ns |
| Del | ~25 ns/op | ~271 ns/op | 增加约246ns |

> 注：以上性能数据基于Intel Core i5-4310U CPU @ 2.00GHz处理器测试得出，实际性能可能因硬件配置和运行环境而异。

## 许可证

FyerCache 在 MIT 许可证下发布。详情请参阅 LICENSE 文件。

---

详细文档请访问：[fyer-cache文档](https://fyerfyer.github.io/fyer-cache)
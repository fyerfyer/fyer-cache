"use strict";(self.webpackChunkdocs=self.webpackChunkdocs||[]).push([[361],{1937:(r,n,e)=>{e.r(n),e.d(n,{assets:()=>d,contentTitle:()=>o,default:()=>l,frontMatter:()=>a,metadata:()=>t,toc:()=>h});const t=JSON.parse('{"id":"getting-started/quickstart","title":"Quick Star","description":"\u5b89\u88c5","source":"@site/docs/getting-started/quickstart.md","sourceDirName":"getting-started","slug":"/getting-started/quickstart","permalink":"/fyer-cache/docs/getting-started/quickstart","draft":false,"unlisted":false,"editUrl":"https://github.com/fyerfyer/fyer-rpc/tree/main/docs/getting-started/quickstart.md","tags":[],"version":"current","frontMatter":{},"sidebar":"tutorialSidebar","previous":{"title":"Remote Cache Client","permalink":"/fyer-cache/docs/distribute-caching/remote-cache"},"next":{"title":"Using Pattern","permalink":"/fyer-cache/docs/getting-started/usingpattern"}}');var i=e(4848),c=e(8453);const a={},o="Quick Star",d={},h=[{value:"\u5b89\u88c5",id:"\u5b89\u88c5",level:2},{value:"\u793a\u4f8b 1\uff1a\u7b80\u5355\u5185\u5b58\u7f13\u5b58",id:"\u793a\u4f8b-1\u7b80\u5355\u5185\u5b58\u7f13\u5b58",level:2},{value:"\u793a\u4f8b 2\uff1a\u914d\u7f6e\u5185\u5b58\u7f13\u5b58",id:"\u793a\u4f8b-2\u914d\u7f6e\u5185\u5b58\u7f13\u5b58",level:2},{value:"\u793a\u4f8b 3\uff1a\u5206\u7247\u7f13\u5b58",id:"\u793a\u4f8b-3\u5206\u7247\u7f13\u5b58",level:2},{value:"\u793a\u4f8b 4\uff1a\u57fa\u4e8e\u8282\u70b9\u7684\u5206\u7247\u7f13\u5b58",id:"\u793a\u4f8b-4\u57fa\u4e8e\u8282\u70b9\u7684\u5206\u7247\u7f13\u5b58",level:2},{value:"\u793a\u4f8b 5\uff1a\u5206\u5e03\u5f0f\u96c6\u7fa4\u7f13\u5b58",id:"\u793a\u4f8b-5\u5206\u5e03\u5f0f\u96c6\u7fa4\u7f13\u5b58",level:2},{value:"\u793a\u4f8b 6\uff1a\u6dfb\u52a0\u76d1\u63a7",id:"\u793a\u4f8b-6\u6dfb\u52a0\u76d1\u63a7",level:2},{value:"\u793a\u4f8b 7\uff1aAPI \u670d\u52a1\u5668",id:"\u793a\u4f8b-7api-\u670d\u52a1\u5668",level:2}];function s(r){const n={code:"code",h1:"h1",h2:"h2",header:"header",p:"p",pre:"pre",...(0,c.R)(),...r.components};return(0,i.jsxs)(i.Fragment,{children:[(0,i.jsx)(n.header,{children:(0,i.jsx)(n.h1,{id:"quick-star",children:"Quick Star"})}),"\n",(0,i.jsx)(n.h2,{id:"\u5b89\u88c5",children:"\u5b89\u88c5"}),"\n",(0,i.jsx)(n.p,{children:"\u4f7f\u7528 Go \u7684\u5305\u7ba1\u7406\u5de5\u5177\u5b89\u88c5 FyerCache\uff1a"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-bash",children:"go get -u github.com/fyerfyer/fyer-cache\n"})}),"\n",(0,i.jsx)(n.p,{children:"\u786e\u4fdd\u60a8\u7684\u9879\u76ee\u4f7f\u7528 Go Modules \u8fdb\u884c\u4f9d\u8d56\u7ba1\u7406\uff0c\u5e76\u4e14 Go \u7248\u672c >= 1.19\u3002"}),"\n",(0,i.jsx)(n.h2,{id:"\u793a\u4f8b-1\u7b80\u5355\u5185\u5b58\u7f13\u5b58",children:"\u793a\u4f8b 1\uff1a\u7b80\u5355\u5185\u5b58\u7f13\u5b58"}),"\n",(0,i.jsx)(n.p,{children:"\u6700\u7b80\u5355\u7684\u7528\u6cd5\u662f\u4f7f\u7528\u5185\u5b58\u7f13\u5b58\uff0c\u9002\u5408\u5355\u673a\u5e94\u7528\u6216\u4f5c\u4e3a\u672c\u5730\u7f13\u5b58\u5c42\u3002"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:'package main\r\n\r\nimport (\r\n    "context"\r\n    "fmt"\r\n    "time"\r\n\r\n    "github.com/fyerfyer/fyer-cache/cache"\r\n)\r\n\r\nfunc main() {\r\n    // \u521b\u5efa\u4e00\u4e2a\u7b80\u5355\u7684\u5185\u5b58\u7f13\u5b58\r\n    memCache := cache.NewMemoryCache()\r\n    ctx := context.Background()\r\n\r\n    // \u8bbe\u7f6e\u7f13\u5b58\uff0c5\u5206\u949f\u8fc7\u671f\r\n    err := memCache.Set(ctx, "user:1001", "John Doe", 5*time.Minute)\r\n    if err != nil {\r\n        fmt.Printf("Failed to set cache: %v\\n", err)\r\n        return\r\n    }\r\n    \r\n    // \u83b7\u53d6\u7f13\u5b58\u503c\r\n    val, err := memCache.Get(ctx, "user:1001")\r\n    if err != nil {\r\n        fmt.Printf("Failed to get from cache: %v\\n", err)\r\n        return\r\n    }\r\n    fmt.Printf("User: %v\\n", val)\r\n    \r\n    // \u5220\u9664\u7f13\u5b58\r\n    err = memCache.Del(ctx, "user:1001")\r\n    if err != nil {\r\n        fmt.Printf("Failed to delete from cache: %v\\n", err)\r\n        return\r\n    }\r\n    fmt.Println("Cache entry deleted")\r\n    \r\n    // \u5c1d\u8bd5\u83b7\u53d6\u5df2\u5220\u9664\u7684\u7f13\u5b58\r\n    _, err = memCache.Get(ctx, "user:1001")\r\n    if err != nil {\r\n        fmt.Printf("Expected error: %v\\n", err)\r\n    }\r\n}\n'})}),"\n",(0,i.jsx)(n.h2,{id:"\u793a\u4f8b-2\u914d\u7f6e\u5185\u5b58\u7f13\u5b58",children:"\u793a\u4f8b 2\uff1a\u914d\u7f6e\u5185\u5b58\u7f13\u5b58"}),"\n",(0,i.jsx)(n.p,{children:"\u5185\u5b58\u7f13\u5b58\u63d0\u4f9b\u591a\u79cd\u914d\u7f6e\u9009\u9879\uff0c\u8ba9\u60a8\u53ef\u4ee5\u6839\u636e\u9700\u6c42\u81ea\u5b9a\u4e49\u7f13\u5b58\u884c\u4e3a\uff1a"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:'package main\r\n\r\nimport (\r\n    "context"\r\n    "fmt"\r\n    "runtime"\r\n    "time"\r\n\r\n    "github.com/fyerfyer/fyer-cache/cache"\r\n)\r\n\r\nfunc main() {\r\n    // \u4f7f\u7528\u591a\u4e2a\u9009\u9879\u914d\u7f6e\u5185\u5b58\u7f13\u5b58\r\n    memCache := cache.NewMemoryCache(\r\n        // \u8bbe\u7f6e\u5206\u7247\u6570\u91cf\u4ee5\u51cf\u5c11\u9501\u7ade\u4e89\uff08\u5efa\u8bae\u8bbe\u7f6e\u4e3aCPU\u6838\u5fc3\u6570\u7684\u500d\u6570\uff09\r\n        cache.WithShardCount(32),\r\n        \r\n        // \u8bbe\u7f6e\u6dd8\u6c70\u56de\u8c03\uff0c\u5728\u6761\u76ee\u88ab\u6dd8\u6c70\u65f6\u6267\u884c\r\n        cache.WithEvictionCallback(func(key string, value any) {\r\n            fmt.Printf("Evicted: %s\\n", key)\r\n        }),\r\n        \r\n        // \u8bbe\u7f6e\u6e05\u7406\u95f4\u9694\uff0c\u5b9a\u671f\u6e05\u7406\u8fc7\u671f\u9879\r\n        cache.WithCleanupInterval(5*time.Minute),\r\n        \r\n        // \u542f\u7528\u5f02\u6b65\u6e05\u7406\uff0c\u51cf\u5c11\u6e05\u7406\u5bf9\u4e3b\u7ebf\u7a0b\u6027\u80fd\u7684\u5f71\u54cd\r\n        cache.WithAsyncCleanup(true),\r\n        \r\n        // \u8bbe\u7f6e\u5f02\u6b65\u6e05\u7406\u7684\u5de5\u4f5c\u534f\u7a0b\u6570\r\n        cache.CacheWithWorkerCount(runtime.NumCPU()),\r\n    )\r\n    \r\n    ctx := context.Background()\r\n    \r\n    // \u6dfb\u52a0\u4e00\u4e2a\u77ed\u671f\u7f13\u5b58\u9879\r\n    memCache.Set(ctx, "temp", "temporary value", 2*time.Second)\r\n    \r\n    // \u7b49\u5f85\u9879\u76ee\u8fc7\u671f\u5e76\u88ab\u6e05\u7406\r\n    time.Sleep(3 * time.Second)\r\n    \r\n    // \u68c0\u67e5\u9879\u76ee\u662f\u5426\u5df2\u88ab\u6e05\u7406\r\n    _, err := memCache.Get(ctx, "temp")\r\n    if err != nil {\r\n        fmt.Println("Item expired and cleaned up as expected")\r\n    }\r\n    \r\n    // \u8bf7\u52a1\u5fc5\u5728\u5e94\u7528\u7ed3\u675f\u524d\u5173\u95ed\u7f13\u5b58\u4ee5\u91ca\u653e\u8d44\u6e90\r\n    memCache.Close()\r\n}\n'})}),"\n",(0,i.jsx)(n.h2,{id:"\u793a\u4f8b-3\u5206\u7247\u7f13\u5b58",children:"\u793a\u4f8b 3\uff1a\u5206\u7247\u7f13\u5b58"}),"\n",(0,i.jsx)(n.p,{children:"\u5206\u7247\u7f13\u5b58\u53ef\u4ee5\u663e\u8457\u63d0\u9ad8\u5e76\u53d1\u6027\u80fd\uff0c\u7279\u522b\u662f\u5728\u591a\u6838\u7cfb\u7edf\u4e0a\uff1a"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:'package main\r\n\r\nimport (\r\n    "context"\r\n    "fmt"\r\n    "sync"\r\n    "time"\r\n\r\n    "github.com/fyerfyer/fyer-cache/cache"\r\n)\r\n\r\nfunc main() {\r\n    // \u521b\u5efa32\u4e2a\u5206\u7247\u7684\u5185\u5b58\u7f13\u5b58\r\n    shardedCache := cache.NewMemoryCache(cache.WithShardCount(32))\r\n    ctx := context.Background()\r\n\r\n    // \u6279\u91cf\u5199\u5165\u793a\u4f8b\r\n    var wg sync.WaitGroup\r\n    for i := 0; i < 1000; i++ {\r\n        wg.Add(1)\r\n        go func(id int) {\r\n            defer wg.Done()\r\n            key := fmt.Sprintf("key:%d", id)\r\n            value := fmt.Sprintf("value:%d", id)\r\n            err := shardedCache.Set(ctx, key, value, 5*time.Minute)\r\n            if err != nil {\r\n                fmt.Printf("Error setting %s: %v\\n", key, err)\r\n            }\r\n        }(i)\r\n    }\r\n    wg.Wait()\r\n    \r\n    fmt.Println("Batch write completed")\r\n    \r\n    // \u6279\u91cf\u8bfb\u53d6\u793a\u4f8b\r\n    hitCount := 0\r\n    var mu sync.Mutex\r\n    \r\n    for i := 0; i < 1000; i++ {\r\n        wg.Add(1)\r\n        go func(id int) {\r\n            defer wg.Done()\r\n            key := fmt.Sprintf("key:%d", id)\r\n            if _, err := shardedCache.Get(ctx, key); err == nil {\r\n                mu.Lock()\r\n                hitCount++\r\n                mu.Unlock()\r\n            }\r\n        }(i)\r\n    }\r\n    wg.Wait()\r\n    \r\n    fmt.Printf("Cache hit rate: %d%%\\n", hitCount/10)\r\n}\n'})}),"\n",(0,i.jsx)(n.h2,{id:"\u793a\u4f8b-4\u57fa\u4e8e\u8282\u70b9\u7684\u5206\u7247\u7f13\u5b58",children:"\u793a\u4f8b 4\uff1a\u57fa\u4e8e\u8282\u70b9\u7684\u5206\u7247\u7f13\u5b58"}),"\n",(0,i.jsx)(n.p,{children:"\u8282\u70b9\u5206\u7247\u7f13\u5b58\u5141\u8bb8\u60a8\u5728\u591a\u4e2a\u540e\u7aef\u5b58\u50a8\u8282\u70b9\u4e4b\u95f4\u5206\u53d1\u6570\u636e\uff0c\u8fd9\u662f\u6784\u5efa\u5206\u5e03\u5f0f\u7f13\u5b58\u7684\u57fa\u7840\uff1a"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:'package main\r\n\r\nimport (\r\n    "context"\r\n    "fmt"\r\n\r\n    "github.com/fyerfyer/fyer-cache/cache"\r\n)\r\n\r\nfunc main() {\r\n    // \u521b\u5efa\u4e00\u4e2a\u57fa\u4e8e\u8282\u70b9\u7684\u5206\u7247\u7f13\u5b58\r\n    // \u8fd9\u91cc\u6211\u4eec\u5b9a\u4e49\u4e09\u4e2a\u540e\u7aef\u5b58\u50a8\u8282\u70b9\r\n    shardedCache := cache.NewNodeShardedCache()\r\n    \r\n    // \u4e3a\u6bcf\u4e2a\u8282\u70b9\u521b\u5efa\u72ec\u7acb\u7684\u5185\u5b58\u7f13\u5b58\r\n    node1 := cache.NewMemoryCache()\r\n    node2 := cache.NewMemoryCache()\r\n    node3 := cache.NewMemoryCache()\r\n    \r\n    // \u6dfb\u52a0\u8282\u70b9\u5230\u5206\u7247\u7f13\u5b58\uff0c\u53c2\u6570\u4e3a\uff1a\u8282\u70b9ID, \u7f13\u5b58\u5b9e\u4f8b, \u6743\u91cd\r\n    err := shardedCache.AddNode("node1", node1, 1)\r\n    if err != nil {\r\n        panic(err)\r\n    }\r\n    \r\n    err = shardedCache.AddNode("node2", node2, 1)\r\n    if err != nil {\r\n        panic(err)\r\n    }\r\n    \r\n    err = shardedCache.AddNode("node3", node3, 2) // node3 \u7684\u6743\u91cd\u4e3a2\uff0c\u5c06\u63a5\u6536\u66f4\u591a\u6570\u636e\r\n    if err != nil {\r\n        panic(err)\r\n    }\r\n    \r\n    fmt.Printf("Created node sharded cache with %d nodes\\n", shardedCache.GetNodeCount())\r\n    \r\n    ctx := context.Background()\r\n    \r\n    // \u6570\u636e\u4f1a\u6839\u636e\u952e\u81ea\u52a8\u8def\u7531\u5230\u6b63\u786e\u7684\u8282\u70b9\r\n    for i := 1; i <= 10; i++ {\r\n        key := fmt.Sprintf("key%d", i)\r\n        value := fmt.Sprintf("value%d", i)\r\n        \r\n        err := shardedCache.Set(ctx, key, value, 0) // 0 \u8868\u793a\u4e0d\u8fc7\u671f\r\n        if err != nil {\r\n            fmt.Printf("Failed to set %s: %v\\n", key, err)\r\n        }\r\n    }\r\n    \r\n    // \u83b7\u53d6\u6570\u636e\u65f6\uff0c\u5206\u7247\u7f13\u5b58\u4f1a\u81ea\u52a8\u4ece\u6b63\u786e\u7684\u8282\u70b9\u83b7\u53d6\r\n    for i := 1; i <= 10; i++ {\r\n        key := fmt.Sprintf("key%d", i)\r\n        val, err := shardedCache.Get(ctx, key)\r\n        if err != nil {\r\n            fmt.Printf("Failed to get %s: %v\\n", key, err)\r\n        } else {\r\n            fmt.Printf("%s => %s\\n", key, val)\r\n        }\r\n    }\r\n}\n'})}),"\n",(0,i.jsx)(n.h2,{id:"\u793a\u4f8b-5\u5206\u5e03\u5f0f\u96c6\u7fa4\u7f13\u5b58",children:"\u793a\u4f8b 5\uff1a\u5206\u5e03\u5f0f\u96c6\u7fa4\u7f13\u5b58"}),"\n",(0,i.jsx)(n.p,{children:"\u8fd9\u4e2a\u793a\u4f8b\u5c55\u793a\u5982\u4f55\u521b\u5efa\u4e00\u4e2a\u7b80\u5355\u7684\u5206\u7247\u96c6\u7fa4\u7f13\u5b58\uff0c\u5728\u591a\u4e2a\u8282\u70b9\u4e4b\u95f4\u5206\u53d1\u6570\u636e\uff1a"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:'package main\r\n\r\nimport (\r\n    "context"\r\n    "fmt"\r\n    "time"\r\n\r\n    "github.com/fyerfyer/fyer-cache/cache"\r\n    "github.com/fyerfyer/fyer-cache/cache/cluster"\r\n    "github.com/fyerfyer/fyer-cache/cache/cluster/integration"\r\n)\r\n\r\nfunc main() {\r\n    // \u521b\u5efa\u4e24\u4e2a\u8282\u70b9\u7684\u5206\u7247\u96c6\u7fa4\r\n    \r\n    // \u4e3a\u6bcf\u4e2a\u8282\u70b9\u521b\u5efa\u672c\u5730\u7f13\u5b58\r\n    localCache1 := cache.NewMemoryCache()\r\n    localCache2 := cache.NewMemoryCache()\r\n    \r\n    // \u521b\u5efa\u7b2c\u4e00\u4e2a\u8282\u70b9\r\n    node1, err := integration.NewShardedClusterCache(\r\n        localCache1,\r\n        "node1",\r\n        "localhost:7001",\r\n        []cluster.NodeOption{\r\n            cluster.WithGossipInterval(200 * time.Millisecond),\r\n        },\r\n        []integration.ShardedClusterCacheOption{\r\n            integration.WithVirtualNodeCount(100),\r\n        },\r\n    )\r\n    if err != nil {\r\n        panic(err)\r\n    }\r\n    \r\n    // \u521b\u5efa\u7b2c\u4e8c\u4e2a\u8282\u70b9\r\n    node2, err := integration.NewShardedClusterCache(\r\n        localCache2,\r\n        "node2",\r\n        "localhost:7002",\r\n        []cluster.NodeOption{\r\n            cluster.WithGossipInterval(200 * time.Millisecond),\r\n        },\r\n        []integration.ShardedClusterCacheOption{\r\n            integration.WithVirtualNodeCount(100),\r\n        },\r\n    )\r\n    if err != nil {\r\n        panic(err)\r\n    }\r\n    \r\n    // \u542f\u52a8\u8282\u70b9\r\n    if err := node1.Start(); err != nil {\r\n        panic(err)\r\n    }\r\n    if err := node2.Start(); err != nil {\r\n        panic(err)\r\n    }\r\n    \r\n    // \u786e\u4fdd\u4f18\u96c5\u9000\u51fa\r\n    defer func() {\r\n        node1.Stop()\r\n        node2.Stop()\r\n    }()\r\n    \r\n    // \u8282\u70b91\u5f62\u6210\u81ea\u5df1\u7684\u96c6\u7fa4\r\n    if err := node1.Join(node1.GetAddress()); err != nil {\r\n        panic(err)\r\n    }\r\n    fmt.Printf("Node %s formed its own cluster\\n", node1.GetNodeID())\r\n    \r\n    // \u8282\u70b92\u52a0\u5165\u8282\u70b91\u7684\u96c6\u7fa4\r\n    if err := node2.Join(node1.GetAddress()); err != nil {\r\n        panic(err)\r\n    }\r\n    fmt.Printf("Node %s joined the cluster\\n", node2.GetNodeID())\r\n    \r\n    // \u7b49\u5f85\u96c6\u7fa4\u7a33\u5b9a\r\n    fmt.Println("Waiting for cluster to stabilize...")\r\n    time.Sleep(2 * time.Second)\r\n    \r\n    // \u4f7f\u7528\u96c6\u7fa4\u5b58\u50a8\u548c\u68c0\u7d22\u6570\u636e\r\n    ctx := context.Background()\r\n    \r\n    // \u5411\u96c6\u7fa4\u5199\u5165\u6570\u636e\r\n    fmt.Println("Storing data in the cluster...")\r\n    err = node1.Set(ctx, "cluster-key", "cluster-value", 10*time.Minute)\r\n    if err != nil {\r\n        fmt.Printf("Failed to set data: %v\\n", err)\r\n    } else {\r\n        fmt.Println("Data stored successfully")\r\n    }\r\n    \r\n    // \u4ece\u53e6\u4e00\u4e2a\u8282\u70b9\u8bfb\u53d6\u6570\u636e\r\n    fmt.Println("Retrieving data from the cluster...")\r\n    val, err := node2.Get(ctx, "cluster-key")\r\n    if err != nil {\r\n        fmt.Printf("Failed to get data: %v\\n", err)\r\n    } else {\r\n        fmt.Printf("Retrieved value: %v\\n", val)\r\n    }\r\n}\n'})}),"\n",(0,i.jsx)(n.h2,{id:"\u793a\u4f8b-6\u6dfb\u52a0\u76d1\u63a7",children:"\u793a\u4f8b 6\uff1a\u6dfb\u52a0\u76d1\u63a7"}),"\n",(0,i.jsx)(n.p,{children:"FyerCache \u63d0\u4f9b\u5185\u7f6e\u76d1\u63a7\u529f\u80fd\uff0c\u53ef\u4ee5\u8f7b\u677e\u8ffd\u8e2a\u7f13\u5b58\u6027\u80fd\u548c\u4f7f\u7528\u60c5\u51b5\uff1a"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:'package main\r\n\r\nimport (\r\n    "context"\r\n    "fmt"\r\n    "time"\r\n\r\n    "github.com/fyerfyer/fyer-cache/cache"\r\n    "github.com/fyerfyer/fyer-cache/cache/metrics"\r\n)\r\n\r\nfunc main() {\r\n    // \u521b\u5efa\u57fa\u7840\u5185\u5b58\u7f13\u5b58\r\n    memCache := cache.NewMemoryCache()\r\n    \r\n    // \u4f7f\u7528\u76d1\u63a7\u88c5\u9970\u5668\u589e\u5f3a\u7f13\u5b58\r\n    monitoredCache := metrics.NewMonitoredCache(\r\n        memCache,\r\n        metrics.WithMetricsServer(":8080"),      // \u57288080\u7aef\u53e3\u66b4\u9732\u6307\u6807\r\n        metrics.WithNamespace("myapp"),          // \u6307\u6807\u547d\u540d\u7a7a\u95f4\r\n        metrics.WithSubsystem("usercache"),      // \u6307\u6807\u5b50\u7cfb\u7edf\r\n        metrics.WithCollectInterval(10*time.Second), // \u6bcf10\u79d2\u6536\u96c6\u4e00\u6b21\u6307\u6807\r\n    )\r\n    \r\n    // \u542f\u52a8\u76d1\u63a7\u670d\u52a1\u5668\r\n    monitoredCache.Start()\r\n    defer monitoredCache.Stop()\r\n    \r\n    fmt.Println("Metrics server started on :8080")\r\n    fmt.Println("Visit http://localhost:8080/metrics to view Prometheus metrics")\r\n    \r\n    ctx := context.Background()\r\n    \r\n    // \u6267\u884c\u4e00\u4e9b\u7f13\u5b58\u64cd\u4f5c\u6765\u751f\u6210\u6307\u6807\r\n    for i := 0; i < 100; i++ {\r\n        key := fmt.Sprintf("user:%d", i)\r\n        monitoredCache.Set(ctx, key, fmt.Sprintf("User %d", i), 10*time.Minute)\r\n    }\r\n    \r\n    // \u8bfb\u53d6\u4e00\u4e9b\u7f13\u5b58\u9879\uff0c\u4ea7\u751f\u547d\u4e2d\u548c\u672a\u547d\u4e2d\r\n    for i := 0; i < 150; i++ {\r\n        key := fmt.Sprintf("user:%d", i)\r\n        monitoredCache.Get(ctx, key)\r\n    }\r\n    \r\n    // \u663e\u793a\u547d\u4e2d\u7387\r\n    fmt.Printf("Current hit rate: %.2f%%\\n", monitoredCache.HitRate()*100)\r\n    \r\n    fmt.Println("Press Ctrl+C to exit")\r\n    \r\n    // \u4f7f\u7a0b\u5e8f\u4fdd\u6301\u8fd0\u884c\u4ee5\u4fbf\u67e5\u770b\u6307\u6807\r\n    select {}\r\n}\n'})}),"\n",(0,i.jsx)(n.h2,{id:"\u793a\u4f8b-7api-\u670d\u52a1\u5668",children:"\u793a\u4f8b 7\uff1aAPI \u670d\u52a1\u5668"}),"\n",(0,i.jsx)(n.p,{children:"FyerCache \u8fd8\u63d0\u4f9b\u4e86 HTTP API \u670d\u52a1\u5668\uff0c\u65b9\u4fbf\u901a\u8fc7 REST API \u7ba1\u7406\u7f13\u5b58\uff1a"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:'package main\r\n\r\nimport (\r\n    "fmt"\r\n    "time"\r\n\r\n    "github.com/fyerfyer/fyer-cache/cache"\r\n    "github.com/fyerfyer/fyer-cache/cache/api"\r\n)\r\n\r\nfunc main() {\r\n    // \u521b\u5efa\u5185\u5b58\u7f13\u5b58\r\n    memCache := cache.NewMemoryCache()\r\n    \r\n    // \u521b\u5efa API \u670d\u52a1\u5668\r\n    apiServer := api.NewAPIServer(\r\n        memCache,\r\n        api.WithBindAddress(":8081"),\r\n        api.WithBasePath("/api/cache"),\r\n        api.WithTimeouts(5*time.Second, 5*time.Second, 30*time.Second),\r\n    )\r\n    \r\n    // \u542f\u52a8 API \u670d\u52a1\u5668\r\n    err := apiServer.Start()\r\n    if err != nil {\r\n        panic(err)\r\n    }\r\n    \r\n    fmt.Println("API server started on :8081")\r\n    fmt.Println("Available endpoints:")\r\n    fmt.Println("  GET  /api/cache/stats - Get cache statistics")\r\n    fmt.Println("  POST /api/cache/invalidate?key=X - Invalidate cache entry")\r\n    \r\n    // \u4f7f\u7a0b\u5e8f\u4fdd\u6301\u8fd0\u884c\r\n    select {}\r\n}\n'})})]})}function l(r={}){const{wrapper:n}={...(0,c.R)(),...r.components};return n?(0,i.jsx)(n,{...r,children:(0,i.jsx)(s,{...r})}):s(r)}},8453:(r,n,e)=>{e.d(n,{R:()=>a,x:()=>o});var t=e(6540);const i={},c=t.createContext(i);function a(r){const n=t.useContext(c);return t.useMemo((function(){return"function"==typeof r?r(n):{...n,...r}}),[n,r])}function o(r){let n;return n=r.disableParentContext?"function"==typeof r.components?r.components(i):r.components||i:a(r.components),t.createElement(c.Provider,{value:n},r.children)}}}]);
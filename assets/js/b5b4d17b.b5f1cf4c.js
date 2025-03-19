"use strict";(self.webpackChunkdocs=self.webpackChunkdocs||[]).push([[92],{3568:(e,n,r)=>{r.r(n),r.d(n,{assets:()=>i,contentTitle:()=>h,default:()=>o,frontMatter:()=>a,metadata:()=>d,toc:()=>l});const d=JSON.parse('{"id":"core-concepts/sharding","title":"Sharding Principles","description":"\u5206\u7247(Sharding)\u662fFyerCache\u7684\u6838\u5fc3\u8bbe\u8ba1\u4e4b\u4e00\uff0c\u901a\u8fc7\u5c06\u6570\u636e\u548c\u8d1f\u8f7d\u5206\u5e03\u5230\u591a\u4e2a\u72ec\u7acb\u5355\u5143\u4e0a\uff0c\u5b9e\u73b0\u9ad8\u6027\u80fd\u3001\u9ad8\u5e76\u53d1\u548c\u53ef\u6269\u5c55\u6027\u3002FyerCache\u5b9e\u73b0\u4e86\u4e24\u4e2a\u5c42\u6b21\u7684\u5206\u7247\u673a\u5236\uff1a\u5185\u5b58\u5206\u7247\u548c\u8282\u70b9\u5206\u7247\uff0c\u5206\u522b\u89e3\u51b3\u5355\u673a\u5e76\u53d1\u548c\u5206\u5e03\u5f0f\u6269\u5c55\u95ee\u9898\u3002","source":"@site/docs/core-concepts/sharding.md","sourceDirName":"core-concepts","slug":"/core-concepts/sharding","permalink":"/fyer-cache/docs/core-concepts/sharding","draft":false,"unlisted":false,"editUrl":"https://github.com/fyerfyer/fyer-rpc/tree/main/docs/core-concepts/sharding.md","tags":[],"version":"current","frontMatter":{},"sidebar":"tutorialSidebar","previous":{"title":"Memcache Implementation","permalink":"/fyer-cache/docs/core-concepts/memory-cache"},"next":{"title":"Cache Consistency Strategy","permalink":"/fyer-cache/docs/distribute-caching/data-consistency"}}');var c=r(4848),s=r(8453);const a={},h="Sharding Principles",i={},l=[{value:"\u5185\u5b58\u5206\u7247 - ShardedMap",id:"\u5185\u5b58\u5206\u7247---shardedmap",level:2},{value:"\u6838\u5fc3\u8bbe\u8ba1",id:"\u6838\u5fc3\u8bbe\u8ba1",level:3},{value:"\u5206\u7247\u7b97\u6cd5",id:"\u5206\u7247\u7b97\u6cd5",level:3},{value:"\u5206\u7247\u6570\u91cf\u9009\u62e9",id:"\u5206\u7247\u6570\u91cf\u9009\u62e9",level:3},{value:"\u5206\u7247\u64cd\u4f5c",id:"\u5206\u7247\u64cd\u4f5c",level:3},{value:"\u8282\u70b9\u5206\u7247 - NodeShardedCache",id:"\u8282\u70b9\u5206\u7247---nodeshardedcache",level:2},{value:"\u6838\u5fc3\u8bbe\u8ba1",id:"\u6838\u5fc3\u8bbe\u8ba1-1",level:3},{value:"\u8282\u70b9\u7ba1\u7406",id:"\u8282\u70b9\u7ba1\u7406",level:3},{value:"\u4e00\u81f4\u6027\u54c8\u5e0c\u5b9e\u73b0",id:"\u4e00\u81f4\u6027\u54c8\u5e0c\u5b9e\u73b0",level:3},{value:"\u865a\u62df\u8282\u70b9\u673a\u5236",id:"\u865a\u62df\u8282\u70b9\u673a\u5236",level:3},{value:"\u6570\u636e\u8def\u7531",id:"\u6570\u636e\u8def\u7531",level:3},{value:"\u526f\u672c\u673a\u5236",id:"\u526f\u672c\u673a\u5236",level:3},{value:"\u8282\u70b9\u53d8\u66f4\u5f71\u54cd",id:"\u8282\u70b9\u53d8\u66f4\u5f71\u54cd",level:3},{value:"\u5206\u5e03\u5f0f\u96c6\u7fa4\u4e2d\u7684\u5206\u7247",id:"\u5206\u5e03\u5f0f\u96c6\u7fa4\u4e2d\u7684\u5206\u7247",level:2},{value:"ShardedClusterCache",id:"shardedclustercache",level:3},{value:"NodeDistributor",id:"nodedistributor",level:3}];function t(e){const n={code:"code",h1:"h1",h2:"h2",h3:"h3",header:"header",li:"li",ol:"ol",p:"p",pre:"pre",strong:"strong",ul:"ul",...(0,s.R)(),...e.components};return(0,c.jsxs)(c.Fragment,{children:[(0,c.jsx)(n.header,{children:(0,c.jsx)(n.h1,{id:"sharding-principles",children:"Sharding Principles"})}),"\n",(0,c.jsx)(n.p,{children:"\u5206\u7247(Sharding)\u662fFyerCache\u7684\u6838\u5fc3\u8bbe\u8ba1\u4e4b\u4e00\uff0c\u901a\u8fc7\u5c06\u6570\u636e\u548c\u8d1f\u8f7d\u5206\u5e03\u5230\u591a\u4e2a\u72ec\u7acb\u5355\u5143\u4e0a\uff0c\u5b9e\u73b0\u9ad8\u6027\u80fd\u3001\u9ad8\u5e76\u53d1\u548c\u53ef\u6269\u5c55\u6027\u3002FyerCache\u5b9e\u73b0\u4e86\u4e24\u4e2a\u5c42\u6b21\u7684\u5206\u7247\u673a\u5236\uff1a\u5185\u5b58\u5206\u7247\u548c\u8282\u70b9\u5206\u7247\uff0c\u5206\u522b\u89e3\u51b3\u5355\u673a\u5e76\u53d1\u548c\u5206\u5e03\u5f0f\u6269\u5c55\u95ee\u9898\u3002"}),"\n",(0,c.jsx)(n.h2,{id:"\u5185\u5b58\u5206\u7247---shardedmap",children:"\u5185\u5b58\u5206\u7247 - ShardedMap"}),"\n",(0,c.jsx)(n.p,{children:"\u5185\u5b58\u5206\u7247\u662fFyerCache\u7684\u57fa\u7840\u5206\u7247\u673a\u5236\uff0c\u89e3\u51b3\u5355\u673a\u591a\u6838\u5e76\u53d1\u8bfb\u5199\u95ee\u9898\u3002"}),"\n",(0,c.jsx)(n.h3,{id:"\u6838\u5fc3\u8bbe\u8ba1",children:"\u6838\u5fc3\u8bbe\u8ba1"}),"\n",(0,c.jsxs)(n.p,{children:[(0,c.jsx)(n.code,{children:"ShardedMap"})," \u5c06\u5355\u4e00map\u62c6\u5206\u4e3a\u591a\u4e2a\u72ec\u7acb\u5206\u7247\uff0c\u6bcf\u4e2a\u5206\u7247\u6709\u81ea\u5df1\u7684\u8bfb\u5199\u9501\uff1a"]}),"\n",(0,c.jsx)(n.pre,{children:(0,c.jsx)(n.code,{className:"language-go",children:"// ShardedMap \u5206\u7247\u6620\u5c04\u7ed3\u6784\r\ntype ShardedMap struct {\r\n    shards    []*Shard     // \u5206\u7247\u6570\u7ec4\r\n    count     int          // \u5206\u7247\u6570\u91cf\r\n    mask      uint32       // \u4f4d\u63a9\u7801\uff0c\u7528\u4e8e\u5feb\u901f\u8ba1\u7b97\u5206\u7247\u7d22\u5f15\r\n    itemCount int64        // \u603b\u9879\u76ee\u6570\u91cf\r\n}\r\n\r\n// Shard \u5355\u4e2a\u5206\u7247\r\ntype Shard struct {\r\n    items map[string]*cacheItem // \u5b9e\u9645\u5b58\u50a8\u7684\u952e\u503c\u5bf9\r\n    mu    sync.RWMutex          // \u5206\u7247\u7ea7\u9501\r\n}\n"})}),"\n",(0,c.jsx)(n.h3,{id:"\u5206\u7247\u7b97\u6cd5",children:"\u5206\u7247\u7b97\u6cd5"}),"\n",(0,c.jsx)(n.p,{children:"FyerCache\u4f7f\u7528FNV\u54c8\u5e0c\u7b97\u6cd5\u548c\u4f4d\u8fd0\u7b97\u5feb\u901f\u786e\u5b9a\u952e\u6240\u5c5e\u5206\u7247\uff1a"}),"\n",(0,c.jsx)(n.pre,{children:(0,c.jsx)(n.code,{className:"language-go",children:"// getShard \u6839\u636e\u952e\u83b7\u53d6\u5bf9\u5e94\u7684\u5206\u7247\r\nfunc (sm *ShardedMap) getShard(key string) *Shard {\r\n    hasher := fnv.New32()\r\n    hasher.Write([]byte(key))\r\n    hash := hasher.Sum32()\r\n\r\n    // \u4f7f\u7528\u4f4d\u63a9\u7801\u5feb\u901f\u8ba1\u7b97\u5206\u7247\u7d22\u5f15\r\n    index := hash & sm.mask\r\n    return sm.shards[index]\r\n}\n"})}),"\n",(0,c.jsx)(n.p,{children:"\u4f4d\u63a9\u7801\u8ba1\u7b97\u76f8\u6bd4\u53d6\u6a21\u8fd0\u7b97\u66f4\u9ad8\u6548\uff0c\u8981\u6c42\u5206\u7247\u6570\u91cf\u662f2\u7684\u5e42\u3002"}),"\n",(0,c.jsx)(n.h3,{id:"\u5206\u7247\u6570\u91cf\u9009\u62e9",children:"\u5206\u7247\u6570\u91cf\u9009\u62e9"}),"\n",(0,c.jsx)(n.p,{children:"FyerCache\u7684\u9ed8\u8ba4\u5206\u7247\u6570\u91cf\u662f32\uff0c\u7528\u6237\u53ef\u4ee5\u6839\u636e\u786c\u4ef6\u7279\u6027\u548c\u6027\u80fd\u9700\u6c42\u8fdb\u884c\u8c03\u6574\uff1a"}),"\n",(0,c.jsx)(n.pre,{children:(0,c.jsx)(n.code,{className:"language-go",children:"// \u901a\u8fc7\u9009\u9879\u8bbe\u7f6e\u5206\u7247\u6570\u91cf\r\ncache := cache.NewMemoryCache(\r\n    cache.WithShardCount(64), // \u4f7f\u752864\u4e2a\u5206\u7247\r\n)\n"})}),"\n",(0,c.jsx)(n.p,{children:"\u5206\u7247\u6570\u91cf\u901a\u5e38\u8bbe\u7f6e\u4e3aCPU\u6838\u5fc3\u6570\u7684\u500d\u6570\uff0c\u4ee5\u83b7\u5f97\u6700\u4f73\u6027\u80fd\u3002\u5982\u679c\u8bbe\u7f6e\u503c\u4e0d\u662f2\u7684\u5e42\uff0c\u7cfb\u7edf\u4f1a\u5411\u4e0a\u53d6\u6574\u5230\u6700\u8fd1\u76842\u7684\u5e42\u3002"}),"\n",(0,c.jsx)(n.h3,{id:"\u5206\u7247\u64cd\u4f5c",children:"\u5206\u7247\u64cd\u4f5c"}),"\n",(0,c.jsx)(n.p,{children:"FyerCache\u5bf9\u5206\u7247\u7684\u6838\u5fc3\u64cd\u4f5c\u5305\u62ec\uff1a"}),"\n",(0,c.jsxs)(n.ul,{children:["\n",(0,c.jsxs)(n.li,{children:[(0,c.jsx)(n.strong,{children:"Store"})," - \u5c06\u952e\u503c\u5bf9\u5b58\u50a8\u5230\u5bf9\u5e94\u5206\u7247"]}),"\n",(0,c.jsxs)(n.li,{children:[(0,c.jsx)(n.strong,{children:"Load"})," - \u4ece\u5bf9\u5e94\u5206\u7247\u52a0\u8f7d\u952e\u503c"]}),"\n",(0,c.jsxs)(n.li,{children:[(0,c.jsx)(n.strong,{children:"Delete"})," - \u4ece\u5bf9\u5e94\u5206\u7247\u5220\u9664\u952e\u503c"]}),"\n",(0,c.jsxs)(n.li,{children:[(0,c.jsx)(n.strong,{children:"Range"})," - \u904d\u5386\u6240\u6709\u5206\u7247\u4e2d\u7684\u952e\u503c\u5bf9"]}),"\n",(0,c.jsxs)(n.li,{children:[(0,c.jsx)(n.strong,{children:"RangeShard"})," - \u4ec5\u904d\u5386\u7279\u5b9a\u5206\u7247\u4e2d\u7684\u952e\u503c\u5bf9"]}),"\n"]}),"\n",(0,c.jsx)(n.p,{children:"\u5206\u7247\u9501\u53ea\u5728\u64cd\u4f5c\u7279\u5b9a\u5206\u7247\u65f6\u83b7\u53d6\uff0c\u5927\u5927\u51cf\u5c11\u4e86\u9501\u7ade\u4e89\uff1a"}),"\n",(0,c.jsx)(n.pre,{children:(0,c.jsx)(n.code,{className:"language-go",children:"// Store \u5b58\u50a8\u952e\u503c\u5bf9\r\nfunc (sm *ShardedMap) Store(key string, value *cacheItem) {\r\n    shard := sm.getShard(key)\r\n    shard.mu.Lock()\r\n    defer shard.mu.Unlock()\r\n\r\n    // \u5b58\u50a8\u64cd\u4f5c...\r\n}\r\n\r\n// Load \u52a0\u8f7d\u6307\u5b9a\u952e\u7684\u503c\r\nfunc (sm *ShardedMap) Load(key string) (*cacheItem, bool) {\r\n    shard := sm.getShard(key)\r\n    shard.mu.RLock()\r\n    defer shard.mu.RUnlock()\r\n\r\n    // \u52a0\u8f7d\u64cd\u4f5c...\r\n}\n"})}),"\n",(0,c.jsx)(n.h2,{id:"\u8282\u70b9\u5206\u7247---nodeshardedcache",children:"\u8282\u70b9\u5206\u7247 - NodeShardedCache"}),"\n",(0,c.jsx)(n.p,{children:"\u8282\u70b9\u5206\u7247\u6784\u5efa\u5728\u5185\u5b58\u5206\u7247\u4e4b\u4e0a\uff0c\u652f\u6301\u8de8\u591a\u4e2a\u8282\u70b9\u5206\u5e03\u6570\u636e\uff0c\u662fFyerCache\u5206\u5e03\u5f0f\u80fd\u529b\u7684\u57fa\u7840\u3002"}),"\n",(0,c.jsx)(n.h3,{id:"\u6838\u5fc3\u8bbe\u8ba1-1",children:"\u6838\u5fc3\u8bbe\u8ba1"}),"\n",(0,c.jsxs)(n.p,{children:[(0,c.jsx)(n.code,{children:"NodeShardedCache"})," \u4f7f\u7528\u4e00\u81f4\u6027\u54c8\u5e0c\u5c06\u6570\u636e\u5206\u5e03\u5230\u591a\u4e2a\u7f13\u5b58\u8282\u70b9\uff1a"]}),"\n",(0,c.jsx)(n.pre,{children:(0,c.jsx)(n.code,{className:"language-go",children:"// NodeShardedCache \u57fa\u4e8e\u8282\u70b9\u7684\u5206\u7247\u7f13\u5b58\u5b9e\u73b0\r\ntype NodeShardedCache struct {\r\n    // \u4e00\u81f4\u6027\u54c8\u5e0c\u73af\r\n    ring Ring\r\n\r\n    // \u8282\u70b9ID\u5230\u7f13\u5b58\u7684\u6620\u5c04\r\n    nodes map[string]Cache\r\n\r\n    // \u4fdd\u62a4\u8282\u70b9\u6620\u5c04\u7684\u5e76\u53d1\u8bbf\u95ee\r\n    mu sync.RWMutex\r\n\r\n    // \u5907\u4efd\u56e0\u5b50\uff0c\u6bcf\u4e2a\u952e\u5b58\u50a8\u5728\u591a\u5c11\u4e2a\u8282\u70b9\u4e0a\uff08\u7528\u4e8e\u5bb9\u9519\uff09\r\n    replicaFactor int\r\n}\n"})}),"\n",(0,c.jsx)(n.h3,{id:"\u8282\u70b9\u7ba1\u7406",children:"\u8282\u70b9\u7ba1\u7406"}),"\n",(0,c.jsx)(n.p,{children:"NodeShardedCache\u652f\u6301\u52a8\u6001\u6dfb\u52a0\u548c\u79fb\u9664\u8282\u70b9\uff1a"}),"\n",(0,c.jsx)(n.pre,{children:(0,c.jsx)(n.code,{className:"language-go",children:'// \u521b\u5efa\u8282\u70b9\u5206\u7247\u7f13\u5b58\r\nshardedCache := cache.NewNodeShardedCache()\r\n\r\n// \u6dfb\u52a0\u4e09\u4e2a\u8282\u70b9\uff0c\u6bcf\u4e2a\u6709\u4e0d\u540c\u6743\u91cd\r\nshardedCache.AddNode("node1", cache1, 1)  // \u6743\u91cd1\r\nshardedCache.AddNode("node2", cache2, 2)  // \u6743\u91cd2\r\nshardedCache.AddNode("node3", cache3, 3)  // \u6743\u91cd3\r\n\r\n// \u79fb\u9664\u8282\u70b9\r\nshardedCache.RemoveNode("node2")\n'})}),"\n",(0,c.jsx)(n.p,{children:"\u8282\u70b9\u6743\u91cd\u5f71\u54cd\u6570\u636e\u5206\u5e03\uff0c\u6743\u91cd\u8d8a\u9ad8\uff0c\u5206\u914d\u7ed9\u8be5\u8282\u70b9\u7684\u6570\u636e\u6bd4\u4f8b\u8d8a\u5927\u3002"}),"\n",(0,c.jsx)(n.h3,{id:"\u4e00\u81f4\u6027\u54c8\u5e0c\u5b9e\u73b0",children:"\u4e00\u81f4\u6027\u54c8\u5e0c\u5b9e\u73b0"}),"\n",(0,c.jsx)(n.p,{children:"\u4e00\u81f4\u6027\u54c8\u5e0c\u662f\u8282\u70b9\u5206\u7247\u7684\u6838\u5fc3\u7b97\u6cd5\uff0c\u786e\u4fdd\u6570\u636e\u5747\u5300\u5206\u5e03\u5e76\u6700\u5c0f\u5316\u8282\u70b9\u53d8\u66f4\u65f6\u7684\u6570\u636e\u8fc1\u79fb\uff1a"}),"\n",(0,c.jsx)(n.pre,{children:(0,c.jsx)(n.code,{className:"language-go",children:"// ConsistentHash \u4e00\u81f4\u6027\u54c8\u5e0c\u73af\u5b9e\u73b0\r\ntype ConsistentHash struct {\r\n    // \u54c8\u5e0c\u51fd\u6570\r\n    hashFunc HashFunc\r\n\r\n    // \u6bcf\u4e2a\u7269\u7406\u8282\u70b9\u5bf9\u5e94\u7684\u865a\u62df\u8282\u70b9\u6570\u91cf\r\n    replicas int\r\n\r\n    // \u5df2\u6392\u5e8f\u7684\u54c8\u5e0c\u503c\r\n    sortedHashes []uint32\r\n\r\n    // \u54c8\u5e0c\u503c\u5230\u8282\u70b9\u7684\u6620\u5c04\r\n    hashMap map[uint32]string\r\n\r\n    // \u8282\u70b9\u6743\u91cd\r\n    weights map[string]int\r\n\r\n    // \u8282\u70b9\u5230\u5176\u54c8\u5e0c\u503c\u7684\u6620\u5c04\uff0c\u7528\u4e8e\u9ad8\u6548\u5220\u9664\r\n    nodeHashes map[string][]uint32\r\n\r\n    // \u4fdd\u62a4\u73af\u7684\u5e76\u53d1\u8bbf\u95ee\r\n    mu sync.RWMutex\r\n}\n"})}),"\n",(0,c.jsx)(n.h3,{id:"\u865a\u62df\u8282\u70b9\u673a\u5236",children:"\u865a\u62df\u8282\u70b9\u673a\u5236"}),"\n",(0,c.jsx)(n.p,{children:"\u4e3a\u63d0\u9ad8\u6570\u636e\u5747\u5300\u6027\uff0cFyerCache\u5b9e\u73b0\u4e86\u865a\u62df\u8282\u70b9\u673a\u5236\uff0c\u9ed8\u8ba4\u6bcf\u4e2a\u7269\u7406\u8282\u70b9\u5bf9\u5e94100\u4e2a\u865a\u62df\u8282\u70b9\uff1a"}),"\n",(0,c.jsx)(n.pre,{children:(0,c.jsx)(n.code,{className:"language-go",children:"// \u521b\u5efa\u8282\u70b9\u5206\u7247\u7f13\u5b58\u65f6\u8bbe\u7f6e\u865a\u62df\u8282\u70b9\u6570\u91cf\r\nshardedCache := cache.NewNodeShardedCache(\r\n    cache.WithNodeHashReplicas(200), // \u6bcf\u4e2a\u7269\u7406\u8282\u70b9200\u4e2a\u865a\u62df\u8282\u70b9\r\n)\n"})}),"\n",(0,c.jsx)(n.h3,{id:"\u6570\u636e\u8def\u7531",children:"\u6570\u636e\u8def\u7531"}),"\n",(0,c.jsx)(n.p,{children:"\u5f53\u9700\u8981\u5b58\u50a8\u6216\u68c0\u7d22\u6570\u636e\u65f6\uff0c\u4e00\u81f4\u6027\u54c8\u5e0c\u786e\u5b9a\u952e\u6240\u5c5e\u7684\u8282\u70b9\uff1a"}),"\n",(0,c.jsx)(n.pre,{children:(0,c.jsx)(n.code,{className:"language-go",children:"// Get \u4ece\u6b63\u786e\u7684\u8282\u70b9\u83b7\u53d6\u7f13\u5b58\u9879\r\nfunc (nsc *NodeShardedCache) Get(ctx context.Context, key string) (any, error) {\r\n    // \u68c0\u67e5\u662f\u5426\u6709\u8282\u70b9\r\n    if nsc.GetNodeCount() == 0 {\r\n        return nil, ferr.ErrNoNodesAvailable\r\n    }\r\n\r\n    // \u83b7\u53d6\u5305\u542b\u8be5\u952e\u7684\u6240\u6709\u8282\u70b9\uff08\u4e3b\u8282\u70b9\u548c\u5907\u4efd\u8282\u70b9\uff09\r\n    nodeIDs := nsc.ring.GetN(key, nsc.replicaFactor)\r\n    // ...\u83b7\u53d6\u6570\u636e\u903b\u8f91...\r\n}\n"})}),"\n",(0,c.jsx)(n.h3,{id:"\u526f\u672c\u673a\u5236",children:"\u526f\u672c\u673a\u5236"}),"\n",(0,c.jsx)(n.p,{children:"\u4e3a\u63d0\u9ad8\u53ef\u7528\u6027\uff0cFyerCache\u652f\u6301\u5c06\u6570\u636e\u590d\u5236\u5230\u591a\u4e2a\u8282\u70b9\uff1a"}),"\n",(0,c.jsx)(n.pre,{children:(0,c.jsx)(n.code,{className:"language-go",children:"// \u521b\u5efa\u5e26\u67093\u4e2a\u526f\u672c\u7684\u8282\u70b9\u5206\u7247\u7f13\u5b58\r\nshardedCache := cache.NewNodeShardedCache(\r\n    cache.WithNodeReplicaFactor(3), // \u6bcf\u4e2a\u952e\u5b58\u50a8\u57283\u4e2a\u8282\u70b9\u4e0a\r\n)\n"})}),"\n",(0,c.jsx)(n.p,{children:"\u5f53\u67d0\u4e2a\u8282\u70b9\u5931\u8d25\u65f6\uff0c\u7cfb\u7edf\u53ef\u4ee5\u4ece\u5907\u4efd\u8282\u70b9\u68c0\u7d22\u6570\u636e\uff1a"}),"\n",(0,c.jsx)(n.pre,{children:(0,c.jsx)(n.code,{className:"language-go",children:"// \u5c1d\u8bd5\u4ece\u6240\u6709\u53ef\u80fd\u7684\u8282\u70b9\u83b7\u53d6\u6570\u636e\r\nfunc (nsc *NodeShardedCache) Get(ctx context.Context, key string) (any, error) {\r\n    // ...\u524d\u9762\u7684\u4ee3\u7801...\r\n    \r\n    // \u9996\u5148\u5c1d\u8bd5\u4e3b\u8282\u70b9\r\n    if val, err := nodeCache.Get(ctx, key); err == nil {\r\n        return val, nil\r\n    }\r\n    \r\n    // \u5982\u679c\u4e3b\u8282\u70b9\u5931\u8d25\uff0c\u5c1d\u8bd5\u4ece\u5907\u4efd\u8282\u70b9\u83b7\u53d6\r\n    for i := 1; i < len(nodes) && i < len(nodeIDs); i++ {\r\n        if val, err := nodes[i].Get(ctx, key); err == nil {\r\n            return val, nil\r\n        }\r\n    }\r\n    \r\n    return nil, ferr.ErrKeyNotFound\r\n}\n"})}),"\n",(0,c.jsx)(n.h3,{id:"\u8282\u70b9\u53d8\u66f4\u5f71\u54cd",children:"\u8282\u70b9\u53d8\u66f4\u5f71\u54cd"}),"\n",(0,c.jsx)(n.p,{children:"\u4e00\u81f4\u6027\u54c8\u5e0c\u7684\u5173\u952e\u4f18\u52bf\u662f\u8282\u70b9\u53d8\u66f4\u65f6\u6700\u5c0f\u5316\u6570\u636e\u8fc1\u79fb\uff1a"}),"\n",(0,c.jsxs)(n.ul,{children:["\n",(0,c.jsx)(n.li,{children:"\u5f53\u8282\u70b9\u52a0\u5165\u6216\u79bb\u5f00\u65f6\uff0c\u53ea\u6709\u73af\u4e0a\u76f8\u90bb\u8282\u70b9\u8d1f\u8d23\u7684\u4e00\u5c0f\u90e8\u5206\u952e\u9700\u8981\u91cd\u65b0\u5206\u914d"}),"\n",(0,c.jsx)(n.li,{children:"\u7406\u8bba\u4e0a\uff0c\u6dfb\u52a0\u6216\u5220\u9664N\u8282\u70b9\u96c6\u7fa4\u4e2d\u76841\u4e2a\u8282\u70b9\uff0c\u53ea\u67091/N\u7684\u6570\u636e\u9700\u8981\u91cd\u65b0\u5206\u914d"}),"\n"]}),"\n",(0,c.jsx)(n.h2,{id:"\u5206\u5e03\u5f0f\u96c6\u7fa4\u4e2d\u7684\u5206\u7247",children:"\u5206\u5e03\u5f0f\u96c6\u7fa4\u4e2d\u7684\u5206\u7247"}),"\n",(0,c.jsx)(n.p,{children:"\u5728\u5b8c\u6574\u7684\u5206\u5e03\u5f0f\u7cfb\u7edf\u4e2d\uff0cFyerCache\u7ed3\u5408\u4e86\u8282\u70b9\u5206\u7247\u548c\u96c6\u7fa4\u7ba1\u7406\u529f\u80fd\uff0c\u63d0\u4f9b\u66f4\u5f3a\u5927\u7684\u5206\u5e03\u5f0f\u7f13\u5b58\u529f\u80fd\u3002"}),"\n",(0,c.jsx)(n.h3,{id:"shardedclustercache",children:"ShardedClusterCache"}),"\n",(0,c.jsxs)(n.p,{children:[(0,c.jsx)(n.code,{children:"ShardedClusterCache"})," \u6574\u5408\u4e86\u4e00\u81f4\u6027\u54c8\u5e0c\u5206\u7247\u4e0e\u96c6\u7fa4\u7ba1\u7406\uff1a"]}),"\n",(0,c.jsx)(n.pre,{children:(0,c.jsx)(n.code,{className:"language-go",children:"// ShardedClusterCache \u6574\u5408\u4e00\u81f4\u6027\u54c8\u5e0c\u5206\u7247\u4e0e\u96c6\u7fa4\u529f\u80fd\u7684\u7f13\u5b58\u5b9e\u73b0\r\ntype ShardedClusterCache struct {\r\n    // \u672c\u5730\u7f13\u5b58\u5b9e\u4f8b\r\n    localCache cache.Cache\r\n\r\n    // \u5206\u7247\u7f13\u5b58\u5b9e\u4f8b\uff0c\u7528\u4e8e\u8def\u7531\u8bf7\u6c42\r\n    shardedCache *cache.NodeShardedCache\r\n\r\n    // \u96c6\u7fa4\u8282\u70b9\r\n    node *cluster.Node\r\n\r\n    // \u8282\u70b9\u5206\u53d1\u5668\r\n    nodeDistributor *NodeDistributor\r\n    \r\n    // ...\u5176\u4ed6\u5b57\u6bb5...\r\n}\n"})}),"\n",(0,c.jsx)(n.p,{children:"\u8be5\u5b9e\u73b0\u63d0\u4f9b\u4ee5\u4e0b\u529f\u80fd\uff1a"}),"\n",(0,c.jsxs)(n.ol,{children:["\n",(0,c.jsxs)(n.li,{children:[(0,c.jsx)(n.strong,{children:"\u52a8\u6001\u96c6\u7fa4\u6210\u5458\u7ba1\u7406"})," - \u8282\u70b9\u53ef\u4ee5\u52a8\u6001\u52a0\u5165\u548c\u79bb\u5f00\u96c6\u7fa4"]}),"\n",(0,c.jsxs)(n.li,{children:[(0,c.jsx)(n.strong,{children:"\u81ea\u52a8\u5206\u7247\u8c03\u6574"})," - \u6839\u636e\u96c6\u7fa4\u6210\u5458\u53d8\u5316\u81ea\u52a8\u8c03\u6574\u5206\u7247\u5206\u914d"]}),"\n",(0,c.jsxs)(n.li,{children:[(0,c.jsx)(n.strong,{children:"\u96c6\u7fa4\u4e8b\u4ef6\u5904\u7406"})," - \u54cd\u5e94\u8282\u70b9\u52a0\u5165\u3001\u79bb\u5f00\u548c\u6545\u969c\u4e8b\u4ef6"]}),"\n",(0,c.jsxs)(n.li,{children:[(0,c.jsx)(n.strong,{children:"\u8fdc\u7a0b\u8282\u70b9\u4ee3\u7406"})," - \u4e3a\u8fdc\u7a0b\u8282\u70b9\u521b\u5efa\u672c\u5730\u4ee3\u7406"]}),"\n"]}),"\n",(0,c.jsx)(n.h3,{id:"nodedistributor",children:"NodeDistributor"}),"\n",(0,c.jsxs)(n.p,{children:[(0,c.jsx)(n.code,{children:"NodeDistributor"})," \u662f\u96c6\u7fa4\u4e0e\u5206\u7247\u7684\u534f\u8c03\u5668\uff0c\u8d1f\u8d23\u7ba1\u7406\u8282\u70b9\u53d8\u66f4\u4e0e\u4e00\u81f4\u6027\u54c8\u5e0c\u73af\u7684\u540c\u6b65\uff1a"]}),"\n",(0,c.jsx)(n.pre,{children:(0,c.jsx)(n.code,{className:"language-go",children:"// NodeDistributor \u7ba1\u7406\u96c6\u7fa4\u8282\u70b9\u548c\u4e00\u81f4\u6027\u54c8\u5e0c\u73af\u4e4b\u95f4\u7684\u6620\u5c04\u5173\u7cfb\r\nfunc (nd *NodeDistributor) HandleClusterEvent(event cache.ClusterEvent) {\r\n    // ...\r\n    switch event.Type {\r\n    case cache.EventNodeJoin:\r\n        // \u8282\u70b9\u52a0\u5165\u65f6\uff0c\u521b\u5efa\u8fdc\u7a0b\u7f13\u5b58\u4ee3\u7406\u5e76\u6dfb\u52a0\u5230\u4e00\u81f4\u6027\u54c8\u5e0c\u73af\r\n        \r\n    case cache.EventNodeLeave, cache.EventNodeFailed:\r\n        // \u8282\u70b9\u79bb\u5f00\u6216\u6545\u969c\u65f6\uff0c\u4ece\u4e00\u81f4\u6027\u54c8\u5e0c\u73af\u4e2d\u79fb\u9664\r\n        \r\n    case cache.EventNodeRecovered:\r\n        // \u8282\u70b9\u6062\u590d\u65f6\uff0c\u91cd\u65b0\u6dfb\u52a0\u5230\u4e00\u81f4\u6027\u54c8\u5e0c\u73af\r\n    }\r\n    // ...\r\n}\n"})})]})}function o(e={}){const{wrapper:n}={...(0,s.R)(),...e.components};return n?(0,c.jsx)(n,{...e,children:(0,c.jsx)(t,{...e})}):t(e)}},8453:(e,n,r)=>{r.d(n,{R:()=>a,x:()=>h});var d=r(6540);const c={},s=d.createContext(c);function a(e){const n=d.useContext(s);return d.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function h(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(c):e.components||c:a(e.components),d.createElement(s.Provider,{value:n},e.children)}}}]);
"use strict";(self.webpackChunkdocs=self.webpackChunkdocs||[]).push([[976],{7879:(n,s,e)=>{e.r(s),e.d(s,{assets:()=>h,contentTitle:()=>d,default:()=>j,frontMatter:()=>c,metadata:()=>r,toc:()=>t});const r=JSON.parse('{"id":"intro","title":"fyer-cache","description":"FyerCache \u662f\u4e00\u4e2a\u9ad8\u6027\u80fd\u3001\u7075\u6d3b\u4e14\u53ef\u6269\u5c55\u7684 Go \u8bed\u8a00\u7f13\u5b58\u7cfb\u7edf\uff0c\u63d0\u4f9b\u4ece\u5355\u673a\u5185\u5b58\u7f13\u5b58\u5230\u590d\u6742\u5206\u5e03\u5f0f\u7f13\u5b58\u96c6\u7fa4\u7684\u5b8c\u6574\u89e3\u51b3\u65b9\u6848\u3002","source":"@site/docs/intro.md","sourceDirName":".","slug":"/intro","permalink":"/fyer-cache/docs/intro","draft":false,"unlisted":false,"editUrl":"https://github.com/fyerfyer/fyer-rpc/tree/main/docs/intro.md","tags":[],"version":"current","frontMatter":{},"sidebar":"tutorialSidebar","previous":{"title":"Using Pattern","permalink":"/fyer-cache/docs/getting-started/usingpattern"}}');var l=e(4848),i=e(8453);const c={},d="fyer-cache",h={},t=[{value:"\u8bbe\u8ba1\u7406\u5ff5",id:"\u8bbe\u8ba1\u7406\u5ff5",level:2},{value:"\u6838\u5fc3\u529f\u80fd",id:"\u6838\u5fc3\u529f\u80fd",level:2},{value:"\u5185\u5b58\u7f13\u5b58",id:"\u5185\u5b58\u7f13\u5b58",level:3},{value:"\u5206\u5e03\u5f0f\u80fd\u529b",id:"\u5206\u5e03\u5f0f\u80fd\u529b",level:3},{value:"\u96c6\u7fa4\u7ba1\u7406",id:"\u96c6\u7fa4\u7ba1\u7406",level:3},{value:"\u591a\u79cd\u96c6\u7fa4\u67b6\u6784",id:"\u591a\u79cd\u96c6\u7fa4\u67b6\u6784",level:3},{value:"\u7f13\u5b58\u4e00\u81f4\u6027",id:"\u7f13\u5b58\u4e00\u81f4\u6027",level:3},{value:"\u76d1\u63a7\u4e0e\u53ef\u89c2\u6d4b\u6027",id:"\u76d1\u63a7\u4e0e\u53ef\u89c2\u6d4b\u6027",level:3},{value:"API \u548c\u7ba1\u7406",id:"api-\u548c\u7ba1\u7406",level:3},{value:"\u4f7f\u7528\u573a\u666f",id:"\u4f7f\u7528\u573a\u666f",level:2},{value:"\u67b6\u6784\u6982\u8ff0",id:"\u67b6\u6784\u6982\u8ff0",level:2}];function x(n){const s={h1:"h1",h2:"h2",h3:"h3",header:"header",li:"li",ol:"ol",p:"p",strong:"strong",ul:"ul",...(0,i.R)(),...n.components};return(0,l.jsxs)(l.Fragment,{children:[(0,l.jsx)(s.header,{children:(0,l.jsx)(s.h1,{id:"fyer-cache",children:"fyer-cache"})}),"\n",(0,l.jsx)(s.p,{children:"FyerCache \u662f\u4e00\u4e2a\u9ad8\u6027\u80fd\u3001\u7075\u6d3b\u4e14\u53ef\u6269\u5c55\u7684 Go \u8bed\u8a00\u7f13\u5b58\u7cfb\u7edf\uff0c\u63d0\u4f9b\u4ece\u5355\u673a\u5185\u5b58\u7f13\u5b58\u5230\u590d\u6742\u5206\u5e03\u5f0f\u7f13\u5b58\u96c6\u7fa4\u7684\u5b8c\u6574\u89e3\u51b3\u65b9\u6848\u3002"}),"\n",(0,l.jsx)(s.h2,{id:"\u8bbe\u8ba1\u7406\u5ff5",children:"\u8bbe\u8ba1\u7406\u5ff5"}),"\n",(0,l.jsx)(s.p,{children:"FyerCache \u7684\u8bbe\u8ba1\u57fa\u4e8e\u4ee5\u4e0b\u5173\u952e\u7406\u5ff5\uff1a"}),"\n",(0,l.jsxs)(s.ul,{children:["\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u7b80\u5355\u6027\u4e0e\u7075\u6d3b\u6027"})," - \u63d0\u4f9b\u7b80\u6d01\u76f4\u89c2\u7684\u63a5\u53e3\uff0c\u540c\u65f6\u901a\u8fc7\u9009\u9879\u6a21\u5f0f\u652f\u6301\u9ad8\u5ea6\u81ea\u5b9a\u4e49"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u5206\u5c42\u67b6\u6784"})," - \u4ece\u5185\u5b58\u7f13\u5b58\u5230\u5206\u5e03\u5f0f\u96c6\u7fa4\u7684\u6e10\u8fdb\u5f0f\u529f\u80fd\u5c42\u6b21\uff0c\u53ef\u6839\u636e\u9700\u6c42\u9009\u62e9\u5408\u9002\u7684\u590d\u6742\u5ea6"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u9ad8\u6027\u80fd"})," - \u901a\u8fc7\u5206\u7247\u51cf\u5c11\u9501\u7ade\u4e89\uff0c\u4f18\u5316\u5e76\u53d1\u6027\u80fd"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u53ef\u6269\u5c55\u6027"})," - \u652f\u6301\u6c34\u5e73\u6269\u5c55\uff0c\u52a8\u6001\u6dfb\u52a0\u548c\u79fb\u9664\u8282\u70b9"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u97e7\u6027\u548c\u53ef\u9760\u6027"})," - \u5185\u7f6e\u6545\u969c\u68c0\u6d4b\u548c\u81ea\u52a8\u6062\u590d\u673a\u5236"]}),"\n"]}),"\n",(0,l.jsx)(s.h2,{id:"\u6838\u5fc3\u529f\u80fd",children:"\u6838\u5fc3\u529f\u80fd"}),"\n",(0,l.jsx)(s.h3,{id:"\u5185\u5b58\u7f13\u5b58",children:"\u5185\u5b58\u7f13\u5b58"}),"\n",(0,l.jsxs)(s.ul,{children:["\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u9ad8\u6548\u7684\u5185\u5b58\u7ba1\u7406"})," - \u4f18\u5316\u7684\u952e\u503c\u5b58\u50a8\uff0c\u652f\u6301\u5404\u79cd\u6570\u636e\u7c7b\u578b"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u81ea\u52a8\u8fc7\u671f\u673a\u5236"})," - \u652f\u6301\u57fa\u4e8e TTL \u7684\u7f13\u5b58\u9879\u8fc7\u671f\u8bbe\u7f6e"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u591a\u79cd\u7f13\u5b58\u6e05\u7406\u7b56\u7565"})," - \u652f\u6301\u540c\u6b65\u548c\u5f02\u6b65\u6e05\u7406\u8fc7\u671f\u9879"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u5206\u7247\u8bbe\u8ba1"})," - \u4f7f\u7528\u591a\u5206\u7247\u7ed3\u6784\u51cf\u5c11\u9501\u7ade\u4e89\uff0c\u63d0\u5347\u5e76\u53d1\u6027\u80fd"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u5185\u5b58\u7ea6\u675f"})," - \u53ef\u9009\u7684\u5185\u5b58\u4f7f\u7528\u9650\u5236\uff0c\u652f\u6301 LRU \u6dd8\u6c70\u7b56\u7565"]}),"\n"]}),"\n",(0,l.jsx)(s.h3,{id:"\u5206\u5e03\u5f0f\u80fd\u529b",children:"\u5206\u5e03\u5f0f\u80fd\u529b"}),"\n",(0,l.jsxs)(s.ul,{children:["\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u4e00\u81f4\u6027\u54c8\u5e0c\u5206\u7247"})," - \u4f7f\u7528\u4e00\u81f4\u6027\u54c8\u5e0c\u7b97\u6cd5\u5b9e\u73b0\u6570\u636e\u5206\u7247\uff0c\u51cf\u5c11\u8282\u70b9\u53d8\u66f4\u65f6\u6570\u636e\u8fc1\u79fb"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u865a\u62df\u8282\u70b9"})," - \u652f\u6301\u865a\u62df\u8282\u70b9\u673a\u5236\uff0c\u63d0\u9ad8\u6570\u636e\u5206\u5e03\u5747\u5300\u6027"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u8282\u70b9\u6743\u91cd"})," - \u53ef\u4e3a\u4e0d\u540c\u8282\u70b9\u5206\u914d\u4e0d\u540c\u6743\u91cd\uff0c\u9002\u5e94\u5f02\u6784\u73af\u5883"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u8fdc\u7a0b\u7f13\u5b58\u5ba2\u6237\u7aef"})," - \u63d0\u4f9b\u900f\u660e\u7684\u8fdc\u7a0b\u8282\u70b9\u8bbf\u95ee\u80fd\u529b"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u53ef\u914d\u7f6e\u526f\u672c\u56e0\u5b50"})," - \u652f\u6301\u6570\u636e\u591a\u8282\u70b9\u5907\u4efd\uff0c\u63d0\u9ad8\u53ef\u7528\u6027"]}),"\n"]}),"\n",(0,l.jsx)(s.h3,{id:"\u96c6\u7fa4\u7ba1\u7406",children:"\u96c6\u7fa4\u7ba1\u7406"}),"\n",(0,l.jsxs)(s.ul,{children:["\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u8282\u70b9\u53d1\u73b0\u548c\u6210\u5458\u7ba1\u7406"})," - \u81ea\u52a8\u53d1\u73b0\u548c\u7ef4\u62a4\u96c6\u7fa4\u6210\u5458\u5217\u8868"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"Gossip\u534f\u8bae"})," - \u57fa\u4e8e Gossip \u7684\u8282\u70b9\u4fe1\u606f\u4f20\u64ad\u673a\u5236"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u6545\u969c\u68c0\u6d4b"})," - \u5065\u5eb7\u68c0\u67e5\u548c\u8282\u70b9\u72b6\u6001\u76d1\u63a7"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u81ea\u52a8\u540c\u6b65"})," - \u8282\u70b9\u95f4\u6570\u636e\u81ea\u52a8\u540c\u6b65\u673a\u5236"]}),"\n"]}),"\n",(0,l.jsx)(s.h3,{id:"\u591a\u79cd\u96c6\u7fa4\u67b6\u6784",children:"\u591a\u79cd\u96c6\u7fa4\u67b6\u6784"}),"\n",(0,l.jsxs)(s.ul,{children:["\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u5206\u7247\u96c6\u7fa4"})," - \u6570\u636e\u901a\u8fc7\u4e00\u81f4\u6027\u54c8\u5e0c\u5206\u5e03\u5728\u6240\u6709\u8282\u70b9\u4e0a"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u4e3b\u4ece\u67b6\u6784"})," - \u652f\u6301\u8bfb\u5199\u5206\u79bb\u7684\u4e3b\u4ece\u590d\u5236\u6a21\u5f0f"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u9009\u4e3e\u673a\u5236"})," - \u652f\u6301\u9886\u5bfc\u8005\u9009\u4e3e\uff0c\u786e\u4fdd\u96c6\u7fa4\u9ad8\u53ef\u7528"]}),"\n"]}),"\n",(0,l.jsx)(s.h3,{id:"\u7f13\u5b58\u4e00\u81f4\u6027",children:"\u7f13\u5b58\u4e00\u81f4\u6027"}),"\n",(0,l.jsxs)(s.ul,{children:["\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"Cache-Aside \u6a21\u5f0f"})," - \u7ecf\u5178\u7684\u7f13\u5b58\u65c1\u8def\u6a21\u5f0f"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"Write-Through \u6a21\u5f0f"})," - \u5199\u900f\u6a21\u5f0f\uff0c\u540c\u65f6\u66f4\u65b0\u6570\u636e\u6e90\u548c\u7f13\u5b58"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u6d88\u606f\u901a\u77e5\u673a\u5236"})," - \u57fa\u4e8e\u6d88\u606f\u961f\u5217\u7684\u4e00\u81f4\u6027\u7b56\u7565\uff0c\u652f\u6301\u8de8\u8282\u70b9\u7f13\u5b58\u5931\u6548"]}),"\n"]}),"\n",(0,l.jsx)(s.h3,{id:"\u76d1\u63a7\u4e0e\u53ef\u89c2\u6d4b\u6027",children:"\u76d1\u63a7\u4e0e\u53ef\u89c2\u6d4b\u6027"}),"\n",(0,l.jsxs)(s.ul,{children:["\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u4e30\u5bcc\u7684\u6307\u6807"})," - \u5185\u7f6e\u547d\u4e2d\u7387\u3001\u5ef6\u8fdf\u3001\u5185\u5b58\u4f7f\u7528\u7387\u7b49\u6307\u6807"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"Prometheus \u96c6\u6210"})," - \u652f\u6301 Prometheus \u683c\u5f0f\u7684\u6307\u6807\u5bfc\u51fa"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u4e8b\u4ef6\u901a\u77e5"})," - \u652f\u6301\u96c6\u7fa4\u4e8b\u4ef6\u901a\u77e5\u548c\u81ea\u5b9a\u4e49\u5904\u7406"]}),"\n"]}),"\n",(0,l.jsx)(s.h3,{id:"api-\u548c\u7ba1\u7406",children:"API \u548c\u7ba1\u7406"}),"\n",(0,l.jsxs)(s.ul,{children:["\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"HTTP API"})," - \u63d0\u4f9b RESTful \u7ba1\u7406\u63a5\u53e3"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u5065\u5eb7\u68c0\u67e5"})," - \u5185\u7f6e\u5065\u5eb7\u68c0\u67e5\u7aef\u70b9"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u52a8\u6001\u914d\u7f6e"})," - \u652f\u6301\u8fd0\u884c\u65f6\u914d\u7f6e\u8c03\u6574"]}),"\n"]}),"\n",(0,l.jsx)(s.h2,{id:"\u4f7f\u7528\u573a\u666f",children:"\u4f7f\u7528\u573a\u666f"}),"\n",(0,l.jsx)(s.p,{children:"FyerCache \u9002\u7528\u4e8e\u591a\u79cd\u573a\u666f\uff0c\u5305\u62ec\u4f46\u4e0d\u9650\u4e8e\uff1a"}),"\n",(0,l.jsxs)(s.ul,{children:["\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u5e94\u7528\u7a0b\u5e8f\u7f13\u5b58"})," - \u51cf\u8f7b\u6570\u636e\u5e93\u8d1f\u8f7d\uff0c\u63d0\u9ad8\u5e94\u7528\u54cd\u5e94\u901f\u5ea6"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u4f1a\u8bdd\u5b58\u50a8"})," - \u9ad8\u6027\u80fd\u7684\u5206\u5e03\u5f0f\u4f1a\u8bdd\u7ba1\u7406"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"API \u8bf7\u6c42\u7ed3\u679c\u7f13\u5b58"})," - \u51cf\u5c11\u91cd\u590d\u8ba1\u7b97\uff0c\u63d0\u9ad8 API \u6027\u80fd"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u6570\u636e\u5e93\u67e5\u8be2\u7f13\u5b58"})," - \u7f13\u5b58\u9891\u7e41\u8bbf\u95ee\u7684\u67e5\u8be2\u7ed3\u679c"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u5206\u5e03\u5f0f\u9501\u548c\u4fe1\u53f7\u91cf"})," - \u652f\u6301\u5206\u5e03\u5f0f\u534f\u8c03"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u9ad8\u9891\u8ba1\u6570\u548c\u7edf\u8ba1"})," - \u5b9e\u65f6\u7edf\u8ba1\u548c\u8ba1\u6570\u5e94\u7528"]}),"\n"]}),"\n",(0,l.jsx)(s.h2,{id:"\u67b6\u6784\u6982\u8ff0",children:"\u67b6\u6784\u6982\u8ff0"}),"\n",(0,l.jsx)(s.p,{children:"FyerCache \u91c7\u7528\u6a21\u5757\u5316\u8bbe\u8ba1\uff0c\u4e3b\u8981\u7531\u4ee5\u4e0b\u51e0\u4e2a\u90e8\u5206\u7ec4\u6210\uff1a"}),"\n",(0,l.jsxs)(s.ol,{children:["\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u6838\u5fc3\u7f13\u5b58\u5f15\u64ce"})," - \u63d0\u4f9b\u57fa\u7840\u7684 Get/Set/Del \u64cd\u4f5c\u548c\u5185\u5b58\u7ba1\u7406"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u5206\u7247\u673a\u5236"})," - \u5b9e\u73b0\u5e76\u53d1\u53cb\u597d\u7684\u6570\u636e\u5206\u7247\u7b56\u7565"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u4e00\u81f4\u6027\u54c8\u5e0c"})," - \u5904\u7406\u5206\u5e03\u5f0f\u73af\u5883\u4e2d\u7684\u6570\u636e\u5206\u7247\u548c\u8d1f\u8f7d\u5747\u8861"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u96c6\u7fa4\u7ba1\u7406"})," - \u8d1f\u8d23\u8282\u70b9\u53d1\u73b0\u3001\u6210\u5458\u7ba1\u7406\u548c\u6545\u969c\u68c0\u6d4b"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u590d\u5236\u7cfb\u7edf"})," - \u5904\u7406\u4e3b\u4ece\u590d\u5236\u548c\u6570\u636e\u540c\u6b65"]}),"\n",(0,l.jsxs)(s.li,{children:[(0,l.jsx)(s.strong,{children:"\u76d1\u63a7\u5b50\u7cfb\u7edf"})," - \u6536\u96c6\u548c\u5bfc\u51fa\u6027\u80fd\u6307\u6807"]}),"\n"]})]})}function j(n={}){const{wrapper:s}={...(0,i.R)(),...n.components};return s?(0,l.jsx)(s,{...n,children:(0,l.jsx)(x,{...n})}):x(n)}},8453:(n,s,e)=>{e.d(s,{R:()=>c,x:()=>d});var r=e(6540);const l={},i=r.createContext(l);function c(n){const s=r.useContext(i);return r.useMemo((function(){return"function"==typeof n?n(s):{...s,...n}}),[s,n])}function d(n){let s;return s=n.disableParentContext?"function"==typeof n.components?n.components(l):n.components||l:c(n.components),r.createElement(i.Provider,{value:s},n.children)}}}]);
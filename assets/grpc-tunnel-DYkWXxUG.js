import{r as e}from"./chunk-BA1M-is3.js";import{t}from"./jsx-runtime-Bc7xHnoj.js";var n=e(t()),r={title:`gRPC Tunnel Design`,description:`Current gRPC stream design and why it replaced polling.`},i=`

gRPC Tunnel Design [#grpc-tunnel-design]

The current transport uses one long-lived gRPC stream per active tunnel.

Benefits:

* outbound-only client connectivity
* low-latency server-to-client request dispatch
* room for heartbeats and future stream extensions
* simpler operator deployment than request polling

Current limitations:

* WebSocket support is still pending
* tunnel metadata is not yet fully server-owned
`,a={contents:[{heading:`grpc-tunnel-design`,content:`The current transport uses one long-lived gRPC stream per active tunnel.`},{heading:`grpc-tunnel-design`,content:`Benefits:`},{heading:`grpc-tunnel-design`,content:`outbound-only client connectivity`},{heading:`grpc-tunnel-design`,content:`low-latency server-to-client request dispatch`},{heading:`grpc-tunnel-design`,content:`room for heartbeats and future stream extensions`},{heading:`grpc-tunnel-design`,content:`simpler operator deployment than request polling`},{heading:`grpc-tunnel-design`,content:`Current limitations:`},{heading:`grpc-tunnel-design`,content:`WebSocket support is still pending`},{heading:`grpc-tunnel-design`,content:`tunnel metadata is not yet fully server-owned`}],headings:[{id:`grpc-tunnel-design`,content:`gRPC Tunnel Design`}]},o=[{depth:1,url:`#grpc-tunnel-design`,title:(0,n.jsx)(n.Fragment,{children:`gRPC Tunnel Design`})}];function s(e){let t={h1:`h1`,li:`li`,p:`p`,ul:`ul`,...e.components};return(0,n.jsxs)(n.Fragment,{children:[(0,n.jsx)(t.h1,{id:`grpc-tunnel-design`,children:`gRPC Tunnel Design`}),`
`,(0,n.jsx)(t.p,{children:`The current transport uses one long-lived gRPC stream per active tunnel.`}),`
`,(0,n.jsx)(t.p,{children:`Benefits:`}),`
`,(0,n.jsxs)(t.ul,{children:[`
`,(0,n.jsx)(t.li,{children:`outbound-only client connectivity`}),`
`,(0,n.jsx)(t.li,{children:`low-latency server-to-client request dispatch`}),`
`,(0,n.jsx)(t.li,{children:`room for heartbeats and future stream extensions`}),`
`,(0,n.jsx)(t.li,{children:`simpler operator deployment than request polling`}),`
`]}),`
`,(0,n.jsx)(t.p,{children:`Current limitations:`}),`
`,(0,n.jsxs)(t.ul,{children:[`
`,(0,n.jsx)(t.li,{children:`WebSocket support is still pending`}),`
`,(0,n.jsx)(t.li,{children:`tunnel metadata is not yet fully server-owned`}),`
`]})]})}function c(e={}){let{wrapper:t}=e.components||{};return t?(0,n.jsx)(t,{...e,children:(0,n.jsx)(s,{...e})}):s(e)}export{i as _markdown,c as default,r as frontmatter,a as structuredData,o as toc};
import{r as e}from"./chunk-BA1M-is3.js";import{t}from"./jsx-runtime-Bc7xHnoj.js";var n=e(t()),r={title:`Architecture / Spec`,description:`Current tunnel flow and the rewrite direction toward a fuller control plane.`},i=`

Architecture / Spec [#architecture--spec]

The current implementation uses:

* public HTTP ingress on the server
* a gRPC tunnel stream from client to server
* request forwarding over that stream
* server-side admin APIs and UI

Architecture Pages [#architecture-pages]

* [Traffic Flow](/docs/architecture/traffic-flow)
* [gRPC Tunnel Design](/docs/architecture/grpc-tunnel)
* [Rewrite Direction](/docs/architecture/rewrite-direction)
`,a={contents:[{heading:`architecture--spec`,content:`The current implementation uses:`},{heading:`architecture--spec`,content:`public HTTP ingress on the server`},{heading:`architecture--spec`,content:`a gRPC tunnel stream from client to server`},{heading:`architecture--spec`,content:`request forwarding over that stream`},{heading:`architecture--spec`,content:`server-side admin APIs and UI`},{heading:`architecture-pages`,content:`Traffic Flow`},{heading:`architecture-pages`,content:`gRPC Tunnel Design`},{heading:`architecture-pages`,content:`Rewrite Direction`}],headings:[{id:`architecture--spec`,content:`Architecture / Spec`},{id:`architecture-pages`,content:`Architecture Pages`}]},o=[{depth:1,url:`#architecture--spec`,title:(0,n.jsx)(n.Fragment,{children:`Architecture / Spec`})},{depth:2,url:`#architecture-pages`,title:(0,n.jsx)(n.Fragment,{children:`Architecture Pages`})}];function s(e){let t={a:`a`,h1:`h1`,h2:`h2`,li:`li`,p:`p`,ul:`ul`,...e.components};return(0,n.jsxs)(n.Fragment,{children:[(0,n.jsx)(t.h1,{id:`architecture--spec`,children:`Architecture / Spec`}),`
`,(0,n.jsx)(t.p,{children:`The current implementation uses:`}),`
`,(0,n.jsxs)(t.ul,{children:[`
`,(0,n.jsx)(t.li,{children:`public HTTP ingress on the server`}),`
`,(0,n.jsx)(t.li,{children:`a gRPC tunnel stream from client to server`}),`
`,(0,n.jsx)(t.li,{children:`request forwarding over that stream`}),`
`,(0,n.jsx)(t.li,{children:`server-side admin APIs and UI`}),`
`]}),`
`,(0,n.jsx)(t.h2,{id:`architecture-pages`,children:`Architecture Pages`}),`
`,(0,n.jsxs)(t.ul,{children:[`
`,(0,n.jsx)(t.li,{children:(0,n.jsx)(t.a,{href:`/docs/architecture/traffic-flow`,children:`Traffic Flow`})}),`
`,(0,n.jsx)(t.li,{children:(0,n.jsx)(t.a,{href:`/docs/architecture/grpc-tunnel`,children:`gRPC Tunnel Design`})}),`
`,(0,n.jsx)(t.li,{children:(0,n.jsx)(t.a,{href:`/docs/architecture/rewrite-direction`,children:`Rewrite Direction`})}),`
`]})]})}function c(e={}){let{wrapper:t}=e.components||{};return t?(0,n.jsx)(t,{...e,children:(0,n.jsx)(s,{...e})}):s(e)}export{i as _markdown,c as default,r as frontmatter,a as structuredData,o as toc};
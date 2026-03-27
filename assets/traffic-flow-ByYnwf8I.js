import{r as e}from"./chunk-BA1M-is3.js";import{t}from"./jsx-runtime-Bc7xHnoj.js";var n=e(t()),r={title:`Traffic Flow`,description:`How public requests reach the client-side local app.`},i=`

Traffic Flow [#traffic-flow]

Current flow:

1. Client opens a gRPC stream to the server.
2. Client registers a hostname.
3. Browser sends a request to \`https://<hostname>\`.
4. Server looks up the active tunnel.
5. Server sends a proxied request over the stream.
6. Client forwards that request to the local app.
7. Client returns the response over the stream.
8. Server returns that response to the browser.

This model works behind NAT because the client initiates the outbound connection.
`,a={contents:[{heading:`traffic-flow`,content:`Current flow:`},{heading:`traffic-flow`,content:`Client opens a gRPC stream to the server.`},{heading:`traffic-flow`,content:`Client registers a hostname.`},{heading:`traffic-flow`,content:"Browser sends a request to `https://<hostname>`."},{heading:`traffic-flow`,content:`Server looks up the active tunnel.`},{heading:`traffic-flow`,content:`Server sends a proxied request over the stream.`},{heading:`traffic-flow`,content:`Client forwards that request to the local app.`},{heading:`traffic-flow`,content:`Client returns the response over the stream.`},{heading:`traffic-flow`,content:`Server returns that response to the browser.`},{heading:`traffic-flow`,content:`This model works behind NAT because the client initiates the outbound connection.`}],headings:[{id:`traffic-flow`,content:`Traffic Flow`}]},o=[{depth:1,url:`#traffic-flow`,title:(0,n.jsx)(n.Fragment,{children:`Traffic Flow`})}];function s(e){let t={code:`code`,h1:`h1`,li:`li`,ol:`ol`,p:`p`,...e.components};return(0,n.jsxs)(n.Fragment,{children:[(0,n.jsx)(t.h1,{id:`traffic-flow`,children:`Traffic Flow`}),`
`,(0,n.jsx)(t.p,{children:`Current flow:`}),`
`,(0,n.jsxs)(t.ol,{children:[`
`,(0,n.jsx)(t.li,{children:`Client opens a gRPC stream to the server.`}),`
`,(0,n.jsx)(t.li,{children:`Client registers a hostname.`}),`
`,(0,n.jsxs)(t.li,{children:[`Browser sends a request to `,(0,n.jsx)(t.code,{children:`https://<hostname>`}),`.`]}),`
`,(0,n.jsx)(t.li,{children:`Server looks up the active tunnel.`}),`
`,(0,n.jsx)(t.li,{children:`Server sends a proxied request over the stream.`}),`
`,(0,n.jsx)(t.li,{children:`Client forwards that request to the local app.`}),`
`,(0,n.jsx)(t.li,{children:`Client returns the response over the stream.`}),`
`,(0,n.jsx)(t.li,{children:`Server returns that response to the browser.`}),`
`]}),`
`,(0,n.jsx)(t.p,{children:`This model works behind NAT because the client initiates the outbound connection.`})]})}function c(e={}){let{wrapper:t}=e.components||{};return t?(0,n.jsx)(t,{...e,children:(0,n.jsx)(s,{...e})}):s(e)}export{i as _markdown,c as default,r as frontmatter,a as structuredData,o as toc};
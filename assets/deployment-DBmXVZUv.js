import{r as e}from"./chunk-BA1M-is3.js";import{t}from"./jsx-runtime-Bc7xHnoj.js";var n=e(t()),r={title:`Deployment`,description:`Deploy fwdx with nginx, TLS, and systemd.`},i=`

Deployment [#deployment]

The recommended deployment model is:

1. \`fwdx\` server running locally on private ports
2. nginx terminating TLS on \`443\` and \`4443\`
3. wildcard DNS for \`*.tunnel.example.com\`

Sections [#sections]

* [nginx](/docs/deployment/nginx)
* [systemd](/docs/deployment/systemd)
* [TLS and DNS](/docs/deployment/tls-dns)
`,a={contents:[{heading:`deployment`,content:`The recommended deployment model is:`},{heading:`deployment`,content:"`fwdx` server running locally on private ports"},{heading:`deployment`,content:"nginx terminating TLS on `443` and `4443`"},{heading:`deployment`,content:"wildcard DNS for `*.tunnel.example.com`"},{heading:`sections`,content:`nginx`},{heading:`sections`,content:`systemd`},{heading:`sections`,content:`TLS and DNS`}],headings:[{id:`deployment`,content:`Deployment`},{id:`sections`,content:`Sections`}]},o=[{depth:1,url:`#deployment`,title:(0,n.jsx)(n.Fragment,{children:`Deployment`})},{depth:2,url:`#sections`,title:(0,n.jsx)(n.Fragment,{children:`Sections`})}];function s(e){let t={a:`a`,code:`code`,h1:`h1`,h2:`h2`,li:`li`,ol:`ol`,p:`p`,ul:`ul`,...e.components};return(0,n.jsxs)(n.Fragment,{children:[(0,n.jsx)(t.h1,{id:`deployment`,children:`Deployment`}),`
`,(0,n.jsx)(t.p,{children:`The recommended deployment model is:`}),`
`,(0,n.jsxs)(t.ol,{children:[`
`,(0,n.jsxs)(t.li,{children:[(0,n.jsx)(t.code,{children:`fwdx`}),` server running locally on private ports`]}),`
`,(0,n.jsxs)(t.li,{children:[`nginx terminating TLS on `,(0,n.jsx)(t.code,{children:`443`}),` and `,(0,n.jsx)(t.code,{children:`4443`})]}),`
`,(0,n.jsxs)(t.li,{children:[`wildcard DNS for `,(0,n.jsx)(t.code,{children:`*.tunnel.example.com`})]}),`
`]}),`
`,(0,n.jsx)(t.h2,{id:`sections`,children:`Sections`}),`
`,(0,n.jsxs)(t.ul,{children:[`
`,(0,n.jsx)(t.li,{children:(0,n.jsx)(t.a,{href:`/docs/deployment/nginx`,children:`nginx`})}),`
`,(0,n.jsx)(t.li,{children:(0,n.jsx)(t.a,{href:`/docs/deployment/systemd`,children:`systemd`})}),`
`,(0,n.jsx)(t.li,{children:(0,n.jsx)(t.a,{href:`/docs/deployment/tls-dns`,children:`TLS and DNS`})}),`
`]})]})}function c(e={}){let{wrapper:t}=e.components||{};return t?(0,n.jsx)(t,{...e,children:(0,n.jsx)(s,{...e})}):s(e)}export{i as _markdown,c as default,r as frontmatter,a as structuredData,o as toc};
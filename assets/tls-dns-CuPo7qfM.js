import{r as e}from"./chunk-BA1M-is3.js";import{t}from"./jsx-runtime-Bc7xHnoj.js";var n=e(t()),r={title:`TLS and DNS`,description:`DNS records and certificates required for public tunnel routing.`},i=`

TLS and DNS [#tls-and-dns]

DNS [#dns]

You need:

* \`tunnel.example.com\` -> server public IP
* \`*.tunnel.example.com\` -> server public IP

TLS [#tls]

The certificate must cover:

* \`tunnel.example.com\`
* \`*.tunnel.example.com\`

If nginx terminates TLS, fwdx itself can stay on plain local ports.
`,a={contents:[{heading:`dns`,content:`You need:`},{heading:`dns`,content:"`tunnel.example.com` -> server public IP"},{heading:`dns`,content:"`*.tunnel.example.com` -> server public IP"},{heading:`tls`,content:`The certificate must cover:`},{heading:`tls`,content:"`tunnel.example.com`"},{heading:`tls`,content:"`*.tunnel.example.com`"},{heading:`tls`,content:`If nginx terminates TLS, fwdx itself can stay on plain local ports.`}],headings:[{id:`tls-and-dns`,content:`TLS and DNS`},{id:`dns`,content:`DNS`},{id:`tls`,content:`TLS`}]},o=[{depth:1,url:`#tls-and-dns`,title:(0,n.jsx)(n.Fragment,{children:`TLS and DNS`})},{depth:2,url:`#dns`,title:(0,n.jsx)(n.Fragment,{children:`DNS`})},{depth:2,url:`#tls`,title:(0,n.jsx)(n.Fragment,{children:`TLS`})}];function s(e){let t={code:`code`,h1:`h1`,h2:`h2`,li:`li`,p:`p`,ul:`ul`,...e.components};return(0,n.jsxs)(n.Fragment,{children:[(0,n.jsx)(t.h1,{id:`tls-and-dns`,children:`TLS and DNS`}),`
`,(0,n.jsx)(t.h2,{id:`dns`,children:`DNS`}),`
`,(0,n.jsx)(t.p,{children:`You need:`}),`
`,(0,n.jsxs)(t.ul,{children:[`
`,(0,n.jsxs)(t.li,{children:[(0,n.jsx)(t.code,{children:`tunnel.example.com`}),` -> server public IP`]}),`
`,(0,n.jsxs)(t.li,{children:[(0,n.jsx)(t.code,{children:`*.tunnel.example.com`}),` -> server public IP`]}),`
`]}),`
`,(0,n.jsx)(t.h2,{id:`tls`,children:`TLS`}),`
`,(0,n.jsx)(t.p,{children:`The certificate must cover:`}),`
`,(0,n.jsxs)(t.ul,{children:[`
`,(0,n.jsx)(t.li,{children:(0,n.jsx)(t.code,{children:`tunnel.example.com`})}),`
`,(0,n.jsx)(t.li,{children:(0,n.jsx)(t.code,{children:`*.tunnel.example.com`})}),`
`]}),`
`,(0,n.jsx)(t.p,{children:`If nginx terminates TLS, fwdx itself can stay on plain local ports.`})]})}function c(e={}){let{wrapper:t}=e.components||{};return t?(0,n.jsx)(t,{...e,children:(0,n.jsx)(s,{...e})}):s(e)}export{i as _markdown,c as default,r as frontmatter,a as structuredData,o as toc};
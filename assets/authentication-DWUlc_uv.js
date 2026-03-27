import{r as e}from"./chunk-BA1M-is3.js";import{t}from"./jsx-runtime-Bc7xHnoj.js";var n=e(t()),r={title:`Authentication`,description:`OIDC-only human authentication and current tunnel runtime auth.`},i=`

Authentication [#authentication]

\`fwdx\` now has a split auth model:

* humans authenticate with OIDC
* tunnel runtimes authenticate with per-agent credentials

Human auth [#human-auth]

* Admin UI: OIDC authorization code flow with PKCE
* CLI: OIDC device authorization flow via \`fwdx login\`

Tunnel runtime auth [#tunnel-runtime-auth]

Tunnel clients do not use a shared global runtime token anymore.

* the server issues a credential to a named agent
* tunnel definitions are stored on the server and assigned to an agent
* the local client stores its assigned agent credential in \`~/.fwdx/client.json\`

Pages [#pages]

* [OIDC UI Login](/docs/authentication/oidc-ui)
* [CLI Device Flow](/docs/authentication/device-flow)
`,a={contents:[{heading:`authentication`,content:"`fwdx` now has a split auth model:"},{heading:`authentication`,content:`humans authenticate with OIDC`},{heading:`authentication`,content:`tunnel runtimes authenticate with per-agent credentials`},{heading:`human-auth`,content:`Admin UI: OIDC authorization code flow with PKCE`},{heading:`human-auth`,content:"CLI: OIDC device authorization flow via `fwdx login`"},{heading:`tunnel-runtime-auth`,content:`Tunnel clients do not use a shared global runtime token anymore.`},{heading:`tunnel-runtime-auth`,content:`the server issues a credential to a named agent`},{heading:`tunnel-runtime-auth`,content:`tunnel definitions are stored on the server and assigned to an agent`},{heading:`tunnel-runtime-auth`,content:"the local client stores its assigned agent credential in `~/.fwdx/client.json`"},{heading:`pages`,content:`OIDC UI Login`},{heading:`pages`,content:`CLI Device Flow`}],headings:[{id:`authentication`,content:`Authentication`},{id:`human-auth`,content:`Human auth`},{id:`tunnel-runtime-auth`,content:`Tunnel runtime auth`},{id:`pages`,content:`Pages`}]},o=[{depth:1,url:`#authentication`,title:(0,n.jsx)(n.Fragment,{children:`Authentication`})},{depth:2,url:`#human-auth`,title:(0,n.jsx)(n.Fragment,{children:`Human auth`})},{depth:2,url:`#tunnel-runtime-auth`,title:(0,n.jsx)(n.Fragment,{children:`Tunnel runtime auth`})},{depth:2,url:`#pages`,title:(0,n.jsx)(n.Fragment,{children:`Pages`})}];function s(e){let t={a:`a`,code:`code`,h1:`h1`,h2:`h2`,li:`li`,p:`p`,ul:`ul`,...e.components};return(0,n.jsxs)(n.Fragment,{children:[(0,n.jsx)(t.h1,{id:`authentication`,children:`Authentication`}),`
`,(0,n.jsxs)(t.p,{children:[(0,n.jsx)(t.code,{children:`fwdx`}),` now has a split auth model:`]}),`
`,(0,n.jsxs)(t.ul,{children:[`
`,(0,n.jsx)(t.li,{children:`humans authenticate with OIDC`}),`
`,(0,n.jsx)(t.li,{children:`tunnel runtimes authenticate with per-agent credentials`}),`
`]}),`
`,(0,n.jsx)(t.h2,{id:`human-auth`,children:`Human auth`}),`
`,(0,n.jsxs)(t.ul,{children:[`
`,(0,n.jsx)(t.li,{children:`Admin UI: OIDC authorization code flow with PKCE`}),`
`,(0,n.jsxs)(t.li,{children:[`CLI: OIDC device authorization flow via `,(0,n.jsx)(t.code,{children:`fwdx login`})]}),`
`]}),`
`,(0,n.jsx)(t.h2,{id:`tunnel-runtime-auth`,children:`Tunnel runtime auth`}),`
`,(0,n.jsx)(t.p,{children:`Tunnel clients do not use a shared global runtime token anymore.`}),`
`,(0,n.jsxs)(t.ul,{children:[`
`,(0,n.jsx)(t.li,{children:`the server issues a credential to a named agent`}),`
`,(0,n.jsx)(t.li,{children:`tunnel definitions are stored on the server and assigned to an agent`}),`
`,(0,n.jsxs)(t.li,{children:[`the local client stores its assigned agent credential in `,(0,n.jsx)(t.code,{children:`~/.fwdx/client.json`})]}),`
`]}),`
`,(0,n.jsx)(t.h2,{id:`pages`,children:`Pages`}),`
`,(0,n.jsxs)(t.ul,{children:[`
`,(0,n.jsx)(t.li,{children:(0,n.jsx)(t.a,{href:`/docs/authentication/oidc-ui`,children:`OIDC UI Login`})}),`
`,(0,n.jsx)(t.li,{children:(0,n.jsx)(t.a,{href:`/docs/authentication/device-flow`,children:`CLI Device Flow`})}),`
`]})]})}function c(e={}){let{wrapper:t}=e.components||{};return t?(0,n.jsx)(t,{...e,children:(0,n.jsx)(s,{...e})}):s(e)}export{i as _markdown,c as default,r as frontmatter,a as structuredData,o as toc};
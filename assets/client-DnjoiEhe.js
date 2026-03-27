import{r as e}from"./chunk-BA1M-is3.js";import{t}from"./jsx-runtime-Bc7xHnoj.js";var n=e(t()),r={title:`Client Install`,description:`Configure and run a local fwdx tunnel client.`},i=`

Client Install [#client-install]

Install or build the same \`fwdx\` binary on the client machine.

Set client configuration:

\`\`\`bash
export FWDX_SERVER=https://tunnel.example.com
export FWDX_TUNNEL_PORT=4443
\`\`\`

Create and start a tunnel:

\`\`\`bash
fwdx tunnel create -l localhost:5173 -s vite --name vite
fwdx tunnel start vite
\`\`\`

The first tunnel create/start on a machine provisions and stores a local agent credential in \`~/.fwdx/client.json\`.

Detached mode:

\`\`\`bash
fwdx tunnel start vite --detach
fwdx logs vite --follow
\`\`\`
`,a={contents:[{heading:`client-install`,content:"Install or build the same `fwdx` binary on the client machine."},{heading:`client-install`,content:`Set client configuration:`},{heading:`client-install`,content:`Create and start a tunnel:`},{heading:`client-install`,content:"The first tunnel create/start on a machine provisions and stores a local agent credential in `~/.fwdx/client.json`."},{heading:`client-install`,content:`Detached mode:`}],headings:[{id:`client-install`,content:`Client Install`}]},o=[{depth:1,url:`#client-install`,title:(0,n.jsx)(n.Fragment,{children:`Client Install`})}];function s(e){let t={code:`code`,h1:`h1`,p:`p`,pre:`pre`,span:`span`,...e.components};return(0,n.jsxs)(n.Fragment,{children:[(0,n.jsx)(t.h1,{id:`client-install`,children:`Client Install`}),`
`,(0,n.jsxs)(t.p,{children:[`Install or build the same `,(0,n.jsx)(t.code,{children:`fwdx`}),` binary on the client machine.`]}),`
`,(0,n.jsx)(t.p,{children:`Set client configuration:`}),`
`,(0,n.jsx)(n.Fragment,{children:(0,n.jsx)(t.pre,{className:`shiki shiki-themes github-light github-dark`,style:{"--shiki-light":`#24292e`,"--shiki-dark":`#e1e4e8`,"--shiki-light-bg":`#fff`,"--shiki-dark-bg":`#24292e`},tabIndex:`0`,icon:`<svg viewBox="0 0 24 24"><path d="m 4,4 a 1,1 0 0 0 -0.7070312,0.2929687 1,1 0 0 0 0,1.4140625 L 8.5859375,11 3.2929688,16.292969 a 1,1 0 0 0 0,1.414062 1,1 0 0 0 1.4140624,0 l 5.9999998,-6 a 1.0001,1.0001 0 0 0 0,-1.414062 L 4.7070312,4.2929687 A 1,1 0 0 0 4,4 Z m 8,14 a 1,1 0 0 0 -1,1 1,1 0 0 0 1,1 h 8 a 1,1 0 0 0 1,-1 1,1 0 0 0 -1,-1 z" fill="currentColor" /></svg>`,children:(0,n.jsxs)(t.code,{children:[(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`export`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:` FWDX_SERVER`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`=`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`https://tunnel.example.com`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`export`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:` FWDX_TUNNEL_PORT`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`=`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:`4443`})]})]})})}),`
`,(0,n.jsx)(t.p,{children:`Create and start a tunnel:`}),`
`,(0,n.jsx)(n.Fragment,{children:(0,n.jsx)(t.pre,{className:`shiki shiki-themes github-light github-dark`,style:{"--shiki-light":`#24292e`,"--shiki-dark":`#e1e4e8`,"--shiki-light-bg":`#fff`,"--shiki-dark-bg":`#24292e`},tabIndex:`0`,icon:`<svg viewBox="0 0 24 24"><path d="m 4,4 a 1,1 0 0 0 -0.7070312,0.2929687 1,1 0 0 0 0,1.4140625 L 8.5859375,11 3.2929688,16.292969 a 1,1 0 0 0 0,1.414062 1,1 0 0 0 1.4140624,0 l 5.9999998,-6 a 1.0001,1.0001 0 0 0 0,-1.414062 L 4.7070312,4.2929687 A 1,1 0 0 0 4,4 Z m 8,14 a 1,1 0 0 0 -1,1 1,1 0 0 0 1,1 h 8 a 1,1 0 0 0 1,-1 1,1 0 0 0 -1,-1 z" fill="currentColor" /></svg>`,children:(0,n.jsxs)(t.code,{children:[(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#6F42C1`,"--shiki-dark":`#B392F0`},children:`fwdx`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` tunnel`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` create`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` -l`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` localhost:5173`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` -s`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` vite`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` --name`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` vite`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#6F42C1`,"--shiki-dark":`#B392F0`},children:`fwdx`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` tunnel`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` start`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` vite`})]})]})})}),`
`,(0,n.jsxs)(t.p,{children:[`The first tunnel create/start on a machine provisions and stores a local agent credential in `,(0,n.jsx)(t.code,{children:`~/.fwdx/client.json`}),`.`]}),`
`,(0,n.jsx)(t.p,{children:`Detached mode:`}),`
`,(0,n.jsx)(n.Fragment,{children:(0,n.jsx)(t.pre,{className:`shiki shiki-themes github-light github-dark`,style:{"--shiki-light":`#24292e`,"--shiki-dark":`#e1e4e8`,"--shiki-light-bg":`#fff`,"--shiki-dark-bg":`#24292e`},tabIndex:`0`,icon:`<svg viewBox="0 0 24 24"><path d="m 4,4 a 1,1 0 0 0 -0.7070312,0.2929687 1,1 0 0 0 0,1.4140625 L 8.5859375,11 3.2929688,16.292969 a 1,1 0 0 0 0,1.414062 1,1 0 0 0 1.4140624,0 l 5.9999998,-6 a 1.0001,1.0001 0 0 0 0,-1.414062 L 4.7070312,4.2929687 A 1,1 0 0 0 4,4 Z m 8,14 a 1,1 0 0 0 -1,1 1,1 0 0 0 1,1 h 8 a 1,1 0 0 0 1,-1 1,1 0 0 0 -1,-1 z" fill="currentColor" /></svg>`,children:(0,n.jsxs)(t.code,{children:[(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#6F42C1`,"--shiki-dark":`#B392F0`},children:`fwdx`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` tunnel`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` start`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` vite`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` --detach`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#6F42C1`,"--shiki-dark":`#B392F0`},children:`fwdx`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` logs`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` vite`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` --follow`})]})]})})})]})}function c(e={}){let{wrapper:t}=e.components||{};return t?(0,n.jsx)(t,{...e,children:(0,n.jsx)(s,{...e})}):s(e)}export{i as _markdown,c as default,r as frontmatter,a as structuredData,o as toc};
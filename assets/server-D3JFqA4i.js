import{r as e}from"./chunk-BA1M-is3.js";import{t}from"./jsx-runtime-Bc7xHnoj.js";var n=e(t()),r={title:`Server Install`,description:`Build and install the fwdx server binary.`},i=`

Server Install [#server-install]

Build from source:

\`\`\`bash
git clone https://github.com/BRAVO68WEB/fwdx.git
cd fwdx
go build -o fwdx .
sudo mv fwdx /usr/local/bin/
\`\`\`

Run behind nginx:

\`\`\`bash
fwdx serve \\
  --hostname tunnel.example.com \\
  --oidc-issuer https://issuer.example.com \\
  --oidc-client-id fwdx-web \\
  --oidc-client-secret YOUR_OIDC_SECRET \\
  --oidc-redirect-url https://tunnel.example.com/auth/oidc/callback \\
  --oidc-admin-emails admin@example.com \\
  --web-port 4040 \\
  --grpc-port 4440 \\
  --data-dir /var/lib/fwdx
\`\`\`

The web port serves proxy traffic, admin UI, OIDC callback routes, and admin APIs. The gRPC port serves tunnel clients.
`,a={contents:[{heading:`server-install`,content:`Build from source:`},{heading:`server-install`,content:`Run behind nginx:`},{heading:`server-install`,content:`The web port serves proxy traffic, admin UI, OIDC callback routes, and admin APIs. The gRPC port serves tunnel clients.`}],headings:[{id:`server-install`,content:`Server Install`}]},o=[{depth:1,url:`#server-install`,title:(0,n.jsx)(n.Fragment,{children:`Server Install`})}];function s(e){let t={code:`code`,h1:`h1`,p:`p`,pre:`pre`,span:`span`,...e.components};return(0,n.jsxs)(n.Fragment,{children:[(0,n.jsx)(t.h1,{id:`server-install`,children:`Server Install`}),`
`,(0,n.jsx)(t.p,{children:`Build from source:`}),`
`,(0,n.jsx)(n.Fragment,{children:(0,n.jsx)(t.pre,{className:`shiki shiki-themes github-light github-dark`,style:{"--shiki-light":`#24292e`,"--shiki-dark":`#e1e4e8`,"--shiki-light-bg":`#fff`,"--shiki-dark-bg":`#24292e`},tabIndex:`0`,icon:`<svg viewBox="0 0 24 24"><path d="m 4,4 a 1,1 0 0 0 -0.7070312,0.2929687 1,1 0 0 0 0,1.4140625 L 8.5859375,11 3.2929688,16.292969 a 1,1 0 0 0 0,1.414062 1,1 0 0 0 1.4140624,0 l 5.9999998,-6 a 1.0001,1.0001 0 0 0 0,-1.414062 L 4.7070312,4.2929687 A 1,1 0 0 0 4,4 Z m 8,14 a 1,1 0 0 0 -1,1 1,1 0 0 0 1,1 h 8 a 1,1 0 0 0 1,-1 1,1 0 0 0 -1,-1 z" fill="currentColor" /></svg>`,children:(0,n.jsxs)(t.code,{children:[(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#6F42C1`,"--shiki-dark":`#B392F0`},children:`git`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` clone`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` https://github.com/BRAVO68WEB/fwdx.git`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:`cd`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` fwdx`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#6F42C1`,"--shiki-dark":`#B392F0`},children:`go`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` build`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` -o`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` fwdx`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` .`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#6F42C1`,"--shiki-dark":`#B392F0`},children:`sudo`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` mv`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` fwdx`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` /usr/local/bin/`})]})]})})}),`
`,(0,n.jsx)(t.p,{children:`Run behind nginx:`}),`
`,(0,n.jsx)(n.Fragment,{children:(0,n.jsx)(t.pre,{className:`shiki shiki-themes github-light github-dark`,style:{"--shiki-light":`#24292e`,"--shiki-dark":`#e1e4e8`,"--shiki-light-bg":`#fff`,"--shiki-dark-bg":`#24292e`},tabIndex:`0`,icon:`<svg viewBox="0 0 24 24"><path d="m 4,4 a 1,1 0 0 0 -0.7070312,0.2929687 1,1 0 0 0 0,1.4140625 L 8.5859375,11 3.2929688,16.292969 a 1,1 0 0 0 0,1.414062 1,1 0 0 0 1.4140624,0 l 5.9999998,-6 a 1.0001,1.0001 0 0 0 0,-1.414062 L 4.7070312,4.2929687 A 1,1 0 0 0 4,4 Z m 8,14 a 1,1 0 0 0 -1,1 1,1 0 0 0 1,1 h 8 a 1,1 0 0 0 1,-1 1,1 0 0 0 -1,-1 z" fill="currentColor" /></svg>`,children:(0,n.jsxs)(t.code,{children:[(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#6F42C1`,"--shiki-dark":`#B392F0`},children:`fwdx`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` serve`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` \\`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:`  --hostname`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` tunnel.example.com`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` \\`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:`  --oidc-issuer`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` https://issuer.example.com`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` \\`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:`  --oidc-client-id`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` fwdx-web`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` \\`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:`  --oidc-client-secret`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` YOUR_OIDC_SECRET`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` \\`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:`  --oidc-redirect-url`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` https://tunnel.example.com/auth/oidc/callback`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` \\`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:`  --oidc-admin-emails`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` admin@example.com`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` \\`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:`  --web-port`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` 4040`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` \\`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:`  --grpc-port`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` 4440`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` \\`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:`  --data-dir`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` /var/lib/fwdx`})]})]})})}),`
`,(0,n.jsx)(t.p,{children:`The web port serves proxy traffic, admin UI, OIDC callback routes, and admin APIs. The gRPC port serves tunnel clients.`})]})}function c(e={}){let{wrapper:t}=e.components||{};return t?(0,n.jsx)(t,{...e,children:(0,n.jsx)(s,{...e})}):s(e)}export{i as _markdown,c as default,r as frontmatter,a as structuredData,o as toc};
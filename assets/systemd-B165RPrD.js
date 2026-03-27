import{r as e}from"./chunk-BA1M-is3.js";import{t}from"./jsx-runtime-Bc7xHnoj.js";var n=e(t()),r={title:`systemd`,description:`Run the fwdx server as a long-lived Linux service.`},i=`

systemd [#systemd]

Example service:

\`\`\`ini
[Unit]
Description=fwdx server
After=network.target nginx.service

[Service]
Type=simple
ExecStart=/usr/local/bin/fwdx serve \\
  --hostname tunnel.example.com \\
  --oidc-issuer https://issuer.example.com \\
  --oidc-client-id fwdx-web \\
  --oidc-client-secret YOUR_OIDC_SECRET \\
  --oidc-redirect-url https://tunnel.example.com/auth/oidc/callback \\
  --oidc-admin-emails admin@example.com \\
  --web-port 4040 \\
  --grpc-port 4440 \\
  --data-dir /var/lib/fwdx
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
\`\`\`

Logs:

\`\`\`bash
journalctl -u fwdx -f
\`\`\`
`,a={contents:[{heading:`systemd`,content:`Example service:`},{heading:`systemd`,content:`Logs:`}],headings:[{id:`systemd`,content:`systemd`}]},o=[{depth:1,url:`#systemd`,title:(0,n.jsx)(n.Fragment,{children:`systemd`})}];function s(e){let t={code:`code`,h1:`h1`,p:`p`,pre:`pre`,span:`span`,...e.components};return(0,n.jsxs)(n.Fragment,{children:[(0,n.jsx)(t.h1,{id:`systemd`,children:`systemd`}),`
`,(0,n.jsx)(t.p,{children:`Example service:`}),`
`,(0,n.jsx)(n.Fragment,{children:(0,n.jsx)(t.pre,{className:`shiki shiki-themes github-light github-dark`,style:{"--shiki-light":`#24292e`,"--shiki-dark":`#e1e4e8`,"--shiki-light-bg":`#fff`,"--shiki-dark-bg":`#24292e`},tabIndex:`0`,icon:`<svg viewBox="0 0 24 24"><path d="M 6,1 C 4.354992,1 3,2.354992 3,4 v 16 c 0,1.645008 1.354992,3 3,3 h 12 c 1.645008,0 3,-1.354992 3,-3 V 8 7 A 1.0001,1.0001 0 0 0 20.707031,6.2929687 l -5,-5 A 1.0001,1.0001 0 0 0 15,1 h -1 z m 0,2 h 7 v 3 c 0,1.645008 1.354992,3 3,3 h 3 v 11 c 0,0.564129 -0.435871,1 -1,1 H 6 C 5.4358712,21 5,20.564129 5,20 V 4 C 5,3.4358712 5.4358712,3 6,3 Z M 15,3.4140625 18.585937,7 H 16 C 15.435871,7 15,6.5641288 15,6 Z" fill="currentColor" /></svg>`,children:(0,n.jsxs)(t.code,{children:[(0,n.jsx)(t.span,{className:`line`,children:(0,n.jsx)(t.span,{style:{"--shiki-light":`#6F42C1`,"--shiki-dark":`#B392F0`},children:`[Unit]`})}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`Description`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`=fwdx server`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`After`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`=network.target nginx.service`})]}),`
`,(0,n.jsx)(t.span,{className:`line`}),`
`,(0,n.jsx)(t.span,{className:`line`,children:(0,n.jsx)(t.span,{style:{"--shiki-light":`#6F42C1`,"--shiki-dark":`#B392F0`},children:`[Service]`})}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`Type`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`=simple`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`ExecStart`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`=/usr/local/bin/fwdx serve \\`})]}),`
`,(0,n.jsx)(t.span,{className:`line`,children:(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`  --hostname tunnel.example.com \\`})}),`
`,(0,n.jsx)(t.span,{className:`line`,children:(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`  --oidc-issuer https://issuer.example.com \\`})}),`
`,(0,n.jsx)(t.span,{className:`line`,children:(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`  --oidc-client-id fwdx-web \\`})}),`
`,(0,n.jsx)(t.span,{className:`line`,children:(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`  --oidc-client-secret YOUR_OIDC_SECRET \\`})}),`
`,(0,n.jsx)(t.span,{className:`line`,children:(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`  --oidc-redirect-url https://tunnel.example.com/auth/oidc/callback \\`})}),`
`,(0,n.jsx)(t.span,{className:`line`,children:(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`  --oidc-admin-emails admin@example.com \\`})}),`
`,(0,n.jsx)(t.span,{className:`line`,children:(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`  --web-port 4040 \\`})}),`
`,(0,n.jsx)(t.span,{className:`line`,children:(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`  --grpc-port 4440 \\`})}),`
`,(0,n.jsx)(t.span,{className:`line`,children:(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`  --data-dir /var/lib/fwdx`})}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`Restart`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`=always`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`RestartSec`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`=5`})]}),`
`,(0,n.jsx)(t.span,{className:`line`}),`
`,(0,n.jsx)(t.span,{className:`line`,children:(0,n.jsx)(t.span,{style:{"--shiki-light":`#6F42C1`,"--shiki-dark":`#B392F0`},children:`[Install]`})}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`WantedBy`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`=multi-user.target`})]})]})})}),`
`,(0,n.jsx)(t.p,{children:`Logs:`}),`
`,(0,n.jsx)(n.Fragment,{children:(0,n.jsx)(t.pre,{className:`shiki shiki-themes github-light github-dark`,style:{"--shiki-light":`#24292e`,"--shiki-dark":`#e1e4e8`,"--shiki-light-bg":`#fff`,"--shiki-dark-bg":`#24292e`},tabIndex:`0`,icon:`<svg viewBox="0 0 24 24"><path d="m 4,4 a 1,1 0 0 0 -0.7070312,0.2929687 1,1 0 0 0 0,1.4140625 L 8.5859375,11 3.2929688,16.292969 a 1,1 0 0 0 0,1.414062 1,1 0 0 0 1.4140624,0 l 5.9999998,-6 a 1.0001,1.0001 0 0 0 0,-1.414062 L 4.7070312,4.2929687 A 1,1 0 0 0 4,4 Z m 8,14 a 1,1 0 0 0 -1,1 1,1 0 0 0 1,1 h 8 a 1,1 0 0 0 1,-1 1,1 0 0 0 -1,-1 z" fill="currentColor" /></svg>`,children:(0,n.jsx)(t.code,{children:(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#6F42C1`,"--shiki-dark":`#B392F0`},children:`journalctl`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` -u`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` fwdx`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` -f`})]})})})})]})}function c(e={}){let{wrapper:t}=e.components||{};return t?(0,n.jsx)(t,{...e,children:(0,n.jsx)(s,{...e})}):s(e)}export{i as _markdown,c as default,r as frontmatter,a as structuredData,o as toc};
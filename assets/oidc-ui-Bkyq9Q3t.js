import{r as e}from"./chunk-BA1M-is3.js";import{t}from"./jsx-runtime-Bc7xHnoj.js";var n=e(t()),r={title:`OIDC UI Login`,description:`Configure browser-based OIDC admin login.`},i=`

OIDC UI Login [#oidc-ui-login]

Required server settings:

\`\`\`bash
export FWDX_OIDC_ISSUER=https://issuer.example.com
export FWDX_OIDC_CLIENT_ID=fwdx-web
export FWDX_OIDC_CLIENT_SECRET=your-secret
export FWDX_OIDC_REDIRECT_URL=https://tunnel.example.com/auth/oidc/callback
export FWDX_OIDC_ADMIN_EMAILS=admin@example.com
\`\`\`

The redirect URL must point back to the \`fwdx\` server:

\`\`\`text
https://tunnel.example.com/auth/oidc/callback
\`\`\`

Once configured, visit:

\`\`\`text
https://tunnel.example.com/admin/ui
\`\`\`

The UI redirects to the OIDC provider and returns with a session cookie issued by \`fwdx\`.
`,a={contents:[{heading:`oidc-ui-login`,content:`Required server settings:`},{heading:`oidc-ui-login`,content:"The redirect URL must point back to the `fwdx` server:"},{heading:`oidc-ui-login`,content:`Once configured, visit:`},{heading:`oidc-ui-login`,content:"The UI redirects to the OIDC provider and returns with a session cookie issued by `fwdx`."}],headings:[{id:`oidc-ui-login`,content:`OIDC UI Login`}]},o=[{depth:1,url:`#oidc-ui-login`,title:(0,n.jsx)(n.Fragment,{children:`OIDC UI Login`})}];function s(e){let t={code:`code`,h1:`h1`,p:`p`,pre:`pre`,span:`span`,...e.components};return(0,n.jsxs)(n.Fragment,{children:[(0,n.jsx)(t.h1,{id:`oidc-ui-login`,children:`OIDC UI Login`}),`
`,(0,n.jsx)(t.p,{children:`Required server settings:`}),`
`,(0,n.jsx)(n.Fragment,{children:(0,n.jsx)(t.pre,{className:`shiki shiki-themes github-light github-dark`,style:{"--shiki-light":`#24292e`,"--shiki-dark":`#e1e4e8`,"--shiki-light-bg":`#fff`,"--shiki-dark-bg":`#24292e`},tabIndex:`0`,icon:`<svg viewBox="0 0 24 24"><path d="m 4,4 a 1,1 0 0 0 -0.7070312,0.2929687 1,1 0 0 0 0,1.4140625 L 8.5859375,11 3.2929688,16.292969 a 1,1 0 0 0 0,1.414062 1,1 0 0 0 1.4140624,0 l 5.9999998,-6 a 1.0001,1.0001 0 0 0 0,-1.414062 L 4.7070312,4.2929687 A 1,1 0 0 0 4,4 Z m 8,14 a 1,1 0 0 0 -1,1 1,1 0 0 0 1,1 h 8 a 1,1 0 0 0 1,-1 1,1 0 0 0 -1,-1 z" fill="currentColor" /></svg>`,children:(0,n.jsxs)(t.code,{children:[(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`export`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:` FWDX_OIDC_ISSUER`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`=`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`https://issuer.example.com`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`export`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:` FWDX_OIDC_CLIENT_ID`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`=`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`fwdx-web`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`export`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:` FWDX_OIDC_CLIENT_SECRET`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`=`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`your-secret`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`export`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:` FWDX_OIDC_REDIRECT_URL`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`=`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`https://tunnel.example.com/auth/oidc/callback`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`export`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:` FWDX_OIDC_ADMIN_EMAILS`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`=`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`admin@example.com`})]})]})})}),`
`,(0,n.jsxs)(t.p,{children:[`The redirect URL must point back to the `,(0,n.jsx)(t.code,{children:`fwdx`}),` server:`]}),`
`,(0,n.jsx)(n.Fragment,{children:(0,n.jsx)(t.pre,{className:`shiki shiki-themes github-light github-dark`,style:{"--shiki-light":`#24292e`,"--shiki-dark":`#e1e4e8`,"--shiki-light-bg":`#fff`,"--shiki-dark-bg":`#24292e`},tabIndex:`0`,icon:`<svg viewBox="0 0 24 24"><path d="M 6,1 C 4.354992,1 3,2.354992 3,4 v 16 c 0,1.645008 1.354992,3 3,3 h 12 c 1.645008,0 3,-1.354992 3,-3 V 8 7 A 1.0001,1.0001 0 0 0 20.707031,6.2929687 l -5,-5 A 1.0001,1.0001 0 0 0 15,1 h -1 z m 0,2 h 7 v 3 c 0,1.645008 1.354992,3 3,3 h 3 v 11 c 0,0.564129 -0.435871,1 -1,1 H 6 C 5.4358712,21 5,20.564129 5,20 V 4 C 5,3.4358712 5.4358712,3 6,3 Z M 15,3.4140625 18.585937,7 H 16 C 15.435871,7 15,6.5641288 15,6 Z" fill="currentColor" /></svg>`,children:(0,n.jsx)(t.code,{children:(0,n.jsx)(t.span,{className:`line`,children:(0,n.jsx)(t.span,{children:`https://tunnel.example.com/auth/oidc/callback`})})})})}),`
`,(0,n.jsx)(t.p,{children:`Once configured, visit:`}),`
`,(0,n.jsx)(n.Fragment,{children:(0,n.jsx)(t.pre,{className:`shiki shiki-themes github-light github-dark`,style:{"--shiki-light":`#24292e`,"--shiki-dark":`#e1e4e8`,"--shiki-light-bg":`#fff`,"--shiki-dark-bg":`#24292e`},tabIndex:`0`,icon:`<svg viewBox="0 0 24 24"><path d="M 6,1 C 4.354992,1 3,2.354992 3,4 v 16 c 0,1.645008 1.354992,3 3,3 h 12 c 1.645008,0 3,-1.354992 3,-3 V 8 7 A 1.0001,1.0001 0 0 0 20.707031,6.2929687 l -5,-5 A 1.0001,1.0001 0 0 0 15,1 h -1 z m 0,2 h 7 v 3 c 0,1.645008 1.354992,3 3,3 h 3 v 11 c 0,0.564129 -0.435871,1 -1,1 H 6 C 5.4358712,21 5,20.564129 5,20 V 4 C 5,3.4358712 5.4358712,3 6,3 Z M 15,3.4140625 18.585937,7 H 16 C 15.435871,7 15,6.5641288 15,6 Z" fill="currentColor" /></svg>`,children:(0,n.jsx)(t.code,{children:(0,n.jsx)(t.span,{className:`line`,children:(0,n.jsx)(t.span,{children:`https://tunnel.example.com/admin/ui`})})})})}),`
`,(0,n.jsxs)(t.p,{children:[`The UI redirects to the OIDC provider and returns with a session cookie issued by `,(0,n.jsx)(t.code,{children:`fwdx`}),`.`]})]})}function c(e={}){let{wrapper:t}=e.components||{};return t?(0,n.jsx)(t,{...e,children:(0,n.jsx)(s,{...e})}):s(e)}export{i as _markdown,c as default,r as frontmatter,a as structuredData,o as toc};
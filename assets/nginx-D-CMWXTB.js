import{r as e}from"./chunk-BA1M-is3.js";import{t}from"./jsx-runtime-Bc7xHnoj.js";var n=e(t()),r={title:`nginx`,description:`Reverse proxy public HTTPS and tunnel gRPC traffic to the fwdx server.`},i=`

nginx [#nginx]

The common layout is:

* \`443\` -> fwdx web port
* \`4443\` -> fwdx gRPC port

If you use ingress IP allowlists, start the server with trusted proxy CIDRs so \`fwdx\` uses the real visitor IP instead of nginx's loopback address:

\`\`\`bash
fwdx serve \\
  --trusted-proxy-cidrs 127.0.0.1/32,::1/128 \\
  ...
\`\`\`

HTTPS vhost [#https-vhost]

\`\`\`nginx
server {
  listen 443 ssl http2;
  listen [::]:443 ssl http2;
  server_name tunnel.example.com *.tunnel.example.com;

  ssl_certificate     /etc/letsencrypt/live/tunnel.example.com/fullchain.pem;
  ssl_certificate_key /etc/letsencrypt/live/tunnel.example.com/privkey.pem;

  location / {
    proxy_pass http://127.0.0.1:4040;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_read_timeout 86400s;
    proxy_send_timeout 86400s;
  }
}
\`\`\`

gRPC stream block [#grpc-stream-block]

\`\`\`nginx
stream {
  server {
    listen 4443 ssl;
    listen [::]:4443 ssl;

    ssl_certificate     /etc/letsencrypt/live/tunnel.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/tunnel.example.com/privkey.pem;
    ssl_alpn h2;

    proxy_pass 127.0.0.1:4440;
    proxy_connect_timeout 10s;
    proxy_timeout 86400s;
  }
}
\`\`\`
`,a={contents:[{heading:`nginx`,content:`The common layout is:`},{heading:`nginx`,content:"`443` -> fwdx web port"},{heading:`nginx`,content:"`4443` -> fwdx gRPC port"},{heading:`nginx`,content:"If you use ingress IP allowlists, start the server with trusted proxy CIDRs so `fwdx` uses the real visitor IP instead of nginx's loopback address:"}],headings:[{id:`nginx`,content:`nginx`},{id:`https-vhost`,content:`HTTPS vhost`},{id:`grpc-stream-block`,content:`gRPC stream block`}]},o=[{depth:1,url:`#nginx`,title:(0,n.jsx)(n.Fragment,{children:`nginx`})},{depth:2,url:`#https-vhost`,title:(0,n.jsx)(n.Fragment,{children:`HTTPS vhost`})},{depth:2,url:`#grpc-stream-block`,title:(0,n.jsx)(n.Fragment,{children:`gRPC stream block`})}];function s(e){let t={code:`code`,h1:`h1`,h2:`h2`,li:`li`,p:`p`,pre:`pre`,span:`span`,ul:`ul`,...e.components};return(0,n.jsxs)(n.Fragment,{children:[(0,n.jsx)(t.h1,{id:`nginx`,children:`nginx`}),`
`,(0,n.jsx)(t.p,{children:`The common layout is:`}),`
`,(0,n.jsxs)(t.ul,{children:[`
`,(0,n.jsxs)(t.li,{children:[(0,n.jsx)(t.code,{children:`443`}),` -> fwdx web port`]}),`
`,(0,n.jsxs)(t.li,{children:[(0,n.jsx)(t.code,{children:`4443`}),` -> fwdx gRPC port`]}),`
`]}),`
`,(0,n.jsxs)(t.p,{children:[`If you use ingress IP allowlists, start the server with trusted proxy CIDRs so `,(0,n.jsx)(t.code,{children:`fwdx`}),` uses the real visitor IP instead of nginx's loopback address:`]}),`
`,(0,n.jsx)(n.Fragment,{children:(0,n.jsx)(t.pre,{className:`shiki shiki-themes github-light github-dark`,style:{"--shiki-light":`#24292e`,"--shiki-dark":`#e1e4e8`,"--shiki-light-bg":`#fff`,"--shiki-dark-bg":`#24292e`},tabIndex:`0`,icon:`<svg viewBox="0 0 24 24"><path d="m 4,4 a 1,1 0 0 0 -0.7070312,0.2929687 1,1 0 0 0 0,1.4140625 L 8.5859375,11 3.2929688,16.292969 a 1,1 0 0 0 0,1.414062 1,1 0 0 0 1.4140624,0 l 5.9999998,-6 a 1.0001,1.0001 0 0 0 0,-1.414062 L 4.7070312,4.2929687 A 1,1 0 0 0 4,4 Z m 8,14 a 1,1 0 0 0 -1,1 1,1 0 0 0 1,1 h 8 a 1,1 0 0 0 1,-1 1,1 0 0 0 -1,-1 z" fill="currentColor" /></svg>`,children:(0,n.jsxs)(t.code,{children:[(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#6F42C1`,"--shiki-dark":`#B392F0`},children:`fwdx`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` serve`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` \\`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:`  --trusted-proxy-cidrs`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:` 127.0.0.1/32,::1/128`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:` \\`})]}),`
`,(0,n.jsx)(t.span,{className:`line`,children:(0,n.jsx)(t.span,{style:{"--shiki-light":`#032F62`,"--shiki-dark":`#9ECBFF`},children:`  ...`})})]})})}),`
`,(0,n.jsx)(t.h2,{id:`https-vhost`,children:`HTTPS vhost`}),`
`,(0,n.jsx)(n.Fragment,{children:(0,n.jsx)(t.pre,{className:`shiki shiki-themes github-light github-dark`,style:{"--shiki-light":`#24292e`,"--shiki-dark":`#e1e4e8`,"--shiki-light-bg":`#fff`,"--shiki-dark-bg":`#24292e`},tabIndex:`0`,icon:`<svg viewBox="0 0 24 24"><path d="M 6,1 C 4.354992,1 3,2.354992 3,4 v 16 c 0,1.645008 1.354992,3 3,3 h 12 c 1.645008,0 3,-1.354992 3,-3 V 8 7 A 1.0001,1.0001 0 0 0 20.707031,6.2929687 l -5,-5 A 1.0001,1.0001 0 0 0 15,1 h -1 z m 0,2 h 7 v 3 c 0,1.645008 1.354992,3 3,3 h 3 v 11 c 0,0.564129 -0.435871,1 -1,1 H 6 C 5.4358712,21 5,20.564129 5,20 V 4 C 5,3.4358712 5.4358712,3 6,3 Z M 15,3.4140625 18.585937,7 H 16 C 15.435871,7 15,6.5641288 15,6 Z" fill="currentColor" /></svg>`,children:(0,n.jsxs)(t.code,{children:[(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`server`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:` {`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`  listen `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:`443`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:` ssl http2;`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`  listen `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`[::]:443 ssl http2;`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`  server_name `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`tunnel.example.com *.tunnel.example.com;`})]}),`
`,(0,n.jsx)(t.span,{className:`line`}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`  ssl_certificate `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`    /etc/letsencrypt/live/tunnel.example.com/fullchain.pem;`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`  ssl_certificate_key `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`/etc/letsencrypt/live/tunnel.example.com/privkey.pem;`})]}),`
`,(0,n.jsx)(t.span,{className:`line`}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`  location`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#6F42C1`,"--shiki-dark":`#B392F0`},children:` / `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`{`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`    proxy_pass `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`http://127.0.0.1:4040;`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`    proxy_http_version `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:`1.1`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`;`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`    proxy_set_header `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`Host $host;`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`    proxy_set_header `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`X-Real-IP $remote_addr;`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`    proxy_set_header `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`X-Forwarded-For $proxy_add_x_forwarded_for;`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`    proxy_set_header `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`X-Forwarded-Proto $scheme;`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`    proxy_read_timeout `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:`86400s`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`;`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`    proxy_send_timeout `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:`86400s`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`;`})]}),`
`,(0,n.jsx)(t.span,{className:`line`,children:(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`  }`})}),`
`,(0,n.jsx)(t.span,{className:`line`,children:(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`}`})})]})})}),`
`,(0,n.jsx)(t.h2,{id:`grpc-stream-block`,children:`gRPC stream block`}),`
`,(0,n.jsx)(n.Fragment,{children:(0,n.jsx)(t.pre,{className:`shiki shiki-themes github-light github-dark`,style:{"--shiki-light":`#24292e`,"--shiki-dark":`#e1e4e8`,"--shiki-light-bg":`#fff`,"--shiki-dark-bg":`#24292e`},tabIndex:`0`,icon:`<svg viewBox="0 0 24 24"><path d="M 6,1 C 4.354992,1 3,2.354992 3,4 v 16 c 0,1.645008 1.354992,3 3,3 h 12 c 1.645008,0 3,-1.354992 3,-3 V 8 7 A 1.0001,1.0001 0 0 0 20.707031,6.2929687 l -5,-5 A 1.0001,1.0001 0 0 0 15,1 h -1 z m 0,2 h 7 v 3 c 0,1.645008 1.354992,3 3,3 h 3 v 11 c 0,0.564129 -0.435871,1 -1,1 H 6 C 5.4358712,21 5,20.564129 5,20 V 4 C 5,3.4358712 5.4358712,3 6,3 Z M 15,3.4140625 18.585937,7 H 16 C 15.435871,7 15,6.5641288 15,6 Z" fill="currentColor" /></svg>`,children:(0,n.jsxs)(t.code,{children:[(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`stream`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:` {`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`  server`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:` {`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`    listen `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:`4443`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:` ssl;`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`    listen `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`[::]:4443 ssl;`})]}),`
`,(0,n.jsx)(t.span,{className:`line`}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`    ssl_certificate `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`    /etc/letsencrypt/live/tunnel.example.com/fullchain.pem;`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`    ssl_certificate_key `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`/etc/letsencrypt/live/tunnel.example.com/privkey.pem;`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`    ssl_alpn `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`h2;`})]}),`
`,(0,n.jsx)(t.span,{className:`line`}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`    proxy_pass `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`127.0.0.1:4440;`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`    proxy_connect_timeout `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:`10s`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`;`})]}),`
`,(0,n.jsxs)(t.span,{className:`line`,children:[(0,n.jsx)(t.span,{style:{"--shiki-light":`#D73A49`,"--shiki-dark":`#F97583`},children:`    proxy_timeout `}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#005CC5`,"--shiki-dark":`#79B8FF`},children:`86400s`}),(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`;`})]}),`
`,(0,n.jsx)(t.span,{className:`line`,children:(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`  }`})}),`
`,(0,n.jsx)(t.span,{className:`line`,children:(0,n.jsx)(t.span,{style:{"--shiki-light":`#24292E`,"--shiki-dark":`#E1E4E8`},children:`}`})})]})})})]})}function c(e={}){let{wrapper:t}=e.components||{};return t?(0,n.jsx)(t,{...e,children:(0,n.jsx)(s,{...e})}):s(e)}export{i as _markdown,c as default,r as frontmatter,a as structuredData,o as toc};
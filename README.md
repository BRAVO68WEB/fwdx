# fwdx

Self-hosted HTTP tunneling over gRPC with your own domain, OIDC-authenticated admin access, and a local tunnel client.

## Build

### Prerequisites
- Go 1.25+

### From source

```bash
git clone https://github.com/BRAVO68WEB/fwdx.git
cd fwdx
go build -o fwdx .
```

## Current auth model

- Human auth is OIDC only.
- Browser admin UI uses OIDC login.
- CLI human access uses `fwdx login` with device flow.
- Tunnel runtime uses per-agent credentials issued by the server control plane.

## Quick start

### 1. Run the server

```bash
export FWDX_HOSTNAME=tunnel.myweb.site
export FWDX_OIDC_ISSUER=https://issuer.example.com
export FWDX_OIDC_CLIENT_ID=fwdx-web
export FWDX_OIDC_CLIENT_SECRET=your-client-secret
export FWDX_OIDC_REDIRECT_URL=https://tunnel.myweb.site/auth/oidc/callback
export FWDX_OIDC_ADMIN_EMAILS=you@example.com

fwdx serve \
  --hostname "$FWDX_HOSTNAME" \
  --oidc-issuer "$FWDX_OIDC_ISSUER" \
  --oidc-client-id "$FWDX_OIDC_CLIENT_ID" \
  --oidc-client-secret "$FWDX_OIDC_CLIENT_SECRET" \
  --oidc-redirect-url "$FWDX_OIDC_REDIRECT_URL" \
  --oidc-admin-emails "$FWDX_OIDC_ADMIN_EMAILS" \
  --web-port 4040 \
  --grpc-port 4440 \
  --data-dir /var/lib/fwdx
```

Create DNS:
- `tunnel.myweb.site -> server IP`
- `*.tunnel.myweb.site -> server IP`

### 2. Log in for human admin actions

```bash
export FWDX_SERVER=https://tunnel.myweb.site
fwdx login
fwdx whoami
```

The admin UI is served at `/admin/ui` and redirects through OIDC.
Tunnel detail pages under `/admin/ui/tunnels/:name` expose assignment, recent events/logs, and ingress access controls.

### 3. Add domains if needed

```bash
fwdx domains add my.domain --server https://tunnel.myweb.site
```

### 4. Start a tunnel client

```bash
export FWDX_SERVER=https://tunnel.myweb.site

fwdx tunnel create -l localhost:8080 -s myapp --name myapp
fwdx tunnel start myapp
```

## CLI

### Human auth

```bash
fwdx login
fwdx whoami
fwdx logout
```

### Admin / management

```bash
fwdx manage tunnels --server https://tunnel.myweb.site
fwdx manage domains list --server https://tunnel.myweb.site
fwdx domains add my.domain --server https://tunnel.myweb.site
```

These commands use the OIDC session created by `fwdx login`.

### Tunnel runtime

```bash
fwdx tunnel create -l localhost:3000 -s app --name app
fwdx tunnel start app --detach
fwdx logs app --follow
fwdx tunnel stop app
```

### Ingress access controls

Each tunnel supports:
- `public`
- `basic_auth`
- `shared_secret_header`
- optional `ip_allowlist`

Access rules are managed from the admin UI tunnel detail page.

## Config summary

### Server
- `FWDX_HOSTNAME`
- `FWDX_OIDC_ISSUER`
- `FWDX_OIDC_CLIENT_ID`
- `FWDX_OIDC_CLIENT_SECRET`
- `FWDX_OIDC_REDIRECT_URL`
- `FWDX_OIDC_SCOPES`
- `FWDX_OIDC_ADMIN_EMAILS`
- `FWDX_OIDC_ADMIN_SUBJECTS`
- `FWDX_OIDC_ADMIN_GROUPS`
- `FWDX_OIDC_SESSION_SECRET`
- `FWDX_OIDC_DEVICE_CLIENT_ID`
- `FWDX_TRUSTED_PROXY_CIDRS`

### Client
- `FWDX_SERVER`
- `FWDX_AGENT_NAME`
- `FWDX_AGENT_TOKEN`
- `FWDX_TUNNEL_PORT`
- `FWDX_MAX_PROXY_BODY_BYTES`
- `FWDX_MAX_RESPONSE_BODY_BYTES`

## Protocol scope

- HTTP forwarding: supported
- SSE: supported
- WebSocket: not implemented yet (`501 Not Implemented`)

## Docs

The Fumadocs site under `docs/` contains:
- Installation
- CLI Usage
- Deployment
- Architecture
- Authentication

## License

[MIT](./LICENSE)

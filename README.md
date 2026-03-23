# fwdx

Self-hosted tunneling CLI and server for exposing local HTTP services by hostname over gRPC (like ngrok or cloudflared, but your own server).

## Installation

### Prerequisites
- Go 1.21+

### From Source

```bash
git clone https://github.com/BRAVO68WEB/fwdx.git
cd fwdx
go build -o fwdx .
```

### Docker

```bash
docker build -t fwdx .
```

## Self-hosting (detailed guide)

For a **step-by-step guide** to run the server on your own VPS (DNS, TLS with Let's Encrypt, systemd or Docker, firewall, first tunnel), see **[docs/SELFHOSTING.md](docs/SELFHOSTING.md)**.

## Quick Start

### 1. Run the server

On a machine with a public IP and a hostname (e.g. `tunnel.myweb.site`):

```bash
# TLS cert and key required (e.g. from Let's Encrypt or self-signed for dev)
export FWDX_HOSTNAME=tunnel.myweb.site
export FWDX_CLIENT_TOKEN=your-client-token
export FWDX_ADMIN_TOKEN=your-admin-token
fwdx serve --hostname $FWDX_HOSTNAME --client-token $FWDX_CLIENT_TOKEN --admin-token $FWDX_ADMIN_TOKEN --tls-cert /path/to/cert.pem --tls-key /path/to/key.pem
```

First-time DNS: create an **A** record: `tunnel.myweb.site` -> your server IP.
Also create wildcard DNS for subdomains: `*.tunnel.myweb.site` -> your server IP.

### 2. Add allowed domains (optional)

To use a custom domain (e.g. `app.my.domain`) instead of a subdomain under the server hostname:

```bash
fwdx domains add my.domain --server https://tunnel.myweb.site --admin-token $FWDX_ADMIN_TOKEN
```

Then create a **CNAME** record: `*.my.domain` -> `tunnel.myweb.site`.

### 3. Create and start a tunnel (client)

On your laptop or dev machine:

```bash
export FWDX_SERVER=https://tunnel.myweb.site
export FWDX_TOKEN=your-client-token

# Subdomain under server hostname (e.g. myapp.tunnel.myweb.site)
fwdx tunnel create -l localhost:8080 -s myapp --name myapp
fwdx tunnel start myapp

# Or custom domain (if added via domains add)
fwdx tunnel create -l localhost:8080 -u app.my.domain --name myapp
fwdx tunnel start myapp
```

Access at `https://myapp.tunnel.myweb.site` or `https://app.my.domain`.

## Commands

### Server
- `fwdx serve` - Start the tunneling server (see flags: --hostname, --client-token, --admin-token, --tls-cert, --tls-key)

### Management (remote)
- `fwdx manage tunnels` - List active tunnels (--server, --admin-token)
- `fwdx manage domains list` - List allowed domains
- `fwdx domains add <domain>` - Add domain to allowed list and print DNS instructions

### Client
- `fwdx tunnel create` - Create a tunnel (--local, --subdomain or --url, --name)
- `fwdx tunnel start <name>` - Start tunnel in foreground (use `--detach` for background)
- `fwdx tunnel stop <name>` - Stop a tunnel
- `fwdx logs <name>` - Show detached tunnel logs (`--follow`)
- `fwdx tunnel list` - List tunnels
- `fwdx tunnel delete <name>` - Delete a tunnel
- `fwdx config` - Show client config (FWDX_SERVER, FWDX_TOKEN)
- `fwdx health` - Check client config and server reachability

## Configuration

**Client:** Set `FWDX_SERVER` and `FWDX_TOKEN` (or create `~/.fwdx/client.json` with `server_url` and `token`). Optional: `server_hostname`, `tunnel_port` (default 4443), `FWDX_MAX_PROXY_BODY_BYTES`, `FWDX_MAX_RESPONSE_BODY_BYTES`.

**Server:** Set `FWDX_HOSTNAME`, `FWDX_CLIENT_TOKEN`, `FWDX_ADMIN_TOKEN` or pass via flags. TLS cert and key are required in direct mode. Admin token is for admin APIs only; tunnel clients authenticate with client token.

Current protocol scope: HTTP forwarding and SSE are supported; WebSocket upgrade is currently returned as `501 Not Implemented`.

## CI & Releases

- **CI** (`.github/workflows/ci.yml`): On push to `main` and on pull requests — build, unit/in-process tests (with `FWDX_SKIP_DOCKER_E2E=1`), and a separate job for Docker e2e tests.
- **Release** (`.github/workflows/release.yml`): On tag push (e.g. `v1.0.0`) — build binaries for **linux** and **darwin** (amd64 + arm64), upload them as workflow artifacts and to the GitHub Release; build and push a multi-arch Docker image (linux/amd64, linux/arm64) to `ghcr.io/<owner>/fwdx`.

## License

[MIT](./LICENSE)

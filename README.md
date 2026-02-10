# fwdx

Self-hosted tunneling CLI and server for exposing local HTTP services by hostname (like ngrok or cloudflared, but your own server).

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

On a machine with a public IP and a hostname (e.g. `tunnel.example.com`):

```bash
# TLS cert and key required (e.g. from Let's Encrypt or self-signed for dev)
export FWDX_HOSTNAME=tunnel.example.com
export FWDX_CLIENT_TOKEN=your-client-token
export FWDX_ADMIN_TOKEN=your-admin-token
fwdx serve --hostname $FWDX_HOSTNAME --client-token $FWDX_CLIENT_TOKEN --admin-token $FWDX_ADMIN_TOKEN --tls-cert /path/to/cert.pem --tls-key /path/to/key.pem
```

First-time DNS: create an **A** record: `tunnel.example.com` -> your server IP.

### 2. Add allowed domains (optional)

To use a custom domain (e.g. `app.my.domain`) instead of a subdomain under the server hostname:

```bash
fwdx domains add my.domain --server https://tunnel.example.com --admin-token $FWDX_ADMIN_TOKEN
```

Then create a **CNAME** record: `*.my.domain` -> `tunnel.example.com`.

### 3. Create and start a tunnel (client)

On your laptop or dev machine:

```bash
export FWDX_SERVER=https://tunnel.example.com
export FWDX_TOKEN=your-client-token

# Subdomain under server hostname (e.g. myapp.tunnel.example.com)
fwdx tunnel create -l localhost:8080 -s myapp --name myapp
fwdx tunnel start myapp

# Or custom domain (if added via domains add)
fwdx tunnel create -l localhost:8080 -u app.my.domain --name myapp
fwdx tunnel start myapp
```

Access at `https://myapp.tunnel.example.com` or `https://app.my.domain`.

## Commands

### Server
- `fwdx serve` - Start the tunneling server (see flags: --hostname, --client-token, --admin-token, --tls-cert, --tls-key)

### Management (remote)
- `fwdx manage tunnels` - List active tunnels (--server, --admin-token)
- `fwdx manage domains list` - List allowed domains
- `fwdx domains add <domain>` - Add domain to allowed list and print DNS instructions

### Client
- `fwdx tunnel create` - Create a tunnel (--local, --subdomain or --url, --name)
- `fwdx tunnel start <name>` - Start a tunnel (--watch, --debug for foreground)
- `fwdx tunnel stop <name>` - Stop a tunnel
- `fwdx tunnel list` - List tunnels
- `fwdx tunnel delete <name>` - Delete a tunnel
- `fwdx config` - Show client config (FWDX_SERVER, FWDX_TOKEN)
- `fwdx health` - Check client config and server reachability

## Configuration

**Client:** Set `FWDX_SERVER` and `FWDX_TOKEN` (or create `~/.fwdx/client.json` with `server_url` and `token`). Optional: `server_hostname`, `tunnel_port` (default 4443).

**Server:** Set `FWDX_HOSTNAME`, `FWDX_CLIENT_TOKEN`, `FWDX_ADMIN_TOKEN` or pass via flags. TLS cert and key are required.

## CI & Releases

- **CI** (`.github/workflows/ci.yml`): On push to `main` and on pull requests — build, unit/in-process tests (with `FWDX_SKIP_DOCKER_E2E=1`), and a separate job for Docker e2e tests.
- **Release** (`.github/workflows/release.yml`): On tag push (e.g. `v1.0.0`) — build binaries for **linux** and **darwin** (amd64 + arm64), upload them as workflow artifacts and to the GitHub Release; build and push a multi-arch Docker image (linux/amd64, linux/arm64) to `ghcr.io/<owner>/fwdx`.

## License

[MIT](./LICENSE)
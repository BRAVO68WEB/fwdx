# Docker E2E

End-to-end test that runs the full fwdx flow in Docker: server, app, and client containers, then asserts the public URL returns the app response.

## Prerequisites

- Docker
- docker compose (v2)

## Layout

- **Dockerfile.fwdx** – Builds the fwdx binary (build context: repo root). Used for server and client.
- **Dockerfile.app** – Minimal Go HTTP server that returns `hello from docker app`.
- **docker-compose.yml** – Defines `server`, `app`, and `client` services.
- **scripts/entrypoint-server.sh** – Generates a self-signed cert and runs `fwdx serve` (hostname `server`).
- **scripts/entrypoint-client.sh** – Waits for server, runs `fwdx tunnel create` and `fwdx tunnel start myapp`.
- **scripts/run-e2e.sh** – Builds images, brings up compose, waits for tunnel, curls `https://localhost:8443/` with `Host: myapp.server`, asserts body contains `hello from docker app`, then tears down.

## Run

From this directory (`tests/docker`):

```bash
sh scripts/run-e2e.sh
```

Or from the repo root via Go test:

```bash
go test ./tests/... -run TestE2E_Docker -v
```

Skip the Docker e2e in CI or when Docker is unavailable:

```bash
FWDX_SKIP_DOCKER_E2E=1 go test ./...
```

## Flow

1. **server** – TLS on 443 and 4443, hostname `server`, client token `e2e-docker-token`.
2. **app** – HTTP server on 8080, returns `hello from docker app`.
3. **client** – After server is healthy, runs `fwdx tunnel create -l app:8080 -s myapp` and `fwdx tunnel start myapp` (in background). Registers hostname `myapp.server`.
4. **run-e2e.sh** – Curls `https://localhost:8443/` with `Host: myapp.server` (port 8443 is mapped from server 443). Expects response body to contain `hello from docker app`.

## TLS

The server uses a self-signed certificate (CN=server, SAN=DNS:server). The client container sets `FWDX_INSECURE_SKIP_VERIFY=1` so the tunnel can connect without installing the CA; use this only for testing.

# Docker E2E

End-to-end coverage for the current `fwdx` architecture:
- mock OIDC provider
- OIDC device-flow CLI login
- server-owned tunnel creation
- per-agent tunnel runtime auth
- admin UI session login through PKCE redirects
- ingress access-rule enforcement

## Services

- `mock-oidc`: deterministic OIDC provider for auth-code + device flow
- `server`: `fwdx serve` with OIDC enabled on `http://server`
- `app`: sample HTTP backend with `/`, `/protected`, and `/sse`
- `client`: logs in, provisions an agent, creates `myapp.server`, and starts the tunnel
- `tester`: runs curl-based assertions against the control plane and ingress paths

## Flow

1. `client` runs `fwdx login` against the mock OIDC provider through the `server`.
2. `client` auto-provisions `FWDX_AGENT_NAME` and creates `myapp.server`.
3. `client` runs `fwdx tunnel start myapp --watch`.
4. `tester` logs into the admin UI by following `/auth/oidc/login?redirect=/admin/ui`.
5. `tester` updates access rules through `/api/tunnels/myapp/access` and verifies:
   - `public`
   - `basic_auth`
   - `shared_secret_header`
   - `ip_allowlist`

## Run

```bash
sh scripts/run-e2e.sh
```

Or:

```bash
go test ./tests/... -run TestE2E_Docker -v
```

Skip Docker e2e when needed:

```bash
FWDX_SKIP_DOCKER_E2E=1 go test ./...
```

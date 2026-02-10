# Tests

## Unit tests

Unit tests live next to the code they test (e.g. `internal/server/registry_test.go`).

Run all unit tests:

```bash
go test ./internal/...
```

Run with verbose output:

```bash
go test ./internal/... -v
```

## E2E integration tests (in-process)

E2E tests in `tests/` exercise the full tunnel flow: server (tunnel + proxy handlers), client connector, and a fake local backend. They do not require a real TLS server or network.

Run e2e tests:

```bash
go test ./tests/... -v
```

## E2E Docker tests

A separate Docker-based e2e runs the full flow in containers: fwdx server, a minimal HTTP app, and fwdx client, orchestrated by `tests/docker/scripts/run-e2e.sh`. Requires Docker and docker compose.

- **From Go:** `go test ./tests/... -run TestE2E_Docker -v` (skip with `FWDX_SKIP_DOCKER_E2E=1`)
- **Standalone:** `cd tests/docker && sh scripts/run-e2e.sh`

See [tests/docker/README.md](docker/README.md) for details.

## Run everything (excluding Docker e2e)

```bash
go test ./...
```

To include Docker e2e, run the test suite without `FWDX_SKIP_DOCKER_E2E=1`.

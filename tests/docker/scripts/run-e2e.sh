#!/bin/sh
# Docker-based e2e: start server, app, client; curl public URL; assert response; teardown.
set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
COMPOSE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$COMPOSE_DIR"

# Build (compose uses context ../.. for fwdx images)
docker compose build

# Start in background
docker compose up -d

# Wait for client to register and tunnel to be ready (client script sleeps 5 after start)
echo "Waiting for tunnel to be ready..."
max=30
for i in $(seq 1 $max); do
  if curl -sk -o /dev/null -w "%{http_code}" -H "Host: myapp.server" https://localhost:8443/ 2>/dev/null | grep -q 200; then
    echo "Tunnel ready after ${i}s"
    break
  fi
  if [ "$i" -eq "$max" ]; then
    echo "Timeout waiting for tunnel"
    docker compose logs
    docker compose down
    exit 1
  fi
  sleep 1
done

# Assert response body
RESP=$(curl -sk -H "Host: myapp.server" https://localhost:8443/)
if echo "$RESP" | grep -q "hello from docker app"; then
  echo "E2E OK: got expected response"
else
  echo "E2E FAIL: unexpected response: $RESP"
  docker compose logs
  docker compose down
  exit 1
fi

docker compose down
echo "Docker e2e passed."

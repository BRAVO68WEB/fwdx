#!/bin/sh
set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
COMPOSE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$COMPOSE_DIR"
export COMPOSE_PROJECT_NAME="fwdx-e2e-$$"

cleanup() {
  docker compose down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

cleanup

docker compose build

docker compose up -d

run_tester() {
  docker compose exec -T tester sh -lc "$1"
}

run_client() {
  docker compose exec -T client sh -lc "$1"
}

echo "Waiting for tunnel to be ready..."
max=60
for i in $(seq 1 $max); do
  if run_tester "curl -fsS -H 'Host: myapp.server' http://server/ | grep -q 'hello from docker app'" >/dev/null 2>&1; then
    echo "Tunnel ready after ${i}s"
    break
  fi
  if [ "$i" -eq "$max" ]; then
    echo "Timeout waiting for tunnel"
    docker compose logs
    exit 1
  fi
  sleep 1
done

echo "Checking CLI device auth state..."
run_client "fwdx whoami | grep -q 'admin@example.com'"

echo "Checking browser OIDC login and admin UI..."
run_tester "rm -f /tmp/fwdx.jar /tmp/fwdx.html && curl -fsSL -c /tmp/fwdx.jar -b /tmp/fwdx.jar 'http://server/auth/oidc/login?redirect=/admin/ui' > /tmp/fwdx.html && grep -q 'Active Tunnels' /tmp/fwdx.html"
run_tester "curl -fsS -b /tmp/fwdx.jar http://server/admin/ui/tunnels/myapp | grep -q 'Access Rules'"

echo "Checking public tunnel response..."
run_tester "curl -fsS -H 'Host: myapp.server' http://server/ | grep -q 'hello from docker app'"

echo "Checking basic auth ingress rule..."
run_tester "curl -fsS -X PATCH -b /tmp/fwdx.jar -H 'Content-Type: application/json' http://server/api/tunnels/myapp/access -d '{\"auth_mode\":\"basic_auth\",\"basic_auth_username\":\"demo\",\"basic_auth_password\":\"secret\"}' >/dev/null"
[ "$(run_tester "curl -s -o /dev/null -w '%{http_code}' -H 'Host: myapp.server' http://server/protected")" = "401" ]
run_tester "curl -fsS -u demo:secret -H 'Host: myapp.server' http://server/protected | grep -q 'protected ok'"

echo "Checking shared secret ingress rule..."
run_tester "curl -fsS -X PATCH -b /tmp/fwdx.jar -H 'Content-Type: application/json' http://server/api/tunnels/myapp/access -d '{\"auth_mode\":\"shared_secret_header\",\"shared_secret_header_name\":\"X-Test-Secret\",\"shared_secret_value\":\"secret-header\"}' >/dev/null"
[ "$(run_tester "curl -s -o /dev/null -w '%{http_code}' -H 'Host: myapp.server' http://server/protected")" = "401" ]
run_tester "curl -fsS -H 'Host: myapp.server' -H 'X-Test-Secret: secret-header' http://server/protected | grep -q 'protected ok'"

echo "Checking IP allowlist ingress rule..."
run_tester "curl -fsS -X PATCH -b /tmp/fwdx.jar -H 'Content-Type: application/json' http://server/api/tunnels/myapp/access -d '{\"auth_mode\":\"public\",\"allowed_ips\":[\"203.0.113.0/24\"]}' >/dev/null"
run_tester "curl -fsS -H 'Host: myapp.server' -H 'X-Forwarded-For: 203.0.113.10' http://server/protected | grep -q 'protected ok'"
[ "$(run_client "curl -s -o /dev/null -w '%{http_code}' -H 'Host: myapp.server' -H 'X-Forwarded-For: 203.0.113.10' http://server/protected")" = "403" ]

echo "Docker e2e passed."

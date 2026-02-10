#!/bin/sh
set -e
# Wait for server to be up
until wget -q -O /dev/null --no-check-certificate https://server:4443/ 2>/dev/null || true; do
  sleep 1
done
sleep 2

export FWDX_SERVER=https://server:4443
export FWDX_TOKEN=e2e-docker-token

# Create tunnel: subdomain myapp -> app:8080
fwdx tunnel create -l app:8080 -s myapp --name myapp
# Start tunnel in foreground (--watch) so container stays running
exec fwdx tunnel start myapp --watch

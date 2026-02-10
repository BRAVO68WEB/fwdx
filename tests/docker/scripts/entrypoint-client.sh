#!/bin/sh
set -e
# Wait for server web (and gRPC) to be up via admin endpoint
until wget -q -O /dev/null --no-check-certificate --header="Authorization: Bearer e2e-docker-admin" https://server:443/admin/info 2>/dev/null; do
  sleep 1
done
sleep 2

export FWDX_SERVER=https://server:4443
export FWDX_TOKEN=e2e-docker-token

# Create tunnel: subdomain myapp -> app:8080
fwdx tunnel create -l app:8080 -s myapp --name myapp
# Start tunnel in foreground (--watch) so container stays running
exec fwdx tunnel start myapp --watch

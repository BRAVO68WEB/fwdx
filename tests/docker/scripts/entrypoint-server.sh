#!/bin/sh
set -e

exec fwdx serve \
  --hostname server \
  --web-port 80 \
  --grpc-port 4443 \
  --data-dir /tmp/data \
  --oidc-issuer http://mock-oidc:8081 \
  --oidc-client-id fwdx-web \
  --oidc-client-secret test-secret \
  --oidc-redirect-url http://server/auth/oidc/callback \
  --oidc-device-client-id fwdx-cli \
  --oidc-admin-emails admin@example.com \
  --trusted-proxy-cidrs 172.28.0.10/32

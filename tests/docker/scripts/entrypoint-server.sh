#!/bin/sh
set -e
# Use hostname "server" so client can connect via https://server:4443 with valid TLS (CN=server).
CERT_DIR=/tmp/certs
mkdir -p "$CERT_DIR"
openssl req -x509 -nodes -newkey rsa:2048 \
  -keyout "$CERT_DIR/key.pem" \
  -out "$CERT_DIR/cert.pem" \
  -subj "/CN=server" \
  -addext "subjectAltName=DNS:server" \
  -days 1
exec fwdx serve \
  --hostname server \
  --client-token e2e-docker-token \
  --admin-token e2e-docker-admin \
  --tls-cert "$CERT_DIR/cert.pem" \
  --tls-key "$CERT_DIR/key.pem" \
  --web-port 443 \
  --grpc-port 4443 \
  --data-dir /tmp/data

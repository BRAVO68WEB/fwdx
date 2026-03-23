# Traffic flow (gRPC tunnel, current implementation)

This document describes the current forwarding model in `fwdx`.

## Overview

`fwdx` uses a persistent gRPC stream between client and server:

1. Client connects to tunnel endpoint (typically `:4443`).
2. Client sends `Register{hostname, local_url}` with client token in metadata.
3. Server accepts exactly one active stream per hostname.
4. Public HTTP requests on `https://<hostname>` are wrapped as `ProxyRequest` and sent over the stream.
5. Client replays request to local app and returns `ProxyResponse`.
6. Server returns that response to the original visitor request.

## Hostname routing

- Visitor host is normalized as `hostWithoutPort`.
- Server looks up active tunnel stream by exact hostname.
- If no tunnel exists, server returns `404 no tunnel for this hostname`.

## Request/response behavior

- Request body cap defaults to `64MiB` (`FWDX_MAX_REQUEST_BODY_BYTES`).
- gRPC payload cap defaults to `64MiB` (`FWDX_MAX_PROXY_BODY_BYTES`).
- Local response cap defaults to `64MiB` (`FWDX_MAX_RESPONSE_BODY_BYTES`).
- Oversized request body returns `413`.
- Oversized local response returns `502`.

## Retry policy

Client retries local proxy calls up to 3 times only for transport/network errors,
and only for idempotent methods (`GET`, `HEAD`, `OPTIONS`).

## WebSocket and SSE

- SSE is forwarded as normal HTTP headers/body.
- WebSocket upgrade is not implemented in this release and returns `501`.

## Registration conflicts

If a second client attempts to register a hostname that already has an active tunnel,
registration is rejected with `RegisterAck{ok:false,error:"hostname_conflict: ..."}`.

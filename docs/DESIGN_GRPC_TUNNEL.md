# Design: gRPC bidirectional tunnel (extra port)

## Why change?

The current design uses **HTTP long-poll**: the client repeatedly does GET `/tunnel/next-request` and blocks until the server has a request. Each request/response is a separate HTTP round-trip. That works but:

- Multiple HTTP requests per logical “tunnel session” (register, then N× next-request, N× response).
- No true bidirectional channel: the server can’t push unless the client is blocked on next-request.
- Harder to add features like server-initiated pings, flow control, or multiplexing many streams.

A **bidirectional connection** (e.g. gRPC streaming) over a **dedicated tunnel port** gives:

- One long-lived connection per tunnel; both sides can send at any time.
- Single port for all tunnel traffic (e.g. **4443**), separate from public HTTPS (**443**).
- Cleaner protocol: register once over the stream, then request/response messages over the same connection.
- Optional: multiplexing, keepalives, and better backpressure.

---

## Target architecture

- **Port 443** (or nginx → backend): unchanged. Public HTTPS; proxy looks up hostname and forwards to the tunnel backend (now gRPC-aware).
- **Port 4443** (tunnel port): **gRPC server**. Clients connect here with TLS (or over nginx grpc_pass). One bidirectional stream per tunnel; server pushes `ProxyRequest`, client sends `ProxyResponse`.

So we **expose an extra port for the tunnel** (4443) and use gRPC on that port. Public traffic stays on 443.

---

## Protocol (gRPC)

### Service definition

```protobuf
service TunnelService {
  // Bidirectional stream: client sends Register + ProxyResponses; server sends ProxyRequests.
  rpc Tunnel(stream TunnelMessage) returns (stream TunnelMessage);
}
```

Or split for clarity:

```protobuf
service TunnelService {
  // Client opens stream, first message must be Register. Then server sends Request, client sends Response.
  rpc Connect(stream ClientMessage) returns (stream ServerMessage);
}
```

**ClientMessage**: oneof { Register, ProxyResponse }  
**ServerMessage**: oneof { RegisterAck, ProxyRequest }

- **Register**: hostname, local_url, token (or token in metadata).
- **RegisterAck**: success / error (e.g. domain not allowed).
- **ProxyRequest**: id, method, path, query, headers, body.
- **ProxyResponse**: id, status, headers, body.

Auth: token in gRPC metadata (or inside Register). Server validates once when the stream starts.

### Flow

1. Client connects to `server:4443` (gRPC + TLS).
2. Client starts bidirectional stream; first message = **Register** (hostname, local_url).
3. Server validates token and hostname; replies **RegisterAck**. Server adds the stream to the Registry (hostname → stream).
4. When a public request arrives (port 443), proxy looks up hostname, gets the gRPC stream, **sends ProxyRequest** on the stream.
5. Client receives ProxyRequest, forwards to local app, gets response, **sends ProxyResponse** on the same stream.
6. Server receives ProxyResponse, matches by id, unblocks the proxy goroutine, which replies to the browser.
7. Connection stays open; steps 4–6 repeat. If the stream breaks, server removes the hostname from the registry (tunnel gone).

---

## Implementation outline

### 1. Proto and codegen

- Proto: `api/tunnel/v1/tunnel.proto` (TunnelService.Connect, ClientMessage/ServerMessage, Register, ProxyRequest/ProxyResponse).
- Generate Go: `make proto` (or run `protoc` with `--go_out` and `--go-grpc_out`). Requires `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc` on PATH.
- Dependencies: `google.golang.org/grpc`, `google.golang.org/protobuf`.

### 2. Server

- **Tunnel port (4443)**: run a **gRPC server** (TLS or behind nginx with grpc_pass). Implement `TunnelService.Connect` (or `Tunnel`):
  - Accept stream; first message must be Register.
  - Validate token (from metadata or Register), hostname, allowed domains.
  - Create a **stream-backed** tunnel connection: “send ProxyRequest” = send on the gRPC stream; “receive ProxyResponse” = receive from the stream. Map request id → response channel for the proxy to wait on.
  - Register hostname → this stream in the Registry.
  - On stream end or error: Unregister(hostname).
- **Registry**: keep the same interface (Register/Get/Unregister/List) but the “connection” can be either:
  - Current: HTTP long-poll (requestQueue + pending channels), or
  - New: gRPC stream (send request on stream, wait for response via a channel fed by a goroutine that reads from the stream).
- **Proxy**: unchanged. It still calls `conn.EnqueueRequest(ctx, pr)` and blocks until response; the only change is how `EnqueueRequest` is implemented (push to gRPC stream and wait on a channel that the stream reader fills when ProxyResponse arrives).
- **Public port (443)**: unchanged; still HTTP/HTTPS proxy.

### 3. Client

- **New gRPC tunnel client**: connect to `server:4443` (TLS), open bidirectional stream.
  - Send Register(hostname, local_url); read RegisterAck.
  - Loop: read ServerMessage. If ProxyRequest, call local app, send ProxyResponse; if stream closed, exit.
- **Tunnel port config**: client already has `FWDX_SERVER` and tunnel port (443 or 4443). For gRPC we’d use 4443 for the tunnel; public URL can stay 443. So `FWDX_SERVER=https://tunnel.example.com` with tunnel_port=4443, and the client opens a gRPC connection to `tunnel.example.com:4443`.
- **Backward compatibility**: keep the existing HTTP long-poll connector as a fallback (e.g. env `FWDX_USE_GRPC=true` to use gRPC). Or switch entirely to gRPC and drop long-poll.

### 4. Nginx

- **443**: unchanged (reverse proxy to fwdx HTTP backend for public traffic).
- **4443**: proxy to fwdx gRPC server. Use `grpc_pass` (nginx with grpc module) or a stream proxy to the gRPC port. TLS can be terminated at nginx or at fwdx.

---

## File layout (sketch)

```
api/tunnel/v1/
  tunnel.proto       # TunnelService, Register, ProxyRequest, ProxyResponse
  tunnel.pb.go       # generated
  tunnel_grpc.pb.go  # generated

internal/server/
  grpc_tunnel.go     # gRPC server, stream handling, register with Registry
  registry.go        # unchanged interface; tunnel conn can be HTTP or gRPC-backed
  ...

internal/tunnel/
  connector.go       # current HTTP long-poll (keep for compat or remove)
  grpc_connector.go  # new: gRPC stream client, register + request/response loop
```

---

## Migration

- **Phase 1**: Add gRPC server on 4443 alongside existing HTTP tunnel API. Registry accepts both “HTTP conn” and “gRPC conn” (e.g. interface TunnelConn with EnqueueRequest). Proxy unchanged.
- **Phase 2**: Client gains gRPC connector; when tunnel_port=4443 and server supports gRPC, use it.
- **Phase 3**: Deprecate HTTP long-poll tunnel API (or keep for simple deployments that don’t want to open 4443).

---

## Summary

- **Extra port for tunnel**: 4443, gRPC + TLS.
- **Bidirectional**: one stream per tunnel; server sends ProxyRequest, client sends ProxyResponse on the same connection.
- **Public traffic**: still 443, proxy unchanged; it only needs to send requests to and receive responses from the tunnel connection (HTTP or gRPC-backed).
- **Benefits**: single persistent connection, cleaner protocol, room for multiplexing and keepalives.

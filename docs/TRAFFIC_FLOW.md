# How traffic is forwarded from the server to your local system

This document explains how an HTTP request to `https://myapp.tunnel.example.com` reaches your local app (e.g. `localhost:8080`). It describes the **current** HTTP long-poll design. For a **bidirectional** design (gRPC, dedicated tunnel port), see [DESIGN_GRPC_TUNNEL.md](DESIGN_GRPC_TUNNEL.md).

## Overview

The client keeps a **long-poll** connection to the server. When someone visits your tunnel’s hostname, the server pushes the request through that connection to your client; the client forwards it to your local app and sends the response back through the same channel.

```
  Browser                    Server (fwdx)                      Your laptop (fwdx client)
     |                             |                                        |
     |  GET https://myapp.../foo   |                                        |
     | --------------------------> |                                        |
     |                             |  (1) Look up hostname in Registry      |
     |                             |      -> get TunnelConn for myapp...    |
     |                             |  (2) Put request in conn.requestQueue  |
     |                             |      and wait on conn.pending[id]      |
     |                             |                                        |
     |                             |  GET /tunnel/next-request (long-poll)  |
     |                             | <--------------------------------------|
     |                             |  (3) Server unblocks, sends request    |
     |                             | -------------------------------------->|
     |                             |                                        | (4) HTTP GET localhost:8080/foo
     |                             |                                        | ----> local app
     |                             |                                        | <---- response
     |                             |  POST /tunnel/response (body: response) |
     |                             | <--------------------------------------|
     |                             |  (5) Server sends response to pending  |
     |                             |      channel -> proxy unblocks         |
     |                             |                                        |
     |  200 OK + body              |                                        |
     | <-------------------------- |                                        |
```

---

## Step-by-step

### Setup (once)

1. You run **`fwdx tunnel start myapp`** on your laptop.
2. The client **POSTs `/register`** with `{ "hostname": "myapp.tunnel.example.com", "local": "http://localhost:8080" }`.  
   **Server** (`internal/server/tunnel_handler.go`): creates a `TunnelConn` and does **`registry.Register(hostname, conn)`**. That connection has:
   - **`requestQueue`**: a channel the server will push incoming HTTP requests into.
   - **`pending`**: map of request ID → channel where the server will wait for the client’s response.
3. The client then starts a **long-poll loop**: it repeatedly does **GET `/tunnel/next-request?hostname=myapp.tunnel.example.com`** and blocks until the server has a request for that hostname.

### When someone visits https://myapp.tunnel.example.com/foo

4. **Browser** → **nginx** → **fwdx server**  
   Request hits the server’s **ProxyHandler** (`internal/server/proxy.go`).

5. **Server: look up tunnel**  
   `hostname := hostWithoutPort(r.Host)` → `"myapp.tunnel.example.com"`.  
   `conn := registry.Get(hostname)` → the `TunnelConn` you registered.

6. **Server: send request to client**  
   Server builds a **ProxyRequest** (Method, Path, Query, Header, Body, ID) and calls **`conn.EnqueueRequest(ctx, pr)`** (`internal/server/tunnel_handler.go`):
   - Puts `pr` on **`conn.requestQueue`** (so the client’s waiting GET `/tunnel/next-request` returns with this request).
   - Creates a channel for this request ID, stores it in **`conn.pending[id]`**, and **blocks** waiting for the response on that channel (with timeout).

7. **Client: receive request**  
   The client’s **GET `/tunnel/next-request`** unblocks and returns the **ProxyRequest** (JSON).  
   (`internal/tunnel/connector.go`: long-poll loop receives from the server.)

8. **Client: forward to local app**  
   Client calls **`proxyToLocal(localURL, &proxyReq)`**: it does an HTTP request to **`http://localhost:8080/foo`** (same method, path, query, headers, body). Your local app handles it and returns a response.

9. **Client: send response back to server**  
   Client **POSTs `/tunnel/response`** with a **ProxyResponse** (ID, Status, Header, Body).  
   **Server** (`handleSendResponse`): finds the channel in **`conn.pending[resp.ID]`** and sends the response on it → the proxy’s **`EnqueueRequest`** unblocks with that response.

10. **Server: respond to browser**  
    ProxyHandler writes the **ProxyResponse** (status, headers, body) back to the original HTTP response → **browser** gets the reply from your local app.

---

## Important pieces

| Piece | Where | Role |
|-------|--------|-----|
| **Registry** | Server (in-memory) | Maps hostname → `TunnelConn` (the channel + pending map for that tunnel). |
| **TunnelConn.requestQueue** | Server | Channel: proxy pushes **ProxyRequest** here; client’s GET `/tunnel/next-request` reads from it (one request per tunnel). |
| **TunnelConn.pending** | Server | Map request ID → channel: proxy waits here for the **ProxyResponse**; client’s POST `/tunnel/response` sends to this channel. |
| **Long-poll** | Client | GET `/tunnel/next-request` blocks until the server has a request, then client proxies to local and POSTs `/tunnel/response`. |

So: **traffic is forwarded** by (1) server putting the incoming request on the tunnel’s **requestQueue**, (2) client reading it via **next-request**, (3) client calling your **local app**, and (4) client posting the **response** back so the server can answer the browser. No extra “tunnel port” or persistent storage of requests; the live connection between client and server is the tunnel.

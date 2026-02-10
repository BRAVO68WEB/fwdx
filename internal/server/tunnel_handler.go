package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	pathRegister      = "/register"
	pathTunnelNext    = "/tunnel/next-request"
	pathTunnelResponse = "/tunnel/response"
)

// TunnelHandler handles the tunnel control endpoint (registration + request/response streaming).
func TunnelHandler(registry *Registry, clientToken string, allowedDomains func() []string, serverHostname string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == pathRegister:
			handleRegister(w, r, registry, clientToken, allowedDomains, serverHostname)
			return
		case r.Method == http.MethodGet && r.URL.Path == pathTunnelNext:
			handleNextRequest(w, r, registry, clientToken)
			return
		case r.Method == http.MethodPost && r.URL.Path == pathTunnelResponse:
			handleSendResponse(w, r, registry, clientToken)
			return
		default:
			http.NotFound(w, r)
			return
		}
	}
}

func handleRegister(w http.ResponseWriter, r *http.Request, registry *Registry, clientToken string, allowedDomains func() []string, serverHostname string) {
	token := bearerToken(r)
	if token == "" || token != clientToken {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	req.Hostname = strings.TrimSpace(strings.ToLower(req.Hostname))
	req.Local = strings.TrimSpace(req.Local)
	if req.Hostname == "" || req.Local == "" {
		http.Error(w, "hostname and local required", http.StatusBadRequest)
		return
	}

	// Subdomain: must be under server hostname (e.g. foo.tunnel.example.com)
	// Custom domain: must be in allowed list
	if strings.HasSuffix(req.Hostname, "."+serverHostname) || req.Hostname == serverHostname {
		// subdomain or self
	} else {
		allowed := allowedDomains()
		domainAllowed := false
		for _, d := range allowed {
			if d == "" {
				continue
			}
			d = strings.ToLower(strings.TrimSpace(d))
			if req.Hostname == d || strings.HasSuffix(req.Hostname, "."+d) {
				domainAllowed = true
				break
			}
		}
		if !domainAllowed {
			http.Error(w, "domain not allowed", http.StatusForbidden)
			return
		}
	}

	conn := &TunnelConn{
		Hostname:   req.Hostname,
		LocalURL:   req.Local,
		RemoteAddr: r.RemoteAddr,
	}
	registry.Register(req.Hostname, conn)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "hostname": req.Hostname})
}

func handleNextRequest(w http.ResponseWriter, r *http.Request, registry *Registry, clientToken string) {
	token := bearerToken(r)
	if token == "" || token != clientToken {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Find which tunnel this connection belongs to by hostname from query or header
	hostname := r.URL.Query().Get("hostname")
	if hostname == "" {
		hostname = r.Header.Get("X-Tunnel-Hostname")
	}
	if hostname == "" {
		http.Error(w, "hostname required", http.StatusBadRequest)
		return
	}
	hostname = strings.TrimSpace(strings.ToLower(hostname))

	conn := registry.Get(hostname)
	if conn == nil {
		http.Error(w, "tunnel not found", http.StatusNotFound)
		return
	}

	// Block until a request is available or client disconnects
	select {
	case proxyReq, ok := <-conn.requestQueue:
		if !ok {
			http.Error(w, "tunnel closed", http.StatusGone)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-ID", proxyReq.ID)
		if err := json.NewEncoder(w).Encode(proxyReq); err != nil {
			return
		}
	case <-r.Context().Done():
		return
	}
}

func handleSendResponse(w http.ResponseWriter, r *http.Request, registry *Registry, clientToken string) {
	token := bearerToken(r)
	if token == "" || token != clientToken {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	hostname := r.URL.Query().Get("hostname")
	if hostname == "" {
		hostname = r.Header.Get("X-Tunnel-Hostname")
	}
	if hostname == "" {
		http.Error(w, "hostname required", http.StatusBadRequest)
		return
	}
	hostname = strings.TrimSpace(strings.ToLower(hostname))

	conn := registry.Get(hostname)
	if conn == nil {
		http.Error(w, "tunnel not found", http.StatusNotFound)
		return
	}

	var resp ProxyResponse
	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	conn.pendingMu.Lock()
	ch, ok := conn.pending[resp.ID]
	delete(conn.pending, resp.ID)
	conn.pendingMu.Unlock()

	if !ok {
		http.Error(w, "unknown request id", http.StatusBadRequest)
		return
	}

	select {
	case ch <- &resp:
	default:
		// proxy already gave up
	}
	w.WriteHeader(http.StatusNoContent)
}

func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) < len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}

// EnqueueRequest sends a proxy request to the tunnel and waits for the response.
// Called from the public proxy handler. Returns (nil, true) if tunnel closed or timeout.
func (c *TunnelConn) EnqueueRequest(ctx context.Context, pr *ProxyRequest) (resp *ProxyResponse, closed bool) {
	if pr.ID == "" {
		pr.ID = uuid.New().String()
	}
	respCh := make(chan *ProxyResponse, 1)
	c.pendingMu.Lock()
	c.pending[pr.ID] = respCh
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, pr.ID)
		c.pendingMu.Unlock()
	}()

	select {
	case c.requestQueue <- pr:
		// sent
	case <-ctx.Done():
		return nil, true
	}

	// Wait for response with timeout
	timeout := time.NewTimer(60 * time.Second)
	defer timeout.Stop()
	select {
	case r := <-respCh:
		return r, false
	case <-ctx.Done():
		return nil, true
	case <-timeout.C:
		return nil, true
	}
}

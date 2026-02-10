package server

import (
	"sync"
)

// Registry maps hostname to the tunnel connection handling it.
type Registry struct {
	mu      sync.RWMutex
	tunnels map[string]*TunnelConn
}

func NewRegistry() *Registry {
	return &Registry{tunnels: make(map[string]*TunnelConn)}
}

func (r *Registry) Register(hostname string, conn *TunnelConn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if old, ok := r.tunnels[hostname]; ok {
		close(old.requestQueue)
	}
	conn.requestQueue = make(chan *ProxyRequest, 64)
	conn.pending = make(map[string]chan *ProxyResponse)
	r.tunnels[hostname] = conn
}

func (r *Registry) Unregister(hostname string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if conn, ok := r.tunnels[hostname]; ok {
		close(conn.requestQueue)
		delete(r.tunnels, hostname)
	}
}

func (r *Registry) Get(hostname string) *TunnelConn {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tunnels[hostname]
}

// List returns a copy of all registered tunnels (hostname -> remote addr).
func (r *Registry) List() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]string, len(r.tunnels))
	for h, c := range r.tunnels {
		out[h] = c.RemoteAddr
	}
	return out
}

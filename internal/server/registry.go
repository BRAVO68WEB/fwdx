package server

import (
	"sync"
)

// Registry maps hostname to the active tunnel connection (gRPC). In-memory only.
type Registry struct {
	mu      sync.RWMutex
	tunnels map[string]TunnelConnection
}

func NewRegistry() *Registry {
	return &Registry{tunnels: make(map[string]TunnelConnection)}
}

func (r *Registry) Register(hostname string, conn TunnelConnection) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if old := r.tunnels[hostname]; old != nil {
		old.Close()
	}
	r.tunnels[hostname] = conn
}

func (r *Registry) Unregister(hostname string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if conn := r.tunnels[hostname]; conn != nil {
		conn.Close()
		delete(r.tunnels, hostname)
	}
}

func (r *Registry) Get(hostname string) TunnelConnection {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tunnels[hostname]
}

// List returns hostname -> client remote address for all registered tunnels.
func (r *Registry) List() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]string, len(r.tunnels))
	for h, c := range r.tunnels {
		out[h] = c.GetRemoteAddr()
	}
	return out
}

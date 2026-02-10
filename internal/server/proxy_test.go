package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHostWithoutPort(t *testing.T) {
	tests := []struct {
		host string
		want string
	}{
		{"example.com", "example.com"},
		{"example.com:443", "example.com"},
		{"localhost:8080", "localhost"},
		{"[::1]:80", "[::1]"},
		{"", ""},
	}
	for _, tt := range tests {
		got := hostWithoutPort(tt.host)
		if got != tt.want {
			t.Errorf("hostWithoutPort(%q) = %q, want %q", tt.host, got, tt.want)
		}
	}
}

func TestProxyHandler_NoTunnel(t *testing.T) {
	reg := NewRegistry()
	handler := ProxyHandler(reg)

	req := httptest.NewRequest(http.MethodGet, "https://unknown.example.com/", nil)
	req.Host = "unknown.example.com"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if rec.Body.String() != "no tunnel for this hostname\n" {
		t.Errorf("body = %q", rec.Body.String())
	}
}

func TestProxyHandler_WithTunnel_NoResponse(t *testing.T) {
	reg := NewRegistry()
	conn := &TunnelConn{
		Hostname:     "app.example.com",
		LocalURL:     "http://localhost:8080",
		RemoteAddr:   "127.0.0.1",
		requestQueue: make(chan *ProxyRequest, 1),
		pending:      make(map[string]chan *ProxyResponse),
	}
	reg.Register("app.example.com", conn)

	handler := ProxyHandler(reg)
	req := httptest.NewRequest(http.MethodGet, "https://app.example.com/", nil)
	req.Host = "app.example.com"
	rec := httptest.NewRecorder()

	// EnqueueRequest will send to requestQueue but no client will respond; wait for timeout
	ctx, cancel := context.WithTimeout(req.Context(), 100*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	handler.ServeHTTP(rec, req)

	// Should get 502 after timeout (tunnel unavailable)
	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", rec.Code)
	}
}

func TestProxyHandler_HostWithPort(t *testing.T) {
	reg := NewRegistry()
	conn := &TunnelConn{
		Hostname:     "app.example.com",
		requestQueue: make(chan *ProxyRequest, 1),
		pending:      make(map[string]chan *ProxyResponse),
	}
	reg.Register("app.example.com", conn)

	handler := ProxyHandler(reg)
	req := httptest.NewRequest(http.MethodGet, "https://app.example.com/", nil)
	req.Host = "app.example.com:443"
	ctx, cancel := context.WithTimeout(req.Context(), 50*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Lookup should use host without port; we get 502 because no one responds to the queue
	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d (host with port should still resolve)", rec.Code)
	}
}

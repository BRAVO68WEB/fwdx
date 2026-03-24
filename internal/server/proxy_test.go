package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
	handler := ProxyHandler(reg, "tunnel.example.com")

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

func TestProxyHandler_ServerHostname_InfoPage(t *testing.T) {
	reg := NewRegistry()
	handler := ProxyHandler(reg, "tunnel.example.com")

	req := httptest.NewRequest(http.MethodGet, "https://tunnel.example.com/", nil)
	req.Host = "tunnel.example.com"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if body == "" || body == "no tunnel for this hostname\n" {
		t.Errorf("expected info body, got %q", body)
	}
	if !strings.Contains(body, "fwdx server at tunnel.example.com") {
		t.Errorf("body should mention server hostname, got %q", body)
	}
}

// blockingConn blocks in EnqueueRequest until context is done, then returns (nil, true).
type blockingConn struct{}

func (b *blockingConn) EnqueueRequest(ctx context.Context, _ *ProxyRequest) (*ProxyResponse, bool) {
	<-ctx.Done()
	return nil, true
}
func (b *blockingConn) GetRemoteAddr() string { return "127.0.0.1" }
func (b *blockingConn) Close()                {}

func TestProxyHandler_WithTunnel_NoResponse(t *testing.T) {
	reg := NewRegistry()
	reg.Register("app.example.com", &blockingConn{})

	handler := ProxyHandler(reg, "tunnel.example.com")
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
	reg.Register("app.example.com", &blockingConn{})

	handler := ProxyHandler(reg, "tunnel.example.com")
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

func TestProxyHandler_WebsocketNotImplemented(t *testing.T) {
	reg := NewRegistry()
	reg.Register("app.example.com", &blockingConn{})
	handler := ProxyHandler(reg, "tunnel.example.com")

	req := httptest.NewRequest(http.MethodGet, "https://app.example.com/ws", nil)
	req.Host = "app.example.com"
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", rec.Code)
	}
}

func TestProxyHandler_RequestBodyTooLarge(t *testing.T) {
	_ = os.Setenv("FWDX_MAX_REQUEST_BODY_BYTES", "8")
	defer os.Unsetenv("FWDX_MAX_REQUEST_BODY_BYTES")

	reg := NewRegistry()
	reg.Register("app.example.com", &blockingConn{})
	handler := ProxyHandler(reg, "tunnel.example.com")

	req := httptest.NewRequest(http.MethodPost, "https://app.example.com/upload", strings.NewReader("123456789"))
	req.Host = "app.example.com"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rec.Code)
	}
}

type captureConn struct {
	last *ProxyRequest
}

func (c *captureConn) EnqueueRequest(_ context.Context, pr *ProxyRequest) (*ProxyResponse, bool) {
	c.last = pr
	return &ProxyResponse{
		ID:     pr.ID,
		Status: http.StatusOK,
		Header: http.Header{"Content-Type": []string{"text/plain"}},
		Body:   []byte("ok"),
	}, false
}
func (c *captureConn) GetRemoteAddr() string { return "127.0.0.1" }
func (c *captureConn) Close()                {}

func TestProxyHandler_AllowsSSEHeaders(t *testing.T) {
	reg := NewRegistry()
	c := &captureConn{}
	reg.Register("app.example.com", c)
	handler := ProxyHandler(reg, "tunnel.example.com")

	req := httptest.NewRequest(http.MethodGet, "https://app.example.com/sse", nil)
	req.Host = "app.example.com"
	req.Header.Set("Accept", "text/event-stream")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		body, _ := io.ReadAll(rec.Body)
		t.Fatalf("status=%d body=%s", rec.Code, string(body))
	}
	if c.last == nil {
		t.Fatal("expected request to be forwarded")
	}
	if got := c.last.Header.Get("Accept"); got != "text/event-stream" {
		t.Fatalf("accept header=%q", got)
	}
}

func TestMaxRequestBodyBytes_InvalidEnvFallsBack(t *testing.T) {
	_ = os.Setenv("FWDX_MAX_REQUEST_BODY_BYTES", "not-a-number")
	defer os.Unsetenv("FWDX_MAX_REQUEST_BODY_BYTES")
	got := maxRequestBodyBytes()
	if got <= 0 {
		t.Fatalf("invalid max body size: %d", got)
	}
	if got == 8 {
		t.Fatalf("unexpected value %d", got)
	}
}

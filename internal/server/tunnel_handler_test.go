package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testToken = "test-client-token"
const testHostname = "tunnel.example.com"

func TestHandleRegister_Unauthorized(t *testing.T) {
	reg := NewRegistry()
	allowed := func() []string { return nil }
	handler := TunnelHandler(reg, testToken, allowed, testHostname)

	body := []byte(`{"hostname":"app.tunnel.example.com","local":"http://localhost:8080"}`)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// no Authorization
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestHandleRegister_WrongToken(t *testing.T) {
	reg := NewRegistry()
	allowed := func() []string { return nil }
	handler := TunnelHandler(reg, testToken, allowed, testHostname)

	body := []byte(`{"hostname":"app.tunnel.example.com","local":"http://localhost:8080"}`)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestHandleRegister_InvalidJSON(t *testing.T) {
	reg := NewRegistry()
	handler := TunnelHandler(reg, testToken, func() []string { return nil }, testHostname)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader([]byte("not json")))
	req.Header.Set("Authorization", "Bearer "+testToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleRegister_MissingHostnameOrLocal(t *testing.T) {
	reg := NewRegistry()
	handler := TunnelHandler(reg, testToken, func() []string { return nil }, testHostname)

	for _, body := range []string{
		`{"local":"http://localhost:8080"}`,
		`{"hostname":"app.example.com"}`,
		`{}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+testToken)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("body %q: status = %d, want 400", body, rec.Code)
		}
	}
}

func TestHandleRegister_SubdomainOK(t *testing.T) {
	reg := NewRegistry()
	handler := TunnelHandler(reg, testToken, func() []string { return nil }, testHostname)

	body := []byte(`{"hostname":"app.tunnel.example.com","local":"http://localhost:8080"}`)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	conn := reg.Get("app.tunnel.example.com")
	if conn == nil {
		t.Fatal("tunnel not registered")
	}
	if conn.LocalURL != "http://localhost:8080" {
		t.Errorf("LocalURL = %q", conn.LocalURL)
	}
}

func TestHandleRegister_ServerHostnameOK(t *testing.T) {
	reg := NewRegistry()
	handler := TunnelHandler(reg, testToken, func() []string { return nil }, testHostname)

	body := []byte(`{"hostname":"tunnel.example.com","local":"http://localhost:9000"}`)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if reg.Get("tunnel.example.com") == nil {
		t.Error("tunnel not registered")
	}
}

func TestHandleRegister_CustomDomain_ForbiddenWhenNotAllowed(t *testing.T) {
	reg := NewRegistry()
	allowed := func() []string { return []string{"other.domain"} }
	handler := TunnelHandler(reg, testToken, allowed, testHostname)

	body := []byte(`{"hostname":"app.my.domain","local":"http://localhost:8080"}`)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestHandleRegister_CustomDomain_Allowed(t *testing.T) {
	reg := NewRegistry()
	allowed := func() []string { return []string{"my.domain"} }
	handler := TunnelHandler(reg, testToken, allowed, testHostname)

	body := []byte(`{"hostname":"app.my.domain","local":"http://localhost:8080"}`)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if reg.Get("app.my.domain") == nil {
		t.Error("tunnel not registered")
	}
}

func TestHandleNextRequest_NoHostname(t *testing.T) {
	reg := NewRegistry()
	handler := TunnelHandler(reg, testToken, nil, testHostname)

	req := httptest.NewRequest(http.MethodGet, "/tunnel/next-request", nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleNextRequest_TunnelNotFound(t *testing.T) {
	reg := NewRegistry()
	handler := TunnelHandler(reg, testToken, nil, testHostname)

	req := httptest.NewRequest(http.MethodGet, "/tunnel/next-request?hostname=unknown.example.com", nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleSendResponse_Unauthorized(t *testing.T) {
	reg := NewRegistry()
	handler := TunnelHandler(reg, testToken, nil, testHostname)

	body := []byte(`{"id":"req-1","status":200,"header":{},"body":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/tunnel/response?hostname=app.example.com", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestHandleSendResponse_TunnelNotFound(t *testing.T) {
	reg := NewRegistry()
	handler := TunnelHandler(reg, testToken, nil, testHostname)

	body := []byte(`{"id":"req-1","status":200,"header":{},"body":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/tunnel/response?hostname=unknown.example.com", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandler_NotFound(t *testing.T) {
	reg := NewRegistry()
	handler := TunnelHandler(reg, testToken, nil, testHostname)

	req := httptest.NewRequest(http.MethodGet, "/other", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestEnqueueRequest_ResponseRoundtrip(t *testing.T) {
	reg := NewRegistry()
	conn := &TunnelConn{
		Hostname:     "app.example.com",
		LocalURL:     "http://localhost:8080",
		RemoteAddr:   "127.0.0.1",
		requestQueue: make(chan *ProxyRequest, 1),
		pending:      make(map[string]chan *ProxyResponse),
	}
	reg.Register("app.example.com", conn)

	// Simulate client: read request, send response
	go func() {
		req := httptest.NewRequest(http.MethodGet, "https://server/tunnel/next-request?hostname=app.example.com", nil)
		req.Header.Set("Authorization", "Bearer "+testToken)
		req.Header.Set("X-Tunnel-Hostname", "app.example.com")
		rec := httptest.NewRecorder()
		handler := TunnelHandler(reg, testToken, nil, testHostname)
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			return
		}
		var pr ProxyRequest
		if json.NewDecoder(rec.Body).Decode(&pr) != nil {
			return
		}
		respBody, _ := json.Marshal(ProxyResponse{ID: pr.ID, Status: 200, Header: nil, Body: []byte("ok")})
		req2 := httptest.NewRequest(http.MethodPost, "/tunnel/response?hostname=app.example.com", bytes.NewReader(respBody))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Authorization", "Bearer "+testToken)
		rec2 := httptest.NewRecorder()
		handler.ServeHTTP(rec2, req2)
	}()

	pr := &ProxyRequest{Method: "GET", Path: "/", Body: nil}
	resp, closed := conn.EnqueueRequest(context.Background(), pr)
	if closed {
		t.Fatal("EnqueueRequest returned closed=true")
	}
	if resp == nil {
		t.Fatal("EnqueueRequest returned nil resp")
	}
	if resp.Status != 200 {
		t.Errorf("resp.Status = %d", resp.Status)
	}
	if string(resp.Body) != "ok" {
		t.Errorf("resp.Body = %q", resp.Body)
	}
}

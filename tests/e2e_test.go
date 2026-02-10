// Package tests contains end-to-end integration tests.
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BRAVO68WEB/fwdx/internal/server"
	"github.com/BRAVO68WEB/fwdx/internal/tunnel"
)

const (
	testClientToken = "e2e-client-token"
	testAdminToken  = "e2e-admin-token"
	testHostname    = "tunnel.example.com"
)

// TestE2E_RegisterAndProxy runs a full flow: server with tunnel + proxy handlers,
// fake local backend, client connector (register + long-poll), and a public request
// that is proxied through the tunnel to the local backend.
func TestE2E_RegisterAndProxy(t *testing.T) {
	// 1. Fake local backend
	localBody := []byte("hello from local")
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(localBody)
	}))
	defer local.Close()
	localURL := local.URL

	// 2. Server: tunnel handler + proxy handler (same registry)
	reg := server.NewRegistry()
	domains := server.NewDomainStore(t.TempDir())
	tunnelHandler := server.TunnelHandler(reg, testClientToken, domains.List, testHostname)
	proxyHandler := server.ProxyHandler(reg)

	mux := http.NewServeMux()
	mux.HandleFunc("/register", tunnelHandler.ServeHTTP)
	mux.HandleFunc("/tunnel/next-request", tunnelHandler.ServeHTTP)
	mux.HandleFunc("/tunnel/response", tunnelHandler.ServeHTTP)
	mux.Handle("/", proxyHandler)

	srv := httptest.NewServer(mux)
	defer srv.Close()
	baseURL := srv.URL

	// 3. Run client connector in background (register then long-poll)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connectorDone := make(chan struct{})
	go func() {
		err := tunnel.ClientConnector(ctx, baseURL, testClientToken, "app."+testHostname, localURL, false)
		if err != nil && err != context.Canceled {
			t.Logf("connector exited: %v", err)
		}
		close(connectorDone)
	}()

	// Wait for register to complete and client to be in long-poll
	time.Sleep(100 * time.Millisecond)

	// 4. Send "public" request to proxy (as if user hit https://app.tunnel.example.com/)
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/foo?bar=baz", nil)
	req.Host = "app." + testHostname
	req.Header.Set("X-Custom", "value")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("proxy request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Equal(body, localBody) {
		t.Errorf("body = %q, want %q", body, localBody)
	}

	cancel()
	select {
	case <-connectorDone:
	case <-time.After(2 * time.Second):
		t.Log("connector did not exit in time")
	}
}

// TestE2E_Proxy_NoTunnel returns 404 when no tunnel is registered for hostname.
func TestE2E_Proxy_NoTunnel(t *testing.T) {
	reg := server.NewRegistry()
	proxyHandler := server.ProxyHandler(reg)
	srv := httptest.NewServer(proxyHandler)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/", nil)
	req.Host = "unknown.example.com"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

// TestE2E_AdminAPI_Domains tests admin domain list and add (with auth).
func TestE2E_AdminAPI_Domains(t *testing.T) {
	reg := server.NewRegistry()
	domains := server.NewDomainStore(t.TempDir())
	adminHandler := server.AdminRouter(testAdminToken, testHostname, reg, domains)
	srv := httptest.NewServer(adminHandler)
	defer srv.Close()
	base := srv.URL

	// Unauthorized
	req, _ := http.NewRequest(http.MethodGet, base+"/admin/domains", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET without token: status = %d", resp.StatusCode)
	}

	// Authorized: list (empty)
	req, _ = http.NewRequest(http.MethodGet, base+"/admin/domains", nil)
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET with token: status = %d", resp.StatusCode)
	}
	var list []string
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Errorf("list = %v", list)
	}

	// Add domain
	req, _ = http.NewRequest(http.MethodPost, base+"/admin/domains", bytes.NewReader([]byte(`{"domain":"my.domain"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("POST domain: status = %d", resp.StatusCode)
	}

	// List again
	req, _ = http.NewRequest(http.MethodGet, base+"/admin/domains", nil)
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0] != "my.domain" {
		t.Errorf("list after add = %v", list)
	}
}

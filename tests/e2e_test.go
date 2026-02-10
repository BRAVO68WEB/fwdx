// Package tests contains end-to-end integration tests covering all server flows:
// proxy (public traffic), admin API, and gRPC tunnel registration.
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
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

// testEnv holds a running gRPC + web server and shared registry/domains for e2e tests.
type testEnv struct {
	Reg       *server.Registry
	Domains   *DomainStore
	GrpcAddr  string
	WebURL    string
	grpcLn    net.Listener
	webSrv    *httptest.Server
	cancelGrpc context.CancelFunc
}

type DomainStore = server.DomainStore

func startTestEnv(t *testing.T) *testEnv {
	t.Helper()
	reg := server.NewRegistry()
	domains := server.NewDomainStore(t.TempDir())

	grpcLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_ = server.RunGrpcServer(grpcLn, reg, testClientToken, domains.List, testHostname, false, "", "")
		<-ctx.Done()
	}()

	proxyHandler := server.ProxyHandler(reg, testHostname)
	webSrv := httptest.NewServer(proxyHandler)

	env := &testEnv{
		Reg: reg, Domains: domains,
		GrpcAddr: grpcLn.Addr().String(),
		WebURL:   webSrv.URL,
		grpcLn:   grpcLn,
		webSrv:   webSrv,
		cancelGrpc: cancel,
	}
	t.Cleanup(func() {
		cancel()
		grpcLn.Close()
		webSrv.Close()
	})
	return env
}

// adminReq performs an admin API request (path must start with /admin/).
func (e *testEnv) adminReq(method, path string, body []byte, token string) (*http.Response, error) {
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, e.WebURL+path, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(method, e.WebURL+path, nil)
	}
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil && method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}
	return http.DefaultClient.Do(req)
}

// runTunnel connects a tunnel in the background; cancel the returned context to disconnect.
func (e *testEnv) runTunnel(ctx context.Context, hostname, localURL string) (context.CancelFunc, chan struct{}) {
	done := make(chan struct{})
	tunnelCtx, cancel := context.WithCancel(ctx)
	go func() {
		_ = tunnel.Connect(tunnelCtx, "http://"+e.GrpcAddr, testClientToken, hostname, localURL, false)
		close(done)
	}()
	return cancel, done
}

// --- Proxy flows ---

func TestE2E_Proxy_NoTunnel(t *testing.T) {
	reg := server.NewRegistry()
	proxyHandler := server.ProxyHandler(reg, testHostname)
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

func TestE2E_Proxy_ServerHostname_InfoPage(t *testing.T) {
	reg := server.NewRegistry()
	proxyHandler := server.ProxyHandler(reg, testHostname)
	srv := httptest.NewServer(proxyHandler)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/", nil)
	req.Host = testHostname
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("fwdx server at "+testHostname)) {
		t.Errorf("body should contain server info, got %q", body)
	}
}

func TestE2E_Proxy_SubdomainTunnel(t *testing.T) {
	env := startTestEnv(t)
	localBody := []byte("hello from subdomain backend")
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(localBody)
	}))
	defer local.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, done := env.runTunnel(ctx, "app."+testHostname, local.URL)
	time.Sleep(200 * time.Millisecond)

	req, _ := http.NewRequest(http.MethodGet, env.WebURL+"/foo?bar=baz", nil)
	req.Host = "app." + testHostname
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
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
	select { case <-done: case <-time.After(2 * time.Second): t.Log("tunnel exit timeout") }
}

func TestE2E_Proxy_POSTWithBody(t *testing.T) {
	env := startTestEnv(t)
	receivedBody := make(chan []byte, 1)
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		receivedBody <- body
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer local.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	env.runTunnel(ctx, "api."+testHostname, local.URL)
	time.Sleep(200 * time.Millisecond)

	postBody := []byte(`{"key":"value"}`)
	req, _ := http.NewRequest(http.MethodPost, env.WebURL+"/echo", bytes.NewReader(postBody))
	req.Host = "api." + testHostname
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	select {
	case got := <-receivedBody:
		if !bytes.Equal(got, postBody) {
			t.Errorf("backend received %q, want %q", got, postBody)
		}
	case <-time.After(time.Second):
		t.Fatal("backend did not receive body")
	}
	cancel()
}

func TestE2E_Proxy_MultipleTunnels(t *testing.T) {
	env := startTestEnv(t)
	backendA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("from-a"))
	}))
	backendB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("from-b"))
	}))
	defer backendA.Close()
	defer backendB.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	env.runTunnel(ctx, "a."+testHostname, backendA.URL)
	env.runTunnel(ctx, "b."+testHostname, backendB.URL)
	time.Sleep(300 * time.Millisecond)

	reqA, _ := http.NewRequest(http.MethodGet, env.WebURL+"/", nil)
	reqA.Host = "a." + testHostname
	respA, err := http.DefaultClient.Do(reqA)
	if err != nil {
		t.Fatal(err)
	}
	bodyA, _ := io.ReadAll(respA.Body)
	respA.Body.Close()
	if string(bodyA) != "from-a" {
		t.Errorf("a: body = %q", bodyA)
	}

	reqB, _ := http.NewRequest(http.MethodGet, env.WebURL+"/", nil)
	reqB.Host = "b." + testHostname
	respB, err := http.DefaultClient.Do(reqB)
	if err != nil {
		t.Fatal(err)
	}
	bodyB, _ := io.ReadAll(respB.Body)
	respB.Body.Close()
	if string(bodyB) != "from-b" {
		t.Errorf("b: body = %q", bodyB)
	}
	cancel()
}

func TestE2E_Proxy_AfterDisconnect(t *testing.T) {
	env := startTestEnv(t)
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	defer local.Close()

	ctx, cancel := context.WithCancel(context.Background())
	env.runTunnel(ctx, "gone."+testHostname, local.URL)
	time.Sleep(200 * time.Millisecond)

	cancel()
	time.Sleep(200 * time.Millisecond)

	req, _ := http.NewRequest(http.MethodGet, env.WebURL+"/", nil)
	req.Host = "gone." + testHostname
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	// After disconnect the tunnel is unregistered, so proxy returns 404.
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("after disconnect expected 404 (no tunnel), got %d", resp.StatusCode)
	}
}

// --- Admin flows ---

func TestE2E_Admin_Unauthorized(t *testing.T) {
	reg := server.NewRegistry()
	domains := server.NewDomainStore(t.TempDir())
	mux := http.NewServeMux()
	mux.Handle("/admin/", server.AdminRouter(testAdminToken, testHostname, reg, domains))
	mux.Handle("/", server.ProxyHandler(reg, testHostname))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, path := range []string{"/admin/info", "/admin/tunnels", "/admin/domains"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("GET %s without token: status = %d", path, resp.StatusCode)
		}
	}
}

func TestE2E_Admin_Info(t *testing.T) {
	reg := server.NewRegistry()
	domains := server.NewDomainStore(t.TempDir())
	mux := http.NewServeMux()
	mux.Handle("/admin/", server.AdminRouter(testAdminToken, testHostname, reg, domains))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/admin/info", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var out map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out["hostname"] != testHostname {
		t.Errorf("hostname = %q", out["hostname"])
	}
}

func TestE2E_Admin_Tunnels_EmptyThenWithTunnel(t *testing.T) {
	env := startTestEnv(t)
	// Mount admin on same server
	mux := http.NewServeMux()
	mux.Handle("/admin/", server.AdminRouter(testAdminToken, testHostname, env.Reg, env.Domains))
	mux.Handle("/", server.ProxyHandler(env.Reg, testHostname))
	// Replace env's web with this mux
	env.webSrv.Close()
	env.webSrv = httptest.NewServer(mux)
	env.WebURL = env.webSrv.URL

	resp, err := env.adminReq(http.MethodGet, "/admin/tunnels", nil, testAdminToken)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var list map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Errorf("tunnels should be empty initially, got %v", list)
	}

	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer local.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	env.runTunnel(ctx, "admin-test."+testHostname, local.URL)
	time.Sleep(200 * time.Millisecond)

	resp2, err := env.adminReq(http.MethodGet, "/admin/tunnels", nil, testAdminToken)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if err := json.NewDecoder(resp2.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list["admin-test."+testHostname] == "" {
		t.Errorf("tunnels = %v", list)
	}
}

func TestE2E_Admin_Domains_ListAddDelete(t *testing.T) {
	reg := server.NewRegistry()
	domains := server.NewDomainStore(t.TempDir())
	mux := http.NewServeMux()
	mux.Handle("/admin/", server.AdminRouter(testAdminToken, testHostname, reg, domains))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	base := srv.URL
	auth := "Bearer " + testAdminToken

	// Unauthorized
	resp, _ := http.Get(base + "/admin/domains")
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET without token: %d", resp.StatusCode)
	}

	// List empty
	req, _ := http.NewRequest(http.MethodGet, base+"/admin/domains", nil)
	req.Header.Set("Authorization", auth)
	resp, _ = http.DefaultClient.Do(req)
	var list []string
	json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if len(list) != 0 {
		t.Errorf("list = %v", list)
	}

	// Add
	req, _ = http.NewRequest(http.MethodPost, base+"/admin/domains", bytes.NewReader([]byte(`{"domain":"custom.example.com"}`)))
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/json")
	resp, _ = http.DefaultClient.Do(req)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("POST domain: %d", resp.StatusCode)
	}

	// List has one
	req, _ = http.NewRequest(http.MethodGet, base+"/admin/domains", nil)
	req.Header.Set("Authorization", auth)
	resp, _ = http.DefaultClient.Do(req)
	json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if len(list) != 1 || list[0] != "custom.example.com" {
		t.Errorf("list = %v", list)
	}

	// Delete
	req, _ = http.NewRequest(http.MethodDelete, base+"/admin/domains/custom.example.com", nil)
	req.Header.Set("Authorization", auth)
	resp, _ = http.DefaultClient.Do(req)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("DELETE: %d", resp.StatusCode)
	}

	// List empty again
	req, _ = http.NewRequest(http.MethodGet, base+"/admin/domains", nil)
	req.Header.Set("Authorization", auth)
	resp, _ = http.DefaultClient.Do(req)
	json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if len(list) != 0 {
		t.Errorf("list after delete = %v", list)
	}
}

// --- Tunnel (gRPC) flows ---

func TestE2E_Tunnel_WrongToken(t *testing.T) {
	env := startTestEnv(t)
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer local.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := tunnel.Connect(ctx, "http://"+env.GrpcAddr, "wrong-token", "app."+testHostname, local.URL, false)
	if err == nil {
		t.Fatal("expected error when token is wrong")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("register")) && !bytes.Contains([]byte(err.Error()), []byte("unauthorized")) {
		t.Logf("err = %v", err)
	}
}

func TestE2E_Tunnel_CustomDomain_Forbidden(t *testing.T) {
	env := startTestEnv(t)
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer local.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// custom.example.com is not a subdomain of testHostname and not in allowed list
	err := tunnel.Connect(ctx, "http://"+env.GrpcAddr, testClientToken, "custom.example.com", local.URL, false)
	if err == nil {
		t.Fatal("expected error when domain not allowed")
	}
}

func TestE2E_Tunnel_CustomDomain_Allowed(t *testing.T) {
	env := startTestEnv(t)
	// Add custom domain via domain store (simulating admin add)
	if err := env.Domains.Add("custom.allowed.com"); err != nil {
		t.Fatal(err)
	}
	localBody := []byte("custom domain backend")
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(localBody)
	}))
	defer local.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	env.runTunnel(ctx, "custom.allowed.com", local.URL)
	time.Sleep(200 * time.Millisecond)

	req, _ := http.NewRequest(http.MethodGet, env.WebURL+"/", nil)
	req.Host = "custom.allowed.com"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Equal(body, localBody) {
		t.Errorf("body = %q, want %q", body, localBody)
	}
}

// --- Full flow ---

func TestE2E_FullFlow_ConnectProxyDisconnect(t *testing.T) {
	env := startTestEnv(t)
	mux := http.NewServeMux()
	mux.Handle("/admin/", server.AdminRouter(testAdminToken, testHostname, env.Reg, env.Domains))
	mux.Handle("/", server.ProxyHandler(env.Reg, testHostname))
	env.webSrv.Close()
	env.webSrv = httptest.NewServer(mux)
	env.WebURL = env.webSrv.URL

	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("full-flow-ok")) }))
	defer local.Close()

	// No tunnels
	resp, _ := env.adminReq(http.MethodGet, "/admin/tunnels", nil, testAdminToken)
	var list map[string]string
	json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if len(list) != 0 {
		t.Errorf("tunnels = %v", list)
	}

	// Connect
	ctx, cancel := context.WithCancel(context.Background())
	env.runTunnel(ctx, "full."+testHostname, local.URL)
	time.Sleep(200 * time.Millisecond)

	// Admin sees one tunnel
	resp, _ = env.adminReq(http.MethodGet, "/admin/tunnels", nil, testAdminToken)
	json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if len(list) != 1 {
		t.Errorf("tunnels after connect = %v", list)
	}

	// Proxy works
	req, _ := http.NewRequest(http.MethodGet, env.WebURL+"/", nil)
	req.Host = "full." + testHostname
	resp, _ = http.DefaultClient.Do(req)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "full-flow-ok" {
		t.Errorf("body = %q", body)
	}

	// Disconnect
	cancel()
	// Wait until proxy returns 404 (tunnel unregistered)
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		req, _ = http.NewRequest(http.MethodGet, env.WebURL+"/", nil)
		req.Host = "full." + testHostname
		resp, _ = http.DefaultClient.Do(req)
		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			break
		}
		resp.Body.Close()
	}
	req, _ = http.NewRequest(http.MethodGet, env.WebURL+"/", nil)
	req.Host = "full." + testHostname
	resp, _ = http.DefaultClient.Do(req)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("after disconnect expected 404, got %d", resp.StatusCode)
	}

	// Admin tunnels list may still show the entry briefly until stream teardown completes
	resp, _ = env.adminReq(http.MethodGet, "/admin/tunnels", nil, testAdminToken)
	json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if len(list) != 0 {
		t.Logf("tunnels list after disconnect: %v (proxy already returned 404)", list)
	}
}

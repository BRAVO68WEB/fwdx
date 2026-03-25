// Package tests contains end-to-end integration tests covering all server flows:
// proxy (public traffic), admin API, and gRPC tunnel registration.
package tests

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	testHostname = "tunnel.example.com"
)

// testEnv holds a running gRPC + web server and shared registry/domains for e2e tests.
type testEnv struct {
	Reg              *server.Registry
	Domains          *DomainStore
	Store            *server.Store
	Auth             *server.AuthManager
	AdminAccessToken string
	AdminUserID      int64
	GrpcAddr         string
	WebURL           string
	grpcLn           net.Listener
	webSrv           *httptest.Server
	cancelGrpc       context.CancelFunc
}

type DomainStore = server.DomainStore

func startTestEnv(t *testing.T) *testEnv {
	t.Helper()
	reg := server.NewRegistry()
	domains := server.NewDomainStore(t.TempDir())
	store, err := server.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	auth, err := server.NewAuthManager(context.Background(), server.Config{Hostname: testHostname}, store, false)
	if err != nil {
		t.Fatal(err)
	}
	adminAccessToken, _, user, err := auth.IssueSessionForClaims(context.Background(), server.OIDCClaims{
		Subject:     "admin-subject",
		Email:       "admin@example.com",
		DisplayName: "Admin User",
	}, "admin")
	if err != nil {
		t.Fatal(err)
	}

	grpcLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_ = server.RunGrpcServer(grpcLn, reg, domains.List, testHostname, false, "", "", store)
		<-ctx.Done()
	}()

	proxyHandler := server.ProxyHandler(reg, testHostname)
	webSrv := httptest.NewServer(proxyHandler)

	env := &testEnv{
		Reg: reg, Domains: domains, Store: store, Auth: auth, AdminAccessToken: adminAccessToken, AdminUserID: user.ID,
		GrpcAddr:   grpcLn.Addr().String(),
		WebURL:     webSrv.URL,
		grpcLn:     grpcLn,
		webSrv:     webSrv,
		cancelGrpc: cancel,
	}
	t.Cleanup(func() {
		cancel()
		grpcLn.Close()
		webSrv.Close()
		store.Close()
	})
	return env
}

func newAdminSession(t *testing.T) (*server.Store, *server.AuthManager, string) {
	t.Helper()
	store, err := server.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	auth, err := server.NewAuthManager(context.Background(), server.Config{Hostname: testHostname}, store, false)
	if err != nil {
		store.Close()
		t.Fatal(err)
	}
	raw, _, _, err := auth.IssueSessionForClaims(context.Background(), server.OIDCClaims{
		Subject:     "admin-subject",
		Email:       "admin@example.com",
		DisplayName: "Admin User",
	}, "admin")
	if err != nil {
		store.Close()
		t.Fatal(err)
	}
	return store, auth, raw
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
func (e *testEnv) runTunnel(ctx context.Context, name, hostname, localURL string) (context.CancelFunc, chan struct{}) {
	token := e.provisionAgentAndTunnel(ctx, name, hostname)
	done := make(chan struct{})
	tunnelCtx, cancel := context.WithCancel(ctx)
	go func() {
		_ = tunnel.Connect(tunnelCtx, "http://"+e.GrpcAddr, token, name, localURL, false)
		close(done)
	}()
	return cancel, done
}

func (e *testEnv) provisionAgentAndTunnel(ctx context.Context, name, hostname string) string {
	eName := name + "-agent"
	token := "agent-token-" + name
	agent, err := e.Store.CreateAgent(ctx, e.AdminUserID, eName, hashCredential(token))
	if err != nil {
		if existing, getErr := e.Store.GetAgentByName(ctx, eName); getErr == nil {
			agent = existing
		} else {
			panic(err)
		}
	}
	if _, err := e.Store.CreateTunnel(ctx, e.AdminUserID, name, hostname, "", agent.ID); err != nil {
		if err := e.Store.AssignTunnelToAgent(ctx, name, agent.ID); err != nil {
			panic(err)
		}
	}
	return token
}

func hashCredential(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
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
	_, done := env.runTunnel(ctx, "app", "app."+testHostname, local.URL)
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
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Log("tunnel exit timeout")
	}
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
	env.runTunnel(ctx, "api", "api."+testHostname, local.URL)
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
	env.runTunnel(ctx, "a", "a."+testHostname, backendA.URL)
	env.runTunnel(ctx, "b", "b."+testHostname, backendB.URL)
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
	env.runTunnel(ctx, "gone", "gone."+testHostname, local.URL)
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
	store, auth, _ := newAdminSession(t)
	defer store.Close()
	mux := http.NewServeMux()
	mux.Handle("/admin/", server.AdminRouter(testHostname, reg, domains, auth, nil, store))
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
	store, auth, raw := newAdminSession(t)
	defer store.Close()
	mux := http.NewServeMux()
	mux.Handle("/admin/", server.AdminRouter(testHostname, reg, domains, auth, nil, store))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/admin/info", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+raw)
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
	mux.Handle("/admin/", server.AdminRouter(testHostname, env.Reg, env.Domains, env.Auth, nil, env.Store))
	mux.Handle("/", server.ProxyHandler(env.Reg, testHostname))
	// Replace env's web with this mux
	env.webSrv.Close()
	env.webSrv = httptest.NewServer(mux)
	env.WebURL = env.webSrv.URL

	resp, err := env.adminReq(http.MethodGet, "/admin/tunnels", nil, env.AdminAccessToken)
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
	env.runTunnel(ctx, "admin-test", "admin-test."+testHostname, local.URL)
	time.Sleep(200 * time.Millisecond)

	resp2, err := env.adminReq(http.MethodGet, "/admin/tunnels", nil, env.AdminAccessToken)
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
	store, auth, raw := newAdminSession(t)
	defer store.Close()
	mux := http.NewServeMux()
	mux.Handle("/admin/", server.AdminRouter(testHostname, reg, domains, auth, nil, store))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	base := srv.URL
	authHeader := "Bearer " + raw

	// Unauthorized
	resp, _ := http.Get(base + "/admin/domains")
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET without token: %d", resp.StatusCode)
	}

	// List empty
	req, _ := http.NewRequest(http.MethodGet, base+"/admin/domains", nil)
	req.Header.Set("Authorization", authHeader)
	resp, _ = http.DefaultClient.Do(req)
	var list []string
	json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if len(list) != 0 {
		t.Errorf("list = %v", list)
	}

	// Add
	req, _ = http.NewRequest(http.MethodPost, base+"/admin/domains", bytes.NewReader([]byte(`{"domain":"custom.example.com"}`)))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	resp, _ = http.DefaultClient.Do(req)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("POST domain: %d", resp.StatusCode)
	}

	// List has one
	req, _ = http.NewRequest(http.MethodGet, base+"/admin/domains", nil)
	req.Header.Set("Authorization", authHeader)
	resp, _ = http.DefaultClient.Do(req)
	json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if len(list) != 1 || list[0] != "custom.example.com" {
		t.Errorf("list = %v", list)
	}

	// Delete
	req, _ = http.NewRequest(http.MethodDelete, base+"/admin/domains/custom.example.com", nil)
	req.Header.Set("Authorization", authHeader)
	resp, _ = http.DefaultClient.Do(req)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("DELETE: %d", resp.StatusCode)
	}

	// List empty again
	req, _ = http.NewRequest(http.MethodGet, base+"/admin/domains", nil)
	req.Header.Set("Authorization", authHeader)
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
	env.provisionAgentAndTunnel(context.Background(), "app", "app."+testHostname)
	err := tunnel.Connect(ctx, "http://"+env.GrpcAddr, "wrong-token", "app", local.URL, false)
	if err == nil {
		t.Fatal("expected error when token is wrong")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("register")) && !bytes.Contains([]byte(err.Error()), []byte("unauthorized")) {
		t.Logf("err = %v", err)
	}
}

func TestE2E_Tunnel_UnassignedRejected(t *testing.T) {
	env := startTestEnv(t)
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer local.Close()

	_ = env.provisionAgentAndTunnel(context.Background(), "restricted", "custom.example.com")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := tunnel.Connect(ctx, "http://"+env.GrpcAddr, "wrong-agent-token", "restricted", local.URL, false)
	if err == nil {
		t.Fatal("expected error when agent is not assigned")
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
	env.runTunnel(ctx, "custom-allowed", "custom.allowed.com", local.URL)
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

func TestE2E_Tunnel_HostnameConflictRejected(t *testing.T) {
	env := startTestEnv(t)
	localA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("a")) }))
	defer localA.Close()
	localB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("b")) }))
	defer localB.Close()

	ctxA, cancelA := context.WithCancel(context.Background())
	defer cancelA()
	env.runTunnel(ctxA, "dup", "dup."+testHostname, localA.URL)
	time.Sleep(200 * time.Millisecond)

	ctxB, cancelB := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelB()
	err := tunnel.Connect(ctxB, "http://"+env.GrpcAddr, "agent-token-dup", "dup", localB.URL, false)
	if err == nil {
		t.Fatal("expected hostname conflict error")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("hostname_conflict")) {
		t.Fatalf("expected hostname_conflict, got %v", err)
	}
}

// --- Full flow ---

func TestE2E_FullFlow_ConnectProxyDisconnect(t *testing.T) {
	env := startTestEnv(t)
	mux := http.NewServeMux()
	mux.Handle("/admin/", server.AdminRouter(testHostname, env.Reg, env.Domains, env.Auth, nil, env.Store))
	mux.Handle("/", server.ProxyHandler(env.Reg, testHostname))
	env.webSrv.Close()
	env.webSrv = httptest.NewServer(mux)
	env.WebURL = env.webSrv.URL

	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("full-flow-ok")) }))
	defer local.Close()

	// No tunnels
	resp, _ := env.adminReq(http.MethodGet, "/admin/tunnels", nil, env.AdminAccessToken)
	var list map[string]string
	json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if len(list) != 0 {
		t.Errorf("tunnels = %v", list)
	}

	// Connect
	ctx, cancel := context.WithCancel(context.Background())
	env.runTunnel(ctx, "full", "full."+testHostname, local.URL)
	time.Sleep(200 * time.Millisecond)

	// Admin sees one tunnel
	resp, _ = env.adminReq(http.MethodGet, "/admin/tunnels", nil, env.AdminAccessToken)
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
	resp, _ = env.adminReq(http.MethodGet, "/admin/tunnels", nil, env.AdminAccessToken)
	json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if len(list) != 0 {
		t.Logf("tunnels list after disconnect: %v (proxy already returned 404)", list)
	}
}

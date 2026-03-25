package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func noRedirectClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func issueAdminCookie(t *testing.T, auth *AuthManager) *http.Cookie {
	t.Helper()
	raw, expiresAt, _, err := auth.IssueSessionForClaims(context.Background(), OIDCClaims{
		Subject:     "admin-subject",
		Email:       "admin@example.com",
		DisplayName: "Admin User",
	}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: userSessionCookieName, Value: raw, Path: "/", Expires: expiresAt}
}

func TestAdminUI_LoginPage(t *testing.T) {
	cfg := Config{
		Hostname:        "tunnel.myweb.site",
		WebPort:         8080,
		GrpcPort:        4440,
		OIDCIssuer:      "https://issuer.example.com",
		OIDCClientID:    "web-client",
		OIDCRedirectURL: "https://tunnel.myweb.site/auth/oidc/callback",
	}
	reg := NewRegistry()
	domains := NewDomainStore(t.TempDir())
	stats := NewStatsStore()
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ui := AdminUIRouter(cfg, reg, domains, stats, store, nil, time.Now().Add(-2*time.Minute), false)
	srv := httptest.NewServer(ui)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/admin/ui/login")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login page status=%d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "Continue with OIDC") {
		t.Fatalf("expected oidc login button")
	}
}

func TestAdminUI_ConfigWithSession(t *testing.T) {
	cfg := Config{
		Hostname: "tunnel.myweb.site",
		WebPort:  8080,
		GrpcPort: 4440,
	}
	reg := NewRegistry()
	domains := NewDomainStore(t.TempDir())
	stats := NewStatsStore()
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	auth, err := NewAuthManager(context.Background(), cfg, store, false)
	if err != nil {
		t.Fatal(err)
	}
	ui := AdminUIRouter(cfg, reg, domains, stats, store, auth, time.Now().Add(-2*time.Minute), false)
	srv := httptest.NewServer(ui)
	defer srv.Close()

	cookie := issueAdminCookie(t, auth)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/admin/ui/config", nil)
	req.AddCookie(cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("config status=%d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "Tunnel Runtime Auth") {
		t.Fatalf("config page missing agent auth section")
	}
	if !strings.Contains(string(body), "Per-agent credential") {
		t.Fatalf("agent auth mode missing, body=%s", string(body))
	}
}

func TestAdminUI_DisconnectAndDomains(t *testing.T) {
	cfg := Config{
		Hostname: "tunnel.myweb.site",
		WebPort:  8080,
		GrpcPort: 4440,
	}
	reg := NewRegistry()
	reg.Register("app.tunnel.myweb.site", &mockConn{remoteAddr: "127.0.0.1:9999"})
	domains := NewDomainStore(t.TempDir())
	stats := NewStatsStore()
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	auth, err := NewAuthManager(context.Background(), cfg, store, false)
	if err != nil {
		t.Fatal(err)
	}
	ui := AdminUIRouter(cfg, reg, domains, stats, store, auth, time.Now(), false)
	srv := httptest.NewServer(ui)
	defer srv.Close()

	cookie := issueAdminCookie(t, auth)

	frm := strings.NewReader("hostname=app.tunnel.myweb.site")
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/admin/ui/tunnels/disconnect", frm)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("disconnect status=%d", resp.StatusCode)
	}
	if reg.Get("app.tunnel.myweb.site") != nil {
		t.Fatal("expected tunnel disconnected")
	}

	frm = strings.NewReader("domain=mycompany.com")
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/admin/ui/domains/add", frm)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("domains add status=%d", resp.StatusCode)
	}
	if len(domains.List()) != 1 {
		t.Fatalf("expected one domain")
	}
}

func TestAdminUI_EventsSSE(t *testing.T) {
	cfg := Config{
		Hostname: "tunnel.myweb.site",
		WebPort:  8080,
		GrpcPort: 4440,
	}
	reg := NewRegistry()
	domains := NewDomainStore(t.TempDir())
	stats := NewStatsStore()
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	auth, err := NewAuthManager(context.Background(), cfg, store, false)
	if err != nil {
		t.Fatal(err)
	}
	ui := AdminUIRouter(cfg, reg, domains, stats, store, auth, time.Now(), false)
	srv := httptest.NewServer(ui)
	defer srv.Close()

	cookie := issueAdminCookie(t, auth)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/admin/ui/events", nil)
	req.AddCookie(cookie)
	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("sse status=%d", resp.StatusCode)
	}
	buf := make([]byte, 1024)
	n, err := resp.Body.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatal(err)
	}
	if n == 0 || !strings.Contains(string(buf[:n]), "event: stats_tick") {
		t.Fatalf("expected SSE event, got %q", string(buf[:n]))
	}
}

func TestAdminUI_TunnelDetailAndAccessUpdate(t *testing.T) {
	cfg := Config{
		Hostname: "tunnel.myweb.site",
		WebPort:  8080,
		GrpcPort: 4440,
	}
	reg := NewRegistry()
	reg.Register("app.tunnel.myweb.site", &mockConn{remoteAddr: "127.0.0.1:9999"})
	domains := NewDomainStore(t.TempDir())
	stats := NewStatsStore()
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	auth, err := NewAuthManager(context.Background(), cfg, store, false)
	if err != nil {
		t.Fatal(err)
	}
	adminCookie := issueAdminCookie(t, auth)
	created, err := store.CreateTunnel(context.Background(), 0, "app", "app.tunnel.myweb.site", "http://localhost:3000", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AddTunnelEvent(context.Background(), created.Hostname, "register", "connected"); err != nil {
		t.Fatal(err)
	}
	if err := store.InsertRequestLog(context.Background(), RequestLogRecord{
		TunnelID:  created.ID,
		Hostname:  created.Hostname,
		Timestamp: time.Now(),
		Method:    http.MethodGet,
		Host:      created.Hostname,
		Path:      "/",
		Status:    200,
		LatencyMS: 10,
		ClientIP:  "127.0.0.1",
	}); err != nil {
		t.Fatal(err)
	}
	ui := AdminUIRouter(cfg, reg, domains, stats, store, auth, time.Now(), false)
	srv := httptest.NewServer(ui)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/admin/ui/tunnels/app", nil)
	req.AddCookie(adminCookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("detail status=%d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "Access Rules") || !strings.Contains(string(body), "Recent Events") {
		t.Fatalf("detail page missing sections: %s", string(body))
	}

	form := strings.NewReader("auth_mode=basic_auth&basic_auth_username=demo&basic_auth_password=secret")
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/admin/ui/tunnels/app/access", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(adminCookie)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("access update status=%d", resp.StatusCode)
	}
	rule, err := store.GetTunnelAccessRule(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rule.AuthMode != "basic_auth" || rule.BasicAuthUsername != "demo" {
		t.Fatalf("unexpected rule: %+v", rule)
	}
}

package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func loginAndGetCookie(t *testing.T, base string) *http.Cookie {
	t.Helper()
	form := url.Values{}
	form.Set("token", "admin-secret")
	resp, err := noRedirectClient().PostForm(base+"/admin/ui/login", form)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("login status=%d want=302", resp.StatusCode)
	}
	for _, c := range resp.Cookies() {
		if c.Name == adminSessionCookieName {
			return c
		}
	}
	t.Fatal("missing session cookie")
	return nil
}

func TestAdminUI_LoginAndConfig(t *testing.T) {
	cfg := Config{
		Hostname:    "tunnel.myweb.site",
		WebPort:     8080,
		GrpcPort:    4440,
		ClientToken: "client-token-123456",
		AdminToken:  "admin-secret",
	}
	reg := NewRegistry()
	domains := NewDomainStore(t.TempDir())
	stats := NewStatsStore()
	ui := AdminUIRouter(cfg, reg, domains, stats, time.Now().Add(-2*time.Minute), false)
	srv := httptest.NewServer(ui)
	defer srv.Close()

	// unauthorized -> redirect to login
	resp, err := noRedirectClient().Get(srv.URL + "/admin/ui")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected redirect, got %d", resp.StatusCode)
	}

	sess := loginAndGetCookie(t, srv.URL)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/admin/ui/config", nil)
	req.AddCookie(sess)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("config status=%d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "Client Token (non-admin)") {
		t.Fatalf("config page missing token section")
	}
	if !strings.Contains(string(body), "clie...3456") {
		t.Fatalf("masked token missing, body=%s", string(body))
	}
}

func TestAdminUI_DisconnectAndDomains(t *testing.T) {
	cfg := Config{
		Hostname:    "tunnel.myweb.site",
		WebPort:     8080,
		GrpcPort:    4440,
		ClientToken: "client-token-123456",
		AdminToken:  "admin-secret",
	}
	reg := NewRegistry()
	reg.Register("app.tunnel.myweb.site", &mockConn{remoteAddr: "127.0.0.1:9999"})
	domains := NewDomainStore(t.TempDir())
	stats := NewStatsStore()
	ui := AdminUIRouter(cfg, reg, domains, stats, time.Now(), false)
	srv := httptest.NewServer(ui)
	defer srv.Close()

	sess := loginAndGetCookie(t, srv.URL)

	// disconnect active tunnel
	frm := url.Values{}
	frm.Set("hostname", "app.tunnel.myweb.site")
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/admin/ui/tunnels/disconnect", strings.NewReader(frm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sess)
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

	// add domain
	frm = url.Values{}
	frm.Set("domain", "mycompany.com")
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/admin/ui/domains/add", strings.NewReader(frm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sess)
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
		Hostname:    "tunnel.myweb.site",
		WebPort:     8080,
		GrpcPort:    4440,
		ClientToken: "client-token-123456",
		AdminToken:  "admin-secret",
	}
	reg := NewRegistry()
	domains := NewDomainStore(t.TempDir())
	stats := NewStatsStore()
	ui := AdminUIRouter(cfg, reg, domains, stats, time.Now(), false)
	srv := httptest.NewServer(ui)
	defer srv.Close()

	sess := loginAndGetCookie(t, srv.URL)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/admin/ui/events", nil)
	req.AddCookie(sess)
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

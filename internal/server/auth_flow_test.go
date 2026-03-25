package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

type mockOIDCProvider struct {
	server     *httptest.Server
	mu         sync.Mutex
	devicePoll map[string]int
}

func newMockOIDCProvider(t *testing.T) *mockOIDCProvider {
	t.Helper()
	p := &mockOIDCProvider{devicePoll: map[string]int{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                        p.server.URL,
			"authorization_endpoint":        p.server.URL + "/authorize",
			"token_endpoint":                p.server.URL + "/token",
			"userinfo_endpoint":             p.server.URL + "/userinfo",
			"jwks_uri":                      p.server.URL + "/jwks",
			"device_authorization_endpoint": p.server.URL + "/device",
		})
	})
	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, r.URL.Query().Get("redirect_uri")+"?state="+url.QueryEscape(r.URL.Query().Get("state"))+"&code=code-admin", http.StatusFound)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		w.Header().Set("Content-Type", "application/json")
		grantType := r.Form.Get("grant_type")
		switch grantType {
		case "authorization_code":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "access-admin", "token_type": "Bearer", "expires_in": 3600})
		case "urn:ietf:params:oauth:grant-type:device_code":
			deviceCode := r.Form.Get("device_code")
			p.mu.Lock()
			p.devicePoll[deviceCode]++
			count := p.devicePoll[deviceCode]
			p.mu.Unlock()
			if count == 1 {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]any{"error": "authorization_pending", "interval": 1})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "access-device", "token_type": "Bearer", "expires_in": 3600})
		default:
			http.Error(w, "unsupported grant", http.StatusBadRequest)
		}
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		switch auth {
		case "Bearer access-admin":
			_ = json.NewEncoder(w).Encode(map[string]any{"sub": "admin-sub", "email": "admin@example.com", "name": "Admin User", "groups": []string{"ops"}})
		case "Bearer access-device":
			_ = json.NewEncoder(w).Encode(map[string]any{"sub": "member-sub", "email": "member@example.com", "name": "Member User"})
		default:
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}
	})
	mux.HandleFunc("/device", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"device_code":               "device-123",
			"user_code":                 "ABCD-EFGH",
			"verification_uri":          p.server.URL + "/verify",
			"verification_uri_complete": p.server.URL + "/verify?user_code=ABCD-EFGH",
			"expires_in":                300,
			"interval":                  1,
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{}})
	})
	p.server = httptest.NewServer(mux)
	t.Cleanup(p.server.Close)
	return p
}

func TestAuthOIDCCallbackCreatesSession(t *testing.T) {
	provider := newMockOIDCProvider(t)
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	cfg := Config{
		OIDCIssuer:        provider.server.URL,
		OIDCClientID:      "fwdx-web",
		OIDCRedirectURL:   "http://fwdx.test/auth/oidc/callback",
		OIDCAdminEmails:   []string{"admin@example.com"},
		OIDCSessionSecret: "test-secret",
	}
	auth, err := NewAuthManager(context.Background(), cfg, store, false)
	if err != nil {
		t.Fatal(err)
	}

	loginReq := httptest.NewRequest(http.MethodGet, "/auth/oidc/login?redirect=/admin/ui", nil)
	loginRec := httptest.NewRecorder()
	auth.handleOIDCLogin(loginRec, loginReq)
	if loginRec.Code != http.StatusFound {
		t.Fatalf("login status=%d", loginRec.Code)
	}
	redir, err := url.Parse(loginRec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	state := redir.Query().Get("state")
	if state == "" {
		t.Fatal("missing state")
	}

	cbReq := httptest.NewRequest(http.MethodGet, "/auth/oidc/callback?state="+url.QueryEscape(state)+"&code=code-admin", nil)
	cbRec := httptest.NewRecorder()
	auth.handleOIDCCallback(cbRec, cbReq)
	if cbRec.Code != http.StatusFound {
		t.Fatalf("callback status=%d body=%s", cbRec.Code, cbRec.Body.String())
	}
	cookie := cbRec.Result().Cookies()[0]
	whoReq := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	whoReq.AddCookie(cookie)
	whoRec := httptest.NewRecorder()
	auth.handleWhoAmI(whoRec, whoReq)
	if whoRec.Code != http.StatusOK {
		t.Fatalf("whoami status=%d body=%s", whoRec.Code, whoRec.Body.String())
	}
	var out map[string]any
	if err := json.NewDecoder(whoRec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out["role"] != "admin" {
		t.Fatalf("expected admin role, got %v", out["role"])
	}
}

func TestAuthDeviceFlowPollIssuesSession(t *testing.T) {
	provider := newMockOIDCProvider(t)
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	cfg := Config{
		OIDCIssuer:        provider.server.URL,
		OIDCClientID:      "fwdx-web",
		OIDCRedirectURL:   "http://fwdx.test/auth/oidc/callback",
		OIDCSessionSecret: "test-secret",
	}
	auth, err := NewAuthManager(context.Background(), cfg, store, false)
	if err != nil {
		t.Fatal(err)
	}

	startReq := httptest.NewRequest(http.MethodPost, "/auth/device/start", nil)
	startRec := httptest.NewRecorder()
	auth.handleDeviceStart(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("device start status=%d", startRec.Code)
	}

	pollBody := strings.NewReader(`{"device_code":"device-123"}`)
	pollReq := httptest.NewRequest(http.MethodPost, "/auth/device/poll", pollBody)
	pollRec := httptest.NewRecorder()
	auth.handleDevicePoll(pollRec, pollReq)
	if pollRec.Code != http.StatusAccepted {
		t.Fatalf("first poll status=%d body=%s", pollRec.Code, pollRec.Body.String())
	}

	pollReq2 := httptest.NewRequest(http.MethodPost, "/auth/device/poll", strings.NewReader(`{"device_code":"device-123"}`))
	pollRec2 := httptest.NewRecorder()
	auth.handleDevicePoll(pollRec2, pollReq2)
	if pollRec2.Code != http.StatusOK {
		t.Fatalf("second poll status=%d body=%s", pollRec2.Code, pollRec2.Body.String())
	}
	var out map[string]any
	if err := json.NewDecoder(pollRec2.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out["access_token"] == "" {
		t.Fatal("missing access_token")
	}
	whoReq := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	whoReq.Header.Set("Authorization", "Bearer "+out["access_token"].(string))
	whoRec := httptest.NewRecorder()
	auth.handleWhoAmI(whoRec, whoReq)
	if whoRec.Code != http.StatusOK {
		t.Fatalf("whoami status=%d body=%s", whoRec.Code, whoRec.Body.String())
	}
}

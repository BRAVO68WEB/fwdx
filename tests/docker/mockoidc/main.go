package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

type provider struct {
	issuer     string
	mu         sync.Mutex
	devicePoll map[string]int
}

func main() {
	addr := ":8081"
	if v := strings.TrimSpace(os.Getenv("PORT")); v != "" {
		addr = ":" + v
	}
	issuer := strings.TrimSpace(os.Getenv("MOCK_OIDC_ISSUER"))
	if issuer == "" {
		issuer = "http://mock-oidc:8081"
	}
	p := &provider{issuer: issuer, devicePoll: map[string]int{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", p.discovery)
	mux.HandleFunc("/authorize", p.authorize)
	mux.HandleFunc("/token", p.token)
	mux.HandleFunc("/userinfo", p.userinfo)
	mux.HandleFunc("/device", p.device)
	mux.HandleFunc("/jwks", p.jwks)
	mux.HandleFunc("/verify", p.verify)
	log.Printf("mock oidc listening on %s issuer=%s", addr, issuer)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func (p *provider) discovery(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"issuer":                        p.issuer,
		"authorization_endpoint":        p.issuer + "/authorize",
		"token_endpoint":                p.issuer + "/token",
		"userinfo_endpoint":             p.issuer + "/userinfo",
		"jwks_uri":                      p.issuer + "/jwks",
		"device_authorization_endpoint": p.issuer + "/device",
	})
}

func (p *provider) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	redirectURI := strings.TrimSpace(q.Get("redirect_uri"))
	state := strings.TrimSpace(q.Get("state"))
	if redirectURI == "" || state == "" {
		http.Error(w, "redirect_uri and state required", http.StatusBadRequest)
		return
	}
	loginHint := strings.TrimSpace(q.Get("login_hint"))
	code := "code-admin"
	if strings.Contains(strings.ToLower(loginHint), "member") {
		code = "code-member"
	}
	target, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}
	values := target.Query()
	values.Set("state", state)
	values.Set("code", code)
	target.RawQuery = values.Encode()
	http.Redirect(w, r, target.String(), http.StatusFound)
}

func (p *provider) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	grantType := r.Form.Get("grant_type")
	w.Header().Set("Content-Type", "application/json")
	switch grantType {
	case "authorization_code":
		code := r.Form.Get("code")
		token := "access-admin"
		if code == "code-member" {
			token = "access-member"
		}
		writeJSON(w, map[string]any{"access_token": token, "token_type": "Bearer", "expires_in": 3600})
	case "urn:ietf:params:oauth:grant-type:device_code":
		deviceCode := strings.TrimSpace(r.Form.Get("device_code"))
		p.mu.Lock()
		p.devicePoll[deviceCode]++
		count := p.devicePoll[deviceCode]
		p.mu.Unlock()
		if count == 1 {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, map[string]any{"error": "authorization_pending", "interval": 1})
			return
		}
		token := "access-admin"
		if strings.Contains(deviceCode, "member") {
			token = "access-member"
		}
		writeJSON(w, map[string]any{"access_token": token, "token_type": "Bearer", "expires_in": 3600})
	default:
		http.Error(w, "unsupported grant", http.StatusBadRequest)
	}
}

func (p *provider) userinfo(w http.ResponseWriter, r *http.Request) {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	w.Header().Set("Content-Type", "application/json")
	switch auth {
	case "Bearer access-admin":
		writeJSON(w, map[string]any{"sub": "admin-sub", "email": "admin@example.com", "name": "Admin User", "groups": []string{"ops"}})
	case "Bearer access-member":
		writeJSON(w, map[string]any{"sub": "member-sub", "email": "member@example.com", "name": "Member User", "groups": []string{"dev"}})
	default:
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}
}

func (p *provider) device(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"device_code":               "device-admin-123",
		"user_code":                 "ABCD-EFGH",
		"verification_uri":          p.issuer + "/verify",
		"verification_uri_complete": p.issuer + "/verify?user_code=ABCD-EFGH",
		"expires_in":                300,
		"interval":                  1,
	})
}

func (p *provider) jwks(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"keys": []any{}})
}

func (p *provider) verify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("mock oidc verification complete\n"))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

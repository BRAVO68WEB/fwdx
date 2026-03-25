package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func (a *AuthManager) handleOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !a.OIDCEnabled() {
		http.Error(w, "oidc not configured", http.StatusServiceUnavailable)
		return
	}
	redirectTo := strings.TrimSpace(r.URL.Query().Get("redirect"))
	if redirectTo == "" || !strings.HasPrefix(redirectTo, "/") {
		redirectTo = "/admin/ui"
	}
	state, err := randomString(24)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	verifier, challenge, err := generatePKCEVerifier()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := a.store.CreateLoginState(r.Context(), state, verifier, redirectTo, time.Now().Add(10*time.Minute)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, a.oidc.authCodeURL(state, challenge), http.StatusFound)
}

func (a *AuthManager) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !a.OIDCEnabled() {
		http.Error(w, "oidc not configured", http.StatusServiceUnavailable)
		return
	}
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if state == "" || code == "" {
		http.Error(w, "missing oidc callback state or code", http.StatusBadRequest)
		return
	}
	loginState, err := a.store.ConsumeLoginState(r.Context(), state)
	if err != nil {
		http.Error(w, "invalid auth state", http.StatusBadRequest)
		return
	}
	claims, err := a.oidc.exchangeCode(r.Context(), code, loginState.CodeVerifier)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	role := a.userRole(claims)
	user, err := a.store.UpsertUserFromOIDC(r.Context(), claims.Subject, claims.Email, claims.DisplayName, claims.Groups, role)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	raw, expiresAt, err := a.issueSession(r.Context(), user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.setSessionCookie(w, raw, expiresAt)
	http.Redirect(w, r, loginState.RedirectTo, http.StatusFound)
}

func (a *AuthManager) handleOIDCLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token := a.requestSessionToken(r)
	if token != "" {
		_ = a.store.DeleteSession(r.Context(), a.sessionHash(token))
	}
	a.clearSessionCookie(w)
	if strings.HasPrefix(r.URL.Path, "/auth/") {
		http.Redirect(w, r, "/admin/ui/login", http.StatusFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *AuthManager) handleDeviceStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !a.OIDCEnabled() {
		http.Error(w, "oidc not configured", http.StatusServiceUnavailable)
		return
	}
	dev, err := a.oidc.startDeviceAuthorization(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(dev)
}

func (a *AuthManager) handleDevicePoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !a.OIDCEnabled() {
		http.Error(w, "oidc not configured", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		DeviceCode string `json:"device_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	req.DeviceCode = strings.TrimSpace(req.DeviceCode)
	if req.DeviceCode == "" {
		http.Error(w, "device_code required", http.StatusBadRequest)
		return
	}
	result, err := a.oidc.pollDeviceAuthorization(r.Context(), req.DeviceCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if result.Pending {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "pending", "interval": result.Interval})
		return
	}
	role := a.userRole(result.Claims)
	user, err := a.store.UpsertUserFromOIDC(r.Context(), result.Claims.Subject, result.Claims.Email, result.Claims.DisplayName, result.Claims.Groups, role)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	raw, expiresAt, err := a.issueSession(r.Context(), user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token": raw,
		"expires_at":   expiresAt.UTC().Format(time.RFC3339Nano),
		"subject":      user.OIDCSubject,
		"email":        user.Email,
		"display_name": user.DisplayName,
		"role":         user.Role,
	})
}

func (a *AuthManager) requireOIDCConfigured() error {
	if !a.OIDCEnabled() {
		return fmt.Errorf("oidc not configured")
	}
	return nil
}

func (a *AuthManager) logAuthEvent(msg string, args ...any) {
	log.Printf("[fwdx] "+msg, args...)
}

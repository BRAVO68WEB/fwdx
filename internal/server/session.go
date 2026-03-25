package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	userSessionCookieName = "fwdx_session"
	defaultSessionTTL     = 12 * time.Hour
)

type AuthManager struct {
	cfg        Config
	store      *Store
	oidc       *oidcManager
	secure     bool
	sessionKey []byte
}

func NewAuthManager(ctx context.Context, cfg Config, store *Store, secure bool) (*AuthManager, error) {
	if store == nil {
		return nil, fmt.Errorf("store required")
	}
	key := strings.TrimSpace(cfg.OIDCSessionSecret)
	if key == "" {
		switch {
		case strings.TrimSpace(cfg.Hostname) != "":
			key = "fwdx-session::" + cfg.Hostname
			if strings.TrimSpace(cfg.OIDCIssuer) != "" {
				log.Printf("[fwdx] warning: deriving OIDC session secret from hostname; set --oidc-session-secret")
			}
		default:
			key = "fwdx-session::default"
		}
	}
	oidcMgr, err := newOIDCManager(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &AuthManager{cfg: cfg, store: store, oidc: oidcMgr, secure: secure, sessionKey: []byte(key)}, nil
}

func (a *AuthManager) OIDCEnabled() bool {
	return a != nil && a.oidc != nil && a.oidc.Enabled()
}

func (a *AuthManager) IssueSessionForClaims(ctx context.Context, claims OIDCClaims, role string) (string, time.Time, UserRecord, error) {
	user, err := a.store.UpsertUserFromOIDC(ctx, claims.Subject, claims.Email, claims.DisplayName, claims.Groups, role)
	if err != nil {
		return "", time.Time{}, UserRecord{}, err
	}
	raw, expiresAt, err := a.issueSession(ctx, user.ID)
	if err != nil {
		return "", time.Time{}, UserRecord{}, err
	}
	return raw, expiresAt, user, nil
}

func (a *AuthManager) sessionTTL() time.Duration {
	return defaultSessionTTL
}

func (a *AuthManager) sessionHash(raw string) string {
	mac := hmac.New(sha256.New, a.sessionKey)
	_, _ = mac.Write([]byte(raw))
	return hex.EncodeToString(mac.Sum(nil))
}

func (a *AuthManager) issueSession(ctx context.Context, userID int64) (string, time.Time, error) {
	raw, err := randomString(32)
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(a.sessionTTL())
	if err := a.store.CreateSession(ctx, userID, a.sessionHash(raw), expiresAt); err != nil {
		return "", time.Time{}, err
	}
	return raw, expiresAt, nil
}

func (a *AuthManager) setSessionCookie(w http.ResponseWriter, raw string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     userSessionCookieName,
		Value:    raw,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
	})
}

func (a *AuthManager) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     userSessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *AuthManager) requestSessionToken(r *http.Request) string {
	if c, err := r.Cookie(userSessionCookieName); err == nil && strings.TrimSpace(c.Value) != "" {
		return strings.TrimSpace(c.Value)
	}
	const prefix = "Bearer "
	h := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(h) >= len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}

func (a *AuthManager) requestUser(ctx context.Context, r *http.Request) (*UserRecord, int, error) {
	token := a.requestSessionToken(r)
	if token == "" {
		return nil, http.StatusUnauthorized, nil
	}
	_, user, err := a.store.GetSessionByToken(ctx, a.sessionHash(token))
	if err != nil {
		return nil, http.StatusUnauthorized, nil
	}
	return &user, http.StatusOK, nil
}

func (a *AuthManager) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, status, _ := a.requestUser(r.Context(), r)
		if status == http.StatusUnauthorized {
			if strings.EqualFold(r.Header.Get("HX-Request"), "true") {
				w.Header().Set("HX-Redirect", "/admin/ui/login")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/admin/ui/login", http.StatusFound)
			return
		}
		if user == nil || user.Role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func (a *AuthManager) authorizeAdmin(r *http.Request) (bool, int) {
	user, status, _ := a.requestUser(r.Context(), r)
	if status != http.StatusOK {
		return false, status
	}
	if user == nil || user.Role != "admin" {
		return false, http.StatusForbidden
	}
	return true, http.StatusOK
}

func (a *AuthManager) userRole(claims OIDCClaims) string {
	email := strings.ToLower(strings.TrimSpace(claims.Email))
	subject := strings.TrimSpace(claims.Subject)
	if containsStringFold(a.cfg.OIDCAdminEmails, email) || containsStringFold(a.cfg.OIDCAdminSubjects, subject) {
		return "admin"
	}
	for _, g := range claims.Groups {
		if containsStringFold(a.cfg.OIDCAdminGroups, g) {
			return "admin"
		}
	}
	return "member"
}

func containsStringFold(list []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, item := range list {
		if strings.ToLower(strings.TrimSpace(item)) == want && want != "" {
			return true
		}
	}
	return false
}

func (a *AuthManager) handleWhoAmI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, status, _ := a.requestUser(r.Context(), r)
	if status != http.StatusOK || user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"subject":      user.OIDCSubject,
		"email":        user.Email,
		"display_name": user.DisplayName,
		"role":         user.Role,
	})
}

package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

// AdminRouter returns an http.Handler that serves /admin/* with admin token auth.
func AdminRouter(adminToken, hostname string, registry *Registry, domains *DomainStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !checkAdminToken(r, adminToken) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/admin/info":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"hostname": hostname})
			return
		case r.Method == http.MethodGet && r.URL.Path == "/admin/tunnels":
			list := registry.List()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(list)
		case r.Method == http.MethodGet && r.URL.Path == "/admin/domains":
			list := domains.List()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(list)
		case r.Method == http.MethodPost && r.URL.Path == "/admin/domains":
			var body struct {
				Domain string `json:"domain"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			body.Domain = strings.TrimSpace(strings.ToLower(body.Domain))
			if body.Domain == "" {
				http.Error(w, "domain required", http.StatusBadRequest)
				return
			}
			if err := domains.Add(body.Domain); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "domain": body.Domain})
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/admin/domains/"):
			domain := strings.TrimPrefix(r.URL.Path, "/admin/domains/")
			domain = strings.TrimSpace(strings.ToLower(domain))
			if domain == "" {
				http.Error(w, "domain required", http.StatusBadRequest)
				return
			}
			if err := domains.Remove(domain); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})
}

func checkAdminToken(r *http.Request, adminToken string) bool {
	if adminToken == "" {
		return false
	}
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) < len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return false
	}
	return strings.TrimSpace(h[len(prefix):]) == adminToken
}

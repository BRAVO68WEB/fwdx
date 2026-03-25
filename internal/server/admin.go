package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

// AdminRouter returns an http.Handler that serves /admin/* with OIDC session auth.
func AdminRouter(hostname string, registry *Registry, domains *DomainStore, auth *AuthManager, stats ...any) http.Handler {
	var st *StatsStore
	var store *Store
	if len(stats) > 0 {
		if v, ok := stats[0].(*StatsStore); ok {
			st = v
		}
	}
	if len(stats) > 1 {
		if v, ok := stats[1].(*Store); ok {
			store = v
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth == nil {
			http.Error(w, "auth not configured", http.StatusServiceUnavailable)
			return
		}
		ok, status := auth.authorizeAdmin(r)
		if !ok {
			http.Error(w, http.StatusText(status), status)
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
			return
		case r.Method == http.MethodGet && r.URL.Path == "/admin/stats/tunnels":
			if st == nil {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode([]TunnelStats{})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(st.Snapshot(registry.List()))
			return
		case r.Method == http.MethodGet && r.URL.Path == "/admin/control/tunnels":
			if store == nil {
				http.Error(w, "store unavailable", http.StatusServiceUnavailable)
				return
			}
			list, err := store.ListTunnels(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(list)
			return
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/admin/logs/tunnels/"):
			if store == nil {
				http.Error(w, "store unavailable", http.StatusServiceUnavailable)
				return
			}
			hostname := strings.TrimPrefix(r.URL.Path, "/admin/logs/tunnels/")
			hostname = strings.TrimSpace(strings.ToLower(hostname))
			if hostname == "" {
				http.Error(w, "hostname required", http.StatusBadRequest)
				return
			}
			logs, err := store.ListRequestLogs(r.Context(), hostname, 100)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(logs)
			return
		case r.Method == http.MethodGet && r.URL.Path == "/admin/domains":
			list := domains.List()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(list)
			return
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
			return
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
			return
		default:
			http.NotFound(w, r)
		}
	})
}

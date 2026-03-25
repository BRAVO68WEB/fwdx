package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func ControlPlaneRouter(cfg Config, domains *DomainStore, store *Store, auth *AuthManager) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/agents", func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireSessionUser(auth, w, r)
		if !ok {
			return
		}
		switch r.Method {
		case http.MethodGet:
			list, err := store.ListAgentsForUser(r.Context(), user.ID, user.Role == "admin")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, list)
		case http.MethodPost:
			var body struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}
			body.Name = normalizeName(body.Name)
			if body.Name == "" {
				http.Error(w, "name required", http.StatusBadRequest)
				return
			}
			raw, err := randomString(32)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			agent, err := store.CreateAgent(r.Context(), user.ID, body.Name, hashCredential(raw))
			if err != nil {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{"agent": agent, "credential": raw})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/agents/", func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireSessionUser(auth, w, r)
		if !ok {
			return
		}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/agents/"), "/")
		if len(parts) != 2 || parts[1] != "revoke" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		agentName := normalizeName(parts[0])
		agent, err := store.GetAgentByName(r.Context(), agentName)
		if err != nil {
			http.Error(w, "agent not found", http.StatusNotFound)
			return
		}
		if user.Role != "admin" && agent.OwnerUserID != user.ID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := store.RevokeAgentByName(r.Context(), agentName); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
	})
	mux.HandleFunc("/api/tunnels", func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireSessionUser(auth, w, r)
		if !ok {
			return
		}
		switch r.Method {
		case http.MethodGet:
			list, err := store.ListTunnelsForUser(r.Context(), user.ID, user.Role == "admin")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, list)
		case http.MethodPost:
			var body struct {
				Name      string `json:"name"`
				Subdomain string `json:"subdomain"`
				URL       string `json:"url"`
				Local     string `json:"local"`
				AgentName string `json:"agent_name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}
			body.Name = normalizeName(body.Name)
			body.Subdomain = normalizeName(body.Subdomain)
			body.URL = strings.TrimSpace(strings.ToLower(body.URL))
			if body.Name == "" || strings.TrimSpace(body.Local) == "" {
				http.Error(w, "name and local are required", http.StatusBadRequest)
				return
			}
			if (body.Subdomain == "") == (body.URL == "") {
				http.Error(w, "use exactly one of subdomain or url", http.StatusBadRequest)
				return
			}
			hostname, err := resolveTunnelHostname(cfg.Hostname, domains.List(), body.Subdomain, body.URL)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			var agentID int64
			if body.AgentName != "" {
				agent, err := store.GetAgentByName(r.Context(), normalizeName(body.AgentName))
				if err != nil {
					http.Error(w, "agent not found", http.StatusBadRequest)
					return
				}
				if user.Role != "admin" && agent.OwnerUserID != user.ID {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
				agentID = agent.ID
			}
			tun, err := store.CreateTunnel(r.Context(), user.ID, body.Name, hostname, body.Local, agentID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			writeJSON(w, http.StatusCreated, tun)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/tunnels/", func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireSessionUser(auth, w, r)
		if !ok {
			return
		}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/tunnels/"), "/")
		name := normalizeName(parts[0])
		if name == "" {
			http.NotFound(w, r)
			return
		}
		tun, err := store.GetTunnelByName(r.Context(), name)
		if err != nil {
			http.Error(w, "tunnel not found", http.StatusNotFound)
			return
		}
		if user.Role != "admin" && tun.OwnerUserID != user.ID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if len(parts) == 1 {
			switch r.Method {
			case http.MethodGet:
				writeJSON(w, http.StatusOK, tun)
			case http.MethodDelete:
				if err := store.DeleteTunnelByName(r.Context(), name); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}
		if len(parts) != 2 {
			http.NotFound(w, r)
			return
		}
		switch parts[1] {
		case "assign":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var body struct {
				AgentName string `json:"agent_name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}
			var agentID int64
			agentName := normalizeName(body.AgentName)
			if agentName != "" {
				agent, err := store.GetAgentByName(r.Context(), agentName)
				if err != nil {
					http.Error(w, "agent not found", http.StatusBadRequest)
					return
				}
				if user.Role != "admin" && agent.OwnerUserID != user.ID {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
				agentID = agent.ID
			}
			if err := store.AssignTunnelToAgent(r.Context(), name, agentID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if agentName == "" {
				_ = store.AddTunnelEvent(r.Context(), tun.Hostname, "assignment_changed", "assignment cleared")
				writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
				return
			}
			_ = store.AddTunnelEvent(r.Context(), tun.Hostname, "assignment_changed", "assigned to "+agentName)
			writeJSON(w, http.StatusOK, map[string]string{"status": "assigned"})
		case "logs":
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			logs, err := store.ListRequestLogsByTunnel(r.Context(), tun.ID, 100)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, logs)
		case "events":
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			events, err := store.ListTunnelEventsByTunnel(r.Context(), tun.ID, 100)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, events)
		case "access":
			switch r.Method {
			case http.MethodGet:
				rule, err := store.GetTunnelAccessRule(r.Context(), tun.ID)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				writeJSON(w, http.StatusOK, rule)
			case http.MethodPatch:
				var body AccessRuleInput
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid json", http.StatusBadRequest)
					return
				}
				if err := store.UpsertTunnelAccessRule(r.Context(), tun.ID, body); err != nil {
					_ = store.AddTunnelEvent(r.Context(), tun.Hostname, "access_rule_invalid", err.Error())
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				_ = store.AddTunnelEvent(r.Context(), tun.Hostname, "access_rule_changed", "access rule updated")
				rule, err := store.GetTunnelAccessRule(r.Context(), tun.ID)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				writeJSON(w, http.StatusOK, rule)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		case "state":
			if r.Method != http.MethodPatch {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var body struct {
				DesiredState string `json:"desired_state"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}
			body.DesiredState = strings.TrimSpace(strings.ToLower(body.DesiredState))
			if body.DesiredState != "running" && body.DesiredState != "stopped" {
				http.Error(w, "desired_state must be running or stopped", http.StatusBadRequest)
				return
			}
			if body.DesiredState == "running" && tun.AssignedAgentID == 0 {
				http.Error(w, "tunnel has no assigned agent", http.StatusConflict)
				return
			}
			if err := store.SetTunnelDesiredState(r.Context(), name, body.DesiredState); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if body.DesiredState == "stopped" {
				_ = store.UpdateTunnelStateByName(r.Context(), name, "", "offline", tun.LastError, tun.LastSeenAt)
			}
			_ = store.AddTunnelEvent(r.Context(), tun.Hostname, "desired_state_changed", "desired state set to "+body.DesiredState)
			writeJSON(w, http.StatusOK, map[string]string{"status": body.DesiredState})
		case "start":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if tun.AssignedAgentID == 0 {
				http.Error(w, "tunnel has no assigned agent", http.StatusConflict)
				return
			}
			if err := store.SetTunnelDesiredState(r.Context(), name, "running"); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			_ = store.AddTunnelEvent(r.Context(), tun.Hostname, "desired_state_changed", "desired state set to running")
			writeJSON(w, http.StatusOK, map[string]string{"status": "running"})
		case "stop":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if err := store.SetTunnelDesiredState(r.Context(), name, "stopped"); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			_ = store.UpdateTunnelStateByName(r.Context(), name, "", "offline", tun.LastError, tun.LastSeenAt)
			_ = store.AddTunnelEvent(r.Context(), tun.Hostname, "desired_state_changed", "desired state set to stopped")
			writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
		default:
			http.NotFound(w, r)
		}
	})
	return mux
}

func requireSessionUser(auth *AuthManager, w http.ResponseWriter, r *http.Request) (*UserRecord, bool) {
	if auth == nil {
		http.Error(w, "auth not configured", http.StatusServiceUnavailable)
		return nil, false
	}
	user, status, _ := auth.requestUser(r.Context(), r)
	if status != http.StatusOK || user == nil {
		http.Error(w, http.StatusText(status), status)
		return nil, false
	}
	return user, true
}

func hashCredential(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}

func normalizeName(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "-")
	return v
}

func resolveTunnelHostname(serverHostname string, allowedDomains []string, subdomain, customURL string) (string, error) {
	if subdomain != "" {
		return subdomain + "." + strings.ToLower(strings.TrimSpace(serverHostname)), nil
	}
	hostname := strings.TrimSpace(strings.ToLower(customURL))
	if hostname == "" {
		return "", fmt.Errorf("hostname required")
	}
	if hostname == serverHostname || strings.HasSuffix(hostname, "."+serverHostname) {
		return hostname, nil
	}
	for _, d := range allowedDomains {
		d = strings.TrimSpace(strings.ToLower(d))
		if d != "" && (hostname == d || strings.HasSuffix(hostname, "."+d)) {
			return hostname, nil
		}
	}
	return "", fmt.Errorf("domain not allowed")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

package server

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
)

type uiTunnelRow struct {
	Name          string
	Hostname      string
	DesiredState  string
	ActualState   string
	AssignedAgent string
	OwnerEmail    string
	OwnerUserID   int64
	Requests      int64
	Errors        int64
	LastSeenLabel string
	Active        bool
}

type tunnelDetailData struct {
	Tunnel             TunnelRecord
	AccessRule         TunnelAccessRuleRecord
	Agents             []AgentRecord
	Logs               []RequestLogRecord
	Events             []TunnelEventRecord
	Active             bool
	ConnectedRemote    string
	SecretConfigured   bool
	PasswordConfigured bool
}

func (s *adminUIServer) dashboardData(ctx context.Context, user *UserRecord) (dashboardData, error) {
	rows, err := s.tunnelRows(ctx, user)
	if err != nil {
		return dashboardData{}, err
	}
	active := 0
	for _, row := range rows {
		if row.Active {
			active++
		}
	}
	return dashboardData{ActiveCount: active, Tunnels: rows}, nil
}

func (s *adminUIServer) tunnelRows(ctx context.Context, user *UserRecord) ([]uiTunnelRow, error) {
	if s.store == nil {
		return nil, fmt.Errorf("store unavailable")
	}
	if user == nil {
		return nil, fmt.Errorf("user unavailable")
	}
	list, err := s.store.ListTunnelsForUser(ctx, user.ID, user.Role == "admin")
	if err != nil {
		return nil, err
	}
	stats := s.stats.Snapshot(s.registry.List())
	statMap := make(map[string]TunnelStats, len(stats))
	for _, st := range stats {
		statMap[st.Hostname] = st
	}
	rows := make([]uiTunnelRow, 0, len(list))
	for _, tun := range list {
		st := statMap[tun.Hostname]
		lastSeen := "-"
		if !tun.LastSeenAt.IsZero() {
			lastSeen = tun.LastSeenAt.Local().Format("2006-01-02 15:04:05")
		}
		rows = append(rows, uiTunnelRow{
			Name:          tun.Name,
			Hostname:      tun.Hostname,
			DesiredState:  tun.DesiredState,
			ActualState:   tun.ActualState,
			AssignedAgent: tun.AssignedAgent,
			OwnerEmail:    tun.OwnerEmail,
			OwnerUserID:   tun.OwnerUserID,
			Requests:      st.Requests,
			Errors:        st.Errors,
			LastSeenLabel: lastSeen,
			Active:        s.registry.Get(tun.Hostname) != nil,
		})
	}
	return rows, nil
}

func (s *adminUIServer) tunnelRouteHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/admin/ui/tunnels/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		http.NotFound(w, r)
		return
	}
	name := normalizeName(parts[0])
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	switch action {
	case "":
		s.tunnelDetailPageHandler(w, r, name)
	case "status":
		s.tunnelStatusHandler(w, r, name)
	case "access":
		if r.Method == http.MethodGet {
			s.tunnelAccessHandler(w, r, name)
		} else if r.Method == http.MethodPost {
			s.tunnelAccessUpdateHandler(w, r, name)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	case "logs":
		s.tunnelLogsHandler(w, r, name)
	case "events":
		s.tunnelEventsHandler(w, r, name)
	case "assign":
		s.tunnelAssignHandler(w, r, name)
	case "state":
		s.tunnelStateHandler(w, r, name)
	case "delete":
		s.tunnelDeleteHandler(w, r, name)
	default:
		http.NotFound(w, r)
	}
}

func (s *adminUIServer) tunnelDetailPageHandler(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := s.currentAdmin(r)
	data, err := s.loadTunnelDetail(r.Context(), user, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	s.render(w, "layout", s.viewData("Tunnel "+data.Tunnel.Name, "tunnel_detail", user, data))
}

func (s *adminUIServer) tunnelStatusHandler(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := s.currentAdmin(r)
	data, err := s.loadTunnelDetail(r.Context(), user, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	s.render(w, "tunnel_status_card", data)
}

func (s *adminUIServer) tunnelAccessHandler(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := s.currentAdmin(r)
	data, err := s.loadTunnelDetail(r.Context(), user, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	s.render(w, "tunnel_access_card", data)
}

func (s *adminUIServer) tunnelLogsHandler(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := s.currentAdmin(r)
	data, err := s.loadTunnelDetail(r.Context(), user, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	s.render(w, "tunnel_request_logs_table", data)
}

func (s *adminUIServer) tunnelEventsHandler(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := s.currentAdmin(r)
	data, err := s.loadTunnelDetail(r.Context(), user, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	s.render(w, "tunnel_events_list", data)
}

func (s *adminUIServer) tunnelAssignHandler(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := s.currentAdmin(r)
	data, err := s.loadTunnelDetail(r.Context(), user, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	agentName := normalizeName(r.FormValue("agent_name"))
	var agentID int64
	if agentName != "" {
		agents, err := s.store.ListAgentsForUser(r.Context(), user.ID, user.Role == "admin")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ok := false
		for _, agent := range agents {
			if agent.Name == agentName {
				agentID = agent.ID
				ok = true
				break
			}
		}
		if !ok {
			http.Error(w, "agent not available", http.StatusBadRequest)
			return
		}
	}
	if err := s.store.AssignTunnelToAgent(r.Context(), name, agentID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if agentName == "" {
		_ = s.store.AddTunnelEvent(r.Context(), data.Tunnel.Hostname, "assignment_changed", "assignment cleared")
	} else {
		_ = s.store.AddTunnelEvent(r.Context(), data.Tunnel.Hostname, "assignment_changed", "assigned to "+agentName)
	}
	updated, err := s.loadTunnelDetail(r.Context(), user, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	s.render(w, "tunnel_assignment_card", updated)
}

func (s *adminUIServer) tunnelAccessUpdateHandler(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := s.currentAdmin(r)
	data, err := s.loadTunnelDetail(r.Context(), user, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	allowedIPs := []string{}
	for _, line := range strings.Split(r.FormValue("allowed_ips"), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			allowedIPs = append(allowedIPs, line)
		}
	}
	input := AccessRuleInput{
		AuthMode:               strings.TrimSpace(r.FormValue("auth_mode")),
		BasicAuthUsername:      strings.TrimSpace(r.FormValue("basic_auth_username")),
		BasicAuthPassword:      r.FormValue("basic_auth_password"),
		SharedSecretHeaderName: strings.TrimSpace(r.FormValue("shared_secret_header_name")),
		SharedSecretValue:      r.FormValue("shared_secret_value"),
		AllowedIPs:             allowedIPs,
	}
	if err := s.store.UpsertTunnelAccessRule(r.Context(), data.Tunnel.ID, input); err != nil {
		_ = s.store.AddTunnelEvent(r.Context(), data.Tunnel.Hostname, "access_rule_invalid", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = s.store.AddTunnelEvent(r.Context(), data.Tunnel.Hostname, "access_rule_changed", "access rule updated")
	updated, err := s.loadTunnelDetail(r.Context(), user, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	s.render(w, "tunnel_access_card", updated)
}

func (s *adminUIServer) tunnelStateHandler(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := s.currentAdmin(r)
	data, err := s.loadTunnelDetail(r.Context(), user, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	desired := strings.TrimSpace(strings.ToLower(r.FormValue("desired_state")))
	if desired != "running" && desired != "stopped" {
		http.Error(w, "desired_state must be running or stopped", http.StatusBadRequest)
		return
	}
	if desired == "running" && data.Tunnel.AssignedAgentID == 0 {
		http.Error(w, "tunnel has no assigned agent", http.StatusConflict)
		return
	}
	if err := s.store.SetTunnelDesiredState(r.Context(), name, desired); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if desired == "stopped" {
		_ = s.store.UpdateTunnelStateByName(r.Context(), name, "", "offline", data.Tunnel.LastError, data.Tunnel.LastSeenAt)
	}
	_ = s.store.AddTunnelEvent(r.Context(), data.Tunnel.Hostname, "desired_state_changed", "desired state set to "+desired)
	updated, err := s.loadTunnelDetail(r.Context(), user, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	s.render(w, "tunnel_status_card", updated)
}

func (s *adminUIServer) tunnelDeleteHandler(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := s.currentAdmin(r)
	data, err := s.loadTunnelDetail(r.Context(), user, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if s.registry.Get(data.Tunnel.Hostname) != nil {
		s.registry.Disconnect(data.Tunnel.Hostname)
	}
	if err := s.store.DeleteTunnelByName(r.Context(), name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/ui", http.StatusFound)
}

func (s *adminUIServer) loadTunnelDetail(ctx context.Context, user *UserRecord, name string) (tunnelDetailData, error) {
	if s.store == nil {
		return tunnelDetailData{}, fmt.Errorf("store unavailable")
	}
	tun, err := s.store.GetTunnelByName(ctx, name)
	if err != nil {
		return tunnelDetailData{}, err
	}
	if user == nil || (user.Role != "admin" && tun.OwnerUserID != user.ID) {
		return tunnelDetailData{}, fmt.Errorf("forbidden")
	}
	rule, err := s.store.GetTunnelAccessRule(ctx, tun.ID)
	if err != nil && err != sql.ErrNoRows {
		return tunnelDetailData{}, err
	}
	if err == sql.ErrNoRows {
		rule = TunnelAccessRuleRecord{TunnelID: tun.ID, AuthMode: "public"}
	}
	logs, err := s.store.ListRequestLogsByTunnel(ctx, tun.ID, 25)
	if err != nil {
		return tunnelDetailData{}, err
	}
	events, err := s.store.ListTunnelEventsByTunnel(ctx, tun.ID, 25)
	if err != nil {
		return tunnelDetailData{}, err
	}
	agents, err := s.store.ListAgentsForUser(ctx, user.ID, user.Role == "admin")
	if err != nil {
		return tunnelDetailData{}, err
	}
	remote := ""
	active := false
	if conn := s.registry.Get(tun.Hostname); conn != nil {
		active = true
		remote = conn.GetRemoteAddr()
	}
	return tunnelDetailData{
		Tunnel:             tun,
		AccessRule:         rule,
		Agents:             agents,
		Logs:               logs,
		Events:             events,
		Active:             active,
		ConnectedRemote:    remote,
		SecretConfigured:   rule.SharedSecretHash != "",
		PasswordConfigured: rule.BasicAuthPasswordHash != "",
	}, nil
}

var adminUITunnelTemplates = `
{{define "tunnel_detail_content"}}
<div class="card">
  <h2>Tunnel: {{.Tunnel.Name}}</h2>
  <p><b>Hostname:</b> {{.Tunnel.Hostname}}</p>
  <p><b>Owner:</b> {{if .Tunnel.OwnerEmail}}{{.Tunnel.OwnerEmail}}{{else}}user #{{.Tunnel.OwnerUserID}}{{end}}</p>
  <p><b>Local Target Hint:</b> {{.Tunnel.LocalHint}}</p>
  <p><b>Created:</b> {{.Tunnel.CreatedAt.Format "2006-01-02 15:04:05"}}</p>
</div>
<div id="tunnel-status">{{template "tunnel_status_card" .}}</div>
<div id="tunnel-assignment">{{template "tunnel_assignment_card" .}}</div>
<div id="tunnel-access">{{template "tunnel_access_card" .}}</div>
<div id="tunnel-events">{{template "tunnel_events_list" .}}</div>
<div id="tunnel-request-logs">{{template "tunnel_request_logs_table" .}}</div>
<div class="card">
  <form method="post" action="/admin/ui/tunnels/{{.Tunnel.Name}}/delete" onsubmit="return confirm('Delete tunnel {{.Tunnel.Name}}?')">
    <button class="btn red" type="submit">Delete Tunnel</button>
  </form>
</div>
{{end}}

{{define "tunnel_status_card"}}
<div class="card">
  <h3>Status</h3>
  <p><b>Desired:</b> {{.Tunnel.DesiredState}}</p>
  <p><b>Actual:</b> {{.Tunnel.ActualState}}</p>
  <p><b>Connected:</b> {{if .Active}}yes{{else}}no{{end}}</p>
  <p><b>Assigned Agent:</b> {{if .Tunnel.AssignedAgent}}{{.Tunnel.AssignedAgent}}{{else}}-{{end}}</p>
  <p><b>Last Seen:</b> {{if .Tunnel.LastSeenAt.IsZero}}-{{else}}{{.Tunnel.LastSeenAt.Format "2006-01-02 15:04:05"}}{{end}}</p>
  <p><b>Last Error:</b> {{if .Tunnel.LastError}}{{.Tunnel.LastError}}{{else}}-{{end}}</p>
  <p><b>Connected Remote:</b> {{if .ConnectedRemote}}{{.ConnectedRemote}}{{else}}-{{end}}</p>
  <form hx-post="/admin/ui/tunnels/{{.Tunnel.Name}}/state" hx-target="#tunnel-status" hx-swap="innerHTML">
    <input type="hidden" name="desired_state" value="{{if eq .Tunnel.DesiredState "running"}}stopped{{else}}running{{end}}" />
    <button class="btn" type="submit">Set {{if eq .Tunnel.DesiredState "running"}}Stopped{{else}}Running{{end}}</button>
  </form>
</div>
{{end}}

{{define "tunnel_assignment_card"}}
<div class="card">
  <h3>Assignment</h3>
  <p><b>Current Agent:</b> {{if .Tunnel.AssignedAgent}}{{.Tunnel.AssignedAgent}}{{else}}unassigned{{end}}</p>
  <form hx-post="/admin/ui/tunnels/{{.Tunnel.Name}}/assign" hx-target="#tunnel-assignment" hx-swap="innerHTML">
    <select name="agent_name">
      <option value="">Unassigned</option>
      {{range .Agents}}
      <option value="{{.Name}}" {{if eq $.Tunnel.AssignedAgent .Name}}selected{{end}}>{{.Name}} ({{.Status}})</option>
      {{end}}
    </select>
    <button class="btn" type="submit">Save Assignment</button>
  </form>
</div>
{{end}}

{{define "tunnel_access_card"}}
<div class="card">
  <h3>Access Rules</h3>
  <form hx-post="/admin/ui/tunnels/{{.Tunnel.Name}}/access" hx-target="#tunnel-access" hx-swap="innerHTML">
    <p>
      <label><b>Primary Mode</b></label><br/>
      <select name="auth_mode">
        <option value="public" {{if eq .AccessRule.AuthMode "public"}}selected{{end}}>public</option>
        <option value="basic_auth" {{if eq .AccessRule.AuthMode "basic_auth"}}selected{{end}}>basic_auth</option>
        <option value="shared_secret_header" {{if eq .AccessRule.AuthMode "shared_secret_header"}}selected{{end}}>shared_secret_header</option>
      </select>
    </p>
    <p>
      <label><b>Basic Auth Username</b></label><br/>
      <input name="basic_auth_username" value="{{.AccessRule.BasicAuthUsername}}" />
    </p>
    <p>
      <label><b>Basic Auth Password</b></label><br/>
      <input type="password" name="basic_auth_password" placeholder="{{if .PasswordConfigured}}configured - leave blank to keep{{else}}set password{{end}}" />
    </p>
    <p>
      <label><b>Shared Secret Header Name</b></label><br/>
      <input name="shared_secret_header_name" value="{{.AccessRule.SharedSecretHeaderName}}" placeholder="X-Fwdx-Secret" />
    </p>
    <p>
      <label><b>Shared Secret Value</b></label><br/>
      <input type="password" name="shared_secret_value" placeholder="{{if .SecretConfigured}}configured - leave blank to keep{{else}}set secret{{end}}" />
    </p>
    <p>
      <label><b>IP Allowlist</b></label><br/>
      <textarea name="allowed_ips" rows="4" style="width:100%; border:1px solid #cbd5e1; border-radius:8px; padding:8px;">{{range .AccessRule.AllowedIPs}}{{.}}
{{end}}</textarea>
    </p>
    <button class="btn" type="submit">Save Access Rule</button>
  </form>
</div>
{{end}}

{{define "tunnel_events_list"}}
<div class="card">
  <h3>Recent Events</h3>
  <table>
    <thead><tr><th>Time</th><th>Type</th><th>Message</th></tr></thead>
    <tbody>
    {{range .Events}}
      <tr>
        <td>{{.CreatedAt.Format "2006-01-02 15:04:05"}}</td>
        <td>{{.EventType}}</td>
        <td>{{.Message}}</td>
      </tr>
    {{else}}
      <tr><td colspan="3" class="muted">No events yet.</td></tr>
    {{end}}
    </tbody>
  </table>
</div>
{{end}}

{{define "tunnel_request_logs_table"}}
<div class="card">
  <h3>Recent Request Logs</h3>
  <table>
    <thead><tr><th>Time</th><th>Method</th><th>Path</th><th>Status</th><th>Latency</th><th>Client IP</th><th>Error</th></tr></thead>
    <tbody>
    {{range .Logs}}
      <tr>
        <td>{{.Timestamp.Format "2006-01-02 15:04:05"}}</td>
        <td>{{.Method}}</td>
        <td>{{.Path}}</td>
        <td>{{.Status}}</td>
        <td>{{.LatencyMS}} ms</td>
        <td>{{.ClientIP}}</td>
        <td>{{if .ErrorText}}{{.ErrorText}}{{else}}-{{end}}</td>
      </tr>
    {{else}}
      <tr><td colspan="7" class="muted">No request logs yet.</td></tr>
    {{end}}
    </tbody>
  </table>
</div>
{{end}}
`

package server

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type adminUIServer struct {
	cfg      Config
	registry *Registry
	domains  *DomainStore
	stats    *StatsStore
	store    *Store
	auth     *AuthManager
	started  time.Time
	tpl      *template.Template
}

type uiViewData struct {
	Title      string
	Page       string
	Hostname   string
	UserEmail  string
	UserName   string
	UserRole   string
	OIDCIssuer string
	Content    any
}

type dashboardData struct {
	ActiveCount int
	Tunnels     []uiTunnelRow
}

type configData struct {
	Hostname      string
	WebPort       int
	GrpcPort      int
	AgentAuthMode string
	Uptime        string
	ActiveTunnels int
	ReqLimit      int64
	MsgLimit      int
	RespLimit     int64
	OIDCIssuer    string
	OIDCScopes    string
	OIDCEnabled   bool
}

type healthData struct {
	Uptime         string
	ActiveTunnels  int
	AllowedDomains int
	RecentErrors   int
	SSEInterval    string
}

type logsData struct {
	Hostname string
	Logs     []RequestLogRecord
}

// AdminUIRouter provides OIDC-authenticated micro frontend under /admin/ui.
func AdminUIRouter(cfg Config, registry *Registry, domains *DomainStore, stats *StatsStore, store *Store, auth *AuthManager, started time.Time, secureCookie bool) http.Handler {
	if stats == nil {
		stats = NewStatsStore()
	}
	s := &adminUIServer{
		cfg:      cfg,
		registry: registry,
		domains:  domains,
		stats:    stats,
		store:    store,
		auth:     auth,
		started:  started,
		tpl:      template.Must(template.New("ui").Parse(adminUITemplates + adminUITunnelTemplates)),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/admin/ui/login", s.loginHandler)
	mux.HandleFunc("/admin/ui/events", s.requireAdmin(s.eventsHandler))
	mux.HandleFunc("/admin/ui/tunnels/", s.requireAdmin(s.tunnelRouteHandler))
	mux.HandleFunc("/admin/ui/tunnels/table", s.requireAdmin(s.tunnelsTableHandler))
	mux.HandleFunc("/admin/ui/tunnels/disconnect", s.requireAdmin(s.disconnectHandler))
	mux.HandleFunc("/admin/ui/domains/add", s.requireAdmin(s.domainsAddHandler))
	mux.HandleFunc("/admin/ui/domains/remove", s.requireAdmin(s.domainsRemoveHandler))
	mux.HandleFunc("/admin/ui/config", s.requireAdmin(s.configPageHandler))
	mux.HandleFunc("/admin/ui/domains", s.requireAdmin(s.domainsPageHandler))
	mux.HandleFunc("/admin/ui/health", s.requireAdmin(s.healthPageHandler))
	mux.HandleFunc("/admin/ui/logs", s.requireAdmin(s.logsPageHandler))
	mux.HandleFunc("/admin/ui", s.requireAdmin(s.dashboardHandler))
	mux.HandleFunc("/admin/ui/", s.requireAdmin(s.dashboardHandler))
	return mux
}

func (s *adminUIServer) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	if s.auth == nil {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "oidc not configured", http.StatusServiceUnavailable)
		}
	}
	return s.auth.requireAdmin(next)
}

func (s *adminUIServer) currentAdmin(r *http.Request) *UserRecord {
	if s.auth == nil {
		return nil
	}
	user, _, _ := s.auth.requestUser(r.Context(), r)
	return user
}

func (s *adminUIServer) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(s.cfg.OIDCIssuer) == "" {
		http.Error(w, "oidc not configured", http.StatusServiceUnavailable)
		return
	}
	s.render(w, "login", uiViewData{Title: "Sign In", Page: "login", Hostname: s.cfg.Hostname, OIDCIssuer: s.cfg.OIDCIssuer})
}

func (s *adminUIServer) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := s.currentAdmin(r)
	data, err := s.dashboardData(r.Context(), user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if isHX(r) {
		s.render(w, "dashboard_content", data)
		return
	}
	s.render(w, "layout", s.viewData("Dashboard", "dashboard", user, data))
}

func (s *adminUIServer) configPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	d := configData{
		Hostname:      s.cfg.Hostname,
		WebPort:       s.cfg.WebPort,
		GrpcPort:      s.cfg.GrpcPort,
		AgentAuthMode: "Per-agent credential",
		Uptime:        time.Since(s.started).Round(time.Second).String(),
		ActiveTunnels: len(s.registry.List()),
		ReqLimit:      maxRequestBodyBytes(),
		MsgLimit:      maxProxyBodyBytes(),
		RespLimit:     s.maxResponseLimit(),
		OIDCIssuer:    s.cfg.OIDCIssuer,
		OIDCScopes:    strings.Join(s.cfg.OIDCScopes, ", "),
		OIDCEnabled:   s.auth != nil && s.auth.OIDCEnabled(),
	}
	if isHX(r) {
		s.render(w, "config_content", d)
		return
	}
	user := s.currentAdmin(r)
	s.render(w, "layout", s.viewData("Config", "config", user, d))
}

func (s *adminUIServer) domainsPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	d := s.domains.List()
	if isHX(r) {
		s.render(w, "domains_content", d)
		return
	}
	user := s.currentAdmin(r)
	s.render(w, "layout", s.viewData("Domains", "domains", user, d))
}

func (s *adminUIServer) domainsAddHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_ = r.ParseForm()
	domain := strings.TrimSpace(strings.ToLower(r.FormValue("domain")))
	if domain == "" {
		http.Error(w, "domain required", http.StatusBadRequest)
		return
	}
	if err := s.domains.Add(domain); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.render(w, "domains_list", s.domains.List())
}

func (s *adminUIServer) domainsRemoveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_ = r.ParseForm()
	domain := strings.TrimSpace(strings.ToLower(r.FormValue("domain")))
	if domain == "" {
		http.Error(w, "domain required", http.StatusBadRequest)
		return
	}
	if err := s.domains.Remove(domain); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.render(w, "domains_list", s.domains.List())
}

func (s *adminUIServer) healthPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	d := healthData{
		Uptime:         time.Since(s.started).Round(time.Second).String(),
		ActiveTunnels:  len(s.registry.List()),
		AllowedDomains: len(s.domains.List()),
		RecentErrors:   s.stats.RecentErrors(5 * time.Minute),
		SSEInterval:    "1s",
	}
	if isHX(r) {
		s.render(w, "health_content", d)
		return
	}
	user := s.currentAdmin(r)
	s.render(w, "layout", s.viewData("Health", "health", user, d))
}

func (s *adminUIServer) logsPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.store == nil {
		http.Error(w, "store unavailable", http.StatusServiceUnavailable)
		return
	}
	hostname := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("hostname")))
	if hostname == "" {
		for h := range s.registry.List() {
			hostname = h
			break
		}
	}
	logs, err := s.store.ListRequestLogs(r.Context(), hostname, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d := logsData{Hostname: hostname, Logs: logs}
	if isHX(r) {
		s.render(w, "logs_content", d)
		return
	}
	user := s.currentAdmin(r)
	s.render(w, "layout", s.viewData("Logs", "logs", user, d))
}

func (s *adminUIServer) tunnelsTableHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := s.currentAdmin(r)
	rows, err := s.tunnelRows(r.Context(), user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.render(w, "tunnels_table", rows)
}

func (s *adminUIServer) disconnectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_ = r.ParseForm()
	hostname := strings.TrimSpace(strings.ToLower(r.FormValue("hostname")))
	if hostname == "" {
		http.Error(w, "hostname required", http.StatusBadRequest)
		return
	}
	if !s.registry.Disconnect(hostname) {
		http.Error(w, "tunnel not found", http.StatusNotFound)
		return
	}
	s.render(w, "tunnels_table", s.stats.Snapshot(s.registry.List()))
}

func (s *adminUIServer) eventsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", http.StatusInternalServerError)
		return
	}

	t := time.NewTicker(1 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case now := <-t.C:
			_, _ = fmt.Fprintf(w, "event: stats_tick\ndata: {\"ts\":%q}\n\n", now.UTC().Format(time.RFC3339))
			_, _ = fmt.Fprintf(w, "event: tunnels_update\ndata: {}\n\n")
			_, _ = fmt.Fprintf(w, "event: health_update\ndata: {}\n\n")
			_, _ = fmt.Fprintf(w, "event: logs_update\ndata: {}\n\n")
			flusher.Flush()
		}
	}
}

func (s *adminUIServer) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.tpl.ExecuteTemplate(w, name, data)
}

func (s *adminUIServer) viewData(title, page string, user *UserRecord, content any) uiViewData {
	v := uiViewData{Title: title, Page: page, Hostname: s.cfg.Hostname, OIDCIssuer: s.cfg.OIDCIssuer, Content: content}
	if user != nil {
		v.UserEmail = user.Email
		v.UserName = user.DisplayName
		v.UserRole = user.Role
	}
	return v
}

func isHX(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("HX-Request"), "true")
}

func maskTokenValue(s string) string {
	if len(s) <= 8 {
		if s == "" {
			return ""
		}
		return "********"
	}
	return s[:4] + "..." + s[len(s)-4:]
}

func (s *adminUIServer) maxResponseLimit() int64 {
	const def = int64(64 << 20)
	v := strings.TrimSpace(os.Getenv("FWDX_MAX_RESPONSE_BODY_BYTES"))
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

var adminUITemplates = `
{{define "layout"}}
<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>fwdx admin</title>
  <script src="https://unpkg.com/htmx.org@1.9.12"></script>
  <style>
    body { font-family: ui-sans-serif, system-ui; margin: 0; background: #f5f7fb; color: #0f172a; }
    .wrap { display: grid; grid-template-columns: 220px 1fr; min-height: 100vh; }
    nav { background: #0b1220; color: #fff; padding: 16px; }
    nav a { color: #cbd5e1; text-decoration: none; display:block; padding: 8px 0; }
    nav a.active { color: #fff; font-weight: 700; }
    main { padding: 20px; }
    .card { background:#fff; border-radius:12px; padding:16px; margin-bottom:16px; box-shadow: 0 1px 4px rgba(0,0,0,0.06); }
    table { width:100%; border-collapse: collapse; background:#fff; border-radius:10px; overflow:hidden; }
    th, td { text-align:left; padding:10px; border-bottom:1px solid #e2e8f0; font-size: 14px; }
    .muted { color:#64748b; font-size:12px; }
    .topbar { display:flex; justify-content:space-between; align-items:center; gap:12px; margin-bottom:12px; }
    .btn { border:0; border-radius:8px; padding:8px 10px; background:#1d4ed8; color:#fff; cursor:pointer; }
    .btn.red { background:#b91c1c; }
    input { border:1px solid #cbd5e1; border-radius:8px; padding:8px; }
  </style>
</head>
<body>
  <div class="wrap">
    <nav>
      <h3>fwdx admin</h3>
      <a href="/admin/ui" class="{{if eq .Page "dashboard"}}active{{end}}" hx-get="/admin/ui" hx-target="#content" hx-push-url="true">Dashboard</a>
      <a href="/admin/ui/config" class="{{if eq .Page "config"}}active{{end}}" hx-get="/admin/ui/config" hx-target="#content" hx-push-url="true">Config</a>
      <a href="/admin/ui/domains" class="{{if eq .Page "domains"}}active{{end}}" hx-get="/admin/ui/domains" hx-target="#content" hx-push-url="true">Domains</a>
      <a href="/admin/ui/health" class="{{if eq .Page "health"}}active{{end}}" hx-get="/admin/ui/health" hx-target="#content" hx-push-url="true">Health</a>
      <a href="/admin/ui/logs" class="{{if eq .Page "logs"}}active{{end}}" hx-get="/admin/ui/logs" hx-target="#content" hx-push-url="true">Logs</a>
      <form method="post" action="/auth/oidc/logout" style="margin-top:16px;">
        <button class="btn red" type="submit">Logout</button>
      </form>
    </nav>
    <main>
      <div class="topbar">
        <div class="muted">Server: {{.Hostname}}{{if .OIDCIssuer}} | OIDC: {{.OIDCIssuer}}{{end}}</div>
        <div class="muted">{{if .UserName}}{{.UserName}}{{else}}{{.UserEmail}}{{end}}{{if .UserRole}} ({{.UserRole}}){{end}}</div>
      </div>
	      <div id="content">
        {{if eq .Page "dashboard"}}{{template "dashboard_content" .Content}}{{end}}
        {{if eq .Page "config"}}{{template "config_content" .Content}}{{end}}
        {{if eq .Page "domains"}}{{template "domains_content" .Content}}{{end}}
        {{if eq .Page "health"}}{{template "health_content" .Content}}{{end}}
        {{if eq .Page "logs"}}{{template "logs_content" .Content}}{{end}}
        {{if eq .Page "tunnel_detail"}}{{template "tunnel_detail_content" .Content}}{{end}}
      </div>
    </main>
  </div>
  <script>
    (function(){
      if (!window.EventSource) return;
      var es = new EventSource('/admin/ui/events');
      function refresh() {
        if (document.getElementById('tunnels-table')) {
          htmx.ajax('GET', '/admin/ui/tunnels/table', '#tunnels-table');
        }
        if (window.location.pathname.indexOf('/admin/ui/tunnels/') === 0) {
          var base = window.location.pathname;
          if (document.getElementById('tunnel-status')) htmx.ajax('GET', base + '/status', '#tunnel-status');
          if (document.getElementById('tunnel-events')) htmx.ajax('GET', base + '/events', '#tunnel-events');
          if (document.getElementById('tunnel-request-logs')) htmx.ajax('GET', base + '/logs', '#tunnel-request-logs');
        }
        if (window.location.pathname === '/admin/ui/health') {
          htmx.ajax('GET', '/admin/ui/health', '#content');
        }
        if (window.location.pathname === '/admin/ui/logs') {
          htmx.ajax('GET', window.location.pathname + window.location.search, '#content');
        }
      }
      es.addEventListener('stats_tick', refresh);
      es.addEventListener('tunnels_update', refresh);
      es.addEventListener('health_update', refresh);
      es.addEventListener('logs_update', refresh);
      es.onerror = function(){};
    })();
    function toggleToken(fullId, shownId) {
      var shown = document.getElementById(shownId);
      var full = document.getElementById(fullId);
      if (!shown || !full) return;
      if (shown.dataset.revealed === '1') {
        shown.textContent = shown.dataset.masked;
        shown.dataset.revealed = '0';
      } else {
        shown.textContent = full.value;
        shown.dataset.revealed = '1';
      }
    }
    function copyToken(fullId) {
      var full = document.getElementById(fullId);
      if (!full) return;
      navigator.clipboard.writeText(full.value);
    }
  </script>
</body>
</html>
{{end}}

{{define "login"}}
<!doctype html>
<html>
<head><meta charset="utf-8"/><title>fwdx admin login</title></head>
<body style="font-family:ui-sans-serif; background:#f8fafc; padding:40px;">
  <div style="max-width:420px; background:#fff; padding:20px; border-radius:12px; box-shadow:0 1px 4px rgba(0,0,0,0.06);">
    <h3>Sign In</h3>
    <p>Admin access is authenticated through your OIDC provider.</p>
    <p class="muted">Issuer: {{.OIDCIssuer}}</p>
    <a href="/auth/oidc/login?redirect=/admin/ui" style="display:inline-block; padding:10px 14px; background:#1d4ed8; color:#fff; border-radius:8px; text-decoration:none;">Continue with OIDC</a>
  </div>
</body>
</html>
{{end}}

{{define "dashboard_content"}}
<div class="card">
  <h2>Active Tunnels: {{.ActiveCount}}</h2>
  <div id="tunnels-table">{{template "tunnels_table" .Tunnels}}</div>
</div>
{{end}}

{{define "tunnels_table"}}
<table>
  <thead>
    <tr><th>Name</th><th>Hostname</th><th>Desired</th><th>Actual</th><th>Agent</th><th>Owner</th><th>Req</th><th>Err</th><th>Last Seen</th><th>Action</th></tr>
  </thead>
  <tbody>
  {{range .}}
    <tr>
      <td><a href="/admin/ui/tunnels/{{.Name}}">{{.Name}}</a></td>
      <td>{{.Hostname}}</td>
      <td>{{.DesiredState}}</td>
      <td>{{.ActualState}}</td>
      <td>{{if .AssignedAgent}}{{.AssignedAgent}}{{else}}-{{end}}</td>
      <td>{{if .OwnerEmail}}{{.OwnerEmail}}{{else}}user #{{.OwnerUserID}}{{end}}</td>
      <td>{{.Requests}}</td>
      <td>{{.Errors}}</td>
      <td>{{.LastSeenLabel}}</td>
      <td>
        {{if .Active}}
        <form hx-post="/admin/ui/tunnels/disconnect" hx-target="#tunnels-table" hx-swap="innerHTML" onsubmit="return confirm('Disconnect {{.Hostname}}?')">
          <input type="hidden" name="hostname" value="{{.Hostname}}" />
          <button class="btn red" type="submit">Disconnect</button>
        </form>
        {{else}}<a href="/admin/ui/tunnels/{{.Name}}">Open</a>{{end}}
      </td>
    </tr>
  {{else}}
    <tr><td colspan="10" class="muted">No tunnels yet.</td></tr>
  {{end}}
  </tbody>
</table>
{{end}}

{{define "config_content"}}
<div class="card">
  <h2>Config</h2>
  <p><b>Hostname:</b> {{.Hostname}}</p>
  <p><b>Web Port:</b> {{.WebPort}}</p>
  <p><b>gRPC Port:</b> {{.GrpcPort}}</p>
  <p><b>Uptime:</b> {{.Uptime}}</p>
  <p><b>Active Tunnels:</b> {{.ActiveTunnels}}</p>
  <p><b>Request Body Limit:</b> {{.ReqLimit}} bytes</p>
  <p><b>gRPC Msg Limit:</b> {{.MsgLimit}} bytes</p>
  <p><b>Response Body Limit:</b> {{.RespLimit}} bytes</p>
  <hr/>
  <h3>OIDC</h3>
  <p><b>Enabled:</b> {{if .OIDCEnabled}}yes{{else}}no{{end}}</p>
  <p><b>Issuer:</b> {{.OIDCIssuer}}</p>
  <p><b>Scopes:</b> {{.OIDCScopes}}</p>
  <hr/>
  <h3>Tunnel Runtime Auth</h3>
  <p class="muted">Tunnel clients authenticate with server-issued agent credentials.</p>
  <p><b>Mode:</b> {{.AgentAuthMode}}</p>
</div>
{{end}}

{{define "domains_content"}}
<div class="card">
  <h2>Domains</h2>
  <form hx-post="/admin/ui/domains/add" hx-target="#domains-list" hx-swap="innerHTML">
    <input name="domain" placeholder="mycompany.com" />
    <button class="btn" type="submit">Add</button>
  </form>
  <div id="domains-list">{{template "domains_list" .}}</div>
</div>
{{end}}

{{define "domains_list"}}
<table>
  <thead><tr><th>Domain</th><th>Action</th></tr></thead>
  <tbody>
  {{range .}}
    <tr>
      <td>{{.}}</td>
      <td>
        <form hx-post="/admin/ui/domains/remove" hx-target="#domains-list" hx-swap="innerHTML" onsubmit="return confirm('Remove {{.}}?')">
          <input type="hidden" name="domain" value="{{.}}" />
          <button class="btn red" type="submit">Remove</button>
        </form>
      </td>
    </tr>
  {{else}}
    <tr><td colspan="2" class="muted">No allowed domains.</td></tr>
  {{end}}
  </tbody>
</table>
{{end}}

{{define "health_content"}}
<div class="card">
  <h2>Health</h2>
  <p><b>Uptime:</b> {{.Uptime}}</p>
  <p><b>Active Tunnels:</b> {{.ActiveTunnels}}</p>
  <p><b>Allowed Domains:</b> {{.AllowedDomains}}</p>
  <p><b>Recent Proxy Errors (5m):</b> {{.RecentErrors}}</p>
  <p><b>SSE Tick:</b> {{.SSEInterval}}</p>
</div>
{{end}}

{{define "logs_content"}}
<div class="card">
  <h2>Request Logs</h2>
  <p class="muted">Showing recent persisted logs{{if .Hostname}} for {{.Hostname}}{{end}}</p>
  <table>
    <thead>
      <tr><th>Time</th><th>Method</th><th>Path</th><th>Status</th><th>Latency ms</th><th>Bytes In</th><th>Bytes Out</th><th>Error</th></tr>
    </thead>
    <tbody>
    {{range .Logs}}
      <tr>
        <td>{{.Timestamp.Format "2006-01-02 15:04:05"}}</td>
        <td>{{.Method}}</td>
        <td>{{.Path}}</td>
        <td>{{.Status}}</td>
        <td>{{.LatencyMS}}</td>
        <td>{{.BytesIn}}</td>
        <td>{{.BytesOut}}</td>
        <td>{{.ErrorText}}</td>
      </tr>
    {{else}}
      <tr><td colspan="8" class="muted">No persisted request logs yet.</td></tr>
    {{end}}
    </tbody>
  </table>
</div>
{{end}}
`

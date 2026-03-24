package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	adminSessionCookieName = "fwdx_admin_session"
	adminSessionTTL        = 12 * time.Hour
)

type adminUIServer struct {
	cfg      Config
	registry *Registry
	domains  *DomainStore
	stats    *StatsStore
	started  time.Time
	secure   bool
	secret   []byte
	tpl      *template.Template
}

type uiViewData struct {
	Title    string
	Page     string
	Hostname string
	Content  any
}

type dashboardData struct {
	ActiveCount int
	Stats       []TunnelStats
}

type configData struct {
	Hostname      string
	WebPort       int
	GrpcPort      int
	ClientToken   string
	MaskedToken   string
	Uptime        string
	ActiveTunnels int
	ReqLimit      int64
	MsgLimit      int
	RespLimit     int64
}

type healthData struct {
	Uptime         string
	ActiveTunnels  int
	AllowedDomains int
	RecentErrors   int
	SSEInterval    string
}

// AdminUIRouter provides cookie-authenticated micro frontend under /admin/ui.
func AdminUIRouter(cfg Config, registry *Registry, domains *DomainStore, stats *StatsStore, started time.Time, secureCookie bool) http.Handler {
	if stats == nil {
		stats = NewStatsStore()
	}
	secret := strings.TrimSpace(os.Getenv("FWDX_UI_SESSION_SECRET"))
	if secret == "" {
		secret = "fwdx-ui::" + cfg.AdminToken
	}
	s := &adminUIServer{
		cfg:      cfg,
		registry: registry,
		domains:  domains,
		stats:    stats,
		started:  started,
		secure:   secureCookie,
		secret:   []byte(secret),
		tpl:      template.Must(template.New("ui").Parse(adminUITemplates)),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/admin/ui/login", s.loginHandler)
	mux.HandleFunc("/admin/ui/logout", s.requireAuth(s.logoutHandler))
	mux.HandleFunc("/admin/ui/events", s.requireAuth(s.eventsHandler))
	mux.HandleFunc("/admin/ui/tunnels/table", s.requireAuth(s.tunnelsTableHandler))
	mux.HandleFunc("/admin/ui/tunnels/disconnect", s.requireAuth(s.disconnectHandler))
	mux.HandleFunc("/admin/ui/domains/add", s.requireAuth(s.domainsAddHandler))
	mux.HandleFunc("/admin/ui/domains/remove", s.requireAuth(s.domainsRemoveHandler))
	mux.HandleFunc("/admin/ui/config", s.requireAuth(s.configPageHandler))
	mux.HandleFunc("/admin/ui/domains", s.requireAuth(s.domainsPageHandler))
	mux.HandleFunc("/admin/ui/health", s.requireAuth(s.healthPageHandler))
	mux.HandleFunc("/admin/ui", s.requireAuth(s.dashboardHandler))
	mux.HandleFunc("/admin/ui/", s.requireAuth(s.dashboardHandler))
	return mux
}

func (s *adminUIServer) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.validateSessionCookie(r) {
			if strings.EqualFold(r.Header.Get("HX-Request"), "true") {
				w.Header().Set("HX-Redirect", "/admin/ui/login")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/admin/ui/login", http.StatusFound)
			return
		}
		next(w, r)
	}
}

func (s *adminUIServer) loginHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.render(w, "login", uiViewData{Title: "Admin Login", Page: "login", Hostname: s.cfg.Hostname})
	case http.MethodPost:
		_ = r.ParseForm()
		token := strings.TrimSpace(r.FormValue("token"))
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.AdminToken)) != 1 {
			http.Error(w, "invalid admin token", http.StatusUnauthorized)
			return
		}
		s.setSessionCookie(w)
		http.Redirect(w, r, "/admin/ui", http.StatusFound)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *adminUIServer) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    "",
		Path:     "/admin/ui",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/admin/ui/login", http.StatusFound)
}

func (s *adminUIServer) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data := dashboardData{ActiveCount: len(s.registry.List()), Stats: s.stats.Snapshot(s.registry.List())}
	if isHX(r) {
		s.render(w, "dashboard_content", data)
		return
	}
	s.render(w, "layout", uiViewData{Title: "Dashboard", Page: "dashboard", Hostname: s.cfg.Hostname, Content: data})
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
		ClientToken:   s.cfg.ClientToken,
		MaskedToken:   maskTokenValue(s.cfg.ClientToken),
		Uptime:        time.Since(s.started).Round(time.Second).String(),
		ActiveTunnels: len(s.registry.List()),
		ReqLimit:      maxRequestBodyBytes(),
		MsgLimit:      maxProxyBodyBytes(),
		RespLimit:     s.maxResponseLimit(),
	}
	if isHX(r) {
		s.render(w, "config_content", d)
		return
	}
	s.render(w, "layout", uiViewData{Title: "Config", Page: "config", Hostname: s.cfg.Hostname, Content: d})
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
	s.render(w, "layout", uiViewData{Title: "Domains", Page: "domains", Hostname: s.cfg.Hostname, Content: d})
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
	s.render(w, "layout", uiViewData{Title: "Health", Page: "health", Hostname: s.cfg.Hostname, Content: d})
}

func (s *adminUIServer) tunnelsTableHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.render(w, "tunnels_table", s.stats.Snapshot(s.registry.List()))
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
			flusher.Flush()
		}
	}
}

func (s *adminUIServer) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.tpl.ExecuteTemplate(w, name, data)
}

func (s *adminUIServer) setSessionCookie(w http.ResponseWriter) {
	payload := fmt.Sprintf("%d:%d", time.Now().Unix(), time.Now().Add(adminSessionTTL).Unix())
	payloadB64 := base64.RawURLEncoding.EncodeToString([]byte(payload))
	sig := hmacSHA256(s.secret, payloadB64)
	val := payloadB64 + "." + base64.RawURLEncoding.EncodeToString(sig)
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    val,
		Path:     "/admin/ui",
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(adminSessionTTL),
	})
}

func (s *adminUIServer) validateSessionCookie(r *http.Request) bool {
	c, err := r.Cookie(adminSessionCookieName)
	if err != nil {
		return false
	}
	parts := strings.Split(c.Value, ".")
	if len(parts) != 2 {
		return false
	}
	want := hmacSHA256(s.secret, parts[0])
	got, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	if !hmac.Equal(want, got) {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	p := strings.Split(string(payload), ":")
	if len(p) != 2 {
		return false
	}
	expUnix, err := strconv.ParseInt(p[1], 10, 64)
	if err != nil {
		return false
	}
	return time.Now().Before(time.Unix(expUnix, 0))
}

func hmacSHA256(secret []byte, msg string) []byte {
	m := hmac.New(sha256.New, secret)
	_, _ = m.Write([]byte(msg))
	return m.Sum(nil)
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
    .row { display:flex; gap:12px; flex-wrap: wrap; }
    .btn { border:0; border-radius:8px; padding:8px 10px; background:#1d4ed8; color:#fff; cursor:pointer; }
    .btn.red { background:#b91c1c; }
    input, select { border:1px solid #cbd5e1; border-radius:8px; padding:8px; }
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
      <form method="post" action="/admin/ui/logout" style="margin-top:16px;">
        <button class="btn red" type="submit">Logout</button>
      </form>
    </nav>
    <main>
      <div class="muted">Server: {{.Hostname}}</div>
      <div id="content">
        {{if eq .Page "dashboard"}}{{template "dashboard_content" .Content}}{{end}}
        {{if eq .Page "config"}}{{template "config_content" .Content}}{{end}}
        {{if eq .Page "domains"}}{{template "domains_content" .Content}}{{end}}
        {{if eq .Page "health"}}{{template "health_content" .Content}}{{end}}
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
        if (window.location.pathname === '/admin/ui/health') {
          htmx.ajax('GET', '/admin/ui/health', '#content');
        }
      }
      es.addEventListener('stats_tick', refresh);
      es.addEventListener('tunnels_update', refresh);
      es.addEventListener('health_update', refresh);
      es.onerror = function(){};
    })();
    function toggleToken(btnId, fullId, shownId) {
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
  <form method="post" action="/admin/ui/login" style="max-width:360px; background:#fff; padding:16px; border-radius:10px;">
    <h3>Admin Login</h3>
    <p>Enter ADMIN token</p>
    <input type="password" name="token" style="width:100%; padding:8px;" />
    <button type="submit" style="margin-top:8px;">Login</button>
  </form>
</body>
</html>
{{end}}

{{define "dashboard_content"}}
<div class="card">
  <h2>Active Tunnels: {{.ActiveCount}}</h2>
  <div id="tunnels-table">{{template "tunnels_table" .Stats}}</div>
</div>
{{end}}

{{define "tunnels_table"}}
<table>
  <thead>
    <tr><th>Hostname</th><th>Active</th><th>Req</th><th>Err</th><th>In</th><th>Out</th><th>Last</th><th>Avg ms</th><th>Action</th></tr>
  </thead>
  <tbody>
  {{range .}}
    <tr>
      <td>{{.Hostname}}</td>
      <td>{{if .Active}}yes{{else}}no{{end}}</td>
      <td>{{.Requests}}</td>
      <td>{{.Errors}}</td>
      <td>{{.BytesIn}}</td>
      <td>{{.BytesOut}}</td>
      <td>{{.LastStatus}}</td>
      <td>{{.LatencyAvgMs}}</td>
      <td>
        {{if .Active}}
        <form hx-post="/admin/ui/tunnels/disconnect" hx-target="#tunnels-table" hx-swap="innerHTML" onsubmit="return confirm('Disconnect {{.Hostname}}?')">
          <input type="hidden" name="hostname" value="{{.Hostname}}" />
          <button class="btn red" type="submit">Disconnect</button>
        </form>
        {{else}}-{{end}}
      </td>
    </tr>
  {{else}}
    <tr><td colspan="9" class="muted">No tunnel stats yet.</td></tr>
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
  <h3>Client Token (non-admin)</h3>
  <p id="client-token-shown" data-masked="{{.MaskedToken}}" data-revealed="0">{{.MaskedToken}}</p>
  <input id="client-token-full" type="hidden" value="{{.ClientToken}}" />
  <button type="button" class="btn" onclick="toggleToken('toggle-btn','client-token-full','client-token-shown')">Reveal / Hide</button>
  <button type="button" class="btn" onclick="copyToken('client-token-full')">Copy</button>
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
`

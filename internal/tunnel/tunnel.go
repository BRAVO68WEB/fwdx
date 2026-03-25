package tunnel

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/BRAVO68WEB/fwdx/internal/config"
	"github.com/mitchellh/go-homedir"
)

var (
	errClientConfigRequired = errors.New("client config required: set FWDX_SERVER or ~/.fwdx/client.json")
	errLoginRequired        = errors.New("login required: run 'fwdx login'")
)

type Tunnel struct {
	TunnelID      string    `json:"tunnel_id,omitempty"`
	Name          string    `json:"name"`
	AccountID     string    `json:"account_id,omitempty"`
	Hostname      string    `json:"hostname"`
	Local         string    `json:"local"`
	AssignedAgent string    `json:"assigned_agent,omitempty"`
	DesiredState  string    `json:"desired_state,omitempty"`
	ActualState   string    `json:"actual_state,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Running       bool      `json:"running,omitempty"`
	PID           int       `json:"pid,omitempty"`
	ConfigPath    string    `json:"config_path,omitempty"`
}

type Manager struct {
	tunnelsDir string
}

type apiTunnel struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Hostname        string    `json:"hostname"`
	LocalHint       string    `json:"local_target_hint"`
	AssignedAgentID int64     `json:"assigned_agent_id"`
	AssignedAgent   string    `json:"assigned_agent"`
	DesiredState    string    `json:"desired_state"`
	ActualState     string    `json:"actual_state"`
	LastError       string    `json:"last_error"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type apiAgentCreateResponse struct {
	Agent struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"agent"`
	Credential string `json:"credential"`
}

func NewManager() *Manager {
	home, _ := homedir.Dir()
	return &Manager{tunnelsDir: filepath.Join(home, ".fwdx", "tunnels")}
}

func (m *Manager) Create(local, subdomain, customURL string, customName string) (*Tunnel, error) {
	cfg, sess, base, err := m.loadControlPlaneContext()
	if err != nil {
		return nil, err
	}
	agentName, err := m.ensureAgentCredential(cfg, sess, base)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(customName)
	if name == "" {
		if subdomain != "" {
			name = subdomain + "-tunnel"
		} else {
			name = strings.ReplaceAll(strings.TrimSpace(customURL), ".", "-") + "-tunnel"
		}
	}
	body, _ := json.Marshal(map[string]any{
		"name":       name,
		"subdomain":  subdomain,
		"url":        customURL,
		"local":      local,
		"agent_name": agentName,
	})
	var rec apiTunnel
	if err := apiJSON(base, sess.AccessToken, http.MethodPost, "/api/tunnels", bytes.NewReader(body), &rec, http.StatusCreated); err != nil {
		return nil, err
	}
	return m.fromAPI(rec), nil
}

func (m *Manager) Get(name string) (*Tunnel, error) {
	_, sess, base, err := m.loadControlPlaneContext()
	if err != nil {
		return nil, err
	}
	var rec apiTunnel
	if err := apiJSON(base, sess.AccessToken, http.MethodGet, "/api/tunnels/"+url.PathEscape(strings.ToLower(name)), nil, &rec, http.StatusOK); err != nil {
		return nil, err
	}
	return m.fromAPI(rec), nil
}

func (m *Manager) List() ([]*Tunnel, error) {
	_, sess, base, err := m.loadControlPlaneContext()
	if err != nil {
		return nil, err
	}
	var list []apiTunnel
	if err := apiJSON(base, sess.AccessToken, http.MethodGet, "/api/tunnels", nil, &list, http.StatusOK); err != nil {
		return nil, err
	}
	out := make([]*Tunnel, 0, len(list))
	for _, rec := range list {
		out = append(out, m.fromAPI(rec))
	}
	return out, nil
}

func (m *Manager) Start(name string, debug bool) error {
	t, err := m.Get(name)
	if err != nil {
		return err
	}
	cfg, sess, base, err := m.loadControlPlaneContext()
	if err != nil {
		return err
	}
	_, err = m.ensureAgentCredential(cfg, sess, base)
	if err != nil {
		return err
	}
	if _, ok := runtimeStateIfRunning(name); ok {
		return fmt.Errorf("tunnel %s is already running", name)
	}
	if _, err := readRuntimeState(name); err == nil {
		log.Printf("[fwdx] removed stale runtime state for tunnel=%s", name)
		removeRuntimeState(name)
	}
	localURL := normalizeLocalURL(t.Local)
	tunnelURL := cfg.TunnelURL()
	log.Printf("[fwdx] connecting tunnel=%s hostname=%s local=%s server=%s", name, t.Hostname, localURL, tunnelURL)
	return Connect(context.Background(), tunnelURL, cfg.AgentToken, t.Name, localURL, debug)
}

func (m *Manager) StartDetached(name string, debug bool) (*RuntimeState, error) {
	t, err := m.Get(name)
	if err != nil {
		return nil, err
	}
	cfg, sess, base, err := m.loadControlPlaneContext()
	if err != nil {
		return nil, err
	}
	_, err = m.ensureAgentCredential(cfg, sess, base)
	if err != nil {
		return nil, err
	}
	if _, ok := runtimeStateIfRunning(name); ok {
		return nil, fmt.Errorf("tunnel %s is already running", name)
	}
	if _, err := readRuntimeState(name); err == nil {
		log.Printf("[fwdx] removed stale runtime state for tunnel=%s before detached start", name)
		removeRuntimeState(name)
	}
	_ = os.MkdirAll(runtimeDir(), 0755)
	logPath := runtimeLogPath(name)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	defer logFile.Close()
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	args := []string{"tunnel", "start", name, "--watch"}
	if debug {
		args = append(args, "--debug")
	}
	cmd := exec.Command(exe, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	st := &RuntimeState{Name: name, Hostname: t.Hostname, Local: t.Local, PID: cmd.Process.Pid, LogPath: logPath, StartedAt: time.Now()}
	if err := writeRuntimeState(st); err != nil {
		_ = cmd.Process.Kill()
		return nil, err
	}
	return st, nil
}

func (m *Manager) Stop(name string) error {
	st, ok := runtimeStateIfRunning(name)
	if !ok {
		removeRuntimeState(name)
		return fmt.Errorf("tunnel %s is not running", name)
	}
	if err := stopPID(st.PID, 5*time.Second); err != nil {
		return err
	}
	removeRuntimeState(name)
	if _, sess, base, err := m.loadControlPlaneContext(); err == nil {
		_ = apiJSON(base, sess.AccessToken, http.MethodPost, "/api/tunnels/"+url.PathEscape(strings.ToLower(name))+"/stop", bytes.NewReader([]byte("{}")), nil, http.StatusOK)
	}
	return nil
}

func (m *Manager) RuntimeState(name string) (*RuntimeState, bool) {
	return runtimeStateIfRunning(name)
}

func (m *Manager) TailLogs(name string, w io.Writer, lines int, follow bool) error {
	if lines <= 0 {
		lines = 100
	}
	st, ok := runtimeStateIfRunning(name)
	logPath := runtimeLogPath(name)
	if ok && st.LogPath != "" {
		logPath = st.LogPath
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		return fmt.Errorf("read logs: %w", err)
	}
	all := strings.Split(string(data), "\n")
	if len(all) > 0 && all[len(all)-1] == "" {
		all = all[:len(all)-1]
	}
	start := 0
	if len(all) > lines {
		start = len(all) - lines
	}
	_, _ = io.WriteString(w, strings.Join(all[start:], "\n"))
	if !follow {
		return nil
	}
	f, err := os.Open(logPath)
	if err != nil {
		return err
	}
	defer f.Close()
	pos, _ := f.Seek(0, io.SeekEnd)
	buf := make([]byte, 4096)
	for {
		n, err := f.ReadAt(buf, pos)
		if n > 0 {
			_, _ = w.Write(buf[:n])
			pos += int64(n)
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		time.Sleep(250 * time.Millisecond)
		if _, running := runtimeStateIfRunning(name); !running {
			return nil
		}
	}
}

func (m *Manager) Delete(name string, force bool) error {
	if _, ok := runtimeStateIfRunning(name); ok {
		if !force {
			return fmt.Errorf("tunnel %s is running; stop it first or use --force", name)
		}
		if err := m.Stop(name); err != nil {
			return err
		}
	}
	_, sess, base, err := m.loadControlPlaneContext()
	if err != nil {
		return err
	}
	return apiJSON(base, sess.AccessToken, http.MethodDelete, "/api/tunnels/"+url.PathEscape(strings.ToLower(name)), nil, nil, http.StatusNoContent)
}

func (m *Manager) fromAPI(rec apiTunnel) *Tunnel {
	t := &Tunnel{
		Name:          rec.Name,
		Hostname:      rec.Hostname,
		Local:         rec.LocalHint,
		AssignedAgent: rec.AssignedAgent,
		DesiredState:  rec.DesiredState,
		ActualState:   rec.ActualState,
		CreatedAt:     rec.CreatedAt,
		UpdatedAt:     rec.UpdatedAt,
	}
	if st, ok := runtimeStateIfRunning(t.Name); ok {
		t.Running = true
		t.PID = st.PID
	}
	return t
}

func (m *Manager) loadControlPlaneContext() (*config.ClientConfig, *config.AuthSession, *url.URL, error) {
	cfg, err := config.LoadClientConfig()
	if err != nil || cfg == nil {
		cfg = &config.ClientConfig{}
	}
	sess, err := config.LoadAuthSession()
	if err != nil || sess == nil || strings.TrimSpace(sess.AccessToken) == "" {
		return nil, nil, nil, errLoginRequired
	}
	if strings.TrimSpace(cfg.ServerURL) == "" {
		cfg.ServerURL = strings.TrimSpace(sess.ServerURL)
		_ = config.SaveClientConfig(cfg)
	}
	if strings.TrimSpace(cfg.ServerURL) == "" {
		return nil, nil, nil, errClientConfigRequired
	}
	base, err := url.Parse(strings.TrimSuffix(cfg.ServerURL, "/"))
	if err != nil {
		return nil, nil, nil, err
	}
	return cfg, sess, base, nil
}

func (m *Manager) ensureAgentCredential(cfg *config.ClientConfig, sess *config.AuthSession, base *url.URL) (string, error) {
	if strings.TrimSpace(cfg.AgentName) != "" && strings.TrimSpace(cfg.AgentToken) != "" {
		return cfg.AgentName, nil
	}
	host, _ := os.Hostname()
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.ReplaceAll(host, ".", "-")
	host = strings.ReplaceAll(host, " ", "-")
	if host == "" {
		host = "local"
	}
	if cfg.AgentName == "" {
		rand, _ := randomString(4)
		cfg.AgentName = fmt.Sprintf("%s-%s", host, strings.ToLower(rand))
	}
	body, _ := json.Marshal(map[string]string{"name": cfg.AgentName})
	var out apiAgentCreateResponse
	if err := apiJSON(base, sess.AccessToken, http.MethodPost, "/api/agents", bytes.NewReader(body), &out, http.StatusCreated); err != nil {
		return "", err
	}
	cfg.AgentName = out.Agent.Name
	cfg.AgentToken = out.Credential
	if err := config.SaveClientConfig(cfg); err != nil {
		return "", err
	}
	return cfg.AgentName, nil
}

func apiJSON(base *url.URL, accessToken, method, path string, body io.Reader, out any, wantStatus int) error {
	if body == nil && (method == http.MethodPost || method == http.MethodPatch) {
		body = bytes.NewReader([]byte("{}"))
	}
	req, _ := http.NewRequest(method, base.ResolveReference(&url.URL{Path: path}).String(), body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(accessToken) != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		msg := strings.TrimSpace(string(data))
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("%s %s: %s", method, path, msg)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func normalizeLocalURL(local string) string {
	local = strings.TrimSpace(local)
	if strings.HasPrefix(local, "http://") || strings.HasPrefix(local, "https://") {
		return local
	}
	return "http://" + local
}

func randomString(n int) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i, b := range buf {
		out[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(out), nil
}

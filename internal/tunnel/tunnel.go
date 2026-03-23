package tunnel

// Client-side tunnel definitions and lifecycle. Tunnel config (name, hostname, local URL)
// is stored only on the client in ~/.fwdx/tunnels/<name>.json. The server does not
// persist tunnel definitions; it only holds active connections in memory (see
// internal/server/registry.go). When you run "fwdx tunnel start", the client
// connects over gRPC and keeps a bidirectional stream; the server's Registry
// maps hostname -> that connection until the client disconnects.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/BRAVO68WEB/fwdx/internal/config"
	"github.com/mitchellh/go-homedir"
)

var (
	errClientConfigRequired   = errors.New("client config required: set FWDX_SERVER and FWDX_TOKEN or ~/.fwdx/client.json")
	errServerHostnameRequired = errors.New("server hostname required for subdomain (set FWDX_SERVER or server_hostname in client.json)")
)

type Tunnel struct {
	TunnelID   string    `json:"tunnel_id,omitempty"`
	Name       string    `json:"name"`
	AccountID  string    `json:"account_id,omitempty"`
	Hostname   string    `json:"hostname"`
	Local      string    `json:"local"`
	Private    bool      `json:"private"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Running    bool      `json:"running,omitempty"`
	PID        int       `json:"pid,omitempty"`
	ConfigPath string    `json:"config_path,omitempty"`
}

type Manager struct {
	tunnelsDir string
}

func NewManager() *Manager {
	home, _ := homedir.Dir()
	return &Manager{
		tunnelsDir: filepath.Join(home, ".fwdx", "tunnels"),
	}
}

func (m *Manager) Create(local, subdomain, customURL string, private bool, customName string) (*Tunnel, error) {
	cfg, err := config.LoadClientConfig()
	if err != nil || cfg == nil {
		return nil, errClientConfigRequired
	}
	if cfg.ServerURL == "" || cfg.Token == "" {
		return nil, errClientConfigRequired
	}

	hostname := customURL
	if subdomain != "" {
		if cfg.ServerHostname == "" {
			return nil, errServerHostnameRequired
		}
		hostname = subdomain + "." + cfg.ServerHostname
	} else if customURL == "" {
		return nil, errors.New("either --subdomain or --url is required")
	}

	tunnelName := customName
	if tunnelName == "" {
		if subdomain != "" {
			tunnelName = subdomain + "-tunnel"
		} else {
			tunnelName = strings.ReplaceAll(strings.TrimPrefix(hostname, "."), ".", "-") + "-tunnel"
		}
	}

	if err := os.MkdirAll(m.tunnelsDir, 0755); err != nil {
		return nil, err
	}

	configPath := filepath.Join(m.tunnelsDir, tunnelName+".json")
	t := &Tunnel{
		Name:       tunnelName,
		Hostname:   strings.ToLower(hostname),
		Local:      local,
		Private:    private,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ConfigPath: configPath,
	}

	if err := m.saveTunnel(t); err != nil {
		return nil, err
	}
	return t, nil
}

func (m *Manager) saveTunnel(t *Tunnel) error {
	data, _ := json.MarshalIndent(t, "", "  ")
	return os.WriteFile(t.ConfigPath, data, 0600)
}

func (m *Manager) updateTunnel(t *Tunnel) error {
	return m.saveTunnel(t)
}

func (m *Manager) Get(name string) (*Tunnel, error) {
	configPath := filepath.Join(m.tunnelsDir, name+".json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var t Tunnel
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	t.ConfigPath = configPath
	if st, ok := runtimeStateIfRunning(name); ok {
		t.Running = true
		t.PID = st.PID
	}
	return &t, nil
}

func (m *Manager) List() ([]*Tunnel, error) {
	if err := os.MkdirAll(m.tunnelsDir, 0755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(m.tunnelsDir)
	if err != nil {
		return nil, err
	}
	var tunnels []*Tunnel
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".json")
		t, err := m.Get(name)
		if err != nil {
			continue
		}
		if t.Hostname == "" {
			continue
		}
		if st, ok := runtimeStateIfRunning(t.Name); ok {
			t.Running = true
			t.PID = st.PID
		}
		tunnels = append(tunnels, t)
	}
	return tunnels, nil
}

func (m *Manager) Start(name string, debug bool) error {
	t, err := m.Get(name)
	if err != nil {
		return err
	}
	cfg, err := config.LoadClientConfig()
	if err != nil || cfg == nil || cfg.ServerURL == "" || cfg.Token == "" {
		return errClientConfigRequired
	}

	if _, ok := runtimeStateIfRunning(name); ok {
		return fmt.Errorf("tunnel %s is already running", name)
	}

	tunnelURL := cfg.TunnelURL()
	localURL := "http://" + t.Local
	ctx := context.Background()
	return Connect(ctx, tunnelURL, cfg.Token, t.Hostname, localURL, debug)
}

func (m *Manager) StartDetached(name string, debug bool) (*RuntimeState, error) {
	t, err := m.Get(name)
	if err != nil {
		return nil, err
	}
	cfg, err := config.LoadClientConfig()
	if err != nil || cfg == nil || cfg.ServerURL == "" || cfg.Token == "" {
		return nil, errClientConfigRequired
	}
	if _, ok := runtimeStateIfRunning(name); ok {
		return nil, fmt.Errorf("tunnel %s is already running", name)
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

	st := &RuntimeState{
		Name:      name,
		Hostname:  t.Hostname,
		Local:     t.Local,
		PID:       cmd.Process.Pid,
		LogPath:   logPath,
		StartedAt: time.Now(),
	}
	if err := writeRuntimeState(st); err != nil {
		_ = cmd.Process.Kill()
		return nil, err
	}
	return st, nil
}

func (m *Manager) Stop(name string) error {
	st, ok := runtimeStateIfRunning(name)
	if !ok {
		// stale or missing state
		removeRuntimeState(name)
		return fmt.Errorf("tunnel %s is not running", name)
	}
	if err := stopPID(st.PID, 5*time.Second); err != nil {
		return err
	}
	removeRuntimeState(name)
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
	t, err := m.Get(name)
	if err != nil {
		return err
	}
	if _, ok := runtimeStateIfRunning(name); ok {
		if !force {
			return fmt.Errorf("tunnel %s is running; stop it first or use --force", name)
		}
		if err := m.Stop(name); err != nil {
			return err
		}
	}
	removeRuntimeState(name)
	_ = os.Remove(t.ConfigPath)
	return nil
}

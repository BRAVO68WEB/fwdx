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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/BRAVO68WEB/fwdx/internal/config"
	"github.com/mitchellh/go-homedir"
)

var (
	errClientConfigRequired = errors.New("client config required: set FWDX_SERVER and FWDX_TOKEN or ~/.fwdx/client.json")
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
	mu         sync.Mutex
	running    map[string]context.CancelFunc
}

func NewManager() *Manager {
	home, _ := homedir.Dir()
	return &Manager{
		tunnelsDir: filepath.Join(home, ".fwdx", "tunnels"),
		running:    make(map[string]context.CancelFunc),
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
		m.mu.Lock()
		_, running := m.running[t.Name]
		m.mu.Unlock()
		t.Running = running
		tunnels = append(tunnels, t)
	}
	return tunnels, nil
}

func (m *Manager) Start(name string, watch, debug bool) error {
	t, err := m.Get(name)
	if err != nil {
		return err
	}
	cfg, err := config.LoadClientConfig()
	if err != nil || cfg == nil || cfg.ServerURL == "" || cfg.Token == "" {
		return errClientConfigRequired
	}

	m.mu.Lock()
	if _, ok := m.running[name]; ok {
		m.mu.Unlock()
		return fmt.Errorf("tunnel %s is already running", name)
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.running[name] = cancel
	m.mu.Unlock()

	tunnelURL := cfg.TunnelURL()
	localURL := "http://" + t.Local

	if watch || debug {
		defer func() {
			m.mu.Lock()
			delete(m.running, name)
			m.mu.Unlock()
			cancel()
		}()
		return Connect(ctx, tunnelURL, cfg.Token, t.Hostname, localURL, debug)
	}

	go func() {
		_ = Connect(ctx, tunnelURL, cfg.Token, t.Hostname, localURL, debug)
		m.mu.Lock()
		delete(m.running, name)
		m.mu.Unlock()
		cancel()
	}()
	return nil
}

func (m *Manager) Stop(name string) error {
	m.mu.Lock()
	cancel, ok := m.running[name]
	if ok {
		delete(m.running, name)
		cancel()
	}
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("tunnel %s is not running", name)
	}
	return nil
}

func (m *Manager) Delete(name string, force bool) error {
	t, err := m.Get(name)
	if err != nil {
		return err
	}
	m.mu.Lock()
	if cancel, ok := m.running[name]; ok {
		delete(m.running, name)
		cancel()
	}
	m.mu.Unlock()
	_ = os.Remove(t.ConfigPath)
	return nil
}

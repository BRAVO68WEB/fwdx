package config

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ClientConfig holds the fwdx client configuration (server URL, token).
type ClientConfig struct {
	ServerURL       string `json:"server_url"`
	Token           string `json:"token"`
	ServerHostname  string `json:"server_hostname,omitempty"` // optional; derived from ServerURL if empty
	TunnelPort      int    `json:"tunnel_port,omitempty"`      // optional; default 4443
}

const clientConfigFile = "client.json"

// LoadClientConfig loads client config from ~/.fwdx/client.json and env (env overrides).
func LoadClientConfig() (*ClientConfig, error) {
	path := filepath.Join(GetConfigDir(), clientConfigFile)
	data, err := os.ReadFile(path)
	cfg := &ClientConfig{TunnelPort: 4443}
	if err == nil {
		_ = json.Unmarshal(data, cfg)
	}
	if cfg.ServerURL == "" {
		cfg.ServerURL = os.Getenv("FWDX_SERVER")
	}
	if cfg.Token == "" {
		cfg.Token = os.Getenv("FWDX_TOKEN")
	}
	if cfg.ServerHostname == "" && cfg.ServerURL != "" {
		u, err := url.Parse(cfg.ServerURL)
		if err == nil {
			cfg.ServerHostname = u.Hostname()
		}
	}
	return cfg, nil
}

// SaveClientConfig writes client config to ~/.fwdx/client.json.
func SaveClientConfig(cfg *ClientConfig) error {
	dir := GetConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, clientConfigFile), data, 0600)
}

// TunnelURL returns the URL clients use to connect for tunnel registration (host:tunnel_port).
func (c *ClientConfig) TunnelURL() string {
	u, err := url.Parse(c.ServerURL)
	if err != nil || u.Host == "" {
		return strings.TrimSuffix(c.ServerURL, "/")
	}
	host := u.Hostname()
	port := c.TunnelPort
	if port == 0 {
		port = 4443
	}
	u.Host = host + ":" + strconv.Itoa(port)
	// Keep scheme
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	return u.String()
}

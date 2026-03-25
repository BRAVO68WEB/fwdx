package config

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ClientConfig holds local client runtime config.
type ClientConfig struct {
	ServerURL      string `json:"server_url"`
	ServerHostname string `json:"server_hostname,omitempty"` // optional; derived from ServerURL if empty
	TunnelPort     int    `json:"tunnel_port,omitempty"`     // optional; default 4443
	AgentName      string `json:"agent_name,omitempty"`
	AgentToken     string `json:"agent_token,omitempty"`
}

const clientConfigFile = "client.json"

// LoadClientConfig loads client config from ~/.fwdx/client.json and env (env overrides).
func LoadClientConfig() (*ClientConfig, error) {
	path := filepath.Join(GetConfigDir(), clientConfigFile)
	data, err := os.ReadFile(path)
	cfg := &ClientConfig{TunnelPort: 0} // 0 = use port from ServerURL (e.g. 443 when behind nginx)
	if err == nil {
		_ = json.Unmarshal(data, cfg)
	}
	if cfg.ServerURL == "" {
		cfg.ServerURL = os.Getenv("FWDX_SERVER")
	}
	if v := os.Getenv("FWDX_TUNNEL_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.TunnelPort = p // 0 = use port from ServerURL (e.g. 443 behind nginx)
		}
	}
	if cfg.ServerHostname == "" && cfg.ServerURL != "" {
		u, err := url.Parse(cfg.ServerURL)
		if err == nil {
			cfg.ServerHostname = u.Hostname()
		}
	}
	if cfg.AgentName == "" {
		cfg.AgentName = os.Getenv("FWDX_AGENT_NAME")
	}
	if cfg.AgentToken == "" {
		cfg.AgentToken = os.Getenv("FWDX_AGENT_TOKEN")
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

// TunnelURL returns the URL clients use to connect for the gRPC tunnel.
// When TunnelPort is 0: if ServerURL has an explicit port, that port is used; otherwise
// the tunnel port defaults to 4443 (gRPC). This way FWDX_SERVER=https://tunnel.example.com
// (port 443) automatically uses 4443 for the tunnel, which nginx forwards to the server's grpc port.
func (c *ClientConfig) TunnelURL() string {
	u, err := url.Parse(c.ServerURL)
	if err != nil || u.Host == "" {
		return strings.TrimSuffix(c.ServerURL, "/")
	}
	host := u.Hostname()
	port := c.TunnelPort
	if port == 0 {
		if u.Port() != "" {
			if p, err := strconv.Atoi(u.Port()); err == nil {
				port = p
			}
		}
		if port == 0 {
			// Server URL has no port: default 443 for https, 4443 for http
			if u.Scheme == "https" {
				port = 443
			} else {
				port = 4443
			}
		}
		// When public URL is 443 (HTTPS), tunnel is always on 4443 (gRPC) behind nginx
		if port == 443 && u.Scheme == "https" {
			port = 4443
		}
	}
	u.Host = host + ":" + strconv.Itoa(port)
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	return u.String()
}

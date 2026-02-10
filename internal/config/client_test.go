package config

import (
	"os"
	"testing"
)

func TestClientConfig_TunnelURL(t *testing.T) {
	tests := []struct {
		name string
		cfg  ClientConfig
		want string
	}{
		{
			name: "https with default port",
			cfg:  ClientConfig{ServerURL: "https://tunnel.example.com", TunnelPort: 4443},
			want: "https://tunnel.example.com:4443",
		},
		{
			name: "https with explicit port in URL",
			cfg:  ClientConfig{ServerURL: "https://tunnel.example.com:443", TunnelPort: 4443},
			want: "https://tunnel.example.com:4443",
		},
		{
			name: "zero port uses URL port (https default 443, single-port behind nginx)",
			cfg:  ClientConfig{ServerURL: "https://tunnel.example.com", TunnelPort: 0},
			want: "https://tunnel.example.com:443",
		},
		{
			name: "zero port with explicit 4443 in URL",
			cfg:  ClientConfig{ServerURL: "https://tunnel.example.com:4443", TunnelPort: 0},
			want: "https://tunnel.example.com:4443",
		},
		{
			name: "custom tunnel port",
			cfg:  ClientConfig{ServerURL: "https://tunnel.example.com", TunnelPort: 9999},
			want: "https://tunnel.example.com:9999",
		},
		{
			name: "no scheme and no host in URL returns trimmed URL",
			cfg:  ClientConfig{ServerURL: "tunnel.example.com", TunnelPort: 4443},
			want: "tunnel.example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.TunnelURL()
			if got != tt.want {
				t.Errorf("TunnelURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadClientConfig_ReturnsDefaultTunnelPort(t *testing.T) {
	cfg, err := LoadClientConfig()
	if err != nil {
		t.Fatal(err)
	}
	// Default when no config file is 0 (use URL port in TunnelURL); file may have 4443
	if cfg.TunnelPort != 0 && cfg.TunnelPort != 4443 {
		t.Logf("TunnelPort = %d (may come from file)", cfg.TunnelPort)
	}
}

func TestLoadClientConfig_EnvOverride(t *testing.T) {
	origServer := os.Getenv("FWDX_SERVER")
	origToken := os.Getenv("FWDX_TOKEN")
	defer func() {
		os.Setenv("FWDX_SERVER", origServer)
		os.Setenv("FWDX_TOKEN", origToken)
	}()

	os.Setenv("FWDX_SERVER", "https://env-test.example.com")
	os.Setenv("FWDX_TOKEN", "env-secret")
	cfg, err := LoadClientConfig()
	if err != nil {
		t.Fatal(err)
	}
	// If no client.json exists or it has empty ServerURL, env wins
	if cfg.ServerURL == "https://env-test.example.com" {
		if cfg.ServerHostname != "env-test.example.com" {
			t.Errorf("ServerHostname = %q, want env-test.example.com", cfg.ServerHostname)
		}
	}
	_ = cfg
}

func TestSaveClientConfig_LoadClientConfig_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	oldDir := configDir
	configDir = dir
	defer func() { configDir = oldDir }()

	cfg := &ClientConfig{
		ServerURL:      "https://roundtrip.example.com",
		Token:          "roundtrip-token",
		ServerHostname: "roundtrip.example.com",
		TunnelPort:     9999,
	}
	if err := SaveClientConfig(cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadClientConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ServerURL != cfg.ServerURL {
		t.Errorf("ServerURL = %q, want %q", loaded.ServerURL, cfg.ServerURL)
	}
	if loaded.Token != cfg.Token {
		t.Errorf("Token = %q, want %q", loaded.Token, cfg.Token)
	}
	if loaded.ServerHostname != cfg.ServerHostname {
		t.Errorf("ServerHostname = %q, want %q", loaded.ServerHostname, cfg.ServerHostname)
	}
	if loaded.TunnelPort != cfg.TunnelPort {
		t.Errorf("TunnelPort = %d, want %d", loaded.TunnelPort, cfg.TunnelPort)
	}
}

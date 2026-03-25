package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type AuthSession struct {
	ServerURL   string    `json:"server_url"`
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
	Subject     string    `json:"subject"`
	Email       string    `json:"email,omitempty"`
	DisplayName string    `json:"display_name,omitempty"`
	Role        string    `json:"role,omitempty"`
}

const authConfigFile = "auth.json"

func AuthConfigPath() string {
	return filepath.Join(GetConfigDir(), authConfigFile)
}

func LoadAuthSession() (*AuthSession, error) {
	data, err := os.ReadFile(AuthConfigPath())
	if err != nil {
		return nil, err
	}
	var sess AuthSession
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func SaveAuthSession(sess *AuthSession) error {
	if err := os.MkdirAll(GetConfigDir(), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(AuthConfigPath(), data, 0600)
}

func DeleteAuthSession() error {
	if err := os.Remove(AuthConfigPath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

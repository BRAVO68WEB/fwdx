package config

import (
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

var configDir string

func init() {
	home, _ := homedir.Dir()
	configDir = filepath.Join(home, ".fwdx")
}

// GetConfigDir returns the fwdx config directory
func GetConfigDir() string {
	return configDir
}

// TunnelsDir returns the directory where tunnel configs are stored
func TunnelsDir() string {
	return filepath.Join(configDir, "tunnels")
}

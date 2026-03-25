package fwdx

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/BRAVO68WEB/fwdx/internal/config"
)

func resolveServerURL(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit), nil
	}
	if env := strings.TrimSpace(os.Getenv("FWDX_SERVER")); env != "" {
		return env, nil
	}
	cfg, err := config.LoadClientConfig()
	if err == nil && strings.TrimSpace(cfg.ServerURL) != "" {
		return strings.TrimSpace(cfg.ServerURL), nil
	}
	return "", fmt.Errorf("server is required (--server or FWDX_SERVER)")
}

func resolveServerBase(explicit string) (*url.URL, error) {
	serverURL, err := resolveServerURL(explicit)
	if err != nil {
		return nil, err
	}
	base, err := url.Parse(strings.TrimSuffix(serverURL, "/"))
	if err != nil {
		return nil, err
	}
	return base, nil
}

func requireAuthSession() (*config.AuthSession, error) {
	sess, err := config.LoadAuthSession()
	if err != nil {
		return nil, fmt.Errorf("not logged in: run 'fwdx login'")
	}
	if sess.AccessToken == "" {
		return nil, fmt.Errorf("not logged in: run 'fwdx login'")
	}
	return sess, nil
}

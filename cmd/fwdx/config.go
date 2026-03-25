package fwdx

import (
	"fmt"

	"github.com/BRAVO68WEB/fwdx/internal/config"
	"github.com/BRAVO68WEB/fwdx/pkg/output"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show fwdx client configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleConfig()
	},
}

func handleConfig() error {
	cfg, err := config.LoadClientConfig()
	if err != nil {
		return output.PrintError("Failed to load config: " + err.Error())
	}

	output.PrintInfo("fwdx client configuration")
	fmt.Println()
	fmt.Printf("  Config dir:    %s\n", config.GetConfigDir())
	fmt.Printf("  Server URL:    %s\n", cfg.ServerURL)
	if cfg.AgentName != "" {
		fmt.Printf("  Agent Name:    %s\n", cfg.AgentName)
	} else {
		fmt.Printf("  Agent Name:    (not provisioned)\n")
	}
	if cfg.AgentToken != "" {
		fmt.Printf("  Agent Token:   %s...\n", maskToken(cfg.AgentToken))
	} else {
		fmt.Printf("  Agent Token:   (not set)\n")
	}
	fmt.Printf("  Server host:   %s\n", cfg.ServerHostname)
	fmt.Printf("  Tunnel port:   %d\n", cfg.TunnelPort)
	if auth, err := config.LoadAuthSession(); err == nil {
		fmt.Printf("  OIDC Subject:  %s\n", auth.Subject)
		fmt.Printf("  OIDC Email:    %s\n", auth.Email)
		fmt.Printf("  OIDC Role:     %s\n", auth.Role)
		fmt.Printf("  Auth Expires:  %s\n", auth.ExpiresAt.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("  OIDC Session:  (not logged in)\n")
	}
	return nil
}

func maskToken(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

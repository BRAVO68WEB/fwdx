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
	if cfg.Token != "" {
		fmt.Printf("  Token:         %s...\n", maskToken(cfg.Token))
	} else {
		fmt.Printf("  Token:         (not set)\n")
	}
	fmt.Printf("  Server host:   %s\n", cfg.ServerHostname)
	fmt.Printf("  Tunnel port:   %d\n", cfg.TunnelPort)
	return nil
}

func maskToken(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

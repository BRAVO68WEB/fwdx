package fwdx

import (
	"net/http"

	"github.com/BRAVO68WEB/fwdx/internal/config"
	"github.com/BRAVO68WEB/fwdx/pkg/output"
	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check client config and server connectivity",
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleHealth()
	},
}

func handleHealth() error {
	cfg, err := config.LoadClientConfig()
	if err != nil {
		return output.PrintError("Failed to load config: " + err.Error())
	}
	if cfg.ServerURL == "" || cfg.Token == "" {
		return output.PrintError("FWDX_SERVER and FWDX_TOKEN (or client.json) must be set")
	}

	resp, err := http.Head(cfg.ServerURL)
	if err != nil {
		return output.PrintError("Cannot reach server: " + err.Error())
	}
	resp.Body.Close()

	output.PrintSuccess("Client config OK and server reachable")
	return nil
}

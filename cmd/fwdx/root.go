package fwdx

import (
	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags "-X github.com/BRAVO68WEB/fwdx/cmd/fwdx.version=..."
var version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "fwdx",
	Short: "Self-hosted tunneling CLI and server",
	Long:  `fwdx runs a tunneling server (fwdx serve) and clients connect to expose local HTTP services by hostname.`,
	Version: version,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(manageCmd)
	rootCmd.AddCommand(domainsCmd)
	rootCmd.AddCommand(tunnelCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(healthCmd)
}

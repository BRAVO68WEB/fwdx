package fwdx

import (
	"fmt"
	"os"

	"github.com/BRAVO68WEB/fwdx/internal/tunnel"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs [tunnel-name]",
	Short: "Show detached tunnel logs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		follow, _ := cmd.Flags().GetBool("follow")
		lines, _ := cmd.Flags().GetInt("lines")
		manager := tunnel.NewManager()
		if err := manager.TailLogs(args[0], os.Stdout, lines, follow); err != nil {
			return fmt.Errorf("logs: %w", err)
		}
		return nil
	},
}

func init() {
	logsCmd.Flags().BoolP("follow", "f", false, "Follow logs until tunnel stops")
	logsCmd.Flags().IntP("lines", "n", 100, "Number of recent lines to show")
}

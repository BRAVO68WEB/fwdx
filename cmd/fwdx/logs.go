package fwdx

import (
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs [tunnel-name]",
	Short: "Stream tunnel logs",
	Long:  "Use 'fwdx tunnel start <name> --watch' to run a tunnel in the foreground and see logs.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Println("Use 'fwdx tunnel start <name> --watch' to stream tunnel logs.")
		return nil
	},
}

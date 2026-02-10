package fwdx

import (
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print fwdx version",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Println(rootCmd.Version)
		return nil
	},
}

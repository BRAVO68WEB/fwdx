package fwdx

import (
	"fmt"

	"github.com/BRAVO68WEB/fwdx/internal/config"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear local OIDC session",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.DeleteAuthSession(); err != nil {
			return err
		}
		fmt.Println("Logged out")
		return nil
	},
}

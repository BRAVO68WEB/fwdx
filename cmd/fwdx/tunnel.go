package fwdx

import (
	"fmt"

	"github.com/BRAVO68WEB/fwdx/internal/tunnel"
	"github.com/BRAVO68WEB/fwdx/pkg/output"
	"github.com/spf13/cobra"
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Manage tunnels",
}

var tunnelCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new tunnel",
	RunE: func(cmd *cobra.Command, args []string) error {
		local, _ := cmd.Flags().GetString("local")
		subdomain, _ := cmd.Flags().GetString("subdomain")
		url, _ := cmd.Flags().GetString("url")
		private, _ := cmd.Flags().GetBool("private")
		name, _ := cmd.Flags().GetString("name")

		if local == "" {
			return output.PrintError("--local is required")
		}
		if subdomain == "" && url == "" {
			return output.PrintError("Either --subdomain or --url is required")
		}
		if subdomain != "" && url != "" {
			return output.PrintError("Cannot use both --subdomain and --url")
		}

		return handleTunnelCreate(local, subdomain, url, private, name)
	},
}

var tunnelStartCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start a tunnel",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		watch, _ := cmd.Flags().GetBool("watch")
		debug, _ := cmd.Flags().GetBool("debug")
		return handleTunnelStart(args[0], watch, debug)
	},
}

var tunnelStopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop a tunnel",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleTunnelStop(args[0])
	},
}

var tunnelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tunnels",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		return handleTunnelList(format)
	},
}

var tunnelShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show tunnel details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleTunnelShow(args[0])
	},
}

var tunnelDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a tunnel",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		return handleTunnelDelete(args[0], force)
	},
}

func init() {
	// tunnel create flags
	tunnelCreateCmd.Flags().StringP("local", "l", "", "Local service address (e.g., localhost:5000)")
	tunnelCreateCmd.Flags().StringP("subdomain", "s", "", "Subdomain under root domain")
	tunnelCreateCmd.Flags().StringP("url", "u", "", "Custom domain")
	tunnelCreateCmd.Flags().BoolP("private", "p", false, "Private tunnel (reserved)")
	tunnelCreateCmd.Flags().String("name", "", "Custom tunnel name")

	// tunnel start flags
	tunnelStartCmd.Flags().BoolP("watch", "w", false, "Run in foreground and stream logs")
	tunnelStartCmd.Flags().BoolP("debug", "d", false, "Run in foreground with debug logs")

	// tunnel list flags
	tunnelListCmd.Flags().StringP("format", "f", "table", "Output format (table, json, yaml)")

	// tunnel delete flags
	tunnelDeleteCmd.Flags().BoolP("force", "f", false, "Force delete without confirmation")

	tunnelCmd.AddCommand(tunnelCreateCmd)
	tunnelCmd.AddCommand(tunnelStartCmd)
	tunnelCmd.AddCommand(tunnelStopCmd)
	tunnelCmd.AddCommand(tunnelListCmd)
	tunnelCmd.AddCommand(tunnelShowCmd)
	tunnelCmd.AddCommand(tunnelDeleteCmd)
}

func handleTunnelCreate(local, subdomain, url string, private bool, name string) error {
	manager := tunnel.NewManager()
	t, err := manager.Create(local, subdomain, url, private, name)
	if err != nil {
		return output.PrintError(fmt.Sprintf("Failed to create tunnel: %v", err))
	}

	output.PrintSuccess(fmt.Sprintf("✅ Tunnel created: %s", t.Name))
	fmt.Printf("   Hostname: https://%s\n", t.Hostname)
	fmt.Printf("   Local:    http://%s\n", t.Local)
	fmt.Printf("   Status:   Not running (use 'fwdx tunnel start %s' to start)\n", t.Name)

	return nil
}

func handleTunnelStart(name string, watch, debug bool) error {
	manager := tunnel.NewManager()
	err := manager.Start(name, watch, debug)
	if err != nil {
		return output.PrintError(fmt.Sprintf("Failed to start tunnel: %v", err))
	}

	// Only print success when running in background (not watch/debug foreground)
	if !watch && !debug {
		output.PrintSuccess(fmt.Sprintf("✅ Tunnel '%s' started", name))
	}
	return nil
}

func handleTunnelStop(name string) error {
	manager := tunnel.NewManager()
	err := manager.Stop(name)
	if err != nil {
		return output.PrintError(fmt.Sprintf("Failed to stop tunnel: %v", err))
	}

	output.PrintSuccess(fmt.Sprintf("✅ Tunnel '%s' stopped", name))
	return nil
}

func handleTunnelList(format string) error {
	manager := tunnel.NewManager()
	tunnels, err := manager.List()
	if err != nil {
		return output.PrintError(fmt.Sprintf("Failed to list tunnels: %v", err))
	}

	output.PrintTunnelList(tunnels, format)
	return nil
}

func handleTunnelShow(name string) error {
	manager := tunnel.NewManager()
	t, err := manager.Get(name)
	if err != nil {
		return output.PrintError(fmt.Sprintf("Tunnel not found: %v", err))
	}

	output.PrintTunnelDetails(t)
	return nil
}

func handleTunnelDelete(name string, force bool) error {
	manager := tunnel.NewManager()
	err := manager.Delete(name, force)
	if err != nil {
		return output.PrintError(fmt.Sprintf("Failed to delete tunnel: %v", err))
	}

	output.PrintSuccess(fmt.Sprintf("✅ Tunnel '%s' deleted", name))
	return nil
}

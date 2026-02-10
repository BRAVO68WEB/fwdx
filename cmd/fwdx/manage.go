package fwdx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var manageCmd = &cobra.Command{
	Use:   "manage",
	Short: "Remote management of the fwdx server",
}

var manageTunnelsCmd = &cobra.Command{
	Use:   "tunnels",
	Short: "List active tunnels",
	RunE:  runManageTunnels,
}

var manageDomainsCmd = &cobra.Command{
	Use:   "domains",
	Short: "Manage allowed domains",
}

var manageDomainsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List allowed domains",
	RunE:  runManageDomainsList,
}

func init() {
	manageCmd.PersistentFlags().String("server", "", "fwdx server URL (or FWDX_SERVER)")
	manageCmd.PersistentFlags().String("admin-token", "", "Admin token (or FWDX_ADMIN_TOKEN)")

	manageCmd.AddCommand(manageTunnelsCmd)
	manageCmd.AddCommand(manageDomainsCmd)
	manageDomainsCmd.AddCommand(manageDomainsListCmd)
}

func manageServerAndToken(cmd *cobra.Command) (base *url.URL, adminToken string, err error) {
	serverURL, _ := cmd.Flags().GetString("server")
	if serverURL == "" {
		serverURL = os.Getenv("FWDX_SERVER")
	}
	if serverURL == "" {
		return nil, "", fmt.Errorf("server is required (--server or FWDX_SERVER)")
	}
	adminToken, _ = cmd.Flags().GetString("admin-token")
	if adminToken == "" {
		adminToken = os.Getenv("FWDX_ADMIN_TOKEN")
	}
	if adminToken == "" {
		return nil, "", fmt.Errorf("admin-token is required (--admin-token or FWDX_ADMIN_TOKEN)")
	}
	base, err = url.Parse(strings.TrimSuffix(serverURL, "/"))
	if err != nil {
		return nil, "", err
	}
	return base, adminToken, nil
}

func runManageTunnels(cmd *cobra.Command, args []string) error {
	base, adminToken, err := manageServerAndToken(cmd)
	if err != nil {
		return err
	}
	u := base.ResolveReference(&url.URL{Path: "/admin/tunnels"}).String()
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("list tunnels: %s", resp.Status)
	}
	var list map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return err
	}
	if len(list) == 0 {
		fmt.Println("No active tunnels.")
		return nil
	}
	fmt.Println("Active tunnels:")
	for hostname, addr := range list {
		fmt.Printf("  %s -> %s\n", hostname, addr)
	}
	return nil
}

func runManageDomainsList(cmd *cobra.Command, args []string) error {
	base, adminToken, err := manageServerAndToken(cmd)
	if err != nil {
		return err
	}
	// manage domains list uses the same persistent flags from manageCmd
	u := base.ResolveReference(&url.URL{Path: "/admin/domains"}).String()
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("list domains: %s", resp.Status)
	}
	var list []string
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return err
	}
	if len(list) == 0 {
		fmt.Println("No allowed domains.")
		return nil
	}
	fmt.Println("Allowed domains:")
	for _, d := range list {
		fmt.Printf("  %s\n", d)
	}
	return nil
}

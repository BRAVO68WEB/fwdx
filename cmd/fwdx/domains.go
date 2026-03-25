package fwdx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

var domainsCmd = &cobra.Command{
	Use:   "domains",
	Short: "Manage allowed domains on the fwdx server",
}

var domainsAddCmd = &cobra.Command{
	Use:   "add [domain]",
	Short: "Add a domain to the allowed list and print DNS instructions",
	Args:  cobra.ExactArgs(1),
	RunE:  runDomainsAdd,
}

func init() {
	domainsCmd.AddCommand(domainsAddCmd)
	domainsAddCmd.Flags().String("server", "", "fwdx server URL (or FWDX_SERVER)")
}

func runDomainsAdd(cmd *cobra.Command, args []string) error {
	domain := strings.TrimSpace(strings.ToLower(args[0]))
	if domain == "" {
		return fmt.Errorf("domain is required")
	}

	serverURL, _ := cmd.Flags().GetString("server")
	base, err := resolveServerBase(serverURL)
	if err != nil {
		return err
	}
	sess, err := requireAuthSession()
	if err != nil {
		return err
	}

	// POST /admin/domains
	domainsURL := base.ResolveReference(&url.URL{Path: "/admin/domains"}).String()
	body, _ := json.Marshal(map[string]string{"domain": domain})
	req, _ := http.NewRequest(http.MethodPost, domainsURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+sess.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("add domain: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("add domain: %s", resp.Status)
	}

	// GET /admin/info for hostname
	infoURL := base.ResolveReference(&url.URL{Path: "/admin/info"}).String()
	req2, _ := http.NewRequest(http.MethodGet, infoURL, nil)
	req2.Header.Set("Authorization", "Bearer "+sess.AccessToken)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		return fmt.Errorf("get server info: %w", err)
	}
	defer resp2.Body.Close()
	var info struct {
		Hostname string `json:"hostname"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&info); err != nil {
		return fmt.Errorf("get server info: %w", err)
	}
	serverHostname := info.Hostname
	if serverHostname == "" {
		serverHostname = base.Hostname()
	}

	fmt.Printf("Added domain: %s\n\n", domain)
	fmt.Println("DNS setup:")
	fmt.Println("  If this is the first time setting up the server, create an A record:")
	fmt.Printf("    A  %s  <server-ip>\n", serverHostname)
	fmt.Println("  Then for your custom domain, create a wildcard CNAME:")
	fmt.Printf("    CNAME  *.%s  %s\n", domain, serverHostname)
	fmt.Println()
	fmt.Printf("  Replace <server-ip> with your server's public IP (e.g. from the machine: curl -s ifconfig.me).\n")

	return nil
}

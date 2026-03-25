package fwdx

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"net/http"
	"net/url"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the current authenticated identity",
	RunE:  runWhoAmI,
}

func runWhoAmI(cmd *cobra.Command, args []string) error {
	sess, err := requireAuthSession()
	if err != nil {
		return err
	}
	base, err := resolveServerBase(sess.ServerURL)
	if err != nil {
		return err
	}
	req, _ := http.NewRequest(http.MethodGet, base.ResolveReference(&url.URL{Path: "/api/users/me"}).String(), nil)
	req.Header.Set("Authorization", "Bearer "+sess.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("whoami: %s", resp.Status)
	}
	var out struct {
		Subject     string `json:"subject"`
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		Role        string `json:"role"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	fmt.Printf("Subject: %s\n", out.Subject)
	fmt.Printf("Email: %s\n", out.Email)
	fmt.Printf("Name: %s\n", out.DisplayName)
	fmt.Printf("Role: %s\n", out.Role)
	return nil
}

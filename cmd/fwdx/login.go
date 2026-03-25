package fwdx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/BRAVO68WEB/fwdx/internal/config"
	"github.com/BRAVO68WEB/fwdx/pkg/output"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate the CLI with OIDC device flow",
	RunE:  runLogin,
}

func init() {
	loginCmd.Flags().String("server", "", "fwdx server URL (or FWDX_SERVER)")
}

func runLogin(cmd *cobra.Command, args []string) error {
	serverURL, _ := cmd.Flags().GetString("server")
	base, err := resolveServerBase(serverURL)
	if err != nil {
		return err
	}
	resp, err := http.Post(base.ResolveReference(&url.URL{Path: "/auth/device/start"}).String(), "application/json", http.NoBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("device auth start: %s", resp.Status)
	}
	var dev struct {
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		ExpiresIn               int    `json:"expires_in"`
		Interval                int    `json:"interval"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dev); err != nil {
		return err
	}
	fmt.Printf("Open: %s\n", dev.VerificationURI)
	fmt.Printf("Code: %s\n", dev.UserCode)
	if dev.VerificationURIComplete != "" {
		fmt.Printf("Direct URL: %s\n", dev.VerificationURIComplete)
	}
	if dev.Interval <= 0 {
		dev.Interval = 5
	}
	deadline := time.Now().Add(time.Duration(dev.ExpiresIn) * time.Second)
	for time.Now().Before(deadline) {
		body, _ := json.Marshal(map[string]string{"device_code": dev.DeviceCode})
		req, _ := http.NewRequest(http.MethodPost, base.ResolveReference(&url.URL{Path: "/auth/device/poll"}).String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		pollResp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		if pollResp.StatusCode == http.StatusAccepted {
			pollResp.Body.Close()
			time.Sleep(time.Duration(dev.Interval) * time.Second)
			continue
		}
		if pollResp.StatusCode != http.StatusOK {
			defer pollResp.Body.Close()
			return fmt.Errorf("device auth poll: %s", pollResp.Status)
		}
		var out struct {
			AccessToken string `json:"access_token"`
			ExpiresAt   string `json:"expires_at"`
			Subject     string `json:"subject"`
			Email       string `json:"email"`
			DisplayName string `json:"display_name"`
			Role        string `json:"role"`
		}
		if err := json.NewDecoder(pollResp.Body).Decode(&out); err != nil {
			pollResp.Body.Close()
			return err
		}
		pollResp.Body.Close()
		expiresAt, _ := time.Parse(time.RFC3339Nano, out.ExpiresAt)
		if err := config.SaveAuthSession(&config.AuthSession{
			ServerURL:   base.String(),
			AccessToken: out.AccessToken,
			ExpiresAt:   expiresAt,
			Subject:     out.Subject,
			Email:       out.Email,
			DisplayName: out.DisplayName,
			Role:        out.Role,
		}); err != nil {
			return err
		}
		output.PrintSuccess("Authenticated")
		fmt.Printf("User: %s\n", out.Email)
		fmt.Printf("Role: %s\n", out.Role)
		return nil
	}
	return fmt.Errorf("device login timed out")
}

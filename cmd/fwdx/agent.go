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

var agentCmd = &cobra.Command{Use: "agent", Short: "Manage tunnel agents"}
var agentCreateCmd = &cobra.Command{Use: "create", Short: "Create a local agent credential", RunE: runAgentCreate}
var agentListCmd = &cobra.Command{Use: "list", Short: "List agents", RunE: runAgentList}
var agentRevokeCmd = &cobra.Command{Use: "revoke <name>", Short: "Revoke an agent", Args: cobra.ExactArgs(1), RunE: runAgentRevoke}

func init() {
	agentCreateCmd.Flags().String("name", "", "Agent name")
	agentCmd.AddCommand(agentCreateCmd, agentListCmd, agentRevokeCmd)
}

func runAgentCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	sess, err := requireAuthSession()
	if err != nil {
		return err
	}
	base, err := resolveServerBase(sess.ServerURL)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]string{"name": name})
	req, _ := http.NewRequest(http.MethodPost, base.ResolveReference(&url.URL{Path: "/api/agents"}).String(), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+sess.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("create agent: %s", resp.Status)
	}
	var out struct {
		Agent struct {
			Name string `json:"name"`
		} `json:"agent"`
		Credential string `json:"credential"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	fmt.Printf("Agent: %s\nCredential: %s\n", out.Agent.Name, out.Credential)
	return nil
}

func runAgentList(cmd *cobra.Command, args []string) error {
	sess, err := requireAuthSession()
	if err != nil {
		return err
	}
	base, err := resolveServerBase(sess.ServerURL)
	if err != nil {
		return err
	}
	req, _ := http.NewRequest(http.MethodGet, base.ResolveReference(&url.URL{Path: "/api/agents"}).String(), nil)
	req.Header.Set("Authorization", "Bearer "+sess.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("list agents: %s", resp.Status)
	}
	var list []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return err
	}
	if len(list) == 0 {
		fmt.Println("No agents.")
		return nil
	}
	for _, a := range list {
		fmt.Printf("%s\t%s\n", a.Name, a.Status)
	}
	return nil
}

func runAgentRevoke(cmd *cobra.Command, args []string) error {
	sess, err := requireAuthSession()
	if err != nil {
		return err
	}
	base, err := resolveServerBase(sess.ServerURL)
	if err != nil {
		return err
	}
	req, _ := http.NewRequest(http.MethodPost, base.ResolveReference(&url.URL{Path: "/api/agents/" + url.PathEscape(strings.ToLower(args[0])) + "/revoke"}).String(), bytes.NewReader([]byte("{}")))
	req.Header.Set("Authorization", "Bearer "+sess.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("revoke agent: %s", resp.Status)
	}
	fmt.Printf("Revoked agent %s\n", args[0])
	return nil
}

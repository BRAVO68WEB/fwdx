package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/BRAVO68WEB/fwdx/internal/tunnel"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

func PrintSuccess(msg string) {
	color.Green(msg)
}

func PrintError(msg string) error {
	color.Red("❌ " + msg)
	return fmt.Errorf("%s", msg)
}

func PrintInfo(msg string) {
	color.Cyan(msg)
}

func PrintTunnelList(tunnels []*tunnel.Tunnel, format string) {
	if format == "json" {
		data, _ := json.MarshalIndent(tunnels, "", "  ")
		fmt.Println(string(data))
		return
	}
	if format == "yaml" {
		data, _ := json.Marshal(tunnels)
		fmt.Println(string(data))
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Hostname", "Local", "Status", "PID"})
	for _, t := range tunnels {
		status := "stopped"
		if t.Running {
			status = "running"
		}
		pid := "-"
		if t.PID > 0 {
			pid = fmt.Sprintf("%d", t.PID)
		}
		table.Append([]string{t.Name, t.Hostname, t.Local, status, pid})
	}
	table.Render()
}

func PrintTunnelDetails(t *tunnel.Tunnel) {
	fmt.Printf("Name:      %s\n", t.Name)
	fmt.Printf("Hostname:  https://%s\n", t.Hostname)
	fmt.Printf("Local:     http://%s\n", t.Local)
	fmt.Printf("Private:   %v\n", t.Private)
	fmt.Printf("Running:   %v\n", t.Running)
	if t.PID > 0 {
		fmt.Printf("PID:       %d\n", t.PID)
	}
	fmt.Printf("Created:   %s\n", t.CreatedAt.Format("2006-01-02 15:04:05"))
}

package fwdx

import (
	"fmt"
	"log"
	"os"

	"github.com/BRAVO68WEB/fwdx/internal/server"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the fwdx tunneling server",
	Long:  `Start the fwdx tunneling server. Clients connect to register tunnels; public traffic is proxied by hostname.`,
	RunE:  runServe,
}

func init() {
	home, _ := homedir.Dir()
	dataDirDefault := home + "/.fwdx-server"

	serveCmd.Flags().String("hostname", "", "Server public hostname (e.g. tunnel.example.com)")
	serveCmd.Flags().Int("https-port", 443, "HTTPS port for public and tunnel traffic")
	serveCmd.Flags().Int("tunnel-port", 4443, "Tunnel registration port (clients connect here)")
	serveCmd.Flags().String("client-token", "", "Token required for tunnel registration (or FWDX_CLIENT_TOKEN)")
	serveCmd.Flags().String("admin-token", "", "Token required for admin API (or FWDX_ADMIN_TOKEN)")
	serveCmd.Flags().String("tls-cert", "", "Path to TLS certificate file (omit when using --http-port behind nginx)")
	serveCmd.Flags().String("tls-key", "", "Path to TLS private key file (omit when using --http-port)")
	serveCmd.Flags().Int("http-port", 0, "Listen HTTP only on this port (single port; use when nginx terminates TLS)")
	serveCmd.Flags().String("data-dir", dataDirDefault, "Directory for allowed_domains.json")
}

func runServe(cmd *cobra.Command, args []string) error {
	hostname, _ := cmd.Flags().GetString("hostname")
	if hostname == "" {
		hostname = os.Getenv("FWDX_HOSTNAME")
	}
	httpsPort, _ := cmd.Flags().GetInt("https-port")
	tunnelPort, _ := cmd.Flags().GetInt("tunnel-port")
	clientToken, _ := cmd.Flags().GetString("client-token")
	if clientToken == "" {
		clientToken = os.Getenv("FWDX_CLIENT_TOKEN")
	}
	adminToken, _ := cmd.Flags().GetString("admin-token")
	if adminToken == "" {
		adminToken = os.Getenv("FWDX_ADMIN_TOKEN")
	}
	tlsCert, _ := cmd.Flags().GetString("tls-cert")
	tlsKey, _ := cmd.Flags().GetString("tls-key")
	httpPort, _ := cmd.Flags().GetInt("http-port")
	dataDir, _ := cmd.Flags().GetString("data-dir")

	if hostname == "" {
		return fmt.Errorf("hostname is required (--hostname or FWDX_HOSTNAME)")
	}
	if clientToken == "" {
		return fmt.Errorf("client-token is required (--client-token or FWDX_CLIENT_TOKEN)")
	}
	if httpPort == 0 && (tlsCert == "" || tlsKey == "") {
		return fmt.Errorf("tls-cert and tls-key are required (or use --http-port when behind a reverse proxy)")
	}
	if httpPort != 0 && (tlsCert != "" || tlsKey != "") {
		return fmt.Errorf("do not set tls-cert/tls-key when using --http-port")
	}

	cfg := server.Config{
		Hostname:    hostname,
		HTTPSPort:   httpsPort,
		TunnelPort:  tunnelPort,
		HTTPPort:    httpPort,
		ClientToken: clientToken,
		AdminToken:  adminToken,
		TLSCertFile: tlsCert,
		TLSKeyFile:  tlsKey,
		DataDir:     dataDir,
	}

	srv, err := server.New(cfg)
	if err != nil {
		return err
	}

	if httpPort != 0 {
		log.Printf("[fwdx] server starting (HTTP, behind proxy): http://:%d", httpPort)
	} else if httpsPort == tunnelPort {
		log.Printf("[fwdx] server starting: https://%s (single port :%d)", hostname, httpsPort)
	} else {
		log.Printf("[fwdx] server starting: https://%s (public :%d, tunnel :%d)", hostname, httpsPort, tunnelPort)
	}
	return srv.Run()
}

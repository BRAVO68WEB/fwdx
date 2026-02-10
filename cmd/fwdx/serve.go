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
	Short: "Start the fwdx server",
	Long:  `Start the fwdx server. Web port: HTTP/HTTPS (proxy + admin). Grpc port: tunnel connections. Use nginx to forward 443 -> web and gRPC stream -> grpc.`,
	RunE:  runServe,
}

func init() {
	home, _ := homedir.Dir()
	dataDirDefault := home + "/.fwdx-server"

	serveCmd.Flags().String("hostname", "", "Server public hostname (e.g. tunnel.example.com)")
	serveCmd.Flags().Int("web-port", 8080, "Port for HTTP/HTTPS (proxy + admin). Nginx forwards 443 here.")
	serveCmd.Flags().Int("grpc-port", 4440, "Port for gRPC tunnel. Nginx forwards gRPC stream here.")
	serveCmd.Flags().String("client-token", "", "Token for tunnel clients (or FWDX_CLIENT_TOKEN)")
	serveCmd.Flags().String("admin-token", "", "Token for admin API (or FWDX_ADMIN_TOKEN)")
	serveCmd.Flags().String("tls-cert", "", "TLS cert path (omit when nginx terminates TLS)")
	serveCmd.Flags().String("tls-key", "", "TLS key path (omit when nginx terminates TLS)")
	serveCmd.Flags().String("data-dir", dataDirDefault, "Directory for allowed_domains.json")
}

func runServe(cmd *cobra.Command, args []string) error {
	hostname, _ := cmd.Flags().GetString("hostname")
	if hostname == "" {
		hostname = os.Getenv("FWDX_HOSTNAME")
	}
	webPort, _ := cmd.Flags().GetInt("web-port")
	grpcPort, _ := cmd.Flags().GetInt("grpc-port")
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
	dataDir, _ := cmd.Flags().GetString("data-dir")

	if hostname == "" {
		return fmt.Errorf("hostname is required (--hostname or FWDX_HOSTNAME)")
	}
	if clientToken == "" {
		return fmt.Errorf("client-token is required (--client-token or FWDX_CLIENT_TOKEN)")
	}

	cfg := server.Config{
		Hostname:    hostname,
		WebPort:     webPort,
		GrpcPort:    grpcPort,
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

	if tlsCert != "" && tlsKey != "" {
		log.Printf("[fwdx] server listening https://:%d (web), grpc://:%d (tunnels)", webPort, grpcPort)
	} else {
		log.Printf("[fwdx] server listening http://:%d (web), grpc://:%d (tunnels) — put nginx in front", webPort, grpcPort)
	}
	return srv.Run()
}

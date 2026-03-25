package fwdx

import (
	"fmt"
	"log"
	"os"
	"strings"

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

	serveCmd.Flags().String("hostname", "", "Server public hostname (e.g. tunnel.myweb.site)")
	serveCmd.Flags().Int("web-port", 8080, "Port for HTTP/HTTPS (proxy + admin). Nginx forwards 443 here.")
	serveCmd.Flags().Int("grpc-port", 4440, "Port for gRPC tunnel. Nginx forwards gRPC stream here.")
	serveCmd.Flags().String("tls-cert", "", "TLS cert path (omit when nginx terminates TLS)")
	serveCmd.Flags().String("tls-key", "", "TLS key path (omit when nginx terminates TLS)")
	serveCmd.Flags().String("data-dir", dataDirDefault, "Directory for allowed_domains.json")
	serveCmd.Flags().String("oidc-issuer", "", "OIDC issuer URL (or FWDX_OIDC_ISSUER)")
	serveCmd.Flags().String("oidc-client-id", "", "OIDC client ID (or FWDX_OIDC_CLIENT_ID)")
	serveCmd.Flags().String("oidc-client-secret", "", "OIDC client secret (or FWDX_OIDC_CLIENT_SECRET)")
	serveCmd.Flags().String("oidc-redirect-url", "", "OIDC redirect URL (or FWDX_OIDC_REDIRECT_URL)")
	serveCmd.Flags().String("oidc-scopes", "openid,profile,email", "Comma-separated OIDC scopes")
	serveCmd.Flags().String("oidc-admin-emails", "", "Comma-separated admin email allowlist")
	serveCmd.Flags().String("oidc-admin-subjects", "", "Comma-separated admin subject allowlist")
	serveCmd.Flags().String("oidc-admin-groups", "", "Comma-separated admin group allowlist")
	serveCmd.Flags().String("oidc-session-secret", "", "Secret used to hash issued session tokens")
	serveCmd.Flags().String("oidc-device-client-id", "", "Optional OIDC device flow client ID override")
	serveCmd.Flags().String("trusted-proxy-cidrs", "", "Comma-separated trusted proxy CIDRs for client IP resolution")
}

func runServe(cmd *cobra.Command, args []string) error {
	hostname, _ := cmd.Flags().GetString("hostname")
	if hostname == "" {
		hostname = os.Getenv("FWDX_HOSTNAME")
	}
	webPort, _ := cmd.Flags().GetInt("web-port")
	grpcPort, _ := cmd.Flags().GetInt("grpc-port")
	tlsCert, _ := cmd.Flags().GetString("tls-cert")
	tlsKey, _ := cmd.Flags().GetString("tls-key")
	dataDir, _ := cmd.Flags().GetString("data-dir")
	oidcIssuer, _ := cmd.Flags().GetString("oidc-issuer")
	if oidcIssuer == "" {
		oidcIssuer = os.Getenv("FWDX_OIDC_ISSUER")
	}
	oidcClientID, _ := cmd.Flags().GetString("oidc-client-id")
	if oidcClientID == "" {
		oidcClientID = os.Getenv("FWDX_OIDC_CLIENT_ID")
	}
	oidcClientSecret, _ := cmd.Flags().GetString("oidc-client-secret")
	if oidcClientSecret == "" {
		oidcClientSecret = os.Getenv("FWDX_OIDC_CLIENT_SECRET")
	}
	oidcRedirectURL, _ := cmd.Flags().GetString("oidc-redirect-url")
	if oidcRedirectURL == "" {
		oidcRedirectURL = os.Getenv("FWDX_OIDC_REDIRECT_URL")
	}
	oidcScopes, _ := cmd.Flags().GetString("oidc-scopes")
	if env := os.Getenv("FWDX_OIDC_SCOPES"); env != "" {
		oidcScopes = env
	}
	oidcAdminEmails, _ := cmd.Flags().GetString("oidc-admin-emails")
	if env := os.Getenv("FWDX_OIDC_ADMIN_EMAILS"); env != "" {
		oidcAdminEmails = env
	}
	oidcAdminSubjects, _ := cmd.Flags().GetString("oidc-admin-subjects")
	if env := os.Getenv("FWDX_OIDC_ADMIN_SUBJECTS"); env != "" {
		oidcAdminSubjects = env
	}
	oidcAdminGroups, _ := cmd.Flags().GetString("oidc-admin-groups")
	if env := os.Getenv("FWDX_OIDC_ADMIN_GROUPS"); env != "" {
		oidcAdminGroups = env
	}
	oidcSessionSecret, _ := cmd.Flags().GetString("oidc-session-secret")
	if oidcSessionSecret == "" {
		oidcSessionSecret = os.Getenv("FWDX_OIDC_SESSION_SECRET")
	}
	oidcDeviceClientID, _ := cmd.Flags().GetString("oidc-device-client-id")
	if oidcDeviceClientID == "" {
		oidcDeviceClientID = os.Getenv("FWDX_OIDC_DEVICE_CLIENT_ID")
	}
	trustedProxyCIDRs, _ := cmd.Flags().GetString("trusted-proxy-cidrs")
	if env := os.Getenv("FWDX_TRUSTED_PROXY_CIDRS"); env != "" {
		trustedProxyCIDRs = env
	}

	if hostname == "" {
		return fmt.Errorf("hostname is required (--hostname or FWDX_HOSTNAME)")
	}
	cfg := server.Config{
		Hostname:           hostname,
		WebPort:            webPort,
		GrpcPort:           grpcPort,
		TLSCertFile:        tlsCert,
		TLSKeyFile:         tlsKey,
		DataDir:            dataDir,
		OIDCIssuer:         oidcIssuer,
		OIDCClientID:       oidcClientID,
		OIDCClientSecret:   oidcClientSecret,
		OIDCRedirectURL:    oidcRedirectURL,
		OIDCScopes:         splitCSV(oidcScopes),
		OIDCAdminEmails:    splitCSV(oidcAdminEmails),
		OIDCAdminSubjects:  splitCSV(oidcAdminSubjects),
		OIDCAdminGroups:    splitCSV(oidcAdminGroups),
		OIDCSessionSecret:  oidcSessionSecret,
		OIDCDeviceClientID: oidcDeviceClientID,
		TrustedProxyCIDRs:  splitCSV(trustedProxyCIDRs),
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

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

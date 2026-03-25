package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// Config holds server configuration.
// Web: HTTP/HTTPS (public traffic). Grpc: tunnel connections (gRPC).
// Behind nginx: no TLS, nginx forwards 443 -> WebPort and gRPC stream -> GrpcPort.
// Direct: set TLSCertFile/TLSKeyFile; both Web and Grpc use TLS.
type Config struct {
	Hostname           string
	WebPort            int // HTTP/HTTPS for public and admin (e.g. 8080 behind nginx, 443 direct)
	GrpcPort           int // gRPC for tunnel client connections (e.g. 4440 behind nginx, 4443 direct)
	TLSCertFile        string
	TLSKeyFile         string
	DataDir            string
	OIDCIssuer         string
	OIDCClientID       string
	OIDCClientSecret   string
	OIDCRedirectURL    string
	OIDCScopes         []string
	OIDCAdminEmails    []string
	OIDCAdminSubjects  []string
	OIDCAdminGroups    []string
	OIDCSessionSecret  string
	OIDCDeviceClientID string
	TrustedProxyCIDRs  []string
}

// Server runs the fwdx server: web (proxy + admin) and gRPC (tunnels).
type Server struct {
	cfg      Config
	registry *Registry
	domains  *DomainStore
	stats    *StatsStore
	store    *Store
	auth     *AuthManager
	started  time.Time

	proxyHandler http.Handler

	webServer    *http.Server
	grpcListener net.Listener
	mu           sync.Mutex
}

// New creates a new Server.
func New(cfg Config) (*Server, error) {
	if cfg.Hostname == "" {
		return nil, fmt.Errorf("hostname is required")
	}
	if cfg.DataDir == "" {
		cfg.DataDir = ".fwdx-server"
	}

	registry := NewRegistry()
	domains := NewDomainStore(cfg.DataDir)
	stats := NewStatsStore()
	store, err := NewStore(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	return &Server{
		cfg:          cfg,
		registry:     registry,
		domains:      domains,
		stats:        stats,
		store:        store,
		started:      time.Now(),
		proxyHandler: ProxyHandlerWithConfig(registry, cfg, stats, store),
	}, nil
}

// Run starts web and gRPC listeners. Use TLS on both if certs are set.
func (s *Server) Run() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	useTLS := s.cfg.TLSCertFile != "" && s.cfg.TLSKeyFile != ""
	auth, err := NewAuthManager(context.Background(), s.cfg, s.store, useTLS)
	if err != nil {
		return err
	}
	s.auth = auth

	mux := http.NewServeMux()
	adminUI := AdminUIRouter(s.cfg, s.registry, s.domains, s.stats, s.store, auth, s.started, useTLS)
	mux.Handle("/admin/ui/", adminUI)
	mux.Handle("/admin/ui", adminUI)
	mux.Handle("/admin/", AdminRouter(s.cfg.Hostname, s.registry, s.domains, auth, s.stats, s.store))
	mux.Handle("/api/", ControlPlaneRouter(s.cfg, s.domains, s.store, auth))
	mux.HandleFunc("/auth/oidc/login", auth.handleOIDCLogin)
	mux.HandleFunc("/auth/oidc/callback", auth.handleOIDCCallback)
	mux.HandleFunc("/auth/oidc/logout", auth.handleOIDCLogout)
	mux.HandleFunc("/auth/device/start", auth.handleDeviceStart)
	mux.HandleFunc("/auth/device/poll", auth.handleDevicePoll)
	mux.HandleFunc("/api/users/me", auth.handleWhoAmI)
	mux.Handle("/", s.proxyHandler)

	s.webServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.WebPort),
		Handler: mux,
	}
	if useTLS {
		tlsConfig, err := s.loadTLS()
		if err != nil {
			return err
		}
		s.webServer.TLSConfig = tlsConfig
	}

	grpcLn, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.GrpcPort))
	if err != nil {
		return err
	}
	s.grpcListener = grpcLn

	var wg sync.WaitGroup
	var runErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		if useTLS {
			runErr = firstErr(runErr, s.webServer.ListenAndServeTLS(s.cfg.TLSCertFile, s.cfg.TLSKeyFile))
		} else {
			runErr = firstErr(runErr, s.webServer.ListenAndServe())
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runErr = firstErr(runErr, RunGrpcServer(grpcLn, s.registry, s.domains.List, s.cfg.Hostname, useTLS, s.cfg.TLSCertFile, s.cfg.TLSKeyFile, s.store))
	}()

	wg.Wait()
	return runErr
}

func firstErr(prev, err error) error {
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return prev
}

func (s *Server) loadTLS() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(s.cfg.TLSCertFile, s.cfg.TLSKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load TLS: %w", err)
	}
	return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
}

func (s *Server) Registry() *Registry   { return s.registry }
func (s *Server) Domains() *DomainStore { return s.domains }

package server

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
)

// Config holds server configuration.
type Config struct {
	Hostname    string
	HTTPSPort   int
	TunnelPort  int
	ClientToken string
	AdminToken  string
	TLSCertFile string
	TLSKeyFile  string
	DataDir     string
}

// Server runs the fwdx tunneling server.
type Server struct {
	cfg      Config
	registry *Registry
	domains  *DomainStore

	tunnelHandler http.Handler
	proxyHandler  http.Handler

	httpsServer  *http.Server
	tunnelServer *http.Server
	mu           sync.Mutex
}

// New creates a new Server.
func New(cfg Config) (*Server, error) {
	if cfg.ClientToken == "" {
		return nil, fmt.Errorf("client-token is required")
	}
	if cfg.Hostname == "" {
		return nil, fmt.Errorf("hostname is required")
	}
	if cfg.DataDir == "" {
		cfg.DataDir = ".fwdx-server"
	}

	registry := NewRegistry()
	domains := NewDomainStore(cfg.DataDir)

	s := &Server{
		cfg:      cfg,
		registry: registry,
		domains:  domains,
		tunnelHandler: TunnelHandler(registry, cfg.ClientToken, domains.List, cfg.Hostname),
		proxyHandler:  ProxyHandler(registry),
	}
	return s, nil
}

// Run starts the HTTPS (public) and tunnel listeners. Blocks until both are running or one fails.
func (s *Server) Run() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tlsConfig, err := s.loadTLS()
	if err != nil {
		return err
	}

	// Public HTTPS: route by path; /register and /tunnel/* go to tunnel handler, else proxy
	mux := http.NewServeMux()
	mux.HandleFunc(pathRegister, s.tunnelHandler.ServeHTTP)
	mux.HandleFunc(pathTunnelNext, s.tunnelHandler.ServeHTTP)
	mux.HandleFunc(pathTunnelResponse, s.tunnelHandler.ServeHTTP)
	mux.Handle("/admin/", AdminRouter(s.cfg.AdminToken, s.cfg.Hostname, s.registry, s.domains))
	mux.Handle("/", s.proxyHandler)

	s.httpsServer = &http.Server{
		Addr:      fmt.Sprintf(":%d", s.cfg.HTTPSPort),
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	s.tunnelServer = &http.Server{
		Addr:      fmt.Sprintf(":%d", s.cfg.TunnelPort),
		Handler:   s.tunnelHandler,
		TLSConfig: tlsConfig,
	}

	var wg sync.WaitGroup
	var runErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.httpsServer.ListenAndServeTLS(s.cfg.TLSCertFile, s.cfg.TLSKeyFile); err != nil && err != http.ErrServerClosed {
			runErr = err
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.tunnelServer.ListenAndServeTLS(s.cfg.TLSCertFile, s.cfg.TLSKeyFile); err != nil && err != http.ErrServerClosed {
			runErr = err
		}
	}()

	wg.Wait()
	return runErr
}

func (s *Server) loadTLS() (*tls.Config, error) {
	if s.cfg.TLSCertFile == "" || s.cfg.TLSKeyFile == "" {
		return nil, fmt.Errorf("tls-cert and tls-key are required")
	}
	cert, err := tls.LoadX509KeyPair(s.cfg.TLSCertFile, s.cfg.TLSKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load TLS: %w", err)
	}
	return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
}

// Registry returns the tunnel registry (for admin list).
func (s *Server) Registry() *Registry { return s.registry }

// Domains returns the domain store.
func (s *Server) Domains() *DomainStore { return s.domains }

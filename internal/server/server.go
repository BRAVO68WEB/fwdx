package server

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
)

// Config holds server configuration.
// Web: HTTP/HTTPS (public traffic). Grpc: tunnel connections (gRPC).
// Behind nginx: no TLS, nginx forwards 443 -> WebPort and gRPC stream -> GrpcPort.
// Direct: set TLSCertFile/TLSKeyFile; both Web and Grpc use TLS.
type Config struct {
	Hostname    string
	WebPort     int    // HTTP/HTTPS for public and admin (e.g. 8080 behind nginx, 443 direct)
	GrpcPort    int    // gRPC for tunnel client connections (e.g. 4440 behind nginx, 4443 direct)
	ClientToken string
	AdminToken  string
	TLSCertFile string
	TLSKeyFile  string
	DataDir     string
}

// Server runs the fwdx server: web (proxy + admin) and gRPC (tunnels).
type Server struct {
	cfg      Config
	registry *Registry
	domains  *DomainStore

	proxyHandler http.Handler

	webServer  *http.Server
	grpcListener net.Listener
	mu         sync.Mutex
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

	return &Server{
		cfg:      cfg,
		registry: registry,
		domains:  domains,
		proxyHandler: ProxyHandler(registry, cfg.Hostname),
	}, nil
}

// Run starts web and gRPC listeners. Use TLS on both if certs are set.
func (s *Server) Run() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	mux := http.NewServeMux()
	mux.Handle("/admin/", AdminRouter(s.cfg.AdminToken, s.cfg.Hostname, s.registry, s.domains))
	mux.Handle("/", s.proxyHandler)

	useTLS := s.cfg.TLSCertFile != "" && s.cfg.TLSKeyFile != ""

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
		runErr = firstErr(runErr, RunGrpcServer(grpcLn, s.registry, s.cfg.ClientToken, s.domains.List, s.cfg.Hostname, useTLS, s.cfg.TLSCertFile, s.cfg.TLSKeyFile))
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

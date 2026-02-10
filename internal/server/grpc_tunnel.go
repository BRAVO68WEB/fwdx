package server

import (
	"context"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	tunnelv1 "github.com/BRAVO68WEB/fwdx/api/tunnel/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// GrpcTunnelConn implements TunnelConnection over a gRPC bidirectional stream.
type GrpcTunnelConn struct {
	hostname   string
	remoteAddr string
	sendCh     chan *tunnelv1.ServerMessage
	pending    map[string]chan *ProxyResponse
	pendingMu  sync.Mutex
	closed     bool
	closedMu   sync.Mutex
}

// GetRemoteAddr implements TunnelConnection.
func (c *GrpcTunnelConn) GetRemoteAddr() string { return c.remoteAddr }

// EnqueueRequest implements TunnelConnection. Sends the request on the gRPC stream and waits for the response.
func (c *GrpcTunnelConn) EnqueueRequest(ctx context.Context, pr *ProxyRequest) (resp *ProxyResponse, closed bool) {
	c.closedMu.Lock()
	if c.closed {
		c.closedMu.Unlock()
		return nil, true
	}
	c.closedMu.Unlock()

	if pr.ID == "" {
		pr.ID = uuid.New().String()
	}
	respCh := make(chan *ProxyResponse, 1)
	c.pendingMu.Lock()
	c.pending[pr.ID] = respCh
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, pr.ID)
		c.pendingMu.Unlock()
	}()

	headers := make(map[string]string)
	for k, vv := range pr.Header {
		if len(vv) > 0 {
			headers[k] = strings.Join(vv, ", ")
		}
	}
	msg := &tunnelv1.ServerMessage{
		Message: &tunnelv1.ServerMessage_ProxyRequest{
			ProxyRequest: &tunnelv1.ProxyRequest{
				Id:      pr.ID,
				Method:  pr.Method,
				Path:    pr.Path,
				Query:   pr.Query,
				Headers: headers,
				Body:    pr.Body,
			},
		},
	}

	select {
	case c.sendCh <- msg:
	case <-ctx.Done():
		return nil, true
	}

	timeout := time.NewTimer(60 * time.Second)
	defer timeout.Stop()
	select {
	case r := <-respCh:
		return r, r == nil
	case <-ctx.Done():
		return nil, true
	case <-timeout.C:
		return nil, true
	}
}

// Close implements TunnelConnection. Stops the send goroutine and unblocks pending requests.
func (c *GrpcTunnelConn) Close() {
	c.closedMu.Lock()
	if c.closed {
		c.closedMu.Unlock()
		return
	}
	c.closed = true
	close(c.sendCh)
	c.pendingMu.Lock()
	for _, ch := range c.pending {
		select {
		case ch <- nil:
		default:
		}
	}
	c.pendingMu.Unlock()
	c.closedMu.Unlock()
}

// grpcTunnelServer implements tunnelv1.TunnelServiceServer.
type grpcTunnelServer struct {
	tunnelv1.UnimplementedTunnelServiceServer
	registry        *Registry
	clientToken     string
	allowedDomains  func() []string
	serverHostname  string
}

func newGrpcTunnelServer(registry *Registry, clientToken string, allowedDomains func() []string, serverHostname string) *grpcTunnelServer {
	return &grpcTunnelServer{
		registry:       registry,
		clientToken:    clientToken,
		allowedDomains: allowedDomains,
		serverHostname: serverHostname,
	}
}

func (s *grpcTunnelServer) Connect(stream grpc.BidiStreamingServer[tunnelv1.ClientMessage, tunnelv1.ServerMessage]) error {
	msg, err := stream.Recv()
	if err != nil {
		return err
	}
	reg := msg.GetRegister()
	if reg == nil {
		_ = stream.Send(&tunnelv1.ServerMessage{
			Message: &tunnelv1.ServerMessage_RegisterAck{RegisterAck: &tunnelv1.RegisterAck{Ok: false, Error: "first message must be Register"}},
		})
		return nil
	}

	hostname := strings.TrimSpace(strings.ToLower(reg.GetHostname()))
	localURL := strings.TrimSpace(reg.GetLocalUrl())
	if hostname == "" || localURL == "" {
		_ = stream.Send(&tunnelv1.ServerMessage{
			Message: &tunnelv1.ServerMessage_RegisterAck{RegisterAck: &tunnelv1.RegisterAck{Ok: false, Error: "hostname and local_url required"}},
		})
		return nil
	}

	if s.clientToken != "" {
		md, _ := metadata.FromIncomingContext(stream.Context())
		token := ""
		if v := md.Get("authorization"); len(v) > 0 && len(v[0]) > 7 && strings.EqualFold(v[0][:7], "Bearer ") {
			token = strings.TrimSpace(v[0][7:])
		}
		if token != s.clientToken {
			_ = stream.Send(&tunnelv1.ServerMessage{
				Message: &tunnelv1.ServerMessage_RegisterAck{RegisterAck: &tunnelv1.RegisterAck{Ok: false, Error: "unauthorized"}},
			})
			return nil
		}
	}

	if !strings.HasSuffix(hostname, "."+s.serverHostname) && hostname != s.serverHostname {
		allowed := s.allowedDomains()
		domainAllowed := false
		for _, d := range allowed {
			d = strings.ToLower(strings.TrimSpace(d))
			if d != "" && (hostname == d || strings.HasSuffix(hostname, "."+d)) {
				domainAllowed = true
				break
			}
		}
		if !domainAllowed {
			_ = stream.Send(&tunnelv1.ServerMessage{
				Message: &tunnelv1.ServerMessage_RegisterAck{RegisterAck: &tunnelv1.RegisterAck{Ok: false, Error: "domain not allowed"}},
			})
			return nil
		}
	}

	peerAddr := "unknown"
	if p, ok := peer.FromContext(stream.Context()); ok && p.Addr != nil {
		peerAddr = p.Addr.String()
	}

	conn := &GrpcTunnelConn{
		hostname:   hostname,
		remoteAddr: peerAddr,
		sendCh:     make(chan *tunnelv1.ServerMessage, 64),
		pending:    make(map[string]chan *ProxyResponse),
	}
	s.registry.Register(hostname, conn)
	defer func() {
		conn.Close()
		s.registry.Unregister(hostname)
		log.Printf("[fwdx] tunnel closed hostname=%s", hostname)
	}()

	log.Printf("[fwdx] tunnel registered hostname=%s local=%s from=%s", hostname, localURL, peerAddr)

	if err := stream.Send(&tunnelv1.ServerMessage{
		Message: &tunnelv1.ServerMessage_RegisterAck{RegisterAck: &tunnelv1.RegisterAck{Ok: true}},
	}); err != nil {
		return err
	}

	go func() {
		for m := range conn.sendCh {
			if err := stream.Send(m); err != nil {
				return
			}
		}
	}()

	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		if resp := msg.GetProxyResponse(); resp != nil {
			headers := make(http.Header)
			for k, v := range resp.Headers {
				headers.Set(k, v)
			}
			pr := &ProxyResponse{
				ID:     resp.Id,
				Status: int(resp.Status),
				Header: headers,
				Body:   resp.Body,
			}
			conn.pendingMu.Lock()
			ch, ok := conn.pending[resp.Id]
			delete(conn.pending, resp.Id)
			conn.pendingMu.Unlock()
			if ok {
				select {
				case ch <- pr:
				default:
				}
			}
		}
	}
}

// RunGrpcServer runs the gRPC tunnel server on the given listener (TLS or plain).
// Exported for tests that need to run gRPC with a custom registry.
func RunGrpcServer(ln net.Listener, registry *Registry, clientToken string, allowedDomains func() []string, serverHostname string, useTLS bool, certFile, keyFile string) error {
	opts := []grpc.ServerOption{}
	if useTLS && certFile != "" && keyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
		if err != nil {
			return err
		}
		opts = append(opts, grpc.Creds(creds))
	}
	srv := grpc.NewServer(opts...)
	tunnelv1.RegisterTunnelServiceServer(srv, newGrpcTunnelServer(registry, clientToken, allowedDomains, serverHostname))
	return srv.Serve(ln)
}

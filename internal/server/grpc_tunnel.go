package server

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
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

func maxProxyBodyBytes() int {
	const def = 64 << 20
	v := strings.TrimSpace(os.Getenv("FWDX_MAX_PROXY_BODY_BYTES"))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

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
	registry       *Registry
	allowedDomains func() []string
	serverHostname string
	store          *Store
}

func newGrpcTunnelServer(registry *Registry, allowedDomains func() []string, serverHostname string, store *Store) *grpcTunnelServer {
	return &grpcTunnelServer{
		registry:       registry,
		allowedDomains: allowedDomains,
		serverHostname: serverHostname,
		store:          store,
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

	tunnelName := strings.TrimSpace(strings.ToLower(reg.GetTunnelName()))
	localURL := strings.TrimSpace(reg.GetLocalUrl())
	if tunnelName == "" || localURL == "" {
		_ = stream.Send(&tunnelv1.ServerMessage{
			Message: &tunnelv1.ServerMessage_RegisterAck{RegisterAck: &tunnelv1.RegisterAck{Ok: false, Error: "tunnel_name and local_url required"}},
		})
		return nil
	}

	if s.store == nil {
		_ = stream.Send(&tunnelv1.ServerMessage{
			Message: &tunnelv1.ServerMessage_RegisterAck{RegisterAck: &tunnelv1.RegisterAck{Ok: false, Error: "store unavailable"}},
		})
		return nil
	}
	md, _ := metadata.FromIncomingContext(stream.Context())
	token := ""
	if v := md.Get("authorization"); len(v) > 0 && len(v[0]) > 7 && strings.EqualFold(v[0][:7], "Bearer ") {
		token = strings.TrimSpace(v[0][7:])
	}
	if token == "" {
		_ = stream.Send(&tunnelv1.ServerMessage{
			Message: &tunnelv1.ServerMessage_RegisterAck{RegisterAck: &tunnelv1.RegisterAck{Ok: false, Error: "unauthorized"}},
		})
		return nil
	}
	agent, err := s.store.GetAgentByCredentialHash(stream.Context(), hashCredential(token))
	if err != nil || agent.Status == "revoked" {
		_ = stream.Send(&tunnelv1.ServerMessage{
			Message: &tunnelv1.ServerMessage_RegisterAck{RegisterAck: &tunnelv1.RegisterAck{Ok: false, Error: "unauthorized"}},
		})
		return nil
	}
	tunnelRec, err := s.store.GetTunnelForAgent(stream.Context(), tunnelName, agent.ID)
	if err != nil {
		_ = stream.Send(&tunnelv1.ServerMessage{
			Message: &tunnelv1.ServerMessage_RegisterAck{RegisterAck: &tunnelv1.RegisterAck{Ok: false, Error: "tunnel not assigned to this agent"}},
		})
		return nil
	}
	hostname := strings.TrimSpace(strings.ToLower(tunnelRec.Hostname))

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
	if ok := s.registry.RegisterIfAbsent(hostname, conn); !ok {
		_ = stream.Send(&tunnelv1.ServerMessage{
			Message: &tunnelv1.ServerMessage_RegisterAck{
				RegisterAck: &tunnelv1.RegisterAck{
					Ok:    false,
					Error: "hostname_conflict: hostname already active",
				},
			},
		})
		return nil
	}
	defer func() {
		conn.Close()
		s.registry.Unregister(hostname)
		_ = s.store.TouchAgent(context.Background(), agent.ID, "offline")
		_ = s.store.UpdateTunnelStateByName(context.Background(), tunnelName, "", "offline", "stream closed", time.Now())
		_ = s.store.AddTunnelEvent(context.Background(), hostname, "disconnect", "tunnel stream closed")
		log.Printf("[fwdx] tunnel closed tunnel=%s hostname=%s", tunnelName, hostname)
	}()

	log.Printf("[fwdx] tunnel registered tunnel=%s hostname=%s local=%s agent=%s from=%s", tunnelName, hostname, localURL, agent.Name, peerAddr)
	_ = s.store.TouchAgent(stream.Context(), agent.ID, "connected")
	_ = s.store.SetTunnelDesiredState(stream.Context(), tunnelName, "running")
	_ = s.store.UpdateTunnelStateByName(stream.Context(), tunnelName, localURL, "running", "", time.Now())
	_ = s.store.AddTunnelEvent(stream.Context(), hostname, "register", "tunnel registered from "+peerAddr)

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
func RunGrpcServer(ln net.Listener, registry *Registry, allowedDomains func() []string, serverHostname string, useTLS bool, certFile, keyFile string, stores ...*Store) error {
	maxBody := maxProxyBodyBytes()
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(maxBody + (1 << 20)),
		grpc.MaxSendMsgSize(maxBody + (1 << 20)),
	}
	if useTLS && certFile != "" && keyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
		if err != nil {
			return err
		}
		opts = append(opts, grpc.Creds(creds))
	}
	srv := grpc.NewServer(opts...)
	var store *Store
	if len(stores) > 0 {
		store = stores[0]
	}
	tunnelv1.RegisterTunnelServiceServer(srv, newGrpcTunnelServer(registry, allowedDomains, serverHostname, store))
	return srv.Serve(ln)
}

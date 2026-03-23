package tunnel

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	tunnelv1 "github.com/BRAVO68WEB/fwdx/api/tunnel/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
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

// Connect runs the tunnel client over gRPC: register, then receive ProxyRequests and send ProxyResponses.
// tunnelURL is the gRPC endpoint (e.g. https://tunnel.example.com:4443). Token is sent in gRPC metadata.
func Connect(ctx context.Context, tunnelURL, token, hostname, localURL string, debug bool) error {
	tunnelURL = strings.TrimSuffix(tunnelURL, "/")
	u, err := url.Parse(tunnelURL)
	if err != nil {
		return fmt.Errorf("tunnel URL: %w", err)
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "4443"
		}
	}
	target := host + ":" + port

	maxBody := maxProxyBodyBytes()
	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxBody+(1<<20)),
			grpc.MaxCallSendMsgSize(maxBody+(1<<20)),
		),
	}
	if u.Scheme == "https" {
		tlsCfg := &tls.Config{}
		if s := os.Getenv("FWDX_INSECURE_SKIP_VERIFY"); s == "1" || strings.EqualFold(s, "true") {
			tlsCfg.InsecureSkipVerify = true
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return fmt.Errorf("grpc dial: %w", err)
	}
	defer conn.Close()

	client := tunnelv1.NewTunnelServiceClient(conn)
	md := metadata.Pairs("authorization", "Bearer "+token)
	streamCtx := metadata.NewOutgoingContext(ctx, md)
	stream, err := client.Connect(streamCtx)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	// First message: Register
	if err := stream.Send(&tunnelv1.ClientMessage{
		Message: &tunnelv1.ClientMessage_Register{
			Register: &tunnelv1.Register{
				Hostname: hostname,
				LocalUrl: localURL,
			},
		},
	}); err != nil {
		return fmt.Errorf("send register: %w", err)
	}

	msg, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("recv register ack: %w", err)
	}
	ack := msg.GetRegisterAck()
	if ack == nil || !ack.Ok {
		errStr := "registration failed"
		if ack != nil && ack.Error != "" {
			errStr = ack.Error
		}
		return fmt.Errorf("register: %s", errStr)
	}

	if debug {
		fmt.Printf("tunnel registered %s -> %s\n", hostname, localURL)
	}

	// Loop: receive ProxyRequest, proxy to local, send ProxyResponse
	for {
		msg, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		preq := msg.GetProxyRequest()
		if preq == nil {
			continue
		}

		pr := &ProxyReq{
			ID:     preq.Id,
			Method: preq.Method,
			Path:   preq.Path,
			Query:  preq.Query,
			Header: make(http.Header),
			Body:   preq.Body,
		}
		for k, v := range preq.Headers {
			pr.Header.Set(k, v)
		}

		var resp *ProxyResp
		var proxyErr error
		for attempt := 0; attempt < 4; attempt++ {
			resp, proxyErr = ProxyToLocal(localURL, pr)
			if proxyErr == nil {
				break
			}
			if !errors.Is(proxyErr, ErrLocalTransport) || !IsIdempotentMethod(pr.Method) || attempt == 3 {
				break
			}
			if debug {
				log.Printf("[fwdx] local transport retry id=%s method=%s attempt=%d err=%v", pr.ID, pr.Method, attempt+1, proxyErr)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(150 * time.Millisecond):
			}
		}
		if proxyErr != nil {
			if debug {
				log.Printf("[fwdx] local proxy failed id=%s method=%s err=%v", pr.ID, pr.Method, proxyErr)
			}
			body := []byte("bad gateway")
			if errors.Is(proxyErr, ErrLocalResponseTooLarge) {
				body = []byte("local response too large")
			}
			resp = &ProxyResp{
				ID:     pr.ID,
				Status: http.StatusBadGateway,
				Body:   body,
				Header: make(http.Header),
			}
		}

		headers := make(map[string]string)
		for k, vv := range resp.Header {
			if len(vv) > 0 {
				headers[k] = strings.Join(vv, ", ")
			}
		}
		if err := stream.Send(&tunnelv1.ClientMessage{
			Message: &tunnelv1.ClientMessage_ProxyResponse{
				ProxyResponse: &tunnelv1.ProxyResponse{
					Id:      resp.ID,
					Status:  int32(resp.Status),
					Headers: headers,
					Body:    resp.Body,
				},
			},
		}); err != nil {
			return err
		}
	}
}

package tunnel

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	tunnelv1 "github.com/BRAVO68WEB/fwdx/api/tunnel/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

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

	opts := []grpc.DialOption{}
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

		resp := ProxyToLocal(localURL, pr)
		if resp == nil {
			resp = &ProxyResp{ID: pr.ID, Status: 502, Body: []byte("bad gateway")}
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

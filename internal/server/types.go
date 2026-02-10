package server

import (
	"context"
	"net/http"
)

// TunnelConnection is the tunnel between server and client. Implemented only by gRPC.
type TunnelConnection interface {
	EnqueueRequest(ctx context.Context, pr *ProxyRequest) (resp *ProxyResponse, closed bool)
	GetRemoteAddr() string
	Close()
}

// ProxyRequest is sent to the client to proxy to the local app.
type ProxyRequest struct {
	ID     string
	Method string
	Path   string
	Query  string
	Header http.Header
	Body   []byte
}

// ProxyResponse is the response from the client (from the local app).
type ProxyResponse struct {
	ID     string
	Status int
	Header http.Header
	Body   []byte
}

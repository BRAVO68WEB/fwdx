package server

import (
	"net/http"
	"sync"
)

// TunnelConn represents a registered tunnel connection (one HTTP/2 connection from a client).
type TunnelConn struct {
	Hostname   string
	LocalURL   string
	RemoteAddr string

	// requestQueue is sent to by the public proxy when a request arrives; client reads via GET /tunnel/next-request
	requestQueue chan *ProxyRequest
	// pending maps request ID -> channel to send response back to the proxy goroutine
	pending   map[string]chan *ProxyResponse
	pendingMu sync.Mutex
}

// ProxyRequest is sent to the client for a request to proxy to local.
type ProxyRequest struct {
	ID     string
	Method string
	Path   string
	Query  string
	Header http.Header
	Body   []byte
}

// ProxyResponse is sent back from the client with the proxied response.
type ProxyResponse struct {
	ID     string
	Status int
	Header http.Header
	Body   []byte
}

// RegisterRequest is the JSON body for POST /register.
type RegisterRequest struct {
	Hostname string `json:"hostname"`
	Local    string `json:"local"`
}

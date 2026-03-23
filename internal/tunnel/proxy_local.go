package tunnel

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// ProxyReq is a request to forward to the local app (used by HTTP and gRPC connectors).
type ProxyReq struct {
	ID     string
	Method string
	Path   string
	Query  string
	Header http.Header
	Body   []byte
}

// ProxyResp is the response from the local app.
type ProxyResp struct {
	ID     string
	Status int
	Header http.Header
	Body   []byte
}

var (
	// ErrLocalTransport indicates a network/transport failure when reaching local app.
	ErrLocalTransport = errors.New("local transport error")
	// ErrLocalResponseTooLarge indicates the local app response exceeded the configured cap.
	ErrLocalResponseTooLarge = errors.New("local response too large")
)

func maxResponseBodyBytes() int64 {
	const def = int64(64 << 20) // 64MiB
	v := strings.TrimSpace(os.Getenv("FWDX_MAX_RESPONSE_BODY_BYTES"))
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// IsIdempotentMethod returns true for methods that are safe to retry by default.
func IsIdempotentMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

// ProxyToLocal forwards the request to localURL and returns the response.
func ProxyToLocal(localURL string, pr *ProxyReq) (*ProxyResp, error) {
	if pr == nil {
		return nil, fmt.Errorf("%w: nil request", ErrLocalTransport)
	}
	target := localURL + pr.Path
	if pr.Query != "" {
		target += "?" + pr.Query
	}
	body := bytes.NewReader(pr.Body)
	req, err := http.NewRequest(pr.Method, target, body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLocalTransport, err)
	}
	for k, vv := range pr.Header {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}
	req.Header.Del("Connection")
	req.Header.Del("X-Tunnel-Hostname")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLocalTransport, err)
	}
	defer resp.Body.Close()

	max := maxResponseBodyBytes()
	outBody, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLocalTransport, err)
	}
	if int64(len(outBody)) > max {
		return nil, fmt.Errorf("%w: response exceeded %d bytes", ErrLocalResponseTooLarge, max)
	}
	return &ProxyResp{
		ID:     pr.ID,
		Status: resp.StatusCode,
		Header: resp.Header.Clone(),
		Body:   outBody,
	}, nil
}

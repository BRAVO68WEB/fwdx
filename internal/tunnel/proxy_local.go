package tunnel

import (
	"bytes"
	"io"
	"net/http"
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

// ProxyToLocal forwards the request to localURL and returns the response (or nil on error).
func ProxyToLocal(localURL string, pr *ProxyReq) *ProxyResp {
	if pr == nil {
		return nil
	}
	target := localURL + pr.Path
	if pr.Query != "" {
		target += "?" + pr.Query
	}
	body := bytes.NewReader(pr.Body)
	req, err := http.NewRequest(pr.Method, target, body)
	if err != nil {
		return nil
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
		return nil
	}
	defer resp.Body.Close()

	outBody, _ := io.ReadAll(resp.Body)
	return &ProxyResp{
		ID:     pr.ID,
		Status: resp.StatusCode,
		Header: resp.Header.Clone(),
		Body:   outBody,
	}
}

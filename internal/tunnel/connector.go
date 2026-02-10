package tunnel

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// ClientConnector runs the tunnel client: register with server, then long-poll for requests and proxy to local.
func ClientConnector(ctx context.Context, tunnelURL, token, hostname, localURL string, debug bool) error {
	tunnelURL = strings.TrimSuffix(tunnelURL, "/")
	base, err := url.Parse(tunnelURL)
	if err != nil {
		return fmt.Errorf("tunnel URL: %w", err)
	}

	transport := &http.Transport{}
	if s := os.Getenv("FWDX_INSECURE_SKIP_VERIFY"); s == "1" || strings.EqualFold(s, "true") {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	client := &http.Client{Timeout: 0, Transport: transport}

	// Register
	regURL := base.ResolveReference(&url.URL{Path: "/register"}).String()
	regBody, _ := json.Marshal(map[string]string{"hostname": hostname, "local": localURL})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, regURL, bytes.NewReader(regBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("register: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("register: %s", resp.Status)
	}

	if debug {
		fmt.Printf("Registered %s -> %s\n", hostname, localURL)
	}

	// Long-poll loop: GET /tunnel/next-request, proxy to local, POST /tunnel/response
	nextURL := base.ResolveReference(&url.URL{Path: "/tunnel/next-request", RawQuery: "hostname=" + url.QueryEscape(hostname)}).String()
	responseURL := base.ResolveReference(&url.URL{Path: "/tunnel/response", RawQuery: "hostname=" + url.QueryEscape(hostname)}).String()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, nextURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Tunnel-Hostname", hostname)

		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			time.Sleep(2 * time.Second)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			if resp.StatusCode == http.StatusGone {
				return fmt.Errorf("tunnel closed by server")
			}
			time.Sleep(2 * time.Second)
			continue
		}

		var proxyReq struct {
			ID     string
			Method string
			Path   string
			Query  string
			Header http.Header
			Body   []byte
		}
		if err := json.NewDecoder(resp.Body).Decode(&proxyReq); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		// Proxy to local
		proxyResp := proxyToLocal(localURL, &proxyReq)
		if proxyResp == nil {
			proxyResp = &struct {
				ID     string
				Status int
				Header http.Header
				Body   []byte
			}{ID: proxyReq.ID, Status: 502, Body: []byte("bad gateway")}
		}

		// Send response back to server
		respBody, _ := json.Marshal(proxyResp)
		req2, _ := http.NewRequestWithContext(ctx, http.MethodPost, responseURL, bytes.NewReader(respBody))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Authorization", "Bearer "+token)
		req2.Header.Set("X-Tunnel-Hostname", hostname)
		if res, err := client.Do(req2); err == nil {
			res.Body.Close()
		}
	}
}

func proxyToLocal(localURL string, pr *struct {
	ID     string
	Method string
	Path   string
	Query  string
	Header http.Header
	Body   []byte
}) *struct {
	ID     string
	Status int
	Header http.Header
	Body   []byte
} {
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
	// Avoid forwarding Hop-by-hop and tunnel headers
	req.Header.Del("Connection")
	req.Header.Del("X-Tunnel-Hostname")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	outBody, _ := io.ReadAll(resp.Body)
	return &struct {
		ID     string
		Status int
		Header http.Header
		Body   []byte
	}{
		ID:     pr.ID,
		Status: resp.StatusCode,
		Header: resp.Header.Clone(),
		Body:   outBody,
	}
}

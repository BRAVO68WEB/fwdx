package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func maxRequestBodyBytes() int64 {
	const def = int64(64 << 20) // 64MiB
	v := strings.TrimSpace(os.Getenv("FWDX_MAX_REQUEST_BODY_BYTES"))
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func isWebsocketUpgrade(r *http.Request) bool {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return false
	}
	return strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

// ProxyHandler handles incoming public HTTPS requests and forwards them to the appropriate tunnel.
// When the request Host matches serverHostname exactly, a short info page is returned instead of 404.
func ProxyHandler(registry *Registry, serverHostname string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hostname := hostWithoutPort(r.Host)
		conn := registry.Get(hostname)
		if conn == nil {
			if hostname != "" && hostname == hostWithoutPort(serverHostname) {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("fwdx server at " + serverHostname + ".\n\n" +
					"Use a subdomain to reach your tunnel (e.g. myapp." + serverHostname + ").\n" +
					"Create a tunnel from your machine: fwdx tunnel create -l localhost:8080 -s myapp --name myapp && fwdx tunnel start myapp\n"))
				log.Printf("[fwdx] proxy host=%s method=%s path=%s status=200 (server info)", hostname, r.Method, r.URL.Path)
				return
			}
			log.Printf("[fwdx] proxy host=%s method=%s path=%s status=404 no tunnel for hostname", hostname, r.Method, r.URL.Path)
			http.Error(w, "no tunnel for this hostname", http.StatusNotFound)
			return
		}

		if isWebsocketUpgrade(r) {
			http.Error(w, "websocket tunneling is not implemented in this release", http.StatusNotImplemented)
			return
		}

		maxBody := maxRequestBodyBytes()
		if r.ContentLength > maxBody {
			http.Error(w, fmt.Sprintf("request body too large (max %d bytes)", maxBody), http.StatusRequestEntityTooLarge)
			return
		}
		body, _ := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
		_ = r.Body.Close()
		if int64(len(body)) > maxBody {
			http.Error(w, fmt.Sprintf("request body too large (max %d bytes)", maxBody), http.StatusRequestEntityTooLarge)
			return
		}

		pr := &ProxyRequest{
			Method: r.Method,
			Path:   r.URL.Path,
			Query:  r.URL.RawQuery,
			Header: r.Header.Clone(),
			Body:   body,
		}

		ctx, cancel := context.WithTimeout(r.Context(), 65*time.Second)
		defer cancel()

		resp, closed := conn.EnqueueRequest(ctx, pr)
		if closed || resp == nil {
			log.Printf("[fwdx] proxy host=%s method=%s path=%s tunnel unavailable (502)", hostname, r.Method, r.URL.Path)
			http.Error(w, "tunnel unavailable", http.StatusBadGateway)
			return
		}

		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.Status)
		if len(resp.Body) > 0 {
			_, _ = io.Copy(w, bytes.NewReader(resp.Body))
		}
		log.Printf("[fwdx] proxy host=%s method=%s path=%s status=%d", hostname, r.Method, r.URL.Path, resp.Status)
	}
}

func hostWithoutPort(host string) string {
	if len(host) > 0 && host[0] == '[' {
		// IPv6: [::1]:80
		if j := strings.IndexByte(host, ']'); j >= 0 {
			return host[:j+1]
		}
	}
	for i := 0; i < len(host); i++ {
		if host[i] == ':' {
			return host[:i]
		}
	}
	return host
}

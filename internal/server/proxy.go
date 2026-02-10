package server

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

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

		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()

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

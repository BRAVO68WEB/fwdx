package server

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

// ProxyHandler handles incoming public HTTPS requests and forwards them to the appropriate tunnel.
func ProxyHandler(registry *Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hostname := hostWithoutPort(r.Host)
		conn := registry.Get(hostname)
		if conn == nil {
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

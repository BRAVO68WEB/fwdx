package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
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
	return ProxyHandlerWithConfig(registry, Config{Hostname: serverHostname}, nil, nil)
}

// ProxyHandlerWithStats is ProxyHandler with optional in-memory stats collection.
func ProxyHandlerWithStats(registry *Registry, serverHostname string, stats *StatsStore, store *Store) http.HandlerFunc {
	return ProxyHandlerWithConfig(registry, Config{Hostname: serverHostname}, stats, store)
}

func ProxyHandlerWithConfig(registry *Registry, cfg Config, stats *StatsStore, store *Store) http.HandlerFunc {
	trustedPrefixes := parseTrustedProxyCIDRs(cfg.TrustedProxyCIDRs)
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		hostname := hostWithoutPort(r.Host)
		clientIP := resolveClientIP(r, trustedPrefixes)
		var tunnelRec TunnelRecord
		record := func(status int, outBytes int, isErr bool, errText string) {
			inBytes := 0
			if r.ContentLength > 0 {
				inBytes = int(r.ContentLength)
			}
			if stats != nil {
				stats.Record(hostname, clientIP, inBytes, outBytes, status, time.Since(start), isErr)
			}
			if store == nil {
				return
			}
			inBytes64 := int64(0)
			if r.ContentLength > 0 {
				inBytes64 = r.ContentLength
			}
			_ = store.InsertRequestLog(r.Context(), RequestLogRecord{
				TunnelID:  tunnelRec.ID,
				Hostname:  hostname,
				Timestamp: time.Now(),
				Method:    r.Method,
				Host:      r.Host,
				Path:      r.URL.Path,
				Status:    status,
				LatencyMS: time.Since(start).Milliseconds(),
				BytesIn:   inBytes64,
				BytesOut:  int64(outBytes),
				ClientIP:  clientIP,
				ErrorText: errText,
				WSUpgrade: isWebsocketUpgrade(r),
			})
		}

		if store != nil {
			rec, err := store.GetTunnelByHostname(r.Context(), hostname)
			if err == nil {
				tunnelRec = rec
			} else if err != sql.ErrNoRows {
				http.Error(w, "tunnel lookup failed", http.StatusInternalServerError)
				record(http.StatusInternalServerError, len("tunnel lookup failed\n"), true, "tunnel lookup failed")
				return
			}
		}

		conn := registry.Get(hostname)
		if conn == nil {
			if hostname != "" && hostname == hostWithoutPort(cfg.Hostname) && tunnelRec.ID == 0 {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("fwdx server at " + cfg.Hostname + ".\n\n" +
					"Use a subdomain to reach your tunnel (e.g. myapp." + cfg.Hostname + ").\n" +
					"Create a tunnel from your machine: fwdx tunnel create -l localhost:8080 -s myapp --name myapp && fwdx tunnel start myapp\n"))
				log.Printf("[fwdx] proxy host=%s method=%s path=%s status=200 (server info)", hostname, r.Method, r.URL.Path)
				record(http.StatusOK, 0, false, "")
				return
			}
			if tunnelRec.ID > 0 {
				log.Printf("[fwdx] proxy host=%s method=%s path=%s tunnel unavailable (502)", hostname, r.Method, r.URL.Path)
				http.Error(w, "tunnel unavailable", http.StatusBadGateway)
				record(http.StatusBadGateway, len("tunnel unavailable\n"), true, "tunnel unavailable")
				return
			}
			log.Printf("[fwdx] proxy host=%s method=%s path=%s status=404 no tunnel for hostname", hostname, r.Method, r.URL.Path)
			http.Error(w, "no tunnel for this hostname", http.StatusNotFound)
			record(http.StatusNotFound, len("no tunnel for this hostname\n"), true, "no tunnel for this hostname")
			return
		}

		if tunnelRec.ID == 0 && store != nil {
			rec, err := store.GetTunnelByHostname(r.Context(), hostname)
			if err == nil {
				tunnelRec = rec
			}
		}

		if tunnelRec.ID > 0 && store != nil {
			rule, err := store.GetTunnelAccessRule(r.Context(), tunnelRec.ID)
			if err != nil && err != sql.ErrNoRows {
				http.Error(w, "tunnel access rule lookup failed", http.StatusInternalServerError)
				record(http.StatusInternalServerError, len("tunnel access rule lookup failed\n"), true, "tunnel access rule lookup failed")
				return
			}
			if err == nil {
				if !allowedByIP(rule, clientIP) {
					http.Error(w, "forbidden", http.StatusForbidden)
					record(http.StatusForbidden, len("forbidden\n"), true, "ip not allowed")
					return
				}
				switch rule.AuthMode {
				case "basic_auth":
					user, pass, ok := r.BasicAuth()
					if !ok || !verifyAccessRuleBasicAuth(rule, user, pass) {
						w.Header().Set("WWW-Authenticate", `Basic realm="fwdx"`)
						http.Error(w, "unauthorized", http.StatusUnauthorized)
						record(http.StatusUnauthorized, len("unauthorized\n"), true, "basic auth failed")
						return
					}
				case "shared_secret_header":
					if !verifyAccessRuleSharedSecret(rule, r.Header.Get(rule.SharedSecretHeaderName)) {
						http.Error(w, "unauthorized", http.StatusUnauthorized)
						record(http.StatusUnauthorized, len("unauthorized\n"), true, "shared secret failed")
						return
					}
				}
			}
		}

		if isWebsocketUpgrade(r) {
			http.Error(w, "websocket tunneling is not implemented in this release", http.StatusNotImplemented)
			record(http.StatusNotImplemented, len("websocket tunneling is not implemented in this release\n"), true, "websocket tunneling is not implemented in this release")
			return
		}

		maxBody := maxRequestBodyBytes()
		if r.ContentLength > maxBody {
			http.Error(w, fmt.Sprintf("request body too large (max %d bytes)", maxBody), http.StatusRequestEntityTooLarge)
			record(http.StatusRequestEntityTooLarge, 0, true, "request body too large")
			return
		}
		body, _ := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
		_ = r.Body.Close()
		if int64(len(body)) > maxBody {
			http.Error(w, fmt.Sprintf("request body too large (max %d bytes)", maxBody), http.StatusRequestEntityTooLarge)
			record(http.StatusRequestEntityTooLarge, 0, true, "request body too large")
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
			record(http.StatusBadGateway, len("tunnel unavailable\n"), true, "tunnel unavailable")
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
		record(resp.Status, len(resp.Body), resp.Status >= 400, "")
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

func parseTrustedProxyCIDRs(values []string) []netip.Prefix {
	out := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if prefix, err := netip.ParsePrefix(value); err == nil {
			out = append(out, prefix)
		}
	}
	return out
}

func resolveClientIP(r *http.Request, trusted []netip.Prefix) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	peerIP, err := netip.ParseAddr(host)
	if err != nil {
		return host
	}
	if peerTrusted(peerIP, trusted) {
		if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
			first := strings.TrimSpace(strings.Split(xff, ",")[0])
			if ip, err := netip.ParseAddr(first); err == nil {
				return ip.String()
			}
		}
		if xrip := strings.TrimSpace(r.Header.Get("X-Real-Ip")); xrip != "" {
			if ip, err := netip.ParseAddr(xrip); err == nil {
				return ip.String()
			}
		}
	}
	return peerIP.String()
}

func peerTrusted(ip netip.Addr, trusted []netip.Prefix) bool {
	for _, prefix := range trusted {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

func allowedByIP(rule TunnelAccessRuleRecord, clientIP string) bool {
	if len(rule.AllowedIPs) == 0 {
		return true
	}
	ip, err := netip.ParseAddr(strings.TrimSpace(clientIP))
	if err != nil {
		return false
	}
	for _, raw := range rule.AllowedIPs {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(raw))
		if err == nil && prefix.Contains(ip) {
			return true
		}
	}
	return false
}

func sharedSecretHeaderValue(authHeader string) (string, bool) {
	if !strings.HasPrefix(authHeader, "Basic ") {
		return "", false
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
	if err != nil {
		return "", false
	}
	return string(raw), true
}

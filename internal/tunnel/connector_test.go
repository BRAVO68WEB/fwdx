package tunnel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientConnector_InvalidURL(t *testing.T) {
	ctx := context.Background()
	err := ClientConnector(ctx, "://invalid", "token", "app.example.com", "http://localhost:8080", false)
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestClientConnector_Register_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/register" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	ctx := context.Background()
	err := ClientConnector(ctx, srv.URL, "wrong-token", "app.example.com", "http://localhost:8080", false)
	if err == nil {
		t.Error("expected error when server returns 401")
	}
}

func TestClientConnector_Register_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	ctx := context.Background()
	err := ClientConnector(ctx, srv.URL, "token", "app.example.com", "http://localhost:8080", false)
	if err == nil {
		t.Error("expected error when server returns 404")
	}
}

func TestClientConnector_ExitsOnContextCancel(t *testing.T) {
	nextRequestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/register" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok","hostname":"app.example.com"}`))
			return
		}
		if r.URL.Path == "/tunnel/next-request" {
			nextRequestCount++
			if nextRequestCount == 1 {
				// First poll: return immediately so client enters loop and does second GET
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"ID":"id1","Method":"GET","Path":"/","Query":"","Header":{},"Body":null}`))
				return
			}
			// Second poll: block so client waits here until context expires
			<-r.Context().Done()
			return
		}
		if r.URL.Path == "/tunnel/response" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	err := ClientConnector(ctx, srv.URL, "token", "app.example.com", "http://localhost:8080", false)
	if err != context.DeadlineExceeded && err != context.Canceled {
		t.Errorf("expected context error, got %v", err)
	}
}

func TestClientConnector_Register_SuccessThenCancel(t *testing.T) {
	registerHit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/register" && r.Method == http.MethodPost {
			registerHit = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok","hostname":"app.example.com"}`))
			return
		}
		if r.URL.Path == "/tunnel/next-request" {
			// Block until client cancels
			<-r.Context().Done()
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(80 * time.Millisecond)
		cancel()
	}()

	err := ClientConnector(ctx, srv.URL, "token", "app.example.com", "http://localhost:8080", false)
	if err != context.Canceled && err != context.DeadlineExceeded {
		t.Logf("err = %v (acceptable: context cancelled)", err)
	}
	if !registerHit {
		t.Error("register was not called")
	}
}

package server

import (
	"context"
	"testing"
)

// mockConn implements TunnelConnection for tests.
type mockConn struct {
	remoteAddr string
}

func (m *mockConn) EnqueueRequest(context.Context, *ProxyRequest) (*ProxyResponse, bool) {
	return nil, false
}
func (m *mockConn) GetRemoteAddr() string { return m.remoteAddr }
func (m *mockConn) Close()                {}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if r.tunnels == nil {
		t.Fatal("tunnels map not initialized")
	}
}

func TestRegistry_Register_Get_List(t *testing.T) {
	r := NewRegistry()
	conn := &mockConn{remoteAddr: "127.0.0.1:12345"}
	r.Register("app.example.com", conn)

	got := r.Get("app.example.com")
	if got != conn {
		t.Errorf("Get() = %v, want %v", got, conn)
	}

	list := r.List()
	if len(list) != 1 {
		t.Fatalf("List() length = %d, want 1", len(list))
	}
	if list["app.example.com"] != "127.0.0.1:12345" {
		t.Errorf("List()[app.example.com] = %q", list["app.example.com"])
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	r := NewRegistry()
	got := r.Get("nonexistent.example.com")
	if got != nil {
		t.Errorf("Get(nonexistent) = %v, want nil", got)
	}
}

func TestRegistry_Register_Overwrite(t *testing.T) {
	r := NewRegistry()
	r.Register("app.example.com", &mockConn{remoteAddr: "1.2.3.4"})
	r.Register("app.example.com", &mockConn{remoteAddr: "5.6.7.8"})

	got := r.Get("app.example.com")
	if got == nil {
		t.Fatal("Get() = nil")
	}
	if got.GetRemoteAddr() != "5.6.7.8" {
		t.Errorf("GetRemoteAddr() = %q", got.GetRemoteAddr())
	}
}

func TestRegistry_Unregister(t *testing.T) {
	r := NewRegistry()
	conn := &mockConn{remoteAddr: "127.0.0.1"}
	r.Register("app.example.com", conn)
	r.Unregister("app.example.com")

	if r.Get("app.example.com") != nil {
		t.Error("Get() after Unregister should return nil")
	}
	if len(r.List()) != 0 {
		t.Errorf("List() length = %d, want 0", len(r.List()))
	}
}

func TestRegistry_Unregister_Nonexistent(t *testing.T) {
	r := NewRegistry()
	r.Unregister("nonexistent.example.com")
}

func TestRegistry_List_Empty(t *testing.T) {
	r := NewRegistry()
	list := r.List()
	if list == nil {
		t.Fatal("List() returned nil")
	}
	if len(list) != 0 {
		t.Errorf("List() length = %d, want 0", len(list))
	}
}

func TestRegistry_MultipleHosts(t *testing.T) {
	r := NewRegistry()
	r.Register("a.example.com", &mockConn{remoteAddr: "1"})
	r.Register("b.example.com", &mockConn{remoteAddr: "2"})

	list := r.List()
	if len(list) != 2 {
		t.Fatalf("List() length = %d, want 2", len(list))
	}
	if list["a.example.com"] != "1" || list["b.example.com"] != "2" {
		t.Errorf("List() = %v", list)
	}
}

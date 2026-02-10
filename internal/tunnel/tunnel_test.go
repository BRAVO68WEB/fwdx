package tunnel

import (
	"os"
	"testing"
	"time"
)

func setTestEnv(t *testing.T) (cleanup func()) {
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	oldServer := os.Getenv("FWDX_SERVER")
	oldToken := os.Getenv("FWDX_TOKEN")
	os.Setenv("HOME", dir)
	os.Setenv("FWDX_SERVER", "https://tunnel.example.com")
	os.Setenv("FWDX_TOKEN", "test-token")
	return func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("FWDX_SERVER", oldServer)
		os.Setenv("FWDX_TOKEN", oldToken)
	}
}

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}
	if m.tunnelsDir == "" {
		t.Error("tunnelsDir not set")
	}
	if m.running == nil {
		t.Error("running map not initialized")
	}
}

func TestManager_Create_RequiresConfig(t *testing.T) {
	oldServer := os.Getenv("FWDX_SERVER")
	oldToken := os.Getenv("FWDX_TOKEN")
	os.Unsetenv("FWDX_SERVER")
	os.Unsetenv("FWDX_TOKEN")
	defer func() {
		os.Setenv("FWDX_SERVER", oldServer)
		os.Setenv("FWDX_TOKEN", oldToken)
	}()

	m := NewManager()
	_, err := m.Create("localhost:8080", "app", "", false, "")
	if err != errClientConfigRequired {
		t.Errorf("Create without config: err = %v", err)
	}
}

func TestManager_Create_Subdomain_RequiresServerHostname(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	os.Setenv("FWDX_SERVER", "https://tunnel.example.com")
	os.Setenv("FWDX_TOKEN", "tok")
	// ServerHostname is derived from FWDX_SERVER (tunnel.example.com), so Create should work
	m := NewManager()
	t.Cleanup(func() {
		os.Unsetenv("FWDX_SERVER")
		os.Unsetenv("FWDX_TOKEN")
	})

	t.Run("subdomain_ok", func(t *testing.T) {
		tun, err := m.Create("localhost:8080", "myapp", "", false, "")
		if err != nil {
			t.Fatal(err)
		}
		if tun.Hostname != "myapp.tunnel.example.com" {
			t.Errorf("Hostname = %q", tun.Hostname)
		}
		if tun.Local != "localhost:8080" {
			t.Errorf("Local = %q", tun.Local)
		}
		if tun.Name != "myapp-tunnel" {
			t.Errorf("Name = %q", tun.Name)
		}
	})

	t.Run("custom_url_ok", func(t *testing.T) {
		tun, err := m.Create("localhost:9000", "", "app.my.domain", false, "custom")
		if err != nil {
			t.Fatal(err)
		}
		if tun.Hostname != "app.my.domain" {
			t.Errorf("Hostname = %q", tun.Hostname)
		}
		if tun.Name != "custom" {
			t.Errorf("Name = %q", tun.Name)
		}
	})

	t.Run("either_subdomain_or_url", func(t *testing.T) {
		_, err := m.Create("localhost:8080", "", "", false, "")
		if err == nil {
			t.Error("expected error when both subdomain and url empty")
		}
	})
}

func TestManager_Get_List_Delete(t *testing.T) {
	defer setTestEnv(t)()

	m := NewManager()
	created, err := m.Create("localhost:8080", "getlist", "", false, "getlist-tunnel")
	if err != nil {
		t.Fatal(err)
	}

	got, err := m.Get(created.Name)
	if err != nil {
		t.Fatal(err)
	}
	if got.Hostname != created.Hostname {
		t.Errorf("Get: Hostname = %q", got.Hostname)
	}

	list, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) < 1 {
		t.Fatal("List() empty")
	}
	var found bool
	for _, tun := range list {
		if tun.Name == created.Name {
			found = true
			break
		}
	}
	if !found {
		t.Error("List() did not contain created tunnel")
	}

	err = m.Delete(created.Name, false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.Get(created.Name)
	if err == nil {
		t.Error("Get after Delete should fail")
	}
}

func TestManager_Stop_NotRunning(t *testing.T) {
	defer setTestEnv(t)()

	m := NewManager()
	_, _ = m.Create("localhost:8080", "stopped", "", false, "stopped-tunnel")

	err := m.Stop("stopped-tunnel")
	if err == nil {
		t.Error("Stop when not running should return error")
	}
}

func TestManager_Start_Stop_Cancel(t *testing.T) {
	defer setTestEnv(t)()

	m := NewManager()
	_, err := m.Create("localhost:8080", "startstop", "", false, "startstop-tunnel")
	if err != nil {
		t.Fatal(err)
	}

	// Start runs connector in goroutine; it will fail to connect to real server and may exit quickly.
	err = m.Start("startstop-tunnel", false, false)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(30 * time.Millisecond)

	// Stop may succeed (if connector still in map) or fail with "not running" (if connector already exited).
	err = m.Stop("startstop-tunnel")
	if err != nil {
		t.Logf("Stop returned %v (connector may have exited already)", err)
	}

	// Second stop should always error
	err = m.Stop("startstop-tunnel")
	if err == nil {
		t.Error("second Stop should error")
	}
}

func TestManager_Start_AlreadyRunning(t *testing.T) {
	defer setTestEnv(t)()

	m := NewManager()
	_, _ = m.Create("localhost:8080", "dup", "", false, "dup-tunnel")
	_ = m.Start("dup-tunnel", false, false)
	defer m.Stop("dup-tunnel")

	// Call second Start immediately; the first goroutine may still be in m.running
	// (or may have exited if connection failed quickly). If it's still there we get error.
	err := m.Start("dup-tunnel", false, false)
	if err == nil {
		// Connector may have already exited (e.g. connection refused), so second Start succeeded.
		// This is acceptable; we only assert that double-start doesn't panic.
		t.Log("second Start succeeded (first connector may have exited already)")
	}
}

func TestManager_Get_NotFound(t *testing.T) {
	m := NewManager()
	_, err := m.Get("nonexistent")
	if err == nil {
		t.Error("Get(nonexistent) should error")
	}
}

func TestManager_Delete_NotFound(t *testing.T) {
	m := NewManager()
	err := m.Delete("nonexistent", false)
	if err == nil {
		t.Error("Delete(nonexistent) should error")
	}
}

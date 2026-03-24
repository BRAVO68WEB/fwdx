package tunnel

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestManager_Start_AlreadyRunning_StateFile(t *testing.T) {
	defer setTestEnv(t)()
	m := NewManager()
	_, _ = m.Create("localhost:8080", "dup", "", false, "dup-tunnel")
	if err := writeRuntimeState(&RuntimeState{
		Name:      "dup-tunnel",
		Hostname:  "dup.tunnel.example.com",
		Local:     "localhost:8080",
		PID:       os.Getpid(),
		LogPath:   runtimeLogPath("dup-tunnel"),
		StartedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	defer removeRuntimeState("dup-tunnel")
	err := m.Start("dup-tunnel", false)
	if err == nil {
		t.Fatal("expected already running error")
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

func TestTailLogs_ReadsRecentLines(t *testing.T) {
	defer setTestEnv(t)()
	m := NewManager()
	if err := os.MkdirAll(runtimeDir(), 0755); err != nil {
		t.Fatal(err)
	}
	logPath := runtimeLogPath("logtest")
	content := "1\n2\n3\n4\n5\n"
	if err := os.WriteFile(logPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	if err := m.TailLogs("logtest", &b, 2, false); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !strings.Contains(out, "4") || !strings.Contains(out, "5") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestRuntimeStatePath(t *testing.T) {
	defer setTestEnv(t)()
	p := runtimeStatePath("abc")
	if filepath.Ext(p) != ".json" {
		t.Fatalf("state path should end with .json, got %s", p)
	}
}

func TestManager_Stop_UsesRuntimeStatePID(t *testing.T) {
	defer setTestEnv(t)()
	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Skipf("sleep command unavailable: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()

	if err := writeRuntimeState(&RuntimeState{
		Name:      "stoptest",
		Hostname:  "stop.tunnel.example.com",
		Local:     "localhost:8080",
		PID:       cmd.Process.Pid,
		LogPath:   runtimeLogPath("stoptest"),
		StartedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	m := NewManager()
	if err := m.Stop("stoptest"); err != nil {
		t.Fatal(err)
	}
	if _, ok := runtimeStateIfRunning("stoptest"); ok {
		t.Fatal("expected state removed after stop")
	}
}

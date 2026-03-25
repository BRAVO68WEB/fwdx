package tunnel

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/BRAVO68WEB/fwdx/internal/config"
)

type mockControlPlane struct {
	tunnels map[string]map[string]any
	agentID int64
}

func newMockControlPlane() *mockControlPlane {
	return &mockControlPlane{tunnels: map[string]map[string]any{}, agentID: 1}
}

func (m *mockControlPlane) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/agents", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"agent":      map[string]any{"id": m.agentID, "name": body["name"]},
			"credential": "agent-secret",
		})
	})
	mux.HandleFunc("/api/tunnels", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			hostname := body["url"]
			if hostname == "" {
				hostname = body["subdomain"] + ".tunnel.example.com"
			}
			rec := map[string]any{
				"name":              body["name"],
				"hostname":          hostname,
				"local_target_hint": body["local"],
				"assigned_agent":    body["agent_name"],
				"desired_state":     "stopped",
				"actual_state":      "offline",
				"created_at":        time.Now().UTC(),
				"updated_at":        time.Now().UTC(),
			}
			m.tunnels[body["name"]] = rec
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(rec)
		case http.MethodGet:
			list := make([]map[string]any, 0, len(m.tunnels))
			for _, v := range m.tunnels {
				list = append(list, v)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(list)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/tunnels/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/api/tunnels/")
		if i := strings.IndexByte(name, '/'); i >= 0 {
			name = name[:i]
		}
		if name == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		rec, ok := m.tunnels[name]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(rec)
		case http.MethodDelete:
			delete(m.tunnels, name)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	return mux
}

func TestHelperTunnelProcess(t *testing.T) {
	if os.Getenv("GO_WANT_TUNNEL_HELPER") != "1" {
		return
	}
	select {}
}

func startTunnelHelperProcess(t *testing.T, name string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperTunnelProcess", "--", "tunnel", "start", name)
	cmd.Env = append(os.Environ(), "GO_WANT_TUNNEL_HELPER=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGKILL)
			_, _ = cmd.Process.Wait()
		}
	})
	time.Sleep(100 * time.Millisecond)
	return cmd
}

func setTestEnv(t *testing.T, serverURL string) {
	t.Helper()
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })
	if err := config.SaveClientConfig(&config.ClientConfig{ServerURL: serverURL, ServerHostname: "tunnel.example.com", TunnelPort: 4443}); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveAuthSession(&config.AuthSession{ServerURL: serverURL, AccessToken: "session-token", ExpiresAt: time.Now().Add(time.Hour), Subject: "sub", Email: "user@example.com", Role: "admin"}); err != nil {
		t.Fatal(err)
	}
}

func TestManager_Create_Get_List_Delete_Remote(t *testing.T) {
	cp := newMockControlPlane()
	srv := httptest.NewServer(cp.handler())
	defer srv.Close()
	setTestEnv(t, srv.URL)

	m := NewManager()
	created, err := m.Create("localhost:8080", "getlist", "", "getlist-tunnel")
	if err != nil {
		t.Fatal(err)
	}
	if created.Hostname != "getlist.tunnel.example.com" {
		t.Fatalf("hostname=%q", created.Hostname)
	}
	got, err := m.Get(created.Name)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != created.Name {
		t.Fatalf("get name=%q", got.Name)
	}
	list, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("list len=%d", len(list))
	}
	if err := m.Delete(created.Name, false); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Get(created.Name); err == nil {
		t.Fatal("expected get to fail after delete")
	}
}

func TestManager_Stop_NotRunning(t *testing.T) {
	cp := newMockControlPlane()
	srv := httptest.NewServer(cp.handler())
	defer srv.Close()
	setTestEnv(t, srv.URL)
	m := NewManager()
	if _, err := m.Create("localhost:8080", "stopped", "", "stopped-tunnel"); err != nil {
		t.Fatal(err)
	}
	if err := m.Stop("stopped-tunnel"); err == nil {
		t.Fatal("expected stop error")
	}
}

func TestManager_Start_AlreadyRunning_StateFile(t *testing.T) {
	cp := newMockControlPlane()
	srv := httptest.NewServer(cp.handler())
	defer srv.Close()
	setTestEnv(t, srv.URL)
	m := NewManager()
	if _, err := m.Create("localhost:8080", "dup", "", "dup-tunnel"); err != nil {
		t.Fatal(err)
	}
	cmd := startTunnelHelperProcess(t, "dup-tunnel")
	if err := writeRuntimeState(&RuntimeState{Name: "dup-tunnel", Hostname: "dup.tunnel.example.com", Local: "localhost:8080", PID: cmd.Process.Pid, LogPath: runtimeLogPath("dup-tunnel"), StartedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	defer removeRuntimeState("dup-tunnel")
	if err := m.Start("dup-tunnel", false); err == nil {
		t.Fatal("expected already running error")
	}
}

func TestTailLogs_ReadsRecentLines(t *testing.T) {
	cp := newMockControlPlane()
	srv := httptest.NewServer(cp.handler())
	defer srv.Close()
	setTestEnv(t, srv.URL)
	m := NewManager()
	if err := os.MkdirAll(runtimeDir(), 0755); err != nil {
		t.Fatal(err)
	}
	logPath := runtimeLogPath("logtest")
	if err := os.WriteFile(logPath, []byte("1\n2\n3\n4\n5\n"), 0600); err != nil {
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
	cp := newMockControlPlane()
	srv := httptest.NewServer(cp.handler())
	defer srv.Close()
	setTestEnv(t, srv.URL)
	p := runtimeStatePath("abc")
	if filepath.Ext(p) != ".json" {
		t.Fatalf("state path should end with .json, got %s", p)
	}
}

func TestManager_Stop_UsesRuntimeStatePID(t *testing.T) {
	cp := newMockControlPlane()
	srv := httptest.NewServer(cp.handler())
	defer srv.Close()
	setTestEnv(t, srv.URL)
	cmd := startTunnelHelperProcess(t, "stoptest")
	if err := writeRuntimeState(&RuntimeState{Name: "stoptest", Hostname: "stop.tunnel.example.com", Local: "localhost:8080", PID: cmd.Process.Pid, LogPath: runtimeLogPath("stoptest"), StartedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	m := NewManager()
	if err := m.Stop("stoptest"); err != nil {
		t.Fatal(err)
	}
}

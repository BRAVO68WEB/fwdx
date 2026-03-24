package tunnel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/BRAVO68WEB/fwdx/internal/config"
)

// RuntimeState stores detached tunnel process metadata.
type RuntimeState struct {
	Name      string    `json:"name"`
	Hostname  string    `json:"hostname"`
	Local     string    `json:"local"`
	PID       int       `json:"pid"`
	LogPath   string    `json:"log_path"`
	StartedAt time.Time `json:"started_at"`
}

func runtimeDir() string {
	return filepath.Join(config.GetConfigDir(), "run")
}

func runtimeStatePath(name string) string {
	return filepath.Join(runtimeDir(), name+".json")
}

func runtimeLogPath(name string) string {
	return filepath.Join(runtimeDir(), name+".log")
}

func writeRuntimeState(st *RuntimeState) error {
	if st == nil || st.Name == "" {
		return fmt.Errorf("invalid runtime state")
	}
	if err := os.MkdirAll(runtimeDir(), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := runtimeStatePath(st.Name) + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, runtimeStatePath(st.Name))
}

func readRuntimeState(name string) (*RuntimeState, error) {
	data, err := os.ReadFile(runtimeStatePath(name))
	if err != nil {
		return nil, err
	}
	var st RuntimeState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func removeRuntimeState(name string) {
	_ = os.Remove(runtimeStatePath(name))
}

func isPIDRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}

func runtimeStateIfRunning(name string) (*RuntimeState, bool) {
	st, err := readRuntimeState(name)
	if err != nil {
		return nil, false
	}
	if !isPIDRunning(st.PID) {
		removeRuntimeState(name)
		return nil, false
	}
	return st, true
}

func stopPID(pid int, timeout time.Duration) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := p.Signal(syscall.Signal(15)); err != nil {
		return err
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !isPIDRunning(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = p.Signal(syscall.Signal(9))
	return nil
}

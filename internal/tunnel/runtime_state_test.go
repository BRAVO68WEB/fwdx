package tunnel

import (
	"testing"
	"time"
)

func TestRuntimeState_WriteReadRemove(t *testing.T) {
	name := "x-runtime-state-test"
	removeRuntimeState(name)
	t.Cleanup(func() { removeRuntimeState(name) })

	st := &RuntimeState{
		Name:      name,
		Hostname:  "x.tunnel.example.com",
		Local:     "localhost:8080",
		PID:       1,
		LogPath:   runtimeLogPath(name),
		StartedAt: time.Now(),
	}
	if err := writeRuntimeState(st); err != nil {
		t.Fatal(err)
	}
	got, err := readRuntimeState(name)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != st.Name || got.PID != st.PID {
		t.Fatalf("unexpected state: %+v", got)
	}
	removeRuntimeState(name)
	if _, err := readRuntimeState(name); err == nil {
		t.Fatal("expected not found after remove")
	}
}

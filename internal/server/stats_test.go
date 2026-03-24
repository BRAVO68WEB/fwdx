package server

import (
	"testing"
	"time"
)

func TestStatsStore_RecordAndSnapshot(t *testing.T) {
	s := NewStatsStore()
	s.Record("app.tunnel.myweb.site", "127.0.0.1:1", 100, 200, 200, 25*time.Millisecond, false)
	s.Record("app.tunnel.myweb.site", "127.0.0.1:1", 50, 10, 502, 40*time.Millisecond, true)

	out := s.Snapshot(map[string]string{"app.tunnel.myweb.site": "127.0.0.1:1"})
	if len(out) != 1 {
		t.Fatalf("snapshot len=%d", len(out))
	}
	got := out[0]
	if got.Requests != 2 || got.Errors != 1 {
		t.Fatalf("requests/errors mismatch: %+v", got)
	}
	if got.BytesIn != 150 || got.BytesOut != 210 {
		t.Fatalf("bytes mismatch: %+v", got)
	}
	if !got.Active {
		t.Fatalf("expected active true")
	}
	if got.LatencyAvgMs <= 0 {
		t.Fatalf("expected latency avg > 0")
	}
}

func TestStatsStore_RecentErrors(t *testing.T) {
	s := NewStatsStore()
	s.Record("x", "", 0, 0, 404, 1*time.Millisecond, true)
	n := s.RecentErrors(5 * time.Minute)
	if n != 1 {
		t.Fatalf("recent errors=%d want=1", n)
	}
	n = s.RecentErrors(1 * time.Nanosecond)
	if n != 0 {
		t.Fatalf("recent errors with tiny window=%d want=0", n)
	}
}

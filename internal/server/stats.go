package server

import (
	"sort"
	"sync"
	"time"
)

const (
	statsRetainAfterInactive = 10 * time.Minute
	recentErrorWindow        = 5 * time.Minute
)

// TunnelStats is a live, in-memory snapshot of per-host traffic.
type TunnelStats struct {
	Hostname       string    `json:"hostname"`
	Requests       int64     `json:"requests"`
	Errors         int64     `json:"errors"`
	BytesIn        int64     `json:"bytes_in"`
	BytesOut       int64     `json:"bytes_out"`
	LastStatus     int       `json:"last_status"`
	LastSeen       time.Time `json:"last_seen"`
	LatencyCount   int64     `json:"latency_count"`
	LatencySumMs   int64     `json:"latency_sum_ms"`
	LatencyMaxMs   int64     `json:"latency_max_ms"`
	LatencyAvgMs   int64     `json:"latency_avg_ms"`
	LastRemoteAddr string    `json:"last_remote_addr,omitempty"`
	Active         bool      `json:"active"`
}

type tunnelStat struct {
	TunnelStats
	lastActiveSeen time.Time
}

// StatsStore collects per-tunnel in-memory traffic stats.
type StatsStore struct {
	mu         sync.RWMutex
	byHost     map[string]*tunnelStat
	errorTimes []time.Time
}

func NewStatsStore() *StatsStore {
	return &StatsStore{
		byHost: make(map[string]*tunnelStat),
	}
}

func (s *StatsStore) Record(hostname, remoteAddr string, inBytes, outBytes int, status int, dur time.Duration, isError bool) {
	if hostname == "" {
		return
	}
	now := time.Now()
	ms := dur.Milliseconds()

	s.mu.Lock()
	defer s.mu.Unlock()

	st := s.byHost[hostname]
	if st == nil {
		st = &tunnelStat{TunnelStats: TunnelStats{Hostname: hostname}}
		s.byHost[hostname] = st
	}

	st.Requests++
	st.BytesIn += int64(inBytes)
	st.BytesOut += int64(outBytes)
	st.LastStatus = status
	st.LastSeen = now
	st.LatencyCount++
	st.LatencySumMs += ms
	if ms > st.LatencyMaxMs {
		st.LatencyMaxMs = ms
	}
	if st.LatencyCount > 0 {
		st.LatencyAvgMs = st.LatencySumMs / st.LatencyCount
	}
	if remoteAddr != "" {
		st.LastRemoteAddr = remoteAddr
	}
	if isError {
		st.Errors++
		s.errorTimes = append(s.errorTimes, now)
	}
}

func (s *StatsStore) Snapshot(active map[string]string) []TunnelStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for h, st := range s.byHost {
		if _, ok := active[h]; ok {
			st.Active = true
			st.lastActiveSeen = now
			if addr := active[h]; addr != "" {
				st.LastRemoteAddr = addr
			}
		} else {
			st.Active = false
		}

		if !st.Active && !st.LastSeen.IsZero() && now.Sub(st.LastSeen) > statsRetainAfterInactive {
			delete(s.byHost, h)
		}
	}

	out := make([]TunnelStats, 0, len(s.byHost))
	for _, st := range s.byHost {
		out = append(out, st.TunnelStats)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Hostname < out[j].Hostname })
	return out
}

func (s *StatsStore) RecentErrors(window time.Duration) int {
	if window <= 0 {
		window = recentErrorWindow
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cut := time.Now().Add(-window)
	keep := s.errorTimes[:0]
	for _, t := range s.errorTimes {
		if t.After(cut) {
			keep = append(keep, t)
		}
	}
	s.errorTimes = keep
	return len(s.errorTimes)
}

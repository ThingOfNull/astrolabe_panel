package local

import (
	"sync"
	"time"
)

// netSnapshot stores NIC counters between fetches.
type netSnapshot struct {
	t  time.Time
	rx float64
	tx float64
}

// netRateSampler keeps process-wide NIC state.
type netRateSampler struct {
	mu   sync.Mutex
	last *netSnapshot
}

func (s *netRateSampler) snapshot() (netSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.last == nil {
		return netSnapshot{}, false
	}
	return *s.last, true
}

func (s *netRateSampler) record(t time.Time, rx, tx float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.last = &netSnapshot{t: t, rx: rx, tx: tx}
}

// Package-level net sampler.
var netSampler = &netRateSampler{}

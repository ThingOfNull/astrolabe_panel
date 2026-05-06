package api

import (
	"sync"
	"time"
)

const weatherCacheTTL = 30 * time.Minute

// weatherMemo caches successful upstream JSON by city ID to reduce repeat calls.
type weatherMemo struct {
	mu    sync.Mutex
	items map[int64]weatherCacheEntry
	ttl   time.Duration
}

type weatherCacheEntry struct {
	body      []byte
	expiresAt time.Time
}

func newWeatherMemo() *weatherMemo {
	return &weatherMemo{
		items: make(map[int64]weatherCacheEntry),
		ttl:   weatherCacheTTL,
	}
}

func (m *weatherMemo) get(id int64) ([]byte, bool) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.items[id]
	if !ok {
		return nil, false
	}
	if now.After(e.expiresAt) {
		delete(m.items, id)
		return nil, false
	}
	return e.body, true
}

func (m *weatherMemo) put(id int64, body []byte) {
	dup := append([]byte(nil), body...)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[id] = weatherCacheEntry{
		body:      dup,
		expiresAt: time.Now().Add(m.ttl),
	}
}

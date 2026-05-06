package api

import (
	"testing"
	"time"
)

func TestWeatherMemo_ttl(t *testing.T) {
	t.Parallel()
	m := &weatherMemo{
		items: make(map[int64]weatherCacheEntry),
		ttl:   50 * time.Millisecond,
	}
	m.put(101, []byte(`{"ok":true}`))

	b, ok := m.get(101)
	if !ok || string(b) != `{"ok":true}` {
		t.Fatalf("want cache hit after put, ok=%v b=%s", ok, b)
	}
	time.Sleep(60 * time.Millisecond)
	if _, ok := m.get(101); ok {
		t.Fatal("want cache miss after ttl")
	}
}

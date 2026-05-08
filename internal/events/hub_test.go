package events

import (
	"context"
	"testing"
	"time"
)

func TestHubFanOut(t *testing.T) {
	h := NewHub()
	defer h.Close()

	const n = 4
	subs := make([]*Subscription, n)
	for i := range subs {
		subs[i] = h.Subscribe(context.Background())
	}

	h.Broadcast(Event{Type: TypeProbeChanged, Payload: 1})

	for i, s := range subs {
		select {
		case got := <-s.Events():
			if got.Type != TypeProbeChanged {
				t.Errorf("sub %d: type = %q, want probe.changed", i, got.Type)
			}
		case <-time.After(200 * time.Millisecond):
			t.Fatalf("sub %d: timed out", i)
		}
	}
}

func TestHubSubscribeCancel(t *testing.T) {
	h := NewHub()
	defer h.Close()

	ctx, cancel := context.WithCancel(context.Background())
	sub := h.Subscribe(ctx)
	cancel()

	// Wait for the background goroutine to unsubscribe.
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if h.Count() == 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if h.Count() != 0 {
		t.Errorf("active subs = %d, want 0", h.Count())
	}
	// Draining a closed channel must not panic.
	for range sub.Events() {
	}
}

func TestHubCloseReleasesSubscribers(t *testing.T) {
	h := NewHub()
	sub := h.Subscribe(context.Background())
	h.Close()

	_, ok := <-sub.Events()
	if ok {
		t.Error("expected closed channel after Close")
	}
	// Broadcast after Close must be a no-op (no panic).
	h.Broadcast(Event{Type: TypeProbeChanged})
}

func TestHubDropsOldestOnOverflow(t *testing.T) {
	h := NewHub()
	defer h.Close()
	sub := h.Subscribe(context.Background())

	// Producer overruns the buffer without any consumer reading. The hub's
	// drop-oldest fallback keeps Broadcast non-blocking and retains the
	// newest events in the buffer.
	over := defaultBuffer * 3
	for i := 0; i < over; i++ {
		h.Broadcast(Event{Type: TypeMetricSample, Payload: i})
	}

	// Buffer must be full with the newest events (payload >= over-defaultBuffer).
	if got := len(sub.Events()); got != defaultBuffer {
		t.Fatalf("buffer len = %d, want %d", got, defaultBuffer)
	}
	first := <-sub.Events()
	idx, _ := first.Payload.(int)
	if idx < over-defaultBuffer-1 {
		t.Errorf("oldest retained = %d, want >= %d (drop-oldest kept newest)",
			idx, over-defaultBuffer-1)
	}
}

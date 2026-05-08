// Package events implements an in-process pub/sub hub for server-sent events.
//
// Producers call Broadcast(event) from anywhere (RPC handlers, probe scheduler,
// metric pipeline) without knowing the transport. Consumers (the SSE HTTP
// endpoint) call Subscribe to obtain a bounded channel of events.
//
// The hub is deliberately minimal: no topics/filters; subscribers decide which
// Event.Type they care about. A slow client's channel drops the oldest event
// rather than blocking publishers — realtime panels tolerate gaps, and we must
// never stall the probe scheduler or RPC response path.
package events

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
)

// Type identifies the event kind; values are the SSE `event:` name sent to
// browsers and the handler key on the frontend.
type Type string

const (
	TypeProbeChanged      Type = "probe.changed"
	TypeWidgetCreated     Type = "widget.created"
	TypeWidgetChanged     Type = "widget.changed"
	TypeWidgetDeleted     Type = "widget.deleted"
	TypeBoardChanged      Type = "board.changed"
	TypeDataSourceChanged Type = "datasource.changed"
	TypeMetricSample      Type = "metric.sample"
)

// Event is a single server-originated notification. Payload is marshaled as
// JSON by the SSE transport.
type Event struct {
	Type    Type `json:"type"`
	Payload any  `json:"payload,omitempty"`
}

// defaultBuffer caps the per-subscriber channel. 64 comfortably absorbs an
// editing burst (a drag emits dozens of widget.changed) without stalling; if
// a client is slower than that we prefer lossy over head-of-line blocking.
const defaultBuffer = 64

// Subscription is a reader handle returned from Hub.Subscribe.
type Subscription struct {
	id    uint64
	ch    chan Event
	hub   *Hub
	drops atomic.Int64
}

// Events returns the receive-only channel for the consumer loop.
func (s *Subscription) Events() <-chan Event { return s.ch }

// Drops reports how many events were discarded because this subscriber's
// buffer was full. Useful for diagnostics.
func (s *Subscription) Drops() int64 { return s.drops.Load() }

// Close removes the subscription from the hub and drains/closes its channel.
// Safe to call multiple times.
func (s *Subscription) Close() {
	if s.hub == nil {
		return
	}
	s.hub.unsubscribe(s.id)
	s.hub = nil
}

// Hub fans events out to every active subscriber.
type Hub struct {
	mu     sync.RWMutex
	subs   map[uint64]*Subscription
	nextID atomic.Uint64
	closed atomic.Bool
}

// NewHub constructs an empty hub.
func NewHub() *Hub {
	return &Hub{subs: make(map[uint64]*Subscription)}
}

// Subscribe returns a handle bound to ctx. When ctx is cancelled the
// subscription is released and Events() is closed on the next publish or
// Close() call.
func (h *Hub) Subscribe(ctx context.Context) *Subscription {
	sub := &Subscription{
		id:  h.nextID.Add(1),
		ch:  make(chan Event, defaultBuffer),
		hub: h,
	}

	h.mu.Lock()
	h.subs[sub.id] = sub
	h.mu.Unlock()

	if ctx != nil {
		go func() {
			<-ctx.Done()
			sub.Close()
		}()
	}
	return sub
}

// Broadcast delivers ev to every subscriber. Slow consumers get oldest events
// dropped (see defaultBuffer comment).
func (h *Hub) Broadcast(ev Event) {
	if h == nil || h.closed.Load() {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, sub := range h.subs {
		select {
		case sub.ch <- ev:
		default:
			// Buffer full: drop oldest then retry once, non-blocking.
			select {
			case <-sub.ch:
			default:
			}
			select {
			case sub.ch <- ev:
			default:
				sub.drops.Add(1)
				slog.Warn("events: subscriber buffer overflow",
					"sub_id", sub.id, "type", ev.Type, "drops", sub.drops.Load())
			}
		}
	}
}

// Close terminates the hub and all subscriptions. After Close, Broadcast is a
// no-op and Subscribe returns an already-closed subscription.
func (h *Hub) Close() {
	if !h.closed.CompareAndSwap(false, true) {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, sub := range h.subs {
		close(sub.ch)
		sub.hub = nil
		delete(h.subs, id)
	}
}

func (h *Hub) unsubscribe(id uint64) {
	h.mu.Lock()
	sub, ok := h.subs[id]
	if ok {
		delete(h.subs, id)
	}
	h.mu.Unlock()
	if ok {
		close(sub.ch)
	}
}

// Count returns the number of active subscriptions; intended for metrics.
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs)
}

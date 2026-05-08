package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"astrolabe/internal/events"
)

// sseHeartbeat keeps intermediaries from reaping idle SSE connections.
//
// Chosen below nginx/envoy's common 60s idle default so a well-tuned proxy
// never kills the channel even without explicit config.
const sseHeartbeat = 15 * time.Second

// registerEventRoutes wires GET /api/events. One subscription is created per
// HTTP request; disconnect (client close or proxy timeout) releases it.
//
// Frame format:
//
//	id: <monotonic>
//	event: <events.Type>
//	data: <json>
//
// Heartbeats are sent as SSE comments (": ping") so the browser's EventSource
// never surfaces them to application code.
func registerEventRoutes(r *gin.Engine, hub *events.Hub) {
	r.GET("/api/events", func(c *gin.Context) {
		w := c.Writer
		h := w.Header()
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache, no-transform")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no") // nginx: do not coalesce chunks

		flusher, ok := w.(http.Flusher)
		if !ok {
			c.String(http.StatusInternalServerError, "streaming unsupported")
			return
		}

		sub := hub.Subscribe(c.Request.Context())
		defer sub.Close()

		// Initial hello: gives the browser a chance to set lastEventId and
		// keeps Safari/Firefox from treating the stream as "connecting".
		if _, err := io.WriteString(w, "retry: 2000\n\n"); err != nil {
			return
		}
		flusher.Flush()

		ticker := time.NewTicker(sseHeartbeat)
		defer ticker.Stop()

		var seq uint64
		slog.Info("sse connected", "remote", c.ClientIP())
		defer slog.Info("sse disconnected",
			"remote", c.ClientIP(), "drops", sub.Drops())

		for {
			select {
			case <-c.Request.Context().Done():
				return
			case <-ticker.C:
				if _, err := io.WriteString(w, ": ping\n\n"); err != nil {
					return
				}
				flusher.Flush()
			case ev, alive := <-sub.Events():
				if !alive {
					return
				}
				seq++
				if err := writeSSE(w, seq, ev); err != nil {
					slog.Debug("sse write failed", "err", err, "remote", c.ClientIP())
					return
				}
				flusher.Flush()
			}
		}
	})
}

func writeSSE(w io.Writer, id uint64, ev events.Event) error {
	buf, err := json.Marshal(ev.Payload)
	if err != nil {
		// Best-effort: fall back to a null payload so a bad producer does not
		// kill the whole stream for this subscriber.
		buf = []byte("null")
	}
	// SSE spec: data lines must be prefixed and newline-delimited.
	_, err = fmt.Fprintf(w,
		"id: %s\nevent: %s\ndata: %s\n\n",
		strconv.FormatUint(id, 10), ev.Type, buf)
	return err
}

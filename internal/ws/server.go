package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Default connection timeouts and WS read sizing.
const (
	readDeadline       = 90 * time.Second
	writeDeadline      = 10 * time.Second
	maxReadMsgBytes    = 64 << 20 // Large frame cap; blobs use HTTP upload /api/config/import too
	readBufferGrowthKB = 64       // read buffer growth step when expanding ReadMessage buffers
)

// Server hosts the JSON-RPC registry and websocket upgrades.
type Server struct {
	Registry *Registry
	upgrader websocket.Upgrader
}

// NewServer builds a Server; CheckOrigin accepts all origins (single-node LAN service).
func NewServer(reg *Registry) *Server {
	return &Server{
		Registry: reg,
		upgrader: websocket.Upgrader{
			CheckOrigin:     func(r *http.Request) bool { return true },
			ReadBufferSize:  readBufferGrowthKB * 1024,
			WriteBufferSize: readBufferGrowthKB * 1024,
		},
	}
}

// HandleHTTP is an http.Handler; each accepted socket runs serve in its own goroutine.
//
// Do not derive the WS lifetime from r.Context(): it ends when HandleHTTP returns and would
// cancel in-flight RPCs. Long-lived sockets use context.Background().
func (s *Server) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("ws upgrade failed", "err", err, "remote", r.RemoteAddr)
		return
	}
	go s.serve(context.Background(), conn, r.RemoteAddr)
}

func (s *Server) serve(parent context.Context, conn *websocket.Conn, remote string) {
	defer func() {
		_ = conn.Close()
	}()

	conn.SetReadLimit(maxReadMsgBytes)
	_ = conn.SetReadDeadline(time.Now().Add(readDeadline))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(readDeadline))
	})

	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	writeMu := &sync.Mutex{}
	writeJSON := func(v any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(writeDeadline))
		return conn.WriteJSON(v)
	}

	slog.Info("ws connected", "remote", remote)
	defer slog.Info("ws disconnected", "remote", remote)

	for {
		messageType, raw, err := conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				slog.Debug("ws read end", "remote", remote, "err", err)
			}
			return
		}
		if messageType != websocket.TextMessage {
			continue
		}
		_ = conn.SetReadDeadline(time.Now().Add(readDeadline))

		go s.handleOne(ctx, raw, writeJSON, remote)
	}
}

func (s *Server) handleOne(ctx context.Context, raw []byte, write func(any) error, remote string) {
	var meta struct {
		Method string `json:"method"`
	}
	_ = json.Unmarshal(raw, &meta)

	resp := dispatch(ctx, s.Registry, raw)
	if resp == nil {
		return
	}

	if meta.Method != "" {
		if meta.Method == "ping" {
			slog.Debug("rpc", "remote", remote, "method", meta.Method)
		} else {
			slog.Info("rpc", "remote", remote, "method", meta.Method)
		}
	}
	if resp.Error != nil && meta.Method != "" {
		slog.Warn("rpc error", "remote", remote, "method", meta.Method, "code", resp.Error.Code)
	}

	if err := write(resp); err != nil {
		slog.Warn("ws write failed", "remote", remote, "err", err)
	}
}

// DispatchRaw runs one JSON-RPC frame for tests (no WebSocket).
func (s *Server) DispatchRaw(ctx context.Context, raw []byte) ([]byte, error) {
	resp := dispatch(ctx, s.Registry, raw)
	if resp == nil {
		return nil, nil
	}
	return json.Marshal(resp)
}

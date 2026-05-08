// Package handlers wires JSON-RPC methods onto an rpc.Registry.
//
// Handlers are transport-agnostic: they are registered once at startup and
// invoked by either HTTP (/api/rpc) or test harnesses.
package handlers

import (
	"context"
	"encoding/json"
	"time"

	"astrolabe/internal/rpc"
)

// PingResult is ping RPC payload.
type PingResult struct {
	Pong bool  `json:"pong"`
	Ts   int64 `json:"ts"`
}

// RegisterSystem exposes ping.
func RegisterSystem(reg *rpc.Registry) {
	reg.Register("ping", func(_ context.Context, _ json.RawMessage) (any, error) {
		return PingResult{Pong: true, Ts: time.Now().Unix()}, nil
	})
}

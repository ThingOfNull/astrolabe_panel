// Package handlers wires JSON-RPC methods.
package handlers

import (
	"context"
	"encoding/json"
	"time"

	"astrolabe/internal/ws"
)

// PingResult is ping RPC payload.
type PingResult struct {
	Pong bool  `json:"pong"`
	Ts   int64 `json:"ts"`
}

// RegisterSystem exposes ping.
func RegisterSystem(reg *ws.Registry) {
	reg.Register("ping", func(_ context.Context, _ json.RawMessage) (any, error) {
		return PingResult{Pong: true, Ts: time.Now().Unix()}, nil
	})
}

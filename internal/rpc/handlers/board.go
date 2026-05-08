package handlers

import (
	"context"
	"encoding/json"
	"errors"

	"gorm.io/gorm"

	"astrolabe/internal/events"
	"astrolabe/internal/rpc"
	"astrolabe/internal/store"
)

// RegisterBoard exposes board RPC. When hub is non-nil, successful writes
// broadcast "board.changed" so connected clients can refresh without polling.
func RegisterBoard(reg *rpc.Registry, s *store.Store, hub *events.Hub) {
	reg.Register("board.get", func(ctx context.Context, _ json.RawMessage) (any, error) {
		b, err := s.GetBoard(ctx, store.DefaultBoardID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, rpc.NewAppError(rpc.ErrCodeBoardNotFound, nil)
			}
			return nil, rpc.NewAppError(rpc.ErrCodeInternal, err.Error())
		}
		return b, nil
	})
	reg.Register("board.update", func(ctx context.Context, raw json.RawMessage) (any, error) {
		var in store.BoardUpdateInput
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, rpc.NewError(rpc.CodeInvalidParams, err.Error(), nil)
		}
		b, err := s.UpdateBoard(ctx, store.DefaultBoardID, in)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, rpc.NewAppError(rpc.ErrCodeBoardNotFound, nil)
			}
			return nil, rpc.NewAppError(rpc.ErrCodeValidation, err.Error())
		}
		if hub != nil {
			hub.Broadcast(events.Event{Type: events.TypeBoardChanged, Payload: b})
		}
		return b, nil
	})
}

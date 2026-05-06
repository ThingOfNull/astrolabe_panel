package handlers

import (
	"context"
	"encoding/json"
	"errors"

	"gorm.io/gorm"

	"astrolabe/internal/store"
	"astrolabe/internal/ws"
)

// RegisterBoard exposes board RPC.
func RegisterBoard(reg *ws.Registry, s *store.Store) {
	reg.Register("board.get", func(ctx context.Context, _ json.RawMessage) (any, error) {
		b, err := s.GetBoard(ctx, store.DefaultBoardID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ws.NewError(ws.CodeServerErrorRangeEnd, "default board not found", nil)
			}
			return nil, err
		}
		return b, nil
	})
	reg.Register("board.update", func(ctx context.Context, raw json.RawMessage) (any, error) {
		var in store.BoardUpdateInput
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, ws.NewError(ws.CodeInvalidParams, err.Error(), nil)
		}
		b, err := s.UpdateBoard(ctx, store.DefaultBoardID, in)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ws.NewError(ws.CodeServerErrorRangeEnd, "default board not found", nil)
			}
			return nil, ws.NewError(ws.CodeInvalidParams, err.Error(), nil)
		}
		return b, nil
	})
}

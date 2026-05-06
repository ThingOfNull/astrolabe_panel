package handlers

import (
	"context"
	"encoding/json"
	"errors"

	"astrolabe/internal/store"
	"astrolabe/internal/ws"
)

// RegisterWidget exposes widget RPC.
func RegisterWidget(reg *ws.Registry, s *store.Store) {
	reg.Register("widget.list", widgetList(s))
	reg.Register("widget.create", widgetCreate(s))
	reg.Register("widget.update", widgetUpdate(s))
	reg.Register("widget.delete", widgetDelete(s))
	reg.Register("widget.batchUpdate", widgetBatchUpdate(s))
}

type listParams struct {
	BoardID *int64 `json:"board_id"`
}

type listResult struct {
	Items []store.WidgetView `json:"items"`
}

func widgetList(s *store.Store) ws.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		p := listParams{}
		if len(raw) > 0 && string(raw) != "null" {
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, ws.NewError(ws.CodeInvalidParams, err.Error(), nil)
			}
		}
		board := store.DefaultBoardID
		if p.BoardID != nil {
			board = *p.BoardID
		}
		items, err := s.ListWidgets(ctx, board)
		if err != nil {
			return nil, err
		}
		return listResult{Items: items}, nil
	}
}

func widgetCreate(s *store.Store) ws.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var in store.WidgetInput
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, ws.NewError(ws.CodeInvalidParams, err.Error(), nil)
		}
		out, err := s.CreateWidget(ctx, in)
		return mapErr(out, err)
	}
}

func widgetUpdate(s *store.Store) ws.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		// Manual double decode: encoding/json cannot merge embedded structs inline.
		envelope := struct {
			ID int64 `json:"id"`
		}{}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return nil, ws.NewError(ws.CodeInvalidParams, err.Error(), nil)
		}
		if envelope.ID <= 0 {
			return nil, ws.NewError(ws.CodeInvalidParams, "id required", nil)
		}
		var in store.WidgetInput
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, ws.NewError(ws.CodeInvalidParams, err.Error(), nil)
		}
		out, err := s.UpdateWidget(ctx, envelope.ID, in)
		return mapErr(out, err)
	}
}

type deleteParams struct {
	ID int64 `json:"id"`
}

func widgetDelete(s *store.Store) ws.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var p deleteParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, ws.NewError(ws.CodeInvalidParams, err.Error(), nil)
		}
		if p.ID <= 0 {
			return nil, ws.NewError(ws.CodeInvalidParams, "id required", nil)
		}
		if err := s.DeleteWidget(ctx, p.ID); err != nil {
			return nil, mapStoreErr(err)
		}
		return map[string]bool{"ok": true}, nil
	}
}

type batchParams struct {
	Items []store.WidgetBatchPatch `json:"items"`
}

func widgetBatchUpdate(s *store.Store) ws.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var p batchParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, ws.NewError(ws.CodeInvalidParams, err.Error(), nil)
		}
		items, err := s.BatchUpdateWidgets(ctx, p.Items)
		if err != nil {
			return nil, mapStoreErr(err)
		}
		return listResult{Items: items}, nil
	}
}

// mapErr converts store errors to JSON-RPC codes.
func mapErr(v *store.WidgetView, err error) (any, error) {
	if err != nil {
		return nil, mapStoreErr(err)
	}
	return v, nil
}

// App error band -32000..-32099.
const (
	codeWidgetNotFound = -32010
	codeWidgetOverlap  = -32011
	codeWidgetInvalid  = -32012
)

func mapStoreErr(err error) *ws.RPCError {
	switch {
	case errors.Is(err, store.ErrWidgetNotFound):
		return ws.NewError(codeWidgetNotFound, err.Error(), nil)
	case errors.Is(err, store.ErrWidgetOverlap):
		return ws.NewError(codeWidgetOverlap, err.Error(), nil)
	case errors.Is(err, store.ErrInvalidWidgetType),
		errors.Is(err, store.ErrInvalidCoordinate),
		errors.Is(err, store.ErrInvalidIconType),
		errors.Is(err, store.ErrInvalidURLProtocol),
		errors.Is(err, store.ErrInvalidConfigJSON),
		errors.Is(err, store.ErrInvalidMetricJSON):
		return ws.NewError(codeWidgetInvalid, err.Error(), nil)
	default:
		return ws.NewError(ws.CodeInternalError, err.Error(), nil)
	}
}

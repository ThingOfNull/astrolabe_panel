package handlers

import (
	"context"
	"encoding/json"
	"errors"

	"astrolabe/internal/events"
	"astrolabe/internal/rpc"
	"astrolabe/internal/store"
)

// RegisterWidget exposes widget RPC. Successful mutations are broadcast on hub
// (widget.created / widget.changed / widget.deleted) so other connected
// clients observe them without polling.
func RegisterWidget(reg *rpc.Registry, s *store.Store, hub *events.Hub) {
	reg.Register("widget.list", widgetList(s))
	reg.Register("widget.create", widgetCreate(s, hub))
	reg.Register("widget.update", widgetUpdate(s, hub))
	reg.Register("widget.delete", widgetDelete(s, hub))
	reg.Register("widget.batchUpdate", widgetBatchUpdate(s, hub))
}

type listParams struct {
	BoardID *int64 `json:"board_id"`
}

type listResult struct {
	Items []store.WidgetView `json:"items"`
}

func widgetList(s *store.Store) rpc.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		p := listParams{}
		if len(raw) > 0 && string(raw) != "null" {
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, rpc.NewError(rpc.CodeInvalidParams, err.Error(), nil)
			}
		}
		board := store.DefaultBoardID
		if p.BoardID != nil {
			board = *p.BoardID
		}
		items, err := s.ListWidgets(ctx, board)
		if err != nil {
			return nil, rpc.NewAppError(rpc.ErrCodeInternal, err.Error())
		}
		return listResult{Items: items}, nil
	}
}

func widgetCreate(s *store.Store, hub *events.Hub) rpc.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var in store.WidgetInput
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, rpc.NewError(rpc.CodeInvalidParams, err.Error(), nil)
		}
		out, err := s.CreateWidget(ctx, in)
		if err != nil {
			return nil, mapStoreErr(err)
		}
		if hub != nil && out != nil {
			hub.Broadcast(events.Event{Type: events.TypeWidgetCreated, Payload: out})
		}
		return out, nil
	}
}

func widgetUpdate(s *store.Store, hub *events.Hub) rpc.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		// Manual double decode: encoding/json cannot merge embedded structs inline.
		envelope := struct {
			ID int64 `json:"id"`
		}{}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return nil, rpc.NewError(rpc.CodeInvalidParams, err.Error(), nil)
		}
		if envelope.ID <= 0 {
			return nil, rpc.NewError(rpc.CodeInvalidParams, "id required", nil)
		}
		var in store.WidgetInput
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, rpc.NewError(rpc.CodeInvalidParams, err.Error(), nil)
		}
		out, err := s.UpdateWidget(ctx, envelope.ID, in)
		if err != nil {
			return nil, mapStoreErr(err)
		}
		if hub != nil && out != nil {
			hub.Broadcast(events.Event{Type: events.TypeWidgetChanged, Payload: out})
		}
		return out, nil
	}
}

type deleteParams struct {
	ID int64 `json:"id"`
}

func widgetDelete(s *store.Store, hub *events.Hub) rpc.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var p deleteParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, rpc.NewError(rpc.CodeInvalidParams, err.Error(), nil)
		}
		if p.ID <= 0 {
			return nil, rpc.NewError(rpc.CodeInvalidParams, "id required", nil)
		}
		if err := s.DeleteWidget(ctx, p.ID); err != nil {
			return nil, mapStoreErr(err)
		}
		if hub != nil {
			hub.Broadcast(events.Event{Type: events.TypeWidgetDeleted, Payload: map[string]int64{"id": p.ID}})
		}
		return map[string]bool{"ok": true}, nil
	}
}

type batchParams struct {
	Items []store.WidgetBatchPatch `json:"items"`
}

func widgetBatchUpdate(s *store.Store, hub *events.Hub) rpc.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var p batchParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, rpc.NewError(rpc.CodeInvalidParams, err.Error(), nil)
		}
		items, err := s.BatchUpdateWidgets(ctx, p.Items)
		if err != nil {
			return nil, mapStoreErr(err)
		}
		if hub != nil {
			// Fan out individual changed events; subscribers merge by id.
			for i := range items {
				hub.Broadcast(events.Event{Type: events.TypeWidgetChanged, Payload: items[i]})
			}
		}
		return listResult{Items: items}, nil
	}
}

// mapStoreErr converts store domain errors to coded RPC errors suitable for
// client-side i18n.
func mapStoreErr(err error) *rpc.RPCError {
	switch {
	case errors.Is(err, store.ErrWidgetNotFound):
		return rpc.NewAppError(rpc.ErrCodeWidgetNotFound, err.Error())
	case errors.Is(err, store.ErrWidgetOverlap):
		return rpc.NewAppError(rpc.ErrCodeWidgetOverlap, err.Error())
	case errors.Is(err, store.ErrInvalidWidgetType),
		errors.Is(err, store.ErrInvalidCoordinate),
		errors.Is(err, store.ErrInvalidIconType),
		errors.Is(err, store.ErrInvalidURLProtocol),
		errors.Is(err, store.ErrInvalidConfigJSON),
		errors.Is(err, store.ErrInvalidMetricJSON):
		return rpc.NewAppError(rpc.ErrCodeWidgetInvalid, err.Error())
	default:
		return rpc.NewAppError(rpc.ErrCodeInternal, err.Error())
	}
}

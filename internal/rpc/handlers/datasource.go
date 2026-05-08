package handlers

import (
	"context"
	"encoding/json"
	"errors"

	"astrolabe/internal/adapter"
	"astrolabe/internal/core/datasource"
	"astrolabe/internal/events"
	"astrolabe/internal/rpc"
	"astrolabe/internal/store"
)

// RegisterDataSource exposes datasource RPC. Successful writes broadcast
// "datasource.changed" so clients can update their pickers immediately.
func RegisterDataSource(reg *rpc.Registry, s *store.Store, mgr *datasource.Manager, hub *events.Hub) {
	reg.Register("datasource.types", dsTypes(mgr))
	reg.Register("datasource.list", dsList(s))
	reg.Register("datasource.create", dsCreate(s, mgr, hub))
	reg.Register("datasource.update", dsUpdate(s, mgr, hub))
	reg.Register("datasource.delete", dsDelete(s, mgr, hub))
	reg.Register("datasource.testConnect", dsTestConnect(mgr))
	reg.Register("datasource.discover", dsDiscover(mgr))
}

type dsListResult struct {
	Items []store.DataSourceView `json:"items"`
}

type dsTypesResult struct {
	Types []string `json:"types"`
}

func dsTypes(mgr *datasource.Manager) rpc.HandlerFn {
	return func(_ context.Context, _ json.RawMessage) (any, error) {
		return dsTypesResult{Types: mgr.Types()}, nil
	}
}

func dsList(s *store.Store) rpc.HandlerFn {
	return func(ctx context.Context, _ json.RawMessage) (any, error) {
		items, err := s.ListDataSources(ctx)
		if err != nil {
			return nil, rpc.NewAppError(rpc.ErrCodeInternal, err.Error())
		}
		return dsListResult{Items: items}, nil
	}
}

func dsCreate(s *store.Store, _ *datasource.Manager, hub *events.Hub) rpc.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var in store.DataSourceInput
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, rpc.NewError(rpc.CodeInvalidParams, err.Error(), nil)
		}
		out, err := s.CreateDataSource(ctx, in)
		if err != nil {
			return nil, mapDSErr(err)
		}
		if hub != nil && out != nil {
			hub.Broadcast(events.Event{
				Type:    events.TypeDataSourceChanged,
				Payload: map[string]any{"id": out.ID, "op": "upsert", "view": out},
			})
		}
		return out, nil
	}
}

type dsUpdateParams struct {
	ID int64 `json:"id"`
}

func dsUpdate(s *store.Store, mgr *datasource.Manager, hub *events.Hub) rpc.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		envelope := dsUpdateParams{}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return nil, rpc.NewError(rpc.CodeInvalidParams, err.Error(), nil)
		}
		if envelope.ID <= 0 {
			return nil, rpc.NewError(rpc.CodeInvalidParams, "id required", nil)
		}
		var in store.DataSourceInput
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, rpc.NewError(rpc.CodeInvalidParams, err.Error(), nil)
		}
		out, err := s.UpdateDataSource(ctx, envelope.ID, in)
		if err != nil {
			return nil, mapDSErr(err)
		}
		mgr.Forget(envelope.ID) // config changed; drop cache
		if hub != nil && out != nil {
			hub.Broadcast(events.Event{
				Type:    events.TypeDataSourceChanged,
				Payload: map[string]any{"id": out.ID, "op": "upsert", "view": out},
			})
		}
		return out, nil
	}
}

type dsIDParams struct {
	ID int64 `json:"id"`
}

func dsDelete(s *store.Store, mgr *datasource.Manager, hub *events.Hub) rpc.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var p dsIDParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, rpc.NewError(rpc.CodeInvalidParams, err.Error(), nil)
		}
		if p.ID <= 0 {
			return nil, rpc.NewError(rpc.CodeInvalidParams, "id required", nil)
		}
		if err := s.DeleteDataSource(ctx, p.ID); err != nil {
			return nil, mapDSErr(err)
		}
		mgr.Forget(p.ID)
		if hub != nil {
			hub.Broadcast(events.Event{
				Type:    events.TypeDataSourceChanged,
				Payload: map[string]any{"id": p.ID, "op": "delete"},
			})
		}
		return map[string]bool{"ok": true}, nil
	}
}

type dsTestConnectParams struct {
	ID       *int64           `json:"id"`
	Name     *string          `json:"name"`
	Type     *string          `json:"type"`
	Endpoint *string          `json:"endpoint"`
	Auth     *json.RawMessage `json:"auth"`
	Extra    *json.RawMessage `json:"extra"`
}

type dsTestConnectResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// dsTestConnect supports inline or saved configs.
func dsTestConnect(mgr *datasource.Manager) rpc.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var p dsTestConnectParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, rpc.NewError(rpc.CodeInvalidParams, err.Error(), nil)
		}
		if p.ID != nil && *p.ID > 0 {
			_, err := mgr.HealthCheck(ctx, *p.ID)
			if err != nil {
				return dsTestConnectResult{OK: false, Error: err.Error()}, nil
			}
			return dsTestConnectResult{OK: true}, nil
		}
		if p.Type == nil {
			return nil, rpc.NewError(rpc.CodeInvalidParams, "type required when id missing", nil)
		}
		view := &store.DataSourceView{
			Name:     coalesceStrPtr(p.Name, "test"),
			Type:     *p.Type,
			Endpoint: coalesceStrPtr(p.Endpoint, ""),
			Auth:     toRawJSON(p.Auth),
			Extra:    toRawJSON(p.Extra),
		}
		if err := mgr.TestConnect(ctx, view); err != nil {
			return dsTestConnectResult{OK: false, Error: err.Error()}, nil
		}
		return dsTestConnectResult{OK: true}, nil
	}
}

type dsDiscoverParams struct {
	ID int64 `json:"id"`
}

type dsDiscoverResult struct {
	Tree adapter.MetricTree `json:"tree"`
}

func dsDiscover(mgr *datasource.Manager) rpc.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var p dsDiscoverParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, rpc.NewError(rpc.CodeInvalidParams, err.Error(), nil)
		}
		if p.ID <= 0 {
			return nil, rpc.NewError(rpc.CodeInvalidParams, "id required", nil)
		}
		tree, err := mgr.Discover(ctx, p.ID)
		if err != nil {
			return nil, rpc.NewAppError(rpc.ErrCodeDataSourceConnect, err.Error())
		}
		return dsDiscoverResult{Tree: tree}, nil
	}
}

func mapDSErr(err error) *rpc.RPCError {
	switch {
	case errors.Is(err, store.ErrDataSourceNotFound):
		return rpc.NewAppError(rpc.ErrCodeDataSourceNotFound, err.Error())
	case errors.Is(err, store.ErrDataSourceInvalid):
		return rpc.NewAppError(rpc.ErrCodeDataSourceInvalid, err.Error())
	default:
		return rpc.NewAppError(rpc.ErrCodeInternal, err.Error())
	}
}

func coalesceStrPtr(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}

func toRawJSON(p *json.RawMessage) json.RawMessage {
	if p == nil {
		return json.RawMessage("null")
	}
	return *p
}

package handlers

import (
	"context"
	"encoding/json"
	"errors"

	"astrolabe/internal/adapter"
	"astrolabe/internal/core/datasource"
	"astrolabe/internal/store"
	"astrolabe/internal/ws"
)

// App error band -32000..-32099.
const (
	codeDSNotFound      = -32020
	codeDSInvalid       = -32021
	codeDSConnectFailed = -32022
)

// RegisterDataSource exposes datasource RPC.
func RegisterDataSource(reg *ws.Registry, s *store.Store, mgr *datasource.Manager) {
	reg.Register("datasource.types", dsTypes(mgr))
	reg.Register("datasource.list", dsList(s))
	reg.Register("datasource.create", dsCreate(s, mgr))
	reg.Register("datasource.update", dsUpdate(s, mgr))
	reg.Register("datasource.delete", dsDelete(s, mgr))
	reg.Register("datasource.testConnect", dsTestConnect(mgr))
	reg.Register("datasource.discover", dsDiscover(mgr))
}

type dsListResult struct {
	Items []store.DataSourceView `json:"items"`
}

type dsTypesResult struct {
	Types []string `json:"types"`
}

func dsTypes(mgr *datasource.Manager) ws.HandlerFn {
	return func(_ context.Context, _ json.RawMessage) (any, error) {
		return dsTypesResult{Types: mgr.Types()}, nil
	}
}

func dsList(s *store.Store) ws.HandlerFn {
	return func(ctx context.Context, _ json.RawMessage) (any, error) {
		items, err := s.ListDataSources(ctx)
		if err != nil {
			return nil, err
		}
		return dsListResult{Items: items}, nil
	}
}

func dsCreate(s *store.Store, _ *datasource.Manager) ws.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var in store.DataSourceInput
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, ws.NewError(ws.CodeInvalidParams, err.Error(), nil)
		}
		out, err := s.CreateDataSource(ctx, in)
		if err != nil {
			return nil, mapDSErr(err)
		}
		return out, nil
	}
}

type dsUpdateParams struct {
	ID int64 `json:"id"`
}

func dsUpdate(s *store.Store, mgr *datasource.Manager) ws.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		envelope := dsUpdateParams{}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return nil, ws.NewError(ws.CodeInvalidParams, err.Error(), nil)
		}
		if envelope.ID <= 0 {
			return nil, ws.NewError(ws.CodeInvalidParams, "id required", nil)
		}
		var in store.DataSourceInput
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, ws.NewError(ws.CodeInvalidParams, err.Error(), nil)
		}
		out, err := s.UpdateDataSource(ctx, envelope.ID, in)
		if err != nil {
			return nil, mapDSErr(err)
		}
		mgr.Forget(envelope.ID) // config changed; drop cache
		return out, nil
	}
}

type dsIDParams struct {
	ID int64 `json:"id"`
}

func dsDelete(s *store.Store, mgr *datasource.Manager) ws.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var p dsIDParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, ws.NewError(ws.CodeInvalidParams, err.Error(), nil)
		}
		if p.ID <= 0 {
			return nil, ws.NewError(ws.CodeInvalidParams, "id required", nil)
		}
		if err := s.DeleteDataSource(ctx, p.ID); err != nil {
			return nil, mapDSErr(err)
		}
		mgr.Forget(p.ID)
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
func dsTestConnect(mgr *datasource.Manager) ws.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var p dsTestConnectParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, ws.NewError(ws.CodeInvalidParams, err.Error(), nil)
		}
		if p.ID != nil && *p.ID > 0 {
			_, err := mgr.HealthCheck(ctx, *p.ID)
			if err != nil {
				return dsTestConnectResult{OK: false, Error: err.Error()}, nil
			}
			return dsTestConnectResult{OK: true}, nil
		}
		if p.Type == nil {
			return nil, ws.NewError(ws.CodeInvalidParams, "type required when id missing", nil)
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

func dsDiscover(mgr *datasource.Manager) ws.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var p dsDiscoverParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, ws.NewError(ws.CodeInvalidParams, err.Error(), nil)
		}
		if p.ID <= 0 {
			return nil, ws.NewError(ws.CodeInvalidParams, "id required", nil)
		}
		tree, err := mgr.Discover(ctx, p.ID)
		if err != nil {
			return nil, ws.NewError(codeDSConnectFailed, err.Error(), nil)
		}
		return dsDiscoverResult{Tree: tree}, nil
	}
}

func mapDSErr(err error) *ws.RPCError {
	switch {
	case errors.Is(err, store.ErrDataSourceNotFound):
		return ws.NewError(codeDSNotFound, err.Error(), nil)
	case errors.Is(err, store.ErrDataSourceInvalid):
		return ws.NewError(codeDSInvalid, err.Error(), nil)
	default:
		return ws.NewError(ws.CodeInternalError, err.Error(), nil)
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

package handlers

import (
	"context"
	"encoding/json"

	"astrolabe/internal/probe"
	"astrolabe/internal/rpc"
	"astrolabe/internal/store/models"
)

// RegisterProbe exposes probe RPC.
//
// With SSE push (probe.changed) now available, the list endpoint remains only
// as a bootstrap call: clients fetch the full set on first load and then
// consume delta events instead of polling.
func RegisterProbe(reg *rpc.Registry, sc *probe.Scheduler) {
	reg.Register("probe.status", probeStatus(sc))
}

type probeStatusParams struct {
	WidgetIDs []int64 `json:"widget_ids"`
}

type probeStatusResult struct {
	Items []models.ProbeStatus `json:"items"`
}

func probeStatus(sc *probe.Scheduler) rpc.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		p := probeStatusParams{}
		if len(raw) > 0 && string(raw) != "null" {
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, rpc.NewError(rpc.CodeInvalidParams, err.Error(), nil)
			}
		}
		items, err := sc.ListStatuses(ctx, p.WidgetIDs)
		if err != nil {
			return nil, rpc.NewAppError(rpc.ErrCodeInternal, err.Error())
		}
		return probeStatusResult{Items: items}, nil
	}
}

package handlers

import (
	"context"
	"encoding/json"

	"astrolabe/internal/probe"
	"astrolabe/internal/store/models"
	"astrolabe/internal/ws"
)

// RegisterProbe exposes probe RPC.
func RegisterProbe(reg *ws.Registry, sc *probe.Scheduler) {
	reg.Register("probe.status", probeStatus(sc))
}

type probeStatusParams struct {
	WidgetIDs []int64 `json:"widget_ids"`
}

type probeStatusResult struct {
	Items []models.ProbeStatus `json:"items"`
}

func probeStatus(sc *probe.Scheduler) ws.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		p := probeStatusParams{}
		if len(raw) > 0 && string(raw) != "null" {
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, ws.NewError(ws.CodeInvalidParams, err.Error(), nil)
			}
		}
		items, err := sc.ListStatuses(ctx, p.WidgetIDs)
		if err != nil {
			return nil, err
		}
		return probeStatusResult{Items: items}, nil
	}
}

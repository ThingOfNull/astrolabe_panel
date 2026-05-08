package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"astrolabe/internal/core/iconify"
	"astrolabe/internal/rpc"
)

// RegisterIconify exposes icon helpers.
func RegisterIconify(reg *rpc.Registry, p *iconify.Proxy) {
	reg.Register("iconify.search", func(ctx context.Context, raw json.RawMessage) (any, error) {
		var in struct {
			Query string `json:"query"`
			Limit int    `json:"limit"`
		}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &in); err != nil {
				return nil, rpc.NewError(rpc.CodeInvalidParams, err.Error(), nil)
			}
		}
		out, err := p.Search(ctx, in.Query, in.Limit)
		if err != nil {
			return nil, rpc.NewAppError(rpc.ErrCodeIconifyFailed, err.Error())
		}
		return out, nil
	})
	reg.Register("iconify.icon", func(ctx context.Context, raw json.RawMessage) (any, error) {
		var in struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, rpc.NewError(rpc.CodeInvalidParams, err.Error(), nil)
		}
		buf, err := p.GetSVG(ctx, in.ID)
		if err != nil {
			return nil, rpc.NewAppError(rpc.ErrCodeIconifyFailed, err.Error())
		}
		return map[string]any{
			"id":       in.ID,
			"svg":      string(buf),
			"data_url": "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(buf),
		}, nil
	})
}

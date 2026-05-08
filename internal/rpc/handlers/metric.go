package handlers

import (
	"context"
	"encoding/json"

	"astrolabe/internal/adapter"
	"astrolabe/internal/core/metric"
	"astrolabe/internal/rpc"
)

// RegisterMetric exposes metric RPC. The metric.sample push channel handles
// incremental updates; metric.fetch remains only for cold-start hydration.
func RegisterMetric(reg *rpc.Registry, p *metric.Pipeline) {
	reg.Register("metric.fetch", metricFetch(p))
	reg.Register("metric.fetchBatch", metricFetchBatch(p))
}

type metricFetchParams struct {
	DataSourceID int64               `json:"data_source_id"`
	Query        adapter.MetricQuery `json:"query"`
}

func metricFetch(p *metric.Pipeline) rpc.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var in metricFetchParams
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, rpc.NewError(rpc.CodeInvalidParams, err.Error(), nil)
		}
		resp, err := p.Fetch(ctx, metric.Request{
			DataSourceID: in.DataSourceID,
			Query:        in.Query,
		})
		if err != nil {
			return nil, rpc.NewAppError(rpc.ErrCodeMetricFetchFailed, err.Error())
		}
		return resp, nil
	}
}

type metricFetchBatchParams struct {
	Items []metricFetchParams `json:"items"`
}

type metricFetchBatchResult struct {
	Items []metric.BatchItem `json:"items"`
}

func metricFetchBatch(p *metric.Pipeline) rpc.HandlerFn {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var in metricFetchBatchParams
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, rpc.NewError(rpc.CodeInvalidParams, err.Error(), nil)
		}
		reqs := make([]metric.Request, len(in.Items))
		for i, it := range in.Items {
			reqs[i] = metric.Request{DataSourceID: it.DataSourceID, Query: it.Query}
		}
		items, err := p.FetchBatch(ctx, reqs)
		if err != nil {
			return nil, rpc.NewAppError(rpc.ErrCodeMetricFetchFailed, err.Error())
		}
		return metricFetchBatchResult{Items: items}, nil
	}
}

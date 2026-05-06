// Package metric implements metric.fetch / fetchBatch pipeline.
//
// Responsibilities:
//   - Route queries through datasource manager;
//   - Persist scalars into metric_samples;
//   - For SQL-backed TimeSeries ingest a scalar refresh first,
//     then QuerySamples assembles TimeSeries payloads,
//   - Coalesce identical fetches + 1s memoization.
package metric

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"astrolabe/internal/adapter"
	"astrolabe/internal/core/datasource"
	"astrolabe/internal/store"
)

// Request models one fetch input.
type Request struct {
	DataSourceID int64
	Query        adapter.MetricQuery
}

// Response wraps fetch output.
type Response struct {
	DataSourceID int64               `json:"data_source_id"`
	Path         string              `json:"path"`
	Shape        adapter.Shape       `json:"shape"`
	Payload      adapter.DataPayload `json:"payload"`
	CachedAt     int64               `json:"cached_at"`
}

// Pipeline is stateless except coalescing.
type Pipeline struct {
	manager *datasource.Manager
	store   *store.Store

	cacheTTL time.Duration

	mu       sync.Mutex
	cache    map[string]cacheEntry
	inflight map[string]*inflightEntry
}

type cacheEntry struct {
	resp   Response
	expire time.Time
}

type inflightEntry struct {
	done chan struct{}
	resp Response
	err  error
}

// New wires pipeline.
func New(mgr *datasource.Manager, s *store.Store) *Pipeline {
	return &Pipeline{
		manager:  mgr,
		store:    s,
		cacheTTL: 1 * time.Second,
		cache:    map[string]cacheEntry{},
		inflight: map[string]*inflightEntry{},
	}
}

// Fetch handles one widget query.
func (p *Pipeline) Fetch(ctx context.Context, req Request) (Response, error) {
	if req.DataSourceID <= 0 {
		return Response{}, errors.New("metric: data_source_id required")
	}
	if req.Query.Path == "" {
		return Response{}, errors.New("metric: path required")
	}
	if req.Query.Shape == "" {
		return Response{}, errors.New("metric: shape required")
	}
	if !adapter.IsValidShape(string(req.Query.Shape)) {
		return Response{}, fmt.Errorf("metric: bad shape %q", req.Query.Shape)
	}

	key := cacheKey(req)
	if resp, ok := p.cacheGet(key); ok {
		return resp, nil
	}

	// In-flight merge shares the first waiter result.
	p.mu.Lock()
	if e, ok := p.inflight[key]; ok {
		p.mu.Unlock()
		select {
		case <-e.done:
			return e.resp, e.err
		case <-ctx.Done():
			return Response{}, ctx.Err()
		}
	}
	entry := &inflightEntry{done: make(chan struct{})}
	p.inflight[key] = entry
	p.mu.Unlock()

	resp, err := p.doFetch(ctx, req)
	entry.resp, entry.err = resp, err
	close(entry.done)

	p.mu.Lock()
	delete(p.inflight, key)
	if err == nil {
		p.cache[key] = cacheEntry{resp: resp, expire: time.Now().Add(p.cacheTTL)}
	}
	p.mu.Unlock()
	return resp, err
}

// FetchBatch walks many bindings.
//
// Serial loop today; cache + merge apply per item.
func (p *Pipeline) FetchBatch(ctx context.Context, reqs []Request) ([]BatchItem, error) {
	out := make([]BatchItem, len(reqs))
	for i, req := range reqs {
		resp, err := p.Fetch(ctx, req)
		out[i] = BatchItem{
			DataSourceID: req.DataSourceID,
			Path:         req.Query.Path,
			Shape:        req.Query.Shape,
		}
		if err != nil {
			out[i].Error = err.Error()
			continue
		}
		out[i].Payload = resp.Payload
	}
	return out, nil
}

// BatchItem keeps string errors isolated.
type BatchItem struct {
	DataSourceID int64               `json:"data_source_id"`
	Path         string              `json:"path"`
	Shape        adapter.Shape       `json:"shape"`
	Payload      adapter.DataPayload `json:"payload,omitempty"`
	Error        string              `json:"error,omitempty"`
}

// doFetch runs adapter + persistence.
//
// TimeSeries handling order:
//
//  1. Prefer native adapter series.
//  2. If ErrUnsupportedShape, ingest scalars first
//     then aggregate from SQL (local path).
func (p *Pipeline) doFetch(ctx context.Context, req Request) (Response, error) {
	q := req.Query
	now := time.Now()
	if q.Shape == adapter.ShapeTimeSeries {
		payload, err := p.manager.Fetch(ctx, req.DataSourceID, q)
		if err == nil {
			if vErr := payload.Validate(); vErr == nil && payload.TimeSeries != nil {
				return Response{
					DataSourceID: req.DataSourceID,
					Path:         q.Path,
					Shape:        q.Shape,
					Payload:      payload,
					CachedAt:     now.Unix(),
				}, nil
			}
		} else if !errors.Is(err, adapter.ErrUnsupportedShape) {
			return Response{}, err
		}
		// Fallback: sample scalars then query SQL.
		scalarQuery := q
		scalarQuery.Shape = adapter.ShapeScalar
		if scalarPayload, err := p.manager.Fetch(ctx, req.DataSourceID, scalarQuery); err == nil {
			p.persistScalar(ctx, req.DataSourceID, q, scalarPayload, now)
		}
		tsPayload, err := p.assembleTimeSeries(ctx, req.DataSourceID, q)
		if err != nil {
			return Response{}, err
		}
		return Response{
			DataSourceID: req.DataSourceID,
			Path:         q.Path,
			Shape:        q.Shape,
			Payload: adapter.DataPayload{
				Shape:      adapter.ShapeTimeSeries,
				TimeSeries: tsPayload,
			},
			CachedAt: now.Unix(),
		}, nil
	}

	payload, err := p.manager.Fetch(ctx, req.DataSourceID, q)
	if err != nil {
		return Response{}, err
	}
	if err := payload.Validate(); err != nil {
		return Response{}, err
	}
	p.persistScalar(ctx, req.DataSourceID, q, payload, now)
	return Response{
		DataSourceID: req.DataSourceID,
		Path:         q.Path,
		Shape:        q.Shape,
		Payload:      payload,
		CachedAt:     now.Unix(),
	}, nil
}

// persistScalar stores scalars; other shapes skip SQL history.
func (p *Pipeline) persistScalar(ctx context.Context, dsID int64, q adapter.MetricQuery, payload adapter.DataPayload, now time.Time) {
	if payload.Shape != adapter.ShapeScalar || payload.Scalar == nil {
		return
	}
	dim := q.Dim
	if dim == "" {
		dim = "_"
	}
	ts := payload.Scalar.TS
	if ts <= 0 {
		ts = now.Unix()
	}
	if err := p.store.InsertSamples(ctx, []store.SampleInsert{{
		DataSourceID: dsID,
		MetricPath:   q.Path,
		Dim:          dim,
		Ts:           ts,
		Value:        payload.Scalar.Value,
	}}); err != nil {
		// DB write errors are non-fatal to responses.
	}
}

// assembleTimeSeries groups SQL samples by dim.
func (p *Pipeline) assembleTimeSeries(ctx context.Context, dsID int64, q adapter.MetricQuery) (*adapter.TimeSeriesPayload, error) {
	window := int64(q.WindowSec)
	if window <= 0 {
		window = 1800
	}
	rows, err := p.store.QuerySamples(ctx, dsID, q.Path, window)
	if err != nil {
		return nil, err
	}
	groups := make(map[string][]adapter.TimeSeriesPoint)
	for _, r := range rows {
		groups[r.Dim] = append(groups[r.Dim], adapter.TimeSeriesPoint{float64(r.Ts), r.Value})
	}
	names := make([]string, 0, len(groups))
	for k := range groups {
		names = append(names, k)
	}
	sort.Strings(names)
	out := &adapter.TimeSeriesPayload{}
	for _, n := range names {
		out.Series = append(out.Series, adapter.TimeSeriesSeries{
			Name:   n,
			Points: groups[n],
		})
	}
	return out, nil
}

func (p *Pipeline) cacheGet(key string) (Response, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	e, ok := p.cache[key]
	if !ok {
		return Response{}, false
	}
	if time.Now().After(e.expire) {
		delete(p.cache, key)
		return Response{}, false
	}
	return e.resp, true
}

func cacheKey(r Request) string {
	return fmt.Sprintf("%d|%s|%s|%d|%d|%s",
		r.DataSourceID, r.Query.Path, r.Query.Shape, r.Query.WindowSec, r.Query.Points, r.Query.Dim,
	)
}

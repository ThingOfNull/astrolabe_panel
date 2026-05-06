package metric

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"astrolabe/internal/adapter"
	_ "astrolabe/internal/adapter/local"
	"astrolabe/internal/core/datasource"
	"astrolabe/internal/store"
)

func setup(t *testing.T) (*Pipeline, *store.Store, int64) {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(context.Background(), store.Options{DBPath: filepath.Join(dir, "p.db")})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	mgr := datasource.NewManager(s, nil)
	t.Cleanup(mgr.Close)

	typ := "local"
	name := "local-1"
	view, err := s.CreateDataSource(context.Background(), store.DataSourceInput{
		Name: &name, Type: &typ,
	})
	if err != nil {
		t.Fatalf("CreateDataSource: %v", err)
	}
	return New(mgr, s), s, view.ID
}

func TestFetchScalarPersists(t *testing.T) {
	pipe, s, dsID := setup(t)
	resp, err := pipe.Fetch(context.Background(), Request{
		DataSourceID: dsID,
		Query:        adapter.MetricQuery{Path: "system/load/1m", Shape: adapter.ShapeScalar},
	})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if resp.Payload.Shape != adapter.ShapeScalar {
		t.Errorf("shape = %q", resp.Payload.Shape)
	}
	rows, err := s.QuerySamples(context.Background(), dsID, "system/load/1m", 60)
	if err != nil {
		t.Fatalf("QuerySamples: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 sample, got %d", len(rows))
	}
}

func TestFetchTimeSeriesAssemblesFromSamples(t *testing.T) {
	pipe, s, dsID := setup(t)
	// Seed a few metric samples manually.
	now := time.Now().Unix()
	if err := s.InsertSamples(context.Background(), []store.SampleInsert{
		{DataSourceID: dsID, MetricPath: "system/cpu/total", Dim: "_", Ts: now - 30, Value: 12.0},
		{DataSourceID: dsID, MetricPath: "system/cpu/total", Dim: "_", Ts: now - 15, Value: 18.0},
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	resp, err := pipe.Fetch(context.Background(), Request{
		DataSourceID: dsID,
		Query: adapter.MetricQuery{
			Path: "system/cpu/total", Shape: adapter.ShapeTimeSeries, WindowSec: 600,
		},
	})
	if err != nil {
		t.Fatalf("fetch ts: %v", err)
	}
	if resp.Payload.Shape != adapter.ShapeTimeSeries {
		t.Fatalf("shape = %q", resp.Payload.Shape)
	}
	if resp.Payload.TimeSeries == nil || len(resp.Payload.TimeSeries.Series) == 0 {
		t.Fatal("expected non-empty series")
	}
	pts := resp.Payload.TimeSeries.Series[0].Points
	if len(pts) < 2 {
		t.Errorf("expected at least 2 points (preexisting samples), got %d", len(pts))
	}
}

func TestFetchCacheCoalescing(t *testing.T) {
	pipe, _, dsID := setup(t)
	q := adapter.MetricQuery{Path: "system/load/1m", Shape: adapter.ShapeScalar}
	r1, err := pipe.Fetch(context.Background(), Request{DataSourceID: dsID, Query: q})
	if err != nil {
		t.Fatalf("fetch1: %v", err)
	}
	r2, err := pipe.Fetch(context.Background(), Request{DataSourceID: dsID, Query: q})
	if err != nil {
		t.Fatalf("fetch2: %v", err)
	}
	if r1.CachedAt != r2.CachedAt {
		t.Errorf("expected cached response within ttl, got %d vs %d", r1.CachedAt, r2.CachedAt)
	}
}

func TestFetchInvalidShape(t *testing.T) {
	pipe, _, dsID := setup(t)
	_, err := pipe.Fetch(context.Background(), Request{
		DataSourceID: dsID,
		Query:        adapter.MetricQuery{Path: "system/load/1m", Shape: "Histogram"},
	})
	if err == nil {
		t.Fatal("expected error for bad shape")
	}
}

func TestCleanupSamplesRespectsWindow(t *testing.T) {
	_, s, dsID := setup(t)
	now := time.Now()
	if err := s.InsertSamples(context.Background(), []store.SampleInsert{
		{DataSourceID: dsID, MetricPath: "x", Dim: "_", Ts: now.Add(-2 * time.Hour).Unix(), Value: 1},
		{DataSourceID: dsID, MetricPath: "x", Dim: "_", Ts: now.Add(-1 * time.Minute).Unix(), Value: 2},
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	deleted, err := s.CleanupSamples(context.Background(), 30)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}
}

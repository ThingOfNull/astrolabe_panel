package local

import (
	"context"
	"errors"
	"strings"
	"testing"

	"astrolabe/internal/adapter"
)

func TestLocalDiscover(t *testing.T) {
	ds, err := New(adapter.Config{Type: Type})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer ds.Close()

	tree, err := ds.Discover(context.Background())
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(tree.Roots) == 0 {
		t.Fatalf("expected non-empty tree")
	}
	// Sanity-check known metric paths exist.
	want := []string{
		"system/cpu/total", "system/cpu/per_core",
		"system/mem/used_pct", "system/mem/used_mb",
		"system/load/1m",
		"system/disk/used_by_mount",
		"system/net/rx_bps", "system/net/tx_bps",
	}
	got := flattenLeafPaths(tree.Roots)
	for _, w := range want {
		if !contains(got, w) {
			t.Errorf("missing leaf %q in %v", w, got)
		}
	}
}

func TestLocalFetchScalar(t *testing.T) {
	ds, _ := New(adapter.Config{Type: Type})
	defer ds.Close()

	cases := []string{
		"system/cpu/total",
		"system/mem/used_pct",
		"system/mem/used_mb",
		"system/mem/total_mb",
		"system/load/1m",
	}
	for _, p := range cases {
		t.Run(p, func(t *testing.T) {
			payload, err := ds.Fetch(context.Background(), adapter.MetricQuery{Path: p, Shape: adapter.ShapeScalar})
			if err != nil {
				t.Fatalf("fetch %s: %v", p, err)
			}
			if err := payload.Validate(); err != nil {
				t.Fatalf("payload invalid: %v", err)
			}
			if payload.Scalar == nil {
				t.Fatal("scalar nil")
			}
		})
	}
}

func TestLocalFetchCategorical(t *testing.T) {
	ds, _ := New(adapter.Config{Type: Type})
	defer ds.Close()

	for _, p := range []string{"system/cpu/per_core", "system/disk/used_by_mount"} {
		t.Run(p, func(t *testing.T) {
			payload, err := ds.Fetch(context.Background(), adapter.MetricQuery{Path: p, Shape: adapter.ShapeCategorical})
			if err != nil {
				t.Fatalf("fetch %s: %v", p, err)
			}
			if err := payload.Validate(); err != nil {
				t.Fatalf("invalid: %v", err)
			}
			if payload.Categorical == nil {
				t.Fatal("categorical nil")
			}
		})
	}
}

func TestLocalFetchUnsupported(t *testing.T) {
	ds, _ := New(adapter.Config{Type: Type})
	defer ds.Close()

	_, err := ds.Fetch(context.Background(), adapter.MetricQuery{Path: "no/such/path", Shape: adapter.ShapeScalar})
	if !errors.Is(err, adapter.ErrUnsupportedPath) {
		t.Errorf("expected ErrUnsupportedPath, got %v", err)
	}
	_, err = ds.Fetch(context.Background(), adapter.MetricQuery{Path: "system/cpu/total", Shape: adapter.ShapeTimeSeries})
	if !errors.Is(err, adapter.ErrUnsupportedShape) {
		t.Errorf("expected ErrUnsupportedShape, got %v", err)
	}
}

func TestLocalHealthCheck(t *testing.T) {
	ds, _ := New(adapter.Config{Type: Type})
	defer ds.Close()
	if err := ds.HealthCheck(context.Background()); err != nil {
		t.Errorf("health: %v", err)
	}
}

func flattenLeafPaths(nodes []adapter.MetricNode) []string {
	out := []string{}
	for _, n := range nodes {
		if n.Leaf {
			out = append(out, n.Path)
		}
		out = append(out, flattenLeafPaths(n.Children)...)
	}
	return out
}

func contains(arr []string, s string) bool {
	for _, x := range arr {
		if x == s || strings.EqualFold(x, s) {
			return true
		}
	}
	return false
}

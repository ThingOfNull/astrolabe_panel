package netdata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"astrolabe/internal/adapter"
)

// fakeServer stubs Netdata HTTP endpoints.
func fakeServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/info", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"version":"v1.0"}`))
	})
	mux.HandleFunc("/api/v1/charts", func(w http.ResponseWriter, r *http.Request) {
		body := `{
            "charts": {
                "system.cpu": {"id":"system.cpu","name":"system.cpu","title":"CPU Usage","family":"cpu","context":"system.cpu","units":"%",
                    "dimensions":{"user":{"name":"user"},"system":{"name":"system"}}},
                "system.ram": {"id":"system.ram","name":"system.ram","title":"RAM","family":"mem","context":"system.ram","units":"MiB",
                    "dimensions":{"used":{"name":"used"},"free":{"name":"free"}}}
            }
        }`
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})
	mux.HandleFunc("/api/v1/data", func(w http.ResponseWriter, r *http.Request) {
		chart := r.URL.Query().Get("chart")
		points := r.URL.Query().Get("points")
		var resp any
		switch chart {
		case "system.cpu":
			if points == "1" {
				resp = map[string]any{
					"labels": []string{"time", "user", "system"},
					"data":   [][]float64{{1700000000, 12.5, 4.2}},
				}
			} else {
				// Points ordered desc like Netdata
				resp = map[string]any{
					"labels": []string{"time", "user", "system"},
					"data":   [][]float64{{1700000060, 13.0, 4.5}, {1700000030, 12.0, 4.3}, {1700000000, 11.0, 4.1}},
				}
			}
		case "system.ram":
			resp = map[string]any{
				"labels": []string{"time", "used", "free"},
				"data":   [][]float64{{1700000000, 1024, 2048}},
			}
		default:
			http.Error(w, "no chart", http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	return httptest.NewServer(mux)
}

func TestNetdataDiscover(t *testing.T) {
	srv := fakeServer(t)
	defer srv.Close()

	ds, err := New(adapter.Config{Type: Type, Endpoint: srv.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer ds.Close()
	if err := ds.HealthCheck(context.Background()); err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}
	tree, err := ds.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(tree.Roots) != 1 {
		t.Fatalf("roots = %d", len(tree.Roots))
	}
	got := dumpLeafPaths(tree.Roots)
	for _, want := range []string{"netdata/system.cpu", "netdata/system.ram"} {
		if !contains(got, want) {
			t.Errorf("missing leaf %q in %v", want, got)
		}
	}
}

func TestNetdataFetchScalar(t *testing.T) {
	srv := fakeServer(t)
	defer srv.Close()
	ds, _ := New(adapter.Config{Type: Type, Endpoint: srv.URL})
	defer ds.Close()

	payload, err := ds.Fetch(context.Background(), adapter.MetricQuery{Path: "netdata/system.cpu", Shape: adapter.ShapeScalar})
	if err != nil {
		t.Fatalf("fetch scalar: %v", err)
	}
	if payload.Scalar == nil || payload.Scalar.Value != 12.5 {
		t.Errorf("payload = %+v", payload.Scalar)
	}
}

func TestNetdataFetchTimeSeriesAscending(t *testing.T) {
	srv := fakeServer(t)
	defer srv.Close()
	ds, _ := New(adapter.Config{Type: Type, Endpoint: srv.URL})
	defer ds.Close()

	payload, err := ds.Fetch(context.Background(), adapter.MetricQuery{
		Path: "netdata/system.cpu", Shape: adapter.ShapeTimeSeries, WindowSec: 600, Points: 60,
	})
	if err != nil {
		t.Fatalf("fetch ts: %v", err)
	}
	if payload.TimeSeries == nil || len(payload.TimeSeries.Series) != 2 {
		t.Fatalf("series = %+v", payload.TimeSeries)
	}
	pts := payload.TimeSeries.Series[0].Points
	if len(pts) != 3 {
		t.Fatalf("points = %d", len(pts))
	}
	// Expect ascending after reverse
	if pts[0][0] >= pts[2][0] {
		t.Errorf("expected ascending ts, got %v", pts)
	}
}

func TestNetdataFetchCategorical(t *testing.T) {
	srv := fakeServer(t)
	defer srv.Close()
	ds, _ := New(adapter.Config{Type: Type, Endpoint: srv.URL})
	defer ds.Close()

	payload, err := ds.Fetch(context.Background(), adapter.MetricQuery{Path: "netdata/system.ram", Shape: adapter.ShapeCategorical})
	if err != nil {
		t.Fatalf("fetch cat: %v", err)
	}
	if payload.Categorical == nil || len(payload.Categorical.Items) != 2 {
		t.Fatalf("cat = %+v", payload.Categorical)
	}
}

func TestNetdataEndpointRequired(t *testing.T) {
	_, err := New(adapter.Config{Type: Type})
	if err == nil || !strings.Contains(err.Error(), "endpoint required") {
		t.Errorf("expected endpoint error, got %v", err)
	}
}

func dumpLeafPaths(nodes []adapter.MetricNode) []string {
	out := []string{}
	for _, n := range nodes {
		if n.Leaf {
			out = append(out, n.Path)
		}
		out = append(out, dumpLeafPaths(n.Children)...)
	}
	return out
}

func contains(arr []string, s string) bool {
	for _, x := range arr {
		if x == s {
			return true
		}
	}
	return false
}

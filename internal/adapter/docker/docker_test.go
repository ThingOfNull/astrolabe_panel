package dockerds

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types/container"

	"astrolabe/internal/adapter"
)

func TestDockerDiscover(t *testing.T) {
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
		t.Fatal("expected roots")
	}
	leaves := flattenLeafPaths(tree.Roots)
	want := []string{
		"docker/containers/list",
		"docker/containers/cpu_top",
		"docker/containers/mem_top",
		"docker/containers/running_count",
	}
	for _, w := range want {
		if !contains(leaves, w) {
			t.Errorf("missing leaf %q in %v", w, leaves)
		}
	}
}

func TestContainerStatusMapping(t *testing.T) {
	cases := map[string]string{
		"running":    "ok",
		"paused":     "warn",
		"restarting": "warn",
		"exited":     "down",
		"dead":       "down",
		"created":    "unknown",
	}
	for in, want := range cases {
		if got := containerStatus(in); got != want {
			t.Errorf("containerStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPrimaryName(t *testing.T) {
	if got := primaryName([]string{"/web", "/web-alias"}); got != "web" {
		t.Errorf("got %q", got)
	}
	if got := primaryName(nil); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestCalcCPUPercent(t *testing.T) {
	s := &container.StatsResponse{}
	s.CPUStats.CPUUsage.TotalUsage = 200
	s.PreCPUStats.CPUUsage.TotalUsage = 100
	s.CPUStats.SystemUsage = 1000
	s.PreCPUStats.SystemUsage = 500
	s.CPUStats.OnlineCPUs = 4
	got := calcCPUPercent(s)
	want := 80.0 // (100/500)*4*100
	if got != want {
		t.Errorf("calcCPUPercent = %v, want %v", got, want)
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
		if x == s {
			return true
		}
	}
	return false
}

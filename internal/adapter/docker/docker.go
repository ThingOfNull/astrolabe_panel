// Package dockerds implements the Docker Engine datasource adapter.
//
// Defaults to /var/run/docker.sock; Endpoint can switch to TCP (TLS via docker client).
package dockerds

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	"astrolabe/internal/adapter"
)

// Type holds the datasource type key.
const Type = "docker"

func init() {
	adapter.Register(Type, New)
}

// New builds Docker adapter using default socket when Endpoint empty.
func New(cfg adapter.Config) (adapter.DataSource, error) {
	return &dockerDS{cfg: cfg}, nil
}

type dockerDS struct {
	cfg adapter.Config

	mu     sync.RWMutex
	cli    *client.Client
	closed bool
}

// Connect builds or returns cached docker client.
func (d *dockerDS) Connect(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return errors.New("docker: data source already closed")
	}
	if d.cli != nil {
		return nil
	}
	opts := []client.Opt{client.WithAPIVersionNegotiation()}
	if d.cfg.Endpoint != "" {
		opts = append(opts, client.WithHost(d.cfg.Endpoint))
	} else {
		opts = append(opts, client.FromEnv)
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	d.cli = cli
	return nil
}

// HealthCheck issues Docker Ping.
func (d *dockerDS) HealthCheck(ctx context.Context) error {
	if err := d.ensureConnected(ctx); err != nil {
		return err
	}
	d.mu.RLock()
	cli := d.cli
	d.mu.RUnlock()
	_, err := cli.Ping(ctx)
	return err
}

// Close disposes docker client.
func (d *dockerDS) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.closed = true
	if d.cli != nil {
		err := d.cli.Close()
		d.cli = nil
		return err
	}
	return nil
}

func (d *dockerDS) ensureConnected(ctx context.Context) error {
	d.mu.RLock()
	if d.cli != nil {
		d.mu.RUnlock()
		return nil
	}
	d.mu.RUnlock()
	return d.Connect(ctx)
}

// Discover lists docker metrics.
func (d *dockerDS) Discover(ctx context.Context) (adapter.MetricTree, error) {
	if err := ctx.Err(); err != nil {
		return adapter.MetricTree{}, err
	}
	scalar := []adapter.Shape{adapter.ShapeScalar, adapter.ShapeTimeSeries}
	cat := []adapter.Shape{adapter.ShapeCategorical}
	entity := []adapter.Shape{adapter.ShapeEntityList}

	tree := adapter.MetricTree{
		Roots: []adapter.MetricNode{
			{
				Path: "docker", Label: "Docker", Leaf: false,
				Children: []adapter.MetricNode{
					{Path: "docker/containers/list", Label: "容器列表", Shapes: entity, Leaf: true},
					{Path: "docker/containers/cpu_top", Label: "容器 CPU 占用 Top", Unit: "%", Shapes: cat, Leaf: true},
					{Path: "docker/containers/mem_top", Label: "容器内存占用 Top", Unit: "MiB", Shapes: cat, Leaf: true},
					{Path: "docker/containers/running_count", Label: "运行中容器数", Shapes: scalar, Leaf: true},
				},
			},
		},
	}
	return tree, nil
}

// Fetch returns one metric.
func (d *dockerDS) Fetch(ctx context.Context, q adapter.MetricQuery) (adapter.DataPayload, error) {
	if err := d.ensureConnected(ctx); err != nil {
		return adapter.DataPayload{}, err
	}
	d.mu.RLock()
	cli := d.cli
	d.mu.RUnlock()
	if cli == nil {
		return adapter.DataPayload{}, adapter.ErrNotConnected
	}

	switch q.Path {
	case "docker/containers/list":
		return d.containersList(ctx, cli, q.Shape)
	case "docker/containers/cpu_top":
		return d.containersCPUTop(ctx, cli, q.Shape)
	case "docker/containers/mem_top":
		return d.containersMemTop(ctx, cli, q.Shape)
	case "docker/containers/running_count":
		return d.runningCount(ctx, cli, q.Shape)
	default:
		return adapter.DataPayload{}, fmt.Errorf("%w: %q", adapter.ErrUnsupportedPath, q.Path)
	}
}

func (d *dockerDS) containersList(ctx context.Context, cli *client.Client, want adapter.Shape) (adapter.DataPayload, error) {
	if want != adapter.ShapeEntityList {
		return adapter.DataPayload{}, fmt.Errorf("%w: %q", adapter.ErrUnsupportedShape, want)
	}
	cs, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return adapter.DataPayload{}, err
	}
	items := make([]adapter.EntityListItem, 0, len(cs))
	for _, c := range cs {
		items = append(items, adapter.EntityListItem{
			ID:     c.ID,
			Label:  primaryName(c.Names),
			Status: containerStatus(c.State),
			Extra: map[string]any{
				"image":   c.Image,
				"state":   c.State,
				"status":  c.Status,
				"names":   c.Names,
				"created": c.Created,
			},
		})
	}
	return adapter.DataPayload{
		Shape:      adapter.ShapeEntityList,
		EntityList: &adapter.EntityListPayload{Items: items},
	}, nil
}

func (d *dockerDS) containersCPUTop(ctx context.Context, cli *client.Client, want adapter.Shape) (adapter.DataPayload, error) {
	if want != adapter.ShapeCategorical {
		return adapter.DataPayload{}, fmt.Errorf("%w: %q", adapter.ErrUnsupportedShape, want)
	}
	stats, err := collectStats(ctx, cli)
	if err != nil {
		return adapter.DataPayload{}, err
	}
	items := make([]adapter.CategoricalItem, 0, len(stats))
	for _, s := range stats {
		items = append(items, adapter.CategoricalItem{Label: s.name, Value: s.cpuPct})
	}
	topN(items, 10)
	return adapter.DataPayload{
		Shape:       adapter.ShapeCategorical,
		Categorical: &adapter.CategoricalPayload{Unit: "%", Items: items},
	}, nil
}

func (d *dockerDS) containersMemTop(ctx context.Context, cli *client.Client, want adapter.Shape) (adapter.DataPayload, error) {
	if want != adapter.ShapeCategorical {
		return adapter.DataPayload{}, fmt.Errorf("%w: %q", adapter.ErrUnsupportedShape, want)
	}
	stats, err := collectStats(ctx, cli)
	if err != nil {
		return adapter.DataPayload{}, err
	}
	items := make([]adapter.CategoricalItem, 0, len(stats))
	for _, s := range stats {
		items = append(items, adapter.CategoricalItem{Label: s.name, Value: s.memMiB})
	}
	topN(items, 10)
	return adapter.DataPayload{
		Shape:       adapter.ShapeCategorical,
		Categorical: &adapter.CategoricalPayload{Unit: "MiB", Items: items},
	}, nil
}

func (d *dockerDS) runningCount(ctx context.Context, cli *client.Client, want adapter.Shape) (adapter.DataPayload, error) {
	if want != adapter.ShapeScalar {
		return adapter.DataPayload{}, fmt.Errorf("%w: %q", adapter.ErrUnsupportedShape, want)
	}
	cs, err := cli.ContainerList(ctx, container.ListOptions{All: false})
	if err != nil {
		return adapter.DataPayload{}, err
	}
	return adapter.DataPayload{
		Shape:  adapter.ShapeScalar,
		Scalar: &adapter.ScalarPayload{Value: float64(len(cs)), TS: time.Now().Unix()},
	}, nil
}

// containerStatusEntry caches one docker stats reading.
type containerStatusEntry struct {
	name   string
	cpuPct float64
	memMiB float64
}

// collectStats snapshots running container stats.
func collectStats(ctx context.Context, cli *client.Client) ([]containerStatusEntry, error) {
	cs, err := cli.ContainerList(ctx, container.ListOptions{All: false})
	if err != nil {
		return nil, err
	}
	out := make([]containerStatusEntry, 0, len(cs))
	for _, c := range cs {
		entry := containerStatusEntry{name: primaryName(c.Names)}
		stats, err := cli.ContainerStatsOneShot(ctx, c.ID)
		if err != nil {
			continue
		}
		var raw container.StatsResponse
		dec := json.NewDecoder(stats.Body)
		if err := dec.Decode(&raw); err == nil {
			entry.cpuPct = calcCPUPercent(&raw)
			entry.memMiB = float64(raw.MemoryStats.Usage) / 1024 / 1024
		}
		_ = stats.Body.Close()
		_, _ = io.Copy(io.Discard, stats.Body)
		out = append(out, entry)
	}
	return out, nil
}

// calcCPUPercent matches docker stats CPU.
func calcCPUPercent(s *container.StatsResponse) float64 {
	cpuDelta := float64(s.CPUStats.CPUUsage.TotalUsage) - float64(s.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(s.CPUStats.SystemUsage) - float64(s.PreCPUStats.SystemUsage)
	if systemDelta <= 0 || cpuDelta <= 0 {
		return 0
	}
	online := float64(s.CPUStats.OnlineCPUs)
	if online == 0 {
		online = float64(len(s.CPUStats.CPUUsage.PercpuUsage))
	}
	if online == 0 {
		online = 1
	}
	return (cpuDelta / systemDelta) * online * 100.0
}

// containerStatus maps docker health to entity status strings.
func containerStatus(state string) string {
	switch state {
	case "running":
		return "ok"
	case "restarting", "paused":
		return "warn"
	case "exited", "dead":
		return "down"
	default:
		return "unknown"
	}
}

func primaryName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return strings.TrimPrefix(names[0], "/")
}

// topN sorts by value desc and caps length unless n<=0.
func topN(items []adapter.CategoricalItem, n int) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].Value > items[i].Value {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	if n > 0 && len(items) > n {
		items = items[:n]
	}
}

// Package local implements the host system datasource adapter.
//
// Design notes:
//   - Emits instant shapes (Scalar / Categorical / EntityList).
//   - TimeSeries rolls up via metric_samples + pipeline,
//     this adapter keeps no history.
//   - Built with gopsutil for Linux / macOS / Windows.
package local

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"

	"astrolabe/internal/adapter"
)

// Type is the datasource type literal.
const Type = "local"

func init() {
	adapter.Register(Type, New)
}

// New builds host adapter; ignores endpoint/auth.
func New(_ adapter.Config) (adapter.DataSource, error) {
	return &localDS{}, nil
}

type localDS struct{}

// Connect is no-op.
func (l *localDS) Connect(ctx context.Context) error {
	return ctx.Err()
}

// HealthCheck probes by reading CPU counters.
func (l *localDS) HealthCheck(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := cpu.PercentWithContext(ctx, 0, false)
	return err
}

// Close is no-op.
func (l *localDS) Close() error { return nil }

// Discover emits the static metric catalog.
func (l *localDS) Discover(ctx context.Context) (adapter.MetricTree, error) {
	if err := ctx.Err(); err != nil {
		return adapter.MetricTree{}, err
	}
	scalarPlusTS := []adapter.Shape{adapter.ShapeScalar, adapter.ShapeTimeSeries}
	scalarOnly := []adapter.Shape{adapter.ShapeScalar}
	cat := []adapter.Shape{adapter.ShapeCategorical}

	tree := adapter.MetricTree{
		Roots: []adapter.MetricNode{
			{
				Path: "system", Label: "本机", Leaf: false,
				Children: []adapter.MetricNode{
					{Path: "system/cpu", Label: "CPU", Leaf: false, Children: []adapter.MetricNode{
						{Path: "system/cpu/total", Label: "总使用率", Unit: "%", Shapes: scalarPlusTS, Leaf: true},
						{Path: "system/cpu/per_core", Label: "各核心使用率", Unit: "%", Shapes: cat, Leaf: true},
					}},
					{Path: "system/mem", Label: "内存", Leaf: false, Children: []adapter.MetricNode{
						{Path: "system/mem/used_pct", Label: "已用百分比", Unit: "%", Shapes: scalarPlusTS, Leaf: true},
						{Path: "system/mem/used_mb", Label: "已用容量", Unit: "MiB", Shapes: scalarPlusTS, Leaf: true},
						{Path: "system/mem/total_mb", Label: "总容量", Unit: "MiB", Shapes: scalarOnly, Leaf: true},
					}},
					{Path: "system/load", Label: "Load Average", Leaf: false, Children: []adapter.MetricNode{
						{Path: "system/load/1m", Label: "1 分钟", Shapes: scalarPlusTS, Leaf: true},
						{Path: "system/load/5m", Label: "5 分钟", Shapes: scalarOnly, Leaf: true},
						{Path: "system/load/15m", Label: "15 分钟", Shapes: scalarOnly, Leaf: true},
					}},
					{Path: "system/disk", Label: "磁盘", Leaf: false, Children: []adapter.MetricNode{
						{Path: "system/disk/used_by_mount", Label: "各挂载点已用容量", Unit: "GiB", Shapes: cat, Leaf: true},
					}},
					{Path: "system/net", Label: "网络", Leaf: false, Children: []adapter.MetricNode{
						{Path: "system/net/rx_bps", Label: "接收速率", Unit: "B/s", Shapes: scalarPlusTS, Leaf: true},
						{Path: "system/net/tx_bps", Label: "发送速率", Unit: "B/s", Shapes: scalarPlusTS, Leaf: true},
					}},
				},
			},
		},
	}
	return tree, nil
}

// Fetch returns instant values; TimeSeries comes from SQL rollup.
func (l *localDS) Fetch(ctx context.Context, q adapter.MetricQuery) (adapter.DataPayload, error) {
	if err := ctx.Err(); err != nil {
		return adapter.DataPayload{}, err
	}
	now := time.Now().Unix()
	switch q.Path {
	case "system/cpu/total":
		return l.cpuTotal(ctx, q.Shape, now)
	case "system/cpu/per_core":
		return l.cpuPerCore(ctx, q.Shape)
	case "system/mem/used_pct":
		return l.memUsedPct(ctx, q.Shape, now)
	case "system/mem/used_mb":
		return l.memUsedMB(ctx, q.Shape, now)
	case "system/mem/total_mb":
		return l.memTotalMB(ctx, q.Shape, now)
	case "system/load/1m", "system/load/5m", "system/load/15m":
		return l.loadAvg(ctx, q.Path, q.Shape, now)
	case "system/disk/used_by_mount":
		return l.diskUsedByMount(ctx, q.Shape)
	case "system/net/rx_bps", "system/net/tx_bps":
		return l.netRate(ctx, q.Path, q.Shape, now)
	default:
		return adapter.DataPayload{}, fmt.Errorf("%w: %q", adapter.ErrUnsupportedPath, q.Path)
	}
}

func mustScalar(want adapter.Shape) error {
	if want != adapter.ShapeScalar {
		return fmt.Errorf("%w: shape=%q", adapter.ErrUnsupportedShape, want)
	}
	return nil
}

func (l *localDS) cpuTotal(ctx context.Context, want adapter.Shape, ts int64) (adapter.DataPayload, error) {
	if err := mustScalar(want); err != nil {
		return adapter.DataPayload{}, err
	}
	pcts, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return adapter.DataPayload{}, err
	}
	v := 0.0
	if len(pcts) > 0 {
		v = pcts[0]
	}
	return adapter.DataPayload{
		Shape:  adapter.ShapeScalar,
		Scalar: &adapter.ScalarPayload{Value: v, Unit: "%", TS: ts},
	}, nil
}

func (l *localDS) cpuPerCore(ctx context.Context, want adapter.Shape) (adapter.DataPayload, error) {
	if want != adapter.ShapeCategorical {
		return adapter.DataPayload{}, fmt.Errorf("%w: %q", adapter.ErrUnsupportedShape, want)
	}
	pcts, err := cpu.PercentWithContext(ctx, 0, true)
	if err != nil {
		return adapter.DataPayload{}, err
	}
	items := make([]adapter.CategoricalItem, len(pcts))
	for i, p := range pcts {
		items[i] = adapter.CategoricalItem{Label: fmt.Sprintf("core-%d", i), Value: p}
	}
	return adapter.DataPayload{
		Shape:       adapter.ShapeCategorical,
		Categorical: &adapter.CategoricalPayload{Unit: "%", Items: items},
	}, nil
}

func (l *localDS) memUsedPct(ctx context.Context, want adapter.Shape, ts int64) (adapter.DataPayload, error) {
	if err := mustScalar(want); err != nil {
		return adapter.DataPayload{}, err
	}
	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return adapter.DataPayload{}, err
	}
	return adapter.DataPayload{
		Shape:  adapter.ShapeScalar,
		Scalar: &adapter.ScalarPayload{Value: v.UsedPercent, Unit: "%", TS: ts},
	}, nil
}

func (l *localDS) memUsedMB(ctx context.Context, want adapter.Shape, ts int64) (adapter.DataPayload, error) {
	if err := mustScalar(want); err != nil {
		return adapter.DataPayload{}, err
	}
	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return adapter.DataPayload{}, err
	}
	return adapter.DataPayload{
		Shape:  adapter.ShapeScalar,
		Scalar: &adapter.ScalarPayload{Value: float64(v.Used) / 1024 / 1024, Unit: "MiB", TS: ts},
	}, nil
}

func (l *localDS) memTotalMB(ctx context.Context, want adapter.Shape, ts int64) (adapter.DataPayload, error) {
	if err := mustScalar(want); err != nil {
		return adapter.DataPayload{}, err
	}
	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return adapter.DataPayload{}, err
	}
	return adapter.DataPayload{
		Shape:  adapter.ShapeScalar,
		Scalar: &adapter.ScalarPayload{Value: float64(v.Total) / 1024 / 1024, Unit: "MiB", TS: ts},
	}, nil
}

func (l *localDS) loadAvg(ctx context.Context, path string, want adapter.Shape, ts int64) (adapter.DataPayload, error) {
	if err := mustScalar(want); err != nil {
		return adapter.DataPayload{}, err
	}
	avg, err := load.AvgWithContext(ctx)
	if err != nil {
		return adapter.DataPayload{}, err
	}
	val := avg.Load1
	switch {
	case strings.HasSuffix(path, "/5m"):
		val = avg.Load5
	case strings.HasSuffix(path, "/15m"):
		val = avg.Load15
	}
	return adapter.DataPayload{
		Shape:  adapter.ShapeScalar,
		Scalar: &adapter.ScalarPayload{Value: val, TS: ts},
	}, nil
}

func (l *localDS) diskUsedByMount(ctx context.Context, want adapter.Shape) (adapter.DataPayload, error) {
	if want != adapter.ShapeCategorical {
		return adapter.DataPayload{}, fmt.Errorf("%w: %q", adapter.ErrUnsupportedShape, want)
	}
	parts, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return adapter.DataPayload{}, err
	}
	items := make([]adapter.CategoricalItem, 0, len(parts))
	for _, p := range parts {
		// Ignore pseudo / read-only mounts.
		if strings.Contains(p.Mountpoint, "/proc") || strings.Contains(p.Mountpoint, "/sys") {
			continue
		}
		usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
		if err != nil || usage == nil {
			continue
		}
		items = append(items, adapter.CategoricalItem{
			Label: p.Mountpoint,
			Value: float64(usage.Used) / (1 << 30),
		})
	}
	return adapter.DataPayload{
		Shape:       adapter.ShapeCategorical,
		Categorical: &adapter.CategoricalPayload{Unit: "GiB", Items: items},
	}, nil
}

// netRate sums NIC throughput; first sample returns 0.
//
// Uses previous sample delta; exposes scalar rate.
func (l *localDS) netRate(ctx context.Context, path string, want adapter.Shape, ts int64) (adapter.DataPayload, error) {
	if err := mustScalar(want); err != nil {
		return adapter.DataPayload{}, err
	}
	stats, err := net.IOCountersWithContext(ctx, false)
	if err != nil || len(stats) == 0 {
		return adapter.DataPayload{}, errors.Join(err, errors.New("net: no counters"))
	}
	rx := float64(stats[0].BytesRecv)
	tx := float64(stats[0].BytesSent)

	prev, hasPrev := netSampler.snapshot()
	now := time.Now()
	netSampler.record(now, rx, tx)

	val := 0.0
	if hasPrev {
		dt := now.Sub(prev.t).Seconds()
		if dt > 0 {
			if strings.HasSuffix(path, "/rx_bps") {
				val = (rx - prev.rx) / dt
			} else {
				val = (tx - prev.tx) / dt
			}
		}
	}
	return adapter.DataPayload{
		Shape:  adapter.ShapeScalar,
		Scalar: &adapter.ScalarPayload{Value: val, Unit: "B/s", TS: ts},
	}, nil
}

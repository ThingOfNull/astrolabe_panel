// Package netdata implements the Netdata REST datasource.
//
// Calls Netdata REST v1:
//
//   - GET /api/v1/charts lists charts (Discover),
//   - GET /api/v1/data fetches points (Fetch).
//
// Notes:
//   - Netdata can emit native TimeSeries windows,
//     so SQL rollup is skipped for TS.
//   - Categorical uses latest timestep per dimension.
//   - Scalar reads first dimension unless Dim overrides.
package netdata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"astrolabe/internal/adapter"
)

// Type literal for netdata rows.
const Type = "netdata"

func init() {
	adapter.Register(Type, New)
}

// New expects Netdata root URL.
func New(cfg adapter.Config) (adapter.DataSource, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("netdata: endpoint required (e.g. http://host:19999)")
	}
	endpoint := strings.TrimRight(cfg.Endpoint, "/")
	timeout := 8 * time.Second
	if v, ok := cfg.Extra["timeout_sec"].(float64); ok && v > 0 {
		timeout = time.Duration(v) * time.Second
	}
	return &netdataDS{
		baseURL: endpoint,
		client:  &http.Client{Timeout: timeout},
	}, nil
}

type netdataDS struct {
	baseURL string
	client  *http.Client

	mu         sync.RWMutex
	chartsBuf  map[string]chartMeta // logical path netdata/<chart_id>
	chartsTime time.Time
}

type chartMeta struct {
	ID         string
	Name       string
	Title      string
	Family     string
	Context    string
	Units      string
	Dimensions []string
}

type chartsResp struct {
	Charts map[string]struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Title      string `json:"title"`
		Family     string `json:"family"`
		Context    string `json:"context"`
		Units      string `json:"units"`
		Dimensions map[string]struct {
			Name string `json:"name"`
		} `json:"dimensions"`
	} `json:"charts"`
}

// Connect is no-op.
func (n *netdataDS) Connect(ctx context.Context) error { return ctx.Err() }

// HealthCheck queries /api/v1/info.
func (n *netdataDS) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, n.baseURL+"/api/v1/info", nil)
	if err != nil {
		return err
	}
	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("netdata: /api/v1/info status %d", resp.StatusCode)
	}
	return nil
}

// Close is no-op.
func (n *netdataDS) Close() error { return nil }

// Discover groups charts by family.
func (n *netdataDS) Discover(ctx context.Context) (adapter.MetricTree, error) {
	charts, err := n.fetchCharts(ctx)
	if err != nil {
		return adapter.MetricTree{}, err
	}
	// Group charts by Netdata family labels.
	groups := map[string][]chartMeta{}
	for _, c := range charts {
		fam := c.Family
		if fam == "" {
			fam = "default"
		}
		groups[fam] = append(groups[fam], c)
	}
	familyKeys := make([]string, 0, len(groups))
	for k := range groups {
		familyKeys = append(familyKeys, k)
	}
	sort.Strings(familyKeys)

	root := adapter.MetricNode{Path: "netdata", Label: "Netdata", Leaf: false}
	scalarShapes := []adapter.Shape{adapter.ShapeScalar, adapter.ShapeTimeSeries}
	catShapes := []adapter.Shape{adapter.ShapeCategorical}
	for _, fam := range familyKeys {
		famNode := adapter.MetricNode{
			Path:  "netdata/" + fam,
			Label: fam,
			Leaf:  false,
		}
		cs := groups[fam]
		sort.Slice(cs, func(i, j int) bool { return cs[i].ID < cs[j].ID })
		for _, c := range cs {
			chartPath := "netdata/" + c.ID
			node := adapter.MetricNode{
				Path:   chartPath,
				Label:  c.Title,
				Unit:   c.Units,
				Shapes: append(append([]adapter.Shape{}, scalarShapes...), catShapes...),
				Leaf:   true,
			}
			famNode.Children = append(famNode.Children, node)
		}
		root.Children = append(root.Children, famNode)
	}
	return adapter.MetricTree{Roots: []adapter.MetricNode{root}}, nil
}

// Fetch loads points for one chart.
func (n *netdataDS) Fetch(ctx context.Context, q adapter.MetricQuery) (adapter.DataPayload, error) {
	chartID, ok := strings.CutPrefix(q.Path, "netdata/")
	if !ok || chartID == "" {
		return adapter.DataPayload{}, fmt.Errorf("%w: %q", adapter.ErrUnsupportedPath, q.Path)
	}
	switch q.Shape {
	case adapter.ShapeScalar:
		return n.fetchScalar(ctx, chartID, q.Dim)
	case adapter.ShapeTimeSeries:
		return n.fetchTimeSeries(ctx, chartID, q.WindowSec, q.Points)
	case adapter.ShapeCategorical:
		return n.fetchCategorical(ctx, chartID)
	default:
		return adapter.DataPayload{}, fmt.Errorf("%w: %q", adapter.ErrUnsupportedShape, q.Shape)
	}
}

// dataResp matches array-style /api/v1/data.
type dataResp struct {
	Labels []string    `json:"labels"`
	Data   [][]float64 `json:"data"`
}

func (n *netdataDS) callData(ctx context.Context, chartID string, after, before, points int) (*dataResp, error) {
	q := url.Values{}
	q.Set("chart", chartID)
	q.Set("format", "json")
	q.Set("options", "absolute|seconds")
	if after != 0 {
		q.Set("after", fmt.Sprintf("%d", after))
	}
	if before != 0 {
		q.Set("before", fmt.Sprintf("%d", before))
	}
	if points > 0 {
		q.Set("points", fmt.Sprintf("%d", points))
	}
	endpoint := n.baseURL + "/api/v1/data?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := n.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("netdata: /data status %d", resp.StatusCode)
	}
	var out dataResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("netdata: decode data: %w", err)
	}
	return &out, nil
}

func (n *netdataDS) fetchScalar(ctx context.Context, chartID, dim string) (adapter.DataPayload, error) {
	r, err := n.callData(ctx, chartID, -1, 0, 1)
	if err != nil {
		return adapter.DataPayload{}, err
	}
	if len(r.Data) == 0 || len(r.Labels) < 2 {
		return adapter.DataPayload{}, errors.New("netdata: empty data")
	}
	row := r.Data[0]
	idx := 1
	if dim != "" {
		for i, lbl := range r.Labels {
			if lbl == dim {
				idx = i
				break
			}
		}
	}
	if idx >= len(row) {
		return adapter.DataPayload{}, errors.New("netdata: dim out of range")
	}
	ts := int64(row[0])
	val := row[idx]
	unit := n.unitForChart(ctx, chartID)
	return adapter.DataPayload{
		Shape:  adapter.ShapeScalar,
		Scalar: &adapter.ScalarPayload{Value: val, Unit: unit, TS: ts},
	}, nil
}

func (n *netdataDS) fetchTimeSeries(ctx context.Context, chartID string, windowSec, points int) (adapter.DataPayload, error) {
	if windowSec <= 0 {
		windowSec = 1800
	}
	if points <= 0 {
		points = 60
	}
	r, err := n.callData(ctx, chartID, -windowSec, 0, points)
	if err != nil {
		return adapter.DataPayload{}, err
	}
	if len(r.Labels) < 2 {
		return adapter.DataPayload{}, errors.New("netdata: empty data")
	}
	dimNames := r.Labels[1:]
	series := make([]adapter.TimeSeriesSeries, len(dimNames))
	for i, name := range dimNames {
		series[i] = adapter.TimeSeriesSeries{Name: name}
	}
	for _, row := range r.Data {
		if len(row) < 2 {
			continue
		}
		ts := row[0]
		for j := range dimNames {
			idx := j + 1
			if idx >= len(row) {
				continue
			}
			series[j].Points = append(series[j].Points, adapter.TimeSeriesPoint{ts, row[idx]})
		}
	}
	// Netdata returns newest-first; reverse to ascending.
	for i := range series {
		pts := series[i].Points
		for l, r := 0, len(pts)-1; l < r; l, r = l+1, r-1 {
			pts[l], pts[r] = pts[r], pts[l]
		}
		series[i].Points = pts
	}
	return adapter.DataPayload{
		Shape: adapter.ShapeTimeSeries,
		TimeSeries: &adapter.TimeSeriesPayload{
			Unit:   n.unitForChart(ctx, chartID),
			Series: series,
		},
	}, nil
}

func (n *netdataDS) fetchCategorical(ctx context.Context, chartID string) (adapter.DataPayload, error) {
	r, err := n.callData(ctx, chartID, -1, 0, 1)
	if err != nil {
		return adapter.DataPayload{}, err
	}
	if len(r.Data) == 0 || len(r.Labels) < 2 {
		return adapter.DataPayload{}, errors.New("netdata: empty data")
	}
	row := r.Data[0]
	items := make([]adapter.CategoricalItem, 0, len(r.Labels)-1)
	for i, lbl := range r.Labels[1:] {
		idx := i + 1
		if idx >= len(row) {
			continue
		}
		items = append(items, adapter.CategoricalItem{Label: lbl, Value: row[idx]})
	}
	return adapter.DataPayload{
		Shape:       adapter.ShapeCategorical,
		Categorical: &adapter.CategoricalPayload{Unit: n.unitForChart(ctx, chartID), Items: items},
	}, nil
}

func (n *netdataDS) unitForChart(ctx context.Context, chartID string) string {
	charts, err := n.fetchCharts(ctx)
	if err != nil {
		return ""
	}
	if c, ok := charts[chartID]; ok {
		return c.Units
	}
	return ""
}

// Charts metadata TTL 60s.
func (n *netdataDS) fetchCharts(ctx context.Context) (map[string]chartMeta, error) {
	n.mu.RLock()
	if n.chartsBuf != nil && time.Since(n.chartsTime) < 60*time.Second {
		buf := n.chartsBuf
		n.mu.RUnlock()
		return buf, nil
	}
	n.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, n.baseURL+"/api/v1/charts", nil)
	if err != nil {
		return nil, err
	}
	resp, err := n.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("netdata: /charts status %d", resp.StatusCode)
	}
	var raw chartsResp
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("netdata: decode charts: %w", err)
	}
	out := make(map[string]chartMeta, len(raw.Charts))
	for id, c := range raw.Charts {
		dims := make([]string, 0, len(c.Dimensions))
		for _, d := range c.Dimensions {
			dims = append(dims, d.Name)
		}
		sort.Strings(dims)
		out[id] = chartMeta{
			ID: c.ID, Name: c.Name, Title: c.Title,
			Family: c.Family, Context: c.Context, Units: c.Units,
			Dimensions: dims,
		}
	}
	n.mu.Lock()
	n.chartsBuf = out
	n.chartsTime = time.Now()
	n.mu.Unlock()
	return out, nil
}

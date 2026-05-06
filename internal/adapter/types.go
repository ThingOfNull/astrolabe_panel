// Package adapter defines the DataSource abstraction for metrics backends.
//
// Data flow shape:
//
//  1. DataSource: connect to backend (host, Docker, Netdata, ...).
//  2. Metric path: a leaf node from Discover() metric tree.
//  3. Renderer (frontend): binds query shape to UI widgets.
//
// Implementations MUST:
//
//   - Be safe for concurrent use.
//   - Respect ctx cancellation.
//   - Fail Discover wholly on error (no partial trees).
//   - Ensure Fetch returns a payload Shape allowed on the bound tree node.
package adapter

import (
	"context"
	"errors"
	"fmt"
)

// DataSource is implemented by every backend adapter.
type DataSource interface {
	// Connect opens handles; idempotent.
	Connect(ctx context.Context) error

	// HealthCheck returns nil when the backend is reachable.
	HealthCheck(ctx context.Context) error

	// Discover returns the metric tree for this source.
	Discover(ctx context.Context) (MetricTree, error)

	// Fetch returns data; payload Shape must match query.Shape.
	Fetch(ctx context.Context, query MetricQuery) (DataPayload, error)

	// Close releases resources.
	Close() error
}

// Factory builds a DataSource from persisted config.
type Factory func(cfg Config) (DataSource, error)

// Config is the row from data_sources used to construct an adapter.
type Config struct {
	ID       int64
	Name     string
	Type     string
	Endpoint string
	Auth     map[string]any
	Extra    map[string]any
}

// Shape names metric payload kinds.
type Shape string

// Supported shape constants.
const (
	ShapeScalar      Shape = "Scalar"
	ShapeTimeSeries  Shape = "TimeSeries"
	ShapeCategorical Shape = "Categorical"
	ShapeEntityList  Shape = "EntityList"
)

// AllShapes returns every valid Shape for validation.
func AllShapes() []Shape {
	return []Shape{ShapeScalar, ShapeTimeSeries, ShapeCategorical, ShapeEntityList}
}

// IsValidShape reports whether s is a known Shape.
func IsValidShape(s string) bool {
	switch Shape(s) {
	case ShapeScalar, ShapeTimeSeries, ShapeCategorical, ShapeEntityList:
		return true
	default:
		return false
	}
}

// MetricNode is one node in the Discover tree.
type MetricNode struct {
	Path     string       `json:"path"`
	Label    string       `json:"label"`
	Unit     string       `json:"unit,omitempty"`
	Shapes   []Shape      `json:"shapes"` // Shapes this node can emit
	Leaf     bool         `json:"leaf"`   // Bindable leaf
	Children []MetricNode `json:"children,omitempty"`
}

// MetricTree is the top-level list of roots from Discover.
type MetricTree struct {
	Roots []MetricNode `json:"roots"`
}

// MetricQuery selects path, shape, and optional window or dimension.
type MetricQuery struct {
	Path      string `json:"path"`
	Shape     Shape  `json:"shape"`
	WindowSec int    `json:"window_sec,omitempty"` // TimeSeries window length (seconds)
	Points    int    `json:"points,omitempty"`     // Optional point cap
	Dim       string `json:"dim,omitempty"`        // Optional series dimension key
}

// ScalarPayload is a single scalar sample.
type ScalarPayload struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit,omitempty"`
	TS    int64   `json:"ts"`
}

// TimeSeriesPoint is one [timestamp, value] pair.
type TimeSeriesPoint [2]float64

// TimeSeriesSeries is a named series for charts.
type TimeSeriesSeries struct {
	Name   string            `json:"name"`
	Points []TimeSeriesPoint `json:"points"`
}

// TimeSeriesPayload holds one or more series.
type TimeSeriesPayload struct {
	Unit   string             `json:"unit,omitempty"`
	Series []TimeSeriesSeries `json:"series"`
}

// CategoricalItem is one bar in a categorical payload.
type CategoricalItem struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

// CategoricalPayload ranks or breakdown values.
type CategoricalPayload struct {
	Unit  string            `json:"unit,omitempty"`
	Items []CategoricalItem `json:"items"`
}

// EntityListItem is one row in a status grid source.
type EntityListItem struct {
	ID     string         `json:"id"`
	Label  string         `json:"label"`
	Status string         `json:"status"` // ok / warn / down / unknown
	Extra  map[string]any `json:"extra,omitempty"`
}

// EntityListPayload lists entities with status.
type EntityListPayload struct {
	Items []EntityListItem `json:"items"`
}

// DataPayload is the union return type of Fetch; only one field matches Shape.
type DataPayload struct {
	Shape       Shape               `json:"shape"`
	Scalar      *ScalarPayload      `json:"scalar,omitempty"`
	TimeSeries  *TimeSeriesPayload  `json:"time_series,omitempty"`
	Categorical *CategoricalPayload `json:"categorical,omitempty"`
	EntityList  *EntityListPayload  `json:"entity_list,omitempty"`
}

// Validate checks that the nested field matches Shape.
func (p DataPayload) Validate() error {
	switch p.Shape {
	case ShapeScalar:
		if p.Scalar == nil {
			return errors.New("DataPayload: shape=Scalar but Scalar field is nil")
		}
	case ShapeTimeSeries:
		if p.TimeSeries == nil {
			return errors.New("DataPayload: shape=TimeSeries but TimeSeries field is nil")
		}
	case ShapeCategorical:
		if p.Categorical == nil {
			return errors.New("DataPayload: shape=Categorical but Categorical field is nil")
		}
	case ShapeEntityList:
		if p.EntityList == nil {
			return errors.New("DataPayload: shape=EntityList but EntityList field is nil")
		}
	default:
		return fmt.Errorf("DataPayload: unsupported shape %q", p.Shape)
	}
	return nil
}

// Common adapter errors.
var (
	ErrUnsupportedPath  = errors.New("adapter: unsupported metric path")
	ErrUnsupportedShape = errors.New("adapter: unsupported shape for path")
	ErrNotConnected     = errors.New("adapter: not connected")
)

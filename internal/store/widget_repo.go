package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"

	"astrolabe/internal/store/models"
)

// Known widget kinds (keep in sync with frontend palette).
const (
	WidgetTypeLink      = "link"
	WidgetTypeSearch    = "search"
	WidgetTypeGauge     = "gauge"
	WidgetTypeBigNumber = "bignumber"
	WidgetTypeLine      = "line"
	WidgetTypeBar       = "bar"
	WidgetTypeGrid      = "grid"
	WidgetTypeText      = "text"
	WidgetTypeDivider   = "divider"
	WidgetTypeWeather   = "weather"
	WidgetTypeClock     = "clock"
)

// AcceptedShapesByType guards datasource bindings.
// Enforced on widget create/update.
var AcceptedShapesByType = map[string][]string{
	WidgetTypeGauge:     {"Scalar"},
	WidgetTypeBigNumber: {"Scalar"},
	WidgetTypeLine:      {"TimeSeries"},
	WidgetTypeBar:       {"Categorical"},
	WidgetTypeGrid:      {"EntityList"},
}

// Known widget icon backends.
const (
	IconTypeInternal = "INTERNAL"
	IconTypeRemote   = "REMOTE"
	IconTypeIconify  = "ICONIFY"
)

// Hard limits keep grid coords sane.
const (
	maxCoordinateMultiplier = 4096
	maxWidgetSizeMultiplier = 1024
)

// Repository errors for widgets.
var (
	ErrInvalidWidgetType  = errors.New("widget: unsupported type")
	ErrInvalidCoordinate  = errors.New("widget: invalid coordinate")
	ErrInvalidIconType    = errors.New("widget: invalid icon type")
	ErrInvalidURLProtocol = errors.New("widget: only http/https URL allowed")
	ErrInvalidConfigJSON  = errors.New("widget: invalid config json")
	ErrInvalidMetricJSON  = errors.New("widget: invalid metric_query json")
	ErrWidgetOverlap      = errors.New("widget: overlap with existing widget")
	ErrWidgetNotFound     = errors.New("widget: not found")
)

// WidgetView is API DTO.
type WidgetView struct {
	ID           int64           `json:"id"`
	BoardID      int64           `json:"board_id"`
	Type         string          `json:"type"`
	X            int             `json:"x"`
	Y            int             `json:"y"`
	W            int             `json:"w"`
	H            int             `json:"h"`
	ZIndex       int             `json:"z_index"`
	IconType     string          `json:"icon_type"`
	IconValue    string          `json:"icon_value"`
	DataSourceID *int64          `json:"data_source_id"`
	MetricQuery  json.RawMessage `json:"metric_query"`
	Config       json.RawMessage `json:"config"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// WidgetInput supports partial updates.
type WidgetInput struct {
	BoardID      *int64           `json:"board_id"`
	Type         *string          `json:"type"`
	X            *int             `json:"x"`
	Y            *int             `json:"y"`
	W            *int             `json:"w"`
	H            *int             `json:"h"`
	ZIndex       *int             `json:"z_index"`
	IconType     *string          `json:"icon_type"`
	IconValue    *string          `json:"icon_value"`
	DataSourceID *int64           `json:"data_source_id"`
	MetricQuery  *json.RawMessage `json:"metric_query"`
	Config       *json.RawMessage `json:"config"`
}

// WidgetBatchPatch limited to drag fields.
type WidgetBatchPatch struct {
	ID     int64 `json:"id"`
	X      int   `json:"x"`
	Y      int   `json:"y"`
	W      int   `json:"w"`
	H      int   `json:"h"`
	ZIndex int   `json:"z_index"`
}

// ListWidgets sorted for paint order.
func (s *Store) ListWidgets(ctx context.Context, boardID int64) ([]WidgetView, error) {
	var rows []models.Widget
	if err := s.DB.WithContext(ctx).
		Where("board_id = ?", boardID).
		Order("z_index ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("widget list: %w", err)
	}
	out := make([]WidgetView, len(rows))
	for i := range rows {
		out[i] = toView(&rows[i])
	}
	return out, nil
}

// GetWidget errors if missing.
func (s *Store) GetWidget(ctx context.Context, id int64) (*WidgetView, error) {
	var row models.Widget
	if err := s.DB.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWidgetNotFound
		}
		return nil, fmt.Errorf("widget get: %w", err)
	}
	v := toView(&row)
	return &v, nil
}

// CreateWidget validates geometry + payload.
func (s *Store) CreateWidget(ctx context.Context, in WidgetInput) (*WidgetView, error) {
	if in.Type == nil {
		return nil, fmt.Errorf("%w: type required", ErrInvalidWidgetType)
	}
	board := DefaultBoardID
	if in.BoardID != nil {
		board = *in.BoardID
	}
	w := models.Widget{
		BoardID:      board,
		Type:         *in.Type,
		IconType:     coalesceStr(in.IconType, IconTypeIconify),
		IconValue:    coalesceStr(in.IconValue, ""),
		DataSourceID: in.DataSourceID,
		MetricQuery:  rawOrEmpty(in.MetricQuery),
		Config:       rawOrEmpty(in.Config),
	}
	w.X = coalesceInt(in.X, 0)
	w.Y = coalesceInt(in.Y, 0)
	w.W = coalesceInt(in.W, 1)
	w.H = coalesceInt(in.H, 1)
	w.ZIndex = coalesceInt(in.ZIndex, 0)

	if err := validateWidget(&w); err != nil {
		return nil, err
	}
	if err := s.assertNoOverlap(ctx, w.BoardID, w.ID, w.X, w.Y, w.W, w.H); err != nil {
		return nil, err
	}

	if err := s.DB.WithContext(ctx).Create(&w).Error; err != nil {
		return nil, fmt.Errorf("widget create: %w", err)
	}
	v := toView(&w)
	return &v, nil
}

// UpdateWidget merges non-nil fields.
func (s *Store) UpdateWidget(ctx context.Context, id int64, in WidgetInput) (*WidgetView, error) {
	var row models.Widget
	if err := s.DB.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWidgetNotFound
		}
		return nil, fmt.Errorf("widget update load: %w", err)
	}

	applyPatch(&row, in)
	if err := validateWidget(&row); err != nil {
		return nil, err
	}
	if err := s.assertNoOverlap(ctx, row.BoardID, row.ID, row.X, row.Y, row.W, row.H); err != nil {
		return nil, err
	}

	if err := s.DB.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, fmt.Errorf("widget update save: %w", err)
	}
	v := toView(&row)
	return &v, nil
}

// DeleteWidget removes probe cache row.
func (s *Store) DeleteWidget(ctx context.Context, id int64) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Delete(&models.Widget{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrWidgetNotFound
		}
		return tx.Where("widget_id = ?", id).Delete(&models.ProbeStatus{}).Error
	})
}

// BatchUpdateWidgets persists drag results.
// Validates overlaps inside txn.
func (s *Store) BatchUpdateWidgets(ctx context.Context, patches []WidgetBatchPatch) ([]WidgetView, error) {
	if len(patches) == 0 {
		return []WidgetView{}, nil
	}
	updated := make([]WidgetView, 0, len(patches))
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ids := make([]int64, 0, len(patches))
		for _, p := range patches {
			ids = append(ids, p.ID)
		}
		var rows []models.Widget
		if err := tx.Where("id IN ?", ids).Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) != len(patches) {
			return ErrWidgetNotFound
		}
		index := make(map[int64]*models.Widget, len(rows))
		for i := range rows {
			index[rows[i].ID] = &rows[i]
		}
		for _, p := range patches {
			row, ok := index[p.ID]
			if !ok {
				return ErrWidgetNotFound
			}
			if p.W < 1 || p.H < 1 || p.X < 0 || p.Y < 0 ||
				p.X+p.W > maxCoordinateMultiplier || p.Y+p.H > maxCoordinateMultiplier {
				return fmt.Errorf("%w: widget %d", ErrInvalidCoordinate, p.ID)
			}
			row.X, row.Y, row.W, row.H, row.ZIndex = p.X, p.Y, p.W, p.H, p.ZIndex
		}
		if err := batchAssertNoOverlap(rows); err != nil {
			return err
		}
		for i := range rows {
			if err := tx.Save(&rows[i]).Error; err != nil {
				return err
			}
			updated = append(updated, toView(&rows[i]))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func validateWidget(w *models.Widget) error {
	switch w.Type {
	case WidgetTypeLink, WidgetTypeSearch,
		WidgetTypeGauge, WidgetTypeBigNumber,
		WidgetTypeLine, WidgetTypeBar, WidgetTypeGrid,
		WidgetTypeText, WidgetTypeDivider, WidgetTypeWeather, WidgetTypeClock:
	default:
		return fmt.Errorf("%w: %q", ErrInvalidWidgetType, w.Type)
	}
	if w.W < 1 || w.H < 1 || w.X < 0 || w.Y < 0 ||
		w.X+w.W > maxCoordinateMultiplier || w.Y+w.H > maxCoordinateMultiplier ||
		w.W > maxWidgetSizeMultiplier || w.H > maxWidgetSizeMultiplier {
		return ErrInvalidCoordinate
	}
	switch w.IconType {
	case "", IconTypeInternal, IconTypeRemote, IconTypeIconify:
	default:
		return fmt.Errorf("%w: %q", ErrInvalidIconType, w.IconType)
	}
	if w.MetricQuery != "" && !json.Valid([]byte(w.MetricQuery)) {
		return ErrInvalidMetricJSON
	}
	if w.Config != "" && !json.Valid([]byte(w.Config)) {
		return ErrInvalidConfigJSON
	}
	if w.Type == WidgetTypeLink {
		if err := validateLinkConfig(w.Config); err != nil {
			return err
		}
	}
	if w.Type == WidgetTypeSearch {
		if err := validateSearchConfig(w.Config); err != nil {
			return err
		}
	}
	if w.Type == WidgetTypeDivider {
		if err := validateDividerConfig(w.Config); err != nil {
			return err
		}
	}
	if w.Type == WidgetTypeText {
		if err := validateTextConfig(w.Config); err != nil {
			return err
		}
	}
	if w.Type == WidgetTypeWeather {
		if err := validateWeatherConfig(w.Config); err != nil {
			return err
		}
	}
	if w.Type == WidgetTypeClock {
		if err := validateClockConfig(w.Config); err != nil {
			return err
		}
	}
	// Data widgets require compatible shapes.
	if accepted, ok := AcceptedShapesByType[w.Type]; ok {
		if err := validateMetricQueryShape(w.MetricQuery, accepted); err != nil {
			return err
		}
		if w.DataSourceID == nil || *w.DataSourceID <= 0 {
			return fmt.Errorf("%w: data_source_id required for %q", ErrInvalidWidgetType, w.Type)
		}
	}
	return nil
}

// validateMetricQueryShape enforces shape allowlist.
func validateMetricQueryShape(raw string, accepted []string) error {
	if raw == "" {
		return fmt.Errorf("%w: metric_query required", ErrInvalidMetricJSON)
	}
	var mq struct {
		Path  string `json:"path"`
		Shape string `json:"shape"`
	}
	if err := json.Unmarshal([]byte(raw), &mq); err != nil {
		return ErrInvalidMetricJSON
	}
	if mq.Path == "" || mq.Shape == "" {
		return fmt.Errorf("%w: metric_query.path and metric_query.shape required", ErrInvalidMetricJSON)
	}
	for _, a := range accepted {
		if a == mq.Shape {
			return nil
		}
	}
	return fmt.Errorf("%w: shape %q not in %v", ErrInvalidMetricJSON, mq.Shape, accepted)
}

// validateLinkConfig enforces http(s) + visibility toggles.
// At least one of icon/title/url must stay visible.
func validateLinkConfig(raw string) error {
	if raw == "" {
		return nil
	}
	var cfg struct {
		URL       string `json:"url"`
		ShowIcon  *bool  `json:"show_icon"`
		ShowTitle *bool  `json:"show_title"`
		ShowURL   *bool  `json:"show_url"`
		Probe     struct {
			Type string `json:"type"`
			Host string `json:"host"`
		} `json:"probe"`
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return ErrInvalidConfigJSON
	}
	if cfg.URL != "" {
		if err := requireHTTPProtocol(cfg.URL); err != nil {
			return err
		}
	}
	tri := func(p *bool) bool {
		return p == nil || *p
	}
	if !tri(cfg.ShowIcon) && !tri(cfg.ShowTitle) && !tri(cfg.ShowURL) {
		return fmt.Errorf("%w: link must show at least one of icon/title/url", ErrInvalidConfigJSON)
	}
	return nil
}

func validateTextConfig(raw string) error {
	if raw == "" {
		return nil
	}
	var cfg struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return ErrInvalidConfigJSON
	}
	_ = cfg
	return nil
}

func validateWeatherConfig(raw string) error {
	if raw == "" {
		return fmt.Errorf("%w: weather.city_id required", ErrInvalidConfigJSON)
	}
	var cfg struct {
		CityID int64 `json:"city_id"`
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return ErrInvalidConfigJSON
	}
	if cfg.CityID <= 0 {
		return fmt.Errorf("%w: invalid city_id", ErrInvalidConfigJSON)
	}
	return nil
}

func validateClockConfig(raw string) error {
	if raw == "" {
		return nil
	}
	var cfg struct {
		Variant  string `json:"variant"`
		Timezone string `json:"timezone"`
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return ErrInvalidConfigJSON
	}
	switch strings.TrimSpace(cfg.Variant) {
	case "", "digital", "flip":
	default:
		return fmt.Errorf("%w: clock.variant", ErrInvalidConfigJSON)
	}
	if tz := strings.TrimSpace(cfg.Timezone); tz != "" {
		if _, err := time.LoadLocation(tz); err != nil {
			return fmt.Errorf("%w: clock.timezone", ErrInvalidConfigJSON)
		}
	}
	return nil
}

func validateDividerConfig(raw string) error {
	if raw == "" {
		return nil
	}
	var cfg struct {
		Orientation string `json:"orientation"`
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return ErrInvalidConfigJSON
	}
	switch cfg.Orientation {
	case "", "horizontal", "vertical":
		return nil
	default:
		return fmt.Errorf("%w: divider orientation", ErrInvalidConfigJSON)
	}
}

// validateSearchConfig checks engine templates.
func validateSearchConfig(raw string) error {
	if raw == "" {
		return nil
	}
	var cfg struct {
		Engines []struct {
			URL string `json:"url"`
		} `json:"engines"`
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return ErrInvalidConfigJSON
	}
	for _, e := range cfg.Engines {
		if e.URL == "" {
			continue
		}
		// Strip {q} placeholder before URL parse.
		raw := strings.ReplaceAll(e.URL, "{q}", "q")
		if err := requireHTTPProtocol(raw); err != nil {
			return err
		}
	}
	return nil
}

func requireHTTPProtocol(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return ErrInvalidURLProtocol
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return nil
	default:
		return fmt.Errorf("%w: scheme=%q", ErrInvalidURLProtocol, u.Scheme)
	}
}

func (s *Store) assertNoOverlap(ctx context.Context, boardID, selfID int64, x, y, w, h int) error {
	var siblings []models.Widget
	q := s.DB.WithContext(ctx).Where("board_id = ?", boardID)
	if selfID > 0 {
		q = q.Where("id <> ?", selfID)
	}
	if err := q.Find(&siblings).Error; err != nil {
		return err
	}
	for i := range siblings {
		o := &siblings[i]
		if rectsOverlap(x, y, w, h, o.X, o.Y, o.W, o.H) {
			return fmt.Errorf("%w: id=%d", ErrWidgetOverlap, o.ID)
		}
	}
	return nil
}

func batchAssertNoOverlap(rows []models.Widget) error {
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			a, b := &rows[i], &rows[j]
			if a.BoardID != b.BoardID {
				continue
			}
			if rectsOverlap(a.X, a.Y, a.W, a.H, b.X, b.Y, b.W, b.H) {
				return fmt.Errorf("%w: ids=%d,%d", ErrWidgetOverlap, a.ID, b.ID)
			}
		}
	}
	return nil
}

func rectsOverlap(ax, ay, aw, ah, bx, by, bw, bh int) bool {
	return ax < bx+bw && ax+aw > bx && ay < by+bh && ay+ah > by
}

func applyPatch(row *models.Widget, in WidgetInput) {
	if in.BoardID != nil {
		row.BoardID = *in.BoardID
	}
	if in.Type != nil {
		row.Type = *in.Type
	}
	if in.X != nil {
		row.X = *in.X
	}
	if in.Y != nil {
		row.Y = *in.Y
	}
	if in.W != nil {
		row.W = *in.W
	}
	if in.H != nil {
		row.H = *in.H
	}
	if in.ZIndex != nil {
		row.ZIndex = *in.ZIndex
	}
	if in.IconType != nil {
		row.IconType = *in.IconType
	}
	if in.IconValue != nil {
		row.IconValue = *in.IconValue
	}
	if in.DataSourceID != nil {
		row.DataSourceID = in.DataSourceID
	}
	if in.MetricQuery != nil {
		row.MetricQuery = string(*in.MetricQuery)
	}
	if in.Config != nil {
		row.Config = string(*in.Config)
	}
}

func toView(w *models.Widget) WidgetView {
	v := WidgetView{
		ID:           w.ID,
		BoardID:      w.BoardID,
		Type:         w.Type,
		X:            w.X,
		Y:            w.Y,
		W:            w.W,
		H:            w.H,
		ZIndex:       w.ZIndex,
		IconType:     w.IconType,
		IconValue:    w.IconValue,
		DataSourceID: w.DataSourceID,
		CreatedAt:    w.CreatedAt,
		UpdatedAt:    w.UpdatedAt,
	}
	v.MetricQuery = stringToRawJSON(w.MetricQuery)
	v.Config = stringToRawJSON(w.Config)
	return v
}

func stringToRawJSON(s string) json.RawMessage {
	if s == "" {
		return json.RawMessage("null")
	}
	if !json.Valid([]byte(s)) {
		return json.RawMessage("null")
	}
	return json.RawMessage(s)
}

func rawOrEmpty(raw *json.RawMessage) string {
	if raw == nil {
		return ""
	}
	return string(*raw)
}

func coalesceStr(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}

func coalesceInt(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

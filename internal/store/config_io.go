package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"astrolabe/internal/store/models"
)

// Config bundle format version (bump when schema breaks).
const ConfigBundleVersion = 1

// JSON payload for HTTP import/export endpoints.
type ConfigBundle struct {
	Version     int                  `json:"version"`
	ExportedAt  time.Time            `json:"exported_at"`
	Board       BoardSnapshot        `json:"board"`
	DataSources []DataSourceSnapshot `json:"data_sources"`
	Widgets     []WidgetSnapshot     `json:"widgets"`
}

// Board snapshot without ids (import pins id=1).
type BoardSnapshot struct {
	Name         string `json:"name"`
	GridBaseUnit int    `json:"grid_base_unit"`
	Wallpaper    string `json:"wallpaper"`
	Theme        string `json:"theme"`
	ThemeCustom  string `json:"theme_custom"`
}

// Datasource snapshot; import mints IDs.
type DataSourceSnapshot struct {
	OriginalID int64           `json:"original_id"`
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Endpoint   string          `json:"endpoint"`
	Auth       json.RawMessage `json:"auth"`
	Extra      json.RawMessage `json:"extra"`
}

// Widget snapshot with legacy id remaps.
type WidgetSnapshot struct {
	OriginalID         int64           `json:"original_id"`
	Type               string          `json:"type"`
	X                  int             `json:"x"`
	Y                  int             `json:"y"`
	W                  int             `json:"w"`
	H                  int             `json:"h"`
	ZIndex             int             `json:"z_index"`
	IconType           string          `json:"icon_type"`
	IconValue          string          `json:"icon_value"`
	OriginalDataSource *int64          `json:"original_data_source_id"`
	MetricQuery        json.RawMessage `json:"metric_query"`
	Config             json.RawMessage `json:"config"`
}

// Export serializes workspace.
func (s *Store) ExportConfig(ctx context.Context) (*ConfigBundle, error) {
	board, err := s.GetBoard(ctx, DefaultBoardID)
	if err != nil {
		return nil, err
	}
	var dsRows []models.DataSource
	if err := s.DB.WithContext(ctx).Order("id ASC").Find(&dsRows).Error; err != nil {
		return nil, err
	}
	var widgetRows []models.Widget
	if err := s.DB.WithContext(ctx).Order("id ASC").Find(&widgetRows).Error; err != nil {
		return nil, err
	}

	bundle := &ConfigBundle{
		Version:    ConfigBundleVersion,
		ExportedAt: time.Now().UTC(),
		Board: BoardSnapshot{
			Name:         board.Name,
			GridBaseUnit: board.GridBaseUnit,
			Wallpaper:    board.Wallpaper,
			Theme:        board.Theme,
			ThemeCustom:  board.ThemeCustom,
		},
	}
	for i := range dsRows {
		d := &dsRows[i]
		bundle.DataSources = append(bundle.DataSources, DataSourceSnapshot{
			OriginalID: d.ID,
			Name:       d.Name,
			Type:       d.Type,
			Endpoint:   d.Endpoint,
			Auth:       stringToRawJSON(d.Auth),
			Extra:      stringToRawJSON(d.Extra),
		})
	}
	for i := range widgetRows {
		w := &widgetRows[i]
		bundle.Widgets = append(bundle.Widgets, WidgetSnapshot{
			OriginalID:         w.ID,
			Type:               w.Type,
			X:                  w.X,
			Y:                  w.Y,
			W:                  w.W,
			H:                  w.H,
			ZIndex:             w.ZIndex,
			IconType:           w.IconType,
			IconValue:          w.IconValue,
			OriginalDataSource: w.DataSourceID,
			MetricQuery:        stringToRawJSON(w.MetricQuery),
			Config:             stringToRawJSON(w.Config),
		})
	}
	return bundle, nil
}

// ImportSummary reports counts.
type ImportSummary struct {
	BoardUpdated     bool  `json:"board_updated"`
	DataSourcesAdded int   `json:"data_sources_added"`
	WidgetsAdded     int   `json:"widgets_added"`
	WidgetsSkipped   []int `json:"widgets_skipped,omitempty"`
}

// Import replaces widgets + datasources from bundle.
// Board overwrites in place; returns id maps.
//
// Note: metric_samples/probe rows remain until datasource/widget cleanup paths run.
func (s *Store) ImportConfig(ctx context.Context, bundle *ConfigBundle) (*ImportSummary, error) {
	if bundle == nil {
		return nil, errors.New("import: bundle nil")
	}
	if bundle.Version <= 0 || bundle.Version > ConfigBundleVersion {
		return nil, fmt.Errorf("import: unsupported version %d (current %d)", bundle.Version, ConfigBundleVersion)
	}
	summary := &ImportSummary{}
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1) delete widgets+datasources explicitly.
		if err := tx.Where("1 = 1").Delete(&models.Widget{}).Error; err != nil {
			return fmt.Errorf("import: clear widgets: %w", err)
		}
		if err := tx.Where("1 = 1").Delete(&models.MetricSample{}).Error; err != nil {
			return fmt.Errorf("import: clear samples: %w", err)
		}
		if err := tx.Where("1 = 1").Delete(&models.DataSource{}).Error; err != nil {
			return fmt.Errorf("import: clear datasources: %w", err)
		}

		// 2) upsert board id=1.
		var board models.Board
		if err := tx.First(&board, DefaultBoardID).Error; err != nil {
			return fmt.Errorf("import: load board: %w", err)
		}
		board.Name = bundle.Board.Name
		if bundle.Board.GridBaseUnit > 0 {
			board.GridBaseUnit = bundle.Board.GridBaseUnit
		}
		board.Wallpaper = bundle.Board.Wallpaper
		if bundle.Board.Theme != "" {
			board.Theme = bundle.Board.Theme
		}
		board.ThemeCustom = bundle.Board.ThemeCustom
		if err := tx.Save(&board).Error; err != nil {
			return fmt.Errorf("import: save board: %w", err)
		}
		summary.BoardUpdated = true

		// 3) insert datasources, track id map.
		dsIDMap := map[int64]int64{}
		for _, snap := range bundle.DataSources {
			row := models.DataSource{
				Name:       snap.Name,
				Type:       snap.Type,
				Endpoint:   snap.Endpoint,
				Auth:       rawOrEmpty(&snap.Auth),
				Extra:      rawOrEmpty(&snap.Extra),
				LastHealth: DataSourceHealthUnknown,
			}
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("import: insert datasource %q: %w", snap.Name, err)
			}
			dsIDMap[snap.OriginalID] = row.ID
			summary.DataSourcesAdded++
		}

		// 4) insert widgets using new FK ids.
		for _, snap := range bundle.Widgets {
			var dsID *int64
			if snap.OriginalDataSource != nil {
				if newID, ok := dsIDMap[*snap.OriginalDataSource]; ok {
					dsID = &newID
				} else {
					summary.WidgetsSkipped = append(summary.WidgetsSkipped, int(snap.OriginalID))
					continue
				}
			}
			row := models.Widget{
				BoardID: DefaultBoardID,
				Type:    snap.Type,
				X:       snap.X, Y: snap.Y, W: snap.W, H: snap.H,
				ZIndex:       snap.ZIndex,
				IconType:     snap.IconType,
				IconValue:    snap.IconValue,
				DataSourceID: dsID,
				MetricQuery:  rawOrEmpty(&snap.MetricQuery),
				Config:       rawOrEmpty(&snap.Config),
			}
			if err := validateWidget(&row); err != nil {
				summary.WidgetsSkipped = append(summary.WidgetsSkipped, int(snap.OriginalID))
				continue
			}
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("import: insert widget #%d: %w", snap.OriginalID, err)
			}
			summary.WidgetsAdded++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return summary, nil
}

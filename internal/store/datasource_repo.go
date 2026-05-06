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

// Health literals for datasource rows.
const (
	DataSourceHealthOK      = "ok"
	DataSourceHealthError   = "error"
	DataSourceHealthUnknown = "unknown"
)

// Repository errors.
var (
	ErrDataSourceNotFound = errors.New("data_source: not found")
	ErrDataSourceInvalid  = errors.New("data_source: invalid input")
)

// DataSourceView is JSON-friendly dto.
type DataSourceView struct {
	ID           int64           `json:"id"`
	Name         string          `json:"name"`
	Type         string          `json:"type"`
	Endpoint     string          `json:"endpoint"`
	Auth         json.RawMessage `json:"auth"`
	Extra        json.RawMessage `json:"extra"`
	LastHealth   string          `json:"last_health"`
	LastHealthAt time.Time       `json:"last_health_at"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// DataSourceInput accepts partial updates via pointers.
type DataSourceInput struct {
	Name     *string          `json:"name"`
	Type     *string          `json:"type"`
	Endpoint *string          `json:"endpoint"`
	Auth     *json.RawMessage `json:"auth"`
	Extra    *json.RawMessage `json:"extra"`
}

// ListDataSources sorted by PK.
func (s *Store) ListDataSources(ctx context.Context) ([]DataSourceView, error) {
	var rows []models.DataSource
	if err := s.DB.WithContext(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("data_source list: %w", err)
	}
	out := make([]DataSourceView, len(rows))
	for i := range rows {
		out[i] = toDSView(&rows[i])
	}
	return out, nil
}

// GetDataSource reads one row.
func (s *Store) GetDataSource(ctx context.Context, id int64) (*DataSourceView, error) {
	var row models.DataSource
	if err := s.DB.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDataSourceNotFound
		}
		return nil, fmt.Errorf("data_source get: %w", err)
	}
	v := toDSView(&row)
	return &v, nil
}

// Create inserts datasource row.
func (s *Store) CreateDataSource(ctx context.Context, in DataSourceInput) (*DataSourceView, error) {
	if in.Type == nil || *in.Type == "" {
		return nil, fmt.Errorf("%w: type required", ErrDataSourceInvalid)
	}
	if in.Name == nil || *in.Name == "" {
		return nil, fmt.Errorf("%w: name required", ErrDataSourceInvalid)
	}
	row := models.DataSource{
		Name:       *in.Name,
		Type:       *in.Type,
		Endpoint:   coalesceStr(in.Endpoint, ""),
		Auth:       rawOrEmpty(in.Auth),
		Extra:      rawOrEmpty(in.Extra),
		LastHealth: DataSourceHealthUnknown,
	}
	if err := validateDataSource(&row); err != nil {
		return nil, err
	}
	if err := s.DB.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, fmt.Errorf("data_source create: %w", err)
	}
	v := toDSView(&row)
	return &v, nil
}

// Update merges changes.
func (s *Store) UpdateDataSource(ctx context.Context, id int64, in DataSourceInput) (*DataSourceView, error) {
	var row models.DataSource
	if err := s.DB.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDataSourceNotFound
		}
		return nil, fmt.Errorf("data_source update load: %w", err)
	}
	if in.Name != nil {
		row.Name = *in.Name
	}
	if in.Type != nil {
		row.Type = *in.Type
	}
	if in.Endpoint != nil {
		row.Endpoint = *in.Endpoint
	}
	if in.Auth != nil {
		row.Auth = string(*in.Auth)
	}
	if in.Extra != nil {
		row.Extra = string(*in.Extra)
	}
	if err := validateDataSource(&row); err != nil {
		return nil, err
	}
	if err := s.DB.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, fmt.Errorf("data_source save: %w", err)
	}
	v := toDSView(&row)
	return &v, nil
}

// DeleteDatasource clears dependents + metric samples.
func (s *Store) DeleteDataSource(ctx context.Context, id int64) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Delete(&models.DataSource{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrDataSourceNotFound
		}
		// Null FK on widgets referencing this datasource.
		if err := tx.Model(&models.Widget{}).
			Where("data_source_id = ?", id).
			Updates(map[string]any{"data_source_id": nil, "metric_query": ""}).Error; err != nil {
			return err
		}
		// purge metric_samples for datasource.
		return tx.Where("data_source_id = ?", id).Delete(&models.MetricSample{}).Error
	})
}

// UpdateDataSourceHealth stores status text.
func (s *Store) UpdateDataSourceHealth(ctx context.Context, id int64, health string) error {
	if health != DataSourceHealthOK && health != DataSourceHealthError && health != DataSourceHealthUnknown {
		return fmt.Errorf("%w: bad health %q", ErrDataSourceInvalid, health)
	}
	return s.DB.WithContext(ctx).
		Model(&models.DataSource{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"last_health":    health,
			"last_health_at": time.Now().UTC(),
		}).Error
}

func validateDataSource(row *models.DataSource) error {
	if row.Auth != "" && !json.Valid([]byte(row.Auth)) {
		return fmt.Errorf("%w: auth not valid json", ErrDataSourceInvalid)
	}
	if row.Extra != "" && !json.Valid([]byte(row.Extra)) {
		return fmt.Errorf("%w: extra not valid json", ErrDataSourceInvalid)
	}
	return nil
}

func toDSView(d *models.DataSource) DataSourceView {
	return DataSourceView{
		ID:           d.ID,
		Name:         d.Name,
		Type:         d.Type,
		Endpoint:     d.Endpoint,
		Auth:         stringToRawJSON(d.Auth),
		Extra:        stringToRawJSON(d.Extra),
		LastHealth:   d.LastHealth,
		LastHealthAt: d.LastHealthAt,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}

package store

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm/clause"

	"astrolabe/internal/store/models"
)

// SampleInsert inserts one observation.
type SampleInsert struct {
	DataSourceID int64
	MetricPath   string
	Dim          string
	Ts           int64
	Value        float64
}

// InsertSamples ignores duplicate keys under same TS.
func (s *Store) InsertSamples(ctx context.Context, items []SampleInsert) error {
	if len(items) == 0 {
		return nil
	}
	rows := make([]models.MetricSample, len(items))
	for i, it := range items {
		dim := it.Dim
		if dim == "" {
			dim = "_"
		}
		rows[i] = models.MetricSample{
			DataSourceID: it.DataSourceID,
			MetricPath:   it.MetricPath,
			Dim:          dim,
			Ts:           it.Ts,
			Value:        it.Value,
		}
	}
	tx := s.DB.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&rows)
	if tx.Error != nil {
		return fmt.Errorf("metric_samples insert: %w", tx.Error)
	}
	return nil
}

// QuerySamples scans dims for path/window.
func (s *Store) QuerySamples(ctx context.Context, dsID int64, path string, windowSec int64) ([]models.MetricSample, error) {
	if windowSec <= 0 {
		windowSec = 1800
	}
	since := time.Now().Add(-time.Duration(windowSec) * time.Second).Unix()
	var rows []models.MetricSample
	if err := s.DB.WithContext(ctx).
		Where("data_source_id = ? AND metric_path = ? AND ts >= ?", dsID, path, since).
		Order("ts ASC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("metric_samples query: %w", err)
	}
	return rows, nil
}

// CleanupSamples trims rows outside retention window.
func (s *Store) CleanupSamples(ctx context.Context, retainMinutes int) (int64, error) {
	if retainMinutes <= 0 {
		retainMinutes = 30
	}
	threshold := time.Now().Add(-time.Duration(retainMinutes) * time.Minute).Unix()
	res := s.DB.WithContext(ctx).
		Where("ts < ?", threshold).
		Delete(&models.MetricSample{})
	if res.Error != nil {
		return 0, fmt.Errorf("metric_samples cleanup: %w", res.Error)
	}
	return res.RowsAffected, nil
}

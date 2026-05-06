package models

// MetricSample stores raw points.
// Indexed by unix seconds.
type MetricSample struct {
	DataSourceID int64   `gorm:"primaryKey"                json:"data_source_id"`
	MetricPath   string  `gorm:"primaryKey;type:text"      json:"metric_path"`
	Dim          string  `gorm:"primaryKey;type:text"      json:"dim"`
	Ts           int64   `gorm:"primaryKey"                json:"ts"`
	Value        float64 `gorm:"not null"                  json:"value"`
}

// TableName sets SQL table.
func (MetricSample) TableName() string { return "metric_samples" }

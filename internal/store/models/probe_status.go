package models

import "time"

// ProbeStatus caches last probe per widget.
type ProbeStatus struct {
	WidgetID  int64     `gorm:"primaryKey"          json:"widget_id"`
	Status    string    `gorm:"type:text;not null"  json:"status"`
	LatencyMs int       `                           json:"latency_ms"`
	CheckedAt time.Time `                           json:"checked_at"`
}

// TableName sets SQL table.
func (ProbeStatus) TableName() string { return "probe_status" }

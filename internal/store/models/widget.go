package models

import "time"

// Widget persists layout multiplier grid units.
type Widget struct {
	ID           int64     `gorm:"primaryKey"                                          json:"id"`
	BoardID      int64     `gorm:"not null;index:idx_widget_board_xy,priority:1"        json:"board_id"`
	Type         string    `gorm:"type:text;not null"                                  json:"type"`
	X            int       `gorm:"not null;index:idx_widget_board_xy,priority:2"        json:"x"`
	Y            int       `gorm:"not null;index:idx_widget_board_xy,priority:3"        json:"y"`
	W            int       `gorm:"not null"                                            json:"w"`
	H            int       `gorm:"not null"                                            json:"h"`
	ZIndex       int       `gorm:"not null;default:0"                                  json:"z_index"`
	IconType     string    `gorm:"type:text"                                           json:"icon_type"`
	IconValue    string    `gorm:"type:text"                                           json:"icon_value"`
	DataSourceID *int64    `                                                            json:"data_source_id"`
	MetricQuery  string    `gorm:"type:text"                                           json:"metric_query"`
	Config       string    `gorm:"type:text"                                           json:"config"`
	CreatedAt    time.Time `                                                            json:"created_at"`
	UpdatedAt    time.Time `                                                            json:"updated_at"`
}

// TableName sets SQL table.
func (Widget) TableName() string { return "widgets" }

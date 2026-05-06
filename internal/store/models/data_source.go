package models

import "time"

// DataSource mirrors adapter settings json.
type DataSource struct {
	ID           int64     `gorm:"primaryKey"             json:"id"`
	Name         string    `gorm:"type:text;not null"     json:"name"`
	Type         string    `gorm:"type:text;not null;index" json:"type"`
	Endpoint     string    `gorm:"type:text"              json:"endpoint"`
	Auth         string    `gorm:"type:text"              json:"auth"`
	Extra        string    `gorm:"type:text"              json:"extra"`
	LastHealth   string    `gorm:"type:text"              json:"last_health"`
	LastHealthAt time.Time `                              json:"last_health_at"`
	CreatedAt    time.Time `                              json:"created_at"`
	UpdatedAt    time.Time `                              json:"updated_at"`
}

// TableName sets SQL table.
func (DataSource) TableName() string { return "data_sources" }

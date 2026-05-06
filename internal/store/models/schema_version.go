package models

import "time"

// SchemaVersion controls migrations.
type SchemaVersion struct {
	ID        int       `gorm:"primaryKey;autoIncrement:false" json:"id"`
	Version   int       `gorm:"not null"                       json:"version"`
	UpdatedAt time.Time `                                       json:"updated_at"`
}

// TableName sets SQL table.
func (SchemaVersion) TableName() string { return "schema_versions" }

// Bump when migrating schema.
const CurrentSchemaVersion = 1

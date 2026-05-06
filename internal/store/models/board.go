package models

import "time"

// Board row (single tenant default id=1).
type Board struct {
	ID           int64     `gorm:"primaryKey;autoIncrement:false" json:"id"`
	Name         string    `gorm:"type:text;not null"             json:"name"`
	GridBaseUnit int       `gorm:"not null;default:10"            json:"grid_base_unit"`
	Wallpaper    string    `gorm:"type:text"                      json:"wallpaper"`
	Theme        string    `gorm:"type:text;not null;default:dark" json:"theme"`
	ThemeCustom  string    `gorm:"type:text;column:theme_custom"  json:"theme_custom"`
	UpdatedAt    time.Time `                                       json:"updated_at"`
}

// TableName overrides GORM default.
func (Board) TableName() string { return "boards" }

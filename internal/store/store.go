// Package store opens sqlite via GORM.
package store

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"astrolabe/internal/store/models"
)

// DefaultBoardID anchors single-board mode.
const DefaultBoardID int64 = 1

// Options configures sqlite URI.
type Options struct {
	DBPath string
}

// Store owns *gorm.DB.
type Store struct {
	DB *gorm.DB
}

// Open ensures db file + migrations.
func Open(ctx context.Context, opts Options) (*Store, error) {
	if opts.DBPath == "" {
		return nil, errors.New("store: DBPath required")
	}

	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", opts.DBPath)
	dbLogger := gormlogger.New(
		log.New(io.Discard, "", 0),
		gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  gormlogger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:                                   dbLogger,
		DisableForeignKeyConstraintWhenMigrating: false,
	})
	if err != nil {
		return nil, fmt.Errorf("store: open sqlite %q: %w", opts.DBPath, err)
	}

	s := &Store{DB: db}
	if err := s.migrate(ctx); err != nil {
		return nil, err
	}
	if err := s.seed(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

// Close releases the underlying sql.DB pool.
func (s *Store) Close() error {
	sqlDB, err := s.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *Store) migrate(ctx context.Context) error {
	db := s.DB.WithContext(ctx)
	if err := db.AutoMigrate(
		&models.Board{},
		&models.DataSource{},
		&models.Widget{},
		&models.MetricSample{},
		&models.ProbeStatus{},
		&models.SchemaVersion{},
	); err != nil {
		return fmt.Errorf("store: auto migrate: %w", err)
	}

	var sv models.SchemaVersion
	if err := db.First(&sv, 1).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("store: read schema version: %w", err)
		}
		sv = models.SchemaVersion{ID: 1, Version: models.CurrentSchemaVersion, UpdatedAt: time.Now().UTC()}
		if err := db.Create(&sv).Error; err != nil {
			return fmt.Errorf("store: write schema version: %w", err)
		}
		slog.Info("schema_version initialized", "version", sv.Version)
		return nil
	}

	if sv.Version > models.CurrentSchemaVersion {
		return fmt.Errorf("store: db schema version %d newer than code %d; refuse to start", sv.Version, models.CurrentSchemaVersion)
	}
	return nil
}

func (s *Store) seed(ctx context.Context) error {
	db := s.DB.WithContext(ctx)
	var count int64
	if err := db.Model(&models.Board{}).Count(&count).Error; err != nil {
		return fmt.Errorf("store: count boards: %w", err)
	}
	if count > 0 {
		return nil
	}
	board := models.Board{
		ID:           DefaultBoardID,
		Name:         "Astrolabe",
		GridBaseUnit: 10,
		Theme:        "dark",
		UpdatedAt:    time.Now().UTC(),
	}
	if err := db.Create(&board).Error; err != nil {
		return fmt.Errorf("store: seed default board: %w", err)
	}
	slog.Info("default board seeded", "id", board.ID, "name", board.Name)
	return nil
}

// GetBoard returns ErrRecordNotFound if missing.
func (s *Store) GetBoard(ctx context.Context, id int64) (*models.Board, error) {
	var b models.Board
	if err := s.DB.WithContext(ctx).First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// BoardUpdateInput partial board updates.
type BoardUpdateInput struct {
	Name         *string `json:"name"`
	GridBaseUnit *int    `json:"grid_base_unit"`
	Wallpaper    *string `json:"wallpaper"`
	Theme        *string `json:"theme"`
	ThemeCustom  *string `json:"theme_custom"`
}

// UpdateBoard writes board fields.
func (s *Store) UpdateBoard(ctx context.Context, id int64, in BoardUpdateInput) (*models.Board, error) {
	var b models.Board
	if err := s.DB.WithContext(ctx).First(&b, id).Error; err != nil {
		return nil, err
	}
	if in.Name != nil {
		b.Name = *in.Name
	}
	if in.GridBaseUnit != nil {
		v := *in.GridBaseUnit
		if v <= 0 || v > 200 {
			return nil, fmt.Errorf("invalid grid_base_unit: %d", v)
		}
		b.GridBaseUnit = v
	}
	if in.Wallpaper != nil {
		b.Wallpaper = *in.Wallpaper
	}
	if in.Theme != nil {
		switch *in.Theme {
		case "dark", "light", "custom", "custom_image":
			b.Theme = *in.Theme
		default:
			return nil, fmt.Errorf("invalid theme: %q", *in.Theme)
		}
	}
	if in.ThemeCustom != nil {
		b.ThemeCustom = *in.ThemeCustom
	}
	if strings.TrimSpace(b.Theme) == "custom_image" {
		if strings.TrimSpace(b.Wallpaper) == "" {
			return nil, fmt.Errorf("custom_image theme requires non-empty wallpaper")
		}
	}
	if err := s.DB.WithContext(ctx).Save(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

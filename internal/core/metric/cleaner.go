package metric

import (
	"context"
	"log/slog"
	"time"

	"astrolabe/internal/store"
)

// Cleaner drops old metric_samples.
type Cleaner struct {
	store         *store.Store
	retainMinutes int
	scanInterval  time.Duration
	logger        *slog.Logger
}

// NewCleaner constructs worker.
func NewCleaner(s *store.Store, retainMinutes, intervalMinutes int) *Cleaner {
	if retainMinutes <= 0 {
		retainMinutes = 30
	}
	if intervalMinutes <= 0 {
		intervalMinutes = 5
	}
	return &Cleaner{
		store:         s,
		retainMinutes: retainMinutes,
		scanInterval:  time.Duration(intervalMinutes) * time.Minute,
		logger:        slog.Default().With("module", "metric_cleaner"),
	}
}

// Run honors ctx cancel.
func (c *Cleaner) Run(ctx context.Context) {
	ticker := time.NewTicker(c.scanInterval)
	defer ticker.Stop()
	c.logger.Info("metric cleaner started",
		"retain_minutes", c.retainMinutes,
		"interval", c.scanInterval,
	)
	c.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("metric cleaner stopped")
			return
		case <-ticker.C:
			c.tick(ctx)
		}
	}
}

func (c *Cleaner) tick(ctx context.Context) {
	n, err := c.store.CleanupSamples(ctx, c.retainMinutes)
	if err != nil {
		c.logger.Warn("cleanup failed", "err", err)
		return
	}
	if n > 0 {
		c.logger.Debug("samples cleaned", "rows", n)
	}
}

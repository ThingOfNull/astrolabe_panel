package probe

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"gorm.io/gorm"

	"astrolabe/internal/events"
	"astrolabe/internal/store"
	"astrolabe/internal/store/models"
)

// Scheduler wakes link probes on interval.
type Scheduler struct {
	store              *store.Store
	defaultIntervalSec int
	defaultTimeoutSec  int
	scanInterval       time.Duration
	mu                 sync.Mutex
	inflight           map[int64]struct{}
	logger             *slog.Logger
	hub                *events.Hub // optional; set via SetHub
}

// SetHub attaches an event hub so the scheduler can push probe.changed on
// status flips. Must be called before Run.
func (sc *Scheduler) SetHub(h *events.Hub) { sc.hub = h }

// Options tune backlog scanning.
type Options struct {
	DefaultIntervalSec int
	DefaultTimeoutSec  int
	ScanInterval       time.Duration
}

// NewScheduler allocates probe runner.
func NewScheduler(s *store.Store, opts Options) *Scheduler {
	scan := opts.ScanInterval
	if scan <= 0 {
		scan = 5 * time.Second
	}
	intv := opts.DefaultIntervalSec
	if intv <= 0 {
		intv = 30
	}
	tmo := opts.DefaultTimeoutSec
	if tmo <= 0 {
		tmo = 4
	}
	return &Scheduler{
		store:              s,
		defaultIntervalSec: intv,
		defaultTimeoutSec:  tmo,
		scanInterval:       scan,
		inflight:           make(map[int64]struct{}),
		logger:             slog.Default().With("module", "probe"),
	}
}

// Run stops quickly on cancel.
func (sc *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(sc.scanInterval)
	defer ticker.Stop()
	sc.logger.Info("probe scheduler started", "scan_interval", sc.scanInterval, "default_interval_sec", sc.defaultIntervalSec)

	sc.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			sc.logger.Info("probe scheduler stopped")
			return
		case <-ticker.C:
			sc.tick(ctx)
		}
	}
}

func (sc *Scheduler) tick(ctx context.Context) {
	widgets, err := sc.store.ListWidgets(ctx, store.DefaultBoardID)
	if err != nil {
		sc.logger.Warn("list widgets failed", "err", err)
		return
	}
	now := time.Now()
	for _, w := range widgets {
		if w.Type != store.WidgetTypeLink {
			continue
		}
		cfg, ok := parseLinkConfig(w.Config)
		if !ok {
			continue
		}
		if cfg.Probe.IntervalSec == 0 {
			cfg.Probe.IntervalSec = sc.defaultIntervalSec
		}
		if cfg.URL == "" && cfg.Probe.Host == "" && cfg.Probe.URL == "" {
			continue
		}
		due, err := sc.dueForCheck(ctx, w.ID, cfg.Probe.IntervalSec, now)
		if err != nil {
			sc.logger.Warn("read probe status failed", "widget_id", w.ID, "err", err)
			continue
		}
		if !due {
			continue
		}
		if !sc.markInflight(w.ID) {
			continue
		}
		go sc.runOne(ctx, w.ID, cfg)
	}
}

// dueForCheck enforces min interval.
func (sc *Scheduler) dueForCheck(ctx context.Context, widgetID int64, intervalSec int, now time.Time) (bool, error) {
	var ps models.ProbeStatus
	err := sc.store.DB.WithContext(ctx).Where("widget_id = ?", widgetID).First(&ps).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil
		}
		return false, err
	}
	return now.Sub(ps.CheckedAt) >= time.Duration(intervalSec)*time.Second, nil
}

func (sc *Scheduler) markInflight(id int64) bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if _, ok := sc.inflight[id]; ok {
		return false
	}
	sc.inflight[id] = struct{}{}
	return true
}

func (sc *Scheduler) clearInflight(id int64) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.inflight, id)
}

func (sc *Scheduler) runOne(ctx context.Context, widgetID int64, cfg LinkProbeConfig) {
	defer sc.clearInflight(widgetID)
	spec := SpecFromLinkConfig(cfg, sc.defaultTimeoutSec)
	res := Probe(ctx, spec)

	// Peek previous status so we only broadcast on actual flips.
	// A missing row (first probe) counts as unknown; any change from the
	// initial unknown → ok/down fires once, as expected.
	prev := StatusUnknown
	var prevRow models.ProbeStatus
	if err := sc.store.DB.WithContext(ctx).
		Where("widget_id = ?", widgetID).
		First(&prevRow).Error; err == nil {
		prev = prevRow.Status
	}

	row := models.ProbeStatus{
		WidgetID:  widgetID,
		Status:    res.Status,
		LatencyMs: res.LatencyMs,
		CheckedAt: time.Now().UTC(),
	}
	if err := sc.store.DB.WithContext(ctx).
		Save(&row).Error; err != nil {
		sc.logger.Warn("write probe status failed", "widget_id", widgetID, "err", err)
		return
	}
	sc.logger.Debug("probe done", "widget_id", widgetID, "status", res.Status, "latency_ms", res.LatencyMs)

	if sc.hub != nil && prev != res.Status {
		sc.hub.Broadcast(events.Event{
			Type: events.TypeProbeChanged,
			Payload: map[string]any{
				"widget_id":  widgetID,
				"status":     res.Status,
				"latency_ms": res.LatencyMs,
				"checked_at": row.CheckedAt,
				"previous":   prev,
			},
		})
	}
}

func parseLinkConfig(raw json.RawMessage) (LinkProbeConfig, bool) {
	var cfg LinkProbeConfig
	if len(raw) == 0 || string(raw) == "null" {
		return cfg, false
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, false
	}
	return cfg, true
}

// ListStatuses returns recent rows.
func (sc *Scheduler) ListStatuses(ctx context.Context, widgetIDs []int64) ([]models.ProbeStatus, error) {
	q := sc.store.DB.WithContext(ctx).Model(&models.ProbeStatus{})
	if len(widgetIDs) > 0 {
		q = q.Where("widget_id IN ?", widgetIDs)
	}
	var rows []models.ProbeStatus
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

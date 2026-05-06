// Package datasource caches live DataSource handles:
// DB rows -> adapter registry -> connected instances.
package datasource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"astrolabe/internal/adapter"
	"astrolabe/internal/store"
)

// Manager caches connected datasource handles.
type Manager struct {
	store    *Storer
	registry *adapter.Registry

	mu        sync.Mutex
	instances map[int64]adapter.DataSource

	logger *slog.Logger
}

// Storer abstracts DB for tests.
type Storer struct {
	*store.Store
}

// NewManager constructs a manager.
func NewManager(s *store.Store, reg *adapter.Registry) *Manager {
	if reg == nil {
		reg = adapter.DefaultRegistry
	}
	return &Manager{
		store:     &Storer{s},
		registry:  reg,
		instances: make(map[int64]adapter.DataSource),
		logger:    slog.Default().With("module", "datasource"),
	}
}

// Close disconnects adapters.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, inst := range m.instances {
		if err := inst.Close(); err != nil {
			m.logger.Warn("ds close failed", "id", id, "err", err)
		}
	}
	m.instances = map[int64]adapter.DataSource{}
}

// Forget drops cache after CRUD.
func (m *Manager) Forget(id int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if inst, ok := m.instances[id]; ok {
		_ = inst.Close()
		delete(m.instances, id)
	}
}

// Get lazy-connects by id.
func (m *Manager) Get(ctx context.Context, id int64) (adapter.DataSource, error) {
	m.mu.Lock()
	if inst, ok := m.instances[id]; ok {
		m.mu.Unlock()
		return inst, nil
	}
	m.mu.Unlock()

	view, err := m.store.GetDataSource(ctx, id)
	if err != nil {
		return nil, err
	}
	cfg, err := buildAdapterConfig(view)
	if err != nil {
		return nil, err
	}
	inst, err := m.registry.New(cfg)
	if err != nil {
		return nil, err
	}
	if err := inst.Connect(ctx); err != nil {
		_ = inst.Close()
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// Double-checked locking: another goroutine may finish first.
	if existing, ok := m.instances[id]; ok {
		_ = inst.Close()
		return existing, nil
	}
	m.instances[id] = inst
	return inst, nil
}

// HealthCheck updates DB health text.
func (m *Manager) HealthCheck(ctx context.Context, id int64) (string, error) {
	inst, err := m.Get(ctx, id)
	if err != nil {
		_ = m.store.UpdateDataSourceHealth(ctx, id, store.DataSourceHealthError)
		return store.DataSourceHealthError, err
	}
	if err := inst.HealthCheck(ctx); err != nil {
		_ = m.store.UpdateDataSourceHealth(ctx, id, store.DataSourceHealthError)
		return store.DataSourceHealthError, err
	}
	if err := m.store.UpdateDataSourceHealth(ctx, id, store.DataSourceHealthOK); err != nil {
		return store.DataSourceHealthOK, err
	}
	return store.DataSourceHealthOK, nil
}

// TestConnect validates draft configs.
func (m *Manager) TestConnect(ctx context.Context, view *store.DataSourceView) error {
	cfg, err := buildAdapterConfig(view)
	if err != nil {
		return err
	}
	inst, err := m.registry.New(cfg)
	if err != nil {
		return err
	}
	defer inst.Close()
	if err := inst.Connect(ctx); err != nil {
		return err
	}
	return inst.HealthCheck(ctx)
}

// Discover proxies Discover RPC.
func (m *Manager) Discover(ctx context.Context, id int64) (adapter.MetricTree, error) {
	inst, err := m.Get(ctx, id)
	if err != nil {
		return adapter.MetricTree{}, err
	}
	return inst.Discover(ctx)
}

// Fetch passes through to adapters for pipelines.
func (m *Manager) Fetch(ctx context.Context, id int64, q adapter.MetricQuery) (adapter.DataPayload, error) {
	inst, err := m.Get(ctx, id)
	if err != nil {
		return adapter.DataPayload{}, err
	}
	return inst.Fetch(ctx, q)
}

// Types lists adapters from registry.
func (m *Manager) Types() []string {
	return m.registry.Types()
}

func buildAdapterConfig(v *store.DataSourceView) (adapter.Config, error) {
	cfg := adapter.Config{
		ID:       v.ID,
		Name:     v.Name,
		Type:     v.Type,
		Endpoint: v.Endpoint,
	}
	if len(v.Auth) > 0 && string(v.Auth) != "null" {
		if err := json.Unmarshal(v.Auth, &cfg.Auth); err != nil {
			return cfg, fmt.Errorf("auth json: %w", err)
		}
	}
	if len(v.Extra) > 0 && string(v.Extra) != "null" {
		if err := json.Unmarshal(v.Extra, &cfg.Extra); err != nil {
			return cfg, fmt.Errorf("extra json: %w", err)
		}
	}
	if cfg.Type == "" {
		return cfg, errors.New("data source type empty")
	}
	return cfg, nil
}

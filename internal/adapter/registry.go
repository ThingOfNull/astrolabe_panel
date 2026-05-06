package adapter

import (
	"fmt"
	"sort"
	"sync"
)

// Registry stores adapter factories for the process.
//
// Adapters call Register from init().
type Registry struct {
	mu        sync.RWMutex
	factories map[string]Factory
}

// NewRegistry isolates tests; production uses DefaultRegistry.
func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]Factory)}
}

// Register panics on duplicate type names.
func (r *Registry) Register(typ string, f Factory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.factories[typ]; exists {
		panic(fmt.Sprintf("adapter.Registry: type %q already registered", typ))
	}
	r.factories[typ] = f
}

// Lookup returns a factory by type.
func (r *Registry) Lookup(typ string) (Factory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.factories[typ]
	return f, ok
}

// Types lists registered kinds sorted.
func (r *Registry) Types() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]string, 0, len(r.factories))
	for t := range r.factories {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

// New instantiates from persisted config.
func (r *Registry) New(cfg Config) (DataSource, error) {
	f, ok := r.Lookup(cfg.Type)
	if !ok {
		return nil, fmt.Errorf("adapter: unknown type %q", cfg.Type)
	}
	return f(cfg)
}

// DefaultRegistry is the global factory map.
var DefaultRegistry = NewRegistry()

// Register aliases DefaultRegistry.Register.
func Register(typ string, f Factory) {
	DefaultRegistry.Register(typ, f)
}

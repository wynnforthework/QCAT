package registry

import (
    "fmt"
    "sync"

    "qcat/internal/strategy/sdk"
    mac "qcat/internal/strategy/strategies/ma_crossover"
    rsi "qcat/internal/strategy/strategies/rsi_mean_reversion"
    brk "qcat/internal/strategy/strategies/breakout"
)

// FactoryFunc creates a strategy from params
type FactoryFunc func(params map[string]interface{}) sdk.Strategy

// Registry holds mapping from type/name to factory functions
type Registry struct {
    mu       sync.RWMutex
    factories map[string]FactoryFunc
}

var defaultRegistry = &Registry{ factories: make(map[string]FactoryFunc) }

func init() {
    // Register built-ins
    defaultRegistry.Register("ma_crossover", func(params map[string]interface{}) sdk.Strategy { return mac.New(params) })
    defaultRegistry.Register("rsi_mean_reversion", func(params map[string]interface{}) sdk.Strategy { return rsi.New(params) })
    defaultRegistry.Register("breakout", func(params map[string]interface{}) sdk.Strategy { return brk.New(params) })
}

// Register adds a factory under a key
func (r *Registry) Register(key string, f FactoryFunc) {
    r.mu.Lock()
    defer r.mu.Unlock()
    if r.factories == nil { r.factories = make(map[string]FactoryFunc) }
    r.factories[key] = f
}

// Get returns a strategy by key and params
func (r *Registry) Get(key string, params map[string]interface{}) (sdk.Strategy, error) {
    r.mu.RLock()
    f, ok := r.factories[key]
    r.mu.RUnlock()
    if !ok { return nil, fmt.Errorf("strategy factory not found: %s", key) }
    return f(params), nil
}

// Global functions
func Register(key string, f FactoryFunc) { defaultRegistry.Register(key, f) }
func Get(key string, params map[string]interface{}) (sdk.Strategy, error) { return defaultRegistry.Get(key, params) }



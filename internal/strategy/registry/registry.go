package registry

import (
    "context"
    "fmt"
    "sync"

    "qcat/internal/exchange"
    legacy "qcat/internal/strategy"
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

// Adapter converts an sdk.Strategy to the legacy strategy.Strategy interface used by sandbox automation
type sdkAdapter struct {
    s sdk.Strategy
    state legacy.State
    result *legacy.Result
}

func (a *sdkAdapter) Initialize(ctx context.Context, params map[string]interface{}) error {
    cfg := &sdk.StrategyConfig{ Parameters: params, Mode: sdk.ModePaper }
    return a.s.Initialize(ctx, cfg)
}
func (a *sdkAdapter) Start(ctx context.Context) error { return nil }
func (a *sdkAdapter) Stop(ctx context.Context) error { return a.s.Stop(ctx) }
func (a *sdkAdapter) OnTick(ctx context.Context, data interface{}) error { return a.s.OnTick(ctx, data) }
func (a *sdkAdapter) OnSignal(ctx context.Context, signal *legacy.Signal) error { return nil }
func (a *sdkAdapter) OnOrder(ctx context.Context, order *exchange.Order) error { return a.s.OnOrderUpdate(ctx, order) }
func (a *sdkAdapter) OnPosition(ctx context.Context, position *exchange.Position) error { return a.s.OnPositionUpdate(ctx, position) }
func (a *sdkAdapter) GetState() legacy.State {
    st := a.s.GetState()
    if st != nil && st.Running { return legacy.StateRunning }
    return legacy.StateStopped
}
func (a *sdkAdapter) GetResult() *legacy.Result {
    if a.result == nil { a.result = &legacy.Result{ Metadata: map[string]interface{}{} } }
    return a.result
}

// GetLegacy returns a legacy strategy.Strategy by adapting sdk.Strategy
func GetLegacy(key string, params map[string]interface{}) (legacy.Strategy, error) {
    s, err := defaultRegistry.Get(key, params)
    if err != nil { return nil, err }
    return &sdkAdapter{s: s}, nil
}



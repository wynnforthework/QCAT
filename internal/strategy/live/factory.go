package live

import (
	"context"
	"fmt"
	"sync"

	"qcat/internal/exchange/order"
	"qcat/internal/exchange/position"
	"qcat/internal/exchange/risk"
	"qcat/internal/market"
	"qcat/internal/strategy"
	"qcat/internal/strategy/sandbox"
)

// Factory manages real-time strategy runners
type Factory struct {
	sandboxFactory *sandbox.Factory
	market         *market.Ingestor
	order          *order.Manager
	position       *position.Manager
	risk           *risk.Manager
	runners        map[string]*Runner
	mu             sync.RWMutex
}

// NewFactory creates a new runner factory
func NewFactory(sandboxFactory *sandbox.Factory, market *market.Ingestor, order *order.Manager, position *position.Manager, risk *risk.Manager) *Factory {
	return &Factory{
		sandboxFactory: sandboxFactory,
		market:         market,
		order:          order,
		position:       position,
		risk:           risk,
		runners:        make(map[string]*Runner),
	}
}

// CreateRunner creates a new real-time strategy runner
func (f *Factory) CreateRunner(strategy strategy.Strategy, config *strategy.Config) (*Runner, error) {
	// Convert config to map[string]interface{}
	configMap := make(map[string]interface{})
	// TODO: 待确认 - 转换配置结构
	if config != nil {
		configMap["name"] = config.Name
		configMap["symbol"] = config.Symbol
		configMap["mode"] = config.Mode
		configMap["params"] = config.Params
	}

	// Create sandbox
	sandbox, err := f.sandboxFactory.CreateSandbox(strategy, configMap, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox: %w", err)
	}

	// Create runner
	runner := NewRunner(sandbox, f.market, f.order, f.position, f.risk)

	// Store runner
	f.mu.Lock()
	f.runners[config.Name] = runner
	f.mu.Unlock()

	return runner, nil
}

// GetRunner returns a strategy runner by name
func (f *Factory) GetRunner(name string) (*Runner, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	runner, exists := f.runners[name]
	return runner, exists
}

// ListRunners returns all strategy runners
func (f *Factory) ListRunners() []*Runner {
	f.mu.RLock()
	defer f.mu.RUnlock()

	runners := make([]*Runner, 0, len(f.runners))
	for _, runner := range f.runners {
		runners = append(runners, runner)
	}
	return runners
}

// DeleteRunner deletes a strategy runner
func (f *Factory) DeleteRunner(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	runner, exists := f.runners[name]
	if !exists {
		return fmt.Errorf("runner not found: %s", name)
	}

	// Stop runner if running
	if runner.GetState() == "running" {
		if err := runner.Stop(context.Background()); err != nil {
			return fmt.Errorf("failed to stop runner: %w", err)
		}
	}

	delete(f.runners, name)
	return nil
}

// StartAll starts all strategy runners
func (f *Factory) StartAll(ctx context.Context) error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	for name, runner := range f.runners {
		if err := runner.Start(ctx); err != nil {
			return fmt.Errorf("failed to start runner %s: %w", name, err)
		}
	}
	return nil
}

// StopAll stops all strategy runners
func (f *Factory) StopAll(ctx context.Context) error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	for name, runner := range f.runners {
		if err := runner.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop runner %s: %w", name, err)
		}
	}
	return nil
}

// GetResults returns results for all strategy runners
func (f *Factory) GetResults() map[string]*strategy.Result {
	f.mu.RLock()
	defer f.mu.RUnlock()

	results := make(map[string]*strategy.Result)
	for name, runner := range f.runners {
		results[name] = runner.GetResult()
	}
	return results
}

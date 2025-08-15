package sandbox

import (
	"context"
	"fmt"
	"sync"

	"qcat/internal/exchange"
	"qcat/internal/strategy"
)

// Factory manages strategy sandboxes
type Factory struct {
	sandboxes map[string]*Sandbox
	mu        sync.RWMutex
}

// NewFactory creates a new sandbox factory
func NewFactory() *Factory {
	return &Factory{
		sandboxes: make(map[string]*Sandbox),
	}
}

// CreateSandbox creates a new strategy sandbox
func (f *Factory) CreateSandbox(strategy strategy.Strategy, config map[string]interface{}, exchange exchange.Exchange) (*Sandbox, error) {
	// Create sandbox
	sandbox := NewSandbox(strategy, config, exchange)

	// Validate sandbox
	if err := sandbox.Validate(); err != nil {
		return nil, fmt.Errorf("invalid sandbox: %w", err)
	}

	// Store sandbox
	f.mu.Lock()
	// TODO: 待确认 - 从配置中获取策略名
	strategyName := "sandbox-strategy"
	if name, ok := config["name"].(string); ok {
		strategyName = name
	}
	f.sandboxes[strategyName] = sandbox
	f.mu.Unlock()

	return sandbox, nil
}

// GetSandbox returns a strategy sandbox by name
func (f *Factory) GetSandbox(name string) (*Sandbox, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	sandbox, exists := f.sandboxes[name]
	return sandbox, exists
}

// ListSandboxes returns all strategy sandboxes
func (f *Factory) ListSandboxes() []*Sandbox {
	f.mu.RLock()
	defer f.mu.RUnlock()

	sandboxes := make([]*Sandbox, 0, len(f.sandboxes))
	for _, sandbox := range f.sandboxes {
		sandboxes = append(sandboxes, sandbox)
	}
	return sandboxes
}

// DeleteSandbox deletes a strategy sandbox
func (f *Factory) DeleteSandbox(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	sandbox, exists := f.sandboxes[name]
	if !exists {
		return fmt.Errorf("sandbox not found: %s", name)
	}

	// Stop sandbox if running
	if sandbox.GetState() == "running" {
		if err := sandbox.Stop(context.Background()); err != nil {
			return fmt.Errorf("failed to stop sandbox: %w", err)
		}
	}

	delete(f.sandboxes, name)
	return nil
}

// StartAll starts all strategy sandboxes
func (f *Factory) StartAll(ctx context.Context) error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	for name, sandbox := range f.sandboxes {
		if err := sandbox.Start(ctx); err != nil {
			return fmt.Errorf("failed to start sandbox %s: %w", name, err)
		}
	}
	return nil
}

// StopAll stops all strategy sandboxes
func (f *Factory) StopAll(ctx context.Context) error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	for name, sandbox := range f.sandboxes {
		if err := sandbox.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop sandbox %s: %w", name, err)
		}
	}
	return nil
}

// GetResults returns results for all strategy sandboxes
func (f *Factory) GetResults() map[string]*strategy.Result {
	f.mu.RLock()
	defer f.mu.RUnlock()

	results := make(map[string]*strategy.Result)
	for name, sandbox := range f.sandboxes {
		results[name] = sandbox.GetResult()
	}
	return results
}

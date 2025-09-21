package operations

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"qcat/internal/automation"
	"qcat/internal/config"
	"qcat/internal/exchange"
	"qcat/internal/stability"
)

type Manager struct {
	cfg        *config.Config
	automation *automation.AutomationSystem
	exchange   exchange.Exchange
	health     *stability.HealthChecker

	modeName   string
	modeConfig *config.OperationalModeConfig

	mu      sync.RWMutex
	checks  map[string]Check
	order   []string
	results map[string]Result

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewManager builds an operational manager and registers default checks.
func NewManager(cfg *config.Config, automation *automation.AutomationSystem, exch exchange.Exchange, health *stability.HealthChecker) *Manager {
	modeName, modeCfg := cfg.Operational.GetActiveMode()
	mgr := &Manager{
		cfg:        cfg,
		automation: automation,
		exchange:   exch,
		health:     health,
		modeName:   modeName,
		modeConfig: modeCfg,
		checks:     make(map[string]Check),
		results:    make(map[string]Result),
	}

	mgr.registerDefaultChecks()
	return mgr
}

// SetAutomation allows updating the automation system pointer after startup.
func (m *Manager) SetAutomation(system *automation.AutomationSystem) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.automation = system
}

// SetExchange updates the exchange client reference.
func (m *Manager) SetExchange(exch exchange.Exchange) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exchange = exch
}

// RunStartupChecks executes all startup checks synchronously. Returns error if any critical failures.
func (m *Manager) RunStartupChecks(ctx context.Context) error {
	m.mu.RLock()
	ids := append([]string(nil), m.order...)
	m.mu.RUnlock()

	var failures []string
	for _, id := range ids {
		check := m.getCheck(id)
		if check == nil {
			continue
		}
		if check.Mode() == CheckModeRuntime {
			continue
		}
		if ctx != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}
		result := m.executeCheck(check)
		if result.Status == StatusCritical {
			failures = append(failures, fmt.Sprintf("%s: %s", result.Name, result.Summary))
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("startup checks failed: %s", strings.Join(failures, "; "))
	}

	return nil
}

// RunAllChecksOnce triggers every registered check once regardless of mode.
func (m *Manager) RunAllChecksOnce() {
	m.mu.RLock()
	ids := append([]string(nil), m.order...)
	m.mu.RUnlock()

	for _, id := range ids {
		if chk := m.getCheck(id); chk != nil {
			m.executeCheck(chk)
		}
	}
}

// StartMonitoring launches runtime checks in background goroutines.
func (m *Manager) StartMonitoring(ctx context.Context) {
	m.mu.Lock()
	if m.ctx != nil {
		m.mu.Unlock()
		return
	}

	if ctx == nil {
		ctx = context.Background()
	}

	m.ctx, m.cancel = context.WithCancel(ctx)

	checks := make([]Check, 0, len(m.checks))
	for _, id := range m.order {
		if c := m.checks[id]; c != nil {
			checks = append(checks, c)
		}
	}
	m.mu.Unlock()

	for _, check := range checks {
		if check.Mode() == CheckModeStartup {
			continue
		}
		interval := check.Interval()
		if interval <= 0 {
			interval = time.Minute
		}

		m.wg.Add(1)
		go func(chk Check, poll time.Duration) {
			defer m.wg.Done()
			// Run immediately once
			m.executeCheck(chk)

			ticker := time.NewTicker(poll)
			defer ticker.Stop()

			for {
				select {
				case <-m.ctx.Done():
					return
				case <-ticker.C:
					m.executeCheck(chk)
				}
			}
		}(check, interval)
	}
}

// Stop terminates runtime monitoring.
func (m *Manager) Stop() {
	m.mu.Lock()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
		m.ctx = nil
	}
	m.mu.Unlock()
	m.wg.Wait()
}

// GetSnapshot returns the latest operational snapshot.
func (m *Manager) GetSnapshot() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	domains := make(map[Domain]DomainSummary)
	for _, id := range m.order {
		res, ok := m.results[id]
		if !ok {
			continue
		}
		summary := domains[res.Domain]
		summary.Status = combineStatus(summary.Status, res.Status)
		summary.Checks = append(summary.Checks, res)
		domains[res.Domain] = summary
	}

	// Ensure we have deterministic order inside each domain
	for domain, summary := range domains {
		sort.SliceStable(summary.Checks, func(i, j int) bool {
			return summary.Checks[i].ID < summary.Checks[j].ID
		})
		domains[domain] = summary
	}

	return Snapshot{
		Mode:        m.modeName,
		GeneratedAt: time.Now(),
		Domains:     domains,
	}
}

func (m *Manager) registerCheck(check Check) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.checks[check.ID()]; exists {
		return
	}
	m.checks[check.ID()] = check
	m.order = append(m.order, check.ID())
}

func (m *Manager) getCheck(id string) Check {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.checks[id]
}

func (m *Manager) executeCheck(check Check) Result {
	start := time.Now()
	result := check.Run()
	if result.ID == "" {
		result.ID = check.ID()
	}
	if result.Domain == "" {
		result.Domain = check.Domain()
	}
	if result.Name == "" {
		result.Name = check.ID()
	}
	if result.Status == "" {
		result.Status = StatusUnknown
	}
	result.StartedAt = start
	result.FinishedAt = time.Now()
	result.Duration = result.FinishedAt.Sub(start)

	m.mu.Lock()
	m.results[result.ID] = result
	m.mu.Unlock()
	return result
}

func (m *Manager) getAutomation() *automation.AutomationSystem {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.automation
}

func (m *Manager) getExchange() exchange.Exchange {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.exchange
}

func (m *Manager) getHealthChecker() *stability.HealthChecker {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.health
}
func combineStatus(current Status, incoming Status) Status {
	order := map[Status]int{
		StatusCritical: 3,
		StatusWarning:  2,
		StatusHealthy:  1,
		StatusUnknown:  0,
		"":             0,
	}
	if order[incoming] > order[current] {
		return incoming
	}
	if current == "" {
		if incoming == "" {
			return StatusUnknown
		}
		return incoming
	}
	return current
}

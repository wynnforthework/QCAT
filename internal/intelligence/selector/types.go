package selector

import (
	"context"
	"sync"
	"time"
)

// MarketContext captures minimal market and account context needed for selection
type MarketContext struct {
	Symbol          string
	Timestamp       time.Time
	Features        map[string]float64
	Regime          string
	AccountEquity   float64
	AvailableMargin float64
}

// EnabledStrategyLite is a light-weight view of a candidate strategy
type EnabledStrategyLite struct {
	ID   string
	Name string
	Type string
}

// SelectionResult describes a selector decision
type SelectionResult struct {
	Symbol           string
	SelectedID       string
	Method           string
	Scores           map[string]float64 // strategyID -> score
	Weights          map[string]float64 // optional: strategyID -> weight
	Regime           string
	Exploration      bool
	Reason           string
	DecisionUnixNano int64
}

// StrategySelector decides which strategy to use under current context
type StrategySelector interface {
	Select(ctx context.Context, symbol string, mctx *MarketContext, candidates []EnabledStrategyLite) (*SelectionResult, error)
	UpdatePerformance(symbol string, strategyID string, sample PerfSample) error
	GetLastDecision(symbol string) (*SelectionResult, bool)
	GetStats(symbol string) map[string]interface{}
}

// RegimeDetector classifies market regime and returns contextual features
type RegimeDetector interface {
	Detect(ctx context.Context, symbol string) (regime string, features map[string]float64, err error)
}

// PerfSample is a single performance observation
type PerfSample struct {
	PnL           float64
	Return        float64
	Drawdown      float64
	Win           bool
	Cost          float64
	Duration      time.Duration
	Timestamp     time.Time
}

// PerfStats is an aggregated view in a time bucket
type PerfStats struct {
	Count        int
	CumReturn    float64
	CumPnL       float64
	Wins         int
	Losses       int
	MaxDrawdown  float64
	AvgCost      float64
	LastUpdate   time.Time
}

// PerformanceStore stores stats per (symbol,strategy,regime,bucket)
type PerformanceStore interface {
	GetStats(symbol, strategyID, regime string, bucket time.Duration) (*PerfStats, bool)
	Upsert(symbol, strategyID, regime string, bucket time.Duration, s PerfSample) error
}

// thread-safe memory store implementation
type memoryStore struct {
	mu   sync.RWMutex
	data map[string]*PerfStats // key: symbol|strategy|regime|bucket
}

func NewMemoryPerformanceStore() PerformanceStore {
	return &memoryStore{data: make(map[string]*PerfStats)}
}

func perfKey(symbol, strategy, regime string, bucket time.Duration) string {
	return symbol + "|" + strategy + "|" + regime + "|" + bucket.String()
}

func (m *memoryStore) GetStats(symbol, strategyID, regime string, bucket time.Duration) (*PerfStats, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ps, ok := m.data[perfKey(symbol, strategyID, regime, bucket)]
	return ps, ok
}

func (m *memoryStore) Upsert(symbol, strategyID, regime string, bucket time.Duration, s PerfSample) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := perfKey(symbol, strategyID, regime, bucket)
	ps, ok := m.data[key]
	if !ok {
		ps = &PerfStats{}
		m.data[key] = ps
	}
	ps.Count++
	ps.CumReturn += s.Return
	ps.CumPnL += s.PnL
	if s.Win {
		ps.Wins++
	} else {
		ps.Losses++
	}
	if s.Drawdown > ps.MaxDrawdown {
		ps.MaxDrawdown = s.Drawdown
	}
	// simple moving average for cost
	if ps.Count == 1 {
		ps.AvgCost = s.Cost
	} else {
		ps.AvgCost = ps.AvgCost + (s.Cost-ps.AvgCost)/float64(ps.Count)
	}
	ps.LastUpdate = time.Now()
	return nil
}



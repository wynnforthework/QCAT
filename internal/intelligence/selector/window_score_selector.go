package selector

import (
	"context"
	"errors"
	"sort"
	"time"
)

// Config for window-score selector
type WindowScoreConfig struct {
	Window       time.Duration `yaml:"window"`
	MinTrades    int           `yaml:"min_trades"`
	Weights      map[string]float64 `yaml:"weights"` // return, sharpe, mdd, cost
	MethodName   string        `yaml:"-"`
}

// WindowScoreSelector implements a simple scoring over recent window
type WindowScoreSelector struct {
	store    PerformanceStore
	regime   RegimeDetector
	config   WindowScoreConfig
	last     map[string]*SelectionResult // by symbol
}

func NewWindowScoreSelector(store PerformanceStore, regime RegimeDetector, cfg WindowScoreConfig) *WindowScoreSelector {
	if cfg.Window == 0 {
		cfg.Window = 14 * 24 * time.Hour
	}
	if cfg.Weights == nil {
		cfg.Weights = map[string]float64{"return": 0.5, "sharpe": 0.3, "mdd": -0.2, "cost": -0.2}
	}
	cfg.MethodName = "window_score"
	return &WindowScoreSelector{store: store, regime: regime, config: cfg, last: make(map[string]*SelectionResult)}
}

func (w *WindowScoreSelector) Select(ctx context.Context, symbol string, mctx *MarketContext, candidates []EnabledStrategyLite) (*SelectionResult, error) {
	if len(candidates) == 0 {
		return nil, errors.New("no candidates")
	}
	regime := mctx.Regime
	if regime == "" && w.regime != nil {
		r, feats, _ := w.regime.Detect(ctx, symbol)
		regime = r
		if mctx.Features == nil {
			mctx.Features = feats
		}
	}
	// score each strategy using simple heuristics from PerfStats
	scores := make(map[string]float64)
	type kv struct{ id string; s float64 }
	var arr []kv
	for _, c := range candidates {
		stat, ok := w.store.GetStats(symbol, c.ID, regime, w.config.Window)
		if !ok || stat.Count < w.config.MinTrades {
			continue
		}
		// approximate sharpe
		avgRet := 0.0
		if stat.Count > 0 {
			avgRet = stat.CumReturn / float64(stat.Count)
		}
		winRate := 0.0
		if (stat.Wins+stat.Losses) > 0 {
			winRate = float64(stat.Wins) / float64(stat.Wins+stat.Losses)
		}
		// proxy sharpe using avgRet and winRate
		sharpeProxy := avgRet * (0.5 + 0.5*winRate)
		s := 0.0
		s += w.config.Weights["return"] * stat.CumReturn
		s += w.config.Weights["sharpe"] * sharpeProxy
		s += w.config.Weights["mdd"] * stat.MaxDrawdown
		s += w.config.Weights["cost"] * stat.AvgCost
		scores[c.ID] = s
		arr = append(arr, kv{id: c.ID, s: s})
	}
	if len(arr) == 0 {
		// fallback: choose first
		res := &SelectionResult{Symbol: symbol, SelectedID: candidates[0].ID, Method: w.config.MethodName, Scores: map[string]float64{candidates[0].ID: 0}, Regime: regime, DecisionUnixNano: time.Now().UnixNano()}
		w.last[symbol] = res
		return res, nil
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].s > arr[j].s })
	res := &SelectionResult{
		Symbol:           symbol,
		SelectedID:       arr[0].id,
		Method:           w.config.MethodName,
		Scores:           scores,
		Regime:           regime,
		Exploration:      false,
		DecisionUnixNano: time.Now().UnixNano(),
	}
	w.last[symbol] = res
	return res, nil
}

func (w *WindowScoreSelector) UpdatePerformance(symbol string, strategyID string, sample PerfSample) error {
	return w.store.Upsert(symbol, strategyID, "", w.config.Window, sample)
}

func (w *WindowScoreSelector) GetLastDecision(symbol string) (*SelectionResult, bool) {
	res, ok := w.last[symbol]
	return res, ok
}

func (w *WindowScoreSelector) GetStats(symbol string) map[string]interface{} {
	res := map[string]interface{}{}
	if last, ok := w.last[symbol]; ok {
		res["last"] = last
	}
	return res
}



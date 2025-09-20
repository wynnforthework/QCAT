package backtesting

import (
    "context"
    "fmt"
    "time"
)

// WalkForwardConfig defines rolling optimization/backtest settings
type WalkForwardConfig struct {
    Symbol        string
    Interval      string
    Start         time.Time
    End           time.Time
    OptimizeWindow time.Duration // e.g., 365*24h
    TestWindow     time.Duration // e.g., 90*24h
    Step           time.Duration // e.g., 90*24h
}

// Optimizer is the minimal interface needed to evaluate params on a window
type Optimizer interface {
    Optimize(ctx context.Context, symbol, interval string, start, end time.Time) (map[string]interface{}, float64, error)
}

// Backtester runs a backtest for a fixed parameter set over a window
type Backtester interface {
    Backtest(ctx context.Context, symbol, interval string, start, end time.Time, params map[string]interface{}) (*WalkForwardSliceResult, error)
}

// WalkForwardSliceResult is the per-slice performance
type WalkForwardSliceResult struct {
    Start   time.Time
    End     time.Time
    Metrics map[string]float64
}

// WalkForwardResult aggregates all slices
type WalkForwardResult struct {
    Slices     []WalkForwardSliceResult
    Parameters []map[string]interface{}
}

// RunWalkForward performs rolling optimize-then-test to avoid overfitting
func RunWalkForward(ctx context.Context, cfg WalkForwardConfig, opt Optimizer, bt Backtester) (*WalkForwardResult, error) {
    if cfg.OptimizeWindow <= 0 || cfg.TestWindow <= 0 || cfg.Step <= 0 {
        return nil, fmt.Errorf("invalid walk-forward windows")
    }
    var result WalkForwardResult
    current := cfg.Start
    for {
        optimizeStart := current
        optimizeEnd := optimizeStart.Add(cfg.OptimizeWindow)
        testStart := optimizeEnd
        testEnd := testStart.Add(cfg.TestWindow)
        if testStart.After(cfg.End) {
            break
        }
        if testEnd.After(cfg.End) {
            testEnd = cfg.End
        }

        // optimize on in-sample
        params, _, err := opt.Optimize(ctx, cfg.Symbol, cfg.Interval, optimizeStart, optimizeEnd)
        if err != nil {
            return nil, fmt.Errorf("optimize failed: %w", err)
        }
        result.Parameters = append(result.Parameters, params)

        // test out-of-sample
        sliceRes, err := bt.Backtest(ctx, cfg.Symbol, cfg.Interval, testStart, testEnd, params)
        if err != nil {
            return nil, fmt.Errorf("backtest failed: %w", err)
        }
        result.Slices = append(result.Slices, *sliceRes)

        // step forward
        current = current.Add(cfg.Step)
        if current.After(cfg.End) {
            break
        }
    }
    return &result, nil
}



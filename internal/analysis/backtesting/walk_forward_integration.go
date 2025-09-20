package backtesting

import (
    "context"
    "time"
)

// DefaultWalkForward config helper: 3 years total, 1y optimize, 3m test, 3m step
func DefaultWalkForward(symbol, interval string, start, end time.Time, opt Optimizer, bt Backtester) (*WalkForwardResult, error) {
    cfg := WalkForwardConfig{
        Symbol: symbol,
        Interval: interval,
        Start: start,
        End: end,
        OptimizeWindow: 365 * 24 * time.Hour,
        TestWindow: 90 * 24 * time.Hour,
        Step: 90 * 24 * time.Hour,
    }
    return RunWalkForward(context.Background(), cfg, opt, bt)
}



package backtesting

import (
    "context"
    "database/sql"
    "time"

    "qcat/internal/strategy/backtest"
    "qcat/internal/strategy/sdk"
)

// SimpleOptimizer uses grid search over given ranges via backtests to pick best Sharpe
type SimpleOptimizer struct {
    DB       *sql.DB
    Ranges   map[string][]interface{}
    MakeStrategy func(params map[string]interface{}) sdk.Strategy
}

func (o *SimpleOptimizer) Optimize(ctx context.Context, symbol, interval string, start, end time.Time) (map[string]interface{}, float64, error) {
    bestParams := map[string]interface{}{}
    var bestScore float64 = -1e9
    // degenerate case: no ranges -> single run
    candidates := []map[string]interface{}{}
    if len(o.Ranges) == 0 {
        candidates = append(candidates, map[string]interface{}{})
    } else {
        // generate cartesian product
        keys := make([]string, 0, len(o.Ranges))
        for k := range o.Ranges {
            keys = append(keys, k)
        }
        var build func(i int, cur map[string]interface{})
        build = func(i int, cur map[string]interface{}) {
            if i == len(keys) {
                c := make(map[string]interface{}, len(cur))
                for k, v := range cur { c[k] = v }
                candidates = append(candidates, c)
                return
            }
            k := keys[i]
            for _, v := range o.Ranges[k] {
                cur[k] = v
                build(i+1, cur)
            }
        }
        build(0, map[string]interface{}{})
    }

    for _, p := range candidates {
    strat := o.MakeStrategy(p)
        res, err := runBacktest(ctx, o.DB, strat, symbol, interval, start, end)
        if err != nil { continue }
        if res.PerformanceStats != nil && res.PerformanceStats.SharpeRatio > bestScore {
            bestScore = res.PerformanceStats.SharpeRatio
            bestParams = p
        }
    }
    return bestParams, bestScore, nil
}

// SimpleBacktester wraps internal/strategy/backtest engine
type SimpleBacktester struct { DB *sql.DB; MakeStrategy func(params map[string]interface{}) sdk.Strategy }

func (b *SimpleBacktester) Backtest(ctx context.Context, symbol, interval string, start, end time.Time, params map[string]interface{}) (*WalkForwardSliceResult, error) {
    strat := b.MakeStrategy(params)
    res, err := runBacktest(ctx, b.DB, strat, symbol, interval, start, end)
    if err != nil { return nil, err }
    metrics := map[string]float64{}
    if res.PerformanceStats != nil {
        metrics["sharpe"] = res.PerformanceStats.SharpeRatio
        metrics["total_return"] = res.PerformanceStats.TotalReturn
        metrics["max_drawdown"] = res.PerformanceStats.MaxDrawdown
    }
    return &WalkForwardSliceResult{ Start: start, End: end, Metrics: metrics }, nil
}

func runBacktest(ctx context.Context, db *sql.DB, strat sdk.Strategy, symbol, interval string, start, end time.Time) (*backtest.Result, error) {
    cfg := &backtest.Config{
        InitialCapital: 10000,
        MarginMode: 0,
        Leverage: 1,
        Symbols: []string{symbol},
        Interval: interval,
        StartTime: start,
        EndTime: end,
        DataTypes: []string{"kline"},
        Capital: 10000,
        DataFeedType: "db",
    }
    feed, err := backtest.NewDBDataFeed(db, cfg)
    if err != nil { return nil, err }
    if err := feed.Load(ctx); err != nil { return nil, err }
    data := &backtest.HistoricalData{ Start: start, End: end, Feed: feed }
    engine := backtest.NewEngine(data, strat, cfg)
    return engine.Run(ctx)
}



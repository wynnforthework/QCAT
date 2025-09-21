package backtesting

import (
    "context"
    "database/sql"
    "time"

    "qcat/internal/exchange"
    "qcat/internal/market/kline"
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
        MarginMode:     exchange.MarginTypeCross,
        Leverage:       1,
        Symbols:        []string{symbol},
        Interval:       interval,
        StartTime:      start,
        EndTime:        end,
        DataTypes:      []string{"kline"},
        Capital:        10000,
    }

    // Load klines directly from DB into HistoricalData
    data := &backtest.HistoricalData{ Symbol: symbol, Start: start, End: end }
    if db != nil {
        rows, err := db.QueryContext(ctx,
            `SELECT timestamp, open, high, low, close, volume
             FROM market_data
             WHERE symbol = $1 AND interval = $2 AND timestamp BETWEEN $3 AND $4
             ORDER BY timestamp ASC`,
            symbol, interval, start, end,
        )
        if err == nil {
            defer rows.Close()
            for rows.Next() {
                var openTime time.Time
                var open, high, low, close, volume float64
                if err := rows.Scan(&openTime, &open, &high, &low, &close, &volume); err != nil {
                    continue
                }
                kl := &kline.Kline{
                    Symbol:    symbol,
                    Interval:  kline.Interval(interval),
                    OpenTime:  openTime,
                    Open:      open,
                    High:      high,
                    Low:       low,
                    Close:     close,
                    Volume:    volume,
                    Complete:  true,
                }
                // derive CloseTime from interval
                kl.CloseTime = openTime.Add(kline.GetIntervalDuration(kline.Interval(interval)))
                data.Klines = append(data.Klines, kl)
            }
            _ = rows.Err()
        }
    }

    engine := backtest.NewEngine(data, strat, cfg)
    return engine.Run(ctx)
}



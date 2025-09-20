package ma_crossover

import (
    "context"
    "time"

    "qcat/internal/market/kline"
    "qcat/internal/strategy/sdk"
)

// Strategy implements a simple moving average crossover
type Strategy struct {
    base        *sdk.BaseStrategy
    short int
    long  int
    prices []float64
    exec   sdk.Execution
}

func New(params map[string]interface{}) *Strategy {
    s := &Strategy{ base: sdk.NewBaseStrategy(), short: 10, long: 30 }
    if v, ok := params["ma_short"].(int); ok && v > 0 { s.short = v }
    if v, ok := params["ma_long"].(int); ok && v > s.short { s.long = v }
    return s
}

func (s *Strategy) Initialize(ctx context.Context, config *sdk.StrategyConfig) error { return s.base.Initialize(ctx, config) }
func (s *Strategy) OnOrderBook(ctx context.Context, _ interface{}) error { return nil }
func (s *Strategy) OnTrade(ctx context.Context, _ interface{}) error { return nil }
func (s *Strategy) OnPositionUpdate(ctx context.Context, _ interface{}) error { return nil }
func (s *Strategy) OnOrderUpdate(ctx context.Context, _ interface{}) error { return nil }
func (s *Strategy) OnTimer(ctx context.Context, _ time.Time) error { return nil }
func (s *Strategy) OnTick(ctx context.Context, tick interface{}) error { if k, ok := tick.(*kline.Kline); ok { return s.OnKline(ctx, k) }; return nil }
func (s *Strategy) GetState() *sdk.StrategyState { return s.base.GetState() }
func (s *Strategy) Stop(ctx context.Context) error { return s.base.Stop(ctx) }

// ApplyParams supports hot parameter updates
func (s *Strategy) ApplyParams(ctx context.Context, params map[string]interface{}) error {
    if v, ok := params["ma_short"].(int); ok && v > 0 { s.short = v }
    if v, ok := params["ma_long"].(int); ok && v > s.short { s.long = v }
    return nil
}

// SetExecution injects execution for backtests
func (s *Strategy) SetExecution(exec sdk.Execution) { s.exec = exec }

func (s *Strategy) OnKline(ctx context.Context, k *kline.Kline) error {
    // append price
    s.prices = append(s.prices, k.Close)
    if len(s.prices) > s.long+2 { s.prices = s.prices[len(s.prices)-(s.long+2):] }
    if len(s.prices) < s.long { return nil }
    var sa, la float64
    for i := len(s.prices)-s.short; i < len(s.prices); i++ { sa += s.prices[i] }
    for i := len(s.prices)-s.long; i < len(s.prices); i++ { la += s.prices[i] }
    sa /= float64(s.short); la /= float64(s.long)
    // generate basic crossover signals -> market orders
    if s.exec != nil {
        // 取当前与上一根，用均线差的变化判断穿越
        // 简化：当前sa>la则做多，否则做空（市价单）
        side := "BUY"
        if sa < la { side = "SELL" }
        ord := &exchange.Order{
            ID:        k.Symbol + "-" + k.OpenTime.Format("20060102150405"),
            Symbol:    k.Symbol,
            Side:      side,
            Type:      string(exchange.OrderTypeMarket),
            Price:     k.Close,
            Quantity:  1.0, // 简化固定手数
            CreatedAt: k.OpenTime,
        }
        _ = s.exec.PlaceOrder(ord)
    }
    return nil
}



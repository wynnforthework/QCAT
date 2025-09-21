package breakout

import (
    "context"
    "time"

    "qcat/internal/market/kline"
    "qcat/internal/market/orderbook"
    "qcat/internal/market/trade"
    "qcat/internal/strategy/sdk"
    "qcat/internal/exchange"
)

type Strategy struct {
    base *sdk.BaseStrategy
    exec sdk.Execution
    lookback int
    closes []float64
}

func New(params map[string]interface{}) *Strategy {
    s := &Strategy{ base: sdk.NewBaseStrategy(), lookback: 20 }
    if v, ok := params["lookback"].(int); ok && v > 1 { s.lookback = v }
    return s
}

func (s *Strategy) Initialize(ctx context.Context, _ *sdk.StrategyConfig) error { return s.base.Initialize(ctx, nil) }
func (s *Strategy) OnKline(ctx context.Context, k *kline.Kline) error {
    s.closes = append(s.closes, k.Close)
    if len(s.closes) < s.lookback { return nil }
    high, low := s.closes[len(s.closes)-s.lookback], s.closes[len(s.closes)-s.lookback]
    for i := len(s.closes)-s.lookback; i < len(s.closes); i++ { if s.closes[i] > high { high = s.closes[i] }; if s.closes[i] < low { low = s.closes[i] } }
    if s.exec != nil {
        var side string
        if k.Close > high { side = "BUY" } else if k.Close < low { side = "SELL" }
        if side != "" { _ = s.exec.PlaceOrder(&exchange.Order{ ID: k.Symbol+"-"+k.OpenTime.Format("20060102150405"), Symbol: k.Symbol, Side: side, Type: string(exchange.OrderTypeMarket), Price: k.Close, Quantity: 1.0, CreatedAt: k.OpenTime }) }
    }
    return nil
}
func (s *Strategy) OnOrderBook(ctx context.Context, ob *orderbook.OrderBook) error { return nil }
func (s *Strategy) OnTrade(ctx context.Context, t *trade.Trade) error { return nil }
func (s *Strategy) OnPositionUpdate(ctx context.Context, pos *exchange.Position) error { return nil }
func (s *Strategy) OnOrderUpdate(ctx context.Context, ord *exchange.Order) error { return nil }
func (s *Strategy) OnTimer(ctx context.Context, _ time.Time) error { return nil }
func (s *Strategy) OnTick(ctx context.Context, tick interface{}) error { if k, ok := tick.(*kline.Kline); ok { return s.OnKline(ctx, k) }; return nil }
func (s *Strategy) GetState() *sdk.StrategyState { return s.base.GetState() }
func (s *Strategy) Stop(ctx context.Context) error { return s.base.Stop(ctx) }
func (s *Strategy) ApplyParams(ctx context.Context, params map[string]interface{}) error { if v, ok := params["lookback"].(int); ok && v > 1 { s.lookback = v }; return nil }
func (s *Strategy) SetExecution(exec sdk.Execution) { s.exec = exec }



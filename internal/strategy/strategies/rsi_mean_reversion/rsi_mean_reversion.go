package rsi_mean_reversion

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
	base  *sdk.BaseStrategy
	exec  sdk.Execution
	period int
	buy    float64
	sell   float64
	closes []float64
}

func New(params map[string]interface{}) *Strategy {
    s := &Strategy{ base: sdk.NewBaseStrategy(), period: 14, buy: 30, sell: 70 }
    if v, ok := params["period"].(int); ok && v > 1 { s.period = v }
    if v, ok := params["buy"].(float64); ok && v > 0 { s.buy = v }
    if v, ok := params["sell"].(float64); ok && v > 0 { s.sell = v }
    return s
}

func (s *Strategy) Initialize(ctx context.Context, config *sdk.StrategyConfig) error {
    return s.base.Initialize(ctx, config)
}

func (s *Strategy) OnKline(ctx context.Context, k *kline.Kline) error {
    s.closes = append(s.closes, k.Close)
    if len(s.closes) < s.period+1 {
        return nil
    }

    rsi := calcRSI(s.closes, s.period)
    if s.exec != nil {
        var side string
        if rsi < s.buy {
            side = "BUY"
        } else if rsi > s.sell {
            side = "SELL"
        }
        if side != "" {
            _ = s.exec.PlaceOrder(&exchange.Order{
                ID:        k.Symbol + "-" + k.OpenTime.Format("20060102150405"),
                Symbol:    k.Symbol,
                Side:      side,
                Type:      string(exchange.OrderTypeMarket),
                Price:     k.Close,
                Quantity:  1.0,
                CreatedAt: k.OpenTime,
            })
        }
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
func (s *Strategy) ApplyParams(ctx context.Context, params map[string]interface{}) error { if v, ok := params["period"].(int); ok && v > 1 { s.period = v }; if v, ok := params["buy"].(float64); ok { s.buy = v }; if v, ok := params["sell"].(float64); ok { s.sell = v }; return nil }
func (s *Strategy) SetExecution(exec sdk.Execution) { s.exec = exec }

func calcRSI(closes []float64, period int) float64 {
	if len(closes) < period+1 { return 50 }
	g, l := 0.0, 0.0
	for i := len(closes)-period; i < len(closes); i++ {
		d := closes[i] - closes[i-1]
		if d > 0 { g += d } else { l -= d }
	}
	if g+l == 0 { return 50 }
	rs := g / (l + 1e-9)
	return 100 - 100/(1+rs)
}

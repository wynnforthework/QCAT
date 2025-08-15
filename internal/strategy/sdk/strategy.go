package sdk

import (
	"context"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/market/kline"
	"qcat/internal/market/orderbook"
	"qcat/internal/market/trade"
)

// Strategy defines the interface that all strategies must implement
type Strategy interface {
	// Initialize initializes the strategy with configuration
	Initialize(ctx context.Context, config *StrategyConfig) error

	// OnKline handles new kline data
	OnKline(ctx context.Context, k *kline.Kline) error

	// OnOrderBook handles orderbook updates
	OnOrderBook(ctx context.Context, ob *orderbook.OrderBook) error

	// OnTrade handles new trade data
	OnTrade(ctx context.Context, t *trade.Trade) error

	// OnPositionUpdate handles position updates
	OnPositionUpdate(ctx context.Context, pos *exchange.Position) error

	// OnOrderUpdate handles order status updates
	OnOrderUpdate(ctx context.Context, order *exchange.Order) error

	// OnTimer handles timer events
	OnTimer(ctx context.Context, t time.Time) error

	// OnTick handles tick data (for backtesting)
	OnTick(ctx context.Context, tick interface{}) error

	// GetState returns current strategy state
	GetState() *StrategyState

	// Stop gracefully stops the strategy
	Stop(ctx context.Context) error
}

// StrategyConfig represents strategy configuration
type StrategyConfig struct {
	ID         string                 // 策略ID
	Name       string                 // 策略名称
	Symbols    []string               // 交易对列表
	Parameters map[string]interface{} // 策略参数
	RiskLimits *RiskLimits            // 风控限制
	Mode       RunMode                // 运行模式
}

// RiskLimits defines risk control limits
type RiskLimits struct {
	MaxPositionValue float64 // 最大仓位价值
	MaxLeverage      float64 // 最大杠杆倍数
	MaxDrawdown      float64 // 最大回撤限制
	StopLoss         float64 // 止损比例
	TakeProfit       float64 // 止盈比例
}

// RunMode defines strategy running mode
type RunMode int

const (
	ModePaper  RunMode = iota // 纸交易模式
	ModeShadow                // 影子跟单模式
	ModeCanary                // 小额实盘模式
	ModeLive                  // 完全实盘模式
)

// StrategyState represents current strategy state
type StrategyState struct {
	Running         bool                 // 是否在运行
	Mode            RunMode              // 当前模式
	StartTime       time.Time            // 启动时间
	TotalPnL        float64              // 总盈亏
	CurrentDrawdown float64              // 当前回撤
	Positions       []*exchange.Position // 当前持仓
	Orders          []*exchange.Order    // 活动订单
}

// StrategyMetrics represents strategy performance metrics
type StrategyMetrics struct {
	StrategyID    string    // 策略ID
	TotalReturn   float64   // 总收益率
	AnnualReturn  float64   // 年化收益率
	SharpeRatio   float64   // 夏普比率
	MaxDrawdown   float64   // 最大回撤
	Volatility    float64   // 波动率
	WinRate       float64   // 胜率
	ProfitFactor  float64   // 盈亏比
	TotalTrades   int       // 总交易次数
	WinningTrades int       // 盈利交易次数
	LosingTrades  int       // 亏损交易次数
	AverageWin    float64   // 平均盈利
	AverageLoss   float64   // 平均亏损
	LastUpdated   time.Time // 最后更新时间
}

// BaseStrategy provides common functionality for strategies
type BaseStrategy struct {
	config *StrategyConfig
	state  *StrategyState
}

// NewBaseStrategy creates a new base strategy
func NewBaseStrategy() *BaseStrategy {
	return &BaseStrategy{
		state: &StrategyState{
			Running: false,
		},
	}
}

// Initialize implements Strategy interface
func (s *BaseStrategy) Initialize(ctx context.Context, config *StrategyConfig) error {
	s.config = config
	s.state.Mode = config.Mode
	s.state.StartTime = time.Now()
	s.state.Running = true
	return nil
}

// OnKline implements Strategy interface
func (s *BaseStrategy) OnKline(ctx context.Context, k *kline.Kline) error {
	return nil
}

// OnOrderBook implements Strategy interface
func (s *BaseStrategy) OnOrderBook(ctx context.Context, ob *orderbook.OrderBook) error {
	return nil
}

// OnTrade implements Strategy interface
func (s *BaseStrategy) OnTrade(ctx context.Context, t *trade.Trade) error {
	return nil
}

// OnPositionUpdate implements Strategy interface
func (s *BaseStrategy) OnPositionUpdate(ctx context.Context, pos *exchange.Position) error {
	return nil
}

// OnOrderUpdate implements Strategy interface
func (s *BaseStrategy) OnOrderUpdate(ctx context.Context, order *exchange.Order) error {
	return nil
}

// OnTimer implements Strategy interface
func (s *BaseStrategy) OnTimer(ctx context.Context, t time.Time) error {
	return nil
}

// GetState implements Strategy interface
func (s *BaseStrategy) GetState() *StrategyState {
	return s.state
}

// Stop implements Strategy interface
func (s *BaseStrategy) Stop(ctx context.Context) error {
	s.state.Running = false
	return nil
}

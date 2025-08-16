package backtest

import (
	"context"
	"fmt"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/strategy/sdk"
)

// Engine represents the backtesting engine
type Engine struct {
	data     *HistoricalData
	strategy sdk.Strategy
	eventMgr *EventManager
	orderMgr *OrderManager
	posMgr   *PositionManager
	statsMgr *StatsManager
	config   *Config
}

// Config represents engine configuration
type Config struct {
	InitialCapital float64
	MarginMode     exchange.MarginType
	Leverage       float64
	SlippageModel  SlippageModel
	FeeModel       FeeModel
	LatencyModel   LatencyModel
	// TODO: Add missing fields for data feed
	Symbols   []string
	StartTime time.Time
	EndTime   time.Time
	DataTypes []string
	Capital   float64
}

// NewEngine creates a new backtesting engine
func NewEngine(data *HistoricalData, strategy sdk.Strategy, config *Config) *Engine {
	eventMgr := NewEventManager()
	orderMgr := NewOrderManager(config.SlippageModel, config.FeeModel, config.LatencyModel)
	posMgr := NewPositionManager(config.InitialCapital, config.MarginMode, config.Leverage)
	statsMgr := NewStatsManager()

	return &Engine{
		data:     data,
		strategy: strategy,
		eventMgr: eventMgr,
		orderMgr: orderMgr,
		posMgr:   posMgr,
		statsMgr: statsMgr,
		config:   config,
	}
}

// Run runs the backtest
func (e *Engine) Run(ctx context.Context) (*Result, error) {
	if err := e.data.Validate(); err != nil {
		return nil, fmt.Errorf("invalid data: %w", err)
	}

	// 初始化策略
	if err := e.strategy.Initialize(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to initialize strategy: %w", err)
	}

	// 运行回测循环
	currentTime := e.data.Start
	for currentTime.Before(e.data.End) {
		// 处理事件队列
		e.eventMgr.ProcessEvents(currentTime)

		// 获取当前市场数据
		kline := e.data.GetKlineAt(currentTime)
		if kline != nil {
			if err := e.strategy.OnTick(ctx, kline); err != nil {
				return nil, fmt.Errorf("strategy OnTick error: %w", err)
			}
		}

		orderbook := e.data.GetOrderbookAt(currentTime)
		if orderbook != nil {
			// 撮合订单
			e.orderMgr.Match(orderbook)
		}

		trades := e.data.GetTradesAt(currentTime)
		for _, t := range trades {
			if err := e.strategy.OnTick(ctx, t); err != nil {
				return nil, fmt.Errorf("strategy OnTick error: %w", err)
			}
		}

		// 处理资金费率
		fundingRate := e.data.GetFundingRateAt(currentTime)
		if fundingRate != nil {
			e.posMgr.ApplyFundingFee(fundingRate)
		}

		// 更新统计数据
		e.statsMgr.Update(currentTime, e.posMgr.GetEquity())

		// 推进时间
		currentTime = currentTime.Add(time.Minute)
	}

	// 获取回测结果
	return e.statsMgr.GetResult(), nil
}

// Result represents backtest results
type Result struct {
	Returns          []float64
	Equity           []float64
	Drawdowns        []float64
	Trades           []*exchange.Trade
	PerformanceStats *PerformanceStats
}

// PerformanceStats represents performance statistics
type PerformanceStats struct {
	Returns        []float64
	TotalReturn    float64
	AnnualReturn   float64
	SharpeRatio    float64
	MaxDrawdown    float64
	WinRate        float64
	ProfitFactor   float64
	TradeCount     int
	AvgTradeReturn float64
	AvgHoldingTime time.Duration
}

// SlippageModel defines the interface for slippage models
type SlippageModel interface {
	CalculateSlippage(price, quantity float64, side exchange.OrderSide) float64
}

// FeeModel defines the interface for fee models
type FeeModel interface {
	CalculateFee(price, quantity float64) float64
}

// LatencyModel defines the interface for latency models
type LatencyModel interface {
	GetLatency() time.Duration
}

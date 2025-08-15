package backtest

import (
	"context"
	"fmt"
	"math"
	"time"

	"qcat/internal/market"
	"qcat/internal/strategy"
	"qcat/internal/strategy/paper"
)

// Engine manages backtest execution
type Engine struct {
	config    *Config
	strategy  strategy.Strategy
	exchange  *paper.Exchange
	dataFeed  DataFeed
	result    *Result
	startTime time.Time
	endTime   time.Time
}

// NewEngine creates a new backtest engine
func NewEngine(config *Config, strategy strategy.Strategy, dataFeed DataFeed) *Engine {
	return &Engine{
		config:    config,
		strategy:  strategy,
		exchange:  paper.NewExchange(nil, map[string]float64{"USDT": config.Capital}),
		dataFeed:  dataFeed,
		startTime: config.StartTime,
		endTime:   config.EndTime,
		result: &Result{
			StartTime:    config.StartTime,
			EndTime:      config.EndTime,
			InitialValue: config.Capital,
			Trades:       make([]*Trade, 0),
			Positions:    make([]*Position, 0),
			EquityCurve:  make([]EquityPoint, 0),
		},
	}
}

// Run runs the backtest
func (e *Engine) Run(ctx context.Context) error {
	// Initialize strategy
	if err := e.strategy.Initialize(ctx, nil); err != nil {
		return fmt.Errorf("failed to initialize strategy: %w", err)
	}

	// Set up strategy context
	strategyCtx := &strategy.Context{
		Mode:      strategy.ModeBacktest,
		Strategy:  "backtest",
		Symbol:    e.config.Symbols[0],
		Exchange:  e.exchange,
		StartTime: e.startTime,
		EndTime:   e.endTime,
	}

	// Set strategy context
	if bs, ok := e.strategy.(interface{ SetContext(*strategy.Context) }); ok {
		bs.SetContext(strategyCtx)
	}

	// Start strategy
	if err := e.strategy.Start(ctx); err != nil {
		return fmt.Errorf("failed to start strategy: %w", err)
	}

	// Process market data
	for e.dataFeed.HasNext() {
		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Get next data point
		data, err := e.dataFeed.Next()
		if err != nil {
			return fmt.Errorf("failed to get next data point: %w", err)
		}

		// Process data
		if err := e.processData(ctx, data); err != nil {
			return fmt.Errorf("failed to process data: %w", err)
		}

		// Update metrics
		if err := e.updateMetrics(data.Timestamp); err != nil {
			return fmt.Errorf("failed to update metrics: %w", err)
		}
	}

	// Stop strategy
	if err := e.strategy.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop strategy: %w", err)
	}

	// Calculate final metrics
	if err := e.calculateMetrics(); err != nil {
		return fmt.Errorf("failed to calculate metrics: %w", err)
	}

	return nil
}

// GetResult returns the backtest result
func (e *Engine) GetResult() *Result {
	return e.result
}

// processData processes a market data point
func (e *Engine) processData(ctx context.Context, data *MarketData) error {
	// Update order book
	if data.Type == "kline" {
		kline := data.Data.(*market.Kline)
		e.exchange.UpdateOrderBook(data.Symbol, []paper.Level{
			{Price: kline.Low, Quantity: 1000000},
		}, []paper.Level{
			{Price: kline.High, Quantity: 1000000},
		})
	}

	// Process data in strategy
	if err := e.strategy.OnTick(ctx, data); err != nil {
		return fmt.Errorf("failed to process tick: %w", err)
	}

	return nil
}

// updateMetrics updates backtest metrics
func (e *Engine) updateMetrics(timestamp time.Time) error {
	// Get account balance
	balances, err := e.exchange.GetAccountBalance(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get account balance: %w", err)
	}

	// Calculate equity
	var equity float64
	for _, balance := range balances {
		equity += balance.Total
	}

	// Get positions
	positions, err := e.exchange.GetPositions(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	// Add unrealized PnL
	for _, pos := range positions {
		equity += pos.UnrealizedPnL
	}

	// Update equity curve
	e.result.EquityCurve = append(e.result.EquityCurve, EquityPoint{
		Timestamp: timestamp,
		Equity:    equity,
		PnL:       equity - e.result.InitialValue,
	})

	// Update drawdown curve
	if len(e.result.DrawdownCurve) == 0 {
		e.result.DrawdownCurve = append(e.result.DrawdownCurve, DrawdownPoint{
			Timestamp: timestamp,
			Drawdown:  0,
			Duration:  0,
		})
	} else {
		maxEquity := e.result.InitialValue
		for _, point := range e.result.EquityCurve {
			if point.Equity > maxEquity {
				maxEquity = point.Equity
			}
		}

		drawdown := (maxEquity - equity) / maxEquity * 100
		duration := timestamp.Sub(e.result.DrawdownCurve[0].Timestamp)

		e.result.DrawdownCurve = append(e.result.DrawdownCurve, DrawdownPoint{
			Timestamp: timestamp,
			Drawdown:  drawdown,
			Duration:  duration,
		})
	}

	return nil
}

// calculateMetrics calculates final backtest metrics
func (e *Engine) calculateMetrics() error {
	if len(e.result.EquityCurve) == 0 {
		return fmt.Errorf("no equity data available")
	}

	// Calculate final value and PnL
	e.result.FinalValue = e.result.EquityCurve[len(e.result.EquityCurve)-1].Equity
	e.result.PnL = e.result.FinalValue - e.result.InitialValue
	e.result.PnLPercent = e.result.PnL / e.result.InitialValue * 100

	// Calculate max drawdown
	maxDrawdown := 0.0
	for _, point := range e.result.DrawdownCurve {
		if point.Drawdown > maxDrawdown {
			maxDrawdown = point.Drawdown
		}
	}
	e.result.MaxDrawdown = maxDrawdown

	// Calculate Sharpe ratio
	var returns []float64
	for i := 1; i < len(e.result.EquityCurve); i++ {
		prev := e.result.EquityCurve[i-1].Equity
		curr := e.result.EquityCurve[i].Equity
		returns = append(returns, (curr-prev)/prev)
	}

	if len(returns) > 0 {
		// Calculate mean return
		meanReturn := 0.0
		for _, r := range returns {
			meanReturn += r
		}
		meanReturn /= float64(len(returns))

		// Calculate standard deviation
		variance := 0.0
		for _, r := range returns {
			diff := r - meanReturn
			variance += diff * diff
		}
		variance /= float64(len(returns))
		stdDev := math.Sqrt(variance)

		// Calculate annualized Sharpe ratio (assuming daily returns)
		if stdDev > 0 {
			e.result.SharpeRatio = (meanReturn / stdDev) * math.Sqrt(252)
		}
	}

	// Calculate win rate
	wins := 0
	for _, trade := range e.result.Trades {
		if trade.PnL > 0 {
			wins++
		}
	}
	if len(e.result.Trades) > 0 {
		e.result.WinRate = float64(wins) / float64(len(e.result.Trades)) * 100
	}

	return nil
}

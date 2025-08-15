package backtest

import (
	"math"
	"time"

	"qcat/internal/exchange"
)

// StatsManager manages performance statistics
type StatsManager struct {
	returns    []float64
	equity     []float64
	drawdowns  []float64
	trades     []*exchange.Trade
	timestamps []time.Time
}

// NewStatsManager creates a new stats manager
func NewStatsManager() *StatsManager {
	return &StatsManager{
		returns:    make([]float64, 0),
		equity:     make([]float64, 0),
		drawdowns:  make([]float64, 0),
		trades:     make([]*exchange.Trade, 0),
		timestamps: make([]time.Time, 0),
	}
}

// Update updates statistics with new data
func (m *StatsManager) Update(timestamp time.Time, equity float64) {
	m.timestamps = append(m.timestamps, timestamp)
	m.equity = append(m.equity, equity)

	// 计算收益率
	if len(m.equity) > 1 {
		ret := (equity - m.equity[len(m.equity)-2]) / m.equity[len(m.equity)-2]
		m.returns = append(m.returns, ret)
	}

	// 计算回撤
	highWaterMark := equity
	for i := len(m.equity) - 1; i >= 0; i-- {
		if m.equity[i] > highWaterMark {
			highWaterMark = m.equity[i]
		}
	}
	drawdown := (highWaterMark - equity) / highWaterMark
	m.drawdowns = append(m.drawdowns, drawdown)
}

// AddTrade adds a trade to the statistics
func (m *StatsManager) AddTrade(trade *exchange.Trade) {
	m.trades = append(m.trades, trade)
}

// GetResult returns the final performance statistics
func (m *StatsManager) GetResult() *Result {
	stats := &PerformanceStats{
		TotalReturn:    m.calculateTotalReturn(),
		AnnualReturn:   m.calculateAnnualReturn(),
		SharpeRatio:    m.calculateSharpeRatio(),
		MaxDrawdown:    m.calculateMaxDrawdown(),
		WinRate:        m.calculateWinRate(),
		ProfitFactor:   m.calculateProfitFactor(),
		TradeCount:     len(m.trades),
		AvgTradeReturn: m.calculateAvgTradeReturn(),
		AvgHoldingTime: m.calculateAvgHoldingTime(),
	}

	return &Result{
		Returns:          m.returns,
		Equity:           m.equity,
		Drawdowns:        m.drawdowns,
		Trades:           m.trades,
		PerformanceStats: stats,
	}
}

// Helper functions for calculating statistics

func (m *StatsManager) calculateTotalReturn() float64 {
	if len(m.equity) < 2 {
		return 0
	}
	return (m.equity[len(m.equity)-1] - m.equity[0]) / m.equity[0]
}

func (m *StatsManager) calculateAnnualReturn() float64 {
	if len(m.timestamps) < 2 {
		return 0
	}

	years := m.timestamps[len(m.timestamps)-1].Sub(m.timestamps[0]).Hours() / (24 * 365)
	if years == 0 {
		return 0
	}

	totalReturn := m.calculateTotalReturn()
	return math.Pow(1+totalReturn, 1/years) - 1
}

func (m *StatsManager) calculateSharpeRatio() float64 {
	if len(m.returns) < 2 {
		return 0
	}

	// 计算年化收益率和标准差
	mean := 0.0
	for _, r := range m.returns {
		mean += r
	}
	mean /= float64(len(m.returns))

	variance := 0.0
	for _, r := range m.returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(m.returns) - 1)
	stdDev := math.Sqrt(variance)

	if stdDev == 0 {
		return 0
	}

	// 假设无风险利率为0
	return mean / stdDev * math.Sqrt(252) // 年化
}

func (m *StatsManager) calculateMaxDrawdown() float64 {
	maxDrawdown := 0.0
	for _, dd := range m.drawdowns {
		if dd > maxDrawdown {
			maxDrawdown = dd
		}
	}
	return maxDrawdown
}

func (m *StatsManager) calculateWinRate() float64 {
	if len(m.trades) == 0 {
		return 0
	}

	wins := 0
	for _, trade := range m.trades {
		if trade.Price > trade.Fee {
			wins++
		}
	}

	return float64(wins) / float64(len(m.trades))
}

func (m *StatsManager) calculateProfitFactor() float64 {
	var grossProfit, grossLoss float64

	for _, trade := range m.trades {
		pnl := trade.Price - trade.Fee
		if pnl > 0 {
			grossProfit += pnl
		} else {
			grossLoss -= pnl
		}
	}

	if grossLoss == 0 {
		return 0
	}
	return grossProfit / grossLoss
}

func (m *StatsManager) calculateAvgTradeReturn() float64 {
	if len(m.trades) == 0 {
		return 0
	}

	totalReturn := 0.0
	for _, trade := range m.trades {
		totalReturn += (trade.Price - trade.Fee) / trade.Price
	}

	return totalReturn / float64(len(m.trades))
}

func (m *StatsManager) calculateAvgHoldingTime() time.Duration {
	if len(m.trades) == 0 {
		return 0
	}

	var totalDuration time.Duration
	for i := 1; i < len(m.trades); i++ {
		duration := m.trades[i].Time.Sub(m.trades[i-1].Time)
		totalDuration += duration
	}

	return totalDuration / time.Duration(len(m.trades))
}

// CalculatePerformanceStats calculates performance statistics from returns
func CalculatePerformanceStats(returns []float64) *PerformanceStats {
	if len(returns) == 0 {
		return &PerformanceStats{}
	}

	// Calculate basic statistics
	totalReturn := 0.0
	for _, ret := range returns {
		totalReturn += ret
	}

	// Calculate volatility
	mean := totalReturn / float64(len(returns))
	variance := 0.0
	for _, ret := range returns {
		diff := ret - mean
		variance += diff * diff
	}
	variance /= float64(len(returns))
	volatility := math.Sqrt(variance)

	// Calculate Sharpe ratio (assuming risk-free rate of 0)
	sharpeRatio := 0.0
	if volatility > 0 {
		sharpeRatio = mean / volatility
	}

	// Calculate max drawdown
	maxDrawdown := 0.0
	peak := 1.0
	equity := 1.0
	for _, ret := range returns {
		equity *= (1 + ret)
		if equity > peak {
			peak = equity
		}
		drawdown := (peak - equity) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return &PerformanceStats{
		TotalReturn:  totalReturn,
		SharpeRatio:  sharpeRatio,
		MaxDrawdown:  maxDrawdown,
	}
}

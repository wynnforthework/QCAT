package portfolio

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/exchange/position"
)

// Manager manages portfolio allocation
type Manager struct {
	exchange    exchange.Exchange
	posManager  *position.Manager
	allocations map[string]*Allocation
	metrics     map[string]*Metrics
	mu          sync.RWMutex
}

// Allocation represents asset allocation
type Allocation struct {
	Symbol        string
	TargetWeight  float64 // 目标权重
	CurrentWeight float64 // 当前权重
	MaxWeight     float64 // 最大权重限制
	RiskBudget    float64 // 风险预算
	TargetVol     float64 // 目标波动率
	RealizedVol   float64 // 实现波动率
	UpdatedAt     time.Time
}

// Metrics represents portfolio metrics
type Metrics struct {
	Symbol        string
	Returns       []float64 // 收益率序列
	Volatility    float64   // 波动率
	SharpeRatio   float64   // 夏普比率
	MaxDrawdown   float64   // 最大回撤
	WinRate       float64   // 胜率
	ProfitFactor  float64   // 盈亏比
	KellyFraction float64   // 凯利比率
	UpdatedAt     time.Time
}

// NewManager creates a new portfolio manager
func NewManager(ex exchange.Exchange, pm *position.Manager) *Manager {
	return &Manager{
		exchange:    ex,
		posManager:  pm,
		allocations: make(map[string]*Allocation),
		metrics:     make(map[string]*Metrics),
	}
}

// SetAllocation sets target allocation for a symbol
func (m *Manager) SetAllocation(symbol string, alloc *Allocation) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if alloc.TargetWeight < 0 || alloc.TargetWeight > 1 {
		return fmt.Errorf("invalid target weight: must be between 0 and 1")
	}
	if alloc.MaxWeight < alloc.TargetWeight {
		return fmt.Errorf("max weight must be greater than target weight")
	}
	if alloc.TargetVol <= 0 {
		return fmt.Errorf("invalid target volatility")
	}

	alloc.UpdatedAt = time.Now()
	m.allocations[symbol] = alloc
	return nil
}

// UpdateMetrics updates portfolio metrics
func (m *Manager) UpdateMetrics(symbol string, returns []float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics := &Metrics{
		Symbol:    symbol,
		Returns:   returns,
		UpdatedAt: time.Now(),
	}

	// Calculate volatility
	metrics.Volatility = calculateVolatility(returns)

	// Calculate Sharpe ratio
	metrics.SharpeRatio = calculateSharpeRatio(returns, 0) // 使用0作为无风险利率

	// Calculate max drawdown
	metrics.MaxDrawdown = calculateMaxDrawdown(returns)

	// Calculate win rate and profit factor
	metrics.WinRate, metrics.ProfitFactor = calculateTradeMetrics(returns)

	// Calculate Kelly fraction
	metrics.KellyFraction = calculateKellyFraction(metrics.WinRate, metrics.ProfitFactor)

	m.metrics[symbol] = metrics
	return nil
}

// CalculateTargetPositions calculates target positions based on risk budgeting
func (m *Manager) CalculateTargetPositions(ctx context.Context, totalEquity float64) (map[string]float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	positions := make(map[string]float64)

	// Calculate position sizes using risk budgeting approach
	for symbol, alloc := range m.allocations {
		metrics, exists := m.metrics[symbol]
		if !exists {
			continue
		}

		// Calculate weight using risk budget formula
		// w_i = min(w_max, risk_budget_i * target_vol / realized_vol_i)
		var weight float64
		if metrics.Volatility > 0 {
			weight = alloc.RiskBudget * alloc.TargetVol / metrics.Volatility
			if weight > alloc.MaxWeight {
				weight = alloc.MaxWeight
			}
		}

		// Calculate position size
		positions[symbol] = totalEquity * weight
	}

	return positions, nil
}

// Rebalance rebalances portfolio positions
func (m *Manager) Rebalance(ctx context.Context, threshold float64) error {
	// Get current positions
	currentPositions, err := m.posManager.GetAllPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	// Get account equity
	balances, err := m.exchange.GetAccountBalance(ctx)
	if err != nil {
		return fmt.Errorf("failed to get account balance: %w", err)
	}

	// Calculate total equity
	totalEquity := 0.0
	for _, balance := range balances {
		totalEquity += balance.Total
	}

	// Calculate target positions
	targetPositions, err := m.CalculateTargetPositions(ctx, totalEquity)
	if err != nil {
		return fmt.Errorf("failed to calculate target positions: %w", err)
	}

	// Rebalance positions that exceed threshold
	for symbol, targetSize := range targetPositions {
		var currentSize float64
		for _, pos := range currentPositions {
			if pos.Symbol == symbol {
				currentSize = pos.Notional // 使用 Notional 字段替代 Value
				break
			}
		}

		// Calculate deviation
		deviation := math.Abs(currentSize-targetSize) / targetSize
		if deviation > threshold {
			// Create rebalancing order
			side := exchange.OrderSideBuy
			if currentSize > targetSize {
				side = exchange.OrderSideSell
			}

			quantity := math.Abs(targetSize - currentSize)
			req := &exchange.OrderRequest{
				Symbol:   symbol,
				Side:     string(side), // 显式转换为 string
				Type:     string(exchange.OrderTypeMarket), // 显式转换为 string
				Quantity: quantity,
			}

			if _, err := m.exchange.PlaceOrder(ctx, req); err != nil {
				return fmt.Errorf("failed to place rebalancing order: %w", err)
			}
		}
	}

	return nil
}

// GetAllocation returns allocation for a symbol
func (m *Manager) GetAllocation(symbol string) (*Allocation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	alloc, exists := m.allocations[symbol]
	if !exists {
		return nil, fmt.Errorf("allocation not found for symbol: %s", symbol)
	}
	return alloc, nil
}

// GetMetrics returns metrics for a symbol
func (m *Manager) GetMetrics(symbol string) (*Metrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics, exists := m.metrics[symbol]
	if !exists {
		return nil, fmt.Errorf("metrics not found for symbol: %s", symbol)
	}
	return metrics, nil
}

// Helper functions for calculating metrics

func calculateVolatility(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(returns) - 1)

	return math.Sqrt(variance)
}

func calculateSharpeRatio(returns []float64, riskFreeRate float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	vol := calculateVolatility(returns)
	if vol == 0 {
		return 0
	}

	return (mean - riskFreeRate) / vol
}

func calculateMaxDrawdown(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	// Calculate cumulative returns
	cumReturns := make([]float64, len(returns))
	cumReturns[0] = 1 + returns[0]
	for i := 1; i < len(returns); i++ {
		cumReturns[i] = cumReturns[i-1] * (1 + returns[i])
	}

	// Calculate running maximum
	maxSoFar := cumReturns[0]
	maxDrawdown := 0.0
	for _, cr := range cumReturns {
		if cr > maxSoFar {
			maxSoFar = cr
		}
		drawdown := (maxSoFar - cr) / maxSoFar
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

func calculateTradeMetrics(returns []float64) (winRate, profitFactor float64) {
	if len(returns) == 0 {
		return 0, 0
	}

	wins := 0
	totalProfit := 0.0
	totalLoss := 0.0

	for _, r := range returns {
		if r > 0 {
			wins++
			totalProfit += r
		} else {
			totalLoss -= r
		}
	}

	winRate = float64(wins) / float64(len(returns))
	if totalLoss == 0 {
		profitFactor = 1
	} else {
		profitFactor = totalProfit / totalLoss
	}

	return winRate, profitFactor
}

func calculateKellyFraction(winRate, profitFactor float64) float64 {
	if winRate == 0 || profitFactor <= 0 {
		return 0
	}

	// Kelly公式: f = p - (1-p)/R
	// 其中p是胜率，R是盈亏比
	return winRate - (1-winRate)/profitFactor
}

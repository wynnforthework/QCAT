package risk

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	exch "qcat/internal/exchange"
	"qcat/internal/exchange/position"
)

// Rebalancer handles portfolio rebalancing
type Rebalancer struct {
	exchange      exch.Exchange
	posManager    *position.Manager
	targetRatios  map[string]float64 // 目标资产配比
	tolerance     float64            // 允许的偏差范围
	minInterval   time.Duration      // 最小再平衡间隔
	lastRebalance time.Time
	mu            sync.RWMutex
}

// NewRebalancer creates a new rebalancer
func NewRebalancer(ex exch.Exchange, pm *position.Manager) *Rebalancer {
	return &Rebalancer{
		exchange:     ex,
		posManager:   pm,
		targetRatios: make(map[string]float64),
		tolerance:    0.05,           // 默认5%偏差容忍度
		minInterval:  time.Hour * 24, // 默认每24小时最多一次再平衡
	}
}

// SetTargetRatio sets target ratio for an asset
func (r *Rebalancer) SetTargetRatio(asset string, ratio float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.targetRatios[asset] = ratio
}

// SetTolerance sets the rebalancing tolerance
func (r *Rebalancer) SetTolerance(tolerance float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tolerance = tolerance
}

// CheckAndRebalance checks if rebalancing is needed and performs it
func (r *Rebalancer) CheckAndRebalance(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查再平衡间隔
	if time.Since(r.lastRebalance) < r.minInterval {
		return fmt.Errorf("cannot rebalance: minimum interval not reached")
	}

	// 获取当前持仓
	positions, err := r.posManager.GetAllPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	// 计算当前总价值和资产比例
	totalValue := 0.0
	currentRatios := make(map[string]float64)
	for _, pos := range positions {
		totalValue += pos.Notional // 使用 Notional 字段替代 Value
	}
	for _, pos := range positions {
		currentRatios[pos.Symbol] = pos.Notional / totalValue // 使用 Notional 字段替代 Value
	}

	// 检查是否需要再平衡
	needRebalance := false
	for asset, targetRatio := range r.targetRatios {
		currentRatio := currentRatios[asset]
		if abs(currentRatio-targetRatio) > r.tolerance {
			needRebalance = true
			break
		}
	}

	if !needRebalance {
		return nil
	}

	// 执行再平衡
	for asset, targetRatio := range r.targetRatios {
		currentRatio := currentRatios[asset]
		if abs(currentRatio-targetRatio) <= r.tolerance {
			continue
		}

		// 找到对应的持仓
		var pos *exch.Position
		for _, p := range positions {
			if p.Symbol == asset {
				pos = p
				break
			}
		}

		if pos == nil {
			continue
		}

		// 计算需要调整的数量
		targetValue := totalValue * targetRatio
		valueDiff := targetValue - pos.Notional // 使用 Notional 字段替代 Value
		sizeDiff := valueDiff / pos.MarkPrice   // 使用 MarkPrice 字段替代 Price

		// 创建市价单调整仓位
		order := &exch.OrderRequest{
			Symbol:   pos.Symbol,
			Type:     string(exch.OrderTypeMarket), // 显式转换为 string
			Quantity: abs(sizeDiff),
		}

		if sizeDiff > 0 {
			order.Side = string(exch.OrderSideBuy) // 显式转换为 string
		} else {
			order.Side = string(exch.OrderSideSell) // 显式转换为 string
		}

		if _, err := r.exchange.PlaceOrder(ctx, order); err != nil {
			log.Printf("Failed to rebalance %s: %v", pos.Symbol, err)
			continue
		}
	}

	r.lastRebalance = time.Now()
	return nil
}

// abs returns the absolute value of x
// TODO: Use math.Abs instead
// func abs(x float64) float64 {
// 	if x < 0 {
// 		return -x
// 	}
// 	return x
// }

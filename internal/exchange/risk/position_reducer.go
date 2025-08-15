package risk

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/exchange/position"
)

// ReduceStrategy defines how to reduce positions
type ReduceStrategy int

const (
	ReduceByPnL  ReduceStrategy = iota // 按照盈亏排序
	ReduceBySize                       // 按照仓位大小排序
	ReduceByRisk                       // 按照风险敞口排序
)

// PositionReducer handles automatic position reduction
type PositionReducer struct {
	exchange      exchange.Exchange
	posManager    *position.Manager
	marginMonitor *MarginMonitor
	strategy      ReduceStrategy
	reduceRatio   float64       // 每次减仓比例
	minInterval   time.Duration // 最小减仓间隔
	lastReduce    time.Time
	mu            sync.RWMutex
}

// NewPositionReducer creates a new position reducer
func NewPositionReducer(ex exchange.Exchange, pm *position.Manager, mm *MarginMonitor) *PositionReducer {
	r := &PositionReducer{
		exchange:      ex,
		posManager:    pm,
		marginMonitor: mm,
		strategy:      ReduceByRisk,
		reduceRatio:   0.2, // 默认每次减仓20%
		minInterval:   time.Minute * 5,
	}

	// 订阅保证金告警
	go r.handleMarginAlerts()

	return r
}

// SetReduceStrategy sets the position reduction strategy
func (r *PositionReducer) SetReduceStrategy(strategy ReduceStrategy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.strategy = strategy
}

// SetReduceRatio sets the position reduction ratio
func (r *PositionReducer) SetReduceRatio(ratio float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reduceRatio = ratio
}

// handleMarginAlerts handles margin alerts and triggers position reduction
func (r *PositionReducer) handleMarginAlerts() {
	alertCh := r.marginMonitor.GetAlertChannel()
	for alert := range alertCh {
		if alert.Level >= exchange.MarginLevelDanger {
			if err := r.ReducePositions(context.Background()); err != nil {
				log.Printf("Failed to reduce positions: %v", err)
			}
		}
	}
}

// ReducePositions reduces positions based on current strategy
func (r *PositionReducer) ReducePositions(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查减仓间隔
	if time.Since(r.lastReduce) < r.minInterval {
		return fmt.Errorf("cannot reduce positions: minimum interval not reached")
	}

	// 获取所有持仓
	positions, err := r.posManager.GetAllPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	// 按策略排序持仓
	sortedPositions := r.sortPositions(positions)

	// 计算需要减仓的总价值
	totalValue := 0.0
	for _, pos := range positions {
		totalValue += pos.Value
	}
	reduceValue := totalValue * r.reduceRatio

	// 执行减仓
	reducedValue := 0.0
	for _, pos := range sortedPositions {
		if reducedValue >= reduceValue {
			break
		}

		// 计算这个仓位需要减少的数量
		reduceSize := pos.Size * r.reduceRatio
		if reduceSize < pos.MinSize {
			reduceSize = pos.MinSize
		}

		// 创建市价单平仓
		order := &exchange.Order{
			Symbol:   pos.Symbol,
			Side:     pos.Side.Reverse(), // 反向开仓
			Type:     exchange.OrderTypeMarket,
			Quantity: reduceSize,
		}

		if _, err := r.exchange.CreateOrder(ctx, order); err != nil {
			log.Printf("Failed to reduce position %s: %v", pos.Symbol, err)
			continue
		}

		reducedValue += pos.Value * r.reduceRatio
	}

	r.lastReduce = time.Now()
	return nil
}

// sortPositions sorts positions based on current strategy
func (r *PositionReducer) sortPositions(positions []*position.Position) []*position.Position {
	switch r.strategy {
	case ReduceByPnL:
		// 按未实现盈亏排序，亏损优先减仓
		sort.Slice(positions, func(i, j int) bool {
			return positions[i].UnrealizedPnL < positions[j].UnrealizedPnL
		})
	case ReduceBySize:
		// 按仓位价值排序，大仓位优先减仓
		sort.Slice(positions, func(i, j int) bool {
			return positions[i].Value > positions[j].Value
		})
	case ReduceByRisk:
		// 按风险敞口排序，高风险优先减仓
		sort.Slice(positions, func(i, j int) bool {
			return positions[i].RiskExposure > positions[j].RiskExposure
		})
	}
	return positions
}

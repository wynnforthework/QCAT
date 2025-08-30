package strategy

import (
	"context"
	"fmt"
	"time"

	"qcat/internal/cache"
	"qcat/internal/database"
	"qcat/internal/monitor"
)

// SimpleUnifiedStrategyService 简化的统一策略服务
// 不依赖有问题的workflow包，专注于核心功能
type SimpleUnifiedStrategyService struct {
	db      *database.DB
	cache   cache.Cacher
	metrics *monitor.MetricsCollector
}

// NewSimpleUnifiedStrategyService 创建简化的统一策略服务
func NewSimpleUnifiedStrategyService(
	db *database.DB,
	cache cache.Cacher,
	metrics *monitor.MetricsCollector,
) *SimpleUnifiedStrategyService {
	return &SimpleUnifiedStrategyService{
		db:      db,
		cache:   cache,
		metrics: metrics,
	}
}

// ListStrategies 获取策略列表
func (s *SimpleUnifiedStrategyService) ListStrategies(ctx context.Context, options StrategyListOptions) (*StrategyListResponse, error) {
	// 设置默认值
	if options.PageSize == 0 {
		options.PageSize = 20
	}
	if options.Page == 0 {
		options.Page = 1
	}

	// 返回空策略列表，避免使用 mock 数据
	strategies := []BasicStrategy{}

	// 应用过滤
	filtered := s.applyFilters(strategies, options)

	// 分页
	start := (options.Page - 1) * options.PageSize
	end := start + options.PageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	if start > len(filtered) {
		start = len(filtered)
	}

	pagedStrategies := filtered[start:end]

	// 转换为统一格式
	unifiedStrategies := make([]UnifiedStrategy, len(pagedStrategies))
	for i, strategy := range pagedStrategies {
		unifiedStrategies[i] = s.convertToUnified(strategy)
	}

	// 生成摘要
	summary := s.generateSummary(unifiedStrategies)

	return &StrategyListResponse{
		Strategies: unifiedStrategies,
		Total:      len(filtered),
		Page:       options.Page,
		PageSize:   options.PageSize,
		Summary:    summary,
	}, nil
}

// GetStrategy 获取单个策略详情
func (s *SimpleUnifiedStrategyService) GetStrategy(ctx context.Context, strategyID string) (*UnifiedStrategy, error) {
	// 不使用 mock 数据，直接返回未找到错误
	return nil, fmt.Errorf("strategy not found: %s (no database connection)", strategyID)
}

// applyFilters 应用过滤条件
func (s *SimpleUnifiedStrategyService) applyFilters(strategies []BasicStrategy, options StrategyListOptions) []BasicStrategy {
	var filtered []BasicStrategy

	for _, strategy := range strategies {
		// 状态过滤
		if len(options.Status) > 0 {
			found := false
			for _, status := range options.Status {
				if strategy.Status == status {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// 类型过滤
		if len(options.Type) > 0 {
			found := false
			for _, typ := range options.Type {
				if strategy.Type == typ {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		filtered = append(filtered, strategy)
	}

	return filtered
}

// convertToUnified 转换为统一格式
func (s *SimpleUnifiedStrategyService) convertToUnified(basic BasicStrategy) UnifiedStrategy {
	return UnifiedStrategy{
		ID:          basic.ID,
		Name:        basic.Name,
		Type:        basic.Type,
		Description: basic.Description,
		Version:     basic.Version,
		CreatedAt:   basic.CreatedAt,
		UpdatedAt:   basic.UpdatedAt,
		Config:      basic.Config,

		Lifecycle: LifecycleInfo{
			Stage:     basic.Stage,
			Status:    basic.Status,
			IsEnabled: basic.Status == "active",
			CanStart:  basic.Status == "inactive",
			CanStop:   basic.Status == "active",
			CanEdit:   basic.Stage != "production",
			CanDelete: basic.Stage == "draft",
		},

		Execution: ExecutionInfo{
			IsRunning:      false,
			LastExecution:  time.Time{},
			NextExecution:  time.Time{},
			ExecutionCount: 0,
			SuccessCount:   0,
			ErrorCount:     0,
			SuccessRate:    0.0,
			AvgLatency:     0.0,
		},
		Performance: PerformanceInfo{
			PNL:          0.0,
			TotalReturn:  0.0,
			SharpeRatio:  0.0,
			MaxDrawdown:  0.0,
			WinRate:      0.0,
			ProfitFactor: 0.0,
			Volatility:   0.0,
			TradeCount:   0,
			AvgTrade:     0.0,
			BestTrade:    0.0,
			WorstTrade:   0.0,
		},
		Pool: PoolInfo{
			PoolStatus: "disabled",
			Priority:   "low",
			ResourceAllocation: ResourceAllocation{
				CPU:    0.0,
				Memory: 0.0,
			},
			PoolMetrics: PoolMetrics{
				QueuePosition:   0,
				ExecutionWeight: 0.0,
				ResourceUsage:   0.0,
				ConflictCount:   0,
			},
			LastSync:   time.Time{},
			SyncStatus: "no_data",
		},
	}
}

// generateSummary 生成摘要信息
func (s *SimpleUnifiedStrategyService) generateSummary(strategies []UnifiedStrategy) StrategySummary {
	summary := StrategySummary{}

	summary.Total.Count = len(strategies)

	var totalReturn, totalSharpe, totalPNL float64
	winningCount := 0

	for _, strategy := range strategies {
		// 统计状态
		switch strategy.Lifecycle.Status {
		case "active":
			summary.Total.Active++
		case "inactive":
			summary.Total.Inactive++
		}

		// 统计阶段
		switch strategy.Lifecycle.Stage {
		case "testing":
			summary.Total.Testing++
		case "production":
			summary.Total.Production++
		}

		// 统计池状态
		switch strategy.Pool.PoolStatus {
		case "enabled":
			summary.Pool.Enabled++
		case "disabled":
			summary.Pool.Disabled++
		case "pending":
			summary.Pool.Pending++
		case "testing":
			summary.Pool.Testing++
		}

		// 统计性能
		totalReturn += strategy.Performance.TotalReturn
		totalSharpe += strategy.Performance.SharpeRatio
		totalPNL += strategy.Performance.PNL

		if strategy.Performance.TotalReturn > 0 {
			winningCount++
		}
	}

	// 计算平均值
	if len(strategies) > 0 {
		summary.Performance.AvgReturn = totalReturn / float64(len(strategies))
		summary.Performance.AvgSharpe = totalSharpe / float64(len(strategies))
	}
	summary.Performance.TotalPNL = totalPNL
	summary.Performance.WinningCount = winningCount

	return summary
}

// 删除了 getMockStrategies 方法，不再使用 mock 数据

// 删除了所有 mock 方法，不再使用 mock 数据

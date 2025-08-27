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

	// 使用模拟数据，避免依赖问题
	strategies := s.getMockStrategies()
	
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
	strategies := s.getMockStrategies()
	
	for _, strategy := range strategies {
		if strategy.ID == strategyID {
			unified := s.convertToUnified(strategy)
			return &unified, nil
		}
	}
	
	return nil, fmt.Errorf("strategy not found: %s", strategyID)
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
		
		Execution: s.getMockExecutionInfo(),
		Performance: s.getMockPerformanceInfo(),
		Pool: s.getMockPoolInfo(),
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

// getMockStrategies 获取模拟策略数据
func (s *SimpleUnifiedStrategyService) getMockStrategies() []BasicStrategy {
	return []BasicStrategy{
		{
			ID:          "strategy_001",
			Name:        "动量策略Alpha",
			Type:        "momentum",
			Description: "基于动量指标的交易策略",
			Version:     "1.2.0",
			Status:      "active",
			Stage:       "production",
			CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-2 * time.Hour),
			Config:      map[string]interface{}{"lookback": 14, "threshold": 0.02},
		},
		{
			ID:          "strategy_002",
			Name:        "均值回归策略Beta",
			Type:        "mean_reversion",
			Description: "基于均值回归的交易策略",
			Version:     "2.1.0",
			Status:      "active",
			Stage:       "production",
			CreatedAt:   time.Now().Add(-45 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-1 * time.Hour),
			Config:      map[string]interface{}{"window": 20, "deviation": 2.0},
		},
		{
			ID:          "strategy_003",
			Name:        "套利策略Gamma",
			Type:        "arbitrage",
			Description: "跨交易所套利策略",
			Version:     "1.0.0",
			Status:      "testing",
			Stage:       "testing",
			CreatedAt:   time.Now().Add(-7 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-30 * time.Minute),
			Config:      map[string]interface{}{"min_spread": 0.001, "max_position": 10000},
		},
	}
}

// getMockExecutionInfo 获取模拟执行信息
func (s *SimpleUnifiedStrategyService) getMockExecutionInfo() ExecutionInfo {
	return ExecutionInfo{
		IsRunning:      true,
		LastExecution:  time.Now().Add(-5 * time.Minute),
		NextExecution:  time.Now().Add(10 * time.Minute),
		ExecutionCount: 1234,
		SuccessCount:   1180,
		ErrorCount:     54,
		SuccessRate:    95.6,
		AvgLatency:     8.5,
	}
}

// getMockPerformanceInfo 获取模拟性能信息
func (s *SimpleUnifiedStrategyService) getMockPerformanceInfo() PerformanceInfo {
	return PerformanceInfo{
		PNL:          15420.50,
		TotalReturn:  0.156,
		SharpeRatio:  2.34,
		MaxDrawdown:  0.08,
		WinRate:      0.67,
		ProfitFactor: 1.85,
		Volatility:   0.12,
		TradeCount:   456,
		AvgTrade:     33.8,
		BestTrade:    245.6,
		WorstTrade:   -89.2,
	}
}

// getMockPoolInfo 获取模拟池信息
func (s *SimpleUnifiedStrategyService) getMockPoolInfo() PoolInfo {
	return PoolInfo{
		PoolStatus: "enabled",
		Priority:   "high",
		ResourceAllocation: ResourceAllocation{
			CPU:    1.2,
			Memory: 2.1,
		},
		PoolMetrics: PoolMetrics{
			QueuePosition:   3,
			ExecutionWeight: 0.85,
			ResourceUsage:   0.65,
			ConflictCount:   0,
		},
		LastSync:   time.Now().Add(-2 * time.Minute),
		SyncStatus: "success",
	}
}
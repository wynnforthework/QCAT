package strategy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"qcat/internal/cache"
	"qcat/internal/database"
	"qcat/internal/monitor"
	"qcat/internal/strategy/workflow"
)

// UnifiedStrategyService 统一策略服务
// 整合策略管理、策略池、执行监控等功能
type UnifiedStrategyService struct {
	db             *database.DB
	cache          cache.Cacher
	metrics        *monitor.MetricsCollector
	workflowSystem *workflow.MultiStrategyWorkflowSystem
	strategyPool   *workflow.TradingStrategyPool
}

// NewUnifiedStrategyService 创建统一策略服务
func NewUnifiedStrategyService(
	db *database.DB,
	cache cache.Cacher,
	metrics *monitor.MetricsCollector,
	workflowSystem *workflow.MultiStrategyWorkflowSystem,
) *UnifiedStrategyService {
	var strategyPool *workflow.TradingStrategyPool
	if workflowSystem != nil {
		strategyPool = workflowSystem.GetStrategyPool()
	}

	return &UnifiedStrategyService{
		db:             db,
		cache:          cache,
		metrics:        metrics,
		workflowSystem: workflowSystem,
		strategyPool:   strategyPool,
	}
}

// UnifiedStrategy 统一策略模型
type UnifiedStrategy struct {
	// 基本信息
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`

	// 生命周期信息
	Lifecycle LifecycleInfo `json:"lifecycle"`

	// 执行信息
	Execution ExecutionInfo `json:"execution"`

	// 性能信息
	Performance PerformanceInfo `json:"performance"`

	// 策略池信息
	Pool PoolInfo `json:"pool"`

	// 配置信息
	Config map[string]interface{} `json:"config"`
}

// LifecycleInfo 生命周期信息
type LifecycleInfo struct {
	Stage     string `json:"stage"`  // draft, testing, production, deprecated
	Status    string `json:"status"` // active, inactive, paused, error
	IsEnabled bool   `json:"isEnabled"`
	CanStart  bool   `json:"canStart"`
	CanStop   bool   `json:"canStop"`
	CanEdit   bool   `json:"canEdit"`
	CanDelete bool   `json:"canDelete"`
}

// ExecutionInfo 执行信息
type ExecutionInfo struct {
	IsRunning      bool      `json:"isRunning"`
	LastExecution  time.Time `json:"lastExecution"`
	NextExecution  time.Time `json:"nextExecution"`
	ExecutionCount int       `json:"executionCount"`
	SuccessCount   int       `json:"successCount"`
	ErrorCount     int       `json:"errorCount"`
	SuccessRate    float64   `json:"successRate"`
	AvgLatency     float64   `json:"avgLatency"`
	LastError      string    `json:"lastError,omitempty"`
}

// PerformanceInfo 性能信息
type PerformanceInfo struct {
	PNL          float64 `json:"pnl"`
	TotalReturn  float64 `json:"totalReturn"`
	SharpeRatio  float64 `json:"sharpeRatio"`
	MaxDrawdown  float64 `json:"maxDrawdown"`
	WinRate      float64 `json:"winRate"`
	ProfitFactor float64 `json:"profitFactor"`
	Volatility   float64 `json:"volatility"`
	TradeCount   int     `json:"tradeCount"`
	AvgTrade     float64 `json:"avgTrade"`
	BestTrade    float64 `json:"bestTrade"`
	WorstTrade   float64 `json:"worstTrade"`
}

// PoolInfo 策略池信息
type PoolInfo struct {
	PoolStatus         string             `json:"poolStatus"` // enabled, disabled, testing, pending
	Priority           string             `json:"priority"`   // high, medium, low
	ResourceAllocation ResourceAllocation `json:"resourceAllocation"`
	PoolMetrics        PoolMetrics        `json:"poolMetrics"`
	LastSync           time.Time          `json:"lastSync"`
	SyncStatus         string             `json:"syncStatus"`
}

// ResourceAllocation 资源分配
type ResourceAllocation struct {
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
	GPU    float64 `json:"gpu,omitempty"`
}

// PoolMetrics 池指标
type PoolMetrics struct {
	QueuePosition   int     `json:"queuePosition"`
	ExecutionWeight float64 `json:"executionWeight"`
	ResourceUsage   float64 `json:"resourceUsage"`
	ConflictCount   int     `json:"conflictCount"`
}

// StrategyListOptions 策略列表选项
type StrategyListOptions struct {
	// 过滤选项
	Status     []string `json:"status,omitempty"`
	Type       []string `json:"type,omitempty"`
	Stage      []string `json:"stage,omitempty"`
	PoolStatus []string `json:"poolStatus,omitempty"`

	// 排序选项
	SortBy    string `json:"sortBy,omitempty"`    // name, performance, created_at, updated_at
	SortOrder string `json:"sortOrder,omitempty"` // asc, desc

	// 分页选项
	Page     int `json:"page,omitempty"`
	PageSize int `json:"pageSize,omitempty"`

	// 视图选项
	View string `json:"view,omitempty"` // list, pool, execution, performance
}

// StrategyListResponse 策略列表响应
type StrategyListResponse struct {
	Strategies []UnifiedStrategy `json:"strategies"`
	Total      int               `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"pageSize"`
	Summary    StrategySummary   `json:"summary"`
}

// StrategySummary 策略摘要
type StrategySummary struct {
	Total struct {
		Count      int `json:"count"`
		Active     int `json:"active"`
		Inactive   int `json:"inactive"`
		Testing    int `json:"testing"`
		Production int `json:"production"`
	} `json:"total"`

	Pool struct {
		Enabled  int `json:"enabled"`
		Disabled int `json:"disabled"`
		Pending  int `json:"pending"`
		Testing  int `json:"testing"`
	} `json:"pool"`

	Performance struct {
		AvgReturn    float64 `json:"avgReturn"`
		AvgSharpe    float64 `json:"avgSharpe"`
		TotalPNL     float64 `json:"totalPnl"`
		WinningCount int     `json:"winningCount"`
	} `json:"performance"`
}

// ListStrategies 获取策略列表
func (s *UnifiedStrategyService) ListStrategies(ctx context.Context, options StrategyListOptions) (*StrategyListResponse, error) {
	// 设置默认值
	if options.PageSize == 0 {
		options.PageSize = 20
	}
	if options.Page == 0 {
		options.Page = 1
	}
	if options.SortBy == "" {
		options.SortBy = "updated_at"
	}
	if options.SortOrder == "" {
		options.SortOrder = "desc"
	}

	// 从数据库获取策略基本信息
	strategies, total, err := s.getStrategiesFromDB(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get strategies from database: %w", err)
	}

	// 增强策略信息（添加执行状态、性能数据、池信息）
	unifiedStrategies := make([]UnifiedStrategy, len(strategies))
	for i, strategy := range strategies {
		unified, err := s.enhanceStrategyInfo(ctx, strategy)
		if err != nil {
			// 记录错误但不中断处理
			s.metrics.RecordAPIError("strategy_enhancement", "enhancement_failed")
			unified = s.createBasicUnifiedStrategy(strategy)
		}
		unifiedStrategies[i] = unified
	}

	// 生成摘要信息
	summary := s.generateSummary(unifiedStrategies)

	return &StrategyListResponse{
		Strategies: unifiedStrategies,
		Total:      total,
		Page:       options.Page,
		PageSize:   options.PageSize,
		Summary:    summary,
	}, nil
}

// GetStrategy 获取单个策略详情
func (s *UnifiedStrategyService) GetStrategy(ctx context.Context, strategyID string) (*UnifiedStrategy, error) {
	// 从数据库获取策略基本信息
	strategy, err := s.getStrategyFromDB(ctx, strategyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get strategy from database: %w", err)
	}

	// 增强策略信息
	unified, err := s.enhanceStrategyInfo(ctx, strategy)
	if err != nil {
		return nil, fmt.Errorf("failed to enhance strategy info: %w", err)
	}

	return &unified, nil
}

// getStrategiesFromDB 从数据库获取策略列表
func (s *UnifiedStrategyService) getStrategiesFromDB(ctx context.Context, options StrategyListOptions) ([]BasicStrategy, int, error) {
	// 构建查询条件
	query := `
		SELECT id, name, type, description, version, status, stage, 
		       created_at, updated_at, config
		FROM strategies 
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// 添加过滤条件
	if len(options.Status) > 0 {
		query += fmt.Sprintf(" AND status = ANY($%d)", argIndex)
		args = append(args, options.Status)
		argIndex++
	}

	if len(options.Type) > 0 {
		query += fmt.Sprintf(" AND type = ANY($%d)", argIndex)
		args = append(args, options.Type)
		argIndex++
	}

	// 添加排序
	query += fmt.Sprintf(" ORDER BY %s %s", options.SortBy, options.SortOrder)

	// 添加分页
	offset := (options.Page - 1) * options.PageSize
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, options.PageSize, offset)

	// 执行查询
	var strategies []BasicStrategy
	var total int

	if s.db != nil && s.db.DB != nil {
		rows, err := s.db.DB.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()

		for rows.Next() {
			var strategy BasicStrategy
			var configJSON sql.NullString

			err := rows.Scan(
				&strategy.ID, &strategy.Name, &strategy.Type, &strategy.Description,
				&strategy.Version, &strategy.Status, &strategy.Stage,
				&strategy.CreatedAt, &strategy.UpdatedAt, &configJSON,
			)
			if err != nil {
				continue
			}

			// 解析配置JSON
			if configJSON.Valid {
				// 解析JSON配置
				var config map[string]interface{}
				if err := json.Unmarshal([]byte(configJSON.String), &config); err != nil {
					log.Printf("Failed to parse strategy config JSON: %v", err)
					strategy.Config = make(map[string]interface{})
				} else {
					strategy.Config = config
				}
			}

			strategies = append(strategies, strategy)
		}

		// 获取总数
		countQuery := "SELECT COUNT(*) FROM strategies WHERE 1=1"
		// 添加相同的过滤条件
		countArgs := make([]interface{}, 0)
		argIndex := 1

		if len(options.Status) > 0 {
			placeholders := make([]string, len(options.Status))
			for i, status := range options.Status {
				placeholders[i] = fmt.Sprintf("$%d", argIndex)
				countArgs = append(countArgs, status)
				argIndex++
			}
			countQuery += fmt.Sprintf(" AND status IN (%s)", strings.Join(placeholders, ","))
		}
		if len(options.Type) > 0 {
			placeholders := make([]string, len(options.Type))
			for i, typeVal := range options.Type {
				placeholders[i] = fmt.Sprintf("$%d", argIndex)
				countArgs = append(countArgs, typeVal)
				argIndex++
			}
			countQuery += fmt.Sprintf(" AND type IN (%s)", strings.Join(placeholders, ","))
		}
		if len(options.Stage) > 0 {
			placeholders := make([]string, len(options.Stage))
			for i, stage := range options.Stage {
				placeholders[i] = fmt.Sprintf("$%d", argIndex)
				countArgs = append(countArgs, stage)
				argIndex++
			}
			countQuery += fmt.Sprintf(" AND stage IN (%s)", strings.Join(placeholders, ","))
		}

		s.db.DB.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	} else {
		// 返回模拟数据用于测试
		strategies = s.getMockStrategies()
		total = len(strategies)
	}

	return strategies, total, nil
}

// BasicStrategy 基础策略信息
type BasicStrategy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Status      string                 `json:"status"`
	Stage       string                 `json:"stage"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	Config      map[string]interface{} `json:"config"`
}

// getStrategyFromDB 从数据库获取单个策略
func (s *UnifiedStrategyService) getStrategyFromDB(ctx context.Context, strategyID string) (BasicStrategy, error) {
	var strategy BasicStrategy

	if s.db != nil && s.db.DB != nil {
		query := `
			SELECT id, name, type, description, version, status, stage,
			       created_at, updated_at, config
			FROM strategies 
			WHERE id = $1
		`

		var configJSON sql.NullString
		err := s.db.DB.QueryRowContext(ctx, query, strategyID).Scan(
			&strategy.ID, &strategy.Name, &strategy.Type, &strategy.Description,
			&strategy.Version, &strategy.Status, &strategy.Stage,
			&strategy.CreatedAt, &strategy.UpdatedAt, &configJSON,
		)

		if err != nil {
			return strategy, err
		}

		// 解析配置JSON
		if configJSON.Valid {
			// TODO: 解析JSON配置
			strategy.Config = make(map[string]interface{})
		}
	} else {
		// 返回模拟数据
		mockStrategies := s.getMockStrategies()
		for _, mock := range mockStrategies {
			if mock.ID == strategyID {
				strategy = mock
				break
			}
		}
		if strategy.ID == "" {
			return strategy, fmt.Errorf("strategy not found: %s", strategyID)
		}
	}

	return strategy, nil
}

// enhanceStrategyInfo 增强策略信息
func (s *UnifiedStrategyService) enhanceStrategyInfo(ctx context.Context, basic BasicStrategy) (UnifiedStrategy, error) {
	unified := s.createBasicUnifiedStrategy(basic)

	// 从策略池获取执行信息
	if s.strategyPool != nil {
		if poolStrategy, err := s.strategyPool.GetStrategyInfo(basic.ID); err == nil {
			unified.Execution = ExecutionInfo{
				IsRunning:      poolStrategy.IsActive && poolStrategy.TradingEnabled,
				LastExecution:  poolStrategy.LastUpdated,
				ExecutionCount: 100, // TODO: 从实际数据获取
				SuccessRate:    95.5,
				AvgLatency:     8.5,
			}

			unified.Pool = PoolInfo{
				PoolStatus: func() string {
					if poolStrategy.TradingEnabled {
						return "enabled"
					}
					return "disabled"
				}(),
				Priority:   "medium",
				LastSync:   poolStrategy.LastUpdated,
				SyncStatus: "success",
			}

			// 性能信息
			if poolStrategy.Performance != nil {
				unified.Performance = PerformanceInfo{
					SharpeRatio:  poolStrategy.Performance.SharpeRatio,
					TotalReturn:  poolStrategy.Performance.TotalReturn,
					MaxDrawdown:  poolStrategy.Performance.MaxDrawdown,
					WinRate:      poolStrategy.Performance.WinRate,
					ProfitFactor: poolStrategy.Performance.ProfitFactor,
					// Volatility:   poolStrategy.Performance.Volatility, // 字段不存在
				}
			}
		}
	}

	// 如果没有从策略池获取到信息，使用模拟数据
	if unified.Execution.ExecutionCount == 0 {
		unified.Execution = s.getMockExecutionInfo()
		unified.Performance = s.getMockPerformanceInfo()
		unified.Pool = s.getMockPoolInfo()
	}

	return unified, nil
}

// createBasicUnifiedStrategy 创建基础统一策略
func (s *UnifiedStrategyService) createBasicUnifiedStrategy(basic BasicStrategy) UnifiedStrategy {
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
	}
}

// generateSummary 生成摘要信息
func (s *UnifiedStrategyService) generateSummary(strategies []UnifiedStrategy) StrategySummary {
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
func (s *UnifiedStrategyService) getMockStrategies() []BasicStrategy {
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
func (s *UnifiedStrategyService) getMockExecutionInfo() ExecutionInfo {
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
func (s *UnifiedStrategyService) getMockPerformanceInfo() PerformanceInfo {
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
func (s *UnifiedStrategyService) getMockPoolInfo() PoolInfo {
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

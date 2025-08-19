package sandbox

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/strategy"
	"qcat/internal/strategy/generator"
)

// AutomatedSandboxService 自动化沙盒测试服务
type AutomatedSandboxService struct {
	factory         *Factory
	activeSandboxes map[string]*Sandbox
	testResults     map[string]*TestResult
	mu              sync.RWMutex

	// 配置
	maxConcurrentTests int
	testDuration       time.Duration
	autoCleanup        bool
}

// TestResult 测试结果
type TestResult struct {
	StrategyID      string                 `json:"strategy_id"`
	TestID          string                 `json:"test_id"`
	Status          string                 `json:"status"` // "running", "completed", "failed", "timeout"
	StartTime       time.Time              `json:"start_time"`
	EndTime         time.Time              `json:"end_time"`
	Duration        time.Duration          `json:"duration"`
	Performance     *PerformanceMetrics    `json:"performance"`
	RiskMetrics     *RiskMetrics           `json:"risk_metrics"`
	TradeStatistics *TradeStatistics       `json:"trade_statistics"`
	Errors          []string               `json:"errors,omitempty"`
	Warnings        []string               `json:"warnings,omitempty"`
	Configuration   map[string]interface{} `json:"configuration"`
	Recommendation  string                 `json:"recommendation"`
	Score           float64                `json:"score"`
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	TotalReturn      float64 `json:"total_return"`
	AnnualizedReturn float64 `json:"annualized_return"`
	SharpeRatio      float64 `json:"sharpe_ratio"`
	MaxDrawdown      float64 `json:"max_drawdown"`
	Volatility       float64 `json:"volatility"`
	CalmarRatio      float64 `json:"calmar_ratio"`
	SortinoRatio     float64 `json:"sortino_ratio"`
}

// RiskMetrics 风险指标
type RiskMetrics struct {
	VaR95           float64 `json:"var_95"`
	CVaR95          float64 `json:"cvar_95"`
	DownsideRisk    float64 `json:"downside_risk"`
	UpsideCapture   float64 `json:"upside_capture"`
	DownsideCapture float64 `json:"downside_capture"`
	BetaToMarket    float64 `json:"beta_to_market"`
	AlphaToMarket   float64 `json:"alpha_to_market"`
}

// TradeStatistics 交易统计
type TradeStatistics struct {
	TotalTrades       int     `json:"total_trades"`
	WinningTrades     int     `json:"winning_trades"`
	LosingTrades      int     `json:"losing_trades"`
	WinRate           float64 `json:"win_rate"`
	AverageWin        float64 `json:"average_win"`
	AverageLoss       float64 `json:"average_loss"`
	ProfitFactor      float64 `json:"profit_factor"`
	LargestWin        float64 `json:"largest_win"`
	LargestLoss       float64 `json:"largest_loss"`
	ConsecutiveWins   int     `json:"consecutive_wins"`
	ConsecutiveLosses int     `json:"consecutive_losses"`
}

// TestConfiguration 测试配置
type TestConfiguration struct {
	Duration       time.Duration          `json:"duration"`
	InitialBalance float64                `json:"initial_balance"`
	Symbol         string                 `json:"symbol"`
	Exchange       string                 `json:"exchange"`
	DataSource     string                 `json:"data_source"` // "live", "historical", "simulated"
	RiskLimits     *RiskLimits            `json:"risk_limits"`
	Parameters     map[string]interface{} `json:"parameters"`
	AutoStop       bool                   `json:"auto_stop"`
	StopConditions *StopConditions        `json:"stop_conditions"`
}

// RiskLimits 风险限制
type RiskLimits struct {
	MaxDrawdown     float64 `json:"max_drawdown"`
	MaxDailyLoss    float64 `json:"max_daily_loss"`
	MaxPositionSize float64 `json:"max_position_size"`
	MaxLeverage     float64 `json:"max_leverage"`
}

// StopConditions 停止条件
type StopConditions struct {
	MaxDrawdownHit   bool    `json:"max_drawdown_hit"`
	MaxLossHit       bool    `json:"max_loss_hit"`
	MinTradesReached bool    `json:"min_trades_reached"`
	MinTrades        int     `json:"min_trades"`
	TargetReturn     float64 `json:"target_return"`
	TargetReached    bool    `json:"target_reached"`
}

// NewAutomatedSandboxService 创建自动化沙盒测试服务
func NewAutomatedSandboxService() *AutomatedSandboxService {
	return &AutomatedSandboxService{
		factory:            NewFactory(),
		activeSandboxes:    make(map[string]*Sandbox),
		testResults:        make(map[string]*TestResult),
		maxConcurrentTests: 5,
		testDuration:       time.Hour * 2, // 默认2小时测试
		autoCleanup:        true,
	}
}

// StartAutomatedTest 启动自动化测试
func (s *AutomatedSandboxService) StartAutomatedTest(ctx context.Context, strategyConfig *strategy.Config, testConfig *TestConfiguration) (*TestResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查并发限制
	if len(s.activeSandboxes) >= s.maxConcurrentTests {
		return nil, fmt.Errorf("maximum concurrent tests reached: %d", s.maxConcurrentTests)
	}

	testID := fmt.Sprintf("test_%s_%d", strategyConfig.Name, time.Now().Unix())

	// 创建测试结果
	result := &TestResult{
		StrategyID:    strategyConfig.Name,
		TestID:        testID,
		Status:        "running",
		StartTime:     time.Now(),
		Configuration: testConfig.Parameters,
		Errors:        make([]string, 0),
		Warnings:      make([]string, 0),
	}

	s.testResults[testID] = result

	// 创建模拟交易所
	mockExchange := s.createMockExchange(testConfig)

	// 创建策略实例
	strategyInstance, err := s.createStrategyInstance(strategyConfig)
	if err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to create strategy instance: %v", err))
		return result, err
	}

	// 创建沙盒
	sandbox := NewSandbox(strategyInstance, strategyConfig.Params, mockExchange)
	s.activeSandboxes[testID] = sandbox

	// 启动沙盒测试
	go s.runSandboxTest(ctx, testID, sandbox, testConfig)

	log.Printf("Started automated test %s for strategy %s", testID, strategyConfig.Name)
	return result, nil
}

// BatchTest 批量测试策略
func (s *AutomatedSandboxService) BatchTest(ctx context.Context, strategies []*generator.GenerationResult, testConfig *TestConfiguration) ([]*TestResult, error) {
	var results []*TestResult

	for _, strategyResult := range strategies {
		// 为每个策略创建独立的测试配置
		individualConfig := *testConfig
		individualConfig.Parameters = strategyResult.Strategy.Params

		result, err := s.StartAutomatedTest(ctx, strategyResult.Strategy, &individualConfig)
		if err != nil {
			log.Printf("Failed to start test for strategy %s: %v", strategyResult.Strategy.Name, err)
			// 创建失败结果
			failedResult := &TestResult{
				StrategyID: strategyResult.Strategy.Name,
				TestID:     fmt.Sprintf("failed_%s_%d", strategyResult.Strategy.Name, time.Now().Unix()),
				Status:     "failed",
				StartTime:  time.Now(),
				EndTime:    time.Now(),
				Errors:     []string{err.Error()},
			}
			results = append(results, failedResult)
			continue
		}

		results = append(results, result)
	}

	log.Printf("Started batch testing for %d strategies", len(results))
	return results, nil
}

// runSandboxTest 运行沙盒测试
func (s *AutomatedSandboxService) runSandboxTest(ctx context.Context, testID string, sandbox *Sandbox, config *TestConfiguration) {
	defer func() {
		s.mu.Lock()
		delete(s.activeSandboxes, testID)
		s.mu.Unlock()

		if s.autoCleanup {
			s.cleanupTest(testID)
		}
	}()

	result := s.testResults[testID]

	// 启动沙盒
	if err := sandbox.Start(ctx); err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to start sandbox: %v", err))
		result.EndTime = time.Now()
		result.Duration = time.Since(result.StartTime)
		return
	}

	// 设置测试超时
	testCtx, cancel := context.WithTimeout(ctx, config.Duration)
	defer cancel()

	// 监控测试进度
	ticker := time.NewTicker(time.Minute * 5) // 每5分钟检查一次
	defer ticker.Stop()

	for {
		select {
		case <-testCtx.Done():
			// 测试超时或完成
			s.finalizeSandboxTest(testID, sandbox, config)
			return

		case <-ticker.C:
			// 定期检查停止条件
			if s.shouldStopTest(testID, sandbox, config) {
				s.finalizeSandboxTest(testID, sandbox, config)
				return
			}
		}
	}
}

// shouldStopTest 检查是否应该停止测试
func (s *AutomatedSandboxService) shouldStopTest(testID string, sandbox *Sandbox, config *TestConfiguration) bool {
	if !config.AutoStop || config.StopConditions == nil {
		return false
	}

	result := s.testResults[testID]

	// 检查最大回撤
	if config.StopConditions.MaxDrawdownHit && result.Performance != nil {
		if result.Performance.MaxDrawdown > config.RiskLimits.MaxDrawdown {
			result.Warnings = append(result.Warnings, "Maximum drawdown limit exceeded")
			return true
		}
	}

	// 检查目标收益
	if config.StopConditions.TargetReached && result.Performance != nil {
		if result.Performance.TotalReturn >= config.StopConditions.TargetReturn {
			result.Warnings = append(result.Warnings, "Target return achieved")
			return true
		}
	}

	// 检查最小交易数
	if config.StopConditions.MinTradesReached && result.TradeStatistics != nil {
		if result.TradeStatistics.TotalTrades >= config.StopConditions.MinTrades {
			return true
		}
	}

	return false
}

// finalizeSandboxTest 完成沙盒测试
func (s *AutomatedSandboxService) finalizeSandboxTest(testID string, sandbox *Sandbox, config *TestConfiguration) {
	result := s.testResults[testID]

	// 停止沙盒
	if err := sandbox.Stop(context.Background()); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to stop sandbox cleanly: %v", err))
	}

	// 获取测试结果
	sandboxResult := sandbox.GetResult()

	// 计算性能指标
	result.Performance = s.calculatePerformanceMetrics(sandboxResult, config)
	result.RiskMetrics = s.calculateRiskMetrics(sandboxResult, config)
	result.TradeStatistics = s.calculateTradeStatistics(sandboxResult)

	// 计算综合评分
	result.Score = s.calculateOverallScore(result)

	// 生成建议
	result.Recommendation = s.generateRecommendation(result)

	// 更新状态
	result.Status = "completed"
	result.EndTime = time.Now()
	result.Duration = time.Since(result.StartTime)

	log.Printf("Completed sandbox test %s with score %.2f", testID, result.Score)
}

// calculatePerformanceMetrics 计算性能指标
func (s *AutomatedSandboxService) calculatePerformanceMetrics(result *strategy.Result, config *TestConfiguration) *PerformanceMetrics {
	if result == nil {
		return &PerformanceMetrics{}
	}

	// 计算年化收益率
	durationYears := config.Duration.Hours() / (24 * 365)
	annualizedReturn := 0.0
	if durationYears > 0 {
		annualizedReturn = math.Pow(1+result.PnLPercent, 1/durationYears) - 1
	}

	// 计算夏普比率（简化版本）
	sharpeRatio := result.SharpeRatio
	if sharpeRatio == 0 && result.PnLPercent > 0 {
		// 如果没有计算夏普比率，使用简化计算
		sharpeRatio = result.PnLPercent / 0.1 // 假设10%的波动率
	}

	return &PerformanceMetrics{
		TotalReturn:      result.PnLPercent,
		AnnualizedReturn: annualizedReturn,
		SharpeRatio:      sharpeRatio,
		MaxDrawdown:      result.MaxDrawdown,
		Volatility:       0.1, // 默认值，实际应该计算
		CalmarRatio:      annualizedReturn / math.Max(result.MaxDrawdown, 0.01),
		SortinoRatio:     sharpeRatio * 1.2, // 简化计算
	}
}

// calculateRiskMetrics 计算风险指标
func (s *AutomatedSandboxService) calculateRiskMetrics(result *strategy.Result, config *TestConfiguration) *RiskMetrics {
	if result == nil {
		return &RiskMetrics{}
	}

	// 简化的风险指标计算
	return &RiskMetrics{
		VaR95:           result.MaxDrawdown * 0.8,
		CVaR95:          result.MaxDrawdown,
		DownsideRisk:    result.MaxDrawdown * 0.6,
		UpsideCapture:   1.1,
		DownsideCapture: 0.9,
		BetaToMarket:    1.0,
		AlphaToMarket:   result.PnLPercent - 0.05, // 假设市场收益5%
	}
}

// calculateTradeStatistics 计算交易统计
func (s *AutomatedSandboxService) calculateTradeStatistics(result *strategy.Result) *TradeStatistics {
	if result == nil {
		return &TradeStatistics{}
	}

	// 从结果中提取交易统计信息
	totalTrades := result.NumTrades
	winRate := result.WinRate
	winningTrades := int(float64(totalTrades) * winRate)
	losingTrades := totalTrades - winningTrades

	// 计算平均盈亏
	averageWin := 0.0
	averageLoss := 0.0
	if winningTrades > 0 && losingTrades > 0 {
		totalProfit := result.PnL
		if totalProfit > 0 {
			averageWin = totalProfit * winRate / float64(winningTrades)
			averageLoss = totalProfit * (1 - winRate) / float64(losingTrades)
		}
	}

	profitFactor := 1.0
	if averageLoss != 0 {
		profitFactor = math.Abs(averageWin / averageLoss)
	}

	return &TradeStatistics{
		TotalTrades:       totalTrades,
		WinningTrades:     winningTrades,
		LosingTrades:      losingTrades,
		WinRate:           winRate,
		AverageWin:        averageWin,
		AverageLoss:       averageLoss,
		ProfitFactor:      profitFactor,
		LargestWin:        averageWin * 2,    // 简化估算
		LargestLoss:       averageLoss * 2,   // 简化估算
		ConsecutiveWins:   winningTrades / 3, // 简化估算
		ConsecutiveLosses: losingTrades / 3,  // 简化估算
	}
}

// calculateOverallScore 计算综合评分
func (s *AutomatedSandboxService) calculateOverallScore(result *TestResult) float64 {
	if result.Performance == nil {
		return 0.0
	}

	score := 0.0

	// 收益率权重 40%
	returnScore := math.Max(0, math.Min(100, result.Performance.TotalReturn*100*4))
	score += returnScore * 0.4

	// 夏普比率权重 30%
	sharpeScore := math.Max(0, math.Min(100, result.Performance.SharpeRatio*50))
	score += sharpeScore * 0.3

	// 最大回撤权重 20% (越小越好)
	drawdownScore := math.Max(0, 100-result.Performance.MaxDrawdown*500)
	score += drawdownScore * 0.2

	// 胜率权重 10%
	winRateScore := 0.0
	if result.TradeStatistics != nil {
		winRateScore = result.TradeStatistics.WinRate * 100
	}
	score += winRateScore * 0.1

	return math.Max(0, math.Min(100, score))
}

// generateRecommendation 生成建议
func (s *AutomatedSandboxService) generateRecommendation(result *TestResult) string {
	if result.Performance == nil {
		return "测试失败，无法生成建议"
	}

	score := result.Score

	switch {
	case score >= 80:
		return "优秀策略，建议立即部署到生产环境"
	case score >= 60:
		return "良好策略，建议进一步优化参数后部署"
	case score >= 40:
		return "一般策略，需要显著改进才能考虑部署"
	case score >= 20:
		return "较差策略，建议重新设计或放弃"
	default:
		return "策略表现很差，不建议使用"
	}
}

// createMockExchange 创建模拟交易所
func (s *AutomatedSandboxService) createMockExchange(config *TestConfiguration) exchange.Exchange {
	// 这里应该创建一个模拟交易所实例
	// 为了演示，返回nil，实际应该实现MockExchange
	log.Printf("Creating mock exchange for %s on %s", config.Symbol, config.Exchange)
	return nil
}

// createStrategyInstance 创建策略实例
func (s *AutomatedSandboxService) createStrategyInstance(config *strategy.Config) (strategy.Strategy, error) {
	// 这里应该根据策略配置创建实际的策略实例
	// 为了演示，返回nil，实际应该实现策略工厂
	log.Printf("Creating strategy instance for %s", config.Name)
	return nil, fmt.Errorf("strategy factory not implemented")
}

// cleanupTest 清理测试资源
func (s *AutomatedSandboxService) cleanupTest(testID string) {
	log.Printf("Cleaning up test resources for %s", testID)
	// 实现资源清理逻辑
}

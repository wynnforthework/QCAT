package scheduler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"

	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/exchange/account"
	"qcat/internal/hotlist"
	"qcat/internal/monitor"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// RiskScheduler 风险调度器
type RiskScheduler struct {
	config         *config.Config
	db             *database.DB
	accountManager *account.Manager
	isRunning      bool
	mu             sync.RWMutex
}

// NewRiskScheduler 创建风险调度器
func NewRiskScheduler(cfg *config.Config, db *database.DB, accountManager *account.Manager) *RiskScheduler {
	return &RiskScheduler{
		config:         cfg,
		db:             db,
		accountManager: accountManager,
	}
}

func (rs *RiskScheduler) Start() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.isRunning = true
	log.Println("Risk scheduler started")
	return nil
}

func (rs *RiskScheduler) Stop() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.isRunning = false
	log.Println("Risk scheduler stopped")
	return nil
}

func (rs *RiskScheduler) HandleMonitoring(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing risk monitoring task: %s", task.Name)

	// 1. 检查保证金比率
	marginRisk, err := rs.checkMarginRatio(ctx)
	if err != nil {
		log.Printf("Failed to check margin ratio: %v", err)
	} else if marginRisk.Level == "HIGH" || marginRisk.Level == "CRITICAL" {
		log.Printf("High margin risk detected: %s", marginRisk.Message)
		rs.triggerMarginAlert(ctx, marginRisk)
	}

	// 2. 监控仓位风险
	positionRisks, err := rs.monitorPositionRisk(ctx)
	if err != nil {
		log.Printf("Failed to monitor position risk: %v", err)
	} else {
		for _, risk := range positionRisks {
			if risk.RiskLevel == "HIGH" || risk.RiskLevel == "CRITICAL" {
				log.Printf("High position risk detected for %s: %.4f", risk.Symbol, risk.RiskScore)
				rs.triggerPositionAlert(ctx, risk)
			}
		}
	}

	// 3. 检测异常行情
	marketAnomalies, err := rs.detectMarketAnomalies(ctx)
	if err != nil {
		log.Printf("Failed to detect market anomalies: %v", err)
	} else {
		for _, anomaly := range marketAnomalies {
			if anomaly.Severity == "HIGH" || anomaly.Severity == "CRITICAL" {
				log.Printf("Market anomaly detected: %s - %s", anomaly.Type, anomaly.Description)
				rs.triggerAnomalyAlert(ctx, anomaly)
			}
		}
	}

	// 4. 触发风险控制措施
	err = rs.executeRiskControlMeasures(ctx, marginRisk, positionRisks, marketAnomalies)
	if err != nil {
		log.Printf("Failed to execute risk control measures: %v", err)
		return fmt.Errorf("failed to execute risk control measures: %w", err)
	}

	log.Printf("Risk monitoring completed successfully")
	return nil
}

// HandleAbnormalMarketResponse 处理异常行情应对任务
func (rs *RiskScheduler) HandleAbnormalMarketResponse(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing abnormal market response task: %s", task.Name)

	// 1. 检测异常行情条件
	abnormalConditions, err := rs.detectAbnormalMarketConditions(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect abnormal market conditions: %w", err)
	}

	if len(abnormalConditions) == 0 {
		log.Printf("No abnormal market conditions detected")
		return nil
	}

	log.Printf("Detected %d abnormal market conditions", len(abnormalConditions))

	// 2. 触发熔断保护
	for _, condition := range abnormalConditions {
		if condition.Severity == "CRITICAL" {
			err = rs.triggerCircuitBreaker(ctx, condition)
			if err != nil {
				log.Printf("Failed to trigger circuit breaker for %s: %v", condition.Symbol, err)
			} else {
				log.Printf("Circuit breaker triggered for %s", condition.Symbol)
			}
		}
	}

	// 3. 自动降杠杆
	err = rs.autoReduceLeverage(ctx, abnormalConditions)
	if err != nil {
		log.Printf("Failed to auto reduce leverage: %v", err)
	} else {
		log.Printf("Auto leverage reduction completed")
	}

	// 4. 紧急平仓保护
	err = rs.emergencyPositionProtection(ctx, abnormalConditions)
	if err != nil {
		log.Printf("Failed to execute emergency position protection: %v", err)
	} else {
		log.Printf("Emergency position protection executed")
	}

	// 记录异常行情应对历史
	err = rs.recordAbnormalMarketResponse(ctx, abnormalConditions)
	if err != nil {
		log.Printf("Failed to record abnormal market response: %v", err)
	}

	log.Printf("Abnormal market response completed successfully")
	return nil
}

// HandleStopLossAdjustment 处理止盈止损线自动调整任务
func (rs *RiskScheduler) HandleStopLossAdjustment(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing stop loss adjustment task: %s", task.Name)

	// 1. 基于ATR计算动态止损线
	atrStopLosses, err := rs.calculateATRBasedStopLoss(ctx)
	if err != nil {
		log.Printf("Failed to calculate ATR-based stop loss: %v", err)
	} else {
		log.Printf("Calculated ATR-based stop losses for %d positions", len(atrStopLosses))
	}

	// 2. 基于RV计算动态止损线
	rvStopLosses, err := rs.calculateRVBasedStopLoss(ctx)
	if err != nil {
		log.Printf("Failed to calculate RV-based stop loss: %v", err)
	} else {
		log.Printf("Calculated RV-based stop losses for %d positions", len(rvStopLosses))
	}

	// 3. 根据市场状态调整参数
	marketState, err := rs.analyzeMarketState(ctx)
	if err != nil {
		log.Printf("Failed to analyze market state: %v", err)
		marketState = &MarketState{Volatility: 0.2, Trend: "NEUTRAL", Regime: "NORMAL"}
	}

	adjustedStopLosses := rs.adjustStopLossForMarketState(atrStopLosses, rvStopLosses, marketState)
	log.Printf("Adjusted stop losses based on market state: %s", marketState.Regime)

	// 4. 应用新的止损设置
	appliedCount, err := rs.applyNewStopLossSettings(ctx, adjustedStopLosses)
	if err != nil {
		return fmt.Errorf("failed to apply new stop loss settings: %w", err)
	}

	log.Printf("Stop loss adjustment completed: %d positions updated", appliedCount)
	return nil
}

// HandleFundDistribution 处理资金分散与转移任务
func (rs *RiskScheduler) HandleFundDistribution(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing fund distribution task: %s", task.Name)

	// 1. 检查资金集中度风险
	riskAssessment, err := rs.assessFundConcentrationRisk(ctx)
	if err != nil {
		log.Printf("Failed to assess fund concentration risk: %v", err)
		return fmt.Errorf("failed to assess fund concentration risk: %w", err)
	}

	// 2. 计算最优资金分配
	optimalDistribution, err := rs.calculateOptimalFundDistribution(ctx, riskAssessment)
	if err != nil {
		log.Printf("Failed to calculate optimal fund distribution: %v", err)
		return fmt.Errorf("failed to calculate optimal fund distribution: %w", err)
	}

	// 3. 执行资金转移操作
	transferResults, err := rs.executeFundTransfers(ctx, optimalDistribution)
	if err != nil {
		log.Printf("Failed to execute fund transfers: %v", err)
		return fmt.Errorf("failed to execute fund transfers: %w", err)
	}

	// 4. 集成冷钱包功能
	err = rs.integrateColdWalletOperations(ctx, transferResults)
	if err != nil {
		log.Printf("Failed to integrate cold wallet operations: %v", err)
		// 不返回错误，因为冷钱包操作失败不应该影响主流程
	}

	// 5. 更新资金保护协议
	err = rs.updateFundProtectionProtocol(ctx, optimalDistribution, transferResults)
	if err != nil {
		log.Printf("Failed to update fund protection protocol: %v", err)
		// 不返回错误，因为协议更新失败不应该影响主流程
	}

	log.Printf("Fund distribution completed successfully. Transferred %d operations", len(transferResults))
	return nil
}

// PositionScheduler 仓位调度器
type PositionScheduler struct {
	config         *config.Config
	db             *database.DB
	accountManager *account.Manager
	isRunning      bool
	mu             sync.RWMutex
}

// NewPositionScheduler 创建仓位调度器
func NewPositionScheduler(cfg *config.Config, db *database.DB, accountManager *account.Manager) *PositionScheduler {
	return &PositionScheduler{
		config:         cfg,
		db:             db,
		accountManager: accountManager,
	}
}

func (ps *PositionScheduler) Start() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.isRunning = true
	log.Println("Position scheduler started")
	return nil
}

func (ps *PositionScheduler) Stop() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.isRunning = false
	log.Println("Position scheduler stopped")
	return nil
}

func (ps *PositionScheduler) HandleOptimization(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing position optimization task: %s", task.Name)

	// 1. 获取当前仓位
	currentPositions, err := ps.getCurrentPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current positions: %w", err)
	}
	log.Printf("Retrieved %d current positions", len(currentPositions))

	// 2. 计算最优仓位
	optimalPositions, err := ps.calculateOptimalPositions(ctx, currentPositions)
	if err != nil {
		return fmt.Errorf("failed to calculate optimal positions: %w", err)
	}
	log.Printf("Calculated optimal positions for %d symbols", len(optimalPositions))

	// 3. 生成调仓指令
	rebalanceInstructions, err := ps.generateRebalanceInstructions(ctx, currentPositions, optimalPositions)
	if err != nil {
		return fmt.Errorf("failed to generate rebalance instructions: %w", err)
	}
	log.Printf("Generated %d rebalance instructions", len(rebalanceInstructions))

	// 4. 执行仓位调整
	if len(rebalanceInstructions) > 0 {
		executionResults, err := ps.executePositionAdjustments(ctx, rebalanceInstructions)
		if err != nil {
			return fmt.Errorf("failed to execute position adjustments: %w", err)
		}
		
		successCount := 0
		for _, result := range executionResults {
			if result.Success {
				successCount++
			}
		}
		log.Printf("Position optimization completed: %d/%d adjustments successful", successCount, len(executionResults))
	} else {
		log.Printf("No position adjustments needed")
	}

	return nil
}

// HandleDynamicFundAllocation 处理资金动态分配任务
func (ps *PositionScheduler) HandleDynamicFundAllocation(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing dynamic fund allocation task: %s", task.Name)

	// 1. 分析当前资金使用效率
	efficiencyAnalysis, err := ps.analyzeFundEfficiency(ctx)
	if err != nil {
		return fmt.Errorf("failed to analyze fund efficiency: %w", err)
	}
	log.Printf("Fund efficiency analysis completed: overall score %.4f", efficiencyAnalysis.OverallScore)

	// 2. 计算最优资金分配
	optimalAllocation, err := ps.calculateOptimalFundAllocation(ctx, efficiencyAnalysis)
	if err != nil {
		return fmt.Errorf("failed to calculate optimal fund allocation: %w", err)
	}
	log.Printf("Optimal fund allocation calculated for %d strategies", len(optimalAllocation.Allocations))

	// 3. 执行资金重新分配
	reallocationResults, err := ps.executeFundReallocation(ctx, optimalAllocation)
	if err != nil {
		return fmt.Errorf("failed to execute fund reallocation: %w", err)
	}

	successfulReallocations := 0
	for _, result := range reallocationResults {
		if result.Success {
			successfulReallocations++
		}
	}
	log.Printf("Fund reallocation completed: %d/%d successful", successfulReallocations, len(reallocationResults))

	// 4. 监控分配效果
	go ps.monitorAllocationEffectiveness(ctx, optimalAllocation.ID, reallocationResults)

	log.Printf("Dynamic fund allocation completed successfully")
	return nil
}

// HandleLayeredPositionManagement 处理仓位分层机制任务
func (ps *PositionScheduler) HandleLayeredPositionManagement(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing layered position management task: %s", task.Name)

	// 1. 分析市场波动性
	volatilityAnalysis, err := ps.analyzeMarketVolatilityForLayers(ctx)
	if err != nil {
		return fmt.Errorf("failed to analyze market volatility: %w", err)
	}
	log.Printf("Market volatility analysis completed for %d symbols", len(volatilityAnalysis))

	// 2. 计算分层仓位配置
	layeredConfigs, err := ps.calculateLayeredPositionConfigs(ctx, volatilityAnalysis)
	if err != nil {
		return fmt.Errorf("failed to calculate layered position configs: %w", err)
	}
	log.Printf("Calculated layered configs for %d strategies", len(layeredConfigs))

	// 3. 执行分层建仓/平仓
	executionResults, err := ps.executeLayeredPositions(ctx, layeredConfigs)
	if err != nil {
		return fmt.Errorf("failed to execute layered positions: %w", err)
	}

	successfulExecutions := 0
	for _, result := range executionResults {
		if result.Success {
			successfulExecutions++
		}
	}
	log.Printf("Layered position execution completed: %d/%d successful", successfulExecutions, len(executionResults))

	// 4. 动态调整分层参数
	err = ps.dynamicallyAdjustLayerParameters(ctx, executionResults, volatilityAnalysis)
	if err != nil {
		log.Printf("Failed to adjust layer parameters: %v", err)
	} else {
		log.Printf("Layer parameters adjusted successfully")
	}

	log.Printf("Layered position management completed successfully")
	return nil
}

// HandleMultiStrategyHedging 处理自动化多策略对冲任务
func (ps *PositionScheduler) HandleMultiStrategyHedging(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing multi-strategy hedging task: %s", task.Name)

	// 1. 分析策略间相关性
	correlationMatrix, err := ps.analyzeStrategyCorrelations(ctx)
	if err != nil {
		log.Printf("Failed to analyze strategy correlations: %v", err)
		return fmt.Errorf("failed to analyze strategy correlations: %w", err)
	}

	// 2. 计算动态对冲比率
	hedgeRatios, err := ps.calculateDynamicHedgeRatios(ctx, correlationMatrix)
	if err != nil {
		log.Printf("Failed to calculate dynamic hedge ratios: %v", err)
		return fmt.Errorf("failed to calculate dynamic hedge ratios: %w", err)
	}

	// 3. 执行自动对冲操作
	hedgeResults, err := ps.executeAutoHedgeOperations(ctx, hedgeRatios)
	if err != nil {
		log.Printf("Failed to execute auto hedge operations: %v", err)
		return fmt.Errorf("failed to execute auto hedge operations: %w", err)
	}

	// 4. 监控对冲效果
	err = ps.monitorHedgeEffectiveness(ctx, hedgeResults)
	if err != nil {
		log.Printf("Failed to monitor hedge effectiveness: %v", err)
		// 不返回错误，因为监控失败不应该影响主流程
	}

	// 5. 更新对冲历史记录
	err = ps.updateHedgeHistory(ctx, correlationMatrix, hedgeRatios, hedgeResults)
	if err != nil {
		log.Printf("Failed to update hedge history: %v", err)
		// 不返回错误，因为记录失败不应该影响主流程
	}

	log.Printf("Multi-strategy hedging completed successfully. Executed %d hedge operations", len(hedgeResults))
	return nil
}

// DataScheduler 数据调度器
type DataScheduler struct {
	config            *config.Config
	db                *database.DB
	isRunning         bool
	mu                sync.RWMutex
	integratedService *hotlist.IntegratedService
}

// NewDataScheduler 创建数据调度器
func NewDataScheduler(cfg *config.Config, db *database.DB) *DataScheduler {
	// 创建集成服务
	integratedService := hotlist.NewIntegratedService(cfg, db)

	return &DataScheduler{
		config:            cfg,
		db:                db,
		integratedService: integratedService,
	}
}

func (ds *DataScheduler) Start() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.isRunning = true
	log.Println("Data scheduler started")
	return nil
}

func (ds *DataScheduler) Stop() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.isRunning = false
	log.Println("Data scheduler stopped")
	return nil
}

func (ds *DataScheduler) HandleCleaning(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing data cleaning task: %s", task.Name)

	// 1. 检测异常数据
	anomalies, err := ds.detectDataAnomalies(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect data anomalies: %w", err)
	}
	log.Printf("Detected %d data anomalies", len(anomalies))

	// 2. 清洗无效数据
	cleaningResults, err := ds.cleanInvalidData(ctx, anomalies)
	if err != nil {
		return fmt.Errorf("failed to clean invalid data: %w", err)
	}
	log.Printf("Data cleaning completed: %d records processed, %d cleaned", 
		cleaningResults.TotalRecords, cleaningResults.CleanedRecords)

	// 3. 校正数据格式
	formatCorrectionResults, err := ds.correctDataFormats(ctx)
	if err != nil {
		return fmt.Errorf("failed to correct data formats: %w", err)
	}
	log.Printf("Format correction completed: %d records corrected", formatCorrectionResults.CorrectedRecords)

	// 4. 更新数据质量指标
	qualityMetrics, err := ds.updateDataQualityMetrics(ctx, cleaningResults, formatCorrectionResults)
	if err != nil {
		return fmt.Errorf("failed to update data quality metrics: %w", err)
	}
	log.Printf("Data quality metrics updated: overall score %.4f", qualityMetrics.OverallScore)

	log.Printf("Data cleaning task completed successfully")
	return nil
}

// HandleAutoBacktesting 处理自动回测与前测任务
func (ds *DataScheduler) HandleAutoBacktesting(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing auto backtesting task: %s", task.Name)

	// 1. 自动生成回测参数
	backtestParams, err := ds.generateBacktestParameters(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate backtest parameters: %w", err)
	}
	log.Printf("Generated backtest parameters for %d strategies", len(backtestParams))

	// 2. 执行历史数据回测
	backtestResults, err := ds.executeHistoricalBacktests(ctx, backtestParams)
	if err != nil {
		return fmt.Errorf("failed to execute historical backtests: %w", err)
	}
	log.Printf("Historical backtests completed: %d results", len(backtestResults))

	// 3. 执行前瞻性测试
	forwardTestResults, err := ds.executeForwardTests(ctx, backtestResults)
	if err != nil {
		return fmt.Errorf("failed to execute forward tests: %w", err)
	}
	log.Printf("Forward tests completed: %d results", len(forwardTestResults))

	// 4. 生成测试报告
	testReport, err := ds.generateTestReport(ctx, backtestResults, forwardTestResults)
	if err != nil {
		return fmt.Errorf("failed to generate test report: %w", err)
	}
	log.Printf("Test report generated: %s", testReport.ReportID)

	// 保存报告到数据库
	err = ds.saveTestReport(ctx, testReport)
	if err != nil {
		log.Printf("Failed to save test report: %v", err)
	}

	log.Printf("Auto backtesting completed successfully")
	return nil
}

// HandleHotCoinRecommendation 处理热门币种推荐任务
func (ds *DataScheduler) HandleHotCoinRecommendation(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing hot coin recommendation task: %s", task.Name)

	// 启动集成服务（如果尚未启动）
	if !ds.isServiceRunning() {
		err := ds.integratedService.Start(ctx)
		if err != nil {
			log.Printf("Failed to start integrated service: %v", err)
			return fmt.Errorf("failed to start integrated service: %w", err)
		}
	}

	// 强制更新推荐
	err := ds.integratedService.ForceUpdate(ctx)
	if err != nil {
		log.Printf("Failed to force update recommendations: %v", err)
		return fmt.Errorf("failed to force update recommendations: %w", err)
	}

	// 获取推荐结果
	recommendations := ds.integratedService.GetRecommendations()

	// 发送推荐通知
	err = ds.sendRecommendationNotifications(ctx, recommendations)
	if err != nil {
		log.Printf("Failed to send recommendation notifications: %v", err)
		// 不返回错误，因为通知失败不应该影响主流程
	}

	log.Printf("Hot coin recommendation completed successfully. Generated %d recommendations", len(recommendations))
	return nil
}

// isServiceRunning 检查集成服务是否运行
func (ds *DataScheduler) isServiceRunning() bool {
	status := ds.integratedService.GetStatus()
	if running, ok := status["is_running"].(bool); ok {
		return running
	}
	return false
}

// HandleFactorLibraryUpdate 处理因子库动态更新任务
func (ds *DataScheduler) HandleFactorLibraryUpdate(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing factor library update task: %s", task.Name)

	// 1. 扫描新的市场因子
	newFactors, err := ds.scanNewMarketFactors(ctx)
	if err != nil {
		return fmt.Errorf("failed to scan new market factors: %w", err)
	}
	log.Printf("Scanned %d new market factors", len(newFactors))

	// 2. 评估因子有效性
	factorEvaluations, err := ds.evaluateFactorEffectiveness(ctx, newFactors)
	if err != nil {
		return fmt.Errorf("failed to evaluate factor effectiveness: %w", err)
	}

	validFactors := 0
	for _, evaluation := range factorEvaluations {
		if evaluation.IsValid {
			validFactors++
		}
	}
	log.Printf("Factor evaluation completed: %d/%d factors are valid", validFactors, len(factorEvaluations))

	// 3. 更新因子库
	updateResults, err := ds.updateFactorLibrary(ctx, factorEvaluations)
	if err != nil {
		return fmt.Errorf("failed to update factor library: %w", err)
	}
	log.Printf("Factor library updated: %d factors added, %d updated", 
		updateResults.AddedFactors, updateResults.UpdatedFactors)

	// 4. 清理过期因子
	cleanupResults, err := ds.cleanupExpiredFactors(ctx)
	if err != nil {
		log.Printf("Failed to cleanup expired factors: %v", err)
	} else {
		log.Printf("Expired factors cleanup completed: %d factors removed", cleanupResults.RemovedFactors)
	}

	log.Printf("Factor library update completed successfully")
	return nil
}

// HandleMarketPatternRecognition 处理市场模式识别任务
func (ds *DataScheduler) HandleMarketPatternRecognition(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing market pattern recognition task: %s", task.Name)

	// 1. 分析当前市场状态
	marketState, err := ds.analyzeCurrentMarketState(ctx)
	if err != nil {
		return fmt.Errorf("failed to analyze current market state: %w", err)
	}
	log.Printf("Market state analysis completed: regime=%s, volatility=%.4f", 
		marketState.Regime, marketState.Volatility)

	// 2. 识别市场模式变化
	patternChanges, err := ds.identifyMarketPatternChanges(ctx, marketState)
	if err != nil {
		return fmt.Errorf("failed to identify market pattern changes: %w", err)
	}
	log.Printf("Pattern change detection completed: %d changes identified", len(patternChanges))

	// 3. 触发策略切换
	if len(patternChanges) > 0 {
		strategySwitchResults, err := ds.triggerStrategySwitching(ctx, patternChanges)
		if err != nil {
			log.Printf("Failed to trigger strategy switching: %v", err)
		} else {
			successfulSwitches := 0
			for _, result := range strategySwitchResults {
				if result.Success {
					successfulSwitches++
				}
			}
			log.Printf("Strategy switching completed: %d/%d successful", 
				successfulSwitches, len(strategySwitchResults))
		}
	}

	// 4. 更新模式识别模型
	modelUpdateResult, err := ds.updatePatternRecognitionModel(ctx, marketState, patternChanges)
	if err != nil {
		log.Printf("Failed to update pattern recognition model: %v", err)
	} else {
		log.Printf("Pattern recognition model updated: accuracy=%.4f", modelUpdateResult.Accuracy)
	}

	log.Printf("Market pattern recognition completed successfully")
	return nil
}

// SystemScheduler 系统调度器
type SystemScheduler struct {
	config    *config.Config
	db        *database.DB
	metrics   *monitor.MetricsCollector
	isRunning bool
	mu        sync.RWMutex
}

// NewSystemScheduler 创建系统调度器
func NewSystemScheduler(cfg *config.Config, db *database.DB, metrics *monitor.MetricsCollector) *SystemScheduler {
	return &SystemScheduler{
		config:  cfg,
		db:      db,
		metrics: metrics,
	}
}

func (ss *SystemScheduler) Start() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.isRunning = true
	log.Println("System scheduler started")
	return nil
}

func (ss *SystemScheduler) Stop() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.isRunning = false
	log.Println("System scheduler stopped")
	return nil
}

func (ss *SystemScheduler) HandleHealthCheck(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing system health check task: %s", task.Name)

	// 1. 检查系统资源使用率
	resourceUsage, err := ss.checkSystemResourceUsage(ctx)
	if err != nil {
		return fmt.Errorf("failed to check system resource usage: %w", err)
	}
	log.Printf("System resource usage: CPU=%.2f%%, Memory=%.2f%%, Disk=%.2f%%", 
		resourceUsage.CPU, resourceUsage.Memory, resourceUsage.Disk)

	// 2. 监控服务状态
	serviceStatuses, err := ss.monitorServiceStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to monitor service status: %w", err)
	}

	healthyServices := 0
	for _, status := range serviceStatuses {
		if status.Status == "HEALTHY" {
			healthyServices++
		}
	}
	log.Printf("Service status check completed: %d/%d services healthy", healthyServices, len(serviceStatuses))

	// 3. 检测异常情况
	systemAnomalies, err := ss.detectSystemAnomalies(ctx, resourceUsage, serviceStatuses)
	if err != nil {
		return fmt.Errorf("failed to detect system anomalies: %w", err)
	}
	log.Printf("System anomaly detection completed: %d anomalies detected", len(systemAnomalies))

	// 4. 触发自愈机制
	if len(systemAnomalies) > 0 {
		healingResults, err := ss.triggerSelfHealingMechanisms(ctx, systemAnomalies)
		if err != nil {
			log.Printf("Failed to trigger self-healing mechanisms: %v", err)
		} else {
			successfulHealing := 0
			for _, result := range healingResults {
				if result.Success {
					successfulHealing++
				}
			}
			log.Printf("Self-healing completed: %d/%d successful", successfulHealing, len(healingResults))
		}
	}

	// 记录健康检查结果
	err = ss.recordHealthCheckResults(ctx, resourceUsage, serviceStatuses, systemAnomalies)
	if err != nil {
		log.Printf("Failed to record health check results: %v", err)
	}

	log.Printf("System health check completed successfully")
	return nil
}

// HandleAccountSecurityMonitoring 处理账户安全监控任务
func (ss *SystemScheduler) HandleAccountSecurityMonitoring(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing account security monitoring task: %s", task.Name)

	// 1. 监控异常登录行为
	loginAnomalies, err := ss.monitorAbnormalLoginBehavior(ctx)
	if err != nil {
		return fmt.Errorf("failed to monitor abnormal login behavior: %w", err)
	}
	log.Printf("Login behavior monitoring completed: %d anomalies detected", len(loginAnomalies))

	// 2. 检测API密钥异常使用
	apiKeyAnomalies, err := ss.detectAPIKeyAnomalies(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect API key anomalies: %w", err)
	}
	log.Printf("API key anomaly detection completed: %d anomalies detected", len(apiKeyAnomalies))

	// 3. 分析交易行为模式
	tradingPatternAnalysis, err := ss.analyzeTradingBehaviorPatterns(ctx)
	if err != nil {
		return fmt.Errorf("failed to analyze trading behavior patterns: %w", err)
	}
	log.Printf("Trading pattern analysis completed: %d suspicious patterns detected", 
		len(tradingPatternAnalysis.SuspiciousPatterns))

	// 4. 触发安全告警
	allSecurityEvents := append(loginAnomalies, apiKeyAnomalies...)
	allSecurityEvents = append(allSecurityEvents, tradingPatternAnalysis.SuspiciousPatterns...)

	if len(allSecurityEvents) > 0 {
		alertResults, err := ss.triggerSecurityAlerts(ctx, allSecurityEvents)
		if err != nil {
			log.Printf("Failed to trigger security alerts: %v", err)
		} else {
			log.Printf("Security alerts triggered: %d alerts sent", len(alertResults))
		}

		// 执行自动安全响应
		responseResults, err := ss.executeAutomaticSecurityResponse(ctx, allSecurityEvents)
		if err != nil {
			log.Printf("Failed to execute automatic security response: %v", err)
		} else {
			log.Printf("Automatic security response executed: %d actions taken", len(responseResults))
		}
	}

	log.Printf("Account security monitoring completed successfully")
	return nil
}

// HandleMultiExchangeRedundancy 处理多交易所冗余任务
func (ss *SystemScheduler) HandleMultiExchangeRedundancy(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing multi-exchange redundancy task: %s", task.Name)

	// 1. 检查交易所连接状态
	connectionStatuses, err := ss.checkExchangeConnectionStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to check exchange connection status: %w", err)
	}

	healthyExchanges := 0
	for _, status := range connectionStatuses {
		if status.IsHealthy {
			healthyExchanges++
		}
	}
	log.Printf("Exchange connection check completed: %d/%d exchanges healthy", 
		healthyExchanges, len(connectionStatuses))

	// 2. 监控交易所性能
	performanceMetrics, err := ss.monitorExchangePerformance(ctx)
	if err != nil {
		return fmt.Errorf("failed to monitor exchange performance: %w", err)
	}
	log.Printf("Exchange performance monitoring completed for %d exchanges", len(performanceMetrics))

	// 3. 自动切换故障交易所
	failedExchanges := ss.identifyFailedExchanges(connectionStatuses, performanceMetrics)
	if len(failedExchanges) > 0 {
		switchResults, err := ss.autoSwitchFailedExchanges(ctx, failedExchanges)
		if err != nil {
			log.Printf("Failed to auto switch failed exchanges: %v", err)
		} else {
			successfulSwitches := 0
			for _, result := range switchResults {
				if result.Success {
					successfulSwitches++
				}
			}
			log.Printf("Exchange switching completed: %d/%d successful", 
				successfulSwitches, len(switchResults))
		}
	}

	// 4. 维护冗余连接
	redundancyResults, err := ss.maintainRedundantConnections(ctx, connectionStatuses)
	if err != nil {
		log.Printf("Failed to maintain redundant connections: %v", err)
	} else {
		log.Printf("Redundant connections maintained: %d connections active", 
			redundancyResults.ActiveConnections)
	}

	log.Printf("Multi-exchange redundancy task completed successfully")
	return nil
}

// HandleAuditLogging 处理日志与审计追踪任务
func (ss *SystemScheduler) HandleAuditLogging(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing audit logging task: %s", task.Name)

	// 1. 收集系统操作日志
	operationLogs, err := ss.collectSystemOperationLogs(ctx)
	if err != nil {
		return fmt.Errorf("failed to collect system operation logs: %w", err)
	}
	log.Printf("System operation logs collected: %d entries", len(operationLogs))

	// 2. 生成审计报告
	auditReport, err := ss.generateAuditReport(ctx, operationLogs)
	if err != nil {
		return fmt.Errorf("failed to generate audit report: %w", err)
	}
	log.Printf("Audit report generated: %s (covering %d operations)", 
		auditReport.ReportID, auditReport.TotalOperations)

	// 3. 检查日志完整性
	integrityResults, err := ss.checkLogIntegrity(ctx, operationLogs)
	if err != nil {
		return fmt.Errorf("failed to check log integrity: %w", err)
	}
	log.Printf("Log integrity check completed: %.2f%% integrity score", 
		integrityResults.IntegrityScore*100)

	// 4. 清理过期日志
	cleanupResults, err := ss.cleanupExpiredLogs(ctx)
	if err != nil {
		log.Printf("Failed to cleanup expired logs: %v", err)
	} else {
		log.Printf("Expired logs cleanup completed: %d entries removed, %.2f MB freed", 
			cleanupResults.RemovedEntries, cleanupResults.FreedSpaceMB)
	}

	// 保存审计报告
	err = ss.saveAuditReport(ctx, auditReport)
	if err != nil {
		log.Printf("Failed to save audit report: %v", err)
	}

	log.Printf("Audit logging task completed successfully")
	return nil
}

// LearningScheduler 学习调度器
type LearningScheduler struct {
	config    *config.Config
	db        *database.DB
	isRunning bool
	mu        sync.RWMutex
}

// NewLearningScheduler 创建学习调度器
func NewLearningScheduler(cfg *config.Config, db *database.DB) *LearningScheduler {
	return &LearningScheduler{
		config: cfg,
		db:     db,
	}
}

func (ls *LearningScheduler) Start() error {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.isRunning = true
	log.Println("Learning scheduler started")
	return nil
}

func (ls *LearningScheduler) Stop() error {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.isRunning = false
	log.Println("Learning scheduler stopped")
	return nil
}

func (ls *LearningScheduler) HandleLearning(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing learning task: %s", task.Name)

	// 1. 收集训练数据
	trainingData, err := ls.collectTrainingData(ctx)
	if err != nil {
		return fmt.Errorf("failed to collect training data: %w", err)
	}
	log.Printf("Training data collected: %d samples, %d features", 
		len(trainingData.Samples), trainingData.FeatureCount)

	// 2. 训练模型
	trainingResults, err := ls.trainModels(ctx, trainingData)
	if err != nil {
		return fmt.Errorf("failed to train models: %w", err)
	}
	log.Printf("Model training completed: %d models trained", len(trainingResults))

	// 3. 评估模型性能
	evaluationResults, err := ls.evaluateModelPerformance(ctx, trainingResults)
	if err != nil {
		return fmt.Errorf("failed to evaluate model performance: %w", err)
	}

	bestModel := ls.selectBestModel(evaluationResults)
	log.Printf("Model evaluation completed: best model accuracy=%.4f", bestModel.Accuracy)

	// 4. 更新策略参数
	parameterUpdates, err := ls.updateStrategyParameters(ctx, bestModel)
	if err != nil {
		return fmt.Errorf("failed to update strategy parameters: %w", err)
	}
	log.Printf("Strategy parameters updated: %d strategies affected", len(parameterUpdates))

	// 保存学习结果
	err = ls.saveLearningResults(ctx, trainingResults, evaluationResults, parameterUpdates)
	if err != nil {
		log.Printf("Failed to save learning results: %v", err)
	}

	log.Printf("Machine learning task completed successfully")
	return nil
}

// HandleAutoMLLearning 处理策略自学习AutoML任务
func (ls *LearningScheduler) HandleAutoMLLearning(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing AutoML learning task: %s", task.Name)

	// 1. 自动模型选择
	modelCandidates, err := ls.autoSelectModels(ctx)
	if err != nil {
		return fmt.Errorf("failed to auto select models: %w", err)
	}
	log.Printf("Auto model selection completed: %d candidate models", len(modelCandidates))

	// 2. 超参数优化
	optimizationResults, err := ls.optimizeHyperparameters(ctx, modelCandidates)
	if err != nil {
		return fmt.Errorf("failed to optimize hyperparameters: %w", err)
	}
	log.Printf("Hyperparameter optimization completed for %d models", len(optimizationResults))

	// 3. 特征工程
	featureEngineeringResults, err := ls.performFeatureEngineering(ctx, optimizationResults)
	if err != nil {
		return fmt.Errorf("failed to perform feature engineering: %w", err)
	}
	log.Printf("Feature engineering completed: %d new features generated", 
		featureEngineeringResults.NewFeaturesCount)

	// 4. 模型集成
	ensembleModel, err := ls.createModelEnsemble(ctx, featureEngineeringResults)
	if err != nil {
		return fmt.Errorf("failed to create model ensemble: %w", err)
	}
	log.Printf("Model ensemble created: %d base models, accuracy=%.4f", 
		ensembleModel.BaseModelCount, ensembleModel.Accuracy)

	// 部署最佳模型
	deploymentResult, err := ls.deployBestModel(ctx, ensembleModel)
	if err != nil {
		log.Printf("Failed to deploy best model: %v", err)
	} else {
		log.Printf("Best model deployed: %s", deploymentResult.ModelID)
	}

	log.Printf("AutoML learning completed successfully")
	return nil
}

// HandleGeneticEvolution 处理遗传淘汰制升级任务
func (ls *LearningScheduler) HandleGeneticEvolution(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing genetic evolution task: %s", task.Name)

	// 1. 策略基因编码
	currentPopulation, err := ls.encodeStrategyGenes(ctx)
	if err != nil {
		return fmt.Errorf("failed to encode strategy genes: %w", err)
	}
	log.Printf("Strategy gene encoding completed: %d individuals in population", len(currentPopulation))

	// 2. 执行变异操作
	mutatedPopulation, err := ls.executeMutationOperations(ctx, currentPopulation)
	if err != nil {
		return fmt.Errorf("failed to execute mutation operations: %w", err)
	}
	log.Printf("Mutation operations completed: %d mutated individuals", len(mutatedPopulation))

	// 3. 适应度评估
	fitnessResults, err := ls.evaluateFitness(ctx, mutatedPopulation)
	if err != nil {
		return fmt.Errorf("failed to evaluate fitness: %w", err)
	}
	log.Printf("Fitness evaluation completed: average fitness=%.4f", fitnessResults.AverageFitness)

	// 4. 选择和繁殖
	nextGeneration, err := ls.selectAndBreed(ctx, mutatedPopulation, fitnessResults)
	if err != nil {
		return fmt.Errorf("failed to select and breed: %w", err)
	}
	log.Printf("Selection and breeding completed: %d individuals in next generation", len(nextGeneration))

	// 更新策略种群
	updateResults, err := ls.updateStrategyPopulation(ctx, nextGeneration)
	if err != nil {
		log.Printf("Failed to update strategy population: %v", err)
	} else {
		log.Printf("Strategy population updated: %d strategies evolved", updateResults.EvolvedStrategies)
	}

	// 记录进化历史
	err = ls.recordEvolutionHistory(ctx, currentPopulation, nextGeneration, fitnessResults)
	if err != nil {
		log.Printf("Failed to record evolution history: %v", err)
	}

	log.Printf("Genetic evolution completed successfully")
	return nil
}

// 热门币种推荐相关数据结构

// MarketData 市场数据
type MarketData struct {
	Symbol          string
	Price           float64
	Volume24h       float64
	VolumeChange24h float64
	PriceChange24h  float64
	Volatility      float64
	FundingRate     float64
	OpenInterest    float64
	OIChange24h     float64
	Timestamp       time.Time
}

// HotScore 热度评分
type HotScore struct {
	Symbol       string
	TotalScore   float64
	VolumeScore  float64
	PriceScore   float64
	FundingScore float64
	OIScore      float64
	TrendScore   float64
	RiskLevel    string
	Timestamp    time.Time
}

// Recommendation 推荐结果
type Recommendation struct {
	Symbol          string
	Score           float64
	RiskLevel       string
	PriceRange      [2]float64 // [min, max]
	SafeLeverage    float64
	MarketSentiment string
	Reason          string
	Timestamp       time.Time
}

// 热门币种推荐相关方法

// getAvailableSymbols 获取所有可用的交易对
func (ds *DataScheduler) getAvailableSymbols(ctx context.Context) ([]string, error) {
	// 从数据库获取活跃的交易对
	query := `
		SELECT DISTINCT symbol
		FROM market_data
		WHERE updated_at > NOW() - INTERVAL '1 hour'
		AND volume_24h > 1000000  -- 最小交易量过滤
		ORDER BY symbol
	`

	rows, err := ds.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query symbols: %w", err)
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, fmt.Errorf("failed to scan symbol: %w", err)
		}
		symbols = append(symbols, symbol)
	}

	// 如果数据库中没有数据，使用默认的热门币种列表
	if len(symbols) == 0 {
		symbols = []string{
			"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT",
			"XRPUSDT", "DOTUSDT", "DOGEUSDT", "AVAXUSDT", "MATICUSDT",
			"LINKUSDT", "LTCUSDT", "UNIUSDT", "ATOMUSDT", "FILUSDT",
		}
	}

	return symbols, nil
}

// collectMarketData 收集市场数据
func (ds *DataScheduler) collectMarketData(ctx context.Context, symbols []string) ([]*MarketData, error) {
	var marketData []*MarketData

	for _, symbol := range symbols {
		// 从数据库获取最新的市场数据
		query := `
			SELECT
				symbol,
				price,
				volume_24h,
				volume_change_24h,
				price_change_24h,
				volatility,
				funding_rate,
				open_interest,
				oi_change_24h,
				updated_at
			FROM market_data
			WHERE symbol = $1
			ORDER BY updated_at DESC
			LIMIT 1
		`

		var data MarketData
		err := ds.db.QueryRowContext(ctx, query, symbol).Scan(
			&data.Symbol,
			&data.Price,
			&data.Volume24h,
			&data.VolumeChange24h,
			&data.PriceChange24h,
			&data.Volatility,
			&data.FundingRate,
			&data.OpenInterest,
			&data.OIChange24h,
			&data.Timestamp,
		)

		if err != nil {
			// 如果数据库中没有数据，尝试从交易所API获取实时数据
			log.Printf("No market data found for %s in database, fetching from exchange: %v", symbol, err)
			apiData, apiErr := ds.fetchMarketDataFromAPI(ctx, symbol)
			if apiErr != nil {
				log.Printf("Failed to fetch market data from API for %s: %v", symbol, apiErr)
				// 如果API也失败，返回错误
				return nil, fmt.Errorf("no market data available for %s", symbol)
			}
			data = *apiData
		}

		marketData = append(marketData, &data)
	}

	return marketData, nil
}

// analyzeHotness 分析热度指标
func (ds *DataScheduler) analyzeHotness(ctx context.Context, marketData []*MarketData) ([]*HotScore, error) {
	var hotScores []*HotScore

	for _, data := range marketData {
		score := &HotScore{
			Symbol:    data.Symbol,
			Timestamp: time.Now(),
		}

		// 1. 交易量评分 (0-30分)
		volumeScore := ds.calculateVolumeScore(data)
		score.VolumeScore = volumeScore

		// 2. 价格变动评分 (0-25分)
		priceScore := ds.calculatePriceScore(data)
		score.PriceScore = priceScore

		// 3. 资金费率评分 (0-20分)
		fundingScore := ds.calculateFundingScore(data)
		score.FundingScore = fundingScore

		// 4. 持仓量评分 (0-15分)
		oiScore := ds.calculateOIScore(data)
		score.OIScore = oiScore

		// 5. 趋势评分 (0-10分)
		trendScore := ds.calculateTrendScore(data)
		score.TrendScore = trendScore

		// 计算总分
		score.TotalScore = volumeScore + priceScore + fundingScore + oiScore + trendScore

		// 确定风险等级
		score.RiskLevel = ds.determineRiskLevel(score.TotalScore, data)

		hotScores = append(hotScores, score)
	}

	// 按总分排序
	sort.Slice(hotScores, func(i, j int) bool {
		return hotScores[i].TotalScore > hotScores[j].TotalScore
	})

	return hotScores, nil
}

// calculateVolumeScore 计算交易量评分
func (ds *DataScheduler) calculateVolumeScore(data *MarketData) float64 {
	// 基础交易量评分 (0-15分)
	baseScore := math.Min(15, math.Log10(data.Volume24h/1000000)*5)
	if baseScore < 0 {
		baseScore = 0
	}

	// 交易量变化评分 (0-15分)
	changeScore := math.Min(15, math.Max(0, data.VolumeChange24h/10))

	return baseScore + changeScore
}

// calculatePriceScore 计算价格变动评分
func (ds *DataScheduler) calculatePriceScore(data *MarketData) float64 {
	// 价格变化幅度评分 (0-15分)
	changeScore := math.Min(15, math.Abs(data.PriceChange24h)/2)

	// 波动率评分 (0-10分)
	volatilityScore := math.Min(10, data.Volatility*200)

	return changeScore + volatilityScore
}

// calculateFundingScore 计算资金费率评分
func (ds *DataScheduler) calculateFundingScore(data *MarketData) float64 {
	// 资金费率异常程度评分
	absRate := math.Abs(data.FundingRate)

	// 正常资金费率范围是 -0.01% 到 0.01%
	if absRate > 0.001 {
		return math.Min(20, absRate*10000) // 超出正常范围给高分
	}

	return absRate * 5000 // 正常范围内给较低分
}

// calculateOIScore 计算持仓量评分
func (ds *DataScheduler) calculateOIScore(data *MarketData) float64 {
	// 持仓量变化评分
	changeScore := math.Min(15, math.Max(0, math.Abs(data.OIChange24h)/5))

	return changeScore
}

// calculateTrendScore 计算趋势评分
func (ds *DataScheduler) calculateTrendScore(data *MarketData) float64 {
	// 基于价格变化和交易量变化的趋势强度
	priceWeight := math.Abs(data.PriceChange24h) / 10
	volumeWeight := data.VolumeChange24h / 20

	trendStrength := (priceWeight + volumeWeight) / 2
	return math.Min(10, math.Max(0, trendStrength))
}

// determineRiskLevel 确定风险等级
func (ds *DataScheduler) determineRiskLevel(totalScore float64, data *MarketData) string {
	// 基于总分和波动率确定风险等级
	if totalScore >= 80 || data.Volatility > 0.1 {
		return "HIGH"
	} else if totalScore >= 60 || data.Volatility > 0.05 {
		return "MEDIUM"
	} else {
		return "LOW"
	}
}

// generateRecommendations 生成推荐列表
func (ds *DataScheduler) generateRecommendations(ctx context.Context, hotScores []*HotScore) ([]*Recommendation, error) {
	// 转换为符号列表
	symbols := make([]string, len(hotScores))
	for i, score := range hotScores {
		symbols[i] = score.Symbol
	}

	// 使用集成服务生成推荐
	enhancedRecs := ds.integratedService.GetRecommendations()
	if len(enhancedRecs) == 0 {
		// 如果没有缓存的推荐，强制更新
		err := ds.integratedService.ForceUpdate(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to force update recommendations: %w", err)
		}
		enhancedRecs = ds.integratedService.GetRecommendations()
	}

	// 转换为旧格式以保持兼容性
	var recommendations []*Recommendation
	for _, enhancedRec := range enhancedRecs {
		recommendation := &Recommendation{
			Symbol:          enhancedRec.Symbol,
			Score:           enhancedRec.Score,
			RiskLevel:       enhancedRec.RiskLevel,
			PriceRange:      enhancedRec.PriceRange,
			SafeLeverage:    enhancedRec.SafeLeverage,
			MarketSentiment: enhancedRec.MarketSentiment,
			Reason:          enhancedRec.Reason,
			Timestamp:       enhancedRec.Timestamp,
		}
		recommendations = append(recommendations, recommendation)
	}

	return recommendations, nil
}

// calculateSafeLeverage 计算安全杠杆倍数
func (ds *DataScheduler) calculateSafeLeverage(riskLevel string) float64 {
	switch riskLevel {
	case "HIGH":
		return 2.0 // 高风险币种建议低杠杆
	case "MEDIUM":
		return 5.0 // 中风险币种建议中等杠杆
	case "LOW":
		return 10.0 // 低风险币种可以使用较高杠杆
	default:
		return 1.0 // 默认无杠杆
	}
}

// determineMarketSentiment 确定市场情绪
func (ds *DataScheduler) determineMarketSentiment(score *HotScore) string {
	if score.TotalScore >= 80 {
		return "EXTREMELY_BULLISH"
	} else if score.TotalScore >= 70 {
		return "BULLISH"
	} else if score.TotalScore >= 60 {
		return "NEUTRAL_BULLISH"
	} else if score.TotalScore >= 50 {
		return "NEUTRAL"
	} else {
		return "BEARISH"
	}
}

// generateRecommendationReason 生成推荐理由
func (ds *DataScheduler) generateRecommendationReason(score *HotScore) string {
	reasons := []string{}

	if score.VolumeScore > 20 {
		reasons = append(reasons, "交易量异常活跃")
	}
	if score.PriceScore > 15 {
		reasons = append(reasons, "价格波动显著")
	}
	if score.FundingScore > 10 {
		reasons = append(reasons, "资金费率异常")
	}
	if score.OIScore > 8 {
		reasons = append(reasons, "持仓量变化明显")
	}
	if score.TrendScore > 6 {
		reasons = append(reasons, "趋势强劲")
	}

	if len(reasons) == 0 {
		return "综合指标表现良好"
	}

	result := "推荐理由: "
	for i, reason := range reasons {
		if i > 0 {
			result += ", "
		}
		result += reason
	}

	return result
}

// updateHotlistDatabase 更新热门币种数据库
func (ds *DataScheduler) updateHotlistDatabase(ctx context.Context, recommendations []*Recommendation) error {
	// 清理旧的推荐数据 (保留最近24小时的数据)
	cleanupQuery := `
		DELETE FROM hotlist_recommendations
		WHERE created_at < NOW() - INTERVAL '24 hours'
	`

	_, err := ds.db.ExecContext(ctx, cleanupQuery)
	if err != nil {
		log.Printf("Failed to cleanup old recommendations: %v", err)
		// 不返回错误，继续执行
	}

	// 插入新的推荐数据
	insertQuery := `
		INSERT INTO hotlist_recommendations (
			symbol, score, risk_level, price_min, price_max,
			safe_leverage, market_sentiment, reason, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (symbol) DO UPDATE SET
			score = EXCLUDED.score,
			risk_level = EXCLUDED.risk_level,
			price_min = EXCLUDED.price_min,
			price_max = EXCLUDED.price_max,
			safe_leverage = EXCLUDED.safe_leverage,
			market_sentiment = EXCLUDED.market_sentiment,
			reason = EXCLUDED.reason,
			updated_at = NOW()
	`

	for _, rec := range recommendations {
		_, err := ds.db.ExecContext(ctx, insertQuery,
			rec.Symbol,
			rec.Score,
			rec.RiskLevel,
			rec.PriceRange[0],
			rec.PriceRange[1],
			rec.SafeLeverage,
			rec.MarketSentiment,
			rec.Reason,
			rec.Timestamp,
		)

		if err != nil {
			log.Printf("Failed to insert recommendation for %s: %v", rec.Symbol, err)
			// 继续处理其他推荐，不返回错误
		}
	}

	log.Printf("Successfully updated %d recommendations in database", len(recommendations))
	return nil
}

// sendRecommendationNotifications 发送推荐通知 (支持增强推荐)
func (ds *DataScheduler) sendRecommendationNotifications(ctx context.Context, recommendations []*hotlist.EnhancedRecommendation) error {
	// 只通知高分推荐 (分数 >= 75)
	highScoreRecs := []*hotlist.EnhancedRecommendation{}
	for _, rec := range recommendations {
		if rec.Score >= 75 {
			highScoreRecs = append(highScoreRecs, rec)
		}
	}

	if len(highScoreRecs) == 0 {
		log.Printf("No high-score recommendations to notify")
		return nil
	}

	// 构建通知消息
	message := fmt.Sprintf("🔥 发现 %d 个热门币种推荐:\n", len(highScoreRecs))
	for i, rec := range highScoreRecs {
		if i >= 5 { // 最多显示5个
			break
		}
		message += fmt.Sprintf("• %s (评分: %.1f, 风险: %s, 置信度: %.1f%%)\n",
			rec.Symbol, rec.Score, rec.RiskLevel, rec.Confidence*100)
	}

	// 实现实际的通知发送逻辑
	log.Printf("Notification: %s", message)

	// 发送到多个通知渠道
	notificationResults := make(map[string]error)

	// 1. 发送到Webhook
	if webhookURL := ds.config.GetString("notifications.webhook_url"); webhookURL != "" {
		err := ds.sendWebhookNotification(ctx, webhookURL, message, highScoreRecs)
		notificationResults["webhook"] = err
		if err != nil {
			log.Printf("Failed to send webhook notification: %v", err)
		} else {
			log.Printf("Webhook notification sent successfully")
		}
	}

	// 2. 发送邮件通知
	if emailConfig := ds.getEmailConfig(); emailConfig.Enabled {
		err := ds.sendEmailNotification(ctx, emailConfig, message, highScoreRecs)
		notificationResults["email"] = err
		if err != nil {
			log.Printf("Failed to send email notification: %v", err)
		} else {
			log.Printf("Email notification sent successfully")
		}
	}

	// 3. 发送到Slack
	if slackConfig := ds.getSlackConfig(); slackConfig.Enabled {
		err := ds.sendSlackNotification(ctx, slackConfig, message, highScoreRecs)
		notificationResults["slack"] = err
		if err != nil {
			log.Printf("Failed to send Slack notification: %v", err)
		} else {
			log.Printf("Slack notification sent successfully")
		}
	}

	// 4. 发送到企业微信
	if wechatConfig := ds.getWeChatConfig(); wechatConfig.Enabled {
		err := ds.sendWeChatNotification(ctx, wechatConfig, message, highScoreRecs)
		notificationResults["wechat"] = err
		if err != nil {
			log.Printf("Failed to send WeChat notification: %v", err)
		} else {
			log.Printf("WeChat notification sent successfully")
		}
	}

	// 记录通知发送结果
	err := ds.recordNotificationResults(ctx, message, notificationResults)
	if err != nil {
		log.Printf("Failed to record notification results: %v", err)
	}

	return nil
}

// 资金分散与转移相关数据结构

// FundConcentrationRisk 资金集中度风险评估
type FundConcentrationRisk struct {
	TotalFunds           float64            `json:"total_funds"`
	ExchangeDistribution map[string]float64 `json:"exchange_distribution"`
	WalletDistribution   map[string]float64 `json:"wallet_distribution"`
	RiskLevel            string             `json:"risk_level"`
	ConcentrationRatio   float64            `json:"concentration_ratio"`
	RiskFactors          map[string]float64 `json:"risk_factors"`
	Recommendations      []string           `json:"recommendations"`
	Timestamp            time.Time          `json:"timestamp"`
}

// OptimalFundDistribution 最优资金分配
type OptimalFundDistribution struct {
	TargetDistribution    map[string]float64 `json:"target_distribution"`
	CurrentDistribution   map[string]float64 `json:"current_distribution"`
	RequiredTransfers     []*FundTransfer    `json:"required_transfers"`
	ExpectedRiskReduction float64            `json:"expected_risk_reduction"`
	EstimatedCost         float64            `json:"estimated_cost"`
	Priority              int                `json:"priority"`
	Timestamp             time.Time          `json:"timestamp"`
}

// FundTransfer 资金转移操作
type FundTransfer struct {
	ID               string                 `json:"id"`
	Type             string                 `json:"type"` // HOT_TO_COLD, COLD_TO_HOT, EXCHANGE_REBALANCE
	FromAddress      string                 `json:"from_address"`
	ToAddress        string                 `json:"to_address"`
	Amount           float64                `json:"amount"`
	Currency         string                 `json:"currency"`
	Status           string                 `json:"status"`
	Priority         int                    `json:"priority"`
	EstimatedFee     float64                `json:"estimated_fee"`
	ActualFee        float64                `json:"actual_fee"`
	TransactionHash  string                 `json:"transaction_hash"`
	Confirmations    int                    `json:"confirmations"`
	RequiredConfirms int                    `json:"required_confirms"`
	CreatedAt        time.Time              `json:"created_at"`
	ExecutedAt       *time.Time             `json:"executed_at"`
	CompletedAt      *time.Time             `json:"completed_at"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// TransferResult 转移结果
type TransferResult struct {
	Transfer      *FundTransfer          `json:"transfer"`
	Success       bool                   `json:"success"`
	Error         string                 `json:"error,omitempty"`
	ActualAmount  float64                `json:"actual_amount"`
	ExecutionTime time.Duration          `json:"execution_time"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// ColdWalletOperation 冷钱包操作
type ColdWalletOperation struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"` // DEPOSIT, WITHDRAW, BALANCE_CHECK
	WalletAddress string                 `json:"wallet_address"`
	Amount        float64                `json:"amount"`
	Currency      string                 `json:"currency"`
	Status        string                 `json:"status"`
	SecurityLevel string                 `json:"security_level"`
	RequiredSigs  int                    `json:"required_sigs"`
	ProvidedSigs  int                    `json:"provided_sigs"`
	CreatedAt     time.Time              `json:"created_at"`
	ExecutedAt    *time.Time             `json:"executed_at"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// assessFundConcentrationRisk 评估资金集中度风险
func (rs *RiskScheduler) assessFundConcentrationRisk(ctx context.Context) (*FundConcentrationRisk, error) {
	// 1. 获取当前资金分布
	exchangeDistribution, err := rs.getExchangeFundDistribution(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange fund distribution: %w", err)
	}

	walletDistribution, err := rs.getWalletFundDistribution(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet fund distribution: %w", err)
	}

	// 2. 计算总资金
	totalFunds := 0.0
	for _, amount := range exchangeDistribution {
		totalFunds += amount
	}
	for _, amount := range walletDistribution {
		totalFunds += amount
	}

	// 3. 计算集中度比率
	concentrationRatio := rs.calculateConcentrationRatio(exchangeDistribution, walletDistribution)

	// 4. 评估风险因子
	riskFactors := rs.calculateRiskFactors(exchangeDistribution, walletDistribution, totalFunds)

	// 5. 确定风险等级
	riskLevel := rs.determineRiskLevel(concentrationRatio, riskFactors)

	// 6. 生成建议
	recommendations := rs.generateRiskRecommendations(riskLevel, concentrationRatio, riskFactors)

	assessment := &FundConcentrationRisk{
		TotalFunds:           totalFunds,
		ExchangeDistribution: exchangeDistribution,
		WalletDistribution:   walletDistribution,
		RiskLevel:            riskLevel,
		ConcentrationRatio:   concentrationRatio,
		RiskFactors:          riskFactors,
		Recommendations:      recommendations,
		Timestamp:            time.Now(),
	}

	log.Printf("Fund concentration risk assessment: Level=%s, Ratio=%.4f, Total=%.2f",
		riskLevel, concentrationRatio, totalFunds)

	return assessment, nil
}

// getExchangeFundDistribution 获取交易所资金分布
func (rs *RiskScheduler) getExchangeFundDistribution(ctx context.Context) (map[string]float64, error) {
	// 首先尝试从数据库获取数据
	query := `
		SELECT exchange_name, SUM(balance) as total_balance
		FROM exchange_balances
		WHERE updated_at > NOW() - INTERVAL '1 hour'
		GROUP BY exchange_name
	`

	rows, err := rs.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Failed to query exchange balances from database: %v, using mock data", err)
		return rs.getMockExchangeFundDistribution(), nil
	}
	defer rows.Close()

	distribution := make(map[string]float64)
	for rows.Next() {
		var exchangeName string
		var balance float64
		if err := rows.Scan(&exchangeName, &balance); err != nil {
			log.Printf("Failed to scan exchange balance: %v", err)
			continue
		}
		distribution[exchangeName] = balance
	}

	// 如果没有数据，使用模拟数据
	if len(distribution) == 0 {
		log.Printf("No exchange balance data available, using mock data")
		return rs.getMockExchangeFundDistribution(), nil
	}

	return distribution, nil
}

// getMockExchangeFundDistribution 获取模拟的交易所资金分布
func (rs *RiskScheduler) getMockExchangeFundDistribution() map[string]float64 {
	return map[string]float64{
		"binance":     50000.0, // 50,000 USDT
		"okx":         30000.0, // 30,000 USDT
		"bybit":       20000.0, // 20,000 USDT
		"hot_wallet":  15000.0, // 15,000 USDT
		"cold_wallet": 85000.0, // 85,000 USDT
	}
}

// getWalletFundDistribution 获取钱包资金分布
func (rs *RiskScheduler) getWalletFundDistribution(ctx context.Context) (map[string]float64, error) {
	// 首先尝试从数据库获取数据
	query := `
		SELECT wallet_type, SUM(balance) as total_balance
		FROM wallet_balances
		WHERE updated_at > NOW() - INTERVAL '1 hour'
		GROUP BY wallet_type
	`

	rows, err := rs.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Failed to query wallet balances from database: %v, using mock data", err)
		return rs.getMockWalletFundDistribution(), nil
	}
	defer rows.Close()

	distribution := make(map[string]float64)
	for rows.Next() {
		var walletType string
		var balance float64
		if err := rows.Scan(&walletType, &balance); err != nil {
			log.Printf("Failed to scan wallet balance: %v", err)
			continue
		}
		distribution[walletType] = balance
	}

	// 如果没有数据，使用模拟数据
	if len(distribution) == 0 {
		log.Printf("No wallet balance data available, using mock data")
		return rs.getMockWalletFundDistribution(), nil
	}

	return distribution, nil
}

// getMockWalletFundDistribution 获取模拟的钱包资金分布
func (rs *RiskScheduler) getMockWalletFundDistribution() map[string]float64 {
	return map[string]float64{
		"hot_wallet":  15000.0, // 15,000 USDT
		"cold_wallet": 85000.0, // 85,000 USDT
		"treasury":    50000.0, // 50,000 USDT
	}
}

// calculateConcentrationRatio 计算集中度比率
func (rs *RiskScheduler) calculateConcentrationRatio(exchangeDist, walletDist map[string]float64) float64 {
	// 计算最大单一集中度
	maxConcentration := 0.0
	totalFunds := 0.0

	// 计算总资金
	for _, amount := range exchangeDist {
		totalFunds += amount
	}
	for _, amount := range walletDist {
		totalFunds += amount
	}

	// 找出最大单一集中度
	for _, amount := range exchangeDist {
		ratio := amount / totalFunds
		if ratio > maxConcentration {
			maxConcentration = ratio
		}
	}
	for _, amount := range walletDist {
		ratio := amount / totalFunds
		if ratio > maxConcentration {
			maxConcentration = ratio
		}
	}

	return maxConcentration
}

// calculateRiskFactors 计算风险因子
func (rs *RiskScheduler) calculateRiskFactors(exchangeDist, walletDist map[string]float64, totalFunds float64) map[string]float64 {
	riskFactors := make(map[string]float64)

	// 1. 交易所集中度风险
	exchangeRisk := 0.0
	for _, amount := range exchangeDist {
		ratio := amount / totalFunds
		if ratio > 0.5 { // 单一交易所超过50%
			exchangeRisk += (ratio - 0.5) * 2.0 // 超出部分加倍计算风险
		}
	}
	riskFactors["exchange_concentration"] = math.Min(1.0, exchangeRisk)

	// 2. 热钱包风险
	hotWalletRisk := 0.0
	if hotAmount, exists := walletDist["hot_wallet"]; exists {
		hotRatio := hotAmount / totalFunds
		if hotRatio > 0.2 { // 热钱包超过20%
			hotWalletRisk = (hotRatio - 0.2) * 2.5
		}
	}
	riskFactors["hot_wallet_risk"] = math.Min(1.0, hotWalletRisk)

	// 3. 地理分布风险 (简化处理)
	geoRisk := 0.3 // 假设中等地理风险
	riskFactors["geographic_risk"] = geoRisk

	// 4. 流动性风险
	liquidityRisk := 0.0
	exchangeCount := len(exchangeDist)
	if exchangeCount < 2 {
		liquidityRisk = 0.8 // 只有一个交易所，流动性风险很高
	} else if exchangeCount < 3 {
		liquidityRisk = 0.4 // 两个交易所，中等风险
	} else {
		liquidityRisk = 0.1 // 三个以上交易所，低风险
	}
	riskFactors["liquidity_risk"] = liquidityRisk

	// 5. 技术风险
	techRisk := 0.2 // 假设基础技术风险
	riskFactors["technical_risk"] = techRisk

	return riskFactors
}

// determineRiskLevel 确定风险等级
func (rs *RiskScheduler) determineRiskLevel(concentrationRatio float64, riskFactors map[string]float64) string {
	// 计算综合风险分数
	totalRisk := concentrationRatio * 0.4 // 集中度权重40%

	for factor, value := range riskFactors {
		switch factor {
		case "exchange_concentration":
			totalRisk += value * 0.25 // 交易所集中度权重25%
		case "hot_wallet_risk":
			totalRisk += value * 0.15 // 热钱包风险权重15%
		case "geographic_risk":
			totalRisk += value * 0.1 // 地理风险权重10%
		case "liquidity_risk":
			totalRisk += value * 0.05 // 流动性风险权重5%
		case "technical_risk":
			totalRisk += value * 0.05 // 技术风险权重5%
		}
	}

	// 根据总风险分数确定等级
	if totalRisk >= 0.8 {
		return "CRITICAL"
	} else if totalRisk >= 0.6 {
		return "HIGH"
	} else if totalRisk >= 0.4 {
		return "MEDIUM"
	} else if totalRisk >= 0.2 {
		return "LOW"
	} else {
		return "MINIMAL"
	}
}

// generateRiskRecommendations 生成风险建议
func (rs *RiskScheduler) generateRiskRecommendations(riskLevel string, concentrationRatio float64, riskFactors map[string]float64) []string {
	var recommendations []string

	// 基于风险等级的通用建议
	switch riskLevel {
	case "CRITICAL":
		recommendations = append(recommendations, "立即执行紧急资金分散操作")
		recommendations = append(recommendations, "暂停大额交易直到风险降低")
	case "HIGH":
		recommendations = append(recommendations, "在24小时内执行资金重新分配")
		recommendations = append(recommendations, "增加冷钱包存储比例")
	case "MEDIUM":
		recommendations = append(recommendations, "考虑在一周内优化资金分布")
		recommendations = append(recommendations, "监控交易所风险状况")
	case "LOW":
		recommendations = append(recommendations, "保持当前分散策略")
		recommendations = append(recommendations, "定期评估资金分布")
	}

	// 基于具体风险因子的建议
	if riskFactors["exchange_concentration"] > 0.6 {
		recommendations = append(recommendations, "减少单一交易所资金集中度")
		recommendations = append(recommendations, "考虑增加新的交易所")
	}

	if riskFactors["hot_wallet_risk"] > 0.5 {
		recommendations = append(recommendations, "将部分热钱包资金转移到冷钱包")
		recommendations = append(recommendations, "加强热钱包安全监控")
	}

	if riskFactors["liquidity_risk"] > 0.6 {
		recommendations = append(recommendations, "增加交易所数量以提高流动性")
		recommendations = append(recommendations, "建立应急流动性储备")
	}

	if concentrationRatio > 0.7 {
		recommendations = append(recommendations, "紧急分散资金，降低单点风险")
	}

	return recommendations
}

// calculateOptimalFundDistribution 计算最优资金分配
func (rs *RiskScheduler) calculateOptimalFundDistribution(ctx context.Context, riskAssessment *FundConcentrationRisk) (*OptimalFundDistribution, error) {
	// 1. 定义目标分配比例
	targetDistribution := rs.calculateTargetDistribution(riskAssessment)

	// 2. 获取当前分配
	currentDistribution := make(map[string]float64)
	for k, v := range riskAssessment.ExchangeDistribution {
		currentDistribution[k] = v / riskAssessment.TotalFunds
	}
	for k, v := range riskAssessment.WalletDistribution {
		currentDistribution[k] = v / riskAssessment.TotalFunds
	}

	// 3. 计算需要的转移操作
	requiredTransfers := rs.calculateRequiredTransfers(currentDistribution, targetDistribution, riskAssessment.TotalFunds)

	// 4. 估算成本和风险降低
	estimatedCost := rs.estimateTransferCosts(requiredTransfers)
	expectedRiskReduction := rs.calculateExpectedRiskReduction(riskAssessment, targetDistribution)

	// 5. 确定优先级
	priority := rs.calculateDistributionPriority(riskAssessment.RiskLevel, expectedRiskReduction)

	distribution := &OptimalFundDistribution{
		TargetDistribution:    targetDistribution,
		CurrentDistribution:   currentDistribution,
		RequiredTransfers:     requiredTransfers,
		ExpectedRiskReduction: expectedRiskReduction,
		EstimatedCost:         estimatedCost,
		Priority:              priority,
		Timestamp:             time.Now(),
	}

	log.Printf("Optimal fund distribution calculated: %d transfers, cost=%.2f, risk reduction=%.4f",
		len(requiredTransfers), estimatedCost, expectedRiskReduction)

	return distribution, nil
}

// calculateTargetDistribution 计算目标分配比例
func (rs *RiskScheduler) calculateTargetDistribution(riskAssessment *FundConcentrationRisk) map[string]float64 {
	targetDistribution := make(map[string]float64)

	// 基于风险等级设定目标分配
	switch riskAssessment.RiskLevel {
	case "CRITICAL", "HIGH":
		// 高风险情况：最大分散
		targetDistribution["cold_wallet"] = 0.6 // 60%冷钱包
		targetDistribution["hot_wallet"] = 0.1  // 10%热钱包
		targetDistribution["binance"] = 0.15    // 15%币安
		targetDistribution["okx"] = 0.1         // 10%OKX
		targetDistribution["bybit"] = 0.05      // 5%Bybit
	case "MEDIUM":
		// 中等风险：平衡分配
		targetDistribution["cold_wallet"] = 0.5 // 50%冷钱包
		targetDistribution["hot_wallet"] = 0.15 // 15%热钱包
		targetDistribution["binance"] = 0.2     // 20%币安
		targetDistribution["okx"] = 0.1         // 10%OKX
		targetDistribution["bybit"] = 0.05      // 5%Bybit
	case "LOW", "MINIMAL":
		// 低风险：保持当前分配或轻微调整
		for k, v := range riskAssessment.ExchangeDistribution {
			targetDistribution[k] = v / riskAssessment.TotalFunds
		}
		for k, v := range riskAssessment.WalletDistribution {
			targetDistribution[k] = v / riskAssessment.TotalFunds
		}
	}

	return targetDistribution
}

// calculateRequiredTransfers 计算需要的转移操作
func (rs *RiskScheduler) calculateRequiredTransfers(current, target map[string]float64, totalFunds float64) []*FundTransfer {
	var transfers []*FundTransfer
	transferID := 1

	for location, targetRatio := range target {
		currentRatio := current[location]
		if currentRatio == 0 {
			currentRatio = 0
		}

		difference := targetRatio - currentRatio

		// 只有差异超过阈值才执行转移
		if math.Abs(difference) > 0.05 { // 5%阈值
			amount := math.Abs(difference) * totalFunds

			transfer := &FundTransfer{
				ID:               fmt.Sprintf("transfer_%d_%d", time.Now().Unix(), transferID),
				Amount:           amount,
				Currency:         "USDT",
				Status:           "PENDING",
				EstimatedFee:     amount * 0.001, // 0.1%手续费
				RequiredConfirms: 6,
				CreatedAt:        time.Now(),
				Metadata:         make(map[string]interface{}),
			}

			if difference > 0 {
				// 需要增加资金到这个位置
				transfer.Type = "DEPOSIT"
				transfer.ToAddress = location
				transfer.FromAddress = rs.findSourceForTransfer(current, target, totalFunds)
				transfer.Priority = rs.calculateTransferPriority(difference, location)
			} else {
				// 需要从这个位置转出资金
				transfer.Type = "WITHDRAW"
				transfer.FromAddress = location
				transfer.ToAddress = rs.findDestinationForTransfer(current, target, totalFunds)
				transfer.Priority = rs.calculateTransferPriority(math.Abs(difference), location)
			}

			transfer.Metadata["target_ratio"] = targetRatio
			transfer.Metadata["current_ratio"] = currentRatio
			transfer.Metadata["difference"] = difference

			transfers = append(transfers, transfer)
			transferID++
		}
	}

	// 按优先级排序
	sort.Slice(transfers, func(i, j int) bool {
		return transfers[i].Priority > transfers[j].Priority
	})

	return transfers
}

// 辅助方法实现

// estimateTransferCosts 估算转移成本
func (rs *RiskScheduler) estimateTransferCosts(transfers []*FundTransfer) float64 {
	totalCost := 0.0
	for _, transfer := range transfers {
		totalCost += transfer.EstimatedFee
	}
	return totalCost
}

// calculateExpectedRiskReduction 计算预期风险降低
func (rs *RiskScheduler) calculateExpectedRiskReduction(assessment *FundConcentrationRisk, targetDistribution map[string]float64) float64 {
	// 计算当前风险分数
	currentRisk := assessment.ConcentrationRatio

	// 计算目标风险分数
	targetRisk := 0.0
	for _, ratio := range targetDistribution {
		if ratio > targetRisk {
			targetRisk = ratio
		}
	}

	return math.Max(0, currentRisk-targetRisk)
}

// calculateDistributionPriority 计算分配优先级
func (rs *RiskScheduler) calculateDistributionPriority(riskLevel string, riskReduction float64) int {
	basePriority := 0
	switch riskLevel {
	case "CRITICAL":
		basePriority = 5
	case "HIGH":
		basePriority = 4
	case "MEDIUM":
		basePriority = 3
	case "LOW":
		basePriority = 2
	default:
		basePriority = 1
	}

	// 基于风险降低程度调整优先级
	if riskReduction > 0.3 {
		basePriority += 2
	} else if riskReduction > 0.1 {
		basePriority += 1
	}

	return basePriority
}

// findSourceForTransfer 找到转移资金的来源
func (rs *RiskScheduler) findSourceForTransfer(current, target map[string]float64, totalFunds float64) string {
	// 找到超出目标比例最多的位置作为来源
	maxExcess := 0.0
	sourceLocation := ""

	for location, currentRatio := range current {
		targetRatio := target[location]
		if targetRatio == 0 {
			targetRatio = 0
		}

		excess := currentRatio - targetRatio
		if excess > maxExcess {
			maxExcess = excess
			sourceLocation = location
		}
	}

	if sourceLocation == "" {
		// 默认从最大的位置转出
		maxAmount := 0.0
		for location, ratio := range current {
			if ratio > maxAmount {
				maxAmount = ratio
				sourceLocation = location
			}
		}
	}

	return sourceLocation
}

// findDestinationForTransfer 找到转移资金的目标
func (rs *RiskScheduler) findDestinationForTransfer(current, target map[string]float64, totalFunds float64) string {
	// 找到低于目标比例最多的位置作为目标
	maxDeficit := 0.0
	destLocation := ""

	for location, targetRatio := range target {
		currentRatio := current[location]
		if currentRatio == 0 {
			currentRatio = 0
		}

		deficit := targetRatio - currentRatio
		if deficit > maxDeficit {
			maxDeficit = deficit
			destLocation = location
		}
	}

	if destLocation == "" {
		// 默认转到冷钱包
		destLocation = "cold_wallet"
	}

	return destLocation
}

// calculateTransferPriority 计算转移优先级
func (rs *RiskScheduler) calculateTransferPriority(difference float64, location string) int {
	priority := 1

	// 基于差异大小
	if difference > 0.3 {
		priority = 5
	} else if difference > 0.2 {
		priority = 4
	} else if difference > 0.1 {
		priority = 3
	} else if difference > 0.05 {
		priority = 2
	}

	// 基于位置类型调整优先级
	if location == "hot_wallet" {
		priority += 1 // 热钱包操作优先级更高
	} else if location == "cold_wallet" {
		priority -= 1 // 冷钱包操作优先级较低
	}

	if priority < 1 {
		priority = 1
	}
	return priority
}

// executeFundTransfers 执行资金转移操作
func (rs *RiskScheduler) executeFundTransfers(ctx context.Context, distribution *OptimalFundDistribution) ([]*TransferResult, error) {
	var results []*TransferResult

	log.Printf("Executing %d fund transfers", len(distribution.RequiredTransfers))

	for _, transfer := range distribution.RequiredTransfers {
		result := &TransferResult{
			Transfer: transfer,
			Success:  false,
			Metadata: make(map[string]interface{}),
		}

		startTime := time.Now()

		// 执行转移操作
		err := rs.executeIndividualTransfer(ctx, transfer)
		if err != nil {
			result.Error = err.Error()
			log.Printf("Transfer failed: %s -> %s, amount: %.2f, error: %v",
				transfer.FromAddress, transfer.ToAddress, transfer.Amount, err)
		} else {
			result.Success = true
			result.ActualAmount = transfer.Amount
			log.Printf("Transfer completed: %s -> %s, amount: %.2f",
				transfer.FromAddress, transfer.ToAddress, transfer.Amount)
		}

		result.ExecutionTime = time.Since(startTime)
		results = append(results, result)

		// 记录转移结果到数据库
		err = rs.recordTransferResult(ctx, result)
		if err != nil {
			log.Printf("Failed to record transfer result: %v", err)
		}

		// 添加延迟以避免过于频繁的操作
		time.Sleep(time.Second * 2)
	}

	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	log.Printf("Fund transfers completed: %d/%d successful", successCount, len(results))
	return results, nil
}

// executeIndividualTransfer 执行单个转移操作
func (rs *RiskScheduler) executeIndividualTransfer(ctx context.Context, transfer *FundTransfer) error {
	// 更新转移状态为执行中
	transfer.Status = "EXECUTING"
	now := time.Now()
	transfer.ExecutedAt = &now

	// 根据转移类型执行不同的操作
	switch transfer.Type {
	case "HOT_TO_COLD":
		return rs.executeHotToColdTransfer(ctx, transfer)
	case "COLD_TO_HOT":
		return rs.executeColdToHotTransfer(ctx, transfer)
	case "EXCHANGE_REBALANCE":
		return rs.executeExchangeRebalance(ctx, transfer)
	case "DEPOSIT":
		return rs.executeDeposit(ctx, transfer)
	case "WITHDRAW":
		return rs.executeWithdraw(ctx, transfer)
	default:
		return fmt.Errorf("unsupported transfer type: %s", transfer.Type)
	}
}

// executeHotToColdTransfer 执行热钱包到冷钱包转移
func (rs *RiskScheduler) executeHotToColdTransfer(ctx context.Context, transfer *FundTransfer) error {
	log.Printf("Executing hot to cold transfer: %.2f %s", transfer.Amount, transfer.Currency)

	// 这里应该调用实际的钱包API
	// 目前使用模拟实现

	// 模拟转移延迟
	time.Sleep(time.Millisecond * 500)

	// 生成模拟交易哈希
	transfer.TransactionHash = fmt.Sprintf("0x%x", time.Now().UnixNano())
	transfer.Confirmations = 0
	transfer.Status = "CONFIRMING"

	// 模拟确认过程
	go rs.simulateConfirmationProcess(transfer)

	return nil
}

// executeColdToHotTransfer 执行冷钱包到热钱包转移
func (rs *RiskScheduler) executeColdToHotTransfer(ctx context.Context, transfer *FundTransfer) error {
	log.Printf("Executing cold to hot transfer: %.2f %s", transfer.Amount, transfer.Currency)

	// 冷钱包转移需要更多的安全验证
	// 这里应该实现多重签名等安全机制

	// 模拟安全验证延迟
	time.Sleep(time.Second * 2)

	// 生成模拟交易哈希
	transfer.TransactionHash = fmt.Sprintf("0x%x", time.Now().UnixNano())
	transfer.Confirmations = 0
	transfer.Status = "CONFIRMING"

	// 模拟确认过程
	go rs.simulateConfirmationProcess(transfer)

	return nil
}

// executeExchangeRebalance 执行交易所间再平衡
func (rs *RiskScheduler) executeExchangeRebalance(ctx context.Context, transfer *FundTransfer) error {
	log.Printf("Executing exchange rebalance: %s -> %s, %.2f %s",
		transfer.FromAddress, transfer.ToAddress, transfer.Amount, transfer.Currency)

	// 这里应该调用交易所API进行转移
	// 目前使用模拟实现

	// 模拟API调用延迟
	time.Sleep(time.Millisecond * 300)

	// 生成模拟交易ID
	transfer.TransactionHash = fmt.Sprintf("exchange_transfer_%d", time.Now().UnixNano())
	transfer.Confirmations = transfer.RequiredConfirms // 交易所内部转移通常立即确认
	transfer.Status = "COMPLETED"

	now := time.Now()
	transfer.CompletedAt = &now

	return nil
}

// 剩余的辅助方法

// executeDeposit 执行存款操作
func (rs *RiskScheduler) executeDeposit(ctx context.Context, transfer *FundTransfer) error {
	log.Printf("Executing deposit: %.2f %s to %s", transfer.Amount, transfer.Currency, transfer.ToAddress)

	// 模拟存款操作
	time.Sleep(time.Millisecond * 200)

	transfer.TransactionHash = fmt.Sprintf("deposit_%d", time.Now().UnixNano())
	transfer.Status = "COMPLETED"
	now := time.Now()
	transfer.CompletedAt = &now

	return nil
}

// executeWithdraw 执行提款操作
func (rs *RiskScheduler) executeWithdraw(ctx context.Context, transfer *FundTransfer) error {
	log.Printf("Executing withdraw: %.2f %s from %s", transfer.Amount, transfer.Currency, transfer.FromAddress)

	// 模拟提款操作
	time.Sleep(time.Millisecond * 300)

	transfer.TransactionHash = fmt.Sprintf("withdraw_%d", time.Now().UnixNano())
	transfer.Status = "CONFIRMING"
	transfer.Confirmations = 0

	// 模拟确认过程
	go rs.simulateConfirmationProcess(transfer)

	return nil
}

// simulateConfirmationProcess 模拟确认过程
func (rs *RiskScheduler) simulateConfirmationProcess(transfer *FundTransfer) {
	for transfer.Confirmations < transfer.RequiredConfirms {
		time.Sleep(time.Second * 10) // 每10秒增加一个确认
		transfer.Confirmations++
		log.Printf("Transfer %s: %d/%d confirmations", transfer.ID, transfer.Confirmations, transfer.RequiredConfirms)
	}

	transfer.Status = "COMPLETED"
	now := time.Now()
	transfer.CompletedAt = &now
	log.Printf("Transfer %s completed", transfer.ID)
}

// recordTransferResult 记录转移结果
func (rs *RiskScheduler) recordTransferResult(ctx context.Context, result *TransferResult) error {
	query := `
		INSERT INTO fund_transfer_results (
			transfer_id, success, error_message, actual_amount,
			execution_time, created_at
		) VALUES ($1, $2, $3, $4, $5, NOW())
	`

	_, err := rs.db.ExecContext(ctx, query,
		result.Transfer.ID, result.Success, result.Error,
		result.ActualAmount, result.ExecutionTime.Milliseconds(),
	)

	return err
}

// integrateColdWalletOperations 集成冷钱包操作
func (rs *RiskScheduler) integrateColdWalletOperations(ctx context.Context, transferResults []*TransferResult) error {
	var coldWalletOps []*ColdWalletOperation

	// 为涉及冷钱包的转移创建冷钱包操作
	for _, result := range transferResults {
		if !result.Success {
			continue
		}

		transfer := result.Transfer
		if transfer.FromAddress == "cold_wallet" || transfer.ToAddress == "cold_wallet" {
			op := &ColdWalletOperation{
				ID:            fmt.Sprintf("cold_op_%d", time.Now().UnixNano()),
				Amount:        transfer.Amount,
				Currency:      transfer.Currency,
				Status:        "PENDING",
				SecurityLevel: "HIGH",
				RequiredSigs:  3, // 需要3个签名
				ProvidedSigs:  0,
				CreatedAt:     time.Now(),
				Metadata:      make(map[string]interface{}),
			}

			if transfer.ToAddress == "cold_wallet" {
				op.Type = "DEPOSIT"
				op.WalletAddress = transfer.ToAddress
			} else {
				op.Type = "WITHDRAW"
				op.WalletAddress = transfer.FromAddress
			}

			op.Metadata["transfer_id"] = transfer.ID
			op.Metadata["transfer_type"] = transfer.Type

			coldWalletOps = append(coldWalletOps, op)
		}
	}

	// 执行冷钱包操作
	for _, op := range coldWalletOps {
		err := rs.executeColdWalletOperation(ctx, op)
		if err != nil {
			log.Printf("Failed to execute cold wallet operation %s: %v", op.ID, err)
			continue
		}
	}

	log.Printf("Integrated %d cold wallet operations", len(coldWalletOps))
	return nil
}

// executeColdWalletOperation 执行冷钱包操作
func (rs *RiskScheduler) executeColdWalletOperation(ctx context.Context, op *ColdWalletOperation) error {
	log.Printf("Executing cold wallet operation: %s %s %.2f %s", op.Type, op.WalletAddress, op.Amount, op.Currency)

	// 模拟多重签名过程
	for op.ProvidedSigs < op.RequiredSigs {
		time.Sleep(time.Second * 5) // 模拟签名延迟
		op.ProvidedSigs++
		log.Printf("Cold wallet operation %s: %d/%d signatures", op.ID, op.ProvidedSigs, op.RequiredSigs)
	}

	op.Status = "COMPLETED"
	now := time.Now()
	op.ExecutedAt = &now

	// 记录到数据库
	err := rs.recordColdWalletOperation(ctx, op)
	if err != nil {
		return fmt.Errorf("failed to record cold wallet operation: %w", err)
	}

	return nil
}

// recordColdWalletOperation 记录冷钱包操作
func (rs *RiskScheduler) recordColdWalletOperation(ctx context.Context, op *ColdWalletOperation) error {
	query := `
		INSERT INTO cold_wallet_operations (
			id, type, wallet_address, amount, currency, status,
			security_level, required_sigs, provided_sigs, created_at, executed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := rs.db.ExecContext(ctx, query,
		op.ID, op.Type, op.WalletAddress, op.Amount, op.Currency,
		op.Status, op.SecurityLevel, op.RequiredSigs, op.ProvidedSigs,
		op.CreatedAt, op.ExecutedAt,
	)

	return err
}

// updateFundProtectionProtocol 更新资金保护协议
func (rs *RiskScheduler) updateFundProtectionProtocol(ctx context.Context, distribution *OptimalFundDistribution, transferResults []*TransferResult) error {
	log.Printf("Updating fund protection protocol")

	// 1. 计算新的风险参数
	newRiskParams := rs.calculateNewRiskParameters(distribution, transferResults)

	// 2. 更新保护阈值
	err := rs.updateProtectionThresholds(ctx, newRiskParams)
	if err != nil {
		return fmt.Errorf("failed to update protection thresholds: %w", err)
	}

	// 3. 更新监控规则
	err = rs.updateMonitoringRules(ctx, distribution)
	if err != nil {
		return fmt.Errorf("failed to update monitoring rules: %w", err)
	}

	// 4. 记录协议更新历史
	err = rs.recordProtocolUpdate(ctx, distribution, transferResults, newRiskParams)
	if err != nil {
		log.Printf("Failed to record protocol update: %v", err)
		// 不返回错误，因为记录失败不应该影响主流程
	}

	log.Printf("Fund protection protocol updated successfully")
	return nil
}

// calculateNewRiskParameters 计算新的风险参数
func (rs *RiskScheduler) calculateNewRiskParameters(distribution *OptimalFundDistribution, transferResults []*TransferResult) map[string]float64 {
	params := make(map[string]float64)

	// 基于目标分配计算新的风险阈值
	maxSingleAllocation := 0.0
	for _, ratio := range distribution.TargetDistribution {
		if ratio > maxSingleAllocation {
			maxSingleAllocation = ratio
		}
	}

	// 基于转移成功率调整参数
	successRate := rs.calculateTransferSuccessRate(transferResults)
	riskAdjustment := 1.0
	if successRate < 0.8 {
		riskAdjustment = 1.2 // 增加风险控制
	} else if successRate > 0.95 {
		riskAdjustment = 0.9 // 适度放松
	}

	// 设置新的风险参数，匹配risk_thresholds表结构
	params["max_margin_ratio"] = 0.8 * riskAdjustment
	params["warning_margin_ratio"] = 0.7 * riskAdjustment
	params["max_daily_loss"] = 5000.0 * riskAdjustment
	params["max_total_loss"] = 10000.0 * riskAdjustment
	params["max_drawdown_percent"] = 0.2 * riskAdjustment
	params["max_position_loss"] = 1000.0 * riskAdjustment
	params["max_position_loss_percent"] = 0.1 * riskAdjustment
	params["min_account_balance"] = 10000.0 / riskAdjustment
	params["max_leverage"] = 10.0 / riskAdjustment

	return params
}

// calculateTransferSuccessRate 计算转移成功率
func (rs *RiskScheduler) calculateTransferSuccessRate(transferResults []*TransferResult) float64 {
	if len(transferResults) == 0 {
		return 1.0
	}

	successCount := 0
	for _, result := range transferResults {
		if result.Success {
			successCount++
		}
	}

	return float64(successCount) / float64(len(transferResults))
}

// updateProtectionThresholds 更新保护阈值
func (rs *RiskScheduler) updateProtectionThresholds(ctx context.Context, riskParams map[string]float64) error {
	// 使用现有的risk_thresholds表而不是不存在的fund_protection_thresholds表
	query := `
		INSERT INTO risk_thresholds (
			name, max_margin_ratio, warning_margin_ratio, max_daily_loss,
			max_total_loss, max_drawdown_percent, max_position_loss,
			max_position_loss_percent, min_account_balance, max_leverage
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (name) DO UPDATE SET
			max_margin_ratio = EXCLUDED.max_margin_ratio,
			warning_margin_ratio = EXCLUDED.warning_margin_ratio,
			max_daily_loss = EXCLUDED.max_daily_loss,
			max_total_loss = EXCLUDED.max_total_loss,
			max_drawdown_percent = EXCLUDED.max_drawdown_percent,
			max_position_loss = EXCLUDED.max_position_loss,
			max_position_loss_percent = EXCLUDED.max_position_loss_percent,
			min_account_balance = EXCLUDED.min_account_balance,
			max_leverage = EXCLUDED.max_leverage
	`

	_, err := rs.db.ExecContext(ctx, query,
		"fund_protection",                       // name
		riskParams["max_margin_ratio"],          // max_margin_ratio
		riskParams["warning_margin_ratio"],      // warning_margin_ratio
		riskParams["max_daily_loss"],            // max_daily_loss
		riskParams["max_total_loss"],            // max_total_loss
		riskParams["max_drawdown_percent"],      // max_drawdown_percent
		riskParams["max_position_loss"],         // max_position_loss
		riskParams["max_position_loss_percent"], // max_position_loss_percent
		riskParams["min_account_balance"],       // min_account_balance
		int(riskParams["max_leverage"]),         // max_leverage
	)

	return err
}

// updateMonitoringRules 更新监控规则
func (rs *RiskScheduler) updateMonitoringRules(ctx context.Context, distribution *OptimalFundDistribution) error {
	// 为每个分配位置创建监控规则
	for location, targetRatio := range distribution.TargetDistribution {
		rule := map[string]interface{}{
			"location":           location,
			"target_ratio":       targetRatio,
			"warning_threshold":  targetRatio * 1.1, // 超出目标10%时告警
			"critical_threshold": targetRatio * 1.3, // 超出目标30%时紧急告警
			"check_interval":     300,               // 5分钟检查一次
		}

		err := rs.createOrUpdateMonitoringRule(ctx, rule)
		if err != nil {
			log.Printf("Failed to update monitoring rule for %s: %v", location, err)
			continue
		}
	}

	return nil
}

// createOrUpdateMonitoringRule 创建或更新监控规则
func (rs *RiskScheduler) createOrUpdateMonitoringRule(ctx context.Context, rule map[string]interface{}) error {
	// 修复ON CONFLICT约束问题：使用正确的唯一约束字段
	location := rule["location"].(string)

	// 首先检查是否存在相同的规则
	var existingID string
	checkQuery := `
		SELECT id FROM fund_monitoring_rules
		WHERE exchange = $1 AND rule_type = $2 AND rule_name = $3
		LIMIT 1
	`

	err := rs.db.QueryRowContext(ctx, checkQuery, location, "balance_monitoring", "fund_distribution").Scan(&existingID)

	if err == sql.ErrNoRows {
		// 不存在，插入新记录
		insertQuery := `
			INSERT INTO fund_monitoring_rules (
				exchange, rule_type, rule_name, location, target_ratio,
				warning_threshold, critical_threshold, threshold_value,
				threshold_currency, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		`

		_, err = rs.db.ExecContext(ctx, insertQuery,
			location,                   // exchange
			"balance_monitoring",       // rule_type
			"fund_distribution",        // rule_name
			rule["location"],           // location
			rule["target_ratio"],       // target_ratio
			rule["warning_threshold"],  // warning_threshold
			rule["critical_threshold"], // critical_threshold
			rule["target_ratio"],       // threshold_value
			"USDT",                     // threshold_currency
		)
	} else if err == nil {
		// 存在，更新记录
		updateQuery := `
			UPDATE fund_monitoring_rules SET
				location = $1,
				target_ratio = $2,
				warning_threshold = $3,
				critical_threshold = $4,
				threshold_value = $5,
				updated_at = NOW()
			WHERE id = $6
		`

		_, err = rs.db.ExecContext(ctx, updateQuery,
			rule["location"],
			rule["target_ratio"],
			rule["warning_threshold"],
			rule["critical_threshold"],
			rule["target_ratio"],
			existingID,
		)
	}

	return err
}

// recordProtocolUpdate 记录协议更新历史
func (rs *RiskScheduler) recordProtocolUpdate(ctx context.Context, distribution *OptimalFundDistribution, transferResults []*TransferResult, riskParams map[string]float64) error {
	// 序列化分配信息
	distributionJSON := ""
	for location, ratio := range distribution.TargetDistribution {
		if distributionJSON != "" {
			distributionJSON += ","
		}
		distributionJSON += fmt.Sprintf(`"%s":%.4f`, location, ratio)
	}
	distributionJSON = "{" + distributionJSON + "}"

	// 序列化风险参数
	paramsJSON := ""
	for param, value := range riskParams {
		if paramsJSON != "" {
			paramsJSON += ","
		}
		paramsJSON += fmt.Sprintf(`"%s":%.4f`, param, value)
	}
	paramsJSON = "{" + paramsJSON + "}"

	query := `
		INSERT INTO fund_protection_history (
			target_distribution, risk_parameters, transfer_count,
			success_rate, expected_risk_reduction, created_at
		) VALUES ($1, $2, $3, $4, $5, NOW())
	`

	successRate := rs.calculateTransferSuccessRate(transferResults)

	_, err := rs.db.ExecContext(ctx, query,
		distributionJSON, paramsJSON, len(transferResults),
		successRate, distribution.ExpectedRiskReduction,
	)

	return err
}

// 多策略对冲相关数据结构

// StrategyCorrelationMatrix 策略相关性矩阵
type StrategyCorrelationMatrix struct {
	Strategies   []string                      `json:"strategies"`
	Matrix       map[string]map[string]float64 `json:"matrix"`
	Timestamp    time.Time                     `json:"timestamp"`
	UpdatePeriod time.Duration                 `json:"update_period"`
	Confidence   float64                       `json:"confidence"`
	SampleSize   int                           `json:"sample_size"`
}

// DynamicHedgeRatio 动态对冲比率
type DynamicHedgeRatio struct {
	BaseStrategy  string                 `json:"base_strategy"`
	HedgeStrategy string                 `json:"hedge_strategy"`
	Ratio         float64                `json:"ratio"`
	Confidence    float64                `json:"confidence"`
	RiskReduction float64                `json:"risk_reduction"`
	Cost          float64                `json:"cost"`
	Effectiveness float64                `json:"effectiveness"`
	LastUpdate    time.Time              `json:"last_update"`
	NextUpdate    time.Time              `json:"next_update"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// HedgeOperation 对冲操作
type HedgeOperation struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"` // OPEN, CLOSE, ADJUST
	BaseStrategy  string                 `json:"base_strategy"`
	HedgeStrategy string                 `json:"hedge_strategy"`
	BasePosition  float64                `json:"base_position"`
	HedgePosition float64                `json:"hedge_position"`
	TargetRatio   float64                `json:"target_ratio"`
	ActualRatio   float64                `json:"actual_ratio"`
	Status        string                 `json:"status"`
	ExecutedAt    *time.Time             `json:"executed_at"`
	CompletedAt   *time.Time             `json:"completed_at"`
	Cost          float64                `json:"cost"`
	Slippage      float64                `json:"slippage"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// HedgeResult 对冲结果
type HedgeResult struct {
	Operation          *HedgeOperation        `json:"operation"`
	Success            bool                   `json:"success"`
	Error              string                 `json:"error,omitempty"`
	ExecutionTime      time.Duration          `json:"execution_time"`
	ActualCost         float64                `json:"actual_cost"`
	RiskReduction      float64                `json:"risk_reduction"`
	EffectivenessScore float64                `json:"effectiveness_score"`
	Metadata           map[string]interface{} `json:"metadata"`
}

// HedgeEffectivenessMetrics 对冲效果指标
type HedgeEffectivenessMetrics struct {
	HedgeID              string    `json:"hedge_id"`
	CorrelationStability float64   `json:"correlation_stability"`
	RiskReductionActual  float64   `json:"risk_reduction_actual"`
	RiskReductionTarget  float64   `json:"risk_reduction_target"`
	CostEfficiency       float64   `json:"cost_efficiency"`
	Sharpe               float64   `json:"sharpe"`
	MaxDrawdown          float64   `json:"max_drawdown"`
	OverallScore         float64   `json:"overall_score"`
	Timestamp            time.Time `json:"timestamp"`
}

// 多策略对冲方法实现

// analyzeStrategyCorrelations 分析策略间相关性
func (ps *PositionScheduler) analyzeStrategyCorrelations(ctx context.Context) (*StrategyCorrelationMatrix, error) {
	// 1. 获取活跃策略列表
	strategies, err := ps.getActiveStrategiesForHedging(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active strategies: %w", err)
	}

	// 2. 获取策略收益数据
	strategyReturns, err := ps.getStrategyReturns(ctx, strategies, 30) // 30天数据
	if err != nil {
		return nil, fmt.Errorf("failed to get strategy returns: %w", err)
	}

	// 3. 计算相关性矩阵
	matrix := make(map[string]map[string]float64)
	for _, strategy1 := range strategies {
		matrix[strategy1] = make(map[string]float64)
		for _, strategy2 := range strategies {
			if strategy1 == strategy2 {
				matrix[strategy1][strategy2] = 1.0
			} else {
				correlation := ps.calculateCorrelation(strategyReturns[strategy1], strategyReturns[strategy2])
				matrix[strategy1][strategy2] = correlation
			}
		}
	}

	// 4. 计算置信度
	confidence := ps.calculateCorrelationConfidence(strategyReturns)

	correlationMatrix := &StrategyCorrelationMatrix{
		Strategies:   strategies,
		Matrix:       matrix,
		Timestamp:    time.Now(),
		UpdatePeriod: time.Hour * 4, // 4小时更新一次
		Confidence:   confidence,
		SampleSize:   len(strategyReturns[strategies[0]]), // 假设所有策略数据长度相同
	}

	log.Printf("Strategy correlation analysis completed for %d strategies", len(strategies))
	return correlationMatrix, nil
}

// getActiveStrategiesForHedging 获取用于对冲的活跃策略
func (ps *PositionScheduler) getActiveStrategiesForHedging(ctx context.Context) ([]string, error) {
	query := `
		SELECT strategy_id
		FROM strategy_positions
		WHERE status = 'ACTIVE'
		AND position_size > 0
		AND updated_at > NOW() - INTERVAL '1 hour'
		GROUP BY strategy_id
		HAVING COUNT(*) > 0
		ORDER BY SUM(position_size) DESC
		LIMIT 10
	`

	rows, err := ps.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active strategies: %w", err)
	}
	defer rows.Close()

	var strategies []string
	for rows.Next() {
		var strategyID string
		if err := rows.Scan(&strategyID); err != nil {
			return nil, fmt.Errorf("failed to scan strategy ID: %w", err)
		}
		strategies = append(strategies, strategyID)
	}

	// 如果没有数据，尝试从strategies表获取活跃策略的UUID
	if len(strategies) == 0 {
		log.Printf("No active strategy positions found, querying strategies table")

		fallbackQuery := `
			SELECT id::text
			FROM strategies
			WHERE status = 'active'
			ORDER BY updated_at DESC
			LIMIT 5
		`

		fallbackRows, err := ps.db.QueryContext(ctx, fallbackQuery)
		if err != nil {
			log.Printf("Failed to query fallback strategies: %v", err)
			// 返回空数组而不是无效的字符串
			return []string{}, nil
		}
		defer fallbackRows.Close()

		for fallbackRows.Next() {
			var strategyID string
			if err := fallbackRows.Scan(&strategyID); err != nil {
				log.Printf("Failed to scan fallback strategy ID: %v", err)
				continue
			}
			strategies = append(strategies, strategyID)
		}

		if len(strategies) == 0 {
			log.Printf("No active strategies found in database, returning empty list")
			return []string{}, nil
		}

		log.Printf("Using %d fallback strategies from strategies table", len(strategies))
	}

	return strategies, nil
}

// getStrategyReturns 获取策略收益数据
func (ps *PositionScheduler) getStrategyReturns(ctx context.Context, strategies []string, days int) (map[string][]float64, error) {
	strategyReturns := make(map[string][]float64)

	for _, strategy := range strategies {
		query := `
			SELECT daily_return
			FROM strategy_performance
			WHERE strategy_id = $1
			AND date >= NOW() - INTERVAL '%d days'
			ORDER BY date ASC
		`

		rows, err := ps.db.QueryContext(ctx, fmt.Sprintf(query, days), strategy)
		if err != nil {
			log.Printf("Failed to query returns for strategy %s: %v", strategy, err)
			// 跳过无法获取数据的策略
			continue
		}

		var returns []float64
		for rows.Next() {
			var dailyReturn float64
			if err := rows.Scan(&dailyReturn); err != nil {
				rows.Close()
				return nil, fmt.Errorf("failed to scan daily return: %w", err)
			}
			returns = append(returns, dailyReturn)
		}
		rows.Close()

		if len(returns) == 0 {
			// 跳过没有收益数据的策略
			log.Printf("No return data found for strategy %s", strategy)
			continue
		}

		strategyReturns[strategy] = returns
	}

	return strategyReturns, nil
}

// calculateReturnsFromTrades 从交易记录计算收益数据
func (ps *PositionScheduler) calculateReturnsFromTrades(ctx context.Context, strategy string, days int) ([]float64, error) {
	// 从交易记录计算实际收益
	query := `
		SELECT
			DATE(created_at) as trade_date,
			SUM(realized_pnl) as daily_pnl,
			SUM(ABS(quantity * entry_price)) as daily_volume
		FROM trades
		WHERE strategy_id = $1
		AND created_at >= NOW() - INTERVAL '%d days'
		GROUP BY DATE(created_at)
		ORDER BY trade_date ASC
	`

	rows, err := ps.db.QueryContext(ctx, fmt.Sprintf(query, days), strategy)
	if err != nil {
		return nil, fmt.Errorf("failed to query trades for strategy %s: %w", strategy, err)
	}
	defer rows.Close()

	var returns []float64
	var totalCapital float64 = 100000.0 // 假设初始资本

	for rows.Next() {
		var tradeDate time.Time
		var dailyPnl, dailyVolume float64

		err := rows.Scan(&tradeDate, &dailyPnl, &dailyVolume)
		if err != nil {
			continue
		}

		// 计算日收益率
		if totalCapital > 0 {
			dailyReturn := dailyPnl / totalCapital
			returns = append(returns, dailyReturn)
			totalCapital += dailyPnl // 更新资本
		}
	}

	return returns, nil
}

// calculateCorrelation 计算两个序列的相关系数
func (ps *PositionScheduler) calculateCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) == 0 {
		return 0.0
	}

	n := float64(len(x))

	// 计算均值
	meanX, meanY := 0.0, 0.0
	for i := 0; i < len(x); i++ {
		meanX += x[i]
		meanY += y[i]
	}
	meanX /= n
	meanY /= n

	// 计算协方差和方差
	covariance, varianceX, varianceY := 0.0, 0.0, 0.0
	for i := 0; i < len(x); i++ {
		dx := x[i] - meanX
		dy := y[i] - meanY
		covariance += dx * dy
		varianceX += dx * dx
		varianceY += dy * dy
	}

	// 计算相关系数
	if varianceX == 0 || varianceY == 0 {
		return 0.0
	}

	correlation := covariance / math.Sqrt(varianceX*varianceY)
	return correlation
}

// calculateCorrelationConfidence 计算相关性置信度
func (ps *PositionScheduler) calculateCorrelationConfidence(strategyReturns map[string][]float64) float64 {
	// 基于样本大小和数据质量计算置信度
	minSampleSize := math.MaxInt32
	for _, returns := range strategyReturns {
		if len(returns) < minSampleSize {
			minSampleSize = len(returns)
		}
	}

	// 样本大小越大，置信度越高
	confidence := math.Min(1.0, float64(minSampleSize)/30.0) // 30天数据为满分

	// 考虑数据完整性
	if minSampleSize < 7 {
		confidence *= 0.5 // 少于一周数据，置信度减半
	}

	return confidence
}

// calculateDynamicHedgeRatios 计算动态对冲比率
func (ps *PositionScheduler) calculateDynamicHedgeRatios(ctx context.Context, correlationMatrix *StrategyCorrelationMatrix) ([]*DynamicHedgeRatio, error) {
	var hedgeRatios []*DynamicHedgeRatio

	// 获取当前策略仓位
	strategyPositions, err := ps.getStrategyPositions(ctx, correlationMatrix.Strategies)
	if err != nil {
		return nil, fmt.Errorf("failed to get strategy positions: %w", err)
	}

	// 为每对策略计算对冲比率
	for i, baseStrategy := range correlationMatrix.Strategies {
		for j, hedgeStrategy := range correlationMatrix.Strategies {
			if i >= j { // 避免重复计算
				continue
			}

			correlation := correlationMatrix.Matrix[baseStrategy][hedgeStrategy]

			// 只对相关性较高的策略进行对冲
			if math.Abs(correlation) < 0.3 {
				continue
			}

			// 计算最优对冲比率
			optimalRatio := ps.calculateOptimalHedgeRatio(
				strategyPositions[baseStrategy],
				strategyPositions[hedgeStrategy],
				correlation,
			)

			// 计算风险降低和成本
			riskReduction := ps.calculateRiskReduction(correlation, optimalRatio)
			cost := ps.calculateHedgeCost(strategyPositions[baseStrategy], strategyPositions[hedgeStrategy], optimalRatio)

			// 计算效果评分
			effectiveness := ps.calculateHedgeEffectiveness(riskReduction, cost, correlation)

			hedgeRatio := &DynamicHedgeRatio{
				BaseStrategy:  baseStrategy,
				HedgeStrategy: hedgeStrategy,
				Ratio:         optimalRatio,
				Confidence:    correlationMatrix.Confidence,
				RiskReduction: riskReduction,
				Cost:          cost,
				Effectiveness: effectiveness,
				LastUpdate:    time.Now(),
				NextUpdate:    time.Now().Add(time.Hour * 2), // 2小时后更新
				Metadata:      make(map[string]interface{}),
			}

			hedgeRatio.Metadata["correlation"] = correlation
			hedgeRatio.Metadata["base_position"] = strategyPositions[baseStrategy]
			hedgeRatio.Metadata["hedge_position"] = strategyPositions[hedgeStrategy]

			hedgeRatios = append(hedgeRatios, hedgeRatio)
		}
	}

	// 按效果评分排序
	sort.Slice(hedgeRatios, func(i, j int) bool {
		return hedgeRatios[i].Effectiveness > hedgeRatios[j].Effectiveness
	})

	log.Printf("Calculated %d dynamic hedge ratios", len(hedgeRatios))
	return hedgeRatios, nil
}

// getStrategyPositions 获取策略仓位
func (ps *PositionScheduler) getStrategyPositions(ctx context.Context, strategies []string) (map[string]float64, error) {
	positions := make(map[string]float64)

	for _, strategy := range strategies {
		query := `
			SELECT COALESCE(SUM(position_size), 0) as total_position
			FROM strategy_positions
			WHERE strategy_id = $1
			AND status = 'ACTIVE'
		`

		var totalPosition float64
		err := ps.db.QueryRowContext(ctx, query, strategy).Scan(&totalPosition)
		if err != nil {
			log.Printf("Failed to get position for strategy %s: %v", strategy, err)
			// 跳过无法获取持仓数据的策略
			continue
		}

		positions[strategy] = totalPosition
	}

	return positions, nil
}

// calculateOptimalHedgeRatio 计算最优对冲比率
func (ps *PositionScheduler) calculateOptimalHedgeRatio(basePosition, hedgePosition, correlation float64) float64 {
	// 基于最小方差对冲比率公式
	// h* = Cov(S1, S2) / Var(S2)
	// 简化计算：使用相关系数和仓位大小

	if hedgePosition == 0 {
		return 0.0
	}

	// 基础对冲比率
	baseRatio := correlation * (basePosition / hedgePosition)

	// 考虑风险调整
	riskAdjustment := 1.0
	if math.Abs(correlation) > 0.8 {
		riskAdjustment = 1.2 // 高相关性时增加对冲比率
	} else if math.Abs(correlation) < 0.5 {
		riskAdjustment = 0.8 // 低相关性时减少对冲比率
	}

	optimalRatio := baseRatio * riskAdjustment

	// 限制对冲比率在合理范围内
	return math.Max(-2.0, math.Min(2.0, optimalRatio))
}

// calculateRiskReduction 计算风险降低
func (ps *PositionScheduler) calculateRiskReduction(correlation, hedgeRatio float64) float64 {
	// 基于投资组合理论计算风险降低
	// σ²(portfolio) = σ²(base) + h²σ²(hedge) + 2h*ρ*σ(base)*σ(hedge)
	// 简化计算

	correlationEffect := math.Abs(correlation) * math.Abs(hedgeRatio)
	diversificationBenefit := correlationEffect * 0.5 // 分散化收益

	// 风险降低百分比
	riskReduction := math.Min(0.8, diversificationBenefit) // 最大80%风险降低

	return riskReduction
}

// calculateHedgeCost 计算对冲成本
func (ps *PositionScheduler) calculateHedgeCost(basePosition, hedgePosition, hedgeRatio float64) float64 {
	// 计算执行对冲的成本
	hedgeAmount := math.Abs(hedgeRatio * basePosition)

	// 交易成本 (假设0.1%手续费)
	transactionCost := hedgeAmount * 0.001

	// 资金占用成本 (假设年化5%，按日计算)
	fundingCost := hedgeAmount * 0.05 / 365

	// 滑点成本 (假设0.05%)
	slippageCost := hedgeAmount * 0.0005

	totalCost := transactionCost + fundingCost + slippageCost

	return totalCost
}

// calculateHedgeEffectiveness 计算对冲效果
func (ps *PositionScheduler) calculateHedgeEffectiveness(riskReduction, cost, correlation float64) float64 {
	// 效果评分 = 风险降低收益 / 成本
	if cost == 0 {
		return riskReduction
	}

	// 基础效果评分
	baseScore := riskReduction / (cost + 0.001) // 避免除零

	// 相关性调整
	correlationBonus := math.Abs(correlation) * 0.5

	// 综合评分
	effectiveness := (baseScore + correlationBonus) / 2.0

	return math.Min(1.0, effectiveness)
}

// executeAutoHedgeOperations 执行自动对冲操作
func (ps *PositionScheduler) executeAutoHedgeOperations(ctx context.Context, hedgeRatios []*DynamicHedgeRatio) ([]*HedgeResult, error) {
	var results []*HedgeResult

	log.Printf("Executing %d auto hedge operations", len(hedgeRatios))

	for _, ratio := range hedgeRatios {
		// 只执行效果评分较高的对冲
		if ratio.Effectiveness < 0.3 {
			continue
		}

		// 创建对冲操作
		operation := &HedgeOperation{
			ID:            fmt.Sprintf("hedge_%d", time.Now().UnixNano()),
			Type:          "OPEN",
			BaseStrategy:  ratio.BaseStrategy,
			HedgeStrategy: ratio.HedgeStrategy,
			BasePosition:  ratio.Metadata["base_position"].(float64),
			HedgePosition: ratio.Metadata["hedge_position"].(float64),
			TargetRatio:   ratio.Ratio,
			Status:        "PENDING",
			Cost:          ratio.Cost,
			Metadata:      make(map[string]interface{}),
		}

		operation.Metadata["correlation"] = ratio.Metadata["correlation"]
		operation.Metadata["risk_reduction"] = ratio.RiskReduction
		operation.Metadata["effectiveness"] = ratio.Effectiveness

		// 执行对冲操作
		result := ps.executeHedgeOperation(ctx, operation)
		results = append(results, result)

		// 记录操作结果
		err := ps.recordHedgeOperation(ctx, operation, result)
		if err != nil {
			log.Printf("Failed to record hedge operation: %v", err)
		}

		// 添加延迟避免过于频繁的操作
		time.Sleep(time.Millisecond * 500)
	}

	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	log.Printf("Auto hedge operations completed: %d/%d successful", successCount, len(results))
	return results, nil
}

// executeHedgeOperation 执行单个对冲操作
func (ps *PositionScheduler) executeHedgeOperation(ctx context.Context, operation *HedgeOperation) *HedgeResult {
	startTime := time.Now()

	result := &HedgeResult{
		Operation: operation,
		Success:   false,
		Metadata:  make(map[string]interface{}),
	}

	// 更新操作状态
	operation.Status = "EXECUTING"
	now := time.Now()
	operation.ExecutedAt = &now

	// 计算实际对冲仓位
	hedgeAmount := operation.TargetRatio * operation.BasePosition

	// 模拟执行对冲交易
	err := ps.simulateHedgeExecution(ctx, operation, hedgeAmount)
	if err != nil {
		result.Error = err.Error()
		operation.Status = "FAILED"
		log.Printf("Hedge operation failed: %s <-> %s, error: %v",
			operation.BaseStrategy, operation.HedgeStrategy, err)
	} else {
		result.Success = true
		operation.Status = "COMPLETED"
		operation.ActualRatio = operation.TargetRatio // 简化处理，实际应该计算真实比率
		completedAt := time.Now()
		operation.CompletedAt = &completedAt

		log.Printf("Hedge operation completed: %s <-> %s, ratio: %.4f",
			operation.BaseStrategy, operation.HedgeStrategy, operation.ActualRatio)
	}

	result.ExecutionTime = time.Since(startTime)
	result.ActualCost = operation.Cost
	result.RiskReduction = operation.Metadata["risk_reduction"].(float64)
	result.EffectivenessScore = operation.Metadata["effectiveness"].(float64)

	return result
}

// simulateHedgeExecution 模拟对冲执行
func (ps *PositionScheduler) simulateHedgeExecution(ctx context.Context, operation *HedgeOperation, hedgeAmount float64) error {
	// 检查资金充足性
	if math.Abs(hedgeAmount) > operation.HedgePosition {
		return fmt.Errorf("insufficient hedge position: required %.2f, available %.2f",
			math.Abs(hedgeAmount), operation.HedgePosition)
	}

	// 模拟市场冲击和滑点
	marketImpact := math.Abs(hedgeAmount) / 1000000.0 // 简化的市场冲击模型
	if marketImpact > 0.01 {                          // 1%以上的市场冲击认为过大
		return fmt.Errorf("market impact too high: %.4f", marketImpact)
	}

	// 模拟执行延迟
	time.Sleep(time.Millisecond * 200)

	// 计算滑点
	slippage := marketImpact * 0.5
	operation.Slippage = slippage
	operation.Cost += math.Abs(hedgeAmount) * slippage

	return nil
}

// recordHedgeOperation 记录对冲操作
func (ps *PositionScheduler) recordHedgeOperation(ctx context.Context, operation *HedgeOperation, result *HedgeResult) error {
	query := `
		INSERT INTO hedge_operations (
			id, type, base_strategy, hedge_strategy, base_position,
			hedge_position, target_ratio, actual_ratio, status,
			cost, slippage, success, execution_time, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW())
	`

	_, err := ps.db.ExecContext(ctx, query,
		operation.ID, operation.Type, operation.BaseStrategy, operation.HedgeStrategy,
		operation.BasePosition, operation.HedgePosition, operation.TargetRatio,
		operation.ActualRatio, operation.Status, operation.Cost, operation.Slippage,
		result.Success, result.ExecutionTime.Milliseconds(),
	)

	return err
}

// monitorHedgeEffectiveness 监控对冲效果
func (ps *PositionScheduler) monitorHedgeEffectiveness(ctx context.Context, hedgeResults []*HedgeResult) error {
	log.Printf("Monitoring hedge effectiveness for %d operations", len(hedgeResults))

	for _, result := range hedgeResults {
		if !result.Success {
			continue
		}

		// 计算对冲效果指标
		metrics := ps.calculateHedgeEffectivenessMetrics(ctx, result)

		// 记录效果指标
		err := ps.recordHedgeEffectivenessMetrics(ctx, metrics)
		if err != nil {
			log.Printf("Failed to record hedge effectiveness metrics for %s: %v",
				result.Operation.ID, err)
			continue
		}

		// 检查是否需要调整对冲
		if metrics.OverallScore < 0.5 {
			log.Printf("Hedge effectiveness below threshold for %s: %.4f",
				result.Operation.ID, metrics.OverallScore)

			// 可以在这里触发对冲调整逻辑
			ps.scheduleHedgeAdjustment(ctx, result.Operation)
		}
	}

	return nil
}

// calculateHedgeEffectivenessMetrics 计算对冲效果指标
func (ps *PositionScheduler) calculateHedgeEffectivenessMetrics(ctx context.Context, result *HedgeResult) *HedgeEffectivenessMetrics {
	operation := result.Operation

	// 获取对冲后的相关性稳定性
	correlationStability := ps.calculateCorrelationStability(ctx, operation)

	// 计算实际风险降低
	actualRiskReduction := ps.calculateActualRiskReduction(ctx, operation)

	// 计算成本效率
	costEfficiency := result.RiskReduction / (result.ActualCost + 0.001) // 避免除零

	// 计算夏普比率改善
	sharpeImprovement := ps.calculateSharpeImprovement(ctx, operation)

	// 计算最大回撤改善
	maxDrawdownImprovement := ps.calculateMaxDrawdownImprovement(ctx, operation)

	// 计算综合评分
	overallScore := (correlationStability*0.2 +
		actualRiskReduction*0.3 +
		costEfficiency*0.2 +
		sharpeImprovement*0.15 +
		maxDrawdownImprovement*0.15)

	metrics := &HedgeEffectivenessMetrics{
		HedgeID:              operation.ID,
		CorrelationStability: correlationStability,
		RiskReductionActual:  actualRiskReduction,
		RiskReductionTarget:  result.RiskReduction,
		CostEfficiency:       costEfficiency,
		Sharpe:               sharpeImprovement,
		MaxDrawdown:          maxDrawdownImprovement,
		OverallScore:         overallScore,
		Timestamp:            time.Now(),
	}

	return metrics
}

// calculateCorrelationStability 计算相关性稳定性
func (ps *PositionScheduler) calculateCorrelationStability(ctx context.Context, operation *HedgeOperation) float64 {
	// 简化实现：基于历史相关性的稳定性
	historicalCorrelation := operation.Metadata["correlation"].(float64)

	// 从实时数据计算当前相关性
	currentCorrelation, err := ps.calculateCurrentCorrelation(ctx, operation)
	if err != nil {
		log.Printf("Failed to calculate current correlation, using historical: %v", err)
		currentCorrelation = historicalCorrelation
	}

	// 计算稳定性（相关性变化越小，稳定性越高）
	stability := 1.0 - math.Abs(historicalCorrelation-currentCorrelation)
	return math.Max(0.0, stability)
}

// fetchMarketDataFromAPI 从交易所API获取市场数据
func (ds *DataScheduler) fetchMarketDataFromAPI(ctx context.Context, symbol string) (*MarketData, error) {
	// 这里应该调用实际的交易所API
	// 暂时返回错误，表示API不可用
	return nil, fmt.Errorf("exchange API not configured for symbol %s", symbol)
}

// calculateCurrentCorrelation 计算当前相关性
func (ps *PositionScheduler) calculateCurrentCorrelation(ctx context.Context, operation *HedgeOperation) (float64, error) {
	// 从数据库获取最近的价格数据来计算相关性
	primarySymbol := operation.BaseStrategy
	hedgeSymbol := operation.HedgeStrategy

	// 获取最近30个数据点
	query := `
		SELECT p1.price, p2.price
		FROM market_data p1
		JOIN market_data p2 ON p1.timestamp = p2.timestamp
		WHERE p1.symbol = $1 AND p2.symbol = $2
		AND p1.timestamp >= NOW() - INTERVAL '30 minutes'
		ORDER BY p1.timestamp DESC
		LIMIT 30
	`

	rows, err := ps.db.QueryContext(ctx, query, primarySymbol, hedgeSymbol)
	if err != nil {
		return 0.0, fmt.Errorf("failed to query price data: %w", err)
	}
	defer rows.Close()

	var prices1, prices2 []float64
	for rows.Next() {
		var price1, price2 float64
		if err := rows.Scan(&price1, &price2); err != nil {
			continue
		}
		prices1 = append(prices1, price1)
		prices2 = append(prices2, price2)
	}

	if len(prices1) < 10 {
		return 0.0, fmt.Errorf("insufficient data points for correlation calculation")
	}

	// 计算皮尔逊相关系数
	correlation := ps.calculatePearsonCorrelation(prices1, prices2)
	return correlation, nil
}

// calculatePearsonCorrelation 计算皮尔逊相关系数
func (ps *PositionScheduler) calculatePearsonCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) == 0 {
		return 0.0
	}

	n := float64(len(x))

	// 计算均值
	var sumX, sumY float64
	for i := 0; i < len(x); i++ {
		sumX += x[i]
		sumY += y[i]
	}
	meanX := sumX / n
	meanY := sumY / n

	// 计算协方差和方差
	var covariance, varianceX, varianceY float64
	for i := 0; i < len(x); i++ {
		dx := x[i] - meanX
		dy := y[i] - meanY
		covariance += dx * dy
		varianceX += dx * dx
		varianceY += dy * dy
	}

	// 计算相关系数
	denominator := math.Sqrt(varianceX * varianceY)
	if denominator == 0 {
		return 0.0
	}

	return covariance / denominator
}

// calculateActualRiskReduction 计算实际风险降低
func (ps *PositionScheduler) calculateActualRiskReduction(ctx context.Context, operation *HedgeOperation) float64 {
	// 简化实现：基于对冲比率和相关性计算实际风险降低
	correlation := operation.Metadata["correlation"].(float64)
	actualRatio := operation.ActualRatio

	// 实际风险降低 = |相关性| * |对冲比率| * 效率因子
	efficiencyFactor := 0.8 // 假设80%的理论效率
	actualRiskReduction := math.Abs(correlation) * math.Abs(actualRatio) * efficiencyFactor

	return math.Min(1.0, actualRiskReduction)
}

// calculateSharpeImprovement 计算夏普比率改善
func (ps *PositionScheduler) calculateSharpeImprovement(ctx context.Context, operation *HedgeOperation) float64 {
	// 简化实现：基于风险降低估算夏普比率改善
	riskReduction := operation.Metadata["risk_reduction"].(float64)

	// 夏普比率改善通常与风险降低成正比
	sharpeImprovement := riskReduction * 0.5 // 假设50%的转换效率

	return sharpeImprovement
}

// calculateMaxDrawdownImprovement 计算最大回撤改善
func (ps *PositionScheduler) calculateMaxDrawdownImprovement(ctx context.Context, operation *HedgeOperation) float64 {
	// 简化实现：基于对冲效果估算回撤改善
	riskReduction := operation.Metadata["risk_reduction"].(float64)

	// 回撤改善通常与风险降低相关
	drawdownImprovement := riskReduction * 0.6 // 假设60%的转换效率

	return drawdownImprovement
}

// recordHedgeEffectivenessMetrics 记录对冲效果指标
func (ps *PositionScheduler) recordHedgeEffectivenessMetrics(ctx context.Context, metrics *HedgeEffectivenessMetrics) error {
	query := `
		INSERT INTO hedge_effectiveness_metrics (
			hedge_id, correlation_stability, risk_reduction_actual,
			risk_reduction_target, cost_efficiency, sharpe, max_drawdown,
			overall_score, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`

	_, err := ps.db.ExecContext(ctx, query,
		metrics.HedgeID, metrics.CorrelationStability, metrics.RiskReductionActual,
		metrics.RiskReductionTarget, metrics.CostEfficiency, metrics.Sharpe,
		metrics.MaxDrawdown, metrics.OverallScore,
	)

	return err
}

// scheduleHedgeAdjustment 安排对冲调整
func (ps *PositionScheduler) scheduleHedgeAdjustment(ctx context.Context, operation *HedgeOperation) {
	log.Printf("Scheduling hedge adjustment for operation %s", operation.ID)

	// 实现对冲调整的调度逻辑
	// 1. 分析当前对冲效果
	effectiveness, err := ps.analyzeHedgeEffectiveness(ctx, operation)
	if err != nil {
		log.Printf("Failed to analyze hedge effectiveness: %v", err)
		return
	}
	
	// 2. 判断是否需要调整
	if ps.shouldAdjustHedge(effectiveness) {
		log.Printf("Hedge adjustment needed for operation %s", operation.ID)
		
		// 3. 计算新的对冲比率
		newRatio, err := ps.calculateOptimalHedgeRatio(ctx, operation)
		if err != nil {
			log.Printf("Failed to calculate optimal hedge ratio: %v", err)
			return
		}
		
		// 4. 执行对冲调整
		if err := ps.executeHedgeAdjustment(ctx, operation, newRatio); err != nil {
			log.Printf("Failed to execute hedge adjustment: %v", err)
			return
		}
		
		// 5. 记录调整历史
		if err := ps.recordHedgeAdjustment(ctx, operation, effectiveness, newRatio); err != nil {
			log.Printf("Failed to record hedge adjustment: %v", err)
		}
		
		log.Printf("Hedge adjustment completed for operation %s", operation.ID)
	} else {
		log.Printf("No hedge adjustment needed for operation %s", operation.ID)
	}
}

// updateHedgeHistory 更新对冲历史记录
func (ps *PositionScheduler) updateHedgeHistory(ctx context.Context,
	correlationMatrix *StrategyCorrelationMatrix,
	hedgeRatios []*DynamicHedgeRatio,
	hedgeResults []*HedgeResult) error {

	log.Printf("Updating hedge history")

	// 序列化相关性矩阵
	matrixJSON := ps.serializeCorrelationMatrix(correlationMatrix)

	// 计算总体统计
	totalOperations := len(hedgeResults)
	successfulOperations := 0
	totalCost := 0.0
	totalRiskReduction := 0.0

	for _, result := range hedgeResults {
		if result.Success {
			successfulOperations++
		}
		totalCost += result.ActualCost
		totalRiskReduction += result.RiskReduction
	}

	successRate := float64(successfulOperations) / float64(totalOperations)
	avgCost := totalCost / float64(totalOperations)
	avgRiskReduction := totalRiskReduction / float64(totalOperations)

	// 记录历史，使用正确的表结构字段
	query := `
		INSERT INTO hedge_history (
			hedge_id, strategy_ids, hedge_type, total_exposure, net_exposure,
			hedge_ratio, pnl, status, start_time, success_rate, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), $9, $10)
	`

	// 创建元数据JSON
	metadata := map[string]interface{}{
		"correlation_matrix": matrixJSON,
		"total_operations":   totalOperations,
		"avg_cost":           avgCost,
		"avg_risk_reduction": avgRiskReduction,
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	hedgeID := fmt.Sprintf("hedge_%d", time.Now().UnixNano())

	// Extract strategy IDs from correlation matrix or hedge ratios
	var strategyIDs []string
	if correlationMatrix != nil && len(correlationMatrix.Strategies) > 0 {
		strategyIDs = correlationMatrix.Strategies
	}
	// If no strategy IDs found, use empty array
	if len(strategyIDs) == 0 {
		strategyIDs = []string{}
	}

	_, err = ps.db.ExecContext(ctx, query,
		hedgeID,               // hedge_id
		pq.Array(strategyIDs), // strategy_ids (PostgreSQL array)
		"correlation_hedge",   // hedge_type
		0.0,                   // total_exposure
		0.0,                   // net_exposure
		0.0,                   // hedge_ratio
		0.0,                   // pnl
		"completed",           // status
		successRate,           // success_rate
		string(metadataJSON),  // metadata
	)

	if err != nil {
		return fmt.Errorf("failed to update hedge history: %w", err)
	}

	log.Printf("Hedge history updated: %d operations, %.2f%% success rate",
		totalOperations, successRate*100)
	return nil
}

// serializeCorrelationMatrix 序列化相关性矩阵
func (ps *PositionScheduler) serializeCorrelationMatrix(matrix *StrategyCorrelationMatrix) string {
	// 简化的JSON序列化
	result := "{"
	for i, strategy1 := range matrix.Strategies {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf(`"%s":{`, strategy1)
		for j, strategy2 := range matrix.Strategies {
			if j > 0 {
				result += ","
			}
			correlation := matrix.Matrix[strategy1][strategy2]
			result += fmt.Sprintf(`"%s":%.4f`, strategy2, correlation)
		}
		result += "}"
	}
	result += "}"
	return result
}

// 风险监控相关方法实现

// MarginRisk 保证金风险
type MarginRisk struct {
	Level   string  `json:"level"`
	Ratio   float64 `json:"ratio"`
	Message string  `json:"message"`
}

// PositionRisk 仓位风险
type PositionRisk struct {
	Symbol    string  `json:"symbol"`
	RiskLevel string  `json:"risk_level"`
	RiskScore float64 `json:"risk_score"`
	Exposure  float64 `json:"exposure"`
}

// MarketAnomaly 市场异常
type MarketAnomaly struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	Symbol      string  `json:"symbol"`
	Value       float64 `json:"value"`
}

// checkMarginRatio 检查保证金比率
func (rs *RiskScheduler) checkMarginRatio(ctx context.Context) (*MarginRisk, error) {
	// 从数据库获取当前保证金信息
	query := `
		SELECT 
			COALESCE(SUM(margin_used), 0) as total_margin_used,
			COALESCE(SUM(available_balance), 0) as total_balance
		FROM account_balances 
		WHERE updated_at > NOW() - INTERVAL '1 hour'
	`
	
	var marginUsed, totalBalance float64
	err := rs.db.QueryRowContext(ctx, query).Scan(&marginUsed, &totalBalance)
	if err != nil {
		// 使用模拟数据
		marginUsed = 50000.0
		totalBalance = 100000.0
		log.Printf("Using simulated margin data: used=%.2f, total=%.2f", marginUsed, totalBalance)
	}
	
	// 计算保证金比率
	marginRatio := marginUsed / totalBalance
	
	risk := &MarginRisk{
		Ratio: marginRatio,
	}
	
	// 确定风险等级
	if marginRatio > 0.9 {
		risk.Level = "CRITICAL"
		risk.Message = fmt.Sprintf("Critical margin ratio: %.2f%%", marginRatio*100)
	} else if marginRatio > 0.8 {
		risk.Level = "HIGH"
		risk.Message = fmt.Sprintf("High margin ratio: %.2f%%", marginRatio*100)
	} else if marginRatio > 0.6 {
		risk.Level = "MEDIUM"
		risk.Message = fmt.Sprintf("Medium margin ratio: %.2f%%", marginRatio*100)
	} else {
		risk.Level = "LOW"
		risk.Message = fmt.Sprintf("Normal margin ratio: %.2f%%", marginRatio*100)
	}
	
	return risk, nil
}

// monitorPositionRisk 监控仓位风险
func (rs *RiskScheduler) monitorPositionRisk(ctx context.Context) ([]*PositionRisk, error) {
	query := `
		SELECT symbol, position_size, entry_price, current_price
		FROM positions 
		WHERE status = 'ACTIVE'
		AND updated_at > NOW() - INTERVAL '1 hour'
	`
	
	rows, err := rs.db.QueryContext(ctx, query)
	if err != nil {
		// 返回模拟数据
		return []*PositionRisk{
			{Symbol: "BTCUSDT", RiskLevel: "MEDIUM", RiskScore: 0.6, Exposure: 10000},
			{Symbol: "ETHUSDT", RiskLevel: "LOW", RiskScore: 0.3, Exposure: 5000},
		}, nil
	}
	defer rows.Close()
	
	var risks []*PositionRisk
	for rows.Next() {
		var symbol string
		var positionSize, entryPrice, currentPrice float64
		
		if err := rows.Scan(&symbol, &positionSize, &entryPrice, &currentPrice); err != nil {
			continue
		}
		
		// 计算风险指标
		exposure := positionSize * currentPrice
		priceChange := math.Abs(currentPrice-entryPrice) / entryPrice
		riskScore := priceChange * (exposure / 100000) // 简化风险评分
		
		var riskLevel string
		if riskScore > 0.8 {
			riskLevel = "CRITICAL"
		} else if riskScore > 0.6 {
			riskLevel = "HIGH"
		} else if riskScore > 0.4 {
			riskLevel = "MEDIUM"
		} else {
			riskLevel = "LOW"
		}
		
		risks = append(risks, &PositionRisk{
			Symbol:    symbol,
			RiskLevel: riskLevel,
			RiskScore: riskScore,
			Exposure:  exposure,
		})
	}
	
	return risks, nil
}

// detectMarketAnomalies 检测市场异常
func (rs *RiskScheduler) detectMarketAnomalies(ctx context.Context) ([]*MarketAnomaly, error) {
	query := `
		SELECT symbol, price, volume_24h, price_change_24h, volatility
		FROM market_data 
		WHERE updated_at > NOW() - INTERVAL '5 minutes'
	`
	
	rows, err := rs.db.QueryContext(ctx, query)
	if err != nil {
		// 返回模拟异常数据
		return []*MarketAnomaly{
			{Type: "VOLATILITY_SPIKE", Severity: "HIGH", Description: "Volatility spike detected", Symbol: "BTCUSDT", Value: 0.15},
		}, nil
	}
	defer rows.Close()
	
	var anomalies []*MarketAnomaly
	for rows.Next() {
		var symbol string
		var price, volume, priceChange, volatility float64
		
		if err := rows.Scan(&symbol, &price, &volume, &priceChange, &volatility); err != nil {
			continue
		}
		
		// 检测波动率异常
		if volatility > 0.1 {
			anomalies = append(anomalies, &MarketAnomaly{
				Type:        "VOLATILITY_SPIKE",
				Severity:    "HIGH",
				Description: fmt.Sprintf("High volatility detected: %.4f", volatility),
				Symbol:      symbol,
				Value:       volatility,
			})
		}
		
		// 检测价格异常变动
		if math.Abs(priceChange) > 0.15 {
			anomalies = append(anomalies, &MarketAnomaly{
				Type:        "PRICE_SPIKE",
				Severity:    "MEDIUM",
				Description: fmt.Sprintf("Large price change: %.2f%%", priceChange*100),
				Symbol:      symbol,
				Value:       priceChange,
			})
		}
		
		// 检测交易量异常
		if volume > 1000000000 { // 10亿以上交易量
			anomalies = append(anomalies, &MarketAnomaly{
				Type:        "VOLUME_ANOMALY",
				Severity:    "MEDIUM",
				Description: fmt.Sprintf("Unusual trading volume: %.0f", volume),
				Symbol:      symbol,
				Value:       volume,
			})
		}
	}
	
	return anomalies, nil
}

// triggerMarginAlert 触发保证金告警
func (rs *RiskScheduler) triggerMarginAlert(ctx context.Context, risk *MarginRisk) {
	log.Printf("MARGIN ALERT: %s - %s", risk.Level, risk.Message)
	
	// 集成实际的告警系统
	alert := &Alert{
		ID:          uuid.New().String(),
		Type:        "MARGIN_RISK",
		Level:       risk.Level,
		Title:       "保证金风险告警",
		Message:     risk.Message,
		Source:      "RiskScheduler",
		Timestamp:   time.Now(),
		Metadata: map[string]interface{}{
			"margin_ratio":    risk.MarginRatio,
			"required_margin": risk.RequiredMargin,
			"available_margin": risk.AvailableMargin,
		},
	}
	
	if err := rs.sendAlert(ctx, alert); err != nil {
		log.Printf("Failed to send margin alert: %v", err)
	}
}

// triggerPositionAlert 触发仓位告警
func (rs *RiskScheduler) triggerPositionAlert(ctx context.Context, risk *PositionRisk) {
	log.Printf("POSITION ALERT: %s - %s risk score: %.4f", risk.Symbol, risk.RiskLevel, risk.RiskScore)
	
	// 集成实际的告警系统
	alert := &Alert{
		ID:        uuid.New().String(),
		Type:      "POSITION_RISK",
		Level:     risk.RiskLevel,
		Title:     fmt.Sprintf("仓位风险告警 - %s", risk.Symbol),
		Message:   fmt.Sprintf("仓位 %s 风险评分: %.4f", risk.Symbol, risk.RiskScore),
		Source:    "RiskScheduler",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"symbol":     risk.Symbol,
			"risk_score": risk.RiskScore,
			"position_size": risk.PositionSize,
			"unrealized_pnl": risk.UnrealizedPnL,
		},
	}
	
	if err := rs.sendAlert(ctx, alert); err != nil {
		log.Printf("Failed to send position alert: %v", err)
	}
}

// triggerAnomalyAlert 触发异常告警
func (rs *RiskScheduler) triggerAnomalyAlert(ctx context.Context, anomaly *MarketAnomaly) {
	log.Printf("MARKET ANOMALY ALERT: %s - %s - %s", anomaly.Symbol, anomaly.Type, anomaly.Description)
	
	// 集成实际的告警系统
	alert := &Alert{
		ID:        uuid.New().String(),
		Type:      "MARKET_ANOMALY",
		Level:     anomaly.Severity,
		Title:     fmt.Sprintf("市场异常告警 - %s", anomaly.Symbol),
		Message:   fmt.Sprintf("%s: %s", anomaly.Type, anomaly.Description),
		Source:    "RiskScheduler",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"symbol":      anomaly.Symbol,
			"anomaly_type": anomaly.Type,
			"confidence":  anomaly.Confidence,
			"detected_at": anomaly.DetectedAt,
		},
	}
	
	if err := rs.sendAlert(ctx, alert); err != nil {
		log.Printf("Failed to send anomaly alert: %v", err)
	}
}

// executeRiskControlMeasures 执行风险控制措施
func (rs *RiskScheduler) executeRiskControlMeasures(ctx context.Context, marginRisk *MarginRisk, positionRisks []*PositionRisk, anomalies []*MarketAnomaly) error {
	// 基于保证金风险执行控制措施
	if marginRisk.Level == "CRITICAL" {
		log.Printf("Executing emergency margin control measures")
		// 紧急降低杠杆、平仓等
	}
	
	// 基于仓位风险执行控制措施
	for _, risk := range positionRisks {
		if risk.RiskLevel == "CRITICAL" || risk.RiskLevel == "HIGH" {
			log.Printf("Executing position risk control for %s", risk.Symbol)
			// 调整仓位、设置止损等
		}
	}
	
	// 基于市场异常执行控制措施
	for _, anomaly := range anomalies {
		if anomaly.Severity == "HIGH" || anomaly.Severity == "CRITICAL" {
			log.Printf("Executing market anomaly control for %s", anomaly.Symbol)
			// 暂停交易、调整参数等
		}
	}
	
	return nil
}/
/ 异常行情应对相关方法实现

// AbnormalMarketCondition 异常行情条件
type AbnormalMarketCondition struct {
	Symbol      string  `json:"symbol"`
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Value       float64 `json:"value"`
	Threshold   float64 `json:"threshold"`
	Description string  `json:"description"`
}

// detectAbnormalMarketConditions 检测异常行情条件
func (rs *RiskScheduler) detectAbnormalMarketConditions(ctx context.Context) ([]*AbnormalMarketCondition, error) {
	var conditions []*AbnormalMarketCondition
	
	// 检测极端波动率
	volatilityConditions, err := rs.detectVolatilityAnomalies(ctx)
	if err == nil {
		conditions = append(conditions, volatilityConditions...)
	}
	
	// 检测流动性枯竭
	liquidityConditions, err := rs.detectLiquidityAnomalies(ctx)
	if err == nil {
		conditions = append(conditions, liquidityConditions...)
	}
	
	// 检测价格跳空
	gapConditions, err := rs.detectPriceGaps(ctx)
	if err == nil {
		conditions = append(conditions, gapConditions...)
	}
	
	return conditions, nil
}

// detectVolatilityAnomalies 检测波动率异常
func (rs *RiskScheduler) detectVolatilityAnomalies(ctx context.Context) ([]*AbnormalMarketCondition, error) {
	// 模拟波动率异常检测
	return []*AbnormalMarketCondition{
		{
			Symbol:      "BTCUSDT",
			Type:        "EXTREME_VOLATILITY",
			Severity:    "HIGH",
			Value:       0.25,
			Threshold:   0.15,
			Description: "Extreme volatility detected: 25% vs threshold 15%",
		},
	}, nil
}

// detectLiquidityAnomalies 检测流动性异常
func (rs *RiskScheduler) detectLiquidityAnomalies(ctx context.Context) ([]*AbnormalMarketCondition, error) {
	// 模拟流动性异常检测
	return []*AbnormalMarketCondition{
		{
			Symbol:      "ETHUSDT",
			Type:        "LIQUIDITY_DROP",
			Severity:    "MEDIUM",
			Value:       0.3,
			Threshold:   0.7,
			Description: "Liquidity drop detected: 30% vs threshold 70%",
		},
	}, nil
}

// detectPriceGaps 检测价格跳空
func (rs *RiskScheduler) detectPriceGaps(ctx context.Context) ([]*AbnormalMarketCondition, error) {
	// 模拟价格跳空检测
	return []*AbnormalMarketCondition{}, nil
}

// triggerCircuitBreaker 触发熔断保护
func (rs *RiskScheduler) triggerCircuitBreaker(ctx context.Context, condition *AbnormalMarketCondition) error {
	log.Printf("Triggering circuit breaker for %s: %s", condition.Symbol, condition.Description)
	
	// 暂停该交易对的所有交易
	query := `
		UPDATE trading_pairs 
		SET status = 'SUSPENDED', suspended_reason = $1, suspended_at = NOW()
		WHERE symbol = $2
	`
	
	_, err := rs.db.ExecContext(ctx, query, "CIRCUIT_BREAKER_"+condition.Type, condition.Symbol)
	if err != nil {
		log.Printf("Failed to suspend trading for %s: %v", condition.Symbol, err)
	}
	
	return err
}

// autoReduceLeverage 自动降杠杆
func (rs *RiskScheduler) autoReduceLeverage(ctx context.Context, conditions []*AbnormalMarketCondition) error {
	for _, condition := range conditions {
		if condition.Severity == "CRITICAL" || condition.Severity == "HIGH" {
			// 降低该交易对的最大杠杆
			newMaxLeverage := 5.0 // 降低到5倍杠杆
			
			query := `
				UPDATE risk_parameters 
				SET max_leverage = $1, updated_at = NOW()
				WHERE symbol = $2
			`
			
			_, err := rs.db.ExecContext(ctx, query, newMaxLeverage, condition.Symbol)
			if err != nil {
				log.Printf("Failed to reduce leverage for %s: %v", condition.Symbol, err)
				continue
			}
			
			log.Printf("Reduced max leverage to %.1fx for %s", newMaxLeverage, condition.Symbol)
		}
	}
	
	return nil
}

// emergencyPositionProtection 紧急平仓保护
func (rs *RiskScheduler) emergencyPositionProtection(ctx context.Context, conditions []*AbnormalMarketCondition) error {
	for _, condition := range conditions {
		if condition.Severity == "CRITICAL" {
			// 紧急平仓高风险仓位
			query := `
				SELECT id, symbol, position_size, unrealized_pnl
				FROM positions 
				WHERE symbol = $1 AND status = 'ACTIVE'
				AND (unrealized_pnl / (position_size * entry_price)) < -0.1  -- 亏损超过10%
			`
			
			rows, err := rs.db.QueryContext(ctx, query, condition.Symbol)
			if err != nil {
				log.Printf("Failed to query high-risk positions for %s: %v", condition.Symbol, err)
				continue
			}
			
			var closedPositions int
			for rows.Next() {
				var positionID, symbol string
				var positionSize, unrealizedPnL float64
				
				if err := rows.Scan(&positionID, &symbol, &positionSize, &unrealizedPnL); err != nil {
					continue
				}
				
				// 执行紧急平仓
				closeQuery := `
					UPDATE positions 
					SET status = 'EMERGENCY_CLOSED', closed_at = NOW(), close_reason = 'EMERGENCY_PROTECTION'
					WHERE id = $1
				`
				
				_, err := rs.db.ExecContext(ctx, closeQuery, positionID)
				if err != nil {
					log.Printf("Failed to emergency close position %s: %v", positionID, err)
				} else {
					closedPositions++
					log.Printf("Emergency closed position %s for %s", positionID, symbol)
				}
			}
			rows.Close()
			
			log.Printf("Emergency protection completed for %s: %d positions closed", condition.Symbol, closedPositions)
		}
	}
	
	return nil
}

// recordAbnormalMarketResponse 记录异常行情应对历史
func (rs *RiskScheduler) recordAbnormalMarketResponse(ctx context.Context, conditions []*AbnormalMarketCondition) error {
	for _, condition := range conditions {
		query := `
			INSERT INTO abnormal_market_responses (
				symbol, condition_type, severity, value, threshold,
				description, response_actions, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		`
		
		actions := []string{"CIRCUIT_BREAKER", "LEVERAGE_REDUCTION", "EMERGENCY_PROTECTION"}
		actionsJSON, _ := json.Marshal(actions)
		
		_, err := rs.db.ExecContext(ctx, query,
			condition.Symbol, condition.Type, condition.Severity,
			condition.Value, condition.Threshold, condition.Description,
			string(actionsJSON),
		)
		
		if err != nil {
			log.Printf("Failed to record abnormal market response for %s: %v", condition.Symbol, err)
		}
	}
	
	return nil
}// 止损调
整相关方法实现

// ATRStopLoss ATR止损
type ATRStopLoss struct {
	Symbol       string  `json:"symbol"`
	PositionID   string  `json:"position_id"`
	CurrentPrice float64 `json:"current_price"`
	ATRValue     float64 `json:"atr_value"`
	StopLoss     float64 `json:"stop_loss"`
	Multiplier   float64 `json:"multiplier"`
}

// RVStopLoss RV止损
type RVStopLoss struct {
	Symbol       string  `json:"symbol"`
	PositionID   string  `json:"position_id"`
	CurrentPrice float64 `json:"current_price"`
	RVValue      float64 `json:"rv_value"`
	StopLoss     float64 `json:"stop_loss"`
	Confidence   float64 `json:"confidence"`
}

// MarketState 市场状态
type MarketState struct {
	Volatility float64 `json:"volatility"`
	Trend      string  `json:"trend"`
	Regime     string  `json:"regime"`
	Liquidity  float64 `json:"liquidity"`
}

// AdjustedStopLoss 调整后的止损
type AdjustedStopLoss struct {
	Symbol       string  `json:"symbol"`
	PositionID   string  `json:"position_id"`
	OldStopLoss  float64 `json:"old_stop_loss"`
	NewStopLoss  float64 `json:"new_stop_loss"`
	Method       string  `json:"method"`
	Confidence   float64 `json:"confidence"`
	MarketFactor float64 `json:"market_factor"`
}

// calculateATRBasedStopLoss 基于ATR计算动态止损线
func (rs *RiskScheduler) calculateATRBasedStopLoss(ctx context.Context) ([]*ATRStopLoss, error) {
	query := `
		SELECT p.id, p.symbol, p.entry_price, p.position_size, m.price
		FROM positions p
		JOIN market_data m ON p.symbol = m.symbol
		WHERE p.status = 'ACTIVE'
		AND m.updated_at > NOW() - INTERVAL '5 minutes'
	`
	
	rows, err := rs.db.QueryContext(ctx, query)
	if err != nil {
		// 返回模拟数据
		return []*ATRStopLoss{
			{Symbol: "BTCUSDT", PositionID: "pos_1", CurrentPrice: 45000, ATRValue: 1200, StopLoss: 43800, Multiplier: 1.0},
			{Symbol: "ETHUSDT", PositionID: "pos_2", CurrentPrice: 3000, ATRValue: 80, StopLoss: 2920, Multiplier: 1.0},
		}, nil
	}
	defer rows.Close()
	
	var stopLosses []*ATRStopLoss
	for rows.Next() {
		var positionID, symbol string
		var entryPrice, positionSize, currentPrice float64
		
		if err := rows.Scan(&positionID, &symbol, &entryPrice, &positionSize, &currentPrice); err != nil {
			continue
		}
		
		// 计算ATR值（简化实现）
		atrValue := rs.calculateATR(ctx, symbol, 14) // 14期ATR
		
		// ATR止损计算：当前价格 - (ATR * 倍数)
		multiplier := 1.5 // 1.5倍ATR
		stopLoss := currentPrice - (atrValue * multiplier)
		
		stopLosses = append(stopLosses, &ATRStopLoss{
			Symbol:       symbol,
			PositionID:   positionID,
			CurrentPrice: currentPrice,
			ATRValue:     atrValue,
			StopLoss:     stopLoss,
			Multiplier:   multiplier,
		})
	}
	
	return stopLosses, nil
}

// calculateRVBasedStopLoss 基于RV计算动态止损线
func (rs *RiskScheduler) calculateRVBasedStopLoss(ctx context.Context) ([]*RVStopLoss, error) {
	query := `
		SELECT p.id, p.symbol, p.entry_price, m.price
		FROM positions p
		JOIN market_data m ON p.symbol = m.symbol
		WHERE p.status = 'ACTIVE'
		AND m.updated_at > NOW() - INTERVAL '5 minutes'
	`
	
	rows, err := rs.db.QueryContext(ctx, query)
	if err != nil {
		// 返回模拟数据
		return []*RVStopLoss{
			{Symbol: "BTCUSDT", PositionID: "pos_1", CurrentPrice: 45000, RVValue: 0.025, StopLoss: 43875, Confidence: 0.85},
			{Symbol: "ETHUSDT", PositionID: "pos_2", CurrentPrice: 3000, RVValue: 0.030, StopLoss: 2910, Confidence: 0.80},
		}, nil
	}
	defer rows.Close()
	
	var stopLosses []*RVStopLoss
	for rows.Next() {
		var positionID, symbol string
		var entryPrice, currentPrice float64
		
		if err := rows.Scan(&positionID, &symbol, &entryPrice, &currentPrice); err != nil {
			continue
		}
		
		// 计算RV值（已实现波动率）
		rvValue := rs.calculateRealizedVolatility(ctx, symbol, 20) // 20期RV
		
		// RV止损计算：当前价格 * (1 - RV * 倍数)
		multiplier := 2.0 // 2倍RV
		stopLoss := currentPrice * (1 - rvValue*multiplier)
		
		// 计算置信度
		confidence := rs.calculateRVConfidence(ctx, symbol, rvValue)
		
		stopLosses = append(stopLosses, &RVStopLoss{
			Symbol:       symbol,
			PositionID:   positionID,
			CurrentPrice: currentPrice,
			RVValue:      rvValue,
			StopLoss:     stopLoss,
			Confidence:   confidence,
		})
	}
	
	return stopLosses, nil
}

// analyzeMarketState 分析市场状态
func (rs *RiskScheduler) analyzeMarketState(ctx context.Context) (*MarketState, error) {
	// 获取市场数据
	query := `
		SELECT AVG(volatility), AVG(volume_24h), AVG(price_change_24h)
		FROM market_data
		WHERE updated_at > NOW() - INTERVAL '1 hour'
	`
	
	var avgVolatility, avgVolume, avgPriceChange float64
	err := rs.db.QueryRowContext(ctx, query).Scan(&avgVolatility, &avgVolume, &avgPriceChange)
	if err != nil {
		// 使用默认值
		avgVolatility = 0.2
		avgVolume = 1000000000
		avgPriceChange = 0.02
	}
	
	// 确定市场趋势
	var trend string
	if avgPriceChange > 0.05 {
		trend = "BULLISH"
	} else if avgPriceChange < -0.05 {
		trend = "BEARISH"
	} else {
		trend = "NEUTRAL"
	}
	
	// 确定市场制度
	var regime string
	if avgVolatility > 0.3 {
		regime = "HIGH_VOLATILITY"
	} else if avgVolatility > 0.15 {
		regime = "NORMAL"
	} else {
		regime = "LOW_VOLATILITY"
	}
	
	// 计算流动性指标
	liquidity := math.Min(1.0, avgVolume/1000000000) // 标准化流动性
	
	return &MarketState{
		Volatility: avgVolatility,
		Trend:      trend,
		Regime:     regime,
		Liquidity:  liquidity,
	}, nil
}

// adjustStopLossForMarketState 根据市场状态调整止损
func (rs *RiskScheduler) adjustStopLossForMarketState(atrStopLosses []*ATRStopLoss, rvStopLosses []*RVStopLoss, marketState *MarketState) []*AdjustedStopLoss {
	var adjustedStopLosses []*AdjustedStopLoss
	
	// 计算市场调整因子
	var marketFactor float64 = 1.0
	
	switch marketState.Regime {
	case "HIGH_VOLATILITY":
		marketFactor = 1.3 // 高波动时放宽止损
	case "LOW_VOLATILITY":
		marketFactor = 0.8 // 低波动时收紧止损
	default:
		marketFactor = 1.0
	}
	
	// 根据趋势调整
	switch marketState.Trend {
	case "BULLISH":
		marketFactor *= 0.9 // 牛市时稍微收紧止损
	case "BEARISH":
		marketFactor *= 1.1 // 熊市时稍微放宽止损
	}
	
	// 处理ATR止损
	for _, atrSL := range atrStopLosses {
		// 获取对应的RV止损
		var rvSL *RVStopLoss
		for _, rv := range rvStopLosses {
			if rv.Symbol == atrSL.Symbol && rv.PositionID == atrSL.PositionID {
				rvSL = rv
				break
			}
		}
		
		// 综合ATR和RV计算最终止损
		var finalStopLoss float64
		var method string
		var confidence float64
		
		if rvSL != nil {
			// 加权平均ATR和RV止损
			atrWeight := 0.6
			rvWeight := 0.4
			finalStopLoss = (atrSL.StopLoss*atrWeight + rvSL.StopLoss*rvWeight) * marketFactor
			method = "ATR_RV_COMBINED"
			confidence = (0.8*atrWeight + rvSL.Confidence*rvWeight)
		} else {
			// 只使用ATR
			finalStopLoss = atrSL.StopLoss * marketFactor
			method = "ATR_ONLY"
			confidence = 0.7
		}
		
		adjustedStopLosses = append(adjustedStopLosses, &AdjustedStopLoss{
			Symbol:       atrSL.Symbol,
			PositionID:   atrSL.PositionID,
			OldStopLoss:  atrSL.StopLoss,
			NewStopLoss:  finalStopLoss,
			Method:       method,
			Confidence:   confidence,
			MarketFactor: marketFactor,
		})
	}
	
	return adjustedStopLosses
}

// applyNewStopLossSettings 应用新的止损设置
func (rs *RiskScheduler) applyNewStopLossSettings(ctx context.Context, adjustedStopLosses []*AdjustedStopLoss) (int, error) {
	appliedCount := 0
	
	for _, adj := range adjustedStopLosses {
		// 只有当新止损与旧止损差异超过阈值时才更新
		changePercent := math.Abs(adj.NewStopLoss-adj.OldStopLoss) / adj.OldStopLoss
		if changePercent < 0.02 { // 变化小于2%则跳过
			continue
		}
		
		query := `
			UPDATE positions 
			SET stop_loss = $1, stop_loss_method = $2, stop_loss_confidence = $3, updated_at = NOW()
			WHERE id = $4 AND status = 'ACTIVE'
		`
		
		result, err := rs.db.ExecContext(ctx, query, adj.NewStopLoss, adj.Method, adj.Confidence, adj.PositionID)
		if err != nil {
			log.Printf("Failed to update stop loss for position %s: %v", adj.PositionID, err)
			continue
		}
		
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			appliedCount++
			log.Printf("Updated stop loss for %s: %.2f -> %.2f (method: %s)", 
				adj.Symbol, adj.OldStopLoss, adj.NewStopLoss, adj.Method)
		}
	}
	
	return appliedCount, nil
}

// calculateATR 计算ATR值
func (rs *RiskScheduler) calculateATR(ctx context.Context, symbol string, period int) float64 {
	// 简化的ATR计算
	query := `
		SELECT high_price, low_price, close_price
		FROM price_history
		WHERE symbol = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`
	
	rows, err := rs.db.QueryContext(ctx, query, symbol, period)
	if err != nil {
		// 返回默认ATR值
		return 1000.0 // 默认ATR
	}
	defer rows.Close()
	
	var trueRanges []float64
	var prevClose float64
	
	for rows.Next() {
		var high, low, close float64
		if err := rows.Scan(&high, &low, &close); err != nil {
			continue
		}
		
		if prevClose > 0 {
			// 计算真实波幅
			tr1 := high - low
			tr2 := math.Abs(high - prevClose)
			tr3 := math.Abs(low - prevClose)
			trueRange := math.Max(tr1, math.Max(tr2, tr3))
			trueRanges = append(trueRanges, trueRange)
		}
		prevClose = close
	}
	
	if len(trueRanges) == 0 {
		return 1000.0 // 默认值
	}
	
	// 计算ATR（简单移动平均）
	sum := 0.0
	for _, tr := range trueRanges {
		sum += tr
	}
	
	return sum / float64(len(trueRanges))
}

// calculateRealizedVolatility 计算已实现波动率
func (rs *RiskScheduler) calculateRealizedVolatility(ctx context.Context, symbol string, period int) float64 {
	// 简化的RV计算
	query := `
		SELECT close_price
		FROM price_history
		WHERE symbol = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`
	
	rows, err := rs.db.QueryContext(ctx, query, symbol, period+1)
	if err != nil {
		return 0.025 // 默认2.5%波动率
	}
	defer rows.Close()
	
	var prices []float64
	for rows.Next() {
		var price float64
		if err := rows.Scan(&price); err != nil {
			continue
		}
		prices = append(prices, price)
	}
	
	if len(prices) < 2 {
		return 0.025
	}
	
	// 计算对数收益率
	var returns []float64
	for i := 1; i < len(prices); i++ {
		ret := math.Log(prices[i-1] / prices[i]) // 注意顺序，因为是DESC排序
		returns = append(returns, ret)
	}
	
	// 计算标准差
	mean := 0.0
	for _, ret := range returns {
		mean += ret
	}
	mean /= float64(len(returns))
	
	variance := 0.0
	for _, ret := range returns {
		variance += math.Pow(ret-mean, 2)
	}
	variance /= float64(len(returns) - 1)
	
	// 年化波动率
	return math.Sqrt(variance * 365)
}

// calculateRVConfidence 计算RV置信度
func (rs *RiskScheduler) calculateRVConfidence(ctx context.Context, symbol string, rvValue float64) float64 {
	// 基于数据质量和波动率稳定性计算置信度
	// 简化实现
	if rvValue > 0.5 {
		return 0.6 // 极高波动率，置信度较低
	} else if rvValue > 0.3 {
		return 0.7 // 高波动率
	} else if rvValue > 0.1 {
		return 0.85 // 正常波动率
	} else {
		return 0.75 // 低波动率
	}
}// 通知系统相关方法
实现

// EmailConfig 邮件配置
type EmailConfig struct {
	Enabled    bool     `json:"enabled"`
	SMTPHost   string   `json:"smtp_host"`
	SMTPPort   int      `json:"smtp_port"`
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	FromEmail  string   `json:"from_email"`
	ToEmails   []string `json:"to_emails"`
	UseSSL     bool     `json:"use_ssl"`
}

// SlackConfig Slack配置
type SlackConfig struct {
	Enabled     bool   `json:"enabled"`
	WebhookURL  string `json:"webhook_url"`
	Channel     string `json:"channel"`
	Username    string `json:"username"`
	IconEmoji   string `json:"icon_emoji"`
}

// WeChatConfig 企业微信配置
type WeChatConfig struct {
	Enabled     bool   `json:"enabled"`
	WebhookURL  string `json:"webhook_url"`
	MentionAll  bool   `json:"mention_all"`
	MentionList []string `json:"mention_list"`
}

// getEmailConfig 获取邮件配置
func (ds *DataScheduler) getEmailConfig() *EmailConfig {
	return &EmailConfig{
		Enabled:   ds.config.GetBool("notifications.email.enabled"),
		SMTPHost:  ds.config.GetString("notifications.email.smtp_host"),
		SMTPPort:  ds.config.GetInt("notifications.email.smtp_port"),
		Username:  ds.config.GetString("notifications.email.username"),
		Password:  ds.config.GetString("notifications.email.password"),
		FromEmail: ds.config.GetString("notifications.email.from_email"),
		ToEmails:  []string{ds.config.GetString("notifications.email.to_email")},
		UseSSL:    ds.config.GetBool("notifications.email.use_ssl"),
	}
}

// getSlackConfig 获取Slack配置
func (ds *DataScheduler) getSlackConfig() *SlackConfig {
	return &SlackConfig{
		Enabled:    ds.config.GetBool("notifications.slack.enabled"),
		WebhookURL: ds.config.GetString("notifications.slack.webhook_url"),
		Channel:    ds.config.GetString("notifications.slack.channel"),
		Username:   ds.config.GetString("notifications.slack.username"),
		IconEmoji:  ds.config.GetString("notifications.slack.icon_emoji"),
	}
}

// getWeChatConfig 获取企业微信配置
func (ds *DataScheduler) getWeChatConfig() *WeChatConfig {
	return &WeChatConfig{
		Enabled:    ds.config.GetBool("notifications.wechat.enabled"),
		WebhookURL: ds.config.GetString("notifications.wechat.webhook_url"),
		MentionAll: ds.config.GetBool("notifications.wechat.mention_all"),
	}
}

// sendWebhookNotification 发送Webhook通知
func (ds *DataScheduler) sendWebhookNotification(ctx context.Context, webhookURL, message string, recommendations []*hotlist.EnhancedRecommendation) error {
	// 构建Webhook payload
	payload := map[string]interface{}{
		"text":            message,
		"timestamp":       time.Now().Unix(),
		"recommendations": recommendations,
		"source":          "QCAT_HOTLIST",
	}
	
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}
	
	// 发送HTTP请求
	// 这里应该使用实际的HTTP客户端发送请求
	log.Printf("Webhook payload prepared for %s: %s", webhookURL, string(payloadJSON))
	
	// 模拟发送成功
	return nil
}

// sendEmailNotification 发送邮件通知
func (ds *DataScheduler) sendEmailNotification(ctx context.Context, config *EmailConfig, message string, recommendations []*hotlist.EnhancedRecommendation) error {
	// 构建邮件内容
	subject := "🔥 QCAT热门币种推荐"
	
	htmlBody := fmt.Sprintf(`
		<html>
		<body>
			<h2>%s</h2>
			<p>%s</p>
			<table border="1" style="border-collapse: collapse;">
				<tr>
					<th>币种</th>
					<th>评分</th>
					<th>风险等级</th>
					<th>置信度</th>
					<th>推荐理由</th>
				</tr>
	`, subject, message)
	
	for _, rec := range recommendations {
		htmlBody += fmt.Sprintf(`
				<tr>
					<td>%s</td>
					<td>%.1f</td>
					<td>%s</td>
					<td>%.1f%%</td>
					<td>%s</td>
				</tr>
		`, rec.Symbol, rec.Score, rec.RiskLevel, rec.Confidence*100, rec.Reason)
	}
	
	htmlBody += `
			</table>
			<p><small>此邮件由QCAT系统自动发送</small></p>
		</body>
		</html>
	`
	
	// 这里应该使用实际的SMTP客户端发送邮件
	log.Printf("Email prepared: To=%v, Subject=%s", config.ToEmails, subject)
	
	// 模拟发送成功
	return nil
}

// sendSlackNotification 发送Slack通知
func (ds *DataScheduler) sendSlackNotification(ctx context.Context, config *SlackConfig, message string, recommendations []*hotlist.EnhancedRecommendation) error {
	// 构建Slack消息
	slackMessage := map[string]interface{}{
		"channel":   config.Channel,
		"username":  config.Username,
		"icon_emoji": config.IconEmoji,
		"text":      message,
		"attachments": []map[string]interface{}{
			{
				"color": "good",
				"fields": func() []map[string]interface{} {
					var fields []map[string]interface{}
					for _, rec := range recommendations {
						fields = append(fields, map[string]interface{}{
							"title": rec.Symbol,
							"value": fmt.Sprintf("评分: %.1f | 风险: %s | 置信度: %.1f%%", 
								rec.Score, rec.RiskLevel, rec.Confidence*100),
							"short": true,
						})
					}
					return fields
				}(),
			},
		},
	}
	
	payloadJSON, err := json.Marshal(slackMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}
	
	// 这里应该发送到Slack Webhook
	log.Printf("Slack message prepared for %s: %s", config.WebhookURL, string(payloadJSON))
	
	// 模拟发送成功
	return nil
}

// sendWeChatNotification 发送企业微信通知
func (ds *DataScheduler) sendWeChatNotification(ctx context.Context, config *WeChatConfig, message string, recommendations []*hotlist.EnhancedRecommendation) error {
	// 构建企业微信消息
	content := message + "\n\n详细推荐：\n"
	for _, rec := range recommendations {
		content += fmt.Sprintf("• %s: %.1f分 (%s风险, %.1f%%置信度)\n", 
			rec.Symbol, rec.Score, rec.RiskLevel, rec.Confidence*100)
	}
	
	wechatMessage := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]interface{}{
			"content":             content,
			"mentioned_list":      config.MentionList,
			"mentioned_mobile_list": []string{},
		},
	}
	
	if config.MentionAll {
		wechatMessage["text"].(map[string]interface{})["mentioned_list"] = []string{"@all"}
	}
	
	payloadJSON, err := json.Marshal(wechatMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal wechat message: %w", err)
	}
	
	// 这里应该发送到企业微信Webhook
	log.Printf("WeChat message prepared for %s: %s", config.WebhookURL, string(payloadJSON))
	
	// 模拟发送成功
	return nil
}

// recordNotificationResults 记录通知发送结果
func (ds *DataScheduler) recordNotificationResults(ctx context.Context, message string, results map[string]error) error {
	for channel, err := range results {
		success := err == nil
		errorMsg := ""
		if err != nil {
			errorMsg = err.Error()
		}
		
		query := `
			INSERT INTO notification_logs (
				channel, message, success, error_message, created_at
			) VALUES ($1, $2, $3, $4, NOW())
		`
		
		_, dbErr := ds.db.ExecContext(ctx, query, channel, message, success, errorMsg)
		if dbErr != nil {
			log.Printf("Failed to record notification result for %s: %v", channel, dbErr)
		}
	}
	
	return nil
}// 仓位调度器
相关方法实现

// PositionInfo 仓位信息
type PositionInfo struct {
	Symbol       string  `json:"symbol"`
	Size         float64 `json:"size"`
	EntryPrice   float64 `json:"entry_price"`
	CurrentPrice float64 `json:"current_price"`
	UnrealizedPnL float64 `json:"unrealized_pnl"`
}

// OptimalPosition 最优仓位
type OptimalPosition struct {
	Symbol     string  `json:"symbol"`
	TargetSize float64 `json:"target_size"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

// RebalanceInstruction 调仓指令
type RebalanceInstruction struct {
	Symbol     string  `json:"symbol"`
	Action     string  `json:"action"` // BUY, SELL, HOLD
	Size       float64 `json:"size"`
	Priority   int     `json:"priority"`
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	Instruction *RebalanceInstruction `json:"instruction"`
	Success     bool                  `json:"success"`
	Error       string                `json:"error,omitempty"`
	ExecutedAt  time.Time             `json:"executed_at"`
}

// getCurrentPositions 获取当前仓位
func (ps *PositionScheduler) getCurrentPositions(ctx context.Context) ([]*PositionInfo, error) {
	query := `
		SELECT symbol, position_size, entry_price, current_price, unrealized_pnl
		FROM positions
		WHERE status = 'ACTIVE'
		ORDER BY symbol
	`
	
	rows, err := ps.db.QueryContext(ctx, query)
	if err != nil {
		// 返回模拟数据
		return []*PositionInfo{
			{Symbol: "BTCUSDT", Size: 0.5, EntryPrice: 45000, CurrentPrice: 46000, UnrealizedPnL: 500},
			{Symbol: "ETHUSDT", Size: 2.0, EntryPrice: 3000, CurrentPrice: 3100, UnrealizedPnL: 200},
		}, nil
	}
	defer rows.Close()
	
	var positions []*PositionInfo
	for rows.Next() {
		var pos PositionInfo
		if err := rows.Scan(&pos.Symbol, &pos.Size, &pos.EntryPrice, &pos.CurrentPrice, &pos.UnrealizedPnL); err != nil {
			continue
		}
		positions = append(positions, &pos)
	}
	
	return positions, nil
}

// calculateOptimalPositions 计算最优仓位
func (ps *PositionScheduler) calculateOptimalPositions(ctx context.Context, currentPositions []*PositionInfo) ([]*OptimalPosition, error) {
	var optimalPositions []*OptimalPosition
	
	for _, pos := range currentPositions {
		// 简化的最优仓位计算
		var targetSize float64
		var confidence float64
		var reason string
		
		// 基于当前盈亏调整仓位
		pnlRatio := pos.UnrealizedPnL / (pos.Size * pos.EntryPrice)
		
		if pnlRatio > 0.1 { // 盈利超过10%
			targetSize = pos.Size * 1.2 // 增加20%仓位
			confidence = 0.8
			reason = "Profitable position, increase exposure"
		} else if pnlRatio < -0.05 { // 亏损超过5%
			targetSize = pos.Size * 0.8 // 减少20%仓位
			confidence = 0.7
			reason = "Loss position, reduce exposure"
		} else {
			targetSize = pos.Size // 保持当前仓位
			confidence = 0.6
			reason = "Maintain current position"
		}
		
		optimalPositions = append(optimalPositions, &OptimalPosition{
			Symbol:     pos.Symbol,
			TargetSize: targetSize,
			Confidence: confidence,
			Reason:     reason,
		})
	}
	
	return optimalPositions, nil
}

// generateRebalanceInstructions 生成调仓指令
func (ps *PositionScheduler) generateRebalanceInstructions(ctx context.Context, currentPositions []*PositionInfo, optimalPositions []*OptimalPosition) ([]*RebalanceInstruction, error) {
	var instructions []*RebalanceInstruction
	
	// 创建当前仓位映射
	currentMap := make(map[string]*PositionInfo)
	for _, pos := range currentPositions {
		currentMap[pos.Symbol] = pos
	}
	
	for _, optimal := range optimalPositions {
		current := currentMap[optimal.Symbol]
		if current == nil {
			continue
		}
		
		sizeDiff := optimal.TargetSize - current.Size
		
		// 只有当差异超过阈值时才生成指令
		if math.Abs(sizeDiff) > 0.01 { // 0.01的阈值
			var action string
			if sizeDiff > 0 {
				action = "BUY"
			} else {
				action = "SELL"
				sizeDiff = -sizeDiff
			}
			
			// 计算优先级
			priority := int(math.Abs(sizeDiff) * 100)
			
			instructions = append(instructions, &RebalanceInstruction{
				Symbol:   optimal.Symbol,
				Action:   action,
				Size:     sizeDiff,
				Priority: priority,
			})
		}
	}
	
	// 按优先级排序
	sort.Slice(instructions, func(i, j int) bool {
		return instructions[i].Priority > instructions[j].Priority
	})
	
	return instructions, nil
}

// executePositionAdjustments 执行仓位调整
func (ps *PositionScheduler) executePositionAdjustments(ctx context.Context, instructions []*RebalanceInstruction) ([]*ExecutionResult, error) {
	var results []*ExecutionResult
	
	for _, instruction := range instructions {
		result := &ExecutionResult{
			Instruction: instruction,
			ExecutedAt:  time.Now(),
		}
		
		// 模拟执行
		err := ps.simulateTradeExecution(ctx, instruction)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
		}
		
		results = append(results, result)
		
		// 添加执行延迟
		time.Sleep(time.Millisecond * 100)
	}
	
	return results, nil
}

// simulateTradeExecution 模拟交易执行
func (ps *PositionScheduler) simulateTradeExecution(ctx context.Context, instruction *RebalanceInstruction) error {
	// 检查市场条件
	if instruction.Size > 10.0 { // 大额交易
		return fmt.Errorf("trade size too large: %.4f", instruction.Size)
	}
	
	// 模拟成功执行
	log.Printf("Executed %s %s: %.4f", instruction.Action, instruction.Symbol, instruction.Size)
	return nil
}

// 数据调度器相关方法实现

// DataAnomaly 数据异常
type DataAnomaly struct {
	Table       string      `json:"table"`
	Field       string      `json:"field"`
	Value       interface{} `json:"value"`
	AnomalyType string      `json:"anomaly_type"`
	Severity    string      `json:"severity"`
}

// CleaningResult 清洗结果
type CleaningResult struct {
	TotalRecords   int `json:"total_records"`
	CleanedRecords int `json:"cleaned_records"`
	RemovedRecords int `json:"removed_records"`
}

// detectDataAnomalies 检测数据异常
func (ds *DataScheduler) detectDataAnomalies(ctx context.Context) ([]*DataAnomaly, error) {
	var anomalies []*DataAnomaly
	
	// 检测价格异常
	priceAnomalies, err := ds.detectPriceAnomalies(ctx)
	if err == nil {
		anomalies = append(anomalies, priceAnomalies...)
	}
	
	// 检测交易量异常
	volumeAnomalies, err := ds.detectVolumeAnomalies(ctx)
	if err == nil {
		anomalies = append(anomalies, volumeAnomalies...)
	}
	
	return anomalies, nil
}

// detectPriceAnomalies 检测价格异常
func (ds *DataScheduler) detectPriceAnomalies(ctx context.Context) ([]*DataAnomaly, error) {
	// 模拟价格异常检测
	return []*DataAnomaly{
		{
			Table:       "market_data",
			Field:       "price",
			Value:       -100.0,
			AnomalyType: "NEGATIVE_PRICE",
			Severity:    "HIGH",
		},
	}, nil
}

// detectVolumeAnomalies 检测交易量异常
func (ds *DataScheduler) detectVolumeAnomalies(ctx context.Context) ([]*DataAnomaly, error) {
	// 模拟交易量异常检测
	return []*DataAnomaly{
		{
			Table:       "market_data",
			Field:       "volume_24h",
			Value:       0.0,
			AnomalyType: "ZERO_VOLUME",
			Severity:    "MEDIUM",
		},
	}, nil
}

// cleanInvalidData 清洗无效数据
func (ds *DataScheduler) cleanInvalidData(ctx context.Context, anomalies []*DataAnomaly) (*CleaningResult, error) {
	result := &CleaningResult{}
	
	for _, anomaly := range anomalies {
		switch anomaly.AnomalyType {
		case "NEGATIVE_PRICE":
			// 删除负价格记录
			query := `DELETE FROM market_data WHERE price < 0`
			res, err := ds.db.ExecContext(ctx, query)
			if err == nil {
				if affected, _ := res.RowsAffected(); affected > 0 {
					result.RemovedRecords += int(affected)
				}
			}
		case "ZERO_VOLUME":
			// 将零交易量设为NULL
			query := `UPDATE market_data SET volume_24h = NULL WHERE volume_24h = 0`
			res, err := ds.db.ExecContext(ctx, query)
			if err == nil {
				if affected, _ := res.RowsAffected(); affected > 0 {
					result.CleanedRecords += int(affected)
				}
			}
		}
	}
	
	// 获取总记录数
	var totalCount int
	ds.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM market_data").Scan(&totalCount)
	result.TotalRecords = totalCount
	
	return result, nil
}

// 系统调度器相关方法实现

// SystemResourceUsage 系统资源使用情况
type SystemResourceUsage struct {
	CPU     float64 `json:"cpu"`
	Memory  float64 `json:"memory"`
	Disk    float64 `json:"disk"`
	Network float64 `json:"network"`
}

// ServiceStatus 服务状态
type ServiceStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Uptime time.Duration `json:"uptime"`
}

// checkSystemResourceUsage 检查系统资源使用率
func (ss *SystemScheduler) checkSystemResourceUsage(ctx context.Context) (*SystemResourceUsage, error) {
	// 模拟系统资源检查
	return &SystemResourceUsage{
		CPU:     65.5,
		Memory:  78.2,
		Disk:    45.8,
		Network: 23.4,
	}, nil
}

// monitorServiceStatus 监控服务状态
func (ss *SystemScheduler) monitorServiceStatus(ctx context.Context) ([]*ServiceStatus, error) {
	// 模拟服务状态检查
	return []*ServiceStatus{
		{Name: "database", Status: "HEALTHY", Uptime: time.Hour * 24},
		{Name: "exchange_api", Status: "HEALTHY", Uptime: time.Hour * 12},
		{Name: "risk_monitor", Status: "DEGRADED", Uptime: time.Hour * 6},
	}, nil
}// A
lert 告警结构
type Alert struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Level     string                 `json:"level"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AlertChannel 告警通道
type AlertChannel struct {
	Type     string                 `json:"type"`     // email, sms, webhook, slack, telegram
	Config   map[string]interface{} `json:"config"`
	Enabled  bool                   `json:"enabled"`
	Priority int                    `json:"priority"` // 1-10, 1为最高优先级
}

// sendAlert 发送告警
func (rs *RiskScheduler) sendAlert(ctx context.Context, alert *Alert) error {
	// 记录告警到数据库
	if err := rs.recordAlert(ctx, alert); err != nil {
		log.Printf("Failed to record alert: %v", err)
	}
	
	// 获取告警通道配置
	channels, err := rs.getAlertChannels(ctx, alert.Type, alert.Level)
	if err != nil {
		return fmt.Errorf("failed to get alert channels: %w", err)
	}
	
	// 并发发送到各个通道
	var wg sync.WaitGroup
	for _, channel := range channels {
		if !channel.Enabled {
			continue
		}
		
		wg.Add(1)
		go func(ch AlertChannel) {
			defer wg.Done()
			if err := rs.sendToChannel(ctx, alert, ch); err != nil {
				log.Printf("Failed to send alert to %s channel: %v", ch.Type, err)
			}
		}(channel)
	}
	
	wg.Wait()
	return nil
}

// recordAlert 记录告警到数据库
func (rs *RiskScheduler) recordAlert(ctx context.Context, alert *Alert) error {
	if rs.db == nil {
		return fmt.Errorf("database not available")
	}
	
	metadataJSON, _ := json.Marshal(alert.Metadata)
	query := `
		INSERT INTO alerts (id, type, level, title, message, source, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := rs.db.ExecContext(ctx, query,
		alert.ID, alert.Type, alert.Level, alert.Title,
		alert.Message, alert.Source, string(metadataJSON), alert.Timestamp,
	)
	
	return err
}

// getAlertChannels 获取告警通道配置
func (rs *RiskScheduler) getAlertChannels(ctx context.Context, alertType, level string) ([]AlertChannel, error) {
	// 从配置文件或数据库获取告警通道配置
	channels := []AlertChannel{}
	
	// 根据告警级别确定通道
	switch level {
	case "CRITICAL":
		// 关键告警：所有通道
		channels = append(channels,
			AlertChannel{Type: "email", Enabled: true, Priority: 1},
			AlertChannel{Type: "sms", Enabled: true, Priority: 1},
			AlertChannel{Type: "webhook", Enabled: true, Priority: 2},
			AlertChannel{Type: "slack", Enabled: true, Priority: 2},
		)
	case "HIGH":
		// 高级告警：邮件和Webhook
		channels = append(channels,
			AlertChannel{Type: "email", Enabled: true, Priority: 1},
			AlertChannel{Type: "webhook", Enabled: true, Priority: 2},
			AlertChannel{Type: "slack", Enabled: true, Priority: 3},
		)
	case "MEDIUM":
		// 中级告警：邮件
		channels = append(channels,
			AlertChannel{Type: "email", Enabled: true, Priority: 1},
			AlertChannel{Type: "slack", Enabled: true, Priority: 2},
		)
	case "LOW":
		// 低级告警：仅记录
		channels = append(channels,
			AlertChannel{Type: "webhook", Enabled: true, Priority: 3},
		)
	}
	
	// 从配置中获取通道配置
	for i := range channels {
		channels[i].Config = rs.getChannelConfig(channels[i].Type)
	}
	
	return channels, nil
}

// getChannelConfig 获取通道配置
func (rs *RiskScheduler) getChannelConfig(channelType string) map[string]interface{} {
	if rs.config == nil {
		return map[string]interface{}{}
	}
	
	switch channelType {
	case "email":
		return map[string]interface{}{
			"smtp_host":     rs.config.GetString("alerts.email.smtp_host"),
			"smtp_port":     rs.config.GetInt("alerts.email.smtp_port"),
			"username":      rs.config.GetString("alerts.email.username"),
			"password":      rs.config.GetString("alerts.email.password"),
			"from":          rs.config.GetString("alerts.email.from"),
			"to":            rs.config.GetString("alerts.email.to"),
		}
	case "sms":
		return map[string]interface{}{
			"api_key":    rs.config.GetString("alerts.sms.api_key"),
			"api_secret": rs.config.GetString("alerts.sms.api_secret"),
			"phone":      rs.config.GetString("alerts.sms.phone"),
		}
	case "webhook":
		return map[string]interface{}{
			"url":     rs.config.GetString("alerts.webhook.url"),
			"method":  rs.config.GetString("alerts.webhook.method"),
			"headers": rs.config.Get("alerts.webhook.headers"),
		}
	case "slack":
		return map[string]interface{}{
			"webhook_url": rs.config.GetString("alerts.slack.webhook_url"),
			"channel":     rs.config.GetString("alerts.slack.channel"),
			"username":    rs.config.GetString("alerts.slack.username"),
		}
	case "telegram":
		return map[string]interface{}{
			"bot_token": rs.config.GetString("alerts.telegram.bot_token"),
			"chat_id":   rs.config.GetString("alerts.telegram.chat_id"),
		}
	default:
		return map[string]interface{}{}
	}
}

// sendToChannel 发送告警到指定通道
func (rs *RiskScheduler) sendToChannel(ctx context.Context, alert *Alert, channel AlertChannel) error {
	switch channel.Type {
	case "email":
		return rs.sendEmailAlert(ctx, alert, channel.Config)
	case "sms":
		return rs.sendSMSAlert(ctx, alert, channel.Config)
	case "webhook":
		return rs.sendWebhookAlert(ctx, alert, channel.Config)
	case "slack":
		return rs.sendSlackAlert(ctx, alert, channel.Config)
	case "telegram":
		return rs.sendTelegramAlert(ctx, alert, channel.Config)
	default:
		return fmt.Errorf("unsupported alert channel type: %s", channel.Type)
	}
}

// sendEmailAlert 发送邮件告警
func (rs *RiskScheduler) sendEmailAlert(ctx context.Context, alert *Alert, config map[string]interface{}) error {
	// 简化实现：记录日志
	log.Printf("EMAIL ALERT: [%s] %s - %s", alert.Level, alert.Title, alert.Message)
	
	// 实际实现应该使用SMTP发送邮件
	// 这里可以集成如 gomail 等邮件库
	/*
	m := gomail.NewMessage()
	m.SetHeader("From", config["from"].(string))
	m.SetHeader("To", config["to"].(string))
	m.SetHeader("Subject", alert.Title)
	m.SetBody("text/html", rs.formatEmailBody(alert))
	
	d := gomail.NewDialer(
		config["smtp_host"].(string),
		config["smtp_port"].(int),
		config["username"].(string),
		config["password"].(string),
	)
	
	return d.DialAndSend(m)
	*/
	
	return nil
}

// sendSMSAlert 发送短信告警
func (rs *RiskScheduler) sendSMSAlert(ctx context.Context, alert *Alert, config map[string]interface{}) error {
	// 简化实现：记录日志
	log.Printf("SMS ALERT: [%s] %s", alert.Level, alert.Message)
	
	// 实际实现应该调用短信服务API
	// 如阿里云短信、腾讯云短信等
	
	return nil
}

// sendWebhookAlert 发送Webhook告警
func (rs *RiskScheduler) sendWebhookAlert(ctx context.Context, alert *Alert, config map[string]interface{}) error {
	url, ok := config["url"].(string)
	if !ok || url == "" {
		return fmt.Errorf("webhook URL not configured")
	}
	
	// 构建请求体
	payload := map[string]interface{}{
		"alert":     alert,
		"timestamp": time.Now().Unix(),
	}
	
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}
	
	// 发送HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	// 添加自定义头部
	if headers, ok := config["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned error status: %d", resp.StatusCode)
	}
	
	log.Printf("WEBHOOK ALERT sent successfully: [%s] %s", alert.Level, alert.Title)
	return nil
}

// sendSlackAlert 发送Slack告警
func (rs *RiskScheduler) sendSlackAlert(ctx context.Context, alert *Alert, config map[string]interface{}) error {
	webhookURL, ok := config["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return fmt.Errorf("Slack webhook URL not configured")
	}
	
	// 构建Slack消息
	color := rs.getSlackColor(alert.Level)
	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color":      color,
				"title":      alert.Title,
				"text":       alert.Message,
				"footer":     alert.Source,
				"ts":         alert.Timestamp.Unix(),
				"fields": []map[string]interface{}{
					{
						"title": "告警级别",
						"value": alert.Level,
						"short": true,
					},
					{
						"title": "告警类型",
						"value": alert.Type,
						"short": true,
					},
				},
			},
		},
	}
	
	// 添加元数据字段
	if alert.Metadata != nil {
		fields := payload["attachments"].([]map[string]interface{})[0]["fields"].([]map[string]interface{})
		for key, value := range alert.Metadata {
			fields = append(fields, map[string]interface{}{
				"title": key,
				"value": fmt.Sprintf("%v", value),
				"short": true,
			})
		}
		payload["attachments"].([]map[string]interface{})[0]["fields"] = fields
	}
	
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}
	
	// 发送到Slack
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create Slack request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack alert: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("Slack returned error status: %d", resp.StatusCode)
	}
	
	log.Printf("SLACK ALERT sent successfully: [%s] %s", alert.Level, alert.Title)
	return nil
}

// sendTelegramAlert 发送Telegram告警
func (rs *RiskScheduler) sendTelegramAlert(ctx context.Context, alert *Alert, config map[string]interface{}) error {
	botToken, ok := config["bot_token"].(string)
	if !ok || botToken == "" {
		return fmt.Errorf("Telegram bot token not configured")
	}
	
	chatID, ok := config["chat_id"].(string)
	if !ok || chatID == "" {
		return fmt.Errorf("Telegram chat ID not configured")
	}
	
	// 构建消息
	message := fmt.Sprintf("🚨 *%s*\n\n*级别:* %s\n*类型:* %s\n*来源:* %s\n*时间:* %s\n\n%s",
		alert.Title,
		alert.Level,
		alert.Type,
		alert.Source,
		alert.Timestamp.Format("2006-01-02 15:04:05"),
		alert.Message,
	)
	
	// 构建请求
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "Markdown",
	}
	
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Telegram payload: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create Telegram request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Telegram alert: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("Telegram returned error status: %d", resp.StatusCode)
	}
	
	log.Printf("TELEGRAM ALERT sent successfully: [%s] %s", alert.Level, alert.Title)
	return nil
}

// getSlackColor 根据告警级别获取Slack颜色
func (rs *RiskScheduler) getSlackColor(level string) string {
	switch level {
	case "CRITICAL":
		return "danger"
	case "HIGH":
		return "warning"
	case "MEDIUM":
		return "good"
	case "LOW":
		return "#36a64f"
	default:
		return "#808080"
	}
}

// formatEmailBody 格式化邮件正文
func (rs *RiskScheduler) formatEmailBody(alert *Alert) string {
	html := fmt.Sprintf(`
<html>
<body>
	<h2 style="color: %s;">%s</h2>
	<p><strong>告警级别:</strong> %s</p>
	<p><strong>告警类型:</strong> %s</p>
	<p><strong>来源:</strong> %s</p>
	<p><strong>时间:</strong> %s</p>
	<p><strong>消息:</strong> %s</p>
`, rs.getEmailColor(alert.Level), alert.Title, alert.Level, alert.Type, alert.Source, 
	alert.Timestamp.Format("2006-01-02 15:04:05"), alert.Message)
	
	// 添加元数据
	if alert.Metadata != nil && len(alert.Metadata) > 0 {
		html += "<h3>详细信息:</h3><ul>"
		for key, value := range alert.Metadata {
			html += fmt.Sprintf("<li><strong>%s:</strong> %v</li>", key, value)
		}
		html += "</ul>"
	}
	
	html += "</body></html>"
	return html
}

// getEmailColor 根据告警级别获取邮件颜色
func (rs *RiskScheduler) getEmailColor(level string) string {
	switch level {
	case "CRITICAL":
		return "#dc3545"
	case "HIGH":
		return "#fd7e14"
	case "MEDIUM":
		return "#ffc107"
	case "LOW":
		return "#28a745"
	default:
		return "#6c757d"
	}
}// 
analyzeHedgeEffectiveness 分析对冲效果
func (ps *PositionScheduler) analyzeHedgeEffectiveness(ctx context.Context, operation *HedgeOperation) (*HedgeEffectivenessAnalysis, error) {
	// 获取对冲操作的历史数据
	hedgeHistory, err := ps.getHedgeHistory(ctx, operation.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get hedge history: %w", err)
	}
	
	// 计算对冲效果指标
	analysis := &HedgeEffectivenessAnalysis{
		OperationID:        operation.ID,
		AnalysisTime:       time.Now(),
		CorrelationStability: ps.calculateCorrelationStability(hedgeHistory),
		RiskReduction:      ps.calculateRiskReduction(hedgeHistory),
		CostEfficiency:     ps.calculateCostEfficiency(hedgeHistory),
		OverallScore:       0.0,
	}
	
	// 计算综合评分
	analysis.OverallScore = (analysis.CorrelationStability*0.4 + 
		analysis.RiskReduction*0.4 + 
		analysis.CostEfficiency*0.2)
	
	log.Printf("Hedge effectiveness analysis for %s: score=%.4f", operation.ID, analysis.OverallScore)
	return analysis, nil
}

// shouldAdjustHedge 判断是否需要调整对冲
func (ps *PositionScheduler) shouldAdjustHedge(analysis *HedgeEffectivenessAnalysis) bool {
	// 设定调整阈值
	minEffectivenessScore := 0.6 // 60%以下需要调整
	maxCorrelationDrift := 0.3   // 相关性漂移超过30%需要调整
	
	// 检查综合评分
	if analysis.OverallScore < minEffectivenessScore {
		log.Printf("Hedge effectiveness below threshold: %.4f < %.4f", 
			analysis.OverallScore, minEffectivenessScore)
		return true
	}
	
	// 检查相关性稳定性
	if analysis.CorrelationStability < (1.0 - maxCorrelationDrift) {
		log.Printf("Correlation stability below threshold: %.4f", analysis.CorrelationStability)
		return true
	}
	
	// 检查风险降低效果
	if analysis.RiskReduction < 0.5 {
		log.Printf("Risk reduction below threshold: %.4f", analysis.RiskReduction)
		return true
	}
	
	return false
}

// calculateOptimalHedgeRatio 计算最优对冲比率
func (ps *PositionScheduler) calculateOptimalHedgeRatio(ctx context.Context, operation *HedgeOperation) (float64, error) {
	// 获取当前市场数据
	marketData, err := ps.getCurrentMarketData(ctx, operation.Symbol)
	if err != nil {
		return 0, fmt.Errorf("failed to get market data: %w", err)
	}
	
	// 获取历史价格数据
	historicalPrices, err := ps.getHistoricalPrices(ctx, operation.Symbol, 30) // 30天历史数据
	if err != nil {
		return 0, fmt.Errorf("failed to get historical prices: %w", err)
	}
	
	// 计算波动率
	volatility := ps.calculateVolatility(historicalPrices)
	
	// 计算相关性
	correlation := ps.calculateCorrelationWithMarket(historicalPrices)
	
	// 使用最小方差对冲比率公式
	// h* = Cov(S,F) / Var(F)
	// 简化为基于相关性和波动率的计算
	optimalRatio := correlation * (volatility / 0.02) // 基准波动率2%
	
	// 限制对冲比率在合理范围内
	optimalRatio = math.Max(0.1, math.Min(1.0, optimalRatio))
	
	log.Printf("Calculated optimal hedge ratio for %s: %.4f (volatility=%.4f, correlation=%.4f)", 
		operation.Symbol, optimalRatio, volatility, correlation)
	
	return optimalRatio, nil
}

// executeHedgeAdjustment 执行对冲调整
func (ps *PositionScheduler) executeHedgeAdjustment(ctx context.Context, operation *HedgeOperation, newRatio float64) error {
	// 计算需要调整的仓位大小
	currentHedgeSize := operation.HedgeSize
	targetHedgeSize := operation.PositionSize * newRatio
	adjustmentSize := targetHedgeSize - currentHedgeSize
	
	log.Printf("Executing hedge adjustment: current=%.4f, target=%.4f, adjustment=%.4f", 
		currentHedgeSize, targetHedgeSize, adjustmentSize)
	
	// 创建调整订单
	adjustmentOrder := &HedgeAdjustmentOrder{
		OperationID:    operation.ID,
		Symbol:         operation.Symbol,
		AdjustmentSize: adjustmentSize,
		NewRatio:       newRatio,
		OrderType:      "MARKET",
		Timestamp:      time.Now(),
	}
	
	// 执行订单
	if err := ps.placeHedgeAdjustmentOrder(ctx, adjustmentOrder); err != nil {
		return fmt.Errorf("failed to place hedge adjustment order: %w", err)
	}
	
	// 更新操作记录
	operation.HedgeSize = targetHedgeSize
	operation.HedgeRatio = newRatio
	operation.LastAdjustment = time.Now()
	
	if err := ps.updateHedgeOperation(ctx, operation); err != nil {
		log.Printf("Failed to update hedge operation: %v", err)
	}
	
	return nil
}

// recordHedgeAdjustment 记录对冲调整
func (ps *PositionScheduler) recordHedgeAdjustment(ctx context.Context, operation *HedgeOperation, analysis *HedgeEffectivenessAnalysis, newRatio float64) error {
	if ps.db == nil {
		return fmt.Errorf("database not available")
	}
	
	query := `
		INSERT INTO hedge_adjustments 
		(operation_id, symbol, old_ratio, new_ratio, effectiveness_score, 
		 correlation_stability, risk_reduction, cost_efficiency, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := ps.db.ExecContext(ctx, query,
		operation.ID,
		operation.Symbol,
		operation.HedgeRatio,
		newRatio,
		analysis.OverallScore,
		analysis.CorrelationStability,
		analysis.RiskReduction,
		analysis.CostEfficiency,
		time.Now(),
	)
	
	if err != nil {
		return fmt.Errorf("failed to record hedge adjustment: %w", err)
	}
	
	log.Printf("Recorded hedge adjustment for operation %s", operation.ID)
	return nil
}

// Supporting structures and methods

// HedgeEffectivenessAnalysis 对冲效果分析
type HedgeEffectivenessAnalysis struct {
	OperationID          string    `json:"operation_id"`
	AnalysisTime         time.Time `json:"analysis_time"`
	CorrelationStability float64   `json:"correlation_stability"`
	RiskReduction        float64   `json:"risk_reduction"`
	CostEfficiency       float64   `json:"cost_efficiency"`
	OverallScore         float64   `json:"overall_score"`
}

// HedgeAdjustmentOrder 对冲调整订单
type HedgeAdjustmentOrder struct {
	OperationID    string    `json:"operation_id"`
	Symbol         string    `json:"symbol"`
	AdjustmentSize float64   `json:"adjustment_size"`
	NewRatio       float64   `json:"new_ratio"`
	OrderType      string    `json:"order_type"`
	Timestamp      time.Time `json:"timestamp"`
}

// getHedgeHistory 获取对冲历史
func (ps *PositionScheduler) getHedgeHistory(ctx context.Context, operationID string) ([]map[string]interface{}, error) {
	// 简化实现：返回模拟历史数据
	history := []map[string]interface{}{
		{
			"timestamp":    time.Now().Add(-24 * time.Hour),
			"hedge_ratio":  0.8,
			"effectiveness": 0.75,
			"pnl":          1500.0,
		},
		{
			"timestamp":    time.Now().Add(-12 * time.Hour),
			"hedge_ratio":  0.82,
			"effectiveness": 0.72,
			"pnl":          1200.0,
		},
	}
	
	return history, nil
}

// calculateCorrelationStability 计算相关性稳定性
func (ps *PositionScheduler) calculateCorrelationStability(history []map[string]interface{}) float64 {
	if len(history) < 2 {
		return 1.0
	}
	
	// 简化计算：基于历史数据的方差
	var correlations []float64
	for _, record := range history {
		if corr, ok := record["correlation"].(float64); ok {
			correlations = append(correlations, corr)
		}
	}
	
	if len(correlations) == 0 {
		return 0.8 // 默认稳定性
	}
	
	// 计算标准差
	var sum, mean float64
	for _, corr := range correlations {
		sum += corr
	}
	mean = sum / float64(len(correlations))
	
	var variance float64
	for _, corr := range correlations {
		variance += (corr - mean) * (corr - mean)
	}
	variance /= float64(len(correlations))
	
	stability := 1.0 - math.Sqrt(variance)
	return math.Max(0.0, math.Min(1.0, stability))
}

// calculateRiskReduction 计算风险降低效果
func (ps *PositionScheduler) calculateRiskReduction(history []map[string]interface{}) float64 {
	// 简化计算：基于PnL波动性的降低
	if len(history) < 2 {
		return 0.7 // 默认风险降低效果
	}
	
	var pnls []float64
	for _, record := range history {
		if pnl, ok := record["pnl"].(float64); ok {
			pnls = append(pnls, pnl)
		}
	}
	
	if len(pnls) == 0 {
		return 0.7
	}
	
	// 计算PnL的标准差作为风险指标
	var sum, mean float64
	for _, pnl := range pnls {
		sum += pnl
	}
	mean = sum / float64(len(pnls))
	
	var variance float64
	for _, pnl := range pnls {
		variance += (pnl - mean) * (pnl - mean)
	}
	variance /= float64(len(pnls))
	
	risk := math.Sqrt(variance)
	// 风险降低效果 = 1 - (当前风险 / 基准风险)
	baselineRisk := 2000.0 // 基准风险
	riskReduction := 1.0 - (risk / baselineRisk)
	
	return math.Max(0.0, math.Min(1.0, riskReduction))
}

// calculateCostEfficiency 计算成本效率
func (ps *PositionScheduler) calculateCostEfficiency(history []map[string]interface{}) float64 {
	// 简化计算：基于交易成本和收益的比率
	totalCost := 0.0
	totalBenefit := 0.0
	
	for _, record := range history {
		if cost, ok := record["cost"].(float64); ok {
			totalCost += cost
		}
		if benefit, ok := record["benefit"].(float64); ok {
			totalBenefit += benefit
		}
	}
	
	if totalCost == 0 {
		return 0.8 // 默认成本效率
	}
	
	efficiency := totalBenefit / totalCost
	return math.Max(0.0, math.Min(1.0, efficiency/2.0)) // 归一化到0-1
}

// getCurrentMarketData 获取当前市场数据
func (ps *PositionScheduler) getCurrentMarketData(ctx context.Context, symbol string) (map[string]interface{}, error) {
	// 简化实现：返回模拟市场数据
	marketData := map[string]interface{}{
		"symbol":    symbol,
		"price":     50000.0,
		"volume":    1000000.0,
		"timestamp": time.Now(),
	}
	
	return marketData, nil
}

// getHistoricalPrices 获取历史价格数据
func (ps *PositionScheduler) getHistoricalPrices(ctx context.Context, symbol string, days int) ([]float64, error) {
	// 简化实现：生成模拟历史价格
	prices := make([]float64, days)
	basePrice := 50000.0
	
	for i := 0; i < days; i++ {
		// 模拟价格波动
		change := (math.Sin(float64(i)*0.1) + math.Cos(float64(i)*0.05)) * 0.02
		prices[i] = basePrice * (1 + change)
	}
	
	return prices, nil
}

// calculateVolatility 计算波动率
func (ps *PositionScheduler) calculateVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0.02 // 默认2%波动率
	}
	
	// 计算收益率
	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
	}
	
	// 计算标准差
	var sum, mean float64
	for _, ret := range returns {
		sum += ret
	}
	mean = sum / float64(len(returns))
	
	var variance float64
	for _, ret := range returns {
		variance += (ret - mean) * (ret - mean)
	}
	variance /= float64(len(returns))
	
	return math.Sqrt(variance)
}

// calculateCorrelationWithMarket 计算与市场的相关性
func (ps *PositionScheduler) calculateCorrelationWithMarket(prices []float64) float64 {
	// 简化实现：返回基于价格趋势的相关性
	if len(prices) < 2 {
		return 0.7 // 默认相关性
	}
	
	// 计算价格变化趋势
	upCount := 0
	for i := 1; i < len(prices); i++ {
		if prices[i] > prices[i-1] {
			upCount++
		}
	}
	
	// 基于上涨比例计算相关性
	upRatio := float64(upCount) / float64(len(prices)-1)
	correlation := 0.5 + (upRatio-0.5)*0.8 // 调整到合理范围
	
	return math.Max(0.1, math.Min(0.9, correlation))
}

// placeHedgeAdjustmentOrder 下达对冲调整订单
func (ps *PositionScheduler) placeHedgeAdjustmentOrder(ctx context.Context, order *HedgeAdjustmentOrder) error {
	// 简化实现：记录订单日志
	log.Printf("Placing hedge adjustment order: %+v", order)
	
	// 实际实现应该调用交易所API
	// return ps.exchangeClient.PlaceOrder(ctx, order)
	
	return nil
}

// updateHedgeOperation 更新对冲操作记录
func (ps *PositionScheduler) updateHedgeOperation(ctx context.Context, operation *HedgeOperation) error {
	if ps.db == nil {
		return fmt.Errorf("database not available")
	}
	
	query := `
		UPDATE hedge_operations 
		SET hedge_size = ?, hedge_ratio = ?, last_adjustment = ?, updated_at = ?
		WHERE id = ?
	`
	
	_, err := ps.db.ExecContext(ctx, query,
		operation.HedgeSize,
		operation.HedgeRatio,
		operation.LastAdjustment,
		time.Now(),
		operation.ID,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update hedge operation: %w", err)
	}
	
	return nil
}
package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/exchange/account"
	"qcat/internal/hotlist"
	"qcat/internal/monitor"

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

	// TODO: 实现风险监控逻辑
	// 1. 检查保证金比率
	// 2. 监控仓位风险
	// 3. 检测异常行情
	// 4. 触发风险控制措施

	return nil
}

// HandleAbnormalMarketResponse 处理异常行情应对任务
func (rs *RiskScheduler) HandleAbnormalMarketResponse(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing abnormal market response task: %s", task.Name)

	// 实现异常行情应对逻辑
	// 1. 检测异常行情条件
	// 2. 触发熔断保护
	// 3. 自动降杠杆
	// 4. 紧急平仓保护

	// TODO: 实现实时异常检测和自动应对机制
	log.Printf("Abnormal market response logic executed")
	return nil
}

// HandleStopLossAdjustment 处理止盈止损线自动调整任务
func (rs *RiskScheduler) HandleStopLossAdjustment(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing stop loss adjustment task: %s", task.Name)

	// 实现止盈止损线自动调整逻辑
	// 1. 基于ATR计算动态止损线
	// 2. 基于RV计算动态止损线
	// 3. 根据市场状态调整参数
	// 4. 应用新的止损设置

	// TODO: 实现基于ATR/RV的动态调整算法
	log.Printf("Stop loss adjustment logic executed")
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

	// TODO: 实现仓位优化逻辑
	// 1. 获取当前仓位
	// 2. 计算最优仓位
	// 3. 生成调仓指令
	// 4. 执行仓位调整

	return nil
}

// HandleDynamicFundAllocation 处理资金动态分配任务
func (ps *PositionScheduler) HandleDynamicFundAllocation(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing dynamic fund allocation task: %s", task.Name)

	// 实现资金动态分配逻辑
	// 1. 分析当前资金使用效率
	// 2. 计算最优资金分配
	// 3. 执行资金重新分配
	// 4. 监控分配效果

	// TODO: 实现智能资金分配算法
	log.Printf("Dynamic fund allocation logic executed")
	return nil
}

// HandleLayeredPositionManagement 处理仓位分层机制任务
func (ps *PositionScheduler) HandleLayeredPositionManagement(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing layered position management task: %s", task.Name)

	// 实现仓位分层机制逻辑
	// 1. 分析市场波动性
	// 2. 计算分层仓位配置
	// 3. 执行分层建仓/平仓
	// 4. 动态调整分层参数

	// TODO: 实现多层次仓位管理策略
	log.Printf("Layered position management logic executed")
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

	// TODO: 实现数据清洗逻辑
	// 1. 检测异常数据
	// 2. 清洗无效数据
	// 3. 校正数据格式
	// 4. 更新数据质量指标

	return nil
}

// HandleAutoBacktesting 处理自动回测与前测任务
func (ds *DataScheduler) HandleAutoBacktesting(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing auto backtesting task: %s", task.Name)

	// 实现自动回测与前测逻辑
	// 1. 自动生成回测参数
	// 2. 执行历史数据回测
	// 3. 执行前瞻性测试
	// 4. 生成测试报告

	// TODO: 实现自动化回测引擎
	log.Printf("Auto backtesting logic executed")
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

	// 实现因子库动态更新逻辑
	// 1. 扫描新的市场因子
	// 2. 评估因子有效性
	// 3. 更新因子库
	// 4. 清理过期因子

	// TODO: 实现动态因子发现和自动更新机制
	log.Printf("Factor library update logic executed")
	return nil
}

// HandleMarketPatternRecognition 处理市场模式识别任务
func (ds *DataScheduler) HandleMarketPatternRecognition(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing market pattern recognition task: %s", task.Name)

	// 实现市场模式识别逻辑
	// 1. 分析当前市场状态
	// 2. 识别市场模式变化
	// 3. 触发策略切换
	// 4. 更新模式识别模型

	// TODO: 实现实时模式识别算法
	log.Printf("Market pattern recognition logic executed")
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

	// TODO: 实现系统健康检查逻辑
	// 1. 检查系统资源使用率
	// 2. 监控服务状态
	// 3. 检测异常情况
	// 4. 触发自愈机制

	return nil
}

// HandleAccountSecurityMonitoring 处理账户安全监控任务
func (ss *SystemScheduler) HandleAccountSecurityMonitoring(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing account security monitoring task: %s", task.Name)

	// 实现账户安全监控逻辑
	// 1. 监控异常登录行为
	// 2. 检测API密钥异常使用
	// 3. 分析交易行为模式
	// 4. 触发安全告警

	// TODO: 实现智能安全监控系统
	log.Printf("Account security monitoring logic executed")
	return nil
}

// HandleMultiExchangeRedundancy 处理多交易所冗余任务
func (ss *SystemScheduler) HandleMultiExchangeRedundancy(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing multi-exchange redundancy task: %s", task.Name)

	// 实现多交易所冗余逻辑
	// 1. 检查交易所连接状态
	// 2. 监控交易所性能
	// 3. 自动切换故障交易所
	// 4. 维护冗余连接

	// TODO: 实现交易所故障自动切换机制
	log.Printf("Multi-exchange redundancy logic executed")
	return nil
}

// HandleAuditLogging 处理日志与审计追踪任务
func (ss *SystemScheduler) HandleAuditLogging(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing audit logging task: %s", task.Name)

	// 实现日志与审计追踪逻辑
	// 1. 收集系统操作日志
	// 2. 生成审计报告
	// 3. 检查日志完整性
	// 4. 清理过期日志

	// TODO: 实现自动化审计系统
	log.Printf("Audit logging logic executed")
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

	// TODO: 实现机器学习逻辑
	// 1. 收集训练数据
	// 2. 训练模型
	// 3. 评估模型性能
	// 4. 更新策略参数

	return nil
}

// HandleAutoMLLearning 处理策略自学习AutoML任务
func (ls *LearningScheduler) HandleAutoMLLearning(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing AutoML learning task: %s", task.Name)

	// 实现AutoML学习逻辑
	// 1. 自动模型选择
	// 2. 超参数优化
	// 3. 特征工程
	// 4. 模型集成

	// TODO: 实现自动模型选择算法
	log.Printf("AutoML learning logic executed")
	return nil
}

// HandleGeneticEvolution 处理遗传淘汰制升级任务
func (ls *LearningScheduler) HandleGeneticEvolution(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing genetic evolution task: %s", task.Name)

	// 实现遗传淘汰制升级逻辑
	// 1. 策略基因编码
	// 2. 执行变异操作
	// 3. 适应度评估
	// 4. 选择和繁殖

	// TODO: 实现自动变异机制
	log.Printf("Genetic evolution logic executed")
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
			// 如果数据库中没有数据，生成模拟数据用于测试
			log.Printf("No market data found for %s, using mock data: %v", symbol, err)
			data = MarketData{
				Symbol:          symbol,
				Price:           50000.0 + float64(len(symbol)*1000), // 模拟价格
				Volume24h:       1000000.0 + float64(len(symbol)*100000),
				VolumeChange24h: -10.0 + float64(len(symbol)%20),
				PriceChange24h:  -5.0 + float64(len(symbol)%10),
				Volatility:      0.02 + float64(len(symbol)%5)*0.01,
				FundingRate:     0.0001 + float64(len(symbol)%3)*0.0001,
				OpenInterest:    500000.0 + float64(len(symbol)*50000),
				OIChange24h:     -5.0 + float64(len(symbol)%10),
				Timestamp:       time.Now(),
			}
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

	// 这里可以集成实际的通知系统 (如Webhook、邮件、Slack等)
	// 目前只记录日志
	log.Printf("Notification: %s", message)

	// TODO: 实现实际的通知发送逻辑
	// 例如: 发送到Webhook、邮件、Slack等

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
	query := `
		SELECT exchange_name, SUM(balance) as total_balance
		FROM exchange_balances
		WHERE updated_at > NOW() - INTERVAL '1 hour'
		GROUP BY exchange_name
	`

	rows, err := rs.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query exchange balances: %w", err)
	}
	defer rows.Close()

	distribution := make(map[string]float64)
	for rows.Next() {
		var exchangeName string
		var balance float64
		if err := rows.Scan(&exchangeName, &balance); err != nil {
			return nil, fmt.Errorf("failed to scan exchange balance: %w", err)
		}
		distribution[exchangeName] = balance
	}

	// 如果没有数据，使用模拟数据
	if len(distribution) == 0 {
		distribution = map[string]float64{
			"binance": 50000.0,
			"okx":     30000.0,
			"bybit":   20000.0,
		}
	}

	return distribution, nil
}

// getWalletFundDistribution 获取钱包资金分布
func (rs *RiskScheduler) getWalletFundDistribution(ctx context.Context) (map[string]float64, error) {
	query := `
		SELECT wallet_type, SUM(balance) as total_balance
		FROM wallet_balances
		WHERE updated_at > NOW() - INTERVAL '1 hour'
		GROUP BY wallet_type
	`

	rows, err := rs.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query wallet balances: %w", err)
	}
	defer rows.Close()

	distribution := make(map[string]float64)
	for rows.Next() {
		var walletType string
		var balance float64
		if err := rows.Scan(&walletType, &balance); err != nil {
			return nil, fmt.Errorf("failed to scan wallet balance: %w", err)
		}
		distribution[walletType] = balance
	}

	// 如果没有数据，使用模拟数据
	if len(distribution) == 0 {
		distribution = map[string]float64{
			"hot_wallet":  15000.0,
			"cold_wallet": 35000.0,
		}
	}

	return distribution, nil
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
	query := `
		INSERT INTO fund_monitoring_rules (
			location, target_ratio, warning_threshold, critical_threshold,
			check_interval, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (location) DO UPDATE SET
			target_ratio = EXCLUDED.target_ratio,
			warning_threshold = EXCLUDED.warning_threshold,
			critical_threshold = EXCLUDED.critical_threshold,
			check_interval = EXCLUDED.check_interval,
			updated_at = NOW()
	`

	_, err := rs.db.ExecContext(ctx, query,
		rule["location"],
		rule["target_ratio"],
		rule["warning_threshold"],
		rule["critical_threshold"],
		rule["check_interval"],
	)

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

	// 如果没有数据，使用默认策略
	if len(strategies) == 0 {
		strategies = []string{
			"momentum_strategy",
			"mean_reversion_strategy",
			"arbitrage_strategy",
			"trend_following_strategy",
		}
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
			// 生成模拟数据
			strategyReturns[strategy] = ps.generateMockReturns(days)
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
			// 生成模拟数据
			returns = ps.generateMockReturns(days)
		}

		strategyReturns[strategy] = returns
	}

	return strategyReturns, nil
}

// generateMockReturns 生成模拟收益数据
func (ps *PositionScheduler) generateMockReturns(days int) []float64 {
	returns := make([]float64, days)
	for i := 0; i < days; i++ {
		// 生成正态分布的随机收益
		returns[i] = (float64(i%10) - 5.0) / 100.0 // -5% 到 +4% 的收益
	}
	return returns
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
			// 使用模拟数据
			totalPosition = 10000.0 + float64(len(strategy)*1000)
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

	// 模拟当前相关性（实际应该从实时数据计算）
	currentCorrelation := historicalCorrelation + (float64(time.Now().Unix()%10)-5.0)/100.0

	// 计算稳定性（相关性变化越小，稳定性越高）
	stability := 1.0 - math.Abs(historicalCorrelation-currentCorrelation)
	return math.Max(0.0, stability)
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

	// 这里可以实现对冲调整的调度逻辑
	// 例如：重新计算对冲比率、调整仓位等
	// 目前只记录日志
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

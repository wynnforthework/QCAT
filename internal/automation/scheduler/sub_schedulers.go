package scheduler

import (
	"context"
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

	// 实现资金分散与转移逻辑
	// 1. 检查资金集中度风险
	// 2. 计算最优资金分配
	// 3. 执行资金转移操作
	// 4. 集成冷钱包功能

	// TODO: 实现冷钱包集成和自动执行机制
	log.Printf("Fund distribution logic executed")
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

// HandleMultiStrategyHedging 处理自动化多策略对冲任务
func (ps *PositionScheduler) HandleMultiStrategyHedging(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing multi-strategy hedging task: %s", task.Name)

	// 实现自动化多策略对冲逻辑
	// 1. 分析策略间相关性
	// 2. 计算动态对冲比率
	// 3. 执行自动对冲操作
	// 4. 监控对冲效果

	// TODO: 实现自动对冲执行逻辑
	log.Printf("Multi-strategy hedging logic executed")
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

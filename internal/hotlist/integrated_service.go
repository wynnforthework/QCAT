package hotlist

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/config"
	"qcat/internal/database"
)

// IntegratedService 集成的热门币种服务
type IntegratedService struct {
	config               *config.Config
	db                   *database.DB
	scorer               *Scorer
	detector             *Detector
	recommendationEngine *RecommendationEngine

	// 数据收集组件
	dataCollector *DataCollector

	// 运行状态
	isRunning bool
	mu        sync.RWMutex

	// 配置参数
	scanInterval   time.Duration
	updateInterval time.Duration

	// 缓存
	lastScanTime   time.Time
	lastUpdateTime time.Time

	// 通知渠道
	recommendationChan chan []*EnhancedRecommendation
	alertChan          chan *Alert
}

// DataCollector 数据收集器
type DataCollector struct {
	db     *database.DB
	config *config.Config

	// 数据源配置
	enableMarketData bool
	enableSocialData bool
	enableNewsData   bool

	// 收集间隔
	marketDataInterval time.Duration
	socialDataInterval time.Duration
	newsDataInterval   time.Duration

	mu sync.RWMutex
}

// Alert 告警信息
type Alert struct {
	Type      string                 `json:"type"`
	Symbol    string                 `json:"symbol"`
	Message   string                 `json:"message"`
	Severity  string                 `json:"severity"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// ServiceConfig 服务配置
type ServiceConfig struct {
	ScanInterval       time.Duration `yaml:"scan_interval"`
	UpdateInterval     time.Duration `yaml:"update_interval"`
	EnableMarketData   bool          `yaml:"enable_market_data"`
	EnableSocialData   bool          `yaml:"enable_social_data"`
	EnableNewsData     bool          `yaml:"enable_news_data"`
	MarketDataInterval time.Duration `yaml:"market_data_interval"`
	SocialDataInterval time.Duration `yaml:"social_data_interval"`
	NewsDataInterval   time.Duration `yaml:"news_data_interval"`
}

// NewIntegratedService 创建集成服务
func NewIntegratedService(cfg *config.Config, db *database.DB) *IntegratedService {
	// 创建核心组件
	scorer := NewScorer(nil, nil, nil, &ScorerConfig{
		VolJumpWindow:    24,
		VolJumpThreshold: 0.02,
		TurnoverWindow:   24,
		OIChangeWindow:   24,
		FundingZWindow:   168,
		RegimeWindow:     48,
	})

	detector := NewDetector(scorer, &DetectorConfig{
		MinScore:        50.0,
		TopN:            20,
		ApprovalTimeout: time.Hour * 4,
	})

	recommendationEngine := NewRecommendationEngine(cfg, db, scorer, detector)

	// 创建数据收集器
	dataCollector := &DataCollector{
		db:                 db,
		config:             cfg,
		enableMarketData:   true,
		enableSocialData:   false, // 暂时禁用社交数据
		enableNewsData:     false, // 暂时禁用新闻数据
		marketDataInterval: time.Minute * 5,
		socialDataInterval: time.Minute * 30,
		newsDataInterval:   time.Minute * 15,
	}

	return &IntegratedService{
		config:               cfg,
		db:                   db,
		scorer:               scorer,
		detector:             detector,
		recommendationEngine: recommendationEngine,
		dataCollector:        dataCollector,
		scanInterval:         time.Minute * 10,
		updateInterval:       time.Minute * 5,
		recommendationChan:   make(chan []*EnhancedRecommendation, 10),
		alertChan:            make(chan *Alert, 100),
	}
}

// Start 启动服务
func (is *IntegratedService) Start(ctx context.Context) error {
	is.mu.Lock()
	defer is.mu.Unlock()

	if is.isRunning {
		return fmt.Errorf("service is already running")
	}

	is.isRunning = true

	// 启动数据收集
	go is.runDataCollection(ctx)

	// 启动热度扫描
	go is.runHotnessScan(ctx)

	// 启动推荐生成
	go is.runRecommendationGeneration(ctx)

	// 启动告警处理
	go is.runAlertProcessing(ctx)

	log.Printf("Integrated hotlist service started")
	return nil
}

// Stop 停止服务
func (is *IntegratedService) Stop() error {
	is.mu.Lock()
	defer is.mu.Unlock()

	if !is.isRunning {
		return fmt.Errorf("service is not running")
	}

	is.isRunning = false

	// 关闭通道
	close(is.recommendationChan)
	close(is.alertChan)

	log.Printf("Integrated hotlist service stopped")
	return nil
}

// runDataCollection 运行数据收集
func (is *IntegratedService) runDataCollection(ctx context.Context) {
	ticker := time.NewTicker(is.dataCollector.marketDataInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !is.isRunning {
				return
			}

			err := is.collectMarketData(ctx)
			if err != nil {
				log.Printf("Failed to collect market data: %v", err)
				is.sendAlert(&Alert{
					Type:      "DATA_COLLECTION_ERROR",
					Message:   fmt.Sprintf("Market data collection failed: %v", err),
					Severity:  "WARNING",
					Timestamp: time.Now(),
				})
			}
		}
	}
}

// runHotnessScan 运行热度扫描
func (is *IntegratedService) runHotnessScan(ctx context.Context) {
	ticker := time.NewTicker(is.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !is.isRunning {
				return
			}

			err := is.performHotnessScan(ctx)
			if err != nil {
				log.Printf("Failed to perform hotness scan: %v", err)
				is.sendAlert(&Alert{
					Type:      "SCAN_ERROR",
					Message:   fmt.Sprintf("Hotness scan failed: %v", err),
					Severity:  "ERROR",
					Timestamp: time.Now(),
				})
			}
		}
	}
}

// runRecommendationGeneration 运行推荐生成
func (is *IntegratedService) runRecommendationGeneration(ctx context.Context) {
	ticker := time.NewTicker(is.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !is.isRunning {
				return
			}

			recommendations, err := is.generateRecommendations(ctx)
			if err != nil {
				log.Printf("Failed to generate recommendations: %v", err)
				is.sendAlert(&Alert{
					Type:      "RECOMMENDATION_ERROR",
					Message:   fmt.Sprintf("Recommendation generation failed: %v", err),
					Severity:  "ERROR",
					Timestamp: time.Now(),
				})
				continue
			}

			// 发送推荐到通道
			select {
			case is.recommendationChan <- recommendations:
			default:
				log.Printf("Recommendation channel is full, dropping recommendations")
			}
		}
	}
}

// runAlertProcessing 运行告警处理
func (is *IntegratedService) runAlertProcessing(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case alert := <-is.alertChan:
			if alert == nil {
				return
			}

			err := is.processAlert(ctx, alert)
			if err != nil {
				log.Printf("Failed to process alert: %v", err)
			}
		}
	}
}

// collectMarketData 收集市场数据
func (is *IntegratedService) collectMarketData(ctx context.Context) error {
	// 获取活跃的交易对列表
	symbols, err := is.getActiveSymbols(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active symbols: %w", err)
	}

	log.Printf("Collecting market data for %d symbols", len(symbols))

	// 这里应该实现实际的市场数据收集逻辑
	// 目前使用模拟数据
	for _, symbol := range symbols {
		err := is.updateMarketDataForSymbol(ctx, symbol)
		if err != nil {
			log.Printf("Failed to update market data for %s: %v", symbol, err)
			continue
		}
	}

	return nil
}

// performHotnessScan 执行热度扫描
func (is *IntegratedService) performHotnessScan(ctx context.Context) error {
	// 获取需要扫描的符号
	symbols, err := is.getActiveSymbols(ctx)
	if err != nil {
		return fmt.Errorf("failed to get symbols for scanning: %w", err)
	}

	log.Printf("Performing hotness scan for %d symbols", len(symbols))

	// 使用detector进行热度检测
	hotSymbols, err := is.detector.DetectHotSymbols(ctx, symbols)
	if err != nil {
		return fmt.Errorf("failed to detect hot symbols: %w", err)
	}

	// 更新热度分数到数据库
	err = is.updateHotScores(ctx, hotSymbols)
	if err != nil {
		return fmt.Errorf("failed to update hot scores: %w", err)
	}

	is.lastScanTime = time.Now()
	log.Printf("Hotness scan completed, found %d hot symbols", len(hotSymbols))

	return nil
}

// generateRecommendations 生成推荐
func (is *IntegratedService) generateRecommendations(ctx context.Context) ([]*EnhancedRecommendation, error) {
	// 获取活跃符号
	symbols, err := is.getActiveSymbols(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get symbols: %w", err)
	}

	// 使用推荐引擎生成推荐
	recommendations, err := is.recommendationEngine.GenerateRecommendations(ctx, symbols)
	if err != nil {
		return nil, fmt.Errorf("failed to generate recommendations: %w", err)
	}

	// 保存推荐到数据库
	err = is.saveRecommendations(ctx, recommendations)
	if err != nil {
		log.Printf("Failed to save recommendations: %v", err)
		// 不返回错误，因为推荐已经生成
	}

	is.lastUpdateTime = time.Now()
	log.Printf("Generated %d recommendations", len(recommendations))

	return recommendations, nil
}

// getActiveSymbols 获取活跃的交易对
func (is *IntegratedService) getActiveSymbols(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT symbol, MAX(volume_24h) as max_volume
		FROM market_data
		WHERE updated_at > NOW() - INTERVAL '2 hours'
		AND volume_24h > 500000  -- 最小交易量过滤
		GROUP BY symbol
		ORDER BY max_volume DESC
		LIMIT 100
	`

	rows, err := is.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active symbols: %w", err)
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		var maxVolume float64
		if err := rows.Scan(&symbol, &maxVolume); err != nil {
			return nil, fmt.Errorf("failed to scan symbol: %w", err)
		}
		symbols = append(symbols, symbol)
	}

	// 如果数据库中没有数据，使用默认列表
	if len(symbols) == 0 {
		symbols = []string{
			"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT",
			"XRPUSDT", "DOTUSDT", "DOGEUSDT", "AVAXUSDT", "MATICUSDT",
			"LINKUSDT", "LTCUSDT", "UNIUSDT", "ATOMUSDT", "FILUSDT",
			"SHIBUSDT", "TRXUSDT", "NEARUSDT", "FTMUSDT", "SANDUSDT",
		}
	}

	return symbols, nil
}

// updateMarketDataForSymbol 更新单个符号的市场数据
func (is *IntegratedService) updateMarketDataForSymbol(ctx context.Context, symbol string) error {
	// 这里应该调用实际的市场数据API
	// 目前使用模拟数据

	query := `
		INSERT INTO market_data (
			symbol, price, volume_24h, volume_change_24h,
			price_change_24h, volatility, funding_rate,
			open_interest, oi_change_24h, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (symbol) DO UPDATE SET
			price = EXCLUDED.price,
			volume_24h = EXCLUDED.volume_24h,
			volume_change_24h = EXCLUDED.volume_change_24h,
			price_change_24h = EXCLUDED.price_change_24h,
			volatility = EXCLUDED.volatility,
			funding_rate = EXCLUDED.funding_rate,
			open_interest = EXCLUDED.open_interest,
			oi_change_24h = EXCLUDED.oi_change_24h,
			updated_at = NOW()
	`

	// 生成模拟数据
	price := 50000.0 + float64(len(symbol)*1000)
	volume24h := 1000000.0 + float64(len(symbol)*100000)
	volumeChange24h := -10.0 + float64(len(symbol)%20)
	priceChange24h := -5.0 + float64(len(symbol)%10)
	volatility := 0.02 + float64(len(symbol)%5)*0.01
	fundingRate := 0.0001 + float64(len(symbol)%3)*0.0001
	openInterest := 500000.0 + float64(len(symbol)*50000)
	oiChange24h := -5.0 + float64(len(symbol)%10)

	_, err := is.db.ExecContext(ctx, query,
		symbol, price, volume24h, volumeChange24h,
		priceChange24h, volatility, fundingRate,
		openInterest, oiChange24h,
	)

	if err != nil {
		return fmt.Errorf("failed to update market data for %s: %w", symbol, err)
	}

	return nil
}

// updateHotScores 更新热度分数
func (is *IntegratedService) updateHotScores(ctx context.Context, scores []*Score) error {
	query := `
		INSERT INTO hotlist_scores (
			symbol, vol_jump_score, turnover_score, oi_change_score,
			funding_z_score, regime_shift_score, total_score,
			risk_level, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		ON CONFLICT (symbol) DO UPDATE SET
			vol_jump_score = EXCLUDED.vol_jump_score,
			turnover_score = EXCLUDED.turnover_score,
			oi_change_score = EXCLUDED.oi_change_score,
			funding_z_score = EXCLUDED.funding_z_score,
			regime_shift_score = EXCLUDED.regime_shift_score,
			total_score = EXCLUDED.total_score,
			risk_level = EXCLUDED.risk_level,
			updated_at = NOW()
	`

	for _, score := range scores {
		// 确定风险等级
		riskLevel := "LOW"
		if score.TotalScore >= 80 {
			riskLevel = "HIGH"
		} else if score.TotalScore >= 60 {
			riskLevel = "MEDIUM"
		}

		_, err := is.db.ExecContext(ctx, query,
			score.Symbol,
			score.Components["vol_jump"],
			score.Components["turnover"],
			score.Components["oi_change"],
			score.Components["funding_z"],
			score.Components["regime_shift"],
			score.TotalScore,
			riskLevel,
		)

		if err != nil {
			log.Printf("Failed to update hot score for %s: %v", score.Symbol, err)
			continue
		}
	}

	return nil
}

// saveRecommendations 保存推荐到数据库
func (is *IntegratedService) saveRecommendations(ctx context.Context, recommendations []*EnhancedRecommendation) error {
	query := `
		INSERT INTO hotlist_recommendations (
			symbol, score, risk_level, risk_score, price_min, price_max,
			safe_leverage, market_sentiment, sentiment_score, reason,
			tags, confidence, time_horizon, expected_return, max_drawdown,
			created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (symbol) DO UPDATE SET
			score = EXCLUDED.score,
			risk_level = EXCLUDED.risk_level,
			risk_score = EXCLUDED.risk_score,
			price_min = EXCLUDED.price_min,
			price_max = EXCLUDED.price_max,
			safe_leverage = EXCLUDED.safe_leverage,
			market_sentiment = EXCLUDED.market_sentiment,
			sentiment_score = EXCLUDED.sentiment_score,
			reason = EXCLUDED.reason,
			tags = EXCLUDED.tags,
			confidence = EXCLUDED.confidence,
			time_horizon = EXCLUDED.time_horizon,
			expected_return = EXCLUDED.expected_return,
			max_drawdown = EXCLUDED.max_drawdown,
			updated_at = NOW(),
			expires_at = EXCLUDED.expires_at
	`

	for _, rec := range recommendations {
		// 将标签转换为字符串
		tagsStr := ""
		if len(rec.Tags) > 0 {
			for i, tag := range rec.Tags {
				if i > 0 {
					tagsStr += ","
				}
				tagsStr += tag
			}
		}

		_, err := is.db.ExecContext(ctx, query,
			rec.Symbol, rec.Score, rec.RiskLevel, rec.RiskScore,
			rec.PriceRange[0], rec.PriceRange[1], rec.SafeLeverage,
			rec.MarketSentiment, rec.SentimentScore, rec.Reason,
			tagsStr, rec.Confidence, rec.TimeHorizon,
			rec.ExpectedReturn, rec.MaxDrawdown,
			rec.Timestamp, rec.ExpiresAt,
		)

		if err != nil {
			log.Printf("Failed to save recommendation for %s: %v", rec.Symbol, err)
			continue
		}
	}

	return nil
}

// processAlert 处理告警
func (is *IntegratedService) processAlert(ctx context.Context, alert *Alert) error {
	// 保存告警到数据库
	query := `
		INSERT INTO hotlist_alerts (
			type, symbol, message, severity, data, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	dataStr := ""
	if alert.Data != nil {
		// 简化处理，实际应该使用JSON序列化
		for k, v := range alert.Data {
			if dataStr != "" {
				dataStr += ","
			}
			dataStr += fmt.Sprintf("%s:%v", k, v)
		}
	}

	_, err := is.db.ExecContext(ctx, query,
		alert.Type, alert.Symbol, alert.Message,
		alert.Severity, dataStr, alert.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to save alert: %w", err)
	}

	// 根据告警严重程度决定是否需要立即处理
	switch alert.Severity {
	case "CRITICAL":
		log.Printf("CRITICAL ALERT: %s - %s", alert.Type, alert.Message)
		// 可以在这里添加紧急通知逻辑
	case "ERROR":
		log.Printf("ERROR ALERT: %s - %s", alert.Type, alert.Message)
	case "WARNING":
		log.Printf("WARNING ALERT: %s - %s", alert.Type, alert.Message)
	default:
		log.Printf("INFO ALERT: %s - %s", alert.Type, alert.Message)
	}

	return nil
}

// sendAlert 发送告警
func (is *IntegratedService) sendAlert(alert *Alert) {
	select {
	case is.alertChan <- alert:
	default:
		log.Printf("Alert channel is full, dropping alert: %s", alert.Message)
	}
}

// GetRecommendations 获取当前推荐
func (is *IntegratedService) GetRecommendations() []*EnhancedRecommendation {
	return is.recommendationEngine.GetCachedRecommendations()
}

// GetRecommendationChannel 获取推荐通道
func (is *IntegratedService) GetRecommendationChannel() <-chan []*EnhancedRecommendation {
	return is.recommendationChan
}

// GetAlertChannel 获取告警通道
func (is *IntegratedService) GetAlertChannel() <-chan *Alert {
	return is.alertChan
}

// UpdateConfig 更新服务配置
func (is *IntegratedService) UpdateConfig(config *ServiceConfig) {
	is.mu.Lock()
	defer is.mu.Unlock()

	if config.ScanInterval > 0 {
		is.scanInterval = config.ScanInterval
	}
	if config.UpdateInterval > 0 {
		is.updateInterval = config.UpdateInterval
	}

	// 更新数据收集器配置
	if is.dataCollector != nil {
		is.dataCollector.mu.Lock()
		is.dataCollector.enableMarketData = config.EnableMarketData
		is.dataCollector.enableSocialData = config.EnableSocialData
		is.dataCollector.enableNewsData = config.EnableNewsData
		if config.MarketDataInterval > 0 {
			is.dataCollector.marketDataInterval = config.MarketDataInterval
		}
		if config.SocialDataInterval > 0 {
			is.dataCollector.socialDataInterval = config.SocialDataInterval
		}
		if config.NewsDataInterval > 0 {
			is.dataCollector.newsDataInterval = config.NewsDataInterval
		}
		is.dataCollector.mu.Unlock()
	}

	log.Printf("Service configuration updated")
}

// GetStatus 获取服务状态
func (is *IntegratedService) GetStatus() map[string]interface{} {
	is.mu.RLock()
	defer is.mu.RUnlock()

	status := map[string]interface{}{
		"is_running":           is.isRunning,
		"last_scan_time":       is.lastScanTime,
		"last_update_time":     is.lastUpdateTime,
		"scan_interval":        is.scanInterval.String(),
		"update_interval":      is.updateInterval.String(),
		"recommendation_count": len(is.recommendationEngine.GetCachedRecommendations()),
	}

	// 添加数据收集器状态
	if is.dataCollector != nil {
		is.dataCollector.mu.RLock()
		status["data_collector"] = map[string]interface{}{
			"enable_market_data":   is.dataCollector.enableMarketData,
			"enable_social_data":   is.dataCollector.enableSocialData,
			"enable_news_data":     is.dataCollector.enableNewsData,
			"market_data_interval": is.dataCollector.marketDataInterval.String(),
			"social_data_interval": is.dataCollector.socialDataInterval.String(),
			"news_data_interval":   is.dataCollector.newsDataInterval.String(),
		}
		is.dataCollector.mu.RUnlock()
	}

	return status
}

// ForceUpdate 强制更新推荐
func (is *IntegratedService) ForceUpdate(ctx context.Context) error {
	if !is.isRunning {
		return fmt.Errorf("service is not running")
	}

	// 强制执行热度扫描
	err := is.performHotnessScan(ctx)
	if err != nil {
		return fmt.Errorf("failed to perform hotness scan: %w", err)
	}

	// 强制生成推荐
	recommendations, err := is.generateRecommendations(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate recommendations: %w", err)
	}

	// 发送推荐到通道
	select {
	case is.recommendationChan <- recommendations:
	default:
		log.Printf("Recommendation channel is full during force update")
	}

	log.Printf("Force update completed, generated %d recommendations", len(recommendations))
	return nil
}

// ClearCache 清除所有缓存
func (is *IntegratedService) ClearCache() {
	is.recommendationEngine.ClearCache()
	log.Printf("All caches cleared")
}

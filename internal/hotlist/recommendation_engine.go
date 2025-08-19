package hotlist

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
)

// RecommendationEngine 推荐引擎
type RecommendationEngine struct {
	config   *config.Config
	db       *database.DB
	scorer   *Scorer
	detector *Detector
	mu       sync.RWMutex

	// 推荐配置
	maxRecommendations int
	minScore           float64
	riskWeights        map[string]float64

	// 缓存
	lastRecommendations []*EnhancedRecommendation
	lastUpdateTime      time.Time
	cacheDuration       time.Duration
}

// EnhancedRecommendation 增强推荐结果
type EnhancedRecommendation struct {
	Symbol          string             `json:"symbol"`
	Score           float64            `json:"score"`
	RiskLevel       string             `json:"risk_level"`
	RiskScore       float64            `json:"risk_score"`
	PriceRange      [2]float64         `json:"price_range"`
	SafeLeverage    float64            `json:"safe_leverage"`
	MarketSentiment string             `json:"market_sentiment"`
	SentimentScore  float64            `json:"sentiment_score"`
	Reason          string             `json:"reason"`
	Tags            []string           `json:"tags"`
	Metrics         map[string]float64 `json:"metrics"`
	Confidence      float64            `json:"confidence"`
	TimeHorizon     string             `json:"time_horizon"`
	ExpectedReturn  float64            `json:"expected_return"`
	MaxDrawdown     float64            `json:"max_drawdown"`
	Timestamp       time.Time          `json:"timestamp"`
	ExpiresAt       time.Time          `json:"expires_at"`
}

// RecommendationConfig 推荐配置
type RecommendationConfig struct {
	MaxRecommendations int                `yaml:"max_recommendations"`
	MinScore           float64            `yaml:"min_score"`
	CacheDuration      time.Duration      `yaml:"cache_duration"`
	RiskWeights        map[string]float64 `yaml:"risk_weights"`
	SentimentWeights   map[string]float64 `yaml:"sentiment_weights"`
	TimeHorizons       []string           `yaml:"time_horizons"`
}

// NewRecommendationEngine 创建推荐引擎
func NewRecommendationEngine(cfg *config.Config, db *database.DB, scorer *Scorer, detector *Detector) *RecommendationEngine {
	return &RecommendationEngine{
		config:             cfg,
		db:                 db,
		scorer:             scorer,
		detector:           detector,
		maxRecommendations: 20,
		minScore:           50.0,
		cacheDuration:      time.Minute * 15,
		riskWeights: map[string]float64{
			"HIGH":     0.3,
			"MEDIUM":   0.6,
			"LOW":      1.0,
			"VERY_LOW": 1.2,
		},
	}
}

// GenerateRecommendations 生成推荐列表
func (re *RecommendationEngine) GenerateRecommendations(ctx context.Context, symbols []string) ([]*EnhancedRecommendation, error) {
	re.mu.Lock()
	defer re.mu.Unlock()

	// 检查缓存
	if time.Since(re.lastUpdateTime) < re.cacheDuration && len(re.lastRecommendations) > 0 {
		log.Printf("Returning cached recommendations (%d items)", len(re.lastRecommendations))
		return re.lastRecommendations, nil
	}

	log.Printf("Generating fresh recommendations for %d symbols", len(symbols))

	// 1. 获取热度评分
	hotScores, err := re.getHotScores(ctx, symbols)
	if err != nil {
		return nil, fmt.Errorf("failed to get hot scores: %w", err)
	}

	// 2. 生成增强推荐
	recommendations := make([]*EnhancedRecommendation, 0, re.maxRecommendations)

	for _, score := range hotScores {
		if score.TotalScore < re.minScore {
			continue
		}

		recommendation, err := re.createEnhancedRecommendation(ctx, score)
		if err != nil {
			log.Printf("Failed to create recommendation for %s: %v", score.Symbol, err)
			continue
		}

		recommendations = append(recommendations, recommendation)

		if len(recommendations) >= re.maxRecommendations {
			break
		}
	}

	// 3. 应用智能排序
	re.applyIntelligentSorting(recommendations)

	// 4. 更新缓存
	re.lastRecommendations = recommendations
	re.lastUpdateTime = time.Now()

	log.Printf("Generated %d recommendations", len(recommendations))
	return recommendations, nil
}

// getHotScores 获取热度评分
func (re *RecommendationEngine) getHotScores(ctx context.Context, symbols []string) ([]*Score, error) {
	var scores []*Score

	for _, symbol := range symbols {
		score, err := re.scorer.CalculateScore(ctx, symbol)
		if err != nil {
			log.Printf("Failed to calculate score for %s: %v", symbol, err)
			continue
		}
		scores = append(scores, score)
	}

	// 按分数排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].TotalScore > scores[j].TotalScore
	})

	return scores, nil
}

// createEnhancedRecommendation 创建增强推荐
func (re *RecommendationEngine) createEnhancedRecommendation(ctx context.Context, score *Score) (*EnhancedRecommendation, error) {
	recommendation := &EnhancedRecommendation{
		Symbol:    score.Symbol,
		Score:     score.TotalScore,
		Timestamp: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour * 4), // 4小时有效期
		Metrics:   make(map[string]float64),
	}

	// 1. 计算风险等级和评分
	recommendation.RiskLevel, recommendation.RiskScore = re.calculateRiskLevel(score)

	// 2. 计算市场情绪
	recommendation.MarketSentiment, recommendation.SentimentScore = re.calculateMarketSentiment(score)

	// 3. 计算价格范围
	recommendation.PriceRange = re.calculatePriceRange(ctx, score.Symbol)

	// 4. 计算安全杠杆
	recommendation.SafeLeverage = re.calculateSafeLeverage(recommendation.RiskLevel, recommendation.SentimentScore)

	// 5. 生成标签
	recommendation.Tags = re.generateTags(score, recommendation)

	// 6. 计算置信度
	recommendation.Confidence = re.calculateConfidence(score, recommendation)

	// 7. 确定时间范围
	recommendation.TimeHorizon = re.determineTimeHorizon(score, recommendation)

	// 8. 计算预期收益和最大回撤
	recommendation.ExpectedReturn, recommendation.MaxDrawdown = re.calculateReturnMetrics(score, recommendation)

	// 9. 生成推荐理由
	recommendation.Reason = re.generateDetailedReason(score, recommendation)

	// 10. 填充详细指标
	re.fillDetailedMetrics(recommendation, score)

	return recommendation, nil
}

// calculateRiskLevel 计算风险等级
func (re *RecommendationEngine) calculateRiskLevel(score *Score) (string, float64) {
	// 基于多个因素计算风险评分
	volatilityRisk := score.Components["vol_jump"] * 0.3
	fundingRisk := score.Components["funding_z"] * 0.25
	oiRisk := score.Components["oi_change"] * 0.2
	turnoverRisk := score.Components["turnover"] * 0.15
	regimeRisk := score.Components["regime_shift"] * 0.1

	riskScore := volatilityRisk + fundingRisk + oiRisk + turnoverRisk + regimeRisk

	var riskLevel string
	if riskScore >= 0.8 {
		riskLevel = "HIGH"
	} else if riskScore >= 0.6 {
		riskLevel = "MEDIUM"
	} else if riskScore >= 0.4 {
		riskLevel = "LOW"
	} else {
		riskLevel = "VERY_LOW"
	}

	return riskLevel, riskScore
}

// calculateMarketSentiment 计算市场情绪
func (re *RecommendationEngine) calculateMarketSentiment(score *Score) (string, float64) {
	// 基于价格趋势和交易量计算情绪
	priceWeight := score.Components["regime_shift"] * 0.4
	volumeWeight := score.Components["turnover"] * 0.3
	fundingWeight := score.Components["funding_z"] * 0.2
	oiWeight := score.Components["oi_change"] * 0.1

	sentimentScore := priceWeight + volumeWeight + fundingWeight + oiWeight

	var sentiment string
	if sentimentScore >= 0.85 {
		sentiment = "EXTREMELY_BULLISH"
	} else if sentimentScore >= 0.7 {
		sentiment = "BULLISH"
	} else if sentimentScore >= 0.55 {
		sentiment = "NEUTRAL_BULLISH"
	} else if sentimentScore >= 0.45 {
		sentiment = "NEUTRAL"
	} else if sentimentScore >= 0.3 {
		sentiment = "NEUTRAL_BEARISH"
	} else {
		sentiment = "BEARISH"
	}

	return sentiment, sentimentScore
}

// calculatePriceRange 计算价格范围
func (re *RecommendationEngine) calculatePriceRange(ctx context.Context, symbol string) [2]float64 {
	// 从数据库获取当前价格和波动率
	query := `
		SELECT price, volatility 
		FROM market_data 
		WHERE symbol = $1 
		ORDER BY updated_at DESC 
		LIMIT 1
	`

	var price, volatility float64
	err := re.db.QueryRowContext(ctx, query, symbol).Scan(&price, &volatility)
	if err != nil {
		// 使用默认值
		price = 50000.0
		volatility = 0.05
	}

	// 计算价格范围 (基于2倍标准差)
	priceRange := [2]float64{
		price * (1 - volatility*2),
		price * (1 + volatility*2),
	}

	return priceRange
}

// calculateSafeLeverage 计算安全杠杆
func (re *RecommendationEngine) calculateSafeLeverage(riskLevel string, sentimentScore float64) float64 {
	baseMultiplier := re.riskWeights[riskLevel]
	if baseMultiplier == 0 {
		baseMultiplier = 0.5 // 默认保守值
	}

	// 根据市场情绪调整
	sentimentMultiplier := 1.0
	if sentimentScore > 0.8 {
		sentimentMultiplier = 1.2 // 强烈看涨时可以稍微提高杠杆
	} else if sentimentScore < 0.3 {
		sentimentMultiplier = 0.7 // 看跌时降低杠杆
	}

	safeLeverage := baseMultiplier * sentimentMultiplier * 10.0

	// 限制在合理范围内
	return math.Min(20.0, math.Max(1.0, safeLeverage))
}

// generateTags 生成标签
func (re *RecommendationEngine) generateTags(score *Score, rec *EnhancedRecommendation) []string {
	var tags []string

	// 基于评分组件生成标签
	if score.Components["vol_jump"] > 0.7 {
		tags = append(tags, "高波动")
	}
	if score.Components["turnover"] > 0.7 {
		tags = append(tags, "高换手")
	}
	if score.Components["funding_z"] > 0.6 {
		tags = append(tags, "资金费率异常")
	}
	if score.Components["oi_change"] > 0.6 {
		tags = append(tags, "持仓量激增")
	}
	if score.Components["regime_shift"] > 0.7 {
		tags = append(tags, "趋势突破")
	}

	// 基于风险等级添加标签
	switch rec.RiskLevel {
	case "HIGH":
		tags = append(tags, "高风险", "短线机会")
	case "MEDIUM":
		tags = append(tags, "中等风险", "波段机会")
	case "LOW":
		tags = append(tags, "低风险", "稳健投资")
	}

	// 基于市场情绪添加标签
	switch rec.MarketSentiment {
	case "EXTREMELY_BULLISH":
		tags = append(tags, "极度看涨", "突破机会")
	case "BULLISH":
		tags = append(tags, "看涨", "上涨趋势")
	case "NEUTRAL_BULLISH":
		tags = append(tags, "偏多", "震荡上行")
	case "BEARISH":
		tags = append(tags, "看跌", "谨慎观望")
	}

	return tags
}

// calculateConfidence 计算置信度
func (re *RecommendationEngine) calculateConfidence(score *Score, rec *EnhancedRecommendation) float64 {
	// 基于多个因素计算置信度
	scoreConfidence := math.Min(1.0, score.TotalScore/100.0)

	// 数据完整性检查
	dataCompleteness := 0.0
	componentCount := 0
	for _, value := range score.Components {
		if value > 0 {
			componentCount++
		}
	}
	dataCompleteness = float64(componentCount) / 5.0 // 5个主要组件

	// 风险一致性检查
	riskConsistency := 1.0
	if rec.RiskLevel == "HIGH" && rec.SentimentScore < 0.5 {
		riskConsistency = 0.7 // 高风险但情绪不强，降低置信度
	}

	confidence := (scoreConfidence*0.5 + dataCompleteness*0.3 + riskConsistency*0.2)
	return math.Min(1.0, math.Max(0.1, confidence))
}

// determineTimeHorizon 确定时间范围
func (re *RecommendationEngine) determineTimeHorizon(score *Score, rec *EnhancedRecommendation) string {
	// 基于波动率和风险等级确定时间范围
	volatility := score.Components["vol_jump"]

	if volatility > 0.8 || rec.RiskLevel == "HIGH" {
		return "SHORT_TERM" // 1-3天
	} else if volatility > 0.5 || rec.RiskLevel == "MEDIUM" {
		return "MEDIUM_TERM" // 3-7天
	} else {
		return "LONG_TERM" // 1-4周
	}
}

// calculateReturnMetrics 计算收益指标
func (re *RecommendationEngine) calculateReturnMetrics(score *Score, rec *EnhancedRecommendation) (float64, float64) {
	// 基于历史数据和当前指标估算预期收益和最大回撤
	baseReturn := score.TotalScore / 100.0 * 0.1 // 基础收益率

	// 根据时间范围调整
	timeMultiplier := 1.0
	switch rec.TimeHorizon {
	case "SHORT_TERM":
		timeMultiplier = 0.5
	case "MEDIUM_TERM":
		timeMultiplier = 1.0
	case "LONG_TERM":
		timeMultiplier = 2.0
	}

	expectedReturn := baseReturn * timeMultiplier

	// 计算最大回撤 (基于风险等级)
	var maxDrawdown float64
	switch rec.RiskLevel {
	case "HIGH":
		maxDrawdown = expectedReturn * 2.0 // 高风险高回撤
	case "MEDIUM":
		maxDrawdown = expectedReturn * 1.5
	case "LOW":
		maxDrawdown = expectedReturn * 1.0
	default:
		maxDrawdown = expectedReturn * 0.8
	}

	return expectedReturn, maxDrawdown
}

// generateDetailedReason 生成详细推荐理由
func (re *RecommendationEngine) generateDetailedReason(score *Score, rec *EnhancedRecommendation) string {
	reasons := []string{}

	// 分析各个组件
	if score.Components["vol_jump"] > 0.6 {
		reasons = append(reasons, fmt.Sprintf("波动率跳跃指标达到%.1f%%，显示价格波动显著增加", score.Components["vol_jump"]*100))
	}

	if score.Components["turnover"] > 0.6 {
		reasons = append(reasons, fmt.Sprintf("换手率指标为%.1f%%，交易活跃度明显提升", score.Components["turnover"]*100))
	}

	if score.Components["funding_z"] > 0.5 {
		reasons = append(reasons, fmt.Sprintf("资金费率Z分数为%.1f，市场情绪出现异常", score.Components["funding_z"]))
	}

	if score.Components["oi_change"] > 0.5 {
		reasons = append(reasons, fmt.Sprintf("持仓量变化指标为%.1f%%，机构资金流入明显", score.Components["oi_change"]*100))
	}

	if score.Components["regime_shift"] > 0.6 {
		reasons = append(reasons, fmt.Sprintf("市场状态切换指标为%.1f%%，趋势发生重要变化", score.Components["regime_shift"]*100))
	}

	// 添加综合评价
	reasons = append(reasons, fmt.Sprintf("综合热度评分%.1f分，建议关注度：%s", score.TotalScore, rec.MarketSentiment))

	if len(reasons) == 0 {
		return "各项指标表现平稳，适合稳健投资"
	}

	result := "推荐理由：\n"
	for i, reason := range reasons {
		result += fmt.Sprintf("%d. %s\n", i+1, reason)
	}

	return result
}

// fillDetailedMetrics 填充详细指标
func (re *RecommendationEngine) fillDetailedMetrics(rec *EnhancedRecommendation, score *Score) {
	rec.Metrics["total_score"] = score.TotalScore
	rec.Metrics["vol_jump"] = score.Components["vol_jump"]
	rec.Metrics["turnover"] = score.Components["turnover"]
	rec.Metrics["funding_z"] = score.Components["funding_z"]
	rec.Metrics["oi_change"] = score.Components["oi_change"]
	rec.Metrics["regime_shift"] = score.Components["regime_shift"]
	rec.Metrics["risk_score"] = rec.RiskScore
	rec.Metrics["sentiment_score"] = rec.SentimentScore
	rec.Metrics["confidence"] = rec.Confidence
	rec.Metrics["expected_return"] = rec.ExpectedReturn
	rec.Metrics["max_drawdown"] = rec.MaxDrawdown
}

// applyIntelligentSorting 应用智能排序
func (re *RecommendationEngine) applyIntelligentSorting(recommendations []*EnhancedRecommendation) {
	// 多因子排序：分数 * 置信度 * 风险调整
	sort.Slice(recommendations, func(i, j int) bool {
		scoreI := recommendations[i].Score * recommendations[i].Confidence * re.riskWeights[recommendations[i].RiskLevel]
		scoreJ := recommendations[j].Score * recommendations[j].Confidence * re.riskWeights[recommendations[j].RiskLevel]
		return scoreI > scoreJ
	})
}

// GetCachedRecommendations 获取缓存的推荐
func (re *RecommendationEngine) GetCachedRecommendations() []*EnhancedRecommendation {
	re.mu.RLock()
	defer re.mu.RUnlock()
	return re.lastRecommendations
}

// ClearCache 清除缓存
func (re *RecommendationEngine) ClearCache() {
	re.mu.Lock()
	defer re.mu.Unlock()
	re.lastRecommendations = nil
	re.lastUpdateTime = time.Time{}
}

// UpdateConfig 更新配置
func (re *RecommendationEngine) UpdateConfig(config *RecommendationConfig) {
	re.mu.Lock()
	defer re.mu.Unlock()

	if config.MaxRecommendations > 0 {
		re.maxRecommendations = config.MaxRecommendations
	}
	if config.MinScore > 0 {
		re.minScore = config.MinScore
	}
	if config.CacheDuration > 0 {
		re.cacheDuration = config.CacheDuration
	}
	if len(config.RiskWeights) > 0 {
		re.riskWeights = config.RiskWeights
	}

	// 清除缓存以应用新配置
	re.lastRecommendations = nil
	re.lastUpdateTime = time.Time{}
}

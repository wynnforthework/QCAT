package generator

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"qcat/internal/database"
	"qcat/internal/exchange/binance"
	"qcat/internal/market/kline"
)

// MarketAnalyzer 市场分析器
type MarketAnalyzer struct {
	db            *database.DB
	binanceClient *binance.Client
	klineManager  *kline.Manager
}

// NewMarketAnalyzer 创建新的市场分析器
func NewMarketAnalyzer(db *database.DB, binanceClient *binance.Client, klineManager *kline.Manager) *MarketAnalyzer {
	analyzer := &MarketAnalyzer{
		db:            db,
		binanceClient: binanceClient,
		klineManager:  klineManager,
	}

	// 如果有klineManager，启用自动回填功能
	if klineManager != nil {
		config := &kline.AutoBackfillConfig{
			Enabled:                true,
			MinCompletenessPercent: 80.0, // 80%完整度阈值
			MaxBackfillDays:        90,   // 最多回填90天
			RetryAttempts:          3,
			RetryDelay:             time.Second * 5,
		}
		klineManager.SetAutoBackfillConfig(config)
	}

	return analyzer
}

// MarketAnalysis 市场分析结果
type MarketAnalysis struct {
	Symbol              string              `json:"symbol"`
	TimeRange           time.Duration       `json:"time_range"`
	Volatility          float64             `json:"volatility"`   // 波动率
	Trend               float64             `json:"trend"`        // 趋势强度 (-1到1)
	SharpeRatio         float64             `json:"sharpe_ratio"` // 夏普比率
	MaxDrawdown         float64             `json:"max_drawdown"` // 最大回撤
	MarketCycle         float64             `json:"market_cycle"` // 市场周期(天)
	Liquidity           float64             `json:"liquidity"`    // 流动性指标
	Correlation         map[string]float64  `json:"correlation"`  // 与其他资产的相关性
	TechnicalIndicators TechnicalIndicators `json:"technical_indicators"`
	MarketRegime        string              `json:"market_regime"` // "trending", "ranging", "volatile"
	Confidence          float64             `json:"confidence"`    // 分析置信度
}

// TechnicalIndicators 技术指标
type TechnicalIndicators struct {
	RSI            float64 `json:"rsi"`  // RSI指标
	MACD           float64 `json:"macd"` // MACD指标
	BollingerBands struct {
		Upper  float64 `json:"upper"`
		Middle float64 `json:"middle"`
		Lower  float64 `json:"lower"`
		Width  float64 `json:"width"`
	} `json:"bollinger_bands"`
	SMA20    float64 `json:"sma_20"`    // 20日简单移动平均
	SMA50    float64 `json:"sma_50"`    // 50日简单移动平均
	EMA12    float64 `json:"ema_12"`    // 12日指数移动平均
	EMA26    float64 `json:"ema_26"`    // 26日指数移动平均
	ATR      float64 `json:"atr"`       // 平均真实波幅
	Volume   float64 `json:"volume"`    // 成交量
	VolumeMA float64 `json:"volume_ma"` // 成交量移动平均
}

// AnalyzeMarket 分析市场数据
func (ma *MarketAnalyzer) AnalyzeMarket(ctx context.Context, symbol string, timeRange time.Duration) (*MarketAnalysis, error) {
	// 从实际数据源获取历史价格数据
	priceData, err := ma.getHistoricalPriceData(ctx, symbol, timeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical price data for %s: %w", symbol, err)
	}

	analysis := &MarketAnalysis{
		Symbol:      symbol,
		TimeRange:   timeRange,
		Correlation: make(map[string]float64),
	}

	// 基于真实数据计算市场特征
	analysis.Volatility = ma.calculateVolatility(priceData)
	analysis.Trend = ma.calculateTrend(priceData)
	analysis.SharpeRatio = ma.calculateSharpeRatio(priceData)
	analysis.MaxDrawdown = ma.calculateMaxDrawdown(priceData)
	analysis.MarketCycle = float64(ma.calculateMarketCycle(priceData))
	analysis.Liquidity = ma.calculateLiquidity(priceData)
	analysis.MarketRegime = ma.determineMarketRegime(priceData)
	analysis.Confidence = ma.calculateConfidence(priceData)

	// 计算技术指标
	analysis.TechnicalIndicators = ma.calculateTechnicalIndicators(symbol)

	// 计算相关性
	analysis.Correlation = ma.calculateCorrelations(symbol)

	return analysis, nil
}

// calculateTechnicalIndicators 计算技术指标
func (ma *MarketAnalyzer) calculateTechnicalIndicators(symbol string) TechnicalIndicators {
	// 基于实际价格数据计算技术指标
	ctx := context.Background()

	// 获取历史价格数据用于计算指标
	priceData, err := ma.getHistoricalPriceData(ctx, symbol, 30*24*time.Hour)
	if err != nil || len(priceData) < 20 {
		log.Printf("Failed to get price data for technical indicators for %s: %v", symbol, err)
		return TechnicalIndicators{} // 返回空指标
	}

	// 计算各种技术指标
	indicators := TechnicalIndicators{}

	// 计算简单移动平均线
	if len(priceData) >= 20 {
		indicators.SMA20 = ma.calculateSimpleMA(priceData, 20)
	}
	if len(priceData) >= 50 {
		indicators.SMA50 = ma.calculateSimpleMA(priceData, 50)
	}

	// 计算指数移动平均线
	if len(priceData) >= 12 {
		indicators.EMA12 = ma.calculateEMA(priceData, 12)
	}
	if len(priceData) >= 26 {
		indicators.EMA26 = ma.calculateEMA(priceData, 26)
		indicators.MACD = indicators.EMA12 - indicators.EMA26
	}

	// 计算RSI
	if len(priceData) >= 14 {
		indicators.RSI = ma.calculateSimpleRSI(priceData, 14)
	}

	// 计算布林带
	if len(priceData) >= 20 {
		indicators.BollingerBands = ma.calculateSimpleBollingerBands(priceData, 20, 2.0)
	}

	// 计算ATR
	if len(priceData) >= 14 {
		indicators.ATR = ma.calculateSimpleATR(priceData, 14)
	}

	// 计算成交量指标
	if len(priceData) > 0 {
		indicators.Volume = priceData[len(priceData)-1].Volume
		if len(priceData) >= 20 {
			indicators.VolumeMA = ma.calculateVolumeMA(priceData, 20)
		}
	}

	return indicators
}

// calculateCorrelations 计算与其他资产的相关性
func (ma *MarketAnalyzer) calculateCorrelations(symbol string) map[string]float64 {
	// 基于实际价格数据计算相关性
	ctx := context.Background()
	correlations := make(map[string]float64)

	// 定义要计算相关性的资产列表
	assets := []string{"BTCUSDT", "ETHUSDT", "ADAUSDT", "BNBUSDT", "SOLUSDT"}

	// 获取目标资产的价格数据
	targetData, err := ma.getHistoricalPriceData(ctx, symbol, 30*24*time.Hour)
	if err != nil || len(targetData) < 10 {
		log.Printf("Failed to get price data for correlation calculation for %s: %v", symbol, err)
		return correlations
	}

	// 计算与其他资产的相关性
	for _, asset := range assets {
		if asset == symbol {
			continue // 跳过自己
		}

		assetData, err := ma.getHistoricalPriceData(ctx, asset, 30*24*time.Hour)
		if err != nil || len(assetData) < 10 {
			continue
		}

		// 计算相关系数
		corr := ma.calculatePearsonCorrelation(targetData, assetData)
		if !math.IsNaN(corr) {
			correlations[asset] = corr
		}
	}

	return correlations
}

// AnalyzePerformance 分析策略历史表现
func (ma *MarketAnalyzer) AnalyzePerformance(ctx context.Context, strategyID string, timeRange time.Duration) (*PerformanceAnalysis, error) {
	// 从数据库获取策略的历史表现数据
	performance := &PerformanceAnalysis{
		StrategyID: strategyID,
		TimeRange:  timeRange,
	}

	// 获取策略基本性能指标
	err := ma.getStrategyPerformanceMetrics(ctx, strategyID, performance)
	if err != nil {
		log.Printf("Failed to get strategy performance metrics for %s: %v", strategyID, err)
		// 如果无法获取数据库数据，返回错误而不是模拟数据
		return nil, fmt.Errorf("failed to get strategy performance data: %w", err)
	}

	// 获取交易统计数据
	err = ma.getStrategyTradeStatistics(ctx, strategyID, timeRange, performance)
	if err != nil {
		log.Printf("Failed to get strategy trade statistics for %s: %v", strategyID, err)
		// 交易统计失败时设置默认值
		performance.TotalTrades = 0
		performance.AvgTrade = 0.0
	}

	// 计算置信度（基于数据可用性和时间范围）
	performance.Confidence = ma.calculatePerformanceConfidence(performance)

	return performance, nil
}

// PerformanceAnalysis 策略表现分析
type PerformanceAnalysis struct {
	StrategyID   string        `json:"strategy_id"`
	TimeRange    time.Duration `json:"time_range"`
	TotalReturn  float64       `json:"total_return"`
	SharpeRatio  float64       `json:"sharpe_ratio"`
	MaxDrawdown  float64       `json:"max_drawdown"`
	WinRate      float64       `json:"win_rate"`
	ProfitFactor float64       `json:"profit_factor"`
	TotalTrades  int           `json:"total_trades"`
	AvgTrade     float64       `json:"avg_trade"`
	Volatility   float64       `json:"volatility"`
	Confidence   float64       `json:"confidence"`
}

// DetectMarketRegime 检测市场状态
func (ma *MarketAnalyzer) DetectMarketRegime(analysis *MarketAnalysis) string {
	// 基于多个指标判断市场状态
	volatilityThreshold := 0.04
	trendThreshold := 0.3

	if analysis.Volatility > volatilityThreshold {
		return "volatile"
	}

	if math.Abs(analysis.Trend) > trendThreshold {
		if analysis.Trend > 0 {
			return "bull_trending"
		} else {
			return "bear_trending"
		}
	}

	return "ranging"
}

// CalculateOptimalTimeframe 计算最优时间框架
func (ma *MarketAnalyzer) CalculateOptimalTimeframe(analysis *MarketAnalysis) time.Duration {
	// 基于市场周期和波动率计算最优时间框架
	baseDuration := time.Hour

	// 高波动市场使用较短时间框架
	if analysis.Volatility > 0.05 {
		return baseDuration / 2
	}

	// 低波动市场使用较长时间框架
	if analysis.Volatility < 0.02 {
		return baseDuration * 2
	}

	return baseDuration
}

// PricePoint 价格数据点
type PricePoint struct {
	Time   time.Time `json:"time"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume float64   `json:"volume"`
}

// getHistoricalPriceData 获取历史价格数据
func (ma *MarketAnalyzer) getHistoricalPriceData(ctx context.Context, symbol string, timeRange time.Duration) ([]PricePoint, error) {
	now := time.Now()
	startTime := now.Add(-timeRange)

	// 确定K线间隔
	interval := "1h"
	if timeRange <= 24*time.Hour {
		interval = "15m"
	} else if timeRange <= 7*24*time.Hour {
		interval = "1h"
	} else {
		interval = "1d"
	}

	// 首先尝试从数据库获取历史数据，如果不完整则自动回填
	if ma.klineManager != nil {
		klines, err := ma.klineManager.GetHistoryWithBackfill(ctx, symbol, kline.Interval(interval), startTime, now)
		if err == nil && len(klines) > 0 {
			log.Printf("Successfully retrieved %d klines from database for %s", len(klines), symbol)
			// 转换为PricePoint格式
			var priceData []PricePoint
			for _, k := range klines {
				priceData = append(priceData, PricePoint{
					Time:   k.OpenTime,
					Open:   k.Open,
					High:   k.High,
					Low:    k.Low,
					Close:  k.Close,
					Volume: k.Volume,
				})
			}
			// 验证数据质量
			if ma.validatePriceData(priceData, symbol) {
				return priceData, nil
			}
		} else {
			log.Printf("Failed to get klines from database for %s: %v", symbol, err)
		}
	} else {
		log.Printf("KlineManager not available for %s, trying direct database query", symbol)
		// 尝试直接从market_data表获取数据
		if ma.db != nil {
			priceData, err := ma.getHistoricalDataFromMarketDataTable(ctx, symbol, startTime, now)
			if err == nil && len(priceData) > 0 {
				log.Printf("Successfully retrieved %d price points from market_data table for %s", len(priceData), symbol)
				if ma.validatePriceData(priceData, symbol) {
					return priceData, nil
				}
			} else {
				log.Printf("Failed to get data from market_data table for %s: %v", symbol, err)
			}
		}
	}

	// 如果数据库中没有数据，尝试从交易所API获取
	if ma.binanceClient != nil {
		klines, err := ma.binanceClient.GetKlines(ctx, symbol, interval, startTime, now, 1000)
		if err == nil && len(klines) > 0 {
			log.Printf("Successfully retrieved %d klines from Binance API for %s", len(klines), symbol)
			// 转换为PricePoint格式
			var priceData []PricePoint
			for _, k := range klines {
				priceData = append(priceData, PricePoint{
					Time:   k.OpenTime,
					Open:   k.Open,
					High:   k.High,
					Low:    k.Low,
					Close:  k.Close,
					Volume: k.Volume,
				})
			}
			// 验证数据质量
			if ma.validatePriceData(priceData, symbol) {
				return priceData, nil
			}
		} else {
			log.Printf("Failed to get klines from Binance API for %s: %v", symbol, err)
		}
	} else {
		log.Printf("Binance client not available for %s", symbol)
	}

	// 如果都失败了，返回错误而不是生成模拟数据
	return nil, fmt.Errorf("failed to get historical price data for %s: no data available from database or exchange API", symbol)
}

// generateMockPriceData 生成模拟价格数据
func (ma *MarketAnalyzer) generateMockPriceData(symbol string, timeRange time.Duration, startTime, endTime time.Time) []PricePoint {
	// 根据时间范围确定数据点数量
	interval := time.Hour
	if timeRange <= 24*time.Hour {
		interval = time.Minute * 15 // 15分钟间隔
	} else if timeRange <= 7*24*time.Hour {
		interval = time.Hour // 1小时间隔
	} else {
		interval = 24 * time.Hour // 1天间隔
	}

	var priceData []PricePoint
	basePrice := 100.0 // 基础价格

	// 根据不同symbol设置不同的基础价格
	switch symbol {
	case "BTCUSDT":
		basePrice = 43000.0
	case "ETHUSDT":
		basePrice = 2750.0
	case "ADAUSDT":
		basePrice = 0.45
	case "BNBUSDT":
		basePrice = 320.0
	case "SOLUSDT":
		basePrice = 95.0
	}

	currentTime := startTime
	currentPrice := basePrice

	for currentTime.Before(endTime) {
		// 模拟价格波动（随机游走 + 趋势）
		volatility := 0.02 // 2%波动率
		trend := 0.0001    // 轻微上升趋势

		// 生成随机价格变化
		change := (math.Sin(float64(currentTime.Unix())/3600)*0.5 +
			math.Cos(float64(currentTime.Unix())/1800)*0.3) * volatility
		change += trend

		newPrice := currentPrice * (1 + change)

		// 生成OHLC数据
		high := newPrice * (1 + math.Abs(change)*0.5)
		low := newPrice * (1 - math.Abs(change)*0.5)
		open := currentPrice
		close := newPrice
		volume := 1000 + math.Abs(change)*10000 // 波动越大成交量越大

		priceData = append(priceData, PricePoint{
			Time:   currentTime,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: volume,
		})

		currentPrice = newPrice
		currentTime = currentTime.Add(interval)
	}

	return priceData
}

// calculateSimpleMA 计算简单移动平均线
func (ma *MarketAnalyzer) calculateSimpleMA(priceData []PricePoint, period int) float64 {
	if len(priceData) < period {
		return 0.0
	}

	var sum float64
	for i := len(priceData) - period; i < len(priceData); i++ {
		sum += priceData[i].Close
	}

	return sum / float64(period)
}

// calculateEMA 计算指数移动平均线
func (ma *MarketAnalyzer) calculateEMA(priceData []PricePoint, period int) float64 {
	if len(priceData) < period {
		return 0.0
	}

	// 计算平滑因子
	multiplier := 2.0 / (float64(period) + 1.0)

	// 使用SMA作为初始EMA值
	ema := ma.calculateSimpleMA(priceData[:period], period)

	// 计算EMA
	for i := period; i < len(priceData); i++ {
		ema = (priceData[i].Close * multiplier) + (ema * (1 - multiplier))
	}

	return ema
}

// calculateSimpleRSI 计算相对强弱指数
func (ma *MarketAnalyzer) calculateSimpleRSI(priceData []PricePoint, period int) float64 {
	if len(priceData) < period+1 {
		return 50.0 // 默认中性值
	}

	var gains, losses []float64

	// 计算价格变化
	for i := 1; i < len(priceData); i++ {
		change := priceData[i].Close - priceData[i-1].Close
		if change > 0 {
			gains = append(gains, change)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -change)
		}
	}

	if len(gains) < period {
		return 50.0
	}

	// 计算平均收益和平均损失
	var avgGain, avgLoss float64
	for i := len(gains) - period; i < len(gains); i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	if avgLoss == 0 {
		return 100.0
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi
}

// calculateSimpleBollingerBands 计算布林带
func (ma *MarketAnalyzer) calculateSimpleBollingerBands(priceData []PricePoint, period int, stdDev float64) struct {
	Upper  float64 `json:"upper"`
	Middle float64 `json:"middle"`
	Lower  float64 `json:"lower"`
	Width  float64 `json:"width"`
} {
	if len(priceData) < period {
		return struct {
			Upper  float64 `json:"upper"`
			Middle float64 `json:"middle"`
			Lower  float64 `json:"lower"`
			Width  float64 `json:"width"`
		}{}
	}

	// 计算中轨（SMA）
	middle := ma.calculateSimpleMA(priceData, period)

	// 计算标准差
	var variance float64
	for i := len(priceData) - period; i < len(priceData); i++ {
		variance += math.Pow(priceData[i].Close-middle, 2)
	}
	variance /= float64(period)
	standardDeviation := math.Sqrt(variance)

	// 计算上轨和下轨
	upper := middle + (stdDev * standardDeviation)
	lower := middle - (stdDev * standardDeviation)

	// 计算带宽
	width := (upper - lower) / middle

	return struct {
		Upper  float64 `json:"upper"`
		Middle float64 `json:"middle"`
		Lower  float64 `json:"lower"`
		Width  float64 `json:"width"`
	}{
		Upper:  upper,
		Middle: middle,
		Lower:  lower,
		Width:  width,
	}
}

// calculateSimpleATR 计算平均真实波幅
func (ma *MarketAnalyzer) calculateSimpleATR(priceData []PricePoint, period int) float64 {
	if len(priceData) < period+1 {
		return 0.0
	}

	var trueRanges []float64

	// 计算真实波幅
	for i := 1; i < len(priceData); i++ {
		high := priceData[i].High
		low := priceData[i].Low
		prevClose := priceData[i-1].Close

		tr1 := high - low
		tr2 := math.Abs(high - prevClose)
		tr3 := math.Abs(low - prevClose)

		trueRange := math.Max(tr1, math.Max(tr2, tr3))
		trueRanges = append(trueRanges, trueRange)
	}

	if len(trueRanges) < period {
		return 0.0
	}

	// 计算ATR（简单移动平均）
	var sum float64
	for i := len(trueRanges) - period; i < len(trueRanges); i++ {
		sum += trueRanges[i]
	}

	return sum / float64(period)
}

// calculateVolumeMA 计算成交量移动平均
func (ma *MarketAnalyzer) calculateVolumeMA(priceData []PricePoint, period int) float64 {
	if len(priceData) < period {
		return 0.0
	}

	var sum float64
	for i := len(priceData) - period; i < len(priceData); i++ {
		sum += priceData[i].Volume
	}

	return sum / float64(period)
}

// calculateVolatility 计算波动率
func (ma *MarketAnalyzer) calculateVolatility(priceData []PricePoint) float64 {
	if len(priceData) < 2 {
		return 0.0
	}

	// 计算收益率
	var returns []float64
	for i := 1; i < len(priceData); i++ {
		if priceData[i-1].Close > 0 {
			ret := (priceData[i].Close - priceData[i-1].Close) / priceData[i-1].Close
			returns = append(returns, ret)
		}
	}

	if len(returns) == 0 {
		return 0.0
	}

	// 计算平均收益率
	var sum float64
	for _, ret := range returns {
		sum += ret
	}
	mean := sum / float64(len(returns))

	// 计算方差
	var variance float64
	for _, ret := range returns {
		variance += math.Pow(ret-mean, 2)
	}
	variance /= float64(len(returns) - 1)

	// 返回年化波动率（假设数据是小时级别的）
	volatility := math.Sqrt(variance)

	// 年化处理：小时数据 * sqrt(24*365)
	annualizedVolatility := volatility * math.Sqrt(24*365)

	return annualizedVolatility
}

// calculateTrend 计算趋势
func (ma *MarketAnalyzer) calculateTrend(priceData []PricePoint) float64 {
	if len(priceData) < 2 {
		return 0.0
	}

	// 使用线性回归计算趋势强度
	n := float64(len(priceData))
	var sumX, sumY, sumXY, sumX2 float64

	for i, point := range priceData {
		x := float64(i)
		y := point.Close
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// 计算回归系数 (斜率)
	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0.0
	}

	slope := (n*sumXY - sumX*sumY) / denominator

	// 将斜率标准化到[-1, 1]范围
	// 使用价格的相对变化来标准化
	if len(priceData) > 0 && priceData[0].Close > 0 {
		normalizedSlope := slope / priceData[0].Close * float64(len(priceData))
		// 限制在[-1, 1]范围内
		if normalizedSlope > 1 {
			return 1.0
		} else if normalizedSlope < -1 {
			return -1.0
		}
		return normalizedSlope
	}

	return 0.0
}

// calculateSharpeRatio 计算夏普比率
func (ma *MarketAnalyzer) calculateSharpeRatio(priceData []PricePoint) float64 {
	if len(priceData) < 2 {
		return 0.0
	}

	// 计算收益率
	var returns []float64
	for i := 1; i < len(priceData); i++ {
		if priceData[i-1].Close > 0 {
			ret := (priceData[i].Close - priceData[i-1].Close) / priceData[i-1].Close
			returns = append(returns, ret)
		}
	}

	if len(returns) == 0 {
		return 0.0
	}

	// 计算平均收益率
	var sum float64
	for _, ret := range returns {
		sum += ret
	}
	meanReturn := sum / float64(len(returns))

	// 计算收益率标准差
	var variance float64
	for _, ret := range returns {
		variance += math.Pow(ret-meanReturn, 2)
	}
	variance /= float64(len(returns) - 1)
	stdDev := math.Sqrt(variance)

	if stdDev == 0 {
		return 0.0
	}

	// 假设无风险利率为年化3%，转换为对应时间周期
	riskFreeRate := 0.03 / (365 * 24) // 假设数据是小时级别

	// 计算夏普比率
	sharpeRatio := (meanReturn - riskFreeRate) / stdDev

	// 年化夏普比率
	return sharpeRatio * math.Sqrt(24*365)
}

// calculateMaxDrawdown 计算最大回撤
func (ma *MarketAnalyzer) calculateMaxDrawdown(priceData []PricePoint) float64 {
	if len(priceData) == 0 {
		return 0.0
	}

	var maxDrawdown float64
	var peak float64 = priceData[0].Close

	for _, point := range priceData {
		// 更新峰值
		if point.Close > peak {
			peak = point.Close
		}

		// 计算当前回撤
		drawdown := (peak - point.Close) / peak

		// 更新最大回撤
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

// calculateMarketCycle 计算市场周期
func (ma *MarketAnalyzer) calculateMarketCycle(priceData []PricePoint) int {
	if len(priceData) < 10 {
		return 0
	}

	// 使用简单的峰谷检测来估算市场周期
	var peaks []int
	var valleys []int

	// 寻找局部峰值和谷值
	for i := 1; i < len(priceData)-1; i++ {
		if priceData[i].Close > priceData[i-1].Close && priceData[i].Close > priceData[i+1].Close {
			peaks = append(peaks, i)
		}
		if priceData[i].Close < priceData[i-1].Close && priceData[i].Close < priceData[i+1].Close {
			valleys = append(valleys, i)
		}
	}

	// 计算平均周期长度
	var totalCycles int
	var cycleCount int

	// 计算峰值之间的距离
	for i := 1; i < len(peaks); i++ {
		totalCycles += peaks[i] - peaks[i-1]
		cycleCount++
	}

	// 计算谷值之间的距离
	for i := 1; i < len(valleys); i++ {
		totalCycles += valleys[i] - valleys[i-1]
		cycleCount++
	}

	if cycleCount == 0 {
		return 0
	}

	avgCycle := totalCycles / cycleCount

	// 转换为天数（假设数据是小时级别）
	return avgCycle / 24
}

// calculateLiquidity 计算流动性
func (ma *MarketAnalyzer) calculateLiquidity(priceData []PricePoint) float64 {
	if len(priceData) == 0 {
		return 0.0
	}

	// 使用成交量和价格波动来估算流动性
	var totalVolume float64
	var totalPriceChange float64
	var validPeriods int

	for i := 1; i < len(priceData); i++ {
		if priceData[i-1].Close > 0 {
			priceChange := math.Abs((priceData[i].Close - priceData[i-1].Close) / priceData[i-1].Close)
			totalVolume += priceData[i].Volume
			totalPriceChange += priceChange
			validPeriods++
		}
	}

	if validPeriods == 0 || totalPriceChange == 0 {
		return 0.0
	}

	avgVolume := totalVolume / float64(validPeriods)
	avgPriceChange := totalPriceChange / float64(validPeriods)

	// 流动性指标：成交量与价格波动的比率
	// 高成交量低波动 = 高流动性
	liquidity := avgVolume / (avgPriceChange * 1000000) // 标准化

	// 限制在合理范围内
	if liquidity > 1.0 {
		return 1.0
	}

	return liquidity
}

// determineMarketRegime 确定市场状态
func (ma *MarketAnalyzer) determineMarketRegime(priceData []PricePoint) string {
	if len(priceData) < 10 {
		return "unknown"
	}

	// 计算趋势和波动率
	trend := ma.calculateTrend(priceData)
	volatility := ma.calculateVolatility(priceData)

	// 定义阈值
	volatilityThreshold := 0.04 // 4%年化波动率
	trendThreshold := 0.3       // 趋势强度阈值

	// 判断市场状态
	if volatility > volatilityThreshold {
		return "volatile"
	}

	if math.Abs(trend) > trendThreshold {
		if trend > 0 {
			return "trending"
		} else {
			return "trending"
		}
	}

	return "ranging"
}

// calculateConfidence 计算置信度
func (ma *MarketAnalyzer) calculateConfidence(priceData []PricePoint) float64 {
	if len(priceData) < 10 {
		return 0.0
	}

	// 基于数据质量和数量计算置信度
	dataQuality := 1.0

	// 检查数据完整性
	var missingData int
	for _, point := range priceData {
		if point.Close <= 0 || point.Volume < 0 {
			missingData++
		}
	}

	// 数据完整性影响置信度
	completeness := 1.0 - float64(missingData)/float64(len(priceData))
	dataQuality *= completeness

	// 数据量影响置信度
	sampleSize := float64(len(priceData))
	sampleConfidence := math.Min(sampleSize/100.0, 1.0) // 100个数据点为满分

	// 价格稳定性影响置信度
	volatility := ma.calculateVolatility(priceData)
	stabilityFactor := 1.0 / (1.0 + volatility*10) // 波动率越高，置信度越低

	// 综合置信度
	confidence := dataQuality * sampleConfidence * stabilityFactor

	// 限制在[0, 1]范围内
	if confidence > 1.0 {
		return 1.0
	} else if confidence < 0.0 {
		return 0.0
	}

	return confidence
}

// calculatePearsonCorrelation 计算两个价格序列的皮尔逊相关系数
func (ma *MarketAnalyzer) calculatePearsonCorrelation(data1, data2 []PricePoint) float64 {
	// 确保两个数据序列长度一致
	minLen := len(data1)
	if len(data2) < minLen {
		minLen = len(data2)
	}

	if minLen < 2 {
		return 0.0
	}

	// 提取收盘价
	prices1 := make([]float64, minLen)
	prices2 := make([]float64, minLen)

	for i := 0; i < minLen; i++ {
		prices1[i] = data1[len(data1)-minLen+i].Close
		prices2[i] = data2[len(data2)-minLen+i].Close
	}

	// 计算均值
	var sum1, sum2 float64
	for i := 0; i < minLen; i++ {
		sum1 += prices1[i]
		sum2 += prices2[i]
	}
	mean1 := sum1 / float64(minLen)
	mean2 := sum2 / float64(minLen)

	// 计算协方差和方差
	var covariance, variance1, variance2 float64
	for i := 0; i < minLen; i++ {
		diff1 := prices1[i] - mean1
		diff2 := prices2[i] - mean2
		covariance += diff1 * diff2
		variance1 += diff1 * diff1
		variance2 += diff2 * diff2
	}

	// 计算相关系数
	denominator := math.Sqrt(variance1 * variance2)
	if denominator == 0 {
		return 0.0
	}

	return covariance / denominator
}

// getStrategyPerformanceMetrics 从数据库获取策略性能指标
func (ma *MarketAnalyzer) getStrategyPerformanceMetrics(ctx context.Context, strategyID string, performance *PerformanceAnalysis) error {
	// 首先尝试从strategy_performance表获取数据
	query := `
		SELECT
			COALESCE(AVG(total_pnl), 0) as total_return,
			COALESCE(AVG(sharpe_ratio), 0) as sharpe_ratio,
			COALESCE(AVG(max_drawdown), 0) as max_drawdown,
			COALESCE(AVG(win_rate), 0) as win_rate,
			COALESCE(AVG(total_trades), 0) as total_trades,
			COALESCE(AVG(winning_trades::float / NULLIF(total_trades, 0)), 0) as profit_factor
		FROM strategy_performance
		WHERE strategy_id = $1
		AND timestamp >= NOW() - INTERVAL '%d hours'
	`

	hours := int(performance.TimeRange.Hours())
	if hours == 0 {
		hours = 24 // 默认24小时
	}

	formattedQuery := fmt.Sprintf(query, hours)

	err := ma.db.QueryRowContext(ctx, formattedQuery, strategyID).Scan(
		&performance.TotalReturn,
		&performance.SharpeRatio,
		&performance.MaxDrawdown,
		&performance.WinRate,
		&performance.TotalTrades,
		&performance.ProfitFactor,
	)

	if err != nil {
		// 如果strategy_performance表没有数据，尝试从strategies表获取
		return ma.getStrategyMetricsFromStrategiesTable(ctx, strategyID, performance)
	}

	return nil
}

// getStrategyMetricsFromStrategiesTable 从strategies表获取策略指标
func (ma *MarketAnalyzer) getStrategyMetricsFromStrategiesTable(ctx context.Context, strategyID string, performance *PerformanceAnalysis) error {
	query := `
		SELECT
			COALESCE(sharpe_ratio, 0) as sharpe_ratio,
			COALESCE(max_drawdown, 0) as max_drawdown,
			COALESCE(total_return, 0) as total_return,
			COALESCE(win_rate, 0) as win_rate,
			COALESCE(profit_factor, 0) as profit_factor,
			COALESCE(volatility, 0) as volatility
		FROM strategies
		WHERE id = $1
	`

	err := ma.db.QueryRowContext(ctx, query, strategyID).Scan(
		&performance.SharpeRatio,
		&performance.MaxDrawdown,
		&performance.TotalReturn,
		&performance.WinRate,
		&performance.ProfitFactor,
		&performance.Volatility,
	)

	return err
}

// getStrategyTradeStatistics 获取策略交易统计数据
func (ma *MarketAnalyzer) getStrategyTradeStatistics(ctx context.Context, strategyID string, timeRange time.Duration, performance *PerformanceAnalysis) error {
	// 计算时间范围
	startTime := time.Now().Add(-timeRange)

	// 查询交易统计数据
	query := `
		SELECT
			COUNT(*) as total_trades,
			COALESCE(AVG(CASE
				WHEN side = 'BUY' THEN (price - LAG(price) OVER (ORDER BY created_at)) / LAG(price) OVER (ORDER BY created_at)
				ELSE (LAG(price) OVER (ORDER BY created_at) - price) / LAG(price) OVER (ORDER BY created_at)
			END), 0) as avg_trade_return
		FROM trades t
		WHERE EXISTS (
			SELECT 1 FROM strategies s
			WHERE s.id = $1
			AND (s.name = t.strategy_id OR s.id::text = t.strategy_id)
		)
		AND t.created_at >= $2
	`

	var totalTrades int
	var avgTradeReturn float64

	err := ma.db.QueryRowContext(ctx, query, strategyID, startTime).Scan(
		&totalTrades,
		&avgTradeReturn,
	)

	if err != nil {
		return err
	}

	performance.TotalTrades = totalTrades
	performance.AvgTrade = avgTradeReturn

	return nil
}

// calculatePerformanceConfidence 计算性能分析置信度
func (ma *MarketAnalyzer) calculatePerformanceConfidence(performance *PerformanceAnalysis) float64 {
	confidence := 0.0

	// 基于交易数量的置信度
	if performance.TotalTrades > 100 {
		confidence += 0.4
	} else if performance.TotalTrades > 50 {
		confidence += 0.3
	} else if performance.TotalTrades > 10 {
		confidence += 0.2
	} else if performance.TotalTrades > 0 {
		confidence += 0.1
	}

	// 基于时间范围的置信度
	days := performance.TimeRange.Hours() / 24
	if days >= 30 {
		confidence += 0.3
	} else if days >= 7 {
		confidence += 0.2
	} else if days >= 1 {
		confidence += 0.1
	}

	// 基于数据质量的置信度
	if performance.SharpeRatio != 0 && performance.MaxDrawdown != 0 {
		confidence += 0.2
	}

	if performance.WinRate > 0 && performance.WinRate <= 1 {
		confidence += 0.1
	}

	// 限制在[0, 1]范围内
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// getHistoricalDataFromMarketDataTable 直接从market_data表获取历史数据
func (ma *MarketAnalyzer) getHistoricalDataFromMarketDataTable(ctx context.Context, symbol string, startTime, endTime time.Time) ([]PricePoint, error) {
	query := `
		SELECT timestamp, open, high, low, close, volume
		FROM market_data
		WHERE symbol = $1
		AND timestamp BETWEEN $2 AND $3
		AND complete = true
		ORDER BY timestamp ASC
	`

	rows, err := ma.db.QueryContext(ctx, query, symbol, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query market_data table: %w", err)
	}
	defer rows.Close()

	var priceData []PricePoint
	for rows.Next() {
		var point PricePoint
		err := rows.Scan(
			&point.Time,
			&point.Open,
			&point.High,
			&point.Low,
			&point.Close,
			&point.Volume,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan market data row: %w", err)
		}
		priceData = append(priceData, point)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating market data rows: %w", err)
	}

	return priceData, nil
}

// validatePriceData 验证价格数据的质量
func (ma *MarketAnalyzer) validatePriceData(priceData []PricePoint, symbol string) bool {
	if len(priceData) == 0 {
		log.Printf("No price data available for %s", symbol)
		return false
	}

	// 检查数据完整性
	validPoints := 0
	for _, point := range priceData {
		if point.Close > 0 && point.Open > 0 && point.High > 0 && point.Low > 0 && point.Volume >= 0 {
			validPoints++
		}
	}

	completeness := float64(validPoints) / float64(len(priceData))
	if completeness < 0.8 { // 要求至少80%的数据完整
		log.Printf("Price data for %s has low completeness: %.2f%% (%d/%d valid points)",
			symbol, completeness*100, validPoints, len(priceData))
		return false
	}

	log.Printf("Price data for %s validated: %d points, %.2f%% completeness",
		symbol, len(priceData), completeness*100)
	return true
}

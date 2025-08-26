package testing

import (
	"math"
	"math/rand"
	"time"
)

// MarketDataPoint 市场数据点
type MarketDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Symbol    string    `json:"symbol"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	VWAP      float64   `json:"vwap"`
}

// TradingSignal 交易信号
type TradingSignal struct {
	Timestamp  time.Time `json:"timestamp"`
	Symbol     string    `json:"symbol"`
	SignalType string    `json:"signal_type"` // buy, sell, hold
	Strength   float64   `json:"strength"`    // 0-1
	Price      float64   `json:"price"`
	Volume     float64   `json:"volume"`
	Confidence float64   `json:"confidence"`
	StrategyID string    `json:"strategy_id"`
}

// MarketCondition 市场状态
type MarketCondition struct {
	Timestamp     time.Time `json:"timestamp"`
	MarketType    string    `json:"market_type"`    // bull, bear, sideways
	Volatility    float64   `json:"volatility"`     // 0-1
	Liquidity     float64   `json:"liquidity"`      // 0-1
	TrendStrength float64   `json:"trend_strength"` // -1 to 1
	Volume        float64   `json:"volume"`
}

// DataGenerator 数据生成器
type DataGenerator struct {
	symbols         []string
	strategies      []string
	currentPrices   map[string]float64
	priceHistory    map[string][]float64
	marketCondition *MarketCondition
	rand            *rand.Rand

	// 配置参数
	volatilityBase   float64
	trendPersistence float64
	volumeBase       float64
	priceChangeLimit float64
}

// NewDataGenerator 创建数据生成器
func NewDataGenerator() *DataGenerator {
	symbols := []string{
		"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "XRPUSDT",
		"SOLUSDT", "DOTUSDT", "DOGEUSDT", "AVAXUSDT", "MATICUSDT",
		"LINKUSDT", "LTCUSDT", "UNIUSDT", "ATOMUSDT", "VETUSDT",
	}

	strategies := []string{
		"momentum_strategy", "mean_reversion", "breakout_strategy",
		"grid_trading", "dca_strategy", "arbitrage_strategy",
		"trend_following", "scalping_strategy", "swing_trading",
		"pairs_trading", "market_making", "volatility_strategy",
	}

	// 初始价格
	initialPrices := map[string]float64{
		"BTCUSDT":   45000.0,
		"ETHUSDT":   3200.0,
		"BNBUSDT":   420.0,
		"ADAUSDT":   1.2,
		"XRPUSDT":   0.65,
		"SOLUSDT":   180.0,
		"DOTUSDT":   35.0,
		"DOGEUSDT":  0.08,
		"AVAXUSDT":  85.0,
		"MATICUSDT": 1.8,
		"LINKUSDT":  28.0,
		"LTCUSDT":   150.0,
		"UNIUSDT":   25.0,
		"ATOMUSDT":  32.0,
		"VETUSDT":   0.045,
	}

	dg := &DataGenerator{
		symbols:          symbols,
		strategies:       strategies,
		currentPrices:    initialPrices,
		priceHistory:     make(map[string][]float64),
		rand:             rand.New(rand.NewSource(time.Now().UnixNano())),
		volatilityBase:   0.02,
		trendPersistence: 0.7,
		volumeBase:       1000000,
		priceChangeLimit: 0.1,
	}

	// 初始化价格历史
	for symbol := range initialPrices {
		dg.priceHistory[symbol] = make([]float64, 0, 1000)
	}

	// 初始化市场状态
	dg.marketCondition = &MarketCondition{
		Timestamp:     time.Now(),
		MarketType:    "sideways",
		Volatility:    0.3,
		Liquidity:     0.8,
		TrendStrength: 0.0,
		Volume:        1.0,
	}

	return dg
}

// GenerateMarketData 生成市场数据
func (dg *DataGenerator) GenerateMarketData(symbol string, timestamp time.Time) *MarketDataPoint {
	currentPrice := dg.currentPrices[symbol]

	// 计算价格变化
	priceChange := dg.calculatePriceChange(symbol)
	newPrice := currentPrice * (1 + priceChange)

	// 确保价格在合理范围内
	if math.Abs(priceChange) > dg.priceChangeLimit {
		priceChange = dg.priceChangeLimit * math.Copysign(1, priceChange)
		newPrice = currentPrice * (1 + priceChange)
	}

	// 生成OHLC数据
	volatility := dg.marketCondition.Volatility * dg.volatilityBase
	high := newPrice * (1 + volatility*dg.rand.Float64())
	low := newPrice * (1 - volatility*dg.rand.Float64())
	open := currentPrice
	close := newPrice

	// 确保OHLC逻辑正确
	if high < math.Max(open, close) {
		high = math.Max(open, close) * (1 + volatility*0.1)
	}
	if low > math.Min(open, close) {
		low = math.Min(open, close) * (1 - volatility*0.1)
	}

	// 生成成交量
	baseVolume := dg.volumeBase * (0.5 + dg.rand.Float64())
	volumeMultiplier := 1 + math.Abs(priceChange)*10 // 价格变化越大，成交量越大
	volume := baseVolume * volumeMultiplier * dg.marketCondition.Volume

	// 计算VWAP
	vwap := (high + low + close) / 3

	// 更新当前价格和历史
	dg.currentPrices[symbol] = close
	dg.priceHistory[symbol] = append(dg.priceHistory[symbol], close)
	if len(dg.priceHistory[symbol]) > 1000 {
		dg.priceHistory[symbol] = dg.priceHistory[symbol][1:]
	}

	return &MarketDataPoint{
		Timestamp: timestamp,
		Symbol:    symbol,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    volume,
		VWAP:      vwap,
	}
}

// calculatePriceChange 计算价格变化
func (dg *DataGenerator) calculatePriceChange(symbol string) float64 {
	// 基础随机变化
	randomChange := (dg.rand.Float64() - 0.5) * 2 * dg.volatilityBase

	// 趋势影响
	trendInfluence := dg.marketCondition.TrendStrength * dg.volatilityBase * 0.5

	// 均值回归影响
	history := dg.priceHistory[symbol]
	meanReversionInfluence := 0.0
	if len(history) >= 20 {
		recentAvg := 0.0
		for i := len(history) - 20; i < len(history); i++ {
			recentAvg += history[i]
		}
		recentAvg /= 20

		currentPrice := dg.currentPrices[symbol]
		deviation := (currentPrice - recentAvg) / recentAvg
		meanReversionInfluence = -deviation * 0.1 // 向均值回归
	}

	// 市场状态影响
	marketInfluence := 0.0
	switch dg.marketCondition.MarketType {
	case "bull":
		marketInfluence = 0.001 // 轻微上涨偏向
	case "bear":
		marketInfluence = -0.001 // 轻微下跌偏向
	}

	totalChange := randomChange + trendInfluence + meanReversionInfluence + marketInfluence

	// 应用趋势持续性
	if dg.rand.Float64() < dg.trendPersistence && len(history) > 0 {
		lastChange := (dg.currentPrices[symbol] - history[len(history)-1]) / history[len(history)-1]
		totalChange = totalChange*0.3 + lastChange*0.7
	}

	return totalChange
}

// GenerateTradingSignal 生成交易信号
func (dg *DataGenerator) GenerateTradingSignal(symbol string, timestamp time.Time) *TradingSignal {
	strategyID := dg.strategies[dg.rand.Intn(len(dg.strategies))]
	currentPrice := dg.currentPrices[symbol]

	// 根据市场状态和策略类型生成信号
	signalType := dg.determineSignalType(symbol, strategyID)
	strength := 0.3 + dg.rand.Float64()*0.7   // 0.3-1.0
	confidence := 0.5 + dg.rand.Float64()*0.5 // 0.5-1.0

	// 调整信号强度基于市场状态
	switch dg.marketCondition.MarketType {
	case "bull":
		if signalType == "buy" {
			strength *= 1.2
			confidence *= 1.1
		}
	case "bear":
		if signalType == "sell" {
			strength *= 1.2
			confidence *= 1.1
		}
	}

	// 确保值在合理范围内
	if strength > 1.0 {
		strength = 1.0
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	// 生成交易量
	volume := dg.volumeBase * 0.01 * strength * (0.5 + dg.rand.Float64())

	return &TradingSignal{
		Timestamp:  timestamp,
		Symbol:     symbol,
		SignalType: signalType,
		Strength:   strength,
		Price:      currentPrice,
		Volume:     volume,
		Confidence: confidence,
		StrategyID: strategyID,
	}
}

// determineSignalType 确定信号类型
func (dg *DataGenerator) determineSignalType(symbol string, strategyID string) string {
	// 基于策略类型的信号倾向
	signalProbability := map[string]map[string]float64{
		"momentum_strategy":   {"buy": 0.4, "sell": 0.4, "hold": 0.2},
		"mean_reversion":      {"buy": 0.35, "sell": 0.35, "hold": 0.3},
		"breakout_strategy":   {"buy": 0.45, "sell": 0.35, "hold": 0.2},
		"grid_trading":        {"buy": 0.3, "sell": 0.3, "hold": 0.4},
		"dca_strategy":        {"buy": 0.6, "sell": 0.2, "hold": 0.2},
		"arbitrage_strategy":  {"buy": 0.4, "sell": 0.4, "hold": 0.2},
		"trend_following":     {"buy": 0.4, "sell": 0.4, "hold": 0.2},
		"scalping_strategy":   {"buy": 0.45, "sell": 0.45, "hold": 0.1},
		"swing_trading":       {"buy": 0.35, "sell": 0.35, "hold": 0.3},
		"pairs_trading":       {"buy": 0.4, "sell": 0.4, "hold": 0.2},
		"market_making":       {"buy": 0.3, "sell": 0.3, "hold": 0.4},
		"volatility_strategy": {"buy": 0.4, "sell": 0.4, "hold": 0.2},
	}

	probs := signalProbability[strategyID]
	if probs == nil {
		probs = map[string]float64{"buy": 0.33, "sell": 0.33, "hold": 0.34}
	}

	// 根据市场状态调整概率
	switch dg.marketCondition.MarketType {
	case "bull":
		probs["buy"] *= 1.3
		probs["sell"] *= 0.7
	case "bear":
		probs["buy"] *= 0.7
		probs["sell"] *= 1.3
	}

	// 归一化概率
	total := probs["buy"] + probs["sell"] + probs["hold"]
	for k := range probs {
		probs[k] /= total
	}

	// 随机选择
	r := dg.rand.Float64()
	if r < probs["buy"] {
		return "buy"
	} else if r < probs["buy"]+probs["sell"] {
		return "sell"
	}
	return "hold"
}

// UpdateMarketCondition 更新市场状态
func (dg *DataGenerator) UpdateMarketCondition(timestamp time.Time) {
	// 随机改变市场状态
	if dg.rand.Float64() < 0.05 { // 5% 概率改变市场类型
		marketTypes := []string{"bull", "bear", "sideways"}
		dg.marketCondition.MarketType = marketTypes[dg.rand.Intn(len(marketTypes))]
	}

	// 更新波动率
	volatilityChange := (dg.rand.Float64() - 0.5) * 0.1
	dg.marketCondition.Volatility += volatilityChange
	if dg.marketCondition.Volatility < 0.1 {
		dg.marketCondition.Volatility = 0.1
	}
	if dg.marketCondition.Volatility > 1.0 {
		dg.marketCondition.Volatility = 1.0
	}

	// 更新趋势强度
	trendChange := (dg.rand.Float64() - 0.5) * 0.2
	dg.marketCondition.TrendStrength += trendChange
	if dg.marketCondition.TrendStrength < -1.0 {
		dg.marketCondition.TrendStrength = -1.0
	}
	if dg.marketCondition.TrendStrength > 1.0 {
		dg.marketCondition.TrendStrength = 1.0
	}

	// 更新流动性
	liquidityChange := (dg.rand.Float64() - 0.5) * 0.05
	dg.marketCondition.Liquidity += liquidityChange
	if dg.marketCondition.Liquidity < 0.3 {
		dg.marketCondition.Liquidity = 0.3
	}
	if dg.marketCondition.Liquidity > 1.0 {
		dg.marketCondition.Liquidity = 1.0
	}

	dg.marketCondition.Timestamp = timestamp
}

// GetSymbols 获取所有交易对
func (dg *DataGenerator) GetSymbols() []string {
	return dg.symbols
}

// GetStrategies 获取所有策略
func (dg *DataGenerator) GetStrategies() []string {
	return dg.strategies
}

// GetCurrentPrices 获取当前价格
func (dg *DataGenerator) GetCurrentPrices() map[string]float64 {
	prices := make(map[string]float64)
	for k, v := range dg.currentPrices {
		prices[k] = v
	}
	return prices
}

// GetMarketCondition 获取市场状态
func (dg *DataGenerator) GetMarketCondition() *MarketCondition {
	return dg.marketCondition
}

// TimeAccelerator 时间加速器
type TimeAccelerator struct {
	startTime          time.Time
	currentTime        time.Time
	accelerationFactor int // 加速倍数
	tickInterval       time.Duration
	realStartTime      time.Time
}

// NewTimeAccelerator 创建时间加速器
func NewTimeAccelerator(startTime time.Time, accelerationFactor int, tickInterval time.Duration) *TimeAccelerator {
	return &TimeAccelerator{
		startTime:          startTime,
		currentTime:        startTime,
		accelerationFactor: accelerationFactor,
		tickInterval:       tickInterval,
		realStartTime:      time.Now(),
	}
}

// GetCurrentTime 获取当前模拟时间
func (ta *TimeAccelerator) GetCurrentTime() time.Time {
	return ta.currentTime
}

// Advance 推进时间
func (ta *TimeAccelerator) Advance() time.Time {
	// 计算实际经过的时间
	realElapsed := time.Since(ta.realStartTime)

	// 计算模拟时间应该推进多少
	simulatedElapsed := time.Duration(int64(realElapsed) * int64(ta.accelerationFactor))

	// 更新当前模拟时间
	ta.currentTime = ta.startTime.Add(simulatedElapsed)

	return ta.currentTime
}

// AdvanceBy 按指定时间推进
func (ta *TimeAccelerator) AdvanceBy(duration time.Duration) time.Time {
	ta.currentTime = ta.currentTime.Add(duration)
	return ta.currentTime
}

// GetAccelerationFactor 获取加速倍数
func (ta *TimeAccelerator) GetAccelerationFactor() int {
	return ta.accelerationFactor
}

// SetAccelerationFactor 设置加速倍数
func (ta *TimeAccelerator) SetAccelerationFactor(factor int) {
	ta.accelerationFactor = factor
	ta.realStartTime = time.Now() // 重置实际开始时间
}

// GetElapsedTime 获取已经过的模拟时间
func (ta *TimeAccelerator) GetElapsedTime() time.Duration {
	return ta.currentTime.Sub(ta.startTime)
}

// GetRealElapsedTime 获取实际经过的时间
func (ta *TimeAccelerator) GetRealElapsedTime() time.Duration {
	return time.Since(ta.realStartTime)
}

// GetTimeRatio 获取时间比率（模拟时间/实际时间）
func (ta *TimeAccelerator) GetTimeRatio() float64 {
	realElapsed := time.Since(ta.realStartTime)
	if realElapsed == 0 {
		return 0
	}
	simulatedElapsed := ta.currentTime.Sub(ta.startTime)
	return float64(simulatedElapsed) / float64(realElapsed)
}

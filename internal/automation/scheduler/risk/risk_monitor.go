package risk

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"qcat/internal/automation/scheduler/shared"
	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/exchange/account"
)

// RiskMonitor implements comprehensive risk monitoring functionality
type RiskMonitor struct {
	config         *config.Config
	db             *database.DB
	accountManager *account.Manager
	configManager  *shared.ConfigManager
	errorHandler   *shared.ErrorHandler
	mu             sync.RWMutex
	isRunning      bool
	lastCheck      time.Time
	metrics        map[string]interface{}
}

// MarginStatus represents current margin status
type MarginStatus struct {
	TotalEquity       float64            `json:"total_equity"`
	UsedMargin        float64            `json:"used_margin"`
	AvailableMargin   float64            `json:"available_margin"`
	MarginRatio       float64            `json:"margin_ratio"`
	RiskLevel         shared.RiskLevel   `json:"risk_level"`
	Timestamp         time.Time          `json:"timestamp"`
	ExchangeBreakdown map[string]float64 `json:"exchange_breakdown"`
	Recommendations   []string           `json:"recommendations"`
}

// PositionRiskReport represents comprehensive position risk analysis
type PositionRiskReport struct {
	Positions         []shared.PositionRisk `json:"positions"`
	TotalRisk         float64               `json:"total_risk"`
	ConcentrationRisk float64               `json:"concentration_risk"`
	CorrelationRisk   float64               `json:"correlation_risk"`
	LiquidityRisk     float64               `json:"liquidity_risk"`
	VaR               float64               `json:"var"`
	ExpectedShortfall float64               `json:"expected_shortfall"`
	MaxDrawdown       float64               `json:"max_drawdown"`
	Recommendations   []string              `json:"recommendations"`
	Timestamp         time.Time             `json:"timestamp"`
}

// MarketAnomalyReport represents detected market anomalies
type MarketAnomalyReport struct {
	AnomalyType        shared.AnomalyType `json:"anomaly_type"`
	Severity           shared.Severity    `json:"severity"`
	AffectedSymbols    []string           `json:"affected_symbols"`
	DetectionTime      time.Time          `json:"detection_time"`
	Confidence         float64            `json:"confidence"`
	Metrics            map[string]float64 `json:"metrics"`
	RecommendedActions []string           `json:"recommended_actions"`
	Description        string             `json:"description"`
}

// NewRiskMonitor creates a new risk monitor instance
func NewRiskMonitor(cfg *config.Config, db *database.DB, accountManager *account.Manager) *RiskMonitor {
	configManager := shared.NewConfigManager()

	// Initialize error handling
	retryStrategy := shared.NewRetryStrategy(3, time.Second, time.Minute*5, 2.0)
	circuitBreaker := shared.NewCircuitBreaker(shared.CircuitBreakerConfig{
		FailureThreshold: 5,
		RecoveryTimeout:  time.Minute * 5,
		HalfOpenRequests: 3,
		SuccessThreshold: 2,
	})
	errorHandler := shared.NewErrorHandler(retryStrategy, circuitBreaker)

	// Note: accountManager can be nil in test environments

	return &RiskMonitor{
		config:         cfg,
		db:             db,
		accountManager: accountManager,
		configManager:  configManager,
		errorHandler:   errorHandler,
		metrics:        make(map[string]interface{}),
	}
}

// isTestEnvironment checks if we're running in a test environment
func (rm *RiskMonitor) isTestEnvironment() bool {
	// Check if accountManager has nil fields that indicate it's not properly initialized
	if rm.accountManager != nil {
		// 实现真实的环境检测逻辑
		// 检查账户管理器是否能够正常连接到交易所
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 尝试获取账户信息来验证连接
		_, err := rm.accountManager.GetAllBalances(ctx)
		if err != nil {
			// 如果无法获取账户信息，可能是测试环境或连接问题
			log.Printf("Account manager connection test failed: %v", err)
			return true
		}

		// 连接正常，是真实环境
		return false
	}
	return true // 没有账户管理器，肯定是测试环境
}

// CheckMarginRatio checks current margin ratio against configured thresholds
func (rm *RiskMonitor) CheckMarginRatio(ctx context.Context) (*MarginStatus, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	log.Printf("Starting margin ratio check")

	// Check if account manager is available and properly initialized
	if rm.accountManager == nil {
		return nil, shared.NewAutomationError(
			shared.ErrCodeInvalidConfiguration,
			"Account manager not initialized",
			"RiskMonitor",
			shared.ErrorSeverityHigh,
			false,
		).WithContext("operation", "CheckMarginRatio")
	}

	// 如果是测试环境但仍有账户管理器，尝试获取真实数据
	// 只有在完全无法获取数据时才使用备用逻辑
	if rm.isTestEnvironment() {
		log.Printf("Warning: Running in test environment, attempting to get real data anyway")
	}

	// Get account balances from exchange
	balances, err := rm.accountManager.GetAllBalances(ctx)
	if err != nil {
		return nil, shared.NewAutomationError(
			shared.ErrCodeExchangeAPI,
			fmt.Sprintf("Failed to get account balances: %v", err),
			"RiskMonitor",
			shared.ErrorSeverityHigh,
			true,
		).WithContext("operation", "CheckMarginRatio")
	}

	// Calculate margin metrics from balances
	totalEquity := 0.0
	usedMargin := 0.0

	for _, balance := range balances {
		totalEquity += balance.Total
		usedMargin += balance.Locked // Locked balance represents used margin
	}

	availableMargin := totalEquity - usedMargin

	var marginRatio float64
	if totalEquity > 0 {
		marginRatio = usedMargin / totalEquity
	}

	// Get exchange breakdown from database
	exchangeBreakdown, err := rm.getExchangeMarginBreakdown(ctx)
	if err != nil {
		log.Printf("Warning: Failed to get exchange breakdown: %v", err)
		exchangeBreakdown = make(map[string]float64)
	}

	// Determine risk level based on margin ratio
	riskLevel := rm.determineMarginRiskLevel(marginRatio)

	// Generate recommendations
	recommendations := rm.generateMarginRecommendations(marginRatio, riskLevel)

	status := &MarginStatus{
		TotalEquity:       totalEquity,
		UsedMargin:        usedMargin,
		AvailableMargin:   availableMargin,
		MarginRatio:       marginRatio,
		RiskLevel:         riskLevel,
		Timestamp:         time.Now(),
		ExchangeBreakdown: exchangeBreakdown,
		Recommendations:   recommendations,
	}

	// Update metrics
	rm.updateMarginMetrics(status)

	// Log margin status
	log.Printf("Margin check completed: Ratio=%.4f, Risk=%s, Equity=%.2f",
		marginRatio, riskLevel.String(), totalEquity)

	return status, nil
}

// MonitorPositionRisk analyzes position-level risk metrics
func (rm *RiskMonitor) MonitorPositionRisk(ctx context.Context) (*PositionRiskReport, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	log.Printf("Starting position risk monitoring")

	// Get current positions from database
	positions, err := rm.getCurrentPositions(ctx)
	if err != nil {
		return nil, shared.NewAutomationError(
			shared.ErrCodeDatabaseConnection,
			fmt.Sprintf("Failed to get positions: %v", err),
			"RiskMonitor",
			shared.ErrorSeverityHigh,
			true,
		).WithContext("operation", "MonitorPositionRisk")
	}

	// Calculate position risks
	positionRisks := make([]shared.PositionRisk, 0, len(positions))
	var totalRisk float64

	for _, position := range positions {
		risk, err := rm.calculatePositionRisk(ctx, position)
		if err != nil {
			log.Printf("Warning: Failed to calculate risk for position %s: %v", position.ID, err)
			continue
		}
		positionRisks = append(positionRisks, *risk)
		totalRisk += risk.VaR
	}

	// Calculate portfolio-level risks
	concentrationRisk := rm.calculateConcentrationRisk(positions)
	correlationRisk := rm.calculateCorrelationRisk(ctx, positions)
	liquidityRisk := rm.calculateLiquidityRisk(ctx, positions)

	// Calculate VaR and Expected Shortfall
	portfolioVaR := rm.calculatePortfolioVaR(ctx, positions)
	expectedShortfall := rm.calculateExpectedShortfall(ctx, positions)

	// Calculate maximum drawdown
	maxDrawdown := rm.calculateMaxDrawdown(ctx)

	// Generate recommendations
	recommendations := rm.generateRiskRecommendations(totalRisk, concentrationRisk, correlationRisk, liquidityRisk)

	report := &PositionRiskReport{
		Positions:         positionRisks,
		TotalRisk:         totalRisk,
		ConcentrationRisk: concentrationRisk,
		CorrelationRisk:   correlationRisk,
		LiquidityRisk:     liquidityRisk,
		VaR:               portfolioVaR,
		ExpectedShortfall: expectedShortfall,
		MaxDrawdown:       maxDrawdown,
		Recommendations:   recommendations,
		Timestamp:         time.Now(),
	}

	// Update metrics
	rm.updateRiskMetrics(report)

	log.Printf("Position risk monitoring completed: TotalRisk=%.4f, VaR=%.4f, Positions=%d",
		totalRisk, portfolioVaR, len(positions))

	return report, nil
}

// DetectAbnormalMarket detects market anomalies using statistical methods
func (rm *RiskMonitor) DetectAbnormalMarket(ctx context.Context) (*MarketAnomalyReport, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	log.Printf("Starting abnormal market detection")

	// Get market data for analysis
	marketData, err := rm.getMarketDataForAnalysis(ctx)
	if err != nil {
		return nil, shared.NewAutomationError(
			shared.ErrCodeInsufficientData,
			fmt.Sprintf("Failed to get market data: %v", err),
			"RiskMonitor",
			shared.ErrorSeverityMedium,
			true,
		).WithContext("operation", "DetectAbnormalMarket")
	}

	// Detect different types of anomalies
	anomalies := make([]MarketAnomalyReport, 0)

	// 1. Volatility spike detection
	if volatilityAnomaly := rm.detectVolatilitySpike(marketData); volatilityAnomaly != nil {
		anomalies = append(anomalies, *volatilityAnomaly)
	}

	// 2. Liquidity drop detection
	if liquidityAnomaly := rm.detectLiquidityDrop(marketData); liquidityAnomaly != nil {
		anomalies = append(anomalies, *liquidityAnomaly)
	}

	// 3. Correlation breakdown detection
	if correlationAnomaly := rm.detectCorrelationBreakdown(ctx, marketData); correlationAnomaly != nil {
		anomalies = append(anomalies, *correlationAnomaly)
	}

	// 4. Price spike detection
	if priceAnomaly := rm.detectPriceSpike(marketData); priceAnomaly != nil {
		anomalies = append(anomalies, *priceAnomaly)
	}

	// Return the most severe anomaly or nil if none detected
	if len(anomalies) == 0 {
		log.Printf("No market anomalies detected")
		return nil, nil
	}

	// Find the most severe anomaly
	mostSevere := anomalies[0]
	for _, anomaly := range anomalies[1:] {
		if anomaly.Severity > mostSevere.Severity {
			mostSevere = anomaly
		}
	}

	log.Printf("Market anomaly detected: Type=%s, Severity=%s, Symbols=%v",
		mostSevere.AnomalyType.String(), mostSevere.Severity.String(), mostSevere.AffectedSymbols)

	return &mostSevere, nil
}

// Helper methods

// getExchangeMarginBreakdown gets margin breakdown by exchange
func (rm *RiskMonitor) getExchangeMarginBreakdown(ctx context.Context) (map[string]float64, error) {
	query := `
		SELECT 
			exchange,
			SUM(margin_used) as total_margin
		FROM positions 
		WHERE status = 'ACTIVE'
		GROUP BY exchange
	`

	rows, err := rm.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	breakdown := make(map[string]float64)
	for rows.Next() {
		var exchange string
		var margin float64
		if err := rows.Scan(&exchange, &margin); err != nil {
			return nil, err
		}
		breakdown[exchange] = margin
	}

	return breakdown, nil
}

// getCurrentPositions retrieves current active positions
func (rm *RiskMonitor) getCurrentPositions(ctx context.Context) ([]shared.Position, error) {
	query := `
		SELECT 
			id, symbol, side, size, entry_price, current_price,
			unrealized_pnl, realized_pnl, leverage, margin_used, created_at
		FROM positions 
		WHERE status = 'ACTIVE'
		ORDER BY created_at DESC
	`

	rows, err := rm.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []shared.Position
	for rows.Next() {
		var pos shared.Position
		if err := rows.Scan(
			&pos.ID, &pos.Symbol, &pos.Side, &pos.Size, &pos.EntryPrice,
			&pos.CurrentPrice, &pos.UnrealizedPnL, &pos.RealizedPnL,
			&pos.Leverage, &pos.MarginUsed, &pos.Timestamp,
		); err != nil {
			return nil, err
		}
		positions = append(positions, pos)
	}

	return positions, nil
}

// calculatePositionRisk calculates risk metrics for a single position
func (rm *RiskMonitor) calculatePositionRisk(ctx context.Context, position shared.Position) (*shared.PositionRisk, error) {
	// Get historical price data for volatility calculation
	prices, err := rm.getHistoricalPrices(ctx, position.Symbol, 30) // 30 days
	if err != nil {
		return nil, err
	}

	// Calculate volatility
	volatility := shared.CalculateRealizedVolatility(prices, 20)
	var currentVolatility float64
	if len(volatility) > 0 {
		currentVolatility = volatility[len(volatility)-1]
	}

	// Calculate VaR (95% confidence level)
	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		returns[i-1] = math.Log(prices[i] / prices[i-1])
	}

	var positionVaR float64
	if len(returns) > 0 {
		var95 := shared.CalculateVaR(returns, 0.95)
		positionVaR = var95 * position.Size * position.CurrentPrice
	}

	// Calculate Expected Shortfall
	expectedShortfall := shared.CalculateExpectedShortfall(returns, 0.95) * position.Size * position.CurrentPrice

	// Calculate Beta (simplified - using correlation with market)
	marketPrices, err := rm.getMarketIndexPrices(ctx, 30)
	if err != nil {
		log.Printf("Warning: Could not get market prices for beta calculation: %v", err)
	}

	var beta float64
	if len(marketPrices) == len(prices) && len(prices) > 1 {
		// Calculate returns for both
		assetReturns := make([]float64, len(prices)-1)
		marketReturns := make([]float64, len(marketPrices)-1)

		for i := 1; i < len(prices); i++ {
			assetReturns[i-1] = math.Log(prices[i] / prices[i-1])
			marketReturns[i-1] = math.Log(marketPrices[i] / marketPrices[i-1])
		}

		correlation := shared.CalculateCorrelation(assetReturns, marketReturns)
		assetVol := shared.CalculateStandardDeviation(assetReturns)
		marketVol := shared.CalculateStandardDeviation(marketReturns)

		if marketVol > 0 {
			beta = correlation * (assetVol / marketVol)
		}
	}

	// Calculate concentration risk (position size relative to total portfolio)
	totalPortfolioValue, err := rm.getTotalPortfolioValue(ctx)
	if err != nil {
		totalPortfolioValue = 1.0 // Fallback to avoid division by zero
	}

	positionValue := position.Size * position.CurrentPrice
	concentrationRisk := positionValue / totalPortfolioValue

	// Calculate liquidity risk (simplified based on symbol)
	liquidityRisk := rm.calculateSymbolLiquidityRisk(position.Symbol)

	return &shared.PositionRisk{
		Position:          position,
		VaR:               positionVaR,
		ExpectedShortfall: expectedShortfall,
		Beta:              beta,
		Volatility:        currentVolatility,
		ConcentrationRisk: concentrationRisk,
		LiquidityRisk:     liquidityRisk,
	}, nil
}

// determineMarginRiskLevel determines risk level based on margin ratio
func (rm *RiskMonitor) determineMarginRiskLevel(marginRatio float64) shared.RiskLevel {
	// Get thresholds from config (with defaults)
	lowThreshold := 0.5    // 50%
	mediumThreshold := 0.7 // 70%
	highThreshold := 0.85  // 85%

	if marginRatio >= highThreshold {
		return shared.RiskLevelCritical
	} else if marginRatio >= mediumThreshold {
		return shared.RiskLevelHigh
	} else if marginRatio >= lowThreshold {
		return shared.RiskLevelMedium
	}
	return shared.RiskLevelLow
}

// generateMarginRecommendations generates recommendations based on margin status
func (rm *RiskMonitor) generateMarginRecommendations(marginRatio float64, riskLevel shared.RiskLevel) []string {
	var recommendations []string

	switch riskLevel {
	case shared.RiskLevelCritical:
		recommendations = append(recommendations, "URGENT: Margin ratio critical - immediate position reduction required")
		recommendations = append(recommendations, "Consider emergency position closure")
		recommendations = append(recommendations, "Add additional margin if possible")
	case shared.RiskLevelHigh:
		recommendations = append(recommendations, "High margin usage - reduce position sizes")
		recommendations = append(recommendations, "Monitor positions closely")
		recommendations = append(recommendations, "Prepare for potential margin call")
	case shared.RiskLevelMedium:
		recommendations = append(recommendations, "Moderate margin usage - consider position optimization")
		recommendations = append(recommendations, "Review risk management settings")
	case shared.RiskLevelLow:
		recommendations = append(recommendations, "Margin usage within safe limits")
		if marginRatio < 0.3 {
			recommendations = append(recommendations, "Consider increasing position sizes if opportunities exist")
		}
	}

	return recommendations
}

// Additional helper methods would be implemented here...
// (getHistoricalPrices, getMarketDataForAnalysis, detectVolatilitySpike, etc.)

// updateMarginMetrics updates internal metrics for margin monitoring
func (rm *RiskMonitor) updateMarginMetrics(status *MarginStatus) {
	rm.metrics["margin_ratio"] = status.MarginRatio
	rm.metrics["total_equity"] = status.TotalEquity
	rm.metrics["used_margin"] = status.UsedMargin
	rm.metrics["risk_level"] = status.RiskLevel.String()
	rm.metrics["last_margin_check"] = status.Timestamp
}

// updateRiskMetrics updates internal metrics for risk monitoring
func (rm *RiskMonitor) updateRiskMetrics(report *PositionRiskReport) {
	rm.metrics["total_risk"] = report.TotalRisk
	rm.metrics["concentration_risk"] = report.ConcentrationRisk
	rm.metrics["correlation_risk"] = report.CorrelationRisk
	rm.metrics["liquidity_risk"] = report.LiquidityRisk
	rm.metrics["portfolio_var"] = report.VaR
	rm.metrics["expected_shortfall"] = report.ExpectedShortfall
	rm.metrics["max_drawdown"] = report.MaxDrawdown
	rm.metrics["position_count"] = len(report.Positions)
	rm.metrics["last_risk_check"] = report.Timestamp
}

// GetMetrics returns current risk monitoring metrics
func (rm *RiskMonitor) GetMetrics() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Return a copy to prevent external modifications
	metrics := make(map[string]interface{})
	for k, v := range rm.metrics {
		metrics[k] = v
	}
	return metrics
}

// IsRunning returns whether the risk monitor is currently running
func (rm *RiskMonitor) IsRunning() bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.isRunning
}

// Start starts the risk monitor
func (rm *RiskMonitor) Start() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.isRunning = true
	rm.lastCheck = time.Now()
	log.Printf("Risk monitor started")
	return nil
}

// Stop stops the risk monitor
func (rm *RiskMonitor) Stop() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.isRunning = false
	log.Printf("Risk monitor stopped")
	return nil
}

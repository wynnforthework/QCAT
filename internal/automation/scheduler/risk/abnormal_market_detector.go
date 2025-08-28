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

// AbnormalMarketDetector implements abnormal market detection functionality
type AbnormalMarketDetector struct {
	config         *config.Config
	db             *database.DB
	accountManager *account.Manager
	configManager  *shared.ConfigManager
	errorHandler   *shared.ErrorHandler
	mu             sync.RWMutex
	isRunning      bool
	lastCheck      time.Time
	metrics        map[string]interface{}
	
	// Detection parameters
	volatilityWindow    int     // Rolling window for volatility calculation
	volatilityThreshold float64 // Threshold for volatility spike detection
	liquidityThreshold  float64 // Threshold for liquidity drop detection
	correlationWindow   int     // Window for correlation analysis
	correlationThreshold float64 // Threshold for correlation breakdown
	circuitBreakerConfig CircuitBreakerConfig
}

// CircuitBreakerConfig defines circuit breaker configuration
type CircuitBreakerConfig struct {
	VolatilityThreshold    float64 `json:"volatility_threshold"`
	LiquidityThreshold     float64 `json:"liquidity_threshold"`
	CorrelationThreshold   float64 `json:"correlation_threshold"`
	PriceChangeThreshold   float64 `json:"price_change_threshold"`
	VolumeChangeThreshold  float64 `json:"volume_change_threshold"`
	ActivationDuration     time.Duration `json:"activation_duration"`
}

// VolatilityAlert represents a volatility spike alert
type VolatilityAlert struct {
	Symbol         string    `json:"symbol"`
	CurrentVol     float64   `json:"current_volatility"`
	HistoricalVol  float64   `json:"historical_volatility"`
	VolRatio       float64   `json:"volatility_ratio"`
	Severity       shared.AlertSeverity `json:"severity"`
	DetectedAt     time.Time `json:"detected_at"`
	Recommendations []string `json:"recommendations"`
}

// LiquidityAlert represents a liquidity drop alert
type LiquidityAlert struct {
	Symbol         string    `json:"symbol"`
	CurrentLiquidity float64 `json:"current_liquidity"`
	HistoricalLiquidity float64 `json:"historical_liquidity"`
	LiquidityRatio float64   `json:"liquidity_ratio"`
	BidAskSpread   float64   `json:"bid_ask_spread"`
	Severity       shared.AlertSeverity `json:"severity"`
	DetectedAt     time.Time `json:"detected_at"`
	Recommendations []string `json:"recommendations"`
}

// CorrelationAlert represents a correlation breakdown alert
type CorrelationAlert struct {
	AssetPairs     []string  `json:"asset_pairs"`
	CurrentCorr    float64   `json:"current_correlation"`
	HistoricalCorr float64   `json:"historical_correlation"`
	CorrChange     float64   `json:"correlation_change"`
	Severity       shared.AlertSeverity `json:"severity"`
	DetectedAt     time.Time `json:"detected_at"`
	Recommendations []string `json:"recommendations"`
}

// MarketStressIndicators represents various market stress indicators
type MarketStressIndicators struct {
	VIXLevel           float64   `json:"vix_level"`
	VolatilitySpread   float64   `json:"volatility_spread"`
	LiquidityIndex     float64   `json:"liquidity_index"`
	CorrelationIndex   float64   `json:"correlation_index"`
	MarketRegime       string    `json:"market_regime"`
	StressLevel        shared.Severity `json:"stress_level"`
	Timestamp          time.Time `json:"timestamp"`
}

// NewAbnormalMarketDetector creates a new abnormal market detector instance
func NewAbnormalMarketDetector(cfg *config.Config, db *database.DB, accountManager *account.Manager) *AbnormalMarketDetector {
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

	// Default detection parameters
	circuitBreakerConfig := CircuitBreakerConfig{
		VolatilityThreshold:    2.0,  // 2x normal volatility
		LiquidityThreshold:     0.5,  // 50% liquidity drop
		CorrelationThreshold:   0.3,  // 30% correlation change
		PriceChangeThreshold:   0.1,  // 10% price change
		VolumeChangeThreshold:  3.0,  // 3x volume increase
		ActivationDuration:     time.Minute * 15, // 15 minutes
	}

	return &AbnormalMarketDetector{
		config:               cfg,
		db:                   db,
		accountManager:       accountManager,
		configManager:        configManager,
		errorHandler:         errorHandler,
		metrics:              make(map[string]interface{}),
		volatilityWindow:     20,  // 20 periods
		volatilityThreshold:  2.0, // 2x threshold
		liquidityThreshold:   0.5, // 50% drop
		correlationWindow:    30,  // 30 periods
		correlationThreshold: 0.3, // 30% change
		circuitBreakerConfig: circuitBreakerConfig,
	}
}

// DetectVolatilitySpikes detects volatility spikes using rolling statistics
func (amd *AbnormalMarketDetector) DetectVolatilitySpikes(ctx context.Context) (*VolatilityAlert, error) {
	amd.mu.Lock()
	defer amd.mu.Unlock()

	log.Printf("Starting volatility spike detection")

	// Get active symbols for monitoring
	symbols, err := amd.getActiveSymbols(ctx)
	if err != nil {
		return nil, shared.NewAutomationError(
			shared.ErrCodeDatabaseConnection,
			fmt.Sprintf("Failed to get active symbols: %v", err),
			"AbnormalMarketDetector",
			shared.ErrorSeverityMedium,
			true,
		).WithContext("operation", "DetectVolatilitySpikes")
	}

	var mostSevereAlert *VolatilityAlert
	maxSeverity := shared.AlertSeverityLow

	for _, symbol := range symbols {
		alert, err := amd.analyzeSymbolVolatility(ctx, symbol)
		if err != nil {
			log.Printf("Warning: Failed to analyze volatility for %s: %v", symbol, err)
			continue
		}

		if alert != nil && alert.Severity > maxSeverity {
			mostSevereAlert = alert
			maxSeverity = alert.Severity
		}
	}

	if mostSevereAlert != nil {
		log.Printf("Volatility spike detected for %s: Current=%.4f, Historical=%.4f, Ratio=%.2f", 
			mostSevereAlert.Symbol, mostSevereAlert.CurrentVol, mostSevereAlert.HistoricalVol, mostSevereAlert.VolRatio)
		
		// Update metrics
		amd.updateVolatilityMetrics(mostSevereAlert)
	}

	return mostSevereAlert, nil
}

// DetectLiquidityDrops detects liquidity drops using order book analysis
func (amd *AbnormalMarketDetector) DetectLiquidityDrops(ctx context.Context) (*LiquidityAlert, error) {
	amd.mu.Lock()
	defer amd.mu.Unlock()

	log.Printf("Starting liquidity drop detection")

	// Get active symbols for monitoring
	symbols, err := amd.getActiveSymbols(ctx)
	if err != nil {
		return nil, shared.NewAutomationError(
			shared.ErrCodeDatabaseConnection,
			fmt.Sprintf("Failed to get active symbols: %v", err),
			"AbnormalMarketDetector",
			shared.ErrorSeverityMedium,
			true,
		).WithContext("operation", "DetectLiquidityDrops")
	}

	var mostSevereAlert *LiquidityAlert
	maxSeverity := shared.AlertSeverityLow

	for _, symbol := range symbols {
		alert, err := amd.analyzeSymbolLiquidity(ctx, symbol)
		if err != nil {
			log.Printf("Warning: Failed to analyze liquidity for %s: %v", symbol, err)
			continue
		}

		if alert != nil && alert.Severity > maxSeverity {
			mostSevereAlert = alert
			maxSeverity = alert.Severity
		}
	}

	if mostSevereAlert != nil {
		log.Printf("Liquidity drop detected for %s: Current=%.4f, Historical=%.4f, Ratio=%.2f", 
			mostSevereAlert.Symbol, mostSevereAlert.CurrentLiquidity, mostSevereAlert.HistoricalLiquidity, mostSevereAlert.LiquidityRatio)
		
		// Update metrics
		amd.updateLiquidityMetrics(mostSevereAlert)
	}

	return mostSevereAlert, nil
}

// DetectCorrelationBreakdown detects correlation breakdown using statistical methods
func (amd *AbnormalMarketDetector) DetectCorrelationBreakdown(ctx context.Context) (*CorrelationAlert, error) {
	amd.mu.Lock()
	defer amd.mu.Unlock()

	log.Printf("Starting correlation breakdown detection")

	// Get correlation pairs for analysis
	pairs, err := amd.getCorrelationPairs(ctx)
	if err != nil {
		return nil, shared.NewAutomationError(
			shared.ErrCodeDatabaseConnection,
			fmt.Sprintf("Failed to get correlation pairs: %v", err),
			"AbnormalMarketDetector",
			shared.ErrorSeverityMedium,
			true,
		).WithContext("operation", "DetectCorrelationBreakdown")
	}

	var mostSevereAlert *CorrelationAlert
	maxSeverity := shared.AlertSeverityLow

	for _, pair := range pairs {
		alert, err := amd.analyzeCorrelationBreakdown(ctx, pair)
		if err != nil {
			log.Printf("Warning: Failed to analyze correlation for %v: %v", pair, err)
			continue
		}

		if alert != nil && alert.Severity > maxSeverity {
			mostSevereAlert = alert
			maxSeverity = alert.Severity
		}
	}

	if mostSevereAlert != nil {
		log.Printf("Correlation breakdown detected for %v: Current=%.4f, Historical=%.4f, Change=%.2f", 
			mostSevereAlert.AssetPairs, mostSevereAlert.CurrentCorr, mostSevereAlert.HistoricalCorr, mostSevereAlert.CorrChange)
		
		// Update metrics
		amd.updateCorrelationMetrics(mostSevereAlert)
	}

	return mostSevereAlert, nil
}

// TriggerCircuitBreaker triggers circuit breaker mechanisms with configurable thresholds
func (amd *AbnormalMarketDetector) TriggerCircuitBreaker(ctx context.Context, severity shared.AlertSeverity) error {
	amd.mu.Lock()
	defer amd.mu.Unlock()

	log.Printf("Triggering circuit breaker with severity: %s", severity.String())

	// Determine circuit breaker actions based on severity
	actions := amd.determineCircuitBreakerActions(severity)

	// Execute circuit breaker actions
	for _, action := range actions {
		if err := amd.executeCircuitBreakerAction(ctx, action); err != nil {
			log.Printf("Error executing circuit breaker action %s: %v", action, err)
			// Continue with other actions even if one fails
		}
	}

	// Update circuit breaker metrics
	amd.updateCircuitBreakerMetrics(severity, actions)

	log.Printf("Circuit breaker triggered successfully with %d actions", len(actions))
	return nil
}

// Helper methods

// analyzeSymbolVolatility analyzes volatility for a specific symbol
func (amd *AbnormalMarketDetector) analyzeSymbolVolatility(ctx context.Context, symbol string) (*VolatilityAlert, error) {
	// Get recent price data
	prices, err := amd.getRecentPrices(ctx, symbol, amd.volatilityWindow*2)
	if err != nil {
		return nil, err
	}

	if len(prices) < amd.volatilityWindow {
		return nil, fmt.Errorf("insufficient price data for %s", symbol)
	}

	// Calculate current volatility (last N periods)
	recentPrices := prices[len(prices)-amd.volatilityWindow:]
	currentVol := shared.CalculateRealizedVolatility(recentPrices, amd.volatilityWindow)
	
	// Calculate historical volatility (previous N periods)
	historicalPrices := prices[:len(prices)-amd.volatilityWindow]
	historicalVol := shared.CalculateRealizedVolatility(historicalPrices, amd.volatilityWindow)

	if len(currentVol) == 0 || len(historicalVol) == 0 {
		return nil, fmt.Errorf("failed to calculate volatility for %s", symbol)
	}

	currentVolValue := currentVol[len(currentVol)-1]
	historicalVolValue := shared.CalculateMean(historicalVol)

	// Calculate volatility ratio
	var volRatio float64
	if historicalVolValue > 0 {
		volRatio = currentVolValue / historicalVolValue
	}

	// Determine if this constitutes a volatility spike
	if volRatio <= amd.volatilityThreshold {
		return nil, nil // No spike detected
	}

	// Determine severity
	severity := amd.determineVolatilitySeverity(volRatio)
	
	// Generate recommendations
	recommendations := amd.generateVolatilityRecommendations(symbol, volRatio, severity)

	return &VolatilityAlert{
		Symbol:          symbol,
		CurrentVol:      currentVolValue,
		HistoricalVol:   historicalVolValue,
		VolRatio:        volRatio,
		Severity:        severity,
		DetectedAt:      time.Now(),
		Recommendations: recommendations,
	}, nil
}

// analyzeSymbolLiquidity analyzes liquidity for a specific symbol
func (amd *AbnormalMarketDetector) analyzeSymbolLiquidity(ctx context.Context, symbol string) (*LiquidityAlert, error) {
	// Get current order book data
	orderBook, err := amd.getCurrentOrderBook(ctx, symbol)
	if err != nil {
		return nil, err
	}

	// Calculate current liquidity metrics
	currentLiquidity := amd.calculateLiquidityMetric(orderBook)
	bidAskSpread := amd.calculateBidAskSpread(orderBook)

	// Get historical liquidity data
	historicalLiquidity, err := amd.getHistoricalLiquidity(ctx, symbol, 30) // 30 periods
	if err != nil {
		return nil, err
	}

	if len(historicalLiquidity) == 0 {
		return nil, fmt.Errorf("no historical liquidity data for %s", symbol)
	}

	historicalAvg := shared.CalculateMean(historicalLiquidity)

	// Calculate liquidity ratio
	var liquidityRatio float64
	if historicalAvg > 0 {
		liquidityRatio = currentLiquidity / historicalAvg
	}

	// Determine if this constitutes a liquidity drop
	if liquidityRatio >= amd.liquidityThreshold {
		return nil, nil // No significant drop detected
	}

	// Determine severity
	severity := amd.determineLiquiditySeverity(liquidityRatio, bidAskSpread)
	
	// Generate recommendations
	recommendations := amd.generateLiquidityRecommendations(symbol, liquidityRatio, severity)

	return &LiquidityAlert{
		Symbol:              symbol,
		CurrentLiquidity:    currentLiquidity,
		HistoricalLiquidity: historicalAvg,
		LiquidityRatio:      liquidityRatio,
		BidAskSpread:        bidAskSpread,
		Severity:            severity,
		DetectedAt:          time.Now(),
		Recommendations:     recommendations,
	}, nil
}

// analyzeCorrelationBreakdown analyzes correlation breakdown for asset pairs
func (amd *AbnormalMarketDetector) analyzeCorrelationBreakdown(ctx context.Context, pair []string) (*CorrelationAlert, error) {
	if len(pair) != 2 {
		return nil, fmt.Errorf("invalid asset pair: %v", pair)
	}

	// Get price data for both assets
	prices1, err := amd.getRecentPrices(ctx, pair[0], amd.correlationWindow*2)
	if err != nil {
		return nil, err
	}

	prices2, err := amd.getRecentPrices(ctx, pair[1], amd.correlationWindow*2)
	if err != nil {
		return nil, err
	}

	if len(prices1) != len(prices2) || len(prices1) < amd.correlationWindow {
		return nil, fmt.Errorf("insufficient or mismatched price data for pair %v", pair)
	}

	// Calculate returns for both assets
	returns1 := amd.calculateReturns(prices1)
	returns2 := amd.calculateReturns(prices2)

	// Calculate current correlation (recent period)
	recentReturns1 := returns1[len(returns1)-amd.correlationWindow:]
	recentReturns2 := returns2[len(returns2)-amd.correlationWindow:]
	currentCorr := shared.CalculateCorrelation(recentReturns1, recentReturns2)

	// Calculate historical correlation (previous period)
	historicalReturns1 := returns1[:len(returns1)-amd.correlationWindow]
	historicalReturns2 := returns2[:len(returns2)-amd.correlationWindow]
	historicalCorr := shared.CalculateCorrelation(historicalReturns1, historicalReturns2)

	// Calculate correlation change
	corrChange := math.Abs(currentCorr - historicalCorr)

	// Determine if this constitutes a correlation breakdown
	if corrChange <= amd.correlationThreshold {
		return nil, nil // No significant breakdown detected
	}

	// Determine severity
	severity := amd.determineCorrelationSeverity(corrChange, currentCorr, historicalCorr)
	
	// Generate recommendations
	recommendations := amd.generateCorrelationRecommendations(pair, corrChange, severity)

	return &CorrelationAlert{
		AssetPairs:      pair,
		CurrentCorr:     currentCorr,
		HistoricalCorr:  historicalCorr,
		CorrChange:      corrChange,
		Severity:        severity,
		DetectedAt:      time.Now(),
		Recommendations: recommendations,
	}, nil
}

// Additional helper methods would be implemented here...
// (getActiveSymbols, getRecentPrices, getCurrentOrderBook, etc.)

// GetMetrics returns current abnormal market detection metrics
func (amd *AbnormalMarketDetector) GetMetrics() map[string]interface{} {
	amd.mu.RLock()
	defer amd.mu.RUnlock()
	
	// Return a copy to prevent external modifications
	metrics := make(map[string]interface{})
	for k, v := range amd.metrics {
		metrics[k] = v
	}
	return metrics
}

// IsRunning returns whether the detector is currently running
func (amd *AbnormalMarketDetector) IsRunning() bool {
	amd.mu.RLock()
	defer amd.mu.RUnlock()
	return amd.isRunning
}

// Start starts the abnormal market detector
func (amd *AbnormalMarketDetector) Start() error {
	amd.mu.Lock()
	defer amd.mu.Unlock()
	
	amd.isRunning = true
	amd.lastCheck = time.Now()
	log.Printf("Abnormal market detector started")
	return nil
}

// Stop stops the abnormal market detector
func (amd *AbnormalMarketDetector) Stop() error {
	amd.mu.Lock()
	defer amd.mu.Unlock()
	
	amd.isRunning = false
	log.Printf("Abnormal market detector stopped")
	return nil
}
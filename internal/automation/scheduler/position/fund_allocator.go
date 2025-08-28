package position

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"qcat/internal/automation/scheduler/shared"
	"qcat/internal/database"
	"qcat/internal/exchange"
	"qcat/internal/logger"
)

// FundAllocator implements intelligent fund allocation across strategies and assets
type FundAllocator struct {
	db             *database.DB
	exchangeClient exchange.Exchange
	logger         logger.Logger
	config         shared.ConfigProvider
	
	// Configuration
	minAllocation      float64
	maxAllocation      float64
	rebalanceThreshold float64
	lookbackPeriod     time.Duration
	riskFreeRate       float64
}

// NewFundAllocator creates a new fund allocator
func NewFundAllocator(
	db *database.DB,
	exchangeClient exchange.Exchange,
	logger logger.Logger,
	config shared.ConfigProvider,
) *FundAllocator {
	return &FundAllocator{
		db:                 db,
		exchangeClient:     exchangeClient,
		logger:             logger,
		config:             config,
		minAllocation:      config.GetFloat64("fund_allocation.min_allocation"),
		maxAllocation:      config.GetFloat64("fund_allocation.max_allocation"),
		rebalanceThreshold: config.GetFloat64("fund_allocation.rebalance_threshold"),
		lookbackPeriod:     config.GetDuration("fund_allocation.lookback_period"),
		riskFreeRate:       config.GetFloat64("fund_allocation.risk_free_rate"),
	}
}

// EfficiencyAnalysis represents fund efficiency analysis results
type EfficiencyAnalysis struct {
	StrategyID         string                 `json:"strategy_id"`
	SharpeRatio        float64                `json:"sharpe_ratio"`
	InformationRatio   float64                `json:"information_ratio"`
	CalmarRatio        float64                `json:"calmar_ratio"`
	SortinoRatio       float64                `json:"sortino_ratio"`
	MaxDrawdown        float64                `json:"max_drawdown"`
	Volatility         float64                `json:"volatility"`
	Returns            float64                `json:"returns"`
	RiskAdjustedReturn float64                `json:"risk_adjusted_return"`
	EfficiencyScore    float64                `json:"efficiency_score"`
	Rank               int                    `json:"rank"`
	Metadata           map[string]interface{} `json:"metadata"`
	AnalyzedAt         time.Time              `json:"analyzed_at"`
}

// EfficiencyReport represents comprehensive efficiency analysis
type EfficiencyReport struct {
	TotalStrategies    int                  `json:"total_strategies"`
	AnalysisResults    []EfficiencyAnalysis `json:"analysis_results"`
	TopPerformers      []string             `json:"top_performers"`
	UnderPerformers    []string             `json:"under_performers"`
	OverallMetrics     map[string]float64   `json:"overall_metrics"`
	Recommendations    []string             `json:"recommendations"`
	GeneratedAt        time.Time            `json:"generated_at"`
}

// AllocationDrift represents allocation drift detection
type AllocationDrift struct {
	StrategyID        string    `json:"strategy_id"`
	TargetAllocation  float64   `json:"target_allocation"`
	CurrentAllocation float64   `json:"current_allocation"`
	Drift             float64   `json:"drift"`
	DriftPercent      float64   `json:"drift_percent"`
	RequiresRebalance bool      `json:"requires_rebalance"`
	DetectedAt        time.Time `json:"detected_at"`
}

// RiskParityAllocation represents risk parity allocation result
type RiskParityAllocation struct {
	StrategyID           string    `json:"strategy_id"`
	RiskContribution     float64   `json:"risk_contribution"`
	TargetRiskBudget     float64   `json:"target_risk_budget"`
	OptimalAllocation    float64   `json:"optimal_allocation"`
	VolatilityAdjustment float64   `json:"volatility_adjustment"`
	CalculatedAt         time.Time `json:"calculated_at"`
}

// FundPerformanceAttribution represents strategy performance attribution for fund allocation
type FundPerformanceAttribution struct {
	StrategyID         string                 `json:"strategy_id"`
	AllocationReturn   float64                `json:"allocation_return"`
	SelectionReturn    float64                `json:"selection_return"`
	InteractionReturn  float64                `json:"interaction_return"`
	TotalReturn        float64                `json:"total_return"`
	BenchmarkReturn    float64                `json:"benchmark_return"`
	ActiveReturn       float64                `json:"active_return"`
	TrackingError      float64                `json:"tracking_error"`
	Contributions      map[string]interface{} `json:"contributions"`
	AttributedAt       time.Time              `json:"attributed_at"`
}

// AnalyzeFundEfficiency performs comprehensive fund efficiency analysis using Sharpe ratios
func (fa *FundAllocator) AnalyzeFundEfficiency(ctx context.Context) (*EfficiencyReport, error) {
	fa.logger.Info("Starting fund efficiency analysis")
	
	// Get all active strategies
	strategies, err := fa.getActiveStrategies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active strategies: %w", err)
	}
	
	if len(strategies) == 0 {
		return nil, fmt.Errorf("no active strategies found for analysis")
	}
	
	analysisResults := make([]EfficiencyAnalysis, 0, len(strategies))
	
	// Analyze each strategy
	for _, strategy := range strategies {
		analysis, err := fa.analyzeStrategyEfficiency(ctx, strategy)
		if err != nil {
			fa.logger.Warn("Failed to analyze strategy efficiency", 
				"strategy_id", strategy.ID, "error", err)
			continue
		}
		analysisResults = append(analysisResults, *analysis)
	}
	
	// Sort by efficiency score
	sort.Slice(analysisResults, func(i, j int) bool {
		return analysisResults[i].EfficiencyScore > analysisResults[j].EfficiencyScore
	})
	
	// Assign ranks
	for i := range analysisResults {
		analysisResults[i].Rank = i + 1
	}
	
	// Generate report
	report := &EfficiencyReport{
		TotalStrategies: len(analysisResults),
		AnalysisResults: analysisResults,
		TopPerformers:   fa.getTopPerformers(analysisResults, 3),
		UnderPerformers: fa.getUnderPerformers(analysisResults, 3),
		OverallMetrics:  fa.calculateOverallMetrics(analysisResults),
		Recommendations: fa.generateEfficiencyRecommendations(analysisResults),
		GeneratedAt:     time.Now(),
	}
	
	fa.logger.Info("Fund efficiency analysis completed", 
		"total_strategies", report.TotalStrategies,
		"top_performers", len(report.TopPerformers),
		"under_performers", len(report.UnderPerformers),
	)
	
	return report, nil
}

// CalculateRiskParityAllocation implements risk parity allocation algorithms
func (fa *FundAllocator) CalculateRiskParityAllocation(
	ctx context.Context, 
	strategies []shared.Strategy,
) ([]RiskParityAllocation, error) {
	fa.logger.Info("Calculating risk parity allocation", "strategy_count", len(strategies))
	
	if len(strategies) == 0 {
		return nil, fmt.Errorf("no strategies provided for risk parity allocation")
	}
	
	// Calculate volatilities for each strategy
	volatilities := make(map[string]float64)
	for _, strategy := range strategies {
		vol, err := fa.calculateStrategyVolatility(ctx, strategy.ID)
		if err != nil {
			fa.logger.Warn("Failed to calculate volatility for strategy", 
				"strategy_id", strategy.ID, "error", err)
			volatilities[strategy.ID] = 0.15 // Default volatility
		} else {
			volatilities[strategy.ID] = vol
		}
	}
	
	// Calculate inverse volatility weights
	totalInverseVol := 0.0
	inverseVols := make(map[string]float64)
	
	for strategyID, vol := range volatilities {
		if vol > 0 {
			inverseVol := 1.0 / vol
			inverseVols[strategyID] = inverseVol
			totalInverseVol += inverseVol
		}
	}
	
	// Calculate risk parity allocations
	allocations := make([]RiskParityAllocation, 0, len(strategies))
	targetRiskBudget := 1.0 / float64(len(strategies)) // Equal risk budget
	
	for _, strategy := range strategies {
		inverseVol := inverseVols[strategy.ID]
		optimalAllocation := inverseVol / totalInverseVol
		
		// Apply volatility adjustment
		volatilityAdjustment := fa.calculateVolatilityAdjustment(volatilities[strategy.ID])
		adjustedAllocation := optimalAllocation * volatilityAdjustment
		
		// Calculate risk contribution
		riskContribution := adjustedAllocation * volatilities[strategy.ID]
		
		allocation := RiskParityAllocation{
			StrategyID:           strategy.ID,
			RiskContribution:     riskContribution,
			TargetRiskBudget:     targetRiskBudget,
			OptimalAllocation:    adjustedAllocation,
			VolatilityAdjustment: volatilityAdjustment,
			CalculatedAt:         time.Now(),
		}
		
		allocations = append(allocations, allocation)
	}
	
	// Normalize allocations to sum to 1
	fa.normalizeAllocations(allocations)
	
	fa.logger.Info("Risk parity allocation calculated", 
		"allocations_count", len(allocations))
	
	return allocations, nil
}

// PerformAttributionAnalysis creates strategy performance attribution analysis
func (fa *FundAllocator) PerformAttributionAnalysis(
	ctx context.Context,
	strategies []shared.Strategy,
	benchmarkReturn float64,
) ([]FundPerformanceAttribution, error) {
	fa.logger.Info("Performing attribution analysis", "strategy_count", len(strategies))
	
	attributions := make([]FundPerformanceAttribution, 0, len(strategies))
	
	for _, strategy := range strategies {
		attribution, err := fa.calculateStrategyAttribution(ctx, strategy, benchmarkReturn)
		if err != nil {
			fa.logger.Warn("Failed to calculate attribution for strategy", 
				"strategy_id", strategy.ID, "error", err)
			continue
		}
		attributions = append(attributions, *attribution)
	}
	
	fa.logger.Info("Attribution analysis completed", 
		"attributions_count", len(attributions))
	
	return attributions, nil
}

// DetectAllocationDrift implements allocation drift detection and monitoring
func (fa *FundAllocator) DetectAllocationDrift(
	ctx context.Context,
	targetAllocations map[string]float64,
) ([]AllocationDrift, error) {
	fa.logger.Info("Detecting allocation drift", "target_allocations", len(targetAllocations))
	
	// Get current allocations
	currentAllocations, err := fa.getCurrentAllocations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current allocations: %w", err)
	}
	
	drifts := make([]AllocationDrift, 0)
	
	for strategyID, targetAllocation := range targetAllocations {
		currentAllocation, exists := currentAllocations[strategyID]
		if !exists {
			currentAllocation = 0.0
		}
		
		drift := currentAllocation - targetAllocation
		driftPercent := 0.0
		if targetAllocation != 0 {
			driftPercent = (drift / targetAllocation) * 100
		}
		
		requiresRebalance := math.Abs(driftPercent) > fa.rebalanceThreshold
		
		allocationDrift := AllocationDrift{
			StrategyID:        strategyID,
			TargetAllocation:  targetAllocation,
			CurrentAllocation: currentAllocation,
			Drift:             drift,
			DriftPercent:      driftPercent,
			RequiresRebalance: requiresRebalance,
			DetectedAt:        time.Now(),
		}
		
		drifts = append(drifts, allocationDrift)
		
		if requiresRebalance {
			fa.logger.Info("Allocation drift detected", 
				"strategy_id", strategyID,
				"drift_percent", driftPercent,
				"threshold", fa.rebalanceThreshold,
			)
		}
	}
	
	fa.logger.Info("Allocation drift detection completed", 
		"total_drifts", len(drifts),
		"requiring_rebalance", fa.countRebalanceRequired(drifts),
	)
	
	return drifts, nil
}

// Private helper methods

func (fa *FundAllocator) getActiveStrategies(ctx context.Context) ([]shared.Strategy, error) {
	// Query database for active strategies
	query := `
		SELECT id, name, type, parameters, created_at, last_updated
		FROM strategies 
		WHERE status = 'ACTIVE' 
		ORDER BY created_at DESC
	`
	
	rows, err := fa.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query strategies: %w", err)
	}
	defer rows.Close()
	
	strategies := make([]shared.Strategy, 0)
	
	for rows.Next() {
		var strategy shared.Strategy
		var parametersJSON string
		
		err := rows.Scan(
			&strategy.ID,
			&strategy.Name,
			&strategy.Type,
			&parametersJSON,
			&strategy.CreatedAt,
			&strategy.LastUpdated,
		)
		if err != nil {
			fa.logger.Warn("Failed to scan strategy row", "error", err)
			continue
		}
		
		// Parse parameters JSON (simplified)
		strategy.Parameters = make(map[string]interface{})
		strategy.Status = "ACTIVE"
		
		strategies = append(strategies, strategy)
	}
	
	return strategies, nil
}

func (fa *FundAllocator) analyzeStrategyEfficiency(
	ctx context.Context, 
	strategy shared.Strategy,
) (*EfficiencyAnalysis, error) {
	// Get strategy performance data
	returns, err := fa.getStrategyReturns(ctx, strategy.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get strategy returns: %w", err)
	}
	
	if len(returns) == 0 {
		return nil, fmt.Errorf("no return data available for strategy %s", strategy.ID)
	}
	
	// Calculate performance metrics
	avgReturn := fa.calculateMean(returns)
	volatility := fa.calculateVolatility(returns)
	maxDrawdown := fa.calculateMaxDrawdown(returns)
	
	// Calculate ratios
	sharpeRatio := fa.calculateSharpeRatio(avgReturn, volatility)
	calmarRatio := fa.calculateCalmarRatio(avgReturn, maxDrawdown)
	sortinoRatio := fa.calculateSortinoRatio(returns, avgReturn)
	informationRatio := fa.calculateInformationRatio(returns, avgReturn, volatility)
	
	// Calculate composite efficiency score
	efficiencyScore := fa.calculateEfficiencyScore(
		sharpeRatio, calmarRatio, sortinoRatio, informationRatio)
	
	analysis := &EfficiencyAnalysis{
		StrategyID:         strategy.ID,
		SharpeRatio:        sharpeRatio,
		InformationRatio:   informationRatio,
		CalmarRatio:        calmarRatio,
		SortinoRatio:       sortinoRatio,
		MaxDrawdown:        maxDrawdown,
		Volatility:         volatility,
		Returns:            avgReturn,
		RiskAdjustedReturn: sharpeRatio * volatility,
		EfficiencyScore:    efficiencyScore,
		Metadata: map[string]interface{}{
			"data_points":    len(returns),
			"analysis_period": fa.lookbackPeriod.String(),
		},
		AnalyzedAt: time.Now(),
	}
	
	return analysis, nil
}

func (fa *FundAllocator) getStrategyReturns(ctx context.Context, strategyID string) ([]float64, error) {
	// Query strategy returns from database
	query := `
		SELECT daily_return 
		FROM strategy_performance 
		WHERE strategy_id = ? 
		AND date >= ? 
		ORDER BY date DESC
		LIMIT 252
	`
	
	startDate := time.Now().Add(-fa.lookbackPeriod)
	
	rows, err := fa.db.QueryContext(ctx, query, strategyID, startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query strategy returns: %w", err)
	}
	defer rows.Close()
	
	returns := make([]float64, 0)
	
	for rows.Next() {
		var dailyReturn float64
		if err := rows.Scan(&dailyReturn); err != nil {
			fa.logger.Warn("Failed to scan return value", "error", err)
			continue
		}
		returns = append(returns, dailyReturn)
	}
	
	return returns, nil
}

func (fa *FundAllocator) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	
	return sum / float64(len(values))
}

func (fa *FundAllocator) calculateVolatility(returns []float64) float64 {
	if len(returns) < 2 {
		return 0.0
	}
	
	mean := fa.calculateMean(returns)
	sumSquaredDiffs := 0.0
	
	for _, ret := range returns {
		diff := ret - mean
		sumSquaredDiffs += diff * diff
	}
	
	variance := sumSquaredDiffs / float64(len(returns)-1)
	return math.Sqrt(variance) * math.Sqrt(252) // Annualized
}

func (fa *FundAllocator) calculateMaxDrawdown(returns []float64) float64 {
	if len(returns) == 0 {
		return 0.0
	}
	
	peak := 1.0
	maxDrawdown := 0.0
	cumulative := 1.0
	
	for _, ret := range returns {
		cumulative *= (1.0 + ret)
		
		if cumulative > peak {
			peak = cumulative
		}
		
		drawdown := (peak - cumulative) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}
	
	return maxDrawdown
}

func (fa *FundAllocator) calculateSharpeRatio(avgReturn, volatility float64) float64 {
	if volatility == 0 {
		return 0.0
	}
	
	excessReturn := avgReturn - fa.riskFreeRate
	return excessReturn / volatility
}

func (fa *FundAllocator) calculateCalmarRatio(avgReturn, maxDrawdown float64) float64 {
	if maxDrawdown == 0 {
		return 0.0
	}
	
	return avgReturn / maxDrawdown
}

func (fa *FundAllocator) calculateSortinoRatio(returns []float64, avgReturn float64) float64 {
	if len(returns) == 0 {
		return 0.0
	}
	
	// Calculate downside deviation
	sumSquaredDownsideDeviations := 0.0
	downsideCount := 0
	
	for _, ret := range returns {
		if ret < fa.riskFreeRate {
			deviation := ret - fa.riskFreeRate
			sumSquaredDownsideDeviations += deviation * deviation
			downsideCount++
		}
	}
	
	if downsideCount == 0 {
		return 0.0
	}
	
	downsideDeviation := math.Sqrt(sumSquaredDownsideDeviations/float64(downsideCount)) * math.Sqrt(252)
	
	if downsideDeviation == 0 {
		return 0.0
	}
	
	excessReturn := avgReturn - fa.riskFreeRate
	return excessReturn / downsideDeviation
}

func (fa *FundAllocator) calculateInformationRatio(returns []float64, avgReturn, volatility float64) float64 {
	// Simplified information ratio calculation
	// In practice, this would compare against a benchmark
	benchmarkReturn := fa.config.GetFloat64("fund_allocation.benchmark_return")
	activeReturn := avgReturn - benchmarkReturn
	
	if volatility == 0 {
		return 0.0
	}
	
	return activeReturn / volatility
}

func (fa *FundAllocator) calculateEfficiencyScore(
	sharpe, calmar, sortino, information float64,
) float64 {
	// Weighted composite score
	weights := map[string]float64{
		"sharpe":      0.4,
		"calmar":      0.3,
		"sortino":     0.2,
		"information": 0.1,
	}
	
	score := weights["sharpe"]*sharpe +
		weights["calmar"]*calmar +
		weights["sortino"]*sortino +
		weights["information"]*information
	
	return math.Max(0, score) // Ensure non-negative
}

func (fa *FundAllocator) getTopPerformers(results []EfficiencyAnalysis, count int) []string {
	topCount := int(math.Min(float64(count), float64(len(results))))
	performers := make([]string, topCount)
	
	for i := 0; i < topCount; i++ {
		performers[i] = results[i].StrategyID
	}
	
	return performers
}

func (fa *FundAllocator) getUnderPerformers(results []EfficiencyAnalysis, count int) []string {
	if len(results) == 0 {
		return []string{}
	}
	
	startIndex := int(math.Max(0, float64(len(results)-count)))
	underCount := len(results) - startIndex
	performers := make([]string, underCount)
	
	for i := 0; i < underCount; i++ {
		performers[i] = results[startIndex+i].StrategyID
	}
	
	return performers
}

func (fa *FundAllocator) calculateOverallMetrics(results []EfficiencyAnalysis) map[string]float64 {
	if len(results) == 0 {
		return map[string]float64{}
	}
	
	totalSharpe := 0.0
	totalEfficiency := 0.0
	totalVolatility := 0.0
	totalReturns := 0.0
	
	for _, result := range results {
		totalSharpe += result.SharpeRatio
		totalEfficiency += result.EfficiencyScore
		totalVolatility += result.Volatility
		totalReturns += result.Returns
	}
	
	count := float64(len(results))
	
	return map[string]float64{
		"avg_sharpe_ratio":    totalSharpe / count,
		"avg_efficiency_score": totalEfficiency / count,
		"avg_volatility":      totalVolatility / count,
		"avg_returns":         totalReturns / count,
		"total_strategies":    count,
	}
}

func (fa *FundAllocator) generateEfficiencyRecommendations(results []EfficiencyAnalysis) []string {
	recommendations := make([]string, 0)
	
	if len(results) == 0 {
		recommendations = append(recommendations, "No strategies available for analysis")
		return recommendations
	}
	
	// Analyze top performers
	if len(results) > 0 {
		topPerformer := results[0]
		recommendations = append(recommendations, 
			fmt.Sprintf("Consider increasing allocation to top performer: %s (Efficiency Score: %.3f)", 
				topPerformer.StrategyID, topPerformer.EfficiencyScore))
	}
	
	// Analyze under-performers
	if len(results) > 2 {
		underPerformer := results[len(results)-1]
		if underPerformer.EfficiencyScore < 0.5 {
			recommendations = append(recommendations, 
				fmt.Sprintf("Consider reducing allocation to under-performer: %s (Efficiency Score: %.3f)", 
					underPerformer.StrategyID, underPerformer.EfficiencyScore))
		}
	}
	
	// Check for high volatility strategies
	for _, result := range results {
		if result.Volatility > 0.3 {
			recommendations = append(recommendations, 
				fmt.Sprintf("Strategy %s has high volatility (%.3f), consider risk management", 
					result.StrategyID, result.Volatility))
		}
	}
	
	return recommendations
}

func (fa *FundAllocator) calculateStrategyVolatility(ctx context.Context, strategyID string) (float64, error) {
	returns, err := fa.getStrategyReturns(ctx, strategyID)
	if err != nil {
		return 0.0, err
	}
	
	return fa.calculateVolatility(returns), nil
}

func (fa *FundAllocator) calculateVolatilityAdjustment(volatility float64) float64 {
	// Apply volatility adjustment factor
	// Higher volatility gets lower adjustment (more conservative)
	baseAdjustment := 1.0
	volatilityPenalty := volatility * 0.5
	
	adjustment := baseAdjustment - volatilityPenalty
	return math.Max(0.1, math.Min(1.5, adjustment)) // Clamp between 0.1 and 1.5
}

func (fa *FundAllocator) normalizeAllocations(allocations []RiskParityAllocation) {
	totalAllocation := 0.0
	for _, allocation := range allocations {
		totalAllocation += allocation.OptimalAllocation
	}
	
	if totalAllocation > 0 {
		for i := range allocations {
			allocations[i].OptimalAllocation /= totalAllocation
		}
	}
}

func (fa *FundAllocator) calculateStrategyAttribution(
	ctx context.Context,
	strategy shared.Strategy,
	benchmarkReturn float64,
) (*FundPerformanceAttribution, error) {
	// Get strategy performance data
	returns, err := fa.getStrategyReturns(ctx, strategy.ID)
	if err != nil {
		return nil, err
	}
	
	if len(returns) == 0 {
		return nil, fmt.Errorf("no return data for strategy %s", strategy.ID)
	}
	
	avgReturn := fa.calculateMean(returns)
	volatility := fa.calculateVolatility(returns)
	
	// Calculate attribution components
	allocationReturn := avgReturn * 0.6  // Simplified allocation effect
	selectionReturn := avgReturn * 0.3   // Simplified selection effect
	interactionReturn := avgReturn * 0.1 // Simplified interaction effect
	
	totalReturn := allocationReturn + selectionReturn + interactionReturn
	activeReturn := totalReturn - benchmarkReturn
	trackingError := volatility
	
	attribution := &FundPerformanceAttribution{
		StrategyID:        strategy.ID,
		AllocationReturn:  allocationReturn,
		SelectionReturn:   selectionReturn,
		InteractionReturn: interactionReturn,
		TotalReturn:       totalReturn,
		BenchmarkReturn:   benchmarkReturn,
		ActiveReturn:      activeReturn,
		TrackingError:     trackingError,
		Contributions: map[string]interface{}{
			"allocation_contribution":  allocationReturn / totalReturn,
			"selection_contribution":   selectionReturn / totalReturn,
			"interaction_contribution": interactionReturn / totalReturn,
		},
		AttributedAt: time.Now(),
	}
	
	return attribution, nil
}

func (fa *FundAllocator) getCurrentAllocations(ctx context.Context) (map[string]float64, error) {
	// Query current allocations from database
	query := `
		SELECT strategy_id, allocation_percent 
		FROM current_allocations 
		WHERE status = 'ACTIVE'
	`
	
	rows, err := fa.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query current allocations: %w", err)
	}
	defer rows.Close()
	
	allocations := make(map[string]float64)
	
	for rows.Next() {
		var strategyID string
		var allocationPercent float64
		
		if err := rows.Scan(&strategyID, &allocationPercent); err != nil {
			fa.logger.Warn("Failed to scan allocation row", "error", err)
			continue
		}
		
		allocations[strategyID] = allocationPercent
	}
	
	return allocations, nil
}

func (fa *FundAllocator) countRebalanceRequired(drifts []AllocationDrift) int {
	count := 0
	for _, drift := range drifts {
		if drift.RequiresRebalance {
			count++
		}
	}
	return count
}
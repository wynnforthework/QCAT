package position

import (
	"time"

	"qcat/internal/automation/scheduler/shared"
)

// OptimizationPerformanceMetrics represents performance metrics for optimization
type OptimizationPerformanceMetrics struct {
	OptimizationID         string    `json:"optimization_id"`
	PreOptimizationValue   float64   `json:"pre_optimization_value"`
	PostOptimizationValue  float64   `json:"post_optimization_value"`
	ReturnImprovement      float64   `json:"return_improvement"`
	PreOptimizationRisk    float64   `json:"pre_optimization_risk"`
	PostOptimizationRisk   float64   `json:"post_optimization_risk"`
	RiskReduction          float64   `json:"risk_reduction"`
	PreOptimizationSharpe  float64   `json:"pre_optimization_sharpe"`
	PostOptimizationSharpe float64   `json:"post_optimization_sharpe"`
	SharpeImprovement      float64   `json:"sharpe_improvement"`
	Timestamp              time.Time `json:"timestamp"`
}

// PerformanceReport represents a comprehensive performance report
type PerformanceReport struct {
	ID                  string                         `json:"id"`
	StartDate           time.Time                      `json:"start_date"`
	EndDate             time.Time                      `json:"end_date"`
	PortfolioMetrics    PortfolioMetrics              `json:"portfolio_metrics"`
	BenchmarkMetrics    *PortfolioMetrics             `json:"benchmark_metrics,omitempty"`
	RelativeMetrics     *RelativePerformanceMetrics   `json:"relative_metrics,omitempty"`
	OptimizationMetrics *OptimizationEffectivenessMetrics `json:"optimization_metrics,omitempty"`
	Attribution         []PerformanceAttribution      `json:"attribution"`
	Insights            []string                       `json:"insights"`
	Recommendations     []string                       `json:"recommendations"`
	GeneratedAt         time.Time                      `json:"generated_at"`
}

// RelativePerformanceMetrics represents performance relative to benchmark
type RelativePerformanceMetrics struct {
	ExcessReturn     float64 `json:"excess_return"`
	TrackingError    float64 `json:"tracking_error"`
	InformationRatio float64 `json:"information_ratio"`
	Beta             float64 `json:"beta"`
	Alpha            float64 `json:"alpha"`
}

// OptimizationEffectivenessMetrics represents optimization effectiveness metrics
type OptimizationEffectivenessMetrics struct {
	TotalOptimizations        int     `json:"total_optimizations"`
	SuccessfulOptimizations   int     `json:"successful_optimizations"`
	SuccessRate               float64 `json:"success_rate"`
	AverageReturnImprovement  float64 `json:"average_return_improvement"`
	AverageRiskReduction      float64 `json:"average_risk_reduction"`
	AverageSharpeImprovement  float64 `json:"average_sharpe_improvement"`
	OptimizationFrequency     float64 `json:"optimization_frequency"`
	CumulativeImprovement     float64 `json:"cumulative_improvement"`
}

// OptimizationEffectivenessReport represents detailed optimization effectiveness analysis
type OptimizationEffectivenessReport struct {
	Period                   time.Duration                     `json:"period"`
	StartDate                time.Time                         `json:"start_date"`
	EndDate                  time.Time                         `json:"end_date"`
	TotalOptimizations       int                               `json:"total_optimizations"`
	SuccessRate              float64                           `json:"success_rate"`
	AverageReturnImprovement float64                           `json:"average_return_improvement"`
	AverageRiskReduction     float64                           `json:"average_risk_reduction"`
	OptimizationFrequency    float64                           `json:"optimization_frequency"`
	TriggerAnalysis          map[string]float64                `json:"trigger_analysis"`
	PerformanceByStrategy    map[string]StrategyPerformance    `json:"performance_by_strategy"`
	CostBenefitAnalysis      CostBenefitAnalysis               `json:"cost_benefit_analysis"`
	GeneratedAt              time.Time                         `json:"generated_at"`
}

// StrategyPerformance represents performance metrics for a specific strategy
type StrategyPerformance struct {
	StrategyName         string    `json:"strategy_name"`
	OptimizationCount    int       `json:"optimization_count"`
	SuccessRate          float64   `json:"success_rate"`
	AverageImprovement   float64   `json:"average_improvement"`
	TotalImprovement     float64   `json:"total_improvement"`
	AverageCost          float64   `json:"average_cost"`
	NetBenefit           float64   `json:"net_benefit"`
	LastOptimization     time.Time `json:"last_optimization"`
}

// CostBenefitAnalysis represents cost-benefit analysis of optimizations
type CostBenefitAnalysis struct {
	TotalCosts           float64 `json:"total_costs"`
	TotalBenefits        float64 `json:"total_benefits"`
	NetBenefit           float64 `json:"net_benefit"`
	ROI                  float64 `json:"roi"`
	BreakEvenPoint       float64 `json:"break_even_point"`
	CostPerOptimization  float64 `json:"cost_per_optimization"`
	BenefitPerOptimization float64 `json:"benefit_per_optimization"`
}

// RealTimePerformanceMetrics represents real-time performance metrics
type RealTimePerformanceMetrics struct {
	Timestamp      time.Time            `json:"timestamp"`
	TotalValue     float64              `json:"total_value"`
	UnrealizedPnL  float64              `json:"unrealized_pnl"`
	RealizedPnL    float64              `json:"realized_pnl"`
	TotalPnL       float64              `json:"total_pnl"`
	DailyReturn    float64              `json:"daily_return"`
	RiskMetrics    shared.RiskMetrics   `json:"risk_metrics"`
	PositionCount  int                  `json:"position_count"`
	Leverage       float64              `json:"leverage"`
	MarginUsage    float64              `json:"margin_usage"`
}

// PerformanceBenchmark represents performance benchmark data
type PerformanceBenchmark struct {
	Symbol         string    `json:"symbol"`
	Name           string    `json:"name"`
	CurrentPrice   float64   `json:"current_price"`
	DailyReturn    float64   `json:"daily_return"`
	Volatility     float64   `json:"volatility"`
	SharpeRatio    float64   `json:"sharpe_ratio"`
	MaxDrawdown    float64   `json:"max_drawdown"`
	LastUpdated    time.Time `json:"last_updated"`
}

// PerformanceAlert represents a performance-related alert
type PerformanceAlert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // UNDERPERFORMANCE, HIGH_DRAWDOWN, LOW_SHARPE
	Severity    shared.Severity        `json:"severity"`
	Message     string                 `json:"message"`
	Threshold   float64                `json:"threshold"`
	CurrentValue float64               `json:"current_value"`
	Triggered   time.Time              `json:"triggered"`
	Resolved    *time.Time             `json:"resolved,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// PerformanceComparison represents comparison between different periods or strategies
type PerformanceComparison struct {
	ID              string                 `json:"id"`
	ComparisonType  string                 `json:"comparison_type"` // PERIOD, STRATEGY, BENCHMARK
	BaselineMetrics PortfolioMetrics       `json:"baseline_metrics"`
	CompareMetrics  PortfolioMetrics       `json:"compare_metrics"`
	Differences     map[string]float64     `json:"differences"`
	Improvements    []string               `json:"improvements"`
	Deteriorations  []string               `json:"deteriorations"`
	Significance    float64                `json:"significance"`
	Confidence      float64                `json:"confidence"`
	GeneratedAt     time.Time              `json:"generated_at"`
}

// PerformanceTarget represents performance targets and goals
type PerformanceTarget struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"` // RETURN, SHARPE, DRAWDOWN, VOLATILITY
	TargetValue     float64   `json:"target_value"`
	CurrentValue    float64   `json:"current_value"`
	Achievement     float64   `json:"achievement"` // Percentage of target achieved
	Status          string    `json:"status"` // ON_TRACK, AT_RISK, MISSED
	Deadline        time.Time `json:"deadline"`
	CreatedAt       time.Time `json:"created_at"`
	LastUpdated     time.Time `json:"last_updated"`
}

// PerformanceDecomposition represents decomposition of portfolio performance
type PerformanceDecomposition struct {
	TotalReturn         float64                    `json:"total_return"`
	AssetAllocation     float64                    `json:"asset_allocation"`
	SecuritySelection   float64                    `json:"security_selection"`
	InteractionEffect   float64                    `json:"interaction_effect"`
	TransactionCosts    float64                    `json:"transaction_costs"`
	MarketTiming        float64                    `json:"market_timing"`
	CurrencyEffect      float64                    `json:"currency_effect"`
	ComponentBreakdown  map[string]float64         `json:"component_breakdown"`
	AttributionDetails  []AttributionDetail        `json:"attribution_details"`
}

// AttributionDetail represents detailed attribution for a specific component
type AttributionDetail struct {
	Component       string    `json:"component"`
	Contribution    float64   `json:"contribution"`
	Weight          float64   `json:"weight"`
	Return          float64   `json:"return"`
	BenchmarkWeight float64   `json:"benchmark_weight"`
	BenchmarkReturn float64   `json:"benchmark_return"`
	AllocationEffect float64  `json:"allocation_effect"`
	SelectionEffect float64   `json:"selection_effect"`
	Timestamp       time.Time `json:"timestamp"`
}

// PerformanceRisk represents risk-adjusted performance metrics
type PerformanceRisk struct {
	SharpeRatio         float64 `json:"sharpe_ratio"`
	SortinoRatio        float64 `json:"sortino_ratio"`
	CalmarRatio         float64 `json:"calmar_ratio"`
	TreynorRatio        float64 `json:"treynor_ratio"`
	InformationRatio    float64 `json:"information_ratio"`
	UpsideCapture       float64 `json:"upside_capture"`
	DownsideCapture     float64 `json:"downside_capture"`
	BattingAverage      float64 `json:"batting_average"`
	WinLossRatio        float64 `json:"win_loss_ratio"`
	ProfitFactor        float64 `json:"profit_factor"`
}

// PerformanceRegime represents performance under different market regimes
type PerformanceRegime struct {
	RegimeType      string           `json:"regime_type"` // BULL, BEAR, SIDEWAYS, VOLATILE
	Period          TimePeriod       `json:"period"`
	PortfolioReturn float64          `json:"portfolio_return"`
	BenchmarkReturn float64          `json:"benchmark_return"`
	ExcessReturn    float64          `json:"excess_return"`
	Volatility      float64          `json:"volatility"`
	MaxDrawdown     float64          `json:"max_drawdown"`
	SharpeRatio     float64          `json:"sharpe_ratio"`
	TradeCount      int              `json:"trade_count"`
	WinRate         float64          `json:"win_rate"`
	Confidence      float64          `json:"confidence"`
}

// TimePeriod represents a time period for analysis
type TimePeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Label string    `json:"label"`
}

// PerformanceOptimizationSuggestion represents suggestions for performance improvement
type PerformanceOptimizationSuggestion struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"` // REBALANCE, RISK_ADJUST, DIVERSIFY
	Priority        int                    `json:"priority"`
	Description     string                 `json:"description"`
	ExpectedImpact  float64                `json:"expected_impact"`
	ImplementationCost float64             `json:"implementation_cost"`
	RiskLevel       shared.RiskLevel       `json:"risk_level"`
	Confidence      float64                `json:"confidence"`
	Parameters      map[string]interface{} `json:"parameters"`
	CreatedAt       time.Time              `json:"created_at"`
	Status          string                 `json:"status"` // PENDING, IMPLEMENTED, REJECTED
}

// PerformanceScenario represents performance under different scenarios
type PerformanceScenario struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Probability     float64                `json:"probability"`
	MarketConditions map[string]float64    `json:"market_conditions"`
	ExpectedReturn  float64                `json:"expected_return"`
	ExpectedRisk    float64                `json:"expected_risk"`
	ExpectedSharpe  float64                `json:"expected_sharpe"`
	WorstCase       float64                `json:"worst_case"`
	BestCase        float64                `json:"best_case"`
	Confidence      float64                `json:"confidence"`
	CreatedAt       time.Time              `json:"created_at"`
}

// PerformanceStress represents stress test results
type PerformanceStress struct {
	TestID          string                 `json:"test_id"`
	TestName        string                 `json:"test_name"`
	StressScenario  string                 `json:"stress_scenario"`
	BaselineValue   float64                `json:"baseline_value"`
	StressedValue   float64                `json:"stressed_value"`
	Impact          float64                `json:"impact"`
	ImpactPercent   float64                `json:"impact_percent"`
	RecoveryTime    time.Duration          `json:"recovery_time"`
	Resilience      float64                `json:"resilience"`
	Recommendations []string               `json:"recommendations"`
	TestDate        time.Time              `json:"test_date"`
}

// PerformanceCorrelation represents correlation analysis
type PerformanceCorrelation struct {
	Asset1          string    `json:"asset1"`
	Asset2          string    `json:"asset2"`
	Correlation     float64   `json:"correlation"`
	RollingCorr     []float64 `json:"rolling_correlation"`
	CorrelationStability float64 `json:"correlation_stability"`
	Period          TimePeriod `json:"period"`
	Significance    float64   `json:"significance"`
	LastUpdated     time.Time `json:"last_updated"`
}

// PerformanceSummary represents a high-level performance summary
type PerformanceSummary struct {
	Period              TimePeriod         `json:"period"`
	TotalReturn         float64            `json:"total_return"`
	AnnualizedReturn    float64            `json:"annualized_return"`
	Volatility          float64            `json:"volatility"`
	SharpeRatio         float64            `json:"sharpe_ratio"`
	MaxDrawdown         float64            `json:"max_drawdown"`
	WinRate             float64            `json:"win_rate"`
	ProfitFactor        float64            `json:"profit_factor"`
	BenchmarkReturn     float64            `json:"benchmark_return"`
	ExcessReturn        float64            `json:"excess_return"`
	TrackingError       float64            `json:"tracking_error"`
	InformationRatio    float64            `json:"information_ratio"`
	OptimizationCount   int                `json:"optimization_count"`
	OptimizationSuccess float64            `json:"optimization_success"`
	Status              string             `json:"status"`
	Grade               string             `json:"grade"`
	GeneratedAt         time.Time          `json:"generated_at"`
}
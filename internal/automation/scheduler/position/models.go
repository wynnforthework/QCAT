package position

import (
	"time"

	"qcat/internal/automation/scheduler/shared"
)

// RebalanceInstruction represents an instruction to rebalance a position
type RebalanceInstruction struct {
	ID              string    `json:"id"`
	Symbol          string    `json:"symbol"`
	CurrentSize     float64   `json:"current_size"`
	TargetSize      float64   `json:"target_size"`
	Adjustment      float64   `json:"adjustment"`
	Priority        int       `json:"priority"`
	TransactionCost float64   `json:"transaction_cost"`
	Status          string    `json:"status"` // PENDING, EXECUTING, COMPLETED, FAILED
	ErrorMessage    string    `json:"error_message,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	ExecutedAt      *time.Time `json:"executed_at,omitempty"`
}

// PortfolioMetrics represents portfolio performance metrics
type PortfolioMetrics struct {
	TotalValue        float64   `json:"total_value"`
	TotalReturn       float64   `json:"total_return"`
	SharpeRatio       float64   `json:"sharpe_ratio"`
	MaxDrawdown       float64   `json:"max_drawdown"`
	Volatility        float64   `json:"volatility"`
	Beta              float64   `json:"beta"`
	Alpha             float64   `json:"alpha"`
	InformationRatio  float64   `json:"information_ratio"`
	TrackingError     float64   `json:"tracking_error"`
	VaR               float64   `json:"var"`
	ExpectedShortfall float64   `json:"expected_shortfall"`
	Timestamp         time.Time `json:"timestamp"`
}

// OptimizationReport represents the results of portfolio optimization
type OptimizationReport struct {
	ID                  string                      `json:"id"`
	OptimizationType    string                      `json:"optimization_type"`
	CurrentPositions    []shared.Position           `json:"current_positions"`
	TargetPositions     []shared.TargetPosition     `json:"target_positions"`
	RebalanceInstructions []RebalanceInstruction    `json:"rebalance_instructions"`
	ExpectedMetrics     PortfolioMetrics            `json:"expected_metrics"`
	CurrentMetrics      PortfolioMetrics            `json:"current_metrics"`
	ImprovementMetrics  map[string]float64          `json:"improvement_metrics"`
	Constraints         shared.OptimizationConstraints `json:"constraints"`
	OptimizationTime    time.Duration               `json:"optimization_time"`
	ExecutionStatus     string                      `json:"execution_status"`
	CreatedAt           time.Time                   `json:"created_at"`
	CompletedAt         *time.Time                  `json:"completed_at,omitempty"`
}

// RiskAdjustedReturn represents risk-adjusted return calculations
type RiskAdjustedReturn struct {
	Symbol            string    `json:"symbol"`
	Return            float64   `json:"return"`
	Risk              float64   `json:"risk"`
	SharpeRatio       float64   `json:"sharpe_ratio"`
	SortinoRatio      float64   `json:"sortino_ratio"`
	CalmarRatio       float64   `json:"calmar_ratio"`
	InformationRatio  float64   `json:"information_ratio"`
	TreynorRatio      float64   `json:"treynor_ratio"`
	JensensAlpha      float64   `json:"jensens_alpha"`
	Timestamp         time.Time `json:"timestamp"`
}

// TransactionCostModel represents transaction cost modeling
type TransactionCostModel struct {
	Symbol          string  `json:"symbol"`
	BaseFee         float64 `json:"base_fee"`
	MarketImpactRate float64 `json:"market_impact_rate"`
	BidAskSpread    float64 `json:"bid_ask_spread"`
	LiquidityFactor float64 `json:"liquidity_factor"`
	VolatilityFactor float64 `json:"volatility_factor"`
	SizeFactor      float64 `json:"size_factor"`
}

// PortfolioConstraints represents additional portfolio constraints
type PortfolioConstraints struct {
	MaxPositions      int                `json:"max_positions"`
	MinPositionSize   float64            `json:"min_position_size"`
	MaxPositionSize   float64            `json:"max_position_size"`
	MaxSectorExposure map[string]float64 `json:"max_sector_exposure"`
	MaxCountryExposure map[string]float64 `json:"max_country_exposure"`
	MaxCorrelation    float64            `json:"max_correlation"`
	MinDiversification float64           `json:"min_diversification"`
	TurnoverLimit     float64            `json:"turnover_limit"`
	LeverageLimit     float64            `json:"leverage_limit"`
}

// OptimizationObjective represents optimization objectives
type OptimizationObjective struct {
	Type        string  `json:"type"` // MAX_RETURN, MIN_RISK, MAX_SHARPE, RISK_PARITY
	Weight      float64 `json:"weight"`
	TargetValue float64 `json:"target_value,omitempty"`
}

// PerformanceAttribution represents performance attribution analysis
type PerformanceAttribution struct {
	Symbol              string    `json:"symbol"`
	Weight              float64   `json:"weight"`
	Return              float64   `json:"return"`
	Contribution        float64   `json:"contribution"`
	ActiveWeight        float64   `json:"active_weight"`
	ActiveReturn        float64   `json:"active_return"`
	SelectionEffect     float64   `json:"selection_effect"`
	AllocationEffect    float64   `json:"allocation_effect"`
	InteractionEffect   float64   `json:"interaction_effect"`
	Timestamp           time.Time `json:"timestamp"`
}

// RebalanceEvent represents a rebalancing event
type RebalanceEvent struct {
	ID                string                 `json:"id"`
	Type              string                 `json:"type"` // SCHEDULED, THRESHOLD, EMERGENCY
	Trigger           string                 `json:"trigger"`
	PreRebalanceValue float64                `json:"pre_rebalance_value"`
	PostRebalanceValue float64               `json:"post_rebalance_value"`
	TotalCost         float64                `json:"total_cost"`
	Instructions      []RebalanceInstruction `json:"instructions"`
	ExecutionTime     time.Duration          `json:"execution_time"`
	Success           bool                   `json:"success"`
	ErrorMessage      string                 `json:"error_message,omitempty"`
	Timestamp         time.Time              `json:"timestamp"`
}

// PositionAnalysis represents detailed position analysis
type PositionAnalysis struct {
	Position            shared.Position        `json:"position"`
	RiskMetrics         shared.RiskMetrics     `json:"risk_metrics"`
	PerformanceMetrics  shared.PerformanceMetrics `json:"performance_metrics"`
	Attribution         PerformanceAttribution `json:"attribution"`
	OptimalWeight       float64                `json:"optimal_weight"`
	CurrentWeight       float64                `json:"current_weight"`
	WeightDeviation     float64                `json:"weight_deviation"`
	RecommendedAction   string                 `json:"recommended_action"`
	Confidence          float64                `json:"confidence"`
	LastAnalyzed        time.Time              `json:"last_analyzed"`
}

// OptimizationConfig represents configuration for portfolio optimization
type OptimizationConfig struct {
	Method              string                    `json:"method"` // MEAN_VARIANCE, BLACK_LITTERMAN, RISK_PARITY
	Objectives          []OptimizationObjective   `json:"objectives"`
	Constraints         PortfolioConstraints      `json:"constraints"`
	RiskModel           string                    `json:"risk_model"`
	ReturnModel         string                    `json:"return_model"`
	LookbackPeriod      time.Duration             `json:"lookback_period"`
	RebalanceFrequency  time.Duration             `json:"rebalance_frequency"`
	RebalanceThreshold  float64                   `json:"rebalance_threshold"`
	TransactionCosts    bool                      `json:"transaction_costs"`
	MarketImpact        bool                      `json:"market_impact"`
	MaxIterations       int                       `json:"max_iterations"`
	Tolerance           float64                   `json:"tolerance"`
}

// MarketRegimeData represents market regime information for optimization
type MarketRegimeData struct {
	Regime              string                 `json:"regime"` // BULL, BEAR, SIDEWAYS, VOLATILE
	Confidence          float64                `json:"confidence"`
	Volatility          float64                `json:"volatility"`
	Correlation         float64                `json:"correlation"`
	Momentum            float64                `json:"momentum"`
	Sentiment           float64                `json:"sentiment"`
	LiquidityCondition  string                 `json:"liquidity_condition"`
	RegimeParameters    map[string]interface{} `json:"regime_parameters"`
	DetectedAt          time.Time              `json:"detected_at"`
}

// OptimizationResult represents comprehensive optimization results
type OptimizationResult struct {
	ID                  string                      `json:"id"`
	Config              OptimizationConfig          `json:"config"`
	CurrentPortfolio    []shared.Position           `json:"current_portfolio"`
	OptimalPortfolio    []shared.TargetPosition     `json:"optimal_portfolio"`
	ExpectedReturn      float64                     `json:"expected_return"`
	ExpectedRisk        float64                     `json:"expected_risk"`
	ExpectedSharpe      float64                     `json:"expected_sharpe"`
	ImprovementMetrics  map[string]float64          `json:"improvement_metrics"`
	RebalanceRequired   bool                        `json:"rebalance_required"`
	Instructions        []RebalanceInstruction      `json:"instructions"`
	TotalCost           float64                     `json:"total_cost"`
	NetBenefit          float64                     `json:"net_benefit"`
	Confidence          float64                     `json:"confidence"`
	MarketRegime        MarketRegimeData            `json:"market_regime"`
	OptimizationTime    time.Duration               `json:"optimization_time"`
	Status              string                      `json:"status"`
	CreatedAt           time.Time                   `json:"created_at"`
}
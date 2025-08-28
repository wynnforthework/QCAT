package shared

import (
	"context"
	"time"
)

// SchedulerInterface defines the common interface for all automation schedulers
type SchedulerInterface interface {
	BaseScheduler
	TaskExecutor
	
	// Scheduler-specific methods
	GetName() string
	GetVersion() string
	GetDescription() string
	GetSupportedTasks() []string
	GetMetrics() map[string]interface{}
	GetHealth() HealthStatus
}

// HealthStatus represents the health status of a scheduler
type HealthStatus struct {
	Status      string                 `json:"status"`      // HEALTHY, DEGRADED, UNHEALTHY
	LastCheck   time.Time              `json:"last_check"`
	Errors      []string               `json:"errors,omitempty"`
	Warnings    []string               `json:"warnings,omitempty"`
	Metrics     map[string]interface{} `json:"metrics"`
	Uptime      time.Duration          `json:"uptime"`
	TasksTotal  int64                  `json:"tasks_total"`
	TasksSuccess int64                 `json:"tasks_success"`
	TasksFailed int64                  `json:"tasks_failed"`
}

// RiskSchedulerInterface defines the interface for risk management schedulers
type RiskSchedulerInterface interface {
	SchedulerInterface
	
	// Risk monitoring methods
	MonitorRisk(ctx context.Context) (*RiskAssessment, error)
	HandleRiskAlert(ctx context.Context, alert *RiskAlert) error
	
	// Abnormal market response methods
	DetectAbnormalMarket(ctx context.Context) (*MarketAnomalyReport, error)
	HandleAbnormalMarket(ctx context.Context, anomaly *MarketAnomalyReport) error
	
	// Stop loss adjustment methods
	AdjustStopLoss(ctx context.Context, positions []Position) error
	CalculateOptimalStopLoss(ctx context.Context, position Position) (float64, error)
}

// PositionSchedulerInterface defines the interface for position management schedulers
type PositionSchedulerInterface interface {
	SchedulerInterface
	
	// Position optimization methods
	OptimizePositions(ctx context.Context) (*OptimizationResult, error)
	RebalancePortfolio(ctx context.Context, target *AllocationTarget) error
	
	// Fund allocation methods
	AllocateFunds(ctx context.Context, strategies []Strategy) (*AllocationPlan, error)
	ExecuteAllocation(ctx context.Context, plan *AllocationPlan) error
	
	// Layered position management methods
	ManageLayeredPositions(ctx context.Context, config *LayerConfig) error
	AdjustLayers(ctx context.Context, marketConditions *MarketConditions) error
	
	// Multi-strategy hedging methods
	CalculateHedgeRatios(ctx context.Context) ([]HedgeRatio, error)
	ExecuteHedging(ctx context.Context, ratios []HedgeRatio) error
}

// DataSchedulerInterface defines the interface for data processing schedulers
type DataSchedulerInterface interface {
	SchedulerInterface
	
	// Data cleaning methods
	CleanData(ctx context.Context, dataset *Dataset) (*CleanedDataset, error)
	ValidateDataQuality(ctx context.Context, dataset *Dataset) (*QualityReport, error)
	
	// Backtesting methods
	RunBacktest(ctx context.Context, config *BacktestConfig) (*BacktestResult, error)
	GenerateBacktestReport(ctx context.Context, results []*BacktestResult) (*BacktestReport, error)
	
	// Factor library methods
	UpdateFactorLibrary(ctx context.Context) error
	EvaluateFactors(ctx context.Context, factors []Factor) ([]FactorScore, error)
	
	// Pattern recognition methods
	RecognizePatterns(ctx context.Context, data *MarketData) (*PatternRecognitionResult, error)
	UpdatePatternModels(ctx context.Context, newData *MarketData) error
}

// SystemSchedulerInterface defines the interface for system monitoring schedulers
type SystemSchedulerInterface interface {
	SchedulerInterface
	
	// Health monitoring methods
	CheckSystemHealth(ctx context.Context) (*SystemHealthReport, error)
	TriggerSelfHealing(ctx context.Context, issues []SystemIssue) error
	
	// Security monitoring methods
	MonitorSecurity(ctx context.Context) (*SecurityReport, error)
	HandleSecurityThreat(ctx context.Context, threat *SecurityThreat) error
	
	// Exchange redundancy methods
	ManageExchangeRedundancy(ctx context.Context) error
	SwitchExchange(ctx context.Context, fromExchange, toExchange string) error
	
	// Audit logging methods
	CollectAuditLogs(ctx context.Context) error
	GenerateAuditReport(ctx context.Context, period *TimePeriod) (*AuditReport, error)
}

// LearningSchedulerInterface defines the interface for machine learning schedulers
type LearningSchedulerInterface interface {
	SchedulerInterface
	
	// ML pipeline methods
	TrainModel(ctx context.Context, config *ModelConfig) (*TrainedModel, error)
	EvaluateModel(ctx context.Context, model *TrainedModel) (*ModelMetrics, error)
	DeployModel(ctx context.Context, model *TrainedModel) error
	
	// AutoML methods
	AutoSelectModel(ctx context.Context, dataset *Dataset) (*ModelSelection, error)
	OptimizeHyperparameters(ctx context.Context, model *Model) (*OptimizedModel, error)
	
	// Genetic evolution methods
	EvolveStrategies(ctx context.Context, population []*GeneticCode) ([]*GeneticCode, error)
	EvaluateFitness(ctx context.Context, individuals []*GeneticCode) ([]FitnessScore, error)
}

// Additional data structures for interfaces

// RiskAssessment represents a comprehensive risk assessment
type RiskAssessment struct {
	OverallRisk       RiskLevel              `json:"overall_risk"`
	PortfolioVaR      float64                `json:"portfolio_var"`
	MaxDrawdown       float64                `json:"max_drawdown"`
	ConcentrationRisk float64                `json:"concentration_risk"`
	LiquidityRisk     float64                `json:"liquidity_risk"`
	CorrelationRisk   float64                `json:"correlation_risk"`
	Recommendations   []string               `json:"recommendations"`
	Metrics           map[string]interface{} `json:"metrics"`
	Timestamp         time.Time              `json:"timestamp"`
}

// RiskAlert represents a risk alert
type RiskAlert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    Severity               `json:"severity"`
	Message     string                 `json:"message"`
	Metrics     map[string]interface{} `json:"metrics"`
	Actions     []string               `json:"actions"`
	Timestamp   time.Time              `json:"timestamp"`
}

// OptimizationResult represents the result of position optimization
type OptimizationResult struct {
	TargetPositions   []TargetPosition       `json:"target_positions"`
	ExpectedReturn    float64                `json:"expected_return"`
	ExpectedRisk      float64                `json:"expected_risk"`
	SharpeRatio       float64                `json:"sharpe_ratio"`
	Constraints       OptimizationConstraints `json:"constraints"`
	OptimizationTime  time.Duration          `json:"optimization_time"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// AllocationTarget represents target allocation
type AllocationTarget struct {
	Allocations       map[string]float64     `json:"allocations"`
	TotalCapital      float64                `json:"total_capital"`
	RiskBudget        float64                `json:"risk_budget"`
	Constraints       map[string]interface{} `json:"constraints"`
	RebalanceReason   string                 `json:"rebalance_reason"`
}

// Strategy represents a trading strategy
type Strategy struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Parameters    map[string]interface{} `json:"parameters"`
	Performance   PerformanceMetrics     `json:"performance"`
	RiskMetrics   RiskMetrics            `json:"risk_metrics"`
	Status        string                 `json:"status"`
	CreatedAt     time.Time              `json:"created_at"`
	LastUpdated   time.Time              `json:"last_updated"`
}

// PerformanceMetrics represents performance metrics
type PerformanceMetrics struct {
	TotalReturn       float64   `json:"total_return"`
	SharpeRatio       float64   `json:"sharpe_ratio"`
	MaxDrawdown       float64   `json:"max_drawdown"`
	WinRate           float64   `json:"win_rate"`
	ProfitFactor      float64   `json:"profit_factor"`
	AverageWin        float64   `json:"average_win"`
	AverageLoss       float64   `json:"average_loss"`
	TotalTrades       int       `json:"total_trades"`
	LastUpdated       time.Time `json:"last_updated"`
}

// RiskMetrics represents risk metrics
type RiskMetrics struct {
	VaR               float64   `json:"var"`
	ExpectedShortfall float64   `json:"expected_shortfall"`
	Beta              float64   `json:"beta"`
	Volatility        float64   `json:"volatility"`
	CorrelationRisk   float64   `json:"correlation_risk"`
	ConcentrationRisk float64   `json:"concentration_risk"`
	LastUpdated       time.Time `json:"last_updated"`
}

// AllocationPlan represents a fund allocation plan
type AllocationPlan struct {
	ID                string                 `json:"id"`
	TargetAllocations map[string]float64     `json:"target_allocations"`
	CurrentAllocations map[string]float64    `json:"current_allocations"`
	RequiredTransfers []FundTransfer         `json:"required_transfers"`
	ExpectedImpact    map[string]interface{} `json:"expected_impact"`
	ExecutionPlan     []ExecutionStep        `json:"execution_plan"`
	CreatedAt         time.Time              `json:"created_at"`
}

// ExecutionStep represents a step in execution plan
type ExecutionStep struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Order       int                    `json:"order"`
	Status      string                 `json:"status"`
	ExecutedAt  *time.Time             `json:"executed_at,omitempty"`
}

// MarketConditions represents current market conditions
type MarketConditions struct {
	Volatility        float64                `json:"volatility"`
	Trend             float64                `json:"trend"`
	Liquidity         float64                `json:"liquidity"`
	Sentiment         string                 `json:"sentiment"`
	RegimeType        string                 `json:"regime_type"`
	Indicators        map[string]interface{} `json:"indicators"`
	Timestamp         time.Time              `json:"timestamp"`
}

// HedgeRatio represents a hedge ratio for multi-strategy hedging
type HedgeRatio struct {
	StrategyID        string    `json:"strategy_id"`
	HedgeStrategyID   string    `json:"hedge_strategy_id"`
	Ratio             float64   `json:"ratio"`
	Confidence        float64   `json:"confidence"`
	EffectivenesScore float64   `json:"effectiveness_score"`
	LastUpdated       time.Time `json:"last_updated"`
}

// CleanedDataset represents a cleaned dataset
type CleanedDataset struct {
	OriginalDataset   *Dataset               `json:"original_dataset"`
	CleanedRecords    []map[string]interface{} `json:"cleaned_records"`
	RemovedRecords    []map[string]interface{} `json:"removed_records"`
	Corrections       []DataCorrection       `json:"corrections"`
	QualityScore      float64                `json:"quality_score"`
	CleaningReport    string                 `json:"cleaning_report"`
	ProcessedAt       time.Time              `json:"processed_at"`
}

// DataCorrection represents a data correction
type DataCorrection struct {
	RecordID      string      `json:"record_id"`
	Field         string      `json:"field"`
	OriginalValue interface{} `json:"original_value"`
	CorrectedValue interface{} `json:"corrected_value"`
	CorrectionType string     `json:"correction_type"`
	Confidence    float64     `json:"confidence"`
}

// QualityReport represents a data quality report
type QualityReport struct {
	DatasetID         string                 `json:"dataset_id"`
	OverallScore      float64                `json:"overall_score"`
	CompletenessScore float64                `json:"completeness_score"`
	AccuracyScore     float64                `json:"accuracy_score"`
	ConsistencyScore  float64                `json:"consistency_score"`
	TimelinessScore   float64                `json:"timeliness_score"`
	Issues            []QualityIssue         `json:"issues"`
	Recommendations   []string               `json:"recommendations"`
	Metadata          map[string]interface{} `json:"metadata"`
	GeneratedAt       time.Time              `json:"generated_at"`
}

// QualityIssue represents a data quality issue
type QualityIssue struct {
	Type        string      `json:"type"`
	Severity    Severity    `json:"severity"`
	Description string      `json:"description"`
	Field       string      `json:"field"`
	Count       int         `json:"count"`
	Examples    []interface{} `json:"examples"`
}

// BacktestResult represents the result of a backtest
type BacktestResult struct {
	ID              string                 `json:"id"`
	StrategyID      string                 `json:"strategy_id"`
	Config          *BacktestConfig        `json:"config"`
	Performance     PerformanceMetrics     `json:"performance"`
	RiskMetrics     RiskMetrics            `json:"risk_metrics"`
	Trades          []Trade                `json:"trades"`
	EquityCurve     []EquityPoint          `json:"equity_curve"`
	Drawdowns       []DrawdownPeriod       `json:"drawdowns"`
	Statistics      map[string]interface{} `json:"statistics"`
	ExecutionTime   time.Duration          `json:"execution_time"`
	CompletedAt     time.Time              `json:"completed_at"`
}

// Trade represents a trade in backtest results
type Trade struct {
	ID          string    `json:"id"`
	Symbol      string    `json:"symbol"`
	Side        string    `json:"side"`
	EntryTime   time.Time `json:"entry_time"`
	ExitTime    time.Time `json:"exit_time"`
	EntryPrice  float64   `json:"entry_price"`
	ExitPrice   float64   `json:"exit_price"`
	Quantity    float64   `json:"quantity"`
	PnL         float64   `json:"pnl"`
	Commission  float64   `json:"commission"`
	Duration    time.Duration `json:"duration"`
}

// EquityPoint represents a point in the equity curve
type EquityPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Equity    float64   `json:"equity"`
	Drawdown  float64   `json:"drawdown"`
}

// DrawdownPeriod represents a drawdown period
type DrawdownPeriod struct {
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Peak      float64   `json:"peak"`
	Trough    float64   `json:"trough"`
	Drawdown  float64   `json:"drawdown"`
	Duration  time.Duration `json:"duration"`
	Recovery  time.Duration `json:"recovery"`
}

// BacktestReport represents a comprehensive backtest report
type BacktestReport struct {
	ID              string                 `json:"id"`
	Results         []*BacktestResult      `json:"results"`
	Comparison      *ComparisonAnalysis    `json:"comparison"`
	Summary         *ReportSummary         `json:"summary"`
	Recommendations []string               `json:"recommendations"`
	Charts          []ChartData            `json:"charts"`
	GeneratedAt     time.Time              `json:"generated_at"`
}

// ComparisonAnalysis represents comparison analysis between strategies
type ComparisonAnalysis struct {
	BestStrategy    string                 `json:"best_strategy"`
	WorstStrategy   string                 `json:"worst_strategy"`
	Metrics         map[string]interface{} `json:"metrics"`
	RankingCriteria string                 `json:"ranking_criteria"`
}

// ReportSummary represents a summary of the report
type ReportSummary struct {
	TotalStrategies   int                    `json:"total_strategies"`
	AvgPerformance    float64                `json:"avg_performance"`
	AvgRisk           float64                `json:"avg_risk"`
	AvgSharpeRatio    float64                `json:"avg_sharpe_ratio"`
	KeyInsights       []string               `json:"key_insights"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// ChartData represents chart data for visualization
type ChartData struct {
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Data        []interface{}          `json:"data"`
	Config      map[string]interface{} `json:"config"`
}

// Factor represents a market factor
type Factor struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	Description     string                 `json:"description"`
	Formula         string                 `json:"formula"`
	Parameters      map[string]interface{} `json:"parameters"`
	Performance     FactorPerformance      `json:"performance"`
	Status          string                 `json:"status"`
	CreatedAt       time.Time              `json:"created_at"`
	LastUpdated     time.Time              `json:"last_updated"`
}

// FactorPerformance represents factor performance metrics
type FactorPerformance struct {
	InformationCoefficient float64   `json:"information_coefficient"`
	Significance           float64   `json:"significance"`
	Correlation            float64   `json:"correlation"`
	Stability              float64   `json:"stability"`
	UsageFrequency         int       `json:"usage_frequency"`
	LastEvaluated          time.Time `json:"last_evaluated"`
}

// FactorScore represents a factor evaluation score
type FactorScore struct {
	FactorID    string    `json:"factor_id"`
	Score       float64   `json:"score"`
	Rank        int       `json:"rank"`
	Percentile  float64   `json:"percentile"`
	Confidence  float64   `json:"confidence"`
	EvaluatedAt time.Time `json:"evaluated_at"`
}

// PatternRecognitionResult represents pattern recognition results
type PatternRecognitionResult struct {
	Patterns        []RecognizedPattern    `json:"patterns"`
	MarketRegime    *MarketRegime          `json:"market_regime"`
	Confidence      float64                `json:"confidence"`
	Recommendations []string               `json:"recommendations"`
	Metadata        map[string]interface{} `json:"metadata"`
	RecognizedAt    time.Time              `json:"recognized_at"`
}

// RecognizedPattern represents a recognized market pattern
type RecognizedPattern struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Confidence  float64                `json:"confidence"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Parameters  map[string]interface{} `json:"parameters"`
	Prediction  *PatternPrediction     `json:"prediction,omitempty"`
}

// PatternPrediction represents a pattern-based prediction
type PatternPrediction struct {
	Direction   string    `json:"direction"` // UP, DOWN, SIDEWAYS
	Magnitude   float64   `json:"magnitude"`
	Probability float64   `json:"probability"`
	TimeHorizon time.Duration `json:"time_horizon"`
	Confidence  float64   `json:"confidence"`
}

// SystemHealthReport represents a system health report
type SystemHealthReport struct {
	OverallStatus   string                 `json:"overall_status"`
	Components      []ComponentHealth      `json:"components"`
	ResourceUsage   ResourceUsage          `json:"resource_usage"`
	Performance     SystemPerformance      `json:"performance"`
	Issues          []SystemIssue          `json:"issues"`
	Recommendations []string               `json:"recommendations"`
	GeneratedAt     time.Time              `json:"generated_at"`
}

// ComponentHealth represents the health of a system component
type ComponentHealth struct {
	Name        string                 `json:"name"`
	Status      string                 `json:"status"`
	Uptime      time.Duration          `json:"uptime"`
	LastCheck   time.Time              `json:"last_check"`
	Metrics     map[string]interface{} `json:"metrics"`
	Errors      []string               `json:"errors,omitempty"`
}

// ResourceUsage represents system resource usage
type ResourceUsage struct {
	CPU         float64   `json:"cpu"`
	Memory      float64   `json:"memory"`
	Disk        float64   `json:"disk"`
	Network     float64   `json:"network"`
	Connections int       `json:"connections"`
	Timestamp   time.Time `json:"timestamp"`
}

// SystemPerformance represents system performance metrics
type SystemPerformance struct {
	Throughput      float64       `json:"throughput"`
	Latency         time.Duration `json:"latency"`
	ErrorRate       float64       `json:"error_rate"`
	SuccessRate     float64       `json:"success_rate"`
	ResponseTime    time.Duration `json:"response_time"`
	QueueLength     int           `json:"queue_length"`
}

// SystemIssue represents a system issue
type SystemIssue struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    Severity               `json:"severity"`
	Component   string                 `json:"component"`
	Description string                 `json:"description"`
	Impact      string                 `json:"impact"`
	Resolution  string                 `json:"resolution,omitempty"`
	DetectedAt  time.Time              `json:"detected_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SecurityReport represents a security monitoring report
type SecurityReport struct {
	OverallRisk     RiskLevel              `json:"overall_risk"`
	Threats         []SecurityThreat       `json:"threats"`
	Incidents       []SecurityIncident     `json:"incidents"`
	Recommendations []string               `json:"recommendations"`
	Metrics         map[string]interface{} `json:"metrics"`
	GeneratedAt     time.Time              `json:"generated_at"`
}

// SecurityIncident represents a security incident
type SecurityIncident struct {
	ID          string                 `json:"id"`
	Type        ThreatType             `json:"type"`
	Severity    Severity               `json:"severity"`
	Status      string                 `json:"status"`
	Description string                 `json:"description"`
	Source      string                 `json:"source"`
	Target      string                 `json:"target"`
	Actions     []string               `json:"actions"`
	DetectedAt  time.Time              `json:"detected_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SecurityThreat represents a security threat (already defined in models.go)

// AuditReport represents an audit report
type AuditReport struct {
	ID              string                 `json:"id"`
	Period          *TimePeriod            `json:"period"`
	TotalEvents     int64                  `json:"total_events"`
	EventsByType    map[string]int64       `json:"events_by_type"`
	EventsBySeverity map[string]int64      `json:"events_by_severity"`
	Compliance      ComplianceStatus       `json:"compliance"`
	Findings        []AuditFinding         `json:"findings"`
	Recommendations []string               `json:"recommendations"`
	GeneratedAt     time.Time              `json:"generated_at"`
}

// ComplianceStatus represents compliance status
type ComplianceStatus struct {
	Overall     string                 `json:"overall"`
	Requirements map[string]string     `json:"requirements"`
	Score       float64                `json:"score"`
	Issues      []ComplianceIssue      `json:"issues"`
}

// ComplianceIssue represents a compliance issue
type ComplianceIssue struct {
	Requirement string    `json:"requirement"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	Severity    Severity  `json:"severity"`
	DetectedAt  time.Time `json:"detected_at"`
}

// AuditFinding represents an audit finding
type AuditFinding struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    Severity               `json:"severity"`
	Description string                 `json:"description"`
	Evidence    []string               `json:"evidence"`
	Impact      string                 `json:"impact"`
	Recommendation string              `json:"recommendation"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ModelConfig represents ML model configuration
type ModelConfig struct {
	Algorithm       string                 `json:"algorithm"`
	Parameters      map[string]interface{} `json:"parameters"`
	TrainingData    *TrainingDataset       `json:"training_data"`
	ValidationSplit float64                `json:"validation_split"`
	EarlyStopping   bool                   `json:"early_stopping"`
	Timeout         time.Duration          `json:"timeout"`
}

// TrainedModel represents a trained ML model
type TrainedModel struct {
	ID              string                 `json:"id"`
	Config          *ModelConfig           `json:"config"`
	Metrics         *ModelMetrics          `json:"metrics"`
	Artifacts       map[string]interface{} `json:"artifacts"`
	Version         string                 `json:"version"`
	Status          string                 `json:"status"`
	TrainedAt       time.Time              `json:"trained_at"`
	DeployedAt      *time.Time             `json:"deployed_at,omitempty"`
}

// ModelSelection represents AutoML model selection result
type ModelSelection struct {
	BestModel       *Model                 `json:"best_model"`
	Candidates      []*Model               `json:"candidates"`
	SelectionCriteria string               `json:"selection_criteria"`
	Metrics         map[string]interface{} `json:"metrics"`
	SelectedAt      time.Time              `json:"selected_at"`
}

// Model represents a machine learning model
type Model struct {
	ID          string                 `json:"id"`
	Algorithm   string                 `json:"algorithm"`
	Parameters  map[string]interface{} `json:"parameters"`
	Performance map[string]float64     `json:"performance"`
	Complexity  float64                `json:"complexity"`
	TrainingTime time.Duration         `json:"training_time"`
}

// OptimizedModel represents a hyperparameter-optimized model
type OptimizedModel struct {
	BaseModel         *Model                 `json:"base_model"`
	OptimalParameters map[string]interface{} `json:"optimal_parameters"`
	OptimizationHistory []OptimizationStep   `json:"optimization_history"`
	ImprovementScore  float64                `json:"improvement_score"`
	OptimizedAt       time.Time              `json:"optimized_at"`
}

// OptimizationStep represents a step in hyperparameter optimization
type OptimizationStep struct {
	Iteration   int                    `json:"iteration"`
	Parameters  map[string]interface{} `json:"parameters"`
	Score       float64                `json:"score"`
	Duration    time.Duration          `json:"duration"`
	Timestamp   time.Time              `json:"timestamp"`
}
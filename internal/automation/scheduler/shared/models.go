package shared

import (
	"context"
	"time"
)

// ErrorSeverity defines the severity level of errors
type ErrorSeverity int

const (
	ErrorSeverityLow ErrorSeverity = iota
	ErrorSeverityMedium
	ErrorSeverityHigh
	ErrorSeverityCritical
)

// String returns the string representation of ErrorSeverity
func (es ErrorSeverity) String() string {
	switch es {
	case ErrorSeverityLow:
		return "LOW"
	case ErrorSeverityMedium:
		return "MEDIUM"
	case ErrorSeverityHigh:
		return "HIGH"
	case ErrorSeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// AutomationError represents an error in the automation system
type AutomationError struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Severity  ErrorSeverity          `json:"severity"`
	Component string                 `json:"component"`
	Timestamp time.Time              `json:"timestamp"`
	Context   map[string]interface{} `json:"context"`
	Retryable bool                   `json:"retryable"`
}

// Error implements the error interface
func (ae *AutomationError) Error() string {
	return ae.Message
}

// RiskLevel defines risk levels for various operations
type RiskLevel int

const (
	RiskLevelLow RiskLevel = iota
	RiskLevelMedium
	RiskLevelHigh
	RiskLevelCritical
)

// String returns the string representation of RiskLevel
func (rl RiskLevel) String() string {
	switch rl {
	case RiskLevelLow:
		return "LOW"
	case RiskLevelMedium:
		return "MEDIUM"
	case RiskLevelHigh:
		return "HIGH"
	case RiskLevelCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// Severity defines alert severity levels
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

// String returns the string representation of Severity
func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARNING"
	case SeverityError:
		return "ERROR"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// AnomalyType defines types of market anomalies
type AnomalyType int

const (
	AnomalyTypeVolatilitySpike AnomalyType = iota
	AnomalyTypeLiquidityDrop
	AnomalyTypeCorrelationBreakdown
	AnomalyTypePriceSpike
	AnomalyTypeVolumeAnomaly
)

// String returns the string representation of AnomalyType
func (at AnomalyType) String() string {
	switch at {
	case AnomalyTypeVolatilitySpike:
		return "VOLATILITY_SPIKE"
	case AnomalyTypeLiquidityDrop:
		return "LIQUIDITY_DROP"
	case AnomalyTypeCorrelationBreakdown:
		return "CORRELATION_BREAKDOWN"
	case AnomalyTypePriceSpike:
		return "PRICE_SPIKE"
	case AnomalyTypeVolumeAnomaly:
		return "VOLUME_ANOMALY"
	default:
		return "UNKNOWN"
	}
}

// AlertSeverity defines alert severity levels
type AlertSeverity int

const (
	AlertSeverityLow AlertSeverity = iota
	AlertSeverityMedium
	AlertSeverityHigh
	AlertSeverityCritical
)

// String returns the string representation of AlertSeverity
func (as AlertSeverity) String() string {
	switch as {
	case AlertSeverityLow:
		return "LOW"
	case AlertSeverityMedium:
		return "MEDIUM"
	case AlertSeverityHigh:
		return "HIGH"
	case AlertSeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// ThreatType defines types of security threats
type ThreatType int

const (
	ThreatTypeUnauthorizedAccess ThreatType = iota
	ThreatTypeAbnormalTrading
	ThreatTypeAPIAbuse
	ThreatTypeSuspiciousLogin
	ThreatTypeDataBreach
)

// String returns the string representation of ThreatType
func (tt ThreatType) String() string {
	switch tt {
	case ThreatTypeUnauthorizedAccess:
		return "UNAUTHORIZED_ACCESS"
	case ThreatTypeAbnormalTrading:
		return "ABNORMAL_TRADING"
	case ThreatTypeAPIAbuse:
		return "API_ABUSE"
	case ThreatTypeSuspiciousLogin:
		return "SUSPICIOUS_LOGIN"
	case ThreatTypeDataBreach:
		return "DATA_BREACH"
	default:
		return "UNKNOWN"
	}
}

// DataType defines types of data
type DataType int

const (
	DataTypeMarket DataType = iota
	DataTypeTrading
	DataTypeRisk
	DataTypePerformance
	DataTypeSystem
)

// String returns the string representation of DataType
func (dt DataType) String() string {
	switch dt {
	case DataTypeMarket:
		return "MARKET"
	case DataTypeTrading:
		return "TRADING"
	case DataTypeRisk:
		return "RISK"
	case DataTypePerformance:
		return "PERFORMANCE"
	case DataTypeSystem:
		return "SYSTEM"
	default:
		return "UNKNOWN"
	}
}

// Position represents a trading position
type Position struct {
	ID           string    `json:"id"`
	Symbol       string    `json:"symbol"`
	Side         string    `json:"side"` // LONG, SHORT
	Size         float64   `json:"size"`
	EntryPrice   float64   `json:"entry_price"`
	CurrentPrice float64   `json:"current_price"`
	UnrealizedPnL float64  `json:"unrealized_pnl"`
	RealizedPnL   float64  `json:"realized_pnl"`
	Leverage     float64   `json:"leverage"`
	MarginUsed   float64   `json:"margin_used"`
	Timestamp    time.Time `json:"timestamp"`
}

// Order represents a trading order
type Order struct {
	ID           string    `json:"id"`
	Symbol       string    `json:"symbol"`
	Side         string    `json:"side"` // BUY, SELL
	Type         string    `json:"type"` // MARKET, LIMIT, STOP
	Size         float64   `json:"size"`
	Price        float64   `json:"price"`
	StopPrice    float64   `json:"stop_price,omitempty"`
	Status       string    `json:"status"` // PENDING, PARTIALLY_FILLED, FILLED, CANCELLED
	FilledSize   float64   `json:"filled_size"`
	RemainingSize float64  `json:"remaining_size"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PositionRisk represents risk metrics for a position
type PositionRisk struct {
	Position         Position `json:"position"`
	VaR              float64  `json:"var"`              // Value at Risk
	ExpectedShortfall float64 `json:"expected_shortfall"`
	Beta             float64  `json:"beta"`
	Volatility       float64  `json:"volatility"`
	ConcentrationRisk float64 `json:"concentration_risk"`
	LiquidityRisk    float64  `json:"liquidity_risk"`
}

// MarketRegime represents the current market state
type MarketRegime struct {
	Type        string    `json:"type"`        // BULL, BEAR, SIDEWAYS, VOLATILE
	Confidence  float64   `json:"confidence"`  // 0-1
	Volatility  float64   `json:"volatility"`
	Trend       float64   `json:"trend"`       // -1 to 1
	Momentum    float64   `json:"momentum"`
	Timestamp   time.Time `json:"timestamp"`
}

// Layer represents a position layer in layered position management
type Layer struct {
	ID          string    `json:"id"`
	Level       int       `json:"level"`
	Size        float64   `json:"size"`
	EntryPrice  float64   `json:"entry_price"`
	StopLoss    float64   `json:"stop_loss"`
	TakeProfit  float64   `json:"take_profit"`
	Status      string    `json:"status"` // PENDING, ACTIVE, CLOSED
	CreatedAt   time.Time `json:"created_at"`
}

// RiskParams represents risk parameters for position management
type RiskParams struct {
	MaxLeverage      float64 `json:"max_leverage"`
	MaxPositionSize  float64 `json:"max_position_size"`
	StopLossPercent  float64 `json:"stop_loss_percent"`
	TakeProfitPercent float64 `json:"take_profit_percent"`
	MaxDrawdown      float64 `json:"max_drawdown"`
	VaRLimit         float64 `json:"var_limit"`
}

// Anomaly represents a data anomaly
type Anomaly struct {
	ID          string                 `json:"id"`
	Type        AnomalyType            `json:"type"`
	Severity    Severity               `json:"severity"`
	Field       string                 `json:"field"`
	Value       interface{}            `json:"value"`
	ExpectedMin interface{}            `json:"expected_min"`
	ExpectedMax interface{}            `json:"expected_max"`
	Confidence  float64                `json:"confidence"`
	Metadata    map[string]interface{} `json:"metadata"`
	Timestamp   time.Time              `json:"timestamp"`
}

// DataSchema represents the schema of a dataset
type DataSchema struct {
	Fields      []FieldSchema          `json:"fields"`
	Constraints map[string]interface{} `json:"constraints"`
	Version     string                 `json:"version"`
}

// FieldSchema represents the schema of a data field
type FieldSchema struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Required bool        `json:"required"`
	Min      interface{} `json:"min,omitempty"`
	Max      interface{} `json:"max,omitempty"`
}

// PreprocessConfig represents preprocessing configuration
type PreprocessConfig struct {
	Normalization bool                   `json:"normalization"`
	Scaling       string                 `json:"scaling"` // STANDARD, MINMAX, ROBUST
	Encoding      map[string]string      `json:"encoding"`
	Imputation    map[string]interface{} `json:"imputation"`
}

// FitnessScore represents a fitness score in genetic algorithms
type FitnessScore struct {
	ID           string                 `json:"id"`
	Score        float64                `json:"score"`
	Metrics      map[string]float64     `json:"metrics"`
	Objectives   []float64              `json:"objectives"`
	Constraints  []float64              `json:"constraints"`
	Metadata     map[string]interface{} `json:"metadata"`
	Timestamp    time.Time              `json:"timestamp"`
}

// TimePeriod represents a time period
type TimePeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// BaseScheduler defines the common interface for all schedulers
type BaseScheduler interface {
	Start() error
	Stop() error
	IsRunning() bool
	GetStatus() map[string]interface{}
}

// TaskExecutor defines the interface for task execution
type TaskExecutor interface {
	Execute(ctx context.Context, task interface{}) error
	CanExecute(task interface{}) bool
	GetExecutorType() string
}

// EventPublisher defines the interface for publishing events
type EventPublisher interface {
	Publish(ctx context.Context, event interface{}) error
	Subscribe(ctx context.Context, eventType string, handler func(interface{}) error) error
}

// MetricsCollector defines the interface for collecting metrics
type MetricsCollector interface {
	Counter(name string, tags map[string]string) error
	Gauge(name string, value float64, tags map[string]string) error
	Histogram(name string, value float64, tags map[string]string) error
	Timer(name string, duration time.Duration, tags map[string]string) error
}

// ConfigProvider defines the interface for configuration management
type ConfigProvider interface {
	Get(key string) interface{}
	GetString(key string) string
	GetInt(key string) int
	GetFloat64(key string) float64
	GetBool(key string) bool
	GetDuration(key string) time.Duration
	Set(key string, value interface{}) error
	Reload() error
}

// Additional data structures needed by interfaces

// MarketAnomalyReport represents detected market anomalies
type MarketAnomalyReport struct {
	AnomalyType        AnomalyType `json:"anomaly_type"`
	Severity           Severity    `json:"severity"`
	AffectedSymbols    []string    `json:"affected_symbols"`
	DetectionTime      time.Time   `json:"detection_time"`
	Confidence         float64     `json:"confidence"`
	Metrics            map[string]float64 `json:"metrics"`
	RecommendedActions []string    `json:"recommended_actions"`
	Description        string      `json:"description"`
}

// Dataset represents a data processing dataset
type Dataset struct {
	ID                 string                 `json:"id"`
	Type               DataType               `json:"type"`
	Records            []map[string]interface{} `json:"records"`
	Schema             DataSchema             `json:"schema"`
	QualityScore       float64                `json:"quality_score"`
	LastUpdated        time.Time              `json:"last_updated"`
}

// BacktestConfig represents backtesting configuration
type BacktestConfig struct {
	StrategyID         string        `json:"strategy_id"`
	StartDate          time.Time     `json:"start_date"`
	EndDate            time.Time     `json:"end_date"`
	InitialCapital     float64       `json:"initial_capital"`
	BenchmarkSymbol    string        `json:"benchmark_symbol"`
	Parameters         map[string]interface{} `json:"parameters"`
}

// TargetPosition represents a target position for optimization
type TargetPosition struct {
	Symbol             string    `json:"symbol"`
	TargetSize         float64   `json:"target_size"`
	CurrentSize        float64   `json:"current_size"`
	Adjustment         float64   `json:"adjustment"`
	Priority           int       `json:"priority"`
	Rationale          string    `json:"rationale"`
}

// OptimizationConstraints represents constraints for position optimization
type OptimizationConstraints struct {
	MaxPositionSize    float64            `json:"max_position_size"`
	MaxLeverage        float64            `json:"max_leverage"`
	MinDiversification float64            `json:"min_diversification"`
	TransactionCosts   map[string]float64 `json:"transaction_costs"`
	RiskBudget         float64            `json:"risk_budget"`
}

// LayerConfig represents layered position configuration
type LayerConfig struct {
	Symbol             string        `json:"symbol"`
	Layers             []Layer       `json:"layers"`
	TotalSize          float64       `json:"total_size"`
	RiskParameters     RiskParams    `json:"risk_parameters"`
	ExecutionStrategy  string        `json:"execution_strategy"`
}

// FundTransfer represents a fund transfer operation
type FundTransfer struct {
	ID               string                 `json:"id"`
	Type             string                 `json:"type"` // HOT_TO_COLD, COLD_TO_HOT, EXCHANGE_REBALANCE
	FromAddress      string                 `json:"from_address"`
	ToAddress        string                 `json:"to_address"`
	Amount           float64                `json:"amount"`
	Currency         string                 `json:"currency"`
	Status           string                 `json:"status"`
	Priority         int                    `json:"priority"`
	EstimatedFee     float64                `json:"estimated_fee"`
	ActualFee        float64                `json:"actual_fee"`
	TransactionHash  string                 `json:"transaction_hash"`
	Confirmations    int                    `json:"confirmations"`
	RequiredConfirms int                    `json:"required_confirms"`
	CreatedAt        time.Time              `json:"created_at"`
	ExecutedAt       *time.Time             `json:"executed_at"`
	CompletedAt      *time.Time             `json:"completed_at"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// MarketData represents market data for analysis
type MarketData struct {
	Symbol      string    `json:"symbol"`
	Price       float64   `json:"price"`
	Volume      float64   `json:"volume"`
	Volatility  float64   `json:"volatility"`
	Liquidity   float64   `json:"liquidity"`
	Timestamp   time.Time `json:"timestamp"`
}

// SecurityThreat represents a security threat
type SecurityThreat struct {
	ID          string                 `json:"id"`
	Type        ThreatType             `json:"type"`
	Severity    Severity               `json:"severity"`
	Source      string                 `json:"source"`
	Target      string                 `json:"target"`
	Description string                 `json:"description"`
	DetectedAt  time.Time              `json:"detected_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	Actions     []string               `json:"actions"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ModelMetrics represents ML model performance metrics
type ModelMetrics struct {
	Accuracy           float64       `json:"accuracy"`
	Precision          float64       `json:"precision"`
	Recall             float64       `json:"recall"`
	F1Score            float64       `json:"f1_score"`
	SharpeRatio        float64       `json:"sharpe_ratio"`
	MaxDrawdown        float64       `json:"max_drawdown"`
	ValidationMetrics  map[string]float64 `json:"validation_metrics"`
}

// TrainingDataset represents ML training data
type TrainingDataset struct {
	Features           [][]float64   `json:"features"`
	Labels             []float64     `json:"labels"`
	FeatureNames       []string      `json:"feature_names"`
	ValidationSplit    float64       `json:"validation_split"`
	Preprocessing      PreprocessConfig `json:"preprocessing"`
}

// GeneticCode represents genetic algorithm code
type GeneticCode struct {
	ID                 string                 `json:"id"`
	Genes              map[string]interface{} `json:"genes"`
	Generation         int                    `json:"generation"`
	ParentIDs          []string               `json:"parent_ids"`
	MutationRate       float64                `json:"mutation_rate"`
	CreatedAt          time.Time              `json:"created_at"`
}
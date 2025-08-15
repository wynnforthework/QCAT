package api

import "time"

// OptimizationRequest represents an optimization request
type OptimizationRequest struct {
	StrategyID string                 `json:"strategy_id" binding:"required"`
	Method     string                 `json:"method" binding:"required"` // grid, bayesian, cmaes, genetic
	Params     map[string]interface{} `json:"params"`
	Objective  string                 `json:"objective"` // sharpe, calmar, sortino
	StartDate  string                 `json:"start_date"`
	EndDate    string                 `json:"end_date"`
}

// OptimizationTask represents an optimization task
type OptimizationTask struct {
	ID         string    `json:"id"`
	StrategyID string    `json:"strategy_id"`
	Method     string    `json:"method"`
	Status     string    `json:"status"` // running, completed, failed
	Progress   float64   `json:"progress"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// OptimizationResult represents optimization results
type OptimizationResult struct {
	TaskID     string                 `json:"task_id"`
	BestParams map[string]interface{} `json:"best_params"`
	Metrics    map[string]float64     `json:"metrics"`
	History    []map[string]interface{} `json:"history"`
}

// Strategy represents a trading strategy
type Strategy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"` // stopped, running, paused
	Version     string                 `json:"version"`
	Params      map[string]interface{} `json:"params"`
	Performance map[string]float64     `json:"performance"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// StrategyPromoteRequest represents a strategy promotion request
type StrategyPromoteRequest struct {
	VersionID string `json:"version_id" binding:"required"`
	Stage     string `json:"stage"` // canary, production
}

// PortfolioOverview represents portfolio overview data
type PortfolioOverview struct {
	TotalEquity    float64 `json:"total_equity"`
	TotalPnL       float64 `json:"total_pnl"`
	Drawdown       float64 `json:"drawdown"`
	SharpeRatio    float64 `json:"sharpe_ratio"`
	Volatility     float64 `json:"volatility"`
	MaxDrawdown    float64 `json:"max_drawdown"`
	WinRate        float64 `json:"win_rate"`
	ProfitFactor   float64 `json:"profit_factor"`
}

// PortfolioAllocation represents portfolio allocation
type PortfolioAllocation struct {
	StrategyID string  `json:"strategy_id"`
	StrategyName string `json:"strategy_name"`
	Weight     float64 `json:"weight"`
	TargetWeight float64 `json:"target_weight"`
	PnL        float64 `json:"pnl"`
	Exposure   float64 `json:"exposure"`
}

// RebalanceRequest represents a rebalance request
type RebalanceRequest struct {
	Mode string `json:"mode"` // bandit, target_vol, manual
}

// RiskOverview represents risk overview data
type RiskOverview struct {
	TotalExposure float64 `json:"total_exposure"`
	MaxDrawdown   float64 `json:"max_drawdown"`
	VaR95         float64 `json:"var_95"`
	VaR99         float64 `json:"var_99"`
	CurrentRisk   float64 `json:"current_risk"`
	RiskBudget    float64 `json:"risk_budget"`
}

// RiskLimit represents a risk limit
type RiskLimit struct {
	Symbol         string  `json:"symbol"`
	MaxLeverage    float64 `json:"max_leverage"`
	MaxPositionSize float64 `json:"max_position_size"`
	MaxDrawdown    float64 `json:"max_drawdown"`
	StopLoss       float64 `json:"stop_loss"`
	TakeProfit     float64 `json:"take_profit"`
}

// CircuitBreaker represents a circuit breaker
type CircuitBreaker struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"` // drawdown, loss, volatility
	Threshold   float64 `json:"threshold"`
	Action      string  `json:"action"` // stop, reduce, alert
	Enabled     bool    `json:"enabled"`
	Triggered   bool    `json:"triggered"`
	TriggeredAt *time.Time `json:"triggered_at,omitempty"`
}

// HotSymbol represents a hot symbol
type HotSymbol struct {
	Symbol      string  `json:"symbol"`
	Score       float64 `json:"score"`
	VolJump     float64 `json:"vol_jump"`
	Turnover    float64 `json:"turnover"`
	OIDelta     float64 `json:"oi_delta"`
	FundingZ    float64 `json:"funding_z"`
	RegimeShift float64 `json:"regime_shift"`
	Approved    bool    `json:"approved"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WhitelistItem represents a whitelist item
type WhitelistItem struct {
	Symbol    string    `json:"symbol"`
	AddedBy   string    `json:"added_by"`
	AddedAt   time.Time `json:"added_at"`
	Reason    string    `json:"reason"`
	RiskLevel string    `json:"risk_level"` // low, medium, high
}

// StrategyMetrics represents strategy metrics
type StrategyMetrics struct {
	StrategyID   string    `json:"strategy_id"`
	PnL          float64   `json:"pnl"`
	Drawdown     float64   `json:"drawdown"`
	SharpeRatio  float64   `json:"sharpe_ratio"`
	CalmarRatio  float64   `json:"calmar_ratio"`
	WinRate      float64   `json:"win_rate"`
	ProfitFactor float64   `json:"profit_factor"`
	MaxDrawdown  float64   `json:"max_drawdown"`
	Volatility   float64   `json:"volatility"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SystemMetrics represents system metrics
type SystemMetrics struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	Goroutines  int     `json:"goroutines"`
	Uptime      float64 `json:"uptime"`
	Timestamp   time.Time `json:"timestamp"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	ResourceID string                `json:"resource_id"`
	Details   map[string]interface{} `json:"details"`
	Result    string                 `json:"result"` // success, failure
	Duration  float64                `json:"duration"`
	Timestamp time.Time              `json:"timestamp"`
}

// DecisionChain represents a decision chain
type DecisionChain struct {
	ID          string                 `json:"id"`
	StrategyID  string                 `json:"strategy_id"`
	Symbol      string                 `json:"symbol"`
	Timestamp   time.Time              `json:"timestamp"`
	Decisions   []Decision             `json:"decisions"`
	FinalAction string                 `json:"final_action"`
	Context     map[string]interface{} `json:"context"`
}

// Decision represents a decision in a decision chain
type Decision struct {
	Step      int                    `json:"step"`
	Type      string                 `json:"type"`
	Input     map[string]interface{} `json:"input"`
	Output    map[string]interface{} `json:"output"`
	Reason    string                 `json:"reason"`
	Timestamp time.Time              `json:"timestamp"`
}

// PerformanceMetric represents a performance metric
type PerformanceMetric struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	Category  string    `json:"category"`
	Timestamp time.Time `json:"timestamp"`
}

// ExportReportRequest represents an export report request
type ExportReportRequest struct {
	Type      string `json:"type" binding:"required"` // audit, performance, strategy
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Format    string `json:"format"` // json, csv, pdf
}

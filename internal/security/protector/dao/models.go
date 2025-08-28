package dao

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// HistoricalReturn represents a daily return record
type HistoricalReturn struct {
	ID              int64     `db:"id" json:"id"`
	Date            time.Time `db:"date" json:"date"`
	ReturnValue     float64   `db:"return_value" json:"return_value"`
	PortfolioValue  *float64  `db:"portfolio_value" json:"portfolio_value,omitempty"`
	BenchmarkReturn *float64  `db:"benchmark_return" json:"benchmark_return,omitempty"`
	Volatility      *float64  `db:"volatility" json:"volatility,omitempty"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

// HistoricalEquity represents an equity value record
type HistoricalEquity struct {
	ID               int64     `db:"id" json:"id"`
	Timestamp        time.Time `db:"timestamp" json:"timestamp"`
	EquityValue      float64   `db:"equity_value" json:"equity_value"`
	AvailableBalance *float64  `db:"available_balance" json:"available_balance,omitempty"`
	LockedBalance    *float64  `db:"locked_balance" json:"locked_balance,omitempty"`
	UnrealizedPnL    *float64  `db:"unrealized_pnl" json:"unrealized_pnl,omitempty"`
	RealizedPnL      *float64  `db:"realized_pnl" json:"realized_pnl,omitempty"`
	TotalPositions   int       `db:"total_positions" json:"total_positions"`
	ActivePositions  int       `db:"active_positions" json:"active_positions"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

// RiskSnapshot represents a risk calculation snapshot
type RiskSnapshot struct {
	ID                int64     `db:"id" json:"id"`
	Timestamp         time.Time `db:"timestamp" json:"timestamp"`
	RiskLevel         string    `db:"risk_level" json:"risk_level"`
	RiskScore         float64   `db:"risk_score" json:"risk_score"`
	VaR95             float64   `db:"var_95" json:"var_95"`
	ExpectedShortfall float64   `db:"expected_shortfall" json:"expected_shortfall"`
	MaxDrawdown       float64   `db:"max_drawdown" json:"max_drawdown"`
	VolatilityIndex   float64   `db:"volatility_index" json:"volatility_index"`
	Leverage          float64   `db:"leverage" json:"leverage"`
	Concentration     float64   `db:"concentration" json:"concentration"`
	PortfolioBeta     *float64  `db:"portfolio_beta" json:"portfolio_beta,omitempty"`
	SharpeRatio       *float64  `db:"sharpe_ratio" json:"sharpe_ratio,omitempty"`
	SortinoRatio      *float64  `db:"sortino_ratio" json:"sortino_ratio,omitempty"`
	CalmarRatio       *float64  `db:"calmar_ratio" json:"calmar_ratio,omitempty"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
}

// TransferRecord represents a fund transfer record
type TransferRecord struct {
	ID                     string                 `db:"id" json:"id"`
	Type                   string                 `db:"type" json:"type"`
	Amount                 float64                `db:"amount" json:"amount"`
	Currency               string                 `db:"currency" json:"currency"`
	FromAddress            string                 `db:"from_address" json:"from_address"`
	ToAddress              string                 `db:"to_address" json:"to_address"`
	Status                 string                 `db:"status" json:"status"`
	TriggerReason          *string                `db:"trigger_reason" json:"trigger_reason,omitempty"`
	TransactionHash        *string                `db:"transaction_hash" json:"transaction_hash,omitempty"`
	EstimatedFee           *float64               `db:"estimated_fee" json:"estimated_fee,omitempty"`
	ActualFee              *float64               `db:"actual_fee" json:"actual_fee,omitempty"`
	Confirmations          int                    `db:"confirmations" json:"confirmations"`
	RequiredConfirmations  int                    `db:"required_confirmations" json:"required_confirmations"`
	Priority               int                    `db:"priority" json:"priority"`
	Metadata               JSONMap                `db:"metadata" json:"metadata,omitempty"`
	CreatedAt              time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt              time.Time              `db:"updated_at" json:"updated_at"`
	ExecutedAt             *time.Time             `db:"executed_at" json:"executed_at,omitempty"`
	CompletedAt            *time.Time             `db:"completed_at" json:"completed_at,omitempty"`
}

// EmergencyEvent represents an emergency event record
type EmergencyEvent struct {
	ID               string     `db:"id" json:"id"`
	Type             string     `db:"type" json:"type"`
	Severity         string     `db:"severity" json:"severity"`
	Description      string     `db:"description" json:"description"`
	TriggerData      JSONMap    `db:"trigger_data" json:"trigger_data,omitempty"`
	ResponseData     JSONMap    `db:"response_data" json:"response_data,omitempty"`
	Status           string     `db:"status" json:"status"`
	ResponseTimeMs   *int       `db:"response_time_ms" json:"response_time_ms,omitempty"`
	ActionsTaken     StringList `db:"actions_taken" json:"actions_taken,omitempty"`
	NotificationsSent int       `db:"notifications_sent" json:"notifications_sent"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	AcknowledgedAt   *time.Time `db:"acknowledged_at" json:"acknowledged_at,omitempty"`
	ResolvedAt       *time.Time `db:"resolved_at" json:"resolved_at,omitempty"`
	EscalatedAt      *time.Time `db:"escalated_at" json:"escalated_at,omitempty"`
}

// PositionSnapshot represents a position snapshot record
type PositionSnapshot struct {
	ID                int64     `db:"id" json:"id"`
	Timestamp         time.Time `db:"timestamp" json:"timestamp"`
	Symbol            string    `db:"symbol" json:"symbol"`
	Side              string    `db:"side" json:"side"`
	Size              float64   `db:"size" json:"size"`
	Notional          float64   `db:"notional" json:"notional"`
	EntryPrice        float64   `db:"entry_price" json:"entry_price"`
	MarkPrice         float64   `db:"mark_price" json:"mark_price"`
	UnrealizedPnL     float64   `db:"unrealized_pnl" json:"unrealized_pnl"`
	RealizedPnL       float64   `db:"realized_pnl" json:"realized_pnl"`
	Leverage          int       `db:"leverage" json:"leverage"`
	MarginType        *string   `db:"margin_type" json:"margin_type,omitempty"`
	IsolatedMargin    *float64  `db:"isolated_margin" json:"isolated_margin,omitempty"`
	MaintenanceMargin *float64  `db:"maintenance_margin" json:"maintenance_margin,omitempty"`
	LiquidationPrice  *float64  `db:"liquidation_price" json:"liquidation_price,omitempty"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
}

// FundStatusSnapshot represents a fund status snapshot record
type FundStatusSnapshot struct {
	ID                int64     `db:"id" json:"id"`
	Timestamp         time.Time `db:"timestamp" json:"timestamp"`
	TotalBalance      float64   `db:"total_balance" json:"total_balance"`
	AvailableBalance  float64   `db:"available_balance" json:"available_balance"`
	LockedBalance     float64   `db:"locked_balance" json:"locked_balance"`
	ProfitLoss        float64   `db:"profit_loss" json:"profit_loss"`
	DailyPL           float64   `db:"daily_pl" json:"daily_pl"`
	UnrealizedPL      float64   `db:"unrealized_pl" json:"unrealized_pl"`
	RealizedPL        float64   `db:"realized_pl" json:"realized_pl"`
	CurrentRisk       *float64  `db:"current_risk" json:"current_risk,omitempty"`
	MaxRisk           *float64  `db:"max_risk" json:"max_risk,omitempty"`
	VaR95             *float64  `db:"var_95" json:"var_95,omitempty"`
	ExpectedShortfall *float64  `db:"expected_shortfall" json:"expected_shortfall,omitempty"`
	TotalPositions    int       `db:"total_positions" json:"total_positions"`
	ActivePositions   int       `db:"active_positions" json:"active_positions"`
	LongPositions     int       `db:"long_positions" json:"long_positions"`
	ShortPositions    int       `db:"short_positions" json:"short_positions"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
}

// CircuitBreakerEvent represents a circuit breaker event record
type CircuitBreakerEvent struct {
	ID                    int64     `db:"id" json:"id"`
	TriggerReason         string    `db:"trigger_reason" json:"trigger_reason"`
	LossRatio             float64   `db:"loss_ratio" json:"loss_ratio"`
	TriggerCount          int       `db:"trigger_count" json:"trigger_count"`
	CooldownPeriodMinutes int       `db:"cooldown_period_minutes" json:"cooldown_period_minutes"`
	TriggeredAt           time.Time `db:"triggered_at" json:"triggered_at"`
	ResetAt               *time.Time `db:"reset_at" json:"reset_at,omitempty"`
	Status                string    `db:"status" json:"status"`
	Metadata              JSONMap   `db:"metadata" json:"metadata,omitempty"`
	CreatedAt             time.Time `db:"created_at" json:"created_at"`
}

// ProtectionMetrics represents protection system metrics
type ProtectionMetrics struct {
	ID                      int64     `db:"id" json:"id"`
	Timestamp               time.Time `db:"timestamp" json:"timestamp"`
	CircuitBreakerTriggered int64     `db:"circuit_breaker_triggered" json:"circuit_breaker_triggered"`
	EmergencyActivations    int64     `db:"emergency_activations" json:"emergency_activations"`
	AutoTransfers           int64     `db:"auto_transfers" json:"auto_transfers"`
	ManualInterventions     int64     `db:"manual_interventions" json:"manual_interventions"`
	LossesPrevented         float64   `db:"losses_prevented" json:"losses_prevented"`
	ProfitsSecured          float64   `db:"profits_secured" json:"profits_secured"`
	MaxLossAvoided          float64   `db:"max_loss_avoided" json:"max_loss_avoided"`
	AvgResponseTimeMs       int       `db:"avg_response_time_ms" json:"avg_response_time_ms"`
	ProtectionAccuracy      float64   `db:"protection_accuracy" json:"protection_accuracy"`
	FalsePositiveRate       float64   `db:"false_positive_rate" json:"false_positive_rate"`
	SystemUptimeSeconds     int64     `db:"system_uptime_seconds" json:"system_uptime_seconds"`
	LastEmergencyTest       *time.Time `db:"last_emergency_test" json:"last_emergency_test,omitempty"`
	CreatedAt               time.Time `db:"created_at" json:"created_at"`
}

// JSONMap is a custom type for handling JSON data in PostgreSQL
type JSONMap map[string]interface{}

// Value implements the driver.Valuer interface for JSONMap
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for JSONMap
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into JSONMap", value)
	}
	
	return json.Unmarshal(bytes, j)
}

// StringList is a custom type for handling string arrays in PostgreSQL
type StringList []string

// Value implements the driver.Valuer interface for StringList
func (s StringList) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	
	// PostgreSQL array format: {item1,item2,item3}
	if len(s) == 0 {
		return "{}", nil
	}
	
	result := "{"
	for i, item := range s {
		if i > 0 {
			result += ","
		}
		// Escape quotes and backslashes in the string
		escaped := fmt.Sprintf(`"%s"`, item)
		result += escaped
	}
	result += "}"
	
	return result, nil
}

// Scan implements the sql.Scanner interface for StringList
func (s *StringList) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	
	str, ok := value.(string)
	if !ok {
		bytes, ok := value.([]byte)
		if !ok {
			return fmt.Errorf("cannot scan %T into StringList", value)
		}
		str = string(bytes)
	}
	
	// Parse PostgreSQL array format
	if str == "{}" || str == "" {
		*s = []string{}
		return nil
	}
	
	// Simple parsing - assumes no complex escaping needed for our use case
	if len(str) >= 2 && str[0] == '{' && str[len(str)-1] == '}' {
		content := str[1 : len(str)-1]
		if content == "" {
			*s = []string{}
			return nil
		}
		
		// Split by comma and clean up quotes
		parts := []string{}
		current := ""
		inQuotes := false
		
		for i, char := range content {
			switch char {
			case '"':
				inQuotes = !inQuotes
			case ',':
				if !inQuotes {
					parts = append(parts, current)
					current = ""
				} else {
					current += string(char)
				}
			default:
				current += string(char)
			}
			
			// Add the last part
			if i == len(content)-1 {
				parts = append(parts, current)
			}
		}
		
		*s = parts
		return nil
	}
	
	return fmt.Errorf("invalid array format: %s", str)
}

// PortfolioStatistics represents calculated portfolio statistics
type PortfolioStatistics struct {
	AvgDailyReturn float64 `db:"avg_daily_return" json:"avg_daily_return"`
	Volatility     float64 `db:"volatility" json:"volatility"`
	MaxDrawdown    float64 `db:"max_drawdown" json:"max_drawdown"`
	SharpeRatio    float64 `db:"sharpe_ratio" json:"sharpe_ratio"`
	TotalReturn    float64 `db:"total_return" json:"total_return"`
	WinRate        float64 `db:"win_rate" json:"win_rate"`
}
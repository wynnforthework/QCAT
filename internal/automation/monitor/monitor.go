package monitor

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/strategy/state"
)

// Monitor manages performance monitoring and alerts
type Monitor struct {
	db           *sql.DB
	stateManager *state.Manager
	exchange     exchange.Exchange
	alerts       map[string]*Alert
	subscribers  map[string][]AlertCallback
	mu           sync.RWMutex
}

// Alert represents a performance alert
type Alert struct {
	ID         string                 `json:"id"`
	Strategy   string                 `json:"strategy"`
	Symbol     string                 `json:"symbol"`
	Type       AlertType              `json:"type"`
	Status     AlertStatus            `json:"status"`
	Condition  Condition              `json:"condition"`
	Value      float64                `json:"value"`
	Threshold  float64                `json:"threshold"`
	Message    string                 `json:"message"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	ResolvedAt time.Time              `json:"resolved_at"`
}

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypePnL        AlertType = "pnl"
	AlertTypeDrawdown   AlertType = "drawdown"
	AlertTypeVolatility AlertType = "volatility"
	AlertTypeExposure   AlertType = "exposure"
	AlertTypeMargin     AlertType = "margin"
	AlertTypeError      AlertType = "error"
)

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusActive   AlertStatus = "active"
	AlertStatusResolved AlertStatus = "resolved"
)

// Condition represents an alert condition
type Condition struct {
	Metric   string  `json:"metric"`
	Operator string  `json:"operator"` // ">", "<", ">=", "<=", "==", "!="
	Value    float64 `json:"value"`
	Duration string  `json:"duration,omitempty"` // e.g., "1h", "24h"
}

// AlertCallback represents an alert callback function
type AlertCallback func(*Alert)

// NewMonitor creates a new monitor
func NewMonitor(db *sql.DB, stateManager *state.Manager, exchange exchange.Exchange) *Monitor {
	return &Monitor{
		db:           db,
		stateManager: stateManager,
		exchange:     exchange,
		alerts:       make(map[string]*Alert),
		subscribers:  make(map[string][]AlertCallback),
	}
}

// Start starts the monitor
func (m *Monitor) Start(ctx context.Context) error {
	// Load existing alerts
	if err := m.loadAlerts(ctx); err != nil {
		return fmt.Errorf("failed to load alerts: %w", err)
	}

	// Start monitoring
	go m.monitor(ctx)

	return nil
}

// CreateAlert creates a new alert
func (m *Monitor) CreateAlert(ctx context.Context, strategy, symbol string, alertType AlertType, condition Condition) (*Alert, error) {
	alert := &Alert{
		ID:        fmt.Sprintf("%s-%s-%s-%d", strategy, symbol, alertType, time.Now().UnixNano()),
		Strategy:  strategy,
		Symbol:    symbol,
		Type:      alertType,
		Status:    AlertStatusActive,
		Condition: condition,
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store alert in database
	if err := m.saveAlert(ctx, alert); err != nil {
		return nil, fmt.Errorf("failed to save alert: %w", err)
	}

	// Store alert in memory
	m.mu.Lock()
	m.alerts[alert.ID] = alert
	m.mu.Unlock()

	return alert, nil
}

// GetAlert returns an alert by ID
func (m *Monitor) GetAlert(ctx context.Context, id string) (*Alert, error) {
	m.mu.RLock()
	alert, exists := m.alerts[id]
	m.mu.RUnlock()

	if exists {
		return alert, nil
	}

	// Load alert from database
	alert, err := m.loadAlert(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load alert: %w", err)
	}

	// Store alert in memory
	m.mu.Lock()
	m.alerts[alert.ID] = alert
	m.mu.Unlock()

	return alert, nil
}

// ListAlerts returns all alerts
func (m *Monitor) ListAlerts(ctx context.Context) ([]*Alert, error) {
	// Load alerts from database
	query := `
		SELECT id, strategy, symbol, type, status, condition, value, threshold,
			message, metadata, created_at, updated_at, resolved_at
		FROM alerts
	`

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query alerts: %w", err)
	}
	defer rows.Close()

	var alerts []*Alert
	for rows.Next() {
		var alert Alert
		var cond, meta []byte
		var resolvedAt sql.NullTime

		if err := rows.Scan(
			&alert.ID,
			&alert.Strategy,
			&alert.Symbol,
			&alert.Type,
			&alert.Status,
			&cond,
			&alert.Value,
			&alert.Threshold,
			&alert.Message,
			&meta,
			&alert.CreatedAt,
			&alert.UpdatedAt,
			&resolvedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}

		if err := json.Unmarshal(cond, &alert.Condition); err != nil {
			return nil, fmt.Errorf("failed to unmarshal condition: %w", err)
		}

		if err := json.Unmarshal(meta, &alert.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		if resolvedAt.Valid {
			alert.ResolvedAt = resolvedAt.Time
		}

		alerts = append(alerts, &alert)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating alerts: %w", err)
	}

	return alerts, nil
}

// Subscribe subscribes to alert updates
func (m *Monitor) Subscribe(alertID string, callback AlertCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.subscribers[alertID] = append(m.subscribers[alertID], callback)
}

// Unsubscribe removes an alert subscription
func (m *Monitor) Unsubscribe(alertID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.subscribers, alertID)
}

// saveAlert saves an alert to the database
func (m *Monitor) saveAlert(ctx context.Context, alert *Alert) error {
	cond, err := json.Marshal(alert.Condition)
	if err != nil {
		return fmt.Errorf("failed to marshal condition: %w", err)
	}

	meta, err := json.Marshal(alert.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO alerts (
			id, strategy, symbol, type, status, condition, value, threshold,
			message, metadata, created_at, updated_at, resolved_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		) ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			value = EXCLUDED.value,
			threshold = EXCLUDED.threshold,
			message = EXCLUDED.message,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at,
			resolved_at = EXCLUDED.resolved_at
	`

	_, err = m.db.ExecContext(ctx, query,
		alert.ID,
		alert.Strategy,
		alert.Symbol,
		alert.Type,
		alert.Status,
		cond,
		alert.Value,
		alert.Threshold,
		alert.Message,
		meta,
		alert.CreatedAt,
		alert.UpdatedAt,
		sql.NullTime{Time: alert.ResolvedAt, Valid: !alert.ResolvedAt.IsZero()},
	)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}

// loadAlert loads an alert from the database
func (m *Monitor) loadAlert(ctx context.Context, id string) (*Alert, error) {
	query := `
		SELECT id, strategy, symbol, type, status, condition, value, threshold,
			message, metadata, created_at, updated_at, resolved_at
		FROM alerts
		WHERE id = $1
	`

	var alert Alert
	var cond, meta []byte
	var resolvedAt sql.NullTime

	if err := m.db.QueryRowContext(ctx, query, id).Scan(
		&alert.ID,
		&alert.Strategy,
		&alert.Symbol,
		&alert.Type,
		&alert.Status,
		&cond,
		&alert.Value,
		&alert.Threshold,
		&alert.Message,
		&meta,
		&alert.CreatedAt,
		&alert.UpdatedAt,
		&resolvedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to scan alert: %w", err)
	}

	if err := json.Unmarshal(cond, &alert.Condition); err != nil {
		return nil, fmt.Errorf("failed to unmarshal condition: %w", err)
	}

	if err := json.Unmarshal(meta, &alert.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	if resolvedAt.Valid {
		alert.ResolvedAt = resolvedAt.Time
	}

	return &alert, nil
}

// loadAlerts loads alerts from the database
func (m *Monitor) loadAlerts(ctx context.Context) error {
	alerts, err := m.ListAlerts(ctx)
	if err != nil {
		return err
	}

	m.mu.Lock()
	for _, alert := range alerts {
		m.alerts[alert.ID] = alert
	}
	m.mu.Unlock()

	return nil
}

// monitor periodically checks alert conditions
func (m *Monitor) monitor(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkAlerts(ctx)
		}
	}
}

// checkAlerts checks all active alerts
func (m *Monitor) checkAlerts(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, alert := range m.alerts {
		if alert.Status != AlertStatusActive {
			continue
		}

		// Check alert condition
		value, err := m.getMetricValue(ctx, alert)
		if err != nil {
			log.Printf("Failed to get metric value for %s: %v", alert.ID, err)
			continue
		}

		alert.Value = value
		alert.UpdatedAt = time.Now()

		// Check if condition is met
		if m.checkCondition(alert.Value, alert.Condition) {
			alert.Status = AlertStatusActive
			alert.Message = fmt.Sprintf("%s alert triggered: %s %s %.2f (threshold: %.2f)",
				alert.Type, alert.Condition.Metric, alert.Condition.Operator, alert.Value, alert.Condition.Value)
		} else {
			alert.Status = AlertStatusResolved
			alert.ResolvedAt = time.Now()
			alert.Message = fmt.Sprintf("%s alert resolved: %s %s %.2f (threshold: %.2f)",
				alert.Type, alert.Condition.Metric, alert.Condition.Operator, alert.Value, alert.Condition.Value)
		}

		// Save alert
		if err := m.saveAlert(ctx, alert); err != nil {
			log.Printf("Failed to save alert %s: %v", alert.ID, err)
			continue
		}

		// Notify subscribers
		m.notifySubscribers(alert)
	}
}

// getMetricValue gets the current value for a metric
func (m *Monitor) getMetricValue(ctx context.Context, alert *Alert) (float64, error) {
	switch alert.Type {
	case AlertTypePnL:
		// Get position
		position, err := m.exchange.GetPosition(ctx, alert.Symbol)
		if err != nil {
			return 0, fmt.Errorf("failed to get position: %w", err)
		}
		if position == nil {
			return 0, nil
		}
		return position.UnrealizedPnL, nil

	case AlertTypeDrawdown:
		// Get position
		position, err := m.exchange.GetPosition(ctx, alert.Symbol)
		if err != nil {
			return 0, fmt.Errorf("failed to get position: %w", err)
		}
		if position == nil {
			return 0, nil
		}
		// Calculate drawdown
		if position.EntryPrice == 0 {
			return 0, nil
		}
		return (position.EntryPrice - position.UnrealizedPnL/position.Quantity) / position.EntryPrice * 100, nil

	case AlertTypeExposure:
		// Get position
		position, err := m.exchange.GetPosition(ctx, alert.Symbol)
		if err != nil {
			return 0, fmt.Errorf("failed to get position: %w", err)
		}
		if position == nil {
			return 0, nil
		}
		return position.Quantity * position.EntryPrice, nil

	case AlertTypeMargin:
		// Get account balance
		balances, err := m.exchange.GetAccountBalance(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to get account balance: %w", err)
		}
		// Find USDT balance
		balance, exists := balances["USDT"]
		if !exists {
			return 0, nil
		}
		return balance.Available, nil

	default:
		return 0, fmt.Errorf("unsupported metric type: %s", alert.Type)
	}
}

// checkCondition checks if a condition is met
func (m *Monitor) checkCondition(value float64, condition Condition) bool {
	switch condition.Operator {
	case ">":
		return value > condition.Value
	case "<":
		return value < condition.Value
	case ">=":
		return value >= condition.Value
	case "<=":
		return value <= condition.Value
	case "==":
		return value == condition.Value
	case "!=":
		return value != condition.Value
	default:
		return false
	}
}

// notifySubscribers notifies alert subscribers
func (m *Monitor) notifySubscribers(alert *Alert) {
	if callbacks, exists := m.subscribers[alert.ID]; exists {
		for _, callback := range callbacks {
			callback(alert)
		}
	}
}

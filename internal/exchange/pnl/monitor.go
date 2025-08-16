package pnl

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/exchange"
)

// Monitor monitors PnL and triggers automated actions
type Monitor struct {
	calculator *Calculator
	thresholds *RiskThresholds
	callbacks  []TriggerCallback
	mu         sync.RWMutex
	
	// State tracking
	lastCheck    time.Time
	alertHistory map[string]time.Time
	
	// Configuration
	checkInterval time.Duration
	cooldownPeriod time.Duration
}

// RiskThresholds defines risk management thresholds
type RiskThresholds struct {
	// Margin thresholds
	MaxMarginRatio     float64 `json:"max_margin_ratio"`     // Maximum margin usage ratio (e.g., 0.8 = 80%)
	WarningMarginRatio float64 `json:"warning_margin_ratio"` // Warning margin ratio (e.g., 0.7 = 70%)
	
	// PnL thresholds
	MaxDailyLoss       float64 `json:"max_daily_loss"`       // Maximum daily loss in USD
	MaxTotalLoss       float64 `json:"max_total_loss"`       // Maximum total loss in USD
	MaxDrawdownPercent float64 `json:"max_drawdown_percent"` // Maximum drawdown percentage
	
	// Position thresholds
	MaxPositionLoss    float64 `json:"max_position_loss"`    // Maximum loss per position in USD
	MaxPositionLossPercent float64 `json:"max_position_loss_percent"` // Maximum loss per position as percentage
	
	// Account thresholds
	MinAccountBalance  float64 `json:"min_account_balance"`  // Minimum account balance in USD
	MaxLeverage        int     `json:"max_leverage"`         // Maximum allowed leverage
}

// TriggerType represents the type of trigger
type TriggerType string

const (
	TriggerTypeMarginCall     TriggerType = "margin_call"
	TriggerTypeDailyLoss      TriggerType = "daily_loss"
	TriggerTypePositionLoss   TriggerType = "position_loss"
	TriggerTypeDrawdown       TriggerType = "drawdown"
	TriggerTypeAccountBalance TriggerType = "account_balance"
	TriggerTypeRebalance      TriggerType = "rebalance"
)

// TriggerAction represents the action to take
type TriggerAction string

const (
	ActionReducePosition TriggerAction = "reduce_position"
	ActionClosePosition  TriggerAction = "close_position"
	ActionReduceLeverage TriggerAction = "reduce_leverage"
	ActionStopTrading    TriggerAction = "stop_trading"
	ActionRebalance      TriggerAction = "rebalance"
	ActionAlert          TriggerAction = "alert"
)

// TriggerEvent represents a trigger event
type TriggerEvent struct {
	Type        TriggerType   `json:"type"`
	Action      TriggerAction `json:"action"`
	Symbol      string        `json:"symbol"`
	CurrentValue float64      `json:"current_value"`
	Threshold   float64       `json:"threshold"`
	Message     string        `json:"message"`
	Severity    string        `json:"severity"`
	Timestamp   time.Time     `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// TriggerCallback represents a callback function for trigger events
type TriggerCallback func(*TriggerEvent) error

// NewMonitor creates a new PnL monitor
func NewMonitor(calculator *Calculator, thresholds *RiskThresholds) *Monitor {
	return &Monitor{
		calculator:     calculator,
		thresholds:     thresholds,
		callbacks:      make([]TriggerCallback, 0),
		alertHistory:   make(map[string]time.Time),
		checkInterval:  time.Second * 5, // Check every 5 seconds
		cooldownPeriod: time.Minute * 5, // 5 minute cooldown between same alerts
	}
}

// AddCallback adds a trigger callback
func (m *Monitor) AddCallback(callback TriggerCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks = append(m.callbacks, callback)
}

// StartMonitoring starts the PnL monitoring process
func (m *Monitor) StartMonitoring(ctx context.Context) error {
	log.Println("Starting PnL monitoring...")
	
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			log.Println("PnL monitoring stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := m.checkTriggers(ctx); err != nil {
				log.Printf("Error checking triggers: %v", err)
			}
		}
	}
}

// checkTriggers checks all trigger conditions
func (m *Monitor) checkTriggers(ctx context.Context) error {
	m.lastCheck = time.Now()
	
	// Check margin ratio
	if err := m.checkMarginRatio(ctx); err != nil {
		log.Printf("Error checking margin ratio: %v", err)
	}
	
	// Check daily loss
	if err := m.checkDailyLoss(ctx); err != nil {
		log.Printf("Error checking daily loss: %v", err)
	}
	
	// Check position losses
	if err := m.checkPositionLosses(ctx); err != nil {
		log.Printf("Error checking position losses: %v", err)
	}
	
	// Check account balance
	if err := m.checkAccountBalance(ctx); err != nil {
		log.Printf("Error checking account balance: %v", err)
	}
	
	// Check drawdown
	if err := m.checkDrawdown(ctx); err != nil {
		log.Printf("Error checking drawdown: %v", err)
	}
	
	return nil
}

// checkMarginRatio checks margin ratio and triggers actions if needed
func (m *Monitor) checkMarginRatio(ctx context.Context) error {
	marginRatio := m.calculator.GetMarginRatio()
	
	if marginRatio > m.thresholds.MaxMarginRatio {
		event := &TriggerEvent{
			Type:         TriggerTypeMarginCall,
			Action:       ActionReducePosition,
			CurrentValue: marginRatio,
			Threshold:    m.thresholds.MaxMarginRatio,
			Message:      fmt.Sprintf("Margin ratio exceeded: %.2f%% > %.2f%%", marginRatio*100, m.thresholds.MaxMarginRatio*100),
			Severity:     "critical",
			Timestamp:    time.Now(),
			Metadata: map[string]interface{}{
				"total_unrealized_pnl": m.calculator.GetTotalUnrealizedPnL(),
				"total_realized_pnl":   m.calculator.GetTotalRealizedPnL(),
			},
		}
		
		return m.triggerEvent(event)
	} else if marginRatio > m.thresholds.WarningMarginRatio {
		event := &TriggerEvent{
			Type:         TriggerTypeMarginCall,
			Action:       ActionAlert,
			CurrentValue: marginRatio,
			Threshold:    m.thresholds.WarningMarginRatio,
			Message:      fmt.Sprintf("Margin ratio warning: %.2f%% > %.2f%%", marginRatio*100, m.thresholds.WarningMarginRatio*100),
			Severity:     "warning",
			Timestamp:    time.Now(),
		}
		
		return m.triggerEvent(event)
	}
	
	return nil
}

// checkDailyLoss checks daily loss limits
func (m *Monitor) checkDailyLoss(ctx context.Context) error {
	// Calculate daily PnL (simplified - would need to track daily start balance)
	totalPnL := m.calculator.GetTotalUnrealizedPnL() + m.calculator.GetTotalRealizedPnL()
	
	if totalPnL < -m.thresholds.MaxDailyLoss {
		event := &TriggerEvent{
			Type:         TriggerTypeDailyLoss,
			Action:       ActionStopTrading,
			CurrentValue: totalPnL,
			Threshold:    -m.thresholds.MaxDailyLoss,
			Message:      fmt.Sprintf("Daily loss limit exceeded: $%.2f < $%.2f", totalPnL, -m.thresholds.MaxDailyLoss),
			Severity:     "critical",
			Timestamp:    time.Now(),
		}
		
		return m.triggerEvent(event)
	}
	
	return nil
}

// checkPositionLosses checks individual position losses
func (m *Monitor) checkPositionLosses(ctx context.Context) error {
	snapshots, err := m.calculator.GetAllPnLSnapshots()
	if err != nil {
		return fmt.Errorf("failed to get PnL snapshots: %w", err)
	}
	
	for _, snapshot := range snapshots {
		// Check absolute loss
		if snapshot.UnrealizedPnL < -m.thresholds.MaxPositionLoss {
			event := &TriggerEvent{
				Type:         TriggerTypePositionLoss,
				Action:       ActionClosePosition,
				Symbol:       snapshot.Symbol,
				CurrentValue: snapshot.UnrealizedPnL,
				Threshold:    -m.thresholds.MaxPositionLoss,
				Message:      fmt.Sprintf("Position loss limit exceeded for %s: $%.2f < $%.2f", snapshot.Symbol, snapshot.UnrealizedPnL, -m.thresholds.MaxPositionLoss),
				Severity:     "high",
				Timestamp:    time.Now(),
				Metadata: map[string]interface{}{
					"position_size": snapshot.PositionSize,
					"entry_price":   snapshot.EntryPrice,
					"mark_price":    snapshot.MarkPrice,
				},
			}
			
			if err := m.triggerEvent(event); err != nil {
				log.Printf("Failed to trigger position loss event for %s: %v", snapshot.Symbol, err)
			}
		}
		
		// Check percentage loss
		if snapshot.EntryPrice > 0 && snapshot.PositionSize != 0 {
			lossPercent := snapshot.UnrealizedPnL / (abs(snapshot.PositionSize) * snapshot.EntryPrice)
			if lossPercent < -m.thresholds.MaxPositionLossPercent {
				event := &TriggerEvent{
					Type:         TriggerTypePositionLoss,
					Action:       ActionReducePosition,
					Symbol:       snapshot.Symbol,
					CurrentValue: lossPercent * 100,
					Threshold:    -m.thresholds.MaxPositionLossPercent * 100,
					Message:      fmt.Sprintf("Position loss percentage exceeded for %s: %.2f%% < %.2f%%", snapshot.Symbol, lossPercent*100, -m.thresholds.MaxPositionLossPercent*100),
					Severity:     "medium",
					Timestamp:    time.Now(),
				}
				
				if err := m.triggerEvent(event); err != nil {
					log.Printf("Failed to trigger position loss percentage event for %s: %v", snapshot.Symbol, err)
				}
			}
		}
	}
	
	return nil
}

// checkAccountBalance checks minimum account balance
func (m *Monitor) checkAccountBalance(ctx context.Context) error {
	// This would need to be implemented based on actual balance tracking
	// For now, we'll use a simplified approach
	totalPnL := m.calculator.GetTotalUnrealizedPnL() + m.calculator.GetTotalRealizedPnL()
	
	// Assume starting balance and calculate current balance
	// In a real implementation, this would track actual account balance
	estimatedBalance := 100000.0 + totalPnL // Assume $100k starting balance
	
	if estimatedBalance < m.thresholds.MinAccountBalance {
		event := &TriggerEvent{
			Type:         TriggerTypeAccountBalance,
			Action:       ActionStopTrading,
			CurrentValue: estimatedBalance,
			Threshold:    m.thresholds.MinAccountBalance,
			Message:      fmt.Sprintf("Account balance below minimum: $%.2f < $%.2f", estimatedBalance, m.thresholds.MinAccountBalance),
			Severity:     "critical",
			Timestamp:    time.Now(),
		}
		
		return m.triggerEvent(event)
	}
	
	return nil
}

// checkDrawdown checks maximum drawdown
func (m *Monitor) checkDrawdown(ctx context.Context) error {
	// This would need historical equity tracking to calculate proper drawdown
	// For now, we'll use a simplified approach based on total PnL
	totalPnL := m.calculator.GetTotalUnrealizedPnL() + m.calculator.GetTotalRealizedPnL()
	
	// Simplified drawdown calculation (would need peak equity tracking)
	if totalPnL < 0 {
		drawdownPercent := abs(totalPnL) / 100000.0 // Assume $100k starting balance
		
		if drawdownPercent > m.thresholds.MaxDrawdownPercent {
			event := &TriggerEvent{
				Type:         TriggerTypeDrawdown,
				Action:       ActionReducePosition,
				CurrentValue: drawdownPercent * 100,
				Threshold:    m.thresholds.MaxDrawdownPercent * 100,
				Message:      fmt.Sprintf("Drawdown limit exceeded: %.2f%% > %.2f%%", drawdownPercent*100, m.thresholds.MaxDrawdownPercent*100),
				Severity:     "high",
				Timestamp:    time.Now(),
			}
			
			return m.triggerEvent(event)
		}
	}
	
	return nil
}

// triggerEvent triggers a risk management event
func (m *Monitor) triggerEvent(event *TriggerEvent) error {
	// Check cooldown period to avoid spam
	alertKey := fmt.Sprintf("%s:%s:%s", event.Type, event.Action, event.Symbol)
	
	m.mu.Lock()
	lastAlert, exists := m.alertHistory[alertKey]
	if exists && time.Since(lastAlert) < m.cooldownPeriod {
		m.mu.Unlock()
		return nil // Skip this alert due to cooldown
	}
	m.alertHistory[alertKey] = time.Now()
	m.mu.Unlock()
	
	log.Printf("Triggering risk event: %s - %s", event.Type, event.Message)
	
	// Execute all callbacks
	for _, callback := range m.callbacks {
		if err := callback(event); err != nil {
			log.Printf("Callback error for event %s: %v", event.Type, err)
		}
	}
	
	return nil
}

// GetCurrentStatus returns current monitoring status
func (m *Monitor) GetCurrentStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return map[string]interface{}{
		"last_check":        m.lastCheck,
		"margin_ratio":      m.calculator.GetMarginRatio(),
		"total_unrealized":  m.calculator.GetTotalUnrealizedPnL(),
		"total_realized":    m.calculator.GetTotalRealizedPnL(),
		"alert_count":       len(m.alertHistory),
		"thresholds":        m.thresholds,
	}
}

// UpdateThresholds updates risk thresholds
func (m *Monitor) UpdateThresholds(thresholds *RiskThresholds) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.thresholds = thresholds
}

// GetThresholds returns current risk thresholds
func (m *Monitor) GetThresholds() *RiskThresholds {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy to prevent external modification
	return &RiskThresholds{
		MaxMarginRatio:         m.thresholds.MaxMarginRatio,
		WarningMarginRatio:     m.thresholds.WarningMarginRatio,
		MaxDailyLoss:           m.thresholds.MaxDailyLoss,
		MaxTotalLoss:           m.thresholds.MaxTotalLoss,
		MaxDrawdownPercent:     m.thresholds.MaxDrawdownPercent,
		MaxPositionLoss:        m.thresholds.MaxPositionLoss,
		MaxPositionLossPercent: m.thresholds.MaxPositionLossPercent,
		MinAccountBalance:      m.thresholds.MinAccountBalance,
		MaxLeverage:            m.thresholds.MaxLeverage,
	}
}

// ClearAlertHistory clears the alert history (useful for testing)
func (m *Monitor) ClearAlertHistory() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alertHistory = make(map[string]time.Time)
}
package position

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/exchange/pnl"
	"qcat/internal/config"
	"qcat/internal/monitor"
)

// AutomationManager handles automated position management
type AutomationManager struct {
	exchange       exchange.Exchange
	pnlCalculator  *pnl.Calculator
	config         *AutomationConfig
	alertManager   *monitor.AlertManager
	mu             sync.RWMutex
	
	// State tracking
	positions      map[string]*PositionState
	lastCheck      time.Time
	running        bool
	stopCh         chan struct{}
}

// AutomationConfig represents automation configuration
type AutomationConfig struct {
	CheckInterval          time.Duration `yaml:"check_interval"`
	MarginThreshold        float64       `yaml:"margin_threshold"`
	StopLossThreshold      float64       `yaml:"stop_loss_threshold"`
	ProfitTakingThreshold  float64       `yaml:"profit_taking_threshold"`
	PositionSizeThreshold  float64       `yaml:"position_size_threshold"`
	EmergencyCloseThreshold float64      `yaml:"emergency_close_threshold"`
	
	// Risk-based reduction
	RiskBasedReduction   RiskReductionConfig `yaml:"risk_based_reduction"`
	
	// Balance-driven actions
	BalanceDriven        BalanceDrivenConfig `yaml:"balance_driven"`
}

// RiskReductionConfig represents risk-based position reduction configuration
type RiskReductionConfig struct {
	Enabled            bool    `yaml:"enabled"`
	MaxDrawdownPercent float64 `yaml:"max_drawdown_percent"`
	ReductionPercent   float64 `yaml:"reduction_percent"`
	MinPositionSize    float64 `yaml:"min_position_size"`
}

// BalanceDrivenConfig represents balance-driven automation configuration
type BalanceDrivenConfig struct {
	Enabled                bool    `yaml:"enabled"`
	EquityChangeThreshold  float64 `yaml:"equity_change_threshold"`
	MarginUtilizationMax   float64 `yaml:"margin_utilization_max"`
	AutoRebalanceEnabled   bool    `yaml:"auto_rebalance_enabled"`
	RebalanceThreshold     float64 `yaml:"rebalance_threshold"`
}

// PositionState tracks automation state for a position
type PositionState struct {
	Symbol            string
	LastUnrealizedPnL float64
	LastEquityCheck   time.Time
	ActionCount       int
	LastAction        string
	LastActionTime    time.Time
}

// AutomationAction represents an automated action
type AutomationAction struct {
	Type        ActionType            `json:"type"`
	Symbol      string                `json:"symbol"`
	Action      string                `json:"action"` // REDUCE, CLOSE, REBALANCE
	Reason      string                `json:"reason"`
	Parameters  map[string]interface{} `json:"parameters"`
	Timestamp   time.Time             `json:"timestamp"`
	Status      string                `json:"status"`
	Result      string                `json:"result,omitempty"`
}

// ActionType represents the type of automation action
type ActionType string

const (
	ActionTypeStopLoss        ActionType = "stop_loss"
	ActionTypeProfitTaking    ActionType = "profit_taking"
	ActionTypeRiskReduction   ActionType = "risk_reduction"
	ActionTypeEmergencyClose  ActionType = "emergency_close"
	ActionTypeRebalance       ActionType = "rebalance"
	ActionTypeMarginCall      ActionType = "margin_call"
)

// NewAutomationManager creates a new automation manager
func NewAutomationManager(ex exchange.Exchange, calc *pnl.Calculator, alertMgr *monitor.AlertManager) *AutomationManager {
	config := getDefaultAutomationConfig()
	
	return &AutomationManager{
		exchange:      ex,
		pnlCalculator: calc,
		config:        config,
		alertManager:  alertMgr,
		positions:     make(map[string]*PositionState),
		stopCh:        make(chan struct{}),
	}
}

// Start starts the automation manager
func (am *AutomationManager) Start() error {
	am.mu.Lock()
	if am.running {
		am.mu.Unlock()
		return fmt.Errorf("automation manager already running")
	}
	am.running = true
	am.mu.Unlock()
	
	log.Println("Starting automated position management...")
	
	// Start monitoring loop
	go am.monitoringLoop()
	
	return nil
}

// Stop stops the automation manager
func (am *AutomationManager) Stop() error {
	am.mu.Lock()
	if !am.running {
		am.mu.Unlock()
		return nil
	}
	am.running = false
	am.mu.Unlock()
	
	close(am.stopCh)
	log.Println("Stopped automated position management")
	return nil
}

// monitoringLoop runs the main monitoring loop
func (am *AutomationManager) monitoringLoop() {
	ticker := time.NewTicker(am.config.CheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-am.stopCh:
			return
		case <-ticker.C:
			if err := am.performAutomationCheck(); err != nil {
				log.Printf("Automation check failed: %v", err)
			}
		}
	}
}

// performAutomationCheck performs a complete automation check
func (am *AutomationManager) performAutomationCheck() error {
	ctx := context.Background()
	
	// 1. Get current positions
	positions, err := am.exchange.GetPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}
	
	// 2. Get account balance
	balance, err := am.exchange.GetAccountBalance(ctx)
	if err != nil {
		return fmt.Errorf("failed to get account balance: %w", err)
	}
	
	// 3. Check each position for automation triggers
	for _, position := range positions {
		if err := am.checkPositionAutomation(ctx, position, balance); err != nil {
			log.Printf("Position automation check failed for %s: %v", position.Symbol, err)
		}
	}
	
	// 4. Check for portfolio-level automation
	if err := am.checkPortfolioAutomation(ctx, positions, balance); err != nil {
		log.Printf("Portfolio automation check failed: %v", err)
	}
	
	am.lastCheck = time.Now()
	return nil
}

// checkPositionAutomation checks automation triggers for a single position
func (am *AutomationManager) checkPositionAutomation(ctx context.Context, position *exchange.Position, balance *exchange.AccountBalance) error {
	// Update position state tracking
	state := am.getOrCreatePositionState(position.Symbol)
	
	// 1. Check for stop loss triggers
	if err := am.checkStopLoss(ctx, position, state); err != nil {
		return fmt.Errorf("stop loss check failed: %w", err)
	}
	
	// 2. Check for profit taking triggers
	if err := am.checkProfitTaking(ctx, position, state); err != nil {
		return fmt.Errorf("profit taking check failed: %w", err)
	}
	
	// 3. Check for risk-based reduction
	if am.config.RiskBasedReduction.Enabled {
		if err := am.checkRiskBasedReduction(ctx, position, state, balance); err != nil {
			return fmt.Errorf("risk-based reduction check failed: %w", err)
		}
	}
	
	// 4. Check for margin calls
	if err := am.checkMarginRequirements(ctx, position, balance); err != nil {
		return fmt.Errorf("margin check failed: %w", err)
	}
	
	// Update state
	state.LastUnrealizedPnL = position.UnrealizedPnL
	state.LastEquityCheck = time.Now()
	
	return nil
}

// checkStopLoss checks if stop loss should be triggered
func (am *AutomationManager) checkStopLoss(ctx context.Context, position *exchange.Position, state *PositionState) error {
	// Calculate loss percentage
	if position.EntryPrice == 0 {
		return nil
	}
	
	lossPercent := position.UnrealizedPnL / (position.Size * position.EntryPrice)
	
	if lossPercent < -am.config.StopLossThreshold {
		action := &AutomationAction{
			Type:      ActionTypeStopLoss,
			Symbol:    position.Symbol,
			Action:    "CLOSE",
			Reason:    fmt.Sprintf("Stop loss triggered: %.2f%% loss", lossPercent*100),
			Timestamp: time.Now(),
			Status:    "pending",
			Parameters: map[string]interface{}{
				"loss_percent": lossPercent,
				"threshold":    am.config.StopLossThreshold,
			},
		}
		
		return am.executeAutomationAction(ctx, action, position)
	}
	
	return nil
}

// checkProfitTaking checks if profit taking should be triggered
func (am *AutomationManager) checkProfitTaking(ctx context.Context, position *exchange.Position, state *PositionState) error {
	if position.EntryPrice == 0 {
		return nil
	}
	
	profitPercent := position.UnrealizedPnL / (position.Size * position.EntryPrice)
	
	if profitPercent > am.config.ProfitTakingThreshold {
		action := &AutomationAction{
			Type:      ActionTypeProfitTaking,
			Symbol:    position.Symbol,
			Action:    "REDUCE",
			Reason:    fmt.Sprintf("Profit taking triggered: %.2f%% profit", profitPercent*100),
			Timestamp: time.Now(),
			Status:    "pending",
			Parameters: map[string]interface{}{
				"profit_percent":    profitPercent,
				"threshold":         am.config.ProfitTakingThreshold,
				"reduction_percent": 0.5, // Take 50% profit
			},
		}
		
		return am.executeAutomationAction(ctx, action, position)
	}
	
	return nil
}

// checkRiskBasedReduction checks for risk-based position reduction
func (am *AutomationManager) checkRiskBasedReduction(ctx context.Context, position *exchange.Position, state *PositionState, balance *exchange.AccountBalance) error {
	config := am.config.RiskBasedReduction
	
	// Calculate account drawdown
	totalEquity := balance.TotalEquity
	if totalEquity == 0 {
		return nil
	}
	
	// Estimate peak equity (simplified)
	peakEquity := totalEquity + balance.TotalUnrealizedPnL // Assume current is peak if all PnL was positive
	drawdown := (peakEquity - totalEquity) / peakEquity
	
	if drawdown > config.MaxDrawdownPercent {
		// Check if position size is significant enough to reduce
		positionValue := position.Size * position.MarkPrice
		if positionValue > config.MinPositionSize {
			action := &AutomationAction{
				Type:      ActionTypeRiskReduction,
				Symbol:    position.Symbol,
				Action:    "REDUCE",
				Reason:    fmt.Sprintf("Risk-based reduction: %.2f%% drawdown", drawdown*100),
				Timestamp: time.Now(),
				Status:    "pending",
				Parameters: map[string]interface{}{
					"drawdown_percent":  drawdown,
					"max_drawdown":      config.MaxDrawdownPercent,
					"reduction_percent": config.ReductionPercent,
				},
			}
			
			return am.executeAutomationAction(ctx, action, position)
		}
	}
	
	return nil
}

// checkMarginRequirements checks margin requirements and triggers margin calls if needed
func (am *AutomationManager) checkMarginRequirements(ctx context.Context, position *exchange.Position, balance *exchange.AccountBalance) error {
	marginRatio := position.MarginUsed / balance.AvailableBalance
	
	if marginRatio > am.config.MarginThreshold {
		action := &AutomationAction{
			Type:      ActionTypeMarginCall,
			Symbol:    position.Symbol,
			Action:    "REDUCE",
			Reason:    fmt.Sprintf("Margin call: %.2f%% margin utilization", marginRatio*100),
			Timestamp: time.Now(),
			Status:    "pending",
			Parameters: map[string]interface{}{
				"margin_ratio": marginRatio,
				"threshold":    am.config.MarginThreshold,
				"reduction_amount": position.Size * 0.3, // Reduce by 30%
			},
		}
		
		return am.executeAutomationAction(ctx, action, position)
	}
	
	return nil
}

// checkPortfolioAutomation checks portfolio-level automation triggers
func (am *AutomationManager) checkPortfolioAutomation(ctx context.Context, positions []*exchange.Position, balance *exchange.AccountBalance) error {
	if !am.config.BalanceDriven.Enabled {
		return nil
	}
	
	config := am.config.BalanceDriven
	
	// Check total margin utilization
	totalMarginUsed := 0.0
	for _, pos := range positions {
		totalMarginUsed += pos.MarginUsed
	}
	
	marginUtilization := totalMarginUsed / balance.AvailableBalance
	
	if marginUtilization > config.MarginUtilizationMax {
		// Trigger portfolio rebalancing
		action := &AutomationAction{
			Type:      ActionTypeRebalance,
			Symbol:    "PORTFOLIO",
			Action:    "REBALANCE",
			Reason:    fmt.Sprintf("High margin utilization: %.2f%%", marginUtilization*100),
			Timestamp: time.Now(),
			Status:    "pending",
			Parameters: map[string]interface{}{
				"margin_utilization": marginUtilization,
				"max_utilization":    config.MarginUtilizationMax,
				"target_reduction":   0.2, // Reduce overall exposure by 20%
			},
		}
		
		return am.executePortfolioAction(ctx, action, positions)
	}
	
	return nil
}

// executeAutomationAction executes an automation action for a single position
func (am *AutomationManager) executeAutomationAction(ctx context.Context, action *AutomationAction, position *exchange.Position) error {
	log.Printf("Executing automation action: %s for %s - %s", action.Type, action.Symbol, action.Reason)
	
	// Send alert
	if am.alertManager != nil {
		alert := &monitor.Alert{
			ID:        fmt.Sprintf("auto-%d", time.Now().UnixNano()),
			Type:      string(action.Type),
			Message:   action.Reason,
			Severity:  monitor.SeverityWarning,
			Timestamp: action.Timestamp,
			Data: map[string]interface{}{
				"symbol":     action.Symbol,
				"action":     action.Action,
				"parameters": action.Parameters,
			},
		}
		am.alertManager.SendAlert(alert)
	}
	
	switch action.Action {
	case "CLOSE":
		return am.closePosition(ctx, position)
	case "REDUCE":
		reductionPercent := 0.5 // Default 50% reduction
		if val, ok := action.Parameters["reduction_percent"].(float64); ok {
			reductionPercent = val
		}
		return am.reducePosition(ctx, position, reductionPercent)
	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}
}

// executePortfolioAction executes a portfolio-level automation action
func (am *AutomationManager) executePortfolioAction(ctx context.Context, action *AutomationAction, positions []*exchange.Position) error {
	log.Printf("Executing portfolio automation action: %s - %s", action.Type, action.Reason)
	
	switch action.Action {
	case "REBALANCE":
		targetReduction := 0.2 // Default 20% reduction
		if val, ok := action.Parameters["target_reduction"].(float64); ok {
			targetReduction = val
		}
		return am.rebalancePortfolio(ctx, positions, targetReduction)
	default:
		return fmt.Errorf("unknown portfolio action: %s", action.Action)
	}
}

// closePosition closes a position completely
func (am *AutomationManager) closePosition(ctx context.Context, position *exchange.Position) error {
	side := "SELL"
	if position.Side == "SHORT" {
		side = "BUY"
	}
	
	order := &exchange.OrderRequest{
		Symbol:   position.Symbol,
		Side:     exchange.OrderSide(side),
		Type:     exchange.OrderTypeMarket,
		Quantity: position.Size,
	}
	
	_, err := am.exchange.PlaceOrder(ctx, order)
	if err != nil {
		return fmt.Errorf("failed to close position: %w", err)
	}
	
	log.Printf("Successfully closed position for %s", position.Symbol)
	return nil
}

// reducePosition reduces a position by a percentage
func (am *AutomationManager) reducePosition(ctx context.Context, position *exchange.Position, reductionPercent float64) error {
	side := "SELL"
	if position.Side == "SHORT" {
		side = "BUY"
	}
	
	reduceSize := position.Size * reductionPercent
	
	order := &exchange.OrderRequest{
		Symbol:   position.Symbol,
		Side:     exchange.OrderSide(side),
		Type:     exchange.OrderTypeMarket,
		Quantity: reduceSize,
	}
	
	_, err := am.exchange.PlaceOrder(ctx, order)
	if err != nil {
		return fmt.Errorf("failed to reduce position: %w", err)
	}
	
	log.Printf("Successfully reduced position for %s by %.1f%%", position.Symbol, reductionPercent*100)
	return nil
}

// rebalancePortfolio rebalances the entire portfolio
func (am *AutomationManager) rebalancePortfolio(ctx context.Context, positions []*exchange.Position, targetReduction float64) error {
	// Sort positions by unrealized PnL (worst performing first)
	// This is a simplified rebalancing - in practice, you'd want more sophisticated logic
	
	for _, position := range positions {
		if position.UnrealizedPnL < 0 {
			// Reduce losing positions more aggressively
			if err := am.reducePosition(ctx, position, targetReduction*1.5); err != nil {
				log.Printf("Failed to reduce position %s during rebalancing: %v", position.Symbol, err)
			}
		} else {
			// Reduce winning positions less aggressively
			if err := am.reducePosition(ctx, position, targetReduction*0.5); err != nil {
				log.Printf("Failed to reduce position %s during rebalancing: %v", position.Symbol, err)
			}
		}
	}
	
	log.Printf("Portfolio rebalancing completed with target reduction of %.1f%%", targetReduction*100)
	return nil
}

// getOrCreatePositionState gets or creates position state
func (am *AutomationManager) getOrCreatePositionState(symbol string) *PositionState {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	if state, exists := am.positions[symbol]; exists {
		return state
	}
	
	state := &PositionState{
		Symbol: symbol,
	}
	am.positions[symbol] = state
	return state
}

// getDefaultAutomationConfig returns default automation configuration
func getDefaultAutomationConfig() *AutomationConfig {
	return &AutomationConfig{
		CheckInterval:           30 * time.Second,
		MarginThreshold:         0.8,  // 80% margin utilization
		StopLossThreshold:       0.05, // 5% loss
		ProfitTakingThreshold:   0.15, // 15% profit
		PositionSizeThreshold:   1000, // $1000 minimum
		EmergencyCloseThreshold: 0.1,  // 10% emergency threshold
		
		RiskBasedReduction: RiskReductionConfig{
			Enabled:            true,
			MaxDrawdownPercent: 0.08, // 8% max drawdown
			ReductionPercent:   0.3,  // 30% reduction
			MinPositionSize:    500,  // $500 minimum
		},
		
		BalanceDriven: BalanceDrivenConfig{
			Enabled:                true,
			EquityChangeThreshold:  0.05, // 5% equity change
			MarginUtilizationMax:   0.7,  // 70% max margin utilization
			AutoRebalanceEnabled:   true,
			RebalanceThreshold:     0.1,  // 10% rebalance threshold
		},
	}
}

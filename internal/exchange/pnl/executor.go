package pnl

import (
	"context"
	"fmt"
	"log"
	"time"

	"qcat/internal/exchange"
)

// Executor executes automated trading actions based on PnL triggers
type Executor struct {
	exchange     exchange.Exchange
	positionMgr  *exchange.PositionManager
	orderMgr     *exchange.OrderManager
	
	// Configuration
	enabled      bool
	dryRun       bool
	maxReduction float64 // Maximum position reduction per action (e.g., 0.5 = 50%)
}

// NewExecutor creates a new PnL executor
func NewExecutor(ex exchange.Exchange, posMgr *exchange.PositionManager, orderMgr *exchange.OrderManager) *Executor {
	return &Executor{
		exchange:     ex,
		positionMgr:  posMgr,
		orderMgr:     orderMgr,
		enabled:      true,
		dryRun:       false,
		maxReduction: 0.5, // 50% max reduction per action
	}
}

// SetEnabled enables or disables the executor
func (e *Executor) SetEnabled(enabled bool) {
	e.enabled = enabled
	log.Printf("PnL Executor enabled: %v", enabled)
}

// SetDryRun enables or disables dry run mode
func (e *Executor) SetDryRun(dryRun bool) {
	e.dryRun = dryRun
	log.Printf("PnL Executor dry run mode: %v", dryRun)
}

// HandleTriggerEvent handles a trigger event from the monitor
func (e *Executor) HandleTriggerEvent(event *TriggerEvent) error {
	if !e.enabled {
		log.Printf("Executor disabled, ignoring event: %s", event.Type)
		return nil
	}
	
	log.Printf("Handling trigger event: %s - %s", event.Type, event.Message)
	
	switch event.Action {
	case ActionReducePosition:
		return e.reducePosition(event)
	case ActionClosePosition:
		return e.closePosition(event)
	case ActionReduceLeverage:
		return e.reduceLeverage(event)
	case ActionStopTrading:
		return e.stopTrading(event)
	case ActionRebalance:
		return e.rebalance(event)
	case ActionAlert:
		return e.sendAlert(event)
	default:
		return fmt.Errorf("unknown action: %s", event.Action)
	}
}

// reducePosition reduces a position by a specified percentage
func (e *Executor) reducePosition(event *TriggerEvent) error {
	if event.Symbol == "" {
		return e.reduceAllPositions(event)
	}
	
	// Get current position
	position, err := e.positionMgr.GetPosition(context.Background(), event.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get position for %s: %w", event.Symbol, err)
	}
	
	if position.Size == 0 {
		log.Printf("No position to reduce for %s", event.Symbol)
		return nil
	}
	
	// Calculate reduction amount
	reductionAmount := abs(position.Size) * e.maxReduction
	
	// Determine order side (opposite of position)
	var side exchange.OrderSide
	if position.Size > 0 {
		side = exchange.OrderSideSell
	} else {
		side = exchange.OrderSideBuy
	}
	
	log.Printf("Reducing position for %s: %.6f %s (%.1f%% reduction)", 
		event.Symbol, reductionAmount, side, e.maxReduction*100)
	
	if e.dryRun {
		log.Printf("DRY RUN: Would reduce position %s by %.6f", event.Symbol, reductionAmount)
		return nil
	}
	
	// Create market order to reduce position
	orderReq := &exchange.OrderRequest{
		Symbol:    event.Symbol,
		Side:      string(side),
		Type:      string(exchange.OrderTypeMarket),
		Quantity:  reductionAmount,
		ReduceOnly: true,
	}
	
	orderResp, err := e.exchange.PlaceOrder(context.Background(), orderReq)
	if err != nil {
		return fmt.Errorf("failed to place reduction order for %s: %w", event.Symbol, err)
	}
	
	log.Printf("Position reduction order placed for %s: %s", event.Symbol, orderResp.OrderID)
	return nil
}

// closePosition closes a position completely
func (e *Executor) closePosition(event *TriggerEvent) error {
	if event.Symbol == "" {
		return fmt.Errorf("symbol required for close position action")
	}
	
	// Get current position
	position, err := e.positionMgr.GetPosition(context.Background(), event.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get position for %s: %w", event.Symbol, err)
	}
	
	if position.Size == 0 {
		log.Printf("No position to close for %s", event.Symbol)
		return nil
	}
	
	// Determine order side (opposite of position)
	var side exchange.OrderSide
	closeAmount := abs(position.Size)
	
	if position.Size > 0 {
		side = exchange.OrderSideSell
	} else {
		side = exchange.OrderSideBuy
	}
	
	log.Printf("Closing position for %s: %.6f %s", event.Symbol, closeAmount, side)
	
	if e.dryRun {
		log.Printf("DRY RUN: Would close position %s (%.6f)", event.Symbol, closeAmount)
		return nil
	}
	
	// Create market order to close position
	orderReq := &exchange.OrderRequest{
		Symbol:    event.Symbol,
		Side:      string(side),
		Type:      string(exchange.OrderTypeMarket),
		Quantity:  closeAmount,
		ReduceOnly: true,
	}
	
	orderResp, err := e.exchange.PlaceOrder(context.Background(), orderReq)
	if err != nil {
		return fmt.Errorf("failed to place close order for %s: %w", event.Symbol, err)
	}
	
	log.Printf("Position close order placed for %s: %s", event.Symbol, orderResp.OrderID)
	return nil
}

// reduceAllPositions reduces all positions proportionally
func (e *Executor) reduceAllPositions(event *TriggerEvent) error {
	// Get all positions
	positions, err := e.positionMgr.GetAllPositions(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get all positions: %w", err)
	}
	
	log.Printf("Reducing all positions by %.1f%%", e.maxReduction*100)
	
	for _, position := range positions {
		if position.Size == 0 {
			continue
		}
		
		// Create event for each position
		positionEvent := &TriggerEvent{
			Type:      event.Type,
			Action:    ActionReducePosition,
			Symbol:    position.Symbol,
			Severity:  event.Severity,
			Timestamp: event.Timestamp,
		}
		
		if err := e.reducePosition(positionEvent); err != nil {
			log.Printf("Failed to reduce position %s: %v", position.Symbol, err)
		}
	}
	
	return nil
}

// reduceLeverage reduces leverage for a position
func (e *Executor) reduceLeverage(event *TriggerEvent) error {
	if event.Symbol == "" {
		return fmt.Errorf("symbol required for reduce leverage action")
	}
	
	// Get current position
	position, err := e.positionMgr.GetPosition(context.Background(), event.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get position for %s: %w", event.Symbol, err)
	}
	
	if position.Size == 0 {
		log.Printf("No position for leverage reduction: %s", event.Symbol)
		return nil
	}
	
	// Calculate new leverage (reduce by 50%)
	newLeverage := int(float64(position.Leverage) * 0.5)
	if newLeverage < 1 {
		newLeverage = 1
	}
	
	log.Printf("Reducing leverage for %s from %dx to %dx", event.Symbol, position.Leverage, newLeverage)
	
	if e.dryRun {
		log.Printf("DRY RUN: Would reduce leverage for %s to %dx", event.Symbol, newLeverage)
		return nil
	}
	
	// Set new leverage
	if err := e.exchange.SetLeverage(context.Background(), event.Symbol, newLeverage); err != nil {
		return fmt.Errorf("failed to set leverage for %s: %w", event.Symbol, err)
	}
	
	log.Printf("Leverage reduced for %s to %dx", event.Symbol, newLeverage)
	return nil
}

// stopTrading stops all trading activities
func (e *Executor) stopTrading(event *TriggerEvent) error {
	log.Printf("EMERGENCY: Stopping all trading due to: %s", event.Message)
	
	if e.dryRun {
		log.Printf("DRY RUN: Would stop all trading")
		return nil
	}
	
	// Cancel all open orders
	if err := e.cancelAllOrders(); err != nil {
		log.Printf("Failed to cancel all orders: %v", err)
	}
	
	// Close all positions (optional - might want to keep positions but stop new trades)
	if event.Severity == "critical" {
		if err := e.closeAllPositions(); err != nil {
			log.Printf("Failed to close all positions: %v", err)
		}
	}
	
	// Disable the executor to prevent further actions
	e.SetEnabled(false)
	
	return nil
}

// rebalance triggers portfolio rebalancing
func (e *Executor) rebalance(event *TriggerEvent) error {
	log.Printf("Triggering portfolio rebalance due to: %s", event.Message)
	
	if e.dryRun {
		log.Printf("DRY RUN: Would trigger portfolio rebalance")
		return nil
	}
	
	// This would integrate with the portfolio manager
	// For now, just log the action
	log.Printf("Portfolio rebalance triggered - would call portfolio manager")
	
	return nil
}

// sendAlert sends an alert notification
func (e *Executor) sendAlert(event *TriggerEvent) error {
	log.Printf("ALERT [%s]: %s", event.Severity, event.Message)
	
	// This would integrate with alerting system (email, Slack, etc.)
	// For now, just log the alert
	
	return nil
}

// cancelAllOrders cancels all open orders
func (e *Executor) cancelAllOrders() error {
	// Get all open orders
	orders, err := e.orderMgr.GetOpenOrders(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get open orders: %w", err)
	}
	
	log.Printf("Cancelling %d open orders", len(orders))
	
	for _, order := range orders {
		cancelReq := &exchange.OrderCancelRequest{
			Symbol:  order.Symbol,
			OrderID: order.OrderID,
		}
		
		if err := e.exchange.CancelOrder(context.Background(), cancelReq); err != nil {
			log.Printf("Failed to cancel order %s: %v", order.OrderID, err)
		} else {
			log.Printf("Cancelled order %s for %s", order.OrderID, order.Symbol)
		}
	}
	
	return nil
}

// closeAllPositions closes all open positions
func (e *Executor) closeAllPositions() error {
	// Get all positions
	positions, err := e.positionMgr.GetAllPositions(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get all positions: %w", err)
	}
	
	log.Printf("Closing %d positions", len(positions))
	
	for _, position := range positions {
		if position.Size == 0 {
			continue
		}
		
		event := &TriggerEvent{
			Type:      TriggerTypeAccountBalance,
			Action:    ActionClosePosition,
			Symbol:    position.Symbol,
			Severity:  "critical",
			Timestamp: time.Now(),
		}
		
		if err := e.closePosition(event); err != nil {
			log.Printf("Failed to close position %s: %v", position.Symbol, err)
		}
	}
	
	return nil
}

// GetStatus returns executor status
func (e *Executor) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"enabled":       e.enabled,
		"dry_run":       e.dryRun,
		"max_reduction": e.maxReduction,
	}
}
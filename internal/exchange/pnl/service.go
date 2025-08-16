package pnl

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/market"
)

// Service provides a complete PnL monitoring and risk management service
type Service struct {
	calculator *Calculator
	monitor    *Monitor
	executor   *Executor
	
	// Dependencies
	db          *sql.DB
	exchange    exchange.Exchange
	positionMgr *exchange.PositionManager
	orderMgr    *exchange.OrderManager
	
	// Market data integration
	marketIngestor MarketDataProvider
	
	// State
	running bool
	mu      sync.RWMutex
	
	// Configuration
	config *ServiceConfig
}

// ServiceConfig represents service configuration
type ServiceConfig struct {
	Enabled            bool          `json:"enabled"`
	DryRun             bool          `json:"dry_run"`
	MonitorInterval    time.Duration `json:"monitor_interval"`
	SnapshotInterval   time.Duration `json:"snapshot_interval"`
	AlertCooldown      time.Duration `json:"alert_cooldown"`
	MaxPositionReduction float64     `json:"max_position_reduction"`
	
	// Risk thresholds
	RiskThresholds *RiskThresholds `json:"risk_thresholds"`
}

// MarketDataProvider interface for market data integration
type MarketDataProvider interface {
	SubscribeTicker(ctx context.Context, symbol string) (<-chan *market.Ticker, error)
	GetCurrentOrderBook(ctx context.Context, symbol string, limit int) (*market.OrderBook, error)
}

// NewService creates a new PnL monitoring service
func NewService(
	db *sql.DB,
	ex exchange.Exchange,
	posMgr *exchange.PositionManager,
	orderMgr *exchange.OrderManager,
	marketProvider MarketDataProvider,
) *Service {
	// Default configuration
	config := &ServiceConfig{
		Enabled:              true,
		DryRun:               false,
		MonitorInterval:      time.Second * 5,
		SnapshotInterval:     time.Second * 30,
		AlertCooldown:        time.Minute * 5,
		MaxPositionReduction: 0.5,
		RiskThresholds: &RiskThresholds{
			MaxMarginRatio:         0.8,
			WarningMarginRatio:     0.7,
			MaxDailyLoss:           5000.0,
			MaxTotalLoss:           10000.0,
			MaxDrawdownPercent:     0.2,
			MaxPositionLoss:        1000.0,
			MaxPositionLossPercent: 0.1,
			MinAccountBalance:      10000.0,
			MaxLeverage:            10,
		},
	}
	
	// Create components
	calculator := NewCalculator(db)
	monitor := NewMonitor(calculator, config.RiskThresholds)
	executor := NewExecutor(ex, posMgr, orderMgr)
	
	service := &Service{
		calculator:     calculator,
		monitor:        monitor,
		executor:       executor,
		db:             db,
		exchange:       ex,
		positionMgr:    posMgr,
		orderMgr:       orderMgr,
		marketIngestor: marketProvider,
		config:         config,
	}
	
	// Set up callbacks
	service.setupCallbacks()
	
	return service
}

// setupCallbacks sets up callbacks between components
func (s *Service) setupCallbacks() {
	// PnL update callback
	s.calculator.SetPnLUpdateCallback(func(symbol string, pnl float64) {
		log.Printf("PnL updated for %s: $%.2f", symbol, pnl)
	})
	
	// Margin alert callback
	s.calculator.SetMarginAlertCallback(func(alert *MarginAlert) {
		log.Printf("Margin alert for %s: %s (%.2f%%)", alert.Symbol, alert.Message, alert.CurrentRatio*100)
		
		// Save alert to database
		if err := s.saveMarginAlert(alert); err != nil {
			log.Printf("Failed to save margin alert: %v", err)
		}
	})
	
	// Risk event callback
	s.monitor.AddCallback(func(event *TriggerEvent) error {
		log.Printf("Risk event triggered: %s - %s", event.Type, event.Message)
		
		// Save event to database
		if err := s.saveRiskEvent(event); err != nil {
			log.Printf("Failed to save risk event: %v", err)
		}
		
		// Execute automated action
		if s.config.Enabled {
			return s.executor.HandleTriggerEvent(event)
		}
		
		return nil
	})
}

// Start starts the PnL monitoring service
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("service already running")
	}
	s.running = true
	s.mu.Unlock()
	
	log.Println("Starting PnL monitoring service...")
	
	// Load initial data
	if err := s.loadInitialData(ctx); err != nil {
		return fmt.Errorf("failed to load initial data: %w", err)
	}
	
	// Start components
	var wg sync.WaitGroup
	
	// Start PnL calculator monitoring
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.calculator.StartRealTimeMonitoring(ctx); err != nil && err != context.Canceled {
			log.Printf("PnL calculator monitoring error: %v", err)
		}
	}()
	
	// Start risk monitoring
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.monitor.StartMonitoring(ctx); err != nil && err != context.Canceled {
			log.Printf("Risk monitoring error: %v", err)
		}
	}()
	
	// Start market data integration
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.startMarketDataIntegration(ctx); err != nil && err != context.Canceled {
			log.Printf("Market data integration error: %v", err)
		}
	}()
	
	// Start periodic tasks
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.runPeriodicTasks(ctx); err != nil && err != context.Canceled {
			log.Printf("Periodic tasks error: %v", err)
		}
	}()
	
	log.Println("PnL monitoring service started successfully")
	
	// Wait for context cancellation
	<-ctx.Done()
	
	log.Println("Stopping PnL monitoring service...")
	
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
	
	// Wait for all goroutines to finish (with timeout)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		log.Println("PnL monitoring service stopped gracefully")
	case <-time.After(10 * time.Second):
		log.Println("PnL monitoring service stopped with timeout")
	}
	
	return nil
}

// loadInitialData loads initial positions and balances
func (s *Service) loadInitialData(ctx context.Context) error {
	log.Println("Loading initial PnL data...")
	
	// Load positions from database
	if err := s.calculator.LoadPositionsFromDB(ctx); err != nil {
		return fmt.Errorf("failed to load positions: %w", err)
	}
	
	// Load current positions from exchange
	positions, err := s.positionMgr.GetAllPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions from exchange: %w", err)
	}
	
	// Update calculator with current positions
	for _, position := range positions {
		s.calculator.UpdatePosition(position)
	}
	
	// Load account balances
	balances, err := s.exchange.GetAccountBalances(ctx)
	if err != nil {
		return fmt.Errorf("failed to get account balances: %w", err)
	}
	
	// Update calculator with balances
	for asset, balance := range balances {
		s.calculator.UpdateBalance(asset, balance)
	}
	
	log.Printf("Loaded %d positions and %d balances", len(positions), len(balances))
	return nil
}

// startMarketDataIntegration starts market data integration for price updates
func (s *Service) startMarketDataIntegration(ctx context.Context) error {
	// Get all symbols from positions
	positions, err := s.positionMgr.GetAllPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions for market data: %w", err)
	}
	
	// Subscribe to ticker updates for each symbol
	for _, position := range positions {
		if position.Size == 0 {
			continue
		}
		
		go func(symbol string) {
			tickerCh, err := s.marketIngestor.SubscribeTicker(ctx, symbol)
			if err != nil {
				log.Printf("Failed to subscribe to ticker for %s: %v", symbol, err)
				return
			}
			
			for {
				select {
				case ticker, ok := <-tickerCh:
					if !ok {
						log.Printf("Ticker channel closed for %s", symbol)
						return
					}
					
					// Update mark price in calculator
					s.calculator.UpdateMarkPrice(symbol, ticker.LastPrice)
					
				case <-ctx.Done():
					return
				}
			}
		}(position.Symbol)
	}
	
	return nil
}

// runPeriodicTasks runs periodic maintenance tasks
func (s *Service) runPeriodicTasks(ctx context.Context) error {
	ticker := time.NewTicker(time.Minute * 5) // Run every 5 minutes
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Update daily PnL summary
			if err := s.updateDailyPnLSummary(ctx); err != nil {
				log.Printf("Failed to update daily PnL summary: %v", err)
			}
			
			// Clean up old data
			if err := s.cleanupOldData(ctx); err != nil {
				log.Printf("Failed to cleanup old data: %v", err)
			}
			
			// Update account equity history
			if err := s.updateAccountEquityHistory(ctx); err != nil {
				log.Printf("Failed to update account equity history: %v", err)
			}
		}
	}
}

// saveRiskEvent saves a risk event to database
func (s *Service) saveRiskEvent(event *TriggerEvent) error {
	query := `
		INSERT INTO risk_events (
			event_type, action_type, symbol, current_value, threshold_value,
			message, severity, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	
	metadataJSON, _ := json.Marshal(event.Metadata)
	
	_, err := s.db.Exec(query,
		event.Type,
		event.Action,
		event.Symbol,
		event.CurrentValue,
		event.Threshold,
		event.Message,
		event.Severity,
		metadataJSON,
		event.Timestamp,
	)
	
	return err
}

// saveMarginAlert saves a margin alert to database
func (s *Service) saveMarginAlert(alert *MarginAlert) error {
	query := `
		INSERT INTO margin_alerts (
			symbol, alert_type, current_ratio, threshold_ratio,
			message, severity, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	
	_, err := s.db.Exec(query,
		alert.Symbol,
		alert.AlertType,
		alert.CurrentRatio,
		alert.Threshold,
		alert.Message,
		alert.Severity,
		alert.Timestamp,
	)
	
	return err
}

// updateDailyPnLSummary updates the daily PnL summary
func (s *Service) updateDailyPnLSummary(ctx context.Context) error {
	today := time.Now().Format("2006-01-02")
	
	totalUnrealized := s.calculator.GetTotalUnrealizedPnL()
	totalRealized := s.calculator.GetTotalRealizedPnL()
	totalPnL := totalUnrealized + totalRealized
	
	query := `
		INSERT INTO daily_pnl_summary (
			trade_date, starting_balance, ending_balance, realized_pnl,
			unrealized_pnl, total_pnl, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (trade_date) DO UPDATE SET
			ending_balance = EXCLUDED.ending_balance,
			realized_pnl = EXCLUDED.realized_pnl,
			unrealized_pnl = EXCLUDED.unrealized_pnl,
			total_pnl = EXCLUDED.total_pnl,
			updated_at = NOW()
	`
	
	// Simplified - would need proper starting balance tracking
	startingBalance := 100000.0
	endingBalance := startingBalance + totalPnL
	
	_, err := s.db.ExecContext(ctx, query,
		today,
		startingBalance,
		endingBalance,
		totalRealized,
		totalUnrealized,
		totalPnL,
	)
	
	return err
}

// updateAccountEquityHistory updates account equity history
func (s *Service) updateAccountEquityHistory(ctx context.Context) error {
	totalUnrealized := s.calculator.GetTotalUnrealizedPnL()
	totalRealized := s.calculator.GetTotalRealizedPnL()
	marginRatio := s.calculator.GetMarginRatio()
	
	// Get position count
	positions, err := s.positionMgr.GetAllPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}
	
	positionCount := 0
	for _, pos := range positions {
		if pos.Size != 0 {
			positionCount++
		}
	}
	
	query := `
		INSERT INTO account_equity_history (
			total_equity, available_balance, used_margin, unrealized_pnl,
			realized_pnl, margin_ratio, position_count, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`
	
	// Simplified calculations - would need real balance data
	totalEquity := 100000.0 + totalRealized + totalUnrealized
	availableBalance := totalEquity * 0.8 // Assume 80% available
	usedMargin := totalEquity * 0.2       // Assume 20% used
	
	_, err = s.db.ExecContext(ctx, query,
		totalEquity,
		availableBalance,
		usedMargin,
		totalUnrealized,
		totalRealized,
		marginRatio,
		positionCount,
	)
	
	return err
}

// cleanupOldData removes old monitoring data
func (s *Service) cleanupOldData(ctx context.Context) error {
	// Keep only last 30 days of snapshots
	cutoff := time.Now().AddDate(0, 0, -30)
	
	tables := []string{
		"pnl_snapshots",
		"margin_alerts",
		"account_equity_history",
	}
	
	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM %s WHERE created_at < $1", table)
		if _, err := s.db.ExecContext(ctx, query, cutoff); err != nil {
			log.Printf("Failed to cleanup %s: %v", table, err)
		}
	}
	
	return nil
}

// GetStatus returns service status
func (s *Service) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return map[string]interface{}{
		"running":           s.running,
		"config":            s.config,
		"calculator_status": s.calculator.GetCurrentStatus(),
		"monitor_status":    s.monitor.GetCurrentStatus(),
		"executor_status":   s.executor.GetStatus(),
	}
}

// UpdateConfig updates service configuration
func (s *Service) UpdateConfig(config *ServiceConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.config = config
	
	// Update component configurations
	s.executor.SetEnabled(config.Enabled)
	s.executor.SetDryRun(config.DryRun)
	s.monitor.UpdateThresholds(config.RiskThresholds)
	
	log.Printf("PnL service configuration updated")
	return nil
}

// GetPnLSummary returns current PnL summary
func (s *Service) GetPnLSummary() map[string]interface{} {
	totalUnrealized := s.calculator.GetTotalUnrealizedPnL()
	totalRealized := s.calculator.GetTotalRealizedPnL()
	marginRatio := s.calculator.GetMarginRatio()
	
	snapshots, _ := s.calculator.GetAllPnLSnapshots()
	
	return map[string]interface{}{
		"total_unrealized_pnl": totalUnrealized,
		"total_realized_pnl":   totalRealized,
		"total_pnl":            totalUnrealized + totalRealized,
		"margin_ratio":         marginRatio,
		"position_count":       len(snapshots),
		"positions":            snapshots,
		"last_update":          time.Now(),
	}
}
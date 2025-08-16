package pnl

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/market"
)

// Calculator handles real-time PnL calculations
type Calculator struct {
	db         *sql.DB
	positions  map[string]*exchange.Position
	markPrices map[string]float64
	balances   map[string]*exchange.AccountBalance
	mu         sync.RWMutex
	
	// Configuration
	updateInterval time.Duration
	
	// Callbacks
	onPnLUpdate    func(symbol string, pnl float64)
	onMarginAlert  func(alert *MarginAlert)
}

// MarginAlert represents a margin alert
type MarginAlert struct {
	Symbol      string    `json:"symbol"`
	AlertType   string    `json:"alert_type"`
	CurrentRatio float64  `json:"current_ratio"`
	Threshold   float64   `json:"threshold"`
	Message     string    `json:"message"`
	Severity    string    `json:"severity"`
	Timestamp   time.Time `json:"timestamp"`
}

// PnLSnapshot represents a PnL snapshot
type PnLSnapshot struct {
	Symbol         string    `json:"symbol"`
	UnrealizedPnL  float64   `json:"unrealized_pnl"`
	RealizedPnL    float64   `json:"realized_pnl"`
	TotalPnL       float64   `json:"total_pnl"`
	MarginUsed     float64   `json:"margin_used"`
	MarginRatio    float64   `json:"margin_ratio"`
	MarkPrice      float64   `json:"mark_price"`
	EntryPrice     float64   `json:"entry_price"`
	PositionSize   float64   `json:"position_size"`
	Timestamp      time.Time `json:"timestamp"`
}

// NewCalculator creates a new PnL calculator
func NewCalculator(db *sql.DB) *Calculator {
	return &Calculator{
		db:             db,
		positions:      make(map[string]*exchange.Position),
		markPrices:     make(map[string]float64),
		balances:       make(map[string]*exchange.AccountBalance),
		updateInterval: time.Second, // Update every second
		onPnLUpdate: func(symbol string, pnl float64) {
			// Default: do nothing
		},
		onMarginAlert: func(alert *MarginAlert) {
			// Default: do nothing
		},
	}
}

// SetPnLUpdateCallback sets the callback for PnL updates
func (c *Calculator) SetPnLUpdateCallback(callback func(string, float64)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onPnLUpdate = callback
}

// SetMarginAlertCallback sets the callback for margin alerts
func (c *Calculator) SetMarginAlertCallback(callback func(*MarginAlert)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onMarginAlert = callback
}

// UpdatePosition updates position information
func (c *Calculator) UpdatePosition(position *exchange.Position) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.positions[position.Symbol] = position
	
	// Calculate and update unrealized PnL
	if markPrice, exists := c.markPrices[position.Symbol]; exists {
		oldPnL := position.UnrealizedPnL
		newPnL := c.calculateUnrealizedPnL(position, markPrice)
		position.UnrealizedPnL = newPnL
		
		// Trigger callback if PnL changed significantly
		if abs(newPnL-oldPnL) > 0.01 { // 1 cent threshold
			go c.onPnLUpdate(position.Symbol, newPnL)
		}
	}
}

// UpdateMarkPrice updates mark price for a symbol
func (c *Calculator) UpdateMarkPrice(symbol string, price float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.markPrices[symbol] = price
	
	// Update PnL for this symbol if position exists
	if position, exists := c.positions[symbol]; exists {
		oldPnL := position.UnrealizedPnL
		newPnL := c.calculateUnrealizedPnL(position, price)
		position.UnrealizedPnL = newPnL
		
		// Trigger callback if PnL changed significantly
		if abs(newPnL-oldPnL) > 0.01 { // 1 cent threshold
			go c.onPnLUpdate(symbol, newPnL)
		}
		
		// Check margin requirements
		c.checkMarginRequirements(position)
	}
}

// UpdateBalance updates account balance information
func (c *Calculator) UpdateBalance(asset string, balance *exchange.AccountBalance) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.balances[asset] = balance
}

// CalculateUnrealizedPnL calculates unrealized PnL for a symbol
func (c *Calculator) CalculateUnrealizedPnL(symbol string) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	position, posExists := c.positions[symbol]
	markPrice, priceExists := c.markPrices[symbol]
	
	if !posExists || !priceExists {
		return 0
	}
	
	return c.calculateUnrealizedPnL(position, markPrice)
}

// calculateUnrealizedPnL internal method to calculate unrealized PnL
func (c *Calculator) calculateUnrealizedPnL(position *exchange.Position, markPrice float64) float64 {
	if position.Size == 0 {
		return 0
	}
	
	// Calculate PnL based on position direction
	priceDiff := markPrice - position.EntryPrice
	
	// For short positions, invert the price difference
	if position.Side == "SHORT" {
		priceDiff = -priceDiff
	}
	
	// Calculate PnL
	pnl := priceDiff * abs(position.Size)
	
	return pnl
}

// GetTotalUnrealizedPnL returns total unrealized PnL across all positions
func (c *Calculator) GetTotalUnrealizedPnL() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	totalPnL := 0.0
	for _, position := range c.positions {
		totalPnL += position.UnrealizedPnL
	}
	
	return totalPnL
}

// GetTotalRealizedPnL returns total realized PnL from balances
func (c *Calculator) GetTotalRealizedPnL() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	totalPnL := 0.0
	for _, balance := range c.balances {
		totalPnL += balance.RealizedPnL
	}
	
	return totalPnL
}

// GetMarginRatio calculates current margin ratio
func (c *Calculator) GetMarginRatio() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	totalEquity := c.getTotalEquity()
	totalMarginUsed := c.getTotalMarginUsed()
	
	if totalMarginUsed == 0 {
		return 0
	}
	
	return totalEquity / totalMarginUsed
}

// getTotalEquity calculates total account equity
func (c *Calculator) getTotalEquity() float64 {
	totalEquity := 0.0
	
	// Add available balances
	for _, balance := range c.balances {
		totalEquity += balance.Available + balance.Locked
	}
	
	// Add unrealized PnL
	totalEquity += c.GetTotalUnrealizedPnL()
	
	return totalEquity
}

// getTotalMarginUsed calculates total margin used
func (c *Calculator) getTotalMarginUsed() float64 {
	totalMargin := 0.0
	
	for _, position := range c.positions {
		if position.Size != 0 {
			// Calculate margin based on position size and leverage
			notional := abs(position.Size) * position.MarkPrice
			margin := notional / float64(position.Leverage)
			totalMargin += margin
		}
	}
	
	return totalMargin
}

// checkMarginRequirements checks if margin requirements are met
func (c *Calculator) checkMarginRequirements(position *exchange.Position) {
	if position.Size == 0 {
		return
	}
	
	// Calculate current margin ratio for this position
	notional := abs(position.Size) * position.MarkPrice
	marginUsed := notional / float64(position.Leverage)
	
	// Get available balance (simplified - assume USDT)
	balance, exists := c.balances["USDT"]
	if !exists {
		return
	}
	
	equity := balance.Available + balance.Locked + position.UnrealizedPnL
	marginRatio := equity / marginUsed
	
	// Check various margin thresholds
	if marginRatio < 1.1 { // 110% - Critical
		c.onMarginAlert(&MarginAlert{
			Symbol:       position.Symbol,
			AlertType:    "margin_call",
			CurrentRatio: marginRatio,
			Threshold:    1.1,
			Message:      fmt.Sprintf("Critical margin level: %.2f%%", marginRatio*100),
			Severity:     "critical",
			Timestamp:    time.Now(),
		})
	} else if marginRatio < 1.3 { // 130% - Warning
		c.onMarginAlert(&MarginAlert{
			Symbol:       position.Symbol,
			AlertType:    "margin_warning",
			CurrentRatio: marginRatio,
			Threshold:    1.3,
			Message:      fmt.Sprintf("Low margin level: %.2f%%", marginRatio*100),
			Severity:     "warning",
			Timestamp:    time.Now(),
		})
	}
}

// GetPnLSnapshot returns current PnL snapshot for a symbol
func (c *Calculator) GetPnLSnapshot(symbol string) (*PnLSnapshot, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	position, exists := c.positions[symbol]
	if !exists {
		return nil, fmt.Errorf("position not found for symbol: %s", symbol)
	}
	
	markPrice := c.markPrices[symbol]
	
	// Calculate margin used
	notional := abs(position.Size) * markPrice
	marginUsed := 0.0
	if position.Leverage > 0 {
		marginUsed = notional / float64(position.Leverage)
	}
	
	// Calculate margin ratio
	balance, _ := c.balances["USDT"] // Simplified
	equity := 0.0
	if balance != nil {
		equity = balance.Available + balance.Locked + position.UnrealizedPnL
	}
	
	marginRatio := 0.0
	if marginUsed > 0 {
		marginRatio = equity / marginUsed
	}
	
	return &PnLSnapshot{
		Symbol:        symbol,
		UnrealizedPnL: position.UnrealizedPnL,
		RealizedPnL:   position.RealizedPnL,
		TotalPnL:      position.UnrealizedPnL + position.RealizedPnL,
		MarginUsed:    marginUsed,
		MarginRatio:   marginRatio,
		MarkPrice:     markPrice,
		EntryPrice:    position.EntryPrice,
		PositionSize:  position.Size,
		Timestamp:     time.Now(),
	}, nil
}

// GetAllPnLSnapshots returns PnL snapshots for all positions
func (c *Calculator) GetAllPnLSnapshots() ([]*PnLSnapshot, error) {
	c.mu.RLock()
	symbols := make([]string, 0, len(c.positions))
	for symbol := range c.positions {
		symbols = append(symbols, symbol)
	}
	c.mu.RUnlock()
	
	snapshots := make([]*PnLSnapshot, 0, len(symbols))
	for _, symbol := range symbols {
		snapshot, err := c.GetPnLSnapshot(symbol)
		if err != nil {
			continue // Skip positions with errors
		}
		snapshots = append(snapshots, snapshot)
	}
	
	return snapshots, nil
}

// StartRealTimeMonitoring starts real-time PnL monitoring
func (c *Calculator) StartRealTimeMonitoring(ctx context.Context) error {
	ticker := time.NewTicker(c.updateInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Update all positions with latest mark prices
			c.updateAllPositions()
			
			// Save snapshots to database
			if err := c.saveSnapshots(ctx); err != nil {
				fmt.Printf("Failed to save PnL snapshots: %v\n", err)
			}
		}
	}
}

// updateAllPositions updates PnL for all positions
func (c *Calculator) updateAllPositions() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	for symbol, position := range c.positions {
		if markPrice, exists := c.markPrices[symbol]; exists {
			oldPnL := position.UnrealizedPnL
			newPnL := c.calculateUnrealizedPnL(position, markPrice)
			position.UnrealizedPnL = newPnL
			
			// Trigger callback if PnL changed significantly
			if abs(newPnL-oldPnL) > 0.01 {
				go c.onPnLUpdate(symbol, newPnL)
			}
			
			// Check margin requirements
			c.checkMarginRequirements(position)
		}
	}
}

// saveSnapshots saves PnL snapshots to database
func (c *Calculator) saveSnapshots(ctx context.Context) error {
	snapshots, err := c.GetAllPnLSnapshots()
	if err != nil {
		return fmt.Errorf("failed to get snapshots: %w", err)
	}
	
	for _, snapshot := range snapshots {
		query := `
			INSERT INTO pnl_snapshots (
				symbol, unrealized_pnl, realized_pnl, total_pnl,
				margin_used, margin_ratio, mark_price, entry_price,
				position_size, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`
		
		_, err := c.db.ExecContext(ctx, query,
			snapshot.Symbol,
			snapshot.UnrealizedPnL,
			snapshot.RealizedPnL,
			snapshot.TotalPnL,
			snapshot.MarginUsed,
			snapshot.MarginRatio,
			snapshot.MarkPrice,
			snapshot.EntryPrice,
			snapshot.PositionSize,
			snapshot.Timestamp,
		)
		
		if err != nil {
			return fmt.Errorf("failed to save snapshot for %s: %w", snapshot.Symbol, err)
		}
	}
	
	return nil
}

// LoadPositionsFromDB loads positions from database
func (c *Calculator) LoadPositionsFromDB(ctx context.Context) error {
	query := `
		SELECT symbol, side, size, entry_price, mark_price, unrealized_pnl,
			   realized_pnl, leverage, margin_type, updated_at
		FROM positions
		WHERE size != 0
	`
	
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query positions: %w", err)
	}
	defer rows.Close()
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	for rows.Next() {
		var pos exchange.Position
		if err := rows.Scan(
			&pos.Symbol,
			&pos.Side,
			&pos.Size,
			&pos.EntryPrice,
			&pos.MarkPrice,
			&pos.UnrealizedPnL,
			&pos.RealizedPnL,
			&pos.Leverage,
			&pos.MarginType,
			&pos.UpdatedAt,
		); err != nil {
			return fmt.Errorf("failed to scan position: %w", err)
		}
		
		c.positions[pos.Symbol] = &pos
		c.markPrices[pos.Symbol] = pos.MarkPrice
	}
	
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating positions: %w", err)
	}
	
	return nil
}

// Helper function to calculate absolute value
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
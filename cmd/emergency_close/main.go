package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"qcat/internal/config"
	"qcat/internal/exchange"
	"qcat/internal/exchange/binance"
)

// EmergencyCloser handles emergency position closure
type EmergencyCloser struct {
	exchange exchange.Exchange
	config   *config.Config
}

// NewEmergencyCloser creates a new emergency closer
func NewEmergencyCloser(cfg *config.Config) (*EmergencyCloser, error) {
	// Initialize Binance client
	exchangeConfig := &exchange.ExchangeConfig{
		APIKey:    cfg.Exchange.APIKey,
		APISecret: cfg.Exchange.APISecret,
		TestNet:   cfg.Exchange.TestNet,
	}

	// Create rate limiter with more conservative settings
	rateLimiter := exchange.NewSimpleRateLimiter(600, time.Minute) // Reduced rate limit

	client := binance.NewClient(exchangeConfig, rateLimiter)

	return &EmergencyCloser{
		exchange: client,
		config:   cfg,
	}, nil
}

// CloseAllPositions closes all open positions and cancels all orders
func (ec *EmergencyCloser) CloseAllPositions(ctx context.Context) error {
	log.Println("=== EMERGENCY POSITION CLOSURE STARTED ===")

	// Step 1: Cancel all open orders for all symbols
	log.Println("Step 1: Canceling all open orders...")
	if err := ec.cancelAllOpenOrders(ctx); err != nil {
		log.Printf("Warning: Failed to cancel some orders: %v", err)
		// Continue with position closure even if order cancellation fails
	}

	// Step 2: Get all current positions
	log.Println("Step 2: Getting current positions...")
	positions, err := ec.exchange.GetPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	if len(positions) == 0 {
		log.Println("No positions found to close")
		return nil
	}

	log.Printf("Found %d positions to close", len(positions))

	// Step 3: Close all positions
	log.Println("Step 3: Closing all positions...")
	var errors []error
	closedCount := 0

	for _, position := range positions {
		if position.Size == 0 {
			log.Printf("Skipping %s - no position size", position.Symbol)
			continue
		}

		log.Printf("Closing position: %s, Size: %.8f, Side: %s",
			position.Symbol, position.Size, position.Side)

		if err := ec.closePosition(ctx, position); err != nil {
			log.Printf("Failed to close position %s: %v", position.Symbol, err)
			errors = append(errors, fmt.Errorf("failed to close %s: %w", position.Symbol, err))
			continue
		}

		closedCount++
		log.Printf("Successfully closed position for %s", position.Symbol)

		// Small delay to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("=== EMERGENCY CLOSURE COMPLETED ===")
	log.Printf("Positions closed: %d/%d", closedCount, len(positions))

	if len(errors) > 0 {
		log.Printf("Encountered %d errors during closure:", len(errors))
		for _, err := range errors {
			log.Printf("  - %v", err)
		}
		return fmt.Errorf("completed with %d errors", len(errors))
	}

	return nil
}

// cancelAllOpenOrders cancels all open orders using the bulk cancel API
func (ec *EmergencyCloser) cancelAllOpenOrders(ctx context.Context) error {
	// Get all symbols with open orders first
	symbols, err := ec.getSymbolsWithOpenOrders(ctx)
	if err != nil {
		return fmt.Errorf("failed to get symbols with open orders: %w", err)
	}

	if len(symbols) == 0 {
		log.Println("No open orders found")
		return nil
	}

	log.Printf("Found open orders for %d symbols", len(symbols))

	// Cancel orders for each symbol using the bulk cancel API
	var errors []error
	for _, symbol := range symbols {
		log.Printf("Canceling all orders for %s...", symbol)

		if err := ec.exchange.CancelAllOrders(ctx, symbol); err != nil {
			log.Printf("Failed to cancel orders for %s: %v", symbol, err)
			errors = append(errors, fmt.Errorf("failed to cancel orders for %s: %w", symbol, err))
			continue
		}

		log.Printf("Successfully canceled all orders for %s", symbol)

		// Small delay to avoid rate limiting
		time.Sleep(50 * time.Millisecond)
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to cancel orders for some symbols: %v", errors)
	}

	return nil
}

// getSymbolsWithOpenOrders gets all symbols that have open orders
func (ec *EmergencyCloser) getSymbolsWithOpenOrders(ctx context.Context) ([]string, error) {
	// Get open orders for all symbols (empty symbol parameter gets all)
	orders, err := ec.exchange.GetOpenOrders(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get open orders: %w", err)
	}

	// Extract unique symbols
	symbolMap := make(map[string]bool)
	for _, order := range orders {
		symbolMap[order.Symbol] = true
	}

	symbols := make([]string, 0, len(symbolMap))
	for symbol := range symbolMap {
		symbols = append(symbols, symbol)
	}

	return symbols, nil
}

// closePosition closes a single position using market order
func (ec *EmergencyCloser) closePosition(ctx context.Context, position *exchange.Position) error {
	// Determine the side for closing the position
	var side string
	closeQuantity := math.Abs(position.Size)

	if position.Size > 0 {
		// Long position - sell to close
		side = "SELL"
	} else {
		// Short position - buy to close
		side = "BUY"
	}

	// Create market order to close position
	orderReq := &exchange.OrderRequest{
		Symbol:     position.Symbol,
		Side:       side,
		Type:       "MARKET",
		Quantity:   closeQuantity,
		ReduceOnly: true, // Important: this ensures we're only closing existing position
	}

	// Retry logic for network issues
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("Attempting to close position %s (attempt %d/%d)", position.Symbol, attempt, maxRetries)

		// Place the order
		orderResp, err := ec.exchange.PlaceOrder(ctx, orderReq)
		if err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("failed to place close order after %d attempts: %w", maxRetries, err)
			}
			log.Printf("Attempt %d failed: %v, retrying in 2 seconds...", attempt, err)
			time.Sleep(2 * time.Second)
			continue
		}

		if !orderResp.Success {
			if attempt == maxRetries {
				return fmt.Errorf("order rejected after %d attempts: %s", maxRetries, orderResp.Error)
			}
			log.Printf("Order rejected on attempt %d: %s, retrying...", attempt, orderResp.Error)
			time.Sleep(2 * time.Second)
			continue
		}

		log.Printf("Close order placed: OrderID=%s, Symbol=%s, Side=%s, Quantity=%.8f",
			orderResp.OrderID, position.Symbol, side, closeQuantity)
		return nil
	}

	return fmt.Errorf("failed to close position after %d attempts", maxRetries)
}

func main() {
	log.Println("Emergency Position Closer - Starting...")

	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Validate that we have API credentials
	if cfg.Exchange.APIKey == "" || cfg.Exchange.APISecret == "" {
		log.Fatalf("API credentials not configured. Please set EXCHANGE_API_KEY and EXCHANGE_API_SECRET environment variables")
	}

	// Create emergency closer
	closer, err := NewEmergencyCloser(cfg)
	if err != nil {
		log.Fatalf("Failed to create emergency closer: %v", err)
	}

	// Confirm before proceeding
	fmt.Print("WARNING: This will close ALL positions and cancel ALL orders. Are you sure? (yes/no): ")
	var confirmation string
	fmt.Scanln(&confirmation)

	if confirmation != "yes" {
		log.Println("Operation cancelled by user")
		os.Exit(0)
	}

	// Execute emergency closure
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := closer.CloseAllPositions(ctx); err != nil {
		log.Fatalf("Emergency closure failed: %v", err)
	}

	log.Println("Emergency closure completed successfully!")
}

package binance

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// EndpointRateLimit represents rate limit configuration for a specific endpoint
type EndpointRateLimit struct {
	Endpoint     string
	RequestsPerMinute int
	RequestsPerSecond int
	BurstLimit   int
	Window       time.Duration
	LastReset    time.Time
	CurrentCount int
	mu           sync.Mutex
}

// RateLimitManager manages rate limits for different Binance endpoints
type RateLimitManager struct {
	limits map[string]*EndpointRateLimit
	mu     sync.RWMutex
}

// NewRateLimitManager creates a new rate limit manager
func NewRateLimitManager() *RateLimitManager {
	rlm := &RateLimitManager{
		limits: make(map[string]*EndpointRateLimit),
	}
	
	// Initialize rate limits for different endpoints based on Binance documentation
	rlm.initializeEndpointLimits()
	
	return rlm
}

// initializeEndpointLimits sets up rate limits for different endpoints
func (rlm *RateLimitManager) initializeEndpointLimits() {
	// Position Risk endpoint - most restrictive
	rlm.limits[MethodPositions] = &EndpointRateLimit{
		Endpoint:          MethodPositions,
		RequestsPerMinute: 5,   // Very conservative for position risk
		RequestsPerSecond: 1,   // Max 1 request per second
		BurstLimit:        2,   // Allow small burst
		Window:            time.Minute,
		LastReset:         time.Now(),
		CurrentCount:      0,
	}
	
	// Account endpoints
	rlm.limits[MethodAccount] = &EndpointRateLimit{
		Endpoint:          MethodAccount,
		RequestsPerMinute: 10,
		RequestsPerSecond: 2,
		BurstLimit:        3,
		Window:            time.Minute,
		LastReset:         time.Now(),
		CurrentCount:      0,
	}
	
	rlm.limits[MethodBalance] = &EndpointRateLimit{
		Endpoint:          MethodBalance,
		RequestsPerMinute: 10,
		RequestsPerSecond: 2,
		BurstLimit:        3,
		Window:            time.Minute,
		LastReset:         time.Now(),
		CurrentCount:      0,
	}
	
	// Order endpoints
	rlm.limits[MethodOrder] = &EndpointRateLimit{
		Endpoint:          MethodOrder,
		RequestsPerMinute: 60,
		RequestsPerSecond: 10,
		BurstLimit:        15,
		Window:            time.Minute,
		LastReset:         time.Now(),
		CurrentCount:      0,
	}
	
	// Market data endpoints
	rlm.limits[MethodTickerPrice] = &EndpointRateLimit{
		Endpoint:          MethodTickerPrice,
		RequestsPerMinute: 40,
		RequestsPerSecond: 5,
		BurstLimit:        10,
		Window:            time.Minute,
		LastReset:         time.Now(),
		CurrentCount:      0,
	}
}

// CheckRateLimit checks if a request to the endpoint is allowed
func (rlm *RateLimitManager) CheckRateLimit(ctx context.Context, endpoint string) error {
	rlm.mu.RLock()
	limit, exists := rlm.limits[endpoint]
	rlm.mu.RUnlock()
	
	if !exists {
		// Use default conservative limit for unknown endpoints
		return rlm.checkDefaultLimit(ctx, endpoint)
	}
	
	limit.mu.Lock()
	defer limit.mu.Unlock()
	
	now := time.Now()
	
	// Reset counter if window has passed
	if now.Sub(limit.LastReset) >= limit.Window {
		limit.CurrentCount = 0
		limit.LastReset = now
	}
	
	// Check if we've exceeded the limit
	if limit.CurrentCount >= limit.RequestsPerMinute {
		return fmt.Errorf("rate limit exceeded for %s: %d requests in %v", 
			endpoint, limit.CurrentCount, limit.Window)
	}
	
	// Increment counter
	limit.CurrentCount++
	
	return nil
}

// WaitForRateLimit waits until a request can be made to the endpoint
func (rlm *RateLimitManager) WaitForRateLimit(ctx context.Context, endpoint string) error {
	maxRetries := 5
	baseDelay := time.Second
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := rlm.CheckRateLimit(ctx, endpoint); err == nil {
			return nil
		}
		
		// Calculate exponential backoff delay
		delay := time.Duration(1<<attempt) * baseDelay
		if endpoint == MethodPositions {
			// Longer delays for position risk endpoint
			delay = time.Duration(1<<attempt) * 5 * time.Second
		}
		
		// Cap maximum delay
		if delay > 60*time.Second {
			delay = 60 * time.Second
		}
		
		log.Printf("Rate limit hit for %s, waiting %v (attempt %d/%d)", 
			endpoint, delay, attempt+1, maxRetries)
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			continue
		}
	}
	
	return fmt.Errorf("rate limit exceeded after %d retries for %s", maxRetries, endpoint)
}

// checkDefaultLimit applies a conservative default rate limit
func (rlm *RateLimitManager) checkDefaultLimit(ctx context.Context, endpoint string) error {
	// Create a default limit for unknown endpoints
	rlm.mu.Lock()
	if _, exists := rlm.limits[endpoint]; !exists {
		rlm.limits[endpoint] = &EndpointRateLimit{
			Endpoint:          endpoint,
			RequestsPerMinute: 5,  // Very conservative
			RequestsPerSecond: 1,
			BurstLimit:        2,
			Window:            time.Minute,
			LastReset:         time.Now(),
			CurrentCount:      0,
		}
	}
	rlm.mu.Unlock()
	
	return rlm.CheckRateLimit(ctx, endpoint)
}

// GetEndpointStats returns statistics for an endpoint
func (rlm *RateLimitManager) GetEndpointStats(endpoint string) map[string]interface{} {
	rlm.mu.RLock()
	limit, exists := rlm.limits[endpoint]
	rlm.mu.RUnlock()
	
	if !exists {
		return map[string]interface{}{
			"endpoint": endpoint,
			"status":   "not_configured",
		}
	}
	
	limit.mu.Lock()
	defer limit.mu.Unlock()
	
	return map[string]interface{}{
		"endpoint":            endpoint,
		"requests_per_minute": limit.RequestsPerMinute,
		"current_count":       limit.CurrentCount,
		"last_reset":          limit.LastReset,
		"window":              limit.Window.String(),
		"utilization":         float64(limit.CurrentCount) / float64(limit.RequestsPerMinute),
	}
}

// GetAllStats returns statistics for all endpoints
func (rlm *RateLimitManager) GetAllStats() map[string]interface{} {
	rlm.mu.RLock()
	defer rlm.mu.RUnlock()
	
	stats := make(map[string]interface{})
	for endpoint := range rlm.limits {
		stats[endpoint] = rlm.GetEndpointStats(endpoint)
	}
	
	return stats
}

// ResetEndpointLimit resets the rate limit counter for an endpoint
func (rlm *RateLimitManager) ResetEndpointLimit(endpoint string) {
	rlm.mu.RLock()
	limit, exists := rlm.limits[endpoint]
	rlm.mu.RUnlock()
	
	if exists {
		limit.mu.Lock()
		limit.CurrentCount = 0
		limit.LastReset = time.Now()
		limit.mu.Unlock()
		
		log.Printf("Reset rate limit for endpoint: %s", endpoint)
	}
}

// UpdateEndpointLimit updates the rate limit configuration for an endpoint
func (rlm *RateLimitManager) UpdateEndpointLimit(endpoint string, requestsPerMinute, requestsPerSecond, burstLimit int) {
	rlm.mu.Lock()
	defer rlm.mu.Unlock()
	
	if limit, exists := rlm.limits[endpoint]; exists {
		limit.mu.Lock()
		limit.RequestsPerMinute = requestsPerMinute
		limit.RequestsPerSecond = requestsPerSecond
		limit.BurstLimit = burstLimit
		limit.mu.Unlock()
		
		log.Printf("Updated rate limit for %s: %d/min, %d/sec, burst: %d", 
			endpoint, requestsPerMinute, requestsPerSecond, burstLimit)
	}
}

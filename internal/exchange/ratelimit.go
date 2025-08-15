package exchange

import (
	"context"
	"fmt"
	"sync"
	"time"

	"qcat/internal/cache"
)

// RateLimiter manages API rate limits
type RateLimiter struct {
	cache       cache.Cacher
	limits      map[string]*Limit
	mu          sync.RWMutex
	defaultWait time.Duration
}

// Limit represents a rate limit
type Limit struct {
	Name      string
	Interval  time.Duration
	MaxTokens int
	Tokens    int
	LastReset time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(cache cache.Cacher, defaultWait time.Duration) *RateLimiter {
	return &RateLimiter{
		cache:       cache,
		limits:      make(map[string]*Limit),
		defaultWait: defaultWait,
	}
}

// AddLimit adds a new rate limit
func (r *RateLimiter) AddLimit(name string, interval time.Duration, maxTokens int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.limits[name] = &Limit{
		Name:      name,
		Interval:  interval,
		MaxTokens: maxTokens,
		Tokens:    maxTokens,
		LastReset: time.Now(),
	}
}

// Wait waits until a rate limit allows an action
func (r *RateLimiter) Wait(ctx context.Context, name string) error {
	r.mu.Lock()
	limit, exists := r.limits[name]
	if !exists {
		r.mu.Unlock()
		return fmt.Errorf("rate limit not found: %s", name)
	}

	now := time.Now()
	if now.Sub(limit.LastReset) >= limit.Interval {
		// Reset tokens if interval has passed
		limit.Tokens = limit.MaxTokens
		limit.LastReset = now
	}

	if limit.Tokens <= 0 {
		// Calculate wait time
		waitTime := limit.LastReset.Add(limit.Interval).Sub(now)
		r.mu.Unlock()

		// Wait for tokens to be available
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			return r.Wait(ctx, name) // Try again
		}
	}

	// Use a token
	limit.Tokens--
	r.mu.Unlock()
	return nil
}

// WaitWithFallback waits with a fallback mechanism using Redis
func (r *RateLimiter) WaitWithFallback(ctx context.Context, exchange, name string, limit int, window time.Duration) error {
	// Try local rate limiter first
	if err := r.Wait(ctx, name); err != nil {
		// Fall back to Redis-based rate limiting
		ok, err := r.cache.CheckRateLimit(ctx, fmt.Sprintf("%s:%s", exchange, name), limit, window)
		if err != nil {
			// If Redis fails, use default wait
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(r.defaultWait):
				return nil
			}
		}
		if !ok {
			return fmt.Errorf("rate limit exceeded for %s", name)
		}
	}
	return nil
}

// Reset resets all rate limits
func (r *RateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for _, limit := range r.limits {
		limit.Tokens = limit.MaxTokens
		limit.LastReset = now
	}
}

// GetLimit returns the current limit state
func (r *RateLimiter) GetLimit(name string) (*Limit, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	limit, exists := r.limits[name]
	return limit, exists
}

// GetAllLimits returns all limit states
func (r *RateLimiter) GetAllLimits() map[string]*Limit {
	r.mu.RLock()
	defer r.mu.RUnlock()

	limits := make(map[string]*Limit)
	for name, limit := range r.limits {
		limits[name] = &Limit{
			Name:      limit.Name,
			Interval:  limit.Interval,
			MaxTokens: limit.MaxTokens,
			Tokens:    limit.Tokens,
			LastReset: limit.LastReset,
		}
	}
	return limits
}

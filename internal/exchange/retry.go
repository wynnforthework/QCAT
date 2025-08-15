package exchange

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// RetryConfig represents retry configuration
type RetryConfig struct {
	MaxRetries  int
	InitialWait time.Duration
	MaxWait     time.Duration
	Factor      float64
	Jitter      float64
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:  3,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     5 * time.Second,
		Factor:      2.0,
		Jitter:      0.1,
	}
}

// Error represents an exchange error
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("exchange error %d: %s", e.Code, e.Message)
}

// RetryableFunc represents a function that can be retried
type RetryableFunc func(ctx context.Context) error

// IsRetryableError determines if an error should be retried
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Add specific error type checks here
	if e, ok := err.(*Error); ok {
		// Common HTTP errors that should be retried
		switch e.Code {
		case 429, // Too Many Requests
			500, // Internal Server Error
			502, // Bad Gateway
			503, // Service Unavailable
			504: // Gateway Timeout
			return true
		}
	}

	// Add more error type checks as needed
	return false
}

// WithRetry wraps a function with retry logic
func WithRetry(ctx context.Context, fn RetryableFunc, config *RetryConfig) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var err error
	wait := config.InitialWait

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		err = fn(ctx)
		if err == nil {
			return nil
		}

		if !IsRetryableError(err) {
			return err
		}

		if attempt == config.MaxRetries {
			return fmt.Errorf("max retries exceeded: %w", err)
		}

		// Calculate next wait duration with exponential backoff and jitter
		jitter := 1.0 + (config.Jitter * (2*rand.Float64() - 1))
		wait = time.Duration(float64(wait) * config.Factor * jitter)

		if wait > config.MaxWait {
			wait = config.MaxWait
		}

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
			continue
		}
	}

	return err
}

// RetryWithResult wraps a function that returns a result with retry logic
func RetryWithResult[T any](ctx context.Context, fn func(context.Context) (T, error), config *RetryConfig) (T, error) {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var (
		result T
		err    error
		wait   = config.InitialWait
	)

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		result, err = fn(ctx)
		if err == nil {
			return result, nil
		}

		if !IsRetryableError(err) {
			return result, err
		}

		if attempt == config.MaxRetries {
			return result, fmt.Errorf("max retries exceeded: %w", err)
		}

		// Calculate next wait duration with exponential backoff and jitter
		jitter := 1.0 + (config.Jitter * (2*rand.Float64() - 1))
		wait = time.Duration(float64(wait) * config.Factor * jitter)

		if wait > config.MaxWait {
			wait = config.MaxWait
		}

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(wait):
			continue
		}
	}

	return result, err
}

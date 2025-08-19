package binance

import (
	"fmt"
	"strings"
)

// BinanceErrorCode represents Binance API error codes
type BinanceErrorCode int

const (
	// Authentication errors
	ErrorCodeAPIKeyFormatInvalid    BinanceErrorCode = -2014
	ErrorCodeInvalidSignature       BinanceErrorCode = -1022
	ErrorCodeTimestampOutOfWindow   BinanceErrorCode = -1021
	ErrorCodeInvalidAPIKey          BinanceErrorCode = -2015
	ErrorCodeIPNotAllowed           BinanceErrorCode = -2016
	
	// Rate limiting errors
	ErrorCodeRateLimitExceeded      BinanceErrorCode = -1003
	ErrorCodeTooManyRequests        BinanceErrorCode = -1015
	
	// Permission errors
	ErrorCodeNoPermission           BinanceErrorCode = -2010
	ErrorCodeInsufficientBalance    BinanceErrorCode = -2019
	
	// Order errors
	ErrorCodeOrderNotFound          BinanceErrorCode = -2013
	ErrorCodeInvalidOrderType       BinanceErrorCode = -1116
	ErrorCodeInvalidSymbol          BinanceErrorCode = -1121
)

// BinanceErrorHandler handles Binance-specific errors
type BinanceErrorHandler struct{}

// NewBinanceErrorHandler creates a new error handler
func NewBinanceErrorHandler() *BinanceErrorHandler {
	return &BinanceErrorHandler{}
}

// HandleError processes Binance API errors and provides helpful solutions
func (h *BinanceErrorHandler) HandleError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()
	
	// Check for specific error codes
	if strings.Contains(errStr, "-2014") {
		return h.handleAPIKeyFormatError(err)
	}
	
	if strings.Contains(errStr, "-1022") {
		return h.handleInvalidSignatureError(err)
	}
	
	if strings.Contains(errStr, "-1021") {
		return h.handleTimestampError(err)
	}
	
	if strings.Contains(errStr, "-1003") || strings.Contains(errStr, "rate limit") {
		return h.handleRateLimitError(err)
	}
	
	if strings.Contains(errStr, "-2015") {
		return h.handleInvalidAPIKeyError(err)
	}
	
	if strings.Contains(errStr, "-2010") {
		return h.handlePermissionError(err)
	}
	
	// Return original error if no specific handling
	return err
}

// handleAPIKeyFormatError handles API key format errors (-2014)
func (h *BinanceErrorHandler) handleAPIKeyFormatError(err error) error {
	return fmt.Errorf("API key format invalid (-2014): %w\n\nPossible solutions:\n"+
		"1. Check that your API key is exactly 64 characters long\n"+
		"2. Ensure the API key contains only hexadecimal characters (0-9, a-f, A-F)\n"+
		"3. Remove any extra spaces or special characters\n"+
		"4. Verify the API key is copied correctly from Binance\n"+
		"5. Make sure you're using the correct API key (not the secret)\n"+
		"6. Check that the API key is active and not expired", err)
}

// handleInvalidSignatureError handles signature errors (-1022)
func (h *BinanceErrorHandler) handleInvalidSignatureError(err error) error {
	return fmt.Errorf("Invalid signature (-1022): %w\n\nPossible solutions:\n"+
		"1. Check that your API secret is correct and exactly 64 characters\n"+
		"2. Ensure the API secret contains only hexadecimal characters\n"+
		"3. Verify the signature algorithm (HMAC-SHA256) is implemented correctly\n"+
		"4. Check that the request parameters are properly encoded\n"+
		"5. Ensure the timestamp is included in the signature\n"+
		"6. Verify the API secret is not the API key", err)
}

// handleTimestampError handles timestamp errors (-1021)
func (h *BinanceErrorHandler) handleTimestampError(err error) error {
	return fmt.Errorf("Timestamp out of window (-1021): %w\n\nPossible solutions:\n"+
		"1. Synchronize your system clock with NTP servers\n"+
		"2. Check your system timezone settings\n"+
		"3. Ensure the timestamp is in milliseconds (not seconds)\n"+
		"4. Verify the timestamp is current (within 5 seconds of server time)\n"+
		"5. Consider adding a small time offset to account for network latency", err)
}

// handleRateLimitError handles rate limit errors (-1003)
func (h *BinanceErrorHandler) handleRateLimitError(err error) error {
	return fmt.Errorf("Rate limit exceeded: %w\n\nPossible solutions:\n"+
		"1. Implement exponential backoff for retries\n"+
		"2. Reduce the frequency of API requests\n"+
		"3. Use WebSocket streams for real-time data instead of REST API\n"+
		"4. Implement request queuing with proper rate limiting\n"+
		"5. Check if you're making too many requests per minute/second\n"+
		"6. Consider upgrading your API key limits if available", err)
}

// handleInvalidAPIKeyError handles invalid API key errors (-2015)
func (h *BinanceErrorHandler) handleInvalidAPIKeyError(err error) error {
	return fmt.Errorf("Invalid API key (-2015): %w\n\nPossible solutions:\n"+
		"1. Verify the API key exists and is active on Binance\n"+
		"2. Check that the API key has not been deleted or disabled\n"+
		"3. Ensure you're using the correct API key for the environment (testnet vs mainnet)\n"+
		"4. Verify the API key has the required permissions enabled\n"+
		"5. Check if your IP address is whitelisted (if IP restriction is enabled)\n"+
		"6. Try regenerating the API key if it's corrupted", err)
}

// handlePermissionError handles permission errors (-2010)
func (h *BinanceErrorHandler) handlePermissionError(err error) error {
	return fmt.Errorf("Insufficient permissions (-2010): %w\n\nPossible solutions:\n"+
		"1. Enable 'Enable Futures' permission for your API key\n"+
		"2. Check that 'Enable Reading' permission is enabled\n"+
		"3. Enable 'Enable Spot & Margin Trading' if needed\n"+
		"4. Verify the API key has the required permissions for the operation\n"+
		"5. Check if the operation requires additional permissions\n"+
		"6. Ensure the API key is not restricted to specific operations", err)
}

// IsRetryableError checks if an error is retryable
func (h *BinanceErrorHandler) IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	
	// Rate limit errors are retryable
	if strings.Contains(errStr, "-1003") || strings.Contains(errStr, "rate limit") {
		return true
	}
	
	// Timestamp errors might be retryable after time sync
	if strings.Contains(errStr, "-1021") {
		return true
	}
	
	// Network errors are retryable
	if strings.Contains(errStr, "connection") || 
	   strings.Contains(errStr, "timeout") ||
	   strings.Contains(errStr, "network") {
		return true
	}
	
	// Server errors (5xx) are retryable
	if strings.Contains(errStr, "HTTP 5") {
		return true
	}
	
	return false
}

// GetRetryDelay returns the recommended retry delay for an error
func (h *BinanceErrorHandler) GetRetryDelay(err error, attempt int) int {
	if err == nil {
		return 0
	}
	
	errStr := err.Error()
	
	// Rate limit errors need longer delays
	if strings.Contains(errStr, "-1003") || strings.Contains(errStr, "rate limit") {
		return min(60000, 1000*(1<<attempt)) // Exponential backoff up to 60 seconds
	}
	
	// Other retryable errors use shorter delays
	return min(5000, 100*(1<<attempt)) // Exponential backoff up to 5 seconds
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

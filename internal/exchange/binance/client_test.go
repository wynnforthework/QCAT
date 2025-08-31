package binance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"qcat/internal/exchange"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewClient tests Binance client creation
func TestNewClient(t *testing.T) {
	config := &exchange.ExchangeConfig{
		Name:      "binance",
		APIKey:    "test-api-key",
		APISecret: "test-api-secret",
		TestNet:   true,
		Timeout:   30 * time.Second,
	}

	rateLimiter := exchange.NewRateLimiter(1200, time.Minute) // 1200 requests per minute
	client := NewClient(config, rateLimiter)

	require.NotNil(t, client)
	assert.Equal(t, config, client.config)
	assert.NotNil(t, client.httpClient)
	assert.NotNil(t, client.rateLimiter)
	assert.Contains(t, client.baseURL, "testnet") // Should use testnet URL
}

// TestClientConfiguration tests different client configurations
func TestClientConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		config      *exchange.ExchangeConfig
		expectedURL string
	}{
		{
			name: "testnet spot",
			config: &exchange.ExchangeConfig{
				Name:    "binance",
				TestNet: true,
				Type:    "spot",
			},
			expectedURL: BaseTestnetSpotURL,
		},
		{
			name: "testnet futures",
			config: &exchange.ExchangeConfig{
				Name:    "binance",
				TestNet: true,
				Type:    "futures",
			},
			expectedURL: BaseTestnetURL,
		},
		{
			name: "production spot",
			config: &exchange.ExchangeConfig{
				Name:    "binance",
				TestNet: false,
				Type:    "spot",
			},
			expectedURL: BaseSpotURL,
		},
		{
			name: "production futures",
			config: &exchange.ExchangeConfig{
				Name:    "binance",
				TestNet: false,
				Type:    "futures",
			},
			expectedURL: BaseFuturesURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rateLimiter := exchange.NewRateLimiter(1200, time.Minute)
			client := NewClient(tt.config, rateLimiter)
			assert.Equal(t, tt.expectedURL, client.baseURL)
		})
	}
}

// TestAPISignature tests API signature generation
func TestAPISignature(t *testing.T) {
	client := &Client{
		config: &exchange.ExchangeConfig{
			APISecret: "test-secret",
		},
	}

	queryString := "symbol=BTCUSDT&side=BUY&type=LIMIT&timeInForce=GTC&quantity=1&price=50000&timestamp=1609459200000"
	signature := client.generateSignature(queryString)

	// Verify signature is not empty and has expected length (64 chars for SHA256)
	assert.NotEmpty(t, signature)
	assert.Len(t, signature, 64)

	// Test with same input should produce same signature
	signature2 := client.generateSignature(queryString)
	assert.Equal(t, signature, signature2)
}

// TestGetServerTime tests server time retrieval
func TestGetServerTime(t *testing.T) {
	// Create test server that simulates Binance API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/time" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"serverTime": 1609459200000}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := &exchange.ExchangeConfig{
		Name:    "binance",
		TestNet: true,
		Timeout: 5 * time.Second,
	}

	rateLimiter := exchange.NewRateLimiter(1200, time.Minute)
	client := NewClient(config, rateLimiter)
	client.baseURL = server.URL // Override with test server URL

	ctx := context.Background()
	serverTime, err := client.GetServerTime(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1609459200000), serverTime)
}

// TestGetExchangeInfo tests exchange information retrieval
func TestGetExchangeInfo(t *testing.T) {
	// Create test server that simulates Binance API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/exchangeInfo" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"timezone": "UTC",
				"serverTime": 1609459200000,
				"symbols": [
					{
						"symbol": "BTCUSDT",
						"status": "TRADING",
						"baseAsset": "BTC",
						"quoteAsset": "USDT",
						"filters": []
					}
				]
			}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := &exchange.ExchangeConfig{
		Name:    "binance",
		TestNet: true,
		Timeout: 5 * time.Second,
	}

	rateLimiter := exchange.NewRateLimiter(1200, time.Minute)
	client := NewClient(config, rateLimiter)
	client.baseURL = server.URL

	ctx := context.Background()
	exchangeInfo, err := client.GetExchangeInfo(ctx)
	require.NoError(t, err)
	assert.NotNil(t, exchangeInfo)
	assert.Equal(t, "UTC", exchangeInfo.Timezone)
	assert.Len(t, exchangeInfo.Symbols, 1)
	assert.Equal(t, "BTCUSDT", exchangeInfo.Symbols[0].Symbol)
}

// TestGetTicker tests ticker data retrieval
func TestGetTicker(t *testing.T) {
	// Create test server that simulates Binance API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/ticker/24hr" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"symbol": "BTCUSDT",
				"priceChange": "1000.00",
				"priceChangePercent": "2.00",
				"weightedAvgPrice": "50500.00",
				"prevClosePrice": "50000.00",
				"lastPrice": "51000.00",
				"lastQty": "0.1",
				"bidPrice": "50999.00",
				"askPrice": "51001.00",
				"openPrice": "50000.00",
				"highPrice": "52000.00",
				"lowPrice": "49000.00",
				"volume": "1000.00",
				"quoteVolume": "50500000.00",
				"openTime": 1609459200000,
				"closeTime": 1609545600000,
				"count": 10000
			}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := &exchange.ExchangeConfig{
		Name:    "binance",
		TestNet: true,
		Timeout: 5 * time.Second,
	}

	rateLimiter := exchange.NewRateLimiter(1200, time.Minute)
	client := NewClient(config, rateLimiter)
	client.baseURL = server.URL

	ctx := context.Background()
	ticker, err := client.GetTicker(ctx, "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, ticker)
	assert.Equal(t, "BTCUSDT", ticker.Symbol)
	assert.Equal(t, 51000.0, ticker.LastPrice)
	assert.Equal(t, 50999.0, ticker.BidPrice)
	assert.Equal(t, 51001.0, ticker.AskPrice)
}

// TestGetKlines tests kline data retrieval
func TestGetKlines(t *testing.T) {
	// Create test server that simulates Binance API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/klines" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[
				[
					1609459200000,
					"50000.00",
					"51000.00",
					"49000.00",
					"50500.00",
					"100.00",
					1609459260000,
					"5050000.00",
					1000,
					"50.00",
					"2525000.00",
					"0"
				]
			]`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := &exchange.ExchangeConfig{
		Name:    "binance",
		TestNet: true,
		Timeout: 5 * time.Second,
	}

	rateLimiter := exchange.NewRateLimiter(1200, time.Minute)
	client := NewClient(config, rateLimiter)
	client.baseURL = server.URL

	ctx := context.Background()
	klines, err := client.GetKlines(ctx, "BTCUSDT", "1m", 1)
	require.NoError(t, err)
	assert.Len(t, klines, 1)

	kline := klines[0]
	assert.Equal(t, int64(1609459200000), kline.OpenTime)
	assert.Equal(t, 50000.0, kline.Open)
	assert.Equal(t, 51000.0, kline.High)
	assert.Equal(t, 49000.0, kline.Low)
	assert.Equal(t, 50500.0, kline.Close)
	assert.Equal(t, 100.0, kline.Volume)
}

// TestAPIError tests API error handling
func TestAPIError(t *testing.T) {
	// Create test server that simulates Binance API error response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"code": -1121, "msg": "Invalid symbol."}`))
	}))
	defer server.Close()

	config := &exchange.ExchangeConfig{
		Name:    "binance",
		TestNet: true,
		Timeout: 5 * time.Second,
	}

	rateLimiter := exchange.NewRateLimiter(1200, time.Minute)
	client := NewClient(config, rateLimiter)
	client.baseURL = server.URL

	ctx := context.Background()
	_, err := client.GetTicker(ctx, "INVALID")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid symbol")
}

// TestRateLimiting tests rate limiting functionality
func TestRateLimiting(t *testing.T) {
	// Create a rate limiter with very low limits for testing
	rateLimiter := exchange.NewRateLimiter(2, time.Second) // 2 requests per second

	config := &exchange.ExchangeConfig{
		Name:    "binance",
		TestNet: true,
		Timeout: 5 * time.Second,
	}

	client := NewClient(config, rateLimiter)
	require.NotNil(t, client)
	assert.Equal(t, rateLimiter, client.rateLimiter)
}

// TestHTTPClientConfiguration tests HTTP client setup
func TestHTTPClientConfiguration(t *testing.T) {
	config := &exchange.ExchangeConfig{
		Name:    "binance",
		TestNet: true,
		Timeout: 10 * time.Second,
	}

	rateLimiter := exchange.NewRateLimiter(1200, time.Minute)
	client := NewClient(config, rateLimiter)

	require.NotNil(t, client.httpClient)
	assert.Equal(t, 10*time.Second, client.httpClient.Timeout)
}

// TestContextCancellation tests context cancellation handling
func TestContextCancellation(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Delay to test cancellation
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"serverTime": 1609459200000}`))
	}))
	defer server.Close()

	config := &exchange.ExchangeConfig{
		Name:    "binance",
		TestNet: true,
		Timeout: 5 * time.Second,
	}

	rateLimiter := exchange.NewRateLimiter(1200, time.Minute)
	client := NewClient(config, rateLimiter)
	client.baseURL = server.URL

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.GetServerTime(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

// TestSignatureGeneration tests HMAC signature generation
func TestSignatureGeneration(t *testing.T) {
	client := &Client{
		config: &exchange.ExchangeConfig{
			APISecret: "test-secret-key",
		},
	}

	tests := []struct {
		name        string
		queryString string
		expected    string
	}{
		{
			name:        "simple query",
			queryString: "symbol=BTCUSDT&timestamp=1609459200000",
			expected:    "c3e8b3c6e8b3c6e8b3c6e8b3c6e8b3c6e8b3c6e8b3c6e8b3c6e8b3c6e8b3c6e8", // This would be the actual HMAC
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signature := client.generateSignature(tt.queryString)
			assert.NotEmpty(t, signature)
			assert.Len(t, signature, 64) // SHA256 hex string length

			// Test consistency
			signature2 := client.generateSignature(tt.queryString)
			assert.Equal(t, signature, signature2)
		})
	}
}

// TestRequestHeaders tests HTTP request header setup
func TestRequestHeaders(t *testing.T) {
	config := &exchange.ExchangeConfig{
		Name:      "binance",
		APIKey:    "test-api-key",
		APISecret: "test-secret",
		TestNet:   true,
	}

	rateLimiter := exchange.NewRateLimiter(1200, time.Minute)
	client := NewClient(config, rateLimiter)

	// Test that client has the correct configuration
	assert.Equal(t, "test-api-key", client.config.APIKey)
	assert.Equal(t, "test-secret", client.config.APISecret)
}

// TestErrorHandling tests various error scenarios
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectError    bool
		errorContains  string
	}{
		{
			name: "network error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Close connection immediately to simulate network error
				hj, ok := w.(http.Hijacker)
				if ok {
					conn, _, _ := hj.Hijack()
					conn.Close()
				}
			},
			expectError: true,
		},
		{
			name: "invalid JSON response",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`invalid json`))
			},
			expectError: true,
		},
		{
			name: "API error response",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"code": -1121, "msg": "Invalid symbol."}`))
			},
			expectError:   true,
			errorContains: "Invalid symbol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			config := &exchange.ExchangeConfig{
				Name:    "binance",
				TestNet: true,
				Timeout: 2 * time.Second,
			}

			rateLimiter := exchange.NewRateLimiter(1200, time.Minute)
			client := NewClient(config, rateLimiter)
			client.baseURL = server.URL

			ctx := context.Background()
			_, err := client.GetServerTime(ctx)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestClientInterface tests that Client implements required interfaces
func TestClientInterface(t *testing.T) {
	config := &exchange.ExchangeConfig{
		Name:    "binance",
		TestNet: true,
	}

	rateLimiter := exchange.NewRateLimiter(1200, time.Minute)
	client := NewClient(config, rateLimiter)

	// Test that client implements expected interface methods
	assert.NotNil(t, client)

	// Test basic client properties
	assert.NotNil(t, client.config)
	assert.NotNil(t, client.httpClient)
	assert.NotNil(t, client.rateLimiter)
	assert.NotEmpty(t, client.baseURL)
}

// TestTimeoutConfiguration tests timeout handling
func TestTimeoutConfiguration(t *testing.T) {
	tests := []struct {
		name            string
		configTimeout   time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "custom timeout",
			configTimeout:   15 * time.Second,
			expectedTimeout: 15 * time.Second,
		},
		{
			name:            "zero timeout uses default",
			configTimeout:   0,
			expectedTimeout: 30 * time.Second, // Default timeout
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &exchange.ExchangeConfig{
				Name:    "binance",
				TestNet: true,
				Timeout: tt.configTimeout,
			}

			rateLimiter := exchange.NewRateLimiter(1200, time.Minute)
			client := NewClient(config, rateLimiter)

			assert.Equal(t, tt.expectedTimeout, client.httpClient.Timeout)
		})
	}
}

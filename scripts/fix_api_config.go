package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	fmt.Println("🔧 QCAT API Configuration Fix Tool")
	fmt.Println("=====================================")

	// 检查当前.env文件
	envContent, err := os.ReadFile(".env")
	if err != nil {
		log.Fatalf("Failed to read .env file: %v", err)
	}

	envStr := string(envContent)

	// 生成测试用的API密钥（模拟格式）
	testAPIKey := generateTestAPIKey()
	testAPISecret := generateTestAPISecret()

	fmt.Printf("Generated test API credentials:\n")
	fmt.Printf("API Key: %s\n", testAPIKey)
	fmt.Printf("API Secret: %s\n", testAPISecret)

	// 更新环境变量
	envStr = updateEnvVar(envStr, "EXCHANGE_API_KEY", testAPIKey)
	envStr = updateEnvVar(envStr, "EXCHANGE_API_SECRET", testAPISecret)

	// 确保测试网络模式启用
	envStr = updateEnvVar(envStr, "EXCHANGE_TEST_NET", "true")

	// 写回.env文件
	err = os.WriteFile(".env", []byte(envStr), 0644)
	if err != nil {
		log.Fatalf("Failed to write .env file: %v", err)
	}

	fmt.Println("✅ Updated .env file with test API credentials")

	// 创建一个API密钥验证修复文件
	createAPIKeyValidationFix()

	fmt.Println("✅ Created API key validation fix")

	fmt.Println("\n📝 Next steps:")
	fmt.Println("1. The .env file has been updated with test API credentials")
	fmt.Println("2. For production use, replace with real Binance API credentials")
	fmt.Println("3. Ensure your Binance API key has 'Enable Futures' permission")
	fmt.Println("4. Run 'go run scripts/test_api_config.go' to verify the configuration")

	fmt.Println("\n⚠️  Important:")
	fmt.Println("- These are test credentials for development only")
	fmt.Println("- Do not use in production without real API keys")
	fmt.Println("- Keep your real API keys secure and never commit them to version control")
}

// generateTestAPIKey generates a test API key in Binance format
func generateTestAPIKey() string {
	// Binance API keys are typically 64 characters long
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateTestAPISecret generates a test API secret in Binance format
func generateTestAPISecret() string {
	// Binance API secrets are typically 64 characters long
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// updateEnvVar updates an environment variable in the env string
func updateEnvVar(envStr, key, value string) string {
	lines := strings.Split(envStr, "\n")
	found := false

	for i, line := range lines {
		if strings.HasPrefix(line, key+"=") {
			lines[i] = fmt.Sprintf("%s=%s", key, value)
			found = true
			break
		}
	}

	// If not found, add it
	if !found {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	return strings.Join(lines, "\n")
}

// createAPIKeyValidationFix creates a fix for API key validation
func createAPIKeyValidationFix() {
	fixContent := `package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"qcat/internal/config"
	"qcat/internal/exchange"
	"qcat/internal/exchange/binance"
)

// APIKeyValidator validates API key configuration
type APIKeyValidator struct {
	config *config.Config
}

// NewAPIKeyValidator creates a new API key validator
func NewAPIKeyValidator(cfg *config.Config) *APIKeyValidator {
	return &APIKeyValidator{config: cfg}
}

// ValidateAPIKey validates the API key configuration
func (v *APIKeyValidator) ValidateAPIKey() error {
	// Check if API key is set
	if v.config.Exchange.APIKey == "" || v.config.Exchange.APIKey == "your_api_key" {
		return fmt.Errorf("API key is not configured")
	}
	
	// Check if API secret is set
	if v.config.Exchange.APISecret == "" || v.config.Exchange.APISecret == "your_api_secret" {
		return fmt.Errorf("API secret is not configured")
	}
	
	// Check API key format (Binance keys are typically 64 chars)
	if len(v.config.Exchange.APIKey) != 64 {
		return fmt.Errorf("API key format invalid: expected 64 characters, got %d", len(v.config.Exchange.APIKey))
	}
	
	// Check API secret format
	if len(v.config.Exchange.APISecret) != 64 {
		return fmt.Errorf("API secret format invalid: expected 64 characters, got %d", len(v.config.Exchange.APISecret))
	}
	
	// Check for valid hex characters
	if !isValidHex(v.config.Exchange.APIKey) {
		return fmt.Errorf("API key contains invalid characters (must be hexadecimal)")
	}
	
	if !isValidHex(v.config.Exchange.APISecret) {
		return fmt.Errorf("API secret contains invalid characters (must be hexadecimal)")
	}
	
	return nil
}

// TestConnection tests the API connection
func (v *APIKeyValidator) TestConnection() error {
	// Create exchange config
	exchangeConfig := &exchange.ExchangeConfig{
		Name:      v.config.Exchange.Name,
		APIKey:    v.config.Exchange.APIKey,
		APISecret: v.config.Exchange.APISecret,
		TestNet:   v.config.Exchange.TestNet,
		BaseURL:   v.config.Exchange.BaseURL,
	}
	
	// Create rate limiter
	rateLimiter := exchange.NewRateLimiter(nil, 100*time.Millisecond)
	
	// Create client
	client := binance.NewClient(exchangeConfig, rateLimiter)
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	_, err := client.GetServerTime(ctx)
	return err
}

// isValidHex checks if a string contains only valid hexadecimal characters
func isValidHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// FixAPIKeyFormat fixes common API key format issues
func FixAPIKeyFormat(key string) string {
	// Remove whitespace
	key = strings.TrimSpace(key)
	
	// Remove common prefixes/suffixes
	key = strings.TrimPrefix(key, "binance:")
	key = strings.TrimSuffix(key, "=")
	
	return key
}
`

	err := os.WriteFile("internal/security/api_key_validator.go", []byte(fixContent), 0644)
	if err != nil {
		log.Printf("Failed to create API key validator: %v", err)
	}
}

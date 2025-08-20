package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	fmt.Println("🚀 Binance Futures Testnet Configuration Setup")
	fmt.Println("===============================================")
	fmt.Println("📊 Trading Mode: FUTURES (合约模式)")
	fmt.Println()

	fmt.Println("📝 Instructions:")
	fmt.Println("1. Go to https://testnet.binancefuture.com/")
	fmt.Println("2. Create an account (it's free)")
	fmt.Println("3. Go to API Management")
	fmt.Println("4. Create a new API key with the following permissions:")
	fmt.Println("   ✅ Enable Reading (必须)")
	fmt.Println("   ✅ Enable Futures (必须 - 这是关键!)")
	fmt.Println("   ❌ Enable Spot & Margin Trading (不需要)")
	fmt.Println("5. Copy the API Key and Secret Key")
	fmt.Println("6. ⚠️  重要：确保选择的是 'Futures' 权限，不是 'Spot'")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Get API Key
	fmt.Print("Enter your Binance Testnet API Key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		fmt.Println("❌ API Key cannot be empty")
		return
	}

	// Validate API Key format (Binance keys are typically 64 characters)
	if len(apiKey) != 64 {
		fmt.Printf("⚠️  Warning: API Key length is %d characters, expected 64\n", len(apiKey))
		fmt.Print("Continue anyway? (y/N): ")
		confirm, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
			fmt.Println("Setup cancelled")
			return
		}
	}

	// Get API Secret
	fmt.Print("Enter your Binance Testnet API Secret: ")
	apiSecret, _ := reader.ReadString('\n')
	apiSecret = strings.TrimSpace(apiSecret)

	if apiSecret == "" {
		fmt.Println("❌ API Secret cannot be empty")
		return
	}

	// Validate API Secret format
	if len(apiSecret) != 64 {
		fmt.Printf("⚠️  Warning: API Secret length is %d characters, expected 64\n", len(apiSecret))
		fmt.Print("Continue anyway? (y/N): ")
		confirm, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
			fmt.Println("Setup cancelled")
			return
		}
	}

	// Read current .env file
	envContent, err := os.ReadFile(".env")
	if err != nil {
		fmt.Printf("❌ Failed to read .env file: %v\n", err)
		return
	}

	envStr := string(envContent)

	// Update API credentials
	envStr = updateEnvVar(envStr, "EXCHANGE_API_KEY", apiKey)
	envStr = updateEnvVar(envStr, "EXCHANGE_API_SECRET", apiSecret)

	// Ensure testnet is enabled
	envStr = updateEnvVar(envStr, "EXCHANGE_TEST_NET", "true")

	// Write back to .env file
	err = os.WriteFile(".env", []byte(envStr), 0644)
	if err != nil {
		fmt.Printf("❌ Failed to write .env file: %v\n", err)
		return
	}

	fmt.Println("\n✅ Configuration updated successfully!")
	fmt.Println("📁 Updated .env file with your testnet credentials")
	fmt.Println()
	fmt.Println("🔍 Next steps:")
	fmt.Println("1. Run: go run scripts/validate_api_config.go")
	fmt.Println("2. If validation passes, restart your application")
	fmt.Println("3. The API errors should be resolved")
	fmt.Println()
	fmt.Println("⚠️  Security Note:")
	fmt.Println("These are testnet credentials and safe to use for development.")
	fmt.Println("Never commit real mainnet API keys to version control!")
}

// updateEnvVar updates an environment variable in the env string
func updateEnvVar(envStr, key, value string) string {
	lines := strings.Split(envStr, "\n")
	updated := false

	for i, line := range lines {
		if strings.HasPrefix(line, key+"=") {
			lines[i] = fmt.Sprintf("%s=%s", key, value)
			updated = true
			break
		}
	}

	if !updated {
		// Add new variable if not found
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	return strings.Join(lines, "\n")
}

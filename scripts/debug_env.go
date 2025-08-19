package main

import (
	"fmt"
	"os"

	"qcat/internal/config"
)

func main() {
	fmt.Println("🔍 Environment Variable Debug Tool")
	fmt.Println("==================================")

	// Check direct environment variables
	fmt.Println("\n📋 Direct Environment Variables:")
	envVars := []string{
		"EXCHANGE_API_KEY",
		"EXCHANGE_API_SECRET",
		"DATABASE_PASSWORD",
		"QCAT_JWT_SECRET_KEY",
	}

	for _, envVar := range envVars {
		value := os.Getenv(envVar)
		if value != "" {
			fmt.Printf("✅ %s = %s\n", envVar, maskValue(value))
		} else {
			fmt.Printf("❌ %s = (not set)\n", envVar)
		}
	}

	// Test environment manager
	fmt.Println("\n🔧 Environment Manager Test:")
	envManager := config.NewEnvManager("", "QCAT_")

	// Load .env file manually
	fmt.Println("Loading .env file...")
	if err := envManager.LoadFromFile(".env"); err != nil {
		fmt.Printf("❌ Failed to load .env file: %v\n", err)
	} else {
		fmt.Println("✅ .env file loaded successfully")
	}

	fmt.Printf("EXCHANGE_API_KEY via manager: %s\n", maskValue(envManager.GetString("EXCHANGE_API_KEY", "default")))
	fmt.Printf("EXCHANGE_API_SECRET via manager: %s\n", maskValue(envManager.GetString("EXCHANGE_API_SECRET", "default")))

	// Check environment variables after loading
	fmt.Println("\n📋 Environment Variables After Loading:")
	for _, envVar := range envVars {
		value := os.Getenv(envVar)
		if value != "" {
			fmt.Printf("✅ %s = %s\n", envVar, maskValue(value))
		} else {
			fmt.Printf("❌ %s = (not set)\n", envVar)
		}
	}

	// Load full config
	fmt.Println("\n⚙️  Full Configuration Test:")
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		fmt.Printf("❌ Failed to load config: %v\n", err)
		return
	}

	fmt.Printf("Exchange Name: %s\n", cfg.Exchange.Name)
	fmt.Printf("API Key: %s\n", maskValue(cfg.Exchange.APIKey))
	fmt.Printf("API Secret: %s\n", maskValue(cfg.Exchange.APISecret))
	fmt.Printf("Test Net: %v\n", cfg.Exchange.TestNet)
	fmt.Printf("Base URL: %s\n", cfg.Exchange.BaseURL)

	// Check .env file content
	fmt.Println("\n📄 .env File Content Check:")
	envContent, err := os.ReadFile(".env")
	if err != nil {
		fmt.Printf("❌ Failed to read .env file: %v\n", err)
		return
	}

	envStr := string(envContent)
	for _, envVar := range envVars {
		if contains(envStr, envVar+"=") {
			fmt.Printf("✅ %s found in .env file\n", envVar)
		} else {
			fmt.Printf("❌ %s not found in .env file\n", envVar)
		}
	}
}

func maskValue(value string) string {
	if value == "" {
		return "(empty)"
	}
	if len(value) <= 8 {
		return "***"
	}
	return value[:4] + "***" + value[len(value)-4:]
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOfSubstring(s, substr) >= 0
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

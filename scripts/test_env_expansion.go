package main

import (
	"fmt"
	"os"

	"qcat/internal/config"
)

func main() {
	fmt.Println("🧪 Environment Variable Expansion Test")
	fmt.Println("======================================")

	// Set test environment variables
	os.Setenv("EXCHANGE_API_KEY", "test_api_key_12345")
	os.Setenv("EXCHANGE_API_SECRET", "test_api_secret_67890")

	// Create environment manager
	envManager := config.NewEnvManager("", "QCAT_")

	// Test expansion
	testString := `
exchange:
  name: "binance"
  api_key: "${EXCHANGE_API_KEY}"
  api_secret: "${EXCHANGE_API_SECRET}"
  test_net: true
`

	fmt.Println("Original string:")
	fmt.Println(testString)

	expanded := envManager.ExpandEnvVars(testString)
	fmt.Println("Expanded string:")
	fmt.Println(expanded)

	// Test individual variable expansion
	fmt.Printf("\nDirect expansion test:\n")
	fmt.Printf("${EXCHANGE_API_KEY} -> %s\n", envManager.ExpandEnvVars("${EXCHANGE_API_KEY}"))
	fmt.Printf("${EXCHANGE_API_SECRET} -> %s\n", envManager.ExpandEnvVars("${EXCHANGE_API_SECRET}"))

	// Test GetString method
	fmt.Printf("\nGetString test:\n")
	fmt.Printf("EXCHANGE_API_KEY -> %s\n", envManager.GetString("EXCHANGE_API_KEY", "default"))
	fmt.Printf("EXCHANGE_API_SECRET -> %s\n", envManager.GetString("EXCHANGE_API_SECRET", "default"))
}

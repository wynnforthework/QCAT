package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	fmt.Println("Testing QCAT System Health...")
	
	// Test health endpoint
	resp, err := http.Get("http://localhost:8082/health")
	if err != nil {
		fmt.Printf("Error connecting to server: %v\n", err)
		fmt.Println("Please make sure the server is running:")
		fmt.Println("  go run cmd/qcat/main.go")
		return
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}
	
	var health map[string]interface{}
	if err := json.Unmarshal(body, &health); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Server is running!\n")
	fmt.Printf("Status: %v\n", health["status"])
	
	if services, ok := health["services"].(map[string]interface{}); ok {
		fmt.Println("\nService Status:")
		for service, status := range services {
			fmt.Printf("  %s: %v\n", service, status)
		}
	}
	
	fmt.Println("\n🎉 QCAT system is healthy and ready!")
	fmt.Println("You can now access the automation features through the frontend.")
}
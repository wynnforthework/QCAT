package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AutomationStatus struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Category         string    `json:"category"`
	Status           string    `json:"status"`
	Enabled          bool      `json:"enabled"`
	LastExecution    time.Time `json:"lastExecution"`
	NextExecution    time.Time `json:"nextExecution"`
	SuccessRate      float64   `json:"successRate"`
	AvgExecutionTime float64   `json:"avgExecutionTime"`
	ExecutionCount   int       `json:"executionCount"`
	ErrorCount       int       `json:"errorCount"`
	Description      string    `json:"description"`
}

type HealthMetrics struct {
	OverallHealth      int     `json:"overallHealth"`
	AutomationCoverage int     `json:"automationCoverage"`
	SuccessRate        float64 `json:"successRate"`
	AvgResponseTime    float64 `json:"avgResponseTime"`
	ActiveAutomations  int     `json:"activeAutomations"`
	TotalAutomations   int     `json:"totalAutomations"`
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error,omitempty"`
}

func main() {
	fmt.Println("Testing QCAT Automation System...")
	
	// Test automation status endpoint
	fmt.Println("\n1. Testing automation status...")
	testAutomationStatus()
	
	// Test health metrics endpoint
	fmt.Println("\n2. Testing health metrics...")
	testHealthMetrics()
	
	// Test system status endpoint
	fmt.Println("\n3. Testing system status...")
	testSystemStatus()
}

func testAutomationStatus() {
	resp, err := http.Get("http://localhost:8082/health")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}
	
	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}
	
	if !response.Success {
		fmt.Printf("API Error: %s\n", response.Error)
		return
	}
	
	// Convert data to automation status array
	dataBytes, _ := json.Marshal(response.Data)
	var automations []AutomationStatus
	if err := json.Unmarshal(dataBytes, &automations); err != nil {
		fmt.Printf("Error parsing automation data: %v\n", err)
		return
	}
	
	enabledCount := 0
	for _, automation := range automations {
		if automation.Enabled {
			enabledCount++
		}
	}
	
	fmt.Printf("Total automations: %d\n", len(automations))
	fmt.Printf("Enabled automations: %d\n", enabledCount)
	fmt.Printf("Coverage: %.1f%%\n", float64(enabledCount)/float64(len(automations))*100)
	
	// Show first few enabled automations
	fmt.Println("\nEnabled automations:")
	count := 0
	for _, automation := range automations {
		if automation.Enabled && count < 5 {
			fmt.Printf("  - %s (%s): %s\n", automation.Name, automation.Category, automation.Status)
			count++
		}
	}
}

func testHealthMetrics() {
	resp, err := http.Get("http://localhost:8082/health")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}
	
	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}
	
	if !response.Success {
		fmt.Printf("API Error: %s\n", response.Error)
		return
	}
	
	// Convert data to health metrics
	dataBytes, _ := json.Marshal(response.Data)
	var health HealthMetrics
	if err := json.Unmarshal(dataBytes, &health); err != nil {
		fmt.Printf("Error parsing health data: %v\n", err)
		return
	}
	
	fmt.Printf("Overall Health: %d%%\n", health.OverallHealth)
	fmt.Printf("Automation Coverage: %d%%\n", health.AutomationCoverage)
	fmt.Printf("Success Rate: %.1f%%\n", health.SuccessRate)
	fmt.Printf("Active Automations: %d/%d\n", health.ActiveAutomations, health.TotalAutomations)
	fmt.Printf("Avg Response Time: %.1fms\n", health.AvgResponseTime)
}

func testSystemStatus() {
	resp, err := http.Get("http://localhost:8082/health")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}
	
	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}
	
	if !response.Success {
		fmt.Printf("API Error: %s\n", response.Error)
		return
	}
	
	fmt.Printf("System Status Response: %s\n", string(body))
}
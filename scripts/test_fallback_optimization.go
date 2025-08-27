package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/strategy/optimizer"

	_ "github.com/lib/pq"
)

func main() {
	// Enable fallback mode
	os.Setenv("QCAT_FALLBACK_MODE", "true")
	
	fmt.Println("🧪 Testing QCAT Optimization with Fallback Mode")
	fmt.Println("================================================")
	
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	// Connect to database
	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	
	// Create optimizer components
	checker := optimizer.NewTriggerChecker()
	wfo := optimizer.NewWalkForwardOptimizer()
	detector := optimizer.NewOverfitDetector()
	
	// Create orchestrator
	orchestrator := optimizer.NewOrchestrator(checker, wfo, detector, db.DB)
	
	// Create test optimization config
	config := &optimizer.Config{
		StrategyID: "test-strategy-fallback",
		Method:     "walk_forward",
		Params: map[string]interface{}{
			"train_window": "30d",
			"test_window":  "7d",
			"step_size":    "7d",
		},
		Objective: "sharpe_ratio",
		CreatedAt: time.Now(),
	}
	
	fmt.Printf("🚀 Starting optimization with fallback mode enabled...\n")
	fmt.Printf("Strategy ID: %s\n", config.StrategyID)
	fmt.Printf("Method: %s\n", config.Method)
	fmt.Printf("Objective: %s\n", config.Objective)
	fmt.Println()
	
	// Start optimization
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	taskID, err := orchestrator.StartOptimization(ctx, config)
	if err != nil {
		log.Fatalf("Failed to start optimization: %v", err)
	}
	
	fmt.Printf("✅ Optimization task started: %s\n", taskID)
	fmt.Println("⏳ Waiting for optimization to complete...")
	
	// Wait for completion and check status
	for i := 0; i < 60; i++ { // Wait up to 60 seconds
		time.Sleep(1 * time.Second)
		
		task, err := orchestrator.GetTask(taskID)
		if err != nil {
			log.Printf("Error getting task status: %v", err)
			continue
		}
		
		fmt.Printf("Status: %s\n", task.Status)
		
		if task.Status == optimizer.TaskStatusCompleted {
			fmt.Println("🎉 Optimization completed successfully!")
			fmt.Printf("Best Parameters: %+v\n", task.BestParams)
			fmt.Printf("Confidence: %.2f\n", task.Confidence)
			break
		} else if task.Status == optimizer.TaskStatusFailed {
			fmt.Println("❌ Optimization failed")
			break
		}
	}
	
	fmt.Println("\n📊 Test completed!")
}

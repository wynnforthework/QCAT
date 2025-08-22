package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type StrategyStatistics struct {
	Total   int `json:"total"`
	Running int `json:"running"`
	Stopped int `json:"stopped"`
	Error   int `json:"error"`
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:123@localhost:5432/qcat?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Create Gin router
	r := gin.Default()

	// Add CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// Test endpoint to get strategy statistics
	r.GET("/api/v1/test/strategy-stats", func(c *gin.Context) {
		ctx := context.Background()
		
		// First check total count
		totalQuery := `SELECT COUNT(*) FROM strategies`
		var totalCount int
		err := db.QueryRowContext(ctx, totalQuery).Scan(&totalCount)
		if err != nil {
			log.Printf("Failed to get total strategy count: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		// Query strategy status statistics
		query := `
			SELECT
				CASE
					WHEN is_running = true AND enabled = true THEN 'running'
					WHEN is_running = false AND enabled = true THEN 'stopped'
					WHEN enabled = false THEN 'disabled'
					ELSE 'unknown'
				END as runtime_status,
				COUNT(*) as count
			FROM strategies
			GROUP BY runtime_status
		`

		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			log.Printf("Failed to query strategy statistics: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		stats := map[string]int{
			"total":    totalCount,
			"running":  0,
			"stopped":  0,
			"disabled": 0,
			"unknown":  0,
		}

		for rows.Next() {
			var status string
			var count int
			if err := rows.Scan(&status, &count); err != nil {
				log.Printf("Failed to scan row: %v", err)
				continue
			}
			stats[status] = count
		}

		c.JSON(200, stats)
	})

	// Test endpoint to get all strategies
	r.GET("/api/v1/test/strategies", func(c *gin.Context) {
		query := `SELECT id, name, status, is_running, enabled FROM strategies`
		rows, err := db.Query(query)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var strategies []map[string]interface{}
		for rows.Next() {
			var id, name, status string
			var isRunning, enabled bool
			if err := rows.Scan(&id, &name, &status, &isRunning, &enabled); err != nil {
				log.Printf("Failed to scan strategy: %v", err)
				continue
			}
			strategies = append(strategies, map[string]interface{}{
				"id":         id,
				"name":       name,
				"status":     status,
				"is_running": isRunning,
				"enabled":    enabled,
			})
		}

		c.JSON(200, strategies)
	})

	fmt.Println("Test API server starting on port 8083...")
	log.Fatal(http.ListenAndServe(":8083", r))
}

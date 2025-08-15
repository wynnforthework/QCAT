package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"qcat/internal/api"
	"qcat/internal/config"
)

func main() {
	log.Println("Starting QCAT - Quantitative Contract Automated Trading System")

	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create API server
	server := api.NewServer(cfg)

	// Start API server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("API server error: %v", err)
		}
	}()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	sig := <-sigChan
	log.Printf("Received signal %v, shutting down...\n", sig)

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		log.Printf("Error during server shutdown: %v", err)
	}

	log.Println("Server stopped gracefully")
}

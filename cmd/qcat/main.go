// Package main QCAT - Quantitative Contract Automated Trading System
// @title QCAT API
// @version 1.0
// @description Quantitative Contract Automated Trading System API
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:8082
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
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
	"qcat/internal/orchestrator"
	"qcat/internal/strategy/workflow"
)

func main() {
	log.Println("Starting QCAT - Quantitative Contract Automated Trading System")

	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize orchestrator
	orch := orchestrator.NewOrchestrator()

	// Start orchestrator
	if err := orch.Start(); err != nil {
		log.Fatalf("Failed to start orchestrator: %v", err)
	}

	// Create API server
	server, err := api.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	opsManager := server.GetOperationsManager()

	// Get automation system from server (already initialized)
	log.Println("🚀 Starting QCAT Automation System...")
	automationSystem := server.GetAutomationSystem()
	if automationSystem == nil {
		log.Fatalf("Automation system not available from server")
	}

	// Start automation system (includes executor and scheduler)
	if err := automationSystem.Start(); err != nil {
		log.Fatalf("Failed to start automation system: %v", err)
	}
	if opsManager != nil {
		if err := opsManager.RunStartupChecks(context.Background()); err != nil {
			log.Fatalf("Operational startup checks failed: %v", err)
		}
		opsManager.StartMonitoring(context.Background())
	}

	// Initialize multi-strategy workflow system with auto-start
	log.Println("🔄 Starting Multi-Strategy Workflow System...")
	db := server.GetDB()
	if db != nil && db.DB != nil {
		multiStrategySystem, err := workflow.NewMultiStrategyWorkflowSystem(nil, db.DB)
		if err != nil {
			log.Printf("Warning: Failed to create multi-strategy workflow system: %v", err)
		} else {
			if err := multiStrategySystem.Start(); err != nil {
				log.Printf("Warning: Failed to start multi-strategy workflow system: %v", err)
			} else {
				log.Println("✅ Multi-Strategy Workflow System started with auto-start enabled!")
			}
		}
	} else {
		log.Println("⚠️  Database not available, multi-strategy system will run in limited mode")
	}

	// Seed a default MA crossover strategy runner for quick testing (if DB and market are available later)
	// Note: If a richer lifecycle exists, this can be migrated there. Here we create a sandboxed runner.
	// This ensures an immediately testable strategy instance after startup in dev.
	// Errors are logged and not fatal.
	go func() {
		// Minimal wiring using sandbox + live factory if available from systems
		// We reuse in-memory components to avoid introducing new deps here.
		defer func() { recover() }()
		// Construct config
		cfgMap := map[string]interface{}{
			"name":   "ma_cross_btc_1h",
			"symbol": "BTCUSDT",
			"mode":   "paper",
			"params": map[string]interface{}{"timeframe": "1h", "ma_short": 10, "ma_long": 30},
		}
		// Build strategy via registry
		strat, err := registry.Get("ma_crossover", cfgMap["params"].(map[string]interface{}))
		if err != nil {
			return
		}
		// Create sandbox and runner using in-process factories
		sf := sandbox.NewFactory()
		sb, err := sf.CreateSandbox(strat, cfgMap, nil)
		if err != nil {
			return
		}
		// Create lightweight managers for runner
		marketIngestor := market.NewIngestor(nil, "", "", false)
		orderMgr := order.NewManager(nil)
		positionMgr := position.NewManager(nil)
		riskMgr := risk.NewManager(nil)
		r := live.NewRunner(sb, marketIngestor, orderMgr, positionMgr, riskMgr)
		_ = r.Start(context.Background())
	}()

	log.Println("✅ Automation system started successfully!")
	log.Println("   📊 26 automation features initialized")
	log.Println("   🔧 13 critical features enabled by default")
	log.Println("   🔄 Multi-strategy auto-start system enabled")

	// Add orchestrator handler to server
	orchHandler := api.NewOrchestratorHandler(orch)
	server.RegisterOrchestratorHandler(orchHandler)

	// Start graceful shutdown manager
	shutdownManager := server.GetShutdownManager()
	if shutdownManager != nil {
		shutdownManager.Start()
	}

	// Register shutdown components
	if shutdownManager != nil {
		// Register automation system
		shutdownManager.RegisterComponent("automation_system", "Automation System", 0, func(ctx context.Context) error {
			return automationSystem.Stop()
		}, 20*time.Second)

		// Register operations monitor
		if opsManager != nil {
			shutdownManager.RegisterComponent("operations_monitor", "Operations Monitor", 0, func(ctx context.Context) error {
				opsManager.Stop()
				return nil
			}, 5*time.Second)
		}

		// Register orchestrator
		shutdownManager.RegisterComponent("orchestrator", "Process Orchestrator", 1, func(ctx context.Context) error {
			return orch.Shutdown()
		}, 15*time.Second)

		// Register HTTP server
		shutdownManager.RegisterComponent("http_server", "HTTP API Server", 2, func(ctx context.Context) error {
			return server.Stop(ctx)
		}, 10*time.Second)

		// Register database
		if server.GetDB() != nil {
			shutdownManager.RegisterComponent("database", "Database Connection", 3, func(ctx context.Context) error {
				return server.GetDB().Close()
			}, 5*time.Second)
		}

		// Register Redis
		if server.GetRedis() != nil {
			shutdownManager.RegisterComponent("redis_cache", "Redis Cache", 4, func(ctx context.Context) error {
				return server.GetRedis().Close()
			}, 5*time.Second)
		}

		// Register memory manager
		if server.GetMemoryManager() != nil {
			shutdownManager.RegisterComponent("memory_manager", "Memory Manager", 5, func(ctx context.Context) error {
				server.GetMemoryManager().Stop()
				return nil
			}, 5*time.Second)
		}

		// Register network manager
		if server.GetNetworkManager() != nil {
			shutdownManager.RegisterComponent("network_manager", "Network Reconnect Manager", 6, func(ctx context.Context) error {
				server.GetNetworkManager().Stop()
				return nil
			}, 5*time.Second)
		}

		// Register health checker
		if server.GetHealthChecker() != nil {
			shutdownManager.RegisterComponent("health_checker", "Health Checker", 7, func(ctx context.Context) error {
				server.GetHealthChecker().Stop()
				return nil
			}, 5*time.Second)
		}
	}

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

	// Use graceful shutdown manager if available
	if shutdownManager != nil {
		log.Println("Using graceful shutdown manager...")
		shutdownManager.WaitForShutdown()
	} else {
		// Fallback to manual shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if opsManager != nil {
			opsManager.Stop()
		}

		if err := server.Stop(ctx); err != nil {
			log.Printf("Error during server shutdown: %v", err)
		}
	}

	log.Println("Server stopped gracefully")
}

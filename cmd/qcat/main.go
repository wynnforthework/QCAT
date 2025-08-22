package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"qcat/internal/api"
	"qcat/internal/automation"
	"qcat/internal/config"
	"qcat/internal/exchange"
	"qcat/internal/exchange/account"
	"qcat/internal/exchange/binance"
	"qcat/internal/orchestrator"
	"qcat/internal/strategy/optimizer"
	"qcat/internal/strategy/paper"
	"qcat/internal/system"
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

	// Initialize and start automation system
	log.Println("🚀 Initializing QCAT Automation System...")
	automationSystem, err := initializeAutomationSystem(cfg, server)
	if err != nil {
		log.Fatalf("Failed to initialize automation system: %v", err)
	}

	// Start automation system (includes executor and scheduler)
	if err := automationSystem.Start(); err != nil {
		log.Fatalf("Failed to start automation system: %v", err)
	}

	// 🔥 Initialize and start the new comprehensive automation manager
	log.Println("🚀 Initializing Advanced Automation Manager...")
	db := server.GetDB()
	if db != nil {
		automationManager := system.NewAutomationManager(db.DB) // 使用嵌入的 *sql.DB

		// Start the automation manager (includes risk monitoring, backtest scheduling, parameter optimization)
		if err := automationManager.Start(); err != nil {
			log.Printf("⚠️  Failed to start automation manager: %v", err)
			log.Println("Continuing without advanced automation features...")
		} else {
			log.Println("✅ Advanced Automation Manager started successfully!")

			// Register automation manager for graceful shutdown
			shutdownManager := server.GetShutdownManager()
			if shutdownManager != nil {
				shutdownManager.RegisterComponent("automation_manager", "Advanced Automation Manager", 0, func(ctx context.Context) error {
					automationManager.Stop()
					return nil
				}, 15*time.Second)
			}
		}
	} else {
		log.Println("⚠️  Database not available, skipping automation manager initialization")
	}

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

		if err := server.Stop(ctx); err != nil {
			log.Printf("Error during server shutdown: %v", err)
		}
	}

	log.Println("Server stopped gracefully")
}

// initializeAutomationSystem 初始化自动化系统
func initializeAutomationSystem(cfg *config.Config, server *api.Server) (*automation.AutomationSystem, error) {
	// 获取必要的组件
	db := server.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}

	// 获取metrics组件
	metricsCollector := server.GetMetricsCollector()
	if metricsCollector == nil {
		return nil, fmt.Errorf("metrics collector not available")
	}

	// 获取Redis缓存
	redis := server.GetRedis()
	if redis == nil {
		return nil, fmt.Errorf("redis cache not available")
	}

	// 创建优化器工厂
	optimizerFactory := optimizer.NewFactory()

	// 创建exchange客户端
	var exchangeClient exchange.Exchange
	if cfg.Exchange.APIKey != "" && cfg.Exchange.APISecret != "" {
		// 创建真实的exchange客户端
		exchangeConfig := &exchange.ExchangeConfig{
			Name:           cfg.Exchange.Name,
			APIKey:         cfg.Exchange.APIKey,
			APISecret:      cfg.Exchange.APISecret,
			TestNet:        cfg.Exchange.TestNet,
			BaseURL:        cfg.Exchange.BaseURL,
			FuturesBaseURL: cfg.Exchange.FuturesBaseURL,
		}

		// 创建速率限制器
		rateLimiter := exchange.NewRateLimiter(redis, time.Second)

		// 创建Binance客户端
		exchangeClient = binance.NewClient(exchangeConfig, rateLimiter)
		log.Printf("Exchange client initialized: %s", cfg.Exchange.Name)
	} else {
		// 使用纸上交易exchange
		log.Printf("Warning: Using paper trading exchange (no API credentials)")
		exchangeClient = paper.NewExchange(nil, map[string]float64{
			"USDT": 100000.0, // 初始资金
		})
	}

	// 创建账户管理器
	accountManager := account.NewManager(db.DB, redis, exchangeClient)

	// 创建自动化系统
	automationSystem := automation.NewAutomationSystem(
		cfg,
		db,
		exchangeClient,
		accountManager,
		metricsCollector,
		optimizerFactory,
	)

	log.Println("✅ Automation system initialized with all components")
	return automationSystem, nil
}

package api

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"qcat/internal/auth"
	"qcat/internal/automation"
	"qcat/internal/cache"
	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/exchange"
	"qcat/internal/exchange/account"
	"qcat/internal/exchange/binance"
	"qcat/internal/monitor"
	"qcat/internal/monitoring"
	"qcat/internal/security"
	"qcat/internal/stability"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Global metrics collector to avoid duplicate registration
var globalMetricsCollector *monitor.MetricsCollector

// Server represents the API server
type Server struct {
	config     *config.Config
	router     *gin.Engine
	httpServer *http.Server
	upgrader   websocket.Upgrader
	handlers   *Handlers

	// Core services
	db               *database.DB
	redis            cache.Cacher
	jwtManager       *auth.JWTManager
	metrics          *monitoring.Metrics
	metricsCollector *monitor.MetricsCollector
	memory           *stability.MemoryManager
	network          *stability.NetworkReconnectManager
	health           *stability.HealthChecker
	shutdown         *stability.GracefulShutdownManager

	// Security services
	keyManager  *security.KeyManager
	auditLogger *security.AuditLogger

	// Automation system
	automationSystem *automation.AutomationSystem
}

// Handlers contains all API handlers
type Handlers struct {
	Optimizer  *OptimizerHandler
	Strategy   *StrategyHandler
	Portfolio  *PortfolioHandler
	Risk       *RiskHandler
	Hotlist    *HotlistHandler
	Metrics    *MetricsHandler
	Audit      *AuditHandler
	WebSocket  *WebSocketHandler
	Auth       *AuthHandler
	Cache      *CacheHandler
	Security   *SecurityHandler
	Dashboard  *DashboardHandler
	Market     *MarketHandler
	Trading    *TradingHandler
	Automation *AutomationHandler
}

// RateLimiter 速率限制器结构
type RateLimiter struct {
	requestsPerMinute int
	burst             int
	clients           map[string]*ClientLimiter
	mu                sync.RWMutex
	cleanupTicker     *time.Ticker
	done              chan bool
}

// ClientLimiter 客户端限制器
type ClientLimiter struct {
	tokens     int
	lastRefill time.Time
	burst      int
	rate       int
}

// NewRateLimiter 创建新的速率限制器
func NewRateLimiter(requestsPerMinute, burst int) *RateLimiter {
	rl := &RateLimiter{
		requestsPerMinute: requestsPerMinute,
		burst:             burst,
		clients:           make(map[string]*ClientLimiter),
		cleanupTicker:     time.NewTicker(time.Minute * 5), // 每5分钟清理一次
		done:              make(chan bool),
	}

	// 启动清理协程
	go rl.cleanup()
	return rl
}

// cleanup 定期清理过期的客户端限制器
func (rl *RateLimiter) cleanup() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.mu.Lock()
			now := time.Now()
			for clientID, limiter := range rl.clients {
				// 如果超过10分钟没有活动，删除该客户端
				if now.Sub(limiter.lastRefill) > time.Minute*10 {
					delete(rl.clients, clientID)
				}
			}
			rl.mu.Unlock()
		case <-rl.done:
			rl.cleanupTicker.Stop()
			return
		}
	}
}

// Stop 停止速率限制器
func (rl *RateLimiter) Stop() {
	close(rl.done)
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(clientID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	limiter, exists := rl.clients[clientID]

	if !exists {
		// 创建新的客户端限制器
		limiter = &ClientLimiter{
			tokens:     rl.burst,
			lastRefill: now,
			burst:      rl.burst,
			rate:       rl.requestsPerMinute,
		}
		rl.clients[clientID] = limiter
	}

	// 计算需要补充的令牌数
	timePassed := now.Sub(limiter.lastRefill)
	tokensToAdd := int(timePassed.Minutes() * float64(limiter.rate))

	if tokensToAdd > 0 {
		limiter.tokens = min(limiter.burst, limiter.tokens+tokensToAdd)
		limiter.lastRefill = now
	}

	// 检查是否有可用令牌
	if limiter.tokens > 0 {
		limiter.tokens--
		return true
	}

	return false
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getClientID 获取客户端标识符
func getClientID(c *gin.Context) string {
	// 优先使用用户ID（如果已认证）
	if userID, exists := c.Get("user_id"); exists {
		return fmt.Sprintf("user:%s", userID)
	}

	// 否则使用IP地址
	return fmt.Sprintf("ip:%s", c.ClientIP())
}

// isIPInWhitelist 检查IP是否在白名单中
func isIPInWhitelist(clientIP string, whitelist []string) bool {
	if len(whitelist) == 0 {
		return false
	}

	for _, whitelistIP := range whitelist {
		if clientIP == whitelistIP {
			return true
		}
		// 支持CIDR格式的IP段匹配（如 192.168.1.0/24）
		if strings.Contains(whitelistIP, "/") {
			if isIPInCIDR(clientIP, whitelistIP) {
				return true
			}
		}
	}
	return false
}

// isIPInCIDR 检查IP是否在CIDR网段中
func isIPInCIDR(clientIP, cidr string) bool {
	// 解析CIDR网段
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		log.Printf("Warning: Invalid CIDR format %s: %v", cidr, err)
		return false
	}

	// 解析客户端IP
	ip := net.ParseIP(clientIP)
	if ip == nil {
		log.Printf("Warning: Invalid IP format %s", clientIP)
		return false
	}

	// 检查IP是否在网段中
	return ipNet.Contains(ip)
}

// NewServer creates a new API server
func NewServer(cfg *config.Config) (*Server, error) {
	// Set Gin mode
	if cfg.App.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Initialize core services
	var db *database.DB
	var redis cache.Cacher
	var err error

	// Try to connect to database, but don't fail if unavailable
	db, err = database.NewConnection(&database.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		DBName:          cfg.Database.DBName,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpen:         cfg.Database.MaxOpen,
		MaxIdle:         cfg.Database.MaxIdle,
		Timeout:         cfg.Database.Timeout,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
	})
	if err != nil {
		log.Printf("Warning: Failed to connect to database: %v", err)
		log.Printf("Server will start without database support")
		db = nil
	}

	// Initialize cache with fallback support
	cacheFactory := cache.NewCacheFactory(&cache.CacheFactoryConfig{
		RedisEnabled:      true,
		RedisAddr:         cfg.Redis.Addr,
		RedisPassword:     cfg.Redis.Password,
		RedisDB:           cfg.Redis.DB,
		RedisPoolSize:     cfg.Redis.PoolSize,
		MemoryEnabled:     true,
		MemoryMaxSize:     10000,
		DatabaseEnabled:   db != nil,
		DatabaseTableName: "cache_entries",
		FallbackConfig:    cache.DefaultFallbackConfig(),
	})

	cacheManager, err := cacheFactory.CreateCache(db)
	if err != nil {
		log.Printf("Warning: Failed to create cache manager: %v", err)
		log.Printf("Server will start with memory-only cache")
		cacheManager = cacheFactory.CreateMemoryOnlyCache()
	}

	// Create cache adapter to maintain interface compatibility
	if cacheManager, ok := cacheManager.(*cache.CacheManager); ok {
		redis = cache.NewCacheAdapter(cacheManager)
	} else {
		// If it's already a Cacher (memory-only cache), use it directly
		redis = cacheManager
	}

	jwtManager := auth.NewJWTManager(cfg.JWT.SecretKey, cfg.JWT.Duration)
	metrics := monitoring.NewMetrics()

	// Create metrics collector only once to avoid duplicate registration
	var metricsCollector *monitor.MetricsCollector
	if globalMetricsCollector == nil {
		metricsCollector = monitor.NewMetricsCollector()
		globalMetricsCollector = metricsCollector
	} else {
		metricsCollector = globalMetricsCollector
	}
	// Initialize memory manager with configuration
	memoryConfig := &stability.MemoryConfig{
		MonitorInterval:      cfg.Memory.MonitorInterval,
		HighWaterMarkPercent: cfg.Memory.HighWaterMarkPercent,
		LowWaterMarkPercent:  cfg.Memory.LowWaterMarkPercent,
		AlertThreshold:       cfg.Memory.AlertThreshold,
		EnableAutoGC:         cfg.Memory.EnableAutoGC,
		GCInterval:           cfg.Memory.GCInterval,
		ForceGCThreshold:     cfg.Memory.ForceGCThreshold,
		MaxMemoryMB:          cfg.Memory.MaxMemoryMB,
		MaxHeapMB:            cfg.Memory.MaxHeapMB,
	}
	memory := stability.NewMemoryManager(memoryConfig)

	// Initialize network reconnect manager with configuration
	networkConfig := &stability.ReconnectConfig{
		MaxRetries:             cfg.Network.MaxRetries,
		InitialDelay:           cfg.Network.InitialDelay,
		MaxDelay:               cfg.Network.MaxDelay,
		BackoffMultiplier:      cfg.Network.BackoffMultiplier,
		JitterFactor:           cfg.Network.JitterFactor,
		HealthCheckInterval:    cfg.Network.HealthCheckInterval,
		ConnectionTimeout:      cfg.Network.ConnectionTimeout,
		PingInterval:           cfg.Network.PingInterval,
		PongTimeout:            cfg.Network.PongTimeout,
		MaxConsecutiveFailures: cfg.Network.MaxConsecutiveFailures,
		AlertThreshold:         cfg.Network.AlertThreshold,
	}
	network := stability.NewNetworkReconnectManager(networkConfig)

	// Initialize health checker with configuration
	healthConfig := &stability.HealthConfig{
		CheckInterval:      cfg.Health.CheckInterval,
		Timeout:            cfg.Health.Timeout,
		RetryCount:         cfg.Health.RetryCount,
		RetryInterval:      cfg.Health.RetryInterval,
		DegradedThreshold:  cfg.Health.DegradedThreshold,
		UnhealthyThreshold: cfg.Health.UnhealthyThreshold,
		AlertThreshold:     cfg.Health.AlertThreshold,
		AlertCooldown:      cfg.Health.AlertCooldown,
	}
	health := stability.NewHealthChecker(healthConfig)

	// Initialize graceful shutdown manager with configuration
	shutdownConfig := &stability.ShutdownConfig{
		ShutdownTimeout:      cfg.Shutdown.ShutdownTimeout,
		ComponentTimeout:     cfg.Shutdown.ComponentTimeout,
		SignalTimeout:        cfg.Shutdown.SignalTimeout,
		EnableSignalHandling: cfg.Shutdown.EnableSignalHandling,
		ForceShutdownAfter:   cfg.Shutdown.ForceShutdownAfter,
		LogShutdownProgress:  cfg.Shutdown.LogShutdownProgress,
		ShutdownOrder:        cfg.Shutdown.ShutdownOrder,
	}
	shutdown := stability.NewGracefulShutdownManager(shutdownConfig)

	server := &Server{
		config: cfg,
		router: router,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
		db:               db,
		redis:            redis,
		jwtManager:       jwtManager,
		metrics:          metrics,
		metricsCollector: metricsCollector,
		memory:           memory,
		network:          network,
		health:           health,
		shutdown:         shutdown,
	}

	// Store cache manager reference for cache handler
	var cacheManagerRef *cache.CacheManager
	if cacheAdapter, ok := redis.(*cache.CacheAdapter); ok {
		cacheManagerRef = cacheAdapter.GetManager()
	}

	// Set up database connection pool monitoring
	if db != nil {
		db.SetMonitorCallback(metrics.UpdateDatabasePoolMetrics)
	}

	// Initialize security system
	keyManager, err := initializeKeyManager(cfg)
	if err != nil {
		log.Printf("Warning: Failed to initialize key manager: %v", err)
		keyManager = nil
	}

	auditLogger, err := initializeAuditLogger(cfg)
	if err != nil {
		log.Printf("Warning: Failed to initialize audit logger: %v", err)
		auditLogger = nil
	}

	// Initialize exchange client and account manager
	var accountManager *account.Manager
	if cfg.Exchange.APIKey != "" && cfg.Exchange.APISecret != "" {
		// Create Binance client
		exchangeConfig := &exchange.ExchangeConfig{
			Name:           cfg.Exchange.Name,
			APIKey:         cfg.Exchange.APIKey,
			APISecret:      cfg.Exchange.APISecret,
			TestNet:        cfg.Exchange.TestNet,
			BaseURL:        cfg.Exchange.BaseURL,
			FuturesBaseURL: cfg.Exchange.FuturesBaseURL,
		}

		// Create rate limiter for Binance
		rateLimiter := exchange.NewRateLimiter(redis, time.Second)

		// Create Binance client with rate limiter
		exchangeClient := binance.NewClient(exchangeConfig, rateLimiter)

		// Create account manager
		accountManager = account.NewManager(db.DB, redis, exchangeClient)
		log.Printf("Account manager initialized successfully")
	} else {
		log.Printf("Warning: Binance API credentials not configured, using mock data")
	}

	// Initialize automation system
	var automationSystem *automation.AutomationSystem
	if db != nil && accountManager != nil {
		// Create a mock exchange client for automation system if needed
		var exchangeClient exchange.Exchange
		if cfg.Exchange.APIKey != "" && cfg.Exchange.APISecret != "" {
			exchangeConfig := &exchange.ExchangeConfig{
				Name:           cfg.Exchange.Name,
				APIKey:         cfg.Exchange.APIKey,
				APISecret:      cfg.Exchange.APISecret,
				TestNet:        cfg.Exchange.TestNet,
				BaseURL:        cfg.Exchange.BaseURL,
				FuturesBaseURL: cfg.Exchange.FuturesBaseURL,
			}
			rateLimiter := exchange.NewRateLimiter(redis, time.Second)
			exchangeClient = binance.NewClient(exchangeConfig, rateLimiter)
		}

		// Create automation system
		automationSystem = automation.NewAutomationSystem(
			cfg, db, exchangeClient, accountManager, metricsCollector, nil,
		)
		log.Printf("Automation system initialized successfully")
	} else {
		log.Printf("Warning: Automation system not initialized due to missing dependencies")
	}

	// Store automation system in server
	server.automationSystem = automationSystem

	// Initialize handlers with dependencies
	server.handlers = &Handlers{
		Optimizer:  NewOptimizerHandler(db, redis, metricsCollector),
		Strategy:   NewStrategyHandler(db, redis, metricsCollector),
		Portfolio:  NewPortfolioHandler(db, redis, metricsCollector),
		Risk:       NewRiskHandler(db, redis, metricsCollector),
		Hotlist:    NewHotlistHandler(db, redis, metricsCollector),
		Metrics:    NewMetricsHandler(db, metricsCollector),
		Audit:      NewAuditHandler(db, metricsCollector),
		WebSocket:  NewWebSocketHandler(server.upgrader, metrics),
		Auth:       NewAuthHandler(jwtManager, db),
		Cache:      NewCacheHandler(cacheManagerRef),
		Security:   NewSecurityHandler(keyManager, auditLogger),
		Dashboard:  NewDashboardHandler(db, metricsCollector, accountManager),
		Market:     NewMarketHandler(db, metricsCollector),
		Trading:    NewTradingHandler(db, metricsCollector),
		Automation: NewAutomationHandler(db, metricsCollector, automationSystem),
	}

	// Store security components for middleware
	server.keyManager = keyManager
	server.auditLogger = auditLogger

	// Setup routes
	server.setupRoutes()

	return server, nil
}

// initializeKeyManager initializes the key manager
func initializeKeyManager(cfg *config.Config) (*security.KeyManager, error) {
	keyManagerConfig := security.DefaultKeyManagerConfig()

	// Override with config values if available
	// This would be expanded based on your config structure

	return security.NewKeyManager(keyManagerConfig)
}

// initializeAuditLogger initializes the audit logger
func initializeAuditLogger(cfg *config.Config) (*security.AuditLogger, error) {
	auditConfig := security.DefaultAuditConfig()

	// Override with config values if available
	// This would be expanded based on your config structure

	return security.NewAuditLogger(auditConfig)
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Middleware
	s.router.Use(gin.Logger())
	s.router.Use(gin.Recovery())
	s.router.Use(corsMiddleware(s.config.CORS))
	s.router.Use(rateLimitMiddleware(s.config.RateLimit))
	s.router.Use(s.metrics.MetricsMiddleware())

	// Swagger documentation
	if s.config.App.Environment == "development" {
		s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// Prometheus metrics
	if s.config.Monitoring.PrometheusEnabled {
		s.router.GET(s.config.Monitoring.PrometheusPath, gin.WrapH(monitoring.PrometheusHandler()))
	}

	// API v1 group
	v1 := s.router.Group("/api/v1")
	{
		// Public routes (no authentication required) - only auth endpoints
		auth := v1.Group("/auth")
		{
			auth.POST("/login", s.handlers.Auth.Login)
			auth.POST("/register", s.handlers.Auth.Register)
			auth.POST("/refresh", s.handlers.Auth.RefreshToken)
		}

		// Protected routes (authentication required)
		protected := v1.Group("")
		protected.Use(s.jwtManager.AuthMiddleware())
		{
			// Dashboard routes (now protected)
			protected.GET("/dashboard", s.handlers.Dashboard.GetDashboardData)

			// Market data routes (now protected)
			protected.GET("/market/data", s.handlers.Market.GetMarketData)

			// Trading activity routes (now protected)
			protected.GET("/trading/activity", s.handlers.Trading.GetTradingActivity)

			// System metrics (now protected)
			protected.GET("/metrics/system", s.handlers.Metrics.GetSystemMetrics)

			// Strategy routes (all protected)
			strategy := protected.Group("/strategy")
			{
				strategy.GET("/", s.handlers.Strategy.ListStrategies) // 移到受保护路由
				strategy.GET("/:id", s.handlers.Strategy.GetStrategy)
				strategy.POST("/", s.handlers.Strategy.CreateStrategy)
				strategy.PUT("/:id", s.handlers.Strategy.UpdateStrategy)
				strategy.DELETE("/:id", s.handlers.Strategy.DeleteStrategy)
				strategy.POST("/:id/promote", s.handlers.Strategy.PromoteStrategy)
				strategy.POST("/:id/start", s.handlers.Strategy.StartStrategy)
				strategy.POST("/:id/stop", s.handlers.Strategy.StopStrategy)
				strategy.POST("/:id/backtest", s.handlers.Strategy.RunBacktest)
			}

			// Optimizer routes
			optimizer := protected.Group("/optimizer")
			{
				optimizer.POST("/run", s.handlers.Optimizer.RunOptimization)
				optimizer.GET("/tasks", s.handlers.Optimizer.GetTasks)
				optimizer.GET("/tasks/:id", s.handlers.Optimizer.GetTask)
				optimizer.GET("/results/:id", s.handlers.Optimizer.GetResults)
			}

			// Portfolio routes
			portfolio := protected.Group("/portfolio")
			{
				portfolio.GET("/overview", s.handlers.Portfolio.GetOverview)
				portfolio.GET("/allocations", s.handlers.Portfolio.GetAllocations)
				portfolio.POST("/rebalance", s.handlers.Portfolio.Rebalance)
				portfolio.GET("/history", s.handlers.Portfolio.GetHistory)
			}

			// Risk routes
			risk := protected.Group("/risk")
			{
				risk.GET("/overview", s.handlers.Risk.GetOverview)
				risk.GET("/limits", s.handlers.Risk.GetLimits)
				risk.POST("/limits", s.handlers.Risk.SetLimits)
				risk.GET("/circuit-breakers", s.handlers.Risk.GetCircuitBreakers)
				risk.POST("/circuit-breakers", s.handlers.Risk.SetCircuitBreakers)
				risk.GET("/violations", s.handlers.Risk.GetViolations)
			}

			// Hotlist routes
			hotlist := protected.Group("/hotlist")
			{
				hotlist.GET("/symbols", s.handlers.Hotlist.GetHotSymbols)
				hotlist.POST("/approve", s.handlers.Hotlist.ApproveSymbol)
				hotlist.GET("/whitelist", s.handlers.Hotlist.GetWhitelist)
				hotlist.POST("/whitelist", s.handlers.Hotlist.AddToWhitelist)
				hotlist.DELETE("/whitelist/:symbol", s.handlers.Hotlist.RemoveFromWhitelist)
			}

			// Metrics routes
			metrics := protected.Group("/metrics")
			{
				metrics.GET("/strategy/:id", s.handlers.Metrics.GetStrategyMetrics)
				metrics.GET("/performance", s.handlers.Metrics.GetPerformanceMetrics)
			}

			// Memory management routes
			memory := protected.Group("/memory")
			{
				memory.GET("/stats", s.getMemoryStats)
				memory.POST("/gc", s.forceGC)
			}

			// Network management routes
			network := protected.Group("/network")
			{
				network.GET("/connections", s.getNetworkConnections)
				network.GET("/connections/:id", s.getNetworkConnection)
				network.POST("/connections/:id/reconnect", s.forceNetworkReconnect)
			}

			// Health management routes
			health := protected.Group("/health")
			{
				health.GET("/status", s.getHealthStatus)
				health.GET("/checks", s.getAllHealthChecks)
				health.GET("/checks/:name", s.getHealthCheck)
				health.POST("/checks/:name/force", s.forceHealthCheck)
			}

			// Shutdown management routes
			shutdown := protected.Group("/shutdown")
			{
				shutdown.GET("/status", s.getShutdownStatus)
				shutdown.POST("/graceful", s.initiateGracefulShutdown)
				shutdown.POST("/force", s.forceShutdown)
			}

			// Audit routes
			audit := protected.Group("/audit")
			{
				audit.GET("/logs", s.handlers.Audit.GetLogs)
				audit.GET("/decisions", s.handlers.Audit.GetDecisionChains)
				audit.GET("/performance", s.handlers.Audit.GetPerformanceMetrics)
				audit.POST("/export", s.handlers.Audit.ExportReport)
			}

			// Cache management routes
			if s.handlers.Cache != nil {
				s.handlers.Cache.RegisterRoutes(protected)
			}

			// Security management routes
			if s.handlers.Security != nil {
				s.handlers.Security.RegisterRoutes(protected)
			}

			// Automation system routes
			if s.handlers.Automation != nil {
				s.handlers.Automation.RegisterRoutes(protected)
			}
		}
	}

	// WebSocket routes (authentication handled in WebSocket handler)
	ws := s.router.Group("/ws")
	{
		ws.GET("/market/:symbol", s.handlers.WebSocket.MarketStream)
		ws.GET("/strategy/:id", s.handlers.WebSocket.StrategyStream)
		ws.GET("/alerts", s.handlers.WebSocket.AlertsStream)
	}

	// Health check
	s.router.GET("/health", func(c *gin.Context) {
		// Check database health
		dbHealth := "ok"
		if s.db != nil {
			if err := s.db.HealthCheck(c.Request.Context()); err != nil {
				dbHealth = "error"
			}
		} else {
			dbHealth = "unavailable"
		}

		// Check Redis health
		redisHealth := "ok"
		if s.redis != nil {
			// Try to perform a simple operation to check health
			if err := s.redis.Set(c.Request.Context(), "health_check", "ok", time.Second); err != nil {
				redisHealth = "error"
			}
		} else {
			redisHealth = "unavailable"
		}

		// Check memory health
		memoryHealth := "ok"
		if s.memory != nil && !s.memory.IsHealthy() {
			memoryHealth = "warning"
		}

		// Check network health
		networkHealth := "ok"
		if s.network != nil && !s.network.IsHealthy() {
			networkHealth = "warning"
		}

		// Check health checker status
		healthCheckerStatus := "ok"
		if s.health != nil && !s.health.IsHealthy() {
			healthCheckerStatus = "warning"
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().UTC(),
			"services": gin.H{
				"database": dbHealth,
				"redis":    redisHealth,
				"memory":   memoryHealth,
				"network":  networkHealth,
				"health":   healthCheckerStatus,
			},
		})
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:           fmt.Sprintf(":%d", s.config.Server.Port),
		Handler:        s.router,
		ReadTimeout:    s.config.Server.ReadTimeout,
		WriteTimeout:   s.config.Server.WriteTimeout,
		MaxHeaderBytes: s.config.Server.MaxHeaderBytes,
	}

	log.Printf("Starting API server on port %d", s.config.Server.Port)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("Shutting down server...")

	// Stop memory manager
	if s.memory != nil {
		s.memory.Stop()
	}

	// Stop network reconnect manager
	if s.network != nil {
		s.network.Stop()
	}

	// Stop health checker
	if s.health != nil {
		s.health.Stop()
	}

	// Stop graceful shutdown manager
	if s.shutdown != nil {
		s.shutdown.Stop()
	}

	// Close database connection
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}

	// Close Redis connection
	if s.redis != nil {
		if err := s.redis.Close(); err != nil {
			log.Printf("Error closing Redis: %v", err)
		}
	}

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("Server stopped gracefully")
	return nil
}

// corsMiddleware adds CORS headers
func corsMiddleware(corsConfig config.CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow all origins for now, in production you should check against allowed origins
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// rateLimitMiddleware adds rate limiting
func rateLimitMiddleware(rateLimitConfig config.RateLimitConfig) gin.HandlerFunc {
	// 如果速率限制未启用，直接返回空中间件
	if !rateLimitConfig.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// 创建速率限制器
	rateLimiter := NewRateLimiter(rateLimitConfig.RequestsPerMinute, rateLimitConfig.Burst)

	return func(c *gin.Context) {
		// 检查IP白名单
		if rateLimitConfig.WhitelistEnabled && isIPInWhitelist(c.ClientIP(), rateLimitConfig.WhitelistIPs) {
			// 白名单IP跳过限流检查
			c.Next()
			return
		}

		// 获取客户端标识符
		clientID := getClientID(c)

		// 检查是否允许请求
		if !rateLimiter.Allow(clientID) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests, please try again later",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// getMemoryStats returns current memory statistics
func (s *Server) getMemoryStats(c *gin.Context) {
	if s.memory == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Memory manager not available",
		})
		return
	}

	stats := s.memory.GetMemoryStats()
	c.JSON(http.StatusOK, gin.H{
		"memory": stats,
	})
}

// forceGC forces a garbage collection
func (s *Server) forceGC(c *gin.Context) {
	if s.memory == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Memory manager not available",
		})
		return
	}

	s.memory.ForceGC()
	c.JSON(http.StatusOK, gin.H{
		"message": "Garbage collection completed",
	})
}

// getNetworkConnections returns all network connection status
func (s *Server) getNetworkConnections(c *gin.Context) {
	if s.network == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Network manager not available",
		})
		return
	}

	connections := s.network.GetAllConnectionStatus()
	c.JSON(http.StatusOK, gin.H{
		"connections": connections,
	})
}

// getNetworkConnection returns status of a specific network connection
func (s *Server) getNetworkConnection(c *gin.Context) {
	if s.network == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Network manager not available",
		})
		return
	}

	connectionID := c.Param("id")
	connection := s.network.GetConnectionStatus(connectionID)
	if connection == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Connection not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"connection": connection,
	})
}

// forceNetworkReconnect forces reconnection of a specific network connection
func (s *Server) forceNetworkReconnect(c *gin.Context) {
	if s.network == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Network manager not available",
		})
		return
	}

	connectionID := c.Param("id")
	if err := s.network.ForceReconnect(connectionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Reconnection initiated",
	})
}

// getHealthStatus returns overall health status
func (s *Server) getHealthStatus(c *gin.Context) {
	if s.health == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Health checker not available",
		})
		return
	}

	status := s.health.GetOverallHealth()
	c.JSON(http.StatusOK, gin.H{
		"health": status,
	})
}

// getAllHealthChecks returns all health check statuses
func (s *Server) getAllHealthChecks(c *gin.Context) {
	if s.health == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Health checker not available",
		})
		return
	}

	checks := s.health.GetAllHealthStatus()
	c.JSON(http.StatusOK, gin.H{
		"checks": checks,
	})
}

// getHealthCheck returns status of a specific health check
func (s *Server) getHealthCheck(c *gin.Context) {
	if s.health == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Health checker not available",
		})
		return
	}

	checkName := c.Param("name")
	check := s.health.GetHealthStatus(checkName)
	if check == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Health check not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"check": check,
	})
}

// forceHealthCheck forces a health check for a specific service
func (s *Server) forceHealthCheck(c *gin.Context) {
	if s.health == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Health checker not available",
		})
		return
	}

	checkName := c.Param("name")
	if err := s.health.ForceCheck(checkName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Health check initiated",
	})
}

// getShutdownStatus returns current shutdown status
func (s *Server) getShutdownStatus(c *gin.Context) {
	if s.shutdown == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Shutdown manager not available",
		})
		return
	}

	status := s.shutdown.GetShutdownStatus()
	c.JSON(http.StatusOK, gin.H{
		"shutdown": status,
	})
}

// initiateGracefulShutdown initiates graceful shutdown process
func (s *Server) initiateGracefulShutdown(c *gin.Context) {
	if s.shutdown == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Shutdown manager not available",
		})
		return
	}

	result := s.shutdown.Shutdown(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{
		"message": "Graceful shutdown initiated",
		"result":  result,
	})
}

// forceShutdown forces immediate shutdown
func (s *Server) forceShutdown(c *gin.Context) {
	if s.shutdown == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Shutdown manager not available",
		})
		return
	}

	s.shutdown.ForceShutdown()
	c.JSON(http.StatusOK, gin.H{
		"message": "Force shutdown initiated",
	})
}

// GetShutdownManager returns the shutdown manager
func (s *Server) GetShutdownManager() *stability.GracefulShutdownManager {
	return s.shutdown
}

// GetDB returns the database connection
func (s *Server) GetDB() *database.DB {
	return s.db
}

// GetRedis returns the Redis cache
func (s *Server) GetRedis() cache.Cacher {
	return s.redis
}

// GetMemoryManager returns the memory manager
func (s *Server) GetMemoryManager() *stability.MemoryManager {
	return s.memory
}

// GetNetworkManager returns the network manager
func (s *Server) GetNetworkManager() *stability.NetworkReconnectManager {
	return s.network
}

// GetHealthChecker returns the health checker
func (s *Server) GetHealthChecker() *stability.HealthChecker {
	return s.health
}

// GetMetricsCollector returns the metrics collector
func (s *Server) GetMetricsCollector() *monitor.MetricsCollector {
	return s.metricsCollector
}

// RegisterOrchestratorHandler registers the orchestrator handler routes
func (s *Server) RegisterOrchestratorHandler(handler *OrchestratorHandler) {
	// Add orchestrator routes to the protected API group
	v1 := s.router.Group("/api/v1")
	protected := v1.Group("")
	protected.Use(s.jwtManager.AuthMiddleware())

	orchestrator := protected.Group("/orchestrator")
	{
		orchestrator.GET("/status", handler.handleStatus)
		orchestrator.GET("/services", handler.handleServices)
		orchestrator.POST("/services/start", handler.handleStartService)
		orchestrator.POST("/services/stop", handler.handleStopService)
		orchestrator.POST("/services/restart", handler.handleRestartService)
		orchestrator.POST("/optimize", handler.handleOptimize)
		orchestrator.GET("/health", handler.handleHealth)
	}
}

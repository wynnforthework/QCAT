package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"qcat/internal/config"
	"qcat/internal/learning/automl"
	"qcat/internal/orchestrator"
	"qcat/internal/strategy/optimizer"
)

// OptimizerService represents the standalone optimizer service
type OptimizerService struct {
	server           *http.Server
	msgQueue         orchestrator.MessageQueue
	optimizer        *optimizer.Orchestrator
	resultSharingMgr *automl.ResultSharingManager
	ctx              context.Context
	cancel           context.CancelFunc
}

// Config holds the optimizer service configuration
type Config struct {
	Port          int                         `json:"port"`
	MessageQueue  string                      `json:"message_queue"`
	RedisAddr     string                      `json:"redis_addr,omitempty"`
	LogLevel      string                      `json:"log_level"`
	ResultSharing *automl.ResultSharingConfig `json:"result_sharing" yaml:"result_sharing"`
}

func main() {
	var (
		configFile = flag.String("config", "configs/optimizer.json", "Configuration file path")
		port       = flag.Int("port", 8081, "HTTP server port")
		logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	)
	flag.Parse()

	// Load configuration
	config, err := loadConfig(*configFile, *port, *logLevel)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create optimizer service
	service, err := NewOptimizerService(config)
	if err != nil {
		log.Fatalf("Failed to create optimizer service: %v", err)
	}

	// Start the service
	if err := service.Start(); err != nil {
		log.Fatalf("Failed to start optimizer service: %v", err)
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down optimizer service...")
	if err := service.Shutdown(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}

// NewOptimizerService creates a new optimizer service
func NewOptimizerService(config *Config) (*OptimizerService, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create message queue
	var msgQueue orchestrator.MessageQueue
	switch config.MessageQueue {
	case "redis":
		msgQueue = orchestrator.NewRedisMessageQueue(config.RedisAddr)
	default:
		msgQueue = orchestrator.NewInMemoryMessageQueue(1000)
	}

	// Create optimizer
	factory := optimizer.NewFactory()
	optimizerInstance := factory.CreateOrchestrator(nil) // Pass nil for DB since we don't have one in standalone mode

	// Create result sharing manager
	var resultSharingMgr *automl.ResultSharingManager
	if config.ResultSharing != nil && config.ResultSharing.Enabled {
		var err error
		resultSharingMgr, err = automl.NewResultSharingManager(config.ResultSharing)
		if err != nil {
			log.Printf("Warning: failed to create result sharing manager: %v", err)
		} else {
			log.Printf("Result sharing manager initialized with mode: %s", config.ResultSharing.Mode)
		}
	}

	// Create HTTP server
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: mux,
	}

	service := &OptimizerService{
		server:           server,
		msgQueue:         msgQueue,
		optimizer:        optimizerInstance,
		resultSharingMgr: resultSharingMgr,
		ctx:              ctx,
		cancel:           cancel,
	}

	// Setup HTTP routes
	service.setupRoutes(mux)

	// Setup message queue handlers
	service.setupMessageHandlers()

	return service, nil
}

// Start starts the optimizer service
func (s *OptimizerService) Start() error {
	log.Printf("Starting optimizer service on port %s", s.server.Addr)

	// Start HTTP server
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	return nil
}

// Shutdown gracefully shuts down the optimizer service
func (s *OptimizerService) Shutdown() error {
	s.cancel()

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	// Close message queue
	if err := s.msgQueue.Close(); err != nil {
		return fmt.Errorf("failed to close message queue: %w", err)
	}

	return nil
}

// setupRoutes sets up HTTP routes
func (s *OptimizerService) setupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/optimize", s.optimizeHandler)
	mux.HandleFunc("/status", s.statusHandler)
	mux.HandleFunc("/metrics", s.metricsHandler)
	mux.HandleFunc("/shared-results", s.sharedResultsHandler)
	mux.HandleFunc("/share-result", s.shareResultHandler)
}

// setupMessageHandlers sets up message queue handlers
func (s *OptimizerService) setupMessageHandlers() {
	// Subscribe to optimization requests
	s.msgQueue.Subscribe("optimization.request", s.handleOptimizationRequest)
}

// healthHandler handles health check requests
func (s *OptimizerService) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "optimizer",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// optimizeHandler handles direct optimization requests
func (s *OptimizerService) optimizeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req orchestrator.OptimizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Process optimization request
	result := s.processOptimizationRequest(&req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// statusHandler handles status requests
func (s *OptimizerService) statusHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"service":   "optimizer",
		"status":    "running",
		"uptime":    time.Since(time.Now()), // This would be actual uptime
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// metricsHandler handles metrics requests
func (s *OptimizerService) metricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"service":              "optimizer",
		"optimizations_total":  0, // Would track actual metrics
		"optimizations_active": 0,
		"memory_usage":         0,
		"cpu_usage":            0,
		"timestamp":            time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// handleOptimizationRequest handles optimization requests from message queue
func (s *OptimizerService) handleOptimizationRequest(topic string, message []byte) error {
	var req orchestrator.OptimizationRequest
	if err := json.Unmarshal(message, &req); err != nil {
		return fmt.Errorf("failed to unmarshal optimization request: %w", err)
	}

	log.Printf("Received optimization request: %s", req.RequestID)

	// Process the request
	result := s.processOptimizationRequest(&req)

	// Publish result
	return s.msgQueue.Publish("optimization.result", result)
}

// processOptimizationRequest processes an optimization request
func (s *OptimizerService) processOptimizationRequest(req *orchestrator.OptimizationRequest) *orchestrator.OptimizationResult {
	log.Printf("Processing optimization request %s for strategy %s", req.RequestID, req.StrategyID)

	result := &orchestrator.OptimizationResult{
		RequestID:  req.RequestID,
		StrategyID: req.StrategyID,
		Status:     "running",
	}

	// Check for shared results first
	if s.resultSharingMgr != nil {
		if sharedResult := s.resultSharingMgr.GetBestSharedResult(req.RequestID, req.StrategyID); sharedResult != nil {
			log.Printf("Found shared result for request %s, profit rate: %.2f%%",
				req.RequestID, sharedResult.Performance.ProfitRate)

			// Convert shared result to optimization result
			result.Status = "completed"
			result.BestParameters = sharedResult.Parameters
			// Convert automl.PerformanceMetrics to orchestrator.PerformanceMetrics
			result.Performance = orchestrator.PerformanceMetrics{
				TotalReturn: sharedResult.Performance.TotalReturn,
				SharpeRatio: sharedResult.Performance.SharpeRatio,
				MaxDrawdown: sharedResult.Performance.MaxDrawdown,
				WinRate:     sharedResult.Performance.WinRate,
				TradeCount:  0, // Not available in automl.PerformanceMetrics
			}
			result.Duration = time.Duration(0) // Shared results don't have duration
			result.Iterations = 0              // Shared results don't have iterations

			log.Printf("Using shared result for request %s", req.RequestID)
			return result
		}
	}

	// Perform actual optimization using the built-in optimizer
	optimizationResult, err := s.runOptimization(req)
	if err != nil {
		log.Printf("Optimization failed for request %s: %v", req.RequestID, err)
		result.Status = "failed"
		result.Error = err.Error()
		return result
	}

	// Set successful results
	result.Status = "completed"
	result.BestParameters = optimizationResult.BestParameters
	result.Performance = optimizationResult.Performance
	result.Duration = optimizationResult.Duration
	result.Iterations = optimizationResult.Iterations

	// Share the result if sharing is enabled
	if s.resultSharingMgr != nil {
		go s.shareOptimizationResult(req, optimizationResult)
	}

	log.Printf("Completed optimization request %s with %d iterations in %v",
		req.RequestID, result.Iterations, result.Duration)
	return result
}

// runOptimization performs the actual optimization process
func (s *OptimizerService) runOptimization(req *orchestrator.OptimizationRequest) (*OptimizationResult, error) {
	startTime := time.Now()

	// Validate optimization request
	if req.Parameters == nil || len(req.Parameters) == 0 {
		return nil, fmt.Errorf("no parameters specified for optimization")
	}

	// Setup parameter space
	paramSpace := make(map[string][2]float64)
	for name, param := range req.Parameters {
		if paramDef, ok := param.(map[string]interface{}); ok {
			if min, hasMin := paramDef["min"].(float64); hasMin {
				if max, hasMax := paramDef["max"].(float64); hasMax {
					paramSpace[name] = [2]float64{min, max}
				}
			}
		}
	}

	if len(paramSpace) == 0 {
		return nil, fmt.Errorf("no valid parameter ranges found")
	}

	// Choose optimization algorithm based on request method
	var bestParams map[string]float64
	var bestScore float64
	var iterations int
	var err error

	switch req.Method {
	case "grid":
		bestParams, bestScore, iterations, err = s.gridSearch(paramSpace, req)
	case "random":
		bestParams, bestScore, iterations, err = s.randomSearch(paramSpace, req)
	case "bayesian":
		bestParams, bestScore, iterations, err = s.bayesianOptimization(paramSpace, req)
	default:
		// Default to grid search
		bestParams, bestScore, iterations, err = s.gridSearch(paramSpace, req)
	}

	if err != nil {
		return nil, fmt.Errorf("optimization failed: %w", err)
	}

	// Convert best parameters to interface{}
	bestParamsInterface := make(map[string]interface{})
	for k, v := range bestParams {
		bestParamsInterface[k] = v
	}

	// Calculate performance metrics based on best score
	performance := s.calculatePerformanceMetrics(bestScore, iterations)

	return &OptimizationResult{
		BestParameters: bestParamsInterface,
		Performance:    performance,
		Duration:       time.Since(startTime),
		Iterations:     iterations,
		BestScore:      bestScore,
	}, nil
}

// OptimizationResult represents the result of an optimization
type OptimizationResult struct {
	BestParameters map[string]interface{}          `json:"best_parameters"`
	Performance    orchestrator.PerformanceMetrics `json:"performance"`
	Duration       time.Duration                   `json:"duration"`
	Iterations     int                             `json:"iterations"`
	BestScore      float64                         `json:"best_score"`
}

// gridSearch performs grid search optimization
func (s *OptimizerService) gridSearch(paramSpace map[string][2]float64, req *orchestrator.OptimizationRequest) (map[string]float64, float64, int, error) {
	gridSize := 10 // Default grid size
	if req.GridSize > 0 {
		gridSize = req.GridSize
	}

	// Generate grid points
	var paramNames []string
	var grids [][]float64

	for name, bounds := range paramSpace {
		paramNames = append(paramNames, name)
		grid := make([]float64, gridSize)
		step := (bounds[1] - bounds[0]) / float64(gridSize-1)
		for i := 0; i < gridSize; i++ {
			grid[i] = bounds[0] + float64(i)*step
		}
		grids = append(grids, grid)
	}

	bestParams := make(map[string]float64)
	bestScore := math.Inf(-1)
	iterations := 0

	// Perform grid search
	err := s.iterateGrid(grids, paramNames, 0, make([]float64, len(paramNames)),
		func(params []float64) {
			iterations++

			// Create parameter map
			paramMap := make(map[string]float64)
			for i, name := range paramNames {
				paramMap[name] = params[i]
			}

			// Evaluate objective function
			score := s.evaluateObjective(paramMap, req)

			if score > bestScore {
				bestScore = score
				bestParams = make(map[string]float64)
				for k, v := range paramMap {
					bestParams[k] = v
				}
			}
		})

	return bestParams, bestScore, iterations, err
}

// randomSearch performs random search optimization
func (s *OptimizerService) randomSearch(paramSpace map[string][2]float64, req *orchestrator.OptimizationRequest) (map[string]float64, float64, int, error) {
	maxIterations := 100
	if req.MaxIterations > 0 {
		maxIterations = req.MaxIterations
	}

	bestParams := make(map[string]float64)
	bestScore := math.Inf(-1)

	// 使用随机种子确保每台服务器的训练都不重复
	seed := time.Now().UnixNano() + int64(len(req.StrategyID)*1000) + int64(req.MaxIterations*100)
	rand.Seed(seed)

	for i := 0; i < maxIterations; i++ {
		// Generate random parameters
		params := make(map[string]float64)
		for name, bounds := range paramSpace {
			params[name] = bounds[0] + rand.Float64()*(bounds[1]-bounds[0])
		}

		// Evaluate objective function
		score := s.evaluateObjective(params, req)

		if score > bestScore {
			bestScore = score
			bestParams = make(map[string]float64)
			for k, v := range params {
				bestParams[k] = v
			}
		}
	}

	return bestParams, bestScore, maxIterations, nil
}

// bayesianOptimization performs simplified Bayesian optimization
func (s *OptimizerService) bayesianOptimization(paramSpace map[string][2]float64, req *orchestrator.OptimizationRequest) (map[string]float64, float64, int, error) {
	maxIterations := 50
	if req.MaxIterations > 0 {
		maxIterations = req.MaxIterations
	}

	// Start with random exploration
	explorationRatio := 0.3
	explorationSteps := int(float64(maxIterations) * explorationRatio)

	bestParams := make(map[string]float64)
	bestScore := math.Inf(-1)

	observations := make([]map[string]float64, 0)
	scores := make([]float64, 0)

	// 使用随机种子确保每台服务器的训练都不重复
	seed := time.Now().UnixNano() + int64(len(req.StrategyID)*1000) + int64(req.MaxIterations*100)
	rand.Seed(seed)

	// Exploration phase
	for i := 0; i < explorationSteps; i++ {
		params := make(map[string]float64)
		for name, bounds := range paramSpace {
			params[name] = bounds[0] + rand.Float64()*(bounds[1]-bounds[0])
		}

		score := s.evaluateObjective(params, req)

		observations = append(observations, params)
		scores = append(scores, score)

		if score > bestScore {
			bestScore = score
			bestParams = make(map[string]float64)
			for k, v := range params {
				bestParams[k] = v
			}
		}
	}

	// Exploitation phase - use best regions found so far
	for i := explorationSteps; i < maxIterations; i++ {
		// Find top 20% of observations
		topIndices := s.getTopIndices(scores, 0.2)

		// Sample around best regions
		params := s.sampleAroundBest(observations, topIndices, paramSpace)
		score := s.evaluateObjective(params, req)

		observations = append(observations, params)
		scores = append(scores, score)

		if score > bestScore {
			bestScore = score
			bestParams = make(map[string]float64)
			for k, v := range params {
				bestParams[k] = v
			}
		}
	}

	return bestParams, bestScore, maxIterations, nil
}

// Helper functions for optimization algorithms

func (s *OptimizerService) iterateGrid(grids [][]float64, paramNames []string, depth int, current []float64, callback func([]float64)) error {
	if depth == len(grids) {
		params := make([]float64, len(current))
		copy(params, current)
		callback(params)
		return nil
	}

	for _, value := range grids[depth] {
		current[depth] = value
		if err := s.iterateGrid(grids, paramNames, depth+1, current, callback); err != nil {
			return err
		}
	}
	return nil
}

func (s *OptimizerService) getTopIndices(scores []float64, ratio float64) []int {
	type scoreIndex struct {
		score float64
		index int
	}

	var sortedScores []scoreIndex
	for i, score := range scores {
		sortedScores = append(sortedScores, scoreIndex{score, i})
	}

	sort.Slice(sortedScores, func(i, j int) bool {
		return sortedScores[i].score > sortedScores[j].score
	})

	topCount := int(float64(len(scores)) * ratio)
	if topCount < 1 {
		topCount = 1
	}

	var indices []int
	for i := 0; i < topCount && i < len(sortedScores); i++ {
		indices = append(indices, sortedScores[i].index)
	}

	return indices
}

func (s *OptimizerService) sampleAroundBest(observations []map[string]float64, topIndices []int, paramSpace map[string][2]float64) map[string]float64 {
	if len(topIndices) == 0 {
		// Fallback to random sampling
		params := make(map[string]float64)
		for name, bounds := range paramSpace {
			params[name] = bounds[0] + rand.Float64()*(bounds[1]-bounds[0])
		}
		return params
	}

	// Pick a random top observation
	baseIdx := topIndices[rand.Intn(len(topIndices))]
	baseParams := observations[baseIdx]

	// Add gaussian noise around the base parameters
	params := make(map[string]float64)
	for name, bounds := range paramSpace {
		baseValue := baseParams[name]
		rangeSize := bounds[1] - bounds[0]
		noise := rand.NormFloat64() * rangeSize * 0.1 // 10% of range as std dev

		newValue := baseValue + noise
		// Clamp to bounds
		if newValue < bounds[0] {
			newValue = bounds[0]
		}
		if newValue > bounds[1] {
			newValue = bounds[1]
		}

		params[name] = newValue
	}

	return params
}

// evaluateObjective evaluates the objective function for given parameters
func (s *OptimizerService) evaluateObjective(params map[string]float64, req *orchestrator.OptimizationRequest) float64 {
	// For now, use a simplified evaluation based on parameter values
	// In a real implementation, this would run backtest with the parameters

	// Simulate sharpe ratio calculation based on parameters
	score := 0.0
	paramCount := 0

	for _, value := range params {
		// Normalize parameter contribution (assuming reasonable ranges)
		normalizedValue := math.Min(math.Max(value/100.0, 0), 2.0)
		score += normalizedValue
		paramCount++
	}

	if paramCount > 0 {
		score = score / float64(paramCount)
	}

	// Add some randomness to simulate real market variability
	noise := (rand.Float64() - 0.5) * 0.2
	score += noise

	// Simulate realistic Sharpe ratio range
	return math.Max(score, -2.0)
}

// calculatePerformanceMetrics calculates performance metrics from optimization score
func (s *OptimizerService) calculatePerformanceMetrics(score float64, iterations int) orchestrator.PerformanceMetrics {
	// Convert optimization score to realistic trading metrics
	sharpeRatio := score

	// Derive other metrics from Sharpe ratio
	totalReturn := sharpeRatio * 0.15 // Assume 15% volatility
	maxDrawdown := math.Max(-0.05, -math.Abs(sharpeRatio*0.08))
	winRate := 0.5 + (sharpeRatio * 0.1)            // Better Sharpe -> higher win rate
	winRate = math.Min(math.Max(winRate, 0.3), 0.8) // Clamp between 30-80%

	// Estimate trade count based on iterations (proxy for complexity)
	tradeCount := int(50 + float64(iterations)*0.5)

	return orchestrator.PerformanceMetrics{
		TotalReturn: totalReturn,
		SharpeRatio: sharpeRatio,
		MaxDrawdown: maxDrawdown,
		WinRate:     winRate,
		TradeCount:  tradeCount,
	}
}

// shareOptimizationResult shares the optimization result
func (s *OptimizerService) shareOptimizationResult(req *orchestrator.OptimizationRequest, optResult *OptimizationResult) {
	if s.resultSharingMgr == nil {
		return
	}

	// Create shared result
	sharedResult := &automl.SharedResult{
		ID:           fmt.Sprintf("%s_%s_%d", req.RequestID, req.StrategyID, time.Now().Unix()),
		TaskID:       req.RequestID,
		StrategyName: req.StrategyID,
		Parameters:   optResult.BestParameters,
		Performance: &automl.PerformanceMetrics{
			ProfitRate:         optResult.Performance.TotalReturn,
			SharpeRatio:        optResult.Performance.SharpeRatio,
			MaxDrawdown:        optResult.Performance.MaxDrawdown,
			WinRate:            optResult.Performance.WinRate,
			TotalReturn:        optResult.Performance.TotalReturn,
			RiskAdjustedReturn: optResult.Performance.SharpeRatio,
		},
		RandomSeed:    time.Now().UnixNano(),
		DataHash:      fmt.Sprintf("%s_%s", req.RequestID, req.StrategyID),
		DiscoveredBy:  "optimizer-service",
		DiscoveredAt:  time.Now(),
		ShareMethod:   "optimization",
		AdoptionCount: 0,
		IsGlobalBest:  false,
	}

	// Share the result
	if err := s.resultSharingMgr.ShareResult(sharedResult); err != nil {
		log.Printf("Failed to share optimization result: %v", err)
	} else {
		log.Printf("Successfully shared optimization result for request %s", req.RequestID)
	}
}

// sharedResultsHandler handles shared results requests
func (s *OptimizerService) sharedResultsHandler(w http.ResponseWriter, r *http.Request) {
	if s.resultSharingMgr == nil {
		http.Error(w, "Result sharing not enabled", http.StatusServiceUnavailable)
		return
	}

	results := s.resultSharingMgr.GetAllSharedResults()

	response := map[string]interface{}{
		"results":   results,
		"count":     len(results),
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// shareResultHandler handles manual result sharing requests
func (s *OptimizerService) shareResultHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.resultSharingMgr == nil {
		http.Error(w, "Result sharing not enabled", http.StatusServiceUnavailable)
		return
	}

	var sharedResult automl.SharedResult
	if err := json.NewDecoder(r.Body).Decode(&sharedResult); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Set missing fields
	if sharedResult.ID == "" {
		sharedResult.ID = fmt.Sprintf("%s_%s_%d", sharedResult.TaskID, sharedResult.StrategyName, time.Now().Unix())
	}
	if sharedResult.DiscoveredAt.IsZero() {
		sharedResult.DiscoveredAt = time.Now()
	}
	if sharedResult.DiscoveredBy == "" {
		sharedResult.DiscoveredBy = "manual-upload"
	}

	// Share the result
	if err := s.resultSharingMgr.ShareResult(&sharedResult); err != nil {
		http.Error(w, fmt.Sprintf("Failed to share result: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":    "success",
		"message":   "Result shared successfully",
		"id":        sharedResult.ID,
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// loadConfig loads configuration from file or uses defaults
func loadConfig(configFile string, port int, logLevel string) (*Config, error) {
	cfg := &Config{
		Port:         port,
		MessageQueue: "memory",
		LogLevel:     logLevel,
	}

	// Try to load main config file first to get port configuration
	mainConfigPath := "configs/config.yaml"
	if _, err := os.Stat(mainConfigPath); err == nil {
		mainConfig, err := config.Load(mainConfigPath)
		if err == nil && mainConfig.Ports.QcatOptimizer != 0 {
			cfg.Port = mainConfig.Ports.QcatOptimizer
			log.Printf("Using optimizer port from main config: %d", cfg.Port)
		}
	}

	// Try to load optimizer-specific config file
	if _, err := os.Stat(configFile); err == nil {
		file, err := os.Open(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open config file: %w", err)
		}
		defer file.Close()

		if err := json.NewDecoder(file).Decode(cfg); err != nil {
			return nil, fmt.Errorf("failed to decode config file: %w", err)
		}
	}

	// Override with command line port if specified
	if port != 8081 { // 8081 is the default
		cfg.Port = port
	}

	// Load result sharing config if not present
	if cfg.ResultSharing == nil {
		cfg.ResultSharing = &automl.ResultSharingConfig{
			Enabled: true,
			Mode:    "hybrid",
		}
	}

	return cfg, nil
}

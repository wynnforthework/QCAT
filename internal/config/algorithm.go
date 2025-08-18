package config

import (
	"fmt"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// AlgorithmConfig represents algorithm configuration
type AlgorithmConfig struct {
	Optimizer     OptimizerConfig     `yaml:"optimizer"`
	Elimination   EliminationConfig   `yaml:"elimination"`
	RiskMgmt      RiskManagementConfig `yaml:"risk_management"`
	Overfitting   OverfittingConfig   `yaml:"overfitting"`
	Performance   PerformanceConfig   `yaml:"performance"`
	MarketData    MarketDataConfig    `yaml:"market_data"`
	Hotlist       HotlistConfig       `yaml:"hotlist"`
	Monitoring    MonitoringConfig    `yaml:"monitoring"`
	Backtesting   BacktestingConfig   `yaml:"backtesting"`
	Automation    AutomationConfig    `yaml:"automation"`
	Logging       LoggingConfig       `yaml:"logging"`
	
	// Internal fields
	mu       sync.RWMutex
	filePath string
	lastMod  time.Time
}

// OptimizerConfig represents optimizer configuration
type OptimizerConfig struct {
	GridSearch   GridSearchConfig   `yaml:"grid_search"`
	Bayesian     BayesianConfig     `yaml:"bayesian"`
	CMAES        CMAESConfig        `yaml:"cma_es"`
	Genetic      GeneticConfig      `yaml:"genetic"`
	WalkForward  WalkForwardConfig  `yaml:"walk_forward"`
}

// GridSearchConfig represents grid search configuration
type GridSearchConfig struct {
	DefaultGridSize       int     `yaml:"default_grid_size"`
	MaxIterations         int     `yaml:"max_iterations"`
	ConvergenceThreshold  float64 `yaml:"convergence_threshold"`
}

// BayesianConfig represents Bayesian optimization configuration
type BayesianConfig struct {
	AcquisitionFunction string `yaml:"acquisition_function"`
	NInitialPoints      int    `yaml:"n_initial_points"`
	NCalls              int    `yaml:"n_calls"`
	RandomState         int    `yaml:"random_state"`
}

// CMAESConfig represents CMA-ES configuration
type CMAESConfig struct {
	PopulationSize int     `yaml:"population_size"`
	Sigma          float64 `yaml:"sigma"`
	MaxGenerations int     `yaml:"max_generations"`
}

// GeneticConfig represents genetic algorithm configuration
type GeneticConfig struct {
	PopulationSize int     `yaml:"population_size"`
	MutationRate   float64 `yaml:"mutation_rate"`
	CrossoverRate  float64 `yaml:"crossover_rate"`
	MaxGenerations int     `yaml:"max_generations"`
	EliteSize      int     `yaml:"elite_size"`
}

// WalkForwardConfig represents walk-forward optimization configuration
type WalkForwardConfig struct {
	TrainRatio      float64 `yaml:"train_ratio"`
	ValidationRatio float64 `yaml:"validation_ratio"`
	TestRatio       float64 `yaml:"test_ratio"`
	MinSamples      int     `yaml:"min_samples"`
	StepSize        int     `yaml:"step_size"`
}

// EliminationConfig represents strategy elimination configuration
type EliminationConfig struct {
	WindowSizeDays        int     `yaml:"window_size_days"`
	MinTrades             int     `yaml:"min_trades"`
	PerformanceThreshold  float64 `yaml:"performance_threshold"`
	CorrelationThreshold  float64 `yaml:"correlation_threshold"`
	VolatilityThreshold   float64 `yaml:"volatility_threshold"`
	Bandit                BanditConfig `yaml:"bandit"`
}

// BanditConfig represents multi-armed bandit configuration
type BanditConfig struct {
	ExplorationRate float64 `yaml:"exploration_rate"`
	DecayRate       float64 `yaml:"decay_rate"`
	MinExploration  float64 `yaml:"min_exploration"`
	ConfidenceLevel float64 `yaml:"confidence_level"`
}

// RiskManagementConfig represents risk management configuration
type RiskManagementConfig struct {
	Position PositionConfig `yaml:"position"`
	StopLoss StopLossConfig `yaml:"stop_loss"`
	Margin   MarginConfig   `yaml:"margin"`
}

// PositionConfig represents position sizing configuration
type PositionConfig struct {
	MaxWeightPercent    float64 `yaml:"max_weight_percent"`
	MinWeightPercent    float64 `yaml:"min_weight_percent"`
	RebalanceThreshold  float64 `yaml:"rebalance_threshold"`
	MaxLeverage         int     `yaml:"max_leverage"`
}

// StopLossConfig represents stop loss configuration
type StopLossConfig struct {
	DefaultATRMultiplier  float64 `yaml:"default_atr_multiplier"`
	TrailingStopPercent   float64 `yaml:"trailing_stop_percent"`
	MaxStopLossPercent    float64 `yaml:"max_stop_loss_percent"`
	MinStopLossPercent    float64 `yaml:"min_stop_loss_percent"`
}

// MarginConfig represents margin management configuration
type MarginConfig struct {
	MaxMarginRatio         float64 `yaml:"max_margin_ratio"`
	WarningMarginRatio     float64 `yaml:"warning_margin_ratio"`
	MaintenanceMarginRatio float64 `yaml:"maintenance_margin_ratio"`
	LiquidationMarginRatio float64 `yaml:"liquidation_margin_ratio"`
}

// OverfittingConfig represents overfitting detection configuration
type OverfittingConfig struct {
	MinSamples                int                    `yaml:"min_samples"`
	ConfidenceLevel           float64                `yaml:"confidence_level"`
	PBOThreshold              float64                `yaml:"pbo_threshold"`
	DeflatedSharpeThreshold   float64                `yaml:"deflated_sharpe_threshold"`
	SensitivityThreshold      float64                `yaml:"sensitivity_threshold"`
	CrossValidation           CrossValidationConfig  `yaml:"cross_validation"`
}

// CrossValidationConfig represents cross-validation configuration
type CrossValidationConfig struct {
	NFolds      int  `yaml:"n_folds"`
	Shuffle     bool `yaml:"shuffle"`
	RandomState int  `yaml:"random_state"`
}

// PerformanceConfig represents performance metrics configuration
type PerformanceConfig struct {
	RiskFreeRate         float64           `yaml:"risk_free_rate"`
	TradingDaysPerYear   int               `yaml:"trading_days_per_year"`
	Benchmark            BenchmarkConfig   `yaml:"benchmark"`
	Metrics              MetricsConfig     `yaml:"metrics"`
}

// BenchmarkConfig represents benchmark configuration
type BenchmarkConfig struct {
	Symbol              string `yaml:"symbol"`
	RebalanceFrequency  string `yaml:"rebalance_frequency"`
}

// MetricsConfig represents metrics calculation configuration
type MetricsConfig struct {
	RollingWindowDays        int       `yaml:"rolling_window_days"`
	VaRConfidenceLevels      []float64 `yaml:"var_confidence_levels"`
	MaxDrawdownLookbackDays  int       `yaml:"max_drawdown_lookback_days"`
}

// MarketDataConfig represents market data configuration
type MarketDataConfig struct {
	Quality   QualityConfig   `yaml:"quality"`
	Sampling  SamplingConfig  `yaml:"sampling"`
	Storage   StorageConfig   `yaml:"storage"`
}

// QualityConfig represents data quality configuration
type QualityConfig struct {
	MaxLatencyMs           int     `yaml:"max_latency_ms"`
	MaxGapDurationSeconds  int     `yaml:"max_gap_duration_seconds"`
	MinQualityScore        float64 `yaml:"min_quality_score"`
	OutlierThresholdStd    float64 `yaml:"outlier_threshold_std"`
}

// SamplingConfig represents data sampling configuration
type SamplingConfig struct {
	TickSampleRate    int      `yaml:"tick_sample_rate"`
	KlineIntervals    []string `yaml:"kline_intervals"`
	OrderbookDepth    int      `yaml:"orderbook_depth"`
}

// StorageConfig represents data storage configuration
type StorageConfig struct {
	RetentionDays      int  `yaml:"retention_days"`
	CompressionEnabled bool `yaml:"compression_enabled"`
	BatchSize          int  `yaml:"batch_size"`
}

// HotlistConfig represents hotlist configuration
type HotlistConfig struct {
	Scoring              ScoringConfig    `yaml:"scoring"`
	Thresholds           ThresholdsConfig `yaml:"thresholds"`
	UpdateIntervalMinutes int             `yaml:"update_interval_minutes"`
	TopNSymbols          int             `yaml:"top_n_symbols"`
}

// ScoringConfig represents scoring weights configuration
type ScoringConfig struct {
	VolumeWeight        float64 `yaml:"volume_weight"`
	VolatilityWeight    float64 `yaml:"volatility_weight"`
	MomentumWeight      float64 `yaml:"momentum_weight"`
	FundingRateWeight   float64 `yaml:"funding_rate_weight"`
	OpenInterestWeight  float64 `yaml:"open_interest_weight"`
}

// ThresholdsConfig represents threshold configuration
type ThresholdsConfig struct {
	MinVolume24h    float64 `yaml:"min_volume_24h"`
	MinMarketCap    float64 `yaml:"min_market_cap"`
	MaxVolatility   float64 `yaml:"max_volatility"`
}



// BacktestingConfig represents backtesting configuration
type BacktestingConfig struct {
	Default   DefaultBacktestConfig   `yaml:"default"`
	Execution ExecutionConfig         `yaml:"execution"`
	Risk      RiskSimulationConfig    `yaml:"risk"`
}

// DefaultBacktestConfig represents default backtesting parameters
type DefaultBacktestConfig struct {
	InitialCapital  float64 `yaml:"initial_capital"`
	CommissionRate  float64 `yaml:"commission_rate"`
	SlippageRate    float64 `yaml:"slippage_rate"`
}

// ExecutionConfig represents execution simulation configuration
type ExecutionConfig struct {
	FillProbability      float64 `yaml:"fill_probability"`
	PartialFillEnabled   bool    `yaml:"partial_fill_enabled"`
	MarketImpactModel    string  `yaml:"market_impact_model"`
	LatencySimulationMs  int     `yaml:"latency_simulation_ms"`
}

// RiskSimulationConfig represents risk simulation configuration
type RiskSimulationConfig struct {
	MarginCallEnabled   bool `yaml:"margin_call_enabled"`
	LiquidationEnabled  bool `yaml:"liquidation_enabled"`
	FundingRateEnabled  bool `yaml:"funding_rate_enabled"`
}

// AutomationConfig represents automation configuration
type AutomationConfig struct {
	Rebalancing     RebalancingConfig     `yaml:"rebalancing"`
	Optimization    OptimizationConfig    `yaml:"optimization"`
	RiskMonitoring  RiskMonitoringConfig  `yaml:"risk_monitoring"`
}

// RebalancingConfig represents rebalancing configuration
type RebalancingConfig struct {
	Frequency           string  `yaml:"frequency"`
	TimeUTC             string  `yaml:"time_utc"`
	MinRebalanceAmount  float64 `yaml:"min_rebalance_amount"`
}

// OptimizationConfig represents optimization scheduling configuration
type OptimizationConfig struct {
	Frequency   string `yaml:"frequency"`
	DayOfWeek   string `yaml:"day_of_week"`
	TimeUTC     string `yaml:"time_utc"`
}

// RiskMonitoringConfig represents risk monitoring configuration
type RiskMonitoringConfig struct {
	CheckIntervalSeconds  int `yaml:"check_interval_seconds"`
	AlertCooldownMinutes  int `yaml:"alert_cooldown_minutes"`
}



// Global algorithm configuration instance
var (
	algorithmConfig *AlgorithmConfig
	algorithmMu     sync.RWMutex
)

// LoadAlgorithmConfig loads algorithm configuration from file
func LoadAlgorithmConfig(configPath string) (*AlgorithmConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read algorithm config file: %w", err)
	}

	var config AlgorithmConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse algorithm config: %w", err)
	}

	// Set internal fields
	config.filePath = configPath
	if stat, err := os.Stat(configPath); err == nil {
		config.lastMod = stat.ModTime()
	}

	// Set global instance
	algorithmMu.Lock()
	algorithmConfig = &config
	algorithmMu.Unlock()

	return &config, nil
}

// GetAlgorithmConfig returns the global algorithm configuration
func GetAlgorithmConfig() *AlgorithmConfig {
	algorithmMu.RLock()
	defer algorithmMu.RUnlock()
	return algorithmConfig
}

// ReloadAlgorithmConfig reloads the algorithm configuration if file has changed
func ReloadAlgorithmConfig() error {
	algorithmMu.RLock()
	if algorithmConfig == nil {
		algorithmMu.RUnlock()
		return fmt.Errorf("algorithm config not loaded")
	}
	
	filePath := algorithmConfig.filePath
	lastMod := algorithmConfig.lastMod
	algorithmMu.RUnlock()

	// Check if file has been modified
	stat, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat config file: %w", err)
	}

	if !stat.ModTime().After(lastMod) {
		return nil // No changes
	}

	// Reload configuration
	_, err = LoadAlgorithmConfig(filePath)
	return err
}

// Validate validates the algorithm configuration
func (c *AlgorithmConfig) Validate() error {
	// Validate optimizer configuration
	if c.Optimizer.GridSearch.DefaultGridSize <= 0 {
		return fmt.Errorf("grid search default_grid_size must be positive")
	}
	
	if c.Optimizer.WalkForward.TrainRatio + c.Optimizer.WalkForward.ValidationRatio + c.Optimizer.WalkForward.TestRatio != 1.0 {
		return fmt.Errorf("walk forward ratios must sum to 1.0")
	}

	// Validate elimination configuration
	if c.Elimination.WindowSizeDays <= 0 {
		return fmt.Errorf("elimination window_size_days must be positive")
	}

	// Validate risk management configuration
	if c.RiskMgmt.Position.MaxWeightPercent <= 0 || c.RiskMgmt.Position.MaxWeightPercent > 100 {
		return fmt.Errorf("max_weight_percent must be between 0 and 100")
	}

	// Validate performance configuration
	if c.Performance.TradingDaysPerYear <= 0 {
		return fmt.Errorf("trading_days_per_year must be positive")
	}

	return nil
}

// GetGridSize returns the configured grid size for optimization
func (c *AlgorithmConfig) GetGridSize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Optimizer.GridSearch.DefaultGridSize
}

// GetWindowSize returns the configured window size for elimination
func (c *AlgorithmConfig) GetWindowSize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Elimination.WindowSizeDays
}

// GetMaxWeight returns the configured maximum weight percentage
func (c *AlgorithmConfig) GetMaxWeight() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.RiskMgmt.Position.MaxWeightPercent / 100.0 // Convert to decimal
}

// GetATRMultiplier returns the configured ATR multiplier for stop loss
func (c *AlgorithmConfig) GetATRMultiplier() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.RiskMgmt.StopLoss.DefaultATRMultiplier
}

// GetMinSamples returns the configured minimum samples for overfitting detection
func (c *AlgorithmConfig) GetMinSamples() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Overfitting.MinSamples
}

// UpdateConfig updates specific configuration values
func (c *AlgorithmConfig) UpdateConfig(updates map[string]interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// This is a simplified update mechanism
	// In a real implementation, you'd want more sophisticated path-based updates
	for key, value := range updates {
		switch key {
		case "optimizer.grid_search.default_grid_size":
			if v, ok := value.(int); ok {
				c.Optimizer.GridSearch.DefaultGridSize = v
			}
		case "elimination.window_size_days":
			if v, ok := value.(int); ok {
				c.Elimination.WindowSizeDays = v
			}
		case "risk_management.position.max_weight_percent":
			if v, ok := value.(float64); ok {
				c.RiskMgmt.Position.MaxWeightPercent = v
			}
		// Add more cases as needed
		}
	}

	return nil
}
package backtesting

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"qcat/internal/config"
	"qcat/internal/market/kline"
)

// AutoBacktestingEngine 自动回测验证引擎
type AutoBacktestingEngine struct {
	config              *config.Config
	dataManager         *BacktestDataManager
	strategyManager     *BacktestStrategyManager
	performanceAnalyzer *PerformanceAnalyzer
	reportGenerator     *ReportGenerator
	walkForwardEngine   *WalkForwardEngine

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
	mu        sync.RWMutex

	// 回测配置
	frequency            time.Duration
	lookbackPeriod       time.Duration
	walkForwardWindow    time.Duration
	performanceThreshold float64

	// 回测状态
	activeBacktests     map[string]*BacktestJob
	completedBacktests  []BacktestResult
	strategyPerformance map[string]*StrategyPerformance

	// 监控指标
	backtestingMetrics *BacktestingMetrics
	validationHistory  []ValidationResult

	// 配置参数
	enabled           bool
	maxConcurrentJobs int
	dataRetentionDays int
	backtestTimeout   time.Duration
	maxIterations     int
}

// BacktestJob 回测任务
type BacktestJob struct {
	ID             string                 `json:"id"`
	StrategyID     string                 `json:"strategy_id"`
	StrategyName   string                 `json:"strategy_name"`
	StartDate      time.Time              `json:"start_date"`
	EndDate        time.Time              `json:"end_date"`
	InitialCapital float64                `json:"initial_capital"`
	Parameters     map[string]interface{} `json:"parameters"`

	// 执行状态
	Status    string        `json:"status"`   // PENDING, RUNNING, COMPLETED, FAILED
	Progress  float64       `json:"progress"` // 0.0 - 1.0
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`

	// 配置选项
	Commission      float64 `json:"commission"`
	Slippage        float64 `json:"slippage"`
	BenchmarkSymbol string  `json:"benchmark_symbol"`
	RiskFreeRate    float64 `json:"risk_free_rate"`

	// 结果
	Result       *BacktestResult `json:"result"`
	ErrorMessage string          `json:"error_message"`

	// 元数据
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy string    `json:"created_by"`
	JobType   string    `json:"job_type"` // SINGLE, WALK_FORWARD, PARAMETER_SWEEP
}

// BacktestResult 回测结果
type BacktestResult struct {
	JobID      string `json:"job_id"`
	StrategyID string `json:"strategy_id"`

	// 基本统计
	TotalReturn      float64 `json:"total_return"`
	AnnualizedReturn float64 `json:"annualized_return"`
	Volatility       float64 `json:"volatility"`
	SharpeRatio      float64 `json:"sharpe_ratio"`
	SortinoRatio     float64 `json:"sortino_ratio"`
	CalmarRatio      float64 `json:"calmar_ratio"`
	MaxDrawdown      float64 `json:"max_drawdown"`

	// 交易统计
	TotalTrades   int     `json:"total_trades"`
	WinningTrades int     `json:"winning_trades"`
	LosingTrades  int     `json:"losing_trades"`
	WinRate       float64 `json:"win_rate"`
	ProfitFactor  float64 `json:"profit_factor"`
	AvgWin        float64 `json:"avg_win"`
	AvgLoss       float64 `json:"avg_loss"`
	LargestWin    float64 `json:"largest_win"`
	LargestLoss   float64 `json:"largest_loss"`

	// 时间序列数据
	EquityCurve    []EquityPoint   `json:"equity_curve"`
	DrawdownCurve  []DrawdownPoint `json:"drawdown_curve"`
	BenchmarkCurve []EquityPoint   `json:"benchmark_curve"`

	// 详细交易记录
	Trades []TradeRecord `json:"trades"`

	// 风险指标
	VaR95            float64 `json:"var_95"`
	CVaR95           float64 `json:"cvar_95"`
	Beta             float64 `json:"beta"`
	Alpha            float64 `json:"alpha"`
	TrackingError    float64 `json:"tracking_error"`
	InformationRatio float64 `json:"information_ratio"`

	// 稳定性指标
	ConsistencyScore float64 `json:"consistency_score"`
	RobustnessScore  float64 `json:"robustness_score"`

	// 元数据
	BacktestDate time.Time `json:"backtest_date"`
	DataPeriod   DateRange `json:"data_period"`
	Benchmark    string    `json:"benchmark"`

	// 验证结果
	ValidationResult *ValidationResult `json:"validation_result"`

	// 风险指标
	RiskMetrics *RiskMetrics `json:"risk_metrics"`
}

// EquityPoint 净值点
type EquityPoint struct {
	Date   time.Time `json:"date"`
	Value  float64   `json:"value"`
	Return float64   `json:"return"`
}

// DrawdownPoint 回撤点
type DrawdownPoint struct {
	Date     time.Time `json:"date"`
	Value    float64   `json:"value"`
	Drawdown float64   `json:"drawdown"`
	Duration int       `json:"duration"` // 回撤持续天数
}

// TradeRecord 交易记录
type TradeRecord struct {
	ID         string        `json:"id"`
	Symbol     string        `json:"symbol"`
	Side       string        `json:"side"` // BUY, SELL
	Quantity   float64       `json:"quantity"`
	EntryPrice float64       `json:"entry_price"`
	ExitPrice  float64       `json:"exit_price"`
	EntryTime  time.Time     `json:"entry_time"`
	ExitTime   time.Time     `json:"exit_time"`
	Duration   time.Duration `json:"duration"`
	PnL        float64       `json:"pnl"`
	PnLPercent float64       `json:"pnl_percent"`
	Commission float64       `json:"commission"`
	Slippage   float64       `json:"slippage"`
	MAE        float64       `json:"mae"` // Maximum Adverse Excursion
	MFE        float64       `json:"mfe"` // Maximum Favorable Excursion
	Tags       []string      `json:"tags"`
}

// DateRange 日期范围
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// BacktestDataManager 回测数据管理器
type BacktestDataManager struct {
	dataSource string
	symbols    []string
	timeframes []string
	dataCache  map[string][]Candle
	lastUpdate time.Time

	// 数据源接口
	db           DatabaseInterface
	apiClient    APIClientInterface
	klineManager KlineManagerInterface // 新增：K线管理器接口

	mu sync.RWMutex
}

// DatabaseInterface 数据库接口
type DatabaseInterface interface {
	QueryHistoricalData(ctx context.Context, symbol string, startDate, endDate time.Time) ([]Candle, error)
}

// APIClientInterface API客户端接口
type APIClientInterface interface {
	GetHistoricalKlines(ctx context.Context, symbol, interval string, startTime, endTime time.Time, limit int) ([]Candle, error)
}

// KlineManagerInterface K线管理器接口
type KlineManagerInterface interface {
	GetHistoryWithBackfill(ctx context.Context, symbol string, interval string, start, end time.Time) ([]KlineData, error)
	EnsureDataAvailable(ctx context.Context, symbol string, interval string, start, end time.Time) error
}

// KlineData K线数据结构（用于回测）
type KlineData struct {
	Symbol    string
	OpenTime  time.Time
	CloseTime time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// Candle K线数据
type Candle struct {
	Symbol string    `json:"symbol"`
	Time   time.Time `json:"time"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume float64   `json:"volume"`
}

// BacktestStrategyManager 回测策略管理器
type BacktestStrategyManager struct {
	strategies     map[string]*BacktestStrategy
	strategyLoader StrategyLoader

	mu sync.RWMutex
}

// BacktestStrategy 回测策略
type BacktestStrategy struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Version     string               `json:"version"`
	Description string               `json:"description"`
	Parameters  map[string]Parameter `json:"parameters"`
	Logic       StrategyLogic        `json:"-"`

	// 性能统计
	BacktestCount    int       `json:"backtest_count"`
	AvgPerformance   float64   `json:"avg_performance"`
	BestPerformance  float64   `json:"best_performance"`
	WorstPerformance float64   `json:"worst_performance"`
	LastBacktest     time.Time `json:"last_backtest"`

	// 配置
	Symbols   []string  `json:"symbols"`
	Timeframe string    `json:"timeframe"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Parameter 策略参数
type Parameter struct {
	Name          string      `json:"name"`
	Type          string      `json:"type"` // int, float, bool, string
	Value         interface{} `json:"value"`
	Min           interface{} `json:"min"`
	Max           interface{} `json:"max"`
	Step          interface{} `json:"step"`
	Description   string      `json:"description"`
	IsOptimizable bool        `json:"is_optimizable"`
}

// StrategyLogic 策略逻辑接口
type StrategyLogic interface {
	Initialize(params map[string]interface{}) error
	ProcessBar(candle Candle, portfolio *Portfolio) (*Signal, error)
	Finalize(portfolio *Portfolio) error
}

// Signal 交易信号
type Signal struct {
	Symbol     string                 `json:"symbol"`
	Action     string                 `json:"action"` // BUY, SELL, HOLD
	Quantity   float64                `json:"quantity"`
	Price      float64                `json:"price"`
	StopLoss   float64                `json:"stop_loss"`
	TakeProfit float64                `json:"take_profit"`
	Reason     string                 `json:"reason"`
	Metadata   map[string]interface{} `json:"metadata"`
	Timestamp  time.Time              `json:"timestamp"`
}

// Portfolio 组合状态
type Portfolio struct {
	Cash      float64              `json:"cash"`
	Positions map[string]*Position `json:"positions"`
	Equity    float64              `json:"equity"`
	UpdatedAt time.Time            `json:"updated_at"`
}

// Position 仓位
type Position struct {
	Symbol       string    `json:"symbol"`
	Quantity     float64   `json:"quantity"`
	AvgPrice     float64   `json:"avg_price"`
	MarketPrice  float64   `json:"market_price"`
	UnrealizedPL float64   `json:"unrealized_pl"`
	OpenTime     time.Time `json:"open_time"`
}

// StrategyLoader 策略加载器接口
type StrategyLoader interface {
	LoadStrategy(id string) (*BacktestStrategy, error)
	ListStrategies() ([]string, error)
}

// PerformanceAnalyzer 性能分析器
type PerformanceAnalyzer struct {
	benchmarkData map[string][]EquityPoint
	riskFreeRate  float64
	analysisCache map[string]*PerformanceAnalysis

	mu sync.RWMutex
}

// PerformanceAnalysis 性能分析
type PerformanceAnalysis struct {
	Returns           []float64          `json:"returns"`
	CumulativeReturns []float64          `json:"cumulative_returns"`
	RollingStats      []RollingStats     `json:"rolling_stats"`
	MonthlyReturns    map[string]float64 `json:"monthly_returns"`
	YearlyReturns     map[string]float64 `json:"yearly_returns"`

	// 风险指标
	RiskMetrics RiskMetrics `json:"risk_metrics"`

	// 稳定性分析
	StabilityAnalysis StabilityAnalysis `json:"stability_analysis"`
}

// RollingStats 滚动统计
type RollingStats struct {
	Date        time.Time `json:"date"`
	Return      float64   `json:"return"`
	Volatility  float64   `json:"volatility"`
	SharpeRatio float64   `json:"sharpe_ratio"`
	MaxDrawdown float64   `json:"max_drawdown"`
	WinRate     float64   `json:"win_rate"`
}

// RiskMetrics 风险指标
type RiskMetrics struct {
	VaR95             float64 `json:"var_95"`
	VaR99             float64 `json:"var_99"`
	CVaR95            float64 `json:"cvar_95"`
	CVaR99            float64 `json:"cvar_99"`
	SkewnessRisk      float64 `json:"skewness_risk"`
	KurtosisRisk      float64 `json:"kurtosis_risk"`
	TailRatio         float64 `json:"tail_ratio"`
	DownsideDeviation float64 `json:"downside_deviation"`
}

// StabilityAnalysis 稳定性分析
type StabilityAnalysis struct {
	ConsistencyScore   float64             `json:"consistency_score"`
	RobustnessScore    float64             `json:"robustness_score"`
	AdaptabilityScore  float64             `json:"adaptability_score"`
	OutOfSampleRatio   float64             `json:"out_of_sample_ratio"`
	ForwardTestPeriods []ForwardTestResult `json:"forward_test_periods"`
}

// ForwardTestResult 前向测试结果
type ForwardTestResult struct {
	Period      DateRange `json:"period"`
	Return      float64   `json:"return"`
	Volatility  float64   `json:"volatility"`
	SharpeRatio float64   `json:"sharpe_ratio"`
	MaxDrawdown float64   `json:"max_drawdown"`
	TradeCount  int       `json:"trade_count"`
}

// ReportGenerator 报告生成器
type ReportGenerator struct {
	templatePath string
	outputPath   string
	reportCache  map[string]*BacktestReport

	mu sync.RWMutex
}

// BacktestReport 回测报告
type BacktestReport struct {
	ID          string    `json:"id"`
	JobID       string    `json:"job_id"`
	Title       string    `json:"title"`
	GeneratedAt time.Time `json:"generated_at"`

	// 报告内容
	Summary          ReportSummary    `json:"summary"`
	PerformanceChart []ChartData      `json:"performance_chart"`
	RiskAnalysis     RiskAnalysis     `json:"risk_analysis"`
	TradeAnalysis    TradeAnalysis    `json:"trade_analysis"`
	Recommendations  []Recommendation `json:"recommendations"`

	// 输出格式
	HTMLPath string `json:"html_path"`
	PDFPath  string `json:"pdf_path"`
	JSONPath string `json:"json_path"`
}

// ReportSummary 报告摘要
type ReportSummary struct {
	StrategyName     string    `json:"strategy_name"`
	TestPeriod       DateRange `json:"test_period"`
	TotalReturn      float64   `json:"total_return"`
	AnnualizedReturn float64   `json:"annualized_return"`
	MaxDrawdown      float64   `json:"max_drawdown"`
	SharpeRatio      float64   `json:"sharpe_ratio"`
	WinRate          float64   `json:"win_rate"`
	TotalTrades      int       `json:"total_trades"`
	Grade            string    `json:"grade"` // A+, A, B+, B, C+, C, D, F
}

// ChartData 图表数据
type ChartData struct {
	Name   string      `json:"name"`
	Type   string      `json:"type"` // line, bar, scatter
	Data   []DataPoint `json:"data"`
	Config ChartConfig `json:"config"`
}

// DataPoint 数据点
type DataPoint struct {
	X        interface{}            `json:"x"`
	Y        interface{}            `json:"y"`
	Label    string                 `json:"label"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ChartConfig 图表配置
type ChartConfig struct {
	Title      string                 `json:"title"`
	XAxisLabel string                 `json:"x_axis_label"`
	YAxisLabel string                 `json:"y_axis_label"`
	Color      string                 `json:"color"`
	Options    map[string]interface{} `json:"options"`
}

// RiskAnalysis 风险分析
type RiskAnalysis struct {
	RiskLevel         string             `json:"risk_level"` // LOW, MEDIUM, HIGH, EXTREME
	RiskFactors       []string           `json:"risk_factors"`
	RiskMetrics       RiskMetrics        `json:"risk_metrics"`
	WorstPeriods      []WorstPeriod      `json:"worst_periods"`
	StressTestResults []StressTestResult `json:"stress_test_results"`
}

// WorstPeriod 最差时期
type WorstPeriod struct {
	Period      DateRange `json:"period"`
	Return      float64   `json:"return"`
	Drawdown    float64   `json:"drawdown"`
	Duration    int       `json:"duration"`
	Recovery    int       `json:"recovery"` // 恢复天数
	Description string    `json:"description"`
}

// StressTestResult 压力测试结果
type StressTestResult struct {
	Scenario    string  `json:"scenario"`
	Return      float64 `json:"return"`
	MaxLoss     float64 `json:"max_loss"`
	Recovery    int     `json:"recovery"`
	Probability float64 `json:"probability"`
}

// TradeAnalysis 交易分析
type TradeAnalysis struct {
	TradingFrequency  float64           `json:"trading_frequency"`
	AvgHoldingPeriod  time.Duration     `json:"avg_holding_period"`
	BestTrades        []TradeRecord     `json:"best_trades"`
	WorstTrades       []TradeRecord     `json:"worst_trades"`
	TradeDistribution TradeDistribution `json:"trade_distribution"`
	PatternAnalysis   PatternAnalysis   `json:"pattern_analysis"`
}

// TradeDistribution 交易分布
type TradeDistribution struct {
	PnLHistogram      []HistogramBin `json:"pnl_histogram"`
	DurationHistogram []HistogramBin `json:"duration_histogram"`
	WinLossRatio      float64        `json:"win_loss_ratio"`
	AvgWinSize        float64        `json:"avg_win_size"`
	AvgLossSize       float64        `json:"avg_loss_size"`
}

// HistogramBin 直方图桶
type HistogramBin struct {
	Range      string  `json:"range"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// PatternAnalysis 模式分析
type PatternAnalysis struct {
	SeasonalPatterns  map[string]float64 `json:"seasonal_patterns"`
	WeekdayPatterns   map[string]float64 `json:"weekday_patterns"`
	TimeOfDayPatterns map[string]float64 `json:"time_of_day_patterns"`
	TrendPatterns     map[string]float64 `json:"trend_patterns"`
}

// Recommendation 建议
type Recommendation struct {
	Type        string  `json:"type"`     // PARAMETER, RISK, TIMING
	Priority    string  `json:"priority"` // HIGH, MEDIUM, LOW
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Action      string  `json:"action"`
	Impact      string  `json:"impact"`
	Confidence  float64 `json:"confidence"`
}

// WalkForwardEngine 前进窗口引擎
type WalkForwardEngine struct {
	windowSize    time.Duration
	stepSize      time.Duration
	minTradeCount int

	mu sync.RWMutex
}

// StrategyPerformance 策略表现
type StrategyPerformance struct {
	StrategyID   string `json:"strategy_id"`
	StrategyName string `json:"strategy_name"`

	// 汇总统计
	BacktestCount    int     `json:"backtest_count"`
	AvgReturn        float64 `json:"avg_return"`
	AvgSharpe        float64 `json:"avg_sharpe"`
	AvgMaxDrawdown   float64 `json:"avg_max_drawdown"`
	ConsistencyScore float64 `json:"consistency_score"`
	BestPerformance  float64 `json:"best_performance"`
	WorstPerformance float64 `json:"worst_performance"`

	// 历史表现
	PerformanceHistory []PerformanceRecord `json:"performance_history"`

	// 参数优化
	OptimalParameters    map[string]interface{} `json:"optimal_parameters"`
	ParameterSensitivity map[string]float64     `json:"parameter_sensitivity"`

	// 验证结果
	ValidationStatus string    `json:"validation_status"` // PASSED, FAILED, NEEDS_REVIEW
	ValidationScore  float64   `json:"validation_score"`
	LastValidation   time.Time `json:"last_validation"`

	// 元数据
	UpdatedAt time.Time `json:"updated_at"`
}

// PerformanceRecord 性能记录
type PerformanceRecord struct {
	Date          time.Time `json:"date"`
	Return        float64   `json:"return"`
	SharpeRatio   float64   `json:"sharpe_ratio"`
	MaxDrawdown   float64   `json:"max_drawdown"`
	TradeCount    int       `json:"trade_count"`
	WinRate       float64   `json:"win_rate"`
	BacktestJobID string    `json:"backtest_job_id"`
}

// BacktestingMetrics 回测指标
type BacktestingMetrics struct {
	mu sync.RWMutex

	// 执行统计
	TotalJobs        int64         `json:"total_jobs"`
	CompletedJobs    int64         `json:"completed_jobs"`
	FailedJobs       int64         `json:"failed_jobs"`
	SuccessRate      float64       `json:"success_rate"`
	AvgExecutionTime time.Duration `json:"avg_execution_time"`

	// 性能统计
	AvgStrategyReturn   float64 `json:"avg_strategy_return"`
	BestStrategyReturn  float64 `json:"best_strategy_return"`
	WorstStrategyReturn float64 `json:"worst_strategy_return"`

	// 验证统计
	ValidationPassRate float64 `json:"validation_pass_rate"`
	AvgValidationScore float64 `json:"avg_validation_score"`

	// 系统指标
	ActiveJobs int     `json:"active_jobs"`
	QueuedJobs int     `json:"queued_jobs"`
	SystemLoad float64 `json:"system_load"`

	LastUpdated time.Time `json:"last_updated"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	JobID          string    `json:"job_id"`
	StrategyID     string    `json:"strategy_id"`
	ValidationDate time.Time `json:"validation_date"`

	// 验证测试
	OutOfSampleTest TestResult `json:"out_of_sample_test"`
	ForwardTest     TestResult `json:"forward_test"`
	StabilityTest   TestResult `json:"stability_test"`
	RobustnessTest  TestResult `json:"robustness_test"`

	// 综合评分
	OverallScore float64 `json:"overall_score"`
	OverallGrade string  `json:"overall_grade"`
	Status       string  `json:"status"`

	// 问题和建议
	Issues          []ValidationIssue `json:"issues"`
	Recommendations []string          `json:"recommendations"`

	// 阈值检查
	ThresholdChecks []ThresholdCheck `json:"threshold_checks"`
}

// TestResult 测试结果
type TestResult struct {
	Passed  bool               `json:"passed"`
	Score   float64            `json:"score"`
	Details string             `json:"details"`
	Metrics map[string]float64 `json:"metrics"`
}

// ValidationIssue 验证问题
type ValidationIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

// ThresholdCheck 阈值检查
type ThresholdCheck struct {
	Metric     string  `json:"metric"`
	Value      float64 `json:"value"`
	Threshold  float64 `json:"threshold"`
	Operator   string  `json:"operator"` // >, <, >=, <=, ==
	Passed     bool    `json:"passed"`
	Importance string  `json:"importance"` // CRITICAL, HIGH, MEDIUM, LOW
}

// NewAutoBacktestingEngine 创建自动回测验证引擎
func NewAutoBacktestingEngine(cfg *config.Config) (*AutoBacktestingEngine, error) {
	ctx, cancel := context.WithCancel(context.Background())

	abe := &AutoBacktestingEngine{
		config:               cfg,
		dataManager:          NewBacktestDataManager(),
		strategyManager:      NewBacktestStrategyManager(),
		performanceAnalyzer:  NewPerformanceAnalyzer(),
		reportGenerator:      NewReportGenerator(),
		walkForwardEngine:    NewWalkForwardEngine(),
		ctx:                  ctx,
		cancel:               cancel,
		activeBacktests:      make(map[string]*BacktestJob),
		completedBacktests:   make([]BacktestResult, 0),
		strategyPerformance:  make(map[string]*StrategyPerformance),
		backtestingMetrics:   &BacktestingMetrics{},
		validationHistory:    make([]ValidationResult, 0),
		frequency:            24 * time.Hour,       // 每日回测
		lookbackPeriod:       365 * 24 * time.Hour, // 1年回看期
		walkForwardWindow:    90 * 24 * time.Hour,  // 3个月前进窗口
		performanceThreshold: 0.02,                 // 2%性能阈值
		enabled:              true,
		maxConcurrentJobs:    4,
		dataRetentionDays:    365,
	}

	// 从配置文件读取参数
	if cfg != nil {
		// 从策略配置读取回测参数
		if cfg.Strategy.Backtest.Enabled {
			abe.enabled = cfg.Strategy.Backtest.Enabled
		}
		if cfg.Strategy.Backtest.Timeout > 0 {
			abe.backtestTimeout = cfg.Strategy.Backtest.Timeout
		}
		if cfg.Strategy.Backtest.MaxConcurrency > 0 {
			abe.maxConcurrentJobs = cfg.Strategy.Backtest.MaxConcurrency
		}
		if cfg.Strategy.Backtest.DataRetentionDays > 0 {
			abe.dataRetentionDays = cfg.Strategy.Backtest.DataRetentionDays
		}

		// 从优化器配置读取参数
		if cfg.Optimizer.GridSearch.MaxIterations > 0 {
			abe.maxIterations = cfg.Optimizer.GridSearch.MaxIterations
		}
	}

	return abe, nil
}

// NewAutoBacktestingEngineWithKline 创建带K线管理器的自动回测引擎
func NewAutoBacktestingEngineWithKline(cfg *config.Config, klineManager *kline.Manager) (*AutoBacktestingEngine, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建带K线管理器的数据管理器
	var dataManager *BacktestDataManager
	if klineManager != nil {
		adapter := NewKlineManagerAdapter(klineManager)
		dataManager = NewBacktestDataManagerWithKline(adapter)
	} else {
		dataManager = NewBacktestDataManager()
	}

	abe := &AutoBacktestingEngine{
		config:               cfg,
		dataManager:          dataManager,
		strategyManager:      NewBacktestStrategyManager(),
		performanceAnalyzer:  NewPerformanceAnalyzer(),
		reportGenerator:      NewReportGenerator(),
		walkForwardEngine:    NewWalkForwardEngine(),
		ctx:                  ctx,
		cancel:               cancel,
		activeBacktests:      make(map[string]*BacktestJob),
		completedBacktests:   make([]BacktestResult, 0),
		strategyPerformance:  make(map[string]*StrategyPerformance),
		backtestingMetrics:   &BacktestingMetrics{},
		validationHistory:    make([]ValidationResult, 0),
		frequency:            24 * time.Hour,       // 每日回测
		lookbackPeriod:       365 * 24 * time.Hour, // 1年回看期
		walkForwardWindow:    90 * 24 * time.Hour,  // 3个月前进窗口
		performanceThreshold: 0.02,                 // 2%性能阈值
		enabled:              true,
		maxConcurrentJobs:    4,
		dataRetentionDays:    365,
	}

	// 从配置文件读取参数
	if cfg != nil {
		// 从策略配置读取回测参数
		if cfg.Strategy.Backtest.Enabled {
			abe.enabled = cfg.Strategy.Backtest.Enabled
		}
		if cfg.Strategy.Backtest.Timeout > 0 {
			abe.backtestTimeout = cfg.Strategy.Backtest.Timeout
		}
		if cfg.Strategy.Backtest.MaxConcurrency > 0 {
			abe.maxConcurrentJobs = cfg.Strategy.Backtest.MaxConcurrency
		}
		if cfg.Strategy.Backtest.DataRetentionDays > 0 {
			abe.dataRetentionDays = cfg.Strategy.Backtest.DataRetentionDays
		}

		// 从优化器配置读取参数
		if cfg.Optimizer.GridSearch.MaxIterations > 0 {
			abe.maxIterations = cfg.Optimizer.GridSearch.MaxIterations
		}
	}

	return abe, nil
}

// NewBacktestDataManager 创建回测数据管理器
func NewBacktestDataManager() *BacktestDataManager {
	return &BacktestDataManager{
		dataSource:   "binance",
		symbols:      []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"},
		timeframes:   []string{"1m", "5m", "15m", "1h", "4h", "1d"},
		dataCache:    make(map[string][]Candle),
		db:           nil, // 需要外部设置
		apiClient:    nil, // 需要外部设置
		klineManager: nil, // 需要外部设置
	}
}

// NewBacktestDataManagerWithKline 创建带K线管理器的回测数据管理器
func NewBacktestDataManagerWithKline(klineManager KlineManagerInterface) *BacktestDataManager {
	return &BacktestDataManager{
		dataSource:   "binance",
		symbols:      []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"},
		timeframes:   []string{"1m", "5m", "15m", "1h", "4h", "1d"},
		dataCache:    make(map[string][]Candle),
		klineManager: klineManager,
	}
}

// SetDatabaseClient 设置数据库客户端
func (bdm *BacktestDataManager) SetDatabaseClient(db DatabaseInterface) {
	bdm.mu.Lock()
	defer bdm.mu.Unlock()
	bdm.db = db
}

// SetAPIClient 设置API客户端
func (bdm *BacktestDataManager) SetAPIClient(client APIClientInterface) {
	bdm.mu.Lock()
	defer bdm.mu.Unlock()
	bdm.apiClient = client
}

// NewBacktestStrategyManager 创建回测策略管理器
func NewBacktestStrategyManager() *BacktestStrategyManager {
	return &BacktestStrategyManager{
		strategies: make(map[string]*BacktestStrategy),
	}
}

// NewPerformanceAnalyzer 创建性能分析器
func NewPerformanceAnalyzer() *PerformanceAnalyzer {
	return &PerformanceAnalyzer{
		benchmarkData: make(map[string][]EquityPoint),
		riskFreeRate:  0.02, // 2%无风险利率
		analysisCache: make(map[string]*PerformanceAnalysis),
	}
}

// NewReportGenerator 创建报告生成器
func NewReportGenerator() *ReportGenerator {
	return &ReportGenerator{
		templatePath: "templates/backtest",
		outputPath:   "reports/backtest",
		reportCache:  make(map[string]*BacktestReport),
	}
}

// NewWalkForwardEngine 创建前进窗口引擎
func NewWalkForwardEngine() *WalkForwardEngine {
	return &WalkForwardEngine{
		windowSize:    252 * 24 * time.Hour, // 1年窗口
		stepSize:      30 * 24 * time.Hour,  // 1月步长
		minTradeCount: 10,                   // 最少交易数
	}
}

// Start 启动自动回测引擎
func (abe *AutoBacktestingEngine) Start() error {
	abe.mu.Lock()
	defer abe.mu.Unlock()

	if abe.isRunning {
		return fmt.Errorf("auto backtesting engine is already running")
	}

	if !abe.enabled {
		return fmt.Errorf("auto backtesting engine is disabled")
	}

	log.Println("Starting Auto Backtesting Engine...")

	// 启动定期回测
	abe.wg.Add(1)
	go abe.runScheduledBacktests()

	// 启动回测任务处理器
	abe.wg.Add(1)
	go abe.runJobProcessor()

	// 启动性能监控
	abe.wg.Add(1)
	go abe.runPerformanceMonitoring()

	// 启动验证检查
	abe.wg.Add(1)
	go abe.runValidationChecks()

	// 启动指标收集
	abe.wg.Add(1)
	go abe.runMetricsCollection()

	abe.isRunning = true
	log.Println("Auto Backtesting Engine started successfully")
	return nil
}

// Stop 停止自动回测引擎
func (abe *AutoBacktestingEngine) Stop() error {
	abe.mu.Lock()
	defer abe.mu.Unlock()

	if !abe.isRunning {
		return fmt.Errorf("auto backtesting engine is not running")
	}

	log.Println("Stopping Auto Backtesting Engine...")

	abe.cancel()
	abe.wg.Wait()

	abe.isRunning = false
	log.Println("Auto Backtesting Engine stopped successfully")
	return nil
}

// runScheduledBacktests 运行定期回测
func (abe *AutoBacktestingEngine) runScheduledBacktests() {
	defer abe.wg.Done()

	ticker := time.NewTicker(abe.frequency)
	defer ticker.Stop()

	log.Println("Scheduled backtests started")

	for {
		select {
		case <-abe.ctx.Done():
			log.Println("Scheduled backtests stopped")
			return
		case <-ticker.C:
			abe.scheduleAutomaticBacktests()
		}
	}
}

// runJobProcessor 运行任务处理器
func (abe *AutoBacktestingEngine) runJobProcessor() {
	defer abe.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("Job processor started")

	for {
		select {
		case <-abe.ctx.Done():
			log.Println("Job processor stopped")
			return
		case <-ticker.C:
			abe.processBacktestJobs()
		}
	}
}

// runPerformanceMonitoring 运行性能监控
func (abe *AutoBacktestingEngine) runPerformanceMonitoring() {
	defer abe.wg.Done()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	log.Println("Performance monitoring started")

	for {
		select {
		case <-abe.ctx.Done():
			log.Println("Performance monitoring stopped")
			return
		case <-ticker.C:
			abe.analyzeStrategyPerformance()
		}
	}
}

// runValidationChecks 运行验证检查
func (abe *AutoBacktestingEngine) runValidationChecks() {
	defer abe.wg.Done()

	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	log.Println("Validation checks started")

	for {
		select {
		case <-abe.ctx.Done():
			log.Println("Validation checks stopped")
			return
		case <-ticker.C:
			abe.performValidationChecks()
		}
	}
}

// runMetricsCollection 运行指标收集
func (abe *AutoBacktestingEngine) runMetricsCollection() {
	defer abe.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("Metrics collection started")

	for {
		select {
		case <-abe.ctx.Done():
			log.Println("Metrics collection stopped")
			return
		case <-ticker.C:
			abe.updateMetrics()
		}
	}
}

// scheduleAutomaticBacktests 安排自动回测
func (abe *AutoBacktestingEngine) scheduleAutomaticBacktests() {
	log.Println("Scheduling automatic backtests...")

	// 获取活跃策略列表
	strategies := abe.getActiveStrategies()

	for _, strategy := range strategies {
		// 检查是否需要回测
		if abe.needsBacktest(strategy) {
			job := abe.createBacktestJob(strategy)
			abe.submitBacktestJob(job)
		}
	}
}

// processBacktestJobs 处理回测任务
func (abe *AutoBacktestingEngine) processBacktestJobs() {
	abe.mu.RLock()
	jobs := make([]*BacktestJob, 0)
	for _, job := range abe.activeBacktests {
		if job.Status == "PENDING" {
			jobs = append(jobs, job)
		}
	}
	abe.mu.RUnlock()

	// 按优先级排序
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
	})

	// 处理任务（限制并发数）
	runningCount := abe.getRunningJobCount()
	availableSlots := abe.maxConcurrentJobs - runningCount

	for i := 0; i < len(jobs) && i < availableSlots; i++ {
		go abe.executeBacktestJob(jobs[i])
	}
}

// executeBacktestJob 执行回测任务
func (abe *AutoBacktestingEngine) executeBacktestJob(job *BacktestJob) {
	log.Printf("Executing backtest job: %s for strategy: %s", job.ID, job.StrategyName)

	// 更新任务状态
	job.Status = "RUNNING"
	job.StartTime = time.Now()
	job.Progress = 0.0

	defer func() {
		job.EndTime = time.Now()
		job.Duration = job.EndTime.Sub(job.StartTime)

		if r := recover(); r != nil {
			job.Status = "FAILED"
			job.ErrorMessage = fmt.Sprintf("Panic: %v", r)
			log.Printf("Backtest job %s failed with panic: %v", job.ID, r)
		}
	}()

	// 执行回测
	result, err := abe.runBacktest(job)
	if err != nil {
		job.Status = "FAILED"
		job.ErrorMessage = err.Error()
		log.Printf("Backtest job %s failed: %v", job.ID, err)
		return
	}

	// 保存结果
	job.Result = result
	job.Status = "COMPLETED"
	job.Progress = 1.0

	// 添加到完成列表
	abe.mu.Lock()
	abe.completedBacktests = append(abe.completedBacktests, *result)
	delete(abe.activeBacktests, job.ID)
	abe.mu.Unlock()

	// 生成报告
	go abe.generateBacktestReport(job)

	// 更新策略性能
	abe.updateStrategyPerformance(job.StrategyID, result)

	log.Printf("Backtest job %s completed successfully", job.ID)
}

// runBacktest 运行回测
func (abe *AutoBacktestingEngine) runBacktest(job *BacktestJob) (*BacktestResult, error) {
	// 获取策略
	strategy, err := abe.strategyManager.GetStrategy(job.StrategyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get strategy: %w", err)
	}

	// 获取历史数据
	data, err := abe.dataManager.GetHistoricalData(strategy.Symbols, job.StartDate, job.EndDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical data: %w", err)
	}

	// 初始化组合
	portfolio := &Portfolio{
		Cash:      job.InitialCapital,
		Positions: make(map[string]*Position),
		Equity:    job.InitialCapital,
		UpdatedAt: job.StartDate,
	}

	// 初始化策略
	err = strategy.Logic.Initialize(job.Parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize strategy: %w", err)
	}

	// 执行回测
	result := &BacktestResult{
		JobID:         job.ID,
		StrategyID:    job.StrategyID,
		EquityCurve:   make([]EquityPoint, 0),
		DrawdownCurve: make([]DrawdownPoint, 0),
		Trades:        make([]TradeRecord, 0),
		BacktestDate:  time.Now(),
		DataPeriod:    DateRange{Start: job.StartDate, End: job.EndDate},
		Benchmark:     job.BenchmarkSymbol,
	}

	// 模拟回测过程
	totalBars := len(data)
	for i, candle := range data {
		// 更新进度
		job.Progress = float64(i) / float64(totalBars)

		// 处理K线
		signal, err := strategy.Logic.ProcessBar(candle, portfolio)
		if err != nil {
			return nil, fmt.Errorf("strategy processing error: %w", err)
		}

		// 执行信号
		if signal != nil && signal.Action != "HOLD" {
			trade := abe.executeSignal(signal, portfolio, candle, job)
			if trade != nil {
				result.Trades = append(result.Trades, *trade)
			}
		}

		// 更新组合价值
		abe.updatePortfolioValue(portfolio, candle)

		// 记录净值点
		equityPoint := EquityPoint{
			Date:   candle.Time,
			Value:  portfolio.Equity,
			Return: (portfolio.Equity - job.InitialCapital) / job.InitialCapital,
		}
		result.EquityCurve = append(result.EquityCurve, equityPoint)
	}

	// 完成策略
	err = strategy.Logic.Finalize(portfolio)
	if err != nil {
		return nil, fmt.Errorf("failed to finalize strategy: %w", err)
	}

	// 计算性能指标
	abe.calculatePerformanceMetrics(result)

	// 执行验证
	validation := abe.validateBacktestResult(result)
	result.ValidationResult = validation

	return result, nil
}

// Helper functions implementation...

func (abe *AutoBacktestingEngine) getActiveStrategies() []*BacktestStrategy {
	abe.strategyManager.mu.RLock()
	defer abe.strategyManager.mu.RUnlock()

	strategies := make([]*BacktestStrategy, 0)
	for _, strategy := range abe.strategyManager.strategies {
		if strategy.IsActive {
			strategies = append(strategies, strategy)
		}
	}
	return strategies
}

func (abe *AutoBacktestingEngine) needsBacktest(strategy *BacktestStrategy) bool {
	// 检查上次回测时间
	if time.Since(strategy.LastBacktest) < abe.frequency {
		return false
	}

	// 检查性能阈值
	if strategy.AvgPerformance < abe.performanceThreshold {
		return true // 表现不佳的策略需要更频繁的验证
	}

	return true
}

func (abe *AutoBacktestingEngine) createBacktestJob(strategy *BacktestStrategy) *BacktestJob {
	endDate := time.Now().AddDate(0, 0, -1) // 前一天
	startDate := endDate.Add(-abe.lookbackPeriod)

	job := &BacktestJob{
		ID:              abe.generateJobID(),
		StrategyID:      strategy.ID,
		StrategyName:    strategy.Name,
		StartDate:       startDate,
		EndDate:         endDate,
		InitialCapital:  100000.0,
		Parameters:      abe.getDefaultParameters(strategy),
		Status:          "PENDING",
		Commission:      0.001,
		Slippage:        0.0005,
		BenchmarkSymbol: "BTCUSDT",
		RiskFreeRate:    0.02,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		CreatedBy:       "auto_backtesting_engine",
		JobType:         "SINGLE",
	}

	return job
}

func (abe *AutoBacktestingEngine) submitBacktestJob(job *BacktestJob) {
	abe.mu.Lock()
	abe.activeBacktests[job.ID] = job
	abe.mu.Unlock()

	log.Printf("Submitted backtest job: %s for strategy: %s", job.ID, job.StrategyName)
}

func (abe *AutoBacktestingEngine) getRunningJobCount() int {
	abe.mu.RLock()
	defer abe.mu.RUnlock()

	count := 0
	for _, job := range abe.activeBacktests {
		if job.Status == "RUNNING" {
			count++
		}
	}
	return count
}

func (abe *AutoBacktestingEngine) executeSignal(signal *Signal, portfolio *Portfolio, candle Candle, job *BacktestJob) *TradeRecord {
	// 实现信号执行逻辑
	if signal == nil || signal.Action == "HOLD" {
		return nil
	}

	// 计算交易数量
	quantity := signal.Quantity
	if quantity <= 0 {
		// 根据可用资金和价格计算数量
		availableCash := portfolio.Cash * 0.95 // 保留5%现金
		if signal.Price > 0 {
			quantity = availableCash / signal.Price
		} else {
			quantity = availableCash / candle.Close
		}
	}

	// 计算实际执行价格（考虑滑点）
	executionPrice := signal.Price
	if executionPrice <= 0 {
		executionPrice = candle.Close
	}

	// 应用滑点
	slippageRate := job.Slippage
	if signal.Action == "BUY" {
		executionPrice *= (1 + slippageRate)
	} else {
		executionPrice *= (1 - slippageRate)
	}

	// 计算手续费
	commission := executionPrice * quantity * job.Commission

	// 检查资金是否充足
	totalCost := executionPrice*quantity + commission
	if signal.Action == "BUY" && totalCost > portfolio.Cash {
		// 调整数量以适应可用资金
		quantity = (portfolio.Cash - commission) / executionPrice
		if quantity <= 0 {
			return nil // 资金不足，无法执行交易
		}
		totalCost = executionPrice*quantity + commission
	}

	// 执行交易
	var pnl float64
	var exitPrice float64
	var exitTime time.Time

	if signal.Action == "BUY" {
		// 买入操作
		portfolio.Cash -= totalCost

		// 更新或创建仓位
		if position, exists := portfolio.Positions[signal.Symbol]; exists {
			// 已有仓位，计算平均价格
			totalQuantity := position.Quantity + quantity
			totalValue := position.Quantity*position.AvgPrice + quantity*executionPrice
			position.AvgPrice = totalValue / totalQuantity
			position.Quantity = totalQuantity
		} else {
			// 新建仓位
			portfolio.Positions[signal.Symbol] = &Position{
				Symbol:      signal.Symbol,
				Quantity:    quantity,
				AvgPrice:    executionPrice,
				MarketPrice: executionPrice,
				OpenTime:    candle.Time,
			}
		}

		exitPrice = executionPrice // 买入时的退出价格就是执行价格
		exitTime = candle.Time

	} else if signal.Action == "SELL" {
		// 卖出操作
		if position, exists := portfolio.Positions[signal.Symbol]; exists && position.Quantity > 0 {
			// 计算卖出数量（不能超过持有数量）
			sellQuantity := math.Min(quantity, position.Quantity)

			// 计算盈亏
			pnl = (executionPrice-position.AvgPrice)*sellQuantity - commission

			// 更新现金
			portfolio.Cash += executionPrice*sellQuantity - commission

			// 更新仓位
			position.Quantity -= sellQuantity
			if position.Quantity <= 0 {
				delete(portfolio.Positions, signal.Symbol)
			}

			quantity = sellQuantity
			exitPrice = executionPrice
			exitTime = candle.Time
		} else {
			return nil // 没有仓位可卖出
		}
	}

	trade := &TradeRecord{
		ID:         abe.generateTradeID(),
		Symbol:     signal.Symbol,
		Side:       signal.Action,
		Quantity:   quantity,
		EntryPrice: executionPrice,
		ExitPrice:  exitPrice,
		EntryTime:  candle.Time,
		ExitTime:   exitTime,
		Duration:   exitTime.Sub(candle.Time),
		PnL:        pnl,
		Commission: commission,
		Slippage:   executionPrice * quantity * job.Slippage,
	}

	return trade
}

func (abe *AutoBacktestingEngine) updatePortfolioValue(portfolio *Portfolio, candle Candle) {
	// 实现组合价值更新逻辑

	// 更新所有仓位的市场价格和未实现盈亏
	totalPositionValue := 0.0
	for symbol, position := range portfolio.Positions {
		// 如果当前K线的交易对与仓位匹配，使用当前价格
		if symbol == candle.Symbol || symbol == fmt.Sprintf("%sUSDT", candle.Symbol) {
			position.MarketPrice = candle.Close
		}

		// 计算仓位价值
		positionValue := position.Quantity * position.MarketPrice
		totalPositionValue += positionValue

		// 计算未实现盈亏
		position.UnrealizedPL = (position.MarketPrice - position.AvgPrice) * position.Quantity
	}

	// 计算总权益 = 现金 + 所有仓位的市场价值
	portfolio.Equity = portfolio.Cash + totalPositionValue

	// 更新时间戳
	portfolio.UpdatedAt = candle.Time
}

func (abe *AutoBacktestingEngine) calculatePerformanceMetrics(result *BacktestResult) {
	if len(result.EquityCurve) == 0 {
		return
	}

	// 计算总收益
	initialValue := result.EquityCurve[0].Value
	finalValue := result.EquityCurve[len(result.EquityCurve)-1].Value
	result.TotalReturn = (finalValue - initialValue) / initialValue

	// 计算年化收益
	days := float64(len(result.EquityCurve))
	result.AnnualizedReturn = math.Pow(1+result.TotalReturn, 365/days) - 1

	// 计算最大回撤
	result.MaxDrawdown = abe.calculateMaxDrawdown(result.EquityCurve)

	// 计算交易统计
	result.TotalTrades = len(result.Trades)
	if result.TotalTrades > 0 {
		winCount := 0
		totalProfit := 0.0
		totalLoss := 0.0

		for _, trade := range result.Trades {
			if trade.PnL > 0 {
				winCount++
				totalProfit += trade.PnL
			} else {
				totalLoss += math.Abs(trade.PnL)
			}
		}

		result.WinningTrades = winCount
		result.LosingTrades = result.TotalTrades - winCount
		result.WinRate = float64(winCount) / float64(result.TotalTrades)

		if totalLoss > 0 {
			result.ProfitFactor = totalProfit / totalLoss
		}

		if winCount > 0 {
			result.AvgWin = totalProfit / float64(winCount)
		}

		if result.LosingTrades > 0 {
			result.AvgLoss = totalLoss / float64(result.LosingTrades)
		}
	}

	// 计算夏普比率
	returns := abe.calculateReturns(result.EquityCurve)
	if len(returns) > 1 {
		avgReturn := abe.mean(returns)
		volatility := abe.stdDev(returns)
		if volatility > 0 {
			result.SharpeRatio = (avgReturn - 0.02/252) / volatility * math.Sqrt(252) // 年化
		}
		result.Volatility = volatility * math.Sqrt(252) // 年化波动率
	}
}

func (abe *AutoBacktestingEngine) calculateMaxDrawdown(equityCurve []EquityPoint) float64 {
	if len(equityCurve) == 0 {
		return 0
	}

	maxDrawdown := 0.0
	peak := equityCurve[0].Value

	for _, point := range equityCurve {
		if point.Value > peak {
			peak = point.Value
		}

		drawdown := (peak - point.Value) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

func (abe *AutoBacktestingEngine) calculateReturns(equityCurve []EquityPoint) []float64 {
	if len(equityCurve) < 2 {
		return []float64{}
	}

	returns := make([]float64, len(equityCurve)-1)
	for i := 1; i < len(equityCurve); i++ {
		returns[i-1] = (equityCurve[i].Value - equityCurve[i-1].Value) / equityCurve[i-1].Value
	}

	return returns
}

func (abe *AutoBacktestingEngine) mean(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func (abe *AutoBacktestingEngine) stdDev(data []float64) float64 {
	if len(data) < 2 {
		return 0
	}

	mean := abe.mean(data)
	sum := 0.0
	for _, v := range data {
		sum += (v - mean) * (v - mean)
	}

	return math.Sqrt(sum / float64(len(data)-1))
}

func (abe *AutoBacktestingEngine) validateBacktestResult(result *BacktestResult) *ValidationResult {
	validation := &ValidationResult{
		JobID:           result.JobID,
		StrategyID:      result.StrategyID,
		ValidationDate:  time.Now(),
		Status:          "PASSED",
		Issues:          make([]ValidationIssue, 0),
		Recommendations: make([]string, 0),
		ThresholdChecks: make([]ThresholdCheck, 0),
	}

	// 执行各种验证测试
	validation.OutOfSampleTest = abe.performOutOfSampleTest(result)
	validation.StabilityTest = abe.performStabilityTest(result)
	validation.RobustnessTest = abe.performRobustnessTest(result)

	// 计算综合评分
	scores := []float64{
		validation.OutOfSampleTest.Score,
		validation.StabilityTest.Score,
		validation.RobustnessTest.Score,
	}
	validation.OverallScore = abe.mean(scores)

	// 确定等级
	if validation.OverallScore >= 0.9 {
		validation.OverallGrade = "A+"
	} else if validation.OverallScore >= 0.8 {
		validation.OverallGrade = "A"
	} else if validation.OverallScore >= 0.7 {
		validation.OverallGrade = "B+"
	} else if validation.OverallScore >= 0.6 {
		validation.OverallGrade = "B"
	} else {
		validation.OverallGrade = "C"
		validation.Status = "NEEDS_REVIEW"
	}

	return validation
}

func (abe *AutoBacktestingEngine) performOutOfSampleTest(result *BacktestResult) TestResult {
	// 实现样本外测试

	// 将数据分为样本内（前70%）和样本外（后30%）
	totalPeriod := result.DataPeriod.End.Sub(result.DataPeriod.Start)
	inSampleEnd := result.DataPeriod.Start.Add(time.Duration(float64(totalPeriod) * 0.7))

	// 计算样本内和样本外的收益率
	var inSampleReturn, outOfSampleReturn float64
	var inSampleEquity, outOfSampleEquity []EquityPoint

	// 分离样本内外的权益曲线
	for _, point := range result.EquityCurve {
		if point.Date.Before(inSampleEnd) {
			inSampleEquity = append(inSampleEquity, point)
		} else {
			outOfSampleEquity = append(outOfSampleEquity, point)
		}
	}

	// 计算样本内收益率
	if len(inSampleEquity) > 1 {
		inSampleReturn = (inSampleEquity[len(inSampleEquity)-1].Value - inSampleEquity[0].Value) / inSampleEquity[0].Value
	}

	// 计算样本外收益率
	if len(outOfSampleEquity) > 1 {
		outOfSampleReturn = (outOfSampleEquity[len(outOfSampleEquity)-1].Value - outOfSampleEquity[0].Value) / outOfSampleEquity[0].Value
	}

	// 计算一致性比率
	consistencyRatio := 1.0
	if inSampleReturn != 0 {
		consistencyRatio = math.Min(1.0, math.Abs(outOfSampleReturn/inSampleReturn))
	}

	// 计算样本外夏普比率
	var outOfSampleSharpe float64
	if len(outOfSampleEquity) > 1 {
		returns := make([]float64, len(outOfSampleEquity)-1)
		for i := 1; i < len(outOfSampleEquity); i++ {
			returns[i-1] = outOfSampleEquity[i].Return
		}

		// 计算平均收益和标准差
		var sum, sumSquares float64
		for _, ret := range returns {
			sum += ret
			sumSquares += ret * ret
		}

		avgReturn := sum / float64(len(returns))
		variance := sumSquares/float64(len(returns)) - avgReturn*avgReturn
		stdDev := math.Sqrt(variance)

		if stdDev > 0 {
			outOfSampleSharpe = (avgReturn - 0.02/252) / stdDev // 假设2%年化无风险利率
		}
	}

	// 评估测试结果
	passed := true
	score := 1.0
	details := "Out-of-sample test completed"

	// 检查一致性
	if consistencyRatio < 0.5 {
		passed = false
		score *= 0.5
		details = "Poor consistency between in-sample and out-of-sample performance"
	} else if consistencyRatio < 0.7 {
		score *= 0.8
		details = "Moderate consistency between in-sample and out-of-sample performance"
	}

	// 检查样本外收益率
	if outOfSampleReturn < -0.1 { // 样本外损失超过10%
		passed = false
		score *= 0.3
		details = "Poor out-of-sample performance with significant losses"
	} else if outOfSampleReturn < 0 {
		score *= 0.7
		details = "Negative out-of-sample returns but within acceptable range"
	}

	// 检查样本外夏普比率
	if outOfSampleSharpe < 0 {
		score *= 0.8
	} else if outOfSampleSharpe > 1.0 {
		score *= 1.1 // 奖励高夏普比率
	}

	return TestResult{
		Passed:  passed,
		Score:   math.Max(0, math.Min(1, score)),
		Details: details,
		Metrics: map[string]float64{
			"out_of_sample_return":  outOfSampleReturn,
			"in_sample_return":      inSampleReturn,
			"consistency_ratio":     consistencyRatio,
			"out_of_sample_sharpe":  outOfSampleSharpe,
			"out_of_sample_periods": float64(len(outOfSampleEquity)),
		},
	}
}

func (abe *AutoBacktestingEngine) performStabilityTest(result *BacktestResult) TestResult {
	// 实现稳定性测试

	if len(result.EquityCurve) < 30 {
		return TestResult{
			Passed:  false,
			Score:   0.0,
			Details: "Insufficient data for stability testing",
			Metrics: map[string]float64{},
		}
	}

	// 计算滚动窗口的性能指标
	windowSize := 30 // 30个数据点的滚动窗口
	var rollingSharpes []float64
	var rollingReturns []float64
	var rollingDrawdowns []float64

	for i := windowSize; i < len(result.EquityCurve); i++ {
		windowData := result.EquityCurve[i-windowSize : i]

		// 计算窗口内的收益率
		returns := make([]float64, len(windowData)-1)
		for j := 1; j < len(windowData); j++ {
			if windowData[j-1].Value > 0 {
				returns[j-1] = (windowData[j].Value - windowData[j-1].Value) / windowData[j-1].Value
			}
		}

		// 计算平均收益和标准差
		var sum, sumSquares float64
		for _, ret := range returns {
			sum += ret
			sumSquares += ret * ret
		}

		avgReturn := sum / float64(len(returns))
		variance := sumSquares/float64(len(returns)) - avgReturn*avgReturn
		stdDev := math.Sqrt(variance)

		// 计算夏普比率
		sharpe := 0.0
		if stdDev > 0 {
			sharpe = (avgReturn - 0.02/252) / stdDev // 假设2%年化无风险利率
		}

		rollingSharpes = append(rollingSharpes, sharpe)
		rollingReturns = append(rollingReturns, avgReturn)

		// 计算窗口内最大回撤
		maxDrawdown := 0.0
		peak := windowData[0].Value
		for _, point := range windowData {
			if point.Value > peak {
				peak = point.Value
			}
			drawdown := (peak - point.Value) / peak
			if drawdown > maxDrawdown {
				maxDrawdown = drawdown
			}
		}
		rollingDrawdowns = append(rollingDrawdowns, maxDrawdown)
	}

	// 计算稳定性指标

	// 1. 夏普比率稳定性（标准差越小越稳定）
	var sharpeSum, sharpeSquareSum float64
	for _, sharpe := range rollingSharpes {
		sharpeSum += sharpe
		sharpeSquareSum += sharpe * sharpe
	}
	avgSharpe := sharpeSum / float64(len(rollingSharpes))
	sharpeVariance := sharpeSquareSum/float64(len(rollingSharpes)) - avgSharpe*avgSharpe
	sharpeStability := 1.0 / (1.0 + math.Sqrt(sharpeVariance))

	// 2. 收益率稳定性
	var returnSum, returnSquareSum float64
	for _, ret := range rollingReturns {
		returnSum += ret
		returnSquareSum += ret * ret
	}
	avgReturn := returnSum / float64(len(rollingReturns))
	returnVariance := returnSquareSum/float64(len(rollingReturns)) - avgReturn*avgReturn
	returnStability := 1.0 / (1.0 + math.Sqrt(returnVariance))

	// 3. 回撤一致性（回撤变化的标准差）
	var drawdownSum, drawdownSquareSum float64
	for _, dd := range rollingDrawdowns {
		drawdownSum += dd
		drawdownSquareSum += dd * dd
	}
	avgDrawdown := drawdownSum / float64(len(rollingDrawdowns))
	drawdownVariance := drawdownSquareSum/float64(len(rollingDrawdowns)) - avgDrawdown*avgDrawdown
	drawdownConsistency := 1.0 / (1.0 + math.Sqrt(drawdownVariance))

	// 4. 正收益期间比例
	positiveReturns := 0
	for _, ret := range rollingReturns {
		if ret > 0 {
			positiveReturns++
		}
	}
	positiveRatio := float64(positiveReturns) / float64(len(rollingReturns))

	// 综合评分
	overallScore := (sharpeStability*0.3 + returnStability*0.3 + drawdownConsistency*0.3 + positiveRatio*0.1)

	// 评估结果
	passed := true
	details := "Strategy shows good stability across different periods"

	if overallScore < 0.5 {
		passed = false
		details = "Strategy shows poor stability with high variance in performance"
	} else if overallScore < 0.7 {
		details = "Strategy shows moderate stability with some performance variance"
	}

	// 额外检查
	if avgSharpe < 0 {
		passed = false
		overallScore *= 0.5
		details = "Strategy has negative average Sharpe ratio"
	}

	if avgDrawdown > 0.3 {
		passed = false
		overallScore *= 0.7
		details = "Strategy has excessive average drawdown"
	}

	return TestResult{
		Passed:  passed,
		Score:   math.Max(0, math.Min(1, overallScore)),
		Details: details,
		Metrics: map[string]float64{
			"rolling_sharpe_stability": sharpeStability,
			"return_stability":         returnStability,
			"drawdown_consistency":     drawdownConsistency,
			"positive_return_ratio":    positiveRatio,
			"avg_rolling_sharpe":       avgSharpe,
			"avg_rolling_drawdown":     avgDrawdown,
			"rolling_windows":          float64(len(rollingSharpes)),
		},
	}
}

func (abe *AutoBacktestingEngine) performRobustnessTest(result *BacktestResult) TestResult {
	// 实现鲁棒性测试

	if len(result.EquityCurve) < 10 {
		return TestResult{
			Passed:  false,
			Score:   0.0,
			Details: "Insufficient data for robustness testing",
			Metrics: map[string]float64{},
		}
	}

	// 1. 测试对噪声的抗性
	noiseResistance := abe.testNoiseResistance(result)

	// 2. 测试对不同时间段的适应性
	periodAdaptability := abe.testPeriodAdaptability(result)

	// 3. 测试交易频率的稳定性
	tradingFrequencyStability := abe.testTradingFrequencyStability(result)

	// 4. 测试收益分布的一致性
	returnDistributionConsistency := abe.testReturnDistributionConsistency(result)

	// 5. 测试最大回撤的控制能力
	drawdownControl := abe.testDrawdownControl(result)

	// 综合评分
	overallScore := (noiseResistance*0.25 + periodAdaptability*0.25 +
		tradingFrequencyStability*0.2 + returnDistributionConsistency*0.15 +
		drawdownControl*0.15)

	// 评估结果
	passed := true
	details := "Strategy demonstrates good robustness"

	if overallScore < 0.5 {
		passed = false
		details = "Strategy shows poor robustness with high sensitivity to market conditions"
	} else if overallScore < 0.7 {
		details = "Strategy shows moderate robustness with some sensitivity to market conditions"
	}

	// 额外检查
	if result.MaxDrawdown > 0.5 {
		passed = false
		overallScore *= 0.6
		details = "Strategy has excessive maximum drawdown indicating poor risk control"
	}

	if result.TotalTrades < 10 {
		overallScore *= 0.8
		details = "Limited number of trades may affect robustness assessment"
	}

	return TestResult{
		Passed:  passed,
		Score:   math.Max(0, math.Min(1, overallScore)),
		Details: details,
		Metrics: map[string]float64{
			"noise_resistance":                noiseResistance,
			"period_adaptability":             periodAdaptability,
			"trading_frequency_stability":     tradingFrequencyStability,
			"return_distribution_consistency": returnDistributionConsistency,
			"drawdown_control":                drawdownControl,
			"total_trades":                    float64(result.TotalTrades),
			"max_drawdown":                    result.MaxDrawdown,
		},
	}
}

// testNoiseResistance 测试对噪声的抗性
func (abe *AutoBacktestingEngine) testNoiseResistance(result *BacktestResult) float64 {
	if len(result.EquityCurve) < 5 {
		return 0.5
	}

	// 计算收益率的变异系数
	returns := make([]float64, len(result.EquityCurve)-1)
	for i := 1; i < len(result.EquityCurve); i++ {
		if result.EquityCurve[i-1].Value > 0 {
			returns[i-1] = result.EquityCurve[i].Return
		}
	}

	var sum, sumSquares float64
	for _, ret := range returns {
		sum += ret
		sumSquares += ret * ret
	}

	avgReturn := sum / float64(len(returns))
	variance := sumSquares/float64(len(returns)) - avgReturn*avgReturn
	stdDev := math.Sqrt(variance)

	// 变异系数越小，抗噪声能力越强
	coefficientOfVariation := 1.0
	if avgReturn != 0 {
		coefficientOfVariation = math.Abs(stdDev / avgReturn)
	}

	// 转换为0-1分数，变异系数越小分数越高
	return math.Max(0, 1.0-math.Min(1.0, coefficientOfVariation))
}

// testPeriodAdaptability 测试对不同时间段的适应性
func (abe *AutoBacktestingEngine) testPeriodAdaptability(result *BacktestResult) float64 {
	if len(result.EquityCurve) < 20 {
		return 0.5
	}

	// 将数据分为4个季度
	quarterSize := len(result.EquityCurve) / 4
	quarterReturns := make([]float64, 4)

	for q := 0; q < 4; q++ {
		start := q * quarterSize
		end := start + quarterSize
		if q == 3 {
			end = len(result.EquityCurve)
		}

		if end > start+1 {
			quarterReturns[q] = (result.EquityCurve[end-1].Value - result.EquityCurve[start].Value) / result.EquityCurve[start].Value
		}
	}

	// 计算季度收益率的标准差
	var sum, sumSquares float64
	for _, ret := range quarterReturns {
		sum += ret
		sumSquares += ret * ret
	}

	avgReturn := sum / 4.0
	variance := sumSquares/4.0 - avgReturn*avgReturn
	stdDev := math.Sqrt(variance)

	// 标准差越小，适应性越好
	return math.Max(0, 1.0-math.Min(1.0, stdDev*2))
}

// testTradingFrequencyStability 测试交易频率的稳定性
func (abe *AutoBacktestingEngine) testTradingFrequencyStability(result *BacktestResult) float64 {
	if len(result.Trades) < 4 {
		return 0.5
	}

	// 计算每个时间段的交易频率
	totalDays := result.DataPeriod.End.Sub(result.DataPeriod.Start).Hours() / 24
	quarterDays := totalDays / 4

	quarterTradeCounts := make([]int, 4)

	for _, trade := range result.Trades {
		daysSinceStart := trade.EntryTime.Sub(result.DataPeriod.Start).Hours() / 24
		quarter := int(daysSinceStart / quarterDays)
		if quarter >= 4 {
			quarter = 3
		}
		quarterTradeCounts[quarter]++
	}

	// 计算交易频率的变异系数
	var sum, sumSquares float64
	for _, count := range quarterTradeCounts {
		frequency := float64(count) / quarterDays
		sum += frequency
		sumSquares += frequency * frequency
	}

	avgFrequency := sum / 4.0
	variance := sumSquares/4.0 - avgFrequency*avgFrequency
	stdDev := math.Sqrt(variance)

	coefficientOfVariation := 1.0
	if avgFrequency > 0 {
		coefficientOfVariation = stdDev / avgFrequency
	}

	return math.Max(0, 1.0-math.Min(1.0, coefficientOfVariation))
}

// testReturnDistributionConsistency 测试收益分布的一致性
func (abe *AutoBacktestingEngine) testReturnDistributionConsistency(result *BacktestResult) float64 {
	if len(result.Trades) < 10 {
		return 0.5
	}

	// 计算交易收益率
	tradeReturns := make([]float64, len(result.Trades))
	for i, trade := range result.Trades {
		if trade.EntryPrice > 0 {
			tradeReturns[i] = trade.PnLPercent
		}
	}

	// 计算偏度和峰度
	var sum, sumSquares, sumCubes, sumFourths float64
	for _, ret := range tradeReturns {
		sum += ret
		sumSquares += ret * ret
		sumCubes += ret * ret * ret
		sumFourths += ret * ret * ret * ret
	}

	n := float64(len(tradeReturns))
	mean := sum / n
	variance := sumSquares/n - mean*mean
	stdDev := math.Sqrt(variance)

	if stdDev == 0 {
		return 0.5
	}

	// 偏度
	skewness := (sumCubes/n - 3*mean*variance - mean*mean*mean) / (stdDev * stdDev * stdDev)

	// 峰度
	kurtosis := (sumFourths/n - 4*mean*sumCubes/n + 6*mean*mean*variance + 3*mean*mean*mean*mean) / (variance * variance)

	// 正态分布的偏度为0，峰度为3
	skewnessScore := math.Max(0, 1.0-math.Abs(skewness)/2.0)
	kurtosisScore := math.Max(0, 1.0-math.Abs(kurtosis-3.0)/3.0)

	return (skewnessScore + kurtosisScore) / 2.0
}

// testDrawdownControl 测试最大回撤的控制能力
func (abe *AutoBacktestingEngine) testDrawdownControl(result *BacktestResult) float64 {
	if len(result.DrawdownCurve) == 0 {
		return 0.5
	}

	// 计算回撤的持续时间分布
	var drawdownDurations []int
	currentDuration := 0
	inDrawdown := false

	for _, point := range result.DrawdownCurve {
		if point.Drawdown > 0.01 { // 1%以上算作回撤
			if !inDrawdown {
				inDrawdown = true
				currentDuration = 1
			} else {
				currentDuration++
			}
		} else {
			if inDrawdown {
				drawdownDurations = append(drawdownDurations, currentDuration)
				inDrawdown = false
				currentDuration = 0
			}
		}
	}

	if len(drawdownDurations) == 0 {
		return 1.0 // 没有显著回撤
	}

	// 计算平均回撤持续时间
	var totalDuration int
	maxDuration := 0
	for _, duration := range drawdownDurations {
		totalDuration += duration
		if duration > maxDuration {
			maxDuration = duration
		}
	}

	avgDuration := float64(totalDuration) / float64(len(drawdownDurations))

	// 评分：最大回撤越小、持续时间越短，分数越高
	maxDrawdownScore := math.Max(0, 1.0-result.MaxDrawdown*2) // 50%回撤得0分
	durationScore := math.Max(0, 1.0-avgDuration/100.0)       // 100天平均持续时间得0分

	return (maxDrawdownScore*0.7 + durationScore*0.3)
}

func (abe *AutoBacktestingEngine) generateBacktestReport(job *BacktestJob) {
	// 实现报告生成
	log.Printf("Generating backtest report for job: %s", job.ID)

	if job.Result == nil {
		log.Printf("No result available for job %s, skipping report generation", job.ID)
		return
	}

	result := job.Result

	// 创建报告
	report := &BacktestReport{
		ID:          fmt.Sprintf("report_%s_%d", job.ID, time.Now().Unix()),
		JobID:       job.ID,
		Title:       fmt.Sprintf("Backtest Report - %s", job.StrategyName),
		GeneratedAt: time.Now(),
	}

	// 生成报告摘要
	report.Summary = ReportSummary{
		StrategyName:     job.StrategyName,
		TestPeriod:       result.DataPeriod,
		TotalReturn:      result.TotalReturn,
		AnnualizedReturn: result.AnnualizedReturn,
		MaxDrawdown:      result.MaxDrawdown,
		SharpeRatio:      result.SharpeRatio,
		WinRate:          result.WinRate,
		TotalTrades:      result.TotalTrades,
		Grade:            abe.calculatePerformanceGrade(result),
	}

	// 生成性能图表数据
	report.PerformanceChart = abe.generatePerformanceCharts(result)

	// 生成风险分析
	report.RiskAnalysis = abe.generateRiskAnalysis(result)

	// 生成交易分析
	report.TradeAnalysis = abe.generateTradeAnalysis(result)

	// 生成建议
	report.Recommendations = abe.generateRecommendations(result)

	// 保存报告到缓存
	abe.reportGenerator.mu.Lock()
	abe.reportGenerator.reportCache[report.ID] = report
	abe.reportGenerator.mu.Unlock()

	// 生成输出文件路径
	timestamp := time.Now().Format("20060102_150405")
	baseFilename := fmt.Sprintf("%s_%s_%s", job.StrategyName, job.ID, timestamp)

	report.HTMLPath = fmt.Sprintf("%s/%s.html", abe.reportGenerator.outputPath, baseFilename)
	report.PDFPath = fmt.Sprintf("%s/%s.pdf", abe.reportGenerator.outputPath, baseFilename)
	report.JSONPath = fmt.Sprintf("%s/%s.json", abe.reportGenerator.outputPath, baseFilename)

	// 异步生成实际文件
	go abe.generateReportFiles(report)

	log.Printf("Backtest report generated successfully for job: %s", job.ID)
}

// calculatePerformanceGrade 计算性能等级
func (abe *AutoBacktestingEngine) calculatePerformanceGrade(result *BacktestResult) string {
	score := 0.0

	// 收益率评分 (30%)
	if result.AnnualizedReturn > 0.3 {
		score += 30
	} else if result.AnnualizedReturn > 0.2 {
		score += 25
	} else if result.AnnualizedReturn > 0.1 {
		score += 20
	} else if result.AnnualizedReturn > 0.05 {
		score += 15
	} else if result.AnnualizedReturn > 0 {
		score += 10
	}

	// 夏普比率评分 (25%)
	if result.SharpeRatio > 2.0 {
		score += 25
	} else if result.SharpeRatio > 1.5 {
		score += 20
	} else if result.SharpeRatio > 1.0 {
		score += 15
	} else if result.SharpeRatio > 0.5 {
		score += 10
	} else if result.SharpeRatio > 0 {
		score += 5
	}

	// 最大回撤评分 (25%)
	if result.MaxDrawdown < 0.05 {
		score += 25
	} else if result.MaxDrawdown < 0.1 {
		score += 20
	} else if result.MaxDrawdown < 0.15 {
		score += 15
	} else if result.MaxDrawdown < 0.2 {
		score += 10
	} else if result.MaxDrawdown < 0.3 {
		score += 5
	}

	// 胜率评分 (20%)
	if result.WinRate > 0.7 {
		score += 20
	} else if result.WinRate > 0.6 {
		score += 15
	} else if result.WinRate > 0.5 {
		score += 10
	} else if result.WinRate > 0.4 {
		score += 5
	}

	// 根据总分确定等级
	if score >= 90 {
		return "A+"
	} else if score >= 85 {
		return "A"
	} else if score >= 80 {
		return "B+"
	} else if score >= 75 {
		return "B"
	} else if score >= 70 {
		return "C+"
	} else if score >= 65 {
		return "C"
	} else if score >= 60 {
		return "D"
	} else {
		return "F"
	}
}

// generatePerformanceCharts 生成性能图表数据
func (abe *AutoBacktestingEngine) generatePerformanceCharts(result *BacktestResult) []ChartData {
	charts := make([]ChartData, 0)

	// 权益曲线图
	equityData := make([]DataPoint, len(result.EquityCurve))
	for i, point := range result.EquityCurve {
		equityData[i] = DataPoint{
			X:     point.Date.Format("2006-01-02"),
			Y:     point.Value,
			Label: fmt.Sprintf("%.2f", point.Value),
		}
	}

	charts = append(charts, ChartData{
		Name: "equity_curve",
		Type: "line",
		Data: equityData,
		Config: ChartConfig{
			Title:      "Equity Curve",
			XAxisLabel: "Date",
			YAxisLabel: "Portfolio Value",
			Color:      "#2E86AB",
		},
	})

	// 回撤曲线图
	if len(result.DrawdownCurve) > 0 {
		drawdownData := make([]DataPoint, len(result.DrawdownCurve))
		for i, point := range result.DrawdownCurve {
			drawdownData[i] = DataPoint{
				X:     point.Date.Format("2006-01-02"),
				Y:     -point.Drawdown, // 负值显示
				Label: fmt.Sprintf("%.2f%%", point.Drawdown*100),
			}
		}

		charts = append(charts, ChartData{
			Name: "drawdown_curve",
			Type: "line",
			Data: drawdownData,
			Config: ChartConfig{
				Title:      "Drawdown Curve",
				XAxisLabel: "Date",
				YAxisLabel: "Drawdown (%)",
				Color:      "#A23B72",
			},
		})
	}

	return charts
}

// generateRiskAnalysis 生成风险分析
func (abe *AutoBacktestingEngine) generateRiskAnalysis(result *BacktestResult) RiskAnalysis {
	riskLevel := "LOW"
	if result.MaxDrawdown > 0.3 {
		riskLevel = "EXTREME"
	} else if result.MaxDrawdown > 0.2 {
		riskLevel = "HIGH"
	} else if result.MaxDrawdown > 0.1 {
		riskLevel = "MEDIUM"
	}

	riskFactors := make([]string, 0)
	if result.MaxDrawdown > 0.15 {
		riskFactors = append(riskFactors, "High maximum drawdown")
	}
	if result.Volatility > 0.3 {
		riskFactors = append(riskFactors, "High volatility")
	}
	if result.SharpeRatio < 0.5 {
		riskFactors = append(riskFactors, "Low risk-adjusted returns")
	}
	if result.WinRate < 0.4 {
		riskFactors = append(riskFactors, "Low win rate")
	}

	riskMetrics := RiskMetrics{}
	if result.RiskMetrics != nil {
		riskMetrics = *result.RiskMetrics
	}

	return RiskAnalysis{
		RiskLevel:   riskLevel,
		RiskFactors: riskFactors,
		RiskMetrics: riskMetrics,
	}
}

// generateTradeAnalysis 生成交易分析
func (abe *AutoBacktestingEngine) generateTradeAnalysis(result *BacktestResult) TradeAnalysis {
	if len(result.Trades) == 0 {
		return TradeAnalysis{}
	}

	// 计算平均持仓时间
	var totalDuration time.Duration
	for _, trade := range result.Trades {
		totalDuration += trade.Duration
	}
	avgHoldingPeriod := totalDuration / time.Duration(len(result.Trades))

	// 找出最佳和最差交易
	bestTrades := make([]TradeRecord, 0)
	worstTrades := make([]TradeRecord, 0)

	// 按盈亏排序
	sortedTrades := make([]TradeRecord, len(result.Trades))
	copy(sortedTrades, result.Trades)
	sort.Slice(sortedTrades, func(i, j int) bool {
		return sortedTrades[i].PnL > sortedTrades[j].PnL
	})

	// 取前5个最佳和最差交易
	maxCount := 5
	if len(sortedTrades) < maxCount {
		maxCount = len(sortedTrades)
	}

	bestTrades = sortedTrades[:maxCount]
	worstTrades = sortedTrades[len(sortedTrades)-maxCount:]

	// 计算交易频率
	if len(result.EquityCurve) > 0 {
		totalDays := result.DataPeriod.End.Sub(result.DataPeriod.Start).Hours() / 24
		tradingFrequency := float64(len(result.Trades)) / totalDays

		return TradeAnalysis{
			TradingFrequency: tradingFrequency,
			AvgHoldingPeriod: avgHoldingPeriod,
			BestTrades:       bestTrades,
			WorstTrades:      worstTrades,
		}
	}

	return TradeAnalysis{
		AvgHoldingPeriod: avgHoldingPeriod,
		BestTrades:       bestTrades,
		WorstTrades:      worstTrades,
	}
}

// generateRecommendations 生成建议
func (abe *AutoBacktestingEngine) generateRecommendations(result *BacktestResult) []Recommendation {
	recommendations := make([]Recommendation, 0)

	// 基于最大回撤的建议
	if result.MaxDrawdown > 0.2 {
		recommendations = append(recommendations, Recommendation{
			Type:        "RISK",
			Priority:    "HIGH",
			Title:       "Reduce Maximum Drawdown",
			Description: "The strategy has a high maximum drawdown which indicates poor risk management.",
			Action:      "Consider implementing tighter stop-loss rules or position sizing controls.",
			Impact:      "Reducing drawdown will improve risk-adjusted returns and investor confidence.",
			Confidence:  0.9,
		})
	}

	// 基于夏普比率的建议
	if result.SharpeRatio < 1.0 {
		recommendations = append(recommendations, Recommendation{
			Type:        "PARAMETER",
			Priority:    "MEDIUM",
			Title:       "Improve Risk-Adjusted Returns",
			Description: "The Sharpe ratio is below 1.0, indicating suboptimal risk-adjusted performance.",
			Action:      "Optimize entry/exit criteria or adjust position sizing to improve the risk-return profile.",
			Impact:      "Higher Sharpe ratio will make the strategy more attractive to investors.",
			Confidence:  0.8,
		})
	}

	// 基于胜率的建议
	if result.WinRate < 0.5 {
		recommendations = append(recommendations, Recommendation{
			Type:        "PARAMETER",
			Priority:    "MEDIUM",
			Title:       "Improve Win Rate",
			Description: "The win rate is below 50%, which may indicate poor entry timing.",
			Action:      "Review and optimize entry signals, consider additional filters or confirmation indicators.",
			Impact:      "Higher win rate will improve strategy consistency and reduce psychological stress.",
			Confidence:  0.7,
		})
	}

	// 基于交易数量的建议
	if result.TotalTrades < 30 {
		recommendations = append(recommendations, Recommendation{
			Type:        "TIMING",
			Priority:    "LOW",
			Title:       "Increase Sample Size",
			Description: "The number of trades is relatively low, which may affect statistical significance.",
			Action:      "Consider extending the backtest period or adjusting parameters to generate more trades.",
			Impact:      "More trades will provide better statistical confidence in the results.",
			Confidence:  0.6,
		})
	}

	return recommendations
}

// generateReportFiles 异步生成报告文件
func (abe *AutoBacktestingEngine) generateReportFiles(report *BacktestReport) {
	// 这里可以实现实际的文件生成逻辑
	// 例如生成HTML、PDF、JSON文件
	log.Printf("Generating report files for report: %s", report.ID)

	// 简化实现：只记录文件路径
	log.Printf("HTML report would be saved to: %s", report.HTMLPath)
	log.Printf("PDF report would be saved to: %s", report.PDFPath)
	log.Printf("JSON report would be saved to: %s", report.JSONPath)
}

func (abe *AutoBacktestingEngine) updateStrategyPerformance(strategyID string, result *BacktestResult) {
	abe.mu.Lock()
	defer abe.mu.Unlock()

	performance, exists := abe.strategyPerformance[strategyID]
	if !exists {
		performance = &StrategyPerformance{
			StrategyID:           strategyID,
			PerformanceHistory:   make([]PerformanceRecord, 0),
			OptimalParameters:    make(map[string]interface{}),
			ParameterSensitivity: make(map[string]float64),
		}
		abe.strategyPerformance[strategyID] = performance
	}

	// 更新统计
	performance.BacktestCount++
	performance.AvgReturn = (performance.AvgReturn*float64(performance.BacktestCount-1) + result.AnnualizedReturn) / float64(performance.BacktestCount)
	performance.AvgSharpe = (performance.AvgSharpe*float64(performance.BacktestCount-1) + result.SharpeRatio) / float64(performance.BacktestCount)
	performance.AvgMaxDrawdown = (performance.AvgMaxDrawdown*float64(performance.BacktestCount-1) + result.MaxDrawdown) / float64(performance.BacktestCount)

	// 更新最佳和最差表现
	if result.AnnualizedReturn > performance.BestPerformance {
		performance.BestPerformance = result.AnnualizedReturn
	}
	if result.AnnualizedReturn < performance.WorstPerformance || performance.WorstPerformance == 0 {
		performance.WorstPerformance = result.AnnualizedReturn
	}

	// 添加性能记录
	record := PerformanceRecord{
		Date:          time.Now(),
		Return:        result.AnnualizedReturn,
		SharpeRatio:   result.SharpeRatio,
		MaxDrawdown:   result.MaxDrawdown,
		TradeCount:    result.TotalTrades,
		WinRate:       result.WinRate,
		BacktestJobID: result.JobID,
	}
	performance.PerformanceHistory = append(performance.PerformanceHistory, record)

	// 更新验证状态
	if result.ValidationResult != nil {
		performance.ValidationScore = result.ValidationResult.OverallScore
		performance.LastValidation = time.Now()

		if result.ValidationResult.Status == "PASSED" {
			performance.ValidationStatus = "PASSED"
		} else {
			performance.ValidationStatus = "NEEDS_REVIEW"
		}
	}

	performance.UpdatedAt = time.Now()
}

func (abe *AutoBacktestingEngine) analyzeStrategyPerformance() {
	log.Println("Analyzing strategy performance...")

	// 实现策略性能分析
	abe.mu.RLock()
	strategies := make(map[string]*StrategyPerformance)
	for k, v := range abe.strategyPerformance {
		strategies[k] = v
	}
	abe.mu.RUnlock()

	if len(strategies) == 0 {
		log.Println("No strategies to analyze")
		return
	}

	// 1. 识别表现优异的策略
	topPerformers := abe.identifyTopPerformers(strategies)
	log.Printf("Identified %d top performing strategies", len(topPerformers))

	// 2. 发现表现衰退的策略
	decliningStrategies := abe.identifyDecliningStrategies(strategies)
	log.Printf("Identified %d declining strategies", len(decliningStrategies))

	// 3. 生成优化建议
	optimizationSuggestions := abe.generateOptimizationSuggestions(strategies)
	log.Printf("Generated %d optimization suggestions", len(optimizationSuggestions))

	// 4. 更新策略排名和状态
	abe.updateStrategyRankings(strategies)

	// 5. 记录分析结果
	abe.recordPerformanceAnalysis(topPerformers, decliningStrategies, optimizationSuggestions)
}

// identifyTopPerformers 识别表现优异的策略
func (abe *AutoBacktestingEngine) identifyTopPerformers(strategies map[string]*StrategyPerformance) []string {
	type strategyScore struct {
		ID    string
		Score float64
	}

	scores := make([]strategyScore, 0, len(strategies))

	for id, perf := range strategies {
		if perf.BacktestCount < 3 {
			continue // 需要至少3次回测才能评估
		}

		// 计算综合评分
		score := 0.0

		// 平均收益率权重 40%
		score += perf.AvgReturn * 0.4

		// 平均夏普比率权重 30%
		score += perf.AvgSharpe * 0.3

		// 一致性评分权重 20%
		score += perf.ConsistencyScore * 0.2

		// 最大回撤惩罚权重 10%
		score -= perf.AvgMaxDrawdown * 0.1

		scores = append(scores, strategyScore{ID: id, Score: score})
	}

	// 按评分排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	// 返回前20%的策略
	topCount := len(scores) / 5
	if topCount < 1 {
		topCount = 1
	}
	if topCount > len(scores) {
		topCount = len(scores)
	}

	topPerformers := make([]string, topCount)
	for i := 0; i < topCount; i++ {
		topPerformers[i] = scores[i].ID
	}

	return topPerformers
}

// identifyDecliningStrategies 识别表现衰退的策略
func (abe *AutoBacktestingEngine) identifyDecliningStrategies(strategies map[string]*StrategyPerformance) []string {
	decliningStrategies := make([]string, 0)

	for id, perf := range strategies {
		if len(perf.PerformanceHistory) < 5 {
			continue // 需要至少5个历史记录才能判断趋势
		}

		// 检查最近的表现趋势
		recentCount := 3
		if len(perf.PerformanceHistory) < recentCount {
			recentCount = len(perf.PerformanceHistory)
		}

		// 计算最近表现的平均值
		recentPerf := perf.PerformanceHistory[len(perf.PerformanceHistory)-recentCount:]
		var recentAvgReturn, recentAvgSharpe float64
		for _, record := range recentPerf {
			recentAvgReturn += record.Return
			recentAvgSharpe += record.SharpeRatio
		}
		recentAvgReturn /= float64(len(recentPerf))
		recentAvgSharpe /= float64(len(recentPerf))

		// 计算历史表现的平均值
		historicalCount := len(perf.PerformanceHistory) - recentCount
		if historicalCount > 0 {
			historicalPerf := perf.PerformanceHistory[:historicalCount]
			var historicalAvgReturn, historicalAvgSharpe float64
			for _, record := range historicalPerf {
				historicalAvgReturn += record.Return
				historicalAvgSharpe += record.SharpeRatio
			}
			historicalAvgReturn /= float64(len(historicalPerf))
			historicalAvgSharpe /= float64(len(historicalPerf))

			// 判断是否衰退
			returnDecline := (historicalAvgReturn - recentAvgReturn) / math.Abs(historicalAvgReturn)
			sharpeDecline := (historicalAvgSharpe - recentAvgSharpe) / math.Abs(historicalAvgSharpe)

			// 如果收益率下降超过20%或夏普比率下降超过30%，认为是衰退
			if returnDecline > 0.2 || sharpeDecline > 0.3 {
				decliningStrategies = append(decliningStrategies, id)
			}
		}

		// 检查验证状态
		if perf.ValidationStatus == "FAILED" {
			decliningStrategies = append(decliningStrategies, id)
		}
	}

	return decliningStrategies
}

// generateOptimizationSuggestions 生成优化建议
func (abe *AutoBacktestingEngine) generateOptimizationSuggestions(strategies map[string]*StrategyPerformance) map[string][]string {
	suggestions := make(map[string][]string)

	for id, perf := range strategies {
		strategySuggestions := make([]string, 0)

		// 基于平均收益率的建议
		if perf.AvgReturn < 0.05 { // 年化收益率低于5%
			strategySuggestions = append(strategySuggestions, "Consider increasing position size or improving entry signals to boost returns")
		}

		// 基于夏普比率的建议
		if perf.AvgSharpe < 1.0 {
			strategySuggestions = append(strategySuggestions, "Optimize risk management to improve risk-adjusted returns")
		}

		// 基于最大回撤的建议
		if perf.AvgMaxDrawdown > 0.15 {
			strategySuggestions = append(strategySuggestions, "Implement stricter stop-loss rules to reduce maximum drawdown")
		}

		// 基于一致性的建议
		if perf.ConsistencyScore < 0.7 {
			strategySuggestions = append(strategySuggestions, "Review parameter stability and consider adaptive mechanisms")
		}

		// 基于回测次数的建议
		if perf.BacktestCount < 10 {
			strategySuggestions = append(strategySuggestions, "Increase backtesting frequency to gather more performance data")
		}

		// 基于验证状态的建议
		if perf.ValidationStatus == "NEEDS_REVIEW" {
			strategySuggestions = append(strategySuggestions, "Strategy requires manual review due to validation concerns")
		} else if perf.ValidationStatus == "FAILED" {
			strategySuggestions = append(strategySuggestions, "Strategy failed validation and should be disabled or redesigned")
		}

		if len(strategySuggestions) > 0 {
			suggestions[id] = strategySuggestions
		}
	}

	return suggestions
}

// updateStrategyRankings 更新策略排名
func (abe *AutoBacktestingEngine) updateStrategyRankings(strategies map[string]*StrategyPerformance) {
	// 计算所有策略的综合评分并排名
	type strategyRanking struct {
		ID    string
		Score float64
		Rank  int
	}

	rankings := make([]strategyRanking, 0, len(strategies))

	for id, perf := range strategies {
		score := perf.AvgReturn*0.4 + perf.AvgSharpe*0.3 + perf.ConsistencyScore*0.2 - perf.AvgMaxDrawdown*0.1
		rankings = append(rankings, strategyRanking{ID: id, Score: score})
	}

	// 按评分排序
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].Score > rankings[j].Score
	})

	// 分配排名
	for i := range rankings {
		rankings[i].Rank = i + 1
	}

	// 更新策略性能记录
	abe.mu.Lock()
	for _, ranking := range rankings {
		if _, exists := abe.strategyPerformance[ranking.ID]; exists {
			// 这里可以添加排名字段到StrategyPerformance结构体
			log.Printf("Strategy %s ranked #%d with score %.4f", ranking.ID, ranking.Rank, ranking.Score)
		}
	}
	abe.mu.Unlock()
}

// recordPerformanceAnalysis 记录性能分析结果
func (abe *AutoBacktestingEngine) recordPerformanceAnalysis(topPerformers, decliningStrategies []string, suggestions map[string][]string) {
	log.Printf("Performance Analysis Summary:")
	log.Printf("- Top Performers: %v", topPerformers)
	log.Printf("- Declining Strategies: %v", decliningStrategies)
	log.Printf("- Strategies with Suggestions: %d", len(suggestions))

	// 更新指标
	abe.backtestingMetrics.mu.Lock()
	if len(topPerformers) > 0 {
		// 可以记录顶级策略的平均表现
		var totalReturn float64
		for _, id := range topPerformers {
			if perf, exists := abe.strategyPerformance[id]; exists {
				totalReturn += perf.AvgReturn
			}
		}
		abe.backtestingMetrics.BestStrategyReturn = totalReturn / float64(len(topPerformers))
	}
	abe.backtestingMetrics.LastUpdated = time.Now()
	abe.backtestingMetrics.mu.Unlock()
}

func (abe *AutoBacktestingEngine) performValidationChecks() {
	log.Println("Performing validation checks...")

	// 实现验证检查
	abe.mu.RLock()
	strategies := make(map[string]*StrategyPerformance)
	for k, v := range abe.strategyPerformance {
		strategies[k] = v
	}
	completedBacktests := make([]BacktestResult, len(abe.completedBacktests))
	copy(completedBacktests, abe.completedBacktests)
	abe.mu.RUnlock()

	validationResults := make([]ValidationResult, 0)

	// 1. 检查策略是否符合性能阈值
	thresholdResults := abe.checkPerformanceThresholds(strategies)
	log.Printf("Performance threshold checks completed for %d strategies", len(thresholdResults))

	// 2. 验证策略的稳定性
	stabilityResults := abe.checkStrategyStability(strategies)
	log.Printf("Stability checks completed for %d strategies", len(stabilityResults))

	// 3. 检查是否存在过拟合
	overfittingResults := abe.checkOverfitting(completedBacktests)
	log.Printf("Overfitting checks completed for %d backtest results", len(overfittingResults))

	// 4. 综合验证结果
	for strategyID, perf := range strategies {
		validationResult := abe.createValidationResult(strategyID, perf, thresholdResults, stabilityResults, overfittingResults)
		validationResults = append(validationResults, validationResult)
	}

	// 5. 更新验证历史
	abe.mu.Lock()
	abe.validationHistory = append(abe.validationHistory, validationResults...)

	// 保持历史记录在合理范围内
	if len(abe.validationHistory) > 1000 {
		abe.validationHistory = abe.validationHistory[len(abe.validationHistory)-1000:]
	}
	abe.mu.Unlock()

	// 6. 更新策略验证状态
	abe.updateStrategyValidationStatus(validationResults)

	log.Printf("Validation checks completed. %d validation results generated", len(validationResults))
}

// checkPerformanceThresholds 检查性能阈值
func (abe *AutoBacktestingEngine) checkPerformanceThresholds(strategies map[string]*StrategyPerformance) map[string][]ThresholdCheck {
	results := make(map[string][]ThresholdCheck)

	for strategyID, perf := range strategies {
		checks := make([]ThresholdCheck, 0)

		// 最小收益率阈值
		checks = append(checks, ThresholdCheck{
			Metric:     "avg_return",
			Value:      perf.AvgReturn,
			Threshold:  abe.performanceThreshold, // 默认2%
			Operator:   ">=",
			Passed:     perf.AvgReturn >= abe.performanceThreshold,
			Importance: "HIGH",
		})

		// 最小夏普比率阈值
		checks = append(checks, ThresholdCheck{
			Metric:     "avg_sharpe",
			Value:      perf.AvgSharpe,
			Threshold:  0.5,
			Operator:   ">=",
			Passed:     perf.AvgSharpe >= 0.5,
			Importance: "HIGH",
		})

		// 最大回撤阈值
		checks = append(checks, ThresholdCheck{
			Metric:     "avg_max_drawdown",
			Value:      perf.AvgMaxDrawdown,
			Threshold:  0.2, // 最大回撤不超过20%
			Operator:   "<=",
			Passed:     perf.AvgMaxDrawdown <= 0.2,
			Importance: "CRITICAL",
		})

		// 一致性评分阈值
		checks = append(checks, ThresholdCheck{
			Metric:     "consistency_score",
			Value:      perf.ConsistencyScore,
			Threshold:  0.6,
			Operator:   ">=",
			Passed:     perf.ConsistencyScore >= 0.6,
			Importance: "MEDIUM",
		})

		// 最小回测次数阈值
		checks = append(checks, ThresholdCheck{
			Metric:     "backtest_count",
			Value:      float64(perf.BacktestCount),
			Threshold:  5, // 至少5次回测
			Operator:   ">=",
			Passed:     perf.BacktestCount >= 5,
			Importance: "MEDIUM",
		})

		results[strategyID] = checks
	}

	return results
}

// checkStrategyStability 检查策略稳定性
func (abe *AutoBacktestingEngine) checkStrategyStability(strategies map[string]*StrategyPerformance) map[string]bool {
	results := make(map[string]bool)

	for strategyID, perf := range strategies {
		stable := true

		// 检查性能历史的变异性
		if len(perf.PerformanceHistory) >= 3 {
			returns := make([]float64, len(perf.PerformanceHistory))
			sharpes := make([]float64, len(perf.PerformanceHistory))

			for i, record := range perf.PerformanceHistory {
				returns[i] = record.Return
				sharpes[i] = record.SharpeRatio
			}

			// 计算收益率的变异系数
			returnCV := abe.calculateCoefficientOfVariation(returns)
			sharpeCV := abe.calculateCoefficientOfVariation(sharpes)

			// 如果变异系数过高，认为不稳定
			if returnCV > 1.0 || sharpeCV > 0.8 {
				stable = false
			}
		}

		// 检查最近表现是否显著下降
		if len(perf.PerformanceHistory) >= 5 {
			recentCount := 2
			recent := perf.PerformanceHistory[len(perf.PerformanceHistory)-recentCount:]
			historical := perf.PerformanceHistory[:len(perf.PerformanceHistory)-recentCount]

			var recentAvg, historicalAvg float64
			for _, record := range recent {
				recentAvg += record.Return
			}
			recentAvg /= float64(len(recent))

			for _, record := range historical {
				historicalAvg += record.Return
			}
			historicalAvg /= float64(len(historical))

			// 如果最近表现比历史平均下降超过50%，认为不稳定
			if historicalAvg > 0 && (historicalAvg-recentAvg)/historicalAvg > 0.5 {
				stable = false
			}
		}

		results[strategyID] = stable
	}

	return results
}

// checkOverfitting 检查过拟合
func (abe *AutoBacktestingEngine) checkOverfitting(backtestResults []BacktestResult) map[string]bool {
	results := make(map[string]bool)

	for _, result := range backtestResults {
		overfitted := false

		// 首先尝试执行PBO测试
		pboResult, err := abe.performPBOTest(result)
		if err != nil {
			log.Printf("PBO test failed for strategy %s: %v", result.StrategyID, err)
			// PBO测试失败时，使用传统的过拟合检测方法
		} else if pboResult.PBOProbability > 0.5 {
			overfitted = true
			log.Printf("Strategy %s failed PBO test with probability %.2f", result.StrategyID, pboResult.PBOProbability)
		}

		// 检查样本内外表现差异
		if result.ValidationResult != nil {
			outOfSampleTest := result.ValidationResult.OutOfSampleTest
			if outOfSampleTest.Metrics != nil {
				inSampleReturn, inOk := outOfSampleTest.Metrics["in_sample_return"]
				outOfSampleReturn, outOk := outOfSampleTest.Metrics["out_of_sample_return"]

				if inOk && outOk && inSampleReturn > 0 {
					// 如果样本外表现比样本内差很多，可能存在过拟合
					performanceDrop := (inSampleReturn - outOfSampleReturn) / inSampleReturn
					if performanceDrop > 0.5 { // 样本外表现下降超过50%
						overfitted = true
					}
				}
			}
		}

		// 检查收益率分布的异常
		if len(result.Trades) > 10 {
			// 计算交易收益率的偏度和峰度
			returns := make([]float64, len(result.Trades))
			for i, trade := range result.Trades {
				returns[i] = trade.PnLPercent
			}

			skewness := abe.calculateSkewness(returns)
			kurtosis := abe.calculateKurtosis(returns)

			// 极端的偏度或峰度可能表明过拟合
			if math.Abs(skewness) > 3.0 || kurtosis > 10.0 {
				overfitted = true
			}
		}

		// 检查胜率是否过高（可能的过拟合信号）
		if result.WinRate > 0.9 && result.TotalTrades > 20 {
			overfitted = true
		}

		results[result.StrategyID] = overfitted
	}

	return results
}

// PBOResult represents the result of a Probability of Backtest Overfitting test
type PBOResult struct {
	PBOProbability    float64
	DeflatedSharpe    float64
	MinBacktestLength int
	Passed            bool
	Details           string
}

// performPBOTest performs Probability of Backtest Overfitting test
func (abe *AutoBacktestingEngine) performPBOTest(result BacktestResult) (*PBOResult, error) {
	// 检查是否有足够的收益数据
	if len(result.EquityCurve) < 100 {
		return nil, fmt.Errorf("insufficient returns data for PBO test: need at least 100 points, got %d", len(result.EquityCurve))
	}

	// 提取收益率序列
	returns := make([]float64, len(result.EquityCurve)-1)
	for i := 1; i < len(result.EquityCurve); i++ {
		if result.EquityCurve[i-1].Value > 0 {
			returns[i-1] = (result.EquityCurve[i].Value - result.EquityCurve[i-1].Value) / result.EquityCurve[i-1].Value
		}
	}

	// 计算基础统计量
	mean, variance := abe.calculateMeanVariance(returns)
	if variance <= 0 {
		return nil, fmt.Errorf("invalid variance for PBO test: %f", variance)
	}

	sharpeRatio := mean / math.Sqrt(variance)

	// 计算PBO概率 - 简化版本
	// 实际的PBO测试需要更复杂的统计分析
	pboProb := abe.calculatePBOProbability(returns, sharpeRatio)

	// 计算通胀调整后的夏普比率
	deflatedSharpe := abe.calculateDeflatedSharpe(returns, len(returns))

	pboResult := &PBOResult{
		PBOProbability:    pboProb,
		DeflatedSharpe:    deflatedSharpe,
		MinBacktestLength: 100,
		Passed:            pboProb <= 0.5 && deflatedSharpe > 1.0,
	}

	if pboResult.Passed {
		pboResult.Details = fmt.Sprintf("PBO test passed: probability=%.3f, deflated_sharpe=%.3f", pboProb, deflatedSharpe)
	} else {
		pboResult.Details = fmt.Sprintf("PBO test failed: probability=%.3f, deflated_sharpe=%.3f", pboProb, deflatedSharpe)
	}

	return pboResult, nil
}

// calculateMeanVariance calculates mean and variance of returns
func (abe *AutoBacktestingEngine) calculateMeanVariance(returns []float64) (float64, float64) {
	if len(returns) == 0 {
		return 0, 0
	}

	var sum, sumSquares float64
	for _, ret := range returns {
		sum += ret
		sumSquares += ret * ret
	}

	mean := sum / float64(len(returns))
	variance := sumSquares/float64(len(returns)) - mean*mean

	return mean, variance
}

// calculatePBOProbability calculates simplified PBO probability
func (abe *AutoBacktestingEngine) calculatePBOProbability(returns []float64, sharpeRatio float64) float64 {
	n := float64(len(returns))

	// 简化的PBO计算 - 基于样本大小和夏普比率
	// 实际实现应该使用更复杂的统计方法
	if n < 100 {
		return 0.9 // 样本太小，高概率过拟合
	}

	// 基于夏普比率和样本大小的启发式计算
	logN := math.Log(n)
	pboProb := math.Max(0, 1.0-(sharpeRatio*math.Sqrt(n))/(2.0*logN))

	return math.Min(1.0, pboProb)
}

// calculateDeflatedSharpe calculates deflated Sharpe ratio
func (abe *AutoBacktestingEngine) calculateDeflatedSharpe(returns []float64, trials int) float64 {
	if len(returns) == 0 {
		return 0
	}

	mean, variance := abe.calculateMeanVariance(returns)
	if variance <= 0 {
		return 0
	}

	sharpeRatio := mean / math.Sqrt(variance)
	n := float64(len(returns))

	// 通胀调整 - 考虑多重测试的影响
	deflationFactor := math.Sqrt((1.0 - math.Gamma(0.5)*math.Sqrt(2.0/(n-1.0))) * math.Log(float64(trials)))

	return sharpeRatio - deflationFactor
}

// createValidationResult 创建验证结果
func (abe *AutoBacktestingEngine) createValidationResult(strategyID string, perf *StrategyPerformance,
	thresholdResults map[string][]ThresholdCheck, stabilityResults map[string]bool,
	overfittingResults map[string]bool) ValidationResult {

	result := ValidationResult{
		JobID:           fmt.Sprintf("validation_%s_%d", strategyID, time.Now().Unix()),
		StrategyID:      strategyID,
		ValidationDate:  time.Now(),
		Issues:          make([]ValidationIssue, 0),
		Recommendations: make([]string, 0),
	}

	// 阈值检查结果
	if checks, exists := thresholdResults[strategyID]; exists {
		result.ThresholdChecks = checks

		passedCount := 0
		for _, check := range checks {
			if check.Passed {
				passedCount++
			} else {
				// 添加问题
				severity := "MEDIUM"
				if check.Importance == "CRITICAL" {
					severity = "HIGH"
				}

				result.Issues = append(result.Issues, ValidationIssue{
					Type:        "THRESHOLD",
					Severity:    severity,
					Description: fmt.Sprintf("Metric %s (%.4f) failed threshold check (%.4f)", check.Metric, check.Value, check.Threshold),
					Suggestion:  fmt.Sprintf("Improve %s to meet minimum requirements", check.Metric),
				})
			}
		}

		// 阈值测试评分
		result.OutOfSampleTest = TestResult{
			Passed:  passedCount == len(checks),
			Score:   float64(passedCount) / float64(len(checks)),
			Details: fmt.Sprintf("Passed %d out of %d threshold checks", passedCount, len(checks)),
		}
	}

	// 稳定性检查结果
	if stable, exists := stabilityResults[strategyID]; exists {
		result.StabilityTest = TestResult{
			Passed:  stable,
			Score:   map[bool]float64{true: 1.0, false: 0.0}[stable],
			Details: map[bool]string{true: "Strategy shows good stability", false: "Strategy shows instability"}[stable],
		}

		if !stable {
			result.Issues = append(result.Issues, ValidationIssue{
				Type:        "STABILITY",
				Severity:    "HIGH",
				Description: "Strategy performance shows high variability",
				Suggestion:  "Review parameter sensitivity and consider adaptive mechanisms",
			})
		}
	}

	// 过拟合检查结果
	if overfitted, exists := overfittingResults[strategyID]; exists {
		result.RobustnessTest = TestResult{
			Passed:  !overfitted,
			Score:   map[bool]float64{true: 0.0, false: 1.0}[overfitted],
			Details: map[bool]string{true: "Potential overfitting detected", false: "No significant overfitting detected"}[overfitted],
		}

		if overfitted {
			result.Issues = append(result.Issues, ValidationIssue{
				Type:        "OVERFITTING",
				Severity:    "HIGH",
				Description: "Strategy may be overfitted to historical data",
				Suggestion:  "Increase out-of-sample testing and reduce model complexity",
			})
		}
	}

	// 计算综合评分
	scores := []float64{result.OutOfSampleTest.Score, result.StabilityTest.Score, result.RobustnessTest.Score}
	totalScore := 0.0
	for _, score := range scores {
		totalScore += score
	}
	result.OverallScore = totalScore / float64(len(scores))

	// 确定状态和等级
	if result.OverallScore >= 0.8 {
		result.Status = "PASSED"
		result.OverallGrade = "A"
	} else if result.OverallScore >= 0.6 {
		result.Status = "PASSED"
		result.OverallGrade = "B"
	} else if result.OverallScore >= 0.4 {
		result.Status = "NEEDS_REVIEW"
		result.OverallGrade = "C"
	} else {
		result.Status = "FAILED"
		result.OverallGrade = "F"
	}

	// 生成建议
	if len(result.Issues) > 0 {
		result.Recommendations = append(result.Recommendations, "Address identified issues to improve strategy performance")
	}
	if result.OverallScore < 0.6 {
		result.Recommendations = append(result.Recommendations, "Consider strategy redesign or parameter optimization")
	}

	return result
}

// updateStrategyValidationStatus 更新策略验证状态
func (abe *AutoBacktestingEngine) updateStrategyValidationStatus(validationResults []ValidationResult) {
	abe.mu.Lock()
	defer abe.mu.Unlock()

	for _, result := range validationResults {
		if perf, exists := abe.strategyPerformance[result.StrategyID]; exists {
			perf.ValidationStatus = result.Status
			perf.ValidationScore = result.OverallScore
			perf.LastValidation = result.ValidationDate
		}
	}
}

// 辅助函数
func (abe *AutoBacktestingEngine) calculateCoefficientOfVariation(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	var sum, sumSquares float64
	for _, v := range values {
		sum += v
		sumSquares += v * v
	}

	mean := sum / float64(len(values))
	if mean == 0 {
		return 0
	}

	variance := sumSquares/float64(len(values)) - mean*mean
	stdDev := math.Sqrt(variance)

	return stdDev / math.Abs(mean)
}

func (abe *AutoBacktestingEngine) calculateSkewness(values []float64) float64 {
	if len(values) < 3 {
		return 0
	}

	var sum, sumSquares, sumCubes float64
	for _, v := range values {
		sum += v
		sumSquares += v * v
		sumCubes += v * v * v
	}

	n := float64(len(values))
	mean := sum / n
	variance := sumSquares/n - mean*mean
	stdDev := math.Sqrt(variance)

	if stdDev == 0 {
		return 0
	}

	return (sumCubes/n - 3*mean*variance - mean*mean*mean) / (stdDev * stdDev * stdDev)
}

func (abe *AutoBacktestingEngine) calculateKurtosis(values []float64) float64 {
	if len(values) < 4 {
		return 0
	}

	var sum, sumSquares, sumFourths float64
	for _, v := range values {
		sum += v
		sumSquares += v * v
		sumFourths += v * v * v * v
	}

	n := float64(len(values))
	mean := sum / n
	variance := sumSquares/n - mean*mean

	if variance == 0 {
		return 0
	}

	return (sumFourths/n - 4*mean*sum*sumSquares/n/n + 6*mean*mean*variance + 3*mean*mean*mean*mean) / (variance * variance)
}

func (abe *AutoBacktestingEngine) updateMetrics() {
	abe.backtestingMetrics.mu.Lock()
	defer abe.backtestingMetrics.mu.Unlock()

	// 更新执行统计
	abe.backtestingMetrics.TotalJobs = int64(len(abe.completedBacktests) + len(abe.activeBacktests))
	abe.backtestingMetrics.CompletedJobs = int64(len(abe.completedBacktests))

	// 计算成功率
	if abe.backtestingMetrics.TotalJobs > 0 {
		abe.backtestingMetrics.SuccessRate = float64(abe.backtestingMetrics.CompletedJobs) / float64(abe.backtestingMetrics.TotalJobs)
	}

	// 更新性能统计
	if len(abe.completedBacktests) > 0 {
		totalReturn := 0.0
		bestReturn := math.Inf(-1)
		worstReturn := math.Inf(1)

		for _, result := range abe.completedBacktests {
			totalReturn += result.AnnualizedReturn
			if result.AnnualizedReturn > bestReturn {
				bestReturn = result.AnnualizedReturn
			}
			if result.AnnualizedReturn < worstReturn {
				worstReturn = result.AnnualizedReturn
			}
		}

		abe.backtestingMetrics.AvgStrategyReturn = totalReturn / float64(len(abe.completedBacktests))
		abe.backtestingMetrics.BestStrategyReturn = bestReturn
		abe.backtestingMetrics.WorstStrategyReturn = worstReturn
	}

	// 更新系统指标
	abe.backtestingMetrics.ActiveJobs = abe.getRunningJobCount()
	abe.backtestingMetrics.QueuedJobs = abe.getPendingJobCount()

	abe.backtestingMetrics.LastUpdated = time.Now()
}

func (abe *AutoBacktestingEngine) getPendingJobCount() int {
	abe.mu.RLock()
	defer abe.mu.RUnlock()

	count := 0
	for _, job := range abe.activeBacktests {
		if job.Status == "PENDING" {
			count++
		}
	}
	return count
}

func (abe *AutoBacktestingEngine) getDefaultParameters(strategy *BacktestStrategy) map[string]interface{} {
	params := make(map[string]interface{})
	for name, param := range strategy.Parameters {
		params[name] = param.Value
	}
	return params
}

func (abe *AutoBacktestingEngine) generateJobID() string {
	return fmt.Sprintf("BT_%d", time.Now().UnixNano())
}

func (abe *AutoBacktestingEngine) generateTradeID() string {
	return fmt.Sprintf("TR_%d", time.Now().UnixNano())
}

// GetStatus 获取引擎状态
func (abe *AutoBacktestingEngine) GetStatus() map[string]interface{} {
	abe.mu.RLock()
	defer abe.mu.RUnlock()

	return map[string]interface{}{
		"running":               abe.isRunning,
		"enabled":               abe.enabled,
		"active_backtests":      len(abe.activeBacktests),
		"completed_backtests":   len(abe.completedBacktests),
		"strategy_count":        len(abe.strategyPerformance),
		"frequency":             abe.frequency,
		"lookback_period":       abe.lookbackPeriod,
		"performance_threshold": abe.performanceThreshold,
		"max_concurrent_jobs":   abe.maxConcurrentJobs,
		"backtesting_metrics":   abe.backtestingMetrics,
	}
}

// GetBacktestingMetrics 获取回测指标
func (abe *AutoBacktestingEngine) GetBacktestingMetrics() *BacktestingMetrics {
	abe.backtestingMetrics.mu.RLock()
	defer abe.backtestingMetrics.mu.RUnlock()

	metrics := *abe.backtestingMetrics
	return &metrics
}

// GetStrategyPerformance 获取策略表现
func (abe *AutoBacktestingEngine) GetStrategyPerformance(strategyID string) (*StrategyPerformance, error) {
	abe.mu.RLock()
	defer abe.mu.RUnlock()

	if performance, exists := abe.strategyPerformance[strategyID]; exists {
		return performance, nil
	}

	return nil, fmt.Errorf("strategy %s not found", strategyID)
}

// GetCompletedBacktests 获取完成的回测
func (abe *AutoBacktestingEngine) GetCompletedBacktests(limit int) []BacktestResult {
	abe.mu.RLock()
	defer abe.mu.RUnlock()

	if limit <= 0 || limit > len(abe.completedBacktests) {
		limit = len(abe.completedBacktests)
	}

	// 返回最新的回测结果
	start := len(abe.completedBacktests) - limit
	return abe.completedBacktests[start:]
}

// 在BacktestStrategyManager中添加方法
func (bsm *BacktestStrategyManager) GetStrategy(id string) (*BacktestStrategy, error) {
	bsm.mu.RLock()
	defer bsm.mu.RUnlock()

	if strategy, exists := bsm.strategies[id]; exists {
		return strategy, nil
	}

	return nil, fmt.Errorf("strategy %s not found", id)
}

// GetHistoricalData 从数据库或API获取历史数据
func (bdm *BacktestDataManager) GetHistoricalData(symbols []string, startDate, endDate time.Time) ([]Candle, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("no symbols provided")
	}

	var allData []Candle

	// 为每个交易对获取数据
	for _, symbol := range symbols {
		data, err := bdm.getHistoricalDataForSymbol(symbol, startDate, endDate)
		if err != nil {
			log.Printf("Failed to get historical data for %s: %v", symbol, err)
			continue
		}
		allData = append(allData, data...)
	}

	if len(allData) == 0 {
		return nil, fmt.Errorf("no historical data found for symbols %v in period %v to %v",
			symbols, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	}

	return allData, nil
}

// getHistoricalDataForSymbol 获取单个交易对的历史数据
func (bdm *BacktestDataManager) getHistoricalDataForSymbol(symbol string, startDate, endDate time.Time) ([]Candle, error) {
	// 优先使用K线管理器的自动回填功能
	if bdm.klineManager != nil {
		data, err := bdm.getDataFromKlineManager(symbol, startDate, endDate)
		if err == nil && len(data) > 0 {
			log.Printf("Retrieved %d candles from kline manager (with auto-backfill) for %s", len(data), symbol)
			return data, nil
		}
		log.Printf("Kline manager query failed for %s: %v", symbol, err)
	}

	// 回退到原有的数据库+API方式
	// 首先尝试从数据库获取
	data, err := bdm.getDataFromDatabase(symbol, startDate, endDate)
	if err == nil && len(data) > 0 {
		log.Printf("Retrieved %d candles from database for %s", len(data), symbol)
		return data, nil
	}

	log.Printf("Database query failed or returned no data for %s: %v", symbol, err)

	// 如果数据库没有数据，尝试从API获取
	data, err = bdm.getDataFromAPI(symbol, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get data from API for %s: %w", symbol, err)
	}

	log.Printf("Retrieved %d candles from API for %s", len(data), symbol)
	return data, nil
}

// getDataFromKlineManager 从K线管理器获取历史数据（带自动回填）
func (bdm *BacktestDataManager) getDataFromKlineManager(symbol string, startDate, endDate time.Time) ([]Candle, error) {
	if bdm.klineManager == nil {
		return nil, fmt.Errorf("kline manager not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second) // 增加超时时间，因为可能需要回填
	defer cancel()

	// 使用1小时间隔获取数据（带自动回填）
	klineData, err := bdm.klineManager.GetHistoryWithBackfill(ctx, symbol, "1h", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get data from kline manager: %w", err)
	}

	// 转换为Candle格式
	var candles []Candle
	for _, kline := range klineData {
		candle := Candle{
			Time:   kline.OpenTime,
			Open:   kline.Open,
			High:   kline.High,
			Low:    kline.Low,
			Close:  kline.Close,
			Volume: kline.Volume,
		}
		candles = append(candles, candle)
	}

	return candles, nil
}

// getDataFromDatabase 从数据库获取历史数据
func (bdm *BacktestDataManager) getDataFromDatabase(symbol string, startDate, endDate time.Time) ([]Candle, error) {
	if bdm.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return bdm.db.QueryHistoricalData(ctx, symbol, startDate, endDate)
}

// getDataFromAPI 从API获取历史数据
func (bdm *BacktestDataManager) getDataFromAPI(symbol string, startDate, endDate time.Time) ([]Candle, error) {
	if bdm.apiClient == nil {
		return nil, fmt.Errorf("API client not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 使用1小时间隔获取数据
	return bdm.apiClient.GetHistoricalKlines(ctx, symbol, "1h", startDate, endDate, 1000)
}

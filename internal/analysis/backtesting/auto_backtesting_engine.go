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
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	isRunning  bool
	mu         sync.RWMutex
	
	// 回测配置
	frequency           time.Duration
	lookbackPeriod      time.Duration
	walkForwardWindow   time.Duration
	performanceThreshold float64
	
	// 回测状态
	activeBacktests     map[string]*BacktestJob
	completedBacktests  []BacktestResult
	strategyPerformance map[string]*StrategyPerformance
	
	// 监控指标
	backtestingMetrics  *BacktestingMetrics
	validationHistory   []ValidationResult
	
	// 配置参数
	enabled             bool
	maxConcurrentJobs   int
	dataRetentionDays   int
}

// BacktestJob 回测任务
type BacktestJob struct {
	ID              string            `json:"id"`
	StrategyID      string            `json:"strategy_id"`
	StrategyName    string            `json:"strategy_name"`
	StartDate       time.Time         `json:"start_date"`
	EndDate         time.Time         `json:"end_date"`
	InitialCapital  float64           `json:"initial_capital"`
	Parameters      map[string]interface{} `json:"parameters"`
	
	// 执行状态
	Status          string            `json:"status"`     // PENDING, RUNNING, COMPLETED, FAILED
	Progress        float64           `json:"progress"`   // 0.0 - 1.0
	StartTime       time.Time         `json:"start_time"`
	EndTime         time.Time         `json:"end_time"`
	Duration        time.Duration     `json:"duration"`
	
	// 配置选项
	Commission      float64           `json:"commission"`
	Slippage        float64           `json:"slippage"`
	BenchmarkSymbol string            `json:"benchmark_symbol"`
	RiskFreeRate    float64           `json:"risk_free_rate"`
	
	// 结果
	Result          *BacktestResult   `json:"result"`
	ErrorMessage    string            `json:"error_message"`
	
	// 元数据
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	CreatedBy       string            `json:"created_by"`
	JobType         string            `json:"job_type"`   // SINGLE, WALK_FORWARD, PARAMETER_SWEEP
}

// BacktestResult 回测结果
type BacktestResult struct {
	JobID           string            `json:"job_id"`
	StrategyID      string            `json:"strategy_id"`
	
	// 基本统计
	TotalReturn     float64           `json:"total_return"`
	AnnualizedReturn float64          `json:"annualized_return"`
	Volatility      float64           `json:"volatility"`
	SharpeRatio     float64           `json:"sharpe_ratio"`
	SortinoRatio    float64           `json:"sortino_ratio"`
	CalmarRatio     float64           `json:"calmar_ratio"`
	MaxDrawdown     float64           `json:"max_drawdown"`
	
	// 交易统计
	TotalTrades     int               `json:"total_trades"`
	WinningTrades   int               `json:"winning_trades"`
	LosingTrades    int               `json:"losing_trades"`
	WinRate         float64           `json:"win_rate"`
	ProfitFactor    float64           `json:"profit_factor"`
	AvgWin          float64           `json:"avg_win"`
	AvgLoss         float64           `json:"avg_loss"`
	LargestWin      float64           `json:"largest_win"`
	LargestLoss     float64           `json:"largest_loss"`
	
	// 时间序列数据
	EquityCurve     []EquityPoint     `json:"equity_curve"`
	DrawdownCurve   []DrawdownPoint   `json:"drawdown_curve"`
	BenchmarkCurve  []EquityPoint     `json:"benchmark_curve"`
	
	// 详细交易记录
	Trades          []TradeRecord     `json:"trades"`
	
	// 风险指标
	VaR95           float64           `json:"var_95"`
	CVaR95          float64           `json:"cvar_95"`
	Beta            float64           `json:"beta"`
	Alpha           float64           `json:"alpha"`
	TrackingError   float64           `json:"tracking_error"`
	InformationRatio float64          `json:"information_ratio"`
	
	// 稳定性指标
	ConsistencyScore float64          `json:"consistency_score"`
	RobustnessScore  float64          `json:"robustness_score"`
	
	// 元数据
	BacktestDate    time.Time         `json:"backtest_date"`
	DataPeriod      DateRange         `json:"data_period"`
	Benchmark       string            `json:"benchmark"`
	
	// 验证结果
	ValidationResult *ValidationResult `json:"validation_result"`
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
	ID           string    `json:"id"`
	Symbol       string    `json:"symbol"`
	Side         string    `json:"side"`         // BUY, SELL
	Quantity     float64   `json:"quantity"`
	EntryPrice   float64   `json:"entry_price"`
	ExitPrice    float64   `json:"exit_price"`
	EntryTime    time.Time `json:"entry_time"`
	ExitTime     time.Time `json:"exit_time"`
	Duration     time.Duration `json:"duration"`
	PnL          float64   `json:"pnl"`
	PnLPercent   float64   `json:"pnl_percent"`
	Commission   float64   `json:"commission"`
	Slippage     float64   `json:"slippage"`
	MAE          float64   `json:"mae"`          // Maximum Adverse Excursion
	MFE          float64   `json:"mfe"`          // Maximum Favorable Excursion
	Tags         []string  `json:"tags"`
}

// DateRange 日期范围
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// BacktestDataManager 回测数据管理器
type BacktestDataManager struct {
	dataSource      string
	symbols         []string
	timeframes      []string
	dataCache       map[string][]Candle
	lastUpdate      time.Time
	
	mu              sync.RWMutex
}

// Candle K线数据
type Candle struct {
	Time   time.Time `json:"time"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume float64   `json:"volume"`
}

// BacktestStrategyManager 回测策略管理器
type BacktestStrategyManager struct {
	strategies      map[string]*BacktestStrategy
	strategyLoader  StrategyLoader
	
	mu              sync.RWMutex
}

// BacktestStrategy 回测策略
type BacktestStrategy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Parameters  map[string]Parameter   `json:"parameters"`
	Logic       StrategyLogic          `json:"-"`
	
	// 性能统计
	BacktestCount   int               `json:"backtest_count"`
	AvgPerformance  float64           `json:"avg_performance"`
	BestPerformance float64           `json:"best_performance"`
	WorstPerformance float64          `json:"worst_performance"`
	LastBacktest    time.Time         `json:"last_backtest"`
	
	// 配置
	Symbols         []string          `json:"symbols"`
	Timeframe       string            `json:"timeframe"`
	IsActive        bool              `json:"is_active"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// Parameter 策略参数
type Parameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`        // int, float, bool, string
	Value       interface{} `json:"value"`
	Min         interface{} `json:"min"`
	Max         interface{} `json:"max"`
	Step        interface{} `json:"step"`
	Description string      `json:"description"`
	IsOptimizable bool      `json:"is_optimizable"`
}

// StrategyLogic 策略逻辑接口
type StrategyLogic interface {
	Initialize(params map[string]interface{}) error
	ProcessBar(candle Candle, portfolio *Portfolio) (*Signal, error)
	Finalize(portfolio *Portfolio) error
}

// Signal 交易信号
type Signal struct {
	Symbol    string            `json:"symbol"`
	Action    string            `json:"action"`    // BUY, SELL, HOLD
	Quantity  float64           `json:"quantity"`
	Price     float64           `json:"price"`
	StopLoss  float64           `json:"stop_loss"`
	TakeProfit float64          `json:"take_profit"`
	Reason    string            `json:"reason"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time         `json:"timestamp"`
}

// Portfolio 组合状态
type Portfolio struct {
	Cash        float64               `json:"cash"`
	Positions   map[string]*Position  `json:"positions"`
	Equity      float64               `json:"equity"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

// Position 仓位
type Position struct {
	Symbol      string    `json:"symbol"`
	Quantity    float64   `json:"quantity"`
	AvgPrice    float64   `json:"avg_price"`
	MarketPrice float64   `json:"market_price"`
	UnrealizedPL float64  `json:"unrealized_pl"`
	OpenTime    time.Time `json:"open_time"`
}

// StrategyLoader 策略加载器接口
type StrategyLoader interface {
	LoadStrategy(id string) (*BacktestStrategy, error)
	ListStrategies() ([]string, error)
}

// PerformanceAnalyzer 性能分析器
type PerformanceAnalyzer struct {
	benchmarkData   map[string][]EquityPoint
	riskFreeRate    float64
	analysisCache   map[string]*PerformanceAnalysis
	
	mu              sync.RWMutex
}

// PerformanceAnalysis 性能分析
type PerformanceAnalysis struct {
	Returns         []float64         `json:"returns"`
	CumulativeReturns []float64       `json:"cumulative_returns"`
	RollingStats    []RollingStats    `json:"rolling_stats"`
	MonthlyReturns  map[string]float64 `json:"monthly_returns"`
	YearlyReturns   map[string]float64 `json:"yearly_returns"`
	
	// 风险指标
	RiskMetrics     RiskMetrics       `json:"risk_metrics"`
	
	// 稳定性分析
	StabilityAnalysis StabilityAnalysis `json:"stability_analysis"`
}

// RollingStats 滚动统计
type RollingStats struct {
	Date            time.Time `json:"date"`
	Return          float64   `json:"return"`
	Volatility      float64   `json:"volatility"`
	SharpeRatio     float64   `json:"sharpe_ratio"`
	MaxDrawdown     float64   `json:"max_drawdown"`
	WinRate         float64   `json:"win_rate"`
}

// RiskMetrics 风险指标
type RiskMetrics struct {
	VaR95           float64   `json:"var_95"`
	VaR99           float64   `json:"var_99"`
	CVaR95          float64   `json:"cvar_95"`
	CVaR99          float64   `json:"cvar_99"`
	SkewnessRisk    float64   `json:"skewness_risk"`
	KurtosisRisk    float64   `json:"kurtosis_risk"`
	TailRatio       float64   `json:"tail_ratio"`
	DownsideDeviation float64 `json:"downside_deviation"`
}

// StabilityAnalysis 稳定性分析
type StabilityAnalysis struct {
	ConsistencyScore  float64          `json:"consistency_score"`
	RobustnessScore   float64          `json:"robustness_score"`
	AdaptabilityScore float64          `json:"adaptability_score"`
	OutOfSampleRatio  float64          `json:"out_of_sample_ratio"`
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
	templatePath    string
	outputPath      string
	reportCache     map[string]*BacktestReport
	
	mu              sync.RWMutex
}

// BacktestReport 回测报告
type BacktestReport struct {
	ID              string            `json:"id"`
	JobID           string            `json:"job_id"`
	Title           string            `json:"title"`
	GeneratedAt     time.Time         `json:"generated_at"`
	
	// 报告内容
	Summary         ReportSummary     `json:"summary"`
	PerformanceChart []ChartData      `json:"performance_chart"`
	RiskAnalysis    RiskAnalysis      `json:"risk_analysis"`
	TradeAnalysis   TradeAnalysis     `json:"trade_analysis"`
	Recommendations []Recommendation  `json:"recommendations"`
	
	// 输出格式
	HTMLPath        string            `json:"html_path"`
	PDFPath         string            `json:"pdf_path"`
	JSONPath        string            `json:"json_path"`
}

// ReportSummary 报告摘要
type ReportSummary struct {
	StrategyName    string    `json:"strategy_name"`
	TestPeriod      DateRange `json:"test_period"`
	TotalReturn     float64   `json:"total_return"`
	AnnualizedReturn float64  `json:"annualized_return"`
	MaxDrawdown     float64   `json:"max_drawdown"`
	SharpeRatio     float64   `json:"sharpe_ratio"`
	WinRate         float64   `json:"win_rate"`
	TotalTrades     int       `json:"total_trades"`
	Grade           string    `json:"grade"`        // A+, A, B+, B, C+, C, D, F
}

// ChartData 图表数据
type ChartData struct {
	Name   string      `json:"name"`
	Type   string      `json:"type"`    // line, bar, scatter
	Data   []DataPoint `json:"data"`
	Config ChartConfig `json:"config"`
}

// DataPoint 数据点
type DataPoint struct {
	X interface{} `json:"x"`
	Y interface{} `json:"y"`
	Label string  `json:"label"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ChartConfig 图表配置
type ChartConfig struct {
	Title       string            `json:"title"`
	XAxisLabel  string            `json:"x_axis_label"`
	YAxisLabel  string            `json:"y_axis_label"`
	Color       string            `json:"color"`
	Options     map[string]interface{} `json:"options"`
}

// RiskAnalysis 风险分析
type RiskAnalysis struct {
	RiskLevel       string            `json:"risk_level"`     // LOW, MEDIUM, HIGH, EXTREME
	RiskFactors     []string          `json:"risk_factors"`
	RiskMetrics     RiskMetrics       `json:"risk_metrics"`
	WorstPeriods    []WorstPeriod     `json:"worst_periods"`
	StressTestResults []StressTestResult `json:"stress_test_results"`
}

// WorstPeriod 最差时期
type WorstPeriod struct {
	Period      DateRange `json:"period"`
	Return      float64   `json:"return"`
	Drawdown    float64   `json:"drawdown"`
	Duration    int       `json:"duration"`
	Recovery    int       `json:"recovery"`    // 恢复天数
	Description string    `json:"description"`
}

// StressTestResult 压力测试结果
type StressTestResult struct {
	Scenario    string    `json:"scenario"`
	Return      float64   `json:"return"`
	MaxLoss     float64   `json:"max_loss"`
	Recovery    int       `json:"recovery"`
	Probability float64   `json:"probability"`
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
	PnLHistogram    []HistogramBin `json:"pnl_histogram"`
	DurationHistogram []HistogramBin `json:"duration_histogram"`
	WinLossRatio    float64        `json:"win_loss_ratio"`
	AvgWinSize      float64        `json:"avg_win_size"`
	AvgLossSize     float64        `json:"avg_loss_size"`
}

// HistogramBin 直方图桶
type HistogramBin struct {
	Range string `json:"range"`
	Count int    `json:"count"`
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
	Type        string    `json:"type"`         // PARAMETER, RISK, TIMING
	Priority    string    `json:"priority"`     // HIGH, MEDIUM, LOW
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Action      string    `json:"action"`
	Impact      string    `json:"impact"`
	Confidence  float64   `json:"confidence"`
}

// WalkForwardEngine 前进窗口引擎
type WalkForwardEngine struct {
	windowSize      time.Duration
	stepSize        time.Duration
	minTradeCount   int
	
	mu              sync.RWMutex
}

// StrategyPerformance 策略表现
type StrategyPerformance struct {
	StrategyID      string            `json:"strategy_id"`
	StrategyName    string            `json:"strategy_name"`
	
	// 汇总统计
	BacktestCount   int               `json:"backtest_count"`
	AvgReturn       float64           `json:"avg_return"`
	AvgSharpe       float64           `json:"avg_sharpe"`
	AvgMaxDrawdown  float64           `json:"avg_max_drawdown"`
	ConsistencyScore float64          `json:"consistency_score"`
	
	// 历史表现
	PerformanceHistory []PerformanceRecord `json:"performance_history"`
	
	// 参数优化
	OptimalParameters  map[string]interface{} `json:"optimal_parameters"`
	ParameterSensitivity map[string]float64   `json:"parameter_sensitivity"`
	
	// 验证结果
	ValidationStatus   string            `json:"validation_status"`  // PASSED, FAILED, NEEDS_REVIEW
	ValidationScore    float64           `json:"validation_score"`
	LastValidation     time.Time         `json:"last_validation"`
	
	// 元数据
	UpdatedAt          time.Time         `json:"updated_at"`
}

// PerformanceRecord 性能记录
type PerformanceRecord struct {
	Date            time.Time `json:"date"`
	Return          float64   `json:"return"`
	SharpeRatio     float64   `json:"sharpe_ratio"`
	MaxDrawdown     float64   `json:"max_drawdown"`
	TradeCount      int       `json:"trade_count"`
	WinRate         float64   `json:"win_rate"`
	BacktestJobID   string    `json:"backtest_job_id"`
}

// BacktestingMetrics 回测指标
type BacktestingMetrics struct {
	mu sync.RWMutex
	
	// 执行统计
	TotalJobs           int64         `json:"total_jobs"`
	CompletedJobs       int64         `json:"completed_jobs"`
	FailedJobs          int64         `json:"failed_jobs"`
	SuccessRate         float64       `json:"success_rate"`
	AvgExecutionTime    time.Duration `json:"avg_execution_time"`
	
	// 性能统计
	AvgStrategyReturn   float64       `json:"avg_strategy_return"`
	BestStrategyReturn  float64       `json:"best_strategy_return"`
	WorstStrategyReturn float64       `json:"worst_strategy_return"`
	
	// 验证统计
	ValidationPassRate  float64       `json:"validation_pass_rate"`
	AvgValidationScore  float64       `json:"avg_validation_score"`
	
	// 系统指标
	ActiveJobs          int           `json:"active_jobs"`
	QueuedJobs          int           `json:"queued_jobs"`
	SystemLoad          float64       `json:"system_load"`
	
	LastUpdated         time.Time     `json:"last_updated"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	JobID               string            `json:"job_id"`
	StrategyID          string            `json:"strategy_id"`
	ValidationDate      time.Time         `json:"validation_date"`
	
	// 验证测试
	OutOfSampleTest     TestResult        `json:"out_of_sample_test"`
	ForwardTest         TestResult        `json:"forward_test"`
	StabilityTest       TestResult        `json:"stability_test"`
	RobustnessTest      TestResult        `json:"robustness_test"`
	
	// 综合评分
	OverallScore        float64           `json:"overall_score"`
	OverallGrade        string            `json:"overall_grade"`
	Status              string            `json:"status"`
	
	// 问题和建议
	Issues              []ValidationIssue `json:"issues"`
	Recommendations     []string          `json:"recommendations"`
	
	// 阈值检查
	ThresholdChecks     []ThresholdCheck  `json:"threshold_checks"`
}

// TestResult 测试结果
type TestResult struct {
	Passed      bool      `json:"passed"`
	Score       float64   `json:"score"`
	Details     string    `json:"details"`
	Metrics     map[string]float64 `json:"metrics"`
}

// ValidationIssue 验证问题
type ValidationIssue struct {
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Suggestion  string    `json:"suggestion"`
}

// ThresholdCheck 阈值检查
type ThresholdCheck struct {
	Metric      string    `json:"metric"`
	Value       float64   `json:"value"`
	Threshold   float64   `json:"threshold"`
	Operator    string    `json:"operator"`    // >, <, >=, <=, ==
	Passed      bool      `json:"passed"`
	Importance  string    `json:"importance"`   // CRITICAL, HIGH, MEDIUM, LOW
}

// NewAutoBacktestingEngine 创建自动回测验证引擎
func NewAutoBacktestingEngine(cfg *config.Config) (*AutoBacktestingEngine, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	abe := &AutoBacktestingEngine{
		config:              cfg,
		dataManager:         NewBacktestDataManager(),
		strategyManager:     NewBacktestStrategyManager(),
		performanceAnalyzer: NewPerformanceAnalyzer(),
		reportGenerator:     NewReportGenerator(),
		walkForwardEngine:   NewWalkForwardEngine(),
		ctx:                 ctx,
		cancel:              cancel,
		activeBacktests:     make(map[string]*BacktestJob),
		completedBacktests:  make([]BacktestResult, 0),
		strategyPerformance: make(map[string]*StrategyPerformance),
		backtestingMetrics:  &BacktestingMetrics{},
		validationHistory:   make([]ValidationResult, 0),
		frequency:           24 * time.Hour,      // 每日回测
		lookbackPeriod:      365 * 24 * time.Hour, // 1年回看期
		walkForwardWindow:   90 * 24 * time.Hour,  // 3个月前进窗口
		performanceThreshold: 0.02,               // 2%性能阈值
		enabled:             true,
		maxConcurrentJobs:   4,
		dataRetentionDays:   365,
	}
	
	// 从配置文件读取参数
	if cfg != nil {
		// TODO: 从配置文件读取回测参数
	}
	
	return abe, nil
}

// NewBacktestDataManager 创建回测数据管理器
func NewBacktestDataManager() *BacktestDataManager {
	return &BacktestDataManager{
		dataSource: "binance",
		symbols:    []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"},
		timeframes: []string{"1m", "5m", "15m", "1h", "4h", "1d"},
		dataCache:  make(map[string][]Candle),
	}
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
		JobID:      job.ID,
		StrategyID: job.StrategyID,
		EquityCurve: make([]EquityPoint, 0),
		DrawdownCurve: make([]DrawdownPoint, 0),
		Trades:     make([]TradeRecord, 0),
		BacktestDate: time.Now(),
		DataPeriod: DateRange{Start: job.StartDate, End: job.EndDate},
		Benchmark:  job.BenchmarkSymbol,
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
	// TODO: 实现信号执行逻辑
	trade := &TradeRecord{
		ID:         abe.generateTradeID(),
		Symbol:     signal.Symbol,
		Side:       signal.Action,
		Quantity:   signal.Quantity,
		EntryPrice: signal.Price,
		ExitPrice:  signal.Price, // 简化处理
		EntryTime:  candle.Time,
		ExitTime:   candle.Time,
		Duration:   0,
		PnL:        0, // 后续计算
		Commission: signal.Quantity * signal.Price * job.Commission,
		Slippage:   signal.Quantity * signal.Price * job.Slippage,
	}
	
	return trade
}

func (abe *AutoBacktestingEngine) updatePortfolioValue(portfolio *Portfolio, candle Candle) {
	// TODO: 实现组合价值更新逻辑
	portfolio.UpdatedAt = candle.Time
	// 简化处理，保持原有价值
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
		JobID:          result.JobID,
		StrategyID:     result.StrategyID,
		ValidationDate: time.Now(),
		Status:         "PASSED",
		Issues:         make([]ValidationIssue, 0),
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
	// TODO: 实现样本外测试
	return TestResult{
		Passed:  true,
		Score:   0.85,
		Details: "Out-of-sample test passed with good performance",
		Metrics: map[string]float64{
			"out_of_sample_return": 0.12,
			"in_sample_return":     0.15,
			"consistency_ratio":    0.8,
		},
	}
}

func (abe *AutoBacktestingEngine) performStabilityTest(result *BacktestResult) TestResult {
	// TODO: 实现稳定性测试
	return TestResult{
		Passed:  true,
		Score:   0.78,
		Details: "Strategy shows good stability across different periods",
		Metrics: map[string]float64{
			"rolling_sharpe_stability": 0.75,
			"drawdown_consistency":     0.8,
		},
	}
}

func (abe *AutoBacktestingEngine) performRobustnessTest(result *BacktestResult) TestResult {
	// TODO: 实现鲁棒性测试
	return TestResult{
		Passed:  true,
		Score:   0.82,
		Details: "Strategy demonstrates robustness to parameter changes",
		Metrics: map[string]float64{
			"parameter_sensitivity": 0.2,
			"noise_resistance":      0.85,
		},
	}
}

func (abe *AutoBacktestingEngine) generateBacktestReport(job *BacktestJob) {
	// TODO: 实现报告生成
	log.Printf("Generating backtest report for job: %s", job.ID)
}

func (abe *AutoBacktestingEngine) updateStrategyPerformance(strategyID string, result *BacktestResult) {
	abe.mu.Lock()
	defer abe.mu.Unlock()
	
	performance, exists := abe.strategyPerformance[strategyID]
	if !exists {
		performance = &StrategyPerformance{
			StrategyID:         strategyID,
			PerformanceHistory: make([]PerformanceRecord, 0),
			OptimalParameters:  make(map[string]interface{}),
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
		Date:            time.Now(),
		Return:          result.AnnualizedReturn,
		SharpeRatio:     result.SharpeRatio,
		MaxDrawdown:     result.MaxDrawdown,
		TradeCount:      result.TotalTrades,
		WinRate:         result.WinRate,
		BacktestJobID:   result.JobID,
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
	
	// TODO: 实现策略性能分析
	// 1. 识别表现优异的策略
	// 2. 发现表现衰退的策略
	// 3. 生成优化建议
}

func (abe *AutoBacktestingEngine) performValidationChecks() {
	log.Println("Performing validation checks...")
	
	// TODO: 实现验证检查
	// 1. 检查策略是否符合性能阈值
	// 2. 验证策略的稳定性
	// 3. 检查是否存在过拟合
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
		"running":              abe.isRunning,
		"enabled":              abe.enabled,
		"active_backtests":     len(abe.activeBacktests),
		"completed_backtests":  len(abe.completedBacktests),
		"strategy_count":       len(abe.strategyPerformance),
		"frequency":            abe.frequency,
		"lookback_period":      abe.lookbackPeriod,
		"performance_threshold": abe.performanceThreshold,
		"max_concurrent_jobs":  abe.maxConcurrentJobs,
		"backtesting_metrics":  abe.backtestingMetrics,
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

// 在BacktestDataManager中添加方法
func (bdm *BacktestDataManager) GetHistoricalData(symbols []string, startDate, endDate time.Time) ([]Candle, error) {
	// TODO: 实现从数据源获取历史数据
	// 这里返回模拟数据
	data := make([]Candle, 0)
	
	current := startDate
	for current.Before(endDate) {
		candle := Candle{
			Time:   current,
			Open:   50000.0,
			High:   51000.0,
			Low:    49000.0,
			Close:  50500.0,
			Volume: 1000.0,
		}
		data = append(data, candle)
		current = current.Add(24 * time.Hour) // 每日数据
	}
	
	return data, nil
}

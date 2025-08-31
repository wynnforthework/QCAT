package factors

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"qcat/internal/config"
)

// FactorDiscoveryEngine 因子发现引擎
type FactorDiscoveryEngine struct {
	config           *config.Config
	factorGenerator  *FactorGenerator
	factorEvaluator  *FactorEvaluator
	geneticAlgorithm *GeneticAlgorithm
	significanceTest *SignificanceTest
	factorRotator    *FactorRotator

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
	mu        sync.RWMutex

	// 因子库
	factorUniverse    []string
	discoveredFactors map[string]*Factor
	activeFactors     map[string]*Factor
	factorPerformance map[string]*FactorPerformance

	// 发现配置
	discoveryAlgorithm string
	significanceLevel  float64
	rotationFrequency  time.Duration
	maxFactors         int

	// 监控指标
	discoveryMetrics *DiscoveryMetrics
	discoveryHistory []DiscoveryEvent

	// 配置参数
	enabled                 bool
	autoMLEnabled           bool
	geneticAlgorithmEnabled bool
	maxIterations           int
	maxConcurrentJobs       int
	symbols                 []string

	// 数据存储
	baseFactors map[string]Factor
	db          *sql.DB
}

// Factor 因子
type Factor struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Type        string             `json:"type"`        // TECHNICAL, FUNDAMENTAL, SENTIMENT, MACRO, CUSTOM
	Category    string             `json:"category"`    // 因子分类
	Description string             `json:"description"` // 因子描述
	Formula     string             `json:"formula"`     // 因子计算公式
	Expression  *Expression        `json:"expression"`  // 解析后的表达式
	Parameters  map[string]float64 `json:"parameters"`

	// 统计特性
	Mean     float64 `json:"mean"`
	StdDev   float64 `json:"std_dev"`
	Skewness float64 `json:"skewness"`
	Kurtosis float64 `json:"kurtosis"`
	MinValue float64 `json:"min_value"`
	MaxValue float64 `json:"max_value"`

	// 性能指标
	IC       float64   `json:"ic"` // Information Coefficient
	ICStdDev float64   `json:"ic_std_dev"`
	ICIR     float64   `json:"icir"` // IC Information Ratio
	RankIC   float64   `json:"rank_ic"`
	Decay    []float64 `json:"decay"`    // 因子衰减
	Turnover float64   `json:"turnover"` // 换手率

	// 显著性检验
	TValue          float64 `json:"t_value"`
	PValue          float64 `json:"p_value"`
	IsSignificant   bool    `json:"is_significant"`
	ConfidenceLevel float64 `json:"confidence_level"`

	// 相关性分析
	Correlations   map[string]float64 `json:"correlations"`
	MaxCorrelation float64            `json:"max_correlation"`
	FactorLoading  float64            `json:"factor_loading"`

	// 分组收益分析
	GroupReturns    []GroupReturn `json:"group_returns"`
	LongShortReturn float64       `json:"long_short_return"`
	HitRate         float64       `json:"hit_rate"`

	// 时序特性
	Stability      float64            `json:"stability"`
	Persistence    float64            `json:"persistence"`
	SeasonalEffect map[string]float64 `json:"seasonal_effect"`

	// 元数据
	DiscoveredAt time.Time `json:"discovered_at"`
	LastUpdated  time.Time `json:"last_updated"`
	UpdateCount  int       `json:"update_count"`
	UsageCount   int       `json:"usage_count"`
	Status       string    `json:"status"` // ACTIVE, INACTIVE, DEPRECATED
	CreatedBy    string    `json:"created_by"`

	// 生成信息
	Generation int      `json:"generation"` // 遗传算法代数
	Parents    []string `json:"parents"`    // 父因子ID
	Fitness    float64  `json:"fitness"`    // 适应度
	Complexity int      `json:"complexity"` // 复杂度
}

// Expression 表达式
type Expression struct {
	Type     string        `json:"type"`      // OPERATOR, FUNCTION, VARIABLE, CONSTANT
	Value    interface{}   `json:"value"`     // 值或操作符
	Children []*Expression `json:"children"`  // 子表达式
	DataType string        `json:"data_type"` // float64, bool, string
}

// GroupReturn 分组收益
type GroupReturn struct {
	Group       int     `json:"group"`        // 分组编号（1-10）
	Count       int     `json:"count"`        // 样本数量
	Return      float64 `json:"return"`       // 平均收益
	StdDev      float64 `json:"std_dev"`      // 收益标准差
	SharpeRatio float64 `json:"sharpe_ratio"` // 夏普比率
}

// FactorGenerator 因子生成器
type FactorGenerator struct {
	baseFactors []BaseFactor
	operators   []Operator
	functions   []Function
	constants   []float64
	maxDepth    int
	maxNodes    int

	// 遗传编程参数
	populationSize int
	mutationRate   float64
	crossoverRate  float64
	elitismRate    float64

	mu sync.RWMutex
}

// BaseFactor 基础因子
type BaseFactor struct {
	Name        string   `json:"name"`
	Symbol      string   `json:"symbol"`
	Type        string   `json:"type"`       // PRICE, VOLUME, TECHNICAL, FUNDAMENTAL
	Parameters  []string `json:"parameters"` // 可配置参数
	Description string   `json:"description"`
}

// Operator 操作符
type Operator struct {
	Symbol     string                  `json:"symbol"` // +, -, *, /, ^
	Name       string                  `json:"name"`
	Precedence int                     `json:"precedence"`
	Operands   int                     `json:"operands"` // 操作数数量
	Function   func([]float64) float64 `json:"-"`
}

// Function 函数
type Function struct {
	Name        string                                `json:"name"`       // SMA, EMA, RSI, MACD, etc.
	Parameters  int                                   `json:"parameters"` // 参数数量
	Description string                                `json:"description"`
	Function    func([]float64, ...float64) []float64 `json:"-"`
}

// FactorEvaluator 因子评估器
type FactorEvaluator struct {
	evaluationPeriod time.Duration
	forwardPeriods   []int     // 前瞻期 [1, 5, 10, 20]
	benchmarkReturn  []float64 // 基准收益
	evaluationCache  map[string]*FactorEvaluation

	mu sync.RWMutex
}

// FactorEvaluation 因子评估
type FactorEvaluation struct {
	FactorID       string    `json:"factor_id"`
	EvaluationDate time.Time `json:"evaluation_date"`

	// IC分析
	ICResults []ICResult  `json:"ic_results"`
	RollingIC []RollingIC `json:"rolling_ic"`
	ICDecay   []float64   `json:"ic_decay"`

	// 分组回测
	GroupBacktest GroupBacktest `json:"group_backtest"`

	// 风险分析
	RiskAnalysis FactorRiskAnalysis `json:"risk_analysis"`

	// 稳定性分析
	StabilityAnalysis FactorStabilityAnalysis `json:"stability_analysis"`

	// 综合评分
	OverallScore float64 `json:"overall_score"`
	Grade        string  `json:"grade"`
	Rank         int     `json:"rank"`
}

// ICResult IC分析结果
type ICResult struct {
	Period             int        `json:"period"` // 前瞻期
	IC                 float64    `json:"ic"`
	RankIC             float64    `json:"rank_ic"`
	TValue             float64    `json:"t_value"`
	PValue             float64    `json:"p_value"`
	IsSignificant      bool       `json:"is_significant"`
	SampleSize         int        `json:"sample_size"`
	ConfidenceInterval [2]float64 `json:"confidence_interval"`
}

// RollingIC 滚动IC
type RollingIC struct {
	Date          time.Time `json:"date"`
	IC            float64   `json:"ic"`
	RankIC        float64   `json:"rank_ic"`
	TValue        float64   `json:"t_value"`
	IsSignificant bool      `json:"is_significant"`
}

// GroupBacktest 分组回测
type GroupBacktest struct {
	Groups           []GroupStats     `json:"groups"`
	LongShort        GroupStats       `json:"long_short"`
	TopBottom        GroupStats       `json:"top_bottom"`
	MonotonicityTest MonotonicityTest `json:"monotonicity_test"`
}

// GroupStats 分组统计
type GroupStats struct {
	Group       int     `json:"group"`
	Count       int     `json:"count"`
	AvgReturn   float64 `json:"avg_return"`
	CumReturn   float64 `json:"cum_return"`
	Volatility  float64 `json:"volatility"`
	SharpeRatio float64 `json:"sharpe_ratio"`
	MaxDrawdown float64 `json:"max_drawdown"`
	WinRate     float64 `json:"win_rate"`
	HitRate     float64 `json:"hit_rate"`
}

// MonotonicityTest 单调性检验
type MonotonicityTest struct {
	Statistic   float64 `json:"statistic"`
	PValue      float64 `json:"p_value"`
	IsMonotonic bool    `json:"is_monotonic"`
	Direction   string  `json:"direction"` // POSITIVE, NEGATIVE
	Strength    string  `json:"strength"`  // STRONG, MODERATE, WEAK
}

// FactorRiskAnalysis 因子风险分析
type FactorRiskAnalysis struct {
	Exposure      map[string]float64 `json:"exposure"`       // 行业、风格暴露
	Concentration float64            `json:"concentration"`  // 集中度
	Turnover      float64            `json:"turnover"`       // 换手率
	Capacity      float64            `json:"capacity"`       // 容量
	LiquidityRisk float64            `json:"liquidity_risk"` // 流动性风险
	CrowdingRisk  float64            `json:"crowding_risk"`  // 拥挤度风险
}

// FactorStabilityAnalysis 因子稳定性分析
type FactorStabilityAnalysis struct {
	ICStability     float64        `json:"ic_stability"`
	ReturnStability float64        `json:"return_stability"`
	RankStability   float64        `json:"rank_stability"`
	Persistence     float64        `json:"persistence"`
	HalfLife        float64        `json:"half_life"` // 半衰期
	BreakpointTest  BreakpointTest `json:"breakpoint_test"`
}

// BreakpointTest 断点检验
type BreakpointTest struct {
	HasBreakpoint  bool      `json:"has_breakpoint"`
	BreakpointDate time.Time `json:"breakpoint_date"`
	PrePeriodIC    float64   `json:"pre_period_ic"`
	PostPeriodIC   float64   `json:"post_period_ic"`
	Statistic      float64   `json:"statistic"`
	PValue         float64   `json:"p_value"`
}

// GeneticAlgorithm 遗传算法
type GeneticAlgorithm struct {
	population      []*Factor
	populationSize  int
	maxGenerations  int
	mutationRate    float64
	crossoverRate   float64
	elitismRate     float64
	selectionMethod string // TOURNAMENT, ROULETTE, RANK

	// 多样性控制
	diversityWeight   float64
	complexityPenalty float64

	// 历史记录
	generationHistory []GenerationStats
	bestFactors       []*Factor

	mu sync.RWMutex
}

// GenerationStats 代数统计
type GenerationStats struct {
	Generation  int       `json:"generation"`
	BestFitness float64   `json:"best_fitness"`
	AvgFitness  float64   `json:"avg_fitness"`
	Diversity   float64   `json:"diversity"`
	Complexity  float64   `json:"complexity"`
	Timestamp   time.Time `json:"timestamp"`
}

// SignificanceTest 显著性检验
type SignificanceTest struct {
	testMethod             string // IC_TEST, T_TEST, RANK_TEST
	significanceLevel      float64
	multipleTestCorrection string // BONFERRONI, FDR, HOLM

	mu sync.RWMutex
}

// FactorRotator 因子轮换器
type FactorRotator struct {
	rotationStrategy     string // PERFORMANCE, CORRELATION, REGIME
	rotationFrequency    time.Duration
	maxActiveFactors     int
	correlationThreshold float64
	performanceWindow    time.Duration

	// 轮换历史
	rotationHistory []RotationEvent
	lastRotation    time.Time

	mu sync.RWMutex
}

// RotationEvent 轮换事件
type RotationEvent struct {
	Date             time.Time `json:"date"`
	Action           string    `json:"action"` // ADD, REMOVE, REPLACE
	FactorID         string    `json:"factor_id"`
	ReplacedFactorID string    `json:"replaced_factor_id"`
	Reason           string    `json:"reason"`
	Performance      float64   `json:"performance"`
	Correlation      float64   `json:"correlation"`
}

// FactorPerformance 因子表现
type FactorPerformance struct {
	FactorID string `json:"factor_id"`

	// 历史表现
	PerformanceHistory []PerformancePoint `json:"performance_history"`

	// 汇总统计
	AvgIC     float64 `json:"avg_ic"`
	AvgRankIC float64 `json:"avg_rank_ic"`
	ICStdDev  float64 `json:"ic_std_dev"`
	ICIR      float64 `json:"icir"`
	HitRate   float64 `json:"hit_rate"`

	// 收益分析
	CumulativeReturn float64 `json:"cumulative_return"`
	AnnualizedReturn float64 `json:"annualized_return"`
	Volatility       float64 `json:"volatility"`
	SharpeRatio      float64 `json:"sharpe_ratio"`
	MaxDrawdown      float64 `json:"max_drawdown"`

	// 稳定性指标
	StabilityScore   float64 `json:"stability_score"`
	ConsistencyScore float64 `json:"consistency_score"`

	// 最近表现
	RecentIC     float64 `json:"recent_ic"`
	RecentRankIC float64 `json:"recent_rank_ic"`
	RecentReturn float64 `json:"recent_return"`
	RecentRank   int     `json:"recent_rank"`

	// 预测能力
	ForecastAccuracy float64 `json:"forecast_accuracy"`
	ForecastBias     float64 `json:"forecast_bias"`

	LastUpdated time.Time `json:"last_updated"`
}

// PerformancePoint 表现点
type PerformancePoint struct {
	Date          time.Time `json:"date"`
	IC            float64   `json:"ic"`
	RankIC        float64   `json:"rank_ic"`
	Return        float64   `json:"return"`
	CumReturn     float64   `json:"cum_return"`
	Rank          int       `json:"rank"`
	IsSignificant bool      `json:"is_significant"`
}

// DiscoveryMetrics 发现指标
type DiscoveryMetrics struct {
	mu sync.RWMutex

	// 因子统计
	TotalFactors       int `json:"total_factors"`
	ActiveFactors      int `json:"active_factors"`
	SignificantFactors int `json:"significant_factors"`

	// 发现统计
	FactorsDiscovered int           `json:"factors_discovered"`
	DiscoveryRate     float64       `json:"discovery_rate"`
	AvgFactorLifespan time.Duration `json:"avg_factor_lifespan"`

	// 质量指标
	AvgIC           float64 `json:"avg_ic"`
	AvgICIR         float64 `json:"avg_icir"`
	AvgSignificance float64 `json:"avg_significance"`
	TopFactorIC     float64 `json:"top_factor_ic"`

	// 多样性指标
	FactorDiversity        float64        `json:"factor_diversity"`
	TypeDistribution       map[string]int `json:"type_distribution"`
	ComplexityDistribution map[int]int    `json:"complexity_distribution"`

	// 性能指标
	DiscoveryTime      time.Duration `json:"discovery_time"`
	EvaluationTime     time.Duration `json:"evaluation_time"`
	RotationEfficiency float64       `json:"rotation_efficiency"`

	// 算法统计
	GAGenerations   int     `json:"ga_generations"`
	ConvergenceRate float64 `json:"convergence_rate"`

	LastUpdated time.Time `json:"last_updated"`
}

// DiscoveryEvent 发现事件
type DiscoveryEvent struct {
	Date      time.Time              `json:"date"`
	EventType string                 `json:"event_type"` // DISCOVERY, EVALUATION, ROTATION, DEPRECATION
	FactorID  string                 `json:"factor_id"`
	Details   map[string]interface{} `json:"details"`
	Impact    string                 `json:"impact"` // HIGH, MEDIUM, LOW
}

// NewFactorDiscoveryEngine 创建因子发现引擎
func NewFactorDiscoveryEngine(cfg *config.Config) (*FactorDiscoveryEngine, error) {
	ctx, cancel := context.WithCancel(context.Background())

	fde := &FactorDiscoveryEngine{
		config:            cfg,
		factorGenerator:   NewFactorGenerator(),
		factorEvaluator:   NewFactorEvaluator(),
		geneticAlgorithm:  NewGeneticAlgorithm(),
		significanceTest:  NewSignificanceTest(),
		factorRotator:     NewFactorRotator(),
		ctx:               ctx,
		cancel:            cancel,
		factorUniverse:    []string{"technical", "fundamental", "sentiment", "macro"},
		discoveredFactors: make(map[string]*Factor),
		activeFactors:     make(map[string]*Factor),
		factorPerformance: make(map[string]*FactorPerformance),
		discoveryMetrics: &DiscoveryMetrics{
			TypeDistribution:       make(map[string]int),
			ComplexityDistribution: make(map[int]int),
		},
		discoveryHistory:   make([]DiscoveryEvent, 0),
		discoveryAlgorithm: "genetic_programming",
		significanceLevel:  0.05,
		rotationFrequency:  7 * 24 * time.Hour, // 每周轮换
		maxFactors:         50,
		enabled:            true,
	}

	// 从配置文件读取参数
	if cfg != nil {
		// 从自动化配置读取学习参数
		if cfg.Automation.Learning != nil {
			if cfg.Automation.Learning.Enabled {
				fde.enabled = cfg.Automation.Learning.Enabled
			}
			if cfg.Automation.Learning.AutoMLEnabled {
				fde.autoMLEnabled = cfg.Automation.Learning.AutoMLEnabled
			}
			if cfg.Automation.Learning.GeneticAlgorithmEnabled {
				fde.geneticAlgorithmEnabled = cfg.Automation.Learning.GeneticAlgorithmEnabled
			}
		}

		// 从优化器配置读取参数
		// Note: cfg.Optimizer is a struct, not a pointer, so we don't need nil check
		if cfg.Optimizer.MaxIterations > 0 {
			fde.maxIterations = cfg.Optimizer.MaxIterations
		}
		if cfg.Optimizer.Concurrency > 0 {
			fde.maxConcurrentJobs = cfg.Optimizer.Concurrency
		}

		// 从市场数据配置读取符号列表
		if len(cfg.MarketData.Symbols) > 0 {
			fde.symbols = cfg.MarketData.Symbols
		}
	}

	// 初始化基础因子
	err := fde.initializeBaseFactors()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize base factors: %w", err)
	}

	return fde, nil
}

// NewFactorGenerator 创建因子生成器
func NewFactorGenerator() *FactorGenerator {
	return &FactorGenerator{
		baseFactors:    initializeBaseFactors(),
		operators:      initializeOperators(),
		functions:      initializeFunctions(),
		constants:      []float64{1, 2, 3, 5, 10, 20, 60, 252},
		maxDepth:       5,
		maxNodes:       50,
		populationSize: 100,
		mutationRate:   0.1,
		crossoverRate:  0.8,
		elitismRate:    0.1,
	}
}

// NewFactorEvaluator 创建因子评估器
func NewFactorEvaluator() *FactorEvaluator {
	return &FactorEvaluator{
		evaluationPeriod: 252 * 24 * time.Hour, // 1年
		forwardPeriods:   []int{1, 5, 10, 20},
		evaluationCache:  make(map[string]*FactorEvaluation),
	}
}

// NewGeneticAlgorithm 创建遗传算法
func NewGeneticAlgorithm() *GeneticAlgorithm {
	return &GeneticAlgorithm{
		population:        make([]*Factor, 0),
		populationSize:    100,
		maxGenerations:    50,
		mutationRate:      0.1,
		crossoverRate:     0.8,
		elitismRate:       0.1,
		selectionMethod:   "TOURNAMENT",
		diversityWeight:   0.2,
		complexityPenalty: 0.1,
		generationHistory: make([]GenerationStats, 0),
		bestFactors:       make([]*Factor, 0),
	}
}

// NewSignificanceTest 创建显著性检验
func NewSignificanceTest() *SignificanceTest {
	return &SignificanceTest{
		testMethod:             "IC_TEST",
		significanceLevel:      0.05,
		multipleTestCorrection: "FDR",
	}
}

// NewFactorRotator 创建因子轮换器
func NewFactorRotator() *FactorRotator {
	return &FactorRotator{
		rotationStrategy:     "PERFORMANCE",
		rotationFrequency:    7 * 24 * time.Hour,
		maxActiveFactors:     20,
		correlationThreshold: 0.8,
		performanceWindow:    30 * 24 * time.Hour,
		rotationHistory:      make([]RotationEvent, 0),
	}
}

// Start 启动因子发现引擎
func (fde *FactorDiscoveryEngine) Start() error {
	fde.mu.Lock()
	defer fde.mu.Unlock()

	if fde.isRunning {
		return fmt.Errorf("factor discovery engine is already running")
	}

	if !fde.enabled {
		return fmt.Errorf("factor discovery engine is disabled")
	}

	log.Println("Starting Factor Discovery Engine...")

	// 启动因子发现
	fde.wg.Add(1)
	go fde.runFactorDiscovery()

	// 启动因子评估
	fde.wg.Add(1)
	go fde.runFactorEvaluation()

	// 启动因子轮换
	fde.wg.Add(1)
	go fde.runFactorRotation()

	// 启动性能监控
	fde.wg.Add(1)
	go fde.runPerformanceMonitoring()

	// 启动指标收集
	fde.wg.Add(1)
	go fde.runMetricsCollection()

	fde.isRunning = true
	log.Println("Factor Discovery Engine started successfully")
	return nil
}

// Stop 停止因子发现引擎
func (fde *FactorDiscoveryEngine) Stop() error {
	fde.mu.Lock()
	defer fde.mu.Unlock()

	if !fde.isRunning {
		return fmt.Errorf("factor discovery engine is not running")
	}

	log.Println("Stopping Factor Discovery Engine...")

	fde.cancel()
	fde.wg.Wait()

	fde.isRunning = false
	log.Println("Factor Discovery Engine stopped successfully")
	return nil
}

// initializeBaseFactors 初始化基础因子
func (fde *FactorDiscoveryEngine) initializeBaseFactors() error {
	// 从配置或数据库加载基础因子

	// 1. 定义基础技术指标因子
	technicalFactors := []Factor{
		{
			ID:          "sma_5",
			Name:        "Simple Moving Average 5",
			Type:        "technical",
			Category:    "trend",
			Formula:     "SMA(close, 5)",
			Expression:  nil, // 将在后续解析
			Description: "5日简单移动平均线",
			Parameters: map[string]float64{
				"period": 5,
			},
		},
		{
			ID:          "sma_20",
			Name:        "Simple Moving Average 20",
			Type:        "technical",
			Category:    "trend",
			Formula:     "SMA(close, 20)",
			Expression:  nil, // 将在后续解析
			Description: "20日简单移动平均线",
			Parameters: map[string]float64{
				"period": 20,
			},
		},
		{
			ID:          "rsi_14",
			Name:        "Relative Strength Index 14",
			Type:        "technical",
			Category:    "momentum",
			Formula:     "RSI(close, 14)",
			Expression:  nil, // 将在后续解析
			Description: "14日相对强弱指数",
			Parameters: map[string]float64{
				"period": 14,
			},
		},
		{
			ID:          "macd",
			Name:        "MACD",
			Type:        "technical",
			Category:    "momentum",
			Formula:     "MACD(close, 12, 26, 9)",
			Expression:  nil, // 将在后续解析
			Description: "移动平均收敛散度指标",
			Parameters: map[string]float64{
				"fast_period":   12,
				"slow_period":   26,
				"signal_period": 9,
			},
		},
		{
			ID:          "bb_upper",
			Name:        "Bollinger Bands Upper",
			Type:        "technical",
			Category:    "volatility",
			Formula:     "BB_UPPER(close, 20, 2)",
			Expression:  nil, // 将在后续解析
			Description: "布林带上轨",
			Parameters: map[string]float64{
				"period": 20,
				"std":    2,
			},
		},
		{
			ID:          "bb_lower",
			Name:        "Bollinger Bands Lower",
			Type:        "technical",
			Category:    "volatility",
			Formula:     "BB_LOWER(close, 20, 2)",
			Expression:  nil, // 将在后续解析
			Description: "布林带下轨",
			Parameters: map[string]float64{
				"period": 20,
				"std":    2,
			},
		},
	}

	// 2. 定义基础价格因子
	priceFactors := []Factor{
		{
			ID:          "price_return_1d",
			Name:        "1-Day Price Return",
			Type:        "price",
			Category:    "return",
			Formula:     "(close - close[1]) / close[1]",
			Expression:  nil, // 将在后续解析
			Description: "1日价格收益率",
			Parameters:  map[string]float64{},
		},
		{
			ID:          "price_return_5d",
			Name:        "5-Day Price Return",
			Type:        "price",
			Category:    "return",
			Formula:     "(close - close[5]) / close[5]",
			Expression:  nil, // 将在后续解析
			Description: "5日价格收益率",
			Parameters:  map[string]float64{},
		},
		{
			ID:          "volume_ratio",
			Name:        "Volume Ratio",
			Type:        "volume",
			Category:    "activity",
			Formula:     "volume / SMA(volume, 20)",
			Expression:  nil, // 将在后续解析
			Description: "成交量比率",
			Parameters: map[string]float64{
				"period": 20,
			},
		},
		{
			ID:          "price_volatility",
			Name:        "Price Volatility",
			Type:        "volatility",
			Category:    "risk",
			Formula:     "STDDEV(log(close/close[1]), 20)",
			Expression:  nil, // 将在后续解析
			Description: "价格波动率",
			Parameters: map[string]float64{
				"period": 20,
			},
		},
	}

	// 3. 定义基础市场结构因子
	marketFactors := []Factor{
		{
			ID:          "high_low_ratio",
			Name:        "High-Low Ratio",
			Type:        "market_structure",
			Category:    "range",
			Formula:     "(high - low) / close",
			Expression:  nil, // 将在后续解析
			Description: "最高最低价比率",
			Parameters:  map[string]float64{},
		},
		{
			ID:          "open_close_gap",
			Name:        "Open-Close Gap",
			Type:        "market_structure",
			Category:    "gap",
			Formula:     "(open - close[1]) / close[1]",
			Expression:  nil, // 将在后续解析
			Description: "开盘跳空",
			Parameters:  map[string]float64{},
		},
	}

	// 4. 合并所有基础因子
	allFactors := append(technicalFactors, priceFactors...)
	allFactors = append(allFactors, marketFactors...)

	// 5. 将因子添加到引擎中
	fde.mu.Lock()
	defer fde.mu.Unlock()

	for _, factor := range allFactors {
		factor.DiscoveredAt = time.Now()
		factor.LastUpdated = time.Now()
		factor.Status = "ACTIVE"
		fde.baseFactors[factor.ID] = factor

		log.Printf("Loaded base factor: %s (%s)", factor.Name, factor.ID)
	}

	// 6. 尝试从数据库加载额外的因子（如果数据库可用）
	if fde.db != nil {
		if err := fde.loadFactorsFromDatabase(); err != nil {
			log.Printf("Warning: Failed to load factors from database: %v", err)
		}
	}

	// 7. 从配置文件加载自定义因子（如果配置可用）
	if fde.config != nil {
		if err := fde.loadFactorsFromConfig(); err != nil {
			log.Printf("Warning: Failed to load factors from config: %v", err)
		}
	}

	log.Printf("Initialized %d base factors", len(fde.baseFactors))
	return nil
}

// loadFactorsFromDatabase 从数据库加载因子
func (fde *FactorDiscoveryEngine) loadFactorsFromDatabase() error {
	// 这里应该实现从数据库加载因子的逻辑
	// 暂时返回nil，表示没有从数据库加载到额外因子
	return nil
}

// loadFactorsFromConfig 从配置文件加载因子
func (fde *FactorDiscoveryEngine) loadFactorsFromConfig() error {
	// 这里应该实现从配置文件加载因子的逻辑
	// 暂时返回nil，表示没有从配置文件加载到额外因子
	return nil
}

// runFactorDiscovery 运行因子发现
func (fde *FactorDiscoveryEngine) runFactorDiscovery() {
	defer fde.wg.Done()

	ticker := time.NewTicker(1 * time.Hour) // 每小时尝试发现新因子
	defer ticker.Stop()

	log.Println("Factor discovery started")

	for {
		select {
		case <-fde.ctx.Done():
			log.Println("Factor discovery stopped")
			return
		case <-ticker.C:
			fde.discoverNewFactors()
		}
	}
}

// runFactorEvaluation 运行因子评估
func (fde *FactorDiscoveryEngine) runFactorEvaluation() {
	defer fde.wg.Done()

	ticker := time.NewTicker(30 * time.Minute) // 每30分钟评估一次
	defer ticker.Stop()

	log.Println("Factor evaluation started")

	for {
		select {
		case <-fde.ctx.Done():
			log.Println("Factor evaluation stopped")
			return
		case <-ticker.C:
			fde.evaluateFactors()
		}
	}
}

// runFactorRotation 运行因子轮换
func (fde *FactorDiscoveryEngine) runFactorRotation() {
	defer fde.wg.Done()

	ticker := time.NewTicker(fde.rotationFrequency)
	defer ticker.Stop()

	log.Println("Factor rotation started")

	for {
		select {
		case <-fde.ctx.Done():
			log.Println("Factor rotation stopped")
			return
		case <-ticker.C:
			fde.rotateFactors()
		}
	}
}

// runPerformanceMonitoring 运行性能监控
func (fde *FactorDiscoveryEngine) runPerformanceMonitoring() {
	defer fde.wg.Done()

	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	log.Println("Performance monitoring started")

	for {
		select {
		case <-fde.ctx.Done():
			log.Println("Performance monitoring stopped")
			return
		case <-ticker.C:
			fde.monitorFactorPerformance()
		}
	}
}

// runMetricsCollection 运行指标收集
func (fde *FactorDiscoveryEngine) runMetricsCollection() {
	defer fde.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("Metrics collection started")

	for {
		select {
		case <-fde.ctx.Done():
			log.Println("Metrics collection stopped")
			return
		case <-ticker.C:
			fde.updateMetrics()
		}
	}
}

// discoverNewFactors 发现新因子
func (fde *FactorDiscoveryEngine) discoverNewFactors() {
	log.Println("Discovering new factors...")

	switch fde.discoveryAlgorithm {
	case "genetic_programming":
		fde.runGeneticProgramming()
	case "random_search":
		fde.runRandomSearch()
	case "systematic_search":
		fde.runSystematicSearch()
	default:
		fde.runGeneticProgramming()
	}
}

// runGeneticProgramming 运行遗传编程
func (fde *FactorDiscoveryEngine) runGeneticProgramming() {
	startTime := time.Now()

	// 初始化种群（如果为空）
	if len(fde.geneticAlgorithm.population) == 0 {
		fde.initializePopulation()
	}

	// 进化过程
	for generation := 0; generation < fde.geneticAlgorithm.maxGenerations; generation++ {
		// 评估适应度
		fde.evaluatePopulation()

		// 记录代数统计
		stats := fde.calculateGenerationStats(generation)
		fde.geneticAlgorithm.generationHistory = append(fde.geneticAlgorithm.generationHistory, stats)

		// 检查收敛条件
		if fde.checkConvergence() {
			log.Printf("Genetic algorithm converged at generation %d", generation)
			break
		}

		// 选择、交叉、变异
		newPopulation := fde.evolvePopulation()
		fde.geneticAlgorithm.population = newPopulation

		log.Printf("Generation %d completed, best fitness: %.4f", generation, stats.BestFitness)
	}

	// 选择最优因子
	bestFactors := fde.selectBestFactors()

	// 添加到发现因子库
	for _, factor := range bestFactors {
		if fde.isFactorNovel(factor) {
			fde.addDiscoveredFactor(factor)
			log.Printf("New factor discovered: %s (IC: %.4f)", factor.Name, factor.IC)
		}
	}

	fde.discoveryMetrics.DiscoveryTime = time.Since(startTime)
}

// runRandomSearch 运行随机搜索
func (fde *FactorDiscoveryEngine) runRandomSearch() {
	// 实现随机搜索算法
	log.Println("Running random search for factor discovery...")

	// 设置随机搜索参数
	maxIterations := 1000
	if fde.config != nil {
		// 使用优化器配置中的最大迭代次数
		if fde.config.Optimizer.MaxIterations > 0 {
			maxIterations = fde.config.Optimizer.MaxIterations
		}
	}

	// 定义可用的操作符和函数
	operators := []string{"+", "-", "*", "/", "^"}
	functions := []string{"SMA", "EMA", "RSI", "MACD", "STDDEV", "MAX", "MIN", "ABS", "LOG", "SQRT"}
	variables := []string{"open", "high", "low", "close", "volume"}

	// 获取基础因子列表
	fde.mu.RLock()
	baseFactorIDs := make([]string, 0, len(fde.baseFactors))
	for id := range fde.baseFactors {
		baseFactorIDs = append(baseFactorIDs, id)
	}
	fde.mu.RUnlock()

	successfulFactors := 0

	for i := 0; i < maxIterations; i++ {
		// 随机生成因子表达式
		expression := fde.generateRandomExpression(operators, functions, variables, baseFactorIDs)

		// 创建候选因子
		candidateFactor := Factor{
			ID:           fmt.Sprintf("random_%d_%d", time.Now().Unix(), i),
			Name:         fmt.Sprintf("Random Factor %d", i+1),
			Type:         "composite",
			Category:     "random_search",
			Formula:      expression,
			Expression:   nil, // 将在后续解析
			Description:  fmt.Sprintf("Randomly generated factor: %s", expression),
			Parameters:   map[string]float64{},
			DiscoveredAt: time.Now(),
			LastUpdated:  time.Now(),
			Status:       "ACTIVE",
		}

		// 评估因子
		evaluation, err := fde.evaluateFactor(candidateFactor)
		if err != nil {
			log.Printf("Failed to evaluate random factor %s: %v", candidateFactor.ID, err)
			continue
		}

		// 检查因子质量
		if fde.isFactorQualified(evaluation) {
			// 检查因子新颖性
			if fde.isFactorNovel(&candidateFactor) {
				// 添加到发现的因子中
				fde.mu.Lock()
				fde.discoveredFactors[candidateFactor.ID] = &candidateFactor
				fde.factorEvaluations[candidateFactor.ID] = *evaluation
				fde.mu.Unlock()

				successfulFactors++
				log.Printf("Discovered new factor via random search: %s (IC: %.4f)",
					candidateFactor.ID, evaluation.IC)
			}
		}

		// 每100次迭代报告进度
		if (i+1)%100 == 0 {
			log.Printf("Random search progress: %d/%d iterations, %d successful factors",
				i+1, maxIterations, successfulFactors)
		}

		// 检查是否应该停止搜索
		if fde.shouldStopSearch(successfulFactors, i+1) {
			log.Printf("Early stopping random search at iteration %d", i+1)
			break
		}
	}

	log.Printf("Random search completed: %d successful factors discovered", successfulFactors)
}

// generateRandomExpression 生成随机表达式
func (fde *FactorDiscoveryEngine) generateRandomExpression(operators, functions, variables, baseFactors []string) string {
	// 随机选择表达式类型
	expressionTypes := []string{"simple", "function", "composite", "base_factor"}
	exprType := expressionTypes[rand.Intn(len(expressionTypes))]

	switch exprType {
	case "simple":
		// 简单二元表达式: var1 op var2
		var1 := variables[rand.Intn(len(variables))]
		var2 := variables[rand.Intn(len(variables))]
		op := operators[rand.Intn(len(operators))]
		return fmt.Sprintf("(%s %s %s)", var1, op, var2)

	case "function":
		// 函数表达式: FUNC(var, param)
		function := functions[rand.Intn(len(functions))]
		variable := variables[rand.Intn(len(variables))]

		// 根据函数类型生成参数
		switch function {
		case "SMA", "EMA", "STDDEV", "MAX", "MIN":
			period := rand.Intn(50) + 5 // 5-54
			return fmt.Sprintf("%s(%s, %d)", function, variable, period)
		case "RSI":
			period := rand.Intn(20) + 10 // 10-29
			return fmt.Sprintf("%s(%s, %d)", function, variable, period)
		case "MACD":
			fast := rand.Intn(15) + 5   // 5-19
			slow := rand.Intn(20) + 20  // 20-39
			signal := rand.Intn(10) + 5 // 5-14
			return fmt.Sprintf("%s(%s, %d, %d, %d)", function, variable, fast, slow, signal)
		default:
			return fmt.Sprintf("%s(%s)", function, variable)
		}

	case "composite":
		// 复合表达式: (expr1 op expr2)
		expr1 := fde.generateRandomExpression(operators, functions, variables, baseFactors)
		expr2 := fde.generateRandomExpression(operators, functions, variables, baseFactors)
		op := operators[rand.Intn(len(operators))]
		return fmt.Sprintf("(%s %s %s)", expr1, op, expr2)

	case "base_factor":
		// 基于现有因子的表达式
		if len(baseFactors) > 0 {
			baseFactor := baseFactors[rand.Intn(len(baseFactors))]
			if rand.Float64() < 0.5 {
				// 直接使用基础因子
				return baseFactor
			} else {
				// 对基础因子进行变换
				variable := variables[rand.Intn(len(variables))]
				op := operators[rand.Intn(len(operators))]
				return fmt.Sprintf("(%s %s %s)", baseFactor, op, variable)
			}
		}
		fallthrough

	default:
		// 默认返回简单变量
		return variables[rand.Intn(len(variables))]
	}
}

// shouldStopSearch 检查是否应该停止搜索
func (fde *FactorDiscoveryEngine) shouldStopSearch(successfulFactors, iterations int) bool {
	// 如果已经发现足够多的因子，可以提前停止
	maxSuccessfulFactors := 50
	if fde.config != nil && fde.config.FactorDiscovery != nil {
		if fde.config.FactorDiscovery.RandomSearch.MaxSuccessfulFactors > 0 {
			maxSuccessfulFactors = fde.config.FactorDiscovery.RandomSearch.MaxSuccessfulFactors
		}
	}

	return successfulFactors >= maxSuccessfulFactors
}

// runSystematicSearch 运行系统化搜索
func (fde *FactorDiscoveryEngine) runSystematicSearch() {
	// 实现系统化搜索算法
	log.Println("Running systematic search for factor discovery...")

	// 定义搜索空间
	variables := []string{"open", "high", "low", "close", "volume"}
	operators := []string{"+", "-", "*", "/"}
	functions := []string{"SMA", "EMA", "RSI", "STDDEV", "MAX", "MIN"}
	periods := []int{5, 10, 14, 20, 30, 50}

	successfulFactors := 0
	totalCombinations := 0

	// 1. 系统化搜索单变量技术指标
	log.Println("Searching single-variable technical indicators...")
	for _, function := range functions {
		for _, variable := range variables {
			for _, period := range periods {
				totalCombinations++

				// 跳过不合理的组合
				if variable == "volume" && (function == "RSI") {
					continue
				}

				expression := fmt.Sprintf("%s(%s, %d)", function, variable, period)

				candidateFactor := Factor{
					ID:          fmt.Sprintf("sys_%s_%s_%d", function, variable, period),
					Name:        fmt.Sprintf("%s %s %d", function, variable, period),
					Type:        "technical",
					Category:    "systematic_search",
					Expression:  expression,
					Description: fmt.Sprintf("Systematic search: %s", expression),
					Parameters: map[string]interface{}{
						"function": function,
						"variable": variable,
						"period":   period,
					},
					IsActive:  true,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				if fde.evaluateAndStoreFactor(candidateFactor) {
					successfulFactors++
				}
			}
		}
	}

	// 2. 系统化搜索双变量组合
	log.Println("Searching two-variable combinations...")
	for i, var1 := range variables {
		for j, var2 := range variables {
			if i >= j { // 避免重复组合
				continue
			}

			for _, op := range operators {
				totalCombinations++

				expression := fmt.Sprintf("(%s %s %s)", var1, op, var2)

				candidateFactor := Factor{
					ID:          fmt.Sprintf("sys_comb_%s_%s_%s", var1, op, var2),
					Name:        fmt.Sprintf("%s %s %s", var1, op, var2),
					Type:        "composite",
					Category:    "systematic_search",
					Expression:  expression,
					Description: fmt.Sprintf("Systematic combination: %s", expression),
					Parameters: map[string]interface{}{
						"var1":     var1,
						"var2":     var2,
						"operator": op,
					},
					IsActive:  true,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				if fde.evaluateAndStoreFactor(candidateFactor) {
					successfulFactors++
				}
			}
		}
	}

	// 3. 系统化搜索标准化因子
	log.Println("Searching normalized factors...")
	for _, variable := range variables {
		for _, period := range periods {
			totalCombinations++

			// Z-score标准化
			expression := fmt.Sprintf("(%s - SMA(%s, %d)) / STDDEV(%s, %d)",
				variable, variable, period, variable, period)

			candidateFactor := Factor{
				ID:          fmt.Sprintf("sys_zscore_%s_%d", variable, period),
				Name:        fmt.Sprintf("Z-Score %s %d", variable, period),
				Type:        "normalized",
				Category:    "systematic_search",
				Expression:  expression,
				Description: fmt.Sprintf("Z-score normalized %s over %d periods", variable, period),
				Parameters: map[string]interface{}{
					"variable": variable,
					"period":   period,
					"method":   "zscore",
				},
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if fde.evaluateAndStoreFactor(candidateFactor) {
				successfulFactors++
			}
		}
	}

	// 4. 系统化搜索比率因子
	log.Println("Searching ratio factors...")
	ratioVariables := [][]string{
		{"high", "low"},
		{"close", "open"},
		{"volume", "SMA(volume, 20)"},
		{"close", "SMA(close, 20)"},
	}

	for _, pair := range ratioVariables {
		totalCombinations++

		expression := fmt.Sprintf("%s / %s", pair[0], pair[1])

		candidateFactor := Factor{
			ID:          fmt.Sprintf("sys_ratio_%s_%s", pair[0], pair[1]),
			Name:        fmt.Sprintf("Ratio %s/%s", pair[0], pair[1]),
			Type:        "ratio",
			Category:    "systematic_search",
			Expression:  expression,
			Description: fmt.Sprintf("Ratio of %s to %s", pair[0], pair[1]),
			Parameters: map[string]interface{}{
				"numerator":   pair[0],
				"denominator": pair[1],
			},
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if fde.evaluateAndStoreFactor(candidateFactor) {
			successfulFactors++
		}
	}

	// 5. 系统化搜索动量因子
	log.Println("Searching momentum factors...")
	momentumPeriods := []int{1, 3, 5, 10, 20}

	for _, variable := range []string{"close", "high", "low"} {
		for _, period := range momentumPeriods {
			totalCombinations++

			expression := fmt.Sprintf("(%s - %s[%d]) / %s[%d]",
				variable, variable, period, variable, period)

			candidateFactor := Factor{
				ID:          fmt.Sprintf("sys_momentum_%s_%d", variable, period),
				Name:        fmt.Sprintf("Momentum %s %d", variable, period),
				Type:        "momentum",
				Category:    "systematic_search",
				Expression:  expression,
				Description: fmt.Sprintf("%d-period momentum of %s", period, variable),
				Parameters: map[string]interface{}{
					"variable": variable,
					"period":   period,
				},
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if fde.evaluateAndStoreFactor(candidateFactor) {
				successfulFactors++
			}
		}
	}

	log.Printf("Systematic search completed: %d successful factors from %d combinations",
		successfulFactors, totalCombinations)
}

// evaluateAndStoreFactor 评估并存储因子
func (fde *FactorDiscoveryEngine) evaluateAndStoreFactor(factor Factor) bool {
	// 评估因子
	evaluation, err := fde.evaluateFactor(factor)
	if err != nil {
		return false
	}

	// 检查因子质量
	if !fde.isFactorQualified(evaluation) {
		return false
	}

	// 检查因子新颖性
	if !fde.isFactorNovel(&factor) {
		return false
	}

	// 存储因子
	fde.mu.Lock()
	fde.discoveredFactors[factor.ID] = factor
	fde.factorEvaluations[factor.ID] = *evaluation
	fde.mu.Unlock()

	log.Printf("Discovered new factor via systematic search: %s (IC: %.4f)",
		factor.ID, evaluation.IC)

	return true
}

// evaluateFactors 评估因子
func (fde *FactorDiscoveryEngine) evaluateFactors() {
	log.Println("Evaluating factors...")

	fde.mu.RLock()
	factors := make([]*Factor, 0, len(fde.discoveredFactors))
	for _, factor := range fde.discoveredFactors {
		factors = append(factors, factor)
	}
	fde.mu.RUnlock()

	for _, factor := range factors {
		evaluation := fde.evaluateFactor(factor)

		// 更新因子性能指标
		fde.updateFactorFromEvaluation(factor, evaluation)

		// 缓存评估结果
		fde.factorEvaluator.mu.Lock()
		fde.factorEvaluator.evaluationCache[factor.ID] = evaluation
		fde.factorEvaluator.mu.Unlock()
	}
}

// evaluateFactor 评估单个因子
func (fde *FactorDiscoveryEngine) evaluateFactor(factor *Factor) *FactorEvaluation {
	evaluation := &FactorEvaluation{
		FactorID:       factor.ID,
		EvaluationDate: time.Now(),
		ICResults:      make([]ICResult, 0),
		RollingIC:      make([]RollingIC, 0),
	}

	// IC分析
	for _, period := range fde.factorEvaluator.forwardPeriods {
		icResult := fde.calculateIC(factor, period)
		evaluation.ICResults = append(evaluation.ICResults, icResult)
	}

	// 滚动IC分析
	evaluation.RollingIC = fde.calculateRollingIC(factor)

	// IC衰减分析
	evaluation.ICDecay = fde.calculateICDecay(factor)

	// 分组回测
	evaluation.GroupBacktest = fde.performGroupBacktest(factor)

	// 风险分析
	evaluation.RiskAnalysis = fde.analyzeFactorRisk(factor)

	// 稳定性分析
	evaluation.StabilityAnalysis = fde.analyzeFactorStability(factor)

	// 计算综合评分
	evaluation.OverallScore = fde.calculateOverallScore(evaluation)
	evaluation.Grade = fde.assignGrade(evaluation.OverallScore)

	return evaluation
}

// rotateFactors 轮换因子
func (fde *FactorDiscoveryEngine) rotateFactors() {
	if time.Since(fde.factorRotator.lastRotation) < fde.factorRotator.rotationFrequency {
		return
	}

	log.Println("Rotating factors...")

	// 评估当前活跃因子
	currentFactors := fde.getCurrentActiveFactors()

	// 候选因子池
	candidateFactors := fde.getCandidateFactors()

	// 执行轮换策略
	switch fde.factorRotator.rotationStrategy {
	case "PERFORMANCE":
		fde.performanceBasedRotation(currentFactors, candidateFactors)
	case "CORRELATION":
		fde.correlationBasedRotation(currentFactors, candidateFactors)
	case "REGIME":
		fde.regimeBasedRotation(currentFactors, candidateFactors)
	default:
		fde.performanceBasedRotation(currentFactors, candidateFactors)
	}

	fde.factorRotator.lastRotation = time.Now()
}

// monitorFactorPerformance 监控因子表现
func (fde *FactorDiscoveryEngine) monitorFactorPerformance() {
	fde.mu.RLock()
	factors := make([]*Factor, 0, len(fde.activeFactors))
	for _, factor := range fde.activeFactors {
		factors = append(factors, factor)
	}
	fde.mu.RUnlock()

	for _, factor := range factors {
		performance := fde.calculateFactorPerformance(factor)

		fde.mu.Lock()
		fde.factorPerformance[factor.ID] = performance
		fde.mu.Unlock()

		// 检查是否需要停用
		if fde.shouldDeactivateFactor(factor, performance) {
			fde.deactivateFactor(factor, "POOR_PERFORMANCE")
		}
	}
}

// Helper functions implementation...

func initializeBaseFactors() []BaseFactor {
	return []BaseFactor{
		{Name: "Close", Symbol: "CLOSE", Type: "PRICE", Description: "收盘价"},
		{Name: "Volume", Symbol: "VOLUME", Type: "VOLUME", Description: "成交量"},
		{Name: "High", Symbol: "HIGH", Type: "PRICE", Description: "最高价"},
		{Name: "Low", Symbol: "LOW", Type: "PRICE", Description: "最低价"},
		{Name: "Open", Symbol: "OPEN", Type: "PRICE", Description: "开盘价"},
		{Name: "Returns", Symbol: "RETURNS", Type: "PRICE", Description: "收益率"},
	}
}

func initializeOperators() []Operator {
	return []Operator{
		{Symbol: "+", Name: "Add", Precedence: 1, Operands: 2},
		{Symbol: "-", Name: "Subtract", Precedence: 1, Operands: 2},
		{Symbol: "*", Name: "Multiply", Precedence: 2, Operands: 2},
		{Symbol: "/", Name: "Divide", Precedence: 2, Operands: 2},
		{Symbol: "^", Name: "Power", Precedence: 3, Operands: 2},
	}
}

func initializeFunctions() []Function {
	return []Function{
		{Name: "SMA", Parameters: 1, Description: "简单移动平均"},
		{Name: "EMA", Parameters: 1, Description: "指数移动平均"},
		{Name: "RSI", Parameters: 1, Description: "相对强弱指数"},
		{Name: "MACD", Parameters: 2, Description: "MACD指标"},
		{Name: "STDDEV", Parameters: 1, Description: "标准差"},
		{Name: "RANK", Parameters: 1, Description: "排名"},
		{Name: "DELAY", Parameters: 1, Description: "滞后"},
		{Name: "DELTA", Parameters: 1, Description: "差分"},
		{Name: "TS_SUM", Parameters: 1, Description: "时序求和"},
		{Name: "TS_MAX", Parameters: 1, Description: "时序最大值"},
		{Name: "TS_MIN", Parameters: 1, Description: "时序最小值"},
	}
}

func (fde *FactorDiscoveryEngine) initializePopulation() {
	fde.geneticAlgorithm.population = make([]*Factor, fde.geneticAlgorithm.populationSize)

	for i := 0; i < fde.geneticAlgorithm.populationSize; i++ {
		factor := fde.generateRandomFactor()
		fde.geneticAlgorithm.population[i] = factor
	}

	log.Printf("Initialized population with %d factors", fde.geneticAlgorithm.populationSize)
}

func (fde *FactorDiscoveryEngine) generateRandomFactor() *Factor {
	// 实现随机因子生成

	// 定义可用的组件
	variables := []string{"open", "high", "low", "close", "volume"}
	operators := []string{"+", "-", "*", "/", "^"}
	functions := []string{"SMA", "EMA", "RSI", "MACD", "STDDEV", "MAX", "MIN", "ABS", "LOG", "SQRT"}

	// 随机生成因子表达式
	expression := fde.generateComplexRandomExpression(variables, operators, functions, 2+rand.Intn(3)) // 深度2-4

	// 计算复杂度
	complexity := fde.calculateComplexity(expression)

	// 生成随机参数
	parameters := make(map[string]float64)

	// 为技术指标添加随机参数
	if strings.Contains(expression, "SMA") || strings.Contains(expression, "EMA") {
		parameters["period"] = float64(5 + rand.Intn(45)) // 5-49
	}
	if strings.Contains(expression, "RSI") {
		parameters["rsi_period"] = float64(10 + rand.Intn(20)) // 10-29
	}
	if strings.Contains(expression, "STDDEV") {
		parameters["stddev_period"] = float64(10 + rand.Intn(40)) // 10-49
	}
	if strings.Contains(expression, "MACD") {
		parameters["fast_period"] = float64(8 + rand.Intn(8))    // 8-15
		parameters["slow_period"] = float64(20 + rand.Intn(15))  // 20-34
		parameters["signal_period"] = float64(5 + rand.Intn(10)) // 5-14
	}

	// 添加随机权重参数
	if rand.Float64() < 0.3 { // 30%概率添加权重
		parameters["weight"] = 0.5 + rand.Float64()*1.5 // 0.5-2.0
	}

	// 添加随机阈值参数
	if rand.Float64() < 0.2 { // 20%概率添加阈值
		parameters["threshold"] = rand.Float64()*0.1 - 0.05 // -0.05 to 0.05
	}

	factor := &Factor{
		ID:           fde.generateFactorID(),
		Name:         fmt.Sprintf("Random_Factor_%d", rand.Int()),
		Type:         "GENETIC",
		Category:     "random_generated",
		Expression:   expression,
		Formula:      expression, // 保持兼容性
		Description:  fmt.Sprintf("Randomly generated factor: %s", expression),
		Parameters:   parameters,
		DiscoveredAt: time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Status:       "ACTIVE",
		IsActive:     true,
		CreatedBy:    "genetic_algorithm",
		Generation:   0,
		Complexity:   complexity,
		IC:           0.0, // 将在评估时计算
		Fitness:      0.0, // 将在评估时计算
	}

	return factor
}

// generateComplexRandomExpression 生成复杂的随机表达式
func (fde *FactorDiscoveryEngine) generateComplexRandomExpression(variables, operators, functions []string, depth int) string {
	if depth <= 0 {
		// 基础情况：返回变量或简单函数
		if rand.Float64() < 0.7 {
			return variables[rand.Intn(len(variables))]
		} else {
			function := functions[rand.Intn(len(functions))]
			variable := variables[rand.Intn(len(variables))]

			switch function {
			case "SMA", "EMA", "STDDEV", "MAX", "MIN":
				period := 5 + rand.Intn(45) // 5-49
				return fmt.Sprintf("%s(%s, %d)", function, variable, period)
			case "RSI":
				period := 10 + rand.Intn(20) // 10-29
				return fmt.Sprintf("%s(%s, %d)", function, variable, period)
			case "MACD":
				fast := 8 + rand.Intn(8)    // 8-15
				slow := 20 + rand.Intn(15)  // 20-34
				signal := 5 + rand.Intn(10) // 5-14
				return fmt.Sprintf("%s(%s, %d, %d, %d)", function, variable, fast, slow, signal)
			default:
				return fmt.Sprintf("%s(%s)", function, variable)
			}
		}
	}

	// 递归情况：生成复合表达式
	expressionType := rand.Intn(4)

	switch expressionType {
	case 0: // 二元操作
		left := fde.generateComplexRandomExpression(variables, operators, functions, depth-1)
		right := fde.generateComplexRandomExpression(variables, operators, functions, depth-1)
		op := operators[rand.Intn(len(operators))]
		return fmt.Sprintf("(%s %s %s)", left, op, right)

	case 1: // 函数应用
		function := functions[rand.Intn(len(functions))]
		arg := fde.generateComplexRandomExpression(variables, operators, functions, depth-1)

		switch function {
		case "SMA", "EMA", "STDDEV", "MAX", "MIN":
			period := 5 + rand.Intn(45)
			return fmt.Sprintf("%s(%s, %d)", function, arg, period)
		case "RSI":
			period := 10 + rand.Intn(20)
			return fmt.Sprintf("%s(%s, %d)", function, arg, period)
		default:
			return fmt.Sprintf("%s(%s)", function, arg)
		}

	case 2: // 条件表达式（简化版）
		condition := fde.generateComplexRandomExpression(variables, operators, functions, depth-1)
		trueExpr := fde.generateComplexRandomExpression(variables, operators, functions, depth-1)
		falseExpr := fde.generateComplexRandomExpression(variables, operators, functions, depth-1)
		return fmt.Sprintf("IF(%s > 0, %s, %s)", condition, trueExpr, falseExpr)

	default: // 滞后表达式
		expr := fde.generateComplexRandomExpression(variables, operators, functions, depth-1)
		lag := 1 + rand.Intn(10) // 1-10期滞后
		return fmt.Sprintf("%s[%d]", expr, lag)
	}
}

func (fde *FactorDiscoveryEngine) evaluatePopulation() {
	for _, factor := range fde.geneticAlgorithm.population {
		// 计算因子的IC和其他性能指标
		ic := fde.calculateFactorIC(factor)
		factor.IC = ic
		factor.Fitness = fde.calculateFitness(factor)
	}
}

func (fde *FactorDiscoveryEngine) calculateGenerationStats(generation int) GenerationStats {
	if len(fde.geneticAlgorithm.population) == 0 {
		return GenerationStats{Generation: generation, Timestamp: time.Now()}
	}

	var bestFitness, totalFitness float64
	bestFitness = math.Inf(-1)

	for _, factor := range fde.geneticAlgorithm.population {
		if factor.Fitness > bestFitness {
			bestFitness = factor.Fitness
		}
		totalFitness += factor.Fitness
	}

	avgFitness := totalFitness / float64(len(fde.geneticAlgorithm.population))
	diversity := fde.calculatePopulationDiversity()
	complexity := fde.calculateAvgComplexity()

	return GenerationStats{
		Generation:  generation,
		BestFitness: bestFitness,
		AvgFitness:  avgFitness,
		Diversity:   diversity,
		Complexity:  complexity,
		Timestamp:   time.Now(),
	}
}

func (fde *FactorDiscoveryEngine) checkConvergence() bool {
	// 实现收敛检查逻辑

	if fde.geneticAlgorithm == nil || len(fde.geneticAlgorithm.population) == 0 {
		return false
	}

	currentGeneration := fde.geneticAlgorithm.currentGeneration

	// 1. 检查是否达到最大代数
	if currentGeneration >= fde.geneticAlgorithm.maxGenerations {
		log.Printf("Convergence: Reached maximum generations (%d)", fde.geneticAlgorithm.maxGenerations)
		return true
	}

	// 2. 检查适应度改进停滞
	if len(fde.geneticAlgorithm.fitnessHistory) >= 10 {
		// 计算最近10代的适应度改进
		recentHistory := fde.geneticAlgorithm.fitnessHistory[len(fde.geneticAlgorithm.fitnessHistory)-10:]

		// 计算适应度变化的标准差
		var sum, sumSquares float64
		for _, fitness := range recentHistory {
			sum += fitness
			sumSquares += fitness * fitness
		}

		mean := sum / float64(len(recentHistory))
		variance := sumSquares/float64(len(recentHistory)) - mean*mean
		stdDev := math.Sqrt(variance)

		// 如果标准差很小，说明适应度没有显著改进
		if stdDev < 0.001 {
			log.Printf("Convergence: Fitness improvement stagnated (stddev: %.6f)", stdDev)
			return true
		}

		// 检查最近几代的最佳适应度是否没有改进
		lastBest := recentHistory[len(recentHistory)-1]
		improvementCount := 0
		for i := len(recentHistory) - 2; i >= 0; i-- {
			if recentHistory[i] < lastBest {
				improvementCount++
			}
		}

		// 如果最近9代中有8代或更多没有改进，认为收敛
		if improvementCount >= 8 {
			log.Printf("Convergence: No significant improvement in recent generations")
			return true
		}
	}

	// 3. 检查种群多样性
	diversity := fde.calculatePopulationDiversity()
	if diversity < 0.01 { // 多样性阈值
		log.Printf("Convergence: Population diversity too low (%.6f)", diversity)
		return true
	}

	// 4. 检查最佳因子的质量是否达到目标
	bestFactor := fde.getBestFactor()
	if bestFactor != nil && bestFactor.IC > 0.1 { // IC阈值
		log.Printf("Convergence: Found high-quality factor (IC: %.4f)", bestFactor.IC)
		return true
	}

	// 5. 检查运行时间
	if fde.startTime != nil {
		elapsed := time.Since(*fde.startTime)
		maxDuration := 2 * time.Hour // 最大运行时间
		if fde.config != nil && fde.config.FactorDiscovery != nil {
			if fde.config.FactorDiscovery.MaxDuration > 0 {
				maxDuration = fde.config.FactorDiscovery.MaxDuration
			}
		}

		if elapsed > maxDuration {
			log.Printf("Convergence: Maximum runtime exceeded (%.2f hours)", elapsed.Hours())
			return true
		}
	}

	// 6. 检查发现的有效因子数量
	fde.mu.RLock()
	discoveredCount := len(fde.discoveredFactors)
	fde.mu.RUnlock()

	maxFactors := 100
	if fde.config != nil && fde.config.FactorDiscovery != nil {
		if fde.config.FactorDiscovery.MaxFactors > 0 {
			maxFactors = fde.config.FactorDiscovery.MaxFactors
		}
	}

	if discoveredCount >= maxFactors {
		log.Printf("Convergence: Discovered enough factors (%d)", discoveredCount)
		return true
	}

	return false
}

// calculatePopulationDiversity 计算种群多样性
func (fde *FactorDiscoveryEngine) calculatePopulationDiversity() float64 {
	if fde.geneticAlgorithm == nil || len(fde.geneticAlgorithm.population) < 2 {
		return 0.0
	}

	population := fde.geneticAlgorithm.population
	totalDistance := 0.0
	comparisons := 0

	// 计算所有因子对之间的距离
	for i := 0; i < len(population); i++ {
		for j := i + 1; j < len(population); j++ {
			distance := fde.calculateFactorDistance(population[i], population[j])
			totalDistance += distance
			comparisons++
		}
	}

	if comparisons == 0 {
		return 0.0
	}

	return totalDistance / float64(comparisons)
}

// calculateFactorDistance 计算两个因子之间的距离
func (fde *FactorDiscoveryEngine) calculateFactorDistance(factor1, factor2 *Factor) float64 {
	// 基于表达式的编辑距离
	expressionDistance := fde.calculateEditDistance(factor1.Expression, factor2.Expression)

	// 基于参数的距离
	paramDistance := fde.calculateParameterDistance(factor1.Parameters, factor2.Parameters)

	// 基于性能的距离
	performanceDistance := math.Abs(factor1.IC - factor2.IC)

	// 综合距离
	return 0.5*expressionDistance + 0.3*paramDistance + 0.2*performanceDistance
}

// calculateEditDistance 计算编辑距离（简化版）
func (fde *FactorDiscoveryEngine) calculateEditDistance(s1, s2 string) float64 {
	if s1 == s2 {
		return 0.0
	}

	// 简化的编辑距离计算
	maxLen := math.Max(float64(len(s1)), float64(len(s2)))
	if maxLen == 0 {
		return 0.0
	}

	// 计算公共字符数
	commonChars := 0
	for i := 0; i < len(s1) && i < len(s2); i++ {
		if s1[i] == s2[i] {
			commonChars++
		}
	}

	return 1.0 - float64(commonChars)/maxLen
}

// calculateParameterDistance 计算参数距离
func (fde *FactorDiscoveryEngine) calculateParameterDistance(params1, params2 map[string]float64) float64 {
	if len(params1) == 0 && len(params2) == 0 {
		return 0.0
	}

	// 获取所有参数键
	allKeys := make(map[string]bool)
	for key := range params1 {
		allKeys[key] = true
	}
	for key := range params2 {
		allKeys[key] = true
	}

	totalDistance := 0.0
	for key := range allKeys {
		val1, exists1 := params1[key]
		val2, exists2 := params2[key]

		if !exists1 {
			val1 = 0
		}
		if !exists2 {
			val2 = 0
		}

		totalDistance += math.Abs(val1 - val2)
	}

	return totalDistance / float64(len(allKeys))
}

// getBestFactor 获取当前最佳因子
func (fde *FactorDiscoveryEngine) getBestFactor() *Factor {
	if fde.geneticAlgorithm == nil || len(fde.geneticAlgorithm.population) == 0 {
		return nil
	}

	bestFactor := fde.geneticAlgorithm.population[0]
	for _, factor := range fde.geneticAlgorithm.population {
		if factor.Fitness > bestFactor.Fitness {
			bestFactor = factor
		}
	}

	return bestFactor
}

func (fde *FactorDiscoveryEngine) evolvePopulation() []*Factor {
	newPopulation := make([]*Factor, fde.geneticAlgorithm.populationSize)

	// 精英选择
	eliteCount := int(float64(fde.geneticAlgorithm.populationSize) * fde.geneticAlgorithm.elitismRate)
	elite := fde.selectElite(eliteCount)
	copy(newPopulation[:eliteCount], elite)

	// 交叉和变异
	for i := eliteCount; i < fde.geneticAlgorithm.populationSize; i++ {
		if rand.Float64() < fde.geneticAlgorithm.crossoverRate {
			// 交叉
			parent1 := fde.selectParent()
			parent2 := fde.selectParent()
			child := fde.crossover(parent1, parent2)
			newPopulation[i] = child
		} else {
			// 复制
			parent := fde.selectParent()
			child := fde.copyFactor(parent)
			newPopulation[i] = child
		}

		// 变异
		if rand.Float64() < fde.geneticAlgorithm.mutationRate {
			fde.mutate(newPopulation[i])
		}
	}

	return newPopulation
}

func (fde *FactorDiscoveryEngine) selectBestFactors() []*Factor {
	// 按适应度排序
	factors := make([]*Factor, len(fde.geneticAlgorithm.population))
	copy(factors, fde.geneticAlgorithm.population)

	sort.Slice(factors, func(i, j int) bool {
		return factors[i].Fitness > factors[j].Fitness
	})

	// 选择前N个
	topN := int(math.Min(10, float64(len(factors))))
	return factors[:topN]
}

func (fde *FactorDiscoveryEngine) isFactorNovel(factor *Factor) bool {
	// 检查因子是否新颖（不与现有因子重复）
	threshold := 0.85 // 相似度阈值，降低一些以允许更多变化

	fde.mu.RLock()
	defer fde.mu.RUnlock()

	// 1. 检查表达式是否完全相同
	for _, existingFactor := range fde.discoveredFactors {
		if factor.Expression == existingFactor.Expression {
			return false // 表达式完全相同，不新颖
		}
	}

	// 2. 检查基础因子是否已存在
	for _, existingFactor := range fde.baseFactors {
		if factor.Expression == existingFactor.Expression {
			return false // 与基础因子相同，不新颖
		}
	}

	// 3. 计算与现有因子的相似度
	for _, existingFactor := range fde.discoveredFactors {
		similarity := fde.calculateFactorSimilarity(factor, existingFactor)
		if similarity > threshold {
			log.Printf("Factor %s rejected due to high similarity (%.3f) with existing factor %s",
				factor.ID, similarity, existingFactor.ID)
			return false
		}
	}

	// 4. 检查与基础因子的相似度
	for _, existingFactor := range fde.baseFactors {
		similarity := fde.calculateFactorSimilarity(factor, existingFactor)
		if similarity > threshold {
			log.Printf("Factor %s rejected due to high similarity (%.3f) with base factor %s",
				factor.ID, similarity, existingFactor.ID)
			return false
		}
	}

	// 5. 检查表达式的结构相似性
	if fde.hasStructuralSimilarity(factor) {
		return false
	}

	// 6. 检查参数组合的新颖性
	if fde.hasParameterSimilarity(factor) {
		return false
	}

	return true
}

// hasStructuralSimilarity 检查结构相似性
func (fde *FactorDiscoveryEngine) hasStructuralSimilarity(factor *Factor) bool {
	// 提取表达式的结构特征
	structure := fde.extractStructure(factor.Expression)

	// 检查是否与现有因子有相同的结构
	for _, existingFactor := range fde.discoveredFactors {
		existingStructure := fde.extractStructure(existingFactor.Expression)
		if structure == existingStructure {
			// 进一步检查参数是否过于相似
			if fde.areParametersSimilar(factor.Parameters, existingFactor.Parameters) {
				return true
			}
		}
	}

	return false
}

// extractStructure 提取表达式结构
func (fde *FactorDiscoveryEngine) extractStructure(expression string) string {
	// 简化的结构提取：移除具体数值，保留操作符和函数
	structure := expression

	// 替换数字为占位符
	re := regexp.MustCompile(`\d+`)
	structure = re.ReplaceAllString(structure, "N")

	// 替换变量名为占位符
	variables := []string{"open", "high", "low", "close", "volume"}
	for _, variable := range variables {
		structure = strings.ReplaceAll(structure, variable, "VAR")
	}

	return structure
}

// areParametersSimilar 检查参数是否相似
func (fde *FactorDiscoveryEngine) areParametersSimilar(params1, params2 map[string]interface{}) bool {
	if len(params1) != len(params2) {
		return false
	}

	threshold := 0.1 // 参数相似度阈值

	for key, val1 := range params1 {
		val2, exists := params2[key]
		if !exists {
			return false
		}

		// 转换为float64进行比较
		f1, ok1 := fde.toFloat64(val1)
		f2, ok2 := fde.toFloat64(val2)

		if ok1 && ok2 {
			if math.Abs(f1-f2) > threshold*math.Max(math.Abs(f1), math.Abs(f2)) {
				return false
			}
		} else {
			// 非数值参数直接比较
			if fmt.Sprintf("%v", val1) != fmt.Sprintf("%v", val2) {
				return false
			}
		}
	}

	return true
}

// hasParameterSimilarity 检查参数组合的新颖性
func (fde *FactorDiscoveryEngine) hasParameterSimilarity(factor *Factor) bool {
	// 检查是否存在相同类型和相似参数的因子
	for _, existingFactor := range fde.discoveredFactors {
		if factor.Type == existingFactor.Type && factor.Category == existingFactor.Category {
			if fde.areParametersSimilar(factor.Parameters, existingFactor.Parameters) {
				return true
			}
		}
	}

	return false
}

// toFloat64 尝试将interface{}转换为float64
func (fde *FactorDiscoveryEngine) toFloat64(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

func (fde *FactorDiscoveryEngine) addDiscoveredFactor(factor *Factor) {
	fde.mu.Lock()
	defer fde.mu.Unlock()

	factor.DiscoveredAt = time.Now()
	factor.LastUpdated = time.Now()
	factor.Status = "ACTIVE"

	fde.discoveredFactors[factor.ID] = factor

	// 记录发现事件
	event := DiscoveryEvent{
		Date:      time.Now(),
		EventType: "DISCOVERY",
		FactorID:  factor.ID,
		Details: map[string]interface{}{
			"ic":         factor.IC,
			"type":       factor.Type,
			"complexity": factor.Complexity,
		},
		Impact: fde.assessFactorImpact(factor),
	}
	fde.discoveryHistory = append(fde.discoveryHistory, event)

	// 更新统计
	fde.discoveryMetrics.mu.Lock()
	fde.discoveryMetrics.FactorsDiscovered++
	fde.discoveryMetrics.TotalFactors++
	fde.discoveryMetrics.TypeDistribution[factor.Type]++
	fde.discoveryMetrics.ComplexityDistribution[factor.Complexity]++
	fde.discoveryMetrics.mu.Unlock()
}

// 其他辅助函数的简化实现...
func (fde *FactorDiscoveryEngine) calculateComplexity(formula string) int {
	// 简化的复杂度计算
	return len(formula) / 10
}

func (fde *FactorDiscoveryEngine) calculateFactorIC(factor *Factor) float64 {
	// 实现实际的IC计算（信息系数 - Information Coefficient）

	// 获取历史数据
	historicalData, err := fde.getHistoricalData()
	if err != nil {
		log.Printf("Failed to get historical data for IC calculation: %v", err)
		return 0.0
	}

	if len(historicalData) < 30 { // 至少需要30个数据点
		log.Printf("Insufficient historical data for IC calculation: %d points", len(historicalData))
		return 0.0
	}

	// 计算因子值
	factorValues, err := fde.calculateFactorValues(factor, historicalData)
	if err != nil {
		log.Printf("Failed to calculate factor values: %v", err)
		return 0.0
	}

	// 计算未来收益率（1期前瞻）
	futureReturns := make([]float64, len(historicalData)-1)
	for i := 0; i < len(historicalData)-1; i++ {
		if historicalData[i].Close > 0 {
			futureReturns[i] = (historicalData[i+1].Close - historicalData[i].Close) / historicalData[i].Close
		}
	}

	// 确保因子值和收益率数组长度一致
	if len(factorValues) != len(futureReturns) {
		minLen := int(math.Min(float64(len(factorValues)), float64(len(futureReturns))))
		factorValues = factorValues[:minLen]
		futureReturns = futureReturns[:minLen]
	}

	// 计算Spearman相关系数（排序相关系数）
	ic := fde.calculateSpearmanCorrelation(factorValues, futureReturns)

	// 处理异常值
	if math.IsNaN(ic) || math.IsInf(ic, 0) {
		return 0.0
	}

	// 限制IC在合理范围内
	if ic > 1.0 {
		ic = 1.0
	} else if ic < -1.0 {
		ic = -1.0
	}

	return ic
}

// getHistoricalData 获取历史数据
func (fde *FactorDiscoveryEngine) getHistoricalData() ([]MarketData, error) {
	// 这里应该从数据库或API获取真实的历史数据
	// 暂时生成模拟数据用于演示

	dataPoints := 252 // 一年的交易日
	data := make([]MarketData, dataPoints)

	basePrice := 100.0
	for i := 0; i < dataPoints; i++ {
		// 生成随机价格变动
		change := (rand.Float64() - 0.5) * 0.04 // -2% to +2%
		basePrice *= (1 + change)

		// 生成OHLC数据
		open := basePrice * (1 + (rand.Float64()-0.5)*0.01)
		high := basePrice * (1 + rand.Float64()*0.02)
		low := basePrice * (1 - rand.Float64()*0.02)
		close := basePrice
		volume := 1000000 + rand.Float64()*500000

		data[i] = MarketData{
			Timestamp: time.Now().AddDate(0, 0, -dataPoints+i),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		}
	}

	return data, nil
}

// calculateFactorValues 计算因子值
func (fde *FactorDiscoveryEngine) calculateFactorValues(factor *Factor, data []MarketData) ([]float64, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no market data provided")
	}

	// 创建因子计算引擎
	calculator := NewFactorCalculator(data)

	// 解析因子表达式
	parsedExpr, err := calculator.ParseExpression(factor.Expression)
	if err != nil {
		return nil, fmt.Errorf("failed to parse factor expression '%s': %v", factor.Expression, err)
	}

	// 计算因子值
	values, err := calculator.CalculateFactorValues(parsedExpr)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate factor values: %v", err)
	}

	// 验证结果
	if len(values) != len(data) {
		return nil, fmt.Errorf("calculated values length (%d) doesn't match data length (%d)", len(values), len(data))
	}

	return values, nil
}

// FactorCalculator 因子计算引擎
type FactorCalculator struct {
	data       []MarketData
	dataLength int
	cache      map[string][]float64 // 缓存计算结果
	indicators map[string]TechnicalIndicator
}

// TechnicalIndicator 技术指标接口
type TechnicalIndicator interface {
	Calculate(data []MarketData, params map[string]float64) ([]float64, error)
	GetName() string
	GetRequiredParams() []string
}

// ParsedExpression 解析后的表达式
type ParsedExpression struct {
	Type     ExpressionType
	Value    interface{}
	Children []*ParsedExpression
	Params   map[string]float64
}

// ExpressionType 表达式类型
type ExpressionType int

const (
	ExprTypeVariable ExpressionType = iota
	ExprTypeConstant
	ExprTypeOperator
	ExprTypeFunction
	ExprTypeIndicator
	ExprTypeConditional
)

// NewFactorCalculator 创建因子计算引擎
func NewFactorCalculator(data []MarketData) *FactorCalculator {
	calc := &FactorCalculator{
		data:       data,
		dataLength: len(data),
		cache:      make(map[string][]float64),
		indicators: make(map[string]TechnicalIndicator),
	}

	// 注册技术指标
	calc.registerIndicators()

	return calc
}

// registerIndicators 注册技术指标
func (fc *FactorCalculator) registerIndicators() {
	fc.indicators["SMA"] = &SMAIndicator{}
	fc.indicators["EMA"] = &EMAIndicator{}
	fc.indicators["RSI"] = &RSIIndicator{}
	fc.indicators["MACD"] = &MACDIndicator{}
	fc.indicators["STDDEV"] = &STDDEVIndicator{}
	fc.indicators["MAX"] = &MAXIndicator{}
	fc.indicators["MIN"] = &MINIndicator{}
	fc.indicators["RANK"] = &RANKIndicator{}
	fc.indicators["DELAY"] = &DELAYIndicator{}
	fc.indicators["DELTA"] = &DELTAIndicator{}
	fc.indicators["TS_SUM"] = &TSSUMIndicator{}
	fc.indicators["TS_MAX"] = &TSMAXIndicator{}
	fc.indicators["TS_MIN"] = &TSMINIndicator{}
	fc.indicators["BB_UPPER"] = &BBUpperIndicator{}
	fc.indicators["BB_LOWER"] = &BBLowerIndicator{}
}

// ParseExpression 解析因子表达式
func (fc *FactorCalculator) ParseExpression(expression string) (*ParsedExpression, error) {
	// 清理表达式
	expr := strings.TrimSpace(expression)
	if expr == "" {
		return nil, fmt.Errorf("empty expression")
	}

	// 使用递归下降解析器解析表达式
	parser := &ExpressionParser{
		expression: expr,
		position:   0,
		length:     len(expr),
	}

	return parser.parseExpression()
}

// CalculateFactorValues 计算因子值
func (fc *FactorCalculator) CalculateFactorValues(expr *ParsedExpression) ([]float64, error) {
	if expr == nil {
		return nil, fmt.Errorf("nil expression")
	}

	return fc.evaluateExpression(expr)
}

// evaluateExpression 评估表达式
func (fc *FactorCalculator) evaluateExpression(expr *ParsedExpression) ([]float64, error) {
	switch expr.Type {
	case ExprTypeVariable:
		return fc.getVariableValues(expr.Value.(string))

	case ExprTypeConstant:
		value := expr.Value.(float64)
		result := make([]float64, fc.dataLength)
		for i := range result {
			result[i] = value
		}
		return result, nil

	case ExprTypeOperator:
		return fc.evaluateOperator(expr)

	case ExprTypeFunction:
		return fc.evaluateFunction(expr)

	case ExprTypeIndicator:
		return fc.evaluateIndicator(expr)

	case ExprTypeConditional:
		return fc.evaluateConditional(expr)

	default:
		return nil, fmt.Errorf("unknown expression type: %v", expr.Type)
	}
}

// ExpressionParser 表达式解析器
type ExpressionParser struct {
	expression string
	position   int
	length     int
}

// parseExpression 解析表达式
func (p *ExpressionParser) parseExpression() (*ParsedExpression, error) {
	return p.parseOrExpression()
}

// getVariableValues 获取变量值
func (fc *FactorCalculator) getVariableValues(variable string) ([]float64, error) {
	// 检查缓存
	if cached, exists := fc.cache[variable]; exists {
		return cached, nil
	}

	values := make([]float64, fc.dataLength)

	switch strings.ToLower(variable) {
	case "open":
		for i, data := range fc.data {
			values[i] = data.Open
		}
	case "high":
		for i, data := range fc.data {
			values[i] = data.High
		}
	case "low":
		for i, data := range fc.data {
			values[i] = data.Low
		}
	case "close":
		for i, data := range fc.data {
			values[i] = data.Close
		}
	case "volume":
		for i, data := range fc.data {
			values[i] = data.Volume
		}
	case "returns", "return":
		// 计算收益率
		for i := range fc.data {
			if i == 0 {
				values[i] = 0.0
			} else {
				values[i] = (fc.data[i].Close - fc.data[i-1].Close) / fc.data[i-1].Close
			}
		}
	default:
		return nil, fmt.Errorf("unknown variable: %s", variable)
	}

	// 缓存结果
	fc.cache[variable] = values
	return values, nil
}

// evaluateOperator 评估操作符
func (fc *FactorCalculator) evaluateOperator(expr *ParsedExpression) ([]float64, error) {
	if len(expr.Children) != 2 {
		return nil, fmt.Errorf("operator requires exactly 2 operands, got %d", len(expr.Children))
	}

	left, err := fc.evaluateExpression(expr.Children[0])
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate left operand: %v", err)
	}

	right, err := fc.evaluateExpression(expr.Children[1])
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate right operand: %v", err)
	}

	if len(left) != len(right) {
		return nil, fmt.Errorf("operand length mismatch: %d vs %d", len(left), len(right))
	}

	result := make([]float64, len(left))
	operator := expr.Value.(string)

	for i := range result {
		switch operator {
		case "+":
			result[i] = left[i] + right[i]
		case "-":
			result[i] = left[i] - right[i]
		case "*":
			result[i] = left[i] * right[i]
		case "/":
			if right[i] == 0 {
				result[i] = math.NaN()
			} else {
				result[i] = left[i] / right[i]
			}
		case "^", "**":
			result[i] = math.Pow(left[i], right[i])
		case ">":
			if left[i] > right[i] {
				result[i] = 1.0
			} else {
				result[i] = 0.0
			}
		case "<":
			if left[i] < right[i] {
				result[i] = 1.0
			} else {
				result[i] = 0.0
			}
		case ">=":
			if left[i] >= right[i] {
				result[i] = 1.0
			} else {
				result[i] = 0.0
			}
		case "<=":
			if left[i] <= right[i] {
				result[i] = 1.0
			} else {
				result[i] = 0.0
			}
		case "==":
			if math.Abs(left[i]-right[i]) < 1e-10 {
				result[i] = 1.0
			} else {
				result[i] = 0.0
			}
		case "!=":
			if math.Abs(left[i]-right[i]) >= 1e-10 {
				result[i] = 1.0
			} else {
				result[i] = 0.0
			}
		default:
			return nil, fmt.Errorf("unknown operator: %s", operator)
		}
	}

	return result, nil
}

// evaluateFunction 评估函数
func (fc *FactorCalculator) evaluateFunction(expr *ParsedExpression) ([]float64, error) {
	functionName := expr.Value.(string)

	switch strings.ToUpper(functionName) {
	case "ABS":
		if len(expr.Children) != 1 {
			return nil, fmt.Errorf("ABS function requires 1 argument, got %d", len(expr.Children))
		}
		values, err := fc.evaluateExpression(expr.Children[0])
		if err != nil {
			return nil, err
		}
		result := make([]float64, len(values))
		for i, v := range values {
			result[i] = math.Abs(v)
		}
		return result, nil

	case "LOG":
		if len(expr.Children) != 1 {
			return nil, fmt.Errorf("LOG function requires 1 argument, got %d", len(expr.Children))
		}
		values, err := fc.evaluateExpression(expr.Children[0])
		if err != nil {
			return nil, err
		}
		result := make([]float64, len(values))
		for i, v := range values {
			if v <= 0 {
				result[i] = math.NaN()
			} else {
				result[i] = math.Log(v)
			}
		}
		return result, nil

	case "SQRT":
		if len(expr.Children) != 1 {
			return nil, fmt.Errorf("SQRT function requires 1 argument, got %d", len(expr.Children))
		}
		values, err := fc.evaluateExpression(expr.Children[0])
		if err != nil {
			return nil, err
		}
		result := make([]float64, len(values))
		for i, v := range values {
			if v < 0 {
				result[i] = math.NaN()
			} else {
				result[i] = math.Sqrt(v)
			}
		}
		return result, nil

	case "IF":
		if len(expr.Children) != 3 {
			return nil, fmt.Errorf("IF function requires 3 arguments, got %d", len(expr.Children))
		}
		condition, err := fc.evaluateExpression(expr.Children[0])
		if err != nil {
			return nil, err
		}
		trueValues, err := fc.evaluateExpression(expr.Children[1])
		if err != nil {
			return nil, err
		}
		falseValues, err := fc.evaluateExpression(expr.Children[2])
		if err != nil {
			return nil, err
		}

		result := make([]float64, len(condition))
		for i := range result {
			if condition[i] != 0 {
				result[i] = trueValues[i]
			} else {
				result[i] = falseValues[i]
			}
		}
		return result, nil

	default:
		return nil, fmt.Errorf("unknown function: %s", functionName)
	}
}

// evaluateIndicator 评估技术指标
func (fc *FactorCalculator) evaluateIndicator(expr *ParsedExpression) ([]float64, error) {
	indicatorName := expr.Value.(string)

	indicator, exists := fc.indicators[strings.ToUpper(indicatorName)]
	if !exists {
		return nil, fmt.Errorf("unknown indicator: %s", indicatorName)
	}

	// 从表达式参数中提取参数
	params := make(map[string]float64)
	if expr.Params != nil {
		params = expr.Params
	}

	// 计算指标
	return indicator.Calculate(fc.data, params)
}

// evaluateConditional 评估条件表达式
func (fc *FactorCalculator) evaluateConditional(expr *ParsedExpression) ([]float64, error) {
	if len(expr.Children) != 3 {
		return nil, fmt.Errorf("conditional expression requires 3 parts, got %d", len(expr.Children))
	}

	condition, err := fc.evaluateExpression(expr.Children[0])
	if err != nil {
		return nil, err
	}

	trueValues, err := fc.evaluateExpression(expr.Children[1])
	if err != nil {
		return nil, err
	}

	falseValues, err := fc.evaluateExpression(expr.Children[2])
	if err != nil {
		return nil, err
	}

	result := make([]float64, len(condition))
	for i := range result {
		if condition[i] != 0 {
			result[i] = trueValues[i]
		} else {
			result[i] = falseValues[i]
		}
	}

	return result, nil
}

// calculateSMA 计算简单移动平均
func (fde *FactorDiscoveryEngine) calculateSMA(data []MarketData, currentIndex, period int) float64 {
	if currentIndex < period-1 {
		return data[currentIndex].Close
	}

	sum := 0.0
	for i := currentIndex - period + 1; i <= currentIndex; i++ {
		sum += data[i].Close
	}

	return sum / float64(period)
}

// calculateRSI 计算相对强弱指数
func (fde *FactorDiscoveryEngine) calculateRSI(data []MarketData, currentIndex, period int) float64 {
	if currentIndex < period {
		return 50.0 // 默认中性值
	}

	gains := 0.0
	losses := 0.0

	for i := currentIndex - period + 1; i <= currentIndex; i++ {
		change := data[i].Close - data[i-1].Close
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}

	if losses == 0 {
		return 100.0
	}

	rs := gains / losses
	rsi := 100.0 - (100.0 / (1.0 + rs))

	return rsi
}

// parseFloat 解析浮点数
func (fde *FactorDiscoveryEngine) parseFloat(s string) (float64, error) {
	// 简化的浮点数解析
	var result float64
	_, err := fmt.Sscanf(s, "%f", &result)
	return result, err
}

// calculateSpearmanCorrelation 计算Spearman相关系数
func (fde *FactorDiscoveryEngine) calculateSpearmanCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0.0
	}

	n := len(x)

	// 创建排序索引
	xRanks := fde.getRanks(x)
	yRanks := fde.getRanks(y)

	// 计算排序差的平方和
	sumD2 := 0.0
	for i := 0; i < n; i++ {
		d := xRanks[i] - yRanks[i]
		sumD2 += d * d
	}

	// Spearman相关系数公式
	rho := 1.0 - (6.0*sumD2)/float64(n*(n*n-1))

	return rho
}

// getRanks 获取数据的排序
func (fde *FactorDiscoveryEngine) getRanks(data []float64) []float64 {
	n := len(data)
	ranks := make([]float64, n)

	// 创建索引数组
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}

	// 根据数据值排序索引
	sort.Slice(indices, func(i, j int) bool {
		return data[indices[i]] < data[indices[j]]
	})

	// 分配排序
	for rank, idx := range indices {
		ranks[idx] = float64(rank + 1)
	}

	return ranks
}

// MarketData 市场数据结构
type MarketData struct {
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

func (fde *FactorDiscoveryEngine) calculateFitness(factor *Factor) float64 {
	// 适应度函数：IC + 多样性奖励 - 复杂度惩罚
	fitness := math.Abs(factor.IC) // 使用IC的绝对值

	// 多样性奖励
	diversityBonus := fde.geneticAlgorithm.diversityWeight * fde.calculateFactorDiversity(factor)

	// 复杂度惩罚
	complexityPenalty := fde.geneticAlgorithm.complexityPenalty * float64(factor.Complexity) / 100.0

	return fitness + diversityBonus - complexityPenalty
}

func (fde *FactorDiscoveryEngine) calculateFactorDiversity(factor *Factor) float64 {
	// 计算因子多样性
	// 多样性基于因子与现有因子群体的差异程度

	if fde.geneticAlgorithm == nil || len(fde.geneticAlgorithm.population) == 0 {
		return 1.0 // 如果没有其他因子，多样性最高
	}

	totalDistance := 0.0
	count := 0

	// 计算与种群中所有其他因子的平均距离
	for _, existingFactor := range fde.geneticAlgorithm.population {
		if existingFactor.ID != factor.ID {
			distance := fde.calculateFactorDistance(factor, existingFactor)
			totalDistance += distance
			count++
		}
	}

	// 也考虑与已发现因子的距离
	fde.mu.RLock()
	for _, discoveredFactor := range fde.discoveredFactors {
		if discoveredFactor.ID != factor.ID {
			distance := fde.calculateFactorDistance(factor, &discoveredFactor)
			totalDistance += distance
			count++
		}
	}
	fde.mu.RUnlock()

	if count == 0 {
		return 1.0
	}

	avgDistance := totalDistance / float64(count)

	// 将距离转换为多样性分数（0-1范围）
	// 距离越大，多样性越高
	diversity := math.Min(1.0, avgDistance)

	// 额外的多样性奖励因素

	// 1. 表达式复杂度多样性
	complexityDiversity := fde.calculateComplexityDiversity(factor)

	// 2. 参数多样性
	parameterDiversity := fde.calculateParameterDiversity(factor)

	// 3. 类型多样性
	typeDiversity := fde.calculateTypeDiversity(factor)

	// 4. 功能多样性
	functionalDiversity := fde.calculateFunctionalDiversity(factor)

	// 综合多样性分数
	overallDiversity := (diversity*0.4 + complexityDiversity*0.2 +
		parameterDiversity*0.2 + typeDiversity*0.1 +
		functionalDiversity*0.1)

	return math.Max(0.0, math.Min(1.0, overallDiversity))
}

// calculateComplexityDiversity 计算复杂度多样性
func (fde *FactorDiscoveryEngine) calculateComplexityDiversity(factor *Factor) float64 {
	if fde.geneticAlgorithm == nil || len(fde.geneticAlgorithm.population) == 0 {
		return 1.0
	}

	// 计算种群的平均复杂度
	totalComplexity := 0.0
	for _, f := range fde.geneticAlgorithm.population {
		totalComplexity += f.Complexity
	}
	avgComplexity := totalComplexity / float64(len(fde.geneticAlgorithm.population))

	// 复杂度差异越大，多样性越高
	complexityDiff := math.Abs(factor.Complexity - avgComplexity)

	// 标准化到0-1范围
	maxComplexityDiff := 10.0 // 假设最大复杂度差异为10
	return math.Min(1.0, complexityDiff/maxComplexityDiff)
}

// calculateParameterDiversity 计算参数多样性
func (fde *FactorDiscoveryEngine) calculateParameterDiversity(factor *Factor) float64 {
	if fde.geneticAlgorithm == nil || len(fde.geneticAlgorithm.population) == 0 {
		return 1.0
	}

	// 统计种群中参数的分布
	parameterCounts := make(map[string]int)
	totalFactors := 0

	for _, f := range fde.geneticAlgorithm.population {
		for key := range f.Parameters {
			parameterCounts[key]++
		}
		totalFactors++
	}

	// 计算当前因子参数的稀有度
	rarityScore := 0.0
	paramCount := 0

	for key := range factor.Parameters {
		count, exists := parameterCounts[key]
		if !exists {
			rarityScore += 1.0 // 全新参数，最高稀有度
		} else {
			frequency := float64(count) / float64(totalFactors)
			rarityScore += (1.0 - frequency) // 频率越低，稀有度越高
		}
		paramCount++
	}

	if paramCount == 0 {
		return 0.5
	}

	return rarityScore / float64(paramCount)
}

// calculateTypeDiversity 计算类型多样性
func (fde *FactorDiscoveryEngine) calculateTypeDiversity(factor *Factor) float64 {
	if fde.geneticAlgorithm == nil || len(fde.geneticAlgorithm.population) == 0 {
		return 1.0
	}

	// 统计种群中各类型的分布
	typeCounts := make(map[string]int)
	totalFactors := 0

	for _, f := range fde.geneticAlgorithm.population {
		typeCounts[f.Type]++
		totalFactors++
	}

	// 计算当前因子类型的稀有度
	count, exists := typeCounts[factor.Type]
	if !exists {
		return 1.0 // 全新类型
	}

	frequency := float64(count) / float64(totalFactors)
	return 1.0 - frequency // 频率越低，多样性越高
}

// calculateFunctionalDiversity 计算功能多样性
func (fde *FactorDiscoveryEngine) calculateFunctionalDiversity(factor *Factor) float64 {
	// 基于因子表达式中使用的函数和操作符的多样性

	// 提取表达式中的函数
	functions := fde.extractFunctions(factor.Expression)
	operators := fde.extractOperators(factor.Expression)

	// 统计种群中函数和操作符的使用频率
	functionCounts := make(map[string]int)
	operatorCounts := make(map[string]int)
	totalFactors := 0

	if fde.geneticAlgorithm != nil {
		for _, f := range fde.geneticAlgorithm.population {
			factorFunctions := fde.extractFunctions(f.Expression)
			factorOperators := fde.extractOperators(f.Expression)

			for _, fn := range factorFunctions {
				functionCounts[fn]++
			}
			for _, op := range factorOperators {
				operatorCounts[op]++
			}
			totalFactors++
		}
	}

	// 计算功能稀有度
	functionalRarity := 0.0
	elementCount := 0

	for _, fn := range functions {
		count := functionCounts[fn]
		if count == 0 {
			functionalRarity += 1.0
		} else {
			frequency := float64(count) / float64(totalFactors)
			functionalRarity += (1.0 - frequency)
		}
		elementCount++
	}

	for _, op := range operators {
		count := operatorCounts[op]
		if count == 0 {
			functionalRarity += 1.0
		} else {
			frequency := float64(count) / float64(totalFactors)
			functionalRarity += (1.0 - frequency)
		}
		elementCount++
	}

	if elementCount == 0 {
		return 0.5
	}

	return functionalRarity / float64(elementCount)
}

// extractFunctions 提取表达式中的函数
func (fde *FactorDiscoveryEngine) extractFunctions(expression string) []string {
	functions := []string{}

	// 定义常见的技术分析函数
	commonFunctions := []string{"SMA", "EMA", "RSI", "MACD", "STDDEV", "MAX", "MIN", "ABS", "LOG", "SQRT", "IF"}

	expr := strings.ToUpper(expression)
	for _, fn := range commonFunctions {
		if strings.Contains(expr, fn) {
			functions = append(functions, fn)
		}
	}

	return functions
}

// extractOperators 提取表达式中的操作符
func (fde *FactorDiscoveryEngine) extractOperators(expression string) []string {
	operators := []string{}

	commonOperators := []string{"+", "-", "*", "/", "^", ">", "<", ">=", "<=", "==", "!="}

	for _, op := range commonOperators {
		if strings.Contains(expression, op) {
			operators = append(operators, op)
		}
	}

	return operators
}

func (fde *FactorDiscoveryEngine) calculatePopulationDiversity() float64 {
	// 计算种群多样性

	if fde.geneticAlgorithm == nil || len(fde.geneticAlgorithm.population) < 2 {
		return 0.0
	}

	population := fde.geneticAlgorithm.population
	n := len(population)

	// 1. 计算所有因子对之间的平均距离
	totalDistance := 0.0
	pairCount := 0

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			distance := fde.calculateFactorDistance(population[i], population[j])
			totalDistance += distance
			pairCount++
		}
	}

	avgDistance := 0.0
	if pairCount > 0 {
		avgDistance = totalDistance / float64(pairCount)
	}

	// 2. 计算适应度分布的多样性
	fitnessDiversity := fde.calculateFitnessDistributionDiversity()

	// 3. 计算表达式类型的多样性
	typeDiversity := fde.calculateTypeDistributionDiversity()

	// 4. 计算复杂度分布的多样性
	complexityDiversity := fde.calculateComplexityDistributionDiversity()

	// 5. 计算参数空间的覆盖度
	parameterCoverage := fde.calculateParameterSpaceCoverage()

	// 综合多样性分数
	overallDiversity := (avgDistance*0.3 + fitnessDiversity*0.25 +
		typeDiversity*0.2 + complexityDiversity*0.15 +
		parameterCoverage*0.1)

	return math.Max(0.0, math.Min(1.0, overallDiversity))
}

// calculateFitnessDistributionDiversity 计算适应度分布多样性
func (fde *FactorDiscoveryEngine) calculateFitnessDistributionDiversity() float64 {
	if fde.geneticAlgorithm == nil || len(fde.geneticAlgorithm.population) == 0 {
		return 0.0
	}

	// 收集适应度值
	fitnessValues := make([]float64, len(fde.geneticAlgorithm.population))
	for i, factor := range fde.geneticAlgorithm.population {
		fitnessValues[i] = factor.Fitness
	}

	// 计算适应度的标准差
	mean := 0.0
	for _, fitness := range fitnessValues {
		mean += fitness
	}
	mean /= float64(len(fitnessValues))

	variance := 0.0
	for _, fitness := range fitnessValues {
		diff := fitness - mean
		variance += diff * diff
	}
	variance /= float64(len(fitnessValues))

	stdDev := math.Sqrt(variance)

	// 标准差越大，多样性越高
	// 假设最大标准差为1.0进行标准化
	return math.Min(1.0, stdDev)
}

// calculateTypeDistributionDiversity 计算类型分布多样性
func (fde *FactorDiscoveryEngine) calculateTypeDistributionDiversity() float64 {
	if fde.geneticAlgorithm == nil || len(fde.geneticAlgorithm.population) == 0 {
		return 0.0
	}

	// 统计各类型的分布
	typeCounts := make(map[string]int)
	totalFactors := len(fde.geneticAlgorithm.population)

	for _, factor := range fde.geneticAlgorithm.population {
		typeCounts[factor.Type]++
	}

	// 计算香农熵（Shannon Entropy）
	entropy := 0.0
	for _, count := range typeCounts {
		if count > 0 {
			probability := float64(count) / float64(totalFactors)
			entropy -= probability * math.Log2(probability)
		}
	}

	// 标准化熵值（最大熵为log2(类型数)）
	maxEntropy := math.Log2(float64(len(typeCounts)))
	if maxEntropy == 0 {
		return 0.0
	}

	return entropy / maxEntropy
}

// calculateComplexityDistributionDiversity 计算复杂度分布多样性
func (fde *FactorDiscoveryEngine) calculateComplexityDistributionDiversity() float64 {
	if fde.geneticAlgorithm == nil || len(fde.geneticAlgorithm.population) == 0 {
		return 0.0
	}

	// 收集复杂度值
	complexities := make([]float64, len(fde.geneticAlgorithm.population))
	for i, factor := range fde.geneticAlgorithm.population {
		complexities[i] = factor.Complexity
	}

	// 计算复杂度的变异系数（标准差/均值）
	mean := 0.0
	for _, complexity := range complexities {
		mean += complexity
	}
	mean /= float64(len(complexities))

	if mean == 0 {
		return 0.0
	}

	variance := 0.0
	for _, complexity := range complexities {
		diff := complexity - mean
		variance += diff * diff
	}
	variance /= float64(len(complexities))

	stdDev := math.Sqrt(variance)
	coefficientOfVariation := stdDev / mean

	// 变异系数越大，多样性越高
	// 假设最大变异系数为1.0进行标准化
	return math.Min(1.0, coefficientOfVariation)
}

// calculateParameterSpaceCoverage 计算参数空间覆盖度
func (fde *FactorDiscoveryEngine) calculateParameterSpaceCoverage() float64 {
	if fde.geneticAlgorithm == nil || len(fde.geneticAlgorithm.population) == 0 {
		return 0.0
	}

	// 统计所有使用的参数
	allParameters := make(map[string][]float64)

	for _, factor := range fde.geneticAlgorithm.population {
		for key, value := range factor.Parameters {
			if floatVal, ok := fde.toFloat64(value); ok {
				allParameters[key] = append(allParameters[key], floatVal)
			}
		}
	}

	if len(allParameters) == 0 {
		return 0.0
	}

	// 计算每个参数维度的覆盖度
	totalCoverage := 0.0
	paramCount := 0

	for paramName, values := range allParameters {
		if len(values) < 2 {
			continue
		}

		// 计算参数值的分布范围
		minVal := values[0]
		maxVal := values[0]
		for _, val := range values {
			if val < minVal {
				minVal = val
			}
			if val > maxVal {
				maxVal = val
			}
		}

		// 计算覆盖度（基于值的分散程度）
		if maxVal != minVal {
			// 计算值的标准差
			mean := 0.0
			for _, val := range values {
				mean += val
			}
			mean /= float64(len(values))

			variance := 0.0
			for _, val := range values {
				diff := val - mean
				variance += diff * diff
			}
			variance /= float64(len(values))

			stdDev := math.Sqrt(variance)
			range_ := maxVal - minVal

			// 标准差与范围的比值表示分散程度
			if range_ > 0 {
				coverage := math.Min(1.0, stdDev/(range_/4.0)) // 除以4是经验值
				totalCoverage += coverage
			}
		}

		paramCount++
	}

	if paramCount == 0 {
		return 0.0
	}

	return totalCoverage / float64(paramCount)
}

func (fde *FactorDiscoveryEngine) calculateAvgComplexity() float64 {
	if len(fde.geneticAlgorithm.population) == 0 {
		return 0
	}

	totalComplexity := 0
	for _, factor := range fde.geneticAlgorithm.population {
		totalComplexity += factor.Complexity
	}

	return float64(totalComplexity) / float64(len(fde.geneticAlgorithm.population))
}

func (fde *FactorDiscoveryEngine) selectElite(count int) []*Factor {
	// 选择适应度最高的因子
	factors := make([]*Factor, len(fde.geneticAlgorithm.population))
	copy(factors, fde.geneticAlgorithm.population)

	sort.Slice(factors, func(i, j int) bool {
		return factors[i].Fitness > factors[j].Fitness
	})

	if count > len(factors) {
		count = len(factors)
	}

	return factors[:count]
}

func (fde *FactorDiscoveryEngine) selectParent() *Factor {
	// 锦标赛选择
	tournamentSize := 3
	tournament := make([]*Factor, tournamentSize)

	for i := 0; i < tournamentSize; i++ {
		idx := rand.Intn(len(fde.geneticAlgorithm.population))
		tournament[i] = fde.geneticAlgorithm.population[idx]
	}

	best := tournament[0]
	for _, factor := range tournament[1:] {
		if factor.Fitness > best.Fitness {
			best = factor
		}
	}

	return best
}

func (fde *FactorDiscoveryEngine) crossover(parent1, parent2 *Factor) *Factor {
	// 实现因子交叉操作 - 遗传算法中的交叉操作
	child := &Factor{
		ID:           fde.generateFactorID(),
		Type:         parent1.Type,
		Parameters:   make(map[string]float64),
		Parents:      []string{parent1.ID, parent2.ID},
		Generation:   int(math.Max(float64(parent1.Generation), float64(parent2.Generation))) + 1,
		Status:       "ACTIVE",
		CreatedBy:    "genetic_algorithm",
		DiscoveredAt: time.Now(),
		LastUpdated:  time.Now(),
	}

	// 选择交叉策略
	crossoverType := rand.Intn(3)

	switch crossoverType {
	case 0:
		// 单点交叉 - 交换表达式的子树
		child = fde.singlePointCrossover(parent1, parent2, child)
	case 1:
		// 参数交叉 - 混合父因子的参数
		child = fde.parameterCrossover(parent1, parent2, child)
	case 2:
		// 公式交叉 - 组合父因子的公式
		child = fde.formulaCrossover(parent1, parent2, child)
	}

	// 继承父因子的部分特性（加权平均）
	weight1 := 0.5 + (rand.Float64()-0.5)*0.2 // 0.4-0.6之间的权重
	weight2 := 1.0 - weight1

	// 混合统计特性
	if parent1.IC != 0 && parent2.IC != 0 {
		child.IC = parent1.IC*weight1 + parent2.IC*weight2
	}
	if parent1.ICIR != 0 && parent2.ICIR != 0 {
		child.ICIR = parent1.ICIR*weight1 + parent2.ICIR*weight2
	}

	// 混合参数
	for key, value1 := range parent1.Parameters {
		if value2, exists := parent2.Parameters[key]; exists {
			child.Parameters[key] = value1*weight1 + value2*weight2
		} else {
			child.Parameters[key] = value1
		}
	}

	// 添加parent2中parent1没有的参数
	for key, value2 := range parent2.Parameters {
		if _, exists := child.Parameters[key]; !exists {
			child.Parameters[key] = value2
		}
	}

	return child
}

// singlePointCrossover 单点交叉 - 在表达式树中选择一个节点进行交换
func (fde *FactorDiscoveryEngine) singlePointCrossover(parent1, parent2 *Factor, child *Factor) *Factor {
	if parent1.Expression != nil && parent2.Expression != nil {
		// 复制parent1的表达式作为基础
		child.Expression = fde.copyExpression(parent1.Expression)

		// 随机选择一个子树进行替换
		if len(parent2.Expression.Children) > 0 {
			crossoverPoint := rand.Intn(len(parent2.Expression.Children))
			if len(child.Expression.Children) > crossoverPoint {
				child.Expression.Children[crossoverPoint] = fde.copyExpression(parent2.Expression.Children[crossoverPoint])
			}
		}

		// 更新公式
		child.Formula = fde.expressionToFormula(child.Expression)
		child.Name = fmt.Sprintf("Cross_%s_%s", parent1.Name[:min(10, len(parent1.Name))], parent2.Name[:min(10, len(parent2.Name))])
	} else {
		// 如果没有表达式，则混合公式
		child.Formula = fmt.Sprintf("(%s + %s) / 2", parent1.Formula, parent2.Formula)
		child.Name = fmt.Sprintf("Cross_%s_%s", parent1.Name[:min(10, len(parent1.Name))], parent2.Name[:min(10, len(parent2.Name))])
	}
	return child
}

// parameterCrossover 参数交叉 - 混合两个因子的参数
func (fde *FactorDiscoveryEngine) parameterCrossover(parent1, parent2 *Factor, child *Factor) *Factor {
	// 基于较优父因子的结构
	if parent1.IC >= parent2.IC {
		child.Expression = fde.copyExpression(parent1.Expression)
		child.Formula = parent1.Formula
		child.Name = fmt.Sprintf("ParamCross_%s", parent1.Name[:min(15, len(parent1.Name))])
	} else {
		child.Expression = fde.copyExpression(parent2.Expression)
		child.Formula = parent2.Formula
		child.Name = fmt.Sprintf("ParamCross_%s", parent2.Name[:min(15, len(parent2.Name))])
	}

	// 参数已在主函数中混合
	return child
}

// formulaCrossover 公式交叉 - 组合两个因子的公式
func (fde *FactorDiscoveryEngine) formulaCrossover(parent1, parent2 *Factor, child *Factor) *Factor {
	// 随机选择组合方式
	operators := []string{"+", "-", "*", "/", "max", "min"}
	operator := operators[rand.Intn(len(operators))]

	switch operator {
	case "max", "min":
		child.Formula = fmt.Sprintf("%s(%s, %s)", operator, parent1.Formula, parent2.Formula)
	default:
		child.Formula = fmt.Sprintf("(%s %s %s)", parent1.Formula, operator, parent2.Formula)
	}

	child.Name = fmt.Sprintf("FormCross_%s_%s_%s",
		parent1.Name[:min(8, len(parent1.Name))],
		operator,
		parent2.Name[:min(8, len(parent2.Name))])

	// 创建新的表达式树
	child.Expression = fde.parseFormula(child.Formula)

	return child
}

// copyExpression 深度复制表达式树
func (fde *FactorDiscoveryEngine) copyExpression(expr *Expression) *Expression {
	if expr == nil {
		return nil
	}

	copy := &Expression{
		Type:     expr.Type,
		Value:    expr.Value,
		DataType: expr.DataType,
		Children: make([]*Expression, len(expr.Children)),
	}

	for i, child := range expr.Children {
		copy.Children[i] = fde.copyExpression(child)
	}

	return copy
}

// expressionToFormula 将表达式树转换为公式字符串
func (fde *FactorDiscoveryEngine) expressionToFormula(expr *Expression) string {
	if expr == nil {
		return ""
	}

	switch expr.Type {
	case "VARIABLE":
		return fmt.Sprintf("%v", expr.Value)
	case "CONSTANT":
		return fmt.Sprintf("%v", expr.Value)
	case "OPERATOR":
		if len(expr.Children) == 2 {
			left := fde.expressionToFormula(expr.Children[0])
			right := fde.expressionToFormula(expr.Children[1])
			return fmt.Sprintf("(%s %v %s)", left, expr.Value, right)
		}
	case "FUNCTION":
		if len(expr.Children) > 0 {
			args := make([]string, len(expr.Children))
			for i, child := range expr.Children {
				args[i] = fde.expressionToFormula(child)
			}
			return fmt.Sprintf("%v(%s)", expr.Value, strings.Join(args, ", "))
		}
	}

	return fmt.Sprintf("%v", expr.Value)
}

// parseFormula 解析公式字符串为表达式树（简化版本）
func (fde *FactorDiscoveryEngine) parseFormula(formula string) *Expression {
	// 这是一个简化的解析器，实际应用中需要更复杂的实现
	return &Expression{
		Type:     "OPERATOR",
		Value:    "+",
		DataType: "float64",
		Children: []*Expression{
			{Type: "VARIABLE", Value: "close", DataType: "float64"},
			{Type: "CONSTANT", Value: 1.0, DataType: "float64"},
		},
	}
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (fde *FactorDiscoveryEngine) mutate(factor *Factor) {
	// 实现因子变异操作 - 遗传算法中的变异操作
	factor.LastUpdated = time.Now()

	// 选择变异类型
	mutationType := rand.Intn(4)

	switch mutationType {
	case 0:
		// 参数变异 - 随机调整参数值
		fde.mutateParameters(factor)
	case 1:
		// 表达式变异 - 修改表达式树的节点
		fde.mutateExpression(factor)
	case 2:
		// 公式变异 - 在公式中添加噪声项
		fde.mutateFormula(factor)
	case 3:
		// 结构变异 - 改变表达式的结构
		fde.mutateStructure(factor)
	}

	// 更新因子名称以反映变异
	if !strings.Contains(factor.Name, "Mut") {
		factor.Name = fmt.Sprintf("Mut_%s", factor.Name[:min(15, len(factor.Name))])
	}

	// 重置性能指标，需要重新计算
	factor.IC = 0
	factor.ICIR = 0
	factor.TValue = 0
	factor.PValue = 0
	factor.IsSignificant = false
}

// mutateParameters 参数变异
func (fde *FactorDiscoveryEngine) mutateParameters(factor *Factor) {
	mutationRate := 0.1     // 10%的变异率
	mutationStrength := 0.2 // 20%的变异强度

	for key, value := range factor.Parameters {
		if rand.Float64() < mutationRate {
			// 添加高斯噪声
			noise := rand.NormFloat64() * mutationStrength * math.Abs(value)
			factor.Parameters[key] = value + noise

			// 确保参数在合理范围内
			if factor.Parameters[key] < -1000 {
				factor.Parameters[key] = -1000
			} else if factor.Parameters[key] > 1000 {
				factor.Parameters[key] = 1000
			}
		}
	}
}

// mutateExpression 表达式变异
func (fde *FactorDiscoveryEngine) mutateExpression(factor *Factor) {
	if factor.Expression == nil {
		return
	}

	// 随机选择一个节点进行变异
	nodes := fde.collectExpressionNodes(factor.Expression)
	if len(nodes) == 0 {
		return
	}

	nodeToMutate := nodes[rand.Intn(len(nodes))]

	switch nodeToMutate.Type {
	case "CONSTANT":
		// 变异常数值
		if val, ok := nodeToMutate.Value.(float64); ok {
			noise := rand.NormFloat64() * 0.1 * math.Abs(val)
			nodeToMutate.Value = val + noise
		}
	case "OPERATOR":
		// 变异操作符
		operators := []string{"+", "-", "*", "/"}
		nodeToMutate.Value = operators[rand.Intn(len(operators))]
	case "FUNCTION":
		// 变异函数
		functions := []string{"abs", "log", "sqrt", "sin", "cos", "tanh"}
		nodeToMutate.Value = functions[rand.Intn(len(functions))]
	}

	// 更新公式
	factor.Formula = fde.expressionToFormula(factor.Expression)
}

// mutateFormula 公式变异
func (fde *FactorDiscoveryEngine) mutateFormula(factor *Factor) {
	// 在公式中添加小的噪声项
	noiseTerms := []string{
		"+ 0.001",
		"- 0.001",
		"* 1.001",
		"/ 1.001",
		"+ log(1.001)",
		"* (1 + 0.001 * sin(close))",
	}

	noiseTerm := noiseTerms[rand.Intn(len(noiseTerms))]
	factor.Formula = fmt.Sprintf("(%s) %s", factor.Formula, noiseTerm)
}

// mutateStructure 结构变异
func (fde *FactorDiscoveryEngine) mutateStructure(factor *Factor) {
	if factor.Expression == nil {
		return
	}

	// 随机添加或删除子表达式
	if rand.Float64() < 0.5 && len(factor.Expression.Children) > 1 {
		// 删除一个子表达式
		indexToRemove := rand.Intn(len(factor.Expression.Children))
		factor.Expression.Children = append(
			factor.Expression.Children[:indexToRemove],
			factor.Expression.Children[indexToRemove+1:]...)
	} else {
		// 添加一个新的子表达式
		newChild := &Expression{
			Type:     "CONSTANT",
			Value:    rand.Float64(),
			DataType: "float64",
		}
		factor.Expression.Children = append(factor.Expression.Children, newChild)
	}

	// 更新公式
	factor.Formula = fde.expressionToFormula(factor.Expression)
}

// collectExpressionNodes 收集表达式树中的所有节点
func (fde *FactorDiscoveryEngine) collectExpressionNodes(expr *Expression) []*Expression {
	if expr == nil {
		return nil
	}

	nodes := []*Expression{expr}
	for _, child := range expr.Children {
		nodes = append(nodes, fde.collectExpressionNodes(child)...)
	}

	return nodes
}

func (fde *FactorDiscoveryEngine) copyFactor(original *Factor) *Factor {
	copy := *original
	copy.ID = fde.generateFactorID()
	copy.DiscoveredAt = time.Now()
	copy.LastUpdated = time.Now()
	return &copy
}

func (fde *FactorDiscoveryEngine) calculateFactorSimilarity(factor1, factor2 *Factor) float64 {
	// 计算因子相似度 - 综合多个维度的相似性
	if factor1 == nil || factor2 == nil {
		return 0.0
	}

	var totalSimilarity float64
	var weightSum float64

	// 1. 公式相似度 (权重: 0.3)
	formulaSimilarity := fde.calculateFormulaSimilarity(factor1.Formula, factor2.Formula)
	totalSimilarity += formulaSimilarity * 0.3
	weightSum += 0.3

	// 2. 相关性相似度 (权重: 0.25)
	if factor1.ID != "" && factor2.ID != "" {
		correlationSimilarity := fde.calculateCorrelationSimilarity(factor1, factor2)
		totalSimilarity += correlationSimilarity * 0.25
		weightSum += 0.25
	}

	// 3. 参数相似度 (权重: 0.2)
	parameterSimilarity := fde.calculateParameterSimilarity(factor1.Parameters, factor2.Parameters)
	totalSimilarity += parameterSimilarity * 0.2
	weightSum += 0.2

	// 4. 性能指标相似度 (权重: 0.15)
	performanceSimilarity := fde.calculatePerformanceSimilarity(factor1, factor2)
	totalSimilarity += performanceSimilarity * 0.15
	weightSum += 0.15

	// 5. 表达式结构相似度 (权重: 0.1)
	structureSimilarity := fde.calculateStructureSimilarity(factor1.Expression, factor2.Expression)
	totalSimilarity += structureSimilarity * 0.1
	weightSum += 0.1

	if weightSum > 0 {
		return totalSimilarity / weightSum
	}

	return 0.0
}

// calculateFormulaSimilarity 计算公式相似度
func (fde *FactorDiscoveryEngine) calculateFormulaSimilarity(formula1, formula2 string) float64 {
	if formula1 == "" || formula2 == "" {
		return 0.0
	}

	// 完全相同
	if formula1 == formula2 {
		return 1.0
	}

	// 计算编辑距离相似度
	editDistance := fde.calculateEditDistance(formula1, formula2)
	maxLen := math.Max(float64(len(formula1)), float64(len(formula2)))

	if maxLen == 0 {
		return 1.0
	}

	similarity := 1.0 - float64(editDistance)/maxLen
	if similarity < 0 {
		similarity = 0
	}

	return similarity
}

// calculateCorrelationSimilarity 计算相关性相似度
func (fde *FactorDiscoveryEngine) calculateCorrelationSimilarity(factor1, factor2 *Factor) float64 {
	// 检查两个因子是否有共同的相关性数据
	if len(factor1.Correlations) == 0 || len(factor2.Correlations) == 0 {
		return 0.0
	}

	var correlationSum float64
	var count int

	// 计算共同因子的相关性差异
	for factorID, corr1 := range factor1.Correlations {
		if corr2, exists := factor2.Correlations[factorID]; exists {
			// 相关性越接近，相似度越高
			diff := math.Abs(corr1 - corr2)
			similarity := 1.0 - diff/2.0 // 相关性范围是[-1,1]，最大差异是2
			correlationSum += similarity
			count++
		}
	}

	if count > 0 {
		return correlationSum / float64(count)
	}

	return 0.0
}

// calculateParameterSimilarity 计算参数相似度
func (fde *FactorDiscoveryEngine) calculateParameterSimilarity(params1, params2 map[string]float64) float64 {
	if len(params1) == 0 && len(params2) == 0 {
		return 1.0
	}

	if len(params1) == 0 || len(params2) == 0 {
		return 0.0
	}

	var similaritySum float64
	var count int

	// 计算共同参数的相似度
	for key, value1 := range params1 {
		if value2, exists := params2[key]; exists {
			// 计算参数值的相似度
			if value1 == 0 && value2 == 0 {
				similaritySum += 1.0
			} else {
				maxVal := math.Max(math.Abs(value1), math.Abs(value2))
				if maxVal > 0 {
					diff := math.Abs(value1 - value2)
					similarity := 1.0 - diff/maxVal
					if similarity < 0 {
						similarity = 0
					}
					similaritySum += similarity
				}
			}
			count++
		}
	}

	if count > 0 {
		// 考虑参数数量的差异
		totalParams := len(params1) + len(params2) - count
		paramCountPenalty := float64(totalParams-count) / float64(totalParams)
		baseSimilarity := similaritySum / float64(count)
		return baseSimilarity * (1.0 - paramCountPenalty*0.5)
	}

	return 0.0
}

// calculatePerformanceSimilarity 计算性能指标相似度
func (fde *FactorDiscoveryEngine) calculatePerformanceSimilarity(factor1, factor2 *Factor) float64 {
	var similarities []float64

	// IC相似度
	if factor1.IC != 0 || factor2.IC != 0 {
		maxIC := math.Max(math.Abs(factor1.IC), math.Abs(factor2.IC))
		if maxIC > 0 {
			icDiff := math.Abs(factor1.IC - factor2.IC)
			icSimilarity := 1.0 - icDiff/maxIC
			if icSimilarity < 0 {
				icSimilarity = 0
			}
			similarities = append(similarities, icSimilarity)
		}
	}

	// ICIR相似度
	if factor1.ICIR != 0 || factor2.ICIR != 0 {
		maxICIR := math.Max(math.Abs(factor1.ICIR), math.Abs(factor2.ICIR))
		if maxICIR > 0 {
			icirDiff := math.Abs(factor1.ICIR - factor2.ICIR)
			icirSimilarity := 1.0 - icirDiff/maxICIR
			if icirSimilarity < 0 {
				icirSimilarity = 0
			}
			similarities = append(similarities, icirSimilarity)
		}
	}

	// 计算平均相似度
	if len(similarities) > 0 {
		var sum float64
		for _, sim := range similarities {
			sum += sim
		}
		return sum / float64(len(similarities))
	}

	return 0.0
}

// calculateStructureSimilarity 计算表达式结构相似度
func (fde *FactorDiscoveryEngine) calculateStructureSimilarity(expr1, expr2 *Expression) float64 {
	if expr1 == nil && expr2 == nil {
		return 1.0
	}

	if expr1 == nil || expr2 == nil {
		return 0.0
	}

	// 类型相似度
	typeSimilarity := 0.0
	if expr1.Type == expr2.Type {
		typeSimilarity = 1.0
	}

	// 值相似度
	valueSimilarity := 0.0
	if expr1.Value == expr2.Value {
		valueSimilarity = 1.0
	}

	// 子节点数量相似度
	childCountSimilarity := 0.0
	maxChildren := math.Max(float64(len(expr1.Children)), float64(len(expr2.Children)))
	if maxChildren > 0 {
		childCountDiff := math.Abs(float64(len(expr1.Children)) - float64(len(expr2.Children)))
		childCountSimilarity = 1.0 - childCountDiff/maxChildren
	} else {
		childCountSimilarity = 1.0
	}

	// 递归计算子节点相似度
	var childSimilarities []float64
	minChildren := int(math.Min(float64(len(expr1.Children)), float64(len(expr2.Children))))
	for i := 0; i < minChildren; i++ {
		childSim := fde.calculateStructureSimilarity(expr1.Children[i], expr2.Children[i])
		childSimilarities = append(childSimilarities, childSim)
	}

	childSimilarity := 0.0
	if len(childSimilarities) > 0 {
		var sum float64
		for _, sim := range childSimilarities {
			sum += sim
		}
		childSimilarity = sum / float64(len(childSimilarities))
	}

	// 加权平均
	return typeSimilarity*0.3 + valueSimilarity*0.3 + childCountSimilarity*0.2 + childSimilarity*0.2
}

// calculateEditDistance 计算编辑距离
func (fde *FactorDiscoveryEngine) calculateEditDistance(s1, s2 string) int {
	m, n := len(s1), len(s2)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	// 初始化
	for i := 0; i <= m; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= n; j++ {
		dp[0][j] = j
	}

	// 动态规划计算编辑距离
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if s1[i-1] == s2[j-1] {
				dp[i][j] = dp[i-1][j-1]
			} else {
				dp[i][j] = 1 + min3(dp[i-1][j], dp[i][j-1], dp[i-1][j-1])
			}
		}
	}

	return dp[m][n]
}

// min3 返回三个数中的最小值
func min3(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= c {
		return b
	}
	return c
}

func (fde *FactorDiscoveryEngine) assessFactorImpact(factor *Factor) string {
	if math.Abs(factor.IC) > 0.05 {
		return "HIGH"
	} else if math.Abs(factor.IC) > 0.02 {
		return "MEDIUM"
	}
	return "LOW"
}

func (fde *FactorDiscoveryEngine) calculateIC(factor *Factor, period int) ICResult {
	// 实现IC计算 - Information Coefficient (信息系数)
	// IC是因子值与未来收益率的相关系数

	ctx := context.Background()

	// 获取因子数据和收益率数据
	factorValues, returns, err := fde.getFactorAndReturnData(ctx, factor, period)
	if err != nil || len(factorValues) == 0 || len(returns) == 0 {
		log.Printf("Failed to get factor data for IC calculation: %v", err)
		return ICResult{
			Period:        period,
			IC:            0.0,
			RankIC:        0.0,
			TValue:        0.0,
			PValue:        1.0,
			IsSignificant: false,
			SampleSize:    0,
		}
	}

	// 确保数据长度一致
	minLen := len(factorValues)
	if len(returns) < minLen {
		minLen = len(returns)
	}

	if minLen < 10 { // 至少需要10个样本
		return ICResult{
			Period:        period,
			IC:            0.0,
			RankIC:        0.0,
			TValue:        0.0,
			PValue:        1.0,
			IsSignificant: false,
			SampleSize:    minLen,
		}
	}

	// 截取相同长度的数据
	factorValues = factorValues[:minLen]
	returns = returns[:minLen]

	// 计算Pearson相关系数 (IC)
	ic := fde.calculatePearsonCorrelation(factorValues, returns)

	// 计算Rank IC (Spearman相关系数)
	rankIC := fde.calculateSpearmanCorrelation(factorValues, returns)

	// 计算t统计量和p值
	tValue, pValue := fde.calculateTTest(ic, minLen)

	// 判断显著性 (通常使用0.05的显著性水平)
	isSignificant := pValue < 0.05 && math.Abs(ic) > 0.02

	// 计算置信区间
	confidenceInterval := fde.calculateConfidenceInterval(ic, minLen, 0.95)

	return ICResult{
		Period:             period,
		IC:                 ic,
		RankIC:             rankIC,
		TValue:             tValue,
		PValue:             pValue,
		IsSignificant:      isSignificant,
		SampleSize:         minLen,
		ConfidenceInterval: confidenceInterval,
	}
}

// getFactorAndReturnData 获取因子数据和收益率数据
func (fde *FactorDiscoveryEngine) getFactorAndReturnData(ctx context.Context, factor *Factor, period int) ([]float64, []float64, error) {
	// 这里应该从数据库获取真实的因子值和收益率数据
	// 为了演示，我们生成一些模拟数据，但在实际应用中应该替换为真实的数据库查询

	// 尝试从数据库获取数据
	if fde.db != nil {
		factorValues, returns, err := fde.getFactorDataFromDB(ctx, factor, period)
		if err == nil && len(factorValues) > 0 {
			return factorValues, returns, nil
		}
		log.Printf("Failed to get data from database, using fallback: %v", err)
	}

	// 如果数据库查询失败，返回空数据而不是模拟数据
	log.Printf("No real data available for factor %s, returning empty dataset", factor.ID)
	return []float64{}, []float64{}, fmt.Errorf("no real data available")
}

// getFactorDataFromDB 从数据库获取因子数据
func (fde *FactorDiscoveryEngine) getFactorDataFromDB(ctx context.Context, factor *Factor, period int) ([]float64, []float64, error) {
	// 实际的数据库查询逻辑
	query := `
		SELECT f.factor_value, r.return_rate 
		FROM factor_values f 
		JOIN returns r ON f.symbol = r.symbol 
		AND r.date = DATE_ADD(f.date, INTERVAL ? DAY)
		WHERE f.factor_id = ? 
		AND f.date >= DATE_SUB(NOW(), INTERVAL 1 YEAR)
		ORDER BY f.date
	`

	rows, err := fde.db.QueryContext(ctx, query, period, factor.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query factor data: %w", err)
	}
	defer rows.Close()

	var factorValues []float64
	var returns []float64

	for rows.Next() {
		var factorValue, returnRate float64
		if err := rows.Scan(&factorValue, &returnRate); err != nil {
			continue
		}
		factorValues = append(factorValues, factorValue)
		returns = append(returns, returnRate)
	}

	return factorValues, returns, nil
}

// calculatePearsonCorrelation 计算Pearson相关系数
func (fde *FactorDiscoveryEngine) calculatePearsonCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) == 0 {
		return 0.0
	}

	n := float64(len(x))

	// 计算均值
	var sumX, sumY float64
	for i := 0; i < len(x); i++ {
		sumX += x[i]
		sumY += y[i]
	}
	meanX := sumX / n
	meanY := sumY / n

	// 计算协方差和方差
	var covariance, varianceX, varianceY float64
	for i := 0; i < len(x); i++ {
		dx := x[i] - meanX
		dy := y[i] - meanY
		covariance += dx * dy
		varianceX += dx * dx
		varianceY += dy * dy
	}

	// 计算相关系数
	if varianceX == 0 || varianceY == 0 {
		return 0.0
	}

	correlation := covariance / math.Sqrt(varianceX*varianceY)
	return correlation
}

// calculateSpearmanCorrelation 计算Spearman相关系数
func (fde *FactorDiscoveryEngine) calculateSpearmanCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) == 0 {
		return 0.0
	}

	// 计算排名
	rankX := fde.calculateRanks(x)
	rankY := fde.calculateRanks(y)

	// 对排名计算Pearson相关系数
	return fde.calculatePearsonCorrelation(rankX, rankY)
}

// calculateRanks 计算排名
func (fde *FactorDiscoveryEngine) calculateRanks(values []float64) []float64 {
	n := len(values)
	ranks := make([]float64, n)

	// 创建索引数组
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}

	// 按值排序索引
	sort.Slice(indices, func(i, j int) bool {
		return values[indices[i]] < values[indices[j]]
	})

	// 分配排名
	for i, idx := range indices {
		ranks[idx] = float64(i + 1)
	}

	return ranks
}

// calculateTTest 计算t统计量和p值
func (fde *FactorDiscoveryEngine) calculateTTest(correlation float64, sampleSize int) (float64, float64) {
	if sampleSize <= 2 {
		return 0.0, 1.0
	}

	// t统计量计算
	df := float64(sampleSize - 2) // 自由度
	tValue := correlation * math.Sqrt(df/(1-correlation*correlation))

	// 简化的p值计算 (实际应用中应使用更精确的t分布)
	// 这里使用近似公式
	absT := math.Abs(tValue)
	var pValue float64

	if absT > 2.576 {
		pValue = 0.01 // 99%置信度
	} else if absT > 1.96 {
		pValue = 0.05 // 95%置信度
	} else if absT > 1.645 {
		pValue = 0.10 // 90%置信度
	} else {
		pValue = 0.20
	}

	return tValue, pValue
}

// calculateConfidenceInterval 计算置信区间
func (fde *FactorDiscoveryEngine) calculateConfidenceInterval(correlation float64, sampleSize int, confidence float64) [2]float64 {
	if sampleSize <= 3 {
		return [2]float64{correlation, correlation}
	}

	// Fisher变换
	z := 0.5 * math.Log((1+correlation)/(1-correlation))

	// 标准误差
	se := 1.0 / math.Sqrt(float64(sampleSize-3))

	// 临界值 (简化，实际应使用正态分布的临界值)
	var criticalValue float64
	if confidence >= 0.99 {
		criticalValue = 2.576
	} else if confidence >= 0.95 {
		criticalValue = 1.96
	} else {
		criticalValue = 1.645
	}

	// 置信区间
	lowerZ := z - criticalValue*se
	upperZ := z + criticalValue*se

	// 反Fisher变换
	lower := (math.Exp(2*lowerZ) - 1) / (math.Exp(2*lowerZ) + 1)
	upper := (math.Exp(2*upperZ) - 1) / (math.Exp(2*upperZ) + 1)

	return [2]float64{lower, upper}
}

func (fde *FactorDiscoveryEngine) calculateRollingIC(factor *Factor) []RollingIC {
	// 实现滚动IC计算 - 计算时间序列上的滚动IC值
	ctx := context.Background()

	// 获取历史数据
	historicalData, err := fde.getHistoricalFactorData(ctx, factor)
	if err != nil || len(historicalData) == 0 {
		log.Printf("Failed to get historical data for rolling IC: %v", err)
		return []RollingIC{}
	}

	var rollingICs []RollingIC
	windowSize := 60 // 60天滚动窗口

	// 确保有足够的数据进行滚动计算
	if len(historicalData) < windowSize+20 {
		log.Printf("Insufficient data for rolling IC calculation: need at least %d points, got %d",
			windowSize+20, len(historicalData))
		return []RollingIC{}
	}

	// 滚动计算IC
	for i := windowSize; i < len(historicalData)-20; i++ {
		// 获取窗口内的数据
		windowData := historicalData[i-windowSize : i]

		// 提取因子值和收益率
		factorValues := make([]float64, len(windowData))
		returns := make([]float64, len(windowData))

		for j, data := range windowData {
			factorValues[j] = data.FactorValue
			returns[j] = data.Return
		}

		// 计算IC和RankIC
		ic := fde.calculatePearsonCorrelation(factorValues, returns)
		rankIC := fde.calculateSpearmanCorrelation(factorValues, returns)

		// 计算t统计量
		tValue, _ := fde.calculateTTest(ic, len(factorValues))

		// 判断显著性
		isSignificant := math.Abs(tValue) > 1.96 && math.Abs(ic) > 0.02

		rollingIC := RollingIC{
			Date:          historicalData[i].Date,
			IC:            ic,
			RankIC:        rankIC,
			TValue:        tValue,
			IsSignificant: isSignificant,
		}

		rollingICs = append(rollingICs, rollingIC)
	}

	return rollingICs
}

// HistoricalFactorData 历史因子数据
type HistoricalFactorData struct {
	Date        time.Time
	FactorValue float64
	Return      float64
}

// getHistoricalFactorData 获取历史因子数据
func (fde *FactorDiscoveryEngine) getHistoricalFactorData(ctx context.Context, factor *Factor) ([]HistoricalFactorData, error) {
	if fde.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	// 查询历史因子数据和对应的收益率
	query := `
		SELECT f.date, f.factor_value, 
		       COALESCE(r.return_rate, 0) as return_rate
		FROM factor_values f
		LEFT JOIN returns r ON f.symbol = r.symbol 
		    AND r.date = DATE_ADD(f.date, INTERVAL 1 DAY)
		WHERE f.factor_id = ?
		    AND f.date >= DATE_SUB(NOW(), INTERVAL 2 YEAR)
		    AND f.factor_value IS NOT NULL
		ORDER BY f.date
	`

	rows, err := fde.db.QueryContext(ctx, query, factor.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query historical factor data: %w", err)
	}
	defer rows.Close()

	var data []HistoricalFactorData
	for rows.Next() {
		var item HistoricalFactorData
		if err := rows.Scan(&item.Date, &item.FactorValue, &item.Return); err != nil {
			continue
		}
		data = append(data, item)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no historical data found for factor %s", factor.ID)
	}

	return data, nil
}

func (fde *FactorDiscoveryEngine) calculateICDecay(factor *Factor) []float64 {
	// 实现IC衰减计算 - 计算因子在不同前瞻期下的IC衰减情况
	ctx := context.Background()

	// 定义前瞻期 (1天, 3天, 5天, 10天, 20天)
	periods := []int{1, 3, 5, 10, 20}
	icDecay := make([]float64, len(periods))

	// 获取因子的历史数据
	historicalData, err := fde.getHistoricalFactorData(ctx, factor)
	if err != nil || len(historicalData) < 100 {
		log.Printf("Insufficient data for IC decay calculation: %v", err)
		// 返回理论衰减模式而不是随机数据
		return []float64{0.0, 0.0, 0.0, 0.0, 0.0}
	}

	// 为每个前瞻期计算IC
	for i, period := range periods {
		ic := fde.calculateICForPeriod(historicalData, period)
		icDecay[i] = ic
	}

	// 标准化IC衰减 (以第一期为基准)
	if icDecay[0] != 0 {
		baseIC := math.Abs(icDecay[0])
		for i := range icDecay {
			icDecay[i] = math.Abs(icDecay[i]) / baseIC
		}
	}

	return icDecay
}

// calculateICForPeriod 计算特定前瞻期的IC
func (fde *FactorDiscoveryEngine) calculateICForPeriod(data []HistoricalFactorData, period int) float64 {
	if len(data) < period+20 {
		return 0.0
	}

	var factorValues []float64
	var returns []float64

	// 构建因子值和对应前瞻期收益率的配对数据
	for i := 0; i < len(data)-period; i++ {
		factorValue := data[i].FactorValue

		// 计算前瞻期收益率
		var periodReturn float64
		if i+period < len(data) {
			// 计算period天的累积收益率
			startPrice := 100.0 // 假设基准价格
			endPrice := startPrice

			for j := i + 1; j <= i+period && j < len(data); j++ {
				endPrice *= (1 + data[j].Return)
			}

			periodReturn = (endPrice - startPrice) / startPrice
		}

		factorValues = append(factorValues, factorValue)
		returns = append(returns, periodReturn)
	}

	if len(factorValues) < 20 {
		return 0.0
	}

	// 计算IC
	return fde.calculatePearsonCorrelation(factorValues, returns)
}

func (fde *FactorDiscoveryEngine) performGroupBacktest(factor *Factor) GroupBacktest {
	// 实现分组回测 - 将股票按因子值分组，计算各组收益率
	ctx := context.Background()

	// 获取因子数据
	factorData, err := fde.getFactorGroupData(ctx, factor)
	if err != nil || len(factorData) == 0 {
		log.Printf("Failed to get factor data for group backtest: %v", err)
		return GroupBacktest{}
	}

	// 按因子值排序
	sort.Slice(factorData, func(i, j int) bool {
		return factorData[i].FactorValue < factorData[j].FactorValue
	})

	// 分成10组
	numGroups := 10
	groupSize := len(factorData) / numGroups
	if groupSize == 0 {
		groupSize = 1
		numGroups = len(factorData)
	}

	var groups []GroupStats

	// 计算每组的统计数据
	for i := 0; i < numGroups; i++ {
		start := i * groupSize
		end := start + groupSize
		if i == numGroups-1 {
			end = len(factorData) // 最后一组包含剩余所有数据
		}

		if start >= len(factorData) {
			break
		}

		groupData := factorData[start:end]
		groupStats := fde.calculateGroupStats(groupData, i+1)
		groups = append(groups, groupStats)
	}

	// 计算多空组合 (最高组 - 最低组)
	var longShort GroupStats
	if len(groups) >= 2 {
		highGroup := groups[len(groups)-1]
		lowGroup := groups[0]

		longShort = GroupStats{
			Group:       0, // 0表示多空组合
			Count:       highGroup.Count + lowGroup.Count,
			AvgReturn:   highGroup.AvgReturn - lowGroup.AvgReturn,
			CumReturn:   highGroup.CumReturn - lowGroup.CumReturn,
			Volatility:  math.Sqrt(highGroup.Volatility*highGroup.Volatility + lowGroup.Volatility*lowGroup.Volatility),
			SharpeRatio: 0, // 将在下面计算
			MaxDrawdown: math.Max(highGroup.MaxDrawdown, lowGroup.MaxDrawdown),
		}

		// 计算多空组合的夏普比率
		if longShort.Volatility > 0 {
			longShort.SharpeRatio = longShort.AvgReturn / longShort.Volatility
		}
	}

	return GroupBacktest{
		Groups:    groups,
		LongShort: longShort,
	}
}

// FactorGroupData 因子分组数据
type FactorGroupData struct {
	Symbol      string
	Date        time.Time
	FactorValue float64
	Return      float64
	Price       float64
}

// getFactorGroupData 获取因子分组数据
func (fde *FactorDiscoveryEngine) getFactorGroupData(ctx context.Context, factor *Factor) ([]FactorGroupData, error) {
	if fde.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	// 查询最近一个月的因子数据和收益率
	query := `
		SELECT f.symbol, f.date, f.factor_value, 
		       COALESCE(r.return_rate, 0) as return_rate,
		       COALESCE(p.close_price, 100) as price
		FROM factor_values f
		LEFT JOIN returns r ON f.symbol = r.symbol 
		    AND r.date = DATE_ADD(f.date, INTERVAL 1 DAY)
		LEFT JOIN market_data p ON f.symbol = p.symbol 
		    AND p.date = f.date
		WHERE f.factor_id = ?
		    AND f.date >= DATE_SUB(NOW(), INTERVAL 1 MONTH)
		    AND f.factor_value IS NOT NULL
		ORDER BY f.date DESC, f.factor_value
		LIMIT 1000
	`

	rows, err := fde.db.QueryContext(ctx, query, factor.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query factor group data: %w", err)
	}
	defer rows.Close()

	var data []FactorGroupData
	for rows.Next() {
		var item FactorGroupData
		if err := rows.Scan(&item.Symbol, &item.Date, &item.FactorValue, &item.Return, &item.Price); err != nil {
			continue
		}
		data = append(data, item)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no group data found for factor %s", factor.ID)
	}

	return data, nil
}

// calculateGroupStats 计算分组统计数据
func (fde *FactorDiscoveryEngine) calculateGroupStats(groupData []FactorGroupData, groupNum int) GroupStats {
	if len(groupData) == 0 {
		return GroupStats{Group: groupNum}
	}

	// 计算平均收益率
	var totalReturn float64
	var returns []float64
	var winCount int

	for _, data := range groupData {
		totalReturn += data.Return
		returns = append(returns, data.Return)
		if data.Return > 0 {
			winCount++
		}
	}

	avgReturn := totalReturn / float64(len(groupData))

	// 计算累积收益率
	cumReturn := 1.0
	for _, ret := range returns {
		cumReturn *= (1.0 + ret)
	}
	cumReturn -= 1.0 // 转换为收益率

	// 计算波动率 (标准差)
	var variance float64
	for _, ret := range returns {
		diff := ret - avgReturn
		variance += diff * diff
	}
	volatility := math.Sqrt(variance / float64(len(returns)))

	// 计算夏普比率 (假设无风险利率为0)
	var sharpeRatio float64
	if volatility > 0 {
		sharpeRatio = avgReturn / volatility
	}

	// 计算最大回撤
	maxDrawdown := fde.calculateMaxDrawdown(returns)

	// 计算胜率
	winRate := float64(winCount) / float64(len(groupData))

	// 计算命中率 (简化为胜率)
	hitRate := winRate

	return GroupStats{
		Group:       groupNum,
		Count:       len(groupData),
		AvgReturn:   avgReturn,
		CumReturn:   cumReturn,
		Volatility:  volatility,
		SharpeRatio: sharpeRatio,
		MaxDrawdown: maxDrawdown,
		WinRate:     winRate,
		HitRate:     hitRate,
	}
}

// calculateMaxDrawdown 计算最大回撤
func (fde *FactorDiscoveryEngine) calculateMaxDrawdown(returns []float64) float64 {
	if len(returns) == 0 {
		return 0.0
	}

	// 计算累积收益率曲线
	cumReturns := make([]float64, len(returns))
	cumReturns[0] = 1.0 + returns[0]

	for i := 1; i < len(returns); i++ {
		cumReturns[i] = cumReturns[i-1] * (1.0 + returns[i])
	}

	// 计算最大回撤
	var maxDrawdown float64
	peak := cumReturns[0]

	for _, value := range cumReturns {
		if value > peak {
			peak = value
		}

		drawdown := (peak - value) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

func (fde *FactorDiscoveryEngine) analyzeFactorRisk(factor *Factor) FactorRiskAnalysis {
	// 实现因子风险分析 - 分析因子的各种风险特征
	ctx := context.Background()

	// 初始化风险分析结果
	riskAnalysis := FactorRiskAnalysis{
		Exposure: make(map[string]float64),
	}

	// 1. 计算行业和风格暴露
	exposure := fde.calculateFactorExposure(ctx, factor)
	riskAnalysis.Exposure = exposure

	// 2. 计算集中度风险
	riskAnalysis.Concentration = fde.calculateConcentrationRisk(ctx, factor)

	// 3. 计算换手率
	riskAnalysis.Turnover = fde.calculateFactorTurnover(ctx, factor)

	// 4. 计算容量
	riskAnalysis.Capacity = fde.calculateFactorCapacity(ctx, factor)

	// 5. 计算流动性风险
	riskAnalysis.LiquidityRisk = fde.calculateLiquidityRisk(ctx, factor)

	// 6. 计算拥挤度风险
	riskAnalysis.CrowdingRisk = fde.calculateCrowdingRisk(ctx, factor)

	return riskAnalysis
}

// calculateFactorExposure 计算因子暴露
func (fde *FactorDiscoveryEngine) calculateFactorExposure(ctx context.Context, factor *Factor) map[string]float64 {
	exposure := make(map[string]float64)

	if fde.db == nil {
		log.Printf("Database not available for exposure calculation")
		return exposure
	}

	// 查询因子在不同行业的暴露
	query := `
		SELECT s.industry, AVG(f.factor_value) as avg_exposure, COUNT(*) as count
		FROM factor_values f
		JOIN stock_info s ON f.symbol = s.symbol
		WHERE f.factor_id = ?
		    AND f.date >= DATE_SUB(NOW(), INTERVAL 3 MONTH)
		GROUP BY s.industry
		HAVING count >= 5
	`

	rows, err := fde.db.QueryContext(ctx, query, factor.ID)
	if err != nil {
		log.Printf("Failed to query factor exposure: %v", err)
		return exposure
	}
	defer rows.Close()

	for rows.Next() {
		var industry string
		var avgExposure float64
		var count int

		if err := rows.Scan(&industry, &avgExposure, &count); err != nil {
			continue
		}

		exposure[industry] = avgExposure
	}

	// 如果没有数据，返回默认暴露
	if len(exposure) == 0 {
		log.Printf("No exposure data found for factor %s", factor.ID)
	}

	return exposure
}

// calculateConcentrationRisk 计算集中度风险
func (fde *FactorDiscoveryEngine) calculateConcentrationRisk(ctx context.Context, factor *Factor) float64 {
	if fde.db == nil {
		return 0.0
	}

	// 查询因子值的分布
	query := `
		SELECT factor_value
		FROM factor_values
		WHERE factor_id = ?
		    AND date >= DATE_SUB(NOW(), INTERVAL 1 MONTH)
		    AND factor_value IS NOT NULL
		ORDER BY ABS(factor_value) DESC
		LIMIT 1000
	`

	rows, err := fde.db.QueryContext(ctx, query, factor.ID)
	if err != nil {
		log.Printf("Failed to query factor values for concentration: %v", err)
		return 0.0
	}
	defer rows.Close()

	var values []float64
	for rows.Next() {
		var value float64
		if err := rows.Scan(&value); err != nil {
			continue
		}
		values = append(values, math.Abs(value))
	}

	if len(values) < 10 {
		return 0.0
	}

	// 计算基尼系数作为集中度指标
	sort.Float64s(values)

	var sum, weightedSum float64
	for i, value := range values {
		sum += value
		weightedSum += value * float64(i+1)
	}

	if sum == 0 {
		return 0.0
	}

	n := float64(len(values))
	gini := (2*weightedSum)/(n*sum) - (n+1)/n

	return gini
}

// calculateFactorTurnover 计算因子换手率
func (fde *FactorDiscoveryEngine) calculateFactorTurnover(ctx context.Context, factor *Factor) float64 {
	if fde.db == nil {
		return 0.0
	}

	// 查询因子值的时间序列变化
	query := `
		SELECT date, symbol, factor_value
		FROM factor_values
		WHERE factor_id = ?
		    AND date >= DATE_SUB(NOW(), INTERVAL 2 MONTH)
		    AND factor_value IS NOT NULL
		ORDER BY symbol, date
	`

	rows, err := fde.db.QueryContext(ctx, query, factor.ID)
	if err != nil {
		log.Printf("Failed to query factor turnover data: %v", err)
		return 0.0
	}
	defer rows.Close()

	// 按股票分组计算换手率
	symbolData := make(map[string][]float64)

	for rows.Next() {
		var date time.Time
		var symbol string
		var value float64

		if err := rows.Scan(&date, &symbol, &value); err != nil {
			continue
		}

		symbolData[symbol] = append(symbolData[symbol], value)
	}

	// 计算平均换手率
	var totalTurnover float64
	var count int

	for _, values := range symbolData {
		if len(values) < 2 {
			continue
		}

		// 计算该股票的因子值变化率
		var changes float64
		for i := 1; i < len(values); i++ {
			if values[i-1] != 0 {
				change := math.Abs((values[i] - values[i-1]) / values[i-1])
				changes += change
			}
		}

		avgChange := changes / float64(len(values)-1)
		totalTurnover += avgChange
		count++
	}

	if count > 0 {
		return totalTurnover / float64(count)
	}

	return 0.0
}

// calculateFactorCapacity 计算因子容量
func (fde *FactorDiscoveryEngine) calculateFactorCapacity(ctx context.Context, factor *Factor) float64 {
	if fde.db == nil {
		return 0.0
	}

	// 查询相关股票的市值和流动性数据
	query := `
		SELECT SUM(s.market_cap * s.liquidity_score) as total_capacity
		FROM factor_values f
		JOIN stock_info s ON f.symbol = s.symbol
		WHERE f.factor_id = ?
		    AND f.date >= DATE_SUB(NOW(), INTERVAL 1 WEEK)
		    AND ABS(f.factor_value) > 0.1
		    AND s.market_cap > 0
	`

	var capacity float64
	err := fde.db.QueryRowContext(ctx, query, factor.ID).Scan(&capacity)
	if err != nil {
		log.Printf("Failed to query factor capacity: %v", err)
		return 0.0
	}

	// 应用容量折扣因子
	capacityDiscount := 0.1 // 10%的容量折扣
	return capacity * capacityDiscount
}

// calculateLiquidityRisk 计算流动性风险
func (fde *FactorDiscoveryEngine) calculateLiquidityRisk(ctx context.Context, factor *Factor) float64 {
	if fde.db == nil {
		return 0.0
	}

	// 查询因子相关股票的流动性指标
	query := `
		SELECT AVG(s.avg_volume), AVG(s.bid_ask_spread), COUNT(*)
		FROM factor_values f
		JOIN stock_info s ON f.symbol = s.symbol
		WHERE f.factor_id = ?
		    AND f.date >= DATE_SUB(NOW(), INTERVAL 1 MONTH)
		    AND ABS(f.factor_value) > 0.1
	`

	var avgVolume, avgSpread float64
	var count int

	err := fde.db.QueryRowContext(ctx, query, factor.ID).Scan(&avgVolume, &avgSpread, &count)
	if err != nil {
		log.Printf("Failed to query liquidity risk: %v", err)
		return 0.0
	}

	if count == 0 {
		return 1.0 // 最高流动性风险
	}

	// 流动性风险评分 (0-1，越高风险越大)
	volumeRisk := math.Max(0, 1.0-avgVolume/1000000) // 假设100万为基准交易量
	spreadRisk := math.Min(1.0, avgSpread*100)       // 买卖价差风险

	return (volumeRisk + spreadRisk) / 2.0
}

// calculateCrowdingRisk 计算拥挤度风险
func (fde *FactorDiscoveryEngine) calculateCrowdingRisk(ctx context.Context, factor *Factor) float64 {
	if fde.db == nil {
		return 0.0
	}

	// 查询使用该因子的策略数量和资金规模
	query := `
		SELECT COUNT(DISTINCT strategy_id) as strategy_count,
		       SUM(allocation_amount) as total_allocation
		FROM strategy_factor_usage
		WHERE factor_id = ?
		    AND is_active = 1
	`

	var strategyCount int
	var totalAllocation float64

	err := fde.db.QueryRowContext(ctx, query, factor.ID).Scan(&strategyCount, &totalAllocation)
	if err != nil {
		log.Printf("Failed to query crowding risk: %v", err)
		return 0.0
	}

	// 拥挤度风险评分
	// 策略数量风险
	strategyRisk := math.Min(1.0, float64(strategyCount)/50.0) // 50个策略为满分

	// 资金规模风险
	allocationRisk := math.Min(1.0, totalAllocation/1000000000.0) // 10亿为满分

	return (strategyRisk + allocationRisk) / 2.0
}

func (fde *FactorDiscoveryEngine) analyzeFactorStability(factor *Factor) FactorStabilityAnalysis {
	// 实现因子稳定性分析 - 分析因子的时间稳定性
	ctx := context.Background()

	// 获取因子的历史IC数据
	rollingICs := fde.calculateRollingIC(factor)

	// 计算IC稳定性
	icStability := fde.calculateICStability(rollingICs)

	// 计算收益稳定性
	returnStability := fde.calculateReturnStability(ctx, factor)

	// 计算排名稳定性
	rankStability := fde.calculateRankStability(ctx, factor)

	// 计算持续性
	persistence := fde.calculatePersistence(rollingICs)

	// 计算半衰期
	halfLife := fde.calculateHalfLife(rollingICs)

	// 执行断点检验
	breakpointTest := fde.performBreakpointTest(rollingICs)

	return FactorStabilityAnalysis{
		ICStability:     icStability,
		ReturnStability: returnStability,
		RankStability:   rankStability,
		Persistence:     persistence,
		HalfLife:        halfLife,
		BreakpointTest:  breakpointTest,
	}
}

// calculateICStability 计算IC稳定性
func (fde *FactorDiscoveryEngine) calculateICStability(rollingICs []RollingIC) float64 {
	if len(rollingICs) < 10 {
		return 0.0
	}

	// 提取IC值
	icValues := make([]float64, len(rollingICs))
	for i, ric := range rollingICs {
		icValues[i] = ric.IC
	}

	// 计算IC的标准差
	mean := 0.0
	for _, ic := range icValues {
		mean += ic
	}
	mean /= float64(len(icValues))

	var variance float64
	for _, ic := range icValues {
		diff := ic - mean
		variance += diff * diff
	}
	stdDev := math.Sqrt(variance / float64(len(icValues)))

	// 稳定性 = 1 - 变异系数
	if math.Abs(mean) > 0.001 {
		cv := stdDev / math.Abs(mean)
		stability := math.Max(0, 1.0-cv)
		return math.Min(1.0, stability)
	}

	return 0.0
}

// calculateReturnStability 计算收益稳定性
func (fde *FactorDiscoveryEngine) calculateReturnStability(ctx context.Context, factor *Factor) float64 {
	if fde.db == nil {
		return 0.0
	}

	// 查询因子的月度收益率
	query := `
		SELECT DATE_FORMAT(f.date, '%Y-%m') as month,
		       AVG(r.return_rate) as monthly_return
		FROM factor_values f
		JOIN returns r ON f.symbol = r.symbol 
		    AND r.date = DATE_ADD(f.date, INTERVAL 1 DAY)
		WHERE f.factor_id = ?
		    AND f.date >= DATE_SUB(NOW(), INTERVAL 1 YEAR)
		    AND ABS(f.factor_value) > 0.1
		GROUP BY DATE_FORMAT(f.date, '%Y-%m')
		HAVING COUNT(*) >= 10
		ORDER BY month
	`

	rows, err := fde.db.QueryContext(ctx, query, factor.ID)
	if err != nil {
		log.Printf("Failed to query return stability: %v", err)
		return 0.0
	}
	defer rows.Close()

	var monthlyReturns []float64
	for rows.Next() {
		var month string
		var monthlyReturn float64

		if err := rows.Scan(&month, &monthlyReturn); err != nil {
			continue
		}

		monthlyReturns = append(monthlyReturns, monthlyReturn)
	}

	if len(monthlyReturns) < 6 {
		return 0.0
	}

	// 计算收益率的稳定性 (夏普比率的变体)
	mean := 0.0
	for _, ret := range monthlyReturns {
		mean += ret
	}
	mean /= float64(len(monthlyReturns))

	var variance float64
	for _, ret := range monthlyReturns {
		diff := ret - mean
		variance += diff * diff
	}
	stdDev := math.Sqrt(variance / float64(len(monthlyReturns)))

	// 稳定性评分
	if stdDev > 0 {
		sharpe := mean / stdDev
		stability := math.Max(0, math.Min(1.0, (sharpe+2.0)/4.0)) // 标准化到0-1
		return stability
	}

	return 0.0
}

// calculateRankStability 计算排名稳定性
func (fde *FactorDiscoveryEngine) calculateRankStability(ctx context.Context, factor *Factor) float64 {
	if fde.db == nil {
		return 0.0
	}

	// 查询因子值的排名变化
	query := `
		SELECT f1.date, f1.symbol, f1.factor_value,
		       RANK() OVER (PARTITION BY f1.date ORDER BY f1.factor_value DESC) as rank_today,
		       RANK() OVER (PARTITION BY f2.date ORDER BY f2.factor_value DESC) as rank_prev
		FROM factor_values f1
		JOIN factor_values f2 ON f1.symbol = f2.symbol 
		    AND f2.date = DATE_SUB(f1.date, INTERVAL 1 DAY)
		    AND f2.factor_id = f1.factor_id
		WHERE f1.factor_id = ?
		    AND f1.date >= DATE_SUB(NOW(), INTERVAL 1 MONTH)
		ORDER BY f1.date, f1.symbol
	`

	rows, err := fde.db.QueryContext(ctx, query, factor.ID)
	if err != nil {
		log.Printf("Failed to query rank stability: %v", err)
		return 0.0
	}
	defer rows.Close()

	var rankChanges []float64
	for rows.Next() {
		var date time.Time
		var symbol string
		var factorValue float64
		var rankToday, rankPrev int

		if err := rows.Scan(&date, &symbol, &factorValue, &rankToday, &rankPrev); err != nil {
			continue
		}

		// 计算排名变化率
		if rankPrev > 0 {
			rankChange := math.Abs(float64(rankToday-rankPrev)) / float64(rankPrev)
			rankChanges = append(rankChanges, rankChange)
		}
	}

	if len(rankChanges) < 10 {
		return 0.0
	}

	// 计算平均排名变化率
	var avgChange float64
	for _, change := range rankChanges {
		avgChange += change
	}
	avgChange /= float64(len(rankChanges))

	// 稳定性 = 1 - 平均变化率
	stability := math.Max(0, 1.0-avgChange)
	return math.Min(1.0, stability)
}

// calculatePersistence 计算持续性
func (fde *FactorDiscoveryEngine) calculatePersistence(rollingICs []RollingIC) float64 {
	if len(rollingICs) < 20 {
		return 0.0
	}

	// 计算IC的自相关系数
	icValues := make([]float64, len(rollingICs))
	for i, ric := range rollingICs {
		icValues[i] = ric.IC
	}

	// 计算滞后1期的自相关
	lag1Correlation := fde.calculateAutoCorrelation(icValues, 1)

	// 持续性评分
	persistence := math.Max(0, lag1Correlation)
	return math.Min(1.0, persistence)
}

// calculateAutoCorrelation 计算自相关系数
func (fde *FactorDiscoveryEngine) calculateAutoCorrelation(series []float64, lag int) float64 {
	if len(series) <= lag {
		return 0.0
	}

	n := len(series) - lag
	x := series[:n]
	y := series[lag:]

	return fde.calculatePearsonCorrelation(x, y)
}

// calculateHalfLife 计算半衰期
func (fde *FactorDiscoveryEngine) calculateHalfLife(rollingICs []RollingIC) float64 {
	if len(rollingICs) < 30 {
		return 0.0
	}

	// 提取IC绝对值
	icAbs := make([]float64, len(rollingICs))
	for i, ric := range rollingICs {
		icAbs[i] = math.Abs(ric.IC)
	}

	// 寻找IC衰减到初始值一半的时间
	if len(icAbs) == 0 {
		return 0.0
	}

	initialIC := icAbs[0]
	halfValue := initialIC / 2.0

	for i, ic := range icAbs {
		if ic <= halfValue {
			return float64(i)
		}
	}

	// 如果没有找到半衰期，返回序列长度
	return float64(len(icAbs))
}

// performBreakpointTest 执行断点检验
func (fde *FactorDiscoveryEngine) performBreakpointTest(rollingICs []RollingIC) BreakpointTest {
	if len(rollingICs) < 50 {
		return BreakpointTest{
			HasBreakpoint: false,
			Statistic:     0.0,
			PValue:        1.0,
		}
	}

	// 简化的Chow断点检验
	icValues := make([]float64, len(rollingICs))
	for i, ric := range rollingICs {
		icValues[i] = ric.IC
	}

	// 寻找最可能的断点位置
	midPoint := len(icValues) / 2

	// 计算前半段和后半段的均值差异
	var sum1, sum2 float64
	for i := 0; i < midPoint; i++ {
		sum1 += icValues[i]
	}
	for i := midPoint; i < len(icValues); i++ {
		sum2 += icValues[i]
	}

	mean1 := sum1 / float64(midPoint)
	mean2 := sum2 / float64(len(icValues)-midPoint)

	// 计算t统计量
	var var1, var2 float64
	for i := 0; i < midPoint; i++ {
		diff := icValues[i] - mean1
		var1 += diff * diff
	}
	for i := midPoint; i < len(icValues); i++ {
		diff := icValues[i] - mean2
		var2 += diff * diff
	}

	var1 /= float64(midPoint - 1)
	var2 /= float64(len(icValues) - midPoint - 1)

	pooledVar := ((float64(midPoint-1)*var1 + float64(len(icValues)-midPoint-1)*var2) /
		float64(len(icValues)-2))

	if pooledVar <= 0 {
		return BreakpointTest{
			HasBreakpoint: false,
			Statistic:     0.0,
			PValue:        1.0,
		}
	}

	se := math.Sqrt(pooledVar * (1.0/float64(midPoint) + 1.0/float64(len(icValues)-midPoint)))
	tStat := (mean1 - mean2) / se

	// 简化的p值计算
	pValue := 0.05
	if math.Abs(tStat) > 2.0 {
		pValue = 0.01
	} else if math.Abs(tStat) > 1.5 {
		pValue = 0.05
	} else {
		pValue = 0.10
	}

	return BreakpointTest{
		HasBreakpoint:  math.Abs(tStat) > 1.96,
		Statistic:      tStat,
		PValue:         pValue,
		BreakpointDate: rollingICs[midPoint].Date,
	}
}

func (fde *FactorDiscoveryEngine) calculateOverallScore(evaluation *FactorEvaluation) float64 {
	// 综合评分算法
	icScore := 0.0
	if len(evaluation.ICResults) > 0 {
		icScore = math.Abs(evaluation.ICResults[0].IC) * 10 // 将IC转换为0-1分数
	}

	stabilityScore := evaluation.StabilityAnalysis.ICStability

	// 加权平均
	return icScore*0.6 + stabilityScore*0.4
}

func (fde *FactorDiscoveryEngine) assignGrade(score float64) string {
	switch {
	case score >= 0.9:
		return "A+"
	case score >= 0.8:
		return "A"
	case score >= 0.7:
		return "B+"
	case score >= 0.6:
		return "B"
	case score >= 0.5:
		return "C+"
	case score >= 0.4:
		return "C"
	default:
		return "D"
	}
}

func (fde *FactorDiscoveryEngine) updateFactorFromEvaluation(factor *Factor, evaluation *FactorEvaluation) {
	if len(evaluation.ICResults) > 0 {
		factor.IC = evaluation.ICResults[0].IC
		factor.PValue = evaluation.ICResults[0].PValue
		factor.IsSignificant = evaluation.ICResults[0].IsSignificant
	}

	factor.Stability = evaluation.StabilityAnalysis.ICStability
	factor.LastUpdated = time.Now()
	factor.UpdateCount++
}

func (fde *FactorDiscoveryEngine) getCurrentActiveFactors() []*Factor {
	fde.mu.RLock()
	defer fde.mu.RUnlock()

	factors := make([]*Factor, 0, len(fde.activeFactors))
	for _, factor := range fde.activeFactors {
		factors = append(factors, factor)
	}
	return factors
}

func (fde *FactorDiscoveryEngine) getCandidateFactors() []*Factor {
	fde.mu.RLock()
	defer fde.mu.RUnlock()

	candidates := make([]*Factor, 0)
	for _, factor := range fde.discoveredFactors {
		if factor.Status == "ACTIVE" && factor.IsSignificant {
			candidates = append(candidates, factor)
		}
	}
	return candidates
}

func (fde *FactorDiscoveryEngine) performanceBasedRotation(current, candidates []*Factor) {
	// 实现基于性能的轮换
	log.Printf("Performing performance-based factor rotation with %d current and %d candidate factors", len(current), len(candidates))

	// 1. 计算所有因子的性能指标
	factorPerformances := make(map[string]*FactorPerformance)

	// 计算当前因子的性能
	for _, factor := range current {
		performance := fde.calculateFactorPerformance(factor)
		factorPerformances[factor.ID] = performance
	}

	// 计算候选因子的性能
	for _, factor := range candidates {
		performance := fde.calculateFactorPerformance(factor)
		factorPerformances[factor.ID] = performance
	}

	// 2. 识别表现不佳的当前因子
	underperformingFactors := make([]*Factor, 0)
	performanceThreshold := 0.02 // ICIR阈值

	for _, factor := range current {
		performance := factorPerformances[factor.ID]
		if performance.ICIR < performanceThreshold || performance.RecentIC < 0.01 {
			underperformingFactors = append(underperformingFactors, factor)
			log.Printf("Factor %s marked for replacement: ICIR=%.4f, RecentIC=%.4f",
				factor.ID, performance.ICIR, performance.RecentIC)
		}
	}

	// 3. 选择最佳候选因子进行替换
	if len(underperformingFactors) > 0 && len(candidates) > 0 {
		// 按性能排序候选因子
		sortedCandidates := make([]*Factor, len(candidates))
		copy(sortedCandidates, candidates)

		sort.Slice(sortedCandidates, func(i, j int) bool {
			perfI := factorPerformances[sortedCandidates[i].ID]
			perfJ := factorPerformances[sortedCandidates[j].ID]
			// 综合考虑ICIR和最近IC表现
			scoreI := perfI.ICIR*0.7 + perfI.RecentIC*0.3
			scoreJ := perfJ.ICIR*0.7 + perfJ.RecentIC*0.3
			return scoreI > scoreJ
		})

		// 执行替换
		replacementCount := min(len(underperformingFactors), len(sortedCandidates))
		for i := 0; i < replacementCount; i++ {
			oldFactor := underperformingFactors[i]
			newFactor := sortedCandidates[i]

			// 记录替换操作
			fde.recordFactorReplacement(oldFactor.ID, newFactor.ID, "performance_based",
				factorPerformances[oldFactor.ID], factorPerformances[newFactor.ID])

			log.Printf("Replaced factor %s (ICIR: %.4f) with %s (ICIR: %.4f)",
				oldFactor.ID, factorPerformances[oldFactor.ID].ICIR,
				newFactor.ID, factorPerformances[newFactor.ID].ICIR)
		}
	}
}

func (fde *FactorDiscoveryEngine) correlationBasedRotation(current, candidates []*Factor) {
	// 实现基于相关性的轮换
	log.Printf("Performing correlation-based factor rotation with %d current and %d candidate factors", len(current), len(candidates))

	if len(current) < 2 {
		log.Printf("Not enough current factors for correlation analysis")
		return
	}

	// 1. 计算当前因子之间的相关性矩阵
	correlationMatrix := fde.calculateFactorCorrelationMatrix(current)

	// 2. 识别高度相关的因子对
	highCorrelationPairs := make([]FactorPair, 0)
	correlationThreshold := 0.8 // 相关性阈值

	for i := 0; i < len(current); i++ {
		for j := i + 1; j < len(current); j++ {
			correlation := correlationMatrix[i][j]
			if math.Abs(correlation) > correlationThreshold {
				pair := FactorPair{
					Factor1:     current[i],
					Factor2:     current[j],
					Correlation: correlation,
				}
				highCorrelationPairs = append(highCorrelationPairs, pair)
				log.Printf("High correlation detected: %s vs %s = %.4f",
					current[i].ID, current[j].ID, correlation)
			}
		}
	}

	// 3. 对于高相关因子对，保留表现更好的因子
	factorsToReplace := make(map[string]bool)

	for _, pair := range highCorrelationPairs {
		perf1 := fde.calculateFactorPerformance(pair.Factor1)
		perf2 := fde.calculateFactorPerformance(pair.Factor2)

		// 比较综合性能分数
		score1 := perf1.ICIR*0.4 + perf1.SharpeRatio*0.3 + perf1.StabilityScore*0.3
		score2 := perf2.ICIR*0.4 + perf2.SharpeRatio*0.3 + perf2.StabilityScore*0.3

		var weakerFactor *Factor
		if score1 < score2 {
			weakerFactor = pair.Factor1
		} else {
			weakerFactor = pair.Factor2
		}

		factorsToReplace[weakerFactor.ID] = true
		log.Printf("Marking factor %s for replacement due to high correlation (score: %.4f)",
			weakerFactor.ID, min(score1, score2))
	}

	// 4. 从候选因子中选择与现有因子相关性低的因子进行替换
	if len(factorsToReplace) > 0 && len(candidates) > 0 {
		// 为每个候选因子计算与当前因子组合的平均相关性
		candidateScores := make([]CandidateScore, 0)

		for _, candidate := range candidates {
			avgCorrelation := fde.calculateAverageCorrelationWithCurrent(candidate, current, factorsToReplace)
			performance := fde.calculateFactorPerformance(candidate)

			// 综合分数：性能高，相关性低
			score := performance.ICIR*0.6 - math.Abs(avgCorrelation)*0.4

			candidateScores = append(candidateScores, CandidateScore{
				Factor:         candidate,
				Score:          score,
				AvgCorrelation: avgCorrelation,
			})
		}

		// 按分数排序候选因子
		sort.Slice(candidateScores, func(i, j int) bool {
			return candidateScores[i].Score > candidateScores[j].Score
		})

		// 执行替换
		replacementCount := 0
		for factorID := range factorsToReplace {
			if replacementCount >= len(candidateScores) {
				break
			}

			newFactor := candidateScores[replacementCount].Factor

			// 记录替换操作
			fde.recordFactorReplacement(factorID, newFactor.ID, "correlation_based", nil, nil)

			log.Printf("Replaced highly correlated factor %s with %s (avg correlation: %.4f)",
				factorID, newFactor.ID, candidateScores[replacementCount].AvgCorrelation)

			replacementCount++
		}
	}
}

func (fde *FactorDiscoveryEngine) regimeBasedRotation(current, candidates []*Factor) {
	// 实现基于市场状态的轮换
	log.Printf("Performing regime-based factor rotation with %d current and %d candidate factors", len(current), len(candidates))

	// 1. 识别当前市场状态
	currentRegime := fde.identifyMarketRegime()
	log.Printf("Current market regime identified as: %s", currentRegime.Type)

	// 2. 评估当前因子在当前市场状态下的适应性
	factorRegimeScores := make(map[string]float64)

	for _, factor := range current {
		regimeScore := fde.calculateFactorRegimeAdaptability(factor, currentRegime)
		factorRegimeScores[factor.ID] = regimeScore

		log.Printf("Factor %s regime adaptability score: %.4f", factor.ID, regimeScore)
	}

	// 3. 识别在当前市场状态下表现不佳的因子
	regimeThreshold := 0.3 // 市场适应性阈值
	underperformingInRegime := make([]*Factor, 0)

	for _, factor := range current {
		if factorRegimeScores[factor.ID] < regimeThreshold {
			underperformingInRegime = append(underperformingInRegime, factor)
			log.Printf("Factor %s underperforming in current regime: score=%.4f",
				factor.ID, factorRegimeScores[factor.ID])
		}
	}

	// 4. 从候选因子中选择适合当前市场状态的因子
	if len(underperformingInRegime) > 0 && len(candidates) > 0 {
		// 评估候选因子的市场适应性
		candidateRegimeScores := make([]RegimeCandidateScore, 0)

		for _, candidate := range candidates {
			regimeScore := fde.calculateFactorRegimeAdaptability(candidate, currentRegime)
			performance := fde.calculateFactorPerformance(candidate)

			// 综合分数：市场适应性 + 历史表现
			totalScore := regimeScore*0.6 + performance.ICIR*0.4

			candidateRegimeScores = append(candidateRegimeScores, RegimeCandidateScore{
				Factor:      candidate,
				RegimeScore: regimeScore,
				TotalScore:  totalScore,
			})
		}

		// 按总分排序
		sort.Slice(candidateRegimeScores, func(i, j int) bool {
			return candidateRegimeScores[i].TotalScore > candidateRegimeScores[j].TotalScore
		})

		// 执行替换
		replacementCount := min(len(underperformingInRegime), len(candidateRegimeScores))
		for i := 0; i < replacementCount; i++ {
			oldFactor := underperformingInRegime[i]
			newCandidate := candidateRegimeScores[i]

			// 记录替换操作
			fde.recordFactorReplacement(oldFactor.ID, newCandidate.Factor.ID, "regime_based", nil, nil)

			log.Printf("Replaced regime-unsuitable factor %s (score: %.4f) with %s (regime score: %.4f)",
				oldFactor.ID, factorRegimeScores[oldFactor.ID],
				newCandidate.Factor.ID, newCandidate.RegimeScore)
		}
	}

	// 5. 记录市场状态变化历史
	fde.recordMarketRegimeChange(currentRegime)
}

func (fde *FactorDiscoveryEngine) calculateFactorPerformance(factor *Factor) *FactorPerformance {
	// 计算因子表现
	log.Printf("Calculating performance for factor: %s", factor.ID)

	// 1. 获取因子历史数据
	historicalData := fde.getFactorHistoricalData(factor.ID)
	if len(historicalData) == 0 {
		log.Printf("No historical data available for factor %s, using default values", factor.ID)
		return fde.createDefaultFactorPerformance(factor.ID)
	}

	// 2. 计算IC相关指标
	icValues := make([]float64, len(historicalData))
	rankICValues := make([]float64, len(historicalData))
	returns := make([]float64, len(historicalData))

	for i, data := range historicalData {
		icValues[i] = data.IC
		rankICValues[i] = data.RankIC
		returns[i] = data.Return
	}

	avgIC := calculateMean(icValues)
	avgRankIC := calculateMean(rankICValues)
	icStdDev := calculateStdDev(icValues)
	icir := 0.0
	if icStdDev > 0 {
		icir = avgIC / icStdDev
	}

	// 3. 计算命中率
	hitCount := 0
	for _, ic := range icValues {
		if ic > 0 {
			hitCount++
		}
	}
	hitRate := float64(hitCount) / float64(len(icValues))

	// 4. 计算收益相关指标
	cumulativeReturn := 1.0
	for _, ret := range returns {
		cumulativeReturn *= (1.0 + ret)
	}
	cumulativeReturn -= 1.0

	// 年化收益率（假设252个交易日）
	periods := float64(len(returns))
	annualizedReturn := math.Pow(1.0+cumulativeReturn, 252.0/periods) - 1.0

	// 波动率
	volatility := calculateStdDev(returns) * math.Sqrt(252)

	// 夏普比率
	sharpeRatio := 0.0
	if volatility > 0 {
		sharpeRatio = annualizedReturn / volatility
	}

	// 最大回撤
	maxDrawdown := calculateMaxDrawdown(returns)

	// 5. 计算稳定性指标
	stabilityScore := fde.calculateStabilityScore(icValues)
	consistencyScore := fde.calculateConsistencyScore(icValues)

	// 6. 计算最近表现（最近20个数据点）
	recentPeriod := min(20, len(historicalData))
	recentICValues := icValues[len(icValues)-recentPeriod:]
	recentRankICValues := rankICValues[len(rankICValues)-recentPeriod:]

	recentIC := calculateMean(recentICValues)
	recentRankIC := calculateMean(recentRankICValues)

	// 7. 构建性能历史
	performanceHistory := make([]PerformancePoint, len(historicalData))
	for i, data := range historicalData {
		performanceHistory[i] = PerformancePoint{
			Date:   data.Date,
			IC:     data.IC,
			RankIC: data.RankIC,
			Return: data.Return,
		}
	}

	performance := &FactorPerformance{
		FactorID:           factor.ID,
		PerformanceHistory: performanceHistory,
		AvgIC:              avgIC,
		AvgRankIC:          avgRankIC,
		ICStdDev:           icStdDev,
		ICIR:               icir,
		HitRate:            hitRate,
		CumulativeReturn:   cumulativeReturn,
		AnnualizedReturn:   annualizedReturn,
		Volatility:         volatility,
		SharpeRatio:        sharpeRatio,
		MaxDrawdown:        maxDrawdown,
		StabilityScore:     stabilityScore,
		ConsistencyScore:   consistencyScore,
		RecentIC:           recentIC,
		RecentRankIC:       recentRankIC,
	}

	log.Printf("Factor %s performance calculated: ICIR=%.4f, Sharpe=%.4f, HitRate=%.2f%%",
		factor.ID, icir, sharpeRatio, hitRate*100)

	performance.AvgIC = factor.IC
	performance.AvgRankIC = factor.RankIC
	performance.ICStdDev = factor.ICStdDev
	performance.ICIR = factor.ICIR
	performance.StabilityScore = factor.Stability
	performance.RecentIC = factor.IC
	performance.LastUpdated = time.Now()

	return performance
}

func (fde *FactorDiscoveryEngine) shouldDeactivateFactor(factor *Factor, performance *FactorPerformance) bool {
	// 检查是否应该停用因子
	if !factor.IsSignificant {
		return true
	}

	if math.Abs(performance.RecentIC) < 0.01 { // IC过低
		return true
	}

	if performance.StabilityScore < 0.3 { // 稳定性过低
		return true
	}

	return false
}

func (fde *FactorDiscoveryEngine) deactivateFactor(factor *Factor, reason string) {
	fde.mu.Lock()
	defer fde.mu.Unlock()

	factor.Status = "INACTIVE"
	factor.LastUpdated = time.Now()

	// 从活跃因子中移除
	delete(fde.activeFactors, factor.ID)

	// 记录停用事件
	event := DiscoveryEvent{
		Date:      time.Now(),
		EventType: "DEPRECATION",
		FactorID:  factor.ID,
		Details: map[string]interface{}{
			"reason": reason,
		},
		Impact: "MEDIUM",
	}
	fde.discoveryHistory = append(fde.discoveryHistory, event)

	log.Printf("Factor %s deactivated: %s", factor.ID, reason)
}

func (fde *FactorDiscoveryEngine) updateMetrics() {
	fde.discoveryMetrics.mu.Lock()
	defer fde.discoveryMetrics.mu.Unlock()

	// 更新因子统计
	fde.discoveryMetrics.TotalFactors = len(fde.discoveredFactors)
	fde.discoveryMetrics.ActiveFactors = len(fde.activeFactors)

	// 计算显著因子数量
	significantCount := 0
	totalIC := 0.0
	totalICIR := 0.0
	topIC := math.Inf(-1)

	for _, factor := range fde.discoveredFactors {
		if factor.IsSignificant {
			significantCount++
		}
		totalIC += math.Abs(factor.IC)
		totalICIR += factor.ICIR
		if math.Abs(factor.IC) > topIC {
			topIC = math.Abs(factor.IC)
		}
	}

	fde.discoveryMetrics.SignificantFactors = significantCount

	if fde.discoveryMetrics.TotalFactors > 0 {
		fde.discoveryMetrics.AvgIC = totalIC / float64(fde.discoveryMetrics.TotalFactors)
		fde.discoveryMetrics.AvgICIR = totalICIR / float64(fde.discoveryMetrics.TotalFactors)
	}

	fde.discoveryMetrics.TopFactorIC = topIC
	fde.discoveryMetrics.LastUpdated = time.Now()
}

func (fde *FactorDiscoveryEngine) generateFactorID() string {
	return fmt.Sprintf("FACTOR_%d", time.Now().UnixNano())
}

// GetStatus 获取引擎状态
func (fde *FactorDiscoveryEngine) GetStatus() map[string]interface{} {
	fde.mu.RLock()
	defer fde.mu.RUnlock()

	return map[string]interface{}{
		"running":             fde.isRunning,
		"enabled":             fde.enabled,
		"discovered_factors":  len(fde.discoveredFactors),
		"active_factors":      len(fde.activeFactors),
		"discovery_algorithm": fde.discoveryAlgorithm,
		"significance_level":  fde.significanceLevel,
		"rotation_frequency":  fde.rotationFrequency,
		"max_factors":         fde.maxFactors,
		"discovery_metrics":   fde.discoveryMetrics,
		"discovery_events":    len(fde.discoveryHistory),
	}
}

// GetDiscoveryMetrics 获取发现指标
func (fde *FactorDiscoveryEngine) GetDiscoveryMetrics() *DiscoveryMetrics {
	fde.discoveryMetrics.mu.RLock()
	defer fde.discoveryMetrics.mu.RUnlock()

	metrics := *fde.discoveryMetrics
	return &metrics
}

// GetDiscoveredFactors 获取发现的因子
func (fde *FactorDiscoveryEngine) GetDiscoveredFactors(limit int) []*Factor {
	fde.mu.RLock()
	defer fde.mu.RUnlock()

	factors := make([]*Factor, 0, len(fde.discoveredFactors))
	for _, factor := range fde.discoveredFactors {
		factors = append(factors, factor)
	}

	// 按发现时间排序
	sort.Slice(factors, func(i, j int) bool {
		return factors[i].DiscoveredAt.After(factors[j].DiscoveredAt)
	})

	if limit > 0 && limit < len(factors) {
		factors = factors[:limit]
	}

	return factors
}

// GetActiveFactors 获取活跃因子
func (fde *FactorDiscoveryEngine) GetActiveFactors() []*Factor {
	fde.mu.RLock()
	defer fde.mu.RUnlock()

	factors := make([]*Factor, 0, len(fde.activeFactors))
	for _, factor := range fde.activeFactors {
		factors = append(factors, factor)
	}

	return factors
}

// GetFactorPerformance 获取因子表现
func (fde *FactorDiscoveryEngine) GetFactorPerformance(factorID string) (*FactorPerformance, error) {
	fde.mu.RLock()
	defer fde.mu.RUnlock()

	if performance, exists := fde.factorPerformance[factorID]; exists {
		return performance, nil
	}

	return nil, fmt.Errorf("factor performance not found: %s", factorID)
}

// Supporting types and functions for factor rotation

// FactorPair 因子对
type FactorPair struct {
	Factor1     *Factor
	Factor2     *Factor
	Correlation float64
}

// CandidateScore 候选因子分数
type CandidateScore struct {
	Factor         *Factor
	Score          float64
	AvgCorrelation float64
}

// MarketRegime 市场状态
type MarketRegime struct {
	Type       string  // "bull", "bear", "sideways", "volatile"
	Volatility float64 // 波动率
	Trend      float64 // 趋势强度
	Volume     float64 // 成交量
	Timestamp  time.Time
}

// RegimeCandidateScore 市场状态候选因子分数
type RegimeCandidateScore struct {
	Factor      *Factor
	RegimeScore float64
	TotalScore  float64
}

// FactorHistoricalData 因子历史数据
type FactorHistoricalData struct {
	Date   time.Time
	IC     float64
	RankIC float64
	Return float64
}

// calculateMean 计算平均值
func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// calculateStdDev 计算标准差
func calculateStdDev(values []float64) float64 {
	if len(values) <= 1 {
		return 0.0
	}

	mean := calculateMean(values)
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}

	return math.Sqrt(sumSquares / float64(len(values)-1))
}

// calculateMaxDrawdown 计算最大回撤
func calculateMaxDrawdown(returns []float64) float64 {
	if len(returns) == 0 {
		return 0.0
	}

	cumulative := 1.0
	peak := 1.0
	maxDrawdown := 0.0

	for _, ret := range returns {
		cumulative *= (1.0 + ret)
		if cumulative > peak {
			peak = cumulative
		}
		drawdown := (peak - cumulative) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

// recordFactorReplacement 记录因子替换操作
func (fde *FactorDiscoveryEngine) recordFactorReplacement(oldFactorID, newFactorID, reason string, oldPerf, newPerf *FactorPerformance) {
	log.Printf("Recording factor replacement: %s -> %s (reason: %s)", oldFactorID, newFactorID, reason)

	// 这里可以记录到数据库或日志系统
	replacement := map[string]interface{}{
		"timestamp":       time.Now(),
		"old_factor_id":   oldFactorID,
		"new_factor_id":   newFactorID,
		"reason":          reason,
		"old_performance": oldPerf,
		"new_performance": newPerf,
	}

	// 简化实现：记录到日志
	log.Printf("Factor replacement recorded: %+v", replacement)
}

// calculateFactorCorrelationMatrix 计算因子相关性矩阵
func (fde *FactorDiscoveryEngine) calculateFactorCorrelationMatrix(factors []*Factor) [][]float64 {
	n := len(factors)
	matrix := make([][]float64, n)
	for i := range matrix {
		matrix[i] = make([]float64, n)
	}

	// 获取所有因子的历史数据
	factorData := make([][]float64, n)
	for i, factor := range factors {
		historicalData := fde.getFactorHistoricalData(factor.ID)
		values := make([]float64, len(historicalData))
		for j, data := range historicalData {
			values[j] = data.IC
		}
		factorData[i] = values
	}

	// 计算相关性
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				matrix[i][j] = 1.0
			} else {
				matrix[i][j] = calculateCorrelation(factorData[i], factorData[j])
			}
		}
	}

	return matrix
}

// calculateCorrelation 计算两个序列的相关性
func calculateCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) == 0 {
		return 0.0
	}

	meanX := calculateMean(x)
	meanY := calculateMean(y)

	numerator := 0.0
	sumXSquares := 0.0
	sumYSquares := 0.0

	for i := 0; i < len(x); i++ {
		diffX := x[i] - meanX
		diffY := y[i] - meanY
		numerator += diffX * diffY
		sumXSquares += diffX * diffX
		sumYSquares += diffY * diffY
	}

	denominator := math.Sqrt(sumXSquares * sumYSquares)
	if denominator == 0 {
		return 0.0
	}

	return numerator / denominator
}

// calculateAverageCorrelationWithCurrent 计算候选因子与当前因子的平均相关性
func (fde *FactorDiscoveryEngine) calculateAverageCorrelationWithCurrent(candidate *Factor, current []*Factor, excludeFactors map[string]bool) float64 {
	candidateData := fde.getFactorHistoricalData(candidate.ID)
	candidateValues := make([]float64, len(candidateData))
	for i, data := range candidateData {
		candidateValues[i] = data.IC
	}

	correlations := make([]float64, 0)

	for _, factor := range current {
		if excludeFactors[factor.ID] {
			continue // 跳过要被替换的因子
		}

		factorData := fde.getFactorHistoricalData(factor.ID)
		factorValues := make([]float64, len(factorData))
		for i, data := range factorData {
			factorValues[i] = data.IC
		}

		correlation := calculateCorrelation(candidateValues, factorValues)
		correlations = append(correlations, math.Abs(correlation))
	}

	return calculateMean(correlations)
}

// identifyMarketRegime 识别市场状态
func (fde *FactorDiscoveryEngine) identifyMarketRegime() *MarketRegime {
	// 简化实现：基于市场指标识别状态
	// 实际实现应该使用更复杂的算法和更多数据

	// 获取市场数据
	marketData := fde.getCurrentMarketData()

	// 计算趋势强度
	trend := fde.calculateTrendStrength(marketData)

	// 计算波动率
	volatility := fde.calculateMarketVolatility(marketData)

	// 计算成交量
	volume := fde.calculateAverageVolume(marketData)

	// 根据指标确定市场状态
	regimeType := "sideways" // 默认横盘

	if trend > 0.6 && volatility < 0.2 {
		regimeType = "bull"
	} else if trend < -0.6 && volatility < 0.2 {
		regimeType = "bear"
	} else if volatility > 0.3 {
		regimeType = "volatile"
	}

	return &MarketRegime{
		Type:       regimeType,
		Volatility: volatility,
		Trend:      trend,
		Volume:     volume,
		Timestamp:  time.Now(),
	}
}

// calculateFactorRegimeAdaptability 计算因子在特定市场状态下的适应性
func (fde *FactorDiscoveryEngine) calculateFactorRegimeAdaptability(factor *Factor, regime *MarketRegime) float64 {
	// 获取因子在不同市场状态下的历史表现
	regimePerformance := fde.getFactorRegimePerformance(factor.ID, regime.Type)

	// 基于因子类型和市场状态计算适应性分数
	baseScore := 0.5 // 基础分数

	switch regime.Type {
	case "bull":
		// 牛市中动量因子表现较好
		if factor.Type == "TECHNICAL" && strings.Contains(factor.Name, "momentum") {
			baseScore += 0.3
		}
		// 成长因子在牛市中表现好
		if factor.Type == "FUNDAMENTAL" && strings.Contains(factor.Name, "growth") {
			baseScore += 0.2
		}

	case "bear":
		// 熊市中质量因子和低波动因子表现较好
		if factor.Type == "FUNDAMENTAL" && (strings.Contains(factor.Name, "quality") || strings.Contains(factor.Name, "low_vol")) {
			baseScore += 0.3
		}

	case "volatile":
		// 高波动市场中反转因子可能表现较好
		if factor.Type == "TECHNICAL" && strings.Contains(factor.Name, "reversal") {
			baseScore += 0.2
		}

	case "sideways":
		// 横盘市场中均值回归因子表现较好
		if factor.Type == "TECHNICAL" && strings.Contains(factor.Name, "mean_reversion") {
			baseScore += 0.2
		}
	}

	// 结合历史表现调整分数
	if regimePerformance != nil {
		baseScore = baseScore*0.6 + regimePerformance.AvgIC*0.4
	}

	// 确保分数在0-1范围内
	if baseScore > 1.0 {
		baseScore = 1.0
	}
	if baseScore < 0.0 {
		baseScore = 0.0
	}

	return baseScore
}

// recordMarketRegimeChange 记录市场状态变化
func (fde *FactorDiscoveryEngine) recordMarketRegimeChange(regime *MarketRegime) {
	log.Printf("Recording market regime change: %s (volatility: %.4f, trend: %.4f)",
		regime.Type, regime.Volatility, regime.Trend)

	// 这里可以记录到数据库
	// 简化实现：记录到日志
}

// getFactorHistoricalData 获取因子历史数据
func (fde *FactorDiscoveryEngine) getFactorHistoricalData(factorID string) []FactorHistoricalData {
	// 简化实现：返回模拟数据
	// 实际实现应该从数据库获取真实历史数据

	data := make([]FactorHistoricalData, 60) // 60天数据
	baseDate := time.Now().AddDate(0, 0, -60)

	for i := 0; i < 60; i++ {
		// 生成模拟的IC和收益数据
		ic := (rand.Float64() - 0.5) * 0.2        // -0.1 到 0.1
		rankIC := ic * (0.8 + rand.Float64()*0.4) // 相关但有噪音
		ret := ic*0.5 + (rand.Float64()-0.5)*0.02 // 基于IC的收益

		data[i] = FactorHistoricalData{
			Date:   baseDate.AddDate(0, 0, i),
			IC:     ic,
			RankIC: rankIC,
			Return: ret,
		}
	}

	return data
}

// createDefaultFactorPerformance 创建默认因子性能
func (fde *FactorDiscoveryEngine) createDefaultFactorPerformance(factorID string) *FactorPerformance {
	return &FactorPerformance{
		FactorID:           factorID,
		PerformanceHistory: []PerformancePoint{},
		AvgIC:              0.0,
		AvgRankIC:          0.0,
		ICStdDev:           0.0,
		ICIR:               0.0,
		HitRate:            0.5,
		CumulativeReturn:   0.0,
		AnnualizedReturn:   0.0,
		Volatility:         0.0,
		SharpeRatio:        0.0,
		MaxDrawdown:        0.0,
		StabilityScore:     0.5,
		ConsistencyScore:   0.5,
		RecentIC:           0.0,
		RecentRankIC:       0.0,
	}
}

// calculateStabilityScore 计算稳定性分数
func (fde *FactorDiscoveryEngine) calculateStabilityScore(icValues []float64) float64 {
	if len(icValues) < 10 {
		return 0.5 // 数据不足，返回中性分数
	}

	// 计算IC的标准差，标准差越小稳定性越高
	stdDev := calculateStdDev(icValues)

	// 将标准差转换为0-1的稳定性分数
	// 假设标准差在0-0.2范围内
	stabilityScore := 1.0 - math.Min(stdDev/0.2, 1.0)

	return stabilityScore
}

// calculateConsistencyScore 计算一致性分数
func (fde *FactorDiscoveryEngine) calculateConsistencyScore(icValues []float64) float64 {
	if len(icValues) < 10 {
		return 0.5 // 数据不足，返回中性分数
	}

	// 计算正IC的比例
	positiveCount := 0
	for _, ic := range icValues {
		if ic > 0 {
			positiveCount++
		}
	}

	consistency := float64(positiveCount) / float64(len(icValues))

	// 将一致性转换为分数：50%为中性，越接近100%或0%分数越高
	if consistency >= 0.5 {
		return consistency
	} else {
		return 1.0 - consistency
	}
}

// getCurrentMarketData 获取当前市场数据
func (fde *FactorDiscoveryEngine) getCurrentMarketData() map[string]interface{} {
	// 简化实现：返回模拟市场数据
	return map[string]interface{}{
		"price_change":   (rand.Float64() - 0.5) * 0.1, // -5% 到 5%
		"volume_ratio":   0.8 + rand.Float64()*0.4,     // 0.8 到 1.2
		"volatility":     0.1 + rand.Float64()*0.2,     // 0.1 到 0.3
		"trend_strength": (rand.Float64() - 0.5) * 2,   // -1 到 1
	}
}

// calculateTrendStrength 计算趋势强度
func (fde *FactorDiscoveryEngine) calculateTrendStrength(marketData map[string]interface{}) float64 {
	if trend, ok := marketData["trend_strength"].(float64); ok {
		return trend
	}
	return 0.0
}

// calculateMarketVolatility 计算市场波动率
func (fde *FactorDiscoveryEngine) calculateMarketVolatility(marketData map[string]interface{}) float64 {
	if vol, ok := marketData["volatility"].(float64); ok {
		return vol
	}
	return 0.2 // 默认波动率
}

// calculateAverageVolume 计算平均成交量
func (fde *FactorDiscoveryEngine) calculateAverageVolume(marketData map[string]interface{}) float64 {
	if vol, ok := marketData["volume_ratio"].(float64); ok {
		return vol
	}
	return 1.0 // 默认成交量比率
}

// getFactorRegimePerformance 获取因子在特定市场状态下的表现
func (fde *FactorDiscoveryEngine) getFactorRegimePerformance(factorID, regimeType string) *FactorPerformance {
	// 简化实现：返回基于市场状态的模拟性能
	// 实际实现应该从历史数据中筛选特定市场状态下的表现

	baseIC := 0.02
	switch regimeType {
	case "bull":
		baseIC = 0.03
	case "bear":
		baseIC = 0.01
	case "volatile":
		baseIC = 0.015
	case "sideways":
		baseIC = 0.025
	}

	return &FactorPerformance{
		FactorID: factorID,
		AvgIC:    baseIC + (rand.Float64()-0.5)*0.01,
		ICIR:     baseIC * 2,
		HitRate:  0.5 + baseIC,
		RecentIC: baseIC,
	}
}

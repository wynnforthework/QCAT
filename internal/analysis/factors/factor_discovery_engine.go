package factors

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"qcat/internal/config"
)

// FactorDiscoveryEngine 因子发现引擎
type FactorDiscoveryEngine struct {
	config               *config.Config
	factorGenerator      *FactorGenerator
	factorEvaluator      *FactorEvaluator
	geneticAlgorithm     *GeneticAlgorithm
	significanceTest     *SignificanceTest
	factorRotator        *FactorRotator
	
	// 运行状态
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	isRunning  bool
	mu         sync.RWMutex
	
	// 因子库
	factorUniverse       []string
	discoveredFactors    map[string]*Factor
	activeFactors        map[string]*Factor
	factorPerformance    map[string]*FactorPerformance
	
	// 发现配置
	discoveryAlgorithm   string
	significanceLevel    float64
	rotationFrequency    time.Duration
	maxFactors           int
	
	// 监控指标
	discoveryMetrics     *DiscoveryMetrics
	discoveryHistory     []DiscoveryEvent
	
	// 配置参数
	enabled              bool
}

// Factor 因子
type Factor struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Type            string            `json:"type"`         // TECHNICAL, FUNDAMENTAL, SENTIMENT, MACRO, CUSTOM
	Formula         string            `json:"formula"`      // 因子计算公式
	Expression      *Expression       `json:"expression"`   // 解析后的表达式
	Parameters      map[string]float64 `json:"parameters"`
	
	// 统计特性
	Mean            float64           `json:"mean"`
	StdDev          float64           `json:"std_dev"`
	Skewness        float64           `json:"skewness"`
	Kurtosis        float64           `json:"kurtosis"`
	MinValue        float64           `json:"min_value"`
	MaxValue        float64           `json:"max_value"`
	
	// 性能指标
	IC              float64           `json:"ic"`              // Information Coefficient
	ICStdDev        float64           `json:"ic_std_dev"`
	ICIR            float64           `json:"icir"`            // IC Information Ratio
	RankIC          float64           `json:"rank_ic"`
	Decay           []float64         `json:"decay"`           // 因子衰减
	Turnover        float64           `json:"turnover"`        // 换手率
	
	// 显著性检验
	TValue          float64           `json:"t_value"`
	PValue          float64           `json:"p_value"`
	IsSignificant   bool              `json:"is_significant"`
	ConfidenceLevel float64           `json:"confidence_level"`
	
	// 相关性分析
	Correlations    map[string]float64 `json:"correlations"`
	MaxCorrelation  float64           `json:"max_correlation"`
	FactorLoading   float64           `json:"factor_loading"`
	
	// 分组收益分析
	GroupReturns    []GroupReturn     `json:"group_returns"`
	LongShortReturn float64           `json:"long_short_return"`
	HitRate         float64           `json:"hit_rate"`
	
	// 时序特性
	Stability       float64           `json:"stability"`
	Persistence     float64           `json:"persistence"`
	SeasonalEffect  map[string]float64 `json:"seasonal_effect"`
	
	// 元数据
	DiscoveredAt    time.Time         `json:"discovered_at"`
	LastUpdated     time.Time         `json:"last_updated"`
	UpdateCount     int               `json:"update_count"`
	UsageCount      int               `json:"usage_count"`
	Status          string            `json:"status"`          // ACTIVE, INACTIVE, DEPRECATED
	CreatedBy       string            `json:"created_by"`
	
	// 生成信息
	Generation      int               `json:"generation"`      // 遗传算法代数
	Parents         []string          `json:"parents"`         // 父因子ID
	Fitness         float64           `json:"fitness"`         // 适应度
	Complexity      int               `json:"complexity"`      // 复杂度
}

// Expression 表达式
type Expression struct {
	Type      string       `json:"type"`      // OPERATOR, FUNCTION, VARIABLE, CONSTANT
	Value     interface{}  `json:"value"`     // 值或操作符
	Children  []*Expression `json:"children"` // 子表达式
	DataType  string       `json:"data_type"` // float64, bool, string
}

// GroupReturn 分组收益
type GroupReturn struct {
	Group      int     `json:"group"`      // 分组编号（1-10）
	Count      int     `json:"count"`      // 样本数量
	Return     float64 `json:"return"`     // 平均收益
	StdDev     float64 `json:"std_dev"`    // 收益标准差
	SharpeRatio float64 `json:"sharpe_ratio"` // 夏普比率
}

// FactorGenerator 因子生成器
type FactorGenerator struct {
	baseFactors      []BaseFactor
	operators        []Operator
	functions        []Function
	constants        []float64
	maxDepth         int
	maxNodes         int
	
	// 遗传编程参数
	populationSize   int
	mutationRate     float64
	crossoverRate    float64
	elitismRate      float64
	
	mu               sync.RWMutex
}

// BaseFactor 基础因子
type BaseFactor struct {
	Name        string   `json:"name"`
	Symbol      string   `json:"symbol"`
	Type        string   `json:"type"`        // PRICE, VOLUME, TECHNICAL, FUNDAMENTAL
	Parameters  []string `json:"parameters"`  // 可配置参数
	Description string   `json:"description"`
}

// Operator 操作符
type Operator struct {
	Symbol      string `json:"symbol"`      // +, -, *, /, ^
	Name        string `json:"name"`
	Precedence  int    `json:"precedence"`
	Operands    int    `json:"operands"`    // 操作数数量
	Function    func([]float64) float64 `json:"-"`
}

// Function 函数
type Function struct {
	Name        string `json:"name"`        // SMA, EMA, RSI, MACD, etc.
	Parameters  int    `json:"parameters"`  // 参数数量
	Description string `json:"description"`
	Function    func([]float64, ...float64) []float64 `json:"-"`
}

// FactorEvaluator 因子评估器
type FactorEvaluator struct {
	evaluationPeriod time.Duration
	forwardPeriods   []int        // 前瞻期 [1, 5, 10, 20]
	benchmarkReturn  []float64    // 基准收益
	evaluationCache  map[string]*FactorEvaluation
	
	mu               sync.RWMutex
}

// FactorEvaluation 因子评估
type FactorEvaluation struct {
	FactorID        string            `json:"factor_id"`
	EvaluationDate  time.Time         `json:"evaluation_date"`
	
	// IC分析
	ICResults       []ICResult        `json:"ic_results"`
	RollingIC       []RollingIC       `json:"rolling_ic"`
	ICDecay         []float64         `json:"ic_decay"`
	
	// 分组回测
	GroupBacktest   GroupBacktest     `json:"group_backtest"`
	
	// 风险分析
	RiskAnalysis    FactorRiskAnalysis `json:"risk_analysis"`
	
	// 稳定性分析
	StabilityAnalysis FactorStabilityAnalysis `json:"stability_analysis"`
	
	// 综合评分
	OverallScore    float64           `json:"overall_score"`
	Grade           string            `json:"grade"`
	Rank            int               `json:"rank"`
}

// ICResult IC分析结果
type ICResult struct {
	Period          int       `json:"period"`         // 前瞻期
	IC              float64   `json:"ic"`
	RankIC          float64   `json:"rank_ic"`
	TValue          float64   `json:"t_value"`
	PValue          float64   `json:"p_value"`
	IsSignificant   bool      `json:"is_significant"`
	SampleSize      int       `json:"sample_size"`
	ConfidenceInterval [2]float64 `json:"confidence_interval"`
}

// RollingIC 滚动IC
type RollingIC struct {
	Date            time.Time `json:"date"`
	IC              float64   `json:"ic"`
	RankIC          float64   `json:"rank_ic"`
	TValue          float64   `json:"t_value"`
	IsSignificant   bool      `json:"is_significant"`
}

// GroupBacktest 分组回测
type GroupBacktest struct {
	Groups          []GroupStats      `json:"groups"`
	LongShort       GroupStats        `json:"long_short"`
	TopBottom       GroupStats        `json:"top_bottom"`
	MonotonicityTest MonotonicityTest  `json:"monotonicity_test"`
}

// GroupStats 分组统计
type GroupStats struct {
	Group           int       `json:"group"`
	Count           int       `json:"count"`
	AvgReturn       float64   `json:"avg_return"`
	CumReturn       float64   `json:"cum_return"`
	Volatility      float64   `json:"volatility"`
	SharpeRatio     float64   `json:"sharpe_ratio"`
	MaxDrawdown     float64   `json:"max_drawdown"`
	WinRate         float64   `json:"win_rate"`
	HitRate         float64   `json:"hit_rate"`
}

// MonotonicityTest 单调性检验
type MonotonicityTest struct {
	Statistic       float64   `json:"statistic"`
	PValue          float64   `json:"p_value"`
	IsMonotonic     bool      `json:"is_monotonic"`
	Direction       string    `json:"direction"`    // POSITIVE, NEGATIVE
	Strength        string    `json:"strength"`     // STRONG, MODERATE, WEAK
}

// FactorRiskAnalysis 因子风险分析
type FactorRiskAnalysis struct {
	Exposure        map[string]float64 `json:"exposure"`        // 行业、风格暴露
	Concentration   float64           `json:"concentration"`    // 集中度
	Turnover        float64           `json:"turnover"`        // 换手率
	Capacity        float64           `json:"capacity"`        // 容量
	LiquidityRisk   float64           `json:"liquidity_risk"`  // 流动性风险
	CrowdingRisk    float64           `json:"crowding_risk"`   // 拥挤度风险
}

// FactorStabilityAnalysis 因子稳定性分析
type FactorStabilityAnalysis struct {
	ICStability     float64           `json:"ic_stability"`
	ReturnStability float64           `json:"return_stability"`
	RankStability   float64           `json:"rank_stability"`
	Persistence     float64           `json:"persistence"`
	HalfLife        float64           `json:"half_life"`        // 半衰期
	BreakpointTest  BreakpointTest    `json:"breakpoint_test"`
}

// BreakpointTest 断点检验
type BreakpointTest struct {
	HasBreakpoint   bool      `json:"has_breakpoint"`
	BreakpointDate  time.Time `json:"breakpoint_date"`
	PrePeriodIC     float64   `json:"pre_period_ic"`
	PostPeriodIC    float64   `json:"post_period_ic"`
	Statistic       float64   `json:"statistic"`
	PValue          float64   `json:"p_value"`
}

// GeneticAlgorithm 遗传算法
type GeneticAlgorithm struct {
	population       []*Factor
	populationSize   int
	maxGenerations   int
	mutationRate     float64
	crossoverRate    float64
	elitismRate      float64
	selectionMethod  string           // TOURNAMENT, ROULETTE, RANK
	
	// 多样性控制
	diversityWeight  float64
	complexityPenalty float64
	
	// 历史记录
	generationHistory []GenerationStats
	bestFactors      []*Factor
	
	mu               sync.RWMutex
}

// GenerationStats 代数统计
type GenerationStats struct {
	Generation      int       `json:"generation"`
	BestFitness     float64   `json:"best_fitness"`
	AvgFitness      float64   `json:"avg_fitness"`
	Diversity       float64   `json:"diversity"`
	Complexity      float64   `json:"complexity"`
	Timestamp       time.Time `json:"timestamp"`
}

// SignificanceTest 显著性检验
type SignificanceTest struct {
	testMethod       string           // IC_TEST, T_TEST, RANK_TEST
	significanceLevel float64
	multipleTestCorrection string    // BONFERRONI, FDR, HOLM
	
	mu               sync.RWMutex
}

// FactorRotator 因子轮换器
type FactorRotator struct {
	rotationStrategy string           // PERFORMANCE, CORRELATION, REGIME
	rotationFrequency time.Duration
	maxActiveFactors int
	correlationThreshold float64
	performanceWindow time.Duration
	
	// 轮换历史
	rotationHistory  []RotationEvent
	lastRotation     time.Time
	
	mu               sync.RWMutex
}

// RotationEvent 轮换事件
type RotationEvent struct {
	Date            time.Time         `json:"date"`
	Action          string            `json:"action"`        // ADD, REMOVE, REPLACE
	FactorID        string            `json:"factor_id"`
	ReplacedFactorID string           `json:"replaced_factor_id"`
	Reason          string            `json:"reason"`
	Performance     float64           `json:"performance"`
	Correlation     float64           `json:"correlation"`
}

// FactorPerformance 因子表现
type FactorPerformance struct {
	FactorID        string            `json:"factor_id"`
	
	// 历史表现
	PerformanceHistory []PerformancePoint `json:"performance_history"`
	
	// 汇总统计
	AvgIC           float64           `json:"avg_ic"`
	AvgRankIC       float64           `json:"avg_rank_ic"`
	ICStdDev        float64           `json:"ic_std_dev"`
	ICIR            float64           `json:"icir"`
	HitRate         float64           `json:"hit_rate"`
	
	// 收益分析
	CumulativeReturn float64          `json:"cumulative_return"`
	AnnualizedReturn float64          `json:"annualized_return"`
	Volatility      float64           `json:"volatility"`
	SharpeRatio     float64           `json:"sharpe_ratio"`
	MaxDrawdown     float64           `json:"max_drawdown"`
	
	// 稳定性指标
	StabilityScore  float64           `json:"stability_score"`
	ConsistencyScore float64          `json:"consistency_score"`
	
	// 最近表现
	RecentIC        float64           `json:"recent_ic"`
	RecentRankIC    float64           `json:"recent_rank_ic"`
	RecentReturn    float64           `json:"recent_return"`
	RecentRank      int               `json:"recent_rank"`
	
	// 预测能力
	ForecastAccuracy float64          `json:"forecast_accuracy"`
	ForecastBias    float64           `json:"forecast_bias"`
	
	LastUpdated     time.Time         `json:"last_updated"`
}

// PerformancePoint 表现点
type PerformancePoint struct {
	Date            time.Time `json:"date"`
	IC              float64   `json:"ic"`
	RankIC          float64   `json:"rank_ic"`
	Return          float64   `json:"return"`
	CumReturn       float64   `json:"cum_return"`
	Rank            int       `json:"rank"`
	IsSignificant   bool      `json:"is_significant"`
}

// DiscoveryMetrics 发现指标
type DiscoveryMetrics struct {
	mu sync.RWMutex
	
	// 因子统计
	TotalFactors        int               `json:"total_factors"`
	ActiveFactors       int               `json:"active_factors"`
	SignificantFactors  int               `json:"significant_factors"`
	
	// 发现统计
	FactorsDiscovered   int               `json:"factors_discovered"`
	DiscoveryRate       float64           `json:"discovery_rate"`
	AvgFactorLifespan   time.Duration     `json:"avg_factor_lifespan"`
	
	// 质量指标
	AvgIC               float64           `json:"avg_ic"`
	AvgICIR             float64           `json:"avg_icir"`
	AvgSignificance     float64           `json:"avg_significance"`
	TopFactorIC         float64           `json:"top_factor_ic"`
	
	// 多样性指标
	FactorDiversity     float64           `json:"factor_diversity"`
	TypeDistribution    map[string]int    `json:"type_distribution"`
	ComplexityDistribution map[int]int    `json:"complexity_distribution"`
	
	// 性能指标
	DiscoveryTime       time.Duration     `json:"discovery_time"`
	EvaluationTime      time.Duration     `json:"evaluation_time"`
	RotationEfficiency  float64           `json:"rotation_efficiency"`
	
	// 算法统计
	GAGenerations       int               `json:"ga_generations"`
	ConvergenceRate     float64           `json:"convergence_rate"`
	
	LastUpdated         time.Time         `json:"last_updated"`
}

// DiscoveryEvent 发现事件
type DiscoveryEvent struct {
	Date            time.Time         `json:"date"`
	EventType       string            `json:"event_type"`    // DISCOVERY, EVALUATION, ROTATION, DEPRECATION
	FactorID        string            `json:"factor_id"`
	Details         map[string]interface{} `json:"details"`
	Impact          string            `json:"impact"`        // HIGH, MEDIUM, LOW
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
		discoveryMetrics:  &DiscoveryMetrics{
			TypeDistribution:      make(map[string]int),
			ComplexityDistribution: make(map[int]int),
		},
		discoveryHistory:  make([]DiscoveryEvent, 0),
		discoveryAlgorithm: "genetic_programming",
		significanceLevel: 0.05,
		rotationFrequency: 7 * 24 * time.Hour, // 每周轮换
		maxFactors:        50,
		enabled:           true,
	}
	
	// 从配置文件读取参数
	if cfg != nil {
		// TODO: 从配置文件读取因子发现参数
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
		baseFactors:      initializeBaseFactors(),
		operators:        initializeOperators(),
		functions:        initializeFunctions(),
		constants:        []float64{1, 2, 3, 5, 10, 20, 60, 252},
		maxDepth:         5,
		maxNodes:         50,
		populationSize:   100,
		mutationRate:     0.1,
		crossoverRate:    0.8,
		elitismRate:      0.1,
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
	// TODO: 从配置或数据库加载基础因子
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
	// TODO: 实现随机搜索算法
	log.Println("Running random search for factor discovery...")
}

// runSystematicSearch 运行系统化搜索
func (fde *FactorDiscoveryEngine) runSystematicSearch() {
	// TODO: 实现系统化搜索算法
	log.Println("Running systematic search for factor discovery...")
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
	// TODO: 实现随机因子生成
	factor := &Factor{
		ID:           fde.generateFactorID(),
		Name:         fmt.Sprintf("Factor_%d", rand.Int()),
		Type:         "CUSTOM",
		Formula:      "CLOSE / SMA(CLOSE, 20)",
		Parameters:   make(map[string]float64),
		DiscoveredAt: time.Now(),
		Status:       "ACTIVE",
		CreatedBy:    "genetic_algorithm",
		Generation:   0,
		Complexity:   fde.calculateComplexity("CLOSE / SMA(CLOSE, 20)"),
	}
	
	return factor
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
	// TODO: 实现收敛检查逻辑
	return false
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
	// TODO: 检查因子是否新颖（不与现有因子重复）
	threshold := 0.95 // 相似度阈值
	
	fde.mu.RLock()
	defer fde.mu.RUnlock()
	
	for _, existingFactor := range fde.discoveredFactors {
		similarity := fde.calculateFactorSimilarity(factor, existingFactor)
		if similarity > threshold {
			return false
		}
	}
	
	return true
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
	// TODO: 实现实际的IC计算
	return rand.Float64()*0.2 - 0.1 // 模拟-0.1到0.1之间的IC
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
	// TODO: 计算因子多样性
	return 0.1
}

func (fde *FactorDiscoveryEngine) calculatePopulationDiversity() float64 {
	// TODO: 计算种群多样性
	return 0.5
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
	// TODO: 实现因子交叉操作
	child := fde.copyFactor(parent1)
	child.ID = fde.generateFactorID()
	child.Parents = []string{parent1.ID, parent2.ID}
	child.Generation = math.Max(float64(parent1.Generation), float64(parent2.Generation)) + 1
	return child
}

func (fde *FactorDiscoveryEngine) mutate(factor *Factor) {
	// TODO: 实现因子变异操作
	factor.LastUpdated = time.Now()
}

func (fde *FactorDiscoveryEngine) copyFactor(original *Factor) *Factor {
	copy := *original
	copy.ID = fde.generateFactorID()
	copy.DiscoveredAt = time.Now()
	copy.LastUpdated = time.Now()
	return &copy
}

func (fde *FactorDiscoveryEngine) calculateFactorSimilarity(factor1, factor2 *Factor) float64 {
	// TODO: 计算因子相似度
	return 0.0
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
	// TODO: 实现IC计算
	ic := rand.Float64()*0.2 - 0.1
	return ICResult{
		Period:        period,
		IC:            ic,
		RankIC:        ic * 0.8,
		TValue:        ic / 0.02,
		PValue:        0.05,
		IsSignificant: math.Abs(ic) > 0.02,
		SampleSize:    1000,
	}
}

func (fde *FactorDiscoveryEngine) calculateRollingIC(factor *Factor) []RollingIC {
	// TODO: 实现滚动IC计算
	return []RollingIC{}
}

func (fde *FactorDiscoveryEngine) calculateICDecay(factor *Factor) []float64 {
	// TODO: 实现IC衰减计算
	return []float64{1.0, 0.8, 0.6, 0.4, 0.2}
}

func (fde *FactorDiscoveryEngine) performGroupBacktest(factor *Factor) GroupBacktest {
	// TODO: 实现分组回测
	return GroupBacktest{}
}

func (fde *FactorDiscoveryEngine) analyzeFactorRisk(factor *Factor) FactorRiskAnalysis {
	// TODO: 实现因子风险分析
	return FactorRiskAnalysis{
		Exposure:      make(map[string]float64),
		Concentration: 0.3,
		Turnover:      0.1,
		Capacity:      1000000.0,
	}
}

func (fde *FactorDiscoveryEngine) analyzeFactorStability(factor *Factor) FactorStabilityAnalysis {
	// TODO: 实现因子稳定性分析
	return FactorStabilityAnalysis{
		ICStability:     0.8,
		ReturnStability: 0.7,
		RankStability:   0.75,
		Persistence:     0.6,
		HalfLife:        30.0,
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
	// TODO: 实现基于性能的轮换
}

func (fde *FactorDiscoveryEngine) correlationBasedRotation(current, candidates []*Factor) {
	// TODO: 实现基于相关性的轮换
}

func (fde *FactorDiscoveryEngine) regimeBasedRotation(current, candidates []*Factor) {
	// TODO: 实现基于市场状态的轮换
}

func (fde *FactorDiscoveryEngine) calculateFactorPerformance(factor *Factor) *FactorPerformance {
	// TODO: 计算因子表现
	return &FactorPerformance{
		FactorID:         factor.ID,
		AvgIC:           factor.IC,
		AvgRankIC:       factor.RankIC,
		ICStdDev:        factor.ICStdDev,
		ICIR:            factor.ICIR,
		StabilityScore:  factor.Stability,
		RecentIC:        factor.IC,
		LastUpdated:     time.Now(),
	}
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
		"running":               fde.isRunning,
		"enabled":               fde.enabled,
		"discovered_factors":    len(fde.discoveredFactors),
		"active_factors":        len(fde.activeFactors),
		"discovery_algorithm":   fde.discoveryAlgorithm,
		"significance_level":    fde.significanceLevel,
		"rotation_frequency":    fde.rotationFrequency,
		"max_factors":           fde.maxFactors,
		"discovery_metrics":     fde.discoveryMetrics,
		"discovery_events":      len(fde.discoveryHistory),
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

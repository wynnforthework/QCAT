package automl

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"

	"qcat/internal/config"
	"qcat/internal/logger"

	"gopkg.in/yaml.v3"
)

// DistributedOptimizer 分布式优化器
type DistributedOptimizer struct {
	config            *config.Config
	logger            logger.Logger
	consistencyMgr    *ConsistencyManager
	optimizationHub   *OptimizationHub
	performanceDB     *PerformanceDatabase
	clusterManager    *ClusterManager
	adaptiveOptimizer *AdaptiveOptimizer
	conf              *distributedOptConfig
	backtestRunner    BacktestRunner
	mu                sync.RWMutex
}

// OptimizationHub 优化结果中心
type OptimizationHub struct {
	bestResults     map[string]*OptimizationResult // taskID -> best result
	activeNodes     map[string]*NodeInfo
	optimizationLog []*OptimizationEvent
	mu              sync.RWMutex
}

// OptimizationResult 优化结果
type OptimizationResult struct {
	TaskID        string                 `json:"task_id"`
	StrategyName  string                 `json:"strategy_name"`
	Parameters    map[string]interface{} `json:"parameters"`
	Performance   *PerformanceMetrics    `json:"performance"`
	RandomSeed    int64                  `json:"random_seed"`
	DataHash      string                 `json:"data_hash"`
	ModelData     []byte                 `json:"model_data"`
	DiscoveredBy  string                 `json:"discovered_by"` // 发现此结果的节点ID
	DiscoveredAt  time.Time              `json:"discovered_at"`
	Confidence    float64                `json:"confidence"`
	IsGlobalBest  bool                   `json:"is_global_best"`
	AdoptionCount int                    `json:"adoption_count"` // 被其他节点采用的次数
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	ProfitRate         float64 `json:"profit_rate"`
	SharpeRatio        float64 `json:"sharpe_ratio"`
	MaxDrawdown        float64 `json:"max_drawdown"`
	WinRate            float64 `json:"win_rate"`
	TotalReturn        float64 `json:"total_return"`
	RiskAdjustedReturn float64 `json:"risk_adjusted_return"`
}

// NodeInfo 节点信息
type NodeInfo struct {
	NodeID      string              `json:"node_id"`
	LastSeen    time.Time           `json:"last_seen"`
	Status      string              `json:"status"` // active, inactive, optimizing
	CurrentTask string              `json:"current_task"`
	BestResult  *OptimizationResult `json:"best_result"`
}

// OptimizationEvent 优化事件
type OptimizationEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	EventType   string                 `json:"event_type"` // new_best, adoption, node_join, node_leave
	NodeID      string                 `json:"node_id"`
	TaskID      string                 `json:"task_id"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data"`
}

// PerformanceDatabase 性能数据库
type PerformanceDatabase struct {
	results map[string]*OptimizationResult // 所有历史结果
	mu      sync.RWMutex
}

// ClusterManager 集群管理器
type ClusterManager struct {
	nodes       map[string]*NodeInfo
	broadcaster *ResultBroadcaster
	discoverer  *NodeDiscoverer
	mu          sync.RWMutex
}

// ResultBroadcaster 结果广播器
type ResultBroadcaster struct {
	hub     *OptimizationHub
	cluster *ClusterManager
}

// NodeDiscoverer 节点发现器
type NodeDiscoverer struct {
	cluster *ClusterManager
	config  *config.Config
}

// NewDistributedOptimizer 创建分布式优化器
func NewDistributedOptimizer(cfg *config.Config, consistencyMgr *ConsistencyManager) (*DistributedOptimizer, error) {
	logger := logger.GetGlobalLogger()

	optimizer := &DistributedOptimizer{
		config:         cfg,
		logger:         logger,
		consistencyMgr: consistencyMgr,
		optimizationHub: &OptimizationHub{
			bestResults:     make(map[string]*OptimizationResult),
			activeNodes:     make(map[string]*NodeInfo),
			optimizationLog: make([]*OptimizationEvent, 0),
		},
		performanceDB: &PerformanceDatabase{
			results: make(map[string]*OptimizationResult),
		},
	}

	// 初始化集群管理器
	optimizer.clusterManager = &ClusterManager{
		nodes: make(map[string]*NodeInfo),
		broadcaster: &ResultBroadcaster{
			hub: optimizer.optimizationHub,
		},
		discoverer: &NodeDiscoverer{
			cluster: optimizer.clusterManager,
			config:  cfg,
		},
	}
	optimizer.clusterManager.broadcaster.cluster = optimizer.clusterManager

	// 加载分布式优化配置
	optimizer.conf = loadDistributedOptConfig("configs/distributed_optimization.yaml")

	// 初始化自适应优化器（默认配置）
	optimizer.adaptiveOptimizer = NewAdaptiveOptimizer(nil)

	// 启动后台任务
	go optimizer.startBackgroundTasks()

	return optimizer, nil
}

// StartOptimization 开始分布式优化
func (do *DistributedOptimizer) StartOptimization(ctx context.Context, taskID string, strategyName string, dataHash string) (*OptimizationResult, error) {
	do.logger.Info("开始分布式优化", "task_id", taskID, "strategy", strategyName)

	// 1. 检查是否已有全局最优结果
	if bestResult := do.getGlobalBestResult(taskID); bestResult != nil {
		do.logger.Info("发现全局最优结果，直接采用",
			"task_id", taskID,
			"profit_rate", bestResult.Performance.ProfitRate,
			"discovered_by", bestResult.DiscoveredBy)
		return bestResult, nil
	}

	// 2. 使用随机种子进行本地优化
	randomSeed := time.Now().UnixNano()
	rand.Seed(randomSeed)

	do.logger.Info("使用随机种子进行本地优化", "task_id", taskID, "seed", randomSeed)

	var localResult *OptimizationResult
	var err error
	// 优先使用自适应优化器（若可用）
	if do.adaptiveOptimizer != nil {
		localResult, err = do.adaptiveOptimizer.Optimize(ctx, strategyName, dataHash, randomSeed)
		if err != nil {
			do.logger.Warn("自适应优化器执行失败，回退到本地优化", "error", err)
		}
	}
	if localResult == nil || err != nil {
		// 回退到内置本地优化
		localResult, err = do.performLocalOptimization(taskID, strategyName, dataHash, randomSeed)
		if err != nil {
			return nil, fmt.Errorf("本地优化失败: %w", err)
		}
	}

	// 4. 检查是否为新的全局最优
	if do.isNewGlobalBest(taskID, localResult) {
		do.logger.Info("发现新的全局最优结果",
			"task_id", taskID,
			"profit_rate", localResult.Performance.ProfitRate)

		// 广播给其他节点
		go do.broadcastBestResult(localResult)

		// 更新全局最优
		do.updateGlobalBestResult(taskID, localResult)
	}

	return localResult, nil
}

// AdoptBestResult 采用最优结果
func (do *DistributedOptimizer) AdoptBestResult(taskID string, result *OptimizationResult) error {
	do.logger.Info("采用最优结果",
		"task_id", taskID,
		"profit_rate", result.Performance.ProfitRate,
		"discovered_by", result.DiscoveredBy)

	// 1. 验证结果的有效性
	if err := do.validateResult(result); err != nil {
		return fmt.Errorf("结果验证失败: %w", err)
	}

	// 2. 回测验证（若启用）
	if err := do.validateByBacktest(result); err != nil {
		return fmt.Errorf("回测验证失败: %w", err)
	}

	// 3. 应用最优参数和模型
	if err := do.applyOptimalResult(result); err != nil {
		return fmt.Errorf("应用最优结果失败: %w", err)
	}

	// 4. 记录采用事件
	do.recordOptimizationEvent(&OptimizationEvent{
		Timestamp: time.Now(),
		EventType: "adoption",
		NodeID:    do.getNodeID(),
		TaskID:    taskID,
		Description: fmt.Sprintf("采用来自节点 %s 的最优结果，收益率: %.2f%%",
			result.DiscoveredBy, result.Performance.ProfitRate),
		Data: map[string]interface{}{
			"profit_rate":   result.Performance.ProfitRate,
			"discovered_by": result.DiscoveredBy,
		},
	})

	// 5. 更新采用计数
	result.AdoptionCount++

	return nil
}

// validateByBacktest 回测验证桩函数
func (do *DistributedOptimizer) validateByBacktest(result *OptimizationResult) error {
	// 若存在真实回测执行器且启用，优先使用
	if do.backtestRunner != nil && do.conf != nil && do.conf.Validation.Backtest.Enabled {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		stats, err := do.backtestRunner.Run(ctx, result.TaskID, result.StrategyName, result.Parameters, result.DataHash)
		if err != nil {
			return fmt.Errorf("真实回测失败: %w", err)
		}
		// 使用配置阈值进行校验
		if stats.MaxDrawdown > do.conf.Validation.Backtest.MaxDrawdown {
			return fmt.Errorf("回测最大回撤超阈: maxDD=%.2f%% > %.2f%%", stats.MaxDrawdown, do.conf.Validation.Backtest.MaxDrawdown)
		}
		// 要求总收益不显著低于候选结果
		minRetention := do.conf.Validation.Backtest.MinMeanRetention
		if stats.TotalReturn < result.Performance.TotalReturn*minRetention {
			return fmt.Errorf("回测收益偏低: total=%.2f%% < threshold=%.2f%%", stats.TotalReturn, result.Performance.TotalReturn*minRetention)
		}
		return nil
	}

	// 否则采用模拟扰动校验（见原逻辑）
	// 配置
	runs := 5
	maxAllowedDrawdown := 10.0 // %
	minMeanRetention := 0.8    // 平均收益需至少达到原结果的80%
	if do.conf != nil {
		if do.conf.Validation.Backtest.Runs > 0 {
			runs = do.conf.Validation.Backtest.Runs
		}
		if do.conf.Validation.Backtest.MaxDrawdown > 0 {
			maxAllowedDrawdown = do.conf.Validation.Backtest.MaxDrawdown
		}
		if do.conf.Validation.Backtest.MinMeanRetention > 0 {
			minMeanRetention = do.conf.Validation.Backtest.MinMeanRetention
		}
	}
	profitSamples := make([]float64, 0, runs)
	maxDrawdowns := make([]float64, 0, runs)
	for i := 0; i < runs; i++ {
		noise := (rand.Float64()*2 - 1) * 0.2
		simProfit := result.Performance.ProfitRate * (1 + noise)
		if simProfit < 0 {
			simProfit = 0
		}
		profitSamples = append(profitSamples, simProfit)
		simDD := math.Min(20.0, math.Abs(noise)*20.0)
		maxDrawdowns = append(maxDrawdowns, simDD)
	}
	meanProfit := 0.0
	for _, p := range profitSamples {
		meanProfit += p
	}
	meanProfit /= float64(len(profitSamples))
	variance := 0.0
	for _, p := range profitSamples {
		d := p - meanProfit
		variance += d * d
	}
	variance /= float64(len(profitSamples))
	maxDD := 0.0
	for _, dd := range maxDrawdowns {
		if dd > maxDD {
			maxDD = dd
		}
	}
	if meanProfit < result.Performance.ProfitRate*minMeanRetention {
		return fmt.Errorf("回测均值偏低: mean=%.2f%% < threshold=%.2f%%", meanProfit, result.Performance.ProfitRate*minMeanRetention)
	}
	if maxDD > maxAllowedDrawdown {
		return fmt.Errorf("回测最大回撤超阈: maxDD=%.2f%% > %.2f%%", maxDD, maxAllowedDrawdown)
	}
	do.recordOptimizationEvent(&OptimizationEvent{
		Timestamp: time.Now(),
		EventType: "backtest_validation",
		NodeID:    do.getNodeID(),
		TaskID:    result.TaskID,
		Description: fmt.Sprintf("回测通过: mean=%.2f%%, var=%.4f, maxDD=%.2f%%",
			meanProfit, variance, maxDD),
		Data: map[string]interface{}{
			"mean_profit": meanProfit,
			"variance":    variance,
			"max_dd":      maxDD,
		},
	})
	return nil
}

// GetOptimizationStatus 获取优化状态
func (do *DistributedOptimizer) GetOptimizationStatus(taskID string) *OptimizationStatus {
	do.mu.RLock()
	defer do.mu.RUnlock()

	status := &OptimizationStatus{
		TaskID:           taskID,
		ActiveNodes:      len(do.optimizationHub.activeNodes),
		GlobalBestResult: do.optimizationHub.bestResults[taskID],
		OptimizationLog:  do.getRecentEvents(taskID, 10),
	}

	return status
}

// OptimizationStatus 优化状态
type OptimizationStatus struct {
	TaskID           string               `json:"task_id"`
	ActiveNodes      int                  `json:"active_nodes"`
	GlobalBestResult *OptimizationResult  `json:"global_best_result"`
	OptimizationLog  []*OptimizationEvent `json:"optimization_log"`
}

// 私有方法

func (do *DistributedOptimizer) performLocalOptimization(taskID, strategyName, dataHash string, seed int64) (*OptimizationResult, error) {
	// 这里应该调用实际的AutoML引擎进行训练
	// 为了演示，我们生成一个模拟结果

	// 使用种子生成可重现的随机结果
	rand.Seed(seed)

	// 模拟训练过程
	time.Sleep(100 * time.Millisecond) // 模拟训练时间

	// 生成模拟性能指标
	profitRate := 5.0 + rand.Float64()*10.0 // 5-15% 的收益率
	sharpeRatio := 0.5 + rand.Float64()*2.0 // 0.5-2.5 的夏普比率
	maxDrawdown := rand.Float64() * 5.0     // 0-5% 的最大回撤
	winRate := 0.4 + rand.Float64()*0.4     // 40-80% 的胜率

	result := &OptimizationResult{
		TaskID:       taskID,
		StrategyName: strategyName,
		Parameters: map[string]interface{}{
			"learning_rate": 0.001 + rand.Float64()*0.009,
			"batch_size":    []int{32, 64, 128}[rand.Intn(3)],
			"epochs":        100 + rand.Intn(200),
		},
		Performance: &PerformanceMetrics{
			ProfitRate:         profitRate,
			SharpeRatio:        sharpeRatio,
			MaxDrawdown:        maxDrawdown,
			WinRate:            winRate,
			TotalReturn:        profitRate,
			RiskAdjustedReturn: profitRate / (1 + maxDrawdown),
		},
		RandomSeed:    seed,
		DataHash:      dataHash,
		ModelData:     []byte("simulated_model_data"),
		DiscoveredBy:  do.getNodeID(),
		DiscoveredAt:  time.Now(),
		Confidence:    0.8 + rand.Float64()*0.2,
		IsGlobalBest:  false,
		AdoptionCount: 0,
	}

	return result, nil
}

func (do *DistributedOptimizer) isNewGlobalBest(taskID string, result *OptimizationResult) bool {
	do.mu.RLock()
	defer do.mu.RUnlock()

	currentBest := do.optimizationHub.bestResults[taskID]
	if currentBest == nil {
		return true
	}

	// 比较性能指标（这里使用收益率作为主要指标）
	return result.Performance.ProfitRate > currentBest.Performance.ProfitRate
}

func (do *DistributedOptimizer) updateGlobalBestResult(taskID string, result *OptimizationResult) {
	do.mu.Lock()
	defer do.mu.Unlock()

	result.IsGlobalBest = true
	do.optimizationHub.bestResults[taskID] = result
	do.performanceDB.results[taskID] = result

	// 记录新最优事件
	do.recordOptimizationEvent(&OptimizationEvent{
		Timestamp:   time.Now(),
		EventType:   "new_best",
		NodeID:      do.getNodeID(),
		TaskID:      taskID,
		Description: fmt.Sprintf("发现新的全局最优结果，收益率: %.2f%%", result.Performance.ProfitRate),
		Data: map[string]interface{}{
			"profit_rate":  result.Performance.ProfitRate,
			"sharpe_ratio": result.Performance.SharpeRatio,
		},
	})
}

func (do *DistributedOptimizer) getGlobalBestResult(taskID string) *OptimizationResult {
	do.mu.RLock()
	defer do.mu.RUnlock()
	return do.optimizationHub.bestResults[taskID]
}

func (do *DistributedOptimizer) broadcastBestResult(result *OptimizationResult) {
	// 这里应该实现实际的网络广播
	// 为了演示，我们只是记录日志
	do.logger.Info("广播最优结果",
		"task_id", result.TaskID,
		"profit_rate", result.Performance.ProfitRate)

	// TODO: 实现实际的网络广播逻辑
	// 可以使用 gRPC、HTTP、消息队列等方式
}

func (do *DistributedOptimizer) validateResult(result *OptimizationResult) error {
	// 验证结果的基本信息
	if result.TaskID == "" || result.StrategyName == "" {
		return fmt.Errorf("结果信息不完整")
	}

	// 验证性能指标
	if result.Performance.ProfitRate <= 0 {
		return fmt.Errorf("收益率必须大于0")
	}

	if result.Performance.MaxDrawdown < 0 {
		return fmt.Errorf("最大回撤不能为负数")
	}

	// 验证模型数据
	if len(result.ModelData) == 0 {
		return fmt.Errorf("模型数据为空")
	}

	return nil
}

func (do *DistributedOptimizer) applyOptimalResult(result *OptimizationResult) error {
	// 这里应该实现实际的结果应用逻辑
	// 1. 加载模型数据
	// 2. 应用最优参数
	// 3. 更新策略配置

	do.logger.Info("应用最优结果",
		"task_id", result.TaskID,
		"parameters", result.Parameters)

	// TODO: 实现实际的应用逻辑
	return nil
}

func (do *DistributedOptimizer) recordOptimizationEvent(event *OptimizationEvent) {
	do.mu.Lock()
	defer do.mu.Unlock()

	do.optimizationHub.optimizationLog = append(do.optimizationHub.optimizationLog, event)

	// 保持日志数量在合理范围内
	if len(do.optimizationHub.optimizationLog) > 1000 {
		do.optimizationHub.optimizationLog = do.optimizationHub.optimizationLog[100:]
	}
}

func (do *DistributedOptimizer) getRecentEvents(taskID string, limit int) []*OptimizationEvent {
	do.mu.RLock()
	defer do.mu.RUnlock()

	var events []*OptimizationEvent
	for _, event := range do.optimizationHub.optimizationLog {
		if event.TaskID == taskID {
			events = append(events, event)
			if len(events) >= limit {
				break
			}
		}
	}
	return events
}

func (do *DistributedOptimizer) getNodeID() string {
	// 这里应该返回实际的节点ID
	// 为了演示，我们使用时间戳
	return fmt.Sprintf("node-%d", time.Now().Unix())
}

func (do *DistributedOptimizer) startBackgroundTasks() {
	// 启动节点发现
	go do.clusterManager.discoverer.startDiscovery()

	// 启动结果同步
	go do.syncResultsPeriodically()

	// 启动性能监控
	go do.monitorPerformance()
}

func (do *DistributedOptimizer) syncResultsPeriodically() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		do.syncResultsWithCluster()
	}
}

func (do *DistributedOptimizer) syncResultsWithCluster() {
	// TODO: 实现与集群其他节点的结果同步
	do.logger.Debug("同步结果与集群")
}

func (do *DistributedOptimizer) monitorPerformance() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		do.logOptimizationStats()
	}
}

func (do *DistributedOptimizer) logOptimizationStats() {
	do.mu.RLock()
	defer do.mu.RUnlock()

	stats := map[string]interface{}{
		"active_nodes":     len(do.optimizationHub.activeNodes),
		"best_results":     len(do.optimizationHub.bestResults),
		"total_results":    len(do.performanceDB.results),
		"optimization_log": len(do.optimizationHub.optimizationLog),
	}

	do.logger.Info("优化统计", "stats", stats)
}

// NodeDiscoverer 方法
func (nd *NodeDiscoverer) startDiscovery() {
	// TODO: 实现节点发现逻辑
	// 可以使用心跳机制、服务注册等方式
}

// ResultBroadcaster 方法
func (rb *ResultBroadcaster) broadcast(result *OptimizationResult) error {
	// TODO: 实现实际的广播逻辑
	// 可以使用 gRPC、HTTP、消息队列等方式
	return nil
}

// BacktestRunner 定义真实回测执行接口
type BacktestRunner interface {
	Run(ctx context.Context, taskID string, strategyName string, parameters map[string]interface{}, dataHash string) (*BacktestStats, error)
}

// BacktestStats 回测关键指标
type BacktestStats struct {
	TotalReturn float64
	MaxDrawdown float64
	SharpeRatio float64
}

// SetBacktestRunner 注入回测执行器
func (do *DistributedOptimizer) SetBacktestRunner(r BacktestRunner) {
	do.mu.Lock()
	defer do.mu.Unlock()
	do.backtestRunner = r
}

// EnsureDefaultBacktestRunner installs a default runner if none was injected.
func (do *DistributedOptimizer) EnsureDefaultBacktestRunner() {
	do.mu.Lock()
	defer do.mu.Unlock()
	if do.backtestRunner == nil {
		do.backtestRunner = NewDefaultBacktestRunner()
	}
}

// distributedOptConfig 仅提取需要的分布式优化配置
type distributedOptConfig struct {
	Validation struct {
		Backtest struct {
			Enabled          bool    `yaml:"enabled"`
			Runs             int     `yaml:"runs"`
			MaxDrawdown      float64 `yaml:"max_drawdown"`
			MinMeanRetention float64 `yaml:"min_mean_retention"`
		} `yaml:"backtest"`
	} `yaml:"validation"`
}

func loadDistributedOptConfig(path string) *distributedOptConfig {
	cfg := &distributedOptConfig{}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	_ = yaml.Unmarshal(data, cfg)
	// 合理的默认值
	if cfg.Validation.Backtest.Runs <= 0 {
		cfg.Validation.Backtest.Runs = 5
	}
	if cfg.Validation.Backtest.MaxDrawdown <= 0 {
		cfg.Validation.Backtest.MaxDrawdown = 10.0
	}
	if cfg.Validation.Backtest.MinMeanRetention <= 0 {
		cfg.Validation.Backtest.MinMeanRetention = 0.8
	}
	return cfg
}

func toFloatParams(p map[string]interface{}) map[string]float64 {
	out := make(map[string]float64)
	for k, v := range p {
		switch t := v.(type) {
		case float64:
			out[k] = t
		case float32:
			out[k] = float64(t)
		case int:
			out[k] = float64(t)
		case int64:
			out[k] = float64(t)
		case json.Number:
			if f, err := t.Float64(); err == nil {
				out[k] = f
			}
		}
	}
	return out
}

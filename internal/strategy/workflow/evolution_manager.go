package workflow

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"qcat/internal/events"
)

// EvolutionConfig 进化配置
type EvolutionConfig struct {
	// 评估配置
	EvaluationInterval  time.Duration `yaml:"evaluation_interval"`
	MinEvaluationPeriod time.Duration `yaml:"min_evaluation_period"`
	PerformanceWindow   time.Duration `yaml:"performance_window"`

	// 进化参数
	MutationRate   float64 `yaml:"mutation_rate"`
	CrossoverRate  float64 `yaml:"crossover_rate"`
	ElitismRate    float64 `yaml:"elitism_rate"`
	PopulationSize int     `yaml:"population_size"`

	// 淘汰参数
	EliminationThreshold float64 `yaml:"elimination_threshold"`
	MinSurvivalRate      float64 `yaml:"min_survival_rate"`
	MaxGenerations       int     `yaml:"max_generations"`

	// 性能阈值
	MinSharpeRatio   float64 `yaml:"min_sharpe_ratio"`
	MaxDrawdownLimit float64 `yaml:"max_drawdown_limit"`
	MinWinRate       float64 `yaml:"min_win_rate"`
	MinProfitFactor  float64 `yaml:"min_profit_factor"`
}

// StrategyGenome 策略基因组
type StrategyGenome struct {
	ID              string              `json:"id"`
	Generation      int                 `json:"generation"`
	Parents         []string            `json:"parents"`
	Genes           map[string]float64  `json:"genes"`
	Fitness         float64             `json:"fitness"`
	Performance     *PerformanceMetrics `json:"performance"`
	CreatedAt       time.Time           `json:"created_at"`
	LastEvaluated   time.Time           `json:"last_evaluated"`
	EvaluationCount int                 `json:"evaluation_count"`
}

// EvolutionGeneration 进化代
type EvolutionGeneration struct {
	Number      int                        `json:"number"`
	Strategies  map[string]*StrategyGenome `json:"strategies"`
	BestFitness float64                    `json:"best_fitness"`
	AvgFitness  float64                    `json:"avg_fitness"`
	CreatedAt   time.Time                  `json:"created_at"`
	Diversity   float64                    `json:"diversity"`
}

// EvolutionManager 进化管理器
type EvolutionManager struct {
	config *EvolutionConfig

	// 进化状态
	currentGeneration *EvolutionGeneration
	generationHistory []*EvolutionGeneration
	population        map[string]*StrategyGenome
	populationMu      sync.RWMutex

	// 事件系统
	eventBus *events.EventBus

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool
	runningMu sync.RWMutex
	wg        sync.WaitGroup

	// 统计信息
	stats   *EvolutionStats
	statsMu sync.RWMutex
}

// EvolutionStats 进化统计信息
type EvolutionStats struct {
	TotalGenerations     int64
	TotalStrategies      int64
	ActiveStrategies     int64
	EliminatedStrategies int64
	BestFitness          float64
	AverageFitness       float64
	DiversityIndex       float64
	LastEvolutionTime    time.Time
	EvolutionDuration    time.Duration
}

// NewEvolutionManager 创建进化管理器
func NewEvolutionManager(config *EvolutionConfig, eventBus *events.EventBus) *EvolutionManager {
	if config == nil {
		config = GetDefaultEvolutionConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &EvolutionManager{
		config:            config,
		population:        make(map[string]*StrategyGenome),
		generationHistory: make([]*EvolutionGeneration, 0),
		eventBus:          eventBus,
		ctx:               ctx,
		cancel:            cancel,
		stats: &EvolutionStats{
			LastEvolutionTime: time.Now(),
		},
	}
}

// Start 启动进化管理器
func (em *EvolutionManager) Start() error {
	em.runningMu.Lock()
	defer em.runningMu.Unlock()

	if em.isRunning {
		return fmt.Errorf("evolution manager is already running")
	}

	log.Println("启动策略进化管理器...")

	// 初始化种群
	if err := em.initializePopulation(); err != nil {
		return fmt.Errorf("failed to initialize population: %w", err)
	}

	// 启动进化循环
	em.wg.Add(1)
	go em.runEvolutionLoop()

	em.isRunning = true

	// 发送启动事件
	em.emitEvent("evolution_manager_started", map[string]interface{}{
		"population_size": len(em.population),
		"generation":      0,
	})

	log.Println("策略进化管理器启动完成")
	return nil
}

// Stop 停止进化管理器
func (em *EvolutionManager) Stop() error {
	em.runningMu.Lock()
	defer em.runningMu.Unlock()

	if !em.isRunning {
		return nil
	}

	log.Println("停止策略进化管理器...")

	// 取消上下文
	em.cancel()

	// 等待进化循环结束
	em.wg.Wait()

	em.isRunning = false

	// 发送停止事件
	em.emitEvent("evolution_manager_stopped", map[string]interface{}{
		"final_generation": em.getCurrentGenerationNumber(),
		"total_strategies": len(em.population),
	})

	log.Println("策略进化管理器已停止")
	return nil
}

// initializePopulation 初始化种群
func (em *EvolutionManager) initializePopulation() error {
	em.populationMu.Lock()
	defer em.populationMu.Unlock()

	// 创建初始种群
	for i := 0; i < em.config.PopulationSize; i++ {
		genome := em.createRandomGenome(0) // 第0代
		em.population[genome.ID] = genome
	}

	// 创建第一代
	em.currentGeneration = &EvolutionGeneration{
		Number:     0,
		Strategies: make(map[string]*StrategyGenome),
		CreatedAt:  time.Now(),
	}

	for id, genome := range em.population {
		em.currentGeneration.Strategies[id] = genome
	}

	log.Printf("初始化种群完成，共 %d 个策略", len(em.population))
	return nil
}

// runEvolutionLoop 运行进化循环
func (em *EvolutionManager) runEvolutionLoop() {
	defer em.wg.Done()

	ticker := time.NewTicker(em.config.EvaluationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-em.ctx.Done():
			return
		case <-ticker.C:
			if err := em.performEvolution(); err != nil {
				log.Printf("Error during evolution: %v", err)
			}
		}
	}
}

// performEvolution 执行进化
func (em *EvolutionManager) performEvolution() error {
	log.Println("开始执行策略进化...")

	startTime := time.Now()

	// 1. 评估当前种群
	if err := em.evaluatePopulation(); err != nil {
		return fmt.Errorf("failed to evaluate population: %w", err)
	}

	// 2. 选择优秀个体
	elites := em.selectElites()

	// 3. 淘汰劣质个体
	eliminated := em.eliminateWorstPerformers()

	// 4. 生成新个体
	newStrategies := em.generateNewStrategies(elites)

	// 5. 更新种群
	em.updatePopulation(elites, newStrategies)

	// 6. 创建新一代
	em.createNextGeneration()

	// 7. 更新统计信息
	em.updateEvolutionStats()

	duration := time.Since(startTime)

	log.Printf("策略进化完成 - 代数: %d, 淘汰: %d, 新增: %d, 耗时: %v",
		em.getCurrentGenerationNumber(), len(eliminated), len(newStrategies), duration)

	// 发送进化完成事件
	em.emitEvent("evolution_completed", map[string]interface{}{
		"generation":       em.getCurrentGenerationNumber(),
		"eliminated_count": len(eliminated),
		"new_strategies":   len(newStrategies),
		"duration":         duration.String(),
		"best_fitness":     em.currentGeneration.BestFitness,
		"average_fitness":  em.currentGeneration.AvgFitness,
	})

	return nil
}

// createRandomGenome 创建随机基因组
func (em *EvolutionManager) createRandomGenome(generation int) *StrategyGenome {
	genome := &StrategyGenome{
		ID:         fmt.Sprintf("strategy_%d_%d", generation, time.Now().UnixNano()),
		Generation: generation,
		Parents:    []string{},
		Genes:      make(map[string]float64),
		CreatedAt:  time.Now(),
	}

	// 生成随机基因
	geneNames := []string{
		"risk_tolerance", "position_size", "stop_loss", "take_profit",
		"entry_threshold", "exit_threshold", "volatility_factor", "momentum_factor",
	}

	for _, geneName := range geneNames {
		genome.Genes[geneName] = em.randomFloat64(0.1, 2.0)
	}

	return genome
}

// randomFloat64 生成随机浮点数
func (em *EvolutionManager) randomFloat64(min, max float64) float64 {
	return min + (max-min)*float64(time.Now().UnixNano()%1000)/1000.0
}

// GetDefaultEvolutionConfig 获取默认进化配置
func GetDefaultEvolutionConfig() *EvolutionConfig {
	return &EvolutionConfig{
		EvaluationInterval:   30 * time.Minute,
		MinEvaluationPeriod:  24 * time.Hour,
		PerformanceWindow:    7 * 24 * time.Hour,
		MutationRate:         0.1,
		CrossoverRate:        0.7,
		ElitismRate:          0.2,
		PopulationSize:       20,
		EliminationThreshold: -0.1,
		MinSurvivalRate:      0.3,
		MaxGenerations:       100,
		MinSharpeRatio:       0.5,
		MaxDrawdownLimit:     0.3,
		MinWinRate:           0.4,
		MinProfitFactor:      1.1,
	}
}

// evaluatePopulation 评估种群
func (em *EvolutionManager) evaluatePopulation() error {
	em.populationMu.RLock()
	strategies := make([]*StrategyGenome, 0, len(em.population))
	for _, genome := range em.population {
		strategies = append(strategies, genome)
	}
	em.populationMu.RUnlock()

	// 并发评估策略
	var wg sync.WaitGroup
	for _, strategy := range strategies {
		wg.Add(1)
		go func(genome *StrategyGenome) {
			defer wg.Done()
			em.evaluateStrategy(genome)
		}(strategy)
	}

	wg.Wait()
	log.Printf("种群评估完成，共评估 %d 个策略", len(strategies))
	return nil
}

// evaluateStrategy 评估单个策略
func (em *EvolutionManager) evaluateStrategy(genome *StrategyGenome) {
	// 模拟策略性能评估
	// 实际应用中应该基于真实的回测和交易数据

	// 基于基因计算适应度
	fitness := em.calculateFitness(genome)

	// 创建性能指标
	performance := &PerformanceMetrics{
		SharpeRatio:  fitness * 2.0,
		SortinoRatio: fitness * 2.2,
		MaxDrawdown:  0.1 + (1.0-fitness)*0.2,
		TotalReturn:  fitness * 0.3,
		WinRate:      0.4 + fitness*0.3,
		ProfitFactor: 1.0 + fitness*0.5,
		LastUpdated:  time.Now(),
		UpdateCount:  int64(genome.EvaluationCount + 1),
	}

	// 更新基因组
	genome.Fitness = fitness
	genome.Performance = performance
	genome.LastEvaluated = time.Now()
	genome.EvaluationCount++

	log.Printf("策略 %s 评估完成: 适应度=%.3f, 夏普比率=%.3f",
		genome.ID, fitness, performance.SharpeRatio)
}

// calculateFitness 计算适应度
func (em *EvolutionManager) calculateFitness(genome *StrategyGenome) float64 {
	// 基于基因的简化适应度计算
	// 实际应用中应该基于策略的历史表现

	fitness := 0.0
	geneCount := 0

	// 基因权重
	geneWeights := map[string]float64{
		"risk_tolerance":    0.15,
		"position_size":     0.12,
		"stop_loss":         0.18,
		"take_profit":       0.15,
		"entry_threshold":   0.10,
		"exit_threshold":    0.10,
		"volatility_factor": 0.10,
		"momentum_factor":   0.10,
	}

	for geneName, geneValue := range genome.Genes {
		if weight, exists := geneWeights[geneName]; exists {
			// 使用正态分布计算基因贡献
			optimalValue := 1.0 // 假设最优值为1.0
			deviation := math.Abs(geneValue - optimalValue)
			geneContribution := math.Exp(-deviation*deviation) * weight
			fitness += geneContribution
			geneCount++
		}
	}

	// 归一化适应度
	if geneCount > 0 {
		fitness = fitness / float64(geneCount) * float64(len(geneWeights))
	}

	// 添加随机噪声模拟市场不确定性
	noise := (em.randomFloat64(0.8, 1.2) - 1.0) * 0.1
	fitness += noise

	// 确保适应度在合理范围内
	if fitness < 0 {
		fitness = 0
	}
	if fitness > 1 {
		fitness = 1
	}

	return fitness
}

// selectElites 选择精英个体
func (em *EvolutionManager) selectElites() []*StrategyGenome {
	em.populationMu.RLock()
	defer em.populationMu.RUnlock()

	// 按适应度排序
	strategies := make([]*StrategyGenome, 0, len(em.population))
	for _, genome := range em.population {
		strategies = append(strategies, genome)
	}

	sort.Slice(strategies, func(i, j int) bool {
		return strategies[i].Fitness > strategies[j].Fitness
	})

	// 选择前N%作为精英
	eliteCount := int(float64(len(strategies)) * em.config.ElitismRate)
	if eliteCount < 1 {
		eliteCount = 1
	}

	elites := strategies[:eliteCount]

	log.Printf("选择了 %d 个精英策略，最高适应度: %.3f",
		len(elites), elites[0].Fitness)

	return elites
}

// eliminateWorstPerformers 淘汰表现最差的个体
func (em *EvolutionManager) eliminateWorstPerformers() []*StrategyGenome {
	em.populationMu.Lock()
	defer em.populationMu.Unlock()

	// 按适应度排序
	strategies := make([]*StrategyGenome, 0, len(em.population))
	for _, genome := range em.population {
		strategies = append(strategies, genome)
	}

	sort.Slice(strategies, func(i, j int) bool {
		return strategies[i].Fitness > strategies[j].Fitness
	})

	// 计算淘汰数量
	minSurvivors := int(float64(len(strategies)) * em.config.MinSurvivalRate)
	eliminateCount := len(strategies) - minSurvivors

	if eliminateCount <= 0 {
		return []*StrategyGenome{}
	}

	// 淘汰表现最差的策略
	eliminated := strategies[len(strategies)-eliminateCount:]

	// 从种群中移除
	for _, genome := range eliminated {
		delete(em.population, genome.ID)
	}

	log.Printf("淘汰了 %d 个表现最差的策略", len(eliminated))

	// 发送淘汰事件
	for _, genome := range eliminated {
		em.emitEvent("strategy_eliminated", map[string]interface{}{
			"strategy_id": genome.ID,
			"fitness":     genome.Fitness,
			"generation":  genome.Generation,
			"reason":      "poor_performance",
		})
	}

	return eliminated
}

// generateNewStrategies 生成新策略
func (em *EvolutionManager) generateNewStrategies(elites []*StrategyGenome) []*StrategyGenome {
	newStrategies := make([]*StrategyGenome, 0)

	// 计算需要生成的新策略数量
	targetPopulation := em.config.PopulationSize
	currentPopulation := len(em.population)
	needCount := targetPopulation - currentPopulation

	if needCount <= 0 {
		return newStrategies
	}

	currentGeneration := em.getCurrentGenerationNumber() + 1

	// 通过交叉和变异生成新策略
	for i := 0; i < needCount; i++ {
		var newGenome *StrategyGenome

		if len(elites) >= 2 && em.randomFloat64(0, 1) < em.config.CrossoverRate {
			// 交叉生成
			parent1 := elites[i%len(elites)]
			parent2 := elites[(i+1)%len(elites)]
			newGenome = em.crossover(parent1, parent2, currentGeneration)
		} else if len(elites) > 0 {
			// 变异生成
			parent := elites[i%len(elites)]
			newGenome = em.mutate(parent, currentGeneration)
		} else {
			// 随机生成
			newGenome = em.createRandomGenome(currentGeneration)
		}

		newStrategies = append(newStrategies, newGenome)
	}

	log.Printf("生成了 %d 个新策略", len(newStrategies))
	return newStrategies
}

// crossover 交叉操作
func (em *EvolutionManager) crossover(parent1, parent2 *StrategyGenome, generation int) *StrategyGenome {
	child := &StrategyGenome{
		ID:         fmt.Sprintf("strategy_%d_%d", generation, time.Now().UnixNano()),
		Generation: generation,
		Parents:    []string{parent1.ID, parent2.ID},
		Genes:      make(map[string]float64),
		CreatedAt:  time.Now(),
	}

	// 单点交叉
	crossoverPoint := len(parent1.Genes) / 2
	geneIndex := 0

	for geneName := range parent1.Genes {
		if geneIndex < crossoverPoint {
			child.Genes[geneName] = parent1.Genes[geneName]
		} else {
			if value, exists := parent2.Genes[geneName]; exists {
				child.Genes[geneName] = value
			} else {
				child.Genes[geneName] = parent1.Genes[geneName]
			}
		}
		geneIndex++
	}

	// 确保所有基因都存在
	for geneName, value := range parent2.Genes {
		if _, exists := child.Genes[geneName]; !exists {
			child.Genes[geneName] = value
		}
	}

	return child
}

// mutate 变异操作
func (em *EvolutionManager) mutate(parent *StrategyGenome, generation int) *StrategyGenome {
	child := &StrategyGenome{
		ID:         fmt.Sprintf("strategy_%d_%d", generation, time.Now().UnixNano()),
		Generation: generation,
		Parents:    []string{parent.ID},
		Genes:      make(map[string]float64),
		CreatedAt:  time.Now(),
	}

	// 复制父代基因
	for geneName, geneValue := range parent.Genes {
		child.Genes[geneName] = geneValue
	}

	// 随机变异部分基因
	for geneName, geneValue := range child.Genes {
		if em.randomFloat64(0, 1) < em.config.MutationRate {
			// 高斯变异
			mutationStrength := 0.1
			mutation := (em.randomFloat64(0, 1) - 0.5) * 2 * mutationStrength
			newValue := geneValue + mutation

			// 确保基因值在合理范围内
			if newValue < 0.1 {
				newValue = 0.1
			}
			if newValue > 2.0 {
				newValue = 2.0
			}

			child.Genes[geneName] = newValue
		}
	}

	return child
}

// updatePopulation 更新种群
func (em *EvolutionManager) updatePopulation(elites, newStrategies []*StrategyGenome) {
	em.populationMu.Lock()
	defer em.populationMu.Unlock()

	// 添加新策略到种群
	for _, genome := range newStrategies {
		em.population[genome.ID] = genome
	}

	log.Printf("种群更新完成，当前种群大小: %d", len(em.population))
}

// createNextGeneration 创建下一代
func (em *EvolutionManager) createNextGeneration() {
	em.populationMu.RLock()
	defer em.populationMu.RUnlock()

	// 保存当前代到历史
	if em.currentGeneration != nil {
		em.generationHistory = append(em.generationHistory, em.currentGeneration)
	}

	// 创建新一代
	newGeneration := &EvolutionGeneration{
		Number:     em.getCurrentGenerationNumber() + 1,
		Strategies: make(map[string]*StrategyGenome),
		CreatedAt:  time.Now(),
	}

	// 计算统计信息
	totalFitness := 0.0
	maxFitness := 0.0
	for _, genome := range em.population {
		newGeneration.Strategies[genome.ID] = genome
		totalFitness += genome.Fitness
		if genome.Fitness > maxFitness {
			maxFitness = genome.Fitness
		}
	}

	if len(em.population) > 0 {
		newGeneration.AvgFitness = totalFitness / float64(len(em.population))
	}
	newGeneration.BestFitness = maxFitness
	newGeneration.Diversity = em.calculateDiversity()

	em.currentGeneration = newGeneration

	log.Printf("创建第 %d 代，种群大小: %d, 平均适应度: %.3f, 最高适应度: %.3f",
		newGeneration.Number, len(em.population), newGeneration.AvgFitness, newGeneration.BestFitness)
}

// calculateDiversity 计算种群多样性
func (em *EvolutionManager) calculateDiversity() float64 {
	// 简化的多样性计算
	// 实际应用中可以使用更复杂的遗传多样性指标

	if len(em.population) < 2 {
		return 0.0
	}

	totalDistance := 0.0
	comparisons := 0

	strategies := make([]*StrategyGenome, 0, len(em.population))
	for _, genome := range em.population {
		strategies = append(strategies, genome)
	}

	// 计算策略间的平均距离
	for i := 0; i < len(strategies); i++ {
		for j := i + 1; j < len(strategies); j++ {
			distance := em.calculateGeneticDistance(strategies[i], strategies[j])
			totalDistance += distance
			comparisons++
		}
	}

	if comparisons > 0 {
		return totalDistance / float64(comparisons)
	}

	return 0.0
}

// calculateGeneticDistance 计算基因距离
func (em *EvolutionManager) calculateGeneticDistance(genome1, genome2 *StrategyGenome) float64 {
	distance := 0.0
	geneCount := 0

	for geneName, value1 := range genome1.Genes {
		if value2, exists := genome2.Genes[geneName]; exists {
			distance += math.Abs(value1 - value2)
			geneCount++
		}
	}

	if geneCount > 0 {
		return distance / float64(geneCount)
	}

	return 0.0
}

// getCurrentGenerationNumber 获取当前代数
func (em *EvolutionManager) getCurrentGenerationNumber() int {
	if em.currentGeneration != nil {
		return em.currentGeneration.Number
	}
	return 0
}

// updateEvolutionStats 更新进化统计信息
func (em *EvolutionManager) updateEvolutionStats() {
	em.statsMu.Lock()
	defer em.statsMu.Unlock()

	em.stats.TotalGenerations = int64(len(em.generationHistory))
	em.stats.TotalStrategies = int64(len(em.population))
	em.stats.ActiveStrategies = int64(len(em.population))
	em.stats.LastEvolutionTime = time.Now()

	if em.currentGeneration != nil {
		em.stats.BestFitness = em.currentGeneration.BestFitness
		em.stats.AverageFitness = em.currentGeneration.AvgFitness
		em.stats.DiversityIndex = em.currentGeneration.Diversity
	}
}

// emitEvent 发送事件
func (em *EvolutionManager) emitEvent(eventType string, data map[string]interface{}) {
	if em.eventBus == nil {
		return
	}

	event := &events.Event{
		Type:      events.EventType(eventType),
		Source:    "evolution_manager",
		Data:      data,
		Timestamp: time.Now(),
	}

	if err := em.eventBus.Publish(event); err != nil {
		log.Printf("Warning: failed to emit event %s: %v", eventType, err)
	}
}

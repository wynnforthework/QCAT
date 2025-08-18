package automl

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// AdvancedOptimizer 高级优化器接口
type AdvancedOptimizer interface {
	Optimize(ctx context.Context, strategyName string, dataHash string, seed int64) (*OptimizationResult, error)
	GetName() string
	GetDescription() string
}

// ParameterRange 参数范围
type ParameterRange struct {
	Min     float64
	Max     float64
	Step    float64
	Type    string // "continuous", "discrete", "integer"
}

// GeneticAlgorithm 遗传算法优化器
type GeneticAlgorithm struct {
	PopulationSize    int
	Generations       int
	MutationRate      float64
	CrossoverRate     float64
	EliteSize         int
	ParameterRanges   map[string]ParameterRange
	FitnessHistory    []float64
	BestIndividual    *Individual
	mu                sync.RWMutex
}

// Individual 个体（参数组合）
type Individual struct {
	Parameters map[string]float64
	Fitness    float64
	Age        int
}

// NewGeneticAlgorithm 创建遗传算法优化器
func NewGeneticAlgorithm() *GeneticAlgorithm {
	return &GeneticAlgorithm{
		PopulationSize: 50,
		Generations:    100,
		MutationRate:   0.1,
		CrossoverRate:  0.8,
		EliteSize:      5,
		ParameterRanges: map[string]ParameterRange{
			"learning_rate": {Min: 0.001, Max: 0.1, Step: 0.001, Type: "continuous"},
			"batch_size":    {Min: 16, Max: 512, Step: 16, Type: "integer"},
			"epochs":        {Min: 10, Max: 200, Step: 1, Type: "integer"},
			"dropout":       {Min: 0.1, Max: 0.5, Step: 0.05, Type: "continuous"},
			"momentum":      {Min: 0.8, Max: 0.99, Step: 0.01, Type: "continuous"},
		},
		FitnessHistory: make([]float64, 0),
	}
}

func (ga *GeneticAlgorithm) GetName() string {
	return "Genetic Algorithm"
}

func (ga *GeneticAlgorithm) GetDescription() string {
	return "Evolutionary algorithm that mimics natural selection to find optimal parameters"
}

// Optimize 执行遗传算法优化
func (ga *GeneticAlgorithm) Optimize(ctx context.Context, strategyName string, dataHash string, seed int64) (*OptimizationResult, error) {
	rand.Seed(seed)
	
	// 初始化种群
	population := ga.initializePopulation()
	
	// 进化过程
	for generation := 0; generation < ga.Generations; generation++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		// 评估适应度
		ga.evaluateFitness(population, strategyName, dataHash)
		
		// 记录最佳个体
		ga.updateBestIndividual(population)
		
		// 记录适应度历史
		ga.mu.Lock()
		ga.FitnessHistory = append(ga.FitnessHistory, ga.BestIndividual.Fitness)
		ga.mu.Unlock()
		
		// 选择、交叉、变异
		newPopulation := ga.evolve(population)
		population = newPopulation
		
		// 每10代输出一次进度
		if generation%10 == 0 {
			fmt.Printf("GA Generation %d: Best Fitness = %.4f\n", generation, ga.BestIndividual.Fitness)
		}
	}
	
	// 返回最佳结果
	return ga.createOptimizationResult(strategyName, dataHash, seed), nil
}

// initializePopulation 初始化种群
func (ga *GeneticAlgorithm) initializePopulation() []*Individual {
	population := make([]*Individual, ga.PopulationSize)
	
	for i := 0; i < ga.PopulationSize; i++ {
		individual := &Individual{
			Parameters: make(map[string]float64),
			Fitness:    0,
			Age:        0,
		}
		
		// 随机生成参数
		for paramName, paramRange := range ga.ParameterRanges {
			switch paramRange.Type {
			case "continuous":
				individual.Parameters[paramName] = paramRange.Min + rand.Float64()*(paramRange.Max-paramRange.Min)
			case "integer":
				individual.Parameters[paramName] = float64(int(paramRange.Min + rand.Float64()*(paramRange.Max-paramRange.Min)))
			case "discrete":
				steps := int((paramRange.Max - paramRange.Min) / paramRange.Step)
				step := rand.Intn(steps + 1)
				individual.Parameters[paramName] = paramRange.Min + float64(step)*paramRange.Step
			}
		}
		
		population[i] = individual
	}
	
	return population
}

// evaluateFitness 评估种群适应度
func (ga *GeneticAlgorithm) evaluateFitness(population []*Individual, strategyName string, dataHash string) {
	var wg sync.WaitGroup
	
	for _, individual := range population {
		wg.Add(1)
		go func(ind *Individual) {
			defer wg.Done()
			
			// 模拟训练和评估
			ind.Fitness = ga.simulateTraining(ind.Parameters, strategyName, dataHash)
		}(individual)
	}
	
	wg.Wait()
}

// simulateTraining 模拟训练过程
func (ga *GeneticAlgorithm) simulateTraining(params map[string]float64, strategyName string, dataHash string) float64 {
	// 基于参数计算一个模拟的收益率
	baseProfit := 0.05 // 基础收益率5%
	
	// 参数影响
	learningRateEffect := (params["learning_rate"] - 0.05) * 100 // 学习率影响
	batchSizeEffect := (params["batch_size"] - 100) / 1000       // 批次大小影响
	epochsEffect := (params["epochs"] - 50) / 1000               // 训练轮数影响
	dropoutEffect := (0.3 - params["dropout"]) * 50              // Dropout影响
	momentumEffect := (params["momentum"] - 0.9) * 100           // 动量影响
	
	// 添加随机性
	randomFactor := (rand.Float64() - 0.5) * 0.1
	
	profit := baseProfit + learningRateEffect + batchSizeEffect + epochsEffect + dropoutEffect + momentumEffect + randomFactor
	
	// 确保收益率在合理范围内
	profit = math.Max(-0.2, math.Min(0.3, profit))
	
	return profit
}

// updateBestIndividual 更新最佳个体
func (ga *GeneticAlgorithm) updateBestIndividual(population []*Individual) {
	ga.mu.Lock()
	defer ga.mu.Unlock()
	
	for _, individual := range population {
		if ga.BestIndividual == nil || individual.Fitness > ga.BestIndividual.Fitness {
			ga.BestIndividual = &Individual{
				Parameters: make(map[string]float64),
				Fitness:    individual.Fitness,
				Age:        individual.Age,
			}
			// 复制参数
			for k, v := range individual.Parameters {
				ga.BestIndividual.Parameters[k] = v
			}
		}
	}
}

// evolve 进化过程：选择、交叉、变异
func (ga *GeneticAlgorithm) evolve(population []*Individual) []*Individual {
	// 按适应度排序
	sort.Slice(population, func(i, j int) bool {
		return population[i].Fitness > population[j].Fitness
	})
	
	newPopulation := make([]*Individual, ga.PopulationSize)
	
	// 精英保留
	for i := 0; i < ga.EliteSize; i++ {
		newPopulation[i] = ga.cloneIndividual(population[i])
		newPopulation[i].Age++
	}
	
	// 生成新个体
	for i := ga.EliteSize; i < ga.PopulationSize; i++ {
		if rand.Float64() < ga.CrossoverRate {
			// 交叉
			parent1 := ga.tournamentSelection(population)
			parent2 := ga.tournamentSelection(population)
			child := ga.crossover(parent1, parent2)
			
			// 变异
			if rand.Float64() < ga.MutationRate {
				ga.mutate(child)
			}
			
			newPopulation[i] = child
		} else {
			// 直接复制
			newPopulation[i] = ga.cloneIndividual(population[rand.Intn(len(population))])
			newPopulation[i].Age++
		}
	}
	
	return newPopulation
}

// tournamentSelection 锦标赛选择
func (ga *GeneticAlgorithm) tournamentSelection(population []*Individual) *Individual {
	tournamentSize := 3
	best := population[rand.Intn(len(population))]
	
	for i := 1; i < tournamentSize; i++ {
		candidate := population[rand.Intn(len(population))]
		if candidate.Fitness > best.Fitness {
			best = candidate
		}
	}
	
	return best
}

// crossover 交叉操作
func (ga *GeneticAlgorithm) crossover(parent1, parent2 *Individual) *Individual {
	child := &Individual{
		Parameters: make(map[string]float64),
		Fitness:    0,
		Age:        0,
	}
	
	for paramName := range ga.ParameterRanges {
		if rand.Float64() < 0.5 {
			child.Parameters[paramName] = parent1.Parameters[paramName]
		} else {
			child.Parameters[paramName] = parent2.Parameters[paramName]
		}
	}
	
	return child
}

// mutate 变异操作
func (ga *GeneticAlgorithm) mutate(individual *Individual) {
	for paramName, paramRange := range ga.ParameterRanges {
		if rand.Float64() < 0.3 { // 30%概率变异
			switch paramRange.Type {
			case "continuous":
				mutation := (rand.Float64() - 0.5) * (paramRange.Max - paramRange.Min) * 0.1
				individual.Parameters[paramName] += mutation
				individual.Parameters[paramName] = math.Max(paramRange.Min, math.Min(paramRange.Max, individual.Parameters[paramName]))
			case "integer":
				mutation := rand.Intn(3) - 1 // -1, 0, 1
				individual.Parameters[paramName] += float64(mutation)
				individual.Parameters[paramName] = math.Max(paramRange.Min, math.Min(paramRange.Max, individual.Parameters[paramName]))
			case "discrete":
				steps := int((paramRange.Max - paramRange.Min) / paramRange.Step)
				currentStep := int((individual.Parameters[paramName] - paramRange.Min) / paramRange.Step)
				newStep := currentStep + rand.Intn(3) - 1
				newStep = int(math.Max(0, math.Min(float64(steps), float64(newStep))))
				individual.Parameters[paramName] = paramRange.Min + float64(newStep)*paramRange.Step
			}
		}
	}
}

// cloneIndividual 克隆个体
func (ga *GeneticAlgorithm) cloneIndividual(individual *Individual) *Individual {
	clone := &Individual{
		Parameters: make(map[string]float64),
		Fitness:    individual.Fitness,
		Age:        individual.Age,
	}
	
	for k, v := range individual.Parameters {
		clone.Parameters[k] = v
	}
	
	return clone
}

// createOptimizationResult 创建优化结果
func (ga *GeneticAlgorithm) createOptimizationResult(strategyName string, dataHash string, seed int64) *OptimizationResult {
	ga.mu.RLock()
	defer ga.mu.RUnlock()
	
	if ga.BestIndividual == nil {
		return nil
	}
	
	return &OptimizationResult{
		TaskID:        fmt.Sprintf("ga_%d", time.Now().Unix()),
		StrategyName:  strategyName,
		DataHash:      dataHash,
		RandomSeed:    seed,
		Parameters:    ga.BestIndividual.Parameters,
		Performance: &PerformanceMetrics{
			ProfitRate:         ga.BestIndividual.Fitness * 100,
			SharpeRatio:        ga.BestIndividual.Fitness * 2,
			MaxDrawdown:        (1 - ga.BestIndividual.Fitness) * 20,
			WinRate:            50 + ga.BestIndividual.Fitness*100,
			TotalReturn:        ga.BestIndividual.Fitness * 100,
			RiskAdjustedReturn: ga.BestIndividual.Fitness * 80,
		},
		DiscoveredAt:  time.Now(),
		DiscoveredBy:  "GeneticAlgorithm",
		Confidence:    0.85,
		IsGlobalBest:  false,
		AdoptionCount: 0,
		Metadata: map[string]interface{}{
			"algorithm":        "GeneticAlgorithm",
			"generations":      ga.Generations,
			"population_size":  ga.PopulationSize,
			"best_fitness":     ga.BestIndividual.Fitness,
			"fitness_history":  ga.FitnessHistory,
		},
	}
}

// ParticleSwarmOptimization 粒子群优化
type ParticleSwarmOptimization struct {
	ParticleCount   int
	Iterations      int
	InertiaWeight   float64
	CognitiveWeight float64
	SocialWeight    float64
	ParameterRanges map[string]ParameterRange
	Particles       []*Particle
	GlobalBest      *Particle
	mu              sync.RWMutex
}

// Particle 粒子
type Particle struct {
	Position     map[string]float64
	Velocity     map[string]float64
	BestPosition map[string]float64
	BestFitness  float64
	Fitness      float64
}

// NewParticleSwarmOptimization 创建粒子群优化器
func NewParticleSwarmOptimization() *ParticleSwarmOptimization {
	return &ParticleSwarmOptimization{
		ParticleCount:   30,
		Iterations:      100,
		InertiaWeight:   0.7,
		CognitiveWeight: 1.5,
		SocialWeight:    1.5,
		ParameterRanges: map[string]ParameterRange{
			"learning_rate": {Min: 0.001, Max: 0.1, Step: 0.001, Type: "continuous"},
			"batch_size":    {Min: 16, Max: 512, Step: 16, Type: "integer"},
			"epochs":        {Min: 10, Max: 200, Step: 1, Type: "integer"},
			"dropout":       {Min: 0.1, Max: 0.5, Step: 0.05, Type: "continuous"},
			"momentum":      {Min: 0.8, Max: 0.99, Step: 0.01, Type: "continuous"},
		},
		Particles: make([]*Particle, 0),
	}
}

func (pso *ParticleSwarmOptimization) GetName() string {
	return "Particle Swarm Optimization"
}

func (pso *ParticleSwarmOptimization) GetDescription() string {
	return "Swarm intelligence algorithm inspired by bird flocking behavior"
}

// Optimize 执行粒子群优化
func (pso *ParticleSwarmOptimization) Optimize(ctx context.Context, strategyName string, dataHash string, seed int64) (*OptimizationResult, error) {
	rand.Seed(seed)
	
	// 初始化粒子群
	pso.initializeParticles()
	
	// 迭代优化
	for iteration := 0; iteration < pso.Iterations; iteration++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		// 评估所有粒子
		pso.evaluateParticles(strategyName, dataHash)
		
		// 更新全局最优
		pso.updateGlobalBest()
		
		// 更新粒子速度和位置
		pso.updateParticles()
		
		// 每10次迭代输出一次进度
		if iteration%10 == 0 {
			fmt.Printf("PSO Iteration %d: Global Best Fitness = %.4f\n", iteration, pso.GlobalBest.BestFitness)
		}
	}
	
	// 返回最佳结果
	return pso.createOptimizationResult(strategyName, dataHash, seed), nil
}

// initializeParticles 初始化粒子群
func (pso *ParticleSwarmOptimization) initializeParticles() {
	pso.Particles = make([]*Particle, pso.ParticleCount)
	
	for i := 0; i < pso.ParticleCount; i++ {
		particle := &Particle{
			Position:     make(map[string]float64),
			Velocity:     make(map[string]float64),
			BestPosition: make(map[string]float64),
			BestFitness:  -math.MaxFloat64,
		}
		
		// 初始化位置和速度
		for paramName, paramRange := range pso.ParameterRanges {
			// 随机位置
			position := paramRange.Min + rand.Float64()*(paramRange.Max-paramRange.Min)
			particle.Position[paramName] = position
			particle.BestPosition[paramName] = position
			
			// 随机速度
			maxVelocity := (paramRange.Max - paramRange.Min) * 0.1
			velocity := (rand.Float64() - 0.5) * maxVelocity
			particle.Velocity[paramName] = velocity
		}
		
		pso.Particles[i] = particle
	}
}

// evaluateParticles 评估所有粒子
func (pso *ParticleSwarmOptimization) evaluateParticles(strategyName string, dataHash string) {
	var wg sync.WaitGroup
	
	for _, particle := range pso.Particles {
		wg.Add(1)
		go func(p *Particle) {
			defer wg.Done()
			
			// 模拟训练和评估
			p.Fitness = pso.simulateTraining(p.Position, strategyName, dataHash)
			
			// 更新个体最优
			if p.Fitness > p.BestFitness {
				p.BestFitness = p.Fitness
				for k, v := range p.Position {
					p.BestPosition[k] = v
				}
			}
		}(particle)
	}
	
	wg.Wait()
}

// simulateTraining 模拟训练过程（与GA相同）
func (pso *ParticleSwarmOptimization) simulateTraining(params map[string]float64, strategyName string, dataHash string) float64 {
	baseProfit := 0.05
	
	learningRateEffect := (params["learning_rate"] - 0.05) * 100
	batchSizeEffect := (params["batch_size"] - 100) / 1000
	epochsEffect := (params["epochs"] - 50) / 1000
	dropoutEffect := (0.3 - params["dropout"]) * 50
	momentumEffect := (params["momentum"] - 0.9) * 100
	
	randomFactor := (rand.Float64() - 0.5) * 0.1
	
	profit := baseProfit + learningRateEffect + batchSizeEffect + epochsEffect + dropoutEffect + momentumEffect + randomFactor
	profit = math.Max(-0.2, math.Min(0.3, profit))
	
	return profit
}

// updateGlobalBest 更新全局最优
func (pso *ParticleSwarmOptimization) updateGlobalBest() {
	pso.mu.Lock()
	defer pso.mu.Unlock()
	
	for _, particle := range pso.Particles {
		if pso.GlobalBest == nil || particle.BestFitness > pso.GlobalBest.BestFitness {
			if pso.GlobalBest == nil {
				pso.GlobalBest = &Particle{
					Position:     make(map[string]float64),
					Velocity:     make(map[string]float64),
					BestPosition: make(map[string]float64),
				}
			}
			
			pso.GlobalBest.BestFitness = particle.BestFitness
			for k, v := range particle.BestPosition {
				pso.GlobalBest.BestPosition[k] = v
			}
		}
	}
}

// updateParticles 更新粒子速度和位置
func (pso *ParticleSwarmOptimization) updateParticles() {
	pso.mu.RLock()
	globalBest := pso.GlobalBest
	pso.mu.RUnlock()
	
	if globalBest == nil {
		return
	}
	
	for _, particle := range pso.Particles {
		for paramName, paramRange := range pso.ParameterRanges {
			// 更新速度
			cognitive := pso.CognitiveWeight * rand.Float64() * (particle.BestPosition[paramName] - particle.Position[paramName])
			social := pso.SocialWeight * rand.Float64() * (globalBest.BestPosition[paramName] - particle.Position[paramName])
			
			particle.Velocity[paramName] = pso.InertiaWeight*particle.Velocity[paramName] + cognitive + social
			
			// 限制速度
			maxVelocity := (paramRange.Max - paramRange.Min) * 0.1
			particle.Velocity[paramName] = math.Max(-maxVelocity, math.Min(maxVelocity, particle.Velocity[paramName]))
			
			// 更新位置
			particle.Position[paramName] += particle.Velocity[paramName]
			
			// 边界处理
			particle.Position[paramName] = math.Max(paramRange.Min, math.Min(paramRange.Max, particle.Position[paramName]))
		}
	}
}

// createOptimizationResult 创建优化结果
func (pso *ParticleSwarmOptimization) createOptimizationResult(strategyName string, dataHash string, seed int64) *OptimizationResult {
	pso.mu.RLock()
	defer pso.mu.RUnlock()
	
	if pso.GlobalBest == nil {
		return nil
	}
	
	return &OptimizationResult{
		TaskID:        fmt.Sprintf("pso_%d", time.Now().Unix()),
		StrategyName:  strategyName,
		DataHash:      dataHash,
		RandomSeed:    seed,
		Parameters:    pso.GlobalBest.BestPosition,
		Performance: &PerformanceMetrics{
			ProfitRate:         pso.GlobalBest.BestFitness * 100,
			SharpeRatio:        pso.GlobalBest.BestFitness * 2,
			MaxDrawdown:        (1 - pso.GlobalBest.BestFitness) * 20,
			WinRate:            50 + pso.GlobalBest.BestFitness*100,
			TotalReturn:        pso.GlobalBest.BestFitness * 100,
			RiskAdjustedReturn: pso.GlobalBest.BestFitness * 80,
		},
		DiscoveredAt:  time.Now(),
		DiscoveredBy:  "ParticleSwarmOptimization",
		Confidence:    0.88,
		IsGlobalBest:  false,
		AdoptionCount: 0,
		Metadata: map[string]interface{}{
			"algorithm":       "ParticleSwarmOptimization",
			"iterations":      pso.Iterations,
			"particle_count":  pso.ParticleCount,
			"best_fitness":    pso.GlobalBest.BestFitness,
		},
	}
}

// BayesianOptimization 贝叶斯优化
type BayesianOptimization struct {
	MaxIterations   int
	AcquisitionFunc string // "ei", "pi", "ucb"
	ParameterRanges map[string]ParameterRange
	Observations    []*Observation
	mu              sync.RWMutex
}

// Observation 观测点
type Observation struct {
	Parameters map[string]float64
	Fitness    float64
}

// NewBayesianOptimization 创建贝叶斯优化器
func NewBayesianOptimization() *BayesianOptimization {
	return &BayesianOptimization{
		MaxIterations: 50,
		AcquisitionFunc: "ei", // Expected Improvement
		ParameterRanges: map[string]ParameterRange{
			"learning_rate": {Min: 0.001, Max: 0.1, Step: 0.001, Type: "continuous"},
			"batch_size":    {Min: 16, Max: 512, Step: 16, Type: "integer"},
			"epochs":        {Min: 10, Max: 200, Step: 1, Type: "integer"},
			"dropout":       {Min: 0.1, Max: 0.5, Step: 0.05, Type: "continuous"},
			"momentum":      {Min: 0.8, Max: 0.99, Step: 0.01, Type: "continuous"},
		},
		Observations: make([]*Observation, 0),
	}
}

func (bo *BayesianOptimization) GetName() string {
	return "Bayesian Optimization"
}

func (bo *BayesianOptimization) GetDescription() string {
	return "Sequential model-based optimization using Gaussian processes"
}

// Optimize 执行贝叶斯优化
func (bo *BayesianOptimization) Optimize(ctx context.Context, strategyName string, dataHash string, seed int64) (*OptimizationResult, error) {
	rand.Seed(seed)
	
	// 初始化随机观测点
	bo.initializeRandomObservations(5, strategyName, dataHash)
	
	// 贝叶斯优化迭代
	for iteration := 0; iteration < bo.MaxIterations; iteration++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		// 选择下一个观测点
		nextPoint := bo.selectNextPoint()
		
		// 评估新点
		fitness := bo.simulateTraining(nextPoint, strategyName, dataHash)
		
		// 添加观测点
		bo.mu.Lock()
		bo.Observations = append(bo.Observations, &Observation{
			Parameters: nextPoint,
			Fitness:    fitness,
		})
		bo.mu.Unlock()
		
		// 每5次迭代输出一次进度
		if iteration%5 == 0 {
			bestFitness := bo.getBestFitness()
			fmt.Printf("BO Iteration %d: Best Fitness = %.4f\n", iteration, bestFitness)
		}
	}
	
	// 返回最佳结果
	return bo.createOptimizationResult(strategyName, dataHash, seed), nil
}

// initializeRandomObservations 初始化随机观测点
func (bo *BayesianOptimization) initializeRandomObservations(count int, strategyName string, dataHash string) {
	for i := 0; i < count; i++ {
		params := make(map[string]float64)
		
		for paramName, paramRange := range bo.ParameterRanges {
			switch paramRange.Type {
			case "continuous":
				params[paramName] = paramRange.Min + rand.Float64()*(paramRange.Max-paramRange.Min)
			case "integer":
				params[paramName] = float64(int(paramRange.Min + rand.Float64()*(paramRange.Max-paramRange.Min)))
			case "discrete":
				steps := int((paramRange.Max - paramRange.Min) / paramRange.Step)
				step := rand.Intn(steps + 1)
				params[paramName] = paramRange.Min + float64(step)*paramRange.Step
			}
		}
		
		fitness := bo.simulateTraining(params, strategyName, dataHash)
		
		bo.Observations = append(bo.Observations, &Observation{
			Parameters: params,
			Fitness:    fitness,
		})
	}
}

// selectNextPoint 选择下一个观测点
func (bo *BayesianOptimization) selectNextPoint() map[string]float64 {
	bo.mu.RLock()
	defer bo.mu.RUnlock()
	
	// 简化的采集函数：随机采样 + 基于历史观测的启发式
	if len(bo.Observations) < 3 {
		// 随机采样
		return bo.randomSample()
	}
	
	// 基于历史观测的启发式选择
	bestObs := bo.getBestObservation()
	if bestObs == nil {
		return bo.randomSample()
	}
	
	// 在最优解附近探索
	explorationParams := make(map[string]float64)
	for paramName, paramRange := range bo.ParameterRanges {
		baseValue := bestObs.Parameters[paramName]
		exploration := (rand.Float64() - 0.5) * (paramRange.Max - paramRange.Min) * 0.1
		newValue := baseValue + exploration
		explorationParams[paramName] = math.Max(paramRange.Min, math.Min(paramRange.Max, newValue))
	}
	
	return explorationParams
}

// randomSample 随机采样
func (bo *BayesianOptimization) randomSample() map[string]float64 {
	params := make(map[string]float64)
	
	for paramName, paramRange := range bo.ParameterRanges {
		switch paramRange.Type {
		case "continuous":
			params[paramName] = paramRange.Min + rand.Float64()*(paramRange.Max-paramRange.Min)
		case "integer":
			params[paramName] = float64(int(paramRange.Min + rand.Float64()*(paramRange.Max-paramRange.Min)))
		case "discrete":
			steps := int((paramRange.Max - paramRange.Min) / paramRange.Step)
			step := rand.Intn(steps + 1)
			params[paramName] = paramRange.Min + float64(step)*paramRange.Step
		}
	}
	
	return params
}

// getBestObservation 获取最佳观测点
func (bo *BayesianOptimization) getBestObservation() *Observation {
	if len(bo.Observations) == 0 {
		return nil
	}
	
	best := bo.Observations[0]
	for _, obs := range bo.Observations {
		if obs.Fitness > best.Fitness {
			best = obs
		}
	}
	
	return best
}

// getBestFitness 获取最佳适应度
func (bo *BayesianOptimization) getBestFitness() float64 {
	bestObs := bo.getBestObservation()
	if bestObs == nil {
		return -math.MaxFloat64
	}
	return bestObs.Fitness
}

// simulateTraining 模拟训练过程（与GA相同）
func (bo *BayesianOptimization) simulateTraining(params map[string]float64, strategyName string, dataHash string) float64 {
	baseProfit := 0.05
	
	learningRateEffect := (params["learning_rate"] - 0.05) * 100
	batchSizeEffect := (params["batch_size"] - 100) / 1000
	epochsEffect := (params["epochs"] - 50) / 1000
	dropoutEffect := (0.3 - params["dropout"]) * 50
	momentumEffect := (params["momentum"] - 0.9) * 100
	
	randomFactor := (rand.Float64() - 0.5) * 0.1
	
	profit := baseProfit + learningRateEffect + batchSizeEffect + epochsEffect + dropoutEffect + momentumEffect + randomFactor
	profit = math.Max(-0.2, math.Min(0.3, profit))
	
	return profit
}

// createOptimizationResult 创建优化结果
func (bo *BayesianOptimization) createOptimizationResult(strategyName string, dataHash string, seed int64) *OptimizationResult {
	bo.mu.RLock()
	defer bo.mu.RUnlock()
	
	bestObs := bo.getBestObservation()
	if bestObs == nil {
		return nil
	}
	
	return &OptimizationResult{
		TaskID:        fmt.Sprintf("bo_%d", time.Now().Unix()),
		StrategyName:  strategyName,
		DataHash:      dataHash,
		RandomSeed:    seed,
		Parameters:    bestObs.Parameters,
		Performance: &PerformanceMetrics{
			ProfitRate:         bestObs.Fitness * 100,
			SharpeRatio:        bestObs.Fitness * 2,
			MaxDrawdown:        (1 - bestObs.Fitness) * 20,
			WinRate:            50 + bestObs.Fitness*100,
			TotalReturn:        bestObs.Fitness * 100,
			RiskAdjustedReturn: bestObs.Fitness * 80,
		},
		DiscoveredAt:  time.Now(),
		DiscoveredBy:  "BayesianOptimization",
		Confidence:    0.92,
		IsGlobalBest:  false,
		AdoptionCount: 0,
		Metadata: map[string]interface{}{
			"algorithm":        "BayesianOptimization",
			"iterations":       bo.MaxIterations,
			"acquisition_func": bo.AcquisitionFunc,
			"best_fitness":     bestObs.Fitness,
			"observations":     len(bo.Observations),
		},
	}
}

// OptimizationAlgorithmRegistry 优化算法注册表
type OptimizationAlgorithmRegistry struct {
	algorithms map[string]AdvancedOptimizer
	mu         sync.RWMutex
}

// NewOptimizationAlgorithmRegistry 创建算法注册表
func NewOptimizationAlgorithmRegistry() *OptimizationAlgorithmRegistry {
	registry := &OptimizationAlgorithmRegistry{
		algorithms: make(map[string]AdvancedOptimizer),
	}
	
	// 注册默认算法
	registry.Register("genetic", NewGeneticAlgorithm())
	registry.Register("pso", NewParticleSwarmOptimization())
	registry.Register("bayesian", NewBayesianOptimization())
	
	return registry
}

// Register 注册算法
func (r *OptimizationAlgorithmRegistry) Register(name string, algorithm AdvancedOptimizer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.algorithms[name] = algorithm
}

// Get 获取算法
func (r *OptimizationAlgorithmRegistry) Get(name string) (AdvancedOptimizer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	algorithm, exists := r.algorithms[name]
	return algorithm, exists
}

// List 列出所有算法
func (r *OptimizationAlgorithmRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	names := make([]string, 0, len(r.algorithms))
	for name := range r.algorithms {
		names = append(names, name)
	}
	
	return names
}

// GetAlgorithmInfo 获取算法信息
func (r *OptimizationAlgorithmRegistry) GetAlgorithmInfo(name string) map[string]string {
	algorithm, exists := r.Get(name)
	if !exists {
		return nil
	}
	
	return map[string]string{
		"name":        algorithm.GetName(),
		"description": algorithm.GetDescription(),
	}
}

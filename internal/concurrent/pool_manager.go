package concurrent

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// PoolManager 池管理器
type PoolManager struct {
	pools        map[string]*GoroutinePool
	loadBalancer *LoadBalancer
	monitor      *ConcurrencyMonitor
	taskQueue    *TaskQueue
	config       *GoroutinePoolConfig
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup

	// 自动扩缩容
	autoScaler *AutoScaler
}

// AutoScaler 自动扩缩容器
type AutoScaler struct {
	manager  *PoolManager
	enabled  bool
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewPoolManager 创建池管理器
func NewPoolManager(config *GoroutinePoolConfig) *PoolManager {
	if config == nil {
		config = GetDefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	pm := &PoolManager{
		pools:  make(map[string]*GoroutinePool),
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}

	// 初始化组件
	pm.initializePools()
	pm.initializeLoadBalancer()
	pm.initializeTaskQueue()
	pm.initializeMonitor()
	pm.initializeAutoScaler()

	log.Println("池管理器初始化完成")
	return pm
}

// initializePools 初始化所有池
func (pm *PoolManager) initializePools() {
	// 创建默认池
	if pm.config.DefaultPool.Enabled {
		pool := NewGoroutinePool(pm.config.DefaultPool.ToPoolConfig())
		pm.pools["default"] = pool
		log.Printf("创建默认池: %d工作者, %d队列大小",
			pm.config.DefaultPool.MaxWorkers, pm.config.DefaultPool.QueueSize)
	}

	// 创建高优先级池
	if pm.config.HighPriorityPool.Enabled {
		pool := NewGoroutinePool(pm.config.HighPriorityPool.ToPoolConfig())
		pm.pools["high_priority"] = pool
		log.Printf("创建高优先级池: %d工作者, %d队列大小",
			pm.config.HighPriorityPool.MaxWorkers, pm.config.HighPriorityPool.QueueSize)
	}

	// 创建低优先级池
	if pm.config.LowPriorityPool.Enabled {
		pool := NewGoroutinePool(pm.config.LowPriorityPool.ToPoolConfig())
		pm.pools["low_priority"] = pool
		log.Printf("创建低优先级池: %d工作者, %d队列大小",
			pm.config.LowPriorityPool.MaxWorkers, pm.config.LowPriorityPool.QueueSize)
	}

	// 创建策略池
	if pm.config.StrategyPool.Enabled {
		pool := NewGoroutinePool(pm.config.StrategyPool.ToPoolConfig())
		pm.pools["strategy"] = pool
		log.Printf("创建策略池: %d工作者, %d队列大小",
			pm.config.StrategyPool.MaxWorkers, pm.config.StrategyPool.QueueSize)
	}

	// 创建数据处理池
	if pm.config.DataProcessingPool.Enabled {
		pool := NewGoroutinePool(pm.config.DataProcessingPool.ToPoolConfig())
		pm.pools["data_processing"] = pool
		log.Printf("创建数据处理池: %d工作者, %d队列大小",
			pm.config.DataProcessingPool.MaxWorkers, pm.config.DataProcessingPool.QueueSize)
	}
}

// initializeLoadBalancer 初始化负载均衡器
func (pm *PoolManager) initializeLoadBalancer() {
	if !pm.config.LoadBalancer.Enabled {
		return
	}

	strategy := pm.config.LoadBalancer.ToLoadBalanceStrategy()
	pm.loadBalancer = NewLoadBalancer(strategy)

	// 添加所有池到负载均衡器
	for _, pool := range pm.pools {
		pm.loadBalancer.AddPool(pool)
	}

	log.Printf("负载均衡器初始化完成，策略: %s", pm.config.LoadBalancer.Strategy)
}

// initializeTaskQueue 初始化任务队列
func (pm *PoolManager) initializeTaskQueue() {
	if !pm.config.TaskQueue.Enabled {
		return
	}

	pm.taskQueue = NewTaskQueue(pm.config.TaskQueue.MaxSize)
	log.Printf("任务队列初始化完成，最大大小: %d", pm.config.TaskQueue.MaxSize)
}

// initializeMonitor 初始化监控器
func (pm *PoolManager) initializeMonitor() {
	if !pm.config.Monitor.Enabled {
		return
	}

	monitorConfig := &MonitorConfig{
		MonitorInterval: pm.config.Monitor.ToMonitorInterval(),
		AlertThresholds: &AlertThresholds{
			MaxCPUUsage:    pm.config.Monitor.AlertThresholds.MaxCPUUsage,
			MaxMemoryUsage: pm.config.Monitor.AlertThresholds.MaxMemoryUsage,
			MaxQueueLength: pm.config.Monitor.AlertThresholds.MaxQueueLength,
		},
	}

	pm.monitor = NewConcurrencyMonitor(monitorConfig)

	// 添加所有池到监控器
	for _, pool := range pm.pools {
		pm.monitor.AddPool(pool)
	}

	// 设置负载均衡器和任务队列
	if pm.loadBalancer != nil {
		pm.monitor.SetLoadBalancer(pm.loadBalancer)
	}
	if pm.taskQueue != nil {
		pm.monitor.SetTaskQueue(pm.taskQueue)
	}

	log.Printf("监控器初始化完成，监控间隔: %v", monitorConfig.MonitorInterval)
}

// initializeAutoScaler 初始化自动扩缩容器
func (pm *PoolManager) initializeAutoScaler() {
	if !pm.config.AutoScaling.Enabled {
		return
	}

	ctx, cancel := context.WithCancel(pm.ctx)
	pm.autoScaler = &AutoScaler{
		manager:  pm,
		enabled:  true,
		interval: time.Duration(pm.config.AutoScaling.ScaleInterval) * time.Second,
		ctx:      ctx,
		cancel:   cancel,
	}

	log.Printf("自动扩缩容器初始化完成，检查间隔: %v", pm.autoScaler.interval)
}

// Start 启动池管理器
func (pm *PoolManager) Start() error {
	log.Println("启动池管理器...")

	// 启动所有池
	for name, pool := range pm.pools {
		pool.Start()
		log.Printf("池 %s 已启动", name)
	}

	// 启动监控器
	if pm.monitor != nil {
		pm.monitor.Start()
		log.Println("监控器已启动")
	}

	// 启动自动扩缩容器
	if pm.autoScaler != nil {
		pm.wg.Add(1)
		go pm.autoScaler.start(&pm.wg)
		log.Println("自动扩缩容器已启动")
	}

	log.Println("池管理器启动完成")
	return nil
}

// Stop 停止池管理器
func (pm *PoolManager) Stop() error {
	log.Println("停止池管理器...")

	// 取消上下文
	pm.cancel()

	// 停止自动扩缩容器
	if pm.autoScaler != nil {
		pm.autoScaler.cancel()
	}

	// 停止监控器
	if pm.monitor != nil {
		pm.monitor.Stop()
		log.Println("监控器已停止")
	}

	// 关闭任务队列
	if pm.taskQueue != nil {
		pm.taskQueue.Close()
		log.Println("任务队列已关闭")
	}

	// 停止所有池
	for name, pool := range pm.pools {
		pool.Stop()
		log.Printf("池 %s 已停止", name)
	}

	// 等待所有goroutine完成
	pm.wg.Wait()

	log.Println("池管理器停止完成")
	return nil
}

// GetPool 获取指定名称的池
func (pm *PoolManager) GetPool(name string) (*GoroutinePool, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	pool, exists := pm.pools[name]
	if !exists {
		return nil, fmt.Errorf("pool not found: %s", name)
	}

	return pool, nil
}

// SubmitTask 提交任务
func (pm *PoolManager) SubmitTask(task Task, poolName string) error {
	if poolName == "" {
		// 使用负载均衡器
		if pm.loadBalancer != nil {
			return pm.loadBalancer.SubmitTask(task)
		}
		// 默认使用默认池
		poolName = "default"
	}

	pool, err := pm.GetPool(poolName)
	if err != nil {
		return err
	}

	return pool.Submit(task)
}

// SubmitTaskWithPriority 提交带优先级的任务
func (pm *PoolManager) SubmitTaskWithPriority(task Task, priority int) error {
	var poolName string

	// 根据优先级选择池
	switch {
	case priority >= 8:
		poolName = "high_priority"
	case priority >= 5:
		poolName = "default"
	case priority >= 2:
		poolName = "low_priority"
	default:
		poolName = "low_priority"
	}

	return pm.SubmitTask(task, poolName)
}

// GetStats 获取统计信息
func (pm *PoolManager) GetStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := make(map[string]interface{})

	// 池统计信息
	poolStats := make(map[string]interface{})
	for name, pool := range pm.pools {
		poolStats[name] = pool.GetStats()
	}
	stats["pools"] = poolStats

	// 负载均衡器统计信息
	if pm.loadBalancer != nil {
		stats["load_balancer"] = map[string]interface{}{
			"strategy":   pm.config.LoadBalancer.Strategy,
			"pool_count": len(pm.pools),
		}
	}

	// 任务队列统计信息
	if pm.taskQueue != nil {
		stats["task_queue"] = pm.taskQueue.GetStats()
	}

	// 监控器统计信息
	if pm.monitor != nil {
		stats["monitor"] = pm.monitor.GetStats()
	}

	return stats
}

// start 启动自动扩缩容器
func (as *AutoScaler) start(wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(as.interval)
	defer ticker.Stop()

	log.Printf("自动扩缩容器开始运行，检查间隔: %v", as.interval)

	for {
		select {
		case <-as.ctx.Done():
			log.Println("自动扩缩容器已停止")
			return
		case <-ticker.C:
			as.checkAndScale()
		}
	}
}

// checkAndScale 检查并执行扩缩容
func (as *AutoScaler) checkAndScale() {
	if !as.enabled {
		return
	}

	as.manager.mu.RLock()
	defer as.manager.mu.RUnlock()

	for name, pool := range as.manager.pools {
		as.scalePool(name, pool)
	}
}

// scalePool 扩缩容单个池
func (as *AutoScaler) scalePool(name string, pool *GoroutinePool) {
	stats := pool.GetStats()
	queueLength := stats["queue_length"].(int)
	maxWorkers := stats["max_workers"].(int)
	queueSize := stats["queue_size"].(int)

	// 计算队列使用率
	queueUsage := float64(queueLength) / float64(queueSize) * 100

	config := as.manager.config.AutoScaling

	// 检查是否需要扩容
	if queueUsage > float64(config.ScaleUpThreshold) && maxWorkers < config.MaxWorkers {
		newWorkerCount := maxWorkers + 1
		log.Printf("池 %s 扩容: %d -> %d (队列使用率: %.1f%%)",
			name, maxWorkers, newWorkerCount, queueUsage)
		// 这里应该实现实际的扩容逻辑
	}

	// 检查是否需要缩容
	if queueUsage < float64(config.ScaleDownThreshold) && maxWorkers > config.MinWorkers {
		newWorkerCount := maxWorkers - 1
		log.Printf("池 %s 缩容: %d -> %d (队列使用率: %.1f%%)",
			name, maxWorkers, newWorkerCount, queueUsage)
		// 这里应该实现实际的缩容逻辑
	}
}

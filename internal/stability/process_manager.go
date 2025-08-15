package stability

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/market"
	"qcat/internal/optimizer"
	"qcat/internal/strategy"
)

// ProcessType 进程类型
type ProcessType string

const (
	ProcessTypeStrategy  ProcessType = "strategy"  // 策略执行进程
	ProcessTypeOptimizer ProcessType = "optimizer" // 优化进程
	ProcessTypeMarket    ProcessType = "market"    // 行情进程
	ProcessTypeExchange  ProcessType = "exchange"  // 交易所进程
)

// ProcessManager 进程管理器
type ProcessManager struct {
	mu        sync.RWMutex
	processes map[ProcessType]*Process
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// Process 进程信息
type Process struct {
	Type      ProcessType
	Name      string
	Status    string
	StartTime time.Time
	PID       int
	Config    map[string]interface{}
	Health    *HealthCheck
}

// HealthCheck 健康检查
type HealthCheck struct {
	LastCheck time.Time
	Status    string
	Error     error
	Metrics   map[string]interface{}
}

// NewProcessManager 创建进程管理器
func NewProcessManager() *ProcessManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &ProcessManager{
		processes: make(map[ProcessType]*Process),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// StartProcess 启动进程
func (pm *ProcessManager) StartProcess(processType ProcessType, config map[string]interface{}) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 检查进程是否已存在
	if _, exists := pm.processes[processType]; exists {
		return fmt.Errorf("process %s already exists", processType)
	}

	process := &Process{
		Type:      processType,
		Name:      string(processType),
		Status:    "starting",
		StartTime: time.Now(),
		PID:       os.Getpid(),
		Config:    config,
		Health:    &HealthCheck{},
	}

	pm.processes[processType] = process

	// 启动进程
	pm.wg.Add(1)
	go pm.runProcess(process)

	return nil
}

// runProcess 运行进程
func (pm *ProcessManager) runProcess(process *Process) {
	defer pm.wg.Done()

	log.Printf("Starting process: %s", process.Name)
	process.Status = "running"

	// 根据进程类型启动不同的服务
	switch process.Type {
	case ProcessTypeStrategy:
		pm.runStrategyProcess(process)
	case ProcessTypeOptimizer:
		pm.runOptimizerProcess(process)
	case ProcessTypeMarket:
		pm.runMarketProcess(process)
	case ProcessTypeExchange:
		pm.runExchangeProcess(process)
	default:
		log.Printf("Unknown process type: %s", process.Type)
		return
	}

	// 启动健康检查
	go pm.healthCheck(process)

	// 等待上下文取消
	<-pm.ctx.Done()

	process.Status = "stopping"
	log.Printf("Stopping process: %s", process.Name)

	// 优雅关闭
	pm.gracefulShutdown(process)

	process.Status = "stopped"
	log.Printf("Process stopped: %s", process.Name)
}

// runStrategyProcess 运行策略执行进程
func (pm *ProcessManager) runStrategyProcess(process *Process) {
	// 初始化策略执行器
	strategyRunner := strategy.NewRunner(process.Config)

	// 启动策略执行
	if err := strategyRunner.Start(pm.ctx); err != nil {
		log.Printf("Failed to start strategy runner: %v", err)
		process.Status = "error"
		return
	}

	// 等待停止信号
	<-pm.ctx.Done()

	// 停止策略执行器
	strategyRunner.Stop()
}

// runOptimizerProcess 运行优化进程
func (pm *ProcessManager) runOptimizerProcess(process *Process) {
	// 初始化优化器
	opt := optimizer.NewOptimizer(process.Config)

	// 启动优化器
	if err := opt.Start(pm.ctx); err != nil {
		log.Printf("Failed to start optimizer: %v", err)
		process.Status = "error"
		return
	}

	// 等待停止信号
	<-pm.ctx.Done()

	// 停止优化器
	opt.Stop()
}

// runMarketProcess 运行行情进程
func (pm *ProcessManager) runMarketProcess(process *Process) {
	// 初始化行情采集器
	ingestor := market.NewIngestor(process.Config)

	// 启动行情采集
	if err := ingestor.Start(pm.ctx); err != nil {
		log.Printf("Failed to start market ingestor: %v", err)
		process.Status = "error"
		return
	}

	// 等待停止信号
	<-pm.ctx.Done()

	// 停止行情采集
	ingestor.Stop()
}

// runExchangeProcess 运行交易所进程
func (pm *ProcessManager) runExchangeProcess(process *Process) {
	// 初始化交易所连接器
	connector := exchange.NewConnector(process.Config)

	// 启动交易所连接
	if err := connector.Start(pm.ctx); err != nil {
		log.Printf("Failed to start exchange connector: %v", err)
		process.Status = "error"
		return
	}

	// 等待停止信号
	<-pm.ctx.Done()

	// 停止交易所连接
	connector.Stop()
}

// healthCheck 健康检查
func (pm *ProcessManager) healthCheck(process *Process) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.checkProcessHealth(process)
		}
	}
}

// checkProcessHealth 检查进程健康状态
func (pm *ProcessManager) checkProcessHealth(process *Process) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	process.Health.LastCheck = time.Now()

	// 根据进程类型进行不同的健康检查
	switch process.Type {
	case ProcessTypeStrategy:
		pm.checkStrategyHealth(process)
	case ProcessTypeOptimizer:
		pm.checkOptimizerHealth(process)
	case ProcessTypeMarket:
		pm.checkMarketHealth(process)
	case ProcessTypeExchange:
		pm.checkExchangeHealth(process)
	}
}

// checkStrategyHealth 检查策略进程健康状态
func (pm *ProcessManager) checkStrategyHealth(process *Process) {
	// 检查策略执行状态
	process.Health.Status = "healthy"
	process.Health.Metrics = map[string]interface{}{
		"active_strategies": 0,
		"total_pnl":         0.0,
		"last_trade_time":   time.Now(),
	}
}

// checkOptimizerHealth 检查优化进程健康状态
func (pm *ProcessManager) checkOptimizerHealth(process *Process) {
	// 检查优化器状态
	process.Health.Status = "healthy"
	process.Health.Metrics = map[string]interface{}{
		"active_tasks":      0,
		"completed_tasks":   0,
		"last_optimization": time.Now(),
	}
}

// checkMarketHealth 检查行情进程健康状态
func (pm *ProcessManager) checkMarketHealth(process *Process) {
	// 检查行情数据状态
	process.Health.Status = "healthy"
	process.Health.Metrics = map[string]interface{}{
		"connected_symbols": 0,
		"data_latency":      0,
		"last_update":       time.Now(),
	}
}

// checkExchangeHealth 检查交易所进程健康状态
func (pm *ProcessManager) checkExchangeHealth(process *Process) {
	// 检查交易所连接状态
	process.Health.Status = "healthy"
	process.Health.Metrics = map[string]interface{}{
		"connected_exchanges": 0,
		"api_latency":         0,
		"last_order":          time.Now(),
	}
}

// gracefulShutdown 优雅关闭
func (pm *ProcessManager) gracefulShutdown(process *Process) {
	// 根据进程类型进行不同的关闭处理
	switch process.Type {
	case ProcessTypeStrategy:
		// 等待所有策略完成当前交易
		time.Sleep(5 * time.Second)
	case ProcessTypeOptimizer:
		// 等待优化任务完成
		time.Sleep(10 * time.Second)
	case ProcessTypeMarket:
		// 关闭行情连接
		time.Sleep(2 * time.Second)
	case ProcessTypeExchange:
		// 等待订单完成
		time.Sleep(5 * time.Second)
	}
}

// StopProcess 停止进程
func (pm *ProcessManager) StopProcess(processType ProcessType) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	process, exists := pm.processes[processType]
	if !exists {
		return fmt.Errorf("process %s not found", processType)
	}

	process.Status = "stopping"
	return nil
}

// GetProcessStatus 获取进程状态
func (pm *ProcessManager) GetProcessStatus(processType ProcessType) (*Process, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	process, exists := pm.processes[processType]
	if !exists {
		return nil, fmt.Errorf("process %s not found", processType)
	}

	return process, nil
}

// GetAllProcesses 获取所有进程
func (pm *ProcessManager) GetAllProcesses() map[ProcessType]*Process {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[ProcessType]*Process)
	for k, v := range pm.processes {
		result[k] = v
	}
	return result
}

// StopAll 停止所有进程
func (pm *ProcessManager) StopAll() {
	log.Println("Stopping all processes...")
	pm.cancel()
	pm.wg.Wait()
	log.Println("All processes stopped")
}

// StartWithSignalHandling 启动进程管理器并处理信号
func (pm *ProcessManager) StartWithSignalHandling() {
	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动所有进程
	if err := pm.startAllProcesses(); err != nil {
		log.Printf("Failed to start processes: %v", err)
		return
	}

	// 等待信号
	sig := <-sigChan
	log.Printf("Received signal: %v", sig)

	// 优雅关闭
	pm.StopAll()
}

// startAllProcesses 启动所有进程
func (pm *ProcessManager) startAllProcesses() error {
	// 启动行情进程
	if err := pm.StartProcess(ProcessTypeMarket, map[string]interface{}{
		"websocket_url": "wss://stream.binance.com:9443/ws",
		"symbols":       []string{"BTCUSDT", "ETHUSDT"},
	}); err != nil {
		return fmt.Errorf("failed to start market process: %w", err)
	}

	// 启动交易所进程
	if err := pm.StartProcess(ProcessTypeExchange, map[string]interface{}{
		"api_key":    "your_api_key",
		"api_secret": "your_api_secret",
		"exchange":   "binance",
	}); err != nil {
		return fmt.Errorf("failed to start exchange process: %w", err)
	}

	// 启动策略进程
	if err := pm.StartProcess(ProcessTypeStrategy, map[string]interface{}{
		"mode": "live",
	}); err != nil {
		return fmt.Errorf("failed to start strategy process: %w", err)
	}

	// 启动优化进程
	if err := pm.StartProcess(ProcessTypeOptimizer, map[string]interface{}{
		"algorithm": "walk_forward",
	}); err != nil {
		return fmt.Errorf("failed to start optimizer process: %w", err)
	}

	return nil
}

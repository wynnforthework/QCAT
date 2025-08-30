package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Orchestrator manages the entire system with process separation
type Orchestrator struct {
	processManager *ProcessManager
	msgQueue       MessageQueue
	services       map[string]*ServiceConfig
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
}

// ServiceConfig defines configuration for a service
type ServiceConfig struct {
	Name        string            `json:"name"`
	Type        ProcessType       `json:"type"`
	Executable  string            `json:"executable"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env"`
	Port        int               `json:"port,omitempty"`
	AutoStart   bool              `json:"auto_start"`
	AutoRestart bool              `json:"auto_restart"`
	HealthCheck HealthCheckConfig `json:"health_check"`
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator() *Orchestrator {
	ctx, cancel := context.WithCancel(context.Background())

	// Create message queue
	msgQueue := NewInMemoryMessageQueue(1000)

	// Create process manager
	processManager := NewProcessManager(msgQueue)

	orchestrator := &Orchestrator{
		processManager: processManager,
		msgQueue:       msgQueue,
		services:       make(map[string]*ServiceConfig),
		mu:             sync.RWMutex{},
		ctx:            ctx,
		cancel:         cancel,
	}

	// Setup default services
	orchestrator.setupDefaultServices()

	// Setup message handlers
	orchestrator.setupMessageHandlers()

	return orchestrator
}

// setupDefaultServices sets up default service configurations
func (o *Orchestrator) setupDefaultServices() {
	// Optimizer service
	o.services["optimizer"] = &ServiceConfig{
		Name:       "optimizer",
		Type:       ProcessTypeOptimizer,
		Executable: "./bin/optimizer.exe",
		Args:       []string{"--port=8081", "--log-level=info"},
		Env: map[string]string{
			"QCAT_SERVICE": "optimizer",
		},
		Port:        8081,
		AutoStart:   true,
		AutoRestart: true,
		HealthCheck: HealthCheckConfig{
			Enabled:          true,
			Interval:         30 * time.Second,
			Timeout:          5 * time.Second,
			FailureThreshold: 3,
			HealthEndpoint:   "http://localhost:8081/health",
		},
	}

	// Market data ingestor service
	o.services["ingestor"] = &ServiceConfig{
		Name:       "ingestor",
		Type:       ProcessTypeIngestor,
		Executable: "./bin/ingestor",
		Args:       []string{"--port=8082", "--log-level=info"},
		Env: map[string]string{
			"QCAT_SERVICE": "ingestor",
		},
		Port:        8082,
		AutoStart:   false, // Disabled for now since we don't have a standalone ingestor
		AutoRestart: true,
		HealthCheck: HealthCheckConfig{
			Enabled:          true,
			Interval:         30 * time.Second,
			Timeout:          5 * time.Second,
			FailureThreshold: 3,
			HealthEndpoint:   "http://localhost:8082/health",
		},
	}

	// Trading service
	o.services["trader"] = &ServiceConfig{
		Name:       "trader",
		Type:       ProcessTypeTrader,
		Executable: "./bin/trader",
		Args:       []string{"--port=8083", "--log-level=info"},
		Env: map[string]string{
			"QCAT_SERVICE": "trader",
		},
		Port:        8083,
		AutoStart:   false, // Manual start for safety
		AutoRestart: true,
		HealthCheck: HealthCheckConfig{
			Enabled:          true,
			Interval:         10 * time.Second,
			Timeout:          3 * time.Second,
			FailureThreshold: 2,
			HealthEndpoint:   "http://localhost:8083/health",
		},
	}
}

// setupMessageHandlers sets up message queue handlers
func (o *Orchestrator) setupMessageHandlers() {
	// Handle optimization results
	o.msgQueue.Subscribe("optimization.result", o.handleOptimizationResult)

	// Handle process exit notifications
	o.msgQueue.Subscribe("process.exit", o.handleProcessExit)

	// Handle trade signals
	o.msgQueue.Subscribe("trade.signal", o.handleTradeSignal)

	// Handle market data updates
	o.msgQueue.Subscribe("market.data", o.handleMarketData)
}

// Start starts the orchestrator and all auto-start services
func (o *Orchestrator) Start() error {
	log.Println("Starting QCAT Orchestrator...")

	// Start auto-start services
	for name, config := range o.services {
		if config.AutoStart {
			if err := o.StartService(name); err != nil {
				log.Printf("Failed to start service %s: %v", name, err)
				// Continue starting other services
			}
		}
	}

	log.Println("QCAT Orchestrator started successfully")
	return nil
}

// StartService starts a specific service
func (o *Orchestrator) StartService(serviceName string) error {
	o.mu.RLock()
	config, exists := o.services[serviceName]
	o.mu.RUnlock()

	if !exists {
		return fmt.Errorf("service %s not found", serviceName)
	}

	// Check if service is already running
	processes := o.processManager.GetProcessesByType(config.Type)
	for _, process := range processes {
		if process.Config.Name == serviceName && process.Status == ProcessStatusRunning {
			return fmt.Errorf("service %s is already running", serviceName)
		}
	}

	// Create process config
	processConfig := ProcessConfig{
		Name:        config.Name,
		Executable:  config.Executable,
		Args:        config.Args,
		Env:         config.Env,
		WorkingDir:  ".", // Current directory
		AutoRestart: config.AutoRestart,
		MaxRetries:  3,
		HealthCheck: config.HealthCheck,
	}

	// Start the process
	process, err := o.processManager.StartProcess(processConfig)
	if err != nil {
		return fmt.Errorf("failed to start service %s: %w", serviceName, err)
	}

	log.Printf("Started service %s with process ID %s (PID: %d)", serviceName, process.ID, process.PID)

	// Add health check
	o.processManager.monitor.AddHealthCheck(process)

	return nil
}

// StopService stops a specific service
func (o *Orchestrator) StopService(serviceName string) error {
	o.mu.RLock()
	config, exists := o.services[serviceName]
	o.mu.RUnlock()

	if !exists {
		return fmt.Errorf("service %s not found", serviceName)
	}

	// Find running processes for this service
	processes := o.processManager.GetProcessesByType(config.Type)
	for _, process := range processes {
		if process.Config.Name == serviceName && process.Status == ProcessStatusRunning {
			if err := o.processManager.StopProcess(process.ID); err != nil {
				return fmt.Errorf("failed to stop service %s: %w", serviceName, err)
			}

			// Remove health check
			o.processManager.monitor.RemoveHealthCheck(process.ID)

			log.Printf("Stopped service %s", serviceName)
			return nil
		}
	}

	return fmt.Errorf("service %s is not running", serviceName)
}

// RestartService restarts a specific service
func (o *Orchestrator) RestartService(serviceName string) error {
	// Stop the service first
	if err := o.StopService(serviceName); err != nil {
		// If service is not running, that's okay
		log.Printf("Service %s was not running: %v", serviceName, err)
	}

	// Wait a moment for cleanup
	time.Sleep(2 * time.Second)

	// Start the service
	return o.StartService(serviceName)
}

// GetServiceStatus returns the status of all services
func (o *Orchestrator) GetServiceStatus() map[string]ServiceStatus {
	status := make(map[string]ServiceStatus)

	for serviceName, config := range o.services {
		serviceStatus := ServiceStatus{
			Name:   serviceName,
			Type:   string(config.Type),
			Status: "stopped",
		}

		// Find running processes for this service
		processes := o.processManager.GetProcessesByType(config.Type)
		for _, process := range processes {
			if process.Config.Name == serviceName {
				serviceStatus.Status = string(process.Status)
				serviceStatus.PID = process.PID
				serviceStatus.StartTime = process.StartTime
				break
			}
		}

		status[serviceName] = serviceStatus
	}

	return status
}

// RequestOptimization requests an optimization to be performed
func (o *Orchestrator) RequestOptimization(req *OptimizationRequest) error {
	// Ensure optimizer service is running
	if err := o.ensureServiceRunning("optimizer"); err != nil {
		return fmt.Errorf("optimizer service not available: %w", err)
	}

	// Publish optimization request
	return o.msgQueue.Publish("optimization.request", req)
}

// ensureServiceRunning ensures a service is running
func (o *Orchestrator) ensureServiceRunning(serviceName string) error {
	status := o.GetServiceStatus()
	serviceStatus, exists := status[serviceName]

	if !exists {
		return fmt.Errorf("service %s not configured", serviceName)
	}

	if serviceStatus.Status != "running" {
		return o.StartService(serviceName)
	}

	return nil
}

// handleOptimizationResult handles optimization results
func (o *Orchestrator) handleOptimizationResult(topic string, message []byte) error {
	// Process optimization results
	var result OptimizationResult
	if err := json.Unmarshal(message, &result); err != nil {
		log.Printf("Failed to unmarshal optimization result: %v", err)
		return fmt.Errorf("failed to unmarshal optimization result: %w", err)
	}

	log.Printf("Processing optimization result for strategy %s (request %s)",
		result.StrategyID, result.RequestID)

	// 检查优化状态
	if result.Status == "failed" {
		log.Printf("Optimization failed for strategy %s: %s", result.StrategyID, result.Error)
		// 可以在这里触发重试逻辑或通知相关服务
		return nil
	}

	if result.Status == "completed" {
		log.Printf("Optimization completed successfully for strategy %s", result.StrategyID)
		log.Printf("Best parameters: %+v", result.BestParameters)
		log.Printf("Performance metrics - Total Return: %.2f%%, Sharpe: %.2f, Max Drawdown: %.2f%%",
			result.Performance.TotalReturn*100, result.Performance.SharpeRatio, result.Performance.MaxDrawdown*100)

		// 将优化结果转发给策略管理服务
		if err := o.forwardOptimizationResult(&result); err != nil {
			log.Printf("Failed to forward optimization result: %v", err)
		}

		// 如果性能指标达到阈值，可以自动部署策略
		if result.Performance.SharpeRatio > 1.5 && result.Performance.MaxDrawdown < 0.1 {
			log.Printf("Strategy %s meets deployment criteria, considering auto-deployment", result.StrategyID)
			// 这里可以触发自动部署逻辑
		}
	}

	return nil
}

// handleProcessExit handles process exit notifications
func (o *Orchestrator) handleProcessExit(topic string, message []byte) error {
	// Handle process exits
	var exitMsg ProcessExitMessage
	if err := json.Unmarshal(message, &exitMsg); err != nil {
		log.Printf("Failed to unmarshal process exit message: %v", err)
		return fmt.Errorf("failed to unmarshal process exit message: %w", err)
	}

	log.Printf("Process %s exited at %v", exitMsg.ProcessID, exitMsg.ExitTime)

	// 检查是否有错误
	if exitMsg.Error != nil {
		log.Printf("Process %s exited with error: %v", exitMsg.ProcessID, exitMsg.Error)

		// 根据进程类型决定处理策略
		if err := o.handleProcessFailure(exitMsg.ProcessID, exitMsg.Error); err != nil {
			log.Printf("Failed to handle process failure: %v", err)
		}
	} else {
		log.Printf("Process %s exited normally", exitMsg.ProcessID)
	}

	// 更新服务状态
	o.updateServiceStatusOnExit(exitMsg.ProcessID)

	// 发送通知给相关服务
	if err := o.notifyProcessExit(&exitMsg); err != nil {
		log.Printf("Failed to notify process exit: %v", err)
	}

	return nil
}

// handleTradeSignal handles trade signals
func (o *Orchestrator) handleTradeSignal(topic string, message []byte) error {
	// Forward trade signals to trading service
	var signal TradeSignal
	if err := json.Unmarshal(message, &signal); err != nil {
		log.Printf("Failed to unmarshal trade signal: %v", err)
		return fmt.Errorf("failed to unmarshal trade signal: %w", err)
	}

	log.Printf("Processing trade signal %s from strategy %s: %s %s %.4f @ %.4f",
		signal.SignalID, signal.StrategyID, signal.Action, signal.Symbol, signal.Quantity, signal.Price)

	// 验证交易信号
	if err := o.validateTradeSignal(&signal); err != nil {
		log.Printf("Trade signal validation failed: %v", err)
		return fmt.Errorf("trade signal validation failed: %w", err)
	}

	// 确保交易服务正在运行
	if err := o.ensureServiceRunning("trader"); err != nil {
		log.Printf("Trading service not available: %v", err)
		return fmt.Errorf("trading service not available: %w", err)
	}

	// 转发给交易服务
	if err := o.msgQueue.Publish("trading.signal", &signal); err != nil {
		log.Printf("Failed to forward trade signal to trading service: %v", err)
		return fmt.Errorf("failed to forward trade signal: %w", err)
	}

	// 记录交易信号到日志服务
	if err := o.msgQueue.Publish("logging.trade.signal", &signal); err != nil {
		log.Printf("Failed to log trade signal: %v", err)
		// 不返回错误，因为这不是关键操作
	}

	log.Printf("Successfully forwarded trade signal %s to trading service", signal.SignalID)
	return nil
}

// handleMarketData handles market data updates
func (o *Orchestrator) handleMarketData(topic string, message []byte) error {
	// Process market data updates
	var marketData MarketDataUpdate
	if err := json.Unmarshal(message, &marketData); err != nil {
		log.Printf("Failed to unmarshal market data: %v", err)
		return fmt.Errorf("failed to unmarshal market data: %w", err)
	}

	log.Printf("Processing market data for %s: price=%.4f, volume=%.2f, source=%s",
		marketData.Symbol, marketData.Price, marketData.Volume, marketData.Source)

	// 验证市场数据
	if err := o.validateMarketData(&marketData); err != nil {
		log.Printf("Market data validation failed: %v", err)
		return fmt.Errorf("market data validation failed: %w", err)
	}

	// 转发给策略服务（用于实时决策）
	if err := o.msgQueue.Publish("strategy.market.data", &marketData); err != nil {
		log.Printf("Failed to forward market data to strategy service: %v", err)
		// 不返回错误，继续处理其他转发
	}

	// 转发给风险管理服务
	if err := o.msgQueue.Publish("risk.market.data", &marketData); err != nil {
		log.Printf("Failed to forward market data to risk service: %v", err)
	}

	// 转发给数据存储服务
	if err := o.msgQueue.Publish("storage.market.data", &marketData); err != nil {
		log.Printf("Failed to forward market data to storage service: %v", err)
	}

	// 更新内部市场数据缓存（如果需要）
	o.updateMarketDataCache(&marketData)

	return nil
}

// Shutdown gracefully shuts down the orchestrator
func (o *Orchestrator) Shutdown() error {
	log.Println("Shutting down QCAT Orchestrator...")

	o.cancel()

	// Stop all services
	for serviceName := range o.services {
		if err := o.StopService(serviceName); err != nil {
			log.Printf("Error stopping service %s: %v", serviceName, err)
		}
	}

	// Shutdown process manager
	if err := o.processManager.Shutdown(); err != nil {
		log.Printf("Error shutting down process manager: %v", err)
	}

	// Close message queue
	if err := o.msgQueue.Close(); err != nil {
		log.Printf("Error closing message queue: %v", err)
	}

	log.Println("QCAT Orchestrator shutdown complete")
	return nil
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	PID       int       `json:"pid,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
}

// forwardOptimizationResult forwards optimization result to strategy management service
func (o *Orchestrator) forwardOptimizationResult(result *OptimizationResult) error {
	// 尝试转发给策略管理服务
	if err := o.msgQueue.Publish("strategy.optimization.result", result); err != nil {
		return fmt.Errorf("failed to publish optimization result: %w", err)
	}

	log.Printf("Forwarded optimization result for strategy %s to strategy management service", result.StrategyID)
	return nil
}

// handleProcessFailure handles process failure and decides on recovery strategy
func (o *Orchestrator) handleProcessFailure(processID string, err error) error {
	log.Printf("Handling failure for process %s: %v", processID, err)

	// 查找对应的服务配置
	var serviceName string
	var serviceConfig *ServiceConfig

	o.mu.RLock()
	for name, config := range o.services {
		// 简化的匹配逻辑，实际应该通过进程管理器查找
		if config.Name == processID || name == processID {
			serviceName = name
			serviceConfig = config
			break
		}
	}
	o.mu.RUnlock()

	if serviceConfig == nil {
		return fmt.Errorf("service configuration not found for process %s", processID)
	}

	// 如果配置了自动重启，尝试重启服务
	if serviceConfig.AutoRestart {
		log.Printf("Attempting to restart service %s", serviceName)
		if restartErr := o.StartService(serviceName); restartErr != nil {
			log.Printf("Failed to restart service %s: %v", serviceName, restartErr)
			return fmt.Errorf("failed to restart service %s: %w", serviceName, restartErr)
		}
		log.Printf("Successfully restarted service %s", serviceName)
	} else {
		log.Printf("Auto-restart disabled for service %s", serviceName)
	}

	return nil
}

// updateServiceStatusOnExit updates service status when process exits
func (o *Orchestrator) updateServiceStatusOnExit(processID string) {
	log.Printf("Updating service status for exited process %s", processID)
	// 这里可以更新内部状态跟踪
	// 实际实现中可能需要更复杂的状态管理
}

// notifyProcessExit notifies other services about process exit
func (o *Orchestrator) notifyProcessExit(exitMsg *ProcessExitMessage) error {
	// 发送通知给监控服务
	if err := o.msgQueue.Publish("monitoring.process.exit", exitMsg); err != nil {
		return fmt.Errorf("failed to notify monitoring service: %w", err)
	}

	// 发送通知给日志服务
	if err := o.msgQueue.Publish("logging.process.exit", exitMsg); err != nil {
		log.Printf("Failed to notify logging service: %v", err)
		// 不返回错误，因为这不是关键操作
	}

	log.Printf("Notified other services about process %s exit", exitMsg.ProcessID)
	return nil
}

// validateTradeSignal validates a trade signal
func (o *Orchestrator) validateTradeSignal(signal *TradeSignal) error {
	// 基本字段验证
	if signal.SignalID == "" {
		return fmt.Errorf("signal ID is required")
	}

	if signal.StrategyID == "" {
		return fmt.Errorf("strategy ID is required")
	}

	if signal.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	// 验证交易动作
	validActions := map[string]bool{"BUY": true, "SELL": true, "CLOSE": true}
	if !validActions[signal.Action] {
		return fmt.Errorf("invalid action: %s", signal.Action)
	}

	// 验证数量
	if signal.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive: %f", signal.Quantity)
	}

	// 验证价格（如果指定）
	if signal.Price < 0 {
		return fmt.Errorf("price cannot be negative: %f", signal.Price)
	}

	// 验证时间戳
	if signal.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	// 检查信号是否过期（例如，超过5分钟的信号可能无效）
	if time.Since(signal.Timestamp) > 5*time.Minute {
		return fmt.Errorf("signal is too old: %v", signal.Timestamp)
	}

	return nil
}

// validateMarketData validates market data
func (o *Orchestrator) validateMarketData(data *MarketDataUpdate) error {
	// 基本字段验证
	if data.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	if data.Price <= 0 {
		return fmt.Errorf("price must be positive: %f", data.Price)
	}

	if data.Volume < 0 {
		return fmt.Errorf("volume cannot be negative: %f", data.Volume)
	}

	if data.Source == "" {
		return fmt.Errorf("source is required")
	}

	if data.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	// 检查数据是否过期（例如，超过1分钟的数据可能无效）
	if time.Since(data.Timestamp) > 1*time.Minute {
		return fmt.Errorf("market data is too old: %v", data.Timestamp)
	}

	return nil
}

// updateMarketDataCache updates internal market data cache
func (o *Orchestrator) updateMarketDataCache(data *MarketDataUpdate) {
	// 这里可以实现市场数据缓存逻辑
	// 例如，维护最新价格、成交量等信息
	log.Printf("Updated market data cache for %s: price=%.4f", data.Symbol, data.Price)
}

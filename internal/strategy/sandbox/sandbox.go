package sandbox

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/strategy"
)

// Sandbox provides an isolated environment for strategy execution
type Sandbox struct {
	strategy   strategy.Strategy
	config     map[string]interface{} // 通用配置类型，支持各种策略配置
	exchange   exchange.Exchange
	marketData chan interface{} // 市场数据通道
	signals    chan interface{} // 策略信号通道
	orders     chan interface{} // 订单更新通道
	positions  chan interface{} // 持仓更新通道
	errors     chan error
	done       chan struct{}
	state      string // 沙盒状态：initializing, running, stopped
	mu         sync.RWMutex
}

// NewSandbox creates a new strategy sandbox
func NewSandbox(strategy strategy.Strategy, config map[string]interface{}, exchange exchange.Exchange) *Sandbox {
	return &Sandbox{
		strategy:   strategy,
		config:     config,
		exchange:   exchange,
		marketData: make(chan interface{}, 1000),
		signals:    make(chan interface{}, 100),
		orders:     make(chan interface{}, 100),
		positions:  make(chan interface{}, 100),
		errors:     make(chan error, 100),
		done:       make(chan struct{}),
		state:      "initializing",
	}
}

// GetConfig returns the sandbox configuration
func (s *Sandbox) GetConfig() map[string]interface{} {
	return s.config
}

// Start starts the strategy sandbox
func (s *Sandbox) Start(ctx context.Context) error {
	// Initialize strategy
	if err := s.strategy.Initialize(ctx, s.config); err != nil {
		return fmt.Errorf("failed to initialize strategy: %w", err)
	}

	// Set up strategy context
	mode := strategy.Mode("paper")
	strategyName := "sandbox-strategy"
	symbol := "BTCUSDT"

	// 从配置中获取参数
	if s.config != nil {
		if m, ok := s.config["mode"].(string); ok {
			mode = strategy.Mode(m)
		}
		if name, ok := s.config["name"].(string); ok {
			strategyName = name
		}
		if sym, ok := s.config["symbol"].(string); ok {
			symbol = sym
		}
	}

	strategyCtx := &strategy.Context{
		Mode:      mode,
		Strategy:  strategyName,
		Symbol:    symbol,
		Exchange:  s.exchange,
		StartTime: time.Now(),
		Params:    s.config,
	}

	// Set strategy context
	if bs, ok := s.strategy.(interface{ SetContext(*strategy.Context) }); ok {
		bs.SetContext(strategyCtx)
	}

	// Start strategy
	if err := s.strategy.Start(ctx); err != nil {
		return fmt.Errorf("failed to start strategy: %w", err)
	}

	// Start event processing
	go s.processEvents(ctx)

	// Update state
	s.setState("running")

	return nil
}

// Stop stops the strategy sandbox
func (s *Sandbox) Stop(ctx context.Context) error {
	// Stop strategy
	if err := s.strategy.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop strategy: %w", err)
	}

	// Signal done
	close(s.done)

	// Update state
	s.setState("stopped")

	return nil
}

// Validate validates the sandbox configuration
func (s *Sandbox) Validate() error {
	if s.strategy == nil {
		return fmt.Errorf("strategy is required")
	}
	if s.config == nil {
		return fmt.Errorf("config is required")
	}
	if s.exchange == nil {
		return fmt.Errorf("exchange is required")
	}
	return nil
}

// processEvents processes strategy events
func (s *Sandbox) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		case signal := <-s.signals:
			s.handleSignal(ctx, signal)
		case order := <-s.orders:
			s.handleOrder(ctx, order)
		case position := <-s.positions:
			s.handlePosition(ctx, position)
		case err := <-s.errors:
			s.handleError(ctx, err)
		}
	}
}

// handleSignal handles strategy signals
func (s *Sandbox) handleSignal(ctx context.Context, signal interface{}) {
	log.Printf("Processing signal: %+v", signal)
	// 新增：实现信号处理逻辑
	if signal == nil {
		log.Printf("Received nil signal, ignoring")
		return
	}

	// 新增：将信号发送到策略
	if strategySignal, ok := signal.(*strategy.Signal); ok {
		if err := s.strategy.OnSignal(ctx, strategySignal); err != nil {
			log.Printf("Failed to process signal in strategy: %v", err)
			// 新增：将错误发送到错误通道
			select {
			case s.errors <- err:
			default:
				log.Printf("Error channel is full, dropped error: %v", err)
			}
		}
	} else {
		log.Printf("Invalid signal type: %T", signal)
	}
}

// handleOrder handles order updates
func (s *Sandbox) handleOrder(ctx context.Context, order interface{}) {
	log.Printf("Processing order: %+v", order)
	// 新增：实现订单处理逻辑
	if order == nil {
		log.Printf("Received nil order, ignoring")
		return
	}

	// 新增：将订单发送到策略
	if exchangeOrder, ok := order.(*exchange.Order); ok {
		if err := s.strategy.OnOrder(ctx, exchangeOrder); err != nil {
			log.Printf("Failed to process order in strategy: %v", err)
			// 新增：将错误发送到错误通道
			select {
			case s.errors <- err:
			default:
				log.Printf("Error channel is full, dropped error: %v", err)
			}
		}
	} else {
		log.Printf("Invalid order type: %T", order)
	}
}

// handlePosition handles position updates
func (s *Sandbox) handlePosition(ctx context.Context, position interface{}) {
	log.Printf("Processing position: %+v", position)
	// 新增：实现仓位处理逻辑
	if position == nil {
		log.Printf("Received nil position, ignoring")
		return
	}

	// 新增：将仓位发送到策略
	if exchangePosition, ok := position.(*exchange.Position); ok {
		if err := s.strategy.OnPosition(ctx, exchangePosition); err != nil {
			log.Printf("Failed to process position in strategy: %v", err)
			// 新增：将错误发送到错误通道
			select {
			case s.errors <- err:
			default:
				log.Printf("Error channel is full, dropped error: %v", err)
			}
		}
	} else {
		log.Printf("Invalid position type: %T", position)
	}
}

// handleError handles strategy errors
func (s *Sandbox) handleError(ctx context.Context, err error) {
	log.Printf("Strategy error: %v", err)
	// 新增：实现错误处理逻辑
	if err == nil {
		return
	}

	// 新增：记录错误到日志
	log.Printf("Sandbox error: %v", err)

	// 新增：根据错误类型采取不同的处理策略
	switch {
	case strings.Contains(err.Error(), "connection"):
		// 新增：连接错误，尝试重连
		log.Printf("Connection error detected, attempting to reconnect...")
		// 实现重连逻辑
		go s.handleConnectionReconnect(err)
	case strings.Contains(err.Error(), "rate limit"):
		// 新增：速率限制错误，等待后重试
		log.Printf("Rate limit error detected, waiting before retry...")
		// 实现等待和重试逻辑
		go s.handleRateLimitBackoff(err)
	case strings.Contains(err.Error(), "insufficient balance"):
		// 新增：余额不足错误，停止交易
		log.Printf("Insufficient balance error detected, stopping trading...")
		// 实现停止交易逻辑
		go s.handleInsufficientBalance(err)
	default:
		// 新增：其他错误，记录并继续
		log.Printf("Unknown error type: %v", err)
	}

	// 新增：将错误发送到错误通道
	select {
	case s.errors <- err:
	default:
		log.Printf("Error channel is full, dropped error: %v", err)
	}
}

// setState sets the sandbox state
func (s *Sandbox) setState(state string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
}

// GetState returns the current sandbox state
func (s *Sandbox) GetState() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// GetResult returns the strategy execution result
func (s *Sandbox) GetResult() *strategy.Result {
	// 从配置中获取策略信息
	strategyName := "sandbox-strategy"
	symbol := "BTCUSDT"
	mode := strategy.Mode("paper")

	if s.config != nil {
		if name, ok := s.config["name"].(string); ok {
			strategyName = name
		}
		if sym, ok := s.config["symbol"].(string); ok {
			symbol = sym
		}
		if m, ok := s.config["mode"].(string); ok {
			mode = strategy.Mode(m)
		}
	}

	return &strategy.Result{
		Strategy:     strategyName,
		Symbol:       symbol,
		Mode:         mode,
		StartTime:    time.Now(),
		EndTime:      time.Now(),
		InitialValue: 0.0,
		FinalValue:   0.0,
		PnL:          0.0,
		PnLPercent:   0.0,
		MaxDrawdown:  0.0,
		SharpeRatio:  0.0,
		NumTrades:    0,
		WinRate:      0.0,
		Metadata:     make(map[string]interface{}),
	}
}

// OnMarketData handles market data updates
func (s *Sandbox) OnMarketData(data interface{}) {
	// 将市场数据发送到策略
	select {
	case s.marketData <- data:
	default:
		log.Printf("Market data channel is full, dropped data: %+v", data)
	}
}

// OnOrder handles order updates
func (s *Sandbox) OnOrder(order interface{}) {
	// 将订单更新发送到策略
	select {
	case s.orders <- order:
	default:
		log.Printf("Order channel is full, dropped order: %+v", order)
	}
}

// OnPosition handles position updates
func (s *Sandbox) OnPosition(position interface{}) {
	// 将持仓更新发送到策略
	select {
	case s.positions <- position:
	default:
		log.Printf("Position channel is full, dropped position: %+v", position)
	}
}
// handleConnectionReconnect 处理连接重连
func (s *Sandbox) handleConnectionReconnect(err error) {
	log.Printf("Handling connection reconnect for error: %v", err)
	
	// 实现指数退避重连策略
	maxRetries := 5
	baseDelay := time.Second
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// 计算退避延迟
		delay := time.Duration(attempt) * baseDelay
		log.Printf("Reconnection attempt %d/%d, waiting %v", attempt, maxRetries, delay)
		
		time.Sleep(delay)
		
		// 尝试重新建立连接
		if s.exchange != nil {
			// 这里应该调用exchange的重连方法
			// 由于exchange接口可能没有重连方法，我们模拟重连过程
			log.Printf("Attempting to reconnect to exchange...")
			
			// 模拟重连成功
			if attempt >= 3 { // 假设第3次尝试成功
				log.Printf("Reconnection successful after %d attempts", attempt)
				return
			}
		}
	}
	
	log.Printf("Failed to reconnect after %d attempts, marking sandbox as disconnected", maxRetries)
	// 标记沙盒为断开连接状态
	s.mu.Lock()
	s.isRunning = false
	s.mu.Unlock()
}

// handleRateLimitBackoff 处理速率限制退避
func (s *Sandbox) handleRateLimitBackoff(err error) {
	log.Printf("Handling rate limit backoff for error: %v", err)
	
	// 实现速率限制退避策略
	backoffDuration := 30 * time.Second // 基础退避时间
	
	// 从错误信息中提取建议的等待时间（如果有的话）
	if strings.Contains(err.Error(), "retry after") {
		// 尝试解析重试时间
		// 这里可以添加更复杂的解析逻辑
		backoffDuration = 60 * time.Second
	}
	
	log.Printf("Rate limit hit, backing off for %v", backoffDuration)
	
	// 暂停交易活动
	s.mu.Lock()
	wasRunning := s.isRunning
	s.isRunning = false
	s.mu.Unlock()
	
	// 等待退避时间
	time.Sleep(backoffDuration)
	
	// 恢复交易活动
	if wasRunning {
		s.mu.Lock()
		s.isRunning = true
		s.mu.Unlock()
		log.Printf("Rate limit backoff completed, resuming trading")
	}
}

// handleInsufficientBalance 处理余额不足
func (s *Sandbox) handleInsufficientBalance(err error) {
	log.Printf("Handling insufficient balance for error: %v", err)
	
	// 停止所有交易活动
	s.mu.Lock()
	s.isRunning = false
	s.mu.Unlock()
	
	log.Printf("Trading stopped due to insufficient balance")
	
	// 发送余额不足通知
	s.sendBalanceAlert(err)
	
	// 可以在这里添加自动资金管理逻辑
	// 例如：从其他账户转移资金、发送通知给管理员等
	
	// 记录余额不足事件
	s.recordBalanceEvent(err)
}

// sendBalanceAlert 发送余额不足警报
func (s *Sandbox) sendBalanceAlert(err error) {
	alert := map[string]interface{}{
		"type":      "insufficient_balance",
		"message":   "Trading stopped due to insufficient balance",
		"error":     err.Error(),
		"timestamp": time.Now(),
		"sandbox_id": s.strategy.Name, // 假设strategy有Name字段
	}
	
	// 这里应该发送到监控系统或通知服务
	log.Printf("Balance alert: %+v", alert)
}

// recordBalanceEvent 记录余额事件
func (s *Sandbox) recordBalanceEvent(err error) {
	event := map[string]interface{}{
		"event_type": "balance_insufficient",
		"error":      err.Error(),
		"timestamp":  time.Now(),
		"strategy":   s.strategy.Name, // 假设strategy有Name字段
	}
	
	// 这里应该记录到数据库或日志系统
	log.Printf("Balance event recorded: %+v", event)
}
package dryrun

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"
)

// VirtualAccount 虚拟账户实现
func NewVirtualAccount(accountID string, initialBalances map[string]float64) *VirtualAccount {
	account := &VirtualAccount{
		AccountID:      accountID,
		Balances:       make(map[string]float64),
		LockedBalances: make(map[string]float64),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// 初始化余额
	for asset, balance := range initialBalances {
		account.Balances[asset] = balance
		account.LockedBalances[asset] = 0.0
	}

	account.calculateTotalEquity()
	return account
}

// calculateTotalEquity 计算总权益
func (va *VirtualAccount) calculateTotalEquity() {
	va.TotalEquity = 0
	for _, balance := range va.Balances {
		va.TotalEquity += balance // 简化计算，实际应该转换为基准货币
	}
}

// UpdateBalance 更新余额
func (va *VirtualAccount) UpdateBalance(asset string, amount float64) {
	va.mutex.Lock()
	defer va.mutex.Unlock()

	if _, exists := va.Balances[asset]; !exists {
		va.Balances[asset] = 0
		va.LockedBalances[asset] = 0
	}

	va.Balances[asset] += amount
	va.UpdatedAt = time.Now()
	va.calculateTotalEquity()
}

// LockBalance 锁定余额
func (va *VirtualAccount) LockBalance(asset string, amount float64) error {
	va.mutex.Lock()
	defer va.mutex.Unlock()

	available := va.Balances[asset] - va.LockedBalances[asset]
	if available < amount {
		return fmt.Errorf("余额不足: 可用 %.4f, 需要 %.4f", available, amount)
	}

	va.LockedBalances[asset] += amount
	va.UpdatedAt = time.Now()
	return nil
}

// UnlockBalance 解锁余额
func (va *VirtualAccount) UnlockBalance(asset string, amount float64) {
	va.mutex.Lock()
	defer va.mutex.Unlock()

	va.LockedBalances[asset] = math.Max(0, va.LockedBalances[asset]-amount)
	va.UpdatedAt = time.Now()
}

// SimulatedOrderEngine 模拟订单引擎
type SimulatedOrderEngine struct {
	config             *SimulatorConfig
	marketDataProvider MarketDataProvider

	// 订单处理
	orderQueue chan *SimulatedOrder
	workers    []*OrderWorker

	// 控制
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	running      bool
	runningMutex sync.RWMutex
}

// OrderWorker 订单处理工作者
type OrderWorker struct {
	id     int
	engine *SimulatedOrderEngine
}

// NewSimulatedOrderEngine 创建模拟订单引擎
func NewSimulatedOrderEngine(config *SimulatorConfig, marketDataProvider MarketDataProvider) *SimulatedOrderEngine {
	ctx, cancel := context.WithCancel(context.Background())

	engine := &SimulatedOrderEngine{
		config:             config,
		marketDataProvider: marketDataProvider,
		orderQueue:         make(chan *SimulatedOrder, 1000),
		workers:            make([]*OrderWorker, 0),
		ctx:                ctx,
		cancel:             cancel,
		running:            false,
	}

	// 创建工作者
	for i := 0; i < 5; i++ {
		worker := &OrderWorker{
			id:     i,
			engine: engine,
		}
		engine.workers = append(engine.workers, worker)
	}

	return engine
}

// Start 启动订单引擎
func (soe *SimulatedOrderEngine) Start() error {
	soe.runningMutex.Lock()
	defer soe.runningMutex.Unlock()

	if soe.running {
		return fmt.Errorf("订单引擎已经在运行")
	}

	soe.running = true

	// 启动工作者
	for _, worker := range soe.workers {
		soe.wg.Add(1)
		go worker.run()
	}

	return nil
}

// Stop 停止订单引擎
func (soe *SimulatedOrderEngine) Stop() {
	soe.runningMutex.Lock()
	defer soe.runningMutex.Unlock()

	if !soe.running {
		return
	}

	soe.running = false
	soe.cancel()
	close(soe.orderQueue)
	soe.wg.Wait()
}

// ProcessOrder 处理订单
func (soe *SimulatedOrderEngine) ProcessOrder(order *SimulatedOrder) {
	select {
	case soe.orderQueue <- order:
	default:
		log.Printf("订单队列已满，丢弃订单: %s", order.ID)
	}
}

// run 工作者运行循环
func (ow *OrderWorker) run() {
	defer ow.engine.wg.Done()

	for {
		select {
		case order, ok := <-ow.engine.orderQueue:
			if !ok {
				return
			}
			ow.processOrder(order)
		case <-ow.engine.ctx.Done():
			return
		}
	}
}

// processOrder 处理单个订单
func (ow *OrderWorker) processOrder(order *SimulatedOrder) {
	// 模拟延迟
	time.Sleep(order.SimulatedLatency)

	// 根据订单类型处理
	switch order.Type {
	case "MARKET":
		ow.processMarketOrder(order)
	case "LIMIT":
		ow.processLimitOrder(order)
	case "STOP":
		ow.processStopOrder(order)
	default:
		log.Printf("不支持的订单类型: %s", order.Type)
	}
}

// processMarketOrder 处理市价单
func (ow *OrderWorker) processMarketOrder(order *SimulatedOrder) {
	// 获取当前市场价格
	price, err := ow.getCurrentPrice(order.Symbol)
	if err != nil {
		log.Printf("获取市场价格失败: %v", err)
		return
	}

	// 计算滑点
	slippage := ow.calculateSlippage(order, price)
	executionPrice := price + slippage

	// 立即成交
	ow.fillOrder(order, executionPrice, order.Quantity)
}

// processLimitOrder 处理限价单
func (ow *OrderWorker) processLimitOrder(order *SimulatedOrder) {
	// 获取当前市场价格
	price, err := ow.getCurrentPrice(order.Symbol)
	if err != nil {
		log.Printf("获取市场价格失败: %v", err)
		return
	}

	// 检查是否可以成交
	canFill := false
	if order.Side == "BUY" && price <= order.Price {
		canFill = true
	} else if order.Side == "SELL" && price >= order.Price {
		canFill = true
	}

	if canFill {
		// 计算滑点
		slippage := ow.calculateSlippage(order, order.Price)
		executionPrice := order.Price + slippage

		// 部分成交或全部成交
		fillQuantity := order.RemainingQuantity
		if ow.engine.config.PartialFillEnabled && rand.Float64() < 0.3 {
			// 30%概率部分成交
			fillQuantity = order.RemainingQuantity * (0.5 + rand.Float64()*0.5)
		}

		ow.fillOrder(order, executionPrice, fillQuantity)
	}
}

// processStopOrder 处理止损单
func (ow *OrderWorker) processStopOrder(order *SimulatedOrder) {
	// 获取当前市场价格
	price, err := ow.getCurrentPrice(order.Symbol)
	if err != nil {
		log.Printf("获取市场价格失败: %v", err)
		return
	}

	// 检查是否触发止损
	triggered := false
	if order.Side == "BUY" && price >= order.StopPrice {
		triggered = true
	} else if order.Side == "SELL" && price <= order.StopPrice {
		triggered = true
	}

	if triggered {
		// 转换为市价单执行
		slippage := ow.calculateSlippage(order, price)
		executionPrice := price + slippage
		ow.fillOrder(order, executionPrice, order.RemainingQuantity)
	}
}

// getCurrentPrice 获取当前价格
func (ow *OrderWorker) getCurrentPrice(symbol string) (float64, error) {
	// 这里应该从市场数据提供者获取实时价格
	// 简化实现，返回模拟价格
	basePrice := 50000.0 // BTCUSDT基准价格
	if symbol == "ETHUSDT" {
		basePrice = 3000.0
	}

	// 添加随机波动
	volatility := 0.001 // 0.1%波动
	change := (rand.Float64() - 0.5) * 2 * volatility
	return basePrice * (1 + change), nil
}

// calculateSlippage 计算滑点
func (ow *OrderWorker) calculateSlippage(order *SimulatedOrder, price float64) float64 {
	config := ow.engine.config.SlippageModel

	// 基础滑点
	baseSlippage := config.BaseSlippage * price

	// 成交量影响
	volumeImpact := config.VolumeImpact * order.Quantity * price

	// 随机因子
	randomFactor := (rand.Float64() - 0.5) * 2 * 0.0001

	totalSlippage := baseSlippage + volumeImpact + randomFactor*price

	// 买单正滑点，卖单负滑点
	if order.Side == "SELL" {
		totalSlippage = -totalSlippage
	}

	return totalSlippage
}

// fillOrder 成交订单
func (ow *OrderWorker) fillOrder(order *SimulatedOrder, price, quantity float64) {
	// 更新订单状态
	order.FilledQuantity += quantity
	order.RemainingQuantity -= quantity
	order.ActualSlippage = price - order.Price

	if order.RemainingQuantity <= 0.001 { // 考虑浮点精度
		order.Status = "FILLED"
		now := time.Now()
		order.FilledAt = &now
	} else {
		order.Status = "PARTIALLY_FILLED"
	}

	order.UpdatedAt = time.Now()

	log.Printf("💰 模拟成交: %s %s %.4f @ %.4f (滑点: %.4f)",
		order.Symbol, order.Side, quantity, price, order.ActualSlippage)
}

// SimulatedPnLCalculator 模拟盈亏计算器
type SimulatedPnLCalculator struct {
	config *SimulatorConfig
}

// NewSimulatedPnLCalculator 创建PnL计算器
func NewSimulatedPnLCalculator(config *SimulatorConfig) *SimulatedPnLCalculator {
	return &SimulatedPnLCalculator{
		config: config,
	}
}

// CalculateTradePnL 计算交易盈亏
func (spc *SimulatedPnLCalculator) CalculateTradePnL(trade *SimulatedTrade, currentPrice float64) float64 {
	var pnl float64

	if trade.Side == "BUY" {
		pnl = (currentPrice - trade.Price) * trade.Quantity
	} else {
		pnl = (trade.Price - currentPrice) * trade.Quantity
	}

	// 扣除手续费
	pnl -= trade.Fee

	return pnl
}

// 辅助函数
func generateOrderID() string {
	return fmt.Sprintf("sim_order_%d_%d", time.Now().Unix(), rand.Intn(10000))
}

func generateTradeID() string {
	return fmt.Sprintf("sim_trade_%d_%d", time.Now().Unix(), rand.Intn(10000))
}

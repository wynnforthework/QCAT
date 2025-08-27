package dryrun

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"qcat/internal/exchange"
)

// MarketDataProvider 市场数据提供者接口
type MarketDataProvider interface {
	GetPrice(symbol string) (float64, error)
}

// TradingSimulator Dry-run交易模拟器
type TradingSimulator struct {
	// 配置
	config *SimulatorConfig

	// 虚拟账户系统
	virtualAccount *VirtualAccount

	// 模拟订单执行引擎
	orderEngine *SimulatedOrderEngine

	// 模拟盈亏计算器
	pnlCalculator *SimulatedPnLCalculator

	// 市场数据提供者
	marketDataProvider MarketDataProvider

	// 交易历史记录
	tradeHistory []*SimulatedTrade
	orderHistory []*SimulatedOrder
	historyMutex sync.RWMutex

	// 实时状态
	positions      map[string]*SimulatedPosition
	openOrders     map[string]*SimulatedOrder
	positionsMutex sync.RWMutex
	ordersMutex    sync.RWMutex

	// 性能统计
	performanceStats *PerformanceStats
	statsMutex       sync.RWMutex

	// 控制
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	running      bool
	runningMutex sync.RWMutex

	// 事件通知
	eventHandlers []EventHandler
	eventMutex    sync.RWMutex
}

// SimulatorConfig 模拟器配置
type SimulatorConfig struct {
	// 基础配置
	Name        string `json:"name"`
	Description string `json:"description"`

	// 虚拟账户配置
	InitialBalance map[string]float64   `json:"initial_balance"`
	TradingFee     float64              `json:"trading_fee"` // 交易手续费率
	SlippageModel  *SlippageModelConfig `json:"slippage_model"`

	// 市场模拟配置
	UseRealMarketData bool           `json:"use_real_market_data"`
	MarketDataSource  string         `json:"market_data_source"`
	LatencySimulation *LatencyConfig `json:"latency_simulation"`

	// 风险管理配置
	MaxPositionSize float64 `json:"max_position_size"`
	MaxDrawdown     float64 `json:"max_drawdown"`
	StopLossEnabled bool    `json:"stop_loss_enabled"`

	// 报告配置
	ReportingInterval  time.Duration `json:"reporting_interval"`
	EnableDetailedLogs bool          `json:"enable_detailed_logs"`

	// 高级配置
	PartialFillEnabled bool    `json:"partial_fill_enabled"`
	OrderBookDepth     int     `json:"order_book_depth"`
	PriceTickSize      float64 `json:"price_tick_size"`
}

// SlippageModelConfig 滑点模型配置
type SlippageModelConfig struct {
	Type             string  `json:"type"`              // linear, sqrt, custom
	BaseSlippage     float64 `json:"base_slippage"`     // 基础滑点
	VolumeImpact     float64 `json:"volume_impact"`     // 成交量影响系数
	VolatilityImpact float64 `json:"volatility_impact"` // 波动率影响系数
	LiquidityFactor  float64 `json:"liquidity_factor"`  // 流动性因子
}

// LatencyConfig 延迟配置
type LatencyConfig struct {
	OrderLatency  time.Duration `json:"order_latency"`  // 下单延迟
	FillLatency   time.Duration `json:"fill_latency"`   // 成交延迟
	CancelLatency time.Duration `json:"cancel_latency"` // 撤单延迟
	RandomJitter  time.Duration `json:"random_jitter"`  // 随机抖动
}

// VirtualAccount 虚拟账户
type VirtualAccount struct {
	AccountID      string             `json:"account_id"`
	Balances       map[string]float64 `json:"balances"`
	LockedBalances map[string]float64 `json:"locked_balances"`
	TotalEquity    float64            `json:"total_equity"`
	UnrealizedPnL  float64            `json:"unrealized_pnl"`
	RealizedPnL    float64            `json:"realized_pnl"`
	TotalFees      float64            `json:"total_fees"`
	TradeCount     int64              `json:"trade_count"`
	WinRate        float64            `json:"win_rate"`
	MaxDrawdown    float64            `json:"max_drawdown"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
	mutex          sync.RWMutex
}

// SimulatedOrder 模拟订单
type SimulatedOrder struct {
	ID                string     `json:"id"`
	Symbol            string     `json:"symbol"`
	Side              string     `json:"side"` // BUY, SELL
	Type              string     `json:"type"` // MARKET, LIMIT, STOP
	Quantity          float64    `json:"quantity"`
	Price             float64    `json:"price"`
	StopPrice         float64    `json:"stop_price"`
	FilledQuantity    float64    `json:"filled_quantity"`
	RemainingQuantity float64    `json:"remaining_quantity"`
	Status            string     `json:"status"`        // NEW, PARTIALLY_FILLED, FILLED, CANCELED
	TimeInForce       string     `json:"time_in_force"` // GTC, IOC, FOK
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	FilledAt          *time.Time `json:"filled_at,omitempty"`
	CanceledAt        *time.Time `json:"canceled_at,omitempty"`

	// 模拟特定字段
	ExpectedSlippage float64       `json:"expected_slippage"`
	ActualSlippage   float64       `json:"actual_slippage"`
	SimulatedLatency time.Duration `json:"simulated_latency"`
}

// SimulatedTrade 模拟交易
type SimulatedTrade struct {
	ID        string    `json:"id"`
	OrderID   string    `json:"order_id"`
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`
	Quantity  float64   `json:"quantity"`
	Price     float64   `json:"price"`
	Fee       float64   `json:"fee"`
	Slippage  float64   `json:"slippage"`
	Timestamp time.Time `json:"timestamp"`

	// PnL相关
	RealizedPnL float64 `json:"realized_pnl"`
	Commission  float64 `json:"commission"`
}

// SimulatedPosition 模拟持仓
type SimulatedPosition struct {
	Symbol        string    `json:"symbol"`
	Side          string    `json:"side"` // LONG, SHORT
	Size          float64   `json:"size"`
	EntryPrice    float64   `json:"entry_price"`
	MarkPrice     float64   `json:"mark_price"`
	UnrealizedPnL float64   `json:"unrealized_pnl"`
	RealizedPnL   float64   `json:"realized_pnl"`
	Margin        float64   `json:"margin"`
	Leverage      float64   `json:"leverage"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// PerformanceStats 性能统计
type PerformanceStats struct {
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`

	// 交易统计
	TotalTrades   int64   `json:"total_trades"`
	WinningTrades int64   `json:"winning_trades"`
	LosingTrades  int64   `json:"losing_trades"`
	WinRate       float64 `json:"win_rate"`

	// 盈亏统计
	TotalPnL  float64 `json:"total_pnl"`
	TotalFees float64 `json:"total_fees"`
	NetPnL    float64 `json:"net_pnl"`
	MaxProfit float64 `json:"max_profit"`
	MaxLoss   float64 `json:"max_loss"`

	// 风险指标
	MaxDrawdown  float64 `json:"max_drawdown"`
	SharpeRatio  float64 `json:"sharpe_ratio"`
	SortinoRatio float64 `json:"sortino_ratio"`
	CalmarRatio  float64 `json:"calmar_ratio"`

	// 其他指标
	AverageWin     float64 `json:"average_win"`
	AverageLoss    float64 `json:"average_loss"`
	ProfitFactor   float64 `json:"profit_factor"`
	RecoveryFactor float64 `json:"recovery_factor"`

	// 更新时间
	LastUpdated time.Time `json:"last_updated"`
}

// EventHandler 事件处理器接口
type EventHandler interface {
	OnOrderPlaced(order *SimulatedOrder)
	OnOrderFilled(order *SimulatedOrder, trade *SimulatedTrade)
	OnOrderCanceled(order *SimulatedOrder)
	OnPositionOpened(position *SimulatedPosition)
	OnPositionClosed(position *SimulatedPosition)
	OnPnLUpdate(stats *PerformanceStats)
}

// DefaultSimulatorConfig 默认配置
func DefaultSimulatorConfig() *SimulatorConfig {
	return &SimulatorConfig{
		Name:        "Default Dry-run Simulator",
		Description: "默认的Dry-run交易模拟器",

		InitialBalance: map[string]float64{
			"USDT": 100000.0, // 10万USDT初始资金
			"BTC":  0.0,
			"ETH":  0.0,
		},
		TradingFee: 0.001, // 0.1% 交易手续费

		SlippageModel: &SlippageModelConfig{
			Type:             "linear",
			BaseSlippage:     0.0001, // 0.01% 基础滑点
			VolumeImpact:     0.00001,
			VolatilityImpact: 0.0001,
			LiquidityFactor:  1.0,
		},

		UseRealMarketData: true,
		MarketDataSource:  "binance",

		LatencySimulation: &LatencyConfig{
			OrderLatency:  50 * time.Millisecond,
			FillLatency:   100 * time.Millisecond,
			CancelLatency: 30 * time.Millisecond,
			RandomJitter:  20 * time.Millisecond,
		},

		MaxPositionSize: 50000.0, // 最大持仓5万USDT
		MaxDrawdown:     0.2,     // 最大回撤20%
		StopLossEnabled: true,

		ReportingInterval:  time.Minute,
		EnableDetailedLogs: true,

		PartialFillEnabled: true,
		OrderBookDepth:     20,
		PriceTickSize:      0.01,
	}
}

// NewTradingSimulator 创建交易模拟器
func NewTradingSimulator(config *SimulatorConfig, marketDataProvider MarketDataProvider) (*TradingSimulator, error) {
	if config == nil {
		config = DefaultSimulatorConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	simulator := &TradingSimulator{
		config:             config,
		marketDataProvider: marketDataProvider,
		tradeHistory:       make([]*SimulatedTrade, 0),
		orderHistory:       make([]*SimulatedOrder, 0),
		positions:          make(map[string]*SimulatedPosition),
		openOrders:         make(map[string]*SimulatedOrder),
		eventHandlers:      make([]EventHandler, 0),
		ctx:                ctx,
		cancel:             cancel,
		running:            false,
	}

	// 初始化虚拟账户
	simulator.virtualAccount = NewVirtualAccount("sim_account_001", config.InitialBalance)

	// 初始化模拟订单引擎
	simulator.orderEngine = NewSimulatedOrderEngine(config, marketDataProvider)

	// 初始化PnL计算器
	simulator.pnlCalculator = NewSimulatedPnLCalculator(config)

	// 初始化性能统计
	simulator.performanceStats = &PerformanceStats{
		StartTime:   time.Now(),
		LastUpdated: time.Now(),
	}

	return simulator, nil
}

// Start 启动模拟器
func (ts *TradingSimulator) Start() error {
	ts.runningMutex.Lock()
	defer ts.runningMutex.Unlock()

	if ts.running {
		return fmt.Errorf("交易模拟器已经在运行")
	}

	ts.running = true
	ts.performanceStats.StartTime = time.Now()

	log.Printf("🚀 启动Dry-run交易模拟器: %s", ts.config.Name)

	// 启动订单引擎
	if err := ts.orderEngine.Start(); err != nil {
		ts.running = false
		return fmt.Errorf("启动订单引擎失败: %v", err)
	}

	// 启动市场数据监控
	ts.wg.Add(1)
	go ts.marketDataLoop()

	// 启动性能统计更新
	ts.wg.Add(1)
	go ts.performanceUpdateLoop()

	// 启动报告生成
	if ts.config.ReportingInterval > 0 {
		ts.wg.Add(1)
		go ts.reportingLoop()
	}

	return nil
}

// Stop 停止模拟器
func (ts *TradingSimulator) Stop() {
	ts.runningMutex.Lock()
	defer ts.runningMutex.Unlock()

	if !ts.running {
		return
	}

	log.Printf("🛑 停止Dry-run交易模拟器...")

	ts.running = false
	ts.cancel()
	ts.wg.Wait()

	// 停止订单引擎
	ts.orderEngine.Stop()

	// 生成最终报告
	ts.generateFinalReport()

	log.Printf("✅ 交易模拟器已停止")
}

// PlaceOrder 下单
func (ts *TradingSimulator) PlaceOrder(ctx context.Context, req *exchange.OrderRequest) (*exchange.OrderResponse, error) {
	if !ts.IsRunning() {
		return nil, fmt.Errorf("交易模拟器未运行")
	}

	// 创建模拟订单
	order := &SimulatedOrder{
		ID:                generateOrderID(),
		Symbol:            req.Symbol,
		Side:              req.Side,
		Type:              req.Type,
		Quantity:          req.Quantity,
		Price:             req.Price,
		RemainingQuantity: req.Quantity,
		Status:            "NEW",
		TimeInForce:       req.TimeInForce,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		SimulatedLatency:  ts.calculateLatency("order"),
	}

	// 验证订单
	if err := ts.validateOrder(order); err != nil {
		return &exchange.OrderResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// 检查账户余额
	if err := ts.checkBalance(order); err != nil {
		return &exchange.OrderResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// 锁定资金
	if err := ts.lockFunds(order); err != nil {
		return &exchange.OrderResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// 添加到订单管理
	ts.ordersMutex.Lock()
	ts.openOrders[order.ID] = order
	ts.ordersMutex.Unlock()

	ts.historyMutex.Lock()
	ts.orderHistory = append(ts.orderHistory, order)
	ts.historyMutex.Unlock()

	// 通知事件处理器
	ts.notifyOrderPlaced(order)

	// 提交给订单引擎处理
	go ts.processOrderAsync(order)

	log.Printf("📝 模拟下单: %s %s %s %.4f @ %.4f",
		order.Type, order.Side, order.Symbol, order.Quantity, order.Price)

	return &exchange.OrderResponse{
		Success: true,
		OrderID: order.ID,
		Order: &exchange.Order{
			ID:       order.ID,
			Symbol:   order.Symbol,
			Side:     order.Side,
			Type:     order.Type,
			Quantity: order.Quantity,
			Price:    order.Price,
			Status:   order.Status,
		},
	}, nil
}

// CancelOrder 撤单
func (ts *TradingSimulator) CancelOrder(ctx context.Context, orderID string) (*exchange.OrderResponse, error) {
	ts.ordersMutex.Lock()
	order, exists := ts.openOrders[orderID]
	if !exists {
		ts.ordersMutex.Unlock()
		return &exchange.OrderResponse{
			Success: false,
			Error:   "订单不存在",
		}, nil
	}

	if order.Status == "FILLED" || order.Status == "CANCELED" {
		ts.ordersMutex.Unlock()
		return &exchange.OrderResponse{
			Success: false,
			Error:   "订单已完成或已取消",
		}, nil
	}

	// 更新订单状态
	order.Status = "CANCELED"
	now := time.Now()
	order.CanceledAt = &now
	order.UpdatedAt = now

	// 从开放订单中移除
	delete(ts.openOrders, orderID)
	ts.ordersMutex.Unlock()

	// 释放锁定资金
	ts.releaseFunds(order)

	// 通知事件处理器
	ts.notifyOrderCanceled(order)

	log.Printf("❌ 模拟撤单: %s", orderID)

	return &exchange.OrderResponse{
		Success: true,
		OrderID: orderID,
	}, nil
}

// GetOrder 获取订单信息
func (ts *TradingSimulator) GetOrder(ctx context.Context, orderID string) (*exchange.Order, error) {
	ts.ordersMutex.RLock()
	order, exists := ts.openOrders[orderID]
	ts.ordersMutex.RUnlock()

	if !exists {
		// 在历史订单中查找
		ts.historyMutex.RLock()
		for _, histOrder := range ts.orderHistory {
			if histOrder.ID == orderID {
				order = histOrder
				exists = true
				break
			}
		}
		ts.historyMutex.RUnlock()
	}

	if !exists {
		return nil, fmt.Errorf("订单不存在: %s", orderID)
	}

	return &exchange.Order{
		ID:        order.ID,
		Symbol:    order.Symbol,
		Side:      order.Side,
		Type:      order.Type,
		Quantity:  order.Quantity,
		Price:     order.Price,
		FilledQty: order.FilledQuantity,
		Status:    order.Status,
	}, nil
}

// GetOpenOrders 获取开放订单
func (ts *TradingSimulator) GetOpenOrders(ctx context.Context, symbol string) ([]*exchange.Order, error) {
	ts.ordersMutex.RLock()
	defer ts.ordersMutex.RUnlock()

	orders := make([]*exchange.Order, 0)
	for _, order := range ts.openOrders {
		if symbol == "" || order.Symbol == symbol {
			orders = append(orders, &exchange.Order{
				ID:        order.ID,
				Symbol:    order.Symbol,
				Side:      order.Side,
				Type:      order.Type,
				Quantity:  order.Quantity,
				Price:     order.Price,
				FilledQty: order.FilledQuantity,
				Status:    order.Status,
			})
		}
	}

	return orders, nil
}

// GetAccountBalance 获取账户余额
func (ts *TradingSimulator) GetAccountBalance(ctx context.Context) (map[string]*exchange.AccountBalance, error) {
	ts.virtualAccount.mutex.RLock()
	defer ts.virtualAccount.mutex.RUnlock()

	balances := make(map[string]*exchange.AccountBalance)
	for asset, balance := range ts.virtualAccount.Balances {
		locked := ts.virtualAccount.LockedBalances[asset]
		balances[asset] = &exchange.AccountBalance{
			Asset:     asset,
			Total:     balance,
			Available: balance - locked,
			Locked:    locked,
			UpdatedAt: ts.virtualAccount.UpdatedAt,
		}
	}

	return balances, nil
}

// GetPositions 获取持仓信息
func (ts *TradingSimulator) GetPositions(ctx context.Context) ([]*exchange.Position, error) {
	ts.positionsMutex.RLock()
	defer ts.positionsMutex.RUnlock()

	positions := make([]*exchange.Position, 0)
	for _, pos := range ts.positions {
		positions = append(positions, &exchange.Position{
			Symbol:        pos.Symbol,
			Side:          pos.Side,
			Size:          pos.Size,
			EntryPrice:    pos.EntryPrice,
			MarkPrice:     pos.MarkPrice,
			UnrealizedPnL: pos.UnrealizedPnL,
		})
	}

	return positions, nil
}

// GetPerformanceStats 获取性能统计
func (ts *TradingSimulator) GetPerformanceStats() *PerformanceStats {
	ts.statsMutex.RLock()
	defer ts.statsMutex.RUnlock()

	// 返回副本
	stats := *ts.performanceStats
	return &stats
}

// IsRunning 检查是否正在运行
func (ts *TradingSimulator) IsRunning() bool {
	ts.runningMutex.RLock()
	defer ts.runningMutex.RUnlock()

	return ts.running
}

// AddEventHandler 添加事件处理器
func (ts *TradingSimulator) AddEventHandler(handler EventHandler) {
	ts.eventMutex.Lock()
	defer ts.eventMutex.Unlock()

	ts.eventHandlers = append(ts.eventHandlers, handler)
}

// marketDataLoop 市场数据监控循环
func (ts *TradingSimulator) marketDataLoop() {
	defer ts.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ts.updateMarketData()
		case <-ts.ctx.Done():
			return
		}
	}
}

// updateMarketData 更新市场数据
func (ts *TradingSimulator) updateMarketData() {
	// 更新持仓的标记价格和未实现盈亏
	ts.positionsMutex.Lock()
	defer ts.positionsMutex.Unlock()

	for symbol, position := range ts.positions {
		// 获取当前价格（简化实现）
		currentPrice := ts.getCurrentPrice(symbol)
		position.MarkPrice = currentPrice

		// 计算未实现盈亏
		if position.Side == "LONG" {
			position.UnrealizedPnL = (currentPrice - position.EntryPrice) * position.Size
		} else {
			position.UnrealizedPnL = (position.EntryPrice - currentPrice) * position.Size
		}

		position.UpdatedAt = time.Now()
	}
}

// performanceUpdateLoop 性能统计更新循环
func (ts *TradingSimulator) performanceUpdateLoop() {
	defer ts.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ts.updatePerformanceStats()
		case <-ts.ctx.Done():
			return
		}
	}
}

// updatePerformanceStats 更新性能统计
func (ts *TradingSimulator) updatePerformanceStats() {
	ts.statsMutex.Lock()
	defer ts.statsMutex.Unlock()

	ts.historyMutex.RLock()
	totalTrades := int64(len(ts.tradeHistory))
	ts.historyMutex.RUnlock()

	// 计算基础统计
	ts.performanceStats.TotalTrades = totalTrades
	ts.performanceStats.EndTime = time.Now()
	ts.performanceStats.Duration = ts.performanceStats.EndTime.Sub(ts.performanceStats.StartTime)

	// 计算盈亏统计
	var totalPnL, totalFees float64
	var winningTrades, losingTrades int64
	var profits, losses []float64

	ts.historyMutex.RLock()
	for _, trade := range ts.tradeHistory {
		totalPnL += trade.RealizedPnL
		totalFees += trade.Fee

		if trade.RealizedPnL > 0 {
			winningTrades++
			profits = append(profits, trade.RealizedPnL)
		} else if trade.RealizedPnL < 0 {
			losingTrades++
			losses = append(losses, math.Abs(trade.RealizedPnL))
		}
	}
	ts.historyMutex.RUnlock()

	ts.performanceStats.TotalPnL = totalPnL
	ts.performanceStats.TotalFees = totalFees
	ts.performanceStats.NetPnL = totalPnL - totalFees
	ts.performanceStats.WinningTrades = winningTrades
	ts.performanceStats.LosingTrades = losingTrades

	if totalTrades > 0 {
		ts.performanceStats.WinRate = float64(winningTrades) / float64(totalTrades)
	}

	// 计算平均盈亏
	if len(profits) > 0 {
		sum := 0.0
		for _, profit := range profits {
			sum += profit
		}
		ts.performanceStats.AverageWin = sum / float64(len(profits))
	}

	if len(losses) > 0 {
		sum := 0.0
		for _, loss := range losses {
			sum += loss
		}
		ts.performanceStats.AverageLoss = sum / float64(len(losses))
	}

	// 计算盈亏比
	if ts.performanceStats.AverageLoss > 0 {
		ts.performanceStats.ProfitFactor = ts.performanceStats.AverageWin / ts.performanceStats.AverageLoss
	}

	ts.performanceStats.LastUpdated = time.Now()
}

// reportingLoop 报告生成循环
func (ts *TradingSimulator) reportingLoop() {
	defer ts.wg.Done()

	ticker := time.NewTicker(ts.config.ReportingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ts.generatePerformanceReport()
		case <-ts.ctx.Done():
			return
		}
	}
}

// generatePerformanceReport 生成性能报告
func (ts *TradingSimulator) generatePerformanceReport() {
	stats := ts.GetPerformanceStats()

	log.Printf("📊 Dry-run交易报告:")
	log.Printf("   运行时间: %v", stats.Duration)
	log.Printf("   总交易数: %d", stats.TotalTrades)
	log.Printf("   胜率: %.2f%%", stats.WinRate*100)
	log.Printf("   总盈亏: %.4f USDT", stats.TotalPnL)
	log.Printf("   净盈亏: %.4f USDT", stats.NetPnL)
	log.Printf("   手续费: %.4f USDT", stats.TotalFees)

	if stats.AverageLoss > 0 {
		log.Printf("   盈亏比: %.2f", stats.ProfitFactor)
	}
}

// generateFinalReport 生成最终报告
func (ts *TradingSimulator) generateFinalReport() {
	ts.updatePerformanceStats()

	stats := ts.GetPerformanceStats()

	log.Printf("🎯 Dry-run交易最终报告:")
	log.Printf("==========================================")
	log.Printf("模拟器名称: %s", ts.config.Name)
	log.Printf("运行时间: %v", stats.Duration)
	log.Printf("总交易数: %d", stats.TotalTrades)
	log.Printf("获胜交易: %d", stats.WinningTrades)
	log.Printf("失败交易: %d", stats.LosingTrades)
	log.Printf("胜率: %.2f%%", stats.WinRate*100)
	log.Printf("总盈亏: %.4f USDT", stats.TotalPnL)
	log.Printf("总手续费: %.4f USDT", stats.TotalFees)
	log.Printf("净盈亏: %.4f USDT", stats.NetPnL)
	log.Printf("平均盈利: %.4f USDT", stats.AverageWin)
	log.Printf("平均亏损: %.4f USDT", stats.AverageLoss)
	if stats.AverageLoss > 0 {
		log.Printf("盈亏比: %.2f", stats.ProfitFactor)
	}
	log.Printf("==========================================")
}

// processOrderAsync 异步处理订单
func (ts *TradingSimulator) processOrderAsync(order *SimulatedOrder) {
	// 提交给订单引擎处理
	ts.orderEngine.ProcessOrder(order)
}

// validateOrder 验证订单
func (ts *TradingSimulator) validateOrder(order *SimulatedOrder) error {
	if order.Symbol == "" {
		return fmt.Errorf("交易对不能为空")
	}

	if order.Side != "BUY" && order.Side != "SELL" {
		return fmt.Errorf("无效的订单方向: %s", order.Side)
	}

	if order.Quantity <= 0 {
		return fmt.Errorf("订单数量必须大于0")
	}

	if order.Type == "LIMIT" && order.Price <= 0 {
		return fmt.Errorf("限价单价格必须大于0")
	}

	return nil
}

// checkBalance 检查账户余额
func (ts *TradingSimulator) checkBalance(order *SimulatedOrder) error {
	ts.virtualAccount.mutex.RLock()
	defer ts.virtualAccount.mutex.RUnlock()

	var requiredAsset string
	var requiredAmount float64

	if order.Side == "BUY" {
		// 买单需要报价货币
		requiredAsset = "USDT" // 简化处理
		if order.Type == "MARKET" {
			// 市价单需要估算成本
			estimatedPrice := ts.getCurrentPrice(order.Symbol)
			requiredAmount = order.Quantity * estimatedPrice * 1.01 // 1%缓冲
		} else {
			requiredAmount = order.Quantity * order.Price
		}
	} else {
		// 卖单需要基础货币
		requiredAsset = "BTC" // 简化处理
		requiredAmount = order.Quantity
	}

	available := ts.virtualAccount.Balances[requiredAsset] - ts.virtualAccount.LockedBalances[requiredAsset]
	if available < requiredAmount {
		return fmt.Errorf("余额不足: %s 可用 %.4f, 需要 %.4f",
			requiredAsset, available, requiredAmount)
	}

	return nil
}

// lockFunds 锁定资金
func (ts *TradingSimulator) lockFunds(order *SimulatedOrder) error {
	var asset string
	var amount float64

	if order.Side == "BUY" {
		asset = "USDT"
		if order.Type == "MARKET" {
			estimatedPrice := ts.getCurrentPrice(order.Symbol)
			amount = order.Quantity * estimatedPrice * 1.01
		} else {
			amount = order.Quantity * order.Price
		}
	} else {
		asset = "BTC"
		amount = order.Quantity
	}

	return ts.virtualAccount.LockBalance(asset, amount)
}

// releaseFunds 释放资金
func (ts *TradingSimulator) releaseFunds(order *SimulatedOrder) {
	var asset string
	var amount float64

	if order.Side == "BUY" {
		asset = "USDT"
		amount = order.RemainingQuantity * order.Price
	} else {
		asset = "BTC"
		amount = order.RemainingQuantity
	}

	ts.virtualAccount.UnlockBalance(asset, amount)
}

// getCurrentPrice 获取当前价格（简化实现）
func (ts *TradingSimulator) getCurrentPrice(symbol string) float64 {
	// 简化实现，返回模拟价格
	basePrice := 50000.0
	if symbol == "ETHUSDT" {
		basePrice = 3000.0
	}
	return basePrice
}

// calculateLatency 计算延迟
func (ts *TradingSimulator) calculateLatency(operationType string) time.Duration {
	config := ts.config.LatencySimulation
	var baseLatency time.Duration

	switch operationType {
	case "order":
		baseLatency = config.OrderLatency
	case "fill":
		baseLatency = config.FillLatency
	case "cancel":
		baseLatency = config.CancelLatency
	default:
		baseLatency = 50 * time.Millisecond
	}

	// 添加随机抖动
	jitter := time.Duration(rand.Int63n(int64(config.RandomJitter)))
	return baseLatency + jitter
}

// 事件通知方法
func (ts *TradingSimulator) notifyOrderPlaced(order *SimulatedOrder) {
	ts.eventMutex.RLock()
	handlers := make([]EventHandler, len(ts.eventHandlers))
	copy(handlers, ts.eventHandlers)
	ts.eventMutex.RUnlock()

	for _, handler := range handlers {
		go handler.OnOrderPlaced(order)
	}
}

func (ts *TradingSimulator) notifyOrderFilled(order *SimulatedOrder, trade *SimulatedTrade) {
	ts.eventMutex.RLock()
	handlers := make([]EventHandler, len(ts.eventHandlers))
	copy(handlers, ts.eventHandlers)
	ts.eventMutex.RUnlock()

	for _, handler := range handlers {
		go handler.OnOrderFilled(order, trade)
	}
}

func (ts *TradingSimulator) notifyOrderCanceled(order *SimulatedOrder) {
	ts.eventMutex.RLock()
	handlers := make([]EventHandler, len(ts.eventHandlers))
	copy(handlers, ts.eventHandlers)
	ts.eventMutex.RUnlock()

	for _, handler := range handlers {
		go handler.OnOrderCanceled(order)
	}
}

func (ts *TradingSimulator) notifyPositionOpened(position *SimulatedPosition) {
	ts.eventMutex.RLock()
	handlers := make([]EventHandler, len(ts.eventHandlers))
	copy(handlers, ts.eventHandlers)
	ts.eventMutex.RUnlock()

	for _, handler := range handlers {
		go handler.OnPositionOpened(position)
	}
}

func (ts *TradingSimulator) notifyPositionClosed(position *SimulatedPosition) {
	ts.eventMutex.RLock()
	handlers := make([]EventHandler, len(ts.eventHandlers))
	copy(handlers, ts.eventHandlers)
	ts.eventMutex.RUnlock()

	for _, handler := range handlers {
		go handler.OnPositionClosed(position)
	}
}

func (ts *TradingSimulator) notifyPnLUpdate(stats *PerformanceStats) {
	ts.eventMutex.RLock()
	handlers := make([]EventHandler, len(ts.eventHandlers))
	copy(handlers, ts.eventHandlers)
	ts.eventMutex.RUnlock()

	for _, handler := range handlers {
		go handler.OnPnLUpdate(stats)
	}
}

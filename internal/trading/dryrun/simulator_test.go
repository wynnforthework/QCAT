package dryrun

import (
	"context"
	"testing"
	"time"

	"qcat/internal/exchange"
)

// MockMarketDataProvider 模拟市场数据提供者
type MockMarketDataProvider struct{}

func (m *MockMarketDataProvider) GetPrice(symbol string) (float64, error) {
	switch symbol {
	case "BTCUSDT":
		return 50000.0, nil
	case "ETHUSDT":
		return 3000.0, nil
	default:
		return 1.0, nil
	}
}

// TestEventHandler 测试事件处理器
type TestEventHandler struct {
	OrdersPlaced   []*SimulatedOrder
	OrdersFilled   []*SimulatedOrder
	OrdersCanceled []*SimulatedOrder
	PositionsOpened []*SimulatedPosition
	PositionsClosed []*SimulatedPosition
	PnLUpdates     []*PerformanceStats
}

func (teh *TestEventHandler) OnOrderPlaced(order *SimulatedOrder) {
	teh.OrdersPlaced = append(teh.OrdersPlaced, order)
}

func (teh *TestEventHandler) OnOrderFilled(order *SimulatedOrder, trade *SimulatedTrade) {
	teh.OrdersFilled = append(teh.OrdersFilled, order)
}

func (teh *TestEventHandler) OnOrderCanceled(order *SimulatedOrder) {
	teh.OrdersCanceled = append(teh.OrdersCanceled, order)
}

func (teh *TestEventHandler) OnPositionOpened(position *SimulatedPosition) {
	teh.PositionsOpened = append(teh.PositionsOpened, position)
}

func (teh *TestEventHandler) OnPositionClosed(position *SimulatedPosition) {
	teh.PositionsClosed = append(teh.PositionsClosed, position)
}

func (teh *TestEventHandler) OnPnLUpdate(stats *PerformanceStats) {
	teh.PnLUpdates = append(teh.PnLUpdates, stats)
}

func TestTradingSimulator_Creation(t *testing.T) {
	config := DefaultSimulatorConfig()
	mockProvider := &MockMarketDataProvider{}
	
	simulator, err := NewTradingSimulator(config, mockProvider)
	if err != nil {
		t.Fatalf("创建交易模拟器失败: %v", err)
	}
	
	if simulator == nil {
		t.Fatal("模拟器不应该为空")
	}
	
	if simulator.IsRunning() {
		t.Error("新创建的模拟器不应该在运行")
	}
	
	if simulator.config.Name != config.Name {
		t.Errorf("期望配置名称 %s，实际得到 %s", config.Name, simulator.config.Name)
	}
	
	// 检查虚拟账户
	if simulator.virtualAccount == nil {
		t.Fatal("虚拟账户不应该为空")
	}
	
	if simulator.virtualAccount.Balances["USDT"] != 100000.0 {
		t.Errorf("期望初始USDT余额 100000.0，实际得到 %f", simulator.virtualAccount.Balances["USDT"])
	}
}

func TestTradingSimulator_DefaultConfig(t *testing.T) {
	config := DefaultSimulatorConfig()
	
	if config == nil {
		t.Fatal("默认配置不应该为空")
	}
	
	if config.TradingFee <= 0 {
		t.Error("交易手续费应该大于0")
	}
	
	if config.InitialBalance["USDT"] <= 0 {
		t.Error("初始USDT余额应该大于0")
	}
	
	if config.SlippageModel == nil {
		t.Error("滑点模型不应该为空")
	}
	
	if config.LatencySimulation == nil {
		t.Error("延迟配置不应该为空")
	}
}

func TestTradingSimulator_StartStop(t *testing.T) {
	config := DefaultSimulatorConfig()
	config.ReportingInterval = 100 * time.Millisecond // 快速报告用于测试
	mockProvider := &MockMarketDataProvider{}
	
	simulator, err := NewTradingSimulator(config, mockProvider)
	if err != nil {
		t.Fatalf("创建模拟器失败: %v", err)
	}
	
	// 测试启动
	err = simulator.Start()
	if err != nil {
		t.Fatalf("启动模拟器失败: %v", err)
	}
	
	if !simulator.IsRunning() {
		t.Error("启动后应该在运行状态")
	}
	
	// 测试重复启动
	err = simulator.Start()
	if err == nil {
		t.Error("重复启动应该返回错误")
	}
	
	// 等待一段时间让模拟器运行
	time.Sleep(200 * time.Millisecond)
	
	// 测试停止
	simulator.Stop()
	
	if simulator.IsRunning() {
		t.Error("停止后不应该在运行状态")
	}
}

func TestTradingSimulator_PlaceOrder(t *testing.T) {
	config := DefaultSimulatorConfig()
	config.ReportingInterval = 0 // 禁用报告
	mockProvider := &MockMarketDataProvider{}
	
	simulator, err := NewTradingSimulator(config, mockProvider)
	if err != nil {
		t.Fatalf("创建模拟器失败: %v", err)
	}
	
	// 添加事件处理器
	eventHandler := &TestEventHandler{}
	simulator.AddEventHandler(eventHandler)
	
	// 启动模拟器
	err = simulator.Start()
	if err != nil {
		t.Fatalf("启动模拟器失败: %v", err)
	}
	defer simulator.Stop()
	
	ctx := context.Background()
	
	// 测试市价买单
	orderReq := &exchange.OrderRequest{
		Symbol:   "BTCUSDT",
		Side:     "BUY",
		Type:     "MARKET",
		Quantity: 0.1,
	}
	
	resp, err := simulator.PlaceOrder(ctx, orderReq)
	if err != nil {
		t.Fatalf("下单失败: %v", err)
	}
	
	if !resp.Success {
		t.Errorf("下单应该成功: %s", resp.Error)
	}
	
	if resp.OrderID == "" {
		t.Error("订单ID不应该为空")
	}
	
	// 等待事件处理
	time.Sleep(100 * time.Millisecond)
	
	// 检查事件是否触发
	if len(eventHandler.OrdersPlaced) == 0 {
		t.Error("应该触发订单下达事件")
	}
	
	// 检查订单状态
	order, err := simulator.GetOrder(ctx, resp.OrderID)
	if err != nil {
		t.Fatalf("获取订单失败: %v", err)
	}
	
	if order.Symbol != "BTCUSDT" {
		t.Errorf("期望交易对 BTCUSDT，实际得到 %s", order.Symbol)
	}
	
	if order.Side != "BUY" {
		t.Errorf("期望订单方向 BUY，实际得到 %s", order.Side)
	}
	
	if order.Quantity != 0.1 {
		t.Errorf("期望订单数量 0.1，实际得到 %f", order.Quantity)
	}
}

func TestTradingSimulator_CancelOrder(t *testing.T) {
	config := DefaultSimulatorConfig()
	config.ReportingInterval = 0
	mockProvider := &MockMarketDataProvider{}
	
	simulator, err := NewTradingSimulator(config, mockProvider)
	if err != nil {
		t.Fatalf("创建模拟器失败: %v", err)
	}
	
	eventHandler := &TestEventHandler{}
	simulator.AddEventHandler(eventHandler)
	
	err = simulator.Start()
	if err != nil {
		t.Fatalf("启动模拟器失败: %v", err)
	}
	defer simulator.Stop()
	
	ctx := context.Background()
	
	// 下一个限价单（不会立即成交）
	orderReq := &exchange.OrderRequest{
		Symbol:   "BTCUSDT",
		Side:     "BUY",
		Type:     "LIMIT",
		Quantity: 0.1,
		Price:    40000.0, // 低于市价，不会立即成交
	}
	
	resp, err := simulator.PlaceOrder(ctx, orderReq)
	if err != nil {
		t.Fatalf("下单失败: %v", err)
	}
	
	// 撤销订单
	cancelResp, err := simulator.CancelOrder(ctx, resp.OrderID)
	if err != nil {
		t.Fatalf("撤单失败: %v", err)
	}
	
	if !cancelResp.Success {
		t.Errorf("撤单应该成功: %s", cancelResp.Error)
	}
	
	// 等待事件处理
	time.Sleep(100 * time.Millisecond)
	
	// 检查撤单事件
	if len(eventHandler.OrdersCanceled) == 0 {
		t.Error("应该触发订单撤销事件")
	}
	
	// 检查订单状态
	order, err := simulator.GetOrder(ctx, resp.OrderID)
	if err != nil {
		t.Fatalf("获取订单失败: %v", err)
	}
	
	if order.Status != "CANCELED" {
		t.Errorf("期望订单状态 CANCELED，实际得到 %s", order.Status)
	}
}

func TestTradingSimulator_GetAccountBalance(t *testing.T) {
	config := DefaultSimulatorConfig()
	mockProvider := &MockMarketDataProvider{}
	
	simulator, err := NewTradingSimulator(config, mockProvider)
	if err != nil {
		t.Fatalf("创建模拟器失败: %v", err)
	}
	
	ctx := context.Background()
	
	balances, err := simulator.GetAccountBalance(ctx)
	if err != nil {
		t.Fatalf("获取账户余额失败: %v", err)
	}
	
	if len(balances) == 0 {
		t.Error("应该有账户余额")
	}
	
	usdtBalance, exists := balances["USDT"]
	if !exists {
		t.Error("应该有USDT余额")
	}
	
	if usdtBalance.Total != 100000.0 {
		t.Errorf("期望USDT总余额 100000.0，实际得到 %f", usdtBalance.Total)
	}
	
	if usdtBalance.Available != 100000.0 {
		t.Errorf("期望USDT可用余额 100000.0，实际得到 %f", usdtBalance.Available)
	}
}

func TestTradingSimulator_GetPerformanceStats(t *testing.T) {
	config := DefaultSimulatorConfig()
	mockProvider := &MockMarketDataProvider{}
	
	simulator, err := NewTradingSimulator(config, mockProvider)
	if err != nil {
		t.Fatalf("创建模拟器失败: %v", err)
	}
	
	stats := simulator.GetPerformanceStats()
	if stats == nil {
		t.Fatal("性能统计不应该为空")
	}
	
	if stats.TotalTrades != 0 {
		t.Errorf("期望初始交易数量 0，实际得到 %d", stats.TotalTrades)
	}
	
	if stats.WinRate != 0 {
		t.Errorf("期望初始胜率 0，实际得到 %f", stats.WinRate)
	}
	
	if stats.TotalPnL != 0 {
		t.Errorf("期望初始总盈亏 0，实际得到 %f", stats.TotalPnL)
	}
}

func TestVirtualAccount_Operations(t *testing.T) {
	initialBalances := map[string]float64{
		"USDT": 10000.0,
		"BTC":  1.0,
	}
	
	account := NewVirtualAccount("test_account", initialBalances)
	
	// 测试初始余额
	if account.Balances["USDT"] != 10000.0 {
		t.Errorf("期望USDT余额 10000.0，实际得到 %f", account.Balances["USDT"])
	}
	
	// 测试余额更新
	account.UpdateBalance("USDT", 1000.0)
	if account.Balances["USDT"] != 11000.0 {
		t.Errorf("期望更新后USDT余额 11000.0，实际得到 %f", account.Balances["USDT"])
	}
	
	// 测试余额锁定
	err := account.LockBalance("USDT", 5000.0)
	if err != nil {
		t.Fatalf("锁定余额失败: %v", err)
	}
	
	if account.LockedBalances["USDT"] != 5000.0 {
		t.Errorf("期望锁定USDT余额 5000.0，实际得到 %f", account.LockedBalances["USDT"])
	}
	
	// 测试余额不足
	err = account.LockBalance("USDT", 10000.0)
	if err == nil {
		t.Error("余额不足时应该返回错误")
	}
	
	// 测试解锁余额
	account.UnlockBalance("USDT", 2000.0)
	if account.LockedBalances["USDT"] != 3000.0 {
		t.Errorf("期望解锁后锁定USDT余额 3000.0，实际得到 %f", account.LockedBalances["USDT"])
	}
}

package protector

import (
	"context"
	"testing"
	"time"

	"qcat/internal/config"
	"qcat/internal/exchange"
	"qcat/internal/security/protector/dao"
)

// TestFundProtectorCreation 测试资金保护器创建
func TestFundProtectorCreation(t *testing.T) {
	cfg := &config.Config{
		Risk: config.RiskConfig{
			MaxDrawdown:   0.05,
			CheckInterval: 5 * time.Minute,
		},
	}

	exchangeProvider := NewMockExchangeProvider()
	mockNotificationService := NewMockNotificationService()
	mockWalletService := NewMockWalletService()
	var mockExchange exchange.Exchange
	var mockDAOManager dao.DAOManager

	fp, err := NewFundProtector(cfg, exchangeProvider, mockExchange, mockDAOManager, mockNotificationService, mockWalletService)
	if err != nil {
		t.Fatalf("Failed to create fund protector: %v", err)
	}

	if fp == nil {
		t.Fatal("Fund protector is nil")
	}

	if fp.maxDailyLoss != 0.05 {
		t.Errorf("Expected max daily loss 0.05, got %f", fp.maxDailyLoss)
	}

	if fp.checkInterval != 5*time.Minute {
		t.Errorf("Expected check interval 5m, got %v", fp.checkInterval)
	}
}

// TestExchangeDataProvider 测试交易所数据提供者
func TestExchangeDataProvider(t *testing.T) {
	provider := NewMockExchangeProvider()

	// 测试健康检查
	if !provider.IsHealthy() {
		t.Error("Mock provider should be healthy")
	}

	// 测试获取资金数据
	ctx := context.Background()
	fundData, err := provider.GetFundData(ctx)
	if err != nil {
		t.Fatalf("Failed to get fund data: %v", err)
	}

	if fundData.TotalBalance != 100000.0 {
		t.Errorf("Expected total balance 100000, got %f", fundData.TotalBalance)
	}

	// 测试获取持仓数据
	positions, err := provider.GetPositions(ctx)
	if err != nil {
		t.Fatalf("Failed to get positions: %v", err)
	}

	if len(positions) != 1 {
		t.Errorf("Expected 1 position, got %d", len(positions))
	}

	if positions[0].Symbol != "BTCUSDT" {
		t.Errorf("Expected BTCUSDT, got %s", positions[0].Symbol)
	}

	// 测试获取历史收益率
	returns, err := provider.GetHistoricalReturns(ctx, 30)
	if err != nil {
		t.Fatalf("Failed to get historical returns: %v", err)
	}

	if len(returns) != 30 {
		t.Errorf("Expected 30 returns, got %d", len(returns))
	}
}

// TestNotificationService 测试通知服务
func TestNotificationService(t *testing.T) {
	service := NewMockNotificationService()
	ctx := context.Background()

	// 测试邮件发送
	err := service.SendEmail(ctx, "test@example.com", "Test Subject", "Test Body")
	if err != nil {
		t.Fatalf("Failed to send email: %v", err)
	}

	emails := service.GetEmailsSent()
	if len(emails) != 1 {
		t.Errorf("Expected 1 email sent, got %d", len(emails))
	}

	// 测试短信发送
	err = service.SendSMS(ctx, "+1234567890", "Test SMS")
	if err != nil {
		t.Fatalf("Failed to send SMS: %v", err)
	}

	sms := service.GetSMSSent()
	if len(sms) != 1 {
		t.Errorf("Expected 1 SMS sent, got %d", len(sms))
	}

	// 测试Webhook发送
	payload := map[string]interface{}{"test": "data"}
	err = service.SendWebhook(ctx, "https://example.com/webhook", payload)
	if err != nil {
		t.Fatalf("Failed to send webhook: %v", err)
	}

	webhooks := service.GetWebhooksSent()
	if len(webhooks) != 1 {
		t.Errorf("Expected 1 webhook sent, got %d", len(webhooks))
	}

	// 测试失败场景
	service.SetShouldFail(true)
	err = service.SendEmail(ctx, "test@example.com", "Test", "Test")
	if err == nil {
		t.Error("Expected email to fail, but it succeeded")
	}
}

// TestWalletService 测试钱包服务
func TestWalletService(t *testing.T) {
	service := NewMockWalletService()
	ctx := context.Background()

	// 测试转账
	request := &TransferRequest{
		Type:        "PROFIT_TRANSFER",
		Amount:      1000.0,
		FromAddress: "0x123",
		ToAddress:   "0x456",
		Priority:    5,
	}

	response, err := service.InitiateTransfer(ctx, request)
	if err != nil {
		t.Fatalf("Failed to initiate transfer: %v", err)
	}

	if response.Status != "CONFIRMED" {
		t.Errorf("Expected status CONFIRMED, got %s", response.Status)
	}

	// 测试获取转账状态
	status, err := service.GetTransferStatus(ctx, response.TransferID)
	if err != nil {
		t.Fatalf("Failed to get transfer status: %v", err)
	}

	if status.Status != "CONFIRMED" {
		t.Errorf("Expected status CONFIRMED, got %s", status.Status)
	}

	// 测试地址验证
	err = service.ValidateAddress("0x1234567890123456789012345678901234567890")
	if err != nil {
		t.Fatalf("Failed to validate address: %v", err)
	}

	err = service.ValidateAddress("invalid")
	if err == nil {
		t.Error("Expected address validation to fail, but it succeeded")
	}

	// 测试手续费估算
	fee, err := service.EstimateTransferFee(ctx, 1000.0, "0x456")
	if err != nil {
		t.Fatalf("Failed to estimate fee: %v", err)
	}

	if fee != 0.001 {
		t.Errorf("Expected fee 0.001, got %f", fee)
	}
}

// TestRiskCalculations 测试风险计算
func TestRiskCalculations(t *testing.T) {
	cfg := &config.Config{
		Risk: config.RiskConfig{
			MaxDrawdown:   0.05,
			CheckInterval: 5 * time.Minute,
		},
	}

	exchangeProvider := NewMockExchangeProvider()
	notificationService := NewMockNotificationService()
	walletService := NewMockWalletService()
	var mockExchange exchange.Exchange
	var mockDAOManager dao.DAOManager

	fp, err := NewFundProtector(cfg, exchangeProvider, mockExchange, mockDAOManager, notificationService, walletService)
	if err != nil {
		t.Fatalf("Failed to create fund protector: %v", err)
	}

	// 测试VaR计算
	returns := []float64{0.01, -0.02, 0.015, -0.01, 0.005, -0.008, 0.012, -0.015, 0.02, -0.005}
	var95 := fp.calculateVaRFromReturns(returns, 0.95)
	if var95 <= 0 {
		t.Errorf("Expected positive VaR, got %f", var95)
	}

	// 测试Expected Shortfall计算
	es := fp.calculateExpectedShortfallFromReturns(returns, 0.95)
	if es <= 0 {
		t.Errorf("Expected positive Expected Shortfall, got %f", es)
	}

	// 测试波动率计算
	volatility := fp.calculateVolatilityFromReturns(returns)
	if volatility <= 0 {
		t.Errorf("Expected positive volatility, got %f", volatility)
	}

	// 测试回撤计算
	equity := []float64{100000, 102000, 98000, 105000, 103000, 99000, 107000}
	drawdown := fp.calculateDrawdownFromEquity(equity)
	if drawdown <= 0 {
		t.Errorf("Expected positive drawdown, got %f", drawdown)
	}
}

// TestEmergencyProtocol 测试紧急协议
func TestEmergencyProtocol(t *testing.T) {
	cfg := &config.Config{
		Risk: config.RiskConfig{
			MaxDrawdown:   0.05,
			CheckInterval: 5 * time.Minute,
		},
	}

	exchangeProvider := NewMockExchangeProvider()
	mockNotificationService := NewMockNotificationService()
	walletService := NewMockWalletService()
	var mockExchange exchange.Exchange
	var mockDAOManager dao.DAOManager

	fp, err := NewFundProtector(cfg, exchangeProvider, mockExchange, mockDAOManager, mockNotificationService, walletService)
	if err != nil {
		t.Fatalf("Failed to create fund protector: %v", err)
	}

	// 添加紧急联系人
	contact := EmergencyContact{
		Name:        "Test Manager",
		Role:        "ADMIN",
		Email:       "manager@example.com",
		Phone:       "+1234567890",
		Priority:    1,
		IsAvailable: true,
		Channels:    []string{"EMAIL", "SMS"},
	}

	fp.emergencyProtocol.emergencyContacts = append(fp.emergencyProtocol.emergencyContacts, contact)

	// 触发紧急事件
	triggerData := map[string]interface{}{
		"daily_loss_ratio": 0.08,
		"max_daily_loss":   0.05,
	}

	fp.triggerEmergency("DAILY_LOSS_EXCEEDED", triggerData)

	// 验证紧急事件被记录
	if len(fp.emergencyEvents) != 1 {
		t.Errorf("Expected 1 emergency event, got %d", len(fp.emergencyEvents))
	}

	event := fp.emergencyEvents[0]
	if event.Type != "DAILY_LOSS_EXCEEDED" {
		t.Errorf("Expected event type DAILY_LOSS_EXCEEDED, got %s", event.Type)
	}

	// 验证通知被发送
	emails := mockNotificationService.GetEmailsSent()
	if len(emails) == 0 {
		t.Error("Expected emergency email to be sent")
	}

	sms := mockNotificationService.GetSMSSent()
	if len(sms) == 0 {
		t.Error("Expected emergency SMS to be sent")
	}
}

// TestCircuitBreaker 测试熔断器
func TestCircuitBreaker(t *testing.T) {
	cfg := &config.Config{
		Risk: config.RiskConfig{
			MaxDrawdown:   0.05,
			CheckInterval: 5 * time.Minute,
		},
	}

	exchangeProvider := NewMockExchangeProvider()
	notificationService := NewMockNotificationService()
	walletService := NewMockWalletService()
	var mockExchange exchange.Exchange
	var mockDAOManager dao.DAOManager

	fp, err := NewFundProtector(cfg, exchangeProvider, mockExchange, mockDAOManager, notificationService, walletService)
	if err != nil {
		t.Fatalf("Failed to create fund protector: %v", err)
	}

	// 设置资金状态以触发熔断器
	fp.fundStatus.TotalBalance = 100000.0
	fp.fundStatus.DailyPL = -6000.0 // 6%的日亏损，超过5%的限制

	// 检查熔断器
	fp.checkCircuitBreaker()

	// 验证熔断器被触发
	if !fp.circuitBreaker.isOpen {
		t.Error("Expected circuit breaker to be open")
	}

	if !fp.circuitBreakerOpen {
		t.Error("Expected circuit breaker open flag to be true")
	}

	// 验证紧急事件被触发
	if len(fp.emergencyEvents) == 0 {
		t.Error("Expected emergency event to be triggered")
	}
}

// TestAutoTransfer 测试自动转账
func TestAutoTransfer(t *testing.T) {
	cfg := &config.Config{
		Risk: config.RiskConfig{
			MaxDrawdown:   0.05,
			CheckInterval: 5 * time.Minute,
		},
	}

	exchangeProvider := NewMockExchangeProvider()
	notificationService := NewMockNotificationService()
	walletService := NewMockWalletService()
	var mockExchange exchange.Exchange
	var mockDAOManager dao.DAOManager

	fp, err := NewFundProtector(cfg, exchangeProvider, mockExchange, mockDAOManager, notificationService, walletService)
	if err != nil {
		t.Fatalf("Failed to create fund protector: %v", err)
	}

	// 设置冷钱包地址
	fp.autoTransferManager.coldWalletAddress = "0x789"

	// 设置资金状态以触发自动转账
	fp.fundStatus.TotalBalance = 100000.0
	fp.fundStatus.RealizedPL = 15000.0 // 15%的已实现利润，超过10%的阈值

	// 检查自动转账
	fp.checkAutoTransfer()

	// 验证转账被执行
	transfers := walletService.GetTransfers()
	if len(transfers) == 0 {
		t.Error("Expected auto transfer to be executed")
	}

	// 验证转账历史被记录
	if len(fp.transferHistory) == 0 {
		t.Error("Expected transfer to be recorded in history")
	}
}

// TestProtectionMetrics 测试保护指标
func TestProtectionMetrics(t *testing.T) {
	cfg := &config.Config{
		Risk: config.RiskConfig{
			MaxDrawdown:   0.05,
			CheckInterval: 5 * time.Minute,
		},
	}

	exchangeProvider := NewMockExchangeProvider()
	notificationService := NewMockNotificationService()
	walletService := NewMockWalletService()
	var mockExchange exchange.Exchange
	var mockDAOManager dao.DAOManager

	fp, err := NewFundProtector(cfg, exchangeProvider, mockExchange, mockDAOManager, notificationService, walletService)
	if err != nil {
		t.Fatalf("Failed to create fund protector: %v", err)
	}

	// 模拟一些保护活动
	fp.protectionMetrics.CircuitBreakerTriggered = 2
	fp.protectionMetrics.EmergencyActivations = 3
	fp.protectionMetrics.AutoTransfers = 5
	fp.protectionMetrics.LossesPrevented = 10000.0

	// 更新保护指标
	fp.updateProtectionMetrics()

	// 验证准确率计算
	if fp.protectionMetrics.ProtectionAccuracy <= 0 {
		t.Error("Expected positive protection accuracy")
	}

	if fp.protectionMetrics.FalsePositiveRate < 0 || fp.protectionMetrics.FalsePositiveRate > 1 {
		t.Errorf("Expected false positive rate between 0 and 1, got %f", fp.protectionMetrics.FalsePositiveRate)
	}

	// 获取保护指标
	metrics := fp.GetProtectionMetrics()
	if metrics.CircuitBreakerTriggered != 2 {
		t.Errorf("Expected 2 circuit breaker triggers, got %d", metrics.CircuitBreakerTriggered)
	}
}

// BenchmarkRiskCalculation 风险计算性能基准测试
func BenchmarkRiskCalculation(b *testing.B) {
	cfg := &config.Config{
		Risk: config.RiskConfig{
			MaxDrawdown:   0.05,
			CheckInterval: 5 * time.Minute,
		},
	}

	exchangeProvider := NewMockExchangeProvider()
	notificationService := NewMockNotificationService()
	walletService := NewMockWalletService()
	var mockExchange exchange.Exchange
	var mockDAOManager dao.DAOManager

	fp, err := NewFundProtector(cfg, exchangeProvider, mockExchange, mockDAOManager, notificationService, walletService)
	if err != nil {
		b.Fatalf("Failed to create fund protector: %v", err)
	}

	// 生成测试数据
	returns := make([]float64, 252) // 一年的日收益率
	for i := 0; i < 252; i++ {
		returns[i] = (float64(i%10) - 5) * 0.01 // -5% to +4%
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fp.calculateVaRFromReturns(returns, 0.95)
	}
}

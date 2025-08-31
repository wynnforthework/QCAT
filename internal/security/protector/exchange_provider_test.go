package protector

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// MockExchangeProvider 测试用交易所数据提供者，实现完整的 ExchangeDataProvider 接口
type MockExchangeProvider struct {
	fundData  *ExchangeFundData
	positions []*Position
	healthy   bool
}

// NewMockExchangeProvider 创建模拟交易所数据提供者
func NewMockExchangeProvider() *MockExchangeProvider {
	return &MockExchangeProvider{
		fundData: &ExchangeFundData{
			TotalBalance:     100000.0,
			AvailableBalance: 80000.0,
			LockedBalance:    20000.0,
			DailyPL:          1500.0,
			UnrealizedPL:     2500.0,
			Timestamp:        time.Now(),
		},
		positions: []*Position{
			{
				Symbol:           "BTCUSDT",
				Side:             "LONG",
				Size:             1.5,
				Notional:         75000.0,
				EntryPrice:       48000.0,
				MarkPrice:        50000.0,
				UnrealizedPnL:    3000.0,
				Leverage:         10,
				MarginType:       "CROSS",
				LiquidationPrice: 45000.0,
			},
		},
		healthy: true,
	}
}

// IsHealthy 检查连接健康状态
func (m *MockExchangeProvider) IsHealthy() bool {
	return m.healthy
}

// GetFundData 获取资金数据
func (m *MockExchangeProvider) GetFundData(ctx context.Context) (*ExchangeFundData, error) {
	if !m.healthy {
		return nil, fmt.Errorf("exchange is not healthy")
	}
	return m.fundData, nil
}

// GetPositions 获取持仓数据
func (m *MockExchangeProvider) GetPositions(ctx context.Context) ([]*Position, error) {
	if !m.healthy {
		return nil, fmt.Errorf("exchange is not healthy")
	}
	return m.positions, nil
}

// GetHistoricalReturns 获取历史收益率
func (m *MockExchangeProvider) GetHistoricalReturns(ctx context.Context, days int) ([]float64, error) {
	if !m.healthy {
		return nil, fmt.Errorf("exchange is not healthy")
	}

	// 生成模拟的历史收益率数据
	returns := make([]float64, days)
	for i := 0; i < days; i++ {
		// 简单的随机游走模型
		returns[i] = (float64(i%7) - 3) * 0.01 // -3% to +3%
	}
	return returns, nil
}

// GetHistoricalEquity 获取历史净值
func (m *MockExchangeProvider) GetHistoricalEquity(ctx context.Context, days int) ([]float64, error) {
	if !m.healthy {
		return nil, fmt.Errorf("exchange is not healthy")
	}

	// 生成模拟的历史净值数据
	equity := make([]float64, days)
	baseValue := 100000.0

	for i := 0; i < days; i++ {
		// 简单的增长模型
		growth := 1.0 + float64(i)*0.001 // 每天0.1%增长
		equity[i] = baseValue * growth
	}
	return equity, nil
}

// GetSymbolPrice 获取交易对价格
func (m *MockExchangeProvider) GetSymbolPrice(ctx context.Context, symbol string) (float64, error) {
	if !m.healthy {
		return 0, fmt.Errorf("exchange is not healthy")
	}

	// 返回模拟价格
	prices := map[string]float64{
		"BTCUSDT": 50000.0,
		"ETHUSDT": 3000.0,
		"BNBUSDT": 400.0,
	}

	if price, exists := prices[symbol]; exists {
		return price, nil
	}

	return 0, fmt.Errorf("symbol %s not found", symbol)
}

// GetOrderBookDepth 获取订单簿深度
func (m *MockExchangeProvider) GetOrderBookDepth(ctx context.Context, symbol string) (*OrderBookDepth, error) {
	if !m.healthy {
		return nil, fmt.Errorf("exchange is not healthy")
	}

	price, err := m.GetSymbolPrice(ctx, symbol)
	if err != nil {
		return nil, err
	}

	return &OrderBookDepth{
		Symbol: symbol,
		Bids: []PriceSize{
			{Price: price - 10, Size: 1.0},
			{Price: price - 20, Size: 2.0},
		},
		Asks: []PriceSize{
			{Price: price + 10, Size: 1.0},
			{Price: price + 20, Size: 2.0},
		},
		Timestamp: time.Now(),
	}, nil
}

// GetTradingVolume 获取交易量数据
func (m *MockExchangeProvider) GetTradingVolume(ctx context.Context, symbol string, period string) (float64, error) {
	if !m.healthy {
		return 0, fmt.Errorf("exchange is not healthy")
	}

	// 返回模拟交易量
	return 1000000.0, nil
}

// SetHealthy 设置健康状态（测试用）
func (m *MockExchangeProvider) SetHealthy(healthy bool) {
	m.healthy = healthy
}

// SetFundData 设置资金数据（测试用）
func (m *MockExchangeProvider) SetFundData(data *ExchangeFundData) {
	m.fundData = data
}

// SetPositions 设置持仓数据（测试用）
func (m *MockExchangeProvider) SetPositions(positions []*Position) {
	m.positions = positions
}

// TestMockExchangeProvider 测试mock交易所提供者
func TestMockExchangeProvider(t *testing.T) {
	ctx := context.Background()
	mock := NewMockExchangeProvider()

	// 测试健康状态
	if !mock.IsHealthy() {
		t.Error("Expected mock to be healthy")
	}

	// 测试获取资金数据
	fundData, err := mock.GetFundData(ctx)
	if err != nil {
		t.Errorf("GetFundData failed: %v", err)
	}
	if fundData.TotalBalance != 100000.0 {
		t.Errorf("Expected total balance 100000.0, got %f", fundData.TotalBalance)
	}

	// 测试获取持仓数据
	positions, err := mock.GetPositions(ctx)
	if err != nil {
		t.Errorf("GetPositions failed: %v", err)
	}
	if len(positions) != 1 {
		t.Errorf("Expected 1 position, got %d", len(positions))
	}

	// 测试不健康状态
	mock.SetHealthy(false)
	_, err = mock.GetFundData(ctx)
	if err == nil {
		t.Error("Expected error when exchange is not healthy")
	}
}

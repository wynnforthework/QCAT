package generator

import (
	"context"
	"testing"
	"time"
)

func TestMarketAnalyzer_AnalyzeMarket(t *testing.T) {
	// 创建一个不带依赖项的分析器（将使用模拟数据）
	analyzer := &MarketAnalyzer{}

	ctx := context.Background()
	symbol := "BTCUSDT"
	timeRange := 24 * time.Hour

	// 测试市场分析
	analysis, err := analyzer.AnalyzeMarket(ctx, symbol, timeRange)
	if err != nil {
		t.Fatalf("AnalyzeMarket failed: %v", err)
	}

	// 验证结果
	if analysis == nil {
		t.Fatal("Analysis result is nil")
	}

	if analysis.Symbol != symbol {
		t.Errorf("Expected symbol %s, got %s", symbol, analysis.Symbol)
	}

	if analysis.TimeRange != timeRange {
		t.Errorf("Expected time range %v, got %v", timeRange, analysis.TimeRange)
	}

	// 验证计算的指标是否合理
	if analysis.Volatility < 0 {
		t.Errorf("Volatility should be non-negative, got %f", analysis.Volatility)
	}

	if analysis.Trend < -1 || analysis.Trend > 1 {
		t.Errorf("Trend should be between -1 and 1, got %f", analysis.Trend)
	}

	if analysis.MaxDrawdown < 0 || analysis.MaxDrawdown > 1 {
		t.Errorf("MaxDrawdown should be between 0 and 1, got %f", analysis.MaxDrawdown)
	}

	if analysis.Confidence < 0 || analysis.Confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %f", analysis.Confidence)
	}

	if analysis.MarketRegime == "" {
		t.Error("MarketRegime should not be empty")
	}

	t.Logf("Analysis results:")
	t.Logf("  Symbol: %s", analysis.Symbol)
	t.Logf("  Volatility: %f", analysis.Volatility)
	t.Logf("  Trend: %f", analysis.Trend)
	t.Logf("  SharpeRatio: %f", analysis.SharpeRatio)
	t.Logf("  MaxDrawdown: %f", analysis.MaxDrawdown)
	t.Logf("  MarketCycle: %f days", analysis.MarketCycle)
	t.Logf("  Liquidity: %f", analysis.Liquidity)
	t.Logf("  MarketRegime: %s", analysis.MarketRegime)
	t.Logf("  Confidence: %f", analysis.Confidence)
}

// TestMarketAnalyzer_GenerateMockPriceData 测试已删除，因为不再使用mock数据生成
// 实际的价格数据应该从数据库或交易所API获取

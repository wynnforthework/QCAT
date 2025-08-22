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

func TestMarketAnalyzer_GenerateMockPriceData(t *testing.T) {
	analyzer := &MarketAnalyzer{}
	
	symbol := "BTCUSDT"
	timeRange := 24 * time.Hour
	endTime := time.Now()
	startTime := endTime.Add(-timeRange)
	
	priceData := analyzer.generateMockPriceData(symbol, timeRange, startTime, endTime)
	
	if len(priceData) == 0 {
		t.Fatal("No price data generated")
	}
	
	// 验证价格数据的基本属性
	for i, point := range priceData {
		if point.Close <= 0 {
			t.Errorf("Invalid close price at index %d: %f", i, point.Close)
		}
		
		if point.High < point.Low {
			t.Errorf("High price should be >= low price at index %d: high=%f, low=%f", i, point.High, point.Low)
		}
		
		if point.Close > point.High || point.Close < point.Low {
			t.Errorf("Close price should be between high and low at index %d: close=%f, high=%f, low=%f", 
				i, point.Close, point.High, point.Low)
		}
		
		if point.Volume < 0 {
			t.Errorf("Volume should be non-negative at index %d: %f", i, point.Volume)
		}
	}
	
	t.Logf("Generated %d price data points for %s", len(priceData), symbol)
	t.Logf("First point: Time=%v, Close=%f", priceData[0].Time, priceData[0].Close)
	t.Logf("Last point: Time=%v, Close=%f", priceData[len(priceData)-1].Time, priceData[len(priceData)-1].Close)
}

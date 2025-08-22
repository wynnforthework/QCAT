package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"qcat/internal/strategy/generator"
)

func main() {
	// 创建一个不带依赖项的MarketAnalyzer（将使用模拟数据）
	analyzer := &generator.MarketAnalyzer{}
	
	ctx := context.Background()
	symbol := "BTCUSDT"
	timeRange := 24 * time.Hour
	
	fmt.Printf("正在分析 %s 的市场数据（时间范围：%v）...\n", symbol, timeRange)
	
	// 执行市场分析
	analysis, err := analyzer.AnalyzeMarket(ctx, symbol, timeRange)
	if err != nil {
		log.Fatalf("市场分析失败: %v", err)
	}
	
	// 打印分析结果
	fmt.Printf("\n=== %s 市场分析结果 ===\n", analysis.Symbol)
	fmt.Printf("时间范围: %v\n", analysis.TimeRange)
	fmt.Printf("波动率: %.4f (%.2f%%)\n", analysis.Volatility, analysis.Volatility*100)
	fmt.Printf("趋势强度: %.4f\n", analysis.Trend)
	fmt.Printf("夏普比率: %.4f\n", analysis.SharpeRatio)
	fmt.Printf("最大回撤: %.4f (%.2f%%)\n", analysis.MaxDrawdown, analysis.MaxDrawdown*100)
	fmt.Printf("市场周期: %.1f 天\n", analysis.MarketCycle)
	fmt.Printf("流动性指标: %.4f\n", analysis.Liquidity)
	fmt.Printf("市场状态: %s\n", analysis.MarketRegime)
	fmt.Printf("分析置信度: %.4f (%.1f%%)\n", analysis.Confidence, analysis.Confidence*100)
	
	// 打印技术指标
	fmt.Printf("\n=== 技术指标 ===\n")
	fmt.Printf("RSI: %.2f\n", analysis.TechnicalIndicators.RSI)
	fmt.Printf("MACD: %.4f\n", analysis.TechnicalIndicators.MACD)
	fmt.Printf("布林带上轨: %.2f\n", analysis.TechnicalIndicators.BollingerBands.Upper)
	fmt.Printf("布林带中轨: %.2f\n", analysis.TechnicalIndicators.BollingerBands.Middle)
	fmt.Printf("布林带下轨: %.2f\n", analysis.TechnicalIndicators.BollingerBands.Lower)
	fmt.Printf("布林带宽度: %.4f\n", analysis.TechnicalIndicators.BollingerBands.Width)
	fmt.Printf("SMA20: %.2f\n", analysis.TechnicalIndicators.SMA20)
	fmt.Printf("SMA50: %.2f\n", analysis.TechnicalIndicators.SMA50)
	fmt.Printf("EMA12: %.2f\n", analysis.TechnicalIndicators.EMA12)
	fmt.Printf("EMA26: %.2f\n", analysis.TechnicalIndicators.EMA26)
	fmt.Printf("ATR: %.2f\n", analysis.TechnicalIndicators.ATR)
	fmt.Printf("成交量: %.0f\n", analysis.TechnicalIndicators.Volume)
	fmt.Printf("成交量MA: %.0f\n", analysis.TechnicalIndicators.VolumeMA)
	
	// 打印相关性
	fmt.Printf("\n=== 资产相关性 ===\n")
	for asset, correlation := range analysis.Correlation {
		fmt.Printf("%s: %.3f\n", asset, correlation)
	}
	
	// 演示策略表现分析
	fmt.Printf("\n正在分析策略历史表现...\n")
	performance, err := analyzer.AnalyzePerformance(ctx, "test_strategy", timeRange)
	if err != nil {
		log.Printf("策略表现分析失败: %v", err)
	} else {
		fmt.Printf("\n=== 策略表现分析 ===\n")
		fmt.Printf("策略ID: %s\n", performance.StrategyID)
		fmt.Printf("总收益: %.4f (%.2f%%)\n", performance.TotalReturn, performance.TotalReturn*100)
		fmt.Printf("夏普比率: %.4f\n", performance.SharpeRatio)
		fmt.Printf("最大回撤: %.4f (%.2f%%)\n", performance.MaxDrawdown, performance.MaxDrawdown*100)
		fmt.Printf("胜率: %.4f (%.1f%%)\n", performance.WinRate, performance.WinRate*100)
		fmt.Printf("盈利因子: %.4f\n", performance.ProfitFactor)
		fmt.Printf("总交易次数: %d\n", performance.TotalTrades)
		fmt.Printf("平均交易收益: %.4f (%.3f%%)\n", performance.AvgTrade, performance.AvgTrade*100)
		fmt.Printf("波动率: %.4f (%.2f%%)\n", performance.Volatility, performance.Volatility*100)
		fmt.Printf("置信度: %.4f (%.1f%%)\n", performance.Confidence, performance.Confidence*100)
	}
	
	// 演示市场状态检测
	fmt.Printf("\n=== 市场状态检测 ===\n")
	regime := analyzer.DetectMarketRegime(analysis)
	fmt.Printf("检测到的市场状态: %s\n", regime)
	
	// 演示最优时间框架计算
	optimalTimeframe := analyzer.CalculateOptimalTimeframe(analysis)
	fmt.Printf("建议的最优时间框架: %v\n", optimalTimeframe)
	
	fmt.Printf("\n分析完成！\n")
}

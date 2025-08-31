package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/trading/dryrun"
)

// DemoMarketDataProvider 演示用市场数据提供者
type DemoMarketDataProvider struct {
	prices map[string]float64
}

func NewDemoMarketDataProvider() *DemoMarketDataProvider {
	return &DemoMarketDataProvider{
		prices: map[string]float64{
			"BTCUSDT": 50000.0,
			"ETHUSDT": 3000.0,
		},
	}
}

func (d *DemoMarketDataProvider) GetPrice(symbol string) (float64, error) {
	if price, exists := d.prices[symbol]; exists {
		return price, nil
	}
	return 0, fmt.Errorf("未找到交易对: %s", symbol)
}

// DemoEventHandler 演示用事件处理器
type DemoEventHandler struct{}

func (deh *DemoEventHandler) OnOrderPlaced(order *dryrun.SimulatedOrder) {
	log.Printf("📝 订单下达事件: %s %s %s %.4f @ %.4f", 
		order.Type, order.Side, order.Symbol, order.Quantity, order.Price)
}

func (deh *DemoEventHandler) OnOrderFilled(order *dryrun.SimulatedOrder, trade *dryrun.SimulatedTrade) {
	log.Printf("💰 订单成交事件: %s %.4f @ %.4f (滑点: %.4f)", 
		order.Symbol, trade.Quantity, trade.Price, trade.Slippage)
}

func (deh *DemoEventHandler) OnOrderCanceled(order *dryrun.SimulatedOrder) {
	log.Printf("❌ 订单撤销事件: %s", order.ID)
}

func (deh *DemoEventHandler) OnPositionOpened(position *dryrun.SimulatedPosition) {
	log.Printf("📈 持仓开启事件: %s %s %.4f @ %.4f", 
		position.Symbol, position.Side, position.Size, position.EntryPrice)
}

func (deh *DemoEventHandler) OnPositionClosed(position *dryrun.SimulatedPosition) {
	log.Printf("📉 持仓关闭事件: %s PnL: %.4f", 
		position.Symbol, position.RealizedPnL)
}

func (deh *DemoEventHandler) OnPnLUpdate(stats *dryrun.PerformanceStats) {
	log.Printf("📊 盈亏更新事件: 总交易 %d, 净盈亏 %.4f USDT", 
		stats.TotalTrades, stats.NetPnL)
}

func main() {
	log.Println("🎯 Dry-run模拟交易系统演示")
	log.Println("========================================")

	// 创建市场数据提供者
	marketDataProvider := NewDemoMarketDataProvider()

	// 创建自定义配置
	config := dryrun.DefaultSimulatorConfig()
	config.Name = "演示交易模拟器"
	config.Description = "用于演示Dry-run交易功能的模拟器"
	config.InitialBalance = map[string]float64{
		"USDT": 50000.0, // 5万USDT初始资金
		"BTC":  0.0,
		"ETH":  0.0,
	}
	config.TradingFee = 0.001 // 0.1%手续费
	config.ReportingInterval = 5 * time.Second // 5秒报告间隔

	// 创建交易模拟器
	simulator, err := dryrun.NewTradingSimulator(config, marketDataProvider)
	if err != nil {
		log.Fatalf("创建交易模拟器失败: %v", err)
	}

	// 添加事件处理器
	eventHandler := &DemoEventHandler{}
	simulator.AddEventHandler(eventHandler)

	// 启动模拟器
	log.Println("🚀 启动交易模拟器...")
	err = simulator.Start()
	if err != nil {
		log.Fatalf("启动模拟器失败: %v", err)
	}

	// 等待模拟器完全启动
	time.Sleep(100 * time.Millisecond)

	ctx := context.Background()

	// 演示1: 查看初始账户余额
	log.Println("\n📋 初始账户余额:")
	balances, err := simulator.GetAccountBalance(ctx)
	if err != nil {
		log.Printf("获取账户余额失败: %v", err)
	} else {
		for asset, balance := range balances {
			log.Printf("   %s: 总额 %.4f, 可用 %.4f, 锁定 %.4f", 
				asset, balance.Total, balance.Available, balance.Locked)
		}
	}

	// 演示2: 下市价买单
	log.Println("\n💰 下市价买单 (0.1 BTC)...")
	buyOrderReq := &exchange.OrderRequest{
		Symbol:   "BTCUSDT",
		Side:     "BUY",
		Type:     "MARKET",
		Quantity: 0.1,
	}

	buyResp, err := simulator.PlaceOrder(ctx, buyOrderReq)
	if err != nil {
		log.Printf("下买单失败: %v", err)
	} else if !buyResp.Success {
		log.Printf("下买单失败: %s", buyResp.Error)
	} else {
		log.Printf("✅ 买单成功: 订单ID %s", buyResp.OrderID)
	}

	// 等待订单处理
	time.Sleep(200 * time.Millisecond)

	// 演示3: 下限价卖单
	log.Println("\n💸 下限价卖单 (0.05 BTC @ 52000 USDT)...")
	sellOrderReq := &exchange.OrderRequest{
		Symbol:      "BTCUSDT",
		Side:        "SELL",
		Type:        "LIMIT",
		Quantity:    0.05,
		Price:       52000.0,
		TimeInForce: "GTC",
	}

	sellResp, err := simulator.PlaceOrder(ctx, sellOrderReq)
	if err != nil {
		log.Printf("下卖单失败: %v", err)
	} else if !sellResp.Success {
		log.Printf("下卖单失败: %s", sellResp.Error)
	} else {
		log.Printf("✅ 卖单成功: 订单ID %s", sellResp.OrderID)
	}

	// 等待订单处理
	time.Sleep(200 * time.Millisecond)

	// 演示4: 查看开放订单
	log.Println("\n📋 当前开放订单:")
	openOrders, err := simulator.GetOpenOrders(ctx, "")
	if err != nil {
		log.Printf("获取开放订单失败: %v", err)
	} else {
		if len(openOrders) == 0 {
			log.Println("   无开放订单")
		} else {
			for _, order := range openOrders {
				log.Printf("   订单 %s: %s %s %s %.4f @ %.4f (状态: %s)", 
					order.ID, order.Type, order.Side, order.Symbol, 
					order.Quantity, order.Price, order.Status)
			}
		}
	}

	// 演示5: 撤销限价单
	if sellResp != nil && sellResp.Success {
		log.Printf("\n❌ 撤销限价卖单: %s", sellResp.OrderID)
		cancelResp, err := simulator.CancelOrder(ctx, sellResp.OrderID)
		if err != nil {
			log.Printf("撤单失败: %v", err)
		} else if !cancelResp.Success {
			log.Printf("撤单失败: %s", cancelResp.Error)
		} else {
			log.Printf("✅ 撤单成功")
		}
	}

	// 等待处理
	time.Sleep(200 * time.Millisecond)

	// 演示6: 查看最终账户余额
	log.Println("\n📋 最终账户余额:")
	finalBalances, err := simulator.GetAccountBalance(ctx)
	if err != nil {
		log.Printf("获取账户余额失败: %v", err)
	} else {
		for asset, balance := range finalBalances {
			log.Printf("   %s: 总额 %.4f, 可用 %.4f, 锁定 %.4f", 
				asset, balance.Total, balance.Available, balance.Locked)
		}
	}

	// 演示7: 查看持仓信息
	log.Println("\n📋 当前持仓:")
	positions, err := simulator.GetPositions(ctx)
	if err != nil {
		log.Printf("获取持仓失败: %v", err)
	} else {
		if len(positions) == 0 {
			log.Println("   无持仓")
		} else {
			for _, pos := range positions {
				log.Printf("   持仓 %s: %s %.4f @ %.4f (未实现盈亏: %.4f)", 
					pos.Symbol, pos.Side, pos.Size, pos.EntryPrice, pos.UnrealizedPnL)
			}
		}
	}

	// 演示8: 查看性能统计
	log.Println("\n📊 性能统计:")
	stats := simulator.GetPerformanceStats()
	log.Printf("   运行时间: %v", stats.Duration)
	log.Printf("   总交易数: %d", stats.TotalTrades)
	log.Printf("   胜率: %.2f%%", stats.WinRate*100)
	log.Printf("   总盈亏: %.4f USDT", stats.TotalPnL)
	log.Printf("   总手续费: %.4f USDT", stats.TotalFees)
	log.Printf("   净盈亏: %.4f USDT", stats.NetPnL)

	// 让模拟器运行一段时间以生成报告
	log.Println("\n⏰ 让模拟器运行10秒以观察报告生成...")
	time.Sleep(10 * time.Second)

	// 停止模拟器
	log.Println("\n🛑 停止交易模拟器...")
	simulator.Stop()

	log.Println("\n🎉 Dry-run模拟交易演示完成!")
	log.Println("========================================")
}

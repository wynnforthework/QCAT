# 开发修复清单（来自 report.md）

## ./internal\analysis\factors\factor_discovery_engine.go
- [x] `./internal\analysis\factors\factor_discovery_engine.go:4981` — **TODO** — // TODO: 实现基于性能的轮换
- [x] `./internal\analysis\factors\factor_discovery_engine.go:4985` — **TODO** — // TODO: 实现基于相关性的轮换
- [x] `./internal\analysis\factors\factor_discovery_engine.go:4989` — **TODO** — // TODO: 实现基于市场状态的轮换
- [x] `./internal\analysis\factors\factor_discovery_engine.go:4993` — **TODO** — // TODO: 计算因子表现

## ./internal\api\server.go
- [x] `./internal\api\server.go:440` — **mock** — log.Printf("Warning: Binance API credentials not configured, using mock data")
- [x] `./internal\api\server.go:446` — **mock** — // Create a mock exchange client for automation system if needed

## ./internal\api\websocket.go
- [x] `./internal\api\websocket.go:228` — **mock** — // Mock market data
- [x] `./internal\api\websocket.go:276` — **mock** — // Mock strategy status
- [x] `./internal\api\websocket.go:325` — **mock** — // Mock alerts

## ./internal\automation\executor\executors.go
- [x] `./internal\automation\executor\executors.go:1894` — **TODO** — // TODO: 实现策略淘汰逻辑
- [x] `./internal\automation\executor\executors.go:1901` — **TODO** — // TODO: 实现新策略引入逻辑

## ./internal\automation\scheduler\strategy_scheduler.go
- [x] `./internal\automation\scheduler\strategy_scheduler.go:3455` — **mock** — log.Printf("Mock: Adjusted %s - SL: %.4f->%.4f, TP: %.4f->%.4f",
- [x] `./internal\automation\scheduler\strategy_scheduler.go:3461` — **mock** — log.Printf("Mock: Completed automatic adjustment for %d positions", adjustmentCount)
- [x] `./internal\automation\scheduler\strategy_scheduler.go:4404` — **mock** — log.Printf("Exchange client not fully implemented, using mock data")
- [x] `./internal\automation\scheduler\strategy_scheduler.go:4405` — **mock** — return ss.getMockMarketData(), nil
- [x] `./internal\automation\scheduler\strategy_scheduler.go:4408` — **mock** — // getMockMarketData 获取模拟市场数据
- [x] `./internal\automation\scheduler\strategy_scheduler.go:4409` — **mock** — func (ss *StrategyScheduler) getMockMarketData() map[string]*MarketData {
- [x] `./internal\automation\scheduler\strategy_scheduler.go:4410` — **mock** — mockData := make(map[string]*MarketData)
- [x] `./internal\automation\scheduler\strategy_scheduler.go:4414` — **mock** — mockData[symbol] = ss.createMockMarketData(symbol)
- [x] `./internal\automation\scheduler\strategy_scheduler.go:4417` — **mock** — return mockData
- [x] `./internal\automation\scheduler\strategy_scheduler.go:4420` — **mock** — // createMockMarketData 创建单个交易对的模拟市场数据
- [x] `./internal\automation\scheduler\strategy_scheduler.go:4421` — **mock** — func (ss *StrategyScheduler) createMockMarketData(symbol string) *MarketData {
- [x] `./internal\automation\scheduler\strategy_scheduler.go:5090` — **TODO** — // TODO: 实现自动参数应用机制

## ./internal\automation\scheduler\sub_schedulers.go
- [x] `./internal\automation\scheduler\sub_schedulers.go:1748` — **mock** — log.Printf("Failed to query exchange balances from database: %v, using mock data", err)
- [x] `./internal\automation\scheduler\sub_schedulers.go:1749` — **mock** — return rs.getMockExchangeFundDistribution(), nil
- [x] `./internal\automation\scheduler\sub_schedulers.go:1766` — **mock** — log.Printf("No exchange balance data available, using mock data")
- [x] `./internal\automation\scheduler\sub_schedulers.go:1767` — **mock** — return rs.getMockExchangeFundDistribution(), nil
- [x] `./internal\automation\scheduler\sub_schedulers.go:1773` — **mock** — // getMockExchangeFundDistribution 获取模拟的交易所资金分布
- [x] `./internal\automation\scheduler\sub_schedulers.go:1774` — **mock** — func (rs *RiskScheduler) getMockExchangeFundDistribution() map[string]float64 {
- [x] `./internal\automation\scheduler\sub_schedulers.go:1796` — **mock** — log.Printf("Failed to query wallet balances from database: %v, using mock data", err)
- [x] `./internal\automation\scheduler\sub_schedulers.go:1797` — **mock** — return rs.getMockWalletFundDistribution(), nil
- [x] `./internal\automation\scheduler\sub_schedulers.go:1814` — **mock** — log.Printf("No wallet balance data available, using mock data")
- [x] `./internal\automation\scheduler\sub_schedulers.go:1815` — **mock** — return rs.getMockWalletFundDistribution(), nil
- [x] `./internal\automation\scheduler\sub_schedulers.go:1821` — **mock** — // getMockWalletFundDistribution 获取模拟的钱包资金分布
- [x] `./internal\automation\scheduler\sub_schedulers.go:1822` — **mock** — func (rs *RiskScheduler) getMockWalletFundDistribution() map[string]float64 {

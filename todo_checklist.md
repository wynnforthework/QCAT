# 开发修复清单（来自 report.md）

## ./internal\analysis\backtesting\auto_backtesting_engine.go
- [ ] `./internal\analysis\backtesting\auto_backtesting_engine.go:1267` — **TODO** — // TODO: 实现信号执行逻辑
- [ ] `./internal\analysis\backtesting\auto_backtesting_engine.go:1287` — **TODO** — // TODO: 实现组合价值更新逻辑
- [ ] `./internal\analysis\backtesting\auto_backtesting_engine.go:1457` — **TODO** — // TODO: 实现样本外测试
- [ ] `./internal\analysis\backtesting\auto_backtesting_engine.go:1471` — **TODO** — // TODO: 实现稳定性测试
- [ ] `./internal\analysis\backtesting\auto_backtesting_engine.go:1484` — **TODO** — // TODO: 实现鲁棒性测试
- [ ] `./internal\analysis\backtesting\auto_backtesting_engine.go:1497` — **TODO** — // TODO: 实现报告生成
- [ ] `./internal\analysis\backtesting\auto_backtesting_engine.go:1560` — **TODO** — // TODO: 实现策略性能分析
- [ ] `./internal\analysis\backtesting\auto_backtesting_engine.go:1569` — **TODO** — // TODO: 实现验证检查

## ./internal\analysis\factors\factor_discovery_engine.go
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:645` — **TODO** — // TODO: 从配置或数据库加载基础因子
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:812` — **TODO** — // TODO: 实现随机搜索算法
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:818` — **TODO** — // TODO: 实现系统化搜索算法
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:986` — **TODO** — // TODO: 实现随机因子生成
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:1042` — **TODO** — // TODO: 实现收敛检查逻辑
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:1093` — **TODO** — // TODO: 检查因子是否新颖（不与现有因子重复）
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:1149` — **TODO** — // TODO: 实现实际的IC计算
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:1167` — **TODO** — // TODO: 计算因子多样性
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:1172` — **TODO** — // TODO: 计算种群多样性
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:1226` — **TODO** — // TODO: 实现因子交叉操作
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:1235` — **TODO** — // TODO: 实现因子变异操作
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:1248` — **TODO** — // TODO: 计算因子相似度
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:1262` — **TODO** — // TODO: 实现IC计算
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:1276` — **TODO** — // TODO: 实现滚动IC计算
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:1281` — **TODO** — // TODO: 实现IC衰减计算
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:1286` — **TODO** — // TODO: 实现分组回测
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:1291` — **TODO** — // TODO: 实现因子风险分析
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:1301` — **TODO** — // TODO: 实现因子稳定性分析
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:1380` — **TODO** — // TODO: 实现基于性能的轮换
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:1384` — **TODO** — // TODO: 实现基于相关性的轮换
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:1388` — **TODO** — // TODO: 实现基于市场状态的轮换
- [ ] `./internal\analysis\factors\factor_discovery_engine.go:1392` — **TODO** — // TODO: 计算因子表现

## ./internal\api\api_test.go
- [ ] `./internal\api\api_test.go:78` — **mock** — mockData := testutils.NewMockData()
- [ ] `./internal\api\api_test.go:81` — **mock** — strategy := mockData.GenerateStrategy()

## ./internal\api\handlers.go
- [ ] `./internal\api\handlers.go:3373` — **TODO** — // TODO: 实现真实的策略接入流程
- [ ] `./internal\api\handlers.go:3484` — **TODO** — // TODO: 实现真实的策略接入状态查询

## ./internal\api\server.go
- [ ] `./internal\api\server.go:440` — **mock** — log.Printf("Warning: Binance API credentials not configured, using mock data")
- [ ] `./internal\api\server.go:446` — **mock** — // Create a mock exchange client for automation system if needed
- [ ] `./internal\api\server.go:576` — **TODO** — // TODO: TEMPORARY - Add audit logs as public route for testing
- [ ] `./internal\api\server.go:583` — **TODO** — // TODO: TEMPORARY - Add strategy routes as public for frontend testing

## ./internal\api\settings_handler.go
- [ ] `./internal\api\settings_handler.go:125` — **TODO** — // TODO 集成到实际的交易系统中
- [ ] `./internal\api\settings_handler.go:129` — **TODO** — // TODO: 集成到实际的交易执行器

## ./internal\api\websocket.go
- [ ] `./internal\api\websocket.go:228` — **mock** — // Mock market data
- [ ] `./internal\api\websocket.go:276` — **mock** — // Mock strategy status
- [ ] `./internal\api\websocket.go:325` — **mock** — // Mock alerts

## ./internal\automation\executor\executors.go
- [ ] `./internal\automation\executor\executors.go:638` — **TODO** — // TODO: 实现暂停新开仓逻辑
- [ ] `./internal\automation\executor\executors.go:711` — **TODO** — // TODO: 实现收紧止损逻辑
- [ ] `./internal\automation\executor\executors.go:792` — **TODO** — // 简单的余额检查（TODO 添加更复杂的逻辑）
- [ ] `./internal\automation\executor\executors.go:1083` — **TODO** — // TODO: 实现参数应用逻辑
- [ ] `./internal\automation\executor\executors.go:1090` — **TODO** — // TODO: 实现策略淘汰逻辑
- [ ] `./internal\automation\executor\executors.go:1097` — **TODO** — // TODO: 实现新策略引入逻辑
- [ ] `./internal\automation\executor\executors.go:1104` — **TODO** — // TODO: 实现策略优化逻辑
- [ ] `./internal\automation\executor\executors.go:1150` — **TODO** — // TODO: 实现数据清洗逻辑
- [ ] `./internal\automation\executor\executors.go:1157` — **TODO** — // TODO: 实现因子更新逻辑
- [ ] `./internal\automation\executor\executors.go:1164` — **TODO** — // TODO: 实现回测逻辑
- [ ] `./internal\automation\executor\executors.go:1171` — **TODO** — // TODO: 实现模式识别逻辑
- [ ] `./internal\automation\executor\executors.go:1219` — **TODO** — // TODO: 实现健康检查逻辑
- [ ] `./internal\automation\executor\executors.go:1226` — **TODO** — // TODO: 实现安全监控逻辑
- [ ] `./internal\automation\executor\executors.go:1233` — **TODO** — // TODO: 实现交易所故障切换逻辑
- [ ] `./internal\automation\executor\executors.go:1240` — **TODO** — // TODO: 实现审计日志处理逻辑

## ./internal\automation\risk\intelligent_controller.go
- [ ] `./internal\automation\risk\intelligent_controller.go:1040` — **TODO** — // TODO 将报告保存到数据库或发送给相关人员

## ./internal\automation\scheduler\strategy_scheduler.go
- [ ] `./internal\automation\scheduler\strategy_scheduler.go:455` — **TODO** — // TODO: 实现参数更新逻辑
- [ ] `./internal\automation\scheduler\strategy_scheduler.go:828` — **TODO** — // TODO 根据策略类型返回不同的默认参数
- [ ] `./internal\automation\scheduler\strategy_scheduler.go:2873` — **mock** — log.Printf("Mock: Adjusted %s - SL: %.4f->%.4f, TP: %.4f->%.4f",
- [ ] `./internal\automation\scheduler\strategy_scheduler.go:2879` — **mock** — log.Printf("Mock: Completed automatic adjustment for %d positions", adjustmentCount)
- [ ] `./internal\automation\scheduler\strategy_scheduler.go:3822` — **mock** — log.Printf("Exchange client not fully implemented, using mock data")
- [ ] `./internal\automation\scheduler\strategy_scheduler.go:3823` — **mock** — return ss.getMockMarketData(), nil
- [ ] `./internal\automation\scheduler\strategy_scheduler.go:3826` — **mock** — // getMockMarketData 获取模拟市场数据
- [ ] `./internal\automation\scheduler\strategy_scheduler.go:3827` — **mock** — func (ss *StrategyScheduler) getMockMarketData() map[string]*MarketData {
- [ ] `./internal\automation\scheduler\strategy_scheduler.go:3828` — **mock** — mockData := make(map[string]*MarketData)
- [ ] `./internal\automation\scheduler\strategy_scheduler.go:3832` — **mock** — mockData[symbol] = ss.createMockMarketData(symbol)
- [ ] `./internal\automation\scheduler\strategy_scheduler.go:3835` — **mock** — return mockData
- [ ] `./internal\automation\scheduler\strategy_scheduler.go:3838` — **mock** — // createMockMarketData 创建单个交易对的模拟市场数据
- [ ] `./internal\automation\scheduler\strategy_scheduler.go:3839` — **mock** — func (ss *StrategyScheduler) createMockMarketData(symbol string) *MarketData {
- [ ] `./internal\automation\scheduler\strategy_scheduler.go:4508` — **TODO** — // TODO: 实现自动参数应用机制

## ./internal\automation\scheduler\sub_schedulers.go
- [ ] `./internal\automation\scheduler\sub_schedulers.go:1745` — **mock** — log.Printf("Failed to query exchange balances from database: %v, using mock data", err)
- [ ] `./internal\automation\scheduler\sub_schedulers.go:1746` — **mock** — return rs.getMockExchangeFundDistribution(), nil
- [ ] `./internal\automation\scheduler\sub_schedulers.go:1763` — **mock** — log.Printf("No exchange balance data available, using mock data")
- [ ] `./internal\automation\scheduler\sub_schedulers.go:1764` — **mock** — return rs.getMockExchangeFundDistribution(), nil
- [ ] `./internal\automation\scheduler\sub_schedulers.go:1770` — **mock** — // getMockExchangeFundDistribution 获取模拟的交易所资金分布
- [ ] `./internal\automation\scheduler\sub_schedulers.go:1771` — **mock** — func (rs *RiskScheduler) getMockExchangeFundDistribution() map[string]float64 {
- [ ] `./internal\automation\scheduler\sub_schedulers.go:1793` — **mock** — log.Printf("Failed to query wallet balances from database: %v, using mock data", err)
- [ ] `./internal\automation\scheduler\sub_schedulers.go:1794` — **mock** — return rs.getMockWalletFundDistribution(), nil
- [ ] `./internal\automation\scheduler\sub_schedulers.go:1811` — **mock** — log.Printf("No wallet balance data available, using mock data")
- [ ] `./internal\automation\scheduler\sub_schedulers.go:1812` — **mock** — return rs.getMockWalletFundDistribution(), nil
- [ ] `./internal\automation\scheduler\sub_schedulers.go:1818` — **mock** — // getMockWalletFundDistribution 获取模拟的钱包资金分布
- [ ] `./internal\automation\scheduler\sub_schedulers.go:1819` — **mock** — func (rs *RiskScheduler) getMockWalletFundDistribution() map[string]float64 {
- [ ] `./internal\automation\scheduler\sub_schedulers.go:3675` — **TODO** — // TODO 实现对冲调整的调度逻辑
- [ ] `./internal\automation\scheduler\sub_schedulers.go:3977` — **TODO** — // TODO 集成实际的告警系统
- [ ] `./internal\automation\scheduler\sub_schedulers.go:3983` — **TODO** — // TODO 集成实际的告警系统
- [ ] `./internal\automation\scheduler\sub_schedulers.go:3989` — **TODO** — // TODO 集成实际的告警系统

## ./internal\automation\scheduler\position\layered_position_test.go
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:14` — **mock** — // Mock implementations for testing
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:15` — **mock** — type mockExchangeClient struct{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:17` — **mock** — func (m *mockExchangeClient) GetCurrentPrice(ctx context.Context, symbol string) (float64, error) {
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:21` — **mock** — func (m *mockExchangeClient) GetHistoricalPrices(ctx context.Context, symbol string, period time.Duration) ([]float64, error) {
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:22` — **mock** — // Return mock historical prices
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:28` — **mock** — func (m *mockExchangeClient) PlaceOrder(ctx context.Context, order interface{}) error { return nil }
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:29` — **mock** — func (m *mockExchangeClient) CancelOrder(ctx context.Context, orderID string) error { return nil }
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:30` — **mock** — func (m *mockExchangeClient) CancelAllOrders(ctx context.Context, symbol string) error { return nil }
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:31` — **mock** — func (m *mockExchangeClient) GetOrderStatus(ctx context.Context, orderID string) (interface{}, error) { return nil, nil }
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:32` — **mock** — func (m *mockExchangeClient) GetBalance(ctx context.Context, currency string) (float64, error) { return 1000.0, nil }
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:33` — **mock** — func (m *mockExchangeClient) GetPositions(ctx context.Context) ([]interface{}, error) { return nil, nil }
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:35` — **mock** — type mockDB struct{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:37` — **mock** — func (m *mockDB) Query(query string, args ...interface{}) error { return nil }
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:38` — **mock** — func (m *mockDB) QueryRow(query string, args ...interface{}) interface{} { return nil }
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:39` — **mock** — func (m *mockDB) Exec(query string, args ...interface{}) error { return nil }
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:40` — **mock** — func (m *mockDB) Begin() (interface{}, error) { return nil, nil }
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:41` — **mock** — func (m *mockDB) Close() error { return nil }
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:43` — **mock** — type mockLogger struct{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:45` — **mock** — func (m *mockLogger) Info(msg string, fields ...interface{}) {}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:46` — **mock** — func (m *mockLogger) Warn(msg string, fields ...interface{}) {}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:47` — **mock** — func (m *mockLogger) Error(msg string, fields ...interface{}) {}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:48` — **mock** — func (m *mockLogger) Debug(msg string, fields ...interface{}) {}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:49` — **mock** — func (m *mockLogger) Fatal(msg string, fields ...interface{}) {}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:50` — **mock** — func (m *mockLogger) Panic(msg string, fields ...interface{}) {}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:52` — **mock** — type mockConfig struct {
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:56` — **mock** — func newMockConfig() *mockConfig {
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:57` — **mock** — return &mockConfig{
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:77` — **mock** — func (m *mockConfig) Get(key string) interface{} {
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:81` — **mock** — func (m *mockConfig) GetString(key string) string {
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:88` — **mock** — func (m *mockConfig) GetInt(key string) int {
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:95` — **mock** — func (m *mockConfig) GetFloat64(key string) float64 {
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:102` — **mock** — func (m *mockConfig) GetBool(key string) bool {
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:109` — **mock** — func (m *mockConfig) GetDuration(key string) time.Duration {
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:116` — **mock** — func (m *mockConfig) Set(key string, value interface{}) error {
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:121` — **mock** — func (m *mockConfig) Reload() error {
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:127` — **mock** — db := &mockDB{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:128` — **mock** — exchangeClient := &mockExchangeClient{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:129` — **mock** — logger := &mockLogger{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:130` — **mock** — config := newMockConfig()
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:206` — **mock** — db := &mockDB{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:207` — **mock** — exchangeClient := &mockExchangeClient{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:208` — **mock** — logger := &mockLogger{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:209` — **mock** — config := newMockConfig()
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:317` — **mock** — db := &mockDB{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:318` — **mock** — exchangeClient := &mockExchangeClient{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:319` — **mock** — logger := &mockLogger{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:320` — **mock** — config := newMockConfig()
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:328` — **mock** — // Create mock volatility analysis
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:368` — **mock** — currentPrice := 100.0 // Mock current price
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:389` — **mock** — db := &mockDB{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:390` — **mock** — exchangeClient := &mockExchangeClient{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:391` — **mock** — logger := &mockLogger{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:392` — **mock** — config := newMockConfig()
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:487` — **mock** — db := &mockDB{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:488` — **mock** — exchangeClient := &mockExchangeClient{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:489` — **mock** — logger := &mockLogger{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:490` — **mock** — config := newMockConfig()
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:588` — **mock** — db := &mockDB{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:589` — **mock** — exchangeClient := &mockExchangeClient{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:590` — **mock** — logger := &mockLogger{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:591` — **mock** — config := newMockConfig()
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:706` — **mock** — db := &mockDB{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:707` — **mock** — exchangeClient := &mockExchangeClient{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:708` — **mock** — logger := &mockLogger{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:709` — **mock** — config := newMockConfig()
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:860` — **mock** — db := &mockDB{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:861` — **mock** — exchangeClient := &mockExchangeClient{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:862` — **mock** — logger := &mockLogger{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:863` — **mock** — config := newMockConfig()
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:881` — **mock** — db := &mockDB{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:882` — **mock** — exchangeClient := &mockExchangeClient{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:883` — **mock** — logger := &mockLogger{}
- [ ] `./internal\automation\scheduler\position\layered_position_test.go:884` — **mock** — config := newMockConfig()

## ./internal\automation\scheduler\risk\abnormal_market_simple_test.go
- [ ] `./internal\automation\scheduler\risk\abnormal_market_simple_test.go:276` — **mock** — // Create a mock detector to test the helper method

## ./internal\automation\scheduler\risk\risk_controller_test.go
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:26` — **mock** — mockRC := NewTestRiskController()
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:33` — **mock** — originalPositions := make([]shared.Position, len(mockRC.testDB.positions))
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:34` — **mock** — copy(originalPositions, mockRC.testDB.positions)
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:36` — **mock** — // Call the mocked version by creating a custom implementation
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:37` — **mock** — action, err := mockRC.triggerPositionReductionMocked(ctx, marginStatus, 0.3)
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:47` — **mock** — history := mockRC.GetActionHistory()
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:53` — **mock** — mockRC := NewTestRiskController()
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:57` — **mock** — action, err := mockRC.triggerEmergencyStopMocked(ctx, reason)
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:63` — **mock** — assert.True(t, mockRC.IsEmergencyMode())
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:67` — **mock** — for _, pos := range mockRC.testDB.positions {
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:72` — **mock** — history := mockRC.GetActionHistory()
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:78` — **mock** — mockRC := NewTestRiskController()
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:82` — **mock** — action, err := mockRC.triggerLeverageReductionMocked(ctx, targetLeverage)
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:92` — **mock** — mockRC := NewTestRiskController()
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:135` — **mock** — newSize, err := mockRC.calculateReducedPositionSize(tt.position, tt.targetLeverage)
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:148` — **mock** — mockRC := NewTestRiskController()
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:175` — **mock** — reductions, err := mockRC.selectPositionsForReduction(ctx, positions, reductionPercent)
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:195` — **mock** — mockRC := NewTestRiskController()
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:198` — **mock** — assert.False(t, mockRC.IsEmergencyMode())
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:203` — **mock** — _, err := mockRC.triggerEmergencyStopMocked(ctx, "Test emergency")
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:207` — **mock** — assert.True(t, mockRC.IsEmergencyMode())
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:210` — **mock** — mockRC.ClearEmergencyMode()
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:211` — **mock** — assert.False(t, mockRC.IsEmergencyMode())
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:215` — **mock** — mockRC := NewTestRiskController()
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:218` — **mock** — err := mockRC.Start()
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:222` — **mock** — err = mockRC.Stop()
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:251` — **mock** — mockRC := NewTestRiskController()
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:268` — **mock** — _, err := mockRC.selectPositionsForReduction(ctx, positions, 0.3)

## ./internal\automation\scheduler\risk\risk_monitor_helpers.go
- [ ] `./internal\automation\scheduler\risk\risk_monitor_helpers.go:49` — **mock** — // If no data in database, return mock data for testing
- [ ] `./internal\automation\scheduler\risk\risk_monitor_helpers.go:51` — **mock** — log.Printf("No historical data found for %s, generating mock data", symbol)
- [ ] `./internal\automation\scheduler\risk\risk_monitor_helpers.go:52` — **mock** — return rm.generateMockPrices(100.0, days*24, 0.02), nil
- [ ] `./internal\automation\scheduler\risk\risk_monitor_helpers.go:417` — **mock** — // If no data in database, generate mock data for testing
- [ ] `./internal\automation\scheduler\risk\risk_monitor_helpers.go:419` — **mock** — log.Printf("No market data found, generating mock data for testing")
- [ ] `./internal\automation\scheduler\risk\risk_monitor_helpers.go:420` — **mock** — return rm.generateMockMarketData(), nil
- [ ] `./internal\automation\scheduler\risk\risk_monitor_helpers.go:648` — **mock** — // generateMockPrices generates mock price data for testing
- [ ] `./internal\automation\scheduler\risk\risk_monitor_helpers.go:649` — **mock** — func (rm *RiskMonitor) generateMockPrices(startPrice float64, count int, volatility float64) []float64 {
- [ ] `./internal\automation\scheduler\risk\risk_monitor_helpers.go:662` — **mock** — // generateMockMarketData generates mock market data for testing
- [ ] `./internal\automation\scheduler\risk\risk_monitor_helpers.go:663` — **mock** — func (rm *RiskMonitor) generateMockMarketData() []MarketData {
- [ ] `./internal\automation\scheduler\risk\risk_monitor_helpers.go:670` — **mock** — Price:      50000 + float64(i)*1000, // Mock prices

## ./internal\automation\scheduler\risk\risk_monitor_test.go
- [ ] `./internal\automation\scheduler\risk\risk_monitor_test.go:344` — **mock** — func TestRiskMonitor_GenerateMockPrices(t *testing.T) {
- [ ] `./internal\automation\scheduler\risk\risk_monitor_test.go:351` — **mock** — prices := rm.generateMockPrices(startPrice, count, volatility)
- [ ] `./internal\automation\scheduler\risk\risk_monitor_test.go:368` — **mock** — func TestRiskMonitor_GenerateMockMarketData(t *testing.T) {
- [ ] `./internal\automation\scheduler\risk\risk_monitor_test.go:371` — **mock** — marketData := rm.generateMockMarketData()

## ./internal\automation\scheduler\risk\risk_reporter.go
- [ ] `./internal\automation\scheduler\risk\risk_reporter.go:672` — **mock** — // For now, return mock data

## ./internal\automation\scheduler\risk\stop_loss_adjuster_test.go
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:13` — **mock** — "github.com/DATA-DOG/go-sqlmock"
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:53` — **mock** — adjuster, mock := createTestStopLossAdjusterWithMock(t)
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:59` — **mock** — // Mock OHLC data query
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:60` — **mock** — ohlcRows := sqlmock.NewRows([]string{"timestamp", "open_price", "high_price", "low_price", "close_price", "volume"})
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:71` — **mock** — mock.ExpectQuery("SELECT timestamp, open_price, high_price, low_price, close_price, volume FROM market_data").
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:75` — **mock** — // Mock position query
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:76` — **mock** — positionRows := sqlmock.NewRows([]string{
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:83` — **mock** — mock.ExpectQuery("SELECT id, symbol, side, size, entry_price, current_price").
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:94` — **mock** — assert.NoError(t, mock.ExpectationsWereMet())
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:98` — **mock** — adjuster, mock := createTestStopLossAdjusterWithMock(t)
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:104` — **mock** — // Mock price data query
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:105` — **mock** — priceRows := sqlmock.NewRows([]string{"close_price"})
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:109` — **mock** — mock.ExpectQuery("SELECT close_price FROM market_data").
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:113` — **mock** — // Mock position query
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:114` — **mock** — positionRows := sqlmock.NewRows([]string{
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:121` — **mock** — mock.ExpectQuery("SELECT id, symbol, side, size, entry_price, current_price").
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:132` — **mock** — assert.NoError(t, mock.ExpectationsWereMet())
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:136` — **mock** — adjuster, mock := createTestStopLossAdjusterWithMock(t)
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:141` — **mock** — // Mock market data query for regime analysis
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:142` — **mock** — marketRows := sqlmock.NewRows([]string{"close_price", "volume", "timestamp"})
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:150` — **mock** — mock.ExpectQuery("SELECT close_price, volume, timestamp FROM market_data").
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:165` — **mock** — assert.NoError(t, mock.ExpectationsWereMet())
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:169` — **mock** — adjuster, mock := createTestStopLossAdjusterWithMock(t)
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:197` — **mock** — // Mock database updates
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:198` — **mock** — mock.ExpectExec("UPDATE positions SET stop_loss = \\?, updated_at = CURRENT_TIMESTAMP").
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:200` — **mock** — WillReturnResult(sqlmock.NewResult(1, 1))
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:202` — **mock** — mock.ExpectExec("INSERT INTO stop_loss_adjustments").
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:204` — **mock** — WillReturnResult(sqlmock.NewResult(1, 1))
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:206` — **mock** — mock.ExpectExec("UPDATE positions SET stop_loss = \\?, updated_at = CURRENT_TIMESTAMP").
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:208` — **mock** — WillReturnResult(sqlmock.NewResult(1, 1))
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:210` — **mock** — mock.ExpectExec("INSERT INTO stop_loss_adjustments").
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:212` — **mock** — WillReturnResult(sqlmock.NewResult(1, 1))
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:224` — **mock** — assert.NoError(t, mock.ExpectationsWereMet())
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:228` — **mock** — adjuster, mock := createTestStopLossAdjusterWithMock(t)
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:242` — **mock** — // Mock OHLC data for ATR calculation
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:243` — **mock** — ohlcRows := sqlmock.NewRows([]string{"timestamp", "open_price", "high_price", "low_price", "close_price", "volume"})
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:254` — **mock** — mock.ExpectQuery("SELECT timestamp, open_price, high_price, low_price, close_price, volume FROM market_data").
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:258` — **mock** — // Mock position query for ATR calculation
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:259` — **mock** — positionRows1 := sqlmock.NewRows([]string{
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:266` — **mock** — mock.ExpectQuery("SELECT id, symbol, side, size, entry_price, current_price").
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:270` — **mock** — // Mock price data for RV calculation
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:271` — **mock** — priceRows := sqlmock.NewRows([]string{"close_price"})
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:275` — **mock** — mock.ExpectQuery("SELECT close_price FROM market_data").
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:279` — **mock** — // Mock position query for RV calculation
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:280` — **mock** — positionRows2 := sqlmock.NewRows([]string{
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:287` — **mock** — mock.ExpectQuery("SELECT id, symbol, side, size, entry_price, current_price").
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:291` — **mock** — // Mock market data for regime analysis
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:292` — **mock** — marketRows := sqlmock.NewRows([]string{"close_price", "volume", "timestamp"})
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:300` — **mock** — mock.ExpectQuery("SELECT close_price, volume, timestamp FROM market_data").
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:310` — **mock** — assert.NoError(t, mock.ExpectationsWereMet())
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:314` — **mock** — adjuster, mock := createTestStopLossAdjusterWithMock(t)
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:319` — **mock** — // Mock active positions query
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:320` — **mock** — positionsRows := sqlmock.NewRows([]string{
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:330` — **mock** — mock.ExpectQuery("SELECT id, symbol, side, size, entry_price, current_price").
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:333` — **mock** — // Mock calculations for each position (simplified - would need full mock setup)
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:341` — **mock** — // We expect 0 adjustments because the mocked calculations will fail
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:374` — **mock** — func createTestStopLossAdjusterWithMock(t *testing.T) (*StopLossAdjuster, sqlmock.Sqlmock) {
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:375` — **mock** — // Create mock database
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:376` — **mock** — mockDB, mock, err := sqlmock.New()
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:380` — **mock** — db := &database.DB{DB: mockDB}
- [ ] `./internal\automation\scheduler\risk\stop_loss_adjuster_test.go:385` — **mock** — return adjuster, mock

## ./internal\automation\scheduler\risk\stop_loss_execution_test.go
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:13` — **mock** — "github.com/DATA-DOG/go-sqlmock"
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:53` — **mock** — executor, mock := createTestStopLossExecutorWithMock(t)
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:58` — **mock** — // Mock active positions query for GenerateStopLossAdjustments
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:59` — **mock** — positionsRows := sqlmock.NewRows([]string{
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:66` — **mock** — mock.ExpectQuery("SELECT id, symbol, side, size, entry_price, current_price").
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:69` — **mock** — // Since GenerateStopLossAdjustments will likely fail due to missing mock data,
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:82` — **mock** — executor, mock := createTestStopLossExecutorWithMock(t)
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:97` — **mock** — // Mock the adjustment execution
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:98` — **mock** — mock.ExpectExec("UPDATE positions SET stop_loss = \\?, updated_at = CURRENT_TIMESTAMP").
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:100` — **mock** — WillReturnResult(sqlmock.NewResult(1, 1))
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:102` — **mock** — mock.ExpectExec("INSERT INTO stop_loss_adjustments").
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:104` — **mock** — WillReturnResult(sqlmock.NewResult(1, 1))
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:106` — **mock** — // Mock performance tracking start
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:107` — **mock** — mock.ExpectQuery("SELECT close_price FROM market_data").
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:109` — **mock** — WillReturnRows(sqlmock.NewRows([]string{"close_price"}).AddRow(51000.0))
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:111` — **mock** — mock.ExpectExec("INSERT INTO stop_loss_performance").
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:112` — **mock** — WillReturnResult(sqlmock.NewResult(1, 1))
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:126` — **mock** — assert.NoError(t, mock.ExpectationsWereMet())
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:194` — **mock** — tracker, mock := createTestPerformanceTrackerWithMock(t)
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:207` — **mock** — // Mock current price query
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:208` — **mock** — mock.ExpectQuery("SELECT close_price FROM market_data").
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:210` — **mock** — WillReturnRows(sqlmock.NewRows([]string{"close_price"}).AddRow(51000.0))
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:212` — **mock** — // Mock performance record insertion
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:213` — **mock** — mock.ExpectExec("INSERT INTO stop_loss_performance").
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:214` — **mock** — WillReturnResult(sqlmock.NewResult(1, 1))
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:232` — **mock** — assert.NoError(t, mock.ExpectationsWereMet())
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:236` — **mock** — tracker, mock := createTestPerformanceTrackerWithMock(t)
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:241` — **mock** — // Mock active tracking records query
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:242` — **mock** — trackingRows := sqlmock.NewRows([]string{
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:253` — **mock** — mock.ExpectQuery("SELECT adjustment_id, position_id, symbol").
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:256` — **mock** — // Mock position status query (position still active)
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:257` — **mock** — positionRows := sqlmock.NewRows([]string{
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:264` — **mock** — mock.ExpectQuery("SELECT id, symbol, side, size, entry_price, current_price").
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:268` — **mock** — // Mock performance update
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:269` — **mock** — mock.ExpectExec("UPDATE stop_loss_performance").
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:270` — **mock** — WillReturnResult(sqlmock.NewResult(1, 1))
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:272` — **mock** — // Mock aggregate metrics calculation
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:273` — **mock** — statsRows := sqlmock.NewRows([]string{
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:277` — **mock** — mock.ExpectQuery("SELECT COUNT\\(\\*\\) as total_adjustments").
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:285` — **mock** — assert.NoError(t, mock.ExpectationsWereMet())
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:428` — **mock** — func createTestStopLossExecutorWithMock(t *testing.T) (*StopLossExecutor, sqlmock.Sqlmock) {
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:429` — **mock** — // Create mock database
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:430` — **mock** — mockDB, mock, err := sqlmock.New()
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:435` — **mock** — db := &database.DB{DB: mockDB}
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:439` — **mock** — executor.adjuster.db = db // Update adjuster's db to use mock
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:441` — **mock** — return executor, mock
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:453` — **mock** — func createTestPerformanceTrackerWithMock(t *testing.T) (*StopLossPerformanceTracker, sqlmock.Sqlmock) {
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:454` — **mock** — // Create mock database
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:455` — **mock** — mockDB, mock, err := sqlmock.New()
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:458` — **mock** — db := &database.DB{DB: mockDB}
- [ ] `./internal\automation\scheduler\risk\stop_loss_execution_test.go:465` — **mock** — return tracker, mock

## ./internal\automation\scheduler\risk\test_mocks.go
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:51` — **mock** — // QueryContext mock implementation
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:54` — **mock** — // In a real implementation, this would parse the query and return appropriate mock data
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:58` — **mock** — // ExecContext mock implementation
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:131` — **mock** — db:            nil, // We'll mock database operations
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:141` — **mock** — // MockRiskController creates a RiskController for testing with mocked database operations
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:142` — **mock** — type MockRiskController struct {
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:148` — **mock** — func NewTestRiskController() *MockRiskController {
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:161` — **mock** — mockRC := &MockRiskController{
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:166` — **mock** — return mockRC
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:170` — **mock** — func (mrc *MockRiskController) getCurrentPositions(ctx context.Context) ([]shared.Position, error) {
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:175` — **mock** — func (mrc *MockRiskController) executePositionReduction(ctx context.Context, reduction PositionReduction) error {
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:187` — **mock** — func (mrc *MockRiskController) executeEmergencyClose(ctx context.Context, position shared.Position) error {
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:199` — **mock** — func (mrc *MockRiskController) cancelAllPendingOrders(ctx context.Context) error {
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:206` — **mock** — func (mrc *MockRiskController) getHighLeveragePositions(ctx context.Context, maxLeverage float64) ([]shared.Position, error) {
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:217` — **mock** — func (mrc *MockRiskController) recordActionInDatabase(ctx context.Context, action RiskAction) error {
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:223` — **mock** — func (mrc *MockRiskController) recordAction(action RiskAction) {
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:235` — **mock** — // triggerPositionReductionMocked is a mocked version of TriggerPositionReduction for testing
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:236` — **mock** — func (mrc *MockRiskController) triggerPositionReductionMocked(ctx context.Context, marginStatus *MarginStatus, reductionPercent float64) (*RiskAction, error) {
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:254` — **mock** — // Get positions from mock data
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:269` — **mock** — // Execute position reductions using mock
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:304` — **mock** — // triggerEmergencyStopMocked is a mocked version of TriggerEmergencyStop for testing
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:305` — **mock** — func (mrc *MockRiskController) triggerEmergencyStopMocked(ctx context.Context, reason string) (*RiskAction, error) {
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:325` — **mock** — // Get all active positions from mock data
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:328` — **mock** — // Close all positions using mock
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:342` — **mock** — // Cancel all pending orders using mock
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:367` — **mock** — // triggerLeverageReductionMocked is a mocked version of TriggerLeverageReduction for testing
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:368` — **mock** — func (mrc *MockRiskController) triggerLeverageReductionMocked(ctx context.Context, targetLeverage float64) (*RiskAction, error) {
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:384` — **mock** — // Get positions with high leverage from mock data
- [ ] `./internal\automation\scheduler\risk\test_mocks.go:396` — **mock** — // Reduce leverage for each position using mock

## ./internal\automation\scheduler\shared\shared_test.go
- [ ] `./internal\automation\scheduler\shared\shared_test.go:376` — **mock** — assert.NotNil(t, tf.mocks)
- [ ] `./internal\automation\scheduler\shared\shared_test.go:380` — **mock** — t.Run("Mock management", func(t *testing.T) {
- [ ] `./internal\automation\scheduler\shared\shared_test.go:383` — **mock** — mockDB := NewMockDatabase()
- [ ] `./internal\automation\scheduler\shared\shared_test.go:384` — **mock** — tf.SetMock("database", mockDB)
- [ ] `./internal\automation\scheduler\shared\shared_test.go:386` — **mock** — retrieved := tf.GetMock("database")
- [ ] `./internal\automation\scheduler\shared\shared_test.go:387` — **mock** — assert.Equal(t, mockDB, retrieved)
- [ ] `./internal\automation\scheduler\shared\shared_test.go:403` — **mock** — func TestMockDatabase(t *testing.T) {
- [ ] `./internal\automation\scheduler\shared\shared_test.go:404` — **mock** — t.Run("NewMockDatabase", func(t *testing.T) {
- [ ] `./internal\automation\scheduler\shared\shared_test.go:405` — **mock** — mockDB := NewMockDatabase()
- [ ] `./internal\automation\scheduler\shared\shared_test.go:406` — **mock** — assert.NotNil(t, mockDB)
- [ ] `./internal\automation\scheduler\shared\shared_test.go:407` — **mock** — assert.NotNil(t, mockDB.queries)
- [ ] `./internal\automation\scheduler\shared\shared_test.go:411` — **mock** — mockDB := NewMockDatabase()
- [ ] `./internal\automation\scheduler\shared\shared_test.go:418` — **mock** — mockDB.SetQueryResult("SELECT * FROM test", results)
- [ ] `./internal\automation\scheduler\shared\shared_test.go:420` — **mock** — storedResults := mockDB.queries["SELECT * FROM test"]
- [ ] `./internal\automation\scheduler\shared\shared_test.go:425` — **mock** — func TestMockExchangeAPI(t *testing.T) {
- [ ] `./internal\automation\scheduler\shared\shared_test.go:426` — **mock** — t.Run("NewMockExchangeAPI", func(t *testing.T) {
- [ ] `./internal\automation\scheduler\shared\shared_test.go:427` — **mock** — mockAPI := NewMockExchangeAPI()
- [ ] `./internal\automation\scheduler\shared\shared_test.go:428` — **mock** — assert.NotNil(t, mockAPI)
- [ ] `./internal\automation\scheduler\shared\shared_test.go:429` — **mock** — assert.NotNil(t, mockAPI.positions)
- [ ] `./internal\automation\scheduler\shared\shared_test.go:430` — **mock** — assert.NotNil(t, mockAPI.marketData)
- [ ] `./internal\automation\scheduler\shared\shared_test.go:431` — **mock** — assert.NotNil(t, mockAPI.orderHistory)
- [ ] `./internal\automation\scheduler\shared\shared_test.go:435` — **mock** — mockAPI := NewMockExchangeAPI()
- [ ] `./internal\automation\scheduler\shared\shared_test.go:446` — **mock** — mockAPI.SetPositions(positions)
- [ ] `./internal\automation\scheduler\shared\shared_test.go:448` — **mock** — mockAPI.On("GetPositions", context.Background()).Return(nil)
- [ ] `./internal\automation\scheduler\shared\shared_test.go:450` — **mock** — retrieved, err := mockAPI.GetPositions(context.Background())
- [ ] `./internal\automation\scheduler\shared\shared_test.go:456` — **mock** — mockAPI := NewMockExchangeAPI()
- [ ] `./internal\automation\scheduler\shared\shared_test.go:463` — **mock** — mockAPI.SetMarketData("BTCUSDT", marketData)
- [ ] `./internal\automation\scheduler\shared\shared_test.go:464` — **mock** — mockAPI.On("GetMarketData", context.Background(), "BTCUSDT").Return(nil)
- [ ] `./internal\automation\scheduler\shared\shared_test.go:466` — **mock** — retrieved, err := mockAPI.GetMarketData(context.Background(), "BTCUSDT")

## ./internal\automation\scheduler\shared\testing.go
- [ ] `./internal\automation\scheduler\shared\testing.go:13` — **mock** — "github.com/stretchr/testify/mock"
- [ ] `./internal\automation\scheduler\shared\testing.go:20` — **mock** — mocks          map[string]interface{}
- [ ] `./internal\automation\scheduler\shared\testing.go:30` — **mock** — mocks:        make(map[string]interface{}),
- [ ] `./internal\automation\scheduler\shared\testing.go:55` — **mock** — // SetMock stores a mock object for later retrieval
- [ ] `./internal\automation\scheduler\shared\testing.go:56` — **mock** — func (tf *TestFramework) SetMock(name string, mockObj interface{}) {
- [ ] `./internal\automation\scheduler\shared\testing.go:60` — **mock** — tf.mocks[name] = mockObj
- [ ] `./internal\automation\scheduler\shared\testing.go:63` — **mock** — // GetMock retrieves a stored mock object
- [ ] `./internal\automation\scheduler\shared\testing.go:64` — **mock** — func (tf *TestFramework) GetMock(name string) interface{} {
- [ ] `./internal\automation\scheduler\shared\testing.go:68` — **mock** — return tf.mocks[name]
- [ ] `./internal\automation\scheduler\shared\testing.go:87` — **mock** — // MockDatabase provides a mock database for testing
- [ ] `./internal\automation\scheduler\shared\testing.go:88` — **mock** — type MockDatabase struct {
- [ ] `./internal\automation\scheduler\shared\testing.go:89` — **mock** — mock.Mock
- [ ] `./internal\automation\scheduler\shared\testing.go:94` — **mock** — // NewMockDatabase creates a new mock database
- [ ] `./internal\automation\scheduler\shared\testing.go:95` — **mock** — func NewMockDatabase() *MockDatabase {
- [ ] `./internal\automation\scheduler\shared\testing.go:96` — **mock** — return &MockDatabase{
- [ ] `./internal\automation\scheduler\shared\testing.go:101` — **mock** — // QueryContext mocks database query execution
- [ ] `./internal\automation\scheduler\shared\testing.go:102` — **mock** — func (mdb *MockDatabase) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
- [ ] `./internal\automation\scheduler\shared\testing.go:106` — **mock** — mockArgs := mdb.Called(ctx, query, args)
- [ ] `./internal\automation\scheduler\shared\testing.go:107` — **mock** — return mockArgs.Get(0).(*sql.Rows), mockArgs.Error(1)
- [ ] `./internal\automation\scheduler\shared\testing.go:110` — **mock** — // QueryRowContext mocks database single row query
- [ ] `./internal\automation\scheduler\shared\testing.go:111` — **mock** — func (mdb *MockDatabase) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
- [ ] `./internal\automation\scheduler\shared\testing.go:115` — **mock** — mockArgs := mdb.Called(ctx, query, args)
- [ ] `./internal\automation\scheduler\shared\testing.go:116` — **mock** — return mockArgs.Get(0).(*sql.Row)
- [ ] `./internal\automation\scheduler\shared\testing.go:119` — **mock** — // ExecContext mocks database execution
- [ ] `./internal\automation\scheduler\shared\testing.go:120` — **mock** — func (mdb *MockDatabase) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
- [ ] `./internal\automation\scheduler\shared\testing.go:124` — **mock** — mockArgs := mdb.Called(ctx, query, args)
- [ ] `./internal\automation\scheduler\shared\testing.go:125` — **mock** — return mockArgs.Get(0).(sql.Result), mockArgs.Error(1)
- [ ] `./internal\automation\scheduler\shared\testing.go:129` — **mock** — func (mdb *MockDatabase) SetQueryResult(query string, results []map[string]interface{}) {
- [ ] `./internal\automation\scheduler\shared\testing.go:136` — **mock** — // MockExchangeAPI provides a mock exchange API for testing
- [ ] `./internal\automation\scheduler\shared\testing.go:137` — **mock** — type MockExchangeAPI struct {
- [ ] `./internal\automation\scheduler\shared\testing.go:138` — **mock** — mock.Mock
- [ ] `./internal\automation\scheduler\shared\testing.go:145` — **mock** — // NewMockExchangeAPI creates a new mock exchange API
- [ ] `./internal\automation\scheduler\shared\testing.go:146` — **mock** — func NewMockExchangeAPI() *MockExchangeAPI {
- [ ] `./internal\automation\scheduler\shared\testing.go:147` — **mock** — return &MockExchangeAPI{
- [ ] `./internal\automation\scheduler\shared\testing.go:154` — **mock** — // GetPositions mocks getting positions from exchange
- [ ] `./internal\automation\scheduler\shared\testing.go:155` — **mock** — func (mea *MockExchangeAPI) GetPositions(ctx context.Context) ([]Position, error) {
- [ ] `./internal\automation\scheduler\shared\testing.go:159` — **mock** — mockArgs := mea.Called(ctx)
- [ ] `./internal\automation\scheduler\shared\testing.go:160` — **mock** — return mea.positions, mockArgs.Error(0)
- [ ] `./internal\automation\scheduler\shared\testing.go:163` — **mock** — // GetMarketData mocks getting market data from exchange
- [ ] `./internal\automation\scheduler\shared\testing.go:164` — **mock** — func (mea *MockExchangeAPI) GetMarketData(ctx context.Context, symbol string) (map[string]interface{}, error) {
- [ ] `./internal\automation\scheduler\shared\testing.go:168` — **mock** — mockArgs := mea.Called(ctx, symbol)
- [ ] `./internal\automation\scheduler\shared\testing.go:170` — **mock** — return data.(map[string]interface{}), mockArgs.Error(0)
- [ ] `./internal\automation\scheduler\shared\testing.go:172` — **mock** — return nil, mockArgs.Error(0)
- [ ] `./internal\automation\scheduler\shared\testing.go:175` — **mock** — // PlaceOrder mocks placing an order
- [ ] `./internal\automation\scheduler\shared\testing.go:176` — **mock** — func (mea *MockExchangeAPI) PlaceOrder(ctx context.Context, order map[string]interface{}) (string, error) {
- [ ] `./internal\automation\scheduler\shared\testing.go:180` — **mock** — mockArgs := mea.Called(ctx, order)
- [ ] `./internal\automation\scheduler\shared\testing.go:193` — **mock** — return orderID, mockArgs.Error(0)
- [ ] `./internal\automation\scheduler\shared\testing.go:196` — **mock** — // SetPositions sets mock positions
- [ ] `./internal\automation\scheduler\shared\testing.go:197` — **mock** — func (mea *MockExchangeAPI) SetPositions(positions []Position) {
- [ ] `./internal\automation\scheduler\shared\testing.go:204` — **mock** — // SetMarketData sets mock market data
- [ ] `./internal\automation\scheduler\shared\testing.go:205` — **mock** — func (mea *MockExchangeAPI) SetMarketData(symbol string, data map[string]interface{}) {
- [ ] `./internal\automation\scheduler\shared\testing.go:213` — **mock** — func (mea *MockExchangeAPI) GetOrderHistory() []map[string]interface{} {
- [ ] `./internal\automation\scheduler\shared\testing.go:222` — **mock** — // MockConfigProvider provides a mock configuration provider for testing
- [ ] `./internal\automation\scheduler\shared\testing.go:223` — **mock** — type MockConfigProvider struct {
- [ ] `./internal\automation\scheduler\shared\testing.go:224` — **mock** — mock.Mock
- [ ] `./internal\automation\scheduler\shared\testing.go:229` — **mock** — // NewMockConfigProvider creates a new mock config provider
- [ ] `./internal\automation\scheduler\shared\testing.go:230` — **mock** — func NewMockConfigProvider() *MockConfigProvider {
- [ ] `./internal\automation\scheduler\shared\testing.go:231` — **mock** — return &MockConfigProvider{
- [ ] `./internal\automation\scheduler\shared\testing.go:236` — **mock** — // Get mocks getting a configuration value
- [ ] `./internal\automation\scheduler\shared\testing.go:237` — **mock** — func (mcp *MockConfigProvider) Get(key string) interface{} {
- [ ] `./internal\automation\scheduler\shared\testing.go:241` — **mock** — mockArgs := mcp.Called(key)
- [ ] `./internal\automation\scheduler\shared\testing.go:245` — **mock** — return mockArgs.Get(0)
- [ ] `./internal\automation\scheduler\shared\testing.go:248` — **mock** — // GetString mocks getting a string configuration value
- [ ] `./internal\automation\scheduler\shared\testing.go:249` — **mock** — func (mcp *MockConfigProvider) GetString(key string) string {
- [ ] `./internal\automation\scheduler\shared\testing.go:257` — **mock** — // GetInt mocks getting an integer configuration value
- [ ] `./internal\automation\scheduler\shared\testing.go:258` — **mock** — func (mcp *MockConfigProvider) GetInt(key string) int {
- [ ] `./internal\automation\scheduler\shared\testing.go:266` — **mock** — // GetFloat64 mocks getting a float64 configuration value
- [ ] `./internal\automation\scheduler\shared\testing.go:267` — **mock** — func (mcp *MockConfigProvider) GetFloat64(key string) float64 {
- [ ] `./internal\automation\scheduler\shared\testing.go:275` — **mock** — // GetBool mocks getting a boolean configuration value
- [ ] `./internal\automation\scheduler\shared\testing.go:276` — **mock** — func (mcp *MockConfigProvider) GetBool(key string) bool {
- [ ] `./internal\automation\scheduler\shared\testing.go:284` — **mock** — // GetDuration mocks getting a duration configuration value
- [ ] `./internal\automation\scheduler\shared\testing.go:285` — **mock** — func (mcp *MockConfigProvider) GetDuration(key string) time.Duration {
- [ ] `./internal\automation\scheduler\shared\testing.go:293` — **mock** — // Set mocks setting a configuration value
- [ ] `./internal\automation\scheduler\shared\testing.go:294` — **mock** — func (mcp *MockConfigProvider) Set(key string, value interface{}) error {
- [ ] `./internal\automation\scheduler\shared\testing.go:298` — **mock** — mockArgs := mcp.Called(key, value)
- [ ] `./internal\automation\scheduler\shared\testing.go:300` — **mock** — return mockArgs.Error(0)
- [ ] `./internal\automation\scheduler\shared\testing.go:303` — **mock** — // Reload mocks reloading configuration
- [ ] `./internal\automation\scheduler\shared\testing.go:304` — **mock** — func (mcp *MockConfigProvider) Reload() error {
- [ ] `./internal\automation\scheduler\shared\testing.go:305` — **mock** — mockArgs := mcp.Called()
- [ ] `./internal\automation\scheduler\shared\testing.go:306` — **mock** — return mockArgs.Error(0)
- [ ] `./internal\automation\scheduler\shared\testing.go:310` — **mock** — func (mcp *MockConfigProvider) SetConfig(key string, value interface{}) {
- [ ] `./internal\automation\scheduler\shared\testing.go:317` — **mock** — // MockMetricsCollector provides a mock metrics collector for testing
- [ ] `./internal\automation\scheduler\shared\testing.go:318` — **mock** — type MockMetricsCollector struct {
- [ ] `./internal\automation\scheduler\shared\testing.go:319` — **mock** — mock.Mock
- [ ] `./internal\automation\scheduler\shared\testing.go:327` — **mock** — // NewMockMetricsCollector creates a new mock metrics collector
- [ ] `./internal\automation\scheduler\shared\testing.go:328` — **mock** — func NewMockMetricsCollector() *MockMetricsCollector {
- [ ] `./internal\automation\scheduler\shared\testing.go:329` — **mock** — return &MockMetricsCollector{
- [ ] `./internal\automation\scheduler\shared\testing.go:337` — **mock** — // Counter mocks incrementing a counter metric
- [ ] `./internal\automation\scheduler\shared\testing.go:338` — **mock** — func (mmc *MockMetricsCollector) Counter(name string, tags map[string]string) error {
- [ ] `./internal\automation\scheduler\shared\testing.go:342` — **mock** — mockArgs := mmc.Called(name, tags)
- [ ] `./internal\automation\scheduler\shared\testing.go:344` — **mock** — return mockArgs.Error(0)
- [ ] `./internal\automation\scheduler\shared\testing.go:347` — **mock** — // Gauge mocks setting a gauge metric
- [ ] `./internal\automation\scheduler\shared\testing.go:348` — **mock** — func (mmc *MockMetricsCollector) Gauge(name string, value float64, tags map[string]string) error {
- [ ] `./internal\automation\scheduler\shared\testing.go:352` — **mock** — mockArgs := mmc.Called(name, value, tags)
- [ ] `./internal\automation\scheduler\shared\testing.go:354` — **mock** — return mockArgs.Error(0)
- [ ] `./internal\automation\scheduler\shared\testing.go:357` — **mock** — // Histogram mocks recording a histogram metric
- [ ] `./internal\automation\scheduler\shared\testing.go:358` — **mock** — func (mmc *MockMetricsCollector) Histogram(name string, value float64, tags map[string]string) error {
- [ ] `./internal\automation\scheduler\shared\testing.go:362` — **mock** — mockArgs := mmc.Called(name, value, tags)
- [ ] `./internal\automation\scheduler\shared\testing.go:364` — **mock** — return mockArgs.Error(0)
- [ ] `./internal\automation\scheduler\shared\testing.go:367` — **mock** — // Timer mocks recording a timer metric
- [ ] `./internal\automation\scheduler\shared\testing.go:368` — **mock** — func (mmc *MockMetricsCollector) Timer(name string, duration time.Duration, tags map[string]string) error {
- [ ] `./internal\automation\scheduler\shared\testing.go:372` — **mock** — mockArgs := mmc.Called(name, duration, tags)
- [ ] `./internal\automation\scheduler\shared\testing.go:374` — **mock** — return mockArgs.Error(0)
- [ ] `./internal\automation\scheduler\shared\testing.go:378` — **mock** — func (mmc *MockMetricsCollector) GetCounterValue(name string) int64 {
- [ ] `./internal\automation\scheduler\shared\testing.go:386` — **mock** — func (mmc *MockMetricsCollector) GetGaugeValue(name string) float64 {
- [ ] `./internal\automation\scheduler\shared\testing.go:394` — **mock** — func (mmc *MockMetricsCollector) GetHistogramValues(name string) []float64 {
- [ ] `./internal\automation\scheduler\shared\testing.go:404` — **mock** — func (mmc *MockMetricsCollector) GetTimerValues(name string) []time.Duration {
- [ ] `./internal\automation\scheduler\shared\testing.go:544` — **mock** — func (ah *AssertionHelpers) AssertMetricsRecorded(collector *MockMetricsCollector, metricName string) {

## ./internal\cache\cache_test.go
- [ ] `./internal\cache\cache_test.go:189` — **mock** — mockData := testutils.NewMockData()
- [ ] `./internal\cache\cache_test.go:193` — **mock** — key := mockData.RandomString(10)
- [ ] `./internal\cache\cache_test.go:194` — **mock** — value := mockData.RandomString(100)

## ./internal\cache\fallback_test.go
- [ ] `./internal\cache\fallback_test.go:11` — **mock** — // Create a mock Redis cache that will fail
- [ ] `./internal\cache\fallback_test.go:12` — **mock** — mockRedis := &MockRedisCache{
- [ ] `./internal\cache\fallback_test.go:22` — **mock** — cm := NewCacheManager(mockRedis, nil, config)
- [ ] `./internal\cache\fallback_test.go:43` — **mock** — mockRedis.shouldFail = true
- [ ] `./internal\cache\fallback_test.go:156` — **mock** — mockRedis := &MockRedisCache{
- [ ] `./internal\cache\fallback_test.go:162` — **mock** — cm := NewCacheManager(mockRedis, nil, config)
- [ ] `./internal\cache\fallback_test.go:224` — **mock** — mockRedis := &MockRedisCache{
- [ ] `./internal\cache\fallback_test.go:229` — **mock** — cm := NewCacheManager(mockRedis, nil, DefaultFallbackConfig())
- [ ] `./internal\cache\fallback_test.go:279` — **mock** — // MockRedisCache is a mock implementation for testing
- [ ] `./internal\cache\fallback_test.go:280` — **mock** — type MockRedisCache struct {
- [ ] `./internal\cache\fallback_test.go:285` — **mock** — func (m *MockRedisCache) Get(ctx context.Context, key string) (interface{}, error) {
- [ ] `./internal\cache\fallback_test.go:287` — **mock** — return nil, fmt.Errorf("mock Redis failure")
- [ ] `./internal\cache\fallback_test.go:298` — **mock** — func (m *MockRedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
- [ ] `./internal\cache\fallback_test.go:300` — **mock** — return fmt.Errorf("mock Redis failure")
- [ ] `./internal\cache\fallback_test.go:307` — **mock** — func (m *MockRedisCache) Delete(ctx context.Context, key string) error {
- [ ] `./internal\cache\fallback_test.go:309` — **mock** — return fmt.Errorf("mock Redis failure")
- [ ] `./internal\cache\fallback_test.go:316` — **mock** — func (m *MockRedisCache) Exists(ctx context.Context, key string) (bool, error) {
- [ ] `./internal\cache\fallback_test.go:318` — **mock** — return false, fmt.Errorf("mock Redis failure")
- [ ] `./internal\cache\fallback_test.go:325` — **mock** — func (m *MockRedisCache) Close() error {
- [ ] `./internal\cache\fallback_test.go:330` — **mock** — func (m *MockRedisCache) HGet(ctx context.Context, key, field string, dest interface{}) error {
- [ ] `./internal\cache\fallback_test.go:334` — **mock** — func (m *MockRedisCache) HSet(ctx context.Context, key, field string, value interface{}) error {
- [ ] `./internal\cache\fallback_test.go:338` — **mock** — func (m *MockRedisCache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
- [ ] `./internal\cache\fallback_test.go:342` — **mock** — func (m *MockRedisCache) HDel(ctx context.Context, key string, fields ...string) error {
- [ ] `./internal\cache\fallback_test.go:346` — **mock** — func (m *MockRedisCache) LPush(ctx context.Context, key string, values ...interface{}) error {
- [ ] `./internal\cache\fallback_test.go:350` — **mock** — func (m *MockRedisCache) RPush(ctx context.Context, key string, values ...interface{}) error {
- [ ] `./internal\cache\fallback_test.go:354` — **mock** — func (m *MockRedisCache) LPop(ctx context.Context, key string, dest interface{}) error {
- [ ] `./internal\cache\fallback_test.go:358` — **mock** — func (m *MockRedisCache) RPop(ctx context.Context, key string, dest interface{}) error {
- [ ] `./internal\cache\fallback_test.go:362` — **mock** — func (m *MockRedisCache) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
- [ ] `./internal\cache\fallback_test.go:366` — **mock** — func (m *MockRedisCache) SAdd(ctx context.Context, key string, members ...interface{}) error {
- [ ] `./internal\cache\fallback_test.go:370` — **mock** — func (m *MockRedisCache) SRem(ctx context.Context, key string, members ...interface{}) error {
- [ ] `./internal\cache\fallback_test.go:374` — **mock** — func (m *MockRedisCache) SMembers(ctx context.Context, key string) ([]string, error) {
- [ ] `./internal\cache\fallback_test.go:378` — **mock** — func (m *MockRedisCache) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
- [ ] `./internal\cache\fallback_test.go:382` — **mock** — func (m *MockRedisCache) ZAdd(ctx context.Context, key string, score float64, member interface{}) error {
- [ ] `./internal\cache\fallback_test.go:386` — **mock** — func (m *MockRedisCache) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
- [ ] `./internal\cache\fallback_test.go:390` — **mock** — func (m *MockRedisCache) ZRangeByScore(ctx context.Context, key string, min, max string) ([]string, error) {
- [ ] `./internal\cache\fallback_test.go:394` — **mock** — func (m *MockRedisCache) ZRem(ctx context.Context, key string, members ...interface{}) error {
- [ ] `./internal\cache\fallback_test.go:398` — **mock** — func (m *MockRedisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
- [ ] `./internal\cache\fallback_test.go:402` — **mock** — func (m *MockRedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
- [ ] `./internal\cache\fallback_test.go:406` — **mock** — func (m *MockRedisCache) Flush(ctx context.Context) error {
- [ ] `./internal\cache\fallback_test.go:411` — **mock** — func (m *MockRedisCache) SetFundingRate(ctx context.Context, symbol string, rate interface{}, expiration time.Duration) error {
- [ ] `./internal\cache\fallback_test.go:415` — **mock** — func (m *MockRedisCache) GetFundingRate(ctx context.Context, symbol string, dest interface{}) error {
- [ ] `./internal\cache\fallback_test.go:419` — **mock** — func (m *MockRedisCache) SetIndexPrice(ctx context.Context, symbol string, price interface{}, expiration time.Duration) error {
- [ ] `./internal\cache\fallback_test.go:423` — **mock** — func (m *MockRedisCache) GetIndexPrice(ctx context.Context, symbol string, dest interface{}) error {
- [ ] `./internal\cache\fallback_test.go:427` — **mock** — func (m *MockRedisCache) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
- [ ] `./internal\cache\fallback_test.go:431` — **mock** — func (m *MockRedisCache) SetOrderBook(ctx context.Context, symbol string, snapshot interface{}, expiration time.Duration) error {
- [ ] `./internal\cache\fallback_test.go:435` — **mock** — func (m *MockRedisCache) GetOrderBook(ctx context.Context, symbol string, dest interface{}) error {

## ./internal\concurrent\goroutine_pool.go
- [ ] `./internal\concurrent\goroutine_pool.go:277` — **TODO** — // TODO 添加结果处理逻辑

## ./internal\events\automation_handlers.go
- [ ] `./internal\events\automation_handlers.go:157` — **TODO** — // TODO 触发功能执行

## ./internal\fund\hedging\smart_hedging_system.go
- [ ] `./internal\fund\hedging\smart_hedging_system.go:337` — **TODO** — // TODO: 从配置文件读取对冲参数
- [ ] `./internal\fund\hedging\smart_hedging_system.go:887` — **TODO** — // TODO: 实现效用最大化对冲比率计算
- [ ] `./internal\fund\hedging\smart_hedging_system.go:892` — **TODO** — // TODO: 实现VaR最小化对冲比率计算
- [ ] `./internal\fund\hedging\smart_hedging_system.go:950` — **TODO** — // TODO: 实现实际的交易执行逻辑
- [ ] `./internal\fund\hedging\smart_hedging_system.go:1093` — **TODO** — // TODO: 实现低效对冲的处理逻辑
- [ ] `./internal\fund\hedging\smart_hedging_system.go:1098` — **TODO** — // TODO: 实现基于真实市场数据的状态检测
- [ ] `./internal\fund\hedging\smart_hedging_system.go:1119` — **TODO** — // TODO: 根据市场条件计算动态调整参数
- [ ] `./internal\fund\hedging\smart_hedging_system.go:1124` — **TODO** — // TODO: 应用动态调整
- [ ] `./internal\fund\hedging\smart_hedging_system.go:1129` — **TODO** — // TODO: 从历史数据计算波动率
- [ ] `./internal\fund\hedging\smart_hedging_system.go:1143` — **TODO** — // TODO: 计算Beta值
- [ ] `./internal\fund\hedging\smart_hedging_system.go:1148` — **TODO** — // TODO: 计算跟踪误差
- [ ] `./internal\fund\hedging\smart_hedging_system.go:1159` — **TODO** — // TODO: 计算基差风险
- [ ] `./internal\fund\hedging\smart_hedging_system.go:1164` — **TODO** — // TODO: 计算组合收益率
- [ ] `./internal\fund\hedging\smart_hedging_system.go:1169` — **TODO** — // TODO: 计算对冲后收益率
- [ ] `./internal\fund\hedging\smart_hedging_system.go:1174` — **TODO** — // TODO: 计算未对冲收益率
- [ ] `./internal\fund\hedging\smart_hedging_system.go:1187` — **TODO** — // TODO: 计算组合跟踪误差
- [ ] `./internal\fund\hedging\smart_hedging_system.go:1237` — **TODO** — // TODO: 计算平均相关性

## ./internal\fund\management\layered_position_manager.go
- [ ] `./internal\fund\management\layered_position_manager.go:301` — **TODO** — // TODO: 从配置文件读取分层管理参数
- [ ] `./internal\fund\management\layered_position_manager.go:805` — **TODO** — // TODO: 实现目标分配计算
- [ ] `./internal\fund\management\layered_position_manager.go:822` — **TODO** — // TODO: 实现具体的变化计算逻辑
- [ ] `./internal\fund\management\layered_position_manager.go:852` — **TODO** — // TODO: 实现预期收益计算
- [ ] `./internal\fund\management\layered_position_manager.go:857` — **TODO** — // TODO: 实现实际收益计算
- [ ] `./internal\fund\management\layered_position_manager.go:862` — **TODO** — // TODO: 实现实际的交易执行
- [ ] `./internal\fund\management\layered_position_manager.go:880` — **TODO** — // TODO: 实现具体的风险指标计算
- [ ] `./internal\fund\management\layered_position_manager.go:893` — **TODO** — // TODO: 实现具体的性能计算
- [ ] `./internal\fund\management\layered_position_manager.go:910` — **TODO** — // TODO: 实现具体的风险响应动作
- [ ] `./internal\fund\management\layered_position_manager.go:988` — **TODO** — // TODO: 实现分配效率计算

## ./internal\hotlist\integrated_service.go
- [ ] `./internal\hotlist\integrated_service.go:98` — **TODO** — // TODO: 正确初始化kline, funding, oi managers
- [ ] `./internal\hotlist\integrated_service.go:737` — **TODO** — // TODO: 实现从实际交易所API获取市场数据

## ./internal\intelligence\controller.go
- [ ] `./internal\intelligence\controller.go:232` — **TODO** — // TODO: 从配置文件读取间隔
- [ ] `./internal\intelligence\controller.go:355` — **TODO** — // TODO: 实现具体的动态优化逻辑
- [ ] `./internal\intelligence\controller.go:395` — **TODO** — // TODO: 实现市场状态检测逻辑
- [ ] `./internal\intelligence\controller.go:427` — **TODO** — // TODO: 实现智能执行逻辑
- [ ] `./internal\intelligence\controller.go:463` — **TODO** — // TODO: 实现利润最大化逻辑
- [ ] `./internal\intelligence\controller.go:495` — **TODO** — // TODO: 实现订单事件处理逻辑
- [ ] `./internal\intelligence\controller.go:505` — **TODO** — // TODO: 实现告警处理逻辑
- [ ] `./internal\intelligence\controller.go:515` — **TODO** — // TODO: 实现通知处理逻辑
- [ ] `./internal\intelligence\controller.go:526` — **TODO** — // TODO: 实现性能指标计算

## ./internal\learning\automl\automl_engine.go
- [ ] `./internal\learning\automl\automl_engine.go:589` — **TODO** — // TODO: 从配置文件读取AutoML参数
- [ ] `./internal\learning\automl\automl_engine.go:786` — **TODO** — // TODO: 实现特征工程初始化
- [ ] `./internal\learning\automl\automl_engine.go:792` — **TODO** — // TODO: 实现模型创建器初始化
- [ ] `./internal\learning\automl\automl_engine.go:799` — **TODO** — // TODO: 实现评估指标初始化
- [ ] `./internal\learning\automl\automl_engine.go:805` — **TODO** — // TODO: 实现集成方法初始化
- [ ] `./internal\learning\automl\automl_engine.go:811` — **TODO** — // TODO: 实现部署目标初始化
- [ ] `./internal\learning\automl\automl_engine.go:1207` — **TODO** — ModelSize:         0, // TODO: 计算实际模型大小
- [ ] `./internal\learning\automl\automl_engine.go:1241` — **TODO** — // TODO: 实现实际的数据预处理逻辑
- [ ] `./internal\learning\automl\automl_engine.go:1282` — **TODO** — // TODO: 实现实际的特征工程逻辑
- [ ] `./internal\learning\automl\automl_engine.go:1329` — **TODO** — // TODO: 实现实际的模型训练逻辑
- [ ] `./internal\learning\automl\automl_engine.go:1335` — **TODO** — // TODO: 实现真实的模型训练过程
- [ ] `./internal\learning\automl\automl_engine.go:1394` — **TODO** — // TODO: 实现实际的集成建模逻辑
- [ ] `./internal\learning\automl\automl_engine.go:1458` — **TODO** — // TODO: 实现实际的在线性能评估
- [ ] `./internal\learning\automl\automl_engine.go:1474` — **TODO** — // TODO: 从监控系统获取真实的在线指标
- [ ] `./internal\learning\automl\automl_engine.go:1527` — **TODO** — // TODO: 实现重训练任务创建
- [ ] `./internal\learning\automl\automl_engine.go:1655` — **mock** — func (engine *AutoMLEngine) generateMockHyperparameters(modelType string) map[string]interface{} {
- [ ] `./internal\learning\automl\automl_engine.go:1797` — **TODO** — // TODO: 实现从不同数据源加载数据的逻辑
- [ ] `./internal\learning\automl\automl_engine.go:1829` — **TODO** — // TODO: 实现数据预处理策略应用
- [ ] `./internal\learning\automl\automl_engine.go:1841` — **TODO** — // TODO: 实现数据库数据加载
- [ ] `./internal\learning\automl\automl_engine.go:1847` — **TODO** — // TODO: 实现文件数据加载
- [ ] `./internal\learning\automl\automl_engine.go:1853` — **TODO** — // TODO: 实现API数据加载
- [ ] `./internal\learning\automl\automl_engine.go:1859` — **TODO** — // TODO: 实现流数据加载

## ./internal\learning\automl\consistency_manager.go
- [ ] `./internal\learning\automl\consistency_manager.go:112` — **TODO** — // TODO: 从配置文件读取一致性参数
- [ ] `./internal\learning\automl\consistency_manager.go:348` — **TODO** — // TODO: 实现实际的网络广播
- [ ] `./internal\learning\automl\consistency_manager.go:354` — **TODO** — // TODO: 实现实际的网络查询
- [ ] `./internal\learning\automl\consistency_manager.go:404` — **TODO** — // TODO: 实现集群状态同步

## ./internal\learning\automl\distributed_optimizer.go
- [ ] `./internal\learning\automl\distributed_optimizer.go:456` — **TODO** — // TODO: 实现实际的网络广播逻辑
- [ ] `./internal\learning\automl\distributed_optimizer.go:493` — **TODO** — // TODO: 实现实际的应用逻辑
- [ ] `./internal\learning\automl\distributed_optimizer.go:552` — **TODO** — // TODO: 实现与集群其他节点的结果同步
- [ ] `./internal\learning\automl\distributed_optimizer.go:581` — **TODO** — // TODO: 实现节点发现逻辑
- [ ] `./internal\learning\automl\distributed_optimizer.go:587` — **TODO** — // TODO: 实现实际的广播逻辑

## ./internal\monitor\metrics.go
- [ ] `./internal\monitor\metrics.go:487` — **TODO** — // TODO: 实现从Prometheus Histogram获取真实统计数据
- [ ] `./internal\monitor\metrics.go:544` — **TODO** — // TODO: 实现从Prometheus Counter获取真实计数值
- [ ] `./internal\monitor\metrics.go:620` — **TODO** — // TODO: 实现真实的CPU使用率获取
- [ ] `./internal\monitor\metrics.go:628` — **TODO** — // TODO: 实现真实的内存使用率获取
- [ ] `./internal\monitor\metrics.go:635` — **TODO** — // TODO: 实现真实的磁盘使用率获取
- [ ] `./internal\monitor\metrics.go:642` — **TODO** — // TODO: 实现真实的网络IO获取
- [ ] `./internal\monitor\metrics.go:649` — **TODO** — // TODO: 实现真实的活跃连接数获取
- [ ] `./internal\monitor\metrics.go:656` — **TODO** — // TODO: 实现真实的数据库连接数获取
- [ ] `./internal\monitor\metrics.go:663` — **TODO** — // TODO: 实现真实的Redis连接数获取
- [ ] `./internal\monitor\metrics.go:670` — **TODO** — // TODO: 实现真实的系统运行时间获取

## ./internal\operations\healing\self_healing_system.go
- [ ] `./internal\operations\healing\self_healing_system.go:711` — **TODO** — // TODO: 从配置文件读取自愈参数
- [ ] `./internal\operations\healing\self_healing_system.go:1544` — **TODO** — // TODO: 实现重试逻辑
- [ ] `./internal\operations\healing\self_healing_system.go:1577` — **TODO** — // TODO: 实现实际的API调用
- [ ] `./internal\operations\healing\self_healing_system.go:1586` — **TODO** — // TODO: 实现实际的配置变更
- [ ] `./internal\operations\healing\self_healing_system.go:1682` — **TODO** — // TODO: 实现异常检测逻辑
- [ ] `./internal\operations\healing\self_healing_system.go:1686` — **TODO** — // TODO: 基于故障更新系统健康状态
- [ ] `./internal\operations\healing\self_healing_system.go:1714` — **TODO** — // TODO: 实现实际的API服务器健康检查
- [ ] `./internal\operations\healing\self_healing_system.go:1731` — **TODO** — // TODO: 实现实际的数据库健康检查
- [ ] `./internal\operations\healing\self_healing_system.go:1748` — **TODO** — // TODO: 实现实际的Redis健康检查
- [ ] `./internal\operations\healing\self_healing_system.go:1765` — **TODO** — // TODO: 实现实际的交易所连接器健康检查
- [ ] `./internal\operations\healing\self_healing_system.go:1787` — **TODO** — // TODO: 实现实际的策略引擎健康检查
- [ ] `./internal\operations\healing\self_healing_system.go:1799` — **TODO** — // TODO: 实现根因分析逻辑
- [ ] `./internal\operations\healing\self_healing_system.go:1810` — **TODO** — // TODO: 实现影响评估逻辑
- [ ] `./internal\operations\healing\self_healing_system.go:1904` — **TODO** — // TODO: 更新知识库，记录成功/失败的恢复案例
- [ ] `./internal\operations\healing\self_healing_system.go:1908` — **TODO** — // TODO: 基于历史数据计算平均时间
- [ ] `./internal\operations\healing\self_healing_system.go:1916` — **TODO** — // TODO: 计算实际的正常运行时间百分比
- [ ] `./internal\operations\healing\self_healing_system.go:2010` — **TODO** — // TODO: 实现从实际监控系统获取指标
- [ ] `./internal\operations\healing\self_healing_system.go:2011` — **TODO** — // TODO 集成Prometheus、InfluxDB等监控系统
- [ ] `./internal\operations\healing\self_healing_system.go:2015` — **TODO** — // TODO: 从监控系统获取API响应时间
- [ ] `./internal\operations\healing\self_healing_system.go:2018` — **TODO** — // TODO: 从监控系统获取错误率
- [ ] `./internal\operations\healing\self_healing_system.go:2021` — **TODO** — // TODO: 从监控系统获取连接成功率
- [ ] `./internal\operations\healing\self_healing_system.go:2024` — **TODO** — // TODO: 从监控系统获取API超时率
- [ ] `./internal\operations\healing\self_healing_system.go:2027` — **TODO** — // TODO: 从监控系统获取CPU使用率
- [ ] `./internal\operations\healing\self_healing_system.go:2030` — **TODO** — // TODO: 从监控系统获取内存使用率
- [ ] `./internal\operations\healing\self_healing_system.go:2033` — **TODO** — // TODO: 从监控系统获取磁盘使用率

## ./internal\operations\routing\smart_exchange_router.go
- [ ] `./internal\operations\routing\smart_exchange_router.go:504` — **TODO** — // TODO: 从配置文件读取路由参数
- [ ] `./internal\operations\routing\smart_exchange_router.go:1081` — **TODO** — // TODO: 实现实际的ping测试
- [ ] `./internal\operations\routing\smart_exchange_router.go:1090` — **TODO** — // TODO: 实现实际的API测试
- [ ] `./internal\operations\routing\smart_exchange_router.go:1099` — **TODO** — // TODO: 实现实际的WebSocket测试
- [ ] `./internal\operations\routing\smart_exchange_router.go:1108` — **TODO** — // TODO: 实现实际的订单簿测试
- [ ] `./internal\operations\routing\smart_exchange_router.go:1564` — **TODO** — // TODO: 实现实际的订单路由执行
- [ ] `./internal\operations\routing\smart_exchange_router.go:1596` — **TODO** — // TODO: 实现可用性统计更新
- [ ] `./internal\operations\routing\smart_exchange_router.go:1608` — **TODO** — // TODO: 基于容量和权重计算理想负载分布
- [ ] `./internal\operations\routing\smart_exchange_router.go:1618` — **TODO** — // TODO: 实现恢复逻辑
- [ ] `./internal\operations\routing\smart_exchange_router.go:1643` — **TODO** — // TODO: 实现实际的故障转移逻辑
- [ ] `./internal\operations\routing\smart_exchange_router.go:1649` — **TODO** — // TODO: 计算当前优化指标
- [ ] `./internal\operations\routing\smart_exchange_router.go:1654` — **TODO** — // TODO: 计算最优路由分布
- [ ] `./internal\operations\routing\smart_exchange_router.go:1659` — **TODO** — // TODO: 应用优化结果
- [ ] `./internal\operations\routing\smart_exchange_router.go:1663` — **TODO** — // TODO: 计算系统负载
- [ ] `./internal\operations\routing\smart_exchange_router.go:1668` — **TODO** — // TODO: 计算订单成功率
- [ ] `./internal\operations\routing\smart_exchange_router.go:1673` — **TODO** — // TODO: 计算路由质量
- [ ] `./internal\operations\routing\smart_exchange_router.go:1704` — **TODO** — // TODO: 计算优化效率
- [ ] `./internal\operations\routing\smart_exchange_router.go:1709` — **TODO** — // TODO: 实现条件评估逻辑

## ./internal\orchestrator\orchestrator.go
- [ ] `./internal\orchestrator\orchestrator.go:306` — **TODO** — // TODO: Process optimization results
- [ ] `./internal\orchestrator\orchestrator.go:313` — **TODO** — // TODO: Handle process exits
- [ ] `./internal\orchestrator\orchestrator.go:320` — **TODO** — // TODO: Forward trade signals to trading service
- [ ] `./internal\orchestrator\orchestrator.go:327` — **TODO** — // TODO: Process market data updates

## ./internal\orchestrator\process_manager.go
- [ ] `./internal\orchestrator\process_manager.go:306` — **TODO** — // TODO: 这里需要与策略守门员集成，检查策略是否在黑名单中

## ./internal\performance\cache.go
- [ ] `./internal\performance\cache.go:531` — **TODO** — // TODO 添加清理逻辑

## ./internal\risk\realtime_monitor.go
- [ ] `./internal\risk\realtime_monitor.go:379` — **TODO** — // TODO: 实现进程停止逻辑
- [ ] `./internal\risk\realtime_monitor.go:413` — **TODO** — // TODO: 实现订单取消逻辑
- [ ] `./internal\risk\realtime_monitor.go:421` — **TODO** — // TODO: 实现缓存清理逻辑

## ./internal\scheduler\task_scheduler.go
- [ ] `./internal\scheduler\task_scheduler.go:525` — **TODO** — // TODO 订阅一些通用事件

## ./internal\security\guardian\account_guardian.go
- [ ] `./internal\security\guardian\account_guardian.go:273` — **TODO** — // TODO: 从配置文件读取安全参数
- [ ] `./internal\security\guardian\account_guardian.go:617` — **TODO** — // TODO: 实现异常登录检测逻辑
- [ ] `./internal\security\guardian\account_guardian.go:622` — **TODO** — // TODO: 实现异常交易检测逻辑
- [ ] `./internal\security\guardian\account_guardian.go:627` — **TODO** — // TODO: 实现设备异常检测逻辑
- [ ] `./internal\security\guardian\account_guardian.go:638` — **TODO** — // TODO: 实现IP地理位置查询
- [ ] `./internal\security\guardian\account_guardian.go:654` — **TODO** — // TODO: 实现设备信息解析
- [ ] `./internal\security\guardian\account_guardian.go:740` — **TODO** — // TODO: 实现具体的行为异常检测逻辑
- [ ] `./internal\security\guardian\account_guardian.go:778` — **TODO** — // TODO: 实现异常升级逻辑
- [ ] `./internal\security\guardian\account_guardian.go:787` — **TODO** — // TODO: 实现账户冻结逻辑
- [ ] `./internal\security\guardian\account_guardian.go:799` — **TODO** — // TODO: 实现待处理响应的处理逻辑
- [ ] `./internal\security\guardian\account_guardian.go:851` — **TODO** — // TODO: 实现基线更新逻辑

## ./internal\security\protector\exchange_provider.go
- [ ] `./internal\security\protector\exchange_provider.go:499` — **mock** — // MockExchangeProvider 模拟交易所数据提供者（用于测试）
- [ ] `./internal\security\protector\exchange_provider.go:500` — **mock** — type MockExchangeProvider struct {
- [ ] `./internal\security\protector\exchange_provider.go:506` — **mock** — // NewMockExchangeProvider 创建模拟交易所数据提供者
- [ ] `./internal\security\protector\exchange_provider.go:507` — **mock** — func NewMockExchangeProvider() *MockExchangeProvider {
- [ ] `./internal\security\protector\exchange_provider.go:508` — **mock** — return &MockExchangeProvider{
- [ ] `./internal\security\protector\exchange_provider.go:536` — **mock** — func (m *MockExchangeProvider) IsHealthy() bool {
- [ ] `./internal\security\protector\exchange_provider.go:541` — **mock** — func (m *MockExchangeProvider) GetFundData(ctx context.Context) (*ExchangeFundData, error) {
- [ ] `./internal\security\protector\exchange_provider.go:549` — **mock** — func (m *MockExchangeProvider) GetPositions(ctx context.Context) ([]*Position, error) {
- [ ] `./internal\security\protector\exchange_provider.go:557` — **mock** — func (m *MockExchangeProvider) GetHistoricalReturns(ctx context.Context, days int) ([]float64, error) {
- [ ] `./internal\security\protector\exchange_provider.go:572` — **mock** — func (m *MockExchangeProvider) GetHistoricalEquity(ctx context.Context, days int) ([]float64, error) {
- [ ] `./internal\security\protector\exchange_provider.go:590` — **mock** — func (m *MockExchangeProvider) GetSymbolPrice(ctx context.Context, symbol string) (float64, error) {
- [ ] `./internal\security\protector\exchange_provider.go:610` — **mock** — func (m *MockExchangeProvider) GetOrderBookDepth(ctx context.Context, symbol string) (*OrderBookDepth, error) {
- [ ] `./internal\security\protector\exchange_provider.go:635` — **mock** — func (m *MockExchangeProvider) GetTradingVolume(ctx context.Context, symbol string, period string) (float64, error) {
- [ ] `./internal\security\protector\exchange_provider.go:645` — **mock** — func (m *MockExchangeProvider) SetHealthy(healthy bool) {
- [ ] `./internal\security\protector\exchange_provider.go:650` — **mock** — func (m *MockExchangeProvider) SetFundData(data *ExchangeFundData) {
- [ ] `./internal\security\protector\exchange_provider.go:655` — **mock** — func (m *MockExchangeProvider) SetPositions(positions []*Position) {

## ./internal\security\protector\fund_protector_test.go
- [ ] `./internal\security\protector\fund_protector_test.go:21` — **mock** — exchangeProvider := NewMockExchangeProvider()
- [ ] `./internal\security\protector\fund_protector_test.go:22` — **mock** — notificationService := NewMockNotificationService()
- [ ] `./internal\security\protector\fund_protector_test.go:23` — **mock** — walletService := NewMockWalletService()
- [ ] `./internal\security\protector\fund_protector_test.go:45` — **mock** — provider := NewMockExchangeProvider()
- [ ] `./internal\security\protector\fund_protector_test.go:49` — **mock** — t.Error("Mock provider should be healthy")
- [ ] `./internal\security\protector\fund_protector_test.go:90` — **mock** — service := NewMockNotificationService()
- [ ] `./internal\security\protector\fund_protector_test.go:137` — **mock** — service := NewMockWalletService()
- [ ] `./internal\security\protector\fund_protector_test.go:199` — **mock** — exchangeProvider := NewMockExchangeProvider()
- [ ] `./internal\security\protector\fund_protector_test.go:200` — **mock** — notificationService := NewMockNotificationService()
- [ ] `./internal\security\protector\fund_protector_test.go:201` — **mock** — walletService := NewMockWalletService()
- [ ] `./internal\security\protector\fund_protector_test.go:244` — **mock** — exchangeProvider := NewMockExchangeProvider()
- [ ] `./internal\security\protector\fund_protector_test.go:245` — **mock** — notificationService := NewMockNotificationService()
- [ ] `./internal\security\protector\fund_protector_test.go:246` — **mock** — walletService := NewMockWalletService()
- [ ] `./internal\security\protector\fund_protector_test.go:285` — **mock** — emails := notificationService.(*MockNotificationService).GetEmailsSent()
- [ ] `./internal\security\protector\fund_protector_test.go:290` — **mock** — sms := notificationService.(*MockNotificationService).GetSMSSent()
- [ ] `./internal\security\protector\fund_protector_test.go:305` — **mock** — exchangeProvider := NewMockExchangeProvider()
- [ ] `./internal\security\protector\fund_protector_test.go:306` — **mock** — notificationService := NewMockNotificationService()
- [ ] `./internal\security\protector\fund_protector_test.go:307` — **mock** — walletService := NewMockWalletService()
- [ ] `./internal\security\protector\fund_protector_test.go:345` — **mock** — exchangeProvider := NewMockExchangeProvider()
- [ ] `./internal\security\protector\fund_protector_test.go:346` — **mock** — notificationService := NewMockNotificationService()
- [ ] `./internal\security\protector\fund_protector_test.go:347` — **mock** — walletService := NewMockWalletService()
- [ ] `./internal\security\protector\fund_protector_test.go:365` — **mock** — transfers := walletService.(*MockWalletService).GetTransfers()
- [ ] `./internal\security\protector\fund_protector_test.go:385` — **mock** — exchangeProvider := NewMockExchangeProvider()
- [ ] `./internal\security\protector\fund_protector_test.go:386` — **mock** — notificationService := NewMockNotificationService()
- [ ] `./internal\security\protector\fund_protector_test.go:387` — **mock** — walletService := NewMockWalletService()
- [ ] `./internal\security\protector\fund_protector_test.go:428` — **mock** — exchangeProvider := NewMockExchangeProvider()
- [ ] `./internal\security\protector\fund_protector_test.go:429` — **mock** — notificationService := NewMockNotificationService()
- [ ] `./internal\security\protector\fund_protector_test.go:430` — **mock** — walletService := NewMockWalletService()

## ./internal\security\protector\notification_service.go
- [ ] `./internal\security\protector\notification_service.go:256` — **mock** — // MockNotificationService 模拟通知服务（用于测试）
- [ ] `./internal\security\protector\notification_service.go:257` — **mock** — type MockNotificationService struct {
- [ ] `./internal\security\protector\notification_service.go:294` — **mock** — // NewMockNotificationService 创建模拟通知服务
- [ ] `./internal\security\protector\notification_service.go:295` — **mock** — func NewMockNotificationService() *MockNotificationService {
- [ ] `./internal\security\protector\notification_service.go:296` — **mock** — return &MockNotificationService{
- [ ] `./internal\security\protector\notification_service.go:306` — **mock** — func (m *MockNotificationService) SendEmail(ctx context.Context, to, subject, body string) error {
- [ ] `./internal\security\protector\notification_service.go:308` — **mock** — return fmt.Errorf("mock email failure")
- [ ] `./internal\security\protector\notification_service.go:318` — **mock** — log.Printf("Mock email sent to %s: %s", to, subject)
- [ ] `./internal\security\protector\notification_service.go:323` — **mock** — func (m *MockNotificationService) SendSMS(ctx context.Context, phone, message string) error {
- [ ] `./internal\security\protector\notification_service.go:325` — **mock** — return fmt.Errorf("mock SMS failure")
- [ ] `./internal\security\protector\notification_service.go:334` — **mock** — log.Printf("Mock SMS sent to %s: %s", phone, message)
- [ ] `./internal\security\protector\notification_service.go:339` — **mock** — func (m *MockNotificationService) SendWebhook(ctx context.Context, url string, payload interface{}) error {
- [ ] `./internal\security\protector\notification_service.go:341` — **mock** — return fmt.Errorf("mock webhook failure")
- [ ] `./internal\security\protector\notification_service.go:350` — **mock** — log.Printf("Mock webhook sent to %s", url)
- [ ] `./internal\security\protector\notification_service.go:355` — **mock** — func (m *MockNotificationService) SendSlack(ctx context.Context, webhook, message string) error {
- [ ] `./internal\security\protector\notification_service.go:357` — **mock** — return fmt.Errorf("mock Slack failure")
- [ ] `./internal\security\protector\notification_service.go:366` — **mock** — log.Printf("Mock Slack message sent: %s", message)
- [ ] `./internal\security\protector\notification_service.go:371` — **mock** — func (m *MockNotificationService) SetShouldFail(shouldFail bool) {
- [ ] `./internal\security\protector\notification_service.go:376` — **mock** — func (m *MockNotificationService) GetEmailsSent() []EmailRecord {
- [ ] `./internal\security\protector\notification_service.go:381` — **mock** — func (m *MockNotificationService) GetSMSSent() []SMSRecord {
- [ ] `./internal\security\protector\notification_service.go:386` — **mock** — func (m *MockNotificationService) GetWebhooksSent() []WebhookRecord {
- [ ] `./internal\security\protector\notification_service.go:391` — **mock** — func (m *MockNotificationService) GetSlackSent() []SlackRecord {
- [ ] `./internal\security\protector\notification_service.go:396` — **mock** — func (m *MockNotificationService) Reset() {

## ./internal\security\protector\trading_operations.go
- [ ] `./internal\security\protector\trading_operations.go:57` — **TODO** — // TODO 将事件记录到数据库或发送到监控系统

## ./internal\security\protector\wallet_service.go
- [ ] `./internal\security\protector\wallet_service.go:351` — **mock** — // MockWalletService 模拟钱包服务（用于测试）
- [ ] `./internal\security\protector\wallet_service.go:352` — **mock** — type MockWalletService struct {
- [ ] `./internal\security\protector\wallet_service.go:358` — **mock** — // NewMockWalletService 创建模拟钱包服务
- [ ] `./internal\security\protector\wallet_service.go:359` — **mock** — func NewMockWalletService() *MockWalletService {
- [ ] `./internal\security\protector\wallet_service.go:360` — **mock** — return &MockWalletService{
- [ ] `./internal\security\protector\wallet_service.go:368` — **mock** — func (m *MockWalletService) InitiateTransfer(ctx context.Context, request *TransferRequest) (*TransferResponse, error) {
- [ ] `./internal\security\protector\wallet_service.go:370` — **mock** — return nil, fmt.Errorf("mock transfer failure")
- [ ] `./internal\security\protector\wallet_service.go:377` — **mock** — return nil, fmt.Errorf("random mock transfer failure")
- [ ] `./internal\security\protector\wallet_service.go:381` — **mock** — transferID := fmt.Sprintf("MOCK_TXF_%d", time.Now().Unix())
- [ ] `./internal\security\protector\wallet_service.go:405` — **mock** — log.Printf("Mock transfer initiated: %s", transferID)
- [ ] `./internal\security\protector\wallet_service.go:410` — **mock** — func (m *MockWalletService) GetTransferStatus(ctx context.Context, transferID string) (*TransferStatus, error) {
- [ ] `./internal\security\protector\wallet_service.go:419` — **mock** — func (m *MockWalletService) CancelTransfer(ctx context.Context, transferID string) error {
- [ ] `./internal\security\protector\wallet_service.go:429` — **mock** — func (m *MockWalletService) GetTransferHistory(ctx context.Context, limit int) ([]*TransferRecord, error) {
- [ ] `./internal\security\protector\wallet_service.go:445` — **mock** — func (m *MockWalletService) ValidateAddress(address string) error {
- [ ] `./internal\security\protector\wallet_service.go:453` — **mock** — func (m *MockWalletService) EstimateTransferFee(ctx context.Context, amount float64, toAddress string) (float64, error) {
- [ ] `./internal\security\protector\wallet_service.go:458` — **mock** — func (m *MockWalletService) SetShouldFail(shouldFail bool) {
- [ ] `./internal\security\protector\wallet_service.go:463` — **mock** — func (m *MockWalletService) SetFailureRate(rate float64) {
- [ ] `./internal\security\protector\wallet_service.go:468` — **mock** — func (m *MockWalletService) GetTransfers() map[string]*TransferStatus {

## ./internal\stability\connection_pool.go
- [ ] `./internal\stability\connection_pool.go:313` — **TODO** — // TODO 添加重启连接池的逻辑

## ./internal\stability\process_manager.go
- [ ] `./internal\stability\process_manager.go:432` — **TODO** — // TODO 实现一个简单的价格缓存机制
- [ ] `./internal\stability\process_manager.go:533` — **TODO** — // TODO 启动定时器、初始化指标等
- [ ] `./internal\stability\process_manager.go:543` — **TODO** — // TODO 停止定时器、清理资源等

## ./internal\stability\redis_fallback.go
- [ ] `./internal\stability\redis_fallback.go:364` — **TODO** — // TODO 添加模式切换的通知逻辑

## ./internal\strategy\unified_service.go
- [ ] `./internal\strategy\unified_service.go:313` — **TODO** — // TODO: 解析JSON配置
- [ ] `./internal\strategy\unified_service.go:322` — **TODO** — // TODO: 添加相同的过滤条件
- [ ] `./internal\strategy\unified_service.go:326` — **mock** — strategies = s.getMockStrategies()
- [ ] `./internal\strategy\unified_service.go:372` — **TODO** — // TODO: 解析JSON配置
- [ ] `./internal\strategy\unified_service.go:377` — **mock** — mockStrategies := s.getMockStrategies()
- [ ] `./internal\strategy\unified_service.go:378` — **mock** — for _, mock := range mockStrategies {
- [ ] `./internal\strategy\unified_service.go:379` — **mock** — if mock.ID == strategyID {
- [ ] `./internal\strategy\unified_service.go:380` — **mock** — strategy = mock
- [ ] `./internal\strategy\unified_service.go:402` — **TODO** — ExecutionCount: 100, // TODO: 从实际数据获取
- [ ] `./internal\strategy\unified_service.go:435` — **mock** — unified.Execution = s.getMockExecutionInfo()
- [ ] `./internal\strategy\unified_service.go:436` — **mock** — unified.Performance = s.getMockPerformanceInfo()
- [ ] `./internal\strategy\unified_service.go:437` — **mock** — unified.Pool = s.getMockPoolInfo()
- [ ] `./internal\strategy\unified_service.go:526` — **mock** — // getMockStrategies 获取模拟策略数据
- [ ] `./internal\strategy\unified_service.go:527` — **mock** — func (s *UnifiedStrategyService) getMockStrategies() []BasicStrategy {
- [ ] `./internal\strategy\unified_service.go:568` — **mock** — // getMockExecutionInfo 获取模拟执行信息
- [ ] `./internal\strategy\unified_service.go:569` — **mock** — func (s *UnifiedStrategyService) getMockExecutionInfo() ExecutionInfo {
- [ ] `./internal\strategy\unified_service.go:582` — **mock** — // getMockPerformanceInfo 获取模拟性能信息
- [ ] `./internal\strategy\unified_service.go:583` — **mock** — func (s *UnifiedStrategyService) getMockPerformanceInfo() PerformanceInfo {
- [ ] `./internal\strategy\unified_service.go:599` — **mock** — // getMockPoolInfo 获取模拟池信息
- [ ] `./internal\strategy\unified_service.go:600` — **mock** — func (s *UnifiedStrategyService) getMockPoolInfo() PoolInfo {

## ./internal\strategy\unified_service_simple.go
- [ ] `./internal\strategy\unified_service_simple.go:45` — **mock** — strategies := s.getMockStrategies()
- [ ] `./internal\strategy\unified_service_simple.go:82` — **mock** — strategies := s.getMockStrategies()
- [ ] `./internal\strategy\unified_service_simple.go:155` — **mock** — Execution: s.getMockExecutionInfo(),
- [ ] `./internal\strategy\unified_service_simple.go:156` — **mock** — Performance: s.getMockPerformanceInfo(),
- [ ] `./internal\strategy\unified_service_simple.go:157` — **mock** — Pool: s.getMockPoolInfo(),
- [ ] `./internal\strategy\unified_service_simple.go:220` — **mock** — // getMockStrategies 获取模拟策略数据
- [ ] `./internal\strategy\unified_service_simple.go:221` — **mock** — func (s *SimpleUnifiedStrategyService) getMockStrategies() []BasicStrategy {
- [ ] `./internal\strategy\unified_service_simple.go:262` — **mock** — // getMockExecutionInfo 获取模拟执行信息
- [ ] `./internal\strategy\unified_service_simple.go:263` — **mock** — func (s *SimpleUnifiedStrategyService) getMockExecutionInfo() ExecutionInfo {
- [ ] `./internal\strategy\unified_service_simple.go:276` — **mock** — // getMockPerformanceInfo 获取模拟性能信息
- [ ] `./internal\strategy\unified_service_simple.go:277` — **mock** — func (s *SimpleUnifiedStrategyService) getMockPerformanceInfo() PerformanceInfo {
- [ ] `./internal\strategy\unified_service_simple.go:293` — **mock** — // getMockPoolInfo 获取模拟池信息
- [ ] `./internal\strategy\unified_service_simple.go:294` — **mock** — func (s *SimpleUnifiedStrategyService) getMockPoolInfo() PoolInfo {

## ./internal\strategy\generator\analyzer.go
- [ ] `./internal\strategy\generator\analyzer.go:390` — **mock** — // generateMockPriceData 生成模拟价格数据
- [ ] `./internal\strategy\generator\analyzer.go:391` — **mock** — func (ma *MarketAnalyzer) generateMockPriceData(symbol string, timeRange time.Duration, startTime, endTime time.Time) []PricePoint {

## ./internal\strategy\generator\analyzer_test.go
- [ ] `./internal\strategy\generator\analyzer_test.go:69` — **mock** — func TestMarketAnalyzer_GenerateMockPriceData(t *testing.T) {
- [ ] `./internal\strategy\generator\analyzer_test.go:77` — **mock** — priceData := analyzer.generateMockPriceData(symbol, timeRange, startTime, endTime)

## ./internal\strategy\onboarding\validator.go
- [ ] `./internal\strategy\onboarding\validator.go:335` — **TODO** — // TODO 添加性能相关的验证

## ./internal\strategy\optimizer\dynamic_stoploss.go
- [ ] `./internal\strategy\optimizer\dynamic_stoploss.go:771` — **mock** — log.Printf("Mock: Updating orders on exchange for %s_%s: SL=%.4f, TP=%.4f",

## ./internal\strategy\optimizer\optimizer_test.go
- [ ] `./internal\strategy\optimizer\optimizer_test.go:16` — **mock** — mockData := testutils.NewMockData()
- [ ] `./internal\strategy\optimizer\optimizer_test.go:51` — **mock** — historicalData := generateMockHistoricalData(1000)
- [ ] `./internal\strategy\optimizer\optimizer_test.go:117` — **mock** — Returns:        generateMockReturns(150),
- [ ] `./internal\strategy\optimizer\optimizer_test.go:225` — **mock** — historicalData := generateMockHistoricalData(500)
- [ ] `./internal\strategy\optimizer\optimizer_test.go:240` — **mock** — func generateMockHistoricalData(count int) []MarketData {
- [ ] `./internal\strategy\optimizer\optimizer_test.go:241` — **mock** — mockData := testutils.NewMockData()
- [ ] `./internal\strategy\optimizer\optimizer_test.go:247` — **mock** — change := mockData.RandomFloat(-0.05, 0.05)
- [ ] `./internal\strategy\optimizer\optimizer_test.go:252` — **mock** — Open:      basePrice * (1 + mockData.RandomFloat(-0.01, 0.01)),
- [ ] `./internal\strategy\optimizer\optimizer_test.go:253` — **mock** — High:      basePrice * (1 + mockData.RandomFloat(0, 0.02)),
- [ ] `./internal\strategy\optimizer\optimizer_test.go:254` — **mock** — Low:       basePrice * (1 + mockData.RandomFloat(-0.02, 0)),
- [ ] `./internal\strategy\optimizer\optimizer_test.go:256` — **mock** — Volume:    mockData.RandomFloat(100, 1000),
- [ ] `./internal\strategy\optimizer\optimizer_test.go:263` — **mock** — func generateMockReturns(count int) []float64 {
- [ ] `./internal\strategy\optimizer\optimizer_test.go:264` — **mock** — mockData := testutils.NewMockData()
- [ ] `./internal\strategy\optimizer\optimizer_test.go:269` — **mock** — returns[i] = mockData.RandomFloat(-0.05, 0.05)

## ./internal\strategy\optimizer\orchestrator_test.go
- [ ] `./internal\strategy\optimizer\orchestrator_test.go:143` — **mock** — // Create a mock orchestrator without database connection

## ./internal\strategy\sandbox\automation.go
- [ ] `./internal\strategy\sandbox\automation.go:153` — **mock** — mockExchange := s.createMockExchange(testConfig)
- [ ] `./internal\strategy\sandbox\automation.go:164` — **mock** — sandbox := NewSandbox(strategyInstance, strategyConfig.Params, mockExchange)
- [ ] `./internal\strategy\sandbox\automation.go:463` — **mock** — // createMockExchange 创建模拟交易所
- [ ] `./internal\strategy\sandbox\automation.go:464` — **mock** — func (s *AutomatedSandboxService) createMockExchange(config *TestConfiguration) exchange.Exchange {
- [ ] `./internal\strategy\sandbox\automation.go:466` — **mock** — // 为了演示，返回nil，实际应该实现MockExchange
- [ ] `./internal\strategy\sandbox\automation.go:467` — **mock** — log.Printf("Creating mock exchange for %s on %s", config.Symbol, config.Exchange)

## ./internal\strategy\sandbox\sandbox.go
- [ ] `./internal\strategy\sandbox\sandbox.go:245` — **TODO** — // 新增：TODO 实现重连逻辑
- [ ] `./internal\strategy\sandbox\sandbox.go:249` — **TODO** — // 新增：TODO 实现等待和重试逻辑
- [ ] `./internal\strategy\sandbox\sandbox.go:253` — **TODO** — // 新增：TODO 实现停止交易逻辑

## ./internal\strategy\validation\strategy_gatekeeper.go
- [ ] `./internal\strategy\validation\strategy_gatekeeper.go:350` — **TODO** — // TODO: 实现实际的策略停止逻辑

## ./internal\testing\enhanced_stress_test.go
- [ ] `./internal\testing\enhanced_stress_test.go:682` — **TODO** — // TODO 获取CPU使用率等指标，暂时跳过

## ./internal\testutils\benchmark.go
- [ ] `./internal\testutils\benchmark.go:156` — **TODO** — // TODO 实现负载测试逻辑

## ./internal\testutils\testutils.go
- [ ] `./internal\testutils\testutils.go:94` — **mock** — suite.setupMockDB()
- [ ] `./internal\testutils\testutils.go:101` — **mock** — suite.setupMockCache()
- [ ] `./internal\testutils\testutils.go:121` — **TODO** — // TODO 连接到测试数据库
- [ ] `./internal\testutils\testutils.go:123` — **mock** — s.setupMockDB()
- [ ] `./internal\testutils\testutils.go:126` — **mock** — // setupMockDB 设置模拟数据库
- [ ] `./internal\testutils\testutils.go:127` — **mock** — func (s *TestSuite) setupMockDB() {
- [ ] `./internal\testutils\testutils.go:142` — **TODO** — // TODO 连接到测试Redis
- [ ] `./internal\testutils\testutils.go:144` — **mock** — s.setupMockCache()
- [ ] `./internal\testutils\testutils.go:147` — **mock** — // setupMockCache 设置模拟缓存
- [ ] `./internal\testutils\testutils.go:148` — **mock** — func (s *TestSuite) setupMockCache() {
- [ ] `./internal\testutils\testutils.go:275` — **mock** — // MockData 模拟数据生成器
- [ ] `./internal\testutils\testutils.go:276` — **mock** — type MockData struct {
- [ ] `./internal\testutils\testutils.go:280` — **mock** — // NewMockData 创建模拟数据生成器
- [ ] `./internal\testutils\testutils.go:281` — **mock** — func NewMockData() *MockData {
- [ ] `./internal\testutils\testutils.go:282` — **mock** — return &MockData{
- [ ] `./internal\testutils\testutils.go:288` — **mock** — func (m *MockData) RandomString(length int) string {
- [ ] `./internal\testutils\testutils.go:298` — **mock** — func (m *MockData) RandomInt(min, max int) int {
- [ ] `./internal\testutils\testutils.go:303` — **mock** — func (m *MockData) RandomFloat(min, max float64) float64 {
- [ ] `./internal\testutils\testutils.go:308` — **mock** — func (m *MockData) RandomBool() bool {
- [ ] `./internal\testutils\testutils.go:313` — **mock** — func (m *MockData) RandomChoice(choices []string) string {
- [ ] `./internal\testutils\testutils.go:318` — **mock** — func (m *MockData) GenerateStrategy() map[string]interface{} {
- [ ] `./internal\testutils\testutils.go:345` — **mock** — func (m *MockData) GenerateOrder() map[string]interface{} {
- [ ] `./internal\testutils\testutils.go:555` — **TODO** — // TODO 实现端口检查逻辑

## ./internal\trading\dryrun\simulator_test.go
- [ ] `./internal\trading\dryrun\simulator_test.go:11` — **mock** — // MockMarketDataProvider 模拟市场数据提供者
- [ ] `./internal\trading\dryrun\simulator_test.go:12` — **mock** — type MockMarketDataProvider struct{}
- [ ] `./internal\trading\dryrun\simulator_test.go:14` — **mock** — func (m *MockMarketDataProvider) GetPrice(symbol string) (float64, error) {
- [ ] `./internal\trading\dryrun\simulator_test.go:61` — **mock** — mockProvider := &MockMarketDataProvider{}
- [ ] `./internal\trading\dryrun\simulator_test.go:63` — **mock** — simulator, err := NewTradingSimulator(config, mockProvider)
- [ ] `./internal\trading\dryrun\simulator_test.go:117` — **mock** — mockProvider := &MockMarketDataProvider{}
- [ ] `./internal\trading\dryrun\simulator_test.go:119` — **mock** — simulator, err := NewTradingSimulator(config, mockProvider)
- [ ] `./internal\trading\dryrun\simulator_test.go:154` — **mock** — mockProvider := &MockMarketDataProvider{}
- [ ] `./internal\trading\dryrun\simulator_test.go:156` — **mock** — simulator, err := NewTradingSimulator(config, mockProvider)
- [ ] `./internal\trading\dryrun\simulator_test.go:225` — **mock** — mockProvider := &MockMarketDataProvider{}
- [ ] `./internal\trading\dryrun\simulator_test.go:227` — **mock** — simulator, err := NewTradingSimulator(config, mockProvider)
- [ ] `./internal\trading\dryrun\simulator_test.go:288` — **mock** — mockProvider := &MockMarketDataProvider{}
- [ ] `./internal\trading\dryrun\simulator_test.go:290` — **mock** — simulator, err := NewTradingSimulator(config, mockProvider)
- [ ] `./internal\trading\dryrun\simulator_test.go:322` — **mock** — mockProvider := &MockMarketDataProvider{}
- [ ] `./internal\trading\dryrun\simulator_test.go:324` — **mock** — simulator, err := NewTradingSimulator(config, mockProvider)

## ./internal\workflow\executors.go
- [ ] `./internal\workflow\executors.go:27` — **mock** — // MockExecutor 模拟执行器（用于测试）
- [ ] `./internal\workflow\executors.go:28` — **mock** — type MockExecutor struct {
- [ ] `./internal\workflow\executors.go:34` — **mock** — // NewMockExecutor 创建模拟执行器
- [ ] `./internal\workflow\executors.go:35` — **mock** — func NewMockExecutor(name string, executionTime time.Duration, simulateFailure bool) *MockExecutor {
- [ ] `./internal\workflow\executors.go:36` — **mock** — return &MockExecutor{
- [ ] `./internal\workflow\executors.go:51` — **mock** — func (me *MockExecutor) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
- [ ] `./internal\workflow\executors.go:76` — **mock** — "mock_data": map[string]interface{}{
- [ ] `./internal\workflow\executors.go:349` — **mock** — executors[id] = NewMockExecutor(name, execTime, false)

# 开发修复清单（来自 report.md）

## ./internal\api\api_test.go
- [ ] `./internal\api\api_test.go:79` — **mock** — // Create test strategy data directly instead of using mock generator

## ./internal\automation\scheduler\strategy_scheduler.go
- [ ] `./internal\automation\scheduler\strategy_scheduler.go:4403` — **TODO** — // TODO: Implement real exchange client integration

## ./internal\automation\scheduler\risk\risk_controller_test.go
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:37` — **mock** — action, err := testRC.triggerPositionReductionMocked(ctx, marginStatus, 0.3)
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:57` — **mock** — action, err := testRC.triggerEmergencyStopMocked(ctx, reason)
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:82` — **mock** — action, err := testRC.triggerLeverageReductionMocked(ctx, targetLeverage)
- [ ] `./internal\automation\scheduler\risk\risk_controller_test.go:203` — **mock** — _, err := testRC.triggerEmergencyStopMocked(ctx, "Test emergency")

## ./internal\automation\scheduler\risk\risk_reporter.go
- [ ] `./internal\automation\scheduler\risk\risk_reporter.go:671` — **TODO** — // TODO: Query the database for actual daily metrics

## ./internal\learning\automl\automl_engine.go
- [ ] `./internal\learning\automl\automl_engine.go:2423` — **mock** — func (engine *AutoMLEngine) generateMockHyperparameters(modelType string) map[string]interface{} {

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

## ./internal\security\guardian\account_guardian.go
- [ ] `./internal\security\guardian\account_guardian.go:1326` — **TODO** — // TODO: 实现待处理响应的处理逻辑

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

## ./internal\strategy\unified_service.go
- [ ] `./internal\strategy\unified_service.go:366` — **mock** — strategies = s.getMockStrategies()
- [ ] `./internal\strategy\unified_service.go:412` — **TODO** — // TODO: 解析JSON配置
- [ ] `./internal\strategy\unified_service.go:417` — **mock** — mockStrategies := s.getMockStrategies()
- [ ] `./internal\strategy\unified_service.go:418` — **mock** — for _, mock := range mockStrategies {
- [ ] `./internal\strategy\unified_service.go:419` — **mock** — if mock.ID == strategyID {
- [ ] `./internal\strategy\unified_service.go:420` — **mock** — strategy = mock
- [ ] `./internal\strategy\unified_service.go:442` — **TODO** — ExecutionCount: 100, // TODO: 从实际数据获取
- [ ] `./internal\strategy\unified_service.go:475` — **mock** — unified.Execution = s.getMockExecutionInfo()
- [ ] `./internal\strategy\unified_service.go:476` — **mock** — unified.Performance = s.getMockPerformanceInfo()
- [ ] `./internal\strategy\unified_service.go:477` — **mock** — unified.Pool = s.getMockPoolInfo()
- [ ] `./internal\strategy\unified_service.go:566` — **mock** — // getMockStrategies 获取模拟策略数据
- [ ] `./internal\strategy\unified_service.go:567` — **mock** — func (s *UnifiedStrategyService) getMockStrategies() []BasicStrategy {
- [ ] `./internal\strategy\unified_service.go:608` — **mock** — // getMockExecutionInfo 获取模拟执行信息
- [ ] `./internal\strategy\unified_service.go:609` — **mock** — func (s *UnifiedStrategyService) getMockExecutionInfo() ExecutionInfo {
- [ ] `./internal\strategy\unified_service.go:622` — **mock** — // getMockPerformanceInfo 获取模拟性能信息
- [ ] `./internal\strategy\unified_service.go:623` — **mock** — func (s *UnifiedStrategyService) getMockPerformanceInfo() PerformanceInfo {
- [ ] `./internal\strategy\unified_service.go:639` — **mock** — // getMockPoolInfo 获取模拟池信息
- [ ] `./internal\strategy\unified_service.go:640` — **mock** — func (s *UnifiedStrategyService) getMockPoolInfo() PoolInfo {

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
- [ ] `./internal\strategy\sandbox\automation.go:154` — **mock** — mockExchange := s.createMockExchange(testConfig)
- [ ] `./internal\strategy\sandbox\automation.go:165` — **mock** — sandbox := NewSandbox(strategyInstance, strategyConfig.Params, mockExchange)
- [ ] `./internal\strategy\sandbox\automation.go:464` — **mock** — // createMockExchange 创建模拟交易所
- [ ] `./internal\strategy\sandbox\automation.go:465` — **mock** — func (s *AutomatedSandboxService) createMockExchange(config *TestConfiguration) exchange.Exchange {

## ./internal\testutils\testutils.go
- [ ] `./internal\testutils\testutils.go:98` — **mock** — suite.setupMockDB()
- [ ] `./internal\testutils\testutils.go:105` — **mock** — suite.setupMockCache()
- [ ] `./internal\testutils\testutils.go:131` — **mock** — s.setupMockDB()
- [ ] `./internal\testutils\testutils.go:138` — **mock** — s.setupMockDB()
- [ ] `./internal\testutils\testutils.go:146` — **mock** — s.setupMockDB()
- [ ] `./internal\testutils\testutils.go:158` — **mock** — // setupMockDB 设置模拟数据库
- [ ] `./internal\testutils\testutils.go:159` — **mock** — func (s *TestSuite) setupMockDB() {
- [ ] `./internal\testutils\testutils.go:201` — **mock** — s.setupMockCache()
- [ ] `./internal\testutils\testutils.go:209` — **mock** — s.setupMockCache()
- [ ] `./internal\testutils\testutils.go:221` — **mock** — // setupMockCache 设置模拟缓存
- [ ] `./internal\testutils\testutils.go:222` — **mock** — func (s *TestSuite) setupMockCache() {
- [ ] `./internal\testutils\testutils.go:349` — **mock** — // MockData 模拟数据生成器
- [ ] `./internal\testutils\testutils.go:350` — **mock** — type MockData struct {
- [ ] `./internal\testutils\testutils.go:354` — **mock** — // NewMockData 创建模拟数据生成器
- [ ] `./internal\testutils\testutils.go:355` — **mock** — func NewMockData() *MockData {
- [ ] `./internal\testutils\testutils.go:356` — **mock** — return &MockData{
- [ ] `./internal\testutils\testutils.go:362` — **mock** — func (m *MockData) RandomString(length int) string {
- [ ] `./internal\testutils\testutils.go:372` — **mock** — func (m *MockData) RandomInt(min, max int) int {
- [ ] `./internal\testutils\testutils.go:377` — **mock** — func (m *MockData) RandomFloat(min, max float64) float64 {
- [ ] `./internal\testutils\testutils.go:382` — **mock** — func (m *MockData) RandomBool() bool {
- [ ] `./internal\testutils\testutils.go:387` — **mock** — func (m *MockData) RandomChoice(choices []string) string {
- [ ] `./internal\testutils\testutils.go:392` — **mock** — func (m *MockData) GenerateStrategy() map[string]interface{} {
- [ ] `./internal\testutils\testutils.go:419` — **mock** — func (m *MockData) GenerateOrder() map[string]interface{} {

## ./internal\workflow\executors.go
- [ ] `./internal\workflow\executors.go:27` — **mock** — // MockExecutor 模拟执行器（用于测试）
- [ ] `./internal\workflow\executors.go:28` — **mock** — type MockExecutor struct {
- [ ] `./internal\workflow\executors.go:34` — **mock** — // NewMockExecutor 创建模拟执行器
- [ ] `./internal\workflow\executors.go:35` — **mock** — func NewMockExecutor(name string, executionTime time.Duration, simulateFailure bool) *MockExecutor {
- [ ] `./internal\workflow\executors.go:36` — **mock** — return &MockExecutor{
- [ ] `./internal\workflow\executors.go:51` — **mock** — func (me *MockExecutor) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
- [ ] `./internal\workflow\executors.go:76` — **mock** — "mock_data": map[string]interface{}{
- [ ] `./internal\workflow\executors.go:349` — **mock** — executors[id] = NewMockExecutor(name, execTime, false)


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

## ./internal\testutils\testutils.go
- [ ] `./internal\testutils\testutils.go:96` — **mock** — suite.setupMockDB()
- [ ] `./internal\testutils\testutils.go:103` — **mock** — suite.setupMockCache()
- [ ] `./internal\testutils\testutils.go:123` — **TODO** — // TODO 连接到测试数据库
- [ ] `./internal\testutils\testutils.go:125` — **mock** — s.setupMockDB()
- [ ] `./internal\testutils\testutils.go:128` — **mock** — // setupMockDB 设置模拟数据库
- [ ] `./internal\testutils\testutils.go:129` — **mock** — func (s *TestSuite) setupMockDB() {
- [ ] `./internal\testutils\testutils.go:144` — **TODO** — // TODO 连接到测试Redis
- [ ] `./internal\testutils\testutils.go:146` — **mock** — s.setupMockCache()
- [ ] `./internal\testutils\testutils.go:149` — **mock** — // setupMockCache 设置模拟缓存
- [ ] `./internal\testutils\testutils.go:150` — **mock** — func (s *TestSuite) setupMockCache() {
- [ ] `./internal\testutils\testutils.go:277` — **mock** — // MockData 模拟数据生成器
- [ ] `./internal\testutils\testutils.go:278` — **mock** — type MockData struct {
- [ ] `./internal\testutils\testutils.go:282` — **mock** — // NewMockData 创建模拟数据生成器
- [ ] `./internal\testutils\testutils.go:283` — **mock** — func NewMockData() *MockData {
- [ ] `./internal\testutils\testutils.go:284` — **mock** — return &MockData{
- [ ] `./internal\testutils\testutils.go:290` — **mock** — func (m *MockData) RandomString(length int) string {
- [ ] `./internal\testutils\testutils.go:300` — **mock** — func (m *MockData) RandomInt(min, max int) int {
- [ ] `./internal\testutils\testutils.go:305` — **mock** — func (m *MockData) RandomFloat(min, max float64) float64 {
- [ ] `./internal\testutils\testutils.go:310` — **mock** — func (m *MockData) RandomBool() bool {
- [ ] `./internal\testutils\testutils.go:315` — **mock** — func (m *MockData) RandomChoice(choices []string) string {
- [ ] `./internal\testutils\testutils.go:320` — **mock** — func (m *MockData) GenerateStrategy() map[string]interface{} {
- [ ] `./internal\testutils\testutils.go:347` — **mock** — func (m *MockData) GenerateOrder() map[string]interface{} {

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

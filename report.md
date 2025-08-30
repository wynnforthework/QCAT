# Code Scan Report

## ./internal\api\api_test.go
- Line 79: **mock** → `// Create test strategy data directly instead of using mock generator`

## ./internal\automation\scheduler\risk\risk_controller_test.go
- Line 37: **mock** → `action, err := testRC.triggerPositionReductionMocked(ctx, marginStatus, 0.3)`
- Line 57: **mock** → `action, err := testRC.triggerEmergencyStopMocked(ctx, reason)`
- Line 82: **mock** → `action, err := testRC.triggerLeverageReductionMocked(ctx, targetLeverage)`
- Line 203: **mock** → `_, err := testRC.triggerEmergencyStopMocked(ctx, "Test emergency")`

## ./internal\automation\scheduler\risk\test_helpers.go
- Line 20: **mock** → `// TestDatabase provides mock database functionality for testing`
- Line 66: **mock** → `// Create mock database connection`
- Line 67: **mock** → `db := &database.DB{} // This would be properly mocked in real implementation`
- Line 84: **mock** → `// triggerPositionReductionMocked provides a mock implementation for testing`
- Line 85: **mock** → `func (trc *TestRiskController) triggerPositionReductionMocked(ctx context.Context, marginStatus *shared.MarginStatus, reductionRatio float64) (*RiskAction, error) {`
- Line 86: **mock** → `// Create a mock risk action`
- Line 116: **mock** → `// triggerEmergencyStopMocked provides a mock implementation for testing`
- Line 117: **mock** → `func (trc *TestRiskController) triggerEmergencyStopMocked(ctx context.Context, reason string) (*RiskAction, error) {`
- Line 118: **mock** → `// Create a mock risk action`
- Line 150: **mock** → `// triggerLeverageReductionMocked provides a mock implementation for testing`
- Line 151: **mock** → `func (trc *TestRiskController) triggerLeverageReductionMocked(ctx context.Context, targetLeverage float64) (*RiskAction, error) {`
- Line 152: **mock** → `// Create a mock risk action`
- Line 201: **mock** → `// Start starts the risk controller (mock implementation)`
- Line 203: **mock** → `// Mock implementation - just return success`
- Line 207: **mock** → `// Stop stops the risk controller (mock implementation)`
- Line 209: **mock** → `// Mock implementation - just return success`

## ./internal\automation\scheduler\shared\testing.go
- Line 13: **mock** → `"github.com/stretchr/testify/mock"`
- Line 20: **mock** → `mocks          map[string]interface{}`
- Line 30: **mock** → `mocks:        make(map[string]interface{}),`
- Line 55: **mock** → `// SetMock stores a mock object for later retrieval`
- Line 56: **mock** → `func (tf *TestFramework) SetMock(name string, mockObj interface{}) {`
- Line 60: **mock** → `tf.mocks[name] = mockObj`
- Line 63: **mock** → `// GetMock retrieves a stored mock object`
- Line 64: **mock** → `func (tf *TestFramework) GetMock(name string) interface{} {`
- Line 68: **mock** → `return tf.mocks[name]`
- Line 87: **mock** → `// MockDatabase provides a mock database for testing`
- Line 88: **mock** → `type MockDatabase struct {`
- Line 89: **mock** → `mock.Mock`
- Line 94: **mock** → `// NewMockDatabase creates a new mock database`
- Line 95: **mock** → `func NewMockDatabase() *MockDatabase {`
- Line 96: **mock** → `return &MockDatabase{`
- Line 101: **mock** → `// QueryContext mocks database query execution`
- Line 102: **mock** → `func (mdb *MockDatabase) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {`
- Line 106: **mock** → `mockArgs := mdb.Called(ctx, query, args)`
- Line 107: **mock** → `return mockArgs.Get(0).(*sql.Rows), mockArgs.Error(1)`
- Line 110: **mock** → `// QueryRowContext mocks database single row query`
- Line 111: **mock** → `func (mdb *MockDatabase) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {`
- Line 115: **mock** → `mockArgs := mdb.Called(ctx, query, args)`
- Line 116: **mock** → `return mockArgs.Get(0).(*sql.Row)`
- Line 119: **mock** → `// ExecContext mocks database execution`
- Line 120: **mock** → `func (mdb *MockDatabase) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {`
- Line 124: **mock** → `mockArgs := mdb.Called(ctx, query, args)`
- Line 125: **mock** → `return mockArgs.Get(0).(sql.Result), mockArgs.Error(1)`
- Line 129: **mock** → `func (mdb *MockDatabase) SetQueryResult(query string, results []map[string]interface{}) {`
- Line 136: **mock** → `// MockExchangeAPI provides a mock exchange API for testing`
- Line 137: **mock** → `type MockExchangeAPI struct {`
- Line 138: **mock** → `mock.Mock`
- Line 145: **mock** → `// NewMockExchangeAPI creates a new mock exchange API`
- Line 146: **mock** → `func NewMockExchangeAPI() *MockExchangeAPI {`
- Line 147: **mock** → `return &MockExchangeAPI{`
- Line 154: **mock** → `// GetPositions mocks getting positions from exchange`
- Line 155: **mock** → `func (mea *MockExchangeAPI) GetPositions(ctx context.Context) ([]Position, error) {`
- Line 159: **mock** → `mockArgs := mea.Called(ctx)`
- Line 160: **mock** → `return mea.positions, mockArgs.Error(0)`
- Line 163: **mock** → `// GetMarketData mocks getting market data from exchange`
- Line 164: **mock** → `func (mea *MockExchangeAPI) GetMarketData(ctx context.Context, symbol string) (map[string]interface{}, error) {`
- Line 168: **mock** → `mockArgs := mea.Called(ctx, symbol)`
- Line 170: **mock** → `return data.(map[string]interface{}), mockArgs.Error(0)`
- Line 172: **mock** → `return nil, mockArgs.Error(0)`
- Line 175: **mock** → `// PlaceOrder mocks placing an order`
- Line 176: **mock** → `func (mea *MockExchangeAPI) PlaceOrder(ctx context.Context, order map[string]interface{}) (string, error) {`
- Line 180: **mock** → `mockArgs := mea.Called(ctx, order)`
- Line 193: **mock** → `return orderID, mockArgs.Error(0)`
- Line 196: **mock** → `// SetPositions sets mock positions`
- Line 197: **mock** → `func (mea *MockExchangeAPI) SetPositions(positions []Position) {`
- Line 204: **mock** → `// SetMarketData sets mock market data`
- Line 205: **mock** → `func (mea *MockExchangeAPI) SetMarketData(symbol string, data map[string]interface{}) {`
- Line 213: **mock** → `func (mea *MockExchangeAPI) GetOrderHistory() []map[string]interface{} {`
- Line 222: **mock** → `// MockConfigProvider provides a mock configuration provider for testing`
- Line 223: **mock** → `type MockConfigProvider struct {`
- Line 224: **mock** → `mock.Mock`
- Line 229: **mock** → `// NewMockConfigProvider creates a new mock config provider`
- Line 230: **mock** → `func NewMockConfigProvider() *MockConfigProvider {`
- Line 231: **mock** → `return &MockConfigProvider{`
- Line 236: **mock** → `// Get mocks getting a configuration value`
- Line 237: **mock** → `func (mcp *MockConfigProvider) Get(key string) interface{} {`
- Line 241: **mock** → `mockArgs := mcp.Called(key)`
- Line 245: **mock** → `return mockArgs.Get(0)`
- Line 248: **mock** → `// GetString mocks getting a string configuration value`
- Line 249: **mock** → `func (mcp *MockConfigProvider) GetString(key string) string {`
- Line 257: **mock** → `// GetInt mocks getting an integer configuration value`
- Line 258: **mock** → `func (mcp *MockConfigProvider) GetInt(key string) int {`
- Line 266: **mock** → `// GetFloat64 mocks getting a float64 configuration value`
- Line 267: **mock** → `func (mcp *MockConfigProvider) GetFloat64(key string) float64 {`
- Line 275: **mock** → `// GetBool mocks getting a boolean configuration value`
- Line 276: **mock** → `func (mcp *MockConfigProvider) GetBool(key string) bool {`
- Line 284: **mock** → `// GetDuration mocks getting a duration configuration value`
- Line 285: **mock** → `func (mcp *MockConfigProvider) GetDuration(key string) time.Duration {`
- Line 293: **mock** → `// Set mocks setting a configuration value`
- Line 294: **mock** → `func (mcp *MockConfigProvider) Set(key string, value interface{}) error {`
- Line 298: **mock** → `mockArgs := mcp.Called(key, value)`
- Line 300: **mock** → `return mockArgs.Error(0)`
- Line 303: **mock** → `// Reload mocks reloading configuration`
- Line 304: **mock** → `func (mcp *MockConfigProvider) Reload() error {`
- Line 305: **mock** → `mockArgs := mcp.Called()`
- Line 306: **mock** → `return mockArgs.Error(0)`
- Line 310: **mock** → `func (mcp *MockConfigProvider) SetConfig(key string, value interface{}) {`
- Line 317: **mock** → `// MockMetricsCollector provides a mock metrics collector for testing`
- Line 318: **mock** → `type MockMetricsCollector struct {`
- Line 319: **mock** → `mock.Mock`
- Line 327: **mock** → `// NewMockMetricsCollector creates a new mock metrics collector`
- Line 328: **mock** → `func NewMockMetricsCollector() *MockMetricsCollector {`
- Line 329: **mock** → `return &MockMetricsCollector{`
- Line 337: **mock** → `// Counter mocks incrementing a counter metric`
- Line 338: **mock** → `func (mmc *MockMetricsCollector) Counter(name string, tags map[string]string) error {`
- Line 342: **mock** → `mockArgs := mmc.Called(name, tags)`
- Line 344: **mock** → `return mockArgs.Error(0)`
- Line 347: **mock** → `// Gauge mocks setting a gauge metric`
- Line 348: **mock** → `func (mmc *MockMetricsCollector) Gauge(name string, value float64, tags map[string]string) error {`
- Line 352: **mock** → `mockArgs := mmc.Called(name, value, tags)`
- Line 354: **mock** → `return mockArgs.Error(0)`
- Line 357: **mock** → `// Histogram mocks recording a histogram metric`
- Line 358: **mock** → `func (mmc *MockMetricsCollector) Histogram(name string, value float64, tags map[string]string) error {`
- Line 362: **mock** → `mockArgs := mmc.Called(name, value, tags)`
- Line 364: **mock** → `return mockArgs.Error(0)`
- Line 367: **mock** → `// Timer mocks recording a timer metric`
- Line 368: **mock** → `func (mmc *MockMetricsCollector) Timer(name string, duration time.Duration, tags map[string]string) error {`
- Line 372: **mock** → `mockArgs := mmc.Called(name, duration, tags)`
- Line 374: **mock** → `return mockArgs.Error(0)`
- Line 378: **mock** → `func (mmc *MockMetricsCollector) GetCounterValue(name string) int64 {`
- Line 386: **mock** → `func (mmc *MockMetricsCollector) GetGaugeValue(name string) float64 {`
- Line 394: **mock** → `func (mmc *MockMetricsCollector) GetHistogramValues(name string) []float64 {`
- Line 404: **mock** → `func (mmc *MockMetricsCollector) GetTimerValues(name string) []time.Duration {`
- Line 544: **mock** → `func (ah *AssertionHelpers) AssertMetricsRecorded(collector *MockMetricsCollector, metricName string) {`

## ./internal\operations\healing\self_healing_system.go
- Line 1667: **TODO** → `// TODO: 实现实际的配置变更`
- Line 1763: **TODO** → `// TODO: 实现异常检测逻辑`
- Line 1767: **TODO** → `// TODO: 基于故障更新系统健康状态`
- Line 1795: **TODO** → `// TODO: 实现实际的API服务器健康检查`
- Line 1812: **TODO** → `// TODO: 实现实际的数据库健康检查`
- Line 1829: **TODO** → `// TODO: 实现实际的Redis健康检查`
- Line 1846: **TODO** → `// TODO: 实现实际的交易所连接器健康检查`
- Line 1868: **TODO** → `// TODO: 实现实际的策略引擎健康检查`
- Line 1880: **TODO** → `// TODO: 实现根因分析逻辑`
- Line 1891: **TODO** → `// TODO: 实现影响评估逻辑`
- Line 1985: **TODO** → `// TODO: 更新知识库，记录成功/失败的恢复案例`
- Line 1989: **TODO** → `// TODO: 基于历史数据计算平均时间`
- Line 1997: **TODO** → `// TODO: 计算实际的正常运行时间百分比`
- Line 2091: **TODO** → `// TODO: 实现从实际监控系统获取指标`
- Line 2092: **TODO** → `// TODO 集成Prometheus、InfluxDB等监控系统`
- Line 2096: **TODO** → `// TODO: 从监控系统获取API响应时间`
- Line 2099: **TODO** → `// TODO: 从监控系统获取错误率`
- Line 2102: **TODO** → `// TODO: 从监控系统获取连接成功率`
- Line 2105: **TODO** → `// TODO: 从监控系统获取API超时率`
- Line 2108: **TODO** → `// TODO: 从监控系统获取CPU使用率`
- Line 2111: **TODO** → `// TODO: 从监控系统获取内存使用率`
- Line 2114: **TODO** → `// TODO: 从监控系统获取磁盘使用率`

## ./internal\security\protector\exchange_provider.go
- Line 499: **mock** → `// MockExchangeProvider 模拟交易所数据提供者（用于测试）`
- Line 500: **mock** → `type MockExchangeProvider struct {`
- Line 506: **mock** → `// NewMockExchangeProvider 创建模拟交易所数据提供者`
- Line 507: **mock** → `func NewMockExchangeProvider() *MockExchangeProvider {`
- Line 508: **mock** → `return &MockExchangeProvider{`
- Line 536: **mock** → `func (m *MockExchangeProvider) IsHealthy() bool {`
- Line 541: **mock** → `func (m *MockExchangeProvider) GetFundData(ctx context.Context) (*ExchangeFundData, error) {`
- Line 549: **mock** → `func (m *MockExchangeProvider) GetPositions(ctx context.Context) ([]*Position, error) {`
- Line 557: **mock** → `func (m *MockExchangeProvider) GetHistoricalReturns(ctx context.Context, days int) ([]float64, error) {`
- Line 572: **mock** → `func (m *MockExchangeProvider) GetHistoricalEquity(ctx context.Context, days int) ([]float64, error) {`
- Line 590: **mock** → `func (m *MockExchangeProvider) GetSymbolPrice(ctx context.Context, symbol string) (float64, error) {`
- Line 610: **mock** → `func (m *MockExchangeProvider) GetOrderBookDepth(ctx context.Context, symbol string) (*OrderBookDepth, error) {`
- Line 635: **mock** → `func (m *MockExchangeProvider) GetTradingVolume(ctx context.Context, symbol string, period string) (float64, error) {`
- Line 645: **mock** → `func (m *MockExchangeProvider) SetHealthy(healthy bool) {`
- Line 650: **mock** → `func (m *MockExchangeProvider) SetFundData(data *ExchangeFundData) {`
- Line 655: **mock** → `func (m *MockExchangeProvider) SetPositions(positions []*Position) {`

## ./internal\security\protector\notification_service.go
- Line 256: **mock** → `// MockNotificationService 模拟通知服务（用于测试）`
- Line 257: **mock** → `type MockNotificationService struct {`
- Line 294: **mock** → `// NewMockNotificationService 创建模拟通知服务`
- Line 295: **mock** → `func NewMockNotificationService() *MockNotificationService {`
- Line 296: **mock** → `return &MockNotificationService{`
- Line 306: **mock** → `func (m *MockNotificationService) SendEmail(ctx context.Context, to, subject, body string) error {`
- Line 308: **mock** → `return fmt.Errorf("mock email failure")`
- Line 318: **mock** → `log.Printf("Mock email sent to %s: %s", to, subject)`
- Line 323: **mock** → `func (m *MockNotificationService) SendSMS(ctx context.Context, phone, message string) error {`
- Line 325: **mock** → `return fmt.Errorf("mock SMS failure")`
- Line 334: **mock** → `log.Printf("Mock SMS sent to %s: %s", phone, message)`
- Line 339: **mock** → `func (m *MockNotificationService) SendWebhook(ctx context.Context, url string, payload interface{}) error {`
- Line 341: **mock** → `return fmt.Errorf("mock webhook failure")`
- Line 350: **mock** → `log.Printf("Mock webhook sent to %s", url)`
- Line 355: **mock** → `func (m *MockNotificationService) SendSlack(ctx context.Context, webhook, message string) error {`
- Line 357: **mock** → `return fmt.Errorf("mock Slack failure")`
- Line 366: **mock** → `log.Printf("Mock Slack message sent: %s", message)`
- Line 371: **mock** → `func (m *MockNotificationService) SetShouldFail(shouldFail bool) {`
- Line 376: **mock** → `func (m *MockNotificationService) GetEmailsSent() []EmailRecord {`
- Line 381: **mock** → `func (m *MockNotificationService) GetSMSSent() []SMSRecord {`
- Line 386: **mock** → `func (m *MockNotificationService) GetWebhooksSent() []WebhookRecord {`
- Line 391: **mock** → `func (m *MockNotificationService) GetSlackSent() []SlackRecord {`
- Line 396: **mock** → `func (m *MockNotificationService) Reset() {`

## ./internal\security\protector\wallet_service.go
- Line 351: **mock** → `// MockWalletService 模拟钱包服务（用于测试）`
- Line 352: **mock** → `type MockWalletService struct {`
- Line 358: **mock** → `// NewMockWalletService 创建模拟钱包服务`
- Line 359: **mock** → `func NewMockWalletService() *MockWalletService {`
- Line 360: **mock** → `return &MockWalletService{`
- Line 368: **mock** → `func (m *MockWalletService) InitiateTransfer(ctx context.Context, request *TransferRequest) (*TransferResponse, error) {`
- Line 370: **mock** → `return nil, fmt.Errorf("mock transfer failure")`
- Line 377: **mock** → `return nil, fmt.Errorf("random mock transfer failure")`
- Line 381: **mock** → `transferID := fmt.Sprintf("MOCK_TXF_%d", time.Now().Unix())`
- Line 405: **mock** → `log.Printf("Mock transfer initiated: %s", transferID)`
- Line 410: **mock** → `func (m *MockWalletService) GetTransferStatus(ctx context.Context, transferID string) (*TransferStatus, error) {`
- Line 419: **mock** → `func (m *MockWalletService) CancelTransfer(ctx context.Context, transferID string) error {`
- Line 429: **mock** → `func (m *MockWalletService) GetTransferHistory(ctx context.Context, limit int) ([]*TransferRecord, error) {`
- Line 445: **mock** → `func (m *MockWalletService) ValidateAddress(address string) error {`
- Line 453: **mock** → `func (m *MockWalletService) EstimateTransferFee(ctx context.Context, amount float64, toAddress string) (float64, error) {`
- Line 458: **mock** → `func (m *MockWalletService) SetShouldFail(shouldFail bool) {`
- Line 463: **mock** → `func (m *MockWalletService) SetFailureRate(rate float64) {`
- Line 468: **mock** → `func (m *MockWalletService) GetTransfers() map[string]*TransferStatus {`

## ./internal\strategy\unified_service.go
- Line 365: **mock** → `// 如果没有数据库连接，返回空结果而不是 mock 数据`
- Line 413: **TODO** → `// TODO: 解析JSON配置`
- Line 417: **mock** → `// 如果没有数据库连接，返回错误而不是 mock 数据`
- Line 434: **TODO** → `ExecutionCount: 100, // TODO: 从实际数据获取`
- Line 465: **mock** → `// 如果没有从策略池获取到信息，使用默认值而不是 mock 数据`
- Line 594: **mock** → `// 删除了 getMockStrategies 方法，不再使用 mock 数据`
- Line 596: **mock** → `// 删除了 getMockExecutionInfo 方法，不再使用 mock 数据`
- Line 598: **mock** → `// 删除了 getMockPerformanceInfo 和 getMockPoolInfo 方法，不再使用 mock 数据`

## ./internal\strategy\unified_service_simple.go
- Line 44: **mock** → `// 返回空策略列表，避免使用 mock 数据`
- Line 82: **mock** → `// 不使用 mock 数据，直接返回未找到错误`
- Line 248: **mock** → `// 删除了 getMockStrategies 方法，不再使用 mock 数据`
- Line 250: **mock** → `// 删除了所有 mock 方法，不再使用 mock 数据`

## ./internal\strategy\generator\analyzer.go
- Line 390: **mock** → `// generateMockPriceData 生成模拟价格数据`
- Line 391: **mock** → `func (ma *MarketAnalyzer) generateMockPriceData(symbol string, timeRange time.Duration, startTime, endTime time.Time) []PricePoint {`

## ./internal\strategy\generator\analyzer_test.go
- Line 69: **mock** → `func TestMarketAnalyzer_GenerateMockPriceData(t *testing.T) {`
- Line 77: **mock** → `priceData := analyzer.generateMockPriceData(symbol, timeRange, startTime, endTime)`

## ./internal\strategy\optimizer\optimizer_test.go
- Line 16: **mock** → `mockData := testutils.NewMockData()`
- Line 51: **mock** → `historicalData := generateMockHistoricalData(1000)`
- Line 117: **mock** → `Returns:        generateMockReturns(150),`
- Line 225: **mock** → `historicalData := generateMockHistoricalData(500)`
- Line 240: **mock** → `func generateMockHistoricalData(count int) []MarketData {`
- Line 241: **mock** → `mockData := testutils.NewMockData()`
- Line 247: **mock** → `change := mockData.RandomFloat(-0.05, 0.05)`
- Line 252: **mock** → `Open:      basePrice * (1 + mockData.RandomFloat(-0.01, 0.01)),`
- Line 253: **mock** → `High:      basePrice * (1 + mockData.RandomFloat(0, 0.02)),`
- Line 254: **mock** → `Low:       basePrice * (1 + mockData.RandomFloat(-0.02, 0)),`
- Line 256: **mock** → `Volume:    mockData.RandomFloat(100, 1000),`
- Line 263: **mock** → `func generateMockReturns(count int) []float64 {`
- Line 264: **mock** → `mockData := testutils.NewMockData()`
- Line 269: **mock** → `returns[i] = mockData.RandomFloat(-0.05, 0.05)`

## ./internal\strategy\optimizer\orchestrator_test.go
- Line 143: **mock** → `// Create a mock orchestrator without database connection`

## ./internal\strategy\sandbox\automation.go
- Line 154: **mock** → `mockExchange := s.createMockExchange(testConfig)`
- Line 165: **mock** → `sandbox := NewSandbox(strategyInstance, strategyConfig.Params, mockExchange)`
- Line 464: **mock** → `// createMockExchange 创建模拟交易所`
- Line 465: **mock** → `func (s *AutomatedSandboxService) createMockExchange(config *TestConfiguration) exchange.Exchange {`

## ./internal\testutils\testutils.go
- Line 98: **mock** → `suite.setupMockDB()`
- Line 105: **mock** → `suite.setupMockCache()`
- Line 131: **mock** → `s.setupMockDB()`
- Line 138: **mock** → `s.setupMockDB()`
- Line 146: **mock** → `s.setupMockDB()`
- Line 158: **mock** → `// setupMockDB 设置模拟数据库`
- Line 159: **mock** → `func (s *TestSuite) setupMockDB() {`
- Line 201: **mock** → `s.setupMockCache()`
- Line 209: **mock** → `s.setupMockCache()`
- Line 221: **mock** → `// setupMockCache 设置模拟缓存`
- Line 222: **mock** → `func (s *TestSuite) setupMockCache() {`
- Line 349: **mock** → `// MockData 模拟数据生成器`
- Line 350: **mock** → `type MockData struct {`
- Line 354: **mock** → `// NewMockData 创建模拟数据生成器`
- Line 355: **mock** → `func NewMockData() *MockData {`
- Line 356: **mock** → `return &MockData{`
- Line 362: **mock** → `func (m *MockData) RandomString(length int) string {`
- Line 372: **mock** → `func (m *MockData) RandomInt(min, max int) int {`
- Line 377: **mock** → `func (m *MockData) RandomFloat(min, max float64) float64 {`
- Line 382: **mock** → `func (m *MockData) RandomBool() bool {`
- Line 387: **mock** → `func (m *MockData) RandomChoice(choices []string) string {`
- Line 392: **mock** → `func (m *MockData) GenerateStrategy() map[string]interface{} {`
- Line 419: **mock** → `func (m *MockData) GenerateOrder() map[string]interface{} {`

## ./internal\workflow\executors.go
- Line 27: **mock** → `// MockExecutor 模拟执行器（用于测试）`
- Line 28: **mock** → `type MockExecutor struct {`
- Line 34: **mock** → `// NewMockExecutor 创建模拟执行器`
- Line 35: **mock** → `func NewMockExecutor(name string, executionTime time.Duration, simulateFailure bool) *MockExecutor {`
- Line 36: **mock** → `return &MockExecutor{`
- Line 51: **mock** → `func (me *MockExecutor) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {`
- Line 355: **mock** → `executors[id] = NewMockExecutor(name, execTime, false)`


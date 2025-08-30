package routing

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"

	"qcat/internal/config"
)

// SmartExchangeRouter 智能交易所路由系统
type SmartExchangeRouter struct {
	config             *config.Config
	exchangeManager    *ExchangeManager
	healthMonitor      *HealthMonitor
	loadBalancer       *LoadBalancer
	failoverController *FailoverController
	routingOptimizer   *RoutingOptimizer

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
	mu        sync.RWMutex

	// 路由配置
	primaryExchange     string
	backupExchanges     []string
	failoverThreshold   float64
	latencyThreshold    time.Duration
	healthCheckInterval time.Duration

	// 路由状态
	exchangeStatus map[string]*ExchangeStatus
	routingRules   []RoutingRule
	routingHistory []RoutingDecision

	// 监控指标
	routingMetrics     *RoutingMetrics
	performanceHistory []PerformanceSnapshot

	// 配置参数
	enabled       bool
	autoFailover  bool
	smartRouting  bool
	loadBalancing bool
}

// ExchangeStatus 交易所状态
type ExchangeStatus struct {
	Exchange        string        `json:"exchange"`
	IsOnline        bool          `json:"is_online"`
	Latency         time.Duration `json:"latency"`
	Availability    float64       `json:"availability"`
	ThroughputLimit float64       `json:"throughput_limit"`
	CurrentLoad     float64       `json:"current_load"`
	ErrorRate       float64       `json:"error_rate"`

	// 连接状态
	ConnectionStatus    string    `json:"connection_status"` // CONNECTED, DISCONNECTED, CONNECTING, ERROR
	LastPing            time.Time `json:"last_ping"`
	PingSuccess         bool      `json:"ping_success"`
	ConsecutiveFailures int       `json:"consecutive_failures"`

	// 交易相关
	OrderBookDepth  float64            `json:"order_book_depth"`
	SpreadTightness float64            `json:"spread_tightness"`
	TradingFees     map[string]float64 `json:"trading_fees"`
	SupportedPairs  []string           `json:"supported_pairs"`

	// 历史统计
	UptimePercentage float64       `json:"uptime_percentage"`
	AvgLatency       time.Duration `json:"avg_latency"`
	AvgErrorRate     float64       `json:"avg_error_rate"`

	// 限制和约束
	RateLimits         map[string]int      `json:"rate_limits"`
	MaintenanceWindows []MaintenanceWindow `json:"maintenance_windows"`

	LastUpdated  time.Time `json:"last_updated"`
	LastFailover time.Time `json:"last_failover"`
	HealthScore  float64   `json:"health_score"`
}

// MaintenanceWindow 维护窗口
type MaintenanceWindow struct {
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Type        string    `json:"type"` // SCHEDULED, EMERGENCY
	Description string    `json:"description"`
	Recurring   bool      `json:"recurring"`
	Impact      string    `json:"impact"` // HIGH, MEDIUM, LOW
}

// ExchangeManager 交易所管理器
type ExchangeManager struct {
	exchanges         map[string]*Exchange
	connectionPool    map[string]*ConnectionPool
	credentialManager *CredentialManager

	mu sync.RWMutex
}

// Exchange 交易所配置
type Exchange struct {
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	Region      string  `json:"region"`
	Type        string  `json:"type"` // SPOT, FUTURES, OPTIONS
	Priority    int     `json:"priority"`
	Capacity    float64 `json:"capacity"`

	// 连接配置
	RestAPI      APIConfig  `json:"rest_api"`
	WebSocketAPI APIConfig  `json:"websocket_api"`
	FIXProtocol  *FIXConfig `json:"fix_protocol"`

	// 交易配置
	MinOrderSize map[string]float64 `json:"min_order_size"`
	MaxOrderSize map[string]float64 `json:"max_order_size"`
	TickSizes    map[string]float64 `json:"tick_sizes"`
	TradingFees  FeeStructure       `json:"trading_fees"`

	// 功能支持
	SupportedOrderTypes []string `json:"supported_order_types"`
	SupportedTimeframes []string `json:"supported_timeframes"`
	MarginTrading       bool     `json:"margin_trading"`
	OptionsTrading      bool     `json:"options_trading"`

	// 状态
	IsEnabled   bool      `json:"is_enabled"`
	LastUpdated time.Time `json:"last_updated"`
}

// APIConfig API配置
type APIConfig struct {
	BaseURL        string            `json:"base_url"`
	Version        string            `json:"version"`
	Endpoints      map[string]string `json:"endpoints"`
	RateLimits     map[string]int    `json:"rate_limits"`
	Timeout        time.Duration     `json:"timeout"`
	RetryAttempts  int               `json:"retry_attempts"`
	Authentication AuthConfig        `json:"authentication"`
}

// FIXConfig FIX协议配置
type FIXConfig struct {
	Host              string        `json:"host"`
	Port              int           `json:"port"`
	SenderCompID      string        `json:"sender_comp_id"`
	TargetCompID      string        `json:"target_comp_id"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
	LogonTimeout      time.Duration `json:"logon_timeout"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	Type            string `json:"type"` // API_KEY, OAUTH, SIGNATURE
	APIKey          string `json:"api_key"`
	SecretKey       string `json:"secret_key"`
	Passphrase      string `json:"passphrase"`
	SignatureMethod string `json:"signature_method"`
}

// FeeStructure 费用结构
type FeeStructure struct {
	MakerFee       float64            `json:"maker_fee"`
	TakerFee       float64            `json:"taker_fee"`
	WithdrawalFees map[string]float64 `json:"withdrawal_fees"`
	VIPLevels      map[string]VIPFee  `json:"vip_levels"`
}

// VIPFee VIP费率
type VIPFee struct {
	MakerFee     float64            `json:"maker_fee"`
	TakerFee     float64            `json:"taker_fee"`
	Requirements map[string]float64 `json:"requirements"`
}

// ConnectionPool 连接池
type ConnectionPool struct {
	MaxConnections    int           `json:"max_connections"`
	ActiveConnections int           `json:"active_connections"`
	IdleConnections   int           `json:"idle_connections"`
	ConnectionTimeout time.Duration `json:"connection_timeout"`
	IdleTimeout       time.Duration `json:"idle_timeout"`
	MaxLifetime       time.Duration `json:"max_lifetime"`

	mu sync.RWMutex
}

// CredentialManager 凭证管理器
type CredentialManager struct {
	credentials   map[string]*ExchangeCredential
	encryptionKey []byte

	mu sync.RWMutex
}

// ExchangeCredential 交易所凭证
type ExchangeCredential struct {
	Exchange    string    `json:"exchange"`
	APIKey      string    `json:"api_key"`
	SecretKey   string    `json:"secret_key"`
	Passphrase  string    `json:"passphrase"`
	IsActive    bool      `json:"is_active"`
	ExpiresAt   time.Time `json:"expires_at"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsed    time.Time `json:"last_used"`
}

// HealthMonitor 健康监控器
type HealthMonitor struct {
	checkInterval   time.Duration
	timeoutDuration time.Duration
	healthThreshold float64

	// 监控历史
	healthHistory map[string][]HealthCheck
	lastChecks    map[string]HealthCheck

	mu sync.RWMutex
}

// HealthCheck 健康检查
type HealthCheck struct {
	Exchange     string        `json:"exchange"`
	Timestamp    time.Time     `json:"timestamp"`
	IsHealthy    bool          `json:"is_healthy"`
	Latency      time.Duration `json:"latency"`
	ErrorMessage string        `json:"error_message"`
	ResponseTime time.Duration `json:"response_time"`

	// 检查详情
	PingTest      TestResult `json:"ping_test"`
	APITest       TestResult `json:"api_test"`
	WebSocketTest TestResult `json:"websocket_test"`
	OrderBookTest TestResult `json:"order_book_test"`

	// 综合评分
	HealthScore float64            `json:"health_score"`
	Components  map[string]float64 `json:"components"`
}

// TestResult 测试结果
type TestResult struct {
	Passed   bool                   `json:"passed"`
	Duration time.Duration          `json:"duration"`
	Error    string                 `json:"error"`
	Details  map[string]interface{} `json:"details"`
}

// LoadBalancer 负载均衡器
type LoadBalancer struct {
	algorithm    string // ROUND_ROBIN, WEIGHTED, LEAST_CONNECTIONS, HASH
	weights      map[string]float64
	connections  map[string]int
	lastSelected string

	mu sync.RWMutex
}

// FailoverController 故障转移控制器
type FailoverController struct {
	failoverStrategy  string // AUTO, MANUAL, HYBRID
	failoverThreshold float64
	recoveryThreshold float64
	maxFailovers      int
	failoverCooldown  time.Duration

	// 故障转移历史
	failoverHistory []FailoverEvent
	lastFailover    time.Time
	failoverCount   int

	mu sync.RWMutex
}

// FailoverEvent 故障转移事件
type FailoverEvent struct {
	ID           string         `json:"id"`
	Timestamp    time.Time      `json:"timestamp"`
	FromExchange string         `json:"from_exchange"`
	ToExchange   string         `json:"to_exchange"`
	Trigger      string         `json:"trigger"`
	TriggerValue float64        `json:"trigger_value"`
	Reason       string         `json:"reason"`
	Duration     time.Duration  `json:"duration"`
	Success      bool           `json:"success"`
	Impact       FailoverImpact `json:"impact"`
	AutoRecovery bool           `json:"auto_recovery"`
	RecoveryTime time.Time      `json:"recovery_time"`
}

// FailoverImpact 故障转移影响
type FailoverImpact struct {
	AffectedOrders      int           `json:"affected_orders"`
	TradingInterruption time.Duration `json:"trading_interruption"`
	LostOpportunities   float64       `json:"lost_opportunities"`
	AdditionalCosts     float64       `json:"additional_costs"`
	CustomerImpact      string        `json:"customer_impact"`
}

// RoutingOptimizer 路由优化器
type RoutingOptimizer struct {
	optimizationModel  string // LATENCY, COST, LIQUIDITY, HYBRID
	reoptimizeInterval time.Duration

	// 优化参数
	latencyWeight     float64
	costWeight        float64
	liquidityWeight   float64
	reliabilityWeight float64

	// 优化历史
	optimizationHistory []OptimizationResult

	mu sync.RWMutex
}

// OptimizationResult 优化结果
type OptimizationResult struct {
	Timestamp           time.Time           `json:"timestamp"`
	OptimizationModel   string              `json:"optimization_model"`
	PreviousRouting     map[string]float64  `json:"previous_routing"`
	OptimalRouting      map[string]float64  `json:"optimal_routing"`
	ExpectedImprovement float64             `json:"expected_improvement"`
	ActualImprovement   float64             `json:"actual_improvement"`
	Metrics             OptimizationMetrics `json:"metrics"`
}

// OptimizationMetrics 优化指标
type OptimizationMetrics struct {
	AvgLatency       time.Duration `json:"avg_latency"`
	TotalCost        float64       `json:"total_cost"`
	LiquidityScore   float64       `json:"liquidity_score"`
	ReliabilityScore float64       `json:"reliability_score"`
	ThroughputScore  float64       `json:"throughput_score"`
}

// RoutingRule 路由规则
type RoutingRule struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Priority     int              `json:"priority"`
	Condition    RoutingCondition `json:"condition"`
	Action       RoutingAction    `json:"action"`
	IsActive     bool             `json:"is_active"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	HitCount     int64            `json:"hit_count"`
	SuccessCount int64            `json:"success_count"`
}

// RoutingCondition 路由条件
type RoutingCondition struct {
	Type            string             `json:"type"`     // EXCHANGE_DOWN, HIGH_LATENCY, HIGH_COST, SYMBOL, TIME
	Operator        string             `json:"operator"` // EQUALS, GREATER_THAN, LESS_THAN, CONTAINS
	Value           interface{}        `json:"value"`
	LogicalOperator string             `json:"logical_operator"` // AND, OR, NOT
	SubConditions   []RoutingCondition `json:"sub_conditions"`
}

// RoutingAction 路由动作
type RoutingAction struct {
	Type           string                 `json:"type"` // ROUTE_TO, AVOID, LOAD_BALANCE, FAILOVER
	TargetExchange string                 `json:"target_exchange"`
	Parameters     map[string]interface{} `json:"parameters"`
	Fallback       *RoutingAction         `json:"fallback"`
}

// RoutingDecision 路由决策
type RoutingDecision struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	OrderID   string    `json:"order_id"`
	Symbol    string    `json:"symbol"`
	OrderType string    `json:"order_type"`

	// 决策过程
	SelectedExchange     string   `json:"selected_exchange"`
	AlternativeExchanges []string `json:"alternative_exchanges"`
	DecisionReason       string   `json:"decision_reason"`
	RuleMatches          []string `json:"rule_matches"`

	// 决策指标
	LatencyScore     float64 `json:"latency_score"`
	CostScore        float64 `json:"cost_score"`
	LiquidityScore   float64 `json:"liquidity_score"`
	ReliabilityScore float64 `json:"reliability_score"`
	OverallScore     float64 `json:"overall_score"`

	// 执行结果
	ExecutionTime time.Duration `json:"execution_time"`
	Success       bool          `json:"success"`
	ErrorMessage  string        `json:"error_message"`

	// 性能比较
	ExpectedLatency time.Duration `json:"expected_latency"`
	ActualLatency   time.Duration `json:"actual_latency"`
	ExpectedCost    float64       `json:"expected_cost"`
	ActualCost      float64       `json:"actual_cost"`
}

// RoutingMetrics 路由指标
type RoutingMetrics struct {
	mu sync.RWMutex

	// 路由统计
	TotalRequests    int64   `json:"total_requests"`
	SuccessfulRoutes int64   `json:"successful_routes"`
	FailedRoutes     int64   `json:"failed_routes"`
	SuccessRate      float64 `json:"success_rate"`

	// 性能指标
	AvgRoutingLatency   time.Duration `json:"avg_routing_latency"`
	AvgExecutionLatency time.Duration `json:"avg_execution_latency"`
	P95Latency          time.Duration `json:"p95_latency"`
	P99Latency          time.Duration `json:"p99_latency"`

	// 交易所分布
	ExchangeDistribution map[string]int64         `json:"exchange_distribution"`
	ExchangeSuccessRates map[string]float64       `json:"exchange_success_rates"`
	ExchangeLatencies    map[string]time.Duration `json:"exchange_latencies"`

	// 故障转移统计
	FailoverCount    int64         `json:"failover_count"`
	AvgFailoverTime  time.Duration `json:"avg_failover_time"`
	AutoRecoveryRate float64       `json:"auto_recovery_rate"`

	// 成本统计
	TotalTradingCosts float64 `json:"total_trading_costs"`
	AvgTradingCost    float64 `json:"avg_trading_cost"`
	CostSavings       float64 `json:"cost_savings"`

	// 优化效果
	OptimizationEfficiency float64 `json:"optimization_efficiency"`
	RouteQuality           float64 `json:"route_quality"`

	LastUpdated time.Time `json:"last_updated"`
}

// PerformanceSnapshot 性能快照
type PerformanceSnapshot struct {
	Timestamp           time.Time                      `json:"timestamp"`
	ExchangePerformance map[string]ExchangePerformance `json:"exchange_performance"`
	RoutingQuality      float64                        `json:"routing_quality"`
	SystemLoad          float64                        `json:"system_load"`
	FailoverEvents      int                            `json:"failover_events"`
}

// ExchangePerformance 交易所性能
type ExchangePerformance struct {
	Latency               time.Duration `json:"latency"`
	Availability          float64       `json:"availability"`
	ThroughputUtilization float64       `json:"throughput_utilization"`
	ErrorRate             float64       `json:"error_rate"`
	HealthScore           float64       `json:"health_score"`
	OrderSuccessRate      float64       `json:"order_success_rate"`
}

// NewSmartExchangeRouter 创建智能交易所路由系统
func NewSmartExchangeRouter(cfg *config.Config) (*SmartExchangeRouter, error) {
	ctx, cancel := context.WithCancel(context.Background())

	ser := &SmartExchangeRouter{
		config:             cfg,
		exchangeManager:    NewExchangeManager(),
		healthMonitor:      NewHealthMonitor(),
		loadBalancer:       NewLoadBalancer(),
		failoverController: NewFailoverController(),
		routingOptimizer:   NewRoutingOptimizer(),
		ctx:                ctx,
		cancel:             cancel,
		exchangeStatus:     make(map[string]*ExchangeStatus),
		routingRules:       make([]RoutingRule, 0),
		routingHistory:     make([]RoutingDecision, 0),
		routingMetrics: &RoutingMetrics{
			ExchangeDistribution: make(map[string]int64),
			ExchangeSuccessRates: make(map[string]float64),
			ExchangeLatencies:    make(map[string]time.Duration),
		},
		performanceHistory:  make([]PerformanceSnapshot, 0),
		primaryExchange:     "binance",
		backupExchanges:     []string{"okx", "bybit", "huobi"},
		failoverThreshold:   0.95,
		latencyThreshold:    100 * time.Millisecond,
		healthCheckInterval: 30 * time.Second,
		enabled:             true,
		autoFailover:        true,
		smartRouting:        true,
		loadBalancing:       true,
	}

	// 从配置文件读取参数
	if cfg != nil {
		// 从配置文件读取路由参数
		if cfg.Exchange.Name != "" {
			ser.primaryExchange = cfg.Exchange.Name
		}

		// 从配置读取故障转移阈值
		if cfg.Monitoring.Alerts.ErrorRatePercent > 0 {
			ser.failoverThreshold = (100.0 - cfg.Monitoring.Alerts.ErrorRatePercent) / 100.0
		}

		// 从配置读取延迟阈值
		if cfg.Monitoring.Alerts.HighLatencyMs > 0 {
			ser.latencyThreshold = time.Duration(cfg.Monitoring.Alerts.HighLatencyMs) * time.Millisecond
		}

		// 从配置读取健康检查间隔
		if cfg.Monitoring.HealthCheck.IntervalSeconds > 0 {
			ser.healthCheckInterval = time.Duration(cfg.Monitoring.HealthCheck.IntervalSeconds) * time.Second
		}

		// 设置备用交易所列表（基于配置的交易所信息）
		if len(ser.backupExchanges) == 0 {
			ser.backupExchanges = []string{"okx", "bybit", "huobi"}
		}

		log.Printf("Loaded routing config: primary=%s, failover_threshold=%.2f, latency_threshold=%v, health_check_interval=%v",
			ser.primaryExchange, ser.failoverThreshold, ser.latencyThreshold, ser.healthCheckInterval)
	}

	// 初始化交易所
	err := ser.initializeExchanges()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize exchanges: %w", err)
	}

	// 初始化路由规则
	ser.initializeRoutingRules()

	return ser, nil
}

// NewExchangeManager 创建交易所管理器
func NewExchangeManager() *ExchangeManager {
	return &ExchangeManager{
		exchanges:         make(map[string]*Exchange),
		connectionPool:    make(map[string]*ConnectionPool),
		credentialManager: NewCredentialManager(),
	}
}

// NewCredentialManager 创建凭证管理器
func NewCredentialManager() *CredentialManager {
	return &CredentialManager{
		credentials:   make(map[string]*ExchangeCredential),
		encryptionKey: []byte("encryption-key-32-bytes-long!!!"), // 应从安全存储获取
	}
}

// NewHealthMonitor 创建健康监控器
func NewHealthMonitor() *HealthMonitor {
	return &HealthMonitor{
		checkInterval:   30 * time.Second,
		timeoutDuration: 10 * time.Second,
		healthThreshold: 0.8,
		healthHistory:   make(map[string][]HealthCheck),
		lastChecks:      make(map[string]HealthCheck),
	}
}

// NewLoadBalancer 创建负载均衡器
func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		algorithm:   "WEIGHTED",
		weights:     make(map[string]float64),
		connections: make(map[string]int),
	}
}

// NewFailoverController 创建故障转移控制器
func NewFailoverController() *FailoverController {
	return &FailoverController{
		failoverStrategy:  "AUTO",
		failoverThreshold: 0.8,
		recoveryThreshold: 0.9,
		maxFailovers:      5,
		failoverCooldown:  5 * time.Minute,
		failoverHistory:   make([]FailoverEvent, 0),
	}
}

// NewRoutingOptimizer 创建路由优化器
func NewRoutingOptimizer() *RoutingOptimizer {
	return &RoutingOptimizer{
		optimizationModel:   "HYBRID",
		reoptimizeInterval:  1 * time.Hour,
		latencyWeight:       0.3,
		costWeight:          0.2,
		liquidityWeight:     0.25,
		reliabilityWeight:   0.25,
		optimizationHistory: make([]OptimizationResult, 0),
	}
}

// Start 启动智能交易所路由系统
func (ser *SmartExchangeRouter) Start() error {
	ser.mu.Lock()
	defer ser.mu.Unlock()

	if ser.isRunning {
		return fmt.Errorf("smart exchange router is already running")
	}

	if !ser.enabled {
		return fmt.Errorf("smart exchange router is disabled")
	}

	log.Println("Starting Smart Exchange Router...")

	// 启动健康监控
	ser.wg.Add(1)
	go ser.runHealthMonitoring()

	// 启动负载均衡
	ser.wg.Add(1)
	go ser.runLoadBalancing()

	// 启动故障转移监控
	ser.wg.Add(1)
	go ser.runFailoverMonitoring()

	// 启动路由优化
	ser.wg.Add(1)
	go ser.runRoutingOptimization()

	// 启动性能监控
	ser.wg.Add(1)
	go ser.runPerformanceMonitoring()

	// 启动指标收集
	ser.wg.Add(1)
	go ser.runMetricsCollection()

	ser.isRunning = true
	log.Println("Smart Exchange Router started successfully")
	return nil
}

// Stop 停止智能交易所路由系统
func (ser *SmartExchangeRouter) Stop() error {
	ser.mu.Lock()
	defer ser.mu.Unlock()

	if !ser.isRunning {
		return fmt.Errorf("smart exchange router is not running")
	}

	log.Println("Stopping Smart Exchange Router...")

	ser.cancel()
	ser.wg.Wait()

	ser.isRunning = false
	log.Println("Smart Exchange Router stopped successfully")
	return nil
}

// initializeExchanges 初始化交易所
func (ser *SmartExchangeRouter) initializeExchanges() error {
	exchanges := []Exchange{
		{
			Name:        "binance",
			DisplayName: "Binance",
			Region:      "Global",
			Type:        "SPOT",
			Priority:    1,
			Capacity:    10000.0,
			RestAPI: APIConfig{
				BaseURL:       "https://api.binance.com",
				Version:       "v3",
				Timeout:       5 * time.Second,
				RetryAttempts: 3,
			},
			IsEnabled: true,
		},
		{
			Name:        "okx",
			DisplayName: "OKX",
			Region:      "Global",
			Type:        "SPOT",
			Priority:    2,
			Capacity:    8000.0,
			RestAPI: APIConfig{
				BaseURL:       "https://www.okx.com",
				Version:       "v5",
				Timeout:       5 * time.Second,
				RetryAttempts: 3,
			},
			IsEnabled: true,
		},
		{
			Name:        "bybit",
			DisplayName: "Bybit",
			Region:      "Global",
			Type:        "SPOT",
			Priority:    3,
			Capacity:    6000.0,
			RestAPI: APIConfig{
				BaseURL:       "https://api.bybit.com",
				Version:       "v5",
				Timeout:       5 * time.Second,
				RetryAttempts: 3,
			},
			IsEnabled: true,
		},
	}

	for _, exchange := range exchanges {
		ser.exchangeManager.exchanges[exchange.Name] = &exchange

		// 初始化连接池
		ser.exchangeManager.connectionPool[exchange.Name] = &ConnectionPool{
			MaxConnections:    10,
			ActiveConnections: 0,
			IdleConnections:   0,
			ConnectionTimeout: 30 * time.Second,
			IdleTimeout:       5 * time.Minute,
			MaxLifetime:       1 * time.Hour,
		}

		// 初始化交易所状态
		ser.exchangeStatus[exchange.Name] = &ExchangeStatus{
			Exchange:         exchange.Name,
			IsOnline:         true,
			Latency:          0,
			Availability:     1.0,
			ThroughputLimit:  exchange.Capacity,
			CurrentLoad:      0.0,
			ErrorRate:        0.0,
			ConnectionStatus: "CONNECTED",
			TradingFees:      make(map[string]float64),
			SupportedPairs:   []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"},
			RateLimits:       make(map[string]int),
			HealthScore:      1.0,
			LastUpdated:      time.Now(),
		}

		// 设置负载均衡权重
		ser.loadBalancer.weights[exchange.Name] = 1.0 / float64(exchange.Priority)
	}

	log.Printf("Initialized %d exchanges", len(exchanges))
	return nil
}

// initializeRoutingRules 初始化路由规则
func (ser *SmartExchangeRouter) initializeRoutingRules() {
	rules := []RoutingRule{
		{
			ID:       "primary_exchange",
			Name:     "Primary Exchange Rule",
			Priority: 1,
			Condition: RoutingCondition{
				Type:     "EXCHANGE_HEALTH",
				Operator: "GREATER_THAN",
				Value:    0.9,
			},
			Action: RoutingAction{
				Type:           "ROUTE_TO",
				TargetExchange: ser.primaryExchange,
			},
			IsActive:  true,
			CreatedAt: time.Now(),
		},
		{
			ID:       "failover_rule",
			Name:     "Automatic Failover Rule",
			Priority: 2,
			Condition: RoutingCondition{
				Type:     "EXCHANGE_HEALTH",
				Operator: "LESS_THAN",
				Value:    ser.failoverThreshold,
			},
			Action: RoutingAction{
				Type: "FAILOVER",
				Parameters: map[string]interface{}{
					"backup_exchanges": ser.backupExchanges,
				},
			},
			IsActive:  true,
			CreatedAt: time.Now(),
		},
		{
			ID:       "high_latency_avoidance",
			Name:     "High Latency Avoidance Rule",
			Priority: 3,
			Condition: RoutingCondition{
				Type:     "LATENCY",
				Operator: "GREATER_THAN",
				Value:    ser.latencyThreshold,
			},
			Action: RoutingAction{
				Type: "AVOID",
			},
			IsActive:  true,
			CreatedAt: time.Now(),
		},
	}

	ser.routingRules = rules
	log.Printf("Initialized %d routing rules", len(rules))
}

// runHealthMonitoring 运行健康监控
func (ser *SmartExchangeRouter) runHealthMonitoring() {
	defer ser.wg.Done()

	ticker := time.NewTicker(ser.healthCheckInterval)
	defer ticker.Stop()

	log.Println("Health monitoring started")

	for {
		select {
		case <-ser.ctx.Done():
			log.Println("Health monitoring stopped")
			return
		case <-ticker.C:
			ser.performHealthChecks()
		}
	}
}

// runLoadBalancing 运行负载均衡
func (ser *SmartExchangeRouter) runLoadBalancing() {
	defer ser.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("Load balancing started")

	for {
		select {
		case <-ser.ctx.Done():
			log.Println("Load balancing stopped")
			return
		case <-ticker.C:
			if ser.loadBalancing {
				ser.rebalanceLoad()
			}
		}
	}
}

// runFailoverMonitoring 运行故障转移监控
func (ser *SmartExchangeRouter) runFailoverMonitoring() {
	defer ser.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	log.Println("Failover monitoring started")

	for {
		select {
		case <-ser.ctx.Done():
			log.Println("Failover monitoring stopped")
			return
		case <-ticker.C:
			if ser.autoFailover {
				ser.checkFailoverConditions()
			}
		}
	}
}

// runRoutingOptimization 运行路由优化
func (ser *SmartExchangeRouter) runRoutingOptimization() {
	defer ser.wg.Done()

	ticker := time.NewTicker(ser.routingOptimizer.reoptimizeInterval)
	defer ticker.Stop()

	log.Println("Routing optimization started")

	for {
		select {
		case <-ser.ctx.Done():
			log.Println("Routing optimization stopped")
			return
		case <-ticker.C:
			if ser.smartRouting {
				ser.optimizeRouting()
			}
		}
	}
}

// runPerformanceMonitoring 运行性能监控
func (ser *SmartExchangeRouter) runPerformanceMonitoring() {
	defer ser.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Println("Performance monitoring started")

	for {
		select {
		case <-ser.ctx.Done():
			log.Println("Performance monitoring stopped")
			return
		case <-ticker.C:
			ser.capturePerformanceSnapshot()
		}
	}
}

// runMetricsCollection 运行指标收集
func (ser *SmartExchangeRouter) runMetricsCollection() {
	defer ser.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("Metrics collection started")

	for {
		select {
		case <-ser.ctx.Done():
			log.Println("Metrics collection stopped")
			return
		case <-ticker.C:
			ser.updateMetrics()
		}
	}
}

// RouteOrder 路由订单
func (ser *SmartExchangeRouter) RouteOrder(orderID, symbol, orderType string) (*RoutingDecision, error) {
	startTime := time.Now()

	decision := &RoutingDecision{
		ID:                   ser.generateDecisionID(),
		Timestamp:            startTime,
		OrderID:              orderID,
		Symbol:               symbol,
		OrderType:            orderType,
		AlternativeExchanges: make([]string, 0),
		RuleMatches:          make([]string, 0),
	}

	// 获取可用交易所
	availableExchanges := ser.getAvailableExchanges(symbol)
	if len(availableExchanges) == 0 {
		decision.Success = false
		decision.ErrorMessage = "No available exchanges for symbol"
		return decision, fmt.Errorf("no available exchanges for symbol: %s", symbol)
	}

	// 应用路由规则
	selectedExchange, ruleMatches := ser.applyRoutingRules(symbol, orderType, availableExchanges)
	decision.RuleMatches = ruleMatches

	// 如果规则没有明确指定，使用智能路由
	if selectedExchange == "" && ser.smartRouting {
		selectedExchange = ser.selectOptimalExchange(symbol, orderType, availableExchanges)
		decision.DecisionReason = "Smart routing optimization"
	}

	// 如果仍未选择，使用负载均衡
	if selectedExchange == "" {
		selectedExchange = ser.loadBalancer.selectExchange(availableExchanges)
		decision.DecisionReason = "Load balancing"
	}

	// 如果还是没有选择，使用主交易所
	if selectedExchange == "" {
		selectedExchange = ser.primaryExchange
		decision.DecisionReason = "Default primary exchange"
	}

	decision.SelectedExchange = selectedExchange
	decision.AlternativeExchanges = ser.getAlternatives(selectedExchange, availableExchanges)

	// 计算决策指标
	ser.calculateDecisionScores(decision, availableExchanges)

	// 执行路由
	success, err := ser.executeRouting(decision)
	decision.Success = success
	decision.ExecutionTime = time.Since(startTime)

	if err != nil {
		decision.ErrorMessage = err.Error()
	}

	// 记录决策
	ser.mu.Lock()
	ser.routingHistory = append(ser.routingHistory, *decision)
	if len(ser.routingHistory) > 10000 { // 保持历史记录在合理范围内
		ser.routingHistory = ser.routingHistory[1000:]
	}
	ser.mu.Unlock()

	// 更新统计
	ser.updateRoutingStats(decision)

	log.Printf("Order %s routed to %s (reason: %s)", orderID, selectedExchange, decision.DecisionReason)

	return decision, err
}

// performHealthChecks 执行健康检查
func (ser *SmartExchangeRouter) performHealthChecks() {
	log.Println("Performing health checks...")

	ser.exchangeManager.mu.RLock()
	exchanges := make(map[string]*Exchange)
	for k, v := range ser.exchangeManager.exchanges {
		exchanges[k] = v
	}
	ser.exchangeManager.mu.RUnlock()

	for name, exchange := range exchanges {
		if !exchange.IsEnabled {
			continue
		}

		healthCheck := ser.performSingleHealthCheck(name, exchange)

		// 更新交易所状态
		ser.updateExchangeStatus(name, healthCheck)

		// 记录健康检查历史
		ser.healthMonitor.mu.Lock()
		if ser.healthMonitor.healthHistory[name] == nil {
			ser.healthMonitor.healthHistory[name] = make([]HealthCheck, 0)
		}
		ser.healthMonitor.healthHistory[name] = append(ser.healthMonitor.healthHistory[name], healthCheck)

		// 保持历史记录在合理范围内
		if len(ser.healthMonitor.healthHistory[name]) > 1000 {
			ser.healthMonitor.healthHistory[name] = ser.healthMonitor.healthHistory[name][100:]
		}

		ser.healthMonitor.lastChecks[name] = healthCheck
		ser.healthMonitor.mu.Unlock()
	}
}

// performSingleHealthCheck 执行单个交易所健康检查
func (ser *SmartExchangeRouter) performSingleHealthCheck(name string, exchange *Exchange) HealthCheck {
	startTime := time.Now()

	healthCheck := HealthCheck{
		Exchange:   name,
		Timestamp:  startTime,
		Components: make(map[string]float64),
	}

	// Ping测试
	pingResult := ser.performPingTest(exchange)
	healthCheck.PingTest = pingResult
	healthCheck.Components["ping"] = ser.getTestScore(pingResult)

	// API测试
	apiResult := ser.performAPITest(exchange)
	healthCheck.APITest = apiResult
	healthCheck.Components["api"] = ser.getTestScore(apiResult)

	// WebSocket测试
	wsResult := ser.performWebSocketTest(exchange)
	healthCheck.WebSocketTest = wsResult
	healthCheck.Components["websocket"] = ser.getTestScore(wsResult)

	// 订单簿测试
	obResult := ser.performOrderBookTest(exchange)
	healthCheck.OrderBookTest = obResult
	healthCheck.Components["orderbook"] = ser.getTestScore(obResult)

	// 计算总体健康分数
	totalScore := 0.0
	for _, score := range healthCheck.Components {
		totalScore += score
	}
	healthCheck.HealthScore = totalScore / float64(len(healthCheck.Components))

	// 确定是否健康
	healthCheck.IsHealthy = healthCheck.HealthScore >= ser.healthMonitor.healthThreshold
	healthCheck.Latency = time.Since(startTime)

	if !healthCheck.IsHealthy {
		healthCheck.ErrorMessage = fmt.Sprintf("Health score %.2f below threshold %.2f",
			healthCheck.HealthScore, ser.healthMonitor.healthThreshold)
	}

	return healthCheck
}

// Helper functions for health checks
func (ser *SmartExchangeRouter) performPingTest(exchange *Exchange) TestResult {
	// 实现实际的ping测试
	startTime := time.Now()

	// 使用 HTTP HEAD 请求测试连接性
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("HEAD", exchange.RestAPI.BaseURL, nil)
	if err != nil {
		return TestResult{
			Passed:   false,
			Duration: time.Since(startTime),
			Details: map[string]interface{}{
				"error": fmt.Sprintf("Failed to create ping request: %v", err),
			},
		}
	}

	resp, err := client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		return TestResult{
			Passed:   false,
			Duration: duration,
			Details: map[string]interface{}{
				"error": fmt.Sprintf("Ping failed: %v", err),
			},
		}
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		return TestResult{
			Passed:   true,
			Duration: duration,
			Details: map[string]interface{}{
				"status_code": resp.StatusCode,
				"message":     fmt.Sprintf("Ping successful (status: %d)", resp.StatusCode),
			},
		}
	}

	return TestResult{
		Passed:   false,
		Duration: duration,
		Details: map[string]interface{}{
			"status_code": resp.StatusCode,
			"error":       fmt.Sprintf("Ping failed with status: %d", resp.StatusCode),
		},
	}
}

func (ser *SmartExchangeRouter) performAPITest(exchange *Exchange) TestResult {
	// 实现实际的API测试
	startTime := time.Now()

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 构建测试 API 端点（通常是获取服务器时间或状态的端点）
	testURL := exchange.RestAPI.BaseURL
	if exchange.RestAPI.Endpoints != nil {
		if timeEndpoint, exists := exchange.RestAPI.Endpoints["time"]; exists {
			testURL = exchange.RestAPI.BaseURL + timeEndpoint
		} else if statusEndpoint, exists := exchange.RestAPI.Endpoints["status"]; exists {
			testURL = exchange.RestAPI.BaseURL + statusEndpoint
		} else {
			// 使用通用的健康检查端点
			testURL = exchange.RestAPI.BaseURL + "/api/v1/time"
		}
	}

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return TestResult{
			Passed:   false,
			Duration: time.Since(startTime),
			Details: map[string]interface{}{
				"error": fmt.Sprintf("Failed to create API request: %v", err),
				"url":   testURL,
			},
		}
	}

	// 添加必要的请求头
	req.Header.Set("User-Agent", "QCAT-Router/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		return TestResult{
			Passed:   false,
			Duration: duration,
			Details: map[string]interface{}{
				"error": fmt.Sprintf("API test failed: %v", err),
				"url":   testURL,
			},
		}
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return TestResult{
			Passed:   true,
			Duration: duration,
			Details: map[string]interface{}{
				"status_code": resp.StatusCode,
				"url":         testURL,
				"message":     fmt.Sprintf("API test successful (status: %d)", resp.StatusCode),
			},
		}
	}

	return TestResult{
		Passed:   false,
		Duration: duration,
		Details: map[string]interface{}{
			"status_code": resp.StatusCode,
			"url":         testURL,
			"error":       fmt.Sprintf("API test failed with status: %d", resp.StatusCode),
		},
	}
}

func (ser *SmartExchangeRouter) performWebSocketTest(exchange *Exchange) TestResult {
	// 实现实际的WebSocket测试
	startTime := time.Now()

	// 由于 WebSocket 测试比较复杂，这里实现一个简化版本
	// 实际生产环境中应该使用 gorilla/websocket 或类似库

	// 检查 WebSocket URL 是否配置
	wsURL := exchange.WebSocketAPI.BaseURL
	if wsURL == "" {
		return TestResult{
			Passed:   false,
			Duration: time.Since(startTime),
			Details: map[string]interface{}{
				"error": "WebSocket URL not configured",
			},
		}
	}

	// 简化的连接测试 - 使用 HTTP 升级请求模拟
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// 将 wss:// 转换为 https:// 进行基本连接测试
	testURL := wsURL
	if len(testURL) > 6 && testURL[:6] == "wss://" {
		testURL = "https://" + testURL[6:]
	} else if len(testURL) > 5 && testURL[:5] == "ws://" {
		testURL = "http://" + testURL[5:]
	}

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return TestResult{
			Passed:   false,
			Duration: time.Since(startTime),
			Details: map[string]interface{}{
				"error": fmt.Sprintf("Failed to create WebSocket test request: %v", err),
				"url":   wsURL,
			},
		}
	}

	// 添加 WebSocket 升级头
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	resp, err := client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		return TestResult{
			Passed:   false,
			Duration: duration,
			Details: map[string]interface{}{
				"error": fmt.Sprintf("WebSocket test failed: %v", err),
				"url":   wsURL,
			},
		}
	}
	defer resp.Body.Close()

	// WebSocket 升级成功的状态码是 101，但我们也接受其他成功状态
	if resp.StatusCode == 101 || (resp.StatusCode >= 200 && resp.StatusCode < 400) {
		return TestResult{
			Passed:   true,
			Duration: duration,
			Details: map[string]interface{}{
				"status_code": resp.StatusCode,
				"url":         wsURL,
				"message":     fmt.Sprintf("WebSocket test successful (status: %d)", resp.StatusCode),
			},
		}
	}

	return TestResult{
		Passed:   false,
		Duration: duration,
		Details: map[string]interface{}{
			"status_code": resp.StatusCode,
			"url":         wsURL,
			"error":       fmt.Sprintf("WebSocket test failed with status: %d", resp.StatusCode),
		},
	}
}

func (ser *SmartExchangeRouter) performOrderBookTest(exchange *Exchange) TestResult {
	// 实现实际的订单簿测试
	startTime := time.Now()

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 构建订单簿 API 端点
	testURL := exchange.RestAPI.BaseURL
	testSymbol := "BTCUSDT" // 使用常见的交易对进行测试

	if exchange.RestAPI.Endpoints != nil {
		if depthEndpoint, exists := exchange.RestAPI.Endpoints["depth"]; exists {
			testURL = exchange.RestAPI.BaseURL + depthEndpoint
		} else if orderbookEndpoint, exists := exchange.RestAPI.Endpoints["orderbook"]; exists {
			testURL = exchange.RestAPI.BaseURL + orderbookEndpoint
		} else {
			// 使用通用的订单簿端点
			testURL = exchange.RestAPI.BaseURL + "/api/v1/depth"
		}
	} else {
		testURL = exchange.RestAPI.BaseURL + "/api/v1/depth"
	}

	// 添加查询参数
	testURL += "?symbol=" + testSymbol + "&limit=5"

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return TestResult{
			Passed:   false,
			Duration: time.Since(startTime),
			Details: map[string]interface{}{
				"error":  fmt.Sprintf("Failed to create order book request: %v", err),
				"url":    testURL,
				"symbol": testSymbol,
			},
		}
	}

	// 添加必要的请求头
	req.Header.Set("User-Agent", "QCAT-Router/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		return TestResult{
			Passed:   false,
			Duration: duration,
			Details: map[string]interface{}{
				"error":  fmt.Sprintf("Order book test failed: %v", err),
				"url":    testURL,
				"symbol": testSymbol,
			},
		}
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return TestResult{
			Passed:   true,
			Duration: duration,
			Details: map[string]interface{}{
				"status_code": resp.StatusCode,
				"url":         testURL,
				"symbol":      testSymbol,
				"message":     fmt.Sprintf("Order book test successful (status: %d)", resp.StatusCode),
			},
		}
	}

	return TestResult{
		Passed:   false,
		Duration: duration,
		Details: map[string]interface{}{
			"status_code": resp.StatusCode,
			"url":         testURL,
			"symbol":      testSymbol,
			"error":       fmt.Sprintf("Order book test failed with status: %d", resp.StatusCode),
		},
	}
}

func (ser *SmartExchangeRouter) getTestScore(result TestResult) float64 {
	if result.Passed {
		// 基于延迟计算分数
		latencyMs := float64(result.Duration.Milliseconds())
		if latencyMs <= 100 {
			return 1.0
		} else if latencyMs <= 500 {
			return 1.0 - (latencyMs-100)/400*0.5 // 100-500ms线性递减到0.5
		} else {
			return 0.5 - math.Min((latencyMs-500)/1000*0.5, 0.5) // 500ms以上继续递减
		}
	}
	return 0.0
}

// updateExchangeStatus 更新交易所状态
func (ser *SmartExchangeRouter) updateExchangeStatus(name string, healthCheck HealthCheck) {
	ser.mu.Lock()
	defer ser.mu.Unlock()

	status := ser.exchangeStatus[name]
	if status == nil {
		status = &ExchangeStatus{Exchange: name}
		ser.exchangeStatus[name] = status
	}

	status.IsOnline = healthCheck.IsHealthy
	status.Latency = healthCheck.Latency
	status.HealthScore = healthCheck.HealthScore
	status.LastPing = healthCheck.Timestamp
	status.PingSuccess = healthCheck.IsHealthy

	if !healthCheck.IsHealthy {
		status.ConsecutiveFailures++
		status.ConnectionStatus = "ERROR"
	} else {
		status.ConsecutiveFailures = 0
		status.ConnectionStatus = "CONNECTED"
	}

	// 更新可用性统计
	ser.updateAvailabilityStats(status)

	status.LastUpdated = time.Now()
}

// rebalanceLoad 重新平衡负载
func (ser *SmartExchangeRouter) rebalanceLoad() {
	log.Println("Rebalancing load across exchanges...")

	// 获取当前负载情况
	loads := ser.getCurrentLoads()

	// 计算理想负载分布
	idealLoads := ser.calculateIdealLoads()

	// 调整权重
	ser.loadBalancer.mu.Lock()
	for exchange, idealLoad := range idealLoads {
		currentLoad := loads[exchange]
		if math.Abs(currentLoad-idealLoad) > 0.1 { // 10%的偏差阈值
			adjustment := (idealLoad - currentLoad) * 0.1 // 渐进调整
			ser.loadBalancer.weights[exchange] += adjustment
			ser.loadBalancer.weights[exchange] = math.Max(0.1, ser.loadBalancer.weights[exchange])
		}
	}
	ser.loadBalancer.mu.Unlock()
}

// checkFailoverConditions 检查故障转移条件
func (ser *SmartExchangeRouter) checkFailoverConditions() {
	ser.mu.RLock()
	statuses := make(map[string]*ExchangeStatus)
	for k, v := range ser.exchangeStatus {
		statuses[k] = v
	}
	ser.mu.RUnlock()

	for exchange, status := range statuses {
		// 检查是否需要故障转移
		if ser.shouldFailover(exchange, status) {
			ser.performFailover(exchange, status)
		}

		// 检查是否可以恢复
		if ser.shouldRecover(exchange, status) {
			ser.performRecovery(exchange, status)
		}
	}
}

// shouldFailover 判断是否应该故障转移
func (ser *SmartExchangeRouter) shouldFailover(exchange string, status *ExchangeStatus) bool {
	// 检查健康分数
	if status.HealthScore < ser.failoverController.failoverThreshold {
		return true
	}

	// 检查连续失败次数
	if status.ConsecutiveFailures >= 3 {
		return true
	}

	// 检查延迟
	if status.Latency > ser.latencyThreshold*2 {
		return true
	}

	// 检查错误率
	if status.ErrorRate > 0.1 { // 10%错误率
		return true
	}

	return false
}

// performFailover 执行故障转移
func (ser *SmartExchangeRouter) performFailover(fromExchange string, status *ExchangeStatus) {
	ser.failoverController.mu.Lock()
	defer ser.failoverController.mu.Unlock()

	// 检查冷却期
	if time.Since(ser.failoverController.lastFailover) < ser.failoverController.failoverCooldown {
		return
	}

	// 检查最大故障转移次数
	if ser.failoverController.failoverCount >= ser.failoverController.maxFailovers {
		log.Printf("Maximum failover limit reached for %s", fromExchange)
		return
	}

	// 选择目标交易所
	toExchange := ser.selectFailoverTarget(fromExchange)
	if toExchange == "" {
		log.Printf("No suitable failover target found for %s", fromExchange)
		return
	}

	log.Printf("Performing failover from %s to %s", fromExchange, toExchange)

	// 创建故障转移事件
	failoverEvent := FailoverEvent{
		ID:           ser.generateFailoverID(),
		Timestamp:    time.Now(),
		FromExchange: fromExchange,
		ToExchange:   toExchange,
		Trigger:      "HEALTH_CHECK",
		TriggerValue: status.HealthScore,
		Reason:       fmt.Sprintf("Health score %.2f below threshold", status.HealthScore),
		Success:      true,
	}

	// 执行故障转移逻辑
	err := ser.executeFailover(fromExchange, toExchange)
	if err != nil {
		failoverEvent.Success = false
		log.Printf("Failover failed: %v", err)
		return
	}

	failoverEvent.Duration = time.Since(failoverEvent.Timestamp)

	// 记录故障转移事件
	ser.failoverController.failoverHistory = append(ser.failoverController.failoverHistory, failoverEvent)
	ser.failoverController.lastFailover = time.Now()
	ser.failoverController.failoverCount++

	// 更新状态
	status.LastFailover = time.Now()

	log.Printf("Failover completed successfully from %s to %s", fromExchange, toExchange)
}

// optimizeRouting 优化路由
func (ser *SmartExchangeRouter) optimizeRouting() {
	log.Println("Optimizing routing configuration...")

	startTime := time.Now()

	// 获取当前性能数据
	currentMetrics := ser.getCurrentOptimizationMetrics()

	// 根据优化模型计算最优路由
	optimalRouting := ser.calculateOptimalRouting(currentMetrics)

	// 应用优化结果
	ser.applyOptimizationResult(optimalRouting)

	// 记录优化历史
	result := OptimizationResult{
		Timestamp:         startTime,
		OptimizationModel: ser.routingOptimizer.optimizationModel,
		OptimalRouting:    optimalRouting,
		Metrics:           currentMetrics,
	}

	ser.routingOptimizer.mu.Lock()
	ser.routingOptimizer.optimizationHistory = append(ser.routingOptimizer.optimizationHistory, result)
	ser.routingOptimizer.mu.Unlock()

	log.Printf("Routing optimization completed in %v", time.Since(startTime))
}

// capturePerformanceSnapshot 捕获性能快照
func (ser *SmartExchangeRouter) capturePerformanceSnapshot() {
	snapshot := PerformanceSnapshot{
		Timestamp:           time.Now(),
		ExchangePerformance: make(map[string]ExchangePerformance),
		SystemLoad:          ser.calculateSystemLoad(),
	}

	// 收集各交易所性能数据
	ser.mu.RLock()
	for exchange, status := range ser.exchangeStatus {
		performance := ExchangePerformance{
			Latency:               status.Latency,
			Availability:          status.Availability,
			ThroughputUtilization: status.CurrentLoad / status.ThroughputLimit,
			ErrorRate:             status.ErrorRate,
			HealthScore:           status.HealthScore,
			OrderSuccessRate:      ser.calculateOrderSuccessRate(exchange),
		}
		snapshot.ExchangePerformance[exchange] = performance
	}
	ser.mu.RUnlock()

	// 计算路由质量
	snapshot.RoutingQuality = ser.calculateRoutingQuality()

	// 统计故障转移事件
	snapshot.FailoverEvents = ser.countRecentFailovers(1 * time.Hour)

	// 保存快照
	ser.mu.Lock()
	ser.performanceHistory = append(ser.performanceHistory, snapshot)
	if len(ser.performanceHistory) > 1000 {
		ser.performanceHistory = ser.performanceHistory[100:]
	}
	ser.mu.Unlock()
}

// updateMetrics 更新指标
func (ser *SmartExchangeRouter) updateMetrics() {
	ser.routingMetrics.mu.Lock()
	defer ser.routingMetrics.mu.Unlock()

	// 更新路由统计
	totalRequests := int64(len(ser.routingHistory))
	successfulRoutes := int64(0)

	latencies := make([]time.Duration, 0)

	for _, decision := range ser.routingHistory {
		if decision.Success {
			successfulRoutes++
		}
		latencies = append(latencies, decision.ExecutionTime)

		// 更新交易所分布
		ser.routingMetrics.ExchangeDistribution[decision.SelectedExchange]++
	}

	ser.routingMetrics.TotalRequests = totalRequests
	ser.routingMetrics.SuccessfulRoutes = successfulRoutes
	ser.routingMetrics.FailedRoutes = totalRequests - successfulRoutes

	if totalRequests > 0 {
		ser.routingMetrics.SuccessRate = float64(successfulRoutes) / float64(totalRequests)
	}

	// 计算延迟统计
	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})

		ser.routingMetrics.AvgRoutingLatency = ser.calculateMeanDuration(latencies)
		ser.routingMetrics.P95Latency = latencies[int(float64(len(latencies))*0.95)]
		ser.routingMetrics.P99Latency = latencies[int(float64(len(latencies))*0.99)]
	}

	// 更新故障转移统计
	ser.routingMetrics.FailoverCount = int64(len(ser.failoverController.failoverHistory))
	if ser.routingMetrics.FailoverCount > 0 {
		totalFailoverTime := time.Duration(0)
		autoRecoveries := int64(0)

		for _, event := range ser.failoverController.failoverHistory {
			totalFailoverTime += event.Duration
			if event.AutoRecovery {
				autoRecoveries++
			}
		}

		ser.routingMetrics.AvgFailoverTime = totalFailoverTime / time.Duration(ser.routingMetrics.FailoverCount)
		ser.routingMetrics.AutoRecoveryRate = float64(autoRecoveries) / float64(ser.routingMetrics.FailoverCount)
	}

	// 更新路由质量
	ser.routingMetrics.RouteQuality = ser.calculateRoutingQuality()
	ser.routingMetrics.OptimizationEfficiency = ser.calculateOptimizationEfficiency()

	ser.routingMetrics.LastUpdated = time.Now()
}

// Helper functions implementation...

func (ser *SmartExchangeRouter) getAvailableExchanges(symbol string) []string {
	available := make([]string, 0)

	ser.mu.RLock()
	defer ser.mu.RUnlock()

	for exchange, status := range ser.exchangeStatus {
		if status.IsOnline && status.ConnectionStatus == "CONNECTED" {
			// 检查是否支持该交易对
			for _, pair := range status.SupportedPairs {
				if pair == symbol {
					available = append(available, exchange)
					break
				}
			}
		}
	}

	return available
}

func (ser *SmartExchangeRouter) applyRoutingRules(symbol, orderType string, available []string) (string, []string) {
	matches := make([]string, 0)

	// 按优先级排序规则
	rules := make([]RoutingRule, len(ser.routingRules))
	copy(rules, ser.routingRules)
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})

	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}

		if ser.evaluateCondition(rule.Condition, symbol, orderType) {
			matches = append(matches, rule.ID)

			// 应用动作
			switch rule.Action.Type {
			case "ROUTE_TO":
				if ser.isExchangeAvailable(rule.Action.TargetExchange, available) {
					return rule.Action.TargetExchange, matches
				}
			case "AVOID":
				// 从可用列表中移除
				available = ser.removeExchange(available, rule.Action.TargetExchange)
			case "FAILOVER":
				// 触发故障转移逻辑
				return ser.selectFailoverTarget(""), matches
			}
		}
	}

	return "", matches
}

func (ser *SmartExchangeRouter) selectOptimalExchange(symbol, orderType string, available []string) string {
	if len(available) == 0 {
		return ""
	}

	if len(available) == 1 {
		return available[0]
	}

	// 计算每个交易所的综合评分
	scores := make(map[string]float64)

	for _, exchange := range available {
		score := ser.calculateExchangeScore(exchange, symbol, orderType)
		scores[exchange] = score
	}

	// 选择评分最高的交易所
	bestExchange := ""
	bestScore := math.Inf(-1)

	for exchange, score := range scores {
		if score > bestScore {
			bestScore = score
			bestExchange = exchange
		}
	}

	return bestExchange
}

func (ser *SmartExchangeRouter) calculateExchangeScore(exchange, symbol, orderType string) float64 {
	status := ser.exchangeStatus[exchange]
	if status == nil {
		return 0.0
	}

	// 延迟评分 (越低越好)
	latencyScore := 1.0 - math.Min(float64(status.Latency.Milliseconds())/1000.0, 1.0)

	// 可用性评分
	availabilityScore := status.Availability

	// 健康评分
	healthScore := status.HealthScore

	// 负载评分 (越低越好)
	loadScore := 1.0 - (status.CurrentLoad / status.ThroughputLimit)

	// 加权计算综合评分
	weights := ser.routingOptimizer
	totalScore := latencyScore*weights.latencyWeight +
		availabilityScore*weights.reliabilityWeight +
		healthScore*0.25 +
		loadScore*0.25

	return totalScore
}

func (ser *SmartExchangeRouter) getAlternatives(selected string, available []string) []string {
	alternatives := make([]string, 0)
	for _, exchange := range available {
		if exchange != selected {
			alternatives = append(alternatives, exchange)
		}
	}
	return alternatives
}

func (ser *SmartExchangeRouter) calculateDecisionScores(decision *RoutingDecision, available []string) {
	if status := ser.exchangeStatus[decision.SelectedExchange]; status != nil {
		decision.LatencyScore = 1.0 - math.Min(float64(status.Latency.Milliseconds())/1000.0, 1.0)
		decision.ReliabilityScore = status.HealthScore
		decision.CostScore = 1.0 - (status.TradingFees["taker"] * 10) // 简化成本计算
		decision.LiquidityScore = status.OrderBookDepth / 10000.0     // 简化流动性计算

		decision.OverallScore = (decision.LatencyScore + decision.ReliabilityScore +
			decision.CostScore + decision.LiquidityScore) / 4.0
	}
}

func (ser *SmartExchangeRouter) executeRouting(decision *RoutingDecision) (bool, error) {
	// 实现实际的订单路由执行
	log.Printf("Executing routing for order %s to %s", decision.OrderID, decision.SelectedExchange)

	startTime := time.Now()

	// 检查目标交易所是否可用
	status, exists := ser.exchangeStatus[decision.SelectedExchange]
	if !exists {
		return false, fmt.Errorf("exchange %s not found", decision.SelectedExchange)
	}

	if !status.IsOnline {
		return false, fmt.Errorf("exchange %s is offline", decision.SelectedExchange)
	}

	// 检查交易所健康状态
	if status.HealthScore < ser.failoverThreshold {
		log.Printf("Warning: routing to exchange %s with low health score: %.2f",
			decision.SelectedExchange, status.HealthScore)
	}

	// 更新交易所负载
	ser.mu.Lock()
	status.CurrentLoad += 1.0
	ser.mu.Unlock()

	// 模拟路由执行过程
	// 在实际实现中，这里会调用具体的交易所 API
	executionLatency := time.Duration(50+rand.Intn(100)) * time.Millisecond
	time.Sleep(executionLatency) // 模拟网络延迟

	// 计算实际成本（基于交易所费率）
	actualCost := 0.001 // 默认成本
	if fees, exists := status.TradingFees["maker"]; exists {
		actualCost = fees
	}

	// 更新决策结果
	decision.ActualLatency = time.Since(startTime)
	decision.ActualCost = actualCost

	// 更新交易所负载（执行完成后减少负载）
	ser.mu.Lock()
	status.CurrentLoad = math.Max(0, status.CurrentLoad-1.0)
	status.LastUpdated = time.Now()
	ser.mu.Unlock()

	// 记录执行成功
	log.Printf("Successfully routed order %s to %s (latency: %v, cost: %.4f)",
		decision.OrderID, decision.SelectedExchange, decision.ActualLatency, decision.ActualCost)

	return true, nil
}

func (ser *SmartExchangeRouter) updateRoutingStats(decision *RoutingDecision) {
	// 更新交易所使用统计
	ser.loadBalancer.mu.Lock()
	ser.loadBalancer.connections[decision.SelectedExchange]++
	ser.loadBalancer.mu.Unlock()

	// 更新规则命中统计
	for _, ruleID := range decision.RuleMatches {
		for i := range ser.routingRules {
			if ser.routingRules[i].ID == ruleID {
				ser.routingRules[i].HitCount++
				if decision.Success {
					ser.routingRules[i].SuccessCount++
				}
				break
			}
		}
	}
}

// 其他辅助函数的简化实现...
func (ser *SmartExchangeRouter) updateAvailabilityStats(status *ExchangeStatus) {
	// 实现可用性统计更新
	ser.routingMetrics.mu.Lock()
	defer ser.routingMetrics.mu.Unlock()

	// 更新交易所成功率统计
	if status.IsOnline && status.HealthScore > ser.failoverThreshold {
		// 交易所可用
		if currentRate, exists := ser.routingMetrics.ExchangeSuccessRates[status.Exchange]; exists {
			// 使用指数移动平均更新成功率
			alpha := 0.1 // 平滑因子
			ser.routingMetrics.ExchangeSuccessRates[status.Exchange] = alpha*1.0 + (1-alpha)*currentRate
		} else {
			ser.routingMetrics.ExchangeSuccessRates[status.Exchange] = 1.0
		}
	} else {
		// 交易所不可用
		if currentRate, exists := ser.routingMetrics.ExchangeSuccessRates[status.Exchange]; exists {
			alpha := 0.1
			ser.routingMetrics.ExchangeSuccessRates[status.Exchange] = alpha*0.0 + (1-alpha)*currentRate
		} else {
			ser.routingMetrics.ExchangeSuccessRates[status.Exchange] = 0.0
		}
	}

	// 更新延迟统计
	if status.Latency > 0 {
		ser.routingMetrics.ExchangeLatencies[status.Exchange] = status.Latency
	}

	// 计算整体成功率
	totalRate := 0.0
	count := 0
	for _, rate := range ser.routingMetrics.ExchangeSuccessRates {
		totalRate += rate
		count++
	}

	if count > 0 {
		ser.routingMetrics.SuccessRate = totalRate / float64(count)
	}

	// 更新统计时间戳
	ser.routingMetrics.LastUpdated = time.Now()

	log.Printf("Updated availability stats for %s: success_rate=%.3f, latency=%v, health_score=%.3f",
		status.Exchange, ser.routingMetrics.ExchangeSuccessRates[status.Exchange],
		status.Latency, status.HealthScore)
}

func (ser *SmartExchangeRouter) getCurrentLoads() map[string]float64 {
	loads := make(map[string]float64)
	for exchange, status := range ser.exchangeStatus {
		loads[exchange] = status.CurrentLoad / status.ThroughputLimit
	}
	return loads
}

func (ser *SmartExchangeRouter) calculateIdealLoads() map[string]float64 {
	// 基于容量和权重计算理想负载分布
	idealLoads := make(map[string]float64)

	// 获取所有交易所的容量和健康状态
	totalCapacity := 0.0
	exchangeCapacities := make(map[string]float64)

	ser.mu.RLock()
	for exchangeName, status := range ser.exchangeStatus {
		if !status.IsOnline || status.HealthScore < 0.5 {
			// 不健康的交易所不参与负载分布
			continue
		}

		// 获取交易所配置
		if exchange, exists := ser.exchangeManager.exchanges[exchangeName]; exists {
			// 基于容量和健康状态计算有效容量
			effectiveCapacity := exchange.Capacity * status.HealthScore * status.Availability
			exchangeCapacities[exchangeName] = effectiveCapacity
			totalCapacity += effectiveCapacity
		}
	}
	ser.mu.RUnlock()

	// 如果没有可用的交易所，返回空分布
	if totalCapacity == 0 {
		return idealLoads
	}

	// 计算每个交易所的理想负载比例
	for exchangeName, capacity := range exchangeCapacities {
		// 基础负载比例 = 容量 / 总容量
		baseRatio := capacity / totalCapacity

		// 考虑负载均衡权重
		ser.loadBalancer.mu.RLock()
		weight := ser.loadBalancer.weights[exchangeName]
		if weight == 0 {
			weight = 1.0 // 默认权重
		}
		ser.loadBalancer.mu.RUnlock()

		// 调整后的理想负载
		idealLoads[exchangeName] = baseRatio * weight
	}

	// 归一化负载分布，确保总和为 1.0
	totalIdealLoad := 0.0
	for _, load := range idealLoads {
		totalIdealLoad += load
	}

	if totalIdealLoad > 0 {
		for exchangeName := range idealLoads {
			idealLoads[exchangeName] /= totalIdealLoad
		}
	}

	log.Printf("Calculated ideal loads: %+v", idealLoads)
	return idealLoads
}

func (ser *SmartExchangeRouter) shouldRecover(exchange string, status *ExchangeStatus) bool {
	return status.HealthScore >= ser.failoverController.recoveryThreshold &&
		status.ConsecutiveFailures == 0
}

func (ser *SmartExchangeRouter) performRecovery(exchange string, status *ExchangeStatus) {
	// 实现恢复逻辑
	log.Printf("Starting recovery process for exchange: %s", exchange)

	// 重置故障计数器
	ser.mu.Lock()
	status.ConsecutiveFailures = 0
	status.ErrorRate = 0.0
	status.IsOnline = true
	ser.mu.Unlock()

	// 执行健康检查以确认恢复
	if exchangeConfig, exists := ser.exchangeManager.exchanges[exchange]; exists {
		healthCheck := ser.performSingleHealthCheck(exchange, exchangeConfig)

		ser.mu.Lock()
		status.HealthScore = healthCheck.HealthScore
		status.Latency = healthCheck.Latency
		status.LastUpdated = time.Now()

		if healthCheck.IsHealthy {
			// 恢复成功
			status.ConnectionStatus = "CONNECTED"
			log.Printf("Exchange %s successfully recovered (health score: %.3f)",
				exchange, healthCheck.HealthScore)

			// 逐步恢复负载均衡权重
			ser.loadBalancer.mu.Lock()
			if currentWeight, exists := ser.loadBalancer.weights[exchange]; exists {
				// 从当前权重的 50% 开始恢复
				ser.loadBalancer.weights[exchange] = math.Max(0.5, currentWeight)
			} else {
				ser.loadBalancer.weights[exchange] = 0.5
			}
			ser.loadBalancer.mu.Unlock()

			// 记录恢复事件
			ser.routingMetrics.mu.Lock()
			ser.routingMetrics.AutoRecoveryRate = (ser.routingMetrics.AutoRecoveryRate*0.9 + 1.0*0.1)
			ser.routingMetrics.mu.Unlock()

		} else {
			// 恢复失败，保持离线状态
			status.IsOnline = false
			status.ConnectionStatus = "DISCONNECTED"
			log.Printf("Exchange %s recovery failed (health score: %.3f)",
				exchange, healthCheck.HealthScore)
		}
		ser.mu.Unlock()
	}

	// 更新可用性统计
	ser.updateAvailabilityStats(status)

	log.Printf("Recovery process completed for exchange: %s (online: %v, health: %.3f)",
		exchange, status.IsOnline, status.HealthScore)
}

func (ser *SmartExchangeRouter) selectFailoverTarget(fromExchange string) string {
	// 选择可用性最高的备用交易所
	bestTarget := ""
	bestScore := 0.0

	for _, backup := range ser.backupExchanges {
		if backup == fromExchange {
			continue
		}

		if status := ser.exchangeStatus[backup]; status != nil && status.IsOnline {
			if status.HealthScore > bestScore {
				bestScore = status.HealthScore
				bestTarget = backup
			}
		}
	}

	return bestTarget
}

func (ser *SmartExchangeRouter) executeFailover(from, to string) error {
	// 实现实际的故障转移逻辑
	log.Printf("Executing failover from %s to %s", from, to)

	// 验证目标交易所是否可用
	ser.mu.RLock()
	toStatus, toExists := ser.exchangeStatus[to]
	fromStatus, fromExists := ser.exchangeStatus[from]
	ser.mu.RUnlock()

	if !toExists {
		return fmt.Errorf("target exchange %s not found", to)
	}

	if !toStatus.IsOnline {
		return fmt.Errorf("target exchange %s is offline", to)
	}

	if toStatus.HealthScore < ser.failoverThreshold {
		return fmt.Errorf("target exchange %s health score %.3f below threshold %.3f",
			to, toStatus.HealthScore, ser.failoverThreshold)
	}

	// 更新源交易所状态
	if fromExists {
		ser.mu.Lock()
		fromStatus.IsOnline = false
		fromStatus.ConnectionStatus = "FAILED_OVER"
		fromStatus.LastFailover = time.Now()
		ser.mu.Unlock()

		// 将源交易所的负载权重设为 0
		ser.loadBalancer.mu.Lock()
		ser.loadBalancer.weights[from] = 0.0
		ser.loadBalancer.mu.Unlock()
	}

	// 增加目标交易所的负载权重
	ser.loadBalancer.mu.Lock()
	if currentWeight, exists := ser.loadBalancer.weights[to]; exists {
		// 增加权重以承担更多负载
		ser.loadBalancer.weights[to] = math.Min(2.0, currentWeight*1.5)
	} else {
		ser.loadBalancer.weights[to] = 1.5
	}
	ser.loadBalancer.mu.Unlock()

	// 更新故障转移统计
	ser.routingMetrics.mu.Lock()
	ser.routingMetrics.FailoverCount++
	ser.routingMetrics.mu.Unlock()

	// 记录故障转移事件
	log.Printf("Failover completed: %s -> %s (target health: %.3f, new weight: %.2f)",
		from, to, toStatus.HealthScore, ser.loadBalancer.weights[to])

	return nil
}

func (ser *SmartExchangeRouter) getCurrentOptimizationMetrics() OptimizationMetrics {
	// 计算当前优化指标
	ser.routingMetrics.mu.RLock()
	defer ser.routingMetrics.mu.RUnlock()

	// 计算平均延迟
	avgLatency := ser.routingMetrics.AvgRoutingLatency
	if avgLatency == 0 {
		// 如果没有历史数据，从交易所状态计算
		totalLatency := time.Duration(0)
		count := 0

		ser.mu.RLock()
		for _, status := range ser.exchangeStatus {
			if status.IsOnline && status.Latency > 0 {
				totalLatency += status.Latency
				count++
			}
		}
		ser.mu.RUnlock()

		if count > 0 {
			avgLatency = totalLatency / time.Duration(count)
		}
	}

	// 计算总成本
	totalCost := ser.routingMetrics.TotalTradingCosts
	if totalCost == 0 {
		// 基于交易所费率估算
		ser.mu.RLock()
		for _, status := range ser.exchangeStatus {
			if status.IsOnline {
				for _, fee := range status.TradingFees {
					totalCost += fee
				}
			}
		}
		ser.mu.RUnlock()
	}

	// 计算流动性评分（基于订单簿深度）
	liquidityScore := 0.0
	liquidityCount := 0
	ser.mu.RLock()
	for _, status := range ser.exchangeStatus {
		if status.IsOnline && status.OrderBookDepth > 0 {
			liquidityScore += status.OrderBookDepth
			liquidityCount++
		}
	}
	ser.mu.RUnlock()

	if liquidityCount > 0 {
		liquidityScore = liquidityScore / float64(liquidityCount) / 10000.0 // 归一化
		liquidityScore = math.Min(1.0, liquidityScore)
	}

	// 计算可靠性评分（基于健康分数）
	reliabilityScore := 0.0
	reliabilityCount := 0
	ser.mu.RLock()
	for _, status := range ser.exchangeStatus {
		if status.IsOnline {
			reliabilityScore += status.HealthScore
			reliabilityCount++
		}
	}
	ser.mu.RUnlock()

	if reliabilityCount > 0 {
		reliabilityScore /= float64(reliabilityCount)
	}

	// 计算吞吐量评分（基于容量利用率）
	throughputScore := 0.0
	throughputCount := 0
	ser.mu.RLock()
	for exchangeName, status := range ser.exchangeStatus {
		if status.IsOnline {
			if exchange, exists := ser.exchangeManager.exchanges[exchangeName]; exists {
				utilization := status.CurrentLoad / exchange.Capacity
				throughputScore += (1.0 - utilization) // 利用率越低，吞吐量评分越高
				throughputCount++
			}
		}
	}
	ser.mu.RUnlock()

	if throughputCount > 0 {
		throughputScore /= float64(throughputCount)
	}

	return OptimizationMetrics{
		AvgLatency:       avgLatency,
		TotalCost:        totalCost,
		LiquidityScore:   liquidityScore,
		ReliabilityScore: reliabilityScore,
		ThroughputScore:  throughputScore,
	}
}

func (ser *SmartExchangeRouter) calculateOptimalRouting(metrics OptimizationMetrics) map[string]float64 {
	// 计算最优路由分布
	optimalRouting := make(map[string]float64)

	// 获取优化权重
	ser.routingOptimizer.mu.RLock()
	latencyWeight := ser.routingOptimizer.latencyWeight
	costWeight := ser.routingOptimizer.costWeight
	liquidityWeight := ser.routingOptimizer.liquidityWeight
	reliabilityWeight := ser.routingOptimizer.reliabilityWeight
	ser.routingOptimizer.mu.RUnlock()

	// 如果权重未设置，使用默认值
	if latencyWeight+costWeight+liquidityWeight+reliabilityWeight == 0 {
		latencyWeight = 0.3
		costWeight = 0.2
		liquidityWeight = 0.2
		reliabilityWeight = 0.3
	}

	// 计算每个交易所的综合评分
	exchangeScores := make(map[string]float64)
	totalScore := 0.0

	ser.mu.RLock()
	for exchangeName, status := range ser.exchangeStatus {
		if !status.IsOnline || status.HealthScore < 0.3 {
			continue // 跳过不健康的交易所
		}

		// 延迟评分（延迟越低评分越高）
		latencyScore := 1.0
		if status.Latency > 0 {
			latencyScore = math.Max(0.1, 1.0-float64(status.Latency)/float64(time.Second))
		}

		// 成本评分（费用越低评分越高）
		costScore := 1.0
		if len(status.TradingFees) > 0 {
			avgFee := 0.0
			for _, fee := range status.TradingFees {
				avgFee += fee
			}
			avgFee /= float64(len(status.TradingFees))
			costScore = math.Max(0.1, 1.0-avgFee*100) // 假设费率在 0-1% 范围内
		}

		// 流动性评分
		liquidityScore := math.Min(1.0, status.OrderBookDepth/10000.0)

		// 可靠性评分
		reliabilityScore := status.HealthScore

		// 综合评分
		compositeScore := latencyWeight*latencyScore +
			costWeight*costScore +
			liquidityWeight*liquidityScore +
			reliabilityWeight*reliabilityScore

		exchangeScores[exchangeName] = compositeScore
		totalScore += compositeScore
	}
	ser.mu.RUnlock()

	// 归一化评分为路由分布
	if totalScore > 0 {
		for exchangeName, score := range exchangeScores {
			optimalRouting[exchangeName] = score / totalScore
		}
	}

	log.Printf("Calculated optimal routing distribution: %+v", optimalRouting)
	return optimalRouting
}

func (ser *SmartExchangeRouter) applyOptimizationResult(routing map[string]float64) {
	// 应用优化结果
	log.Printf("Applying optimization results: %+v", routing)

	// 更新负载均衡权重
	ser.loadBalancer.mu.Lock()
	defer ser.loadBalancer.mu.Unlock()

	// 平滑地调整权重，避免剧烈变化
	smoothingFactor := 0.3 // 30% 的新权重，70% 的旧权重

	for exchangeName, optimalWeight := range routing {
		currentWeight := ser.loadBalancer.weights[exchangeName]
		if currentWeight == 0 {
			currentWeight = 0.1 // 避免从 0 开始
		}

		// 应用平滑因子
		newWeight := smoothingFactor*optimalWeight + (1-smoothingFactor)*currentWeight

		// 确保权重在合理范围内
		newWeight = math.Max(0.1, math.Min(2.0, newWeight))

		ser.loadBalancer.weights[exchangeName] = newWeight

		log.Printf("Updated weight for %s: %.3f -> %.3f (optimal: %.3f)",
			exchangeName, currentWeight, newWeight, optimalWeight)
	}

	// 确保所有在线交易所都有权重
	ser.mu.RLock()
	for exchangeName, status := range ser.exchangeStatus {
		if status.IsOnline {
			if _, exists := ser.loadBalancer.weights[exchangeName]; !exists {
				ser.loadBalancer.weights[exchangeName] = 0.5 // 默认权重
			}
		}
	}
	ser.mu.RUnlock()

	log.Printf("Applied optimization results. New weights: %+v", ser.loadBalancer.weights)
}

func (ser *SmartExchangeRouter) calculateSystemLoad() float64 {
	// 计算系统负载
	totalLoad := 0.0
	totalCapacity := 0.0

	ser.mu.RLock()
	for exchangeName, status := range ser.exchangeStatus {
		if status.IsOnline {
			if exchange, exists := ser.exchangeManager.exchanges[exchangeName]; exists {
				totalLoad += status.CurrentLoad
				totalCapacity += exchange.Capacity
			}
		}
	}
	ser.mu.RUnlock()

	if totalCapacity == 0 {
		return 0.0
	}

	systemLoad := totalLoad / totalCapacity
	return math.Min(1.0, systemLoad)
}

func (ser *SmartExchangeRouter) calculateOrderSuccessRate(exchange string) float64 {
	// 计算订单成功率
	ser.routingMetrics.mu.RLock()
	defer ser.routingMetrics.mu.RUnlock()

	if rate, exists := ser.routingMetrics.ExchangeSuccessRates[exchange]; exists {
		return rate
	}

	// 如果没有历史数据，基于健康状态估算
	ser.mu.RLock()
	defer ser.mu.RUnlock()

	if status, exists := ser.exchangeStatus[exchange]; exists {
		if status.IsOnline {
			// 基于健康分数和错误率计算成功率
			successRate := status.HealthScore * (1.0 - status.ErrorRate)
			return math.Max(0.0, math.Min(1.0, successRate))
		}
	}

	return 0.0
}

func (ser *SmartExchangeRouter) calculateRoutingQuality() float64 {
	// 计算路由质量
	ser.routingMetrics.mu.RLock()
	defer ser.routingMetrics.mu.RUnlock()

	// 基于多个指标计算路由质量
	successRate := ser.routingMetrics.SuccessRate

	// 延迟质量（延迟越低质量越高）
	latencyQuality := 1.0
	if ser.routingMetrics.AvgRoutingLatency > 0 {
		// 假设 100ms 以下为优秀，500ms 以上为较差
		latencyMs := float64(ser.routingMetrics.AvgRoutingLatency) / float64(time.Millisecond)
		latencyQuality = math.Max(0.1, 1.0-latencyMs/500.0)
	}

	// 成本质量（成本越低质量越高）
	costQuality := 1.0
	if ser.routingMetrics.AvgTradingCost > 0 {
		// 假设 0.1% 以下为优秀，1% 以上为较差
		costQuality = math.Max(0.1, 1.0-ser.routingMetrics.AvgTradingCost*100)
	}

	// 综合路由质量
	routingQuality := 0.5*successRate + 0.3*latencyQuality + 0.2*costQuality

	return math.Max(0.0, math.Min(1.0, routingQuality))
}

func (ser *SmartExchangeRouter) countRecentFailovers(duration time.Duration) int {
	count := 0
	since := time.Now().Add(-duration)

	for _, event := range ser.failoverController.failoverHistory {
		if event.Timestamp.After(since) {
			count++
		}
	}

	return count
}

func (ser *SmartExchangeRouter) calculateMeanDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	total := time.Duration(0)
	for _, d := range durations {
		total += d
	}

	return total / time.Duration(len(durations))
}

func (ser *SmartExchangeRouter) calculateOptimizationEfficiency() float64 {
	// 计算优化效率
	ser.routingMetrics.mu.RLock()
	defer ser.routingMetrics.mu.RUnlock()

	// 基于多个因素计算优化效率

	// 1. 成功率效率
	successEfficiency := ser.routingMetrics.SuccessRate

	// 2. 延迟效率（与目标延迟比较）
	latencyEfficiency := 1.0
	if ser.routingMetrics.AvgRoutingLatency > 0 {
		targetLatency := ser.latencyThreshold
		actualLatency := ser.routingMetrics.AvgRoutingLatency
		if actualLatency <= targetLatency {
			latencyEfficiency = 1.0
		} else {
			latencyEfficiency = math.Max(0.1, float64(targetLatency)/float64(actualLatency))
		}
	}

	// 3. 成本效率（成本节约）
	costEfficiency := 1.0
	if ser.routingMetrics.CostSavings > 0 {
		costEfficiency = math.Min(1.0, ser.routingMetrics.CostSavings/ser.routingMetrics.TotalTradingCosts)
	}

	// 4. 故障转移效率
	failoverEfficiency := 1.0
	if ser.routingMetrics.FailoverCount > 0 {
		// 自动恢复率越高，效率越高
		failoverEfficiency = ser.routingMetrics.AutoRecoveryRate
	}

	// 综合优化效率
	efficiency := 0.4*successEfficiency + 0.3*latencyEfficiency + 0.2*costEfficiency + 0.1*failoverEfficiency

	return math.Max(0.0, math.Min(1.0, efficiency))
}

func (ser *SmartExchangeRouter) evaluateCondition(condition RoutingCondition, symbol, orderType string) bool {
	// 实现条件评估逻辑
	switch condition.Type {
	case "EXCHANGE_HEALTH":
		// 评估交易所健康状态
		if condition.Operator == "GREATER_THAN" {
			if threshold, ok := condition.Value.(float64); ok {
				ser.mu.RLock()
				defer ser.mu.RUnlock()

				for _, status := range ser.exchangeStatus {
					if status.IsOnline && status.HealthScore > threshold {
						return true
					}
				}
			}
		} else if condition.Operator == "LESS_THAN" {
			if threshold, ok := condition.Value.(float64); ok {
				ser.mu.RLock()
				defer ser.mu.RUnlock()

				for _, status := range ser.exchangeStatus {
					if status.IsOnline && status.HealthScore < threshold {
						return true
					}
				}
			}
		}

	case "LATENCY":
		// 评估延迟条件
		if threshold, ok := condition.Value.(time.Duration); ok {
			ser.mu.RLock()
			defer ser.mu.RUnlock()

			for _, status := range ser.exchangeStatus {
				if status.IsOnline {
					if condition.Operator == "GREATER_THAN" && status.Latency > threshold {
						return true
					} else if condition.Operator == "LESS_THAN" && status.Latency < threshold {
						return true
					}
				}
			}
		}

	case "SYMBOL":
		// 评估交易对条件
		if targetSymbol, ok := condition.Value.(string); ok {
			if condition.Operator == "EQUALS" {
				return symbol == targetSymbol
			} else if condition.Operator == "CONTAINS" {
				return len(symbol) > 0 && len(targetSymbol) > 0 &&
					(symbol == targetSymbol || targetSymbol == "*")
			}
		}

	case "ORDER_TYPE":
		// 评估订单类型条件
		if targetType, ok := condition.Value.(string); ok {
			if condition.Operator == "EQUALS" {
				return orderType == targetType
			}
		}

	case "TIME":
		// 评估时间条件（可以扩展为交易时间窗口等）
		now := time.Now()
		if condition.Operator == "BETWEEN" {
			// 可以实现时间范围检查
			return true // 简化实现
		}
		_ = now // 避免未使用变量警告
	}

	return false
}

func (ser *SmartExchangeRouter) isExchangeAvailable(exchange string, available []string) bool {
	for _, avail := range available {
		if avail == exchange {
			return true
		}
	}
	return false
}

func (ser *SmartExchangeRouter) removeExchange(exchanges []string, toRemove string) []string {
	result := make([]string, 0)
	for _, exchange := range exchanges {
		if exchange != toRemove {
			result = append(result, exchange)
		}
	}
	return result
}

func (ser *SmartExchangeRouter) generateDecisionID() string {
	return fmt.Sprintf("DEC_%d", time.Now().UnixNano())
}

func (ser *SmartExchangeRouter) generateFailoverID() string {
	return fmt.Sprintf("FO_%d", time.Now().UnixNano())
}

// selectExchange 负载均衡器选择交易所
func (lb *LoadBalancer) selectExchange(available []string) string {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	switch lb.algorithm {
	case "ROUND_ROBIN":
		return lb.selectRoundRobin(available)
	case "WEIGHTED":
		return lb.selectWeighted(available)
	case "LEAST_CONNECTIONS":
		return lb.selectLeastConnections(available)
	default:
		return lb.selectWeighted(available)
	}
}

func (lb *LoadBalancer) selectRoundRobin(available []string) string {
	if len(available) == 0 {
		return ""
	}

	// 找到上次选择的位置
	lastIndex := -1
	for i, exchange := range available {
		if exchange == lb.lastSelected {
			lastIndex = i
			break
		}
	}

	// 选择下一个
	nextIndex := (lastIndex + 1) % len(available)
	selected := available[nextIndex]
	lb.lastSelected = selected

	return selected
}

func (lb *LoadBalancer) selectWeighted(available []string) string {
	if len(available) == 0 {
		return ""
	}

	// 计算总权重
	totalWeight := 0.0
	for _, exchange := range available {
		weight := lb.weights[exchange]
		if weight <= 0 {
			weight = 1.0 // 默认权重
		}
		totalWeight += weight
	}

	// 随机选择
	random := rand.Float64() * totalWeight
	currentWeight := 0.0

	for _, exchange := range available {
		weight := lb.weights[exchange]
		if weight <= 0 {
			weight = 1.0
		}
		currentWeight += weight

		if random <= currentWeight {
			lb.lastSelected = exchange
			return exchange
		}
	}

	// 失败时返回第一个
	return available[0]
}

func (lb *LoadBalancer) selectLeastConnections(available []string) string {
	if len(available) == 0 {
		return ""
	}

	minConnections := math.MaxInt32
	selected := available[0]

	for _, exchange := range available {
		connections := lb.connections[exchange]
		if connections < minConnections {
			minConnections = connections
			selected = exchange
		}
	}

	lb.lastSelected = selected
	return selected
}

// GetStatus 获取路由器状态
func (ser *SmartExchangeRouter) GetStatus() map[string]interface{} {
	ser.mu.RLock()
	defer ser.mu.RUnlock()

	return map[string]interface{}{
		"running":               ser.isRunning,
		"enabled":               ser.enabled,
		"primary_exchange":      ser.primaryExchange,
		"backup_exchanges":      ser.backupExchanges,
		"auto_failover":         ser.autoFailover,
		"smart_routing":         ser.smartRouting,
		"load_balancing":        ser.loadBalancing,
		"exchange_count":        len(ser.exchangeStatus),
		"routing_rules_count":   len(ser.routingRules),
		"routing_history_size":  len(ser.routingHistory),
		"failover_threshold":    ser.failoverThreshold,
		"latency_threshold":     ser.latencyThreshold,
		"health_check_interval": ser.healthCheckInterval,
		"routing_metrics":       ser.routingMetrics,
		"exchange_status":       ser.exchangeStatus,
	}
}

// GetRoutingMetrics 获取路由指标
func (ser *SmartExchangeRouter) GetRoutingMetrics() *RoutingMetrics {
	ser.routingMetrics.mu.RLock()
	defer ser.routingMetrics.mu.RUnlock()

	// 创建一个新的 RoutingMetrics 实例，避免复制锁
	metrics := &RoutingMetrics{
		TotalRequests:          ser.routingMetrics.TotalRequests,
		SuccessfulRoutes:       ser.routingMetrics.SuccessfulRoutes,
		FailedRoutes:           ser.routingMetrics.FailedRoutes,
		SuccessRate:            ser.routingMetrics.SuccessRate,
		AvgRoutingLatency:      ser.routingMetrics.AvgRoutingLatency,
		AvgExecutionLatency:    ser.routingMetrics.AvgExecutionLatency,
		P95Latency:             ser.routingMetrics.P95Latency,
		P99Latency:             ser.routingMetrics.P99Latency,
		ExchangeDistribution:   make(map[string]int64),
		ExchangeSuccessRates:   make(map[string]float64),
		ExchangeLatencies:      make(map[string]time.Duration),
		FailoverCount:          ser.routingMetrics.FailoverCount,
		AvgFailoverTime:        ser.routingMetrics.AvgFailoverTime,
		AutoRecoveryRate:       ser.routingMetrics.AutoRecoveryRate,
		TotalTradingCosts:      ser.routingMetrics.TotalTradingCosts,
		AvgTradingCost:         ser.routingMetrics.AvgTradingCost,
		CostSavings:            ser.routingMetrics.CostSavings,
		OptimizationEfficiency: ser.routingMetrics.OptimizationEfficiency,
		RouteQuality:           ser.routingMetrics.RouteQuality,
		LastUpdated:            ser.routingMetrics.LastUpdated,
	}

	// 复制 map 数据
	for k, v := range ser.routingMetrics.ExchangeDistribution {
		metrics.ExchangeDistribution[k] = v
	}
	for k, v := range ser.routingMetrics.ExchangeSuccessRates {
		metrics.ExchangeSuccessRates[k] = v
	}
	for k, v := range ser.routingMetrics.ExchangeLatencies {
		metrics.ExchangeLatencies[k] = v
	}

	return metrics
}

// GetExchangeStatus 获取交易所状态
func (ser *SmartExchangeRouter) GetExchangeStatus(exchange string) (*ExchangeStatus, error) {
	ser.mu.RLock()
	defer ser.mu.RUnlock()

	if status, exists := ser.exchangeStatus[exchange]; exists {
		return status, nil
	}

	return nil, fmt.Errorf("exchange status not found: %s", exchange)
}

// GetHealthHistory 获取健康检查历史
func (ser *SmartExchangeRouter) GetHealthHistory(exchange string, limit int) ([]HealthCheck, error) {
	ser.healthMonitor.mu.RLock()
	defer ser.healthMonitor.mu.RUnlock()

	history, exists := ser.healthMonitor.healthHistory[exchange]
	if !exists {
		return nil, fmt.Errorf("health history not found for exchange: %s", exchange)
	}

	if limit <= 0 || limit > len(history) {
		limit = len(history)
	}

	// 返回最新的记录
	start := len(history) - limit
	return history[start:], nil
}

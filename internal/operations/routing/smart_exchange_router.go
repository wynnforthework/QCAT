package routing

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"qcat/internal/config"
)

// SmartExchangeRouter 智能交易所路由系统
type SmartExchangeRouter struct {
	config               *config.Config
	exchangeManager      *ExchangeManager
	healthMonitor        *HealthMonitor
	loadBalancer         *LoadBalancer
	failoverController   *FailoverController
	routingOptimizer     *RoutingOptimizer
	
	// 运行状态
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	isRunning  bool
	mu         sync.RWMutex
	
	// 路由配置
	primaryExchange      string
	backupExchanges      []string
	failoverThreshold    float64
	latencyThreshold     time.Duration
	healthCheckInterval  time.Duration
	
	// 路由状态
	exchangeStatus       map[string]*ExchangeStatus
	routingRules         []RoutingRule
	routingHistory       []RoutingDecision
	
	// 监控指标
	routingMetrics       *RoutingMetrics
	performanceHistory   []PerformanceSnapshot
	
	// 配置参数
	enabled              bool
	autoFailover         bool
	smartRouting         bool
	loadBalancing        bool
}

// ExchangeStatus 交易所状态
type ExchangeStatus struct {
	Exchange            string            `json:"exchange"`
	IsOnline            bool              `json:"is_online"`
	Latency             time.Duration     `json:"latency"`
	Availability        float64           `json:"availability"`
	ThroughputLimit     float64           `json:"throughput_limit"`
	CurrentLoad         float64           `json:"current_load"`
	ErrorRate           float64           `json:"error_rate"`
	
	// 连接状态
	ConnectionStatus    string            `json:"connection_status"`  // CONNECTED, DISCONNECTED, CONNECTING, ERROR
	LastPing            time.Time         `json:"last_ping"`
	PingSuccess         bool              `json:"ping_success"`
	ConsecutiveFailures int               `json:"consecutive_failures"`
	
	// 交易相关
	OrderBookDepth      float64           `json:"order_book_depth"`
	SpreadTightness     float64           `json:"spread_tightness"`
	TradingFees         map[string]float64 `json:"trading_fees"`
	SupportedPairs      []string          `json:"supported_pairs"`
	
	// 历史统计
	UptimePercentage    float64           `json:"uptime_percentage"`
	AvgLatency          time.Duration     `json:"avg_latency"`
	AvgErrorRate        float64           `json:"avg_error_rate"`
	
	// 限制和约束
	RateLimits          map[string]int    `json:"rate_limits"`
	MaintenanceWindows  []MaintenanceWindow `json:"maintenance_windows"`
	
	LastUpdated         time.Time         `json:"last_updated"`
	LastFailover        time.Time         `json:"last_failover"`
	HealthScore         float64           `json:"health_score"`
}

// MaintenanceWindow 维护窗口
type MaintenanceWindow struct {
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Type        string    `json:"type"`        // SCHEDULED, EMERGENCY
	Description string    `json:"description"`
	Recurring   bool      `json:"recurring"`
	Impact      string    `json:"impact"`      // HIGH, MEDIUM, LOW
}

// ExchangeManager 交易所管理器
type ExchangeManager struct {
	exchanges           map[string]*Exchange
	connectionPool      map[string]*ConnectionPool
	credentialManager   *CredentialManager
	
	mu                  sync.RWMutex
}

// Exchange 交易所配置
type Exchange struct {
	Name                string            `json:"name"`
	DisplayName         string            `json:"display_name"`
	Region              string            `json:"region"`
	Type                string            `json:"type"`            // SPOT, FUTURES, OPTIONS
	Priority            int               `json:"priority"`
	Capacity            float64           `json:"capacity"`
	
	// 连接配置
	RestAPI             APIConfig         `json:"rest_api"`
	WebSocketAPI        APIConfig         `json:"websocket_api"`
	FIXProtocol         *FIXConfig        `json:"fix_protocol"`
	
	// 交易配置
	MinOrderSize        map[string]float64 `json:"min_order_size"`
	MaxOrderSize        map[string]float64 `json:"max_order_size"`
	TickSizes           map[string]float64 `json:"tick_sizes"`
	TradingFees         FeeStructure      `json:"trading_fees"`
	
	// 功能支持
	SupportedOrderTypes []string          `json:"supported_order_types"`
	SupportedTimeframes []string          `json:"supported_timeframes"`
	MarginTrading       bool              `json:"margin_trading"`
	OptionsTrading      bool              `json:"options_trading"`
	
	// 状态
	IsEnabled           bool              `json:"is_enabled"`
	LastUpdated         time.Time         `json:"last_updated"`
}

// APIConfig API配置
type APIConfig struct {
	BaseURL             string            `json:"base_url"`
	Version             string            `json:"version"`
	Endpoints           map[string]string `json:"endpoints"`
	RateLimits          map[string]int    `json:"rate_limits"`
	Timeout             time.Duration     `json:"timeout"`
	RetryAttempts       int               `json:"retry_attempts"`
	Authentication      AuthConfig        `json:"authentication"`
}

// FIXConfig FIX协议配置
type FIXConfig struct {
	Host                string            `json:"host"`
	Port                int               `json:"port"`
	SenderCompID        string            `json:"sender_comp_id"`
	TargetCompID        string            `json:"target_comp_id"`
	HeartbeatInterval   time.Duration     `json:"heartbeat_interval"`
	LogonTimeout        time.Duration     `json:"logon_timeout"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	Type                string            `json:"type"`            // API_KEY, OAUTH, SIGNATURE
	APIKey              string            `json:"api_key"`
	SecretKey           string            `json:"secret_key"`
	Passphrase          string            `json:"passphrase"`
	SignatureMethod     string            `json:"signature_method"`
}

// FeeStructure 费用结构
type FeeStructure struct {
	MakerFee            float64           `json:"maker_fee"`
	TakerFee            float64           `json:"taker_fee"`
	WithdrawalFees      map[string]float64 `json:"withdrawal_fees"`
	VIPLevels           map[string]VIPFee `json:"vip_levels"`
}

// VIPFee VIP费率
type VIPFee struct {
	MakerFee            float64           `json:"maker_fee"`
	TakerFee            float64           `json:"taker_fee"`
	Requirements        map[string]float64 `json:"requirements"`
}

// ConnectionPool 连接池
type ConnectionPool struct {
	MaxConnections      int               `json:"max_connections"`
	ActiveConnections   int               `json:"active_connections"`
	IdleConnections     int               `json:"idle_connections"`
	ConnectionTimeout   time.Duration     `json:"connection_timeout"`
	IdleTimeout         time.Duration     `json:"idle_timeout"`
	MaxLifetime         time.Duration     `json:"max_lifetime"`
	
	mu                  sync.RWMutex
}

// CredentialManager 凭证管理器
type CredentialManager struct {
	credentials         map[string]*ExchangeCredential
	encryptionKey       []byte
	
	mu                  sync.RWMutex
}

// ExchangeCredential 交易所凭证
type ExchangeCredential struct {
	Exchange            string            `json:"exchange"`
	APIKey              string            `json:"api_key"`
	SecretKey           string            `json:"secret_key"`
	Passphrase          string            `json:"passphrase"`
	IsActive            bool              `json:"is_active"`
	ExpiresAt           time.Time         `json:"expires_at"`
	Permissions         []string          `json:"permissions"`
	CreatedAt           time.Time         `json:"created_at"`
	LastUsed            time.Time         `json:"last_used"`
}

// HealthMonitor 健康监控器
type HealthMonitor struct {
	checkInterval       time.Duration
	timeoutDuration     time.Duration
	healthThreshold     float64
	
	// 监控历史
	healthHistory       map[string][]HealthCheck
	lastChecks          map[string]HealthCheck
	
	mu                  sync.RWMutex
}

// HealthCheck 健康检查
type HealthCheck struct {
	Exchange            string            `json:"exchange"`
	Timestamp           time.Time         `json:"timestamp"`
	IsHealthy           bool              `json:"is_healthy"`
	Latency             time.Duration     `json:"latency"`
	ErrorMessage        string            `json:"error_message"`
	ResponseTime        time.Duration     `json:"response_time"`
	
	// 检查详情
	PingTest            TestResult        `json:"ping_test"`
	APITest             TestResult        `json:"api_test"`
	WebSocketTest       TestResult        `json:"websocket_test"`
	OrderBookTest       TestResult        `json:"order_book_test"`
	
	// 综合评分
	HealthScore         float64           `json:"health_score"`
	Components          map[string]float64 `json:"components"`
}

// TestResult 测试结果
type TestResult struct {
	Passed              bool              `json:"passed"`
	Duration            time.Duration     `json:"duration"`
	Error               string            `json:"error"`
	Details             map[string]interface{} `json:"details"`
}

// LoadBalancer 负载均衡器
type LoadBalancer struct {
	algorithm           string            // ROUND_ROBIN, WEIGHTED, LEAST_CONNECTIONS, HASH
	weights             map[string]float64
	connections         map[string]int
	lastSelected        string
	
	mu                  sync.RWMutex
}

// FailoverController 故障转移控制器
type FailoverController struct {
	failoverStrategy    string            // AUTO, MANUAL, HYBRID
	failoverThreshold   float64
	recoveryThreshold   float64
	maxFailovers        int
	failoverCooldown    time.Duration
	
	// 故障转移历史
	failoverHistory     []FailoverEvent
	lastFailover        time.Time
	failoverCount       int
	
	mu                  sync.RWMutex
}

// FailoverEvent 故障转移事件
type FailoverEvent struct {
	ID                  string            `json:"id"`
	Timestamp           time.Time         `json:"timestamp"`
	FromExchange        string            `json:"from_exchange"`
	ToExchange          string            `json:"to_exchange"`
	Trigger             string            `json:"trigger"`
	TriggerValue        float64           `json:"trigger_value"`
	Reason              string            `json:"reason"`
	Duration            time.Duration     `json:"duration"`
	Success             bool              `json:"success"`
	Impact              FailoverImpact    `json:"impact"`
	AutoRecovery        bool              `json:"auto_recovery"`
	RecoveryTime        time.Time         `json:"recovery_time"`
}

// FailoverImpact 故障转移影响
type FailoverImpact struct {
	AffectedOrders      int               `json:"affected_orders"`
	TradingInterruption time.Duration     `json:"trading_interruption"`
	LostOpportunities   float64           `json:"lost_opportunities"`
	AdditionalCosts     float64           `json:"additional_costs"`
	CustomerImpact      string            `json:"customer_impact"`
}

// RoutingOptimizer 路由优化器
type RoutingOptimizer struct {
	optimizationModel   string            // LATENCY, COST, LIQUIDITY, HYBRID
	reoptimizeInterval  time.Duration
	
	// 优化参数
	latencyWeight       float64
	costWeight          float64
	liquidityWeight     float64
	reliabilityWeight   float64
	
	// 优化历史
	optimizationHistory []OptimizationResult
	
	mu                  sync.RWMutex
}

// OptimizationResult 优化结果
type OptimizationResult struct {
	Timestamp           time.Time         `json:"timestamp"`
	OptimizationModel   string            `json:"optimization_model"`
	PreviousRouting     map[string]float64 `json:"previous_routing"`
	OptimalRouting      map[string]float64 `json:"optimal_routing"`
	ExpectedImprovement float64           `json:"expected_improvement"`
	ActualImprovement   float64           `json:"actual_improvement"`
	Metrics             OptimizationMetrics `json:"metrics"`
}

// OptimizationMetrics 优化指标
type OptimizationMetrics struct {
	AvgLatency          time.Duration     `json:"avg_latency"`
	TotalCost           float64           `json:"total_cost"`
	LiquidityScore      float64           `json:"liquidity_score"`
	ReliabilityScore    float64           `json:"reliability_score"`
	ThroughputScore     float64           `json:"throughput_score"`
}

// RoutingRule 路由规则
type RoutingRule struct {
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	Priority            int               `json:"priority"`
	Condition           RoutingCondition  `json:"condition"`
	Action              RoutingAction     `json:"action"`
	IsActive            bool              `json:"is_active"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
	HitCount            int64             `json:"hit_count"`
	SuccessCount        int64             `json:"success_count"`
}

// RoutingCondition 路由条件
type RoutingCondition struct {
	Type                string            `json:"type"`            // EXCHANGE_DOWN, HIGH_LATENCY, HIGH_COST, SYMBOL, TIME
	Operator            string            `json:"operator"`        // EQUALS, GREATER_THAN, LESS_THAN, CONTAINS
	Value               interface{}       `json:"value"`
	LogicalOperator     string            `json:"logical_operator"` // AND, OR, NOT
	SubConditions       []RoutingCondition `json:"sub_conditions"`
}

// RoutingAction 路由动作
type RoutingAction struct {
	Type                string            `json:"type"`            // ROUTE_TO, AVOID, LOAD_BALANCE, FAILOVER
	TargetExchange      string            `json:"target_exchange"`
	Parameters          map[string]interface{} `json:"parameters"`
	Fallback            *RoutingAction    `json:"fallback"`
}

// RoutingDecision 路由决策
type RoutingDecision struct {
	ID                  string            `json:"id"`
	Timestamp           time.Time         `json:"timestamp"`
	OrderID             string            `json:"order_id"`
	Symbol              string            `json:"symbol"`
	OrderType           string            `json:"order_type"`
	
	// 决策过程
	SelectedExchange    string            `json:"selected_exchange"`
	AlternativeExchanges []string         `json:"alternative_exchanges"`
	DecisionReason      string            `json:"decision_reason"`
	RuleMatches         []string          `json:"rule_matches"`
	
	// 决策指标
	LatencyScore        float64           `json:"latency_score"`
	CostScore           float64           `json:"cost_score"`
	LiquidityScore      float64           `json:"liquidity_score"`
	ReliabilityScore    float64           `json:"reliability_score"`
	OverallScore        float64           `json:"overall_score"`
	
	// 执行结果
	ExecutionTime       time.Duration     `json:"execution_time"`
	Success             bool              `json:"success"`
	ErrorMessage        string            `json:"error_message"`
	
	// 性能比较
	ExpectedLatency     time.Duration     `json:"expected_latency"`
	ActualLatency       time.Duration     `json:"actual_latency"`
	ExpectedCost        float64           `json:"expected_cost"`
	ActualCost          float64           `json:"actual_cost"`
}

// RoutingMetrics 路由指标
type RoutingMetrics struct {
	mu sync.RWMutex
	
	// 路由统计
	TotalRequests       int64             `json:"total_requests"`
	SuccessfulRoutes    int64             `json:"successful_routes"`
	FailedRoutes        int64             `json:"failed_routes"`
	SuccessRate         float64           `json:"success_rate"`
	
	// 性能指标
	AvgRoutingLatency   time.Duration     `json:"avg_routing_latency"`
	AvgExecutionLatency time.Duration     `json:"avg_execution_latency"`
	P95Latency          time.Duration     `json:"p95_latency"`
	P99Latency          time.Duration     `json:"p99_latency"`
	
	// 交易所分布
	ExchangeDistribution map[string]int64 `json:"exchange_distribution"`
	ExchangeSuccessRates map[string]float64 `json:"exchange_success_rates"`
	ExchangeLatencies   map[string]time.Duration `json:"exchange_latencies"`
	
	// 故障转移统计
	FailoverCount       int64             `json:"failover_count"`
	AvgFailoverTime     time.Duration     `json:"avg_failover_time"`
	AutoRecoveryRate    float64           `json:"auto_recovery_rate"`
	
	// 成本统计
	TotalTradingCosts   float64           `json:"total_trading_costs"`
	AvgTradingCost      float64           `json:"avg_trading_cost"`
	CostSavings         float64           `json:"cost_savings"`
	
	// 优化效果
	OptimizationEfficiency float64        `json:"optimization_efficiency"`
	RouteQuality        float64           `json:"route_quality"`
	
	LastUpdated         time.Time         `json:"last_updated"`
}

// PerformanceSnapshot 性能快照
type PerformanceSnapshot struct {
	Timestamp           time.Time         `json:"timestamp"`
	ExchangePerformance map[string]ExchangePerformance `json:"exchange_performance"`
	RoutingQuality      float64           `json:"routing_quality"`
	SystemLoad          float64           `json:"system_load"`
	FailoverEvents      int               `json:"failover_events"`
}

// ExchangePerformance 交易所性能
type ExchangePerformance struct {
	Latency             time.Duration     `json:"latency"`
	Availability        float64           `json:"availability"`
	ThroughputUtilization float64         `json:"throughput_utilization"`
	ErrorRate           float64           `json:"error_rate"`
	HealthScore         float64           `json:"health_score"`
	OrderSuccessRate    float64           `json:"order_success_rate"`
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
		routingMetrics:     &RoutingMetrics{
			ExchangeDistribution: make(map[string]int64),
			ExchangeSuccessRates: make(map[string]float64),
			ExchangeLatencies:    make(map[string]time.Duration),
		},
		performanceHistory: make([]PerformanceSnapshot, 0),
		primaryExchange:    "binance",
		backupExchanges:    []string{"okx", "bybit", "huobi"},
		failoverThreshold:  0.95,
		latencyThreshold:   100 * time.Millisecond,
		healthCheckInterval: 30 * time.Second,
		enabled:            true,
		autoFailover:       true,
		smartRouting:       true,
		loadBalancing:      true,
	}
	
	// 从配置文件读取参数
	if cfg != nil {
		// TODO: 从配置文件读取路由参数
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
				BaseURL: "https://api.binance.com",
				Version: "v3",
				Timeout: 5 * time.Second,
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
				BaseURL: "https://www.okx.com",
				Version: "v5",
				Timeout: 5 * time.Second,
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
				BaseURL: "https://api.bybit.com",
				Version: "v5",
				Timeout: 5 * time.Second,
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
		ID:                  ser.generateDecisionID(),
		Timestamp:           startTime,
		OrderID:             orderID,
		Symbol:              symbol,
		OrderType:           orderType,
		AlternativeExchanges: make([]string, 0),
		RuleMatches:         make([]string, 0),
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
	// TODO: 实现实际的ping测试
	return TestResult{
		Passed:   true,
		Duration: 50 * time.Millisecond,
		Details:  make(map[string]interface{}),
	}
}

func (ser *SmartExchangeRouter) performAPITest(exchange *Exchange) TestResult {
	// TODO: 实现实际的API测试
	return TestResult{
		Passed:   true,
		Duration: 200 * time.Millisecond,
		Details:  make(map[string]interface{}),
	}
}

func (ser *SmartExchangeRouter) performWebSocketTest(exchange *Exchange) TestResult {
	// TODO: 实现实际的WebSocket测试
	return TestResult{
		Passed:   true,
		Duration: 100 * time.Millisecond,
		Details:  make(map[string]interface{}),
	}
}

func (ser *SmartExchangeRouter) performOrderBookTest(exchange *Exchange) TestResult {
	// TODO: 实现实际的订单簿测试
	return TestResult{
		Passed:   true,
		Duration: 150 * time.Millisecond,
		Details:  make(map[string]interface{}),
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
	// TODO: 实现实际的订单路由执行
	log.Printf("Executing routing for order %s to %s", decision.OrderID, decision.SelectedExchange)
	
	// 模拟执行
	decision.ActualLatency = time.Duration(50+rand.Intn(100)) * time.Millisecond
	decision.ActualCost = 0.001 // 模拟交易成本
	
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
	// TODO: 实现可用性统计更新
}

func (ser *SmartExchangeRouter) getCurrentLoads() map[string]float64 {
	loads := make(map[string]float64)
	for exchange, status := range ser.exchangeStatus {
		loads[exchange] = status.CurrentLoad / status.ThroughputLimit
	}
	return loads
}

func (ser *SmartExchangeRouter) calculateIdealLoads() map[string]float64 {
	// TODO: 基于容量和权重计算理想负载分布
	return make(map[string]float64)
}

func (ser *SmartExchangeRouter) shouldRecover(exchange string, status *ExchangeStatus) bool {
	return status.HealthScore >= ser.failoverController.recoveryThreshold &&
		status.ConsecutiveFailures == 0
}

func (ser *SmartExchangeRouter) performRecovery(exchange string, status *ExchangeStatus) {
	// TODO: 实现恢复逻辑
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
	// TODO: 实现实际的故障转移逻辑
	log.Printf("Executing failover from %s to %s", from, to)
	return nil
}

func (ser *SmartExchangeRouter) getCurrentOptimizationMetrics() OptimizationMetrics {
	// TODO: 计算当前优化指标
	return OptimizationMetrics{}
}

func (ser *SmartExchangeRouter) calculateOptimalRouting(metrics OptimizationMetrics) map[string]float64 {
	// TODO: 计算最优路由分布
	return make(map[string]float64)
}

func (ser *SmartExchangeRouter) applyOptimizationResult(routing map[string]float64) {
	// TODO: 应用优化结果
}

func (ser *SmartExchangeRouter) calculateSystemLoad() float64 {
	// TODO: 计算系统负载
	return 0.5
}

func (ser *SmartExchangeRouter) calculateOrderSuccessRate(exchange string) float64 {
	// TODO: 计算订单成功率
	return 0.95
}

func (ser *SmartExchangeRouter) calculateRoutingQuality() float64 {
	// TODO: 计算路由质量
	return 0.85
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
	// TODO: 计算优化效率
	return 0.8
}

func (ser *SmartExchangeRouter) evaluateCondition(condition RoutingCondition, symbol, orderType string) bool {
	// TODO: 实现条件评估逻辑
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
	
	metrics := *ser.routingMetrics
	return &metrics
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

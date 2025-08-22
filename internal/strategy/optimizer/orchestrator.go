package optimizer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"qcat/internal/market"
)

// 新增：Config represents optimization configuration
type Config struct {
	StrategyID string                 `json:"strategy_id"`
	Method     string                 `json:"method"`
	Params     map[string]interface{} `json:"params"`
	Objective  string                 `json:"objective"`
	CreatedAt  time.Time              `json:"created_at"`
}

// OptimizerTask represents an optimization task
type OptimizerTask struct {
	ID         string
	StrategyID string
	Trigger    string
	Status     TaskStatus
	Params     map[string]interface{}
	BestParams map[string]float64
	Confidence float64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TaskStatus represents the status of an optimization task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

// Orchestrator manages optimization tasks
type Orchestrator struct {
	checker    *TriggerChecker
	optimizer  *WalkForwardOptimizer
	overfitDet *OverfitDetector
	tasks      map[string]*OptimizerTask
	db         *sql.DB // 新增：数据库连接用于获取真实数据
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(checker *TriggerChecker, optimizer *WalkForwardOptimizer, detector *OverfitDetector, db *sql.DB) *Orchestrator {
	return &Orchestrator{
		checker:    checker,
		optimizer:  optimizer,
		overfitDet: detector,
		tasks:      make(map[string]*OptimizerTask),
		db:         db,
	}
}

// 新增：StartOptimization starts a new optimization task
func (o *Orchestrator) StartOptimization(ctx context.Context, config *Config) (string, error) {
	// 创建优化任务
	task := &OptimizerTask{
		ID:         generateTaskID(),
		StrategyID: config.StrategyID,
		Trigger:    config.Method,
		Status:     TaskStatusPending,
		Params:     config.Params,
		CreatedAt:  config.CreatedAt,
		UpdatedAt:  config.CreatedAt,
	}

	// 存储任务
	o.tasks[task.ID] = task

	// 异步执行优化任务
	go func() {
		if err := o.RunTask(ctx, task.ID); err != nil {
			// 记录错误但不返回，因为这是异步执行
			fmt.Printf("Optimization task %s failed: %v\n", task.ID, err)
		}
	}()

	return task.ID, nil
}

// CreateTask creates a new optimization task
func (o *Orchestrator) CreateTask(ctx context.Context, strategyID string, trigger string) (*OptimizerTask, error) {
	task := &OptimizerTask{
		ID:         generateTaskID(),
		StrategyID: strategyID,
		Trigger:    trigger,
		Status:     TaskStatusPending,
		Params:     make(map[string]interface{}),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	o.tasks[task.ID] = task
	return task, nil
}

// RunTask executes an optimization task
func (o *Orchestrator) RunTask(ctx context.Context, taskID string) error {
	task, exists := o.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// 更新任务状态
	task.Status = TaskStatusRunning
	task.UpdatedAt = time.Now()

	// 执行WFO优化
	// 获取真实市场数据用于优化
	data, err := o.fetchRealMarketData(ctx, task.StrategyID)
	if err != nil {
		task.Status = TaskStatusFailed
		task.UpdatedAt = time.Now()
		return fmt.Errorf("failed to fetch market data: %w", err)
	}

	paramSpace := map[string][2]float64{
		"param1": {0.1, 0.5},
		"param2": {10, 50},
	}

	result, err := o.optimizer.Optimize(ctx, data, paramSpace)
	if err != nil {
		task.Status = TaskStatusFailed
		task.UpdatedAt = time.Now()
		return fmt.Errorf("optimization failed: %w", err)
	}

	// 过拟合检测
	overfitResult, err := o.overfitDet.CheckOverfitting(ctx, result.InSampleStats, result.OutSampleStats)
	if err != nil {
		task.Status = TaskStatusFailed
		task.UpdatedAt = time.Now()
		return fmt.Errorf("overfitting check failed: %w", err)
	}

	// 更新最佳参数
	task.BestParams = result.Parameters
	task.Confidence = calculateConfidence(overfitResult)
	task.Status = TaskStatusCompleted
	task.UpdatedAt = time.Now()

	return nil
}

// GetTask retrieves a task by ID
func (o *Orchestrator) GetTask(taskID string) (*OptimizerTask, error) {
	task, exists := o.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	return task, nil
}

// ListTasks lists all tasks
func (o *Orchestrator) ListTasks() []*OptimizerTask {
	tasks := make([]*OptimizerTask, 0, len(o.tasks))
	for _, task := range o.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// Helper functions

func generateTaskID() string {
	return fmt.Sprintf("opt_%d", time.Now().UnixNano())
}

func calculateConfidence(result *OverfitResult) float64 {
	// 基于多个指标计算置信度
	confidence := 1.0

	// Deflated Sharpe影响
	if result.DeflatedSharpe < 0.5 {
		confidence *= 0.8
	}

	// PBO得分影响
	if result.PBOScore > 0.2 {
		confidence *= 0.9
	}

	// 参数敏感度影响
	for _, sensitivity := range result.ParamSensitivity {
		if sensitivity > 0.3 {
			confidence *= 0.95
		}
	}

	return confidence
}

// fetchRealMarketData fetches real market data for optimization
func (o *Orchestrator) fetchRealMarketData(ctx context.Context, strategyID string) (*DataSet, error) {
	// 创建独立的上下文用于数据获取，避免被父上下文取消影响
	dataCtx, dataCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer dataCancel()

	// 首先获取策略配置以确定需要的交易对和时间范围
	strategyConfig, err := o.getStrategyConfig(dataCtx, strategyID)
	if err != nil {
		// 检查是否是上下文取消错误
		if err == context.Canceled {
			return nil, fmt.Errorf("strategy config query was canceled - this may indicate system shutdown")
		}
		if err == context.DeadlineExceeded {
			return nil, fmt.Errorf("strategy config query timed out - database may be slow or unavailable")
		}
		return nil, fmt.Errorf("failed to get strategy config: %w", err)
	}

	symbol := strategyConfig.Symbol
	if symbol == "" {
		symbol = "BTCUSDT" // 默认交易对
	}

	// 获取最近6个月的日线数据用于优化
	endTime := time.Now()
	startTime := endTime.AddDate(0, -6, 0) // 6个月前

	// 从数据库获取K线数据，使用独立的上下文
	klines, err := o.fetchKlineData(dataCtx, symbol, "1d", startTime, endTime)
	if err != nil {
		// 检查是否是上下文相关错误
		if err == context.Canceled {
			return nil, fmt.Errorf("kline data query was canceled - this may indicate system shutdown")
		}
		if err == context.DeadlineExceeded {
			return nil, fmt.Errorf("kline data query timed out - database may be slow or unavailable")
		}
		return nil, fmt.Errorf("failed to fetch kline data: %w", err)
	}

	if len(klines) < 30 {
		// 尝试获取更长时间范围的数据
		log.Printf("Warning: insufficient data for %s (%d klines), trying longer time range", symbol, len(klines))

		// 尝试获取1年的数据，使用独立的上下文
		startTime = endTime.AddDate(-1, 0, 0)
		klines, err = o.fetchKlineData(dataCtx, symbol, "1d", startTime, endTime)
		if err != nil {
			if err == context.Canceled {
				return nil, fmt.Errorf("extended kline data query was canceled")
			}
			if err == context.DeadlineExceeded {
				return nil, fmt.Errorf("extended kline data query timed out")
			}
			return nil, fmt.Errorf("failed to fetch extended kline data: %w", err)
		}

		if len(klines) < 10 {
			// 如果还是不够，尝试从外部API获取历史数据
			log.Printf("Warning: still insufficient data (%d klines), attempting to fetch from external API", len(klines))

			externalKlines, err := o.fetchFromExternalAPI(dataCtx, symbol, "1d", startTime, endTime)
			if err != nil {
				if err == context.Canceled {
					return nil, fmt.Errorf("external API data fetch was canceled")
				}
				if err == context.DeadlineExceeded {
					return nil, fmt.Errorf("external API data fetch timed out")
				}
				return nil, fmt.Errorf("failed to fetch data from external API: %w", err)
			}

			if len(externalKlines) < 10 {
				return nil, fmt.Errorf("insufficient historical data available: only %d klines found, need at least 10 for optimization", len(externalKlines))
			}

			klines = externalKlines
			log.Printf("Fetched %d klines from external API", len(klines))
		}

		log.Printf("Found %d klines with extended time range", len(klines))
	}

	// 转换为优化器需要的数据格式
	prices := make([]float64, len(klines))
	returns := make([]float64, len(klines)-1)
	volumes := make([]float64, len(klines))
	timestamps := make([]time.Time, len(klines))

	for i, kline := range klines {
		prices[i] = kline.Close
		volumes[i] = kline.Volume
		timestamps[i] = kline.OpenTime

		// 计算收益率
		if i > 0 {
			returns[i-1] = (kline.Close - klines[i-1].Close) / klines[i-1].Close
		}
	}

	// 获取交易数据用于更精确的分析，使用独立的上下文
	trades, err := o.fetchTradeData(dataCtx, symbol, startTime, endTime)
	if err != nil {
		// 交易数据不是必需的，记录警告但继续
		if err == context.Canceled {
			log.Printf("Warning: trade data fetch was canceled for %s", symbol)
		} else if err == context.DeadlineExceeded {
			log.Printf("Warning: trade data fetch timed out for %s", symbol)
		} else {
			log.Printf("Warning: failed to fetch trade data for %s: %v", symbol, err)
		}
	}

	// 创建数据集
	dataSet := &DataSet{
		Symbol:     symbol,
		Prices:     prices,
		Returns:    returns,
		Volumes:    volumes,
		Timestamps: timestamps,
		Trades:     trades,
		StartTime:  startTime,
		EndTime:    endTime,
	}

	return dataSet, nil
}

// getStrategyConfig retrieves strategy configuration from database
func (o *Orchestrator) getStrategyConfig(ctx context.Context, strategyID string) (*StrategyConfig, error) {
	// First get basic strategy info (strategies table doesn't have symbol column)
	query := `
		SELECT id, name, COALESCE(parameters, '{}') as config
		FROM strategies
		WHERE id = $1
	`

	var config StrategyConfig
	var configJSON []byte

	err := o.db.QueryRowContext(ctx, query, strategyID).Scan(
		&config.ID,
		&config.Name,
		&configJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("strategy not found: %s", strategyID)
		}
		if err == context.Canceled {
			return nil, fmt.Errorf("optimization canceled: %w", err)
		}
		if err == context.DeadlineExceeded {
			return nil, fmt.Errorf("optimization timeout: %w", err)
		}
		return nil, fmt.Errorf("failed to query strategy: %w", err)
	}

	// Get symbol from strategy_params table
	symbolQuery := `
		SELECT param_value
		FROM strategy_params
		WHERE strategy_id = $1 AND param_name = 'symbol'
		LIMIT 1
	`

	var symbol sql.NullString
	err = o.db.QueryRowContext(ctx, symbolQuery, strategyID).Scan(&symbol)
	if err != nil && err != sql.ErrNoRows {
		if err == context.Canceled {
			return nil, fmt.Errorf("optimization canceled while querying symbol: %w", err)
		}
		if err == context.DeadlineExceeded {
			return nil, fmt.Errorf("optimization timeout while querying symbol: %w", err)
		}
		return nil, fmt.Errorf("failed to query strategy symbol: %w", err)
	}

	if symbol.Valid {
		config.Symbol = symbol.String
	} else {
		config.Symbol = "BTCUSDT" // Default symbol if not found
	}

	// 解析配置JSON
	if len(configJSON) > 0 {
		var params map[string]interface{}
		if err := json.Unmarshal(configJSON, &params); err != nil {
			return nil, fmt.Errorf("failed to parse strategy config: %w", err)
		}
		config.Params = params
	} else {
		config.Params = make(map[string]interface{})
	}

	return &config, nil
}

// fetchKlineData fetches kline data from database
func (o *Orchestrator) fetchKlineData(ctx context.Context, symbol, interval string, startTime, endTime time.Time) ([]*market.Kline, error) {
	query := `
		SELECT symbol, interval, timestamp, open, high, low, close, volume, complete
		FROM market_data
		WHERE symbol = $1 AND interval = $2 AND timestamp BETWEEN $3 AND $4
		ORDER BY timestamp ASC
	`

	rows, err := o.db.QueryContext(ctx, query, symbol, interval, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query klines: %w", err)
	}
	defer rows.Close()

	var klines []*market.Kline
	for rows.Next() {
		var k market.Kline
		if err := rows.Scan(
			&k.Symbol,
			&k.Interval,
			&k.OpenTime,
			&k.Open,
			&k.High,
			&k.Low,
			&k.Close,
			&k.Volume,
			&k.Complete,
		); err != nil {
			return nil, fmt.Errorf("failed to scan kline: %w", err)
		}
		klines = append(klines, &k)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating klines: %w", err)
	}

	return klines, nil
}

// fetchTradeData fetches trade data from database
func (o *Orchestrator) fetchTradeData(ctx context.Context, symbol string, startTime, endTime time.Time) ([]*market.Trade, error) {
	query := `
		SELECT id, symbol, price, size, side, fee, fee_currency, created_at
		FROM trades
		WHERE symbol = $1 AND created_at BETWEEN $2 AND $3
		ORDER BY created_at ASC
		LIMIT 10000
	`

	rows, err := o.db.QueryContext(ctx, query, symbol, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query trades: %w", err)
	}
	defer rows.Close()

	var trades []*market.Trade
	for rows.Next() {
		var t market.Trade
		if err := rows.Scan(
			&t.ID,
			&t.Symbol,
			&t.Price,
			&t.Quantity,
			&t.Side,
			&t.Fee,
			&t.FeeCoin,
			&t.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("failed to scan trade: %w", err)
		}
		trades = append(trades, &t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating trades: %w", err)
	}

	return trades, nil
}

// fetchFromExternalAPI fetches market data from external API (e.g., Binance)
func (o *Orchestrator) fetchFromExternalAPI(ctx context.Context, symbol, interval string, startTime, endTime time.Time) ([]*market.Kline, error) {
	log.Printf("Fetching historical data from external API for %s from %v to %v", symbol, startTime, endTime)

	// 创建Binance API客户端
	client := &BinanceAPIClient{
		BaseURL: "https://api.binance.com",
		Timeout: 30 * time.Second,
	}

	// 调用Binance API获取K线数据
	klines, err := client.GetKlines(ctx, symbol, interval, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data from Binance API: %w", err)
	}

	// 将获取的数据保存到数据库以供将来使用
	if len(klines) > 0 {
		if err := o.saveKlinesToDatabase(ctx, klines); err != nil {
			log.Printf("Warning: failed to save fetched klines to database: %v", err)
		} else {
			log.Printf("Successfully saved %d klines to database", len(klines))
		}
	}

	return klines, nil
}

// saveKlinesToDatabase saves kline data to the database
func (o *Orchestrator) saveKlinesToDatabase(ctx context.Context, klines []*market.Kline) error {
	if len(klines) == 0 {
		return nil
	}

	// 批量插入K线数据
	query := `
		INSERT INTO klines (symbol, interval, open_time, close_time, open_price, high_price, low_price, close_price, volume, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (symbol, interval, open_time) DO UPDATE SET
			close_time = EXCLUDED.close_time,
			open_price = EXCLUDED.open_price,
			high_price = EXCLUDED.high_price,
			low_price = EXCLUDED.low_price,
			close_price = EXCLUDED.close_price,
			volume = EXCLUDED.volume,
			updated_at = NOW()
	`

	// 准备批量插入
	stmt, err := o.db.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	// 插入每个K线数据
	for _, kline := range klines {
		_, err := stmt.ExecContext(ctx,
			kline.Symbol,
			kline.Interval,
			kline.OpenTime,
			kline.CloseTime,
			kline.Open,
			kline.High,
			kline.Low,
			kline.Close,
			kline.Volume,
		)
		if err != nil {
			log.Printf("Warning: failed to insert kline for %s at %v: %v", kline.Symbol, kline.OpenTime, err)
		}
	}

	return nil
}

// BinanceAPIClient represents a client for Binance API
type BinanceAPIClient struct {
	BaseURL string
	Timeout time.Duration
	client  *http.Client
}

// BinanceKlineResponse represents Binance API kline response
type BinanceKlineResponse [][]interface{}

// GetKlines fetches kline data from Binance API
func (c *BinanceAPIClient) GetKlines(ctx context.Context, symbol, interval string, startTime, endTime time.Time) ([]*market.Kline, error) {
	if c.client == nil {
		c.client = &http.Client{Timeout: c.Timeout}
	}

	// 构建API URL
	baseURL := c.BaseURL + "/api/v3/klines"
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", interval)
	params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	params.Set("limit", "1000")

	fullURL := baseURL + "?" + params.Encode()

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var binanceData BinanceKlineResponse
	if err := json.NewDecoder(resp.Body).Decode(&binanceData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 转换为内部格式
	klines := make([]*market.Kline, 0, len(binanceData))
	for _, item := range binanceData {
		if len(item) < 11 {
			continue
		}

		kline := &market.Kline{
			Symbol:    symbol,
			Interval:  interval,
			OpenTime:  time.UnixMilli(int64(item[0].(float64))),
			CloseTime: time.UnixMilli(int64(item[6].(float64))),
		}

		// 解析价格数据
		if open, err := strconv.ParseFloat(item[1].(string), 64); err == nil {
			kline.Open = open
		}
		if high, err := strconv.ParseFloat(item[2].(string), 64); err == nil {
			kline.High = high
		}
		if low, err := strconv.ParseFloat(item[3].(string), 64); err == nil {
			kline.Low = low
		}
		if close, err := strconv.ParseFloat(item[4].(string), 64); err == nil {
			kline.Close = close
		}
		if volume, err := strconv.ParseFloat(item[5].(string), 64); err == nil {
			kline.Volume = volume
		}

		klines = append(klines, kline)
	}

	return klines, nil
}

// StrategyConfig represents strategy configuration
type StrategyConfig struct {
	ID     string                 `json:"id"`
	Name   string                 `json:"name"`
	Symbol string                 `json:"symbol"`
	Params map[string]interface{} `json:"params"`
}

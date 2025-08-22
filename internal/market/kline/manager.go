package kline

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/exchange/binance"
)

// Manager manages kline data for multiple symbols and intervals
type Manager struct {
	db                 *sql.DB
	binanceClient      *binance.Client
	klines             map[string]map[Interval]*Kline
	subscribers        map[string][]chan *Kline
	mu                 sync.RWMutex
	batchSize          int
	batchTimeout       time.Duration
	batchBuffer        []*Kline
	bufferMu           sync.Mutex
	autoBackfillConfig *AutoBackfillConfig
}

// NewManager creates a new kline manager
func NewManager(db *sql.DB) *Manager {
	m := &Manager{
		db:           db,
		klines:       make(map[string]map[Interval]*Kline),
		subscribers:  make(map[string][]chan *Kline),
		batchSize:    100,
		batchTimeout: 5 * time.Second,
		batchBuffer:  make([]*Kline, 0, 100),
	}

	// Start batch processor
	go m.processBatch()

	return m
}

// NewManagerWithBinance creates a new kline manager with Binance client
func NewManagerWithBinance(db *sql.DB, binanceClient *binance.Client) *Manager {
	m := &Manager{
		db:                 db,
		binanceClient:      binanceClient,
		klines:             make(map[string]map[Interval]*Kline),
		subscribers:        make(map[string][]chan *Kline),
		batchSize:          100,
		batchTimeout:       5 * time.Second,
		batchBuffer:        make([]*Kline, 0, 100),
		autoBackfillConfig: DefaultAutoBackfillConfig(),
	}

	// Start batch processor
	go m.processBatch()

	return m
}

// Subscribe subscribes to kline updates for a symbol and interval
func (m *Manager) Subscribe(symbol string, interval Interval) chan *Kline {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan *Kline, 100)
	key := fmt.Sprintf("%s-%s", symbol, interval)
	m.subscribers[key] = append(m.subscribers[key], ch)
	return ch
}

// Unsubscribe removes a subscription
func (m *Manager) Unsubscribe(symbol string, interval Interval, ch chan *Kline) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s-%s", symbol, interval)
	subs := m.subscribers[key]
	for i, sub := range subs {
		if sub == ch {
			m.subscribers[key] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

// UpdateTrade updates klines with a new trade
func (m *Manager) UpdateTrade(symbol string, price, volume float64, timestamp time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize symbol map if not exists
	if _, exists := m.klines[symbol]; !exists {
		m.klines[symbol] = make(map[Interval]*Kline)
	}

	// Update all intervals
	intervals := []Interval{
		Interval1m, Interval3m, Interval5m, Interval15m, Interval30m,
		Interval1h, Interval2h, Interval4h, Interval6h, Interval8h,
		Interval12h, Interval1d, Interval3d, Interval1w, Interval1M,
	}

	for _, interval := range intervals {
		kline := m.klines[symbol][interval]
		if kline == nil || timestamp.After(kline.CloseTime) {
			// Store completed kline
			if kline != nil && kline.Complete {
				if err := m.storeBatch(kline); err != nil {
					return fmt.Errorf("failed to store kline: %w", err)
				}
			}

			// Create new kline
			openTime := timestamp.Truncate(GetIntervalDuration(interval))
			kline = NewKline(symbol, interval, openTime)
			m.klines[symbol][interval] = kline
		}

		// Update kline
		kline.Update(price, volume, timestamp)

		// Notify subscribers
		key := fmt.Sprintf("%s-%s", symbol, interval)
		for _, ch := range m.subscribers[key] {
			select {
			case ch <- kline:
			default:
				// Channel is full, skip
			}
		}
	}

	return nil
}

// GetKline returns the current kline for a symbol and interval
func (m *Manager) GetKline(symbol string, interval Interval) *Kline {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if symbolKlines, exists := m.klines[symbol]; exists {
		return symbolKlines[interval]
	}
	return nil
}

// LoadHistoricalKlines loads historical klines from the database
func (m *Manager) LoadHistoricalKlines(ctx context.Context, symbol string, interval Interval, start, end time.Time) ([]*Kline, error) {
	query := `
		SELECT symbol, interval, timestamp, open, high, low, close, volume
		FROM market_data
		WHERE symbol = $1 AND interval = $2 AND timestamp BETWEEN $3 AND $4
		ORDER BY timestamp ASC
	`

	rows, err := m.db.QueryContext(ctx, query, symbol, interval, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query historical klines: %w", err)
	}
	defer rows.Close()

	var klines []*Kline
	for rows.Next() {
		var k Kline
		var timestamp time.Time
		if err := rows.Scan(
			&k.Symbol,
			&k.Interval,
			&timestamp,
			&k.Open,
			&k.High,
			&k.Low,
			&k.Close,
			&k.Volume,
		); err != nil {
			return nil, fmt.Errorf("failed to scan kline: %w", err)
		}

		k.OpenTime = timestamp
		k.CloseTime = getCloseTime(timestamp, k.Interval)
		k.Complete = true
		klines = append(klines, &k)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating klines: %w", err)
	}

	return klines, nil
}

// GetHistory returns historical klines for a symbol within a time range
// 自动检测数据缺失并回填
func (m *Manager) GetHistory(ctx context.Context, symbol string, start, end time.Time) ([]*Kline, error) {
	// Use 1-hour interval as default for historical data
	return m.GetHistoryWithBackfill(ctx, symbol, Interval1h, start, end)
}

// GetHistoryForInterval 获取指定间隔的历史数据，自动回填缺失数据
func (m *Manager) GetHistoryForInterval(ctx context.Context, symbol string, interval Interval, start, end time.Time) ([]*Kline, error) {
	return m.GetHistoryWithBackfill(ctx, symbol, interval, start, end)
}

// storeBatch adds a kline to the batch buffer
func (m *Manager) storeBatch(kline *Kline) error {
	m.bufferMu.Lock()
	m.batchBuffer = append(m.batchBuffer, kline)
	m.bufferMu.Unlock()

	return nil
}

// processBatch processes the batch buffer periodically
func (m *Manager) processBatch() {
	ticker := time.NewTicker(m.batchTimeout)
	defer ticker.Stop()

	for range ticker.C {
		m.bufferMu.Lock()
		if len(m.batchBuffer) == 0 {
			m.bufferMu.Unlock()
			continue
		}

		// Copy buffer and reset
		buffer := make([]*Kline, len(m.batchBuffer))
		copy(buffer, m.batchBuffer)
		m.batchBuffer = m.batchBuffer[:0]
		m.bufferMu.Unlock()

		// Store klines in database
		if err := m.storeKlines(buffer); err != nil {
			log.Printf("Error storing klines: %v", err)
		}
	}
}

// storeKlines stores multiple klines in the database
func (m *Manager) storeKlines(klines []*Kline) error {
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO market_data (
			symbol, interval, timestamp, open, high, low, close, volume, complete
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		) ON CONFLICT (symbol, timestamp, interval) DO UPDATE SET
			open = EXCLUDED.open,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			close = EXCLUDED.close,
			volume = EXCLUDED.volume,
			complete = EXCLUDED.complete
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, k := range klines {
		_, err := stmt.Exec(
			k.Symbol,
			k.Interval,
			k.OpenTime,
			k.Open,
			k.High,
			k.Low,
			k.Close,
			k.Volume,
			k.Complete,
		)
		if err != nil {
			return fmt.Errorf("failed to execute statement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// BackfillHistoricalData 回填历史数据
func (m *Manager) BackfillHistoricalData(ctx context.Context, symbol string, interval Interval, startTime, endTime time.Time) error {
	if m.binanceClient == nil {
		return fmt.Errorf("binance client not configured")
	}

	log.Printf("Starting historical data backfill for %s %s from %v to %v",
		symbol, interval, startTime.Format("2006-01-02"), endTime.Format("2006-01-02"))

	// 检查数据库中已有的数据
	existingData, err := m.LoadHistoricalKlines(ctx, symbol, interval, startTime, endTime)
	if err != nil {
		log.Printf("Warning: failed to check existing data: %v", err)
	}

	// 创建已有数据的时间戳映射
	existingTimes := make(map[int64]bool)
	for _, kline := range existingData {
		existingTimes[kline.OpenTime.Unix()] = true
	}

	// 分批获取数据（Binance API限制每次最多1000条）
	batchSize := 1000
	current := startTime
	totalFetched := 0
	totalSaved := 0

	for current.Before(endTime) {
		// 计算批次结束时间
		batchEnd := current.Add(time.Duration(batchSize) * GetIntervalDuration(interval))
		if batchEnd.After(endTime) {
			batchEnd = endTime
		}

		// 从Binance API获取数据
		klines, err := m.binanceClient.GetKlines(ctx, symbol, string(interval), current, batchEnd, batchSize)
		if err != nil {
			log.Printf("Failed to fetch klines for %s %s: %v", symbol, interval, err)
			// 继续处理下一批
			current = batchEnd
			continue
		}

		totalFetched += len(klines)

		// 转换并保存新数据
		var newKlines []*Kline
		for _, apiKline := range klines {
			// 检查是否已存在
			if existingTimes[apiKline.OpenTime.Unix()] {
				continue
			}

			// 转换为内部格式
			kline := &Kline{
				Symbol:    apiKline.Symbol,
				Interval:  interval,
				OpenTime:  apiKline.OpenTime,
				CloseTime: apiKline.CloseTime,
				Open:      apiKline.Open,
				High:      apiKline.High,
				Low:       apiKline.Low,
				Close:     apiKline.Close,
				Volume:    apiKline.Volume,
				Complete:  apiKline.Complete,
			}
			newKlines = append(newKlines, kline)
		}

		// 批量保存到数据库
		if len(newKlines) > 0 {
			if err := m.storeKlines(newKlines); err != nil {
				log.Printf("Failed to save klines batch: %v", err)
			} else {
				totalSaved += len(newKlines)
				log.Printf("Saved %d new klines for %s %s (batch: %v to %v)",
					len(newKlines), symbol, interval,
					current.Format("2006-01-02 15:04"), batchEnd.Format("2006-01-02 15:04"))
			}
		}

		// 移动到下一批
		current = batchEnd

		// 添加延迟以避免API限制
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("Historical data backfill completed for %s %s: fetched %d, saved %d new records",
		symbol, interval, totalFetched, totalSaved)

	return nil
}

// GetHistoryWithBackfill 获取历史数据，如果数据库中没有则自动从API回填
func (m *Manager) GetHistoryWithBackfill(ctx context.Context, symbol string, interval Interval, start, end time.Time) ([]*Kline, error) {
	config := m.GetAutoBackfillConfig()

	// 首先尝试从数据库获取
	klines, err := m.LoadHistoricalKlines(ctx, symbol, interval, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to load from database: %w", err)
	}

	// 如果自动回填被禁用，直接返回现有数据
	if !config.Enabled || m.binanceClient == nil {
		return klines, nil
	}

	// 检查数据完整性
	expectedCount := int(end.Sub(start) / GetIntervalDuration(interval))
	actualCount := len(klines)
	completeness := float64(actualCount) / float64(expectedCount) * 100

	// 检查是否需要回填
	needsBackfill := completeness < config.MinCompletenessPercent

	// 检查时间范围是否在允许的回填范围内
	maxBackfillTime := time.Now().AddDate(0, 0, -config.MaxBackfillDays)
	if start.Before(maxBackfillTime) {
		log.Printf("Requested start time %v is beyond max backfill range (%d days), skipping backfill",
			start.Format("2006-01-02"), config.MaxBackfillDays)
		needsBackfill = false
	}

	if needsBackfill {
		log.Printf("Data incomplete for %s %s (%.1f%% complete, %d/%d), starting auto-backfill",
			symbol, interval, completeness, actualCount, expectedCount)

		// 带重试的回填
		var backfillErr error
		for attempt := 1; attempt <= config.RetryAttempts; attempt++ {
			backfillErr = m.BackfillHistoricalData(ctx, symbol, interval, start, end)
			if backfillErr == nil {
				break
			}

			log.Printf("Backfill attempt %d/%d failed: %v", attempt, config.RetryAttempts, backfillErr)
			if attempt < config.RetryAttempts {
				time.Sleep(config.RetryDelay)
			}
		}

		if backfillErr == nil {
			// 重新加载数据
			klines, err = m.LoadHistoricalKlines(ctx, symbol, interval, start, end)
			if err != nil {
				return nil, fmt.Errorf("failed to reload after backfill: %w", err)
			}

			// 记录回填结果
			newCount := len(klines)
			newCompleteness := float64(newCount) / float64(expectedCount) * 100
			log.Printf("Auto-backfill completed for %s %s: %d -> %d records (%.1f%% -> %.1f%%)",
				symbol, interval, actualCount, newCount, completeness, newCompleteness)
		} else {
			log.Printf("Auto-backfill failed after %d attempts for %s %s: %v",
				config.RetryAttempts, symbol, interval, backfillErr)
		}
	}

	return klines, nil
}

// CheckDataIntegrity 检查数据完整性
func (m *Manager) CheckDataIntegrity(ctx context.Context, symbol string, interval Interval, start, end time.Time) (*DataIntegrityReport, error) {
	klines, err := m.LoadHistoricalKlines(ctx, symbol, interval, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to load klines: %w", err)
	}

	expectedCount := int(end.Sub(start) / GetIntervalDuration(interval))
	actualCount := len(klines)

	// 检查数据间隙
	var gaps []DataGap
	if len(klines) > 1 {
		intervalDuration := GetIntervalDuration(interval)
		for i := 1; i < len(klines); i++ {
			expectedTime := klines[i-1].OpenTime.Add(intervalDuration)
			if !klines[i].OpenTime.Equal(expectedTime) {
				gaps = append(gaps, DataGap{
					Start: expectedTime,
					End:   klines[i].OpenTime,
				})
			}
		}
	}

	completeness := float64(actualCount) / float64(expectedCount) * 100
	if expectedCount == 0 {
		completeness = 0
	}

	return &DataIntegrityReport{
		Symbol:        symbol,
		Interval:      interval,
		StartTime:     start,
		EndTime:       end,
		ExpectedCount: expectedCount,
		ActualCount:   actualCount,
		Completeness:  completeness,
		Gaps:          gaps,
		HasGaps:       len(gaps) > 0,
	}, nil
}

// DataIntegrityReport 数据完整性报告
type DataIntegrityReport struct {
	Symbol        string    `json:"symbol"`
	Interval      Interval  `json:"interval"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	ExpectedCount int       `json:"expected_count"`
	ActualCount   int       `json:"actual_count"`
	Completeness  float64   `json:"completeness"` // 百分比
	Gaps          []DataGap `json:"gaps"`
	HasGaps       bool      `json:"has_gaps"`
}

// DataGap 数据间隙
type DataGap struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// AutoBackfillConfig 自动回填配置
type AutoBackfillConfig struct {
	Enabled                bool          `json:"enabled"`                  // 是否启用自动回填
	MinCompletenessPercent float64       `json:"min_completeness_percent"` // 最小完整度百分比，低于此值触发回填
	MaxBackfillDays        int           `json:"max_backfill_days"`        // 最大回填天数
	RetryAttempts          int           `json:"retry_attempts"`           // 重试次数
	RetryDelay             time.Duration `json:"retry_delay"`              // 重试延迟
}

// DefaultAutoBackfillConfig 默认自动回填配置
func DefaultAutoBackfillConfig() *AutoBackfillConfig {
	return &AutoBackfillConfig{
		Enabled:                true,
		MinCompletenessPercent: 80.0, // 80%完整度
		MaxBackfillDays:        90,   // 最多回填90天
		RetryAttempts:          3,
		RetryDelay:             time.Second * 5,
	}
}

// SetAutoBackfillConfig 设置自动回填配置
func (m *Manager) SetAutoBackfillConfig(config *AutoBackfillConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.autoBackfillConfig = config
}

// GetAutoBackfillConfig 获取自动回填配置
func (m *Manager) GetAutoBackfillConfig() *AutoBackfillConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.autoBackfillConfig == nil {
		return DefaultAutoBackfillConfig()
	}
	return m.autoBackfillConfig
}

// EnsureDataAvailable 确保指定时间范围的数据可用，如果不可用则自动回填
// 这是一个通用的装饰器函数，可以在任何需要历史数据的地方调用
func (m *Manager) EnsureDataAvailable(ctx context.Context, symbol string, interval Interval, start, end time.Time) error {
	config := m.GetAutoBackfillConfig()

	if !config.Enabled || m.binanceClient == nil {
		return nil // 自动回填被禁用或没有API客户端
	}

	// 检查数据完整性
	report, err := m.CheckDataIntegrity(ctx, symbol, interval, start, end)
	if err != nil {
		return fmt.Errorf("failed to check data integrity: %w", err)
	}

	// 如果数据足够完整，不需要回填
	if report.Completeness >= config.MinCompletenessPercent {
		return nil
	}

	// 检查时间范围
	maxBackfillTime := time.Now().AddDate(0, 0, -config.MaxBackfillDays)
	if start.Before(maxBackfillTime) {
		return fmt.Errorf("requested start time %v is beyond max backfill range (%d days)",
			start.Format("2006-01-02"), config.MaxBackfillDays)
	}

	log.Printf("Ensuring data availability for %s %s (%.1f%% complete), starting backfill",
		symbol, interval, report.Completeness)

	// 执行回填
	return m.BackfillHistoricalData(ctx, symbol, interval, start, end)
}

// WithAutoBackfill 装饰器函数，为任何需要历史数据的操作提供自动回填
func (m *Manager) WithAutoBackfill(ctx context.Context, symbol string, interval Interval, start, end time.Time,
	operation func([]*Kline) error) error {

	// 确保数据可用
	if err := m.EnsureDataAvailable(ctx, symbol, interval, start, end); err != nil {
		log.Printf("Warning: failed to ensure data availability: %v", err)
	}

	// 获取数据
	klines, err := m.LoadHistoricalKlines(ctx, symbol, interval, start, end)
	if err != nil {
		return fmt.Errorf("failed to load historical data: %w", err)
	}

	// 执行操作
	return operation(klines)
}

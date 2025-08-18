package quality

import (
	"fmt"
	"math"
	"sync"
	"time"

	"qcat/internal/types"
)

// QualityMetrics represents data quality metrics
type QualityMetrics struct {
	Symbol           string        `json:"symbol"`
	DataType         string        `json:"data_type"`
	TotalMessages    int64         `json:"total_messages"`
	ValidMessages    int64         `json:"valid_messages"`
	InvalidMessages  int64         `json:"invalid_messages"`
	MissingMessages  int64         `json:"missing_messages"`
	DuplicateMessages int64        `json:"duplicate_messages"`
	LatencyP50       time.Duration `json:"latency_p50"`
	LatencyP95       time.Duration `json:"latency_p95"`
	LatencyP99       time.Duration `json:"latency_p99"`
	LastUpdate       time.Time     `json:"last_update"`
	QualityScore     float64       `json:"quality_score"`
}

// QualityIssue represents a data quality issue
type QualityIssue struct {
	ID          string                 `json:"id"`
	Symbol      string                 `json:"symbol"`
	DataType    string                 `json:"data_type"`
	IssueType   string                 `json:"issue_type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
	Resolved    bool                   `json:"resolved"`
}

// Monitor monitors data quality
type Monitor struct {
	metrics     map[string]*QualityMetrics
	issues      []QualityIssue
	latencies   map[string][]time.Duration
	lastSeen    map[string]time.Time
	duplicates  map[string]map[string]time.Time
	mu          sync.RWMutex
	
	// Configuration
	maxLatency      time.Duration
	maxGapDuration  time.Duration
	minQualityScore float64
	
	// Callbacks
	onIssue func(QualityIssue)
}

// NewMonitor creates a new quality monitor
func NewMonitor() *Monitor {
	return &Monitor{
		metrics:         make(map[string]*QualityMetrics),
		issues:          make([]QualityIssue, 0),
		latencies:       make(map[string][]time.Duration),
		lastSeen:        make(map[string]time.Time),
		duplicates:      make(map[string]map[string]time.Time),
		maxLatency:      5 * time.Second,
		maxGapDuration:  30 * time.Second,
		minQualityScore: 0.95,
		onIssue: func(issue QualityIssue) {
			// Default: do nothing
		},
	}
}

// SetIssueCallback sets the callback for quality issues
func (m *Monitor) SetIssueCallback(callback func(QualityIssue)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onIssue = callback
}

// CheckTrade validates and monitors trade data quality
func (m *Monitor) CheckTrade(trade *types.Trade) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:trade", trade.Symbol)
	
	// Initialize metrics if not exists
	if _, exists := m.metrics[key]; !exists {
		m.metrics[key] = &QualityMetrics{
			Symbol:   trade.Symbol,
			DataType: "trade",
		}
		m.latencies[key] = make([]time.Duration, 0, 1000)
		m.duplicates[key] = make(map[string]time.Time)
	}

	metrics := m.metrics[key]
	metrics.TotalMessages++
	metrics.LastUpdate = time.Now()

	// Check for basic validation
	if err := m.validateTrade(trade); err != nil {
		metrics.InvalidMessages++
		m.reportIssue(QualityIssue{
			ID:          fmt.Sprintf("%s-%d", key, time.Now().UnixNano()),
			Symbol:      trade.Symbol,
			DataType:    "trade",
			IssueType:   "validation_error",
			Severity:    "medium",
			Description: fmt.Sprintf("Trade validation failed: %v", err),
			Data:        map[string]interface{}{"trade": trade},
			Timestamp:   time.Now(),
		})
		return err
	}

	// Check for duplicates
	if lastSeen, exists := m.duplicates[key][trade.ID]; exists {
		if time.Since(lastSeen) < time.Minute {
			metrics.DuplicateMessages++
			m.reportIssue(QualityIssue{
				ID:          fmt.Sprintf("%s-dup-%d", key, time.Now().UnixNano()),
				Symbol:      trade.Symbol,
				DataType:    "trade",
				IssueType:   "duplicate",
				Severity:    "low",
				Description: fmt.Sprintf("Duplicate trade ID: %s", trade.ID),
				Data:        map[string]interface{}{"trade": trade},
				Timestamp:   time.Now(),
			})
		}
	}
	m.duplicates[key][trade.ID] = time.Now()

	// Check latency
	latency := time.Since(trade.Timestamp)
	m.latencies[key] = append(m.latencies[key], latency)
	if len(m.latencies[key]) > 1000 {
		m.latencies[key] = m.latencies[key][1:]
	}

	if latency > m.maxLatency {
		m.reportIssue(QualityIssue{
			ID:          fmt.Sprintf("%s-latency-%d", key, time.Now().UnixNano()),
			Symbol:      trade.Symbol,
			DataType:    "trade",
			IssueType:   "high_latency",
			Severity:    "medium",
			Description: fmt.Sprintf("High latency: %v", latency),
			Data:        map[string]interface{}{"latency": latency.String()},
			Timestamp:   time.Now(),
		})
	}

	// Check for gaps
	if lastSeen, exists := m.lastSeen[key]; exists {
		gap := time.Since(lastSeen)
		if gap > m.maxGapDuration {
			metrics.MissingMessages++
			m.reportIssue(QualityIssue{
				ID:          fmt.Sprintf("%s-gap-%d", key, time.Now().UnixNano()),
				Symbol:      trade.Symbol,
				DataType:    "trade",
				IssueType:   "data_gap",
				Severity:    "high",
				Description: fmt.Sprintf("Data gap detected: %v", gap),
				Data:        map[string]interface{}{"gap_duration": gap.String()},
				Timestamp:   time.Now(),
			})
		}
	}
	m.lastSeen[key] = time.Now()

	metrics.ValidMessages++
	m.updateQualityScore(metrics)

	return nil
}

// CheckKline validates and monitors kline data quality
func (m *Monitor) CheckKline(kline *types.Kline) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:kline:%s", kline.Symbol, kline.Interval)
	
	// Initialize metrics if not exists
	if _, exists := m.metrics[key]; !exists {
		m.metrics[key] = &QualityMetrics{
			Symbol:   kline.Symbol,
			DataType: fmt.Sprintf("kline:%s", kline.Interval),
		}
		m.latencies[key] = make([]time.Duration, 0, 1000)
	}

	metrics := m.metrics[key]
	metrics.TotalMessages++
	metrics.LastUpdate = time.Now()

	// Check for basic validation
	if err := m.validateKline(kline); err != nil {
		metrics.InvalidMessages++
		m.reportIssue(QualityIssue{
			ID:          fmt.Sprintf("%s-%d", key, time.Now().UnixNano()),
			Symbol:      kline.Symbol,
			DataType:    fmt.Sprintf("kline:%s", kline.Interval),
			IssueType:   "validation_error",
			Severity:    "medium",
			Description: fmt.Sprintf("Kline validation failed: %v", err),
			Data:        map[string]interface{}{"kline": kline},
			Timestamp:   time.Now(),
		})
		return err
	}

	// Check latency
	latency := time.Since(kline.CloseTime)
	if kline.Complete {
		m.latencies[key] = append(m.latencies[key], latency)
		if len(m.latencies[key]) > 1000 {
			m.latencies[key] = m.latencies[key][1:]
		}

		if latency > m.maxLatency {
			m.reportIssue(QualityIssue{
				ID:          fmt.Sprintf("%s-latency-%d", key, time.Now().UnixNano()),
				Symbol:      kline.Symbol,
				DataType:    fmt.Sprintf("kline:%s", kline.Interval),
				IssueType:   "high_latency",
				Severity:    "medium",
				Description: fmt.Sprintf("High latency: %v", latency),
				Data:        map[string]interface{}{"latency": latency.String()},
				Timestamp:   time.Now(),
			})
		}
	}

	metrics.ValidMessages++
	m.updateQualityScore(metrics)

	return nil
}

// CheckOrderBook validates and monitors order book data quality
func (m *Monitor) CheckOrderBook(orderBook *types.OrderBook) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:orderbook", orderBook.Symbol)
	
	// Initialize metrics if not exists
	if _, exists := m.metrics[key]; !exists {
		m.metrics[key] = &QualityMetrics{
			Symbol:   orderBook.Symbol,
			DataType: "orderbook",
		}
		m.latencies[key] = make([]time.Duration, 0, 1000)
	}

	metrics := m.metrics[key]
	metrics.TotalMessages++
	metrics.LastUpdate = time.Now()

	// Check for basic validation
	if err := m.validateOrderBook(orderBook); err != nil {
		metrics.InvalidMessages++
		m.reportIssue(QualityIssue{
			ID:          fmt.Sprintf("%s-%d", key, time.Now().UnixNano()),
			Symbol:      orderBook.Symbol,
			DataType:    "orderbook",
			IssueType:   "validation_error",
			Severity:    "medium",
			Description: fmt.Sprintf("OrderBook validation failed: %v", err),
			Data:        map[string]interface{}{"orderbook_size": len(orderBook.Bids) + len(orderBook.Asks)},
			Timestamp:   time.Now(),
		})
		return err
	}

	// Check latency
	latency := time.Since(orderBook.UpdatedAt)
	m.latencies[key] = append(m.latencies[key], latency)
	if len(m.latencies[key]) > 1000 {
		m.latencies[key] = m.latencies[key][1:]
	}

	if latency > m.maxLatency {
		m.reportIssue(QualityIssue{
			ID:          fmt.Sprintf("%s-latency-%d", key, time.Now().UnixNano()),
			Symbol:      orderBook.Symbol,
			DataType:    "orderbook",
			IssueType:   "high_latency",
			Severity:    "medium",
			Description: fmt.Sprintf("High latency: %v", latency),
			Data:        map[string]interface{}{"latency": latency.String()},
			Timestamp:   time.Now(),
		})
	}

	metrics.ValidMessages++
	m.updateQualityScore(metrics)

	return nil
}

// validateTrade validates trade data
func (m *Monitor) validateTrade(trade *types.Trade) error {
	if trade.Symbol == "" {
		return fmt.Errorf("missing symbol")
	}
	if trade.Price <= 0 {
		return fmt.Errorf("invalid price: %f", trade.Price)
	}
	if trade.Quantity <= 0 {
		return fmt.Errorf("invalid quantity: %f", trade.Quantity)
	}
	if trade.Side != "BUY" && trade.Side != "SELL" {
		return fmt.Errorf("invalid side: %s", trade.Side)
	}
	if trade.Timestamp.IsZero() {
		return fmt.Errorf("missing timestamp")
	}
	if time.Since(trade.Timestamp) > 24*time.Hour {
		return fmt.Errorf("timestamp too old: %v", trade.Timestamp)
	}
	return nil
}

// validateKline validates kline data
func (m *Monitor) validateKline(kline *types.Kline) error {
	if kline.Symbol == "" {
		return fmt.Errorf("missing symbol")
	}
	if kline.High < kline.Low {
		return fmt.Errorf("high price less than low price: %f < %f", kline.High, kline.Low)
	}
	if kline.Open <= 0 || kline.High <= 0 || kline.Low <= 0 || kline.Close <= 0 {
		return fmt.Errorf("invalid OHLC prices: O=%f H=%f L=%f C=%f", kline.Open, kline.High, kline.Low, kline.Close)
	}
	if kline.Volume < 0 {
		return fmt.Errorf("invalid volume: %f", kline.Volume)
	}
	if kline.OpenTime.IsZero() || kline.CloseTime.IsZero() {
		return fmt.Errorf("missing timestamps")
	}
	if kline.CloseTime.Before(kline.OpenTime) {
		return fmt.Errorf("close time before open time")
	}
	return nil
}

// validateOrderBook validates order book data
func (m *Monitor) validateOrderBook(orderBook *types.OrderBook) error {
	if orderBook.Symbol == "" {
		return fmt.Errorf("missing symbol")
	}
	if len(orderBook.Bids) == 0 && len(orderBook.Asks) == 0 {
		return fmt.Errorf("empty order book")
	}
	
	// Check bid prices are in descending order
	for i := 1; i < len(orderBook.Bids); i++ {
		if orderBook.Bids[i].Price > orderBook.Bids[i-1].Price {
			return fmt.Errorf("bids not in descending order")
		}
		if orderBook.Bids[i].Price <= 0 || orderBook.Bids[i].Quantity <= 0 {
			return fmt.Errorf("invalid bid level: price=%f quantity=%f", orderBook.Bids[i].Price, orderBook.Bids[i].Quantity)
		}
	}
	
	// Check ask prices are in ascending order
	for i := 1; i < len(orderBook.Asks); i++ {
		if orderBook.Asks[i].Price < orderBook.Asks[i-1].Price {
			return fmt.Errorf("asks not in ascending order")
		}
		if orderBook.Asks[i].Price <= 0 || orderBook.Asks[i].Quantity <= 0 {
			return fmt.Errorf("invalid ask level: price=%f quantity=%f", orderBook.Asks[i].Price, orderBook.Asks[i].Quantity)
		}
	}
	
	// Check spread is reasonable
	if len(orderBook.Bids) > 0 && len(orderBook.Asks) > 0 {
		bestBid := orderBook.Bids[0].Price
		bestAsk := orderBook.Asks[0].Price
		if bestAsk <= bestBid {
			return fmt.Errorf("invalid spread: bid=%f >= ask=%f", bestBid, bestAsk)
		}
		
		spread := (bestAsk - bestBid) / bestBid
		if spread > 0.1 { // 10% spread seems unreasonable
			return fmt.Errorf("spread too wide: %f%%", spread*100)
		}
	}
	
	return nil
}

// updateQualityScore calculates and updates quality score
func (m *Monitor) updateQualityScore(metrics *QualityMetrics) {
	if metrics.TotalMessages == 0 {
		metrics.QualityScore = 0
		return
	}

	// Calculate base quality score
	validRatio := float64(metrics.ValidMessages) / float64(metrics.TotalMessages)
	
	// Penalize for missing messages
	missingPenalty := float64(metrics.MissingMessages) / float64(metrics.TotalMessages)
	
	// Penalize for duplicates
	duplicatePenalty := float64(metrics.DuplicateMessages) / float64(metrics.TotalMessages)
	
	// Calculate latency score
	latencyScore := 1.0
	key := fmt.Sprintf("%s:%s", metrics.Symbol, metrics.DataType)
	if latencies, exists := m.latencies[key]; exists && len(latencies) > 0 {
		// Calculate P95 latency
		p95Index := int(float64(len(latencies)) * 0.95)
		if p95Index >= len(latencies) {
			p95Index = len(latencies) - 1
		}
		
		// Sort latencies for percentile calculation
		sortedLatencies := make([]time.Duration, len(latencies))
		copy(sortedLatencies, latencies)
		
		// Simple bubble sort for small arrays
		for i := 0; i < len(sortedLatencies); i++ {
			for j := i + 1; j < len(sortedLatencies); j++ {
				if sortedLatencies[i] > sortedLatencies[j] {
					sortedLatencies[i], sortedLatencies[j] = sortedLatencies[j], sortedLatencies[i]
				}
			}
		}
		
		p95Latency := sortedLatencies[p95Index]
		metrics.LatencyP95 = p95Latency
		
		if len(sortedLatencies) > 1 {
			p50Index := len(sortedLatencies) / 2
			p99Index := int(float64(len(sortedLatencies)) * 0.99)
			if p99Index >= len(sortedLatencies) {
				p99Index = len(sortedLatencies) - 1
			}
			
			metrics.LatencyP50 = sortedLatencies[p50Index]
			metrics.LatencyP99 = sortedLatencies[p99Index]
		}
		
		// Penalize high latency
		if p95Latency > m.maxLatency {
			latencyScore = math.Max(0, 1.0-float64(p95Latency-m.maxLatency)/float64(m.maxLatency))
		}
	}

	// Combine all factors
	metrics.QualityScore = validRatio * latencyScore * (1.0 - missingPenalty) * (1.0 - duplicatePenalty)
	
	// Ensure score is between 0 and 1
	if metrics.QualityScore < 0 {
		metrics.QualityScore = 0
	}
	if metrics.QualityScore > 1 {
		metrics.QualityScore = 1
	}

	// Report low quality score
	if metrics.QualityScore < m.minQualityScore {
		m.reportIssue(QualityIssue{
			ID:          fmt.Sprintf("%s-quality-%d", key, time.Now().UnixNano()),
			Symbol:      metrics.Symbol,
			DataType:    metrics.DataType,
			IssueType:   "low_quality_score",
			Severity:    "high",
			Description: fmt.Sprintf("Quality score below threshold: %.2f < %.2f", metrics.QualityScore, m.minQualityScore),
			Data: map[string]interface{}{
				"quality_score": metrics.QualityScore,
				"threshold":     m.minQualityScore,
				"metrics":       metrics,
			},
			Timestamp: time.Now(),
		})
	}
}

// reportIssue reports a quality issue
func (m *Monitor) reportIssue(issue QualityIssue) {
	m.issues = append(m.issues, issue)
	
	// Keep only recent issues (last 1000)
	if len(m.issues) > 1000 {
		m.issues = m.issues[len(m.issues)-1000:]
	}
	
	// Call callback
	go m.onIssue(issue)
}

// GetMetrics returns current quality metrics
func (m *Monitor) GetMetrics() map[string]*QualityMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[string]*QualityMetrics)
	for k, v := range m.metrics {
		// Create a copy
		metrics := *v
		result[k] = &metrics
	}
	
	return result
}

// GetIssues returns recent quality issues
func (m *Monitor) GetIssues(limit int) []QualityIssue {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if limit <= 0 || limit > len(m.issues) {
		limit = len(m.issues)
	}
	
	// Return the most recent issues
	start := len(m.issues) - limit
	if start < 0 {
		start = 0
	}
	
	result := make([]QualityIssue, limit)
	copy(result, m.issues[start:])
	
	return result
}

// GetQualityScore returns overall quality score
func (m *Monitor) GetQualityScore() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if len(m.metrics) == 0 {
		return 0
	}
	
	totalScore := 0.0
	count := 0
	
	for _, metrics := range m.metrics {
		totalScore += metrics.QualityScore
		count++
	}
	
	return totalScore / float64(count)
}
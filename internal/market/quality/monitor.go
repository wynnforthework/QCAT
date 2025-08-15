package quality

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"
)

// Monitor manages data quality monitoring and alerting
type Monitor struct {
	db          *sql.DB
	metrics     map[string]map[DataType]*Metric
	thresholds  map[DataType]Threshold
	subscribers []chan Alert
	mu          sync.RWMutex
}

// NewMonitor creates a new data quality monitor
func NewMonitor(db *sql.DB) *Monitor {
	m := &Monitor{
		db:          db,
		metrics:     make(map[string]map[DataType]*Metric),
		thresholds:  make(map[DataType]Threshold),
		subscribers: make([]chan Alert, 0),
	}

	// Set default thresholds
	m.setDefaultThresholds()

	// Start monitoring routine
	go m.monitor()

	return m
}

// Subscribe subscribes to data quality alerts
func (m *Monitor) Subscribe() chan Alert {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan Alert, 100)
	m.subscribers = append(m.subscribers, ch)
	return ch
}

// Unsubscribe removes a subscription
func (m *Monitor) Unsubscribe(ch chan Alert) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, sub := range m.subscribers {
		if sub == ch {
			m.subscribers = append(m.subscribers[:i], m.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}

// UpdateMetric updates data quality metrics
func (m *Monitor) UpdateMetric(symbol string, dataType DataType, update func(*Metric)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.metrics[symbol]; !exists {
		m.metrics[symbol] = make(map[DataType]*Metric)
	}

	metric, exists := m.metrics[symbol][dataType]
	if !exists {
		metric = &Metric{
			Symbol:     symbol,
			DataType:   dataType,
			LastUpdate: time.Now(),
		}
		m.metrics[symbol][dataType] = metric
	}

	update(metric)
	m.checkThresholds(symbol, dataType, metric)
}

// GetMetric returns data quality metrics for a symbol and data type
func (m *Monitor) GetMetric(symbol string, dataType DataType) *Metric {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if metrics, exists := m.metrics[symbol]; exists {
		return metrics[dataType]
	}
	return nil
}

// SetThreshold sets alert thresholds for a data type
func (m *Monitor) SetThreshold(dataType DataType, threshold Threshold) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.thresholds[dataType] = threshold
}

// setDefaultThresholds sets default alert thresholds
func (m *Monitor) setDefaultThresholds() {
	defaults := map[DataType]Threshold{
		DataTypeTicker: {
			MaxUpdateInterval: 5 * time.Second,
			MaxMissingData:    10,
			MaxErrors:         5,
			MaxLatency:        1.0,  // seconds
			MaxStaleness:      30.0, // seconds
			MinCompleteness:   0.99,
			MinAccuracy:       0.99,
		},
		DataTypeOrderBook: {
			MaxUpdateInterval: 1 * time.Second,
			MaxMissingData:    5,
			MaxErrors:         3,
			MaxLatency:        0.5,
			MaxStaleness:      10.0,
			MinCompleteness:   0.995,
			MinAccuracy:       0.995,
		},
		DataTypeTrade: {
			MaxUpdateInterval: 1 * time.Second,
			MaxMissingData:    5,
			MaxErrors:         3,
			MaxLatency:        0.5,
			MaxStaleness:      10.0,
			MinCompleteness:   0.99,
			MinAccuracy:       0.99,
		},
		DataTypeKline: {
			MaxUpdateInterval: 1 * time.Minute,
			MaxMissingData:    5,
			MaxErrors:         3,
			MaxLatency:        1.0,
			MaxStaleness:      60.0,
			MinCompleteness:   0.99,
			MinAccuracy:       0.99,
		},
		DataTypeFunding: {
			MaxUpdateInterval: 1 * time.Hour,
			MaxMissingData:    3,
			MaxErrors:         2,
			MaxLatency:        5.0,
			MaxStaleness:      3600.0,
			MinCompleteness:   0.99,
			MinAccuracy:       0.99,
		},
		DataTypeOI: {
			MaxUpdateInterval: 1 * time.Minute,
			MaxMissingData:    5,
			MaxErrors:         3,
			MaxLatency:        1.0,
			MaxStaleness:      60.0,
			MinCompleteness:   0.99,
			MinAccuracy:       0.99,
		},
		DataTypeIndex: {
			MaxUpdateInterval: 1 * time.Second,
			MaxMissingData:    5,
			MaxErrors:         3,
			MaxLatency:        0.5,
			MaxStaleness:      10.0,
			MinCompleteness:   0.995,
			MinAccuracy:       0.995,
		},
	}

	for dataType, threshold := range defaults {
		m.thresholds[dataType] = threshold
	}
}

// checkThresholds checks if metrics exceed thresholds and generates alerts
func (m *Monitor) checkThresholds(symbol string, dataType DataType, metric *Metric) {
	threshold, exists := m.thresholds[dataType]
	if !exists {
		return
	}

	now := time.Now()

	// Check update interval
	if metric.LastUpdate.Add(threshold.MaxUpdateInterval).Before(now) {
		m.alert(Alert{
			Symbol:      symbol,
			DataType:    dataType,
			Level:       "warning",
			Message:     "Data update interval exceeded threshold",
			Timestamp:   now,
			MetricValue: now.Sub(metric.LastUpdate).Seconds(),
			Threshold:   threshold.MaxUpdateInterval.Seconds(),
		})
	}

	// Check missing data count
	if metric.MissingDataCount > threshold.MaxMissingData {
		m.alert(Alert{
			Symbol:      symbol,
			DataType:    dataType,
			Level:       "warning",
			Message:     "Missing data count exceeded threshold",
			Timestamp:   now,
			MetricValue: float64(metric.MissingDataCount),
			Threshold:   float64(threshold.MaxMissingData),
		})
	}

	// Check error count
	if metric.ErrorCount > threshold.MaxErrors {
		m.alert(Alert{
			Symbol:      symbol,
			DataType:    dataType,
			Level:       "error",
			Message:     "Error count exceeded threshold",
			Timestamp:   now,
			MetricValue: float64(metric.ErrorCount),
			Threshold:   float64(threshold.MaxErrors),
		})
	}

	// Check latency
	if metric.Latency > threshold.MaxLatency {
		m.alert(Alert{
			Symbol:      symbol,
			DataType:    dataType,
			Level:       "warning",
			Message:     "Latency exceeded threshold",
			Timestamp:   now,
			MetricValue: metric.Latency,
			Threshold:   threshold.MaxLatency,
		})
	}

	// Check staleness
	if metric.Staleness > threshold.MaxStaleness {
		m.alert(Alert{
			Symbol:      symbol,
			DataType:    dataType,
			Level:       "warning",
			Message:     "Data staleness exceeded threshold",
			Timestamp:   now,
			MetricValue: metric.Staleness,
			Threshold:   threshold.MaxStaleness,
		})
	}

	// Check completeness
	if metric.Completeness < threshold.MinCompleteness {
		m.alert(Alert{
			Symbol:      symbol,
			DataType:    dataType,
			Level:       "warning",
			Message:     "Data completeness below threshold",
			Timestamp:   now,
			MetricValue: metric.Completeness,
			Threshold:   threshold.MinCompleteness,
		})
	}

	// Check accuracy
	if metric.Accuracy < threshold.MinAccuracy {
		m.alert(Alert{
			Symbol:      symbol,
			DataType:    dataType,
			Level:       "warning",
			Message:     "Data accuracy below threshold",
			Timestamp:   now,
			MetricValue: metric.Accuracy,
			Threshold:   threshold.MinAccuracy,
		})
	}
}

// alert sends an alert to all subscribers and stores it in the database
func (m *Monitor) alert(alert Alert) {
	// Store alert in database
	if err := m.storeAlert(alert); err != nil {
		log.Printf("Error storing alert: %v", err)
	}

	// Notify subscribers
	m.mu.RLock()
	for _, ch := range m.subscribers {
		select {
		case ch <- alert:
		default:
			// Channel is full, skip
		}
	}
	m.mu.RUnlock()
}

// storeAlert stores an alert in the database
func (m *Monitor) storeAlert(alert Alert) error {
	query := `
		INSERT INTO data_quality_alerts (
			symbol, data_type, level, message, timestamp,
			metric_value, threshold
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := m.db.Exec(query,
		alert.Symbol,
		alert.DataType,
		alert.Level,
		alert.Message,
		alert.Timestamp,
		alert.MetricValue,
		alert.Threshold,
	)
	if err != nil {
		return fmt.Errorf("failed to store alert: %w", err)
	}

	return nil
}

// monitor periodically checks data quality metrics
func (m *Monitor) monitor() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.RLock()
		for symbol, metrics := range m.metrics {
			for dataType, metric := range metrics {
				m.checkThresholds(symbol, dataType, metric)
			}
		}
		m.mu.RUnlock()
	}
}

// GetAlertHistory returns historical alerts
func (m *Monitor) GetAlertHistory(ctx context.Context, symbol string, dataType DataType, start, end time.Time) ([]Alert, error) {
	query := `
		SELECT symbol, data_type, level, message, timestamp,
			   metric_value, threshold
		FROM data_quality_alerts
		WHERE symbol = $1 AND data_type = $2
		  AND timestamp BETWEEN $3 AND $4
		ORDER BY timestamp DESC
	`

	rows, err := m.db.QueryContext(ctx, query, symbol, dataType, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query alert history: %w", err)
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var alert Alert
		if err := rows.Scan(
			&alert.Symbol,
			&alert.DataType,
			&alert.Level,
			&alert.Message,
			&alert.Timestamp,
			&alert.MetricValue,
			&alert.Threshold,
		); err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}
		alerts = append(alerts, alert)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating alerts: %w", err)
	}

	return alerts, nil
}

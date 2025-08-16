package report

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	"sync"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/strategy/state"
)

// Reporter manages automated reporting
type Reporter struct {
	db           *sql.DB
	stateManager *state.Manager
	exchange     exchange.Exchange
	reports      map[string]*Report
	subscribers  map[string][]ReportCallback
	mu           sync.RWMutex
}

// Report represents a performance report
type Report struct {
	ID          string                 `json:"id"`
	Strategy    string                 `json:"strategy"`
	Symbol      string                 `json:"symbol"`
	Type        ReportType             `json:"type"`
	Status      ReportStatus           `json:"status"`
	Schedule    Schedule               `json:"schedule"`
	Content     string                 `json:"content"`
	Format      ReportFormat           `json:"format"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	GeneratedAt time.Time              `json:"generated_at"`
}

// ReportType represents the type of report
type ReportType string

const (
	ReportTypeDaily   ReportType = "daily"
	ReportTypeWeekly  ReportType = "weekly"
	ReportTypeMonthly ReportType = "monthly"
	ReportTypeCustom  ReportType = "custom"
)

// ReportStatus represents the status of a report
type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusRunning   ReportStatus = "running"
	ReportStatusCompleted ReportStatus = "completed"
	ReportStatusFailed    ReportStatus = "failed"
)

// ReportFormat represents the format of a report
type ReportFormat string

const (
	ReportFormatHTML ReportFormat = "html"
	ReportFormatJSON ReportFormat = "json"
	ReportFormatCSV  ReportFormat = "csv"
)

// Schedule represents a report schedule
type Schedule struct {
	Type      string    `json:"type"`              // "daily", "weekly", "monthly", "custom"
	Time      string    `json:"time"`              // "HH:MM"
	Weekday   int       `json:"weekday,omitempty"` // 0-6 (Sunday-Saturday)
	Day       int       `json:"day,omitempty"`     // 1-31
	StartDate time.Time `json:"start_date,omitempty"`
	EndDate   time.Time `json:"end_date,omitempty"`
}

// ReportCallback represents a report callback function
type ReportCallback func(*Report)

// NewReporter creates a new reporter
func NewReporter(db *sql.DB, stateManager *state.Manager, exchange exchange.Exchange) *Reporter {
	return &Reporter{
		db:           db,
		stateManager: stateManager,
		exchange:     exchange,
		reports:      make(map[string]*Report),
		subscribers:  make(map[string][]ReportCallback),
	}
}

// Start starts the reporter
func (r *Reporter) Start(ctx context.Context) error {
	// Load existing reports
	if err := r.loadReports(ctx); err != nil {
		return fmt.Errorf("failed to load reports: %w", err)
	}

	// Start report generation
	go r.generateReports(ctx)

	return nil
}

// CreateReport creates a new report
func (r *Reporter) CreateReport(ctx context.Context, strategy, symbol string, reportType ReportType, schedule Schedule, format ReportFormat) (*Report, error) {
	report := &Report{
		ID:        fmt.Sprintf("%s-%s-%s-%d", strategy, symbol, reportType, time.Now().UnixNano()),
		Strategy:  strategy,
		Symbol:    symbol,
		Type:      reportType,
		Status:    ReportStatusPending,
		Schedule:  schedule,
		Format:    format,
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store report in database
	if err := r.saveReport(ctx, report); err != nil {
		return nil, fmt.Errorf("failed to save report: %w", err)
	}

	// Store report in memory
	r.mu.Lock()
	r.reports[report.ID] = report
	r.mu.Unlock()

	return report, nil
}

// GetReport returns a report by ID
func (r *Reporter) GetReport(ctx context.Context, id string) (*Report, error) {
	r.mu.RLock()
	report, exists := r.reports[id]
	r.mu.RUnlock()

	if exists {
		return report, nil
	}

	// Load report from database
	report, err := r.loadReport(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load report: %w", err)
	}

	// Store report in memory
	r.mu.Lock()
	r.reports[report.ID] = report
	r.mu.Unlock()

	return report, nil
}

// ListReports returns all reports
func (r *Reporter) ListReports(ctx context.Context) ([]*Report, error) {
	// Load reports from database
	query := `
		SELECT id, strategy, symbol, type, status, schedule, content, format,
			metadata, created_at, updated_at, generated_at
		FROM reports
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query reports: %w", err)
	}
	defer rows.Close()

	var reports []*Report
	for rows.Next() {
		var report Report
		var sched, meta []byte
		var generatedAt sql.NullTime

		if err := rows.Scan(
			&report.ID,
			&report.Strategy,
			&report.Symbol,
			&report.Type,
			&report.Status,
			&sched,
			&report.Content,
			&report.Format,
			&meta,
			&report.CreatedAt,
			&report.UpdatedAt,
			&generatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan report: %w", err)
		}

		if err := json.Unmarshal(sched, &report.Schedule); err != nil {
			return nil, fmt.Errorf("failed to unmarshal schedule: %w", err)
		}

		if err := json.Unmarshal(meta, &report.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		if generatedAt.Valid {
			report.GeneratedAt = generatedAt.Time
		}

		reports = append(reports, &report)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reports: %w", err)
	}

	return reports, nil
}

// Subscribe subscribes to report updates
func (r *Reporter) Subscribe(reportID string, callback ReportCallback) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.subscribers[reportID] = append(r.subscribers[reportID], callback)
}

// Unsubscribe removes a report subscription
func (r *Reporter) Unsubscribe(reportID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.subscribers, reportID)
}

// saveReport saves a report to the database
func (r *Reporter) saveReport(ctx context.Context, report *Report) error {
	sched, err := json.Marshal(report.Schedule)
	if err != nil {
		return fmt.Errorf("failed to marshal schedule: %w", err)
	}

	meta, err := json.Marshal(report.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO reports (
			id, strategy, symbol, type, status, schedule, content, format,
			metadata, created_at, updated_at, generated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		) ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			content = EXCLUDED.content,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at,
			generated_at = EXCLUDED.generated_at
	`

	_, err = r.db.ExecContext(ctx, query,
		report.ID,
		report.Strategy,
		report.Symbol,
		report.Type,
		report.Status,
		sched,
		report.Content,
		report.Format,
		meta,
		report.CreatedAt,
		report.UpdatedAt,
		sql.NullTime{Time: report.GeneratedAt, Valid: !report.GeneratedAt.IsZero()},
	)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}

// loadReport loads a report from the database
func (r *Reporter) loadReport(ctx context.Context, id string) (*Report, error) {
	query := `
		SELECT id, strategy, symbol, type, status, schedule, content, format,
			metadata, created_at, updated_at, generated_at
		FROM reports
		WHERE id = $1
	`

	var report Report
	var sched, meta []byte
	var generatedAt sql.NullTime

	if err := r.db.QueryRowContext(ctx, query, id).Scan(
		&report.ID,
		&report.Strategy,
		&report.Symbol,
		&report.Type,
		&report.Status,
		&sched,
		&report.Content,
		&report.Format,
		&meta,
		&report.CreatedAt,
		&report.UpdatedAt,
		&generatedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to scan report: %w", err)
	}

	if err := json.Unmarshal(sched, &report.Schedule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schedule: %w", err)
	}

	if err := json.Unmarshal(meta, &report.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	if generatedAt.Valid {
		report.GeneratedAt = generatedAt.Time
	}

	return &report, nil
}

// loadReports loads reports from the database
func (r *Reporter) loadReports(ctx context.Context) error {
	reports, err := r.ListReports(ctx)
	if err != nil {
		return err
	}

	r.mu.Lock()
	for _, report := range reports {
		r.reports[report.ID] = report
	}
	r.mu.Unlock()

	return nil
}

// generateReports periodically generates reports
func (r *Reporter) generateReports(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.checkReports(ctx)
		}
	}
}

// checkReports checks if any reports need to be generated
func (r *Reporter) checkReports(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for _, report := range r.reports {
		if report.Status == ReportStatusRunning {
			continue
		}

		if r.shouldGenerateReport(report, now) {
			go r.generateReport(ctx, report)
		}
	}
}

// shouldGenerateReport checks if a report should be generated
func (r *Reporter) shouldGenerateReport(report *Report, now time.Time) bool {
	// Check if report is within schedule
	if !report.Schedule.StartDate.IsZero() && now.Before(report.Schedule.StartDate) {
		return false
	}
	if !report.Schedule.EndDate.IsZero() && now.After(report.Schedule.EndDate) {
		return false
	}

	// Parse schedule time
	schedTime, err := time.Parse("15:04", report.Schedule.Time)
	if err != nil {
		log.Printf("Invalid schedule time for report %s: %v", report.ID, err)
		return false
	}

	// Check if it's time to generate the report
	switch report.Schedule.Type {
	case "daily":
		return now.Hour() == schedTime.Hour() && now.Minute() == schedTime.Minute()

	case "weekly":
		return now.Weekday() == time.Weekday(report.Schedule.Weekday) &&
			now.Hour() == schedTime.Hour() && now.Minute() == schedTime.Minute()

	case "monthly":
		return now.Day() == report.Schedule.Day &&
			now.Hour() == schedTime.Hour() && now.Minute() == schedTime.Minute()

	default:
		return false
	}
}

// generateReport generates a report
func (r *Reporter) generateReport(ctx context.Context, report *Report) {
	// Update report status
	report.Status = ReportStatusRunning
	report.UpdatedAt = time.Now()
	if err := r.saveReport(ctx, report); err != nil {
		log.Printf("Failed to save report %s: %v", report.ID, err)
		return
	}

	// Get strategy state
	state, err := r.stateManager.GetState(ctx, report.Strategy)
	if err != nil {
		report.Status = ReportStatusFailed
		report.Content = fmt.Sprintf("Failed to get strategy state: %v", err)
		report.UpdatedAt = time.Now()
		r.saveReport(ctx, report)
		return
	}

	// Get performance metrics
	metrics, err := r.getPerformanceMetrics(ctx, report)
	if err != nil {
		report.Status = ReportStatusFailed
		report.Content = fmt.Sprintf("Failed to get performance metrics: %v", err)
		report.UpdatedAt = time.Now()
		r.saveReport(ctx, report)
		return
	}

	// Generate report content
	var content string
	switch report.Format {
	case ReportFormatHTML:
		content, err = r.generateHTMLReport(report, state, metrics)
	case ReportFormatJSON:
		content, err = r.generateJSONReport(report, state, metrics)
	case ReportFormatCSV:
		content, err = r.generateCSVReport(report, state, metrics)
	default:
		err = fmt.Errorf("unsupported report format: %s", report.Format)
	}

	if err != nil {
		report.Status = ReportStatusFailed
		report.Content = fmt.Sprintf("Failed to generate report: %v", err)
		report.UpdatedAt = time.Now()
		r.saveReport(ctx, report)
		return
	}

	// Update report
	report.Status = ReportStatusCompleted
	report.Content = content
	report.GeneratedAt = time.Now()
	report.UpdatedAt = time.Now()
	if err := r.saveReport(ctx, report); err != nil {
		log.Printf("Failed to save report %s: %v", report.ID, err)
		return
	}

	// Notify subscribers
	r.notifySubscribers(report)
}

// getPerformanceMetrics gets performance metrics for a report
func (r *Reporter) getPerformanceMetrics(ctx context.Context, report *Report) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// Get position
	position, err := r.exchange.GetPosition(ctx, report.Symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get position: %w", err)
	}

	if position != nil {
		metrics["position_size"] = position.Quantity
		metrics["entry_price"] = position.EntryPrice
		metrics["unrealized_pnl"] = position.UnrealizedPnL
		metrics["leverage"] = position.Leverage
		metrics["mark_price"] = position.MarkPrice
		metrics["liquidation_price"] = position.LiquidationPrice
		metrics["margin_type"] = position.MarginType
		metrics["maintenance_margin"] = position.MaintenanceMargin
	}

	// Get account balance
	balances, err := r.exchange.GetAccountBalance(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get account balance: %w", err)
	}

	if balance, exists := balances["USDT"]; exists {
		metrics["balance"] = balance.Total
		metrics["available"] = balance.Available
		metrics["locked"] = balance.Locked
		metrics["unrealized_pnl_total"] = balance.UnrealizedPnL
		metrics["realized_pnl_total"] = balance.RealizedPnL
	}

	// Get margin information
	marginInfo, err := r.exchange.GetMarginInfo(ctx)
	if err == nil && marginInfo != nil {
		metrics["total_asset_value"] = marginInfo.TotalAssetValue
		metrics["total_debt_value"] = marginInfo.TotalDebtValue
		metrics["margin_ratio"] = marginInfo.MarginRatio
		metrics["maintenance_margin_ratio"] = marginInfo.MaintenanceMargin
		metrics["margin_call_ratio"] = marginInfo.MarginCallRatio
		metrics["liquidation_ratio"] = marginInfo.LiquidationRatio
	}

	// Get trade history from database
	tradeHistory, err := r.getTradeHistory(ctx, report.Strategy, report.Symbol)
	if err == nil {
		metrics["total_trades"] = len(tradeHistory)
		metrics["trade_history"] = tradeHistory

		// Calculate trade-based metrics
		if len(tradeHistory) > 0 {
			tradeMetrics := r.calculateTradeMetrics(tradeHistory)
			metrics["win_rate"] = tradeMetrics.WinRate
			metrics["profit_factor"] = tradeMetrics.ProfitFactor
			metrics["avg_trade_return"] = tradeMetrics.AvgTradeReturn
			metrics["total_volume"] = tradeMetrics.TotalVolume
			metrics["total_fees"] = tradeMetrics.TotalFees
			metrics["largest_win"] = tradeMetrics.LargestWin
			metrics["largest_loss"] = tradeMetrics.LargestLoss
			metrics["avg_holding_time"] = tradeMetrics.AvgHoldingTime
		}
	}

	// Get performance metrics from database
	perfMetrics, err := r.getPerformanceMetricsFromDB(ctx, report.Strategy)
	if err == nil && len(perfMetrics) > 0 {
		// Get the latest performance metrics
		latest := perfMetrics[len(perfMetrics)-1]
		metrics["equity"] = latest.Equity
		metrics["pnl_daily"] = latest.PnLDaily
		metrics["sharpe_ratio"] = latest.SharpeRatio
		metrics["sortino_ratio"] = latest.SortinoRatio
		metrics["max_drawdown"] = latest.MaxDrawdown
		metrics["win_rate_db"] = latest.WinRate

		// Calculate additional performance ratios
		if len(perfMetrics) > 1 {
			metrics["equity_curve"] = perfMetrics
			metrics["total_return"] = (latest.Equity - perfMetrics[0].Equity) / perfMetrics[0].Equity
			metrics["volatility"] = r.calculateVolatility(perfMetrics)
			metrics["calmar_ratio"] = r.calculateCalmarRatio(latest.Equity, perfMetrics[0].Equity, latest.MaxDrawdown)
		}
	}

	// Get order history
	orderHistory, err := r.exchange.GetOrderHistory(ctx, report.Symbol, time.Now().Add(-30*24*time.Hour), time.Now())
	if err == nil {
		metrics["total_orders"] = len(orderHistory)
		metrics["filled_orders"] = r.countFilledOrders(orderHistory)
		metrics["pending_orders"] = r.countPendingOrders(orderHistory)
		metrics["order_success_rate"] = float64(r.countFilledOrders(orderHistory)) / float64(len(orderHistory))
	}

	// Get risk limits
	riskLimits, err := r.exchange.GetRiskLimits(ctx, report.Symbol)
	if err == nil && riskLimits != nil {
		metrics["max_leverage"] = riskLimits.MaxLeverage
		metrics["max_position_value"] = riskLimits.MaxPositionValue
		metrics["max_order_value"] = riskLimits.MaxOrderValue
		metrics["min_order_value"] = riskLimits.MinOrderValue
	}

	// Calculate additional performance ratios
	if position != nil && len(tradeHistory) > 0 {
		metrics["roi"] = r.calculateROI(position, tradeHistory)
		metrics["risk_reward_ratio"] = r.calculateRiskRewardRatio(tradeHistory)
		metrics["expectancy"] = r.calculateExpectancy(tradeHistory)
	}

	return metrics, nil
}

// TradeMetrics represents calculated trade metrics
type TradeMetrics struct {
	WinRate        float64
	ProfitFactor   float64
	AvgTradeReturn float64
	TotalVolume    float64
	TotalFees      float64
	LargestWin     float64
	LargestLoss    float64
	AvgHoldingTime time.Duration
}

// PerformanceMetric represents performance metric from database
type PerformanceMetric struct {
	Equity       float64
	PnLDaily     float64
	SharpeRatio  float64
	SortinoRatio float64
	MaxDrawdown  float64
	WinRate      float64
	Timestamp    time.Time
}

// getTradeHistory retrieves trade history from database
func (r *Reporter) getTradeHistory(ctx context.Context, strategy, symbol string) ([]*exchange.Trade, error) {
	query := `
		SELECT id, symbol, side, size, price, fee, fee_currency, created_at
		FROM trades
		WHERE strategy_id = (SELECT id FROM strategies WHERE name = $1)
		AND symbol = $2
		ORDER BY created_at DESC
		LIMIT 1000
	`

	rows, err := r.db.QueryContext(ctx, query, strategy, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to query trade history: %w", err)
	}
	defer rows.Close()

	var trades []*exchange.Trade
	for rows.Next() {
		var trade exchange.Trade
		var id string
		var side string
		var size, price, fee float64
		var feeCurrency string
		var createdAt time.Time

		if err := rows.Scan(&id, &trade.Symbol, &side, &size, &price, &fee, &feeCurrency, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan trade: %w", err)
		}

		trade.ID = id
		trade.Side = side
		trade.Quantity = size
		trade.Price = price
		trade.Fee = fee
		trade.FeeCurrency = feeCurrency
		trade.Time = createdAt

		trades = append(trades, &trade)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating trades: %w", err)
	}

	return trades, nil
}

// getPerformanceMetricsFromDB retrieves performance metrics from database
func (r *Reporter) getPerformanceMetricsFromDB(ctx context.Context, strategy string) ([]PerformanceMetric, error) {
	query := `
		SELECT equity, pnl_daily, sharpe_ratio, sortino_ratio, max_drawdown, win_rate, timestamp
		FROM performance_metrics
		WHERE strategy_id = (SELECT id FROM strategies WHERE name = $1)
		ORDER BY timestamp ASC
		LIMIT 1000
	`

	rows, err := r.db.QueryContext(ctx, query, strategy)
	if err != nil {
		return nil, fmt.Errorf("failed to query performance metrics: %w", err)
	}
	defer rows.Close()

	var metrics []PerformanceMetric
	for rows.Next() {
		var metric PerformanceMetric
		var sharpeRatio, sortinoRatio, maxDrawdown, winRate sql.NullFloat64

		if err := rows.Scan(
			&metric.Equity,
			&metric.PnLDaily,
			&sharpeRatio,
			&sortinoRatio,
			&maxDrawdown,
			&winRate,
			&metric.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("failed to scan performance metric: %w", err)
		}

		if sharpeRatio.Valid {
			metric.SharpeRatio = sharpeRatio.Float64
		}
		if sortinoRatio.Valid {
			metric.SortinoRatio = sortinoRatio.Float64
		}
		if maxDrawdown.Valid {
			metric.MaxDrawdown = maxDrawdown.Float64
		}
		if winRate.Valid {
			metric.WinRate = winRate.Float64
		}

		metrics = append(metrics, metric)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating performance metrics: %w", err)
	}

	return metrics, nil
}

// calculateTradeMetrics calculates metrics from trade history
func (r *Reporter) calculateTradeMetrics(trades []*exchange.Trade) *TradeMetrics {
	if len(trades) == 0 {
		return &TradeMetrics{}
	}

	metrics := &TradeMetrics{}
	var wins, losses int
	var totalProfit, totalLoss, totalVolume, totalFees float64
	var totalHoldingTime time.Duration

	for _, trade := range trades {
		// Calculate trade PnL (simplified - assuming buy/sell pairs)
		tradeValue := trade.Price * trade.Quantity
		totalVolume += tradeValue
		totalFees += trade.Fee

		// For simplicity, assume positive PnL for buy trades and negative for sell trades
		// In a real implementation, you would need to track entry/exit pairs
		if trade.Side == "BUY" {
			totalProfit += tradeValue
			wins++
			if tradeValue > metrics.LargestWin {
				metrics.LargestWin = tradeValue
			}
		} else {
			totalLoss += tradeValue
			losses++
			if tradeValue > metrics.LargestLoss {
				metrics.LargestLoss = tradeValue
			}
		}
	}

	// Calculate metrics
	totalTrades := len(trades)
	if totalTrades > 0 {
		metrics.WinRate = float64(wins) / float64(totalTrades)
		metrics.AvgTradeReturn = (totalProfit - totalLoss) / float64(totalTrades)
		metrics.AvgHoldingTime = totalHoldingTime / time.Duration(totalTrades)
	}

	if totalLoss > 0 {
		metrics.ProfitFactor = totalProfit / totalLoss
	}

	metrics.TotalVolume = totalVolume
	metrics.TotalFees = totalFees

	return metrics
}

// calculateVolatility calculates volatility from performance metrics
func (r *Reporter) calculateVolatility(metrics []PerformanceMetric) float64 {
	if len(metrics) < 2 {
		return 0
	}

	var returns []float64
	for i := 1; i < len(metrics); i++ {
		ret := (metrics[i].Equity - metrics[i-1].Equity) / metrics[i-1].Equity
		returns = append(returns, ret)
	}

	if len(returns) == 0 {
		return 0
	}

	// Calculate mean
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	// Calculate variance
	variance := 0.0
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(returns) - 1)

	return math.Sqrt(variance)
}

// calculateCalmarRatio calculates Calmar ratio
func (r *Reporter) calculateCalmarRatio(currentEquity, initialEquity, maxDrawdown float64) float64 {
	if maxDrawdown == 0 {
		return 0
	}

	totalReturn := (currentEquity - initialEquity) / initialEquity
	return totalReturn / maxDrawdown
}

// calculateROI calculates Return on Investment
func (r *Reporter) calculateROI(position *exchange.Position, trades []*exchange.Trade) float64 {
	if position == nil || len(trades) == 0 {
		return 0
	}

	// Simplified ROI calculation
	// In a real implementation, you would track the initial investment
	totalInvestment := position.EntryPrice * position.Quantity
	if totalInvestment == 0 {
		return 0
	}

	return position.UnrealizedPnL / totalInvestment
}

// calculateRiskRewardRatio calculates risk/reward ratio
func (r *Reporter) calculateRiskRewardRatio(trades []*exchange.Trade) float64 {
	if len(trades) == 0 {
		return 0
	}

	var totalRisk, totalReward float64
	for _, trade := range trades {
		tradeValue := trade.Price * trade.Quantity
		if trade.Side == "BUY" {
			totalReward += tradeValue
		} else {
			totalRisk += tradeValue
		}
	}

	if totalRisk == 0 {
		return 0
	}

	return totalReward / totalRisk
}

// calculateExpectancy calculates expectancy
func (r *Reporter) calculateExpectancy(trades []*exchange.Trade) float64 {
	if len(trades) == 0 {
		return 0
	}

	var totalExpectancy float64
	for _, trade := range trades {
		tradeValue := trade.Price * trade.Quantity
		if trade.Side == "BUY" {
			totalExpectancy += tradeValue
		} else {
			totalExpectancy -= tradeValue
		}
	}

	return totalExpectancy / float64(len(trades))
}

// countFilledOrders counts filled orders
func (r *Reporter) countFilledOrders(orders []*exchange.Order) int {
	count := 0
	for _, order := range orders {
		if order.Status == "FILLED" {
			count++
		}
	}
	return count
}

// countPendingOrders counts pending orders
func (r *Reporter) countPendingOrders(orders []*exchange.Order) int {
	count := 0
	for _, order := range orders {
		if order.Status == "NEW" || order.Status == "PENDING" || order.Status == "PARTIALLY_FILLED" {
			count++
		}
	}
	return count
}

// generateHTMLReport generates an HTML report
func (r *Reporter) generateHTMLReport(report *Report, state *state.State, metrics map[string]interface{}) (string, error) {
	tmpl := template.Must(template.New("report").Parse(`
		<html>
		<head>
			<title>{{.Title}}</title>
			<style>
				body { font-family: Arial, sans-serif; }
				table { border-collapse: collapse; width: 100%; }
				th, td { padding: 8px; text-align: left; border-bottom: 1px solid #ddd; }
				th { background-color: #f2f2f2; }
			</style>
		</head>
		<body>
			<h1>{{.Title}}</h1>
			<h2>Strategy Information</h2>
			<table>
				<tr><th>Strategy</th><td>{{.Strategy}}</td></tr>
				<tr><th>Symbol</th><td>{{.Symbol}}</td></tr>
				<tr><th>Status</th><td>{{.Status}}</td></tr>
				<tr><th>Generated At</th><td>{{.GeneratedAt}}</td></tr>
			</table>
			<h2>Performance Metrics</h2>
			<table>
				{{range $key, $value := .Metrics}}
				<tr><th>{{$key}}</th><td>{{$value}}</td></tr>
				{{end}}
			</table>
		</body>
		</html>
	`))

	data := struct {
		Title       string
		Strategy    string
		Symbol      string
		Status      string
		GeneratedAt string
		Metrics     map[string]interface{}
	}{
		Title:       fmt.Sprintf("%s Report - %s", report.Type, report.Strategy),
		Strategy:    report.Strategy,
		Symbol:      report.Symbol,
		Status:      string(state.Status),
		GeneratedAt: time.Now().Format(time.RFC3339),
		Metrics:     metrics,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// generateJSONReport generates a JSON report
func (r *Reporter) generateJSONReport(report *Report, state *state.State, metrics map[string]interface{}) (string, error) {
	data := struct {
		Strategy    string                 `json:"strategy"`
		Symbol      string                 `json:"symbol"`
		Type        ReportType             `json:"type"`
		Status      string                 `json:"status"`
		GeneratedAt time.Time              `json:"generated_at"`
		Metrics     map[string]interface{} `json:"metrics"`
	}{
		Strategy:    report.Strategy,
		Symbol:      report.Symbol,
		Type:        report.Type,
		Status:      string(state.Status),
		GeneratedAt: time.Now(),
		Metrics:     metrics,
	}

	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(content), nil
}

// generateCSVReport generates a CSV report
func (r *Reporter) generateCSVReport(report *Report, state *state.State, metrics map[string]interface{}) (string, error) {
	var buf bytes.Buffer

	// Write header
	buf.WriteString("Metric,Value\n")

	// Write strategy info
	buf.WriteString(fmt.Sprintf("Strategy,%s\n", report.Strategy))
	buf.WriteString(fmt.Sprintf("Symbol,%s\n", report.Symbol))
	buf.WriteString(fmt.Sprintf("Type,%s\n", report.Type))
	buf.WriteString(fmt.Sprintf("Status,%s\n", state.Status))
	buf.WriteString(fmt.Sprintf("Generated At,%s\n", time.Now().Format(time.RFC3339)))

	// Write metrics
	for key, value := range metrics {
		buf.WriteString(fmt.Sprintf("%s,%v\n", key, value))
	}

	return buf.String(), nil
}

// notifySubscribers notifies report subscribers
func (r *Reporter) notifySubscribers(report *Report) {
	if callbacks, exists := r.subscribers[report.ID]; exists {
		for _, callback := range callbacks {
			callback(report)
		}
	}
}

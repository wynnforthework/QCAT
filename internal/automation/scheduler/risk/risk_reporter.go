package risk

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/automation/scheduler/shared"
	"qcat/internal/config"
	"qcat/internal/database"
)

// RiskReporter handles comprehensive risk reporting and logging
type RiskReporter struct {
	config      *config.Config
	db          *database.DB
	riskMonitor *RiskMonitor
	mu          sync.RWMutex
	isRunning   bool
	reports     []RiskReport
	maxReports  int
}

// RiskReport represents a comprehensive risk report
type RiskReport struct {
	ID                string                 `json:"id"`
	Type              ReportType             `json:"type"`
	GeneratedAt       time.Time              `json:"generated_at"`
	Period            *shared.TimePeriod     `json:"period"`
	MarginStatus      *MarginStatus          `json:"margin_status"`
	PositionRisk      *PositionRiskReport    `json:"position_risk"`
	MarketAnomalies   []*MarketAnomalyReport `json:"market_anomalies"`
	RiskActions       []RiskAction           `json:"risk_actions"`
	Summary           *RiskSummary           `json:"summary"`
	Recommendations   []string               `json:"recommendations"`
	Metrics           map[string]interface{} `json:"metrics"`
	Alerts            []RiskAlert            `json:"alerts"`
}

// ReportType defines types of risk reports
type ReportType int

const (
	ReportTypeRealTime ReportType = iota
	ReportTypeDaily
	ReportTypeWeekly
	ReportTypeMonthly
	ReportTypeIncident
	ReportTypeCompliance
)

// String returns string representation of ReportType
func (rt ReportType) String() string {
	switch rt {
	case ReportTypeRealTime:
		return "REAL_TIME"
	case ReportTypeDaily:
		return "DAILY"
	case ReportTypeWeekly:
		return "WEEKLY"
	case ReportTypeMonthly:
		return "MONTHLY"
	case ReportTypeIncident:
		return "INCIDENT"
	case ReportTypeCompliance:
		return "COMPLIANCE"
	default:
		return "UNKNOWN"
	}
}

// RiskSummary provides a high-level summary of risk metrics
type RiskSummary struct {
	OverallRiskLevel    shared.RiskLevel `json:"overall_risk_level"`
	TotalExposure       float64          `json:"total_exposure"`
	MarginUtilization   float64          `json:"margin_utilization"`
	PortfolioVaR        float64          `json:"portfolio_var"`
	MaxDrawdown         float64          `json:"max_drawdown"`
	ActivePositions     int              `json:"active_positions"`
	RiskActionsToday    int              `json:"risk_actions_today"`
	AnomaliesDetected   int              `json:"anomalies_detected"`
	ComplianceStatus    string           `json:"compliance_status"`
	LastUpdated         time.Time        `json:"last_updated"`
}

// RiskAlert represents a risk alert
type RiskAlert struct {
	ID          string                 `json:"id"`
	Type        AlertType              `json:"type"`
	Severity    shared.Severity        `json:"severity"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Metrics     map[string]interface{} `json:"metrics"`
	Threshold   float64                `json:"threshold"`
	CurrentValue float64               `json:"current_value"`
	CreatedAt   time.Time              `json:"created_at"`
	AcknowledgedAt *time.Time          `json:"acknowledged_at,omitempty"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
}

// AlertType defines types of risk alerts
type AlertType int

const (
	AlertTypeMarginThreshold AlertType = iota
	AlertTypeVaRExceeded
	AlertTypeDrawdownLimit
	AlertTypeConcentrationRisk
	AlertTypeVolatilitySpike
	AlertTypeLiquidityDrop
	AlertTypeCorrelationBreakdown
	AlertTypeSystemHealth
)

// String returns string representation of AlertType
func (at AlertType) String() string {
	switch at {
	case AlertTypeMarginThreshold:
		return "MARGIN_THRESHOLD"
	case AlertTypeVaRExceeded:
		return "VAR_EXCEEDED"
	case AlertTypeDrawdownLimit:
		return "DRAWDOWN_LIMIT"
	case AlertTypeConcentrationRisk:
		return "CONCENTRATION_RISK"
	case AlertTypeVolatilitySpike:
		return "VOLATILITY_SPIKE"
	case AlertTypeLiquidityDrop:
		return "LIQUIDITY_DROP"
	case AlertTypeCorrelationBreakdown:
		return "CORRELATION_BREAKDOWN"
	case AlertTypeSystemHealth:
		return "SYSTEM_HEALTH"
	default:
		return "UNKNOWN"
	}
}

// NewRiskReporter creates a new risk reporter
func NewRiskReporter(cfg *config.Config, db *database.DB, riskMonitor *RiskMonitor) *RiskReporter {
	return &RiskReporter{
		config:      cfg,
		db:          db,
		riskMonitor: riskMonitor,
		reports:     make([]RiskReport, 0),
		maxReports:  1000, // Keep last 1000 reports in memory
	}
}

// GenerateRealTimeReport generates a real-time risk report
func (rr *RiskReporter) GenerateRealTimeReport(ctx context.Context) (*RiskReport, error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	log.Printf("Generating real-time risk report")

	report := &RiskReport{
		ID:          shared.GenerateID("risk_report"),
		Type:        ReportTypeRealTime,
		GeneratedAt: time.Now(),
		Metrics:     make(map[string]interface{}),
		Alerts:      make([]RiskAlert, 0),
	}

	// Get current margin status
	marginStatus, err := rr.riskMonitor.CheckMarginRatio(ctx)
	if err != nil {
		log.Printf("Warning: Could not get margin status for report: %v", err)
	} else {
		report.MarginStatus = marginStatus
	}

	// Get position risk report
	positionRisk, err := rr.riskMonitor.MonitorPositionRisk(ctx)
	if err != nil {
		log.Printf("Warning: Could not get position risk for report: %v", err)
	} else {
		report.PositionRisk = positionRisk
	}

	// Check for market anomalies
	anomaly, err := rr.riskMonitor.DetectAbnormalMarket(ctx)
	if err != nil {
		log.Printf("Warning: Could not detect market anomalies for report: %v", err)
	} else if anomaly != nil {
		report.MarketAnomalies = []*MarketAnomalyReport{anomaly}
	}

	// Get recent risk actions
	riskActions, err := rr.getRecentRiskActions(ctx, time.Hour*24) // Last 24 hours
	if err != nil {
		log.Printf("Warning: Could not get recent risk actions: %v", err)
	} else {
		report.RiskActions = riskActions
	}

	// Generate summary
	report.Summary = rr.generateRiskSummary(report)

	// Generate alerts
	alerts := rr.generateAlerts(ctx, report)
	report.Alerts = alerts

	// Generate recommendations
	report.Recommendations = rr.generateRecommendations(report)

	// Update metrics
	rr.updateReportMetrics(report)

	// Store report
	rr.storeReport(*report)

	// Save to database
	go func() {
		err := rr.saveReportToDatabase(context.Background(), report)
		if err != nil {
			log.Printf("Error saving report to database: %v", err)
		}
	}()

	log.Printf("Real-time risk report generated: %s", report.ID)
	return report, nil
}

// GenerateDailyReport generates a daily risk report
func (rr *RiskReporter) GenerateDailyReport(ctx context.Context, date time.Time) (*RiskReport, error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	log.Printf("Generating daily risk report for %s", date.Format("2006-01-02"))

	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	report := &RiskReport{
		ID:          shared.GenerateID("daily_report"),
		Type:        ReportTypeDaily,
		GeneratedAt: time.Now(),
		Period: &shared.TimePeriod{
			Start: startOfDay,
			End:   endOfDay,
		},
		Metrics: make(map[string]interface{}),
		Alerts:  make([]RiskAlert, 0),
	}

	// Get historical data for the day
	dailyMetrics, err := rr.getDailyMetrics(ctx, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily metrics: %w", err)
	}
	report.Metrics = dailyMetrics

	// Get risk actions for the day
	riskActions, err := rr.getRiskActionsForPeriod(ctx, startOfDay, endOfDay)
	if err != nil {
		log.Printf("Warning: Could not get risk actions for daily report: %v", err)
	} else {
		report.RiskActions = riskActions
	}

	// Get alerts for the day
	alerts, err := rr.getAlertsForPeriod(ctx, startOfDay, endOfDay)
	if err != nil {
		log.Printf("Warning: Could not get alerts for daily report: %v", err)
	} else {
		report.Alerts = alerts
	}

	// Generate summary for the day
	report.Summary = rr.generateDailySummary(report, dailyMetrics)

	// Generate recommendations
	report.Recommendations = rr.generateDailyRecommendations(report)

	// Store report
	rr.storeReport(*report)

	// Save to database
	go func() {
		err := rr.saveReportToDatabase(context.Background(), report)
		if err != nil {
			log.Printf("Error saving daily report to database: %v", err)
		}
	}()

	log.Printf("Daily risk report generated: %s", report.ID)
	return report, nil
}

// GenerateIncidentReport generates an incident-specific risk report
func (rr *RiskReporter) GenerateIncidentReport(ctx context.Context, incidentID string, description string) (*RiskReport, error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	log.Printf("Generating incident risk report for: %s", incidentID)

	report := &RiskReport{
		ID:          shared.GenerateID("incident_report"),
		Type:        ReportTypeIncident,
		GeneratedAt: time.Now(),
		Metrics:     make(map[string]interface{}),
		Alerts:      make([]RiskAlert, 0),
	}

	// Add incident-specific information
	report.Metrics["incident_id"] = incidentID
	report.Metrics["incident_description"] = description

	// Get current state at time of incident
	marginStatus, err := rr.riskMonitor.CheckMarginRatio(ctx)
	if err != nil {
		log.Printf("Warning: Could not get margin status for incident report: %v", err)
	} else {
		report.MarginStatus = marginStatus
	}

	positionRisk, err := rr.riskMonitor.MonitorPositionRisk(ctx)
	if err != nil {
		log.Printf("Warning: Could not get position risk for incident report: %v", err)
	} else {
		report.PositionRisk = positionRisk
	}

	// Get recent risk actions (last hour)
	riskActions, err := rr.getRecentRiskActions(ctx, time.Hour)
	if err != nil {
		log.Printf("Warning: Could not get recent risk actions for incident report: %v", err)
	} else {
		report.RiskActions = riskActions
	}

	// Generate incident-specific summary
	report.Summary = rr.generateIncidentSummary(report, incidentID, description)

	// Generate incident-specific recommendations
	report.Recommendations = rr.generateIncidentRecommendations(report)

	// Store report
	rr.storeReport(*report)

	// Save to database with high priority
	err = rr.saveReportToDatabase(ctx, report)
	if err != nil {
		log.Printf("Error saving incident report to database: %v", err)
	}

	log.Printf("Incident risk report generated: %s", report.ID)
	return report, nil
}

// Helper methods

// generateRiskSummary generates a risk summary for real-time reports
func (rr *RiskReporter) generateRiskSummary(report *RiskReport) *RiskSummary {
	summary := &RiskSummary{
		OverallRiskLevel:  shared.RiskLevelLow,
		ComplianceStatus:  "COMPLIANT",
		LastUpdated:       time.Now(),
	}

	// Determine overall risk level
	if report.MarginStatus != nil {
		summary.MarginUtilization = report.MarginStatus.MarginRatio
		summary.TotalExposure = report.MarginStatus.TotalEquity
		
		if report.MarginStatus.RiskLevel > summary.OverallRiskLevel {
			summary.OverallRiskLevel = report.MarginStatus.RiskLevel
		}
	}

	if report.PositionRisk != nil {
		summary.PortfolioVaR = report.PositionRisk.VaR
		summary.MaxDrawdown = report.PositionRisk.MaxDrawdown
		summary.ActivePositions = len(report.PositionRisk.Positions)
	}

	// Count anomalies
	summary.AnomaliesDetected = len(report.MarketAnomalies)

	// Count risk actions
	summary.RiskActionsToday = len(report.RiskActions)

	// Determine compliance status
	if summary.OverallRiskLevel >= shared.RiskLevelHigh {
		summary.ComplianceStatus = "NON_COMPLIANT"
	} else if summary.OverallRiskLevel >= shared.RiskLevelMedium {
		summary.ComplianceStatus = "MONITORING"
	}

	return summary
}

// generateAlerts generates risk alerts based on current conditions
func (rr *RiskReporter) generateAlerts(ctx context.Context, report *RiskReport) []RiskAlert {
	var alerts []RiskAlert

	// Margin threshold alerts
	if report.MarginStatus != nil {
		if report.MarginStatus.MarginRatio > 0.8 {
			alert := RiskAlert{
				ID:           shared.GenerateID("alert"),
				Type:         AlertTypeMarginThreshold,
				Severity:     rr.determineSeverityFromRiskLevel(report.MarginStatus.RiskLevel),
				Title:        "High Margin Utilization",
				Message:      fmt.Sprintf("Margin utilization is %.2f%%, exceeding 80%% threshold", report.MarginStatus.MarginRatio*100),
				Threshold:    0.8,
				CurrentValue: report.MarginStatus.MarginRatio,
				CreatedAt:    time.Now(),
				Metrics: map[string]interface{}{
					"margin_ratio":     report.MarginStatus.MarginRatio,
					"total_equity":     report.MarginStatus.TotalEquity,
					"used_margin":      report.MarginStatus.UsedMargin,
				},
			}
			alerts = append(alerts, alert)
		}
	}

	// VaR threshold alerts
	if report.PositionRisk != nil {
		varThreshold := 0.05 // 5% of portfolio
		if report.PositionRisk.VaR > varThreshold {
			alert := RiskAlert{
				ID:           shared.GenerateID("alert"),
				Type:         AlertTypeVaRExceeded,
				Severity:     shared.SeverityError,
				Title:        "Portfolio VaR Exceeded",
				Message:      fmt.Sprintf("Portfolio VaR is %.4f, exceeding %.4f threshold", report.PositionRisk.VaR, varThreshold),
				Threshold:    varThreshold,
				CurrentValue: report.PositionRisk.VaR,
				CreatedAt:    time.Now(),
				Metrics: map[string]interface{}{
					"portfolio_var":      report.PositionRisk.VaR,
					"expected_shortfall": report.PositionRisk.ExpectedShortfall,
					"total_risk":         report.PositionRisk.TotalRisk,
				},
			}
			alerts = append(alerts, alert)
		}

		// Concentration risk alerts
		if report.PositionRisk.ConcentrationRisk > 0.5 {
			alert := RiskAlert{
				ID:           shared.GenerateID("alert"),
				Type:         AlertTypeConcentrationRisk,
				Severity:     shared.SeverityWarning,
				Title:        "High Concentration Risk",
				Message:      fmt.Sprintf("Portfolio concentration risk is %.2f%%, exceeding 50%% threshold", report.PositionRisk.ConcentrationRisk*100),
				Threshold:    0.5,
				CurrentValue: report.PositionRisk.ConcentrationRisk,
				CreatedAt:    time.Now(),
				Metrics: map[string]interface{}{
					"concentration_risk": report.PositionRisk.ConcentrationRisk,
					"position_count":     len(report.PositionRisk.Positions),
				},
			}
			alerts = append(alerts, alert)
		}
	}

	// Market anomaly alerts
	for _, anomaly := range report.MarketAnomalies {
		alert := RiskAlert{
			ID:        shared.GenerateID("alert"),
			Type:      rr.alertTypeFromAnomalyType(anomaly.AnomalyType),
			Severity:  anomaly.Severity,
			Title:     fmt.Sprintf("Market Anomaly: %s", anomaly.AnomalyType.String()),
			Message:   anomaly.Description,
			CreatedAt: time.Now(),
			Metrics: map[string]interface{}{
				"anomaly_type":      anomaly.AnomalyType.String(),
				"affected_symbols":  len(anomaly.AffectedSymbols),
				"confidence":        anomaly.Confidence,
			},
		}
		alerts = append(alerts, alert)
	}

	return alerts
}

// generateRecommendations generates recommendations based on the report
func (rr *RiskReporter) generateRecommendations(report *RiskReport) []string {
	var recommendations []string

	// Margin-based recommendations
	if report.MarginStatus != nil {
		recommendations = append(recommendations, report.MarginStatus.Recommendations...)
	}

	// Position risk-based recommendations
	if report.PositionRisk != nil {
		recommendations = append(recommendations, report.PositionRisk.Recommendations...)
	}

	// Market anomaly-based recommendations
	for _, anomaly := range report.MarketAnomalies {
		recommendations = append(recommendations, anomaly.RecommendedActions...)
	}

	// General recommendations based on overall risk level
	if report.Summary != nil {
		switch report.Summary.OverallRiskLevel {
		case shared.RiskLevelCritical:
			recommendations = append(recommendations, "URGENT: Consider emergency risk reduction measures")
			recommendations = append(recommendations, "Review all positions immediately")
			recommendations = append(recommendations, "Increase monitoring frequency to real-time")
		case shared.RiskLevelHigh:
			recommendations = append(recommendations, "Reduce position sizes to lower risk exposure")
			recommendations = append(recommendations, "Tighten stop-loss levels")
			recommendations = append(recommendations, "Increase cash reserves")
		case shared.RiskLevelMedium:
			recommendations = append(recommendations, "Monitor positions closely")
			recommendations = append(recommendations, "Review risk management parameters")
		}
	}

	// Remove duplicates
	return rr.removeDuplicateRecommendations(recommendations)
}

// Helper methods for database operations and utilities

func (rr *RiskReporter) getRecentRiskActions(ctx context.Context, duration time.Duration) ([]RiskAction, error) {
	query := `
		SELECT 
			id, type, trigger_reason, description, success, 
			amount_reduced, executed_at, duration_ms
		FROM risk_actions 
		WHERE executed_at >= $1
		ORDER BY executed_at DESC
		LIMIT 100
	`
	
	since := time.Now().Add(-duration)
	rows, err := rr.db.QueryContext(ctx, query, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []RiskAction
	for rows.Next() {
		var action RiskAction
		var durationMs int64
		
		err := rows.Scan(
			&action.ID, &action.Type, &action.Trigger, &action.Description,
			&action.Result.Success, &action.Result.AmountReduced,
			&action.ExecutedAt, &durationMs,
		)
		if err != nil {
			continue
		}
		
		action.Duration = time.Duration(durationMs) * time.Millisecond
		actions = append(actions, action)
	}

	return actions, nil
}

func (rr *RiskReporter) storeReport(report RiskReport) {
	rr.reports = append(rr.reports, report)
	
	// Keep only the most recent reports
	if len(rr.reports) > rr.maxReports {
		rr.reports = rr.reports[1:]
	}
}

func (rr *RiskReporter) saveReportToDatabase(ctx context.Context, report *RiskReport) error {
	// Convert report to JSON
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	query := `
		INSERT INTO risk_reports (
			id, type, generated_at, period_start, period_end, 
			report_data, summary_risk_level, summary_margin_utilization,
			summary_portfolio_var, summary_active_positions
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	var periodStart, periodEnd *time.Time
	if report.Period != nil {
		periodStart = &report.Period.Start
		periodEnd = &report.Period.End
	}

	var summaryRiskLevel string
	var summaryMarginUtil, summaryVaR float64
	var summaryActivePositions int

	if report.Summary != nil {
		summaryRiskLevel = report.Summary.OverallRiskLevel.String()
		summaryMarginUtil = report.Summary.MarginUtilization
		summaryVaR = report.Summary.PortfolioVaR
		summaryActivePositions = report.Summary.ActivePositions
	}

	_, err = rr.db.ExecContext(ctx, query,
		report.ID,
		report.Type.String(),
		report.GeneratedAt,
		periodStart,
		periodEnd,
		string(reportJSON),
		summaryRiskLevel,
		summaryMarginUtil,
		summaryVaR,
		summaryActivePositions,
	)

	return err
}

// Additional helper methods would be implemented here...
// (getDailyMetrics, getRiskActionsForPeriod, getAlertsForPeriod, etc.)

func (rr *RiskReporter) determineSeverityFromRiskLevel(riskLevel shared.RiskLevel) shared.Severity {
	switch riskLevel {
	case shared.RiskLevelCritical:
		return shared.SeverityCritical
	case shared.RiskLevelHigh:
		return shared.SeverityError
	case shared.RiskLevelMedium:
		return shared.SeverityWarning
	default:
		return shared.SeverityInfo
	}
}

func (rr *RiskReporter) alertTypeFromAnomalyType(anomalyType shared.AnomalyType) AlertType {
	switch anomalyType {
	case shared.AnomalyTypeVolatilitySpike:
		return AlertTypeVolatilitySpike
	case shared.AnomalyTypeLiquidityDrop:
		return AlertTypeLiquidityDrop
	case shared.AnomalyTypeCorrelationBreakdown:
		return AlertTypeCorrelationBreakdown
	default:
		return AlertTypeSystemHealth
	}
}

func (rr *RiskReporter) removeDuplicateRecommendations(recommendations []string) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, rec := range recommendations {
		if !seen[rec] {
			seen[rec] = true
			result = append(result, rec)
		}
	}
	
	return result
}

// getDailyMetrics gets daily metrics for a time period
func (rr *RiskReporter) getDailyMetrics(ctx context.Context, start, end time.Time) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})
	
	// This would query the database for daily metrics
	// For now, return mock data
	metrics["avg_margin_ratio"] = 0.65
	metrics["max_margin_ratio"] = 0.85
	metrics["min_margin_ratio"] = 0.45
	metrics["avg_portfolio_var"] = 0.03
	metrics["max_drawdown"] = 0.02
	
	return metrics, nil
}

// getRiskActionsForPeriod gets risk actions for a time period
func (rr *RiskReporter) getRiskActionsForPeriod(ctx context.Context, start, end time.Time) ([]RiskAction, error) {
	// This would query the database for risk actions in the period
	// For now, return empty slice
	return []RiskAction{}, nil
}

// getAlertsForPeriod gets alerts for a time period
func (rr *RiskReporter) getAlertsForPeriod(ctx context.Context, start, end time.Time) ([]RiskAlert, error) {
	// This would query the database for alerts in the period
	// For now, return empty slice
	return []RiskAlert{}, nil
}

// generateDailySummary generates a summary for daily reports
func (rr *RiskReporter) generateDailySummary(report *RiskReport, dailyMetrics map[string]interface{}) *RiskSummary {
	summary := &RiskSummary{
		OverallRiskLevel:  shared.RiskLevelLow,
		ComplianceStatus:  "COMPLIANT",
		LastUpdated:       time.Now(),
	}
	
	// Extract metrics from dailyMetrics
	if avgMargin, ok := dailyMetrics["avg_margin_ratio"].(float64); ok {
		summary.MarginUtilization = avgMargin
	}
	if avgVar, ok := dailyMetrics["avg_portfolio_var"].(float64); ok {
		summary.PortfolioVaR = avgVar
	}
	if maxDrawdown, ok := dailyMetrics["max_drawdown"].(float64); ok {
		summary.MaxDrawdown = maxDrawdown
	}
	
	summary.RiskActionsToday = len(report.RiskActions)
	summary.AnomaliesDetected = len(report.MarketAnomalies)
	
	return summary
}

// generateDailyRecommendations generates recommendations for daily reports
func (rr *RiskReporter) generateDailyRecommendations(report *RiskReport) []string {
	var recommendations []string
	
	if report.Summary != nil {
		if report.Summary.MarginUtilization > 0.8 {
			recommendations = append(recommendations, "Daily average margin utilization is high - consider reducing overall exposure")
		}
		if report.Summary.PortfolioVaR > 0.05 {
			recommendations = append(recommendations, "Daily portfolio VaR is elevated - review risk management parameters")
		}
		if report.Summary.RiskActionsToday > 5 {
			recommendations = append(recommendations, "Multiple risk actions triggered today - review trading strategies")
		}
	}
	
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Daily risk metrics within acceptable ranges")
	}
	
	return recommendations
}

// generateIncidentSummary generates a summary for incident reports
func (rr *RiskReporter) generateIncidentSummary(report *RiskReport, incidentID, description string) *RiskSummary {
	summary := &RiskSummary{
		OverallRiskLevel:  shared.RiskLevelHigh, // Incidents are typically high risk
		ComplianceStatus:  "INCIDENT",
		LastUpdated:       time.Now(),
	}
	
	if report.MarginStatus != nil {
		summary.MarginUtilization = report.MarginStatus.MarginRatio
		summary.TotalExposure = report.MarginStatus.TotalEquity
	}
	
	if report.PositionRisk != nil {
		summary.PortfolioVaR = report.PositionRisk.VaR
		summary.MaxDrawdown = report.PositionRisk.MaxDrawdown
		summary.ActivePositions = len(report.PositionRisk.Positions)
	}
	
	summary.RiskActionsToday = len(report.RiskActions)
	summary.AnomaliesDetected = len(report.MarketAnomalies)
	
	return summary
}

// generateIncidentRecommendations generates recommendations for incident reports
func (rr *RiskReporter) generateIncidentRecommendations(report *RiskReport) []string {
	var recommendations []string
	
	recommendations = append(recommendations, "INCIDENT: Immediate review required")
	recommendations = append(recommendations, "Investigate root cause of incident")
	recommendations = append(recommendations, "Review and update risk management procedures")
	recommendations = append(recommendations, "Consider additional safeguards to prevent recurrence")
	
	if report.MarginStatus != nil && report.MarginStatus.RiskLevel >= shared.RiskLevelHigh {
		recommendations = append(recommendations, "High margin risk detected - consider emergency position reduction")
	}
	
	if report.PositionRisk != nil && report.PositionRisk.VaR > 0.1 {
		recommendations = append(recommendations, "Elevated portfolio VaR - implement immediate risk reduction measures")
	}
	
	return recommendations
}

func (rr *RiskReporter) updateReportMetrics(report *RiskReport) {
	report.Metrics["generation_time"] = time.Now()
	report.Metrics["report_size"] = len(fmt.Sprintf("%+v", report))
	report.Metrics["alert_count"] = len(report.Alerts)
	report.Metrics["recommendation_count"] = len(report.Recommendations)
	
	if report.MarginStatus != nil {
		report.Metrics["margin_status_available"] = true
	}
	if report.PositionRisk != nil {
		report.Metrics["position_risk_available"] = true
	}
}

// GetReports returns recent reports
func (rr *RiskReporter) GetReports(limit int) []RiskReport {
	rr.mu.RLock()
	defer rr.mu.RUnlock()
	
	if limit <= 0 || limit > len(rr.reports) {
		limit = len(rr.reports)
	}
	
	// Return the most recent reports
	start := len(rr.reports) - limit
	reports := make([]RiskReport, limit)
	copy(reports, rr.reports[start:])
	
	return reports
}

// Start starts the risk reporter
func (rr *RiskReporter) Start() error {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	
	rr.isRunning = true
	log.Printf("Risk reporter started")
	return nil
}

// Stop stops the risk reporter
func (rr *RiskReporter) Stop() error {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	
	rr.isRunning = false
	log.Printf("Risk reporter stopped")
	return nil
}
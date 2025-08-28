package dao

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// postgresProtectionMetricsDAO implements ProtectionMetricsDAO for PostgreSQL
type postgresProtectionMetricsDAO struct {
	*baseDAO
}

// Insert inserts a new protection metrics record
func (dao *postgresProtectionMetricsDAO) Insert(ctx context.Context, metrics *ProtectionMetrics) error {
	query := `
		INSERT INTO protection_metrics (
			timestamp, circuit_breaker_triggered, emergency_activations, auto_transfers,
			manual_interventions, losses_prevented, profits_secured, max_loss_avoided,
			avg_response_time_ms, protection_accuracy, false_positive_rate,
			system_uptime_seconds, last_emergency_test
		) VALUES (
			:timestamp, :circuit_breaker_triggered, :emergency_activations, :auto_transfers,
			:manual_interventions, :losses_prevented, :profits_secured, :max_loss_avoided,
			:avg_response_time_ms, :protection_accuracy, :false_positive_rate,
			:system_uptime_seconds, :last_emergency_test
		) RETURNING id, created_at`
	
	rows, err := dao.db.NamedQuery(query, metrics)
	if err != nil {
		return fmt.Errorf("failed to insert protection metrics: %w", err)
	}
	defer rows.Close()
	
	if rows.Next() {
		err = rows.Scan(&metrics.ID, &metrics.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to scan inserted protection metrics: %w", err)
		}
	}
	
	return nil
}

// Update updates an existing protection metrics record
func (dao *postgresProtectionMetricsDAO) Update(ctx context.Context, metrics *ProtectionMetrics) error {
	query := `
		UPDATE protection_metrics SET
			timestamp = :timestamp,
			circuit_breaker_triggered = :circuit_breaker_triggered,
			emergency_activations = :emergency_activations,
			auto_transfers = :auto_transfers,
			manual_interventions = :manual_interventions,
			losses_prevented = :losses_prevented,
			profits_secured = :profits_secured,
			max_loss_avoided = :max_loss_avoided,
			avg_response_time_ms = :avg_response_time_ms,
			protection_accuracy = :protection_accuracy,
			false_positive_rate = :false_positive_rate,
			system_uptime_seconds = :system_uptime_seconds,
			last_emergency_test = :last_emergency_test
		WHERE id = :id`
	
	result, err := dao.db.NamedExecContext(ctx, query, metrics)
	if err != nil {
		return fmt.Errorf("failed to update protection metrics: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("protection metrics with id %d not found", metrics.ID)
	}
	
	return nil
}

// GetLatest retrieves the most recent protection metrics
func (dao *postgresProtectionMetricsDAO) GetLatest(ctx context.Context) (*ProtectionMetrics, error) {
	query := `
		SELECT id, timestamp, circuit_breaker_triggered, emergency_activations, auto_transfers,
			   manual_interventions, losses_prevented, profits_secured, max_loss_avoided,
			   avg_response_time_ms, protection_accuracy, false_positive_rate,
			   system_uptime_seconds, last_emergency_test, created_at
		FROM protection_metrics
		ORDER BY timestamp DESC
		LIMIT 1`
	
	var metrics ProtectionMetrics
	err := dao.db.GetContext(ctx, &metrics, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest protection metrics: %w", err)
	}
	
	return &metrics, nil
}

// GetByTimeRange retrieves protection metrics within a time range
func (dao *postgresProtectionMetricsDAO) GetByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*ProtectionMetrics, error) {
	query := `
		SELECT id, timestamp, circuit_breaker_triggered, emergency_activations, auto_transfers,
			   manual_interventions, losses_prevented, profits_secured, max_loss_avoided,
			   avg_response_time_ms, protection_accuracy, false_positive_rate,
			   system_uptime_seconds, last_emergency_test, created_at
		FROM protection_metrics
		WHERE timestamp >= $1 AND timestamp <= $2
		ORDER BY timestamp ASC`
	
	var metrics []*ProtectionMetrics
	err := dao.db.SelectContext(ctx, &metrics, query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get protection metrics by time range: %w", err)
	}
	
	return metrics, nil
}
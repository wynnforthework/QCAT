-- Migration rollback: Drop risk_metrics table and related objects

BEGIN;

-- Drop indexes
DROP INDEX IF EXISTS idx_risk_alerts_strategy_id;
DROP INDEX IF EXISTS idx_risk_alerts_symbol;
DROP INDEX IF EXISTS idx_risk_alerts_created_at;
DROP INDEX IF EXISTS idx_risk_alerts_severity;
DROP INDEX IF EXISTS idx_risk_alerts_status;

DROP INDEX IF EXISTS idx_risk_metrics_risk_score;
DROP INDEX IF EXISTS idx_risk_metrics_created_at;
DROP INDEX IF EXISTS idx_risk_metrics_symbol;
DROP INDEX IF EXISTS idx_risk_metrics_strategy_id;

-- Drop tables
DROP TABLE IF EXISTS risk_alerts;
DROP TABLE IF EXISTS risk_metrics;

COMMIT;

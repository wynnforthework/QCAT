-- Migration: Rollback database error fixes
-- Version: 000017
-- Description: Rollback all database error fixes

-- Drop indexes
DROP INDEX IF EXISTS idx_onboarding_reports_request_id;
DROP INDEX IF EXISTS idx_strategies_symbol;
DROP INDEX IF EXISTS idx_account_balances_updated_at;
DROP INDEX IF EXISTS idx_account_balances_exchange_asset;
DROP INDEX IF EXISTS idx_profit_optimization_history_started_at;
DROP INDEX IF EXISTS idx_profit_optimization_history_status;
DROP INDEX IF EXISTS idx_profit_optimization_history_optimization_id;
DROP INDEX IF EXISTS idx_profit_optimization_history_strategy_id;
DROP INDEX IF EXISTS idx_performance_forecasts_created_at;
DROP INDEX IF EXISTS idx_performance_forecasts_strategy_id;

-- Remove added columns
ALTER TABLE onboarding_reports DROP COLUMN IF EXISTS request_id;
ALTER TABLE wallet_balances DROP COLUMN IF EXISTS wallet_type;
ALTER TABLE hedge_history DROP COLUMN IF EXISTS correlation_matrix;
ALTER TABLE strategies DROP COLUMN IF EXISTS symbol;

-- Drop created tables
DROP TABLE IF EXISTS account_balances;
DROP TABLE IF EXISTS profit_optimization_history;
DROP TABLE IF EXISTS performance_forecasts;

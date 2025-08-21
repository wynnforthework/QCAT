-- Migration: Rollback additional field fixes
-- Version: 000018
-- Description: Rollback additional missing field fixes

-- Drop indexes
DROP INDEX IF EXISTS idx_wallet_balances_balance;
DROP INDEX IF EXISTS idx_hedge_history_total_operations;
DROP INDEX IF EXISTS idx_profit_optimization_history_objective_value;
DROP INDEX IF EXISTS idx_strategies_config;

-- Remove added columns
ALTER TABLE wallet_balances DROP COLUMN IF EXISTS balance;
ALTER TABLE hedge_history DROP COLUMN IF EXISTS total_operations;
ALTER TABLE profit_optimization_history DROP COLUMN IF EXISTS objective_value;
ALTER TABLE strategies DROP COLUMN IF EXISTS config;

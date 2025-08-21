-- Migration: Rollback missing columns and database schema fixes
-- Version: 000020
-- Description: Removes added columns and tables

-- Drop trigger
DROP TRIGGER IF EXISTS update_optimization_history_updated_at ON optimization_history;

-- Drop optimization_history table
DROP TABLE IF EXISTS optimization_history;

-- Remove success_rate column from hedge_history table
ALTER TABLE hedge_history DROP COLUMN IF EXISTS success_rate;

-- Remove symbol column from strategy_performance table if it was added
-- Note: Only remove if it was added by this migration
-- ALTER TABLE strategy_performance DROP COLUMN IF EXISTS symbol;

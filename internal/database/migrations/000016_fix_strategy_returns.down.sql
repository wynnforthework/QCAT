-- Migration: Rollback strategy returns and performance tables
-- Version: 000016
-- Description: Removes strategy_returns table and added fields from strategy_performance

-- Drop indexes
DROP INDEX IF EXISTS idx_strategy_performance_daily_return;
DROP INDEX IF EXISTS idx_strategy_performance_date;
DROP INDEX IF EXISTS idx_strategy_returns_return_type;
DROP INDEX IF EXISTS idx_strategy_returns_created_at;
DROP INDEX IF EXISTS idx_strategy_returns_strategy_id;

-- Remove added columns from strategy_performance
ALTER TABLE strategy_performance DROP COLUMN IF EXISTS date;
ALTER TABLE strategy_performance DROP COLUMN IF EXISTS daily_return;

-- Drop strategy_returns table
DROP TABLE IF EXISTS strategy_returns;

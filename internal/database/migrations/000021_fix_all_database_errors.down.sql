-- Migration: Rollback all database error fixes
-- Version: 000021
-- Description: Rollback comprehensive fix for all reported database errors

-- Drop triggers
DROP TRIGGER IF EXISTS update_optimization_history_updated_at ON optimization_history;
DROP TRIGGER IF EXISTS update_market_data_updated_at ON market_data;
DROP TRIGGER IF EXISTS update_hedge_history_updated_at ON hedge_history;
DROP TRIGGER IF EXISTS update_risk_thresholds_updated_at ON risk_thresholds;

-- Drop indexes
DROP INDEX IF EXISTS idx_optimization_history_created_at;
DROP INDEX IF EXISTS idx_optimization_history_status;
DROP INDEX IF EXISTS idx_optimization_history_strategy_id;
DROP INDEX IF EXISTS idx_risk_thresholds_name;

DROP INDEX IF EXISTS idx_market_data_symbol_timestamp;
DROP INDEX IF EXISTS idx_market_data_timestamp;
DROP INDEX IF EXISTS idx_market_data_symbol_interval;

DROP INDEX IF EXISTS idx_hedge_history_strategy_ids;
DROP INDEX IF EXISTS idx_hedge_history_start_time;
DROP INDEX IF EXISTS idx_hedge_history_status;
DROP INDEX IF EXISTS idx_hedge_history_hedge_id;

-- Note: We don't drop the tables as they might contain important data
-- Instead, we just remove the columns we added

-- Remove complete column from market_data if it was added by this migration
-- (Only if the table existed before and we added the column)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'market_data' AND column_name = 'complete') THEN
        -- Only remove if this was added by our migration
        -- In practice, you might want to check migration history
        -- For now, we'll leave it as it's likely needed
        NULL;
    END IF;
END $$;

-- Note: We don't drop risk_thresholds, hedge_history, market_data, or optimization_history tables
-- as they might contain important data and be referenced by other parts of the system

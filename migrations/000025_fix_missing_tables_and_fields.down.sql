-- Migration 000025 rollback: Remove added tables and fields

BEGIN;

-- Drop trigger and function
DROP TRIGGER IF EXISTS update_optimization_results_updated_at ON optimization_results;
DROP FUNCTION IF EXISTS update_optimization_results_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_optimization_results_task_id;
DROP INDEX IF EXISTS idx_optimization_results_strategy_id;
DROP INDEX IF EXISTS idx_optimization_results_status;
DROP INDEX IF EXISTS idx_optimization_results_score;
DROP INDEX IF EXISTS idx_optimization_results_created_at;

DROP INDEX IF EXISTS idx_open_interest_symbol;
DROP INDEX IF EXISTS idx_open_interest_timestamp;
DROP INDEX IF EXISTS idx_open_interest_symbol_timestamp;

-- Remove optimization_score column if it was added by this migration
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns 
               WHERE table_name = 'optimization_results' AND column_name = 'optimization_score') THEN
        ALTER TABLE optimization_results DROP COLUMN optimization_score;
    END IF;
END $$;

-- Note: We don't drop the tables as they might contain important data
-- If you need to drop them completely, uncomment the following lines:
-- DROP TABLE IF EXISTS optimization_results;
-- DROP TABLE IF EXISTS open_interest;

COMMIT;

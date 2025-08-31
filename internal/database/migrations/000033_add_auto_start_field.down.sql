-- Migration rollback: Remove auto_start field from strategies table
-- Version: 000033
-- Description: Removes auto_start field and related columns

-- Drop indexes
DROP INDEX IF EXISTS idx_strategies_auto_start_priority;
DROP INDEX IF EXISTS idx_strategies_startup_priority;
DROP INDEX IF EXISTS idx_strategies_auto_start;

-- Remove columns
ALTER TABLE strategies DROP COLUMN IF EXISTS last_auto_start;
ALTER TABLE strategies DROP COLUMN IF EXISTS startup_priority;
ALTER TABLE strategies DROP COLUMN IF EXISTS auto_start;

-- Success message
DO $$ 
BEGIN
    RAISE NOTICE 'Migration 000033 rollback completed successfully';
    RAISE NOTICE 'Removed auto_start functionality from strategies table';
END $$;

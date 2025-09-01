-- Rollback migration for funding rate next_time fixes

BEGIN;

-- Drop the trigger and function
DROP TRIGGER IF EXISTS trigger_set_funding_next_time ON funding_rates;
DROP FUNCTION IF EXISTS set_funding_next_time();

-- Drop the index
DROP INDEX IF EXISTS idx_funding_rates_next_time;

-- Remove the check constraint (if it was added)
-- ALTER TABLE funding_rates DROP CONSTRAINT IF EXISTS check_next_time_future;

-- Remove default values (optional, as they don't hurt)
-- ALTER TABLE funding_rates ALTER COLUMN next_time DROP DEFAULT;
-- ALTER TABLE funding_rates ALTER COLUMN next_rate DROP DEFAULT;

COMMIT;

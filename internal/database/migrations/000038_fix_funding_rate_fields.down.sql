-- Rollback migration for funding rate field fixes

BEGIN;

-- Drop indexes that were added
DROP INDEX IF EXISTS idx_funding_rates_last_updated;
DROP INDEX IF EXISTS idx_funding_rates_funding_time;
DROP INDEX IF EXISTS idx_funding_rates_timestamp;
DROP INDEX IF EXISTS idx_funding_rates_symbol_funding_time;

-- Note: We don't remove the fields as they might be needed for compatibility
-- This is a conservative rollback that only removes the new indexes

COMMIT;

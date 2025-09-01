-- Rollback migration for abnormal_market_responses table

BEGIN;

-- Drop indexes
DROP INDEX IF EXISTS idx_abnormal_market_responses_active;
DROP INDEX IF EXISTS idx_abnormal_market_responses_symbol_detected_at;
DROP INDEX IF EXISTS idx_abnormal_market_responses_symbol_severity;
DROP INDEX IF EXISTS idx_abnormal_market_responses_execution_status;
DROP INDEX IF EXISTS idx_abnormal_market_responses_resolved;
DROP INDEX IF EXISTS idx_abnormal_market_responses_created_at;
DROP INDEX IF EXISTS idx_abnormal_market_responses_detected_at;
DROP INDEX IF EXISTS idx_abnormal_market_responses_severity;
DROP INDEX IF EXISTS idx_abnormal_market_responses_condition_type;
DROP INDEX IF EXISTS idx_abnormal_market_responses_symbol;

-- Drop the table
DROP TABLE IF EXISTS abnormal_market_responses;

COMMIT;

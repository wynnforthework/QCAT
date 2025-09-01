-- Rollback migration for fund_protection_history exchange field fixes

BEGIN;

-- Drop indexes
DROP INDEX IF EXISTS idx_fund_protection_history_created_at;
DROP INDEX IF EXISTS idx_fund_protection_history_protocol_type;
DROP INDEX IF EXISTS idx_fund_protection_history_triggered_at;
DROP INDEX IF EXISTS idx_fund_protection_history_risk_level;
DROP INDEX IF EXISTS idx_fund_protection_history_status;
DROP INDEX IF EXISTS idx_fund_protection_history_exchange;

-- Note: We don't remove the fields or restore NOT NULL constraints as they might be needed for compatibility
-- This is a conservative rollback that only removes the new indexes

COMMIT;

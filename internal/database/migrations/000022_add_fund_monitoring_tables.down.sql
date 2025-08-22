-- Migration: Rollback fund monitoring and protection tables
-- Version: 000022
-- Description: Drop fund_monitoring_rules and fund_protection_history tables

-- Drop triggers first
DROP TRIGGER IF EXISTS update_fund_monitoring_rules_updated_at ON fund_monitoring_rules;
DROP TRIGGER IF EXISTS update_fund_protection_history_updated_at ON fund_protection_history;

-- Drop indexes
DROP INDEX IF EXISTS idx_fund_monitoring_rules_exchange;
DROP INDEX IF EXISTS idx_fund_monitoring_rules_enabled;
DROP INDEX IF EXISTS idx_fund_monitoring_rules_type;
DROP INDEX IF EXISTS idx_fund_monitoring_rules_updated;

DROP INDEX IF EXISTS idx_fund_protection_history_exchange;
DROP INDEX IF EXISTS idx_fund_protection_history_status;
DROP INDEX IF EXISTS idx_fund_protection_history_risk_level;
DROP INDEX IF EXISTS idx_fund_protection_history_triggered;
DROP INDEX IF EXISTS idx_fund_protection_history_protocol;

-- Drop tables
DROP TABLE IF EXISTS fund_protection_history;
DROP TABLE IF EXISTS fund_monitoring_rules;

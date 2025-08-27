-- Migration 000027: Fix missing database fields (DOWN)
-- This migration removes the added fields and tables

BEGIN;

-- Remove indexes
DROP INDEX IF EXISTS idx_strategy_blacklist_active;
DROP INDEX IF EXISTS idx_strategy_blacklist_strategy_id;
DROP INDEX IF EXISTS idx_backtest_trades_entry_time;
DROP INDEX IF EXISTS idx_backtest_trades_symbol;
DROP INDEX IF EXISTS idx_backtest_trades_strategy_id;

-- Remove unique constraint
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fund_monitoring_rules_exchange_rule_type_unique' 
        AND table_name = 'fund_monitoring_rules'
    ) THEN
        ALTER TABLE fund_monitoring_rules 
        DROP CONSTRAINT fund_monitoring_rules_exchange_rule_type_unique;
        RAISE NOTICE 'Removed unique constraint from fund_monitoring_rules table';
    END IF;
END $$;

-- Drop tables
DROP TABLE IF EXISTS strategy_blacklist;
DROP TABLE IF EXISTS backtest_trades;

-- Remove fields from fund_protection_history table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'fund_protection_history' AND column_name = 'target_distribution') THEN
        ALTER TABLE fund_protection_history DROP COLUMN target_distribution;
        RAISE NOTICE 'Removed target_distribution column from fund_protection_history table';
    END IF;
END $$;

DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'fund_protection_history' AND column_name = 'risk_parameters') THEN
        ALTER TABLE fund_protection_history DROP COLUMN risk_parameters;
        RAISE NOTICE 'Removed risk_parameters column from fund_protection_history table';
    END IF;
END $$;

-- Remove fields from strategy_onboarding table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'symbols') THEN
        ALTER TABLE strategy_onboarding DROP COLUMN symbols;
        RAISE NOTICE 'Removed symbols column from strategy_onboarding table';
    END IF;
END $$;

DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'max_strategies') THEN
        ALTER TABLE strategy_onboarding DROP COLUMN max_strategies;
        RAISE NOTICE 'Removed max_strategies column from strategy_onboarding table';
    END IF;
END $$;

-- Remove fields from backtest_tasks table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'backtest_tasks' AND column_name = 'task_type') THEN
        ALTER TABLE backtest_tasks DROP COLUMN task_type;
        RAISE NOTICE 'Removed task_type column from backtest_tasks table';
    END IF;
END $$;

COMMIT;

-- Migration: Rollback missing tables and fields fix
-- Version: 000012
-- Description: Removes tables and fields created in up migration

-- Drop indexes
DROP INDEX IF EXISTS idx_onboarding_reports_time;
DROP INDEX IF EXISTS idx_onboarding_reports_status;
DROP INDEX IF EXISTS idx_onboarding_reports_strategy;

DROP INDEX IF EXISTS idx_strategy_performance_timestamp;
DROP INDEX IF EXISTS idx_strategy_performance_strategy;

DROP INDEX IF EXISTS idx_elimination_reports_time;

DROP INDEX IF EXISTS idx_strategy_positions_status;
DROP INDEX IF EXISTS idx_strategy_positions_symbol;
DROP INDEX IF EXISTS idx_strategy_positions_strategy;

DROP INDEX IF EXISTS idx_exchange_balances_updated_at;
DROP INDEX IF EXISTS idx_exchange_balances_asset;
DROP INDEX IF EXISTS idx_exchange_balances_exchange;

-- Drop tables
DROP TABLE IF EXISTS onboarding_reports;
DROP TABLE IF EXISTS strategy_performance;
DROP TABLE IF EXISTS elimination_reports;
DROP TABLE IF EXISTS strategy_positions;
DROP TABLE IF EXISTS exchange_balances;

-- Remove added columns (be careful with existing data)
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'symbols' AND column_name = 'updated_at') THEN
        ALTER TABLE symbols DROP COLUMN updated_at;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        NULL;
END $$;

DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'portfolio_allocations' AND column_name = 'symbol') THEN
        ALTER TABLE portfolio_allocations DROP COLUMN symbol;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        NULL;
END $$;

DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'positions' AND column_name = 'symbol') THEN
        ALTER TABLE positions DROP COLUMN symbol;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        NULL;
END $$;

DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'market_data' AND column_name = 'price') THEN
        ALTER TABLE market_data DROP COLUMN price;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        NULL;
END $$;

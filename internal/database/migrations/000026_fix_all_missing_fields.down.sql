-- Migration 000026 Down: Rollback all missing database fields and tables

BEGIN;

-- 6. Drop strategy_blacklist table
DROP TRIGGER IF EXISTS update_strategy_blacklist_updated_at ON strategy_blacklist;
DROP TABLE IF EXISTS strategy_blacklist;

-- 5. Remove target_distribution field from fund_protection_history table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'fund_protection_history' AND column_name = 'target_distribution') THEN
        ALTER TABLE fund_protection_history DROP COLUMN target_distribution;
        RAISE NOTICE 'Removed target_distribution column from fund_protection_history table';
    END IF;
END $$;

-- 4. Remove location and other fields from fund_monitoring_rules table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'fund_monitoring_rules' AND column_name = 'location') THEN
        ALTER TABLE fund_monitoring_rules DROP COLUMN location;
        RAISE NOTICE 'Removed location column from fund_monitoring_rules table';
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'fund_monitoring_rules' AND column_name = 'target_ratio') THEN
        ALTER TABLE fund_monitoring_rules DROP COLUMN target_ratio;
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'fund_monitoring_rules' AND column_name = 'check_interval') THEN
        ALTER TABLE fund_monitoring_rules DROP COLUMN check_interval;
    END IF;
END $$;

-- 3. Drop backtest_trades table
DROP TABLE IF EXISTS backtest_trades;

-- 2. Remove volume_24h field from tickers table (if we added it)
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'volume_24h') THEN
        ALTER TABLE tickers DROP COLUMN volume_24h;
        RAISE NOTICE 'Removed volume_24h column from tickers table';
    END IF;
END $$;

-- 1. Remove symbols field from strategy_onboarding table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'symbols') THEN
        ALTER TABLE strategy_onboarding DROP COLUMN symbols;
        RAISE NOTICE 'Removed symbols column from strategy_onboarding table';
    END IF;
END $$;

COMMIT;

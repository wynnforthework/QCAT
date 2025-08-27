-- Migration 000026: Fix all missing database fields and tables
-- This migration addresses all the database errors found in the log file

BEGIN;

-- 1. Add missing 'symbols' field to strategy_onboarding table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'strategy_onboarding') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'symbols') THEN
            ALTER TABLE strategy_onboarding ADD COLUMN symbols JSONB DEFAULT '[]';
            RAISE NOTICE 'Added symbols column to strategy_onboarding table';
        END IF;
    END IF;
END $$;

-- 2. Fix tickers table - ensure volume_24h field exists
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'tickers') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'volume_24h') THEN
            ALTER TABLE tickers ADD COLUMN volume_24h DECIMAL(30,10) DEFAULT 0;
            RAISE NOTICE 'Added volume_24h column to tickers table';
        END IF;
    ELSE
        -- Create tickers table if it doesn't exist
        CREATE TABLE tickers (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            symbol VARCHAR(20) NOT NULL,
            price DECIMAL(20,8) NOT NULL,
            volume DECIMAL(20,8) NOT NULL,
            volume_24h DECIMAL(30,10) DEFAULT 0,
            change_24h DECIMAL(10,4),
            high_24h DECIMAL(20,8),
            low_24h DECIMAL(20,8),
            timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            
            UNIQUE(symbol, timestamp)
        );
        
        CREATE INDEX IF NOT EXISTS idx_tickers_symbol ON tickers(symbol);
        CREATE INDEX IF NOT EXISTS idx_tickers_timestamp ON tickers(timestamp DESC);
        RAISE NOTICE 'Created tickers table with volume_24h column';
    END IF;
END $$;

-- 3. Create backtest_trades table if it doesn't exist
CREATE TABLE IF NOT EXISTS backtest_trades (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id VARCHAR(255) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    side VARCHAR(10) NOT NULL CHECK (side IN ('BUY', 'SELL', 'LONG', 'SHORT')),
    entry_price DECIMAL(20,8) NOT NULL,
    exit_price DECIMAL(20,8),
    quantity DECIMAL(20,8) NOT NULL,
    entry_time TIMESTAMP WITH TIME ZONE NOT NULL,
    exit_time TIMESTAMP WITH TIME ZONE,
    pnl DECIMAL(20,8) DEFAULT 0,
    fee DECIMAL(20,8) DEFAULT 0,
    commission DECIMAL(20,8) DEFAULT 0,
    trade_duration_seconds INTEGER,
    backtest_run_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for backtest_trades
CREATE INDEX IF NOT EXISTS idx_backtest_trades_strategy_id ON backtest_trades(strategy_id);
CREATE INDEX IF NOT EXISTS idx_backtest_trades_symbol ON backtest_trades(symbol);
CREATE INDEX IF NOT EXISTS idx_backtest_trades_entry_time ON backtest_trades(entry_time DESC);
CREATE INDEX IF NOT EXISTS idx_backtest_trades_exit_time ON backtest_trades(exit_time DESC);
CREATE INDEX IF NOT EXISTS idx_backtest_trades_backtest_run_id ON backtest_trades(backtest_run_id);

-- 4. Add missing 'location' field to fund_monitoring_rules table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'fund_monitoring_rules') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'fund_monitoring_rules' AND column_name = 'location') THEN
            ALTER TABLE fund_monitoring_rules ADD COLUMN location VARCHAR(50);
            RAISE NOTICE 'Added location column to fund_monitoring_rules table';
        END IF;
        
        -- Add other missing fields that might be needed
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'fund_monitoring_rules' AND column_name = 'target_ratio') THEN
            ALTER TABLE fund_monitoring_rules ADD COLUMN target_ratio DECIMAL(5,4) DEFAULT 0;
        END IF;
        
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'fund_monitoring_rules' AND column_name = 'check_interval') THEN
            ALTER TABLE fund_monitoring_rules ADD COLUMN check_interval INTEGER DEFAULT 300;
        END IF;
    END IF;
END $$;

-- 5. Add missing 'target_distribution' field to fund_protection_history table
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'fund_protection_history') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'fund_protection_history' AND column_name = 'target_distribution') THEN
            ALTER TABLE fund_protection_history ADD COLUMN target_distribution JSONB DEFAULT '{}';
            RAISE NOTICE 'Added target_distribution column to fund_protection_history table';
        END IF;
    END IF;
END $$;

-- 6. Create strategy_blacklist table if it doesn't exist
CREATE TABLE IF NOT EXISTS strategy_blacklist (
    id SERIAL PRIMARY KEY,
    strategy_id VARCHAR(255) NOT NULL UNIQUE,
    reason TEXT NOT NULL,
    blocked_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    blocked_by VARCHAR(100) NOT NULL DEFAULT 'system',
    permanent BOOLEAN NOT NULL DEFAULT true,
    expires_at TIMESTAMP WITH TIME ZONE NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for strategy_blacklist
CREATE INDEX IF NOT EXISTS idx_strategy_blacklist_strategy_id ON strategy_blacklist(strategy_id);
CREATE INDEX IF NOT EXISTS idx_strategy_blacklist_blocked_at ON strategy_blacklist(blocked_at);
CREATE INDEX IF NOT EXISTS idx_strategy_blacklist_expires_at ON strategy_blacklist(expires_at);
CREATE INDEX IF NOT EXISTS idx_strategy_blacklist_permanent ON strategy_blacklist(permanent);

-- Add comments for strategy_blacklist
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'strategy_blacklist') THEN
        EXECUTE 'COMMENT ON TABLE strategy_blacklist IS ''策略黑名单表，用于记录被禁用的策略''';
        EXECUTE 'COMMENT ON COLUMN strategy_blacklist.strategy_id IS ''策略ID''';
        EXECUTE 'COMMENT ON COLUMN strategy_blacklist.reason IS ''禁用原因''';
        EXECUTE 'COMMENT ON COLUMN strategy_blacklist.blocked_at IS ''禁用时间''';
        EXECUTE 'COMMENT ON COLUMN strategy_blacklist.blocked_by IS ''禁用者（系统/用户）''';
        EXECUTE 'COMMENT ON COLUMN strategy_blacklist.permanent IS ''是否永久禁用''';
        EXECUTE 'COMMENT ON COLUMN strategy_blacklist.expires_at IS ''过期时间（仅当permanent=false时有效）''';
        RAISE NOTICE 'Created strategy_blacklist table with indexes and comments';
    END IF;
END $$;

-- Create or replace trigger function for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger for strategy_blacklist
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'strategy_blacklist') THEN
        DROP TRIGGER IF EXISTS update_strategy_blacklist_updated_at ON strategy_blacklist;
        CREATE TRIGGER update_strategy_blacklist_updated_at
            BEFORE UPDATE ON strategy_blacklist
            FOR EACH ROW
            EXECUTE FUNCTION update_updated_at_column();
        RAISE NOTICE 'Created trigger for strategy_blacklist table';
    END IF;
END $$;

COMMIT;

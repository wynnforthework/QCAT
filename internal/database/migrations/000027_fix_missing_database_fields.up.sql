-- Migration 000027: Fix missing database fields
-- This migration addresses all the database errors found in the error messages

BEGIN;

-- 1. Add missing 'task_type' field to backtest_tasks table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'backtest_tasks') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'backtest_tasks' AND column_name = 'task_type') THEN
            ALTER TABLE backtest_tasks ADD COLUMN task_type VARCHAR(50) DEFAULT 'periodic';
            RAISE NOTICE 'Added task_type column to backtest_tasks table';
        END IF;
    END IF;
END $$;

-- 2. Add missing 'max_strategies' field to strategy_onboarding table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'strategy_onboarding') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'max_strategies') THEN
            ALTER TABLE strategy_onboarding ADD COLUMN max_strategies INTEGER DEFAULT 5;
            RAISE NOTICE 'Added max_strategies column to strategy_onboarding table';
        END IF;
    END IF;
END $$;

-- 3. Add missing 'symbols' field to strategy_onboarding table (if not already added)
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'strategy_onboarding') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'symbols') THEN
            ALTER TABLE strategy_onboarding ADD COLUMN symbols JSONB DEFAULT '[]';
            RAISE NOTICE 'Added symbols column to strategy_onboarding table';
        END IF;
    END IF;
END $$;

-- 4. Add missing 'risk_parameters' field to fund_protection_history table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'fund_protection_history') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'fund_protection_history' AND column_name = 'risk_parameters') THEN
            ALTER TABLE fund_protection_history ADD COLUMN risk_parameters JSONB DEFAULT '{}';
            RAISE NOTICE 'Added risk_parameters column to fund_protection_history table';
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

-- 6. Create missing backtest_trades table if it doesn't exist
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

-- 7. Add missing fields to strategy_blacklist table
DO $$
BEGIN
    -- Add is_active field if it doesn't exist
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'strategy_blacklist') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_blacklist' AND column_name = 'is_active') THEN
            ALTER TABLE strategy_blacklist ADD COLUMN is_active BOOLEAN DEFAULT true;
            RAISE NOTICE 'Added is_active column to strategy_blacklist table';
        END IF;
    ELSE
        -- Create table if it doesn't exist
        CREATE TABLE strategy_blacklist (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            strategy_id VARCHAR(255) NOT NULL,
            reason VARCHAR(500) NOT NULL,
            blacklisted_by VARCHAR(100),
            blacklisted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            expires_at TIMESTAMP WITH TIME ZONE,
            is_active BOOLEAN DEFAULT true,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );
        RAISE NOTICE 'Created strategy_blacklist table';
    END IF;
END $$;

-- 8. Add unique constraint for fund_monitoring_rules to fix ON CONFLICT issues
DO $$
BEGIN
    -- First check if the constraint already exists
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fund_monitoring_rules_exchange_rule_type_unique' 
        AND table_name = 'fund_monitoring_rules'
    ) THEN
        -- Add unique constraint
        ALTER TABLE fund_monitoring_rules 
        ADD CONSTRAINT fund_monitoring_rules_exchange_rule_type_unique 
        UNIQUE (exchange, rule_type, rule_name);
        RAISE NOTICE 'Added unique constraint to fund_monitoring_rules table';
    END IF;
EXCEPTION
    WHEN duplicate_table THEN
        RAISE NOTICE 'Unique constraint already exists on fund_monitoring_rules';
END $$;

-- 9. Add indexes for performance
CREATE INDEX IF NOT EXISTS idx_backtest_trades_strategy_id ON backtest_trades(strategy_id);
CREATE INDEX IF NOT EXISTS idx_backtest_trades_symbol ON backtest_trades(symbol);
CREATE INDEX IF NOT EXISTS idx_backtest_trades_entry_time ON backtest_trades(entry_time);

CREATE INDEX IF NOT EXISTS idx_strategy_blacklist_strategy_id ON strategy_blacklist(strategy_id);
CREATE INDEX IF NOT EXISTS idx_strategy_blacklist_active ON strategy_blacklist(is_active);

COMMIT;

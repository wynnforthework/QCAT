-- Migration: Fix all database errors
-- Version: 000021
-- Description: Comprehensive fix for all reported database errors

-- 1. Create risk_thresholds table if it doesn't exist
CREATE TABLE IF NOT EXISTS risk_thresholds (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    max_margin_ratio DECIMAL(5,4) NOT NULL DEFAULT 0.8000,
    warning_margin_ratio DECIMAL(5,4) NOT NULL DEFAULT 0.7000,
    max_daily_loss DECIMAL(20,8) NOT NULL DEFAULT 5000.00000000,
    max_total_loss DECIMAL(20,8) NOT NULL DEFAULT 10000.00000000,
    max_drawdown_percent DECIMAL(5,4) NOT NULL DEFAULT 0.2000,
    max_position_loss DECIMAL(20,8) NOT NULL DEFAULT 1000.00000000,
    max_position_loss_percent DECIMAL(5,4) NOT NULL DEFAULT 0.1000,
    min_account_balance DECIMAL(20,8) NOT NULL DEFAULT 10000.00000000,
    max_leverage INTEGER NOT NULL DEFAULT 10,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert default risk thresholds if not exists
INSERT INTO risk_thresholds (
    name, max_margin_ratio, warning_margin_ratio, max_daily_loss, 
    max_total_loss, max_drawdown_percent, max_position_loss, 
    max_position_loss_percent, min_account_balance, max_leverage
) VALUES (
    'default',
    0.8000,  -- 80% max margin ratio
    0.7000,  -- 70% warning margin ratio
    5000.00000000,  -- $5000 max daily loss
    10000.00000000, -- $10000 max total loss
    0.2000,  -- 20% max drawdown
    1000.00000000,  -- $1000 max position loss
    0.1000,  -- 10% max position loss percentage
    10000.00000000, -- $10000 min account balance
    10       -- 10x max leverage
) ON CONFLICT (name) DO NOTHING;

-- 2. Fix hedge_history table strategy_ids constraint
-- First, check if hedge_history table exists and fix it
DO $$
BEGIN
    -- Check if hedge_history table exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'hedge_history') THEN
        -- Make strategy_ids nullable temporarily to fix existing data
        ALTER TABLE hedge_history ALTER COLUMN strategy_ids DROP NOT NULL;
        
        -- Update any NULL strategy_ids with empty array
        UPDATE hedge_history SET strategy_ids = '{}' WHERE strategy_ids IS NULL;
        
        -- Add back NOT NULL constraint
        ALTER TABLE hedge_history ALTER COLUMN strategy_ids SET NOT NULL;
    ELSE
        -- Create hedge_history table if it doesn't exist
        CREATE TABLE hedge_history (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            hedge_id VARCHAR(100) NOT NULL,
            strategy_ids UUID[] NOT NULL DEFAULT '{}', -- Array of strategy IDs involved in hedge
            hedge_type VARCHAR(50) NOT NULL, -- 'cross_strategy', 'market_neutral', 'pairs_trading'
            total_exposure DECIMAL(30,10) NOT NULL DEFAULT 0,
            net_exposure DECIMAL(30,10) NOT NULL DEFAULT 0,
            hedge_ratio DECIMAL(10,6) NOT NULL DEFAULT 0,
            pnl DECIMAL(30,10) NOT NULL DEFAULT 0,
            status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'active', 'closed', 'failed'
            start_time TIMESTAMP WITH TIME ZONE NOT NULL,
            end_time TIMESTAMP WITH TIME ZONE,
            metadata JSONB DEFAULT '{}',
            correlation_matrix JSONB DEFAULT '{}',
            success_rate DECIMAL(5,4) DEFAULT 0.0000,
            total_operations INTEGER DEFAULT 0,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        );
        
        -- Create indexes
        CREATE INDEX IF NOT EXISTS idx_hedge_history_hedge_id ON hedge_history(hedge_id);
        CREATE INDEX IF NOT EXISTS idx_hedge_history_status ON hedge_history(status);
        CREATE INDEX IF NOT EXISTS idx_hedge_history_start_time ON hedge_history(start_time DESC);
        CREATE INDEX IF NOT EXISTS idx_hedge_history_strategy_ids ON hedge_history USING GIN(strategy_ids);
    END IF;
END $$;

-- 3. Fix market_data table to ensure complete column exists
DO $$
BEGIN
    -- Check if market_data table exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'market_data') THEN
        -- Add complete column if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'market_data' AND column_name = 'complete') THEN
            ALTER TABLE market_data ADD COLUMN complete BOOLEAN NOT NULL DEFAULT false;
        END IF;
        
        -- Ensure updated_at column exists
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'market_data' AND column_name = 'updated_at') THEN
            ALTER TABLE market_data ADD COLUMN updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();
        END IF;
    ELSE
        -- Create market_data table with proper schema
        CREATE TABLE market_data (
            id BIGSERIAL PRIMARY KEY,
            symbol VARCHAR(20) NOT NULL,
            interval VARCHAR(10) NOT NULL,
            timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
            open DECIMAL(20,8) NOT NULL,
            high DECIMAL(20,8) NOT NULL,
            low DECIMAL(20,8) NOT NULL,
            close DECIMAL(20,8) NOT NULL,
            volume DECIMAL(20,8) NOT NULL DEFAULT 0,
            complete BOOLEAN NOT NULL DEFAULT false,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            
            UNIQUE(symbol, interval, timestamp)
        );
        
        -- Create indexes for performance
        CREATE INDEX IF NOT EXISTS idx_market_data_symbol_interval ON market_data(symbol, interval);
        CREATE INDEX IF NOT EXISTS idx_market_data_timestamp ON market_data(timestamp DESC);
        CREATE INDEX IF NOT EXISTS idx_market_data_symbol_timestamp ON market_data(symbol, timestamp DESC);
    END IF;
END $$;

-- 4. Create missing optimization_history table if needed
CREATE TABLE IF NOT EXISTS optimization_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    optimization_id VARCHAR(100) NOT NULL UNIQUE,
    optimization_type VARCHAR(50) NOT NULL,
    strategy_id UUID,
    parameters_before JSONB DEFAULT '{}',
    parameters_after JSONB DEFAULT '{}',
    performance_before JSONB DEFAULT '{}',
    performance_after JSONB DEFAULT '{}',
    improvement_score DECIMAL(10,6) DEFAULT 0,
    objective_value DECIMAL(10,6) DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 5. Create updated_at trigger function if it doesn't exist
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 6. Create triggers for updated_at columns (ignore errors if they already exist)
DO $$
BEGIN
    CREATE TRIGGER update_risk_thresholds_updated_at
        BEFORE UPDATE ON risk_thresholds
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TRIGGER update_hedge_history_updated_at
        BEFORE UPDATE ON hedge_history
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TRIGGER update_market_data_updated_at
        BEFORE UPDATE ON market_data
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TRIGGER update_optimization_history_updated_at
        BEFORE UPDATE ON optimization_history
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- 7. Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_risk_thresholds_name ON risk_thresholds(name);
CREATE INDEX IF NOT EXISTS idx_optimization_history_strategy_id ON optimization_history(strategy_id);
CREATE INDEX IF NOT EXISTS idx_optimization_history_status ON optimization_history(status);
CREATE INDEX IF NOT EXISTS idx_optimization_history_created_at ON optimization_history(created_at DESC);

-- 8. Verify all tables exist and show status
DO $$
BEGIN
    RAISE NOTICE 'Migration 000021 completed successfully';

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'risk_thresholds') THEN
        RAISE NOTICE 'risk_thresholds table: EXISTS';
    ELSE
        RAISE NOTICE 'risk_thresholds table: MISSING';
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'hedge_history') THEN
        RAISE NOTICE 'hedge_history table: EXISTS';
    ELSE
        RAISE NOTICE 'hedge_history table: MISSING';
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'market_data') THEN
        RAISE NOTICE 'market_data table: EXISTS';
    ELSE
        RAISE NOTICE 'market_data table: MISSING';
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'market_data' AND column_name = 'complete') THEN
        RAISE NOTICE 'market_data.complete column: EXISTS';
    ELSE
        RAISE NOTICE 'market_data.complete column: MISSING';
    END IF;
END $$;

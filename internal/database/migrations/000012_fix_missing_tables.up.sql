-- Migration: Fix missing tables and fields
-- Version: 000012
-- Description: Creates missing tables and fixes field issues

-- Create exchange_balances table
CREATE TABLE IF NOT EXISTS exchange_balances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    exchange_name VARCHAR(50) NOT NULL,
    asset VARCHAR(20) NOT NULL,
    balance DECIMAL(30,10) NOT NULL DEFAULT 0,
    available DECIMAL(30,10) NOT NULL DEFAULT 0,
    locked DECIMAL(30,10) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(exchange_name, asset)
);

-- Create strategy_positions table
CREATE TABLE IF NOT EXISTS strategy_positions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id VARCHAR(100) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    position_size DECIMAL(30,10) NOT NULL DEFAULT 0,
    entry_price DECIMAL(30,10),
    current_price DECIMAL(30,10),
    unrealized_pnl DECIMAL(30,10) DEFAULT 0,
    realized_pnl DECIMAL(30,10) DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    side VARCHAR(10) NOT NULL, -- 'LONG' or 'SHORT'
    leverage INTEGER DEFAULT 1,
    margin_used DECIMAL(30,10) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(strategy_id, symbol)
);

-- Create elimination_reports table
CREATE TABLE IF NOT EXISTS elimination_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    report_time TIMESTAMP WITH TIME ZONE NOT NULL,
    total_strategies INTEGER NOT NULL DEFAULT 0,
    active_strategies INTEGER NOT NULL DEFAULT 0,
    disabled_strategies INTEGER NOT NULL DEFAULT 0,
    eliminated_strategies INTEGER NOT NULL DEFAULT 0,
    cooldown_pool_size INTEGER NOT NULL DEFAULT 0,
    report_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create strategy_performance table
CREATE TABLE IF NOT EXISTS strategy_performance (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id VARCHAR(100) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    total_pnl DECIMAL(30,10) NOT NULL DEFAULT 0,
    daily_pnl DECIMAL(30,10) NOT NULL DEFAULT 0,
    win_rate DECIMAL(5,4) DEFAULT 0,
    sharpe_ratio DECIMAL(10,4) DEFAULT 0,
    max_drawdown DECIMAL(5,4) DEFAULT 0,
    total_trades INTEGER DEFAULT 0,
    winning_trades INTEGER DEFAULT 0,
    losing_trades INTEGER DEFAULT 0,
    avg_win DECIMAL(30,10) DEFAULT 0,
    avg_loss DECIMAL(30,10) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(strategy_id, timestamp)
);

-- Create onboarding_reports table
CREATE TABLE IF NOT EXISTS onboarding_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id VARCHAR(100) NOT NULL,
    report_time TIMESTAMP WITH TIME ZONE NOT NULL,
    onboarding_status VARCHAR(50) NOT NULL, -- 'pending', 'testing', 'approved', 'rejected'
    test_results JSONB,
    performance_metrics JSONB,
    risk_assessment JSONB,
    approval_notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Fix market_data table - add missing price field if not exists
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'market_data' AND column_name = 'price') THEN
        ALTER TABLE market_data ADD COLUMN price DECIMAL(30,10);
        -- Update price column with close price for existing records
        UPDATE market_data SET price = close WHERE price IS NULL;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        NULL;
END $$;

-- Fix positions table - add missing symbol field if not exists
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'positions' AND column_name = 'symbol') THEN
        ALTER TABLE positions ADD COLUMN symbol VARCHAR(20);
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        NULL;
END $$;

-- Fix portfolio_allocations table - add missing symbol field if not exists
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'portfolio_allocations' AND column_name = 'symbol') THEN
        ALTER TABLE portfolio_allocations ADD COLUMN symbol VARCHAR(20);
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        NULL;
END $$;

-- Fix symbols table - add missing updated_at field if not exists
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'symbols' AND column_name = 'updated_at') THEN
        ALTER TABLE symbols ADD COLUMN updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        NULL;
END $$;

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_exchange_balances_exchange ON exchange_balances(exchange_name);
CREATE INDEX IF NOT EXISTS idx_exchange_balances_asset ON exchange_balances(asset);
CREATE INDEX IF NOT EXISTS idx_exchange_balances_updated_at ON exchange_balances(updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_strategy_positions_strategy ON strategy_positions(strategy_id);
CREATE INDEX IF NOT EXISTS idx_strategy_positions_symbol ON strategy_positions(symbol);
CREATE INDEX IF NOT EXISTS idx_strategy_positions_status ON strategy_positions(status);

CREATE INDEX IF NOT EXISTS idx_elimination_reports_time ON elimination_reports(report_time DESC);

CREATE INDEX IF NOT EXISTS idx_strategy_performance_strategy ON strategy_performance(strategy_id);
CREATE INDEX IF NOT EXISTS idx_strategy_performance_timestamp ON strategy_performance(timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_onboarding_reports_strategy ON onboarding_reports(strategy_id);
CREATE INDEX IF NOT EXISTS idx_onboarding_reports_status ON onboarding_reports(onboarding_status);
CREATE INDEX IF NOT EXISTS idx_onboarding_reports_time ON onboarding_reports(report_time DESC);

-- Insert sample data for testing
INSERT INTO exchange_balances (exchange_name, asset, balance, available, locked)
SELECT 'binance', 'USDT', 10000.0, 9500.0, 500.0
WHERE NOT EXISTS (SELECT 1 FROM exchange_balances WHERE exchange_name = 'binance' AND asset = 'USDT');

INSERT INTO exchange_balances (exchange_name, asset, balance, available, locked)
SELECT 'binance', 'BTC', 0.5, 0.4, 0.1
WHERE NOT EXISTS (SELECT 1 FROM exchange_balances WHERE exchange_name = 'binance' AND asset = 'BTC');

INSERT INTO strategy_performance (strategy_id, timestamp, total_pnl, daily_pnl, win_rate, sharpe_ratio)
SELECT 'strategy_001', CURRENT_TIMESTAMP, 1500.0, 150.0, 0.65, 1.8
WHERE NOT EXISTS (SELECT 1 FROM strategy_performance WHERE strategy_id = 'strategy_001');

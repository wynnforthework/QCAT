-- Migration: Add strategy_eliminations table
-- Version: 000014
-- Description: Creates strategy_eliminations table for tracking eliminated and disabled strategies

-- Create strategy_eliminations table
CREATE TABLE IF NOT EXISTS strategy_eliminations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id UUID NOT NULL REFERENCES strategies(id),
    reason VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'eliminated', -- 'eliminated', 'disabled', 'reactivated'
    eliminated_at TIMESTAMP WITH TIME ZONE,
    disabled_until TIMESTAMP WITH TIME ZONE, -- NULL for permanent elimination
    reactivated_at TIMESTAMP WITH TIME ZONE,
    performance_data JSONB, -- Store performance metrics at time of elimination
    metadata JSONB DEFAULT '{}', -- Additional metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(strategy_id) -- One elimination record per strategy
);

-- Create hedge_history table (also missing)
CREATE TABLE IF NOT EXISTS hedge_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    hedge_id VARCHAR(100) NOT NULL,
    strategy_ids UUID[] NOT NULL, -- Array of strategy IDs involved in hedge
    hedge_type VARCHAR(50) NOT NULL, -- 'cross_strategy', 'market_neutral', 'pairs_trading'
    total_exposure DECIMAL(30,10) NOT NULL DEFAULT 0,
    net_exposure DECIMAL(30,10) NOT NULL DEFAULT 0,
    hedge_ratio DECIMAL(10,6) NOT NULL DEFAULT 0,
    pnl DECIMAL(30,10) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'active', 'closed', 'failed'
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create wallet_balances table (also missing)
CREATE TABLE IF NOT EXISTS wallet_balances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    wallet_id VARCHAR(100) NOT NULL,
    exchange_name VARCHAR(50) NOT NULL,
    asset VARCHAR(20) NOT NULL,
    total_balance DECIMAL(30,10) NOT NULL DEFAULT 0,
    available_balance DECIMAL(30,10) NOT NULL DEFAULT 0,
    locked_balance DECIMAL(30,10) NOT NULL DEFAULT 0,
    cross_margin DECIMAL(30,10) NOT NULL DEFAULT 0,
    isolated_margin DECIMAL(30,10) NOT NULL DEFAULT 0,
    unrealized_pnl DECIMAL(30,10) NOT NULL DEFAULT 0,
    realized_pnl DECIMAL(30,10) NOT NULL DEFAULT 0,
    last_updated TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(wallet_id, exchange_name, asset)
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_strategy_eliminations_strategy_id ON strategy_eliminations(strategy_id);
CREATE INDEX IF NOT EXISTS idx_strategy_eliminations_status ON strategy_eliminations(status);
CREATE INDEX IF NOT EXISTS idx_strategy_eliminations_eliminated_at ON strategy_eliminations(eliminated_at DESC);
CREATE INDEX IF NOT EXISTS idx_strategy_eliminations_disabled_until ON strategy_eliminations(disabled_until);

CREATE INDEX IF NOT EXISTS idx_hedge_history_hedge_id ON hedge_history(hedge_id);
CREATE INDEX IF NOT EXISTS idx_hedge_history_status ON hedge_history(status);
CREATE INDEX IF NOT EXISTS idx_hedge_history_start_time ON hedge_history(start_time DESC);
CREATE INDEX IF NOT EXISTS idx_hedge_history_strategy_ids ON hedge_history USING GIN(strategy_ids);

CREATE INDEX IF NOT EXISTS idx_wallet_balances_wallet_id ON wallet_balances(wallet_id);
CREATE INDEX IF NOT EXISTS idx_wallet_balances_exchange ON wallet_balances(exchange_name);
CREATE INDEX IF NOT EXISTS idx_wallet_balances_asset ON wallet_balances(asset);
CREATE INDEX IF NOT EXISTS idx_wallet_balances_updated_at ON wallet_balances(updated_at DESC);

-- Add comments for documentation
COMMENT ON TABLE strategy_eliminations IS 'Tracks eliminated and disabled strategies with their reasons and timing';
COMMENT ON TABLE hedge_history IS 'Records history of hedge operations across multiple strategies';
COMMENT ON TABLE wallet_balances IS 'Stores wallet balance information across different exchanges';

COMMENT ON COLUMN strategy_eliminations.status IS 'Status: eliminated (permanent), disabled (temporary), reactivated';
COMMENT ON COLUMN strategy_eliminations.disabled_until IS 'NULL for permanent elimination, timestamp for temporary disable';
COMMENT ON COLUMN hedge_history.strategy_ids IS 'Array of strategy UUIDs involved in the hedge operation';
COMMENT ON COLUMN wallet_balances.wallet_id IS 'Unique identifier for the wallet (can be user_id or account_id)';

-- Migration: Fix database errors
-- Version: 000017
-- Description: Fixes all database-related errors including missing tables, fields, and SQL syntax issues

-- 1. Add missing symbol field to strategies table
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS symbol VARCHAR(20);

-- 2. Create missing performance_forecasts table
CREATE TABLE IF NOT EXISTS performance_forecasts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id UUID REFERENCES strategies(id),
    expected_return_1d DECIMAL(10,6) NOT NULL DEFAULT 0,
    expected_return_7d DECIMAL(10,6) NOT NULL DEFAULT 0,
    expected_return_30d DECIMAL(10,6) NOT NULL DEFAULT 0,
    volatility DECIMAL(10,6) NOT NULL DEFAULT 0,
    max_drawdown DECIMAL(10,6) NOT NULL DEFAULT 0,
    var_95 DECIMAL(10,6) NOT NULL DEFAULT 0,
    confidence DECIMAL(5,4) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. Create missing profit_optimization_history table
CREATE TABLE IF NOT EXISTS profit_optimization_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id UUID REFERENCES strategies(id),
    optimization_id VARCHAR(100) NOT NULL,
    optimization_type VARCHAR(50) NOT NULL, -- 'parameter_tuning', 'risk_adjustment', 'position_sizing'
    parameters_before JSONB,
    parameters_after JSONB,
    performance_before JSONB,
    performance_after JSONB,
    improvement_score DECIMAL(10,6) DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'running', 'completed', 'failed'
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 4. Create missing account_balances table
CREATE TABLE IF NOT EXISTS account_balances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    exchange_name VARCHAR(50) NOT NULL,
    asset VARCHAR(20) NOT NULL,
    total DECIMAL(30,10) NOT NULL DEFAULT 0,
    available DECIMAL(30,10) NOT NULL DEFAULT 0,
    locked DECIMAL(30,10) NOT NULL DEFAULT 0,
    cross_margin DECIMAL(30,10) NOT NULL DEFAULT 0,
    isolated_margin DECIMAL(30,10) NOT NULL DEFAULT 0,
    unrealized_pnl DECIMAL(30,10) NOT NULL DEFAULT 0,
    realized_pnl DECIMAL(30,10) NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(exchange_name, asset)
);

-- 5. Add missing correlation_matrix field to hedge_history table
ALTER TABLE hedge_history ADD COLUMN IF NOT EXISTS correlation_matrix JSONB DEFAULT '{}';

-- 6. Add missing wallet_type field to wallet_balances table
ALTER TABLE wallet_balances ADD COLUMN IF NOT EXISTS wallet_type VARCHAR(50) DEFAULT 'spot';

-- 7. Add missing request_id field to onboarding_reports table
ALTER TABLE onboarding_reports ADD COLUMN IF NOT EXISTS request_id VARCHAR(100);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_performance_forecasts_strategy_id ON performance_forecasts(strategy_id);
CREATE INDEX IF NOT EXISTS idx_performance_forecasts_created_at ON performance_forecasts(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_profit_optimization_history_strategy_id ON profit_optimization_history(strategy_id);
CREATE INDEX IF NOT EXISTS idx_profit_optimization_history_optimization_id ON profit_optimization_history(optimization_id);
CREATE INDEX IF NOT EXISTS idx_profit_optimization_history_status ON profit_optimization_history(status);
CREATE INDEX IF NOT EXISTS idx_profit_optimization_history_started_at ON profit_optimization_history(started_at DESC);

CREATE INDEX IF NOT EXISTS idx_account_balances_exchange_asset ON account_balances(exchange_name, asset);
CREATE INDEX IF NOT EXISTS idx_account_balances_updated_at ON account_balances(updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_strategies_symbol ON strategies(symbol);
CREATE INDEX IF NOT EXISTS idx_onboarding_reports_request_id ON onboarding_reports(request_id);

-- Add comments for documentation
COMMENT ON COLUMN strategies.symbol IS 'Trading symbol associated with the strategy (e.g., BTCUSDT)';
COMMENT ON TABLE performance_forecasts IS 'Stores performance forecasts for strategies';
COMMENT ON TABLE profit_optimization_history IS 'Records history of profit optimization operations';
COMMENT ON TABLE account_balances IS 'Stores account balance information across exchanges';
COMMENT ON COLUMN hedge_history.correlation_matrix IS 'Correlation matrix data for hedge operations';
COMMENT ON COLUMN wallet_balances.wallet_type IS 'Type of wallet: spot, margin, futures, etc.';
COMMENT ON COLUMN onboarding_reports.request_id IS 'Unique identifier for onboarding requests';

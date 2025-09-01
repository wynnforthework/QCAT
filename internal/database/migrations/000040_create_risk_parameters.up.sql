-- Migration to create risk_parameters table
-- This fixes the "risk_parameters" relation does not exist error

BEGIN;

-- Create risk_parameters table if it doesn't exist
CREATE TABLE IF NOT EXISTS risk_parameters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol VARCHAR(20) NOT NULL,
    strategy_id UUID, -- Optional reference to strategies table
    
    -- Leverage parameters
    max_leverage DECIMAL(10,2) NOT NULL DEFAULT 10.0,
    current_leverage DECIMAL(10,2) DEFAULT 1.0,
    leverage_step DECIMAL(10,2) DEFAULT 0.5, -- Step size for leverage adjustments
    
    -- Position size parameters
    max_position_size DECIMAL(20,8) NOT NULL DEFAULT 1000000.0,
    max_position_value DECIMAL(20,8) DEFAULT 100000.0,
    position_size_limit_pct DECIMAL(5,4) DEFAULT 0.1, -- 10% of portfolio
    
    -- Risk thresholds
    max_drawdown DECIMAL(5,4) NOT NULL DEFAULT 0.2, -- 20% max drawdown
    stop_loss_pct DECIMAL(5,4) DEFAULT 0.05, -- 5% stop loss
    take_profit_pct DECIMAL(5,4) DEFAULT 0.10, -- 10% take profit
    trailing_stop_pct DECIMAL(5,4) DEFAULT 0.03, -- 3% trailing stop
    
    -- Circuit breaker parameters
    circuit_breaker_threshold DECIMAL(5,4) NOT NULL DEFAULT 0.15, -- 15% loss triggers circuit breaker
    circuit_breaker_cooldown INTEGER DEFAULT 3600, -- 1 hour cooldown in seconds
    volatility_threshold DECIMAL(5,4) DEFAULT 0.20, -- 20% volatility threshold
    
    -- Margin parameters
    initial_margin_rate DECIMAL(5,4) DEFAULT 0.10, -- 10% initial margin
    maintenance_margin_rate DECIMAL(5,4) DEFAULT 0.05, -- 5% maintenance margin
    margin_call_threshold DECIMAL(5,4) DEFAULT 0.80, -- 80% margin usage triggers warning
    
    -- Order parameters
    max_order_value DECIMAL(20,8) DEFAULT 50000.0,
    min_order_value DECIMAL(20,8) DEFAULT 10.0,
    max_order_qty DECIMAL(20,8) DEFAULT 1000.0,
    min_order_qty DECIMAL(20,8) DEFAULT 0.001,
    
    -- Risk scoring parameters
    risk_score_threshold DECIMAL(5,2) DEFAULT 75.0, -- Risk score 0-100
    var_confidence_level DECIMAL(5,4) DEFAULT 0.95, -- 95% VaR confidence
    expected_shortfall_threshold DECIMAL(5,4) DEFAULT 0.10, -- 10% ES threshold
    
    -- Time-based parameters
    max_holding_period INTEGER DEFAULT 86400, -- 24 hours in seconds
    rebalance_frequency INTEGER DEFAULT 3600, -- 1 hour in seconds
    risk_check_interval INTEGER DEFAULT 300, -- 5 minutes in seconds
    
    -- Market condition parameters
    high_volatility_multiplier DECIMAL(5,4) DEFAULT 0.5, -- Reduce limits by 50% in high vol
    low_liquidity_multiplier DECIMAL(5,4) DEFAULT 0.7, -- Reduce limits by 30% in low liquidity
    stress_test_multiplier DECIMAL(5,4) DEFAULT 0.3, -- Reduce limits by 70% in stress
    
    -- Status and control
    is_active BOOLEAN DEFAULT TRUE,
    auto_adjust BOOLEAN DEFAULT TRUE, -- Allow automatic parameter adjustments
    override_enabled BOOLEAN DEFAULT FALSE, -- Allow manual overrides
    last_adjustment TIMESTAMP WITH TIME ZONE,
    
    -- Metadata
    risk_profile VARCHAR(20) DEFAULT 'MEDIUM' CHECK (risk_profile IN ('LOW', 'MEDIUM', 'HIGH', 'AGGRESSIVE')),
    notes TEXT,
    created_by VARCHAR(100),
    approved_by VARCHAR(100),
    approval_date TIMESTAMP WITH TIME ZONE,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    UNIQUE(symbol, strategy_id),
    CHECK (max_leverage >= 1.0 AND max_leverage <= 100.0),
    CHECK (max_drawdown >= 0.01 AND max_drawdown <= 1.0),
    CHECK (circuit_breaker_threshold >= 0.01 AND circuit_breaker_threshold <= 1.0),
    CHECK (risk_score_threshold >= 0.0 AND risk_score_threshold <= 100.0)
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_risk_parameters_symbol ON risk_parameters(symbol);
CREATE INDEX IF NOT EXISTS idx_risk_parameters_strategy_id ON risk_parameters(strategy_id);
CREATE INDEX IF NOT EXISTS idx_risk_parameters_is_active ON risk_parameters(is_active);
CREATE INDEX IF NOT EXISTS idx_risk_parameters_risk_profile ON risk_parameters(risk_profile);
CREATE INDEX IF NOT EXISTS idx_risk_parameters_created_at ON risk_parameters(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_parameters_updated_at ON risk_parameters(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_parameters_symbol_active ON risk_parameters(symbol, is_active);
CREATE INDEX IF NOT EXISTS idx_risk_parameters_max_leverage ON risk_parameters(max_leverage);
CREATE INDEX IF NOT EXISTS idx_risk_parameters_last_adjustment ON risk_parameters(last_adjustment DESC);

-- Create a composite index for common queries
CREATE INDEX IF NOT EXISTS idx_risk_parameters_active_symbol ON risk_parameters(symbol, is_active, risk_profile) 
WHERE is_active = TRUE;

-- Insert sample risk parameters for common trading pairs
INSERT INTO risk_parameters (
    symbol, max_leverage, max_position_size, max_drawdown, circuit_breaker_threshold,
    stop_loss_pct, take_profit_pct, trailing_stop_pct, initial_margin_rate, maintenance_margin_rate,
    max_order_value, min_order_value, risk_score_threshold, risk_profile, notes, created_by
) VALUES 
(
    'BTCUSDT', 10.0, 1000000.0, 0.15, 0.12,
    0.05, 0.10, 0.03, 0.10, 0.05,
    100000.0, 10.0, 70.0, 'MEDIUM', 'Default risk parameters for BTCUSDT', 'system'
),
(
    'ETHUSDT', 8.0, 800000.0, 0.18, 0.15,
    0.06, 0.12, 0.04, 0.12, 0.06,
    80000.0, 10.0, 75.0, 'MEDIUM', 'Default risk parameters for ETHUSDT', 'system'
),
(
    'BNBUSDT', 6.0, 500000.0, 0.20, 0.18,
    0.07, 0.15, 0.05, 0.15, 0.08,
    50000.0, 5.0, 80.0, 'HIGH', 'Default risk parameters for BNBUSDT', 'system'
),
(
    'ADAUSDT', 5.0, 300000.0, 0.25, 0.20,
    0.08, 0.18, 0.06, 0.20, 0.10,
    30000.0, 5.0, 85.0, 'HIGH', 'Default risk parameters for ADAUSDT', 'system'
),
(
    'SOLUSDT', 7.0, 400000.0, 0.22, 0.18,
    0.07, 0.14, 0.05, 0.14, 0.07,
    40000.0, 5.0, 78.0, 'MEDIUM', 'Default risk parameters for SOLUSDT', 'system'
);

-- Create a trigger to automatically update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_risk_parameters_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_risk_parameters_updated_at
    BEFORE UPDATE ON risk_parameters
    FOR EACH ROW
    EXECUTE FUNCTION update_risk_parameters_updated_at();

-- Add table comment for documentation
COMMENT ON TABLE risk_parameters IS 'Stores risk management parameters for trading symbols and strategies';

-- Add column comments for better understanding
COMMENT ON COLUMN risk_parameters.max_leverage IS 'Maximum allowed leverage for this symbol/strategy';
COMMENT ON COLUMN risk_parameters.max_position_size IS 'Maximum position size in base currency';
COMMENT ON COLUMN risk_parameters.max_drawdown IS 'Maximum allowed drawdown as a percentage (0.0-1.0)';
COMMENT ON COLUMN risk_parameters.circuit_breaker_threshold IS 'Loss percentage that triggers circuit breaker';
COMMENT ON COLUMN risk_parameters.risk_score_threshold IS 'Risk score threshold (0-100) for position limits';
COMMENT ON COLUMN risk_parameters.auto_adjust IS 'Whether parameters can be automatically adjusted';
COMMENT ON COLUMN risk_parameters.risk_profile IS 'Risk profile category (LOW, MEDIUM, HIGH, AGGRESSIVE)';

COMMIT;

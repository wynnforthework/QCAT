-- Migration to create abnormal_market_responses table
-- This fixes the "abnormal_market_responses" relation does not exist error

BEGIN;

-- Create abnormal_market_responses table if it doesn't exist
CREATE TABLE IF NOT EXISTS abnormal_market_responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol VARCHAR(20) NOT NULL,
    condition_type VARCHAR(50) NOT NULL, -- 'VOLATILITY_SPIKE', 'VOLUME_ANOMALY', 'PRICE_GAP', 'LIQUIDITY_CRISIS', etc.
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    value DECIMAL(20,8) NOT NULL, -- The actual value that triggered the condition
    threshold DECIMAL(20,8) NOT NULL, -- The threshold that was exceeded
    description TEXT NOT NULL,
    response_actions JSONB DEFAULT '[]', -- Array of actions taken in response
    
    -- Market context at the time of detection
    market_price DECIMAL(20,8),
    market_volume DECIMAL(20,8),
    market_volatility DECIMAL(10,6),
    
    -- Response metadata
    response_time_ms INTEGER, -- Time taken to respond in milliseconds
    actions_executed JSONB DEFAULT '[]', -- Actions that were actually executed
    execution_status VARCHAR(20) DEFAULT 'PENDING' CHECK (execution_status IN ('PENDING', 'EXECUTING', 'COMPLETED', 'FAILED', 'PARTIAL')),
    
    -- Risk assessment
    risk_score DECIMAL(5,2) DEFAULT 0, -- Risk score from 0-100
    impact_assessment TEXT,
    
    -- Resolution tracking
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolution_notes TEXT,
    
    -- Timestamps
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_abnormal_market_responses_symbol ON abnormal_market_responses(symbol);
CREATE INDEX IF NOT EXISTS idx_abnormal_market_responses_condition_type ON abnormal_market_responses(condition_type);
CREATE INDEX IF NOT EXISTS idx_abnormal_market_responses_severity ON abnormal_market_responses(severity);
CREATE INDEX IF NOT EXISTS idx_abnormal_market_responses_detected_at ON abnormal_market_responses(detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_abnormal_market_responses_created_at ON abnormal_market_responses(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_abnormal_market_responses_resolved ON abnormal_market_responses(resolved);
CREATE INDEX IF NOT EXISTS idx_abnormal_market_responses_execution_status ON abnormal_market_responses(execution_status);
CREATE INDEX IF NOT EXISTS idx_abnormal_market_responses_symbol_severity ON abnormal_market_responses(symbol, severity);
CREATE INDEX IF NOT EXISTS idx_abnormal_market_responses_symbol_detected_at ON abnormal_market_responses(symbol, detected_at DESC);

-- Create a composite index for common queries
CREATE INDEX IF NOT EXISTS idx_abnormal_market_responses_active ON abnormal_market_responses(symbol, resolved, severity) 
WHERE resolved = FALSE;

-- Insert sample data to demonstrate the table structure and prevent empty result errors
INSERT INTO abnormal_market_responses (
    symbol, condition_type, severity, value, threshold, description, 
    response_actions, market_price, market_volume, market_volatility,
    response_time_ms, actions_executed, execution_status, risk_score,
    impact_assessment, resolved, detected_at
) VALUES 
(
    'BTCUSDT', 'VOLATILITY_SPIKE', 'HIGH', 0.15, 0.10, 
    'Volatility spike detected: 15% vs threshold 10%',
    '["CIRCUIT_BREAKER", "LEVERAGE_REDUCTION", "EMERGENCY_PROTECTION"]',
    50000.0, 1000000.0, 0.15,
    250, '["CIRCUIT_BREAKER", "LEVERAGE_REDUCTION"]', 'COMPLETED', 85.5,
    'High volatility spike requiring immediate risk reduction measures',
    TRUE, NOW() - INTERVAL '2 hours'
),
(
    'ETHUSDT', 'VOLUME_ANOMALY', 'MEDIUM', 5000000.0, 2000000.0,
    'Unusual volume spike detected: 5M vs normal 2M',
    '["POSITION_MONITORING", "LIQUIDITY_CHECK"]',
    3000.0, 5000000.0, 0.08,
    150, '["POSITION_MONITORING"]', 'COMPLETED', 65.0,
    'Volume anomaly requiring enhanced monitoring',
    TRUE, NOW() - INTERVAL '1 hour'
),
(
    'BNBUSDT', 'PRICE_GAP', 'CRITICAL', 0.25, 0.05,
    'Large price gap detected: 25% vs threshold 5%',
    '["EMERGENCY_STOP", "POSITION_CLOSURE", "MARKET_HALT"]',
    300.0, 800000.0, 0.25,
    100, '["EMERGENCY_STOP", "POSITION_CLOSURE"]', 'COMPLETED', 95.0,
    'Critical price gap requiring immediate emergency response',
    FALSE, NOW() - INTERVAL '30 minutes'
),
(
    'ADAUSDT', 'LIQUIDITY_CRISIS', 'HIGH', 0.02, 0.10,
    'Liquidity crisis detected: 2% vs required 10%',
    '["TRADING_HALT", "POSITION_REDUCTION"]',
    0.5, 100000.0, 0.12,
    300, '["TRADING_HALT"]', 'EXECUTING', 80.0,
    'Low liquidity requiring trading restrictions',
    FALSE, NOW() - INTERVAL '15 minutes'
),
(
    'SOLUSDT', 'FUNDING_RATE_SPIKE', 'MEDIUM', 0.005, 0.002,
    'Funding rate spike detected: 0.5% vs normal 0.2%',
    '["POSITION_ADJUSTMENT", "HEDGE_ACTIVATION"]',
    150.0, 2000000.0, 0.06,
    200, '["POSITION_ADJUSTMENT"]', 'COMPLETED', 55.0,
    'Funding rate anomaly requiring position adjustments',
    TRUE, NOW() - INTERVAL '45 minutes'
);

-- Add table comment for documentation
COMMENT ON TABLE abnormal_market_responses IS 'Records abnormal market conditions and the automated responses taken to mitigate risks';

-- Add column comments for better understanding
COMMENT ON COLUMN abnormal_market_responses.condition_type IS 'Type of abnormal condition detected (VOLATILITY_SPIKE, VOLUME_ANOMALY, etc.)';
COMMENT ON COLUMN abnormal_market_responses.severity IS 'Severity level of the condition (LOW, MEDIUM, HIGH, CRITICAL)';
COMMENT ON COLUMN abnormal_market_responses.value IS 'The actual measured value that triggered the condition';
COMMENT ON COLUMN abnormal_market_responses.threshold IS 'The threshold value that was exceeded';
COMMENT ON COLUMN abnormal_market_responses.response_actions IS 'JSON array of planned response actions';
COMMENT ON COLUMN abnormal_market_responses.actions_executed IS 'JSON array of actions that were actually executed';
COMMENT ON COLUMN abnormal_market_responses.execution_status IS 'Status of response execution (PENDING, EXECUTING, COMPLETED, FAILED, PARTIAL)';
COMMENT ON COLUMN abnormal_market_responses.risk_score IS 'Calculated risk score from 0-100';

COMMIT;

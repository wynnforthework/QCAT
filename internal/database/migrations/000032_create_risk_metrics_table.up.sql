-- Migration: Create risk_metrics table and fix dashboard API issues
-- This migration creates the missing risk_metrics table that is referenced in the dashboard API

BEGIN;

-- Create risk_metrics table
CREATE TABLE IF NOT EXISTS risk_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id UUID REFERENCES strategies(id),
    symbol VARCHAR(20),
    risk_score DECIMAL(10,4) NOT NULL DEFAULT 0,
    var_95 DECIMAL(20,8) NOT NULL DEFAULT 0,
    var_99 DECIMAL(20,8) NOT NULL DEFAULT 0,
    max_drawdown DECIMAL(10,4) NOT NULL DEFAULT 0,
    leverage DECIMAL(10,4) NOT NULL DEFAULT 0,
    position_size DECIMAL(20,8) NOT NULL DEFAULT 0,
    margin_used DECIMAL(20,8) NOT NULL DEFAULT 0,
    liquidation_distance DECIMAL(10,4) NOT NULL DEFAULT 0,
    correlation_risk DECIMAL(10,4) NOT NULL DEFAULT 0,
    concentration_risk DECIMAL(10,4) NOT NULL DEFAULT 0,
    liquidity_risk DECIMAL(10,4) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_risk_metrics_strategy_id ON risk_metrics(strategy_id);
CREATE INDEX IF NOT EXISTS idx_risk_metrics_symbol ON risk_metrics(symbol);
CREATE INDEX IF NOT EXISTS idx_risk_metrics_created_at ON risk_metrics(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_metrics_risk_score ON risk_metrics(risk_score DESC);

-- Create risk_alerts table if it doesn't exist (referenced in dashboard API)
CREATE TABLE IF NOT EXISTS risk_alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    symbol VARCHAR(20),
    strategy_id UUID REFERENCES strategies(id),
    message TEXT NOT NULL,
    threshold_value DECIMAL(20,8),
    current_value DECIMAL(20,8),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'resolved', 'ignored')),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by VARCHAR(100)
);

-- Create indexes for risk_alerts
CREATE INDEX IF NOT EXISTS idx_risk_alerts_status ON risk_alerts(status);
CREATE INDEX IF NOT EXISTS idx_risk_alerts_severity ON risk_alerts(severity);
CREATE INDEX IF NOT EXISTS idx_risk_alerts_created_at ON risk_alerts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_alerts_symbol ON risk_alerts(symbol);
CREATE INDEX IF NOT EXISTS idx_risk_alerts_strategy_id ON risk_alerts(strategy_id);

-- Insert sample data for testing
INSERT INTO risk_metrics (strategy_id, symbol, risk_score, var_95, max_drawdown, leverage) 
SELECT 
    s.id,
    'BTCUSDT',
    RANDOM() * 10,
    RANDOM() * 1000,
    RANDOM() * 0.2,
    RANDOM() * 5 + 1
FROM strategies s 
WHERE s.status = 'active'
ON CONFLICT DO NOTHING;

-- Insert sample risk alerts
INSERT INTO risk_alerts (alert_type, severity, symbol, message, status)
VALUES 
    ('high_leverage', 'medium', 'BTCUSDT', 'Leverage exceeds recommended threshold', 'active'),
    ('max_drawdown', 'high', 'ETHUSDT', 'Maximum drawdown limit reached', 'active'),
    ('position_size', 'low', 'BNBUSDT', 'Position size approaching limit', 'resolved')
ON CONFLICT DO NOTHING;

-- Add comments for documentation
COMMENT ON TABLE risk_metrics IS 'Real-time risk metrics for strategies and positions';
COMMENT ON TABLE risk_alerts IS 'Risk management alerts and notifications';

COMMIT;

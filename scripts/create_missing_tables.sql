-- Create missing tables for API endpoints

-- Create hotlist_scores table for hot symbol scoring
CREATE TABLE IF NOT EXISTS hotlist_scores (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    symbol VARCHAR(20) NOT NULL,
    vol_jump_score DECIMAL(10,6) NOT NULL DEFAULT 0,
    turnover_score DECIMAL(10,6) NOT NULL DEFAULT 0,
    oi_change_score DECIMAL(10,6) NOT NULL DEFAULT 0,
    funding_z_score DECIMAL(10,6) NOT NULL DEFAULT 0,
    regime_shift_score DECIMAL(10,6) NOT NULL DEFAULT 0,
    total_score DECIMAL(10,6) NOT NULL DEFAULT 0,
    risk_level VARCHAR(20) NOT NULL DEFAULT 'medium',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create trading_whitelist table for approved trading symbols
CREATE TABLE IF NOT EXISTS trading_whitelist (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    symbol VARCHAR(20) NOT NULL UNIQUE,
    approved_by UUID,
    approved_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    reason TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create audit_logs table for system audit logging
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID,
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    session_id VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'success',
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create audit_decisions table for decision chain tracking
CREATE TABLE IF NOT EXISTS audit_decisions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    decision_id VARCHAR(100) NOT NULL UNIQUE,
    strategy_id UUID,
    decision_type VARCHAR(50) NOT NULL,
    input_data JSONB NOT NULL,
    output_data JSONB NOT NULL,
    decision_path JSONB,
    confidence_score DECIMAL(10,6),
    execution_time_ms INTEGER,
    status VARCHAR(20) NOT NULL DEFAULT 'executed',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create audit_performance table for performance metrics tracking
CREATE TABLE IF NOT EXISTS audit_performance (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    metric_type VARCHAR(50) NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    value DECIMAL(30,10) NOT NULL,
    unit VARCHAR(20) NOT NULL DEFAULT 'ms',
    tags JSONB,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Fix circuit_breakers table structure
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'circuit_breakers' AND column_name = 'action') THEN
        ALTER TABLE circuit_breakers ADD COLUMN action VARCHAR(50) DEFAULT 'halt';
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'circuit_breakers' AND column_name = 'triggered_at') THEN
        ALTER TABLE circuit_breakers ADD COLUMN triggered_at TIMESTAMP WITH TIME ZONE;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        NULL;
END $$;

-- Fix risk_violations table structure
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_violations' AND column_name = 'symbol') THEN
        ALTER TABLE risk_violations ADD COLUMN symbol VARCHAR(20);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_violations' AND column_name = 'message') THEN
        ALTER TABLE risk_violations ADD COLUMN message TEXT;
    END IF;
    
    -- Rename columns if they exist with old names
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_violations' AND column_name = 'violation_type') THEN
        ALTER TABLE risk_violations RENAME COLUMN violation_type TO type;
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_violations' AND column_name = 'threshold_value') THEN
        ALTER TABLE risk_violations RENAME COLUMN threshold_value TO threshold;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        NULL;
END $$;

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_hotlist_scores_symbol ON hotlist_scores(symbol);
CREATE INDEX IF NOT EXISTS idx_hotlist_scores_total_score ON hotlist_scores(total_score DESC);
CREATE INDEX IF NOT EXISTS idx_trading_whitelist_symbol ON trading_whitelist(symbol);
CREATE INDEX IF NOT EXISTS idx_trading_whitelist_status ON trading_whitelist(status);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_decisions_strategy_id ON audit_decisions(strategy_id);
CREATE INDEX IF NOT EXISTS idx_audit_decisions_created_at ON audit_decisions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_performance_metric ON audit_performance(metric_type, metric_name);

-- Insert sample data
INSERT INTO hotlist_scores (symbol, vol_jump_score, turnover_score, oi_change_score, funding_z_score, regime_shift_score, total_score, risk_level) 
SELECT 'BTCUSDT', 0.85, 0.92, 0.78, 0.65, 0.88, 0.816, 'high'
WHERE NOT EXISTS (SELECT 1 FROM hotlist_scores WHERE symbol = 'BTCUSDT');

INSERT INTO hotlist_scores (symbol, vol_jump_score, turnover_score, oi_change_score, funding_z_score, regime_shift_score, total_score, risk_level) 
SELECT 'ETHUSDT', 0.72, 0.85, 0.69, 0.58, 0.75, 0.718, 'medium'
WHERE NOT EXISTS (SELECT 1 FROM hotlist_scores WHERE symbol = 'ETHUSDT');

INSERT INTO trading_whitelist (symbol, status, reason) 
SELECT 'BTCUSDT', 'active', 'High liquidity and volume'
WHERE NOT EXISTS (SELECT 1 FROM trading_whitelist WHERE symbol = 'BTCUSDT');

INSERT INTO trading_whitelist (symbol, status, reason) 
SELECT 'ETHUSDT', 'active', 'Major cryptocurrency'
WHERE NOT EXISTS (SELECT 1 FROM trading_whitelist WHERE symbol = 'ETHUSDT');

INSERT INTO audit_performance (metric_type, metric_name, value, unit, tags) 
SELECT 'api_response_time', 'GET /api/v1/dashboard', 150.5, 'ms', '{"endpoint": "/api/v1/dashboard", "method": "GET"}'::jsonb
WHERE NOT EXISTS (SELECT 1 FROM audit_performance WHERE metric_name = 'GET /api/v1/dashboard');

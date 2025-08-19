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
    risk_level VARCHAR(20) NOT NULL DEFAULT 'medium', -- 'low', 'medium', 'high'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(symbol, created_at)
);

-- Create trading_whitelist table for approved trading symbols
CREATE TABLE IF NOT EXISTS trading_whitelist (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    symbol VARCHAR(20) NOT NULL,
    approved_by UUID, -- references users(id)
    approved_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'active', 'suspended', 'removed'
    reason TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(symbol)
);

-- Create audit_logs table for system audit logging
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID, -- references users(id)
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL, -- 'strategy', 'portfolio', 'order', 'user', etc.
    resource_id UUID,
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    session_id VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'success', -- 'success', 'failed', 'pending'
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create audit_decisions table for decision chain tracking
CREATE TABLE IF NOT EXISTS audit_decisions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    decision_id VARCHAR(100) NOT NULL,
    strategy_id UUID, -- references strategies(id)
    decision_type VARCHAR(50) NOT NULL, -- 'entry', 'exit', 'risk_check', 'rebalance'
    input_data JSONB NOT NULL,
    output_data JSONB NOT NULL,
    decision_path JSONB, -- array of decision steps
    confidence_score DECIMAL(10,6),
    execution_time_ms INTEGER,
    status VARCHAR(20) NOT NULL DEFAULT 'executed', -- 'executed', 'rejected', 'pending'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(decision_id)
);

-- Create audit_performance table for performance metrics tracking
CREATE TABLE IF NOT EXISTS audit_performance (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    metric_type VARCHAR(50) NOT NULL, -- 'api_response_time', 'db_query_time', 'strategy_execution'
    metric_name VARCHAR(100) NOT NULL,
    value DECIMAL(30,10) NOT NULL,
    unit VARCHAR(20) NOT NULL DEFAULT 'ms', -- 'ms', 'seconds', 'count', 'percentage'
    tags JSONB, -- additional metadata
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Fix circuit_breakers table structure to match handler expectations
DO $$ 
BEGIN
    -- Add missing columns to circuit_breakers if they don't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'circuit_breakers' AND column_name = 'action') THEN
        ALTER TABLE circuit_breakers ADD COLUMN action VARCHAR(50) DEFAULT 'halt';
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'circuit_breakers' AND column_name = 'triggered_at') THEN
        ALTER TABLE circuit_breakers ADD COLUMN triggered_at TIMESTAMP WITH TIME ZONE;
    END IF;
END $$;

-- Fix risk_violations table structure to match handler expectations  
DO $$ 
BEGIN
    -- Add missing columns to risk_violations if they don't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_violations' AND column_name = 'symbol') THEN
        ALTER TABLE risk_violations ADD COLUMN symbol VARCHAR(20);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_violations' AND column_name = 'message') THEN
        ALTER TABLE risk_violations ADD COLUMN message TEXT;
    END IF;
    
    -- Rename columns to match handler expectations
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_violations' AND column_name = 'violation_type') THEN
        ALTER TABLE risk_violations RENAME COLUMN violation_type TO type;
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_violations' AND column_name = 'threshold_value') THEN
        ALTER TABLE risk_violations RENAME COLUMN threshold_value TO threshold;
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_violations' AND column_name = 'actual_value') THEN
        ALTER TABLE risk_violations RENAME COLUMN actual_value TO actual_value;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        -- Ignore errors during column operations
        NULL;
END $$;

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_hotlist_scores_symbol ON hotlist_scores(symbol);
CREATE INDEX IF NOT EXISTS idx_hotlist_scores_total_score ON hotlist_scores(total_score DESC);
CREATE INDEX IF NOT EXISTS idx_hotlist_scores_created_at ON hotlist_scores(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_trading_whitelist_symbol ON trading_whitelist(symbol);
CREATE INDEX IF NOT EXISTS idx_trading_whitelist_status ON trading_whitelist(status);
CREATE INDEX IF NOT EXISTS idx_trading_whitelist_approved_at ON trading_whitelist(approved_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_decisions_strategy_id ON audit_decisions(strategy_id);
CREATE INDEX IF NOT EXISTS idx_audit_decisions_type ON audit_decisions(decision_type);
CREATE INDEX IF NOT EXISTS idx_audit_decisions_created_at ON audit_decisions(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_performance_metric ON audit_performance(metric_type, metric_name);
CREATE INDEX IF NOT EXISTS idx_audit_performance_timestamp ON audit_performance(timestamp DESC);

-- Insert sample data for testing (only if tables are empty)
DO $$
BEGIN
    -- Insert sample hotlist scores if table is empty
    IF NOT EXISTS (SELECT 1 FROM hotlist_scores LIMIT 1) THEN
        INSERT INTO hotlist_scores (symbol, vol_jump_score, turnover_score, oi_change_score, funding_z_score, regime_shift_score, total_score, risk_level) VALUES
        ('BTCUSDT', 0.85, 0.92, 0.78, 0.65, 0.88, 0.816, 'high'),
        ('ETHUSDT', 0.72, 0.85, 0.69, 0.58, 0.75, 0.718, 'medium'),
        ('ADAUSDT', 0.45, 0.52, 0.38, 0.42, 0.48, 0.45, 'low'),
        ('SOLUSDT', 0.68, 0.75, 0.62, 0.55, 0.71, 0.662, 'medium'),
        ('DOTUSDT', 0.35, 0.42, 0.28, 0.31, 0.38, 0.348, 'low');
    END IF;

    -- Insert sample whitelist if table is empty
    IF NOT EXISTS (SELECT 1 FROM trading_whitelist LIMIT 1) THEN
        INSERT INTO trading_whitelist (symbol, status, reason) VALUES
        ('BTCUSDT', 'active', 'High liquidity and volume'),
        ('ETHUSDT', 'active', 'Major cryptocurrency'),
        ('ADAUSDT', 'active', 'Stable trading pair'),
        ('SOLUSDT', 'active', 'Growing ecosystem'),
        ('BNBUSDT', 'suspended', 'Under review');
    END IF;

    -- Insert sample audit performance if table is empty
    IF NOT EXISTS (SELECT 1 FROM audit_performance LIMIT 1) THEN
        INSERT INTO audit_performance (metric_type, metric_name, value, unit, tags) VALUES
        ('api_response_time', 'GET /api/v1/dashboard', 150.5, 'ms', '{"endpoint": "/api/v1/dashboard", "method": "GET"}'),
        ('db_query_time', 'portfolio_overview_query', 45.2, 'ms', '{"table": "portfolios", "operation": "select"}'),
        ('strategy_execution', 'momentum_strategy_cycle', 2500, 'ms', '{"strategy_type": "momentum", "symbol": "BTCUSDT"}');
    END IF;
END $$;

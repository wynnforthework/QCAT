-- Create portfolio_history table for portfolio history tracking
CREATE TABLE portfolio_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    portfolio_id UUID,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    total_value DECIMAL(30,10) NOT NULL DEFAULT 0,
    total_pnl DECIMAL(30,10) NOT NULL DEFAULT 0,
    daily_return DECIMAL(10,6) NOT NULL DEFAULT 0,
    cumulative_return DECIMAL(10,6) NOT NULL DEFAULT 0,
    sharpe_ratio DECIMAL(10,6),
    max_drawdown DECIMAL(10,6),
    volatility DECIMAL(10,6),
    allocations JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create circuit_breakers table for risk management
CREATE TABLE circuit_breakers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL, -- 'position', 'portfolio', 'strategy', 'market'
    threshold DECIMAL(10,6) NOT NULL,
    current_value DECIMAL(10,6) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'active', 'triggered', 'disabled'
    trigger_count INTEGER NOT NULL DEFAULT 0,
    last_triggered_at TIMESTAMP WITH TIME ZONE,
    config JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name)
);

-- Create risk_violations table for risk violation tracking
CREATE TABLE risk_violations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    violation_type VARCHAR(50) NOT NULL, -- 'position_limit', 'drawdown', 'var', 'circuit_breaker'
    severity VARCHAR(20) NOT NULL DEFAULT 'medium', -- 'low', 'medium', 'high', 'critical'
    entity_type VARCHAR(50) NOT NULL, -- 'strategy', 'portfolio', 'position', 'market'
    entity_id UUID,
    description TEXT NOT NULL,
    threshold_value DECIMAL(30,10),
    actual_value DECIMAL(30,10),
    status VARCHAR(20) NOT NULL DEFAULT 'open', -- 'open', 'acknowledged', 'resolved'
    resolved_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Update optimizer_tasks table to add missing columns if they don't exist
DO $$ 
BEGIN
    -- Add missing columns to optimizer_tasks if they don't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'optimizer_tasks' AND column_name = 'algorithm') THEN
        ALTER TABLE optimizer_tasks ADD COLUMN algorithm VARCHAR(50) DEFAULT 'walk_forward';
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'optimizer_tasks' AND column_name = 'parameters') THEN
        ALTER TABLE optimizer_tasks ADD COLUMN parameters JSONB;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'optimizer_tasks' AND column_name = 'results') THEN
        ALTER TABLE optimizer_tasks ADD COLUMN results JSONB;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'optimizer_tasks' AND column_name = 'progress') THEN
        ALTER TABLE optimizer_tasks ADD COLUMN progress INTEGER DEFAULT 0;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'optimizer_tasks' AND column_name = 'error_message') THEN
        ALTER TABLE optimizer_tasks ADD COLUMN error_message TEXT;
    END IF;
EXCEPTION
    WHEN undefined_table THEN
        -- Create optimizer_tasks table if it doesn't exist
        CREATE TABLE optimizer_tasks (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            strategy_id UUID REFERENCES strategies(id),
            algorithm VARCHAR(50) NOT NULL DEFAULT 'walk_forward',
            status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'running', 'completed', 'failed'
            progress INTEGER NOT NULL DEFAULT 0,
            parameters JSONB,
            results JSONB,
            error_message TEXT,
            started_at TIMESTAMP WITH TIME ZONE,
            completed_at TIMESTAMP WITH TIME ZONE,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        );
END $$;

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_portfolio_history_timestamp ON portfolio_history(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_portfolio_history_portfolio_id ON portfolio_history(portfolio_id);

CREATE INDEX IF NOT EXISTS idx_circuit_breakers_type ON circuit_breakers(type);
CREATE INDEX IF NOT EXISTS idx_circuit_breakers_status ON circuit_breakers(status);
CREATE INDEX IF NOT EXISTS idx_circuit_breakers_name ON circuit_breakers(name);

CREATE INDEX IF NOT EXISTS idx_risk_violations_type ON risk_violations(violation_type);
CREATE INDEX IF NOT EXISTS idx_risk_violations_severity ON risk_violations(severity);
CREATE INDEX IF NOT EXISTS idx_risk_violations_status ON risk_violations(status);
CREATE INDEX IF NOT EXISTS idx_risk_violations_entity ON risk_violations(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_risk_violations_created_at ON risk_violations(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_optimizer_tasks_strategy_id ON optimizer_tasks(strategy_id);
CREATE INDEX IF NOT EXISTS idx_optimizer_tasks_status ON optimizer_tasks(status);
CREATE INDEX IF NOT EXISTS idx_optimizer_tasks_algorithm ON optimizer_tasks(algorithm);
CREATE INDEX IF NOT EXISTS idx_optimizer_tasks_created_at ON optimizer_tasks(created_at DESC);

-- Insert sample data for testing
INSERT INTO circuit_breakers (name, type, threshold, current_value, status, config) VALUES
('Portfolio Max Drawdown', 'portfolio', 0.15, 0.05, 'active', '{"check_interval": "1m", "alert_threshold": 0.10}'),
('Position Size Limit', 'position', 0.20, 0.12, 'active', '{"max_position_size": 100000}'),
('Daily Loss Limit', 'portfolio', 0.05, 0.02, 'active', '{"reset_time": "00:00:00"}'),
('Strategy Drawdown', 'strategy', 0.10, 0.03, 'active', '{"lookback_period": "30d"}')

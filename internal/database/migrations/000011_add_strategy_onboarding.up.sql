-- Migration 000011: Add strategy onboarding tables
-- This migration adds tables to support automated strategy onboarding process

BEGIN;

-- Create strategy_onboarding table to track onboarding process
CREATE TABLE IF NOT EXISTS strategy_onboarding (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'processing', 'validation_failed', 'risk_rejected', 'approved_pending_deployment', 'deployed', 'failed'
    risk_level VARCHAR(20) NOT NULL DEFAULT 'medium', -- 'low', 'medium', 'high', 'unacceptable'
    validation_score DECIMAL(5,2) DEFAULT 0,
    risk_score DECIMAL(5,2) DEFAULT 0,
    auto_deploy BOOLEAN DEFAULT false,
    test_mode BOOLEAN DEFAULT false,
    
    -- Validation results
    validation_errors JSONB,
    validation_warnings JSONB,
    validation_passed JSONB,
    
    -- Risk assessment results
    risk_assessment JSONB,
    risk_recommendations JSONB,
    
    -- Deployment information
    deployment_id VARCHAR(255),
    deployment_environment VARCHAR(20), -- 'test', 'staging', 'production'
    deployment_status VARCHAR(50),
    deployment_config JSONB,
    
    -- Monitoring information
    monitoring_started_at TIMESTAMP WITH TIME ZONE,
    monitoring_status VARCHAR(20), -- 'active', 'paused', 'stopped'
    
    -- Timestamps
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create onboarding_stages table to track individual stages
CREATE TABLE IF NOT EXISTS onboarding_stages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    onboarding_id UUID NOT NULL REFERENCES strategy_onboarding(id) ON DELETE CASCADE,
    stage_name VARCHAR(50) NOT NULL, -- 'validation', 'risk_assessment', 'deployment', 'monitoring'
    stage_order INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'running', 'completed', 'failed', 'skipped'
    
    -- Stage execution details
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_seconds INTEGER,
    
    -- Stage results
    result JSONB,
    error_message TEXT,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create validation_rules table to store validation rules
CREATE TABLE IF NOT EXISTS validation_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_name VARCHAR(100) NOT NULL UNIQUE,
    rule_type VARCHAR(50) NOT NULL, -- 'config', 'parameter', 'risk', 'code_quality', 'performance', 'security'
    severity VARCHAR(20) NOT NULL DEFAULT 'error', -- 'error', 'warning', 'info'
    description TEXT,
    rule_config JSONB,
    is_active BOOLEAN DEFAULT true,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create risk_models table to store risk assessment models
CREATE TABLE IF NOT EXISTS risk_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_name VARCHAR(100) NOT NULL UNIQUE,
    model_type VARCHAR(50) NOT NULL, -- 'drawdown', 'volatility', 'leverage', 'concentration', 'liquidity'
    weight DECIMAL(3,2) NOT NULL DEFAULT 0.20, -- Model weight in overall assessment
    description TEXT,
    model_config JSONB,
    is_active BOOLEAN DEFAULT true,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create deployment_history table to track deployment history
CREATE TABLE IF NOT EXISTS deployment_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id VARCHAR(255) NOT NULL,
    onboarding_id UUID REFERENCES strategy_onboarding(id),
    deployment_id VARCHAR(255) NOT NULL,
    
    -- Deployment details
    environment VARCHAR(20) NOT NULL, -- 'test', 'staging', 'production'
    status VARCHAR(50) NOT NULL, -- 'deploying', 'deployed', 'failed', 'rolled_back'
    version VARCHAR(50),
    
    -- Configuration
    deployment_config JSONB,
    rollback_config JSONB,
    
    -- Health check information
    health_status VARCHAR(20), -- 'healthy', 'unhealthy', 'unknown'
    health_checks_passed INTEGER DEFAULT 0,
    health_checks_failed INTEGER DEFAULT 0,
    last_health_check TIMESTAMP WITH TIME ZONE,
    
    -- Timestamps
    deployed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    rolled_back_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_strategy_onboarding_strategy_id ON strategy_onboarding(strategy_id);
CREATE INDEX IF NOT EXISTS idx_strategy_onboarding_status ON strategy_onboarding(status);
CREATE INDEX IF NOT EXISTS idx_strategy_onboarding_risk_level ON strategy_onboarding(risk_level);
CREATE INDEX IF NOT EXISTS idx_strategy_onboarding_created_at ON strategy_onboarding(created_at);

CREATE INDEX IF NOT EXISTS idx_onboarding_stages_onboarding_id ON onboarding_stages(onboarding_id);
CREATE INDEX IF NOT EXISTS idx_onboarding_stages_stage_name ON onboarding_stages(stage_name);
CREATE INDEX IF NOT EXISTS idx_onboarding_stages_status ON onboarding_stages(status);

CREATE INDEX IF NOT EXISTS idx_validation_rules_rule_type ON validation_rules(rule_type);
CREATE INDEX IF NOT EXISTS idx_validation_rules_is_active ON validation_rules(is_active);

CREATE INDEX IF NOT EXISTS idx_risk_models_model_type ON risk_models(model_type);
CREATE INDEX IF NOT EXISTS idx_risk_models_is_active ON risk_models(is_active);

CREATE INDEX IF NOT EXISTS idx_deployment_history_strategy_id ON deployment_history(strategy_id);
CREATE INDEX IF NOT EXISTS idx_deployment_history_deployment_id ON deployment_history(deployment_id);
CREATE INDEX IF NOT EXISTS idx_deployment_history_environment ON deployment_history(environment);
CREATE INDEX IF NOT EXISTS idx_deployment_history_status ON deployment_history(status);

-- Insert default validation rules
INSERT INTO validation_rules (rule_name, rule_type, severity, description, rule_config) VALUES
('config_validation', 'config', 'error', 'Validates strategy configuration completeness and format', '{"required_fields": ["name", "symbol", "exchange"]}'),
('parameter_validation', 'parameter', 'error', 'Validates strategy parameters are within acceptable ranges', '{"stop_loss_range": [0.001, 0.5], "position_size_range": [0.01, 1.0]}'),
('risk_validation', 'risk', 'error', 'Validates risk profile settings', '{"max_drawdown_range": [0.01, 0.5], "max_leverage_range": [1, 100]}'),
('code_quality', 'code_quality', 'warning', 'Checks strategy code for quality and security issues', '{"max_code_size": 100000, "dangerous_patterns": ["os.system", "exec(", "eval("]}'),
('performance_validation', 'performance', 'warning', 'Validates expected performance metrics', '{"return_range": [-0.5, 5.0]}'),
('security_validation', 'security', 'error', 'Validates security aspects of strategy', '{"id_pattern": "^[a-zA-Z0-9_-]+$"}');

-- Insert default risk models
INSERT INTO risk_models (model_name, model_type, weight, description, model_config) VALUES
('drawdown_risk', 'drawdown', 0.30, 'Assesses maximum drawdown risk', '{"thresholds": {"low": 0.05, "medium": 0.1, "high": 0.2, "extreme": 0.3}}'),
('leverage_risk', 'leverage', 0.25, 'Assesses leverage risk', '{"thresholds": {"low": 2, "medium": 5, "high": 10, "extreme": 20}}'),
('volatility_risk', 'volatility', 0.20, 'Assesses market volatility risk', '{"symbol_categories": {"major": ["BTCUSDT", "ETHUSDT"], "minor": [], "exotic": []}}'),
('concentration_risk', 'concentration', 0.15, 'Assesses position concentration risk', '{"thresholds": {"low": 0.1, "medium": 0.2, "high": 0.5}}'),
('liquidity_risk', 'liquidity', 0.10, 'Assesses market liquidity risk', '{"high_liquidity": ["BTCUSDT", "ETHUSDT", "BNBUSDT"]}');

-- Add comments for documentation
COMMENT ON TABLE strategy_onboarding IS 'Tracks the automated strategy onboarding process';
COMMENT ON TABLE onboarding_stages IS 'Tracks individual stages of the onboarding process';
COMMENT ON TABLE validation_rules IS 'Stores validation rules for strategy onboarding';
COMMENT ON TABLE risk_models IS 'Stores risk assessment models and their configurations';
COMMENT ON TABLE deployment_history IS 'Tracks strategy deployment history and status';

COMMIT;

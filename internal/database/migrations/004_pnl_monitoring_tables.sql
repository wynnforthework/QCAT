-- Migration: Create PnL monitoring tables
-- Version: 004
-- Description: Creates tables for PnL monitoring and risk management

-- PnL snapshots table
CREATE TABLE IF NOT EXISTS pnl_snapshots (
    id BIGSERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    unrealized_pnl DECIMAL(20,8) NOT NULL DEFAULT 0,
    realized_pnl DECIMAL(20,8) NOT NULL DEFAULT 0,
    total_pnl DECIMAL(20,8) NOT NULL DEFAULT 0,
    margin_used DECIMAL(20,8) NOT NULL DEFAULT 0,
    margin_ratio DECIMAL(10,4) NOT NULL DEFAULT 0,
    mark_price DECIMAL(20,8) NOT NULL DEFAULT 0,
    entry_price DECIMAL(20,8) NOT NULL DEFAULT 0,
    position_size DECIMAL(20,8) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    INDEX idx_pnl_snapshots_symbol (symbol),
    INDEX idx_pnl_snapshots_created_at (created_at),
    INDEX idx_pnl_snapshots_symbol_created_at (symbol, created_at)
);

-- Risk thresholds configuration table
CREATE TABLE IF NOT EXISTS risk_thresholds (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    max_margin_ratio DECIMAL(5,4) NOT NULL DEFAULT 0.8000,
    warning_margin_ratio DECIMAL(5,4) NOT NULL DEFAULT 0.7000,
    max_daily_loss DECIMAL(20,8) NOT NULL DEFAULT 5000.00000000,
    max_total_loss DECIMAL(20,8) NOT NULL DEFAULT 10000.00000000,
    max_drawdown_percent DECIMAL(5,4) NOT NULL DEFAULT 0.2000,
    max_position_loss DECIMAL(20,8) NOT NULL DEFAULT 1000.00000000,
    max_position_loss_percent DECIMAL(5,4) NOT NULL DEFAULT 0.1000,
    min_account_balance DECIMAL(20,8) NOT NULL DEFAULT 10000.00000000,
    max_leverage INTEGER NOT NULL DEFAULT 10,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Risk events table
CREATE TABLE IF NOT EXISTS risk_events (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL,
    action_type VARCHAR(50) NOT NULL,
    symbol VARCHAR(20),
    current_value DECIMAL(20,8) NOT NULL,
    threshold_value DECIMAL(20,8) NOT NULL,
    message TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    metadata JSONB,
    executed BOOLEAN DEFAULT false,
    execution_result TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    executed_at TIMESTAMP WITH TIME ZONE,
    
    INDEX idx_risk_events_type (event_type),
    INDEX idx_risk_events_symbol (symbol),
    INDEX idx_risk_events_severity (severity),
    INDEX idx_risk_events_created_at (created_at),
    INDEX idx_risk_events_executed (executed)
);

-- Margin alerts table
CREATE TABLE IF NOT EXISTS margin_alerts (
    id BIGSERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    alert_type VARCHAR(50) NOT NULL,
    current_ratio DECIMAL(10,4) NOT NULL,
    threshold_ratio DECIMAL(10,4) NOT NULL,
    message TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('warning', 'critical')),
    resolved BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE,
    
    INDEX idx_margin_alerts_symbol (symbol),
    INDEX idx_margin_alerts_type (alert_type),
    INDEX idx_margin_alerts_severity (severity),
    INDEX idx_margin_alerts_resolved (resolved),
    INDEX idx_margin_alerts_created_at (created_at)
);

-- Account equity history table
CREATE TABLE IF NOT EXISTS account_equity_history (
    id BIGSERIAL PRIMARY KEY,
    total_equity DECIMAL(20,8) NOT NULL,
    available_balance DECIMAL(20,8) NOT NULL,
    used_margin DECIMAL(20,8) NOT NULL,
    unrealized_pnl DECIMAL(20,8) NOT NULL DEFAULT 0,
    realized_pnl DECIMAL(20,8) NOT NULL DEFAULT 0,
    margin_ratio DECIMAL(10,4) NOT NULL DEFAULT 0,
    position_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    INDEX idx_account_equity_created_at (created_at)
);

-- Daily PnL summary table
CREATE TABLE IF NOT EXISTS daily_pnl_summary (
    id BIGSERIAL PRIMARY KEY,
    trade_date DATE NOT NULL UNIQUE,
    starting_balance DECIMAL(20,8) NOT NULL,
    ending_balance DECIMAL(20,8) NOT NULL,
    realized_pnl DECIMAL(20,8) NOT NULL DEFAULT 0,
    unrealized_pnl DECIMAL(20,8) NOT NULL DEFAULT 0,
    total_pnl DECIMAL(20,8) NOT NULL DEFAULT 0,
    max_drawdown DECIMAL(20,8) NOT NULL DEFAULT 0,
    max_equity DECIMAL(20,8) NOT NULL DEFAULT 0,
    trade_count INTEGER NOT NULL DEFAULT 0,
    win_count INTEGER NOT NULL DEFAULT 0,
    loss_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    INDEX idx_daily_pnl_trade_date (trade_date)
);

-- Position risk metrics table
CREATE TABLE IF NOT EXISTS position_risk_metrics (
    id BIGSERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    position_size DECIMAL(20,8) NOT NULL,
    entry_price DECIMAL(20,8) NOT NULL,
    mark_price DECIMAL(20,8) NOT NULL,
    unrealized_pnl DECIMAL(20,8) NOT NULL,
    unrealized_pnl_percent DECIMAL(10,4) NOT NULL,
    margin_used DECIMAL(20,8) NOT NULL,
    leverage INTEGER NOT NULL,
    liquidation_price DECIMAL(20,8),
    distance_to_liquidation DECIMAL(10,4),
    var_95 DECIMAL(20,8),
    var_99 DECIMAL(20,8),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    INDEX idx_position_risk_symbol (symbol),
    INDEX idx_position_risk_created_at (created_at),
    INDEX idx_position_risk_symbol_created_at (symbol, created_at)
);

-- Automated actions log table
CREATE TABLE IF NOT EXISTS automated_actions_log (
    id BIGSERIAL PRIMARY KEY,
    action_type VARCHAR(50) NOT NULL,
    symbol VARCHAR(20),
    trigger_event_id BIGINT REFERENCES risk_events(id),
    action_details JSONB NOT NULL,
    execution_status VARCHAR(20) NOT NULL CHECK (execution_status IN ('pending', 'executing', 'completed', 'failed')),
    execution_result TEXT,
    dry_run BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    executed_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    
    INDEX idx_automated_actions_type (action_type),
    INDEX idx_automated_actions_symbol (symbol),
    INDEX idx_automated_actions_status (execution_status),
    INDEX idx_automated_actions_created_at (created_at)
);

-- Insert default risk thresholds
INSERT INTO risk_thresholds (
    name, max_margin_ratio, warning_margin_ratio, max_daily_loss, 
    max_total_loss, max_drawdown_percent, max_position_loss, 
    max_position_loss_percent, min_account_balance, max_leverage
) VALUES (
    'default',
    0.8000,  -- 80% max margin ratio
    0.7000,  -- 70% warning margin ratio
    5000.00000000,  -- $5000 max daily loss
    10000.00000000, -- $10000 max total loss
    0.2000,  -- 20% max drawdown
    1000.00000000,  -- $1000 max position loss
    0.1000,  -- 10% max position loss percentage
    10000.00000000, -- $10000 min account balance
    10       -- 10x max leverage
) ON CONFLICT (name) DO NOTHING;

-- Create triggers for updated_at
CREATE TRIGGER update_risk_thresholds_updated_at 
    BEFORE UPDATE ON risk_thresholds 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_daily_pnl_summary_updated_at 
    BEFORE UPDATE ON daily_pnl_summary 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create views for easier querying

-- Current positions with risk metrics
CREATE OR REPLACE VIEW current_position_risk AS
SELECT 
    p.symbol,
    p.side,
    p.size as position_size,
    p.entry_price,
    p.mark_price,
    p.unrealized_pnl,
    p.leverage,
    p.margin_type,
    CASE 
        WHEN p.size != 0 THEN p.unrealized_pnl / (ABS(p.size) * p.entry_price) * 100
        ELSE 0 
    END as unrealized_pnl_percent,
    CASE 
        WHEN p.leverage > 0 THEN (ABS(p.size) * p.mark_price) / p.leverage
        ELSE 0 
    END as margin_used,
    p.updated_at
FROM positions p
WHERE p.size != 0;

-- Daily PnL performance view
CREATE OR REPLACE VIEW daily_pnl_performance AS
SELECT 
    trade_date,
    total_pnl,
    total_pnl / starting_balance * 100 as daily_return_percent,
    CASE WHEN trade_count > 0 THEN win_count::DECIMAL / trade_count * 100 ELSE 0 END as win_rate_percent,
    max_drawdown,
    max_drawdown / starting_balance * 100 as max_drawdown_percent
FROM daily_pnl_summary
ORDER BY trade_date DESC;

-- Recent risk events summary
CREATE OR REPLACE VIEW recent_risk_events AS
SELECT 
    event_type,
    symbol,
    severity,
    COUNT(*) as event_count,
    MAX(created_at) as last_occurrence,
    COUNT(CASE WHEN executed THEN 1 END) as executed_count
FROM risk_events 
WHERE created_at >= NOW() - INTERVAL '24 hours'
GROUP BY event_type, symbol, severity
ORDER BY last_occurrence DESC;

-- Add comments for documentation
COMMENT ON TABLE pnl_snapshots IS 'Real-time PnL snapshots for all positions';
COMMENT ON TABLE risk_thresholds IS 'Configurable risk management thresholds';
COMMENT ON TABLE risk_events IS 'Log of all risk management events and triggers';
COMMENT ON TABLE margin_alerts IS 'Margin-related alerts and notifications';
COMMENT ON TABLE account_equity_history IS 'Historical account equity tracking';
COMMENT ON TABLE daily_pnl_summary IS 'Daily profit and loss summary statistics';
COMMENT ON TABLE position_risk_metrics IS 'Detailed risk metrics for each position';
COMMENT ON TABLE automated_actions_log IS 'Log of all automated risk management actions';

COMMENT ON VIEW current_position_risk IS 'Current positions with calculated risk metrics';
COMMENT ON VIEW daily_pnl_performance IS 'Daily performance metrics and statistics';
COMMENT ON VIEW recent_risk_events IS 'Summary of recent risk events by type and severity';
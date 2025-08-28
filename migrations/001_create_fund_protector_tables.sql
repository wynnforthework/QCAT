-- Fund Protector Database Schema Migration
-- This migration creates all tables needed for the fund protector system

-- Historical returns table for storing daily return data
CREATE TABLE IF NOT EXISTS historical_returns (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL,
    return_value DECIMAL(15,8) NOT NULL,
    portfolio_value DECIMAL(20,8),
    benchmark_return DECIMAL(15,8),
    volatility DECIMAL(10,6),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(date)
);

-- Create index for efficient date-based queries
CREATE INDEX IF NOT EXISTS idx_historical_returns_date ON historical_returns(date DESC);

-- Historical equity table for tracking portfolio value over time
CREATE TABLE IF NOT EXISTS historical_equity (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    equity_value DECIMAL(20,8) NOT NULL,
    available_balance DECIMAL(20,8),
    locked_balance DECIMAL(20,8),
    unrealized_pnl DECIMAL(20,8),
    realized_pnl DECIMAL(20,8),
    total_positions INTEGER DEFAULT 0,
    active_positions INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(timestamp)
);

-- Create index for efficient timestamp-based queries
CREATE INDEX IF NOT EXISTS idx_historical_equity_timestamp ON historical_equity(timestamp DESC);

-- Risk snapshots table for storing calculated risk metrics
CREATE TABLE IF NOT EXISTS risk_snapshots (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    risk_level VARCHAR(20) NOT NULL CHECK (risk_level IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    risk_score DECIMAL(10,6) NOT NULL,
    var_95 DECIMAL(15,8) NOT NULL,
    expected_shortfall DECIMAL(15,8) NOT NULL,
    max_drawdown DECIMAL(10,6) NOT NULL,
    volatility_index DECIMAL(10,6) NOT NULL,
    leverage DECIMAL(10,4) NOT NULL,
    concentration DECIMAL(10,6) NOT NULL,
    portfolio_beta DECIMAL(10,6),
    sharpe_ratio DECIMAL(10,6),
    sortino_ratio DECIMAL(10,6),
    calmar_ratio DECIMAL(10,6),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create index for efficient timestamp and risk level queries
CREATE INDEX IF NOT EXISTS idx_risk_snapshots_timestamp ON risk_snapshots(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_risk_snapshots_risk_level ON risk_snapshots(risk_level, timestamp DESC);

-- Transfer records table for tracking fund transfers
CREATE TABLE IF NOT EXISTS transfer_records (
    id VARCHAR(50) PRIMARY KEY,
    type VARCHAR(30) NOT NULL CHECK (type IN ('PROFIT_TRANSFER', 'EMERGENCY_TRANSFER', 'REBALANCE_TRANSFER', 'MANUAL_TRANSFER')),
    amount DECIMAL(20,8) NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'USDT',
    from_address VARCHAR(100) NOT NULL,
    to_address VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED', 'CANCELLED')),
    trigger_reason VARCHAR(100),
    transaction_hash VARCHAR(100),
    estimated_fee DECIMAL(15,8),
    actual_fee DECIMAL(15,8),
    confirmations INTEGER DEFAULT 0,
    required_confirmations INTEGER DEFAULT 6,
    priority INTEGER DEFAULT 1 CHECK (priority BETWEEN 1 AND 5),
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    executed_at TIMESTAMP,
    completed_at TIMESTAMP
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_transfer_records_status ON transfer_records(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transfer_records_type ON transfer_records(type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transfer_records_created_at ON transfer_records(created_at DESC);

-- Emergency events table for tracking emergency situations
CREATE TABLE IF NOT EXISTS emergency_events (
    id VARCHAR(50) PRIMARY KEY,
    type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    description TEXT NOT NULL,
    trigger_data JSONB,
    response_data JSONB,
    status VARCHAR(20) DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'RESOLVED', 'ACKNOWLEDGED', 'ESCALATED')),
    response_time_ms INTEGER,
    actions_taken TEXT[],
    notifications_sent INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    acknowledged_at TIMESTAMP,
    resolved_at TIMESTAMP,
    escalated_at TIMESTAMP
);

-- Create indexes for emergency event queries
CREATE INDEX IF NOT EXISTS idx_emergency_events_severity ON emergency_events(severity, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_emergency_events_status ON emergency_events(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_emergency_events_type ON emergency_events(type, created_at DESC);

-- Position snapshots table for tracking position changes over time
CREATE TABLE IF NOT EXISTS position_snapshots (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    side VARCHAR(10) NOT NULL CHECK (side IN ('LONG', 'SHORT')),
    size DECIMAL(20,8) NOT NULL,
    notional DECIMAL(20,8) NOT NULL,
    entry_price DECIMAL(15,8) NOT NULL,
    mark_price DECIMAL(15,8) NOT NULL,
    unrealized_pnl DECIMAL(20,8) NOT NULL,
    realized_pnl DECIMAL(20,8) DEFAULT 0,
    leverage INTEGER NOT NULL,
    margin_type VARCHAR(10) CHECK (margin_type IN ('ISOLATED', 'CROSSED')),
    isolated_margin DECIMAL(20,8),
    maintenance_margin DECIMAL(20,8),
    liquidation_price DECIMAL(15,8),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for position snapshot queries
CREATE INDEX IF NOT EXISTS idx_position_snapshots_timestamp ON position_snapshots(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_position_snapshots_symbol ON position_snapshots(symbol, timestamp DESC);

-- Fund status snapshots table for tracking overall fund health
CREATE TABLE IF NOT EXISTS fund_status_snapshots (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    total_balance DECIMAL(20,8) NOT NULL,
    available_balance DECIMAL(20,8) NOT NULL,
    locked_balance DECIMAL(20,8) NOT NULL,
    profit_loss DECIMAL(20,8) NOT NULL,
    daily_pl DECIMAL(20,8) NOT NULL,
    unrealized_pl DECIMAL(20,8) NOT NULL,
    realized_pl DECIMAL(20,8) NOT NULL,
    current_risk DECIMAL(10,6),
    max_risk DECIMAL(10,6),
    var_95 DECIMAL(15,8),
    expected_shortfall DECIMAL(15,8),
    total_positions INTEGER DEFAULT 0,
    active_positions INTEGER DEFAULT 0,
    long_positions INTEGER DEFAULT 0,
    short_positions INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create index for fund status queries
CREATE INDEX IF NOT EXISTS idx_fund_status_snapshots_timestamp ON fund_status_snapshots(timestamp DESC);

-- Circuit breaker events table for tracking circuit breaker activations
CREATE TABLE IF NOT EXISTS circuit_breaker_events (
    id SERIAL PRIMARY KEY,
    trigger_reason VARCHAR(100) NOT NULL,
    loss_ratio DECIMAL(10,6) NOT NULL,
    trigger_count INTEGER NOT NULL,
    cooldown_period_minutes INTEGER NOT NULL,
    triggered_at TIMESTAMP NOT NULL,
    reset_at TIMESTAMP,
    status VARCHAR(20) DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'RESET', 'MANUAL_RESET')),
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create index for circuit breaker events
CREATE INDEX IF NOT EXISTS idx_circuit_breaker_events_triggered_at ON circuit_breaker_events(triggered_at DESC);
CREATE INDEX IF NOT EXISTS idx_circuit_breaker_events_status ON circuit_breaker_events(status, triggered_at DESC);

-- Protection metrics table for tracking system performance
CREATE TABLE IF NOT EXISTS protection_metrics (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    circuit_breaker_triggered BIGINT DEFAULT 0,
    emergency_activations BIGINT DEFAULT 0,
    auto_transfers BIGINT DEFAULT 0,
    manual_interventions BIGINT DEFAULT 0,
    losses_prevented DECIMAL(20,8) DEFAULT 0,
    profits_secured DECIMAL(20,8) DEFAULT 0,
    max_loss_avoided DECIMAL(20,8) DEFAULT 0,
    avg_response_time_ms INTEGER DEFAULT 0,
    protection_accuracy DECIMAL(5,4) DEFAULT 0,
    false_positive_rate DECIMAL(5,4) DEFAULT 0,
    system_uptime_seconds BIGINT DEFAULT 0,
    last_emergency_test TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create index for protection metrics
CREATE INDEX IF NOT EXISTS idx_protection_metrics_timestamp ON protection_metrics(timestamp DESC);

-- Create a view for latest fund status
CREATE OR REPLACE VIEW latest_fund_status AS
SELECT *
FROM fund_status_snapshots
WHERE timestamp = (SELECT MAX(timestamp) FROM fund_status_snapshots);

-- Create a view for recent risk snapshots (last 30 days)
CREATE OR REPLACE VIEW recent_risk_snapshots AS
SELECT *
FROM risk_snapshots
WHERE timestamp >= NOW() - INTERVAL '30 days'
ORDER BY timestamp DESC;

-- Create a view for active emergency events
CREATE OR REPLACE VIEW active_emergency_events AS
SELECT *
FROM emergency_events
WHERE status = 'ACTIVE'
ORDER BY severity DESC, created_at DESC;

-- Create a function to clean up old data
CREATE OR REPLACE FUNCTION cleanup_old_fund_protector_data(retention_days INTEGER DEFAULT 365)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER := 0;
    temp_count INTEGER;
BEGIN
    -- Clean up old historical returns (keep last retention_days)
    DELETE FROM historical_returns 
    WHERE date < CURRENT_DATE - INTERVAL '1 day' * retention_days;
    GET DIAGNOSTICS temp_count = ROW_COUNT;
    deleted_count := deleted_count + temp_count;
    
    -- Clean up old historical equity (keep last retention_days)
    DELETE FROM historical_equity 
    WHERE timestamp < NOW() - INTERVAL '1 day' * retention_days;
    GET DIAGNOSTICS temp_count = ROW_COUNT;
    deleted_count := deleted_count + temp_count;
    
    -- Clean up old risk snapshots (keep last retention_days)
    DELETE FROM risk_snapshots 
    WHERE timestamp < NOW() - INTERVAL '1 day' * retention_days;
    GET DIAGNOSTICS temp_count = ROW_COUNT;
    deleted_count := deleted_count + temp_count;
    
    -- Clean up old position snapshots (keep last retention_days)
    DELETE FROM position_snapshots 
    WHERE timestamp < NOW() - INTERVAL '1 day' * retention_days;
    GET DIAGNOSTICS temp_count = ROW_COUNT;
    deleted_count := deleted_count + temp_count;
    
    -- Clean up old fund status snapshots (keep last retention_days)
    DELETE FROM fund_status_snapshots 
    WHERE timestamp < NOW() - INTERVAL '1 day' * retention_days;
    GET DIAGNOSTICS temp_count = ROW_COUNT;
    deleted_count := deleted_count + temp_count;
    
    -- Keep transfer records and emergency events longer (2 years)
    DELETE FROM transfer_records 
    WHERE created_at < NOW() - INTERVAL '1 day' * (retention_days * 2);
    GET DIAGNOSTICS temp_count = ROW_COUNT;
    deleted_count := deleted_count + temp_count;
    
    DELETE FROM emergency_events 
    WHERE created_at < NOW() - INTERVAL '1 day' * (retention_days * 2);
    GET DIAGNOSTICS temp_count = ROW_COUNT;
    deleted_count := deleted_count + temp_count;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create a function to get portfolio statistics
CREATE OR REPLACE FUNCTION get_portfolio_statistics(days_back INTEGER DEFAULT 30)
RETURNS TABLE (
    avg_daily_return DECIMAL(10,6),
    volatility DECIMAL(10,6),
    max_drawdown DECIMAL(10,6),
    sharpe_ratio DECIMAL(10,6),
    total_return DECIMAL(10,6),
    win_rate DECIMAL(5,4)
) AS $$
BEGIN
    RETURN QUERY
    WITH daily_stats AS (
        SELECT 
            return_value,
            equity_value,
            LAG(equity_value) OVER (ORDER BY date) as prev_equity
        FROM historical_returns hr
        JOIN historical_equity he ON DATE(he.timestamp) = hr.date
        WHERE hr.date >= CURRENT_DATE - INTERVAL '1 day' * days_back
        ORDER BY hr.date
    ),
    returns_stats AS (
        SELECT 
            AVG(return_value) as avg_ret,
            STDDEV(return_value) as vol,
            COUNT(CASE WHEN return_value > 0 THEN 1 END)::DECIMAL / COUNT(*)::DECIMAL as win_rt,
            (MAX(equity_value) - MIN(equity_value)) / MIN(equity_value) as tot_ret
        FROM daily_stats
        WHERE prev_equity IS NOT NULL
    )
    SELECT 
        avg_ret,
        vol,
        0.0::DECIMAL(10,6), -- max_drawdown placeholder
        CASE WHEN vol > 0 THEN avg_ret / vol ELSE 0 END,
        tot_ret,
        win_rt
    FROM returns_stats;
END;
$$ LANGUAGE plpgsql;

-- Grant necessary permissions (adjust as needed for your user)
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO qcat_user;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO qcat_user;
-- GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO qcat_user;

-- Insert initial data for testing
INSERT INTO protection_metrics (timestamp) VALUES (NOW()) ON CONFLICT DO NOTHING;

COMMENT ON TABLE historical_returns IS 'Stores daily return data for portfolio performance analysis';
COMMENT ON TABLE historical_equity IS 'Tracks portfolio equity value over time';
COMMENT ON TABLE risk_snapshots IS 'Stores calculated risk metrics at regular intervals';
COMMENT ON TABLE transfer_records IS 'Tracks all fund transfer operations';
COMMENT ON TABLE emergency_events IS 'Records emergency events and responses';
COMMENT ON TABLE position_snapshots IS 'Historical position data for analysis';
COMMENT ON TABLE fund_status_snapshots IS 'Overall fund health snapshots';
COMMENT ON TABLE circuit_breaker_events IS 'Circuit breaker activation history';
COMMENT ON TABLE protection_metrics IS 'System performance and protection metrics';
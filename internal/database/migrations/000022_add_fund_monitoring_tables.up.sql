-- Migration: Add fund monitoring and protection tables
-- Version: 000022
-- Description: Create missing fund_monitoring_rules and fund_protection_history tables

-- 1. Create fund_monitoring_rules table
CREATE TABLE IF NOT EXISTS fund_monitoring_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    exchange VARCHAR(50) NOT NULL, -- 'binance', 'okx', 'bybit', 'hot_wallet', 'cold_wallet'
    rule_type VARCHAR(50) NOT NULL, -- 'balance_threshold', 'withdrawal_limit', 'daily_limit'
    rule_name VARCHAR(100) NOT NULL,
    threshold_value DECIMAL(30,10) NOT NULL DEFAULT 0,
    threshold_currency VARCHAR(10) NOT NULL DEFAULT 'USDT',
    warning_threshold DECIMAL(30,10),
    critical_threshold DECIMAL(30,10),
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    notification_channels TEXT[], -- Array of notification channels
    last_triggered TIMESTAMP WITH TIME ZONE,
    trigger_count INTEGER NOT NULL DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(exchange, rule_type, rule_name)
);

-- 2. Create fund_protection_history table
CREATE TABLE IF NOT EXISTS fund_protection_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    protocol_type VARCHAR(50) NOT NULL, -- 'emergency_stop', 'position_limit', 'withdrawal_freeze'
    exchange VARCHAR(50) NOT NULL,
    trigger_reason VARCHAR(200) NOT NULL,
    action_taken VARCHAR(100) NOT NULL,
    affected_amount DECIMAL(30,10) DEFAULT 0,
    affected_currency VARCHAR(10) DEFAULT 'USDT',
    risk_level VARCHAR(20) NOT NULL, -- 'low', 'medium', 'high', 'critical'
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'active', 'resolved', 'cancelled'
    triggered_by VARCHAR(100), -- User or system that triggered the protection
    resolution_notes TEXT,
    metadata JSONB DEFAULT '{}',
    triggered_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 3. Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_fund_monitoring_rules_exchange ON fund_monitoring_rules(exchange);
CREATE INDEX IF NOT EXISTS idx_fund_monitoring_rules_enabled ON fund_monitoring_rules(is_enabled);
CREATE INDEX IF NOT EXISTS idx_fund_monitoring_rules_type ON fund_monitoring_rules(rule_type);
CREATE INDEX IF NOT EXISTS idx_fund_monitoring_rules_updated ON fund_monitoring_rules(updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_fund_protection_history_exchange ON fund_protection_history(exchange);
CREATE INDEX IF NOT EXISTS idx_fund_protection_history_status ON fund_protection_history(status);
CREATE INDEX IF NOT EXISTS idx_fund_protection_history_risk_level ON fund_protection_history(risk_level);
CREATE INDEX IF NOT EXISTS idx_fund_protection_history_triggered ON fund_protection_history(triggered_at DESC);
CREATE INDEX IF NOT EXISTS idx_fund_protection_history_protocol ON fund_protection_history(protocol_type);

-- 4. Create updated_at triggers
DO $$
BEGIN
    CREATE TRIGGER update_fund_monitoring_rules_updated_at
        BEFORE UPDATE ON fund_monitoring_rules
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TRIGGER update_fund_protection_history_updated_at
        BEFORE UPDATE ON fund_protection_history
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- 5. Insert default monitoring rules
INSERT INTO fund_monitoring_rules (
    exchange, rule_type, rule_name, threshold_value, threshold_currency, 
    warning_threshold, critical_threshold, notification_channels
) VALUES 
    ('binance', 'balance_threshold', 'minimum_balance', 1000.00, 'USDT', 500.00, 100.00, ARRAY['email', 'slack']),
    ('okx', 'balance_threshold', 'minimum_balance', 1000.00, 'USDT', 500.00, 100.00, ARRAY['email', 'slack']),
    ('bybit', 'balance_threshold', 'minimum_balance', 1000.00, 'USDT', 500.00, 100.00, ARRAY['email', 'slack']),
    ('hot_wallet', 'balance_threshold', 'minimum_balance', 10000.00, 'USDT', 5000.00, 1000.00, ARRAY['email', 'slack', 'sms']),
    ('cold_wallet', 'balance_threshold', 'minimum_balance', 100000.00, 'USDT', 50000.00, 10000.00, ARRAY['email', 'slack', 'sms']),
    ('binance', 'withdrawal_limit', 'daily_withdrawal', 50000.00, 'USDT', 40000.00, 45000.00, ARRAY['email']),
    ('okx', 'withdrawal_limit', 'daily_withdrawal', 50000.00, 'USDT', 40000.00, 45000.00, ARRAY['email']),
    ('bybit', 'withdrawal_limit', 'daily_withdrawal', 50000.00, 'USDT', 40000.00, 45000.00, ARRAY['email'])
ON CONFLICT (exchange, rule_type, rule_name) DO NOTHING;

-- 6. Verify tables were created
DO $$
BEGIN
    RAISE NOTICE 'Migration 000022 completed successfully';

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'fund_monitoring_rules') THEN
        RAISE NOTICE 'fund_monitoring_rules table: EXISTS';
    ELSE
        RAISE NOTICE 'fund_monitoring_rules table: MISSING';
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'fund_protection_history') THEN
        RAISE NOTICE 'fund_protection_history table: EXISTS';
    ELSE
        RAISE NOTICE 'fund_protection_history table: MISSING';
    END IF;
END $$;

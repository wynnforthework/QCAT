-- Migration to fix fund_protection_history exchange field constraint
-- This fixes the "null value in column exchange violates not-null constraint" error

BEGIN;

-- Fix fund_protection_history table exchange field constraint
DO $$
BEGIN
    -- Check if fund_protection_history table exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'fund_protection_history') THEN
        
        -- Remove NOT NULL constraint from exchange field if it exists
        IF EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'fund_protection_history' 
            AND column_name = 'exchange' 
            AND is_nullable = 'NO'
        ) THEN
            ALTER TABLE fund_protection_history ALTER COLUMN exchange DROP NOT NULL;
            RAISE NOTICE 'Removed NOT NULL constraint from exchange column in fund_protection_history';
        END IF;
        
        -- Set default value for exchange field
        IF EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'fund_protection_history' AND column_name = 'exchange') THEN
            ALTER TABLE fund_protection_history ALTER COLUMN exchange SET DEFAULT 'default';
            RAISE NOTICE 'Set default value for exchange column in fund_protection_history';
        ELSE
            -- Add exchange field if it doesn't exist
            ALTER TABLE fund_protection_history ADD COLUMN exchange VARCHAR(50) DEFAULT 'default';
            RAISE NOTICE 'Added exchange column to fund_protection_history table';
        END IF;

        -- Update existing NULL exchange values with default
        UPDATE fund_protection_history 
        SET exchange = 'default' 
        WHERE exchange IS NULL;
        
        -- Add other missing fields that might be needed
        
        -- Add trigger_reason field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'fund_protection_history' AND column_name = 'trigger_reason') THEN
            ALTER TABLE fund_protection_history ADD COLUMN trigger_reason VARCHAR(200) DEFAULT 'system_triggered';
            RAISE NOTICE 'Added trigger_reason column to fund_protection_history table';
        END IF;

        -- Add action_taken field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'fund_protection_history' AND column_name = 'action_taken') THEN
            ALTER TABLE fund_protection_history ADD COLUMN action_taken VARCHAR(100) DEFAULT 'protection_activated';
            RAISE NOTICE 'Added action_taken column to fund_protection_history table';
        END IF;

        -- Add affected_amount field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'fund_protection_history' AND column_name = 'affected_amount') THEN
            ALTER TABLE fund_protection_history ADD COLUMN affected_amount DECIMAL(30,10) DEFAULT 0;
            RAISE NOTICE 'Added affected_amount column to fund_protection_history table';
        END IF;

        -- Add affected_currency field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'fund_protection_history' AND column_name = 'affected_currency') THEN
            ALTER TABLE fund_protection_history ADD COLUMN affected_currency VARCHAR(10) DEFAULT 'USDT';
            RAISE NOTICE 'Added affected_currency column to fund_protection_history table';
        END IF;

        -- Add risk_level field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'fund_protection_history' AND column_name = 'risk_level') THEN
            ALTER TABLE fund_protection_history ADD COLUMN risk_level VARCHAR(20) DEFAULT 'medium';
            RAISE NOTICE 'Added risk_level column to fund_protection_history table';
        END IF;

        -- Add status field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'fund_protection_history' AND column_name = 'status') THEN
            ALTER TABLE fund_protection_history ADD COLUMN status VARCHAR(20) DEFAULT 'active';
            RAISE NOTICE 'Added status column to fund_protection_history table';
        END IF;

        -- Add triggered_by field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'fund_protection_history' AND column_name = 'triggered_by') THEN
            ALTER TABLE fund_protection_history ADD COLUMN triggered_by VARCHAR(100) DEFAULT 'system';
            RAISE NOTICE 'Added triggered_by column to fund_protection_history table';
        END IF;

        -- Add triggered_at field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'fund_protection_history' AND column_name = 'triggered_at') THEN
            ALTER TABLE fund_protection_history ADD COLUMN triggered_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();
            RAISE NOTICE 'Added triggered_at column to fund_protection_history table';
        END IF;

        -- Add resolution_notes field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'fund_protection_history' AND column_name = 'resolution_notes') THEN
            ALTER TABLE fund_protection_history ADD COLUMN resolution_notes TEXT;
            RAISE NOTICE 'Added resolution_notes column to fund_protection_history table';
        END IF;

        -- Add metadata field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'fund_protection_history' AND column_name = 'metadata') THEN
            ALTER TABLE fund_protection_history ADD COLUMN metadata JSONB DEFAULT '{}';
            RAISE NOTICE 'Added metadata column to fund_protection_history table';
        END IF;

        -- Ensure protocol_type has a default value
        IF EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'fund_protection_history' AND column_name = 'protocol_type') THEN
            ALTER TABLE fund_protection_history ALTER COLUMN protocol_type SET DEFAULT 'standard';
            UPDATE fund_protection_history SET protocol_type = 'standard' WHERE protocol_type IS NULL;
        END IF;

        -- Update existing records with missing required values
        UPDATE fund_protection_history 
        SET 
            exchange = COALESCE(exchange, 'default'),
            trigger_reason = COALESCE(trigger_reason, 'system_triggered'),
            action_taken = COALESCE(action_taken, 'protection_activated'),
            risk_level = COALESCE(risk_level, 'medium'),
            status = COALESCE(status, 'active'),
            triggered_by = COALESCE(triggered_by, 'system'),
            triggered_at = COALESCE(triggered_at, created_at, NOW()),
            protocol_type = COALESCE(protocol_type, 'standard'),
            affected_currency = COALESCE(affected_currency, 'USDT'),
            metadata = COALESCE(metadata, '{}')
        WHERE 
            exchange IS NULL 
            OR trigger_reason IS NULL 
            OR action_taken IS NULL 
            OR risk_level IS NULL 
            OR status IS NULL 
            OR triggered_by IS NULL 
            OR triggered_at IS NULL 
            OR protocol_type IS NULL 
            OR affected_currency IS NULL 
            OR metadata IS NULL;

    ELSE
        -- Create fund_protection_history table if it doesn't exist with proper structure
        CREATE TABLE fund_protection_history (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            protocol_type VARCHAR(50) DEFAULT 'standard',
            exchange VARCHAR(50) DEFAULT 'default',
            trigger_reason VARCHAR(200) DEFAULT 'system_triggered',
            action_taken VARCHAR(100) DEFAULT 'protection_activated',
            affected_amount DECIMAL(30,10) DEFAULT 0,
            affected_currency VARCHAR(10) DEFAULT 'USDT',
            risk_level VARCHAR(20) DEFAULT 'medium',
            status VARCHAR(20) DEFAULT 'active',
            triggered_by VARCHAR(100) DEFAULT 'system',
            triggered_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            resolution_notes TEXT,
            metadata JSONB DEFAULT '{}',
            target_distribution JSONB DEFAULT '{}',
            risk_parameters JSONB DEFAULT '{}',
            transfer_count INTEGER DEFAULT 0,
            success_rate DECIMAL(5,4) DEFAULT 0.0000,
            expected_risk_reduction DECIMAL(10,6) DEFAULT 0.000000,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );
        RAISE NOTICE 'Created fund_protection_history table with all required fields';
    END IF;
END $$;

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_fund_protection_history_exchange ON fund_protection_history(exchange);
CREATE INDEX IF NOT EXISTS idx_fund_protection_history_status ON fund_protection_history(status);
CREATE INDEX IF NOT EXISTS idx_fund_protection_history_risk_level ON fund_protection_history(risk_level);
CREATE INDEX IF NOT EXISTS idx_fund_protection_history_triggered_at ON fund_protection_history(triggered_at DESC);
CREATE INDEX IF NOT EXISTS idx_fund_protection_history_protocol_type ON fund_protection_history(protocol_type);
CREATE INDEX IF NOT EXISTS idx_fund_protection_history_created_at ON fund_protection_history(created_at DESC);

-- Insert sample data if the table is empty
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM fund_protection_history LIMIT 1) THEN
        INSERT INTO fund_protection_history (
            protocol_type, exchange, trigger_reason, action_taken, affected_amount, 
            affected_currency, risk_level, status, triggered_by, metadata
        ) VALUES 
        (
            'emergency_stop', 'binance', 'high_volatility_detected', 'trading_halted', 
            50000.0, 'USDT', 'high', 'resolved', 'risk_monitor', 
            '{"volatility": 0.25, "threshold": 0.20}'
        ),
        (
            'position_limit', 'okx', 'position_size_exceeded', 'position_reduced', 
            25000.0, 'USDT', 'medium', 'active', 'position_manager', 
            '{"original_size": 100000, "new_size": 75000}'
        ),
        (
            'withdrawal_freeze', 'default', 'suspicious_activity', 'withdrawals_frozen', 
            0.0, 'USDT', 'critical', 'active', 'security_monitor', 
            '{"alert_type": "unusual_withdrawal_pattern"}'
        );
        
        RAISE NOTICE 'Added sample fund protection history records';
    END IF;
END $$;

COMMIT;

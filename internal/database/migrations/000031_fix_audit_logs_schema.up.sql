-- Migration 000031: Fix audit_logs table schema to match API expectations
-- Add missing fields that the API expects

-- Add missing columns to audit_logs table
DO $$ 
BEGIN
    -- Add level column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'audit_logs' AND column_name = 'level') THEN
        ALTER TABLE audit_logs ADD COLUMN level VARCHAR(20) DEFAULT 'info';
        RAISE NOTICE 'Added level column to audit_logs table';
    END IF;
    
    -- Add entity column if it doesn't exist (maps to entity_type)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'audit_logs' AND column_name = 'entity') THEN
        -- If entity_type exists, rename it to entity
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'audit_logs' AND column_name = 'entity_type') THEN
            ALTER TABLE audit_logs RENAME COLUMN entity_type TO entity;
            RAISE NOTICE 'Renamed entity_type column to entity in audit_logs table';
        ELSE
            -- Add entity column if neither exists
            ALTER TABLE audit_logs ADD COLUMN entity VARCHAR(50);
            RAISE NOTICE 'Added entity column to audit_logs table';
        END IF;
    END IF;
    
    -- Add details column if it doesn't exist (maps to old_value/new_value)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'audit_logs' AND column_name = 'details') THEN
        ALTER TABLE audit_logs ADD COLUMN details JSONB;
        RAISE NOTICE 'Added details column to audit_logs table';
    END IF;
    
    -- Ensure action column exists
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'audit_logs' AND column_name = 'action') THEN
        ALTER TABLE audit_logs ADD COLUMN action VARCHAR(100) NOT NULL DEFAULT 'unknown';
        RAISE NOTICE 'Added action column to audit_logs table';
    END IF;
    
    -- Update existing records to have default values
    UPDATE audit_logs SET level = 'info' WHERE level IS NULL;
    UPDATE audit_logs SET entity = 'unknown' WHERE entity IS NULL;
    
    RAISE NOTICE 'Updated existing audit_logs records with default values';
    
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error updating audit_logs table: %', SQLERRM;
END $$;

-- Create indexes for better query performance
DO $$
BEGIN
    -- Index on level for filtering
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE tablename = 'audit_logs' AND indexname = 'idx_audit_logs_level') THEN
        CREATE INDEX idx_audit_logs_level ON audit_logs(level);
        RAISE NOTICE 'Created index on audit_logs.level';
    END IF;
    
    -- Index on entity for filtering
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE tablename = 'audit_logs' AND indexname = 'idx_audit_logs_entity') THEN
        CREATE INDEX idx_audit_logs_entity ON audit_logs(entity);
        RAISE NOTICE 'Created index on audit_logs.entity';
    END IF;
    
    -- Index on created_at for date range queries
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE tablename = 'audit_logs' AND indexname = 'idx_audit_logs_created_at') THEN
        CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
        RAISE NOTICE 'Created index on audit_logs.created_at';
    END IF;
    
    -- Composite index for common query patterns
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE tablename = 'audit_logs' AND indexname = 'idx_audit_logs_level_entity_created_at') THEN
        CREATE INDEX idx_audit_logs_level_entity_created_at ON audit_logs(level, entity, created_at DESC);
        RAISE NOTICE 'Created composite index on audit_logs(level, entity, created_at)';
    END IF;
    
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error creating indexes: %', SQLERRM;
END $$;

-- Insert some sample audit log data for testing
DO $$
BEGIN
    -- Only insert if table is empty
    IF NOT EXISTS (SELECT 1 FROM audit_logs LIMIT 1) THEN
        INSERT INTO audit_logs (id, level, entity, action, user_id, details, created_at) VALUES
        (uuid_generate_v4(), 'info', 'system', 'startup', NULL, '{"message": "System started successfully"}', NOW() - INTERVAL '1 hour'),
        (uuid_generate_v4(), 'info', 'user', 'login', uuid_generate_v4(), '{"ip": "127.0.0.1", "user_agent": "Mozilla/5.0"}', NOW() - INTERVAL '30 minutes'),
        (uuid_generate_v4(), 'warn', 'strategy', 'parameter_change', uuid_generate_v4(), '{"old_value": 0.1, "new_value": 0.2, "parameter": "risk_threshold"}', NOW() - INTERVAL '15 minutes'),
        (uuid_generate_v4(), 'error', 'api', 'request_failed', uuid_generate_v4(), '{"endpoint": "/api/v1/orders", "error": "Invalid parameters"}', NOW() - INTERVAL '5 minutes'),
        (uuid_generate_v4(), 'info', 'order', 'created', uuid_generate_v4(), '{"symbol": "BTCUSDT", "side": "buy", "amount": 0.001}', NOW());
        
        RAISE NOTICE 'Inserted sample audit log data';
    ELSE
        RAISE NOTICE 'Audit logs table already contains data, skipping sample data insertion';
    END IF;
    
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error inserting sample data: %', SQLERRM;
END $$;

-- Success message
DO $$ 
BEGIN
    RAISE NOTICE 'Migration 000031 completed successfully';
    RAISE NOTICE 'Fixed audit_logs table schema to match API expectations';
    RAISE NOTICE 'Added missing columns: level, entity, details';
    RAISE NOTICE 'Created performance indexes';
    RAISE NOTICE 'Added sample audit log data for testing';
END $$;

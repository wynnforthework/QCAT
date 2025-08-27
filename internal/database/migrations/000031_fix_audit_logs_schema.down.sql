-- Migration 000031 Down: Revert audit_logs table schema changes

-- Drop indexes
DO $$
BEGIN
    -- Drop composite index
    IF EXISTS (SELECT 1 FROM pg_indexes WHERE tablename = 'audit_logs' AND indexname = 'idx_audit_logs_level_entity_created_at') THEN
        DROP INDEX idx_audit_logs_level_entity_created_at;
        RAISE NOTICE 'Dropped composite index idx_audit_logs_level_entity_created_at';
    END IF;
    
    -- Drop created_at index
    IF EXISTS (SELECT 1 FROM pg_indexes WHERE tablename = 'audit_logs' AND indexname = 'idx_audit_logs_created_at') THEN
        DROP INDEX idx_audit_logs_created_at;
        RAISE NOTICE 'Dropped index idx_audit_logs_created_at';
    END IF;
    
    -- Drop entity index
    IF EXISTS (SELECT 1 FROM pg_indexes WHERE tablename = 'audit_logs' AND indexname = 'idx_audit_logs_entity') THEN
        DROP INDEX idx_audit_logs_entity;
        RAISE NOTICE 'Dropped index idx_audit_logs_entity';
    END IF;
    
    -- Drop level index
    IF EXISTS (SELECT 1 FROM pg_indexes WHERE tablename = 'audit_logs' AND indexname = 'idx_audit_logs_level') THEN
        DROP INDEX idx_audit_logs_level;
        RAISE NOTICE 'Dropped index idx_audit_logs_level';
    END IF;
    
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error dropping indexes: %', SQLERRM;
END $$;

-- Remove added columns
DO $$ 
BEGIN
    -- Remove level column if it exists
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'audit_logs' AND column_name = 'level') THEN
        ALTER TABLE audit_logs DROP COLUMN level;
        RAISE NOTICE 'Removed level column from audit_logs table';
    END IF;
    
    -- Rename entity back to entity_type if needed
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'audit_logs' AND column_name = 'entity') THEN
        -- Check if this was renamed from entity_type
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'audit_logs' AND column_name = 'entity_type') THEN
            ALTER TABLE audit_logs RENAME COLUMN entity TO entity_type;
            RAISE NOTICE 'Renamed entity column back to entity_type in audit_logs table';
        ELSE
            -- If entity_type already exists, just drop entity
            ALTER TABLE audit_logs DROP COLUMN entity;
            RAISE NOTICE 'Removed entity column from audit_logs table';
        END IF;
    END IF;
    
    -- Remove details column if it was added by this migration
    -- Note: We need to be careful here as details might have existed before
    -- For safety, we'll keep it but could remove if certain it was added by this migration
    -- IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'audit_logs' AND column_name = 'details') THEN
    --     ALTER TABLE audit_logs DROP COLUMN details;
    --     RAISE NOTICE 'Removed details column from audit_logs table';
    -- END IF;
    
    RAISE NOTICE 'Reverted audit_logs table schema changes';
    
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error reverting audit_logs table: %', SQLERRM;
END $$;

-- Clear sample data that was inserted by the up migration
DO $$
BEGIN
    -- Remove sample data based on specific details content
    DELETE FROM audit_logs WHERE details::text LIKE '%System started successfully%';
    DELETE FROM audit_logs WHERE details::text LIKE '%Mozilla/5.0%' AND entity = 'user';
    DELETE FROM audit_logs WHERE details::text LIKE '%risk_threshold%';
    DELETE FROM audit_logs WHERE details::text LIKE '%Invalid parameters%';
    DELETE FROM audit_logs WHERE details::text LIKE '%BTCUSDT%' AND entity = 'order';
    
    RAISE NOTICE 'Removed sample audit log data';
    
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error removing sample data: %', SQLERRM;
END $$;

-- Success message
DO $$ 
BEGIN
    RAISE NOTICE 'Migration 000031 rollback completed successfully';
    RAISE NOTICE 'Reverted audit_logs table schema changes';
    RAISE NOTICE 'Removed added indexes and columns';
    RAISE NOTICE 'Cleaned up sample data';
END $$;

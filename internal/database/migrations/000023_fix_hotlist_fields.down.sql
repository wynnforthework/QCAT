-- Migration 000022 rollback: Remove hotlist table fields

BEGIN;

-- Drop trigger
DROP TRIGGER IF EXISTS update_hotlist_last_updated_trigger ON hotlist;
DROP FUNCTION IF EXISTS update_hotlist_last_updated();

-- Drop indexes
DROP INDEX IF EXISTS idx_hotlist_last_scanned;
DROP INDEX IF EXISTS idx_hotlist_last_updated;
DROP INDEX IF EXISTS idx_hotlist_is_enabled;

-- Remove added columns from hotlist table
DO $$ 
BEGIN
    -- Remove last_scanned field if it exists
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hotlist' AND column_name = 'last_scanned') THEN
        ALTER TABLE hotlist DROP COLUMN last_scanned;
    END IF;
    
    -- Remove is_enabled field if it exists
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hotlist' AND column_name = 'is_enabled') THEN
        ALTER TABLE hotlist DROP COLUMN is_enabled;
    END IF;
    
    -- Remove metrics field if it exists
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hotlist' AND column_name = 'metrics') THEN
        ALTER TABLE hotlist DROP COLUMN metrics;
    END IF;
    
    -- Rename last_updated back to updated_at if needed
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hotlist' AND column_name = 'last_updated') THEN
        ALTER TABLE hotlist RENAME COLUMN last_updated TO updated_at;
    END IF;
END $$;

-- Remove added columns from other tables
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'funding_rates') THEN
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'funding_rates' AND column_name = 'last_updated') THEN
            ALTER TABLE funding_rates DROP COLUMN last_updated;
        END IF;
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'strategy_performance') THEN
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_performance' AND column_name = 'last_updated') THEN
            ALTER TABLE strategy_performance DROP COLUMN last_updated;
        END IF;
    END IF;
END $$;

COMMIT;

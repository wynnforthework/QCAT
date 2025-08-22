-- Migration 000022: Fix hotlist table missing fields
-- This migration adds missing fields to the hotlist table

BEGIN;

-- Add missing fields to hotlist table
DO $$ 
BEGIN
    -- Add last_scanned field if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hotlist' AND column_name = 'last_scanned') THEN
        ALTER TABLE hotlist ADD COLUMN last_scanned TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;
        RAISE NOTICE 'Added last_scanned column to hotlist table';
    END IF;
    
    -- Add last_updated field if it doesn't exist (rename updated_at to last_updated for consistency)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hotlist' AND column_name = 'last_updated') THEN
        -- If updated_at exists, rename it to last_updated
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hotlist' AND column_name = 'updated_at') THEN
            ALTER TABLE hotlist RENAME COLUMN updated_at TO last_updated;
            RAISE NOTICE 'Renamed updated_at to last_updated in hotlist table';
        ELSE
            -- Otherwise, add last_updated field
            ALTER TABLE hotlist ADD COLUMN last_updated TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;
            RAISE NOTICE 'Added last_updated column to hotlist table';
        END IF;
    END IF;
    
    -- Add is_enabled field if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hotlist' AND column_name = 'is_enabled') THEN
        ALTER TABLE hotlist ADD COLUMN is_enabled BOOLEAN DEFAULT true;
        RAISE NOTICE 'Added is_enabled column to hotlist table';
    END IF;
    
    -- Add metrics field if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hotlist' AND column_name = 'metrics') THEN
        ALTER TABLE hotlist ADD COLUMN metrics JSONB DEFAULT '{}';
        RAISE NOTICE 'Added metrics column to hotlist table';
    END IF;
    
    -- Update existing records to have default values
    UPDATE hotlist SET 
        last_scanned = COALESCE(last_scanned, created_at),
        last_updated = COALESCE(last_updated, created_at),
        is_enabled = COALESCE(is_enabled, true),
        metrics = COALESCE(metrics, '{}')
    WHERE last_scanned IS NULL OR last_updated IS NULL OR is_enabled IS NULL OR metrics IS NULL;
    
END $$;

-- Add missing fields to funding_rates table if it exists
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'funding_rates') THEN
        -- Add last_updated field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'funding_rates' AND column_name = 'last_updated') THEN
            ALTER TABLE funding_rates ADD COLUMN last_updated TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;
            RAISE NOTICE 'Added last_updated column to funding_rates table';
        END IF;
    END IF;
END $$;

-- Add missing fields to strategy_performance table if it exists
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'strategy_performance') THEN
        -- Add last_updated field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_performance' AND column_name = 'last_updated') THEN
            ALTER TABLE strategy_performance ADD COLUMN last_updated TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;
            RAISE NOTICE 'Added last_updated column to strategy_performance table';
        END IF;
    END IF;
END $$;

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_hotlist_last_scanned ON hotlist(last_scanned);
CREATE INDEX IF NOT EXISTS idx_hotlist_last_updated ON hotlist(last_updated);
CREATE INDEX IF NOT EXISTS idx_hotlist_is_enabled ON hotlist(is_enabled);

-- Create trigger to automatically update last_updated
CREATE OR REPLACE FUNCTION update_hotlist_last_updated()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_updated = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Drop trigger if exists and recreate
DROP TRIGGER IF EXISTS update_hotlist_last_updated_trigger ON hotlist;
CREATE TRIGGER update_hotlist_last_updated_trigger
    BEFORE UPDATE ON hotlist
    FOR EACH ROW EXECUTE FUNCTION update_hotlist_last_updated();

COMMIT;

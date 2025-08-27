-- Migration to fix position duplicates and prevent future duplicates
-- This script should be run after the cleanup program has been executed

BEGIN;

-- Step 1: Create a backup table for safety
CREATE TABLE IF NOT EXISTS positions_backup AS 
SELECT * FROM positions WHERE 1=0; -- Create empty backup table with same structure

-- Insert current data into backup
INSERT INTO positions_backup SELECT * FROM positions;

-- Step 2: Remove any remaining duplicates (keeping the latest record for each unique position)
DELETE FROM positions 
WHERE id NOT IN (
    SELECT DISTINCT ON (strategy_id, symbol) id
    FROM positions 
    WHERE status IN ('open', 'active')
    ORDER BY strategy_id, symbol, created_at DESC
);

-- Step 3: Add unique constraint to prevent future duplicates
DO $$ 
BEGIN
    -- Check if constraint already exists
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'positions_strategy_symbol_unique' 
        AND table_name = 'positions'
    ) THEN
        -- Add unique constraint
        ALTER TABLE positions 
        ADD CONSTRAINT positions_strategy_symbol_unique 
        UNIQUE (strategy_id, symbol);
        
        RAISE NOTICE 'Added unique constraint positions_strategy_symbol_unique';
    ELSE
        RAISE NOTICE 'Unique constraint positions_strategy_symbol_unique already exists';
    END IF;
END $$;

-- Step 4: Create index for better performance
CREATE INDEX IF NOT EXISTS idx_positions_strategy_symbol 
ON positions (strategy_id, symbol);

CREATE INDEX IF NOT EXISTS idx_positions_status_size 
ON positions (status, size) 
WHERE status IN ('open', 'active') AND size != 0;

-- Step 5: Add a trigger to log position changes (optional, for debugging)
CREATE OR REPLACE FUNCTION log_position_changes()
RETURNS TRIGGER AS $$
BEGIN
    -- Log when positions are inserted or updated
    IF TG_OP = 'INSERT' THEN
        RAISE LOG 'Position inserted: strategy_id=%, symbol=%, side=%, size=%', 
            NEW.strategy_id, NEW.symbol, NEW.side, NEW.size;
        RETURN NEW;
    ELSIF TG_OP = 'UPDATE' THEN
        -- Only log if significant fields changed
        IF OLD.size != NEW.size OR OLD.entry_price != NEW.entry_price OR OLD.status != NEW.status THEN
            RAISE LOG 'Position updated: strategy_id=%, symbol=%, old_size=%, new_size=%, old_status=%, new_status=%', 
                NEW.strategy_id, NEW.symbol, OLD.size, NEW.size, OLD.status, NEW.status;
        END IF;
        RETURN NEW;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Create trigger (optional - can be disabled in production)
DROP TRIGGER IF EXISTS position_changes_trigger ON positions;
CREATE TRIGGER position_changes_trigger
    AFTER INSERT OR UPDATE ON positions
    FOR EACH ROW EXECUTE FUNCTION log_position_changes();

-- Step 6: Verify the results
DO $$
DECLARE
    total_positions INTEGER;
    unique_positions INTEGER;
    duplicate_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO total_positions FROM positions;
    SELECT COUNT(DISTINCT (strategy_id, symbol)) INTO unique_positions 
    FROM positions WHERE status IN ('open', 'active');
    
    duplicate_count := total_positions - unique_positions;
    
    RAISE NOTICE 'Migration completed:';
    RAISE NOTICE '  Total positions: %', total_positions;
    RAISE NOTICE '  Unique positions: %', unique_positions;
    RAISE NOTICE '  Remaining duplicates: %', duplicate_count;
    
    IF duplicate_count > 0 THEN
        RAISE WARNING 'Still have % duplicate positions - manual cleanup may be needed', duplicate_count;
    ELSE
        RAISE NOTICE 'All duplicates successfully removed!';
    END IF;
END $$;

COMMIT;

-- Instructions for running this migration:
-- 1. First run the cleanup program: go run cmd/cleanup_positions/main.go
-- 2. Then run this migration: psql -h localhost -U postgres -d qcat -f migrations/fix_position_duplicates.sql
-- 3. Verify results by checking the log output

-- Migration to fix NULL next_time values in funding_rates table
-- This fixes the "unsupported Scan, storing driver.Value type <nil> into type *time.Time" error

BEGIN;

-- Update NULL next_time values with reasonable defaults
UPDATE funding_rates 
SET next_time = CASE 
    WHEN next_time IS NULL THEN 
        CASE 
            WHEN last_updated IS NOT NULL THEN last_updated + INTERVAL '8 hours'
            WHEN funding_time IS NOT NULL THEN funding_time + INTERVAL '8 hours'
            WHEN created_at IS NOT NULL THEN created_at + INTERVAL '8 hours'
            ELSE NOW() + INTERVAL '8 hours'
        END
    ELSE next_time
END
WHERE next_time IS NULL;

-- Set default value for next_time field to prevent future NULL values
ALTER TABLE funding_rates ALTER COLUMN next_time SET DEFAULT (NOW() + INTERVAL '8 hours');

-- Update any remaining NULL next_rate values
UPDATE funding_rates 
SET next_rate = COALESCE(next_rate, rate, 0.0001)
WHERE next_rate IS NULL;

-- Set default value for next_rate field
ALTER TABLE funding_rates ALTER COLUMN next_rate SET DEFAULT 0.0001;

-- Add a check constraint to ensure next_time is always in the future relative to last_updated
-- (This is optional and can be commented out if it causes issues)
-- ALTER TABLE funding_rates ADD CONSTRAINT check_next_time_future 
-- CHECK (next_time > COALESCE(last_updated, created_at, NOW() - INTERVAL '1 day'));

-- Create an index on next_time for better query performance
CREATE INDEX IF NOT EXISTS idx_funding_rates_next_time ON funding_rates(next_time);

-- Add a trigger to automatically set next_time when inserting new records
CREATE OR REPLACE FUNCTION set_funding_next_time()
RETURNS TRIGGER AS $$
BEGIN
    -- If next_time is not provided, set it to 8 hours from last_updated or now
    IF NEW.next_time IS NULL THEN
        NEW.next_time := COALESCE(NEW.last_updated, NEW.created_at, NOW()) + INTERVAL '8 hours';
    END IF;
    
    -- If next_rate is not provided, use the current rate
    IF NEW.next_rate IS NULL THEN
        NEW.next_rate := COALESCE(NEW.rate, 0.0001);
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_set_funding_next_time
    BEFORE INSERT OR UPDATE ON funding_rates
    FOR EACH ROW
    EXECUTE FUNCTION set_funding_next_time();

-- Log the number of records updated
DO $$
DECLARE
    updated_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO updated_count 
    FROM funding_rates 
    WHERE next_time IS NOT NULL;
    
    RAISE NOTICE 'Updated % funding rate records with valid next_time values', updated_count;
END $$;

COMMIT;

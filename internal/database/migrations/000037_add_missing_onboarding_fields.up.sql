-- Migration to add missing fields to strategy_onboarding table
-- This fixes the "deployed_strategies" field not found error

BEGIN;

-- Add missing fields to strategy_onboarding table if they don't exist
DO $$
BEGIN
    -- Add request_id field if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'strategy_onboarding' AND column_name = 'request_id') THEN
        ALTER TABLE strategy_onboarding ADD COLUMN request_id UUID UNIQUE;
        RAISE NOTICE 'Added request_id column to strategy_onboarding table';
    END IF;

    -- Add progress field if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'strategy_onboarding' AND column_name = 'progress') THEN
        ALTER TABLE strategy_onboarding ADD COLUMN progress DECIMAL(5,4) DEFAULT 0.0;
        RAISE NOTICE 'Added progress column to strategy_onboarding table';
    END IF;

    -- Add current_stage field if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'strategy_onboarding' AND column_name = 'current_stage') THEN
        ALTER TABLE strategy_onboarding ADD COLUMN current_stage VARCHAR(100);
        RAISE NOTICE 'Added current_stage column to strategy_onboarding table';
    END IF;

    -- Add generated_strategies field if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'strategy_onboarding' AND column_name = 'generated_strategies') THEN
        ALTER TABLE strategy_onboarding ADD COLUMN generated_strategies JSONB DEFAULT '[]';
        RAISE NOTICE 'Added generated_strategies column to strategy_onboarding table';
    END IF;

    -- Add test_results field if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'strategy_onboarding' AND column_name = 'test_results') THEN
        ALTER TABLE strategy_onboarding ADD COLUMN test_results JSONB DEFAULT '[]';
        RAISE NOTICE 'Added test_results column to strategy_onboarding table';
    END IF;

    -- Add deployed_strategies field if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'strategy_onboarding' AND column_name = 'deployed_strategies') THEN
        ALTER TABLE strategy_onboarding ADD COLUMN deployed_strategies JSONB DEFAULT '[]';
        RAISE NOTICE 'Added deployed_strategies column to strategy_onboarding table';
    END IF;

    -- Add errors field if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'strategy_onboarding' AND column_name = 'errors') THEN
        ALTER TABLE strategy_onboarding ADD COLUMN errors JSONB DEFAULT '[]';
        RAISE NOTICE 'Added errors column to strategy_onboarding table';
    END IF;

    -- Add warnings field if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'strategy_onboarding' AND column_name = 'warnings') THEN
        ALTER TABLE strategy_onboarding ADD COLUMN warnings JSONB DEFAULT '[]';
        RAISE NOTICE 'Added warnings column to strategy_onboarding table';
    END IF;

    -- Add start_time field if it doesn't exist (rename from started_at)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'strategy_onboarding' AND column_name = 'start_time') THEN
        ALTER TABLE strategy_onboarding ADD COLUMN start_time TIMESTAMP WITH TIME ZONE DEFAULT NOW();
        -- Copy data from started_at if it exists
        IF EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'strategy_onboarding' AND column_name = 'started_at') THEN
            UPDATE strategy_onboarding SET start_time = started_at WHERE started_at IS NOT NULL;
        END IF;
        RAISE NOTICE 'Added start_time column to strategy_onboarding table';
    END IF;

    -- Add end_time field if it doesn't exist (rename from completed_at)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'strategy_onboarding' AND column_name = 'end_time') THEN
        ALTER TABLE strategy_onboarding ADD COLUMN end_time TIMESTAMP WITH TIME ZONE;
        -- Copy data from completed_at if it exists
        IF EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'strategy_onboarding' AND column_name = 'completed_at') THEN
            UPDATE strategy_onboarding SET end_time = completed_at WHERE completed_at IS NOT NULL;
        END IF;
        RAISE NOTICE 'Added end_time column to strategy_onboarding table';
    END IF;

    -- Add duration field if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'strategy_onboarding' AND column_name = 'duration') THEN
        ALTER TABLE strategy_onboarding ADD COLUMN duration INTERVAL;
        -- Calculate duration for existing records
        UPDATE strategy_onboarding 
        SET duration = end_time - start_time 
        WHERE end_time IS NOT NULL AND start_time IS NOT NULL;
        RAISE NOTICE 'Added duration column to strategy_onboarding table';
    END IF;
END $$;

-- Create indexes for new fields
CREATE INDEX IF NOT EXISTS idx_strategy_onboarding_request_id ON strategy_onboarding(request_id);
CREATE INDEX IF NOT EXISTS idx_strategy_onboarding_current_stage ON strategy_onboarding(current_stage);
CREATE INDEX IF NOT EXISTS idx_strategy_onboarding_start_time ON strategy_onboarding(start_time);

-- Update existing records with default values for new fields
UPDATE strategy_onboarding 
SET 
    progress = 0.0,
    current_stage = CASE 
        WHEN status = 'pending' THEN 'initialization'
        WHEN status = 'processing' THEN 'validation'
        WHEN status = 'deployed' THEN 'completed'
        ELSE 'unknown'
    END,
    generated_strategies = '[]',
    test_results = '[]',
    deployed_strategies = '[]',
    errors = '[]',
    warnings = '[]'
WHERE 
    progress IS NULL 
    OR current_stage IS NULL 
    OR generated_strategies IS NULL 
    OR test_results IS NULL 
    OR deployed_strategies IS NULL 
    OR errors IS NULL 
    OR warnings IS NULL;

-- Generate request_id for existing records that don't have one
UPDATE strategy_onboarding
SET request_id = gen_random_uuid()
WHERE request_id IS NULL;

COMMIT;

-- Migration: Add auto_start field to strategies table
-- Version: 000033
-- Description: Adds auto_start field to control automatic strategy startup

-- Add auto_start field to strategies table
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS auto_start BOOLEAN DEFAULT false;

-- Add startup_priority field for controlling startup order
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS startup_priority INTEGER DEFAULT 100;

-- Add last_auto_start field to track when strategy was last auto-started
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS last_auto_start TIMESTAMP WITH TIME ZONE;

-- Update existing strategies to enable auto_start for active ones
UPDATE strategies SET 
    auto_start = true,
    startup_priority = CASE 
        WHEN status = 'active' AND enabled = true THEN 10  -- High priority for active strategies
        WHEN enabled = true THEN 50                        -- Medium priority for enabled strategies
        ELSE 100                                           -- Low priority for others
    END
WHERE enabled = true AND status IN ('active', 'inactive');

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_strategies_auto_start ON strategies(auto_start);
CREATE INDEX IF NOT EXISTS idx_strategies_startup_priority ON strategies(startup_priority);
CREATE INDEX IF NOT EXISTS idx_strategies_auto_start_priority ON strategies(auto_start, startup_priority);

-- Add comments for documentation
COMMENT ON COLUMN strategies.auto_start IS 'Whether the strategy should be automatically started on system startup';
COMMENT ON COLUMN strategies.startup_priority IS 'Priority for automatic startup (lower number = higher priority)';
COMMENT ON COLUMN strategies.last_auto_start IS 'Timestamp of last automatic startup';

-- Success message
DO $$ 
BEGIN
    RAISE NOTICE 'Migration 000033 completed successfully';
    RAISE NOTICE 'Added auto_start functionality to strategies table';
    RAISE NOTICE 'Enabled auto_start for existing active strategies';
END $$;

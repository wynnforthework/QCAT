-- Migration 000010: Add strategy optimization fields
-- This migration adds missing fields to the strategies table for optimization tracking

BEGIN;

-- Add optimization-related fields to strategies table
DO $$ 
BEGIN
    -- Add last_optimized field
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategies' AND column_name = 'last_optimized') THEN
        ALTER TABLE strategies ADD COLUMN last_optimized TIMESTAMP WITH TIME ZONE;
    END IF;
    
    -- Add performance field
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategies' AND column_name = 'performance') THEN
        ALTER TABLE strategies ADD COLUMN performance DECIMAL(10,6) DEFAULT 0;
    END IF;
    
    -- Add sharpe_ratio field
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategies' AND column_name = 'sharpe_ratio') THEN
        ALTER TABLE strategies ADD COLUMN sharpe_ratio DECIMAL(10,6) DEFAULT 0;
    END IF;
    
    -- Add max_drawdown field
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategies' AND column_name = 'max_drawdown') THEN
        ALTER TABLE strategies ADD COLUMN max_drawdown DECIMAL(10,6) DEFAULT 0;
    END IF;
    
    -- Add optimization_count field to track how many times strategy has been optimized
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategies' AND column_name = 'optimization_count') THEN
        ALTER TABLE strategies ADD COLUMN optimization_count INTEGER DEFAULT 0;
    END IF;
    
    -- Add optimization_status field to track current optimization state
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategies' AND column_name = 'optimization_status') THEN
        ALTER TABLE strategies ADD COLUMN optimization_status VARCHAR(20) DEFAULT 'none'; -- 'none', 'pending', 'running', 'completed', 'failed'
    END IF;
    
    -- Add next_optimization_due field for scheduling
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategies' AND column_name = 'next_optimization_due') THEN
        ALTER TABLE strategies ADD COLUMN next_optimization_due TIMESTAMP WITH TIME ZONE;
    END IF;
    
    -- Add optimization_config field for storing optimization parameters
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategies' AND column_name = 'optimization_config') THEN
        ALTER TABLE strategies ADD COLUMN optimization_config JSONB;
    END IF;
END $$;

-- Create indexes for optimization queries
CREATE INDEX IF NOT EXISTS idx_strategies_last_optimized ON strategies(last_optimized);
CREATE INDEX IF NOT EXISTS idx_strategies_optimization_status ON strategies(optimization_status);
CREATE INDEX IF NOT EXISTS idx_strategies_next_optimization_due ON strategies(next_optimization_due);
CREATE INDEX IF NOT EXISTS idx_strategies_performance ON strategies(performance);
CREATE INDEX IF NOT EXISTS idx_strategies_sharpe_ratio ON strategies(sharpe_ratio);
CREATE INDEX IF NOT EXISTS idx_strategies_max_drawdown ON strategies(max_drawdown);

-- Create composite index for optimization queries
CREATE INDEX IF NOT EXISTS idx_strategies_optimization_query ON strategies(status, last_optimized, sharpe_ratio, max_drawdown) 
WHERE status = 'active';

-- Add comments for documentation
COMMENT ON COLUMN strategies.last_optimized IS 'Timestamp of the last optimization run for this strategy';
COMMENT ON COLUMN strategies.performance IS 'Current performance metric (e.g., total return percentage)';
COMMENT ON COLUMN strategies.sharpe_ratio IS 'Current Sharpe ratio of the strategy';
COMMENT ON COLUMN strategies.max_drawdown IS 'Maximum drawdown percentage of the strategy';
COMMENT ON COLUMN strategies.optimization_count IS 'Number of times this strategy has been optimized';
COMMENT ON COLUMN strategies.optimization_status IS 'Current optimization status: none, pending, running, completed, failed';
COMMENT ON COLUMN strategies.next_optimization_due IS 'Scheduled time for next optimization';
COMMENT ON COLUMN strategies.optimization_config IS 'JSON configuration for optimization parameters';

COMMIT;

-- Migration 000010 rollback: Remove strategy optimization fields

BEGIN;

-- Drop indexes
DROP INDEX IF EXISTS idx_strategies_optimization_query;
DROP INDEX IF EXISTS idx_strategies_max_drawdown;
DROP INDEX IF EXISTS idx_strategies_sharpe_ratio;
DROP INDEX IF EXISTS idx_strategies_performance;
DROP INDEX IF EXISTS idx_strategies_next_optimization_due;
DROP INDEX IF EXISTS idx_strategies_optimization_status;
DROP INDEX IF EXISTS idx_strategies_last_optimized;

-- Remove optimization-related fields from strategies table
DO $$ 
BEGIN
    -- Remove optimization_config field
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategies' AND column_name = 'optimization_config') THEN
        ALTER TABLE strategies DROP COLUMN optimization_config;
    END IF;
    
    -- Remove next_optimization_due field
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategies' AND column_name = 'next_optimization_due') THEN
        ALTER TABLE strategies DROP COLUMN next_optimization_due;
    END IF;
    
    -- Remove optimization_status field
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategies' AND column_name = 'optimization_status') THEN
        ALTER TABLE strategies DROP COLUMN optimization_status;
    END IF;
    
    -- Remove optimization_count field
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategies' AND column_name = 'optimization_count') THEN
        ALTER TABLE strategies DROP COLUMN optimization_count;
    END IF;
    
    -- Remove max_drawdown field
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategies' AND column_name = 'max_drawdown') THEN
        ALTER TABLE strategies DROP COLUMN max_drawdown;
    END IF;
    
    -- Remove sharpe_ratio field
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategies' AND column_name = 'sharpe_ratio') THEN
        ALTER TABLE strategies DROP COLUMN sharpe_ratio;
    END IF;
    
    -- Remove performance field
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategies' AND column_name = 'performance') THEN
        ALTER TABLE strategies DROP COLUMN performance;
    END IF;
    
    -- Remove last_optimized field
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategies' AND column_name = 'last_optimized') THEN
        ALTER TABLE strategies DROP COLUMN last_optimized;
    END IF;
END $$;

COMMIT;

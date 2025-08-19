-- Drop indexes
DROP INDEX IF EXISTS idx_portfolio_history_timestamp;
DROP INDEX IF EXISTS idx_portfolio_history_portfolio_id;

DROP INDEX IF EXISTS idx_circuit_breakers_type;
DROP INDEX IF EXISTS idx_circuit_breakers_status;
DROP INDEX IF EXISTS idx_circuit_breakers_name;

DROP INDEX IF EXISTS idx_risk_violations_type;
DROP INDEX IF EXISTS idx_risk_violations_severity;
DROP INDEX IF EXISTS idx_risk_violations_status;
DROP INDEX IF EXISTS idx_risk_violations_entity;
DROP INDEX IF EXISTS idx_risk_violations_created_at;

DROP INDEX IF EXISTS idx_optimizer_tasks_strategy_id;
DROP INDEX IF EXISTS idx_optimizer_tasks_status;
DROP INDEX IF EXISTS idx_optimizer_tasks_algorithm;
DROP INDEX IF EXISTS idx_optimizer_tasks_created_at;

-- Drop tables
DROP TABLE IF EXISTS portfolio_history;
DROP TABLE IF EXISTS circuit_breakers;
DROP TABLE IF EXISTS risk_violations;

-- Remove added columns from optimizer_tasks (if they were added by this migration)
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'optimizer_tasks' AND column_name = 'algorithm') THEN
        ALTER TABLE optimizer_tasks DROP COLUMN IF EXISTS algorithm;
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'optimizer_tasks' AND column_name = 'parameters') THEN
        ALTER TABLE optimizer_tasks DROP COLUMN IF EXISTS parameters;
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'optimizer_tasks' AND column_name = 'results') THEN
        ALTER TABLE optimizer_tasks DROP COLUMN IF EXISTS results;
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'optimizer_tasks' AND column_name = 'progress') THEN
        ALTER TABLE optimizer_tasks DROP COLUMN IF EXISTS progress;
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'optimizer_tasks' AND column_name = 'error_message') THEN
        ALTER TABLE optimizer_tasks DROP COLUMN IF EXISTS error_message;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        -- Ignore errors during rollback
        NULL;
END $$;

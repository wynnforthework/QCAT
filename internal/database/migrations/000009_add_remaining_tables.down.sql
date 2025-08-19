-- Drop indexes
DROP INDEX IF EXISTS idx_hotlist_scores_symbol;
DROP INDEX IF EXISTS idx_hotlist_scores_total_score;
DROP INDEX IF EXISTS idx_hotlist_scores_created_at;

DROP INDEX IF EXISTS idx_trading_whitelist_symbol;
DROP INDEX IF EXISTS idx_trading_whitelist_status;
DROP INDEX IF EXISTS idx_trading_whitelist_approved_at;

DROP INDEX IF EXISTS idx_audit_logs_user_id;
DROP INDEX IF EXISTS idx_audit_logs_action;
DROP INDEX IF EXISTS idx_audit_logs_resource;
DROP INDEX IF EXISTS idx_audit_logs_created_at;

DROP INDEX IF EXISTS idx_audit_decisions_strategy_id;
DROP INDEX IF EXISTS idx_audit_decisions_type;
DROP INDEX IF EXISTS idx_audit_decisions_created_at;

DROP INDEX IF EXISTS idx_audit_performance_metric;
DROP INDEX IF EXISTS idx_audit_performance_timestamp;

-- Drop tables
DROP TABLE IF EXISTS hotlist_scores;
DROP TABLE IF EXISTS trading_whitelist;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS audit_decisions;
DROP TABLE IF EXISTS audit_performance;

-- Revert circuit_breakers table changes
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'circuit_breakers' AND column_name = 'action') THEN
        ALTER TABLE circuit_breakers DROP COLUMN IF EXISTS action;
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'circuit_breakers' AND column_name = 'triggered_at') THEN
        ALTER TABLE circuit_breakers DROP COLUMN IF EXISTS triggered_at;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        -- Ignore errors during rollback
        NULL;
END $$;

-- Revert risk_violations table changes
DO $$ 
BEGIN
    -- Rename columns back
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_violations' AND column_name = 'type') THEN
        ALTER TABLE risk_violations RENAME COLUMN type TO violation_type;
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_violations' AND column_name = 'threshold') THEN
        ALTER TABLE risk_violations RENAME COLUMN threshold TO threshold_value;
    END IF;
    
    -- Remove added columns
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_violations' AND column_name = 'symbol') THEN
        ALTER TABLE risk_violations DROP COLUMN IF EXISTS symbol;
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_violations' AND column_name = 'message') THEN
        ALTER TABLE risk_violations DROP COLUMN IF EXISTS message;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        -- Ignore errors during rollback
        NULL;
END $$;

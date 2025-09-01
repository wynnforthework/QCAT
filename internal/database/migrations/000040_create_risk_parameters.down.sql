-- Rollback migration for risk_parameters table

BEGIN;

-- Drop the trigger and function
DROP TRIGGER IF EXISTS trigger_update_risk_parameters_updated_at ON risk_parameters;
DROP FUNCTION IF EXISTS update_risk_parameters_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_risk_parameters_last_adjustment;
DROP INDEX IF EXISTS idx_risk_parameters_max_leverage;
DROP INDEX IF EXISTS idx_risk_parameters_active_symbol;
DROP INDEX IF EXISTS idx_risk_parameters_symbol_active;
DROP INDEX IF EXISTS idx_risk_parameters_updated_at;
DROP INDEX IF EXISTS idx_risk_parameters_created_at;
DROP INDEX IF EXISTS idx_risk_parameters_risk_profile;
DROP INDEX IF EXISTS idx_risk_parameters_is_active;
DROP INDEX IF EXISTS idx_risk_parameters_strategy_id;
DROP INDEX IF EXISTS idx_risk_parameters_symbol;

-- Drop the table
DROP TABLE IF EXISTS risk_parameters;

COMMIT;

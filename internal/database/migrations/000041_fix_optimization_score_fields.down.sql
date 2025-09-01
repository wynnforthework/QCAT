-- Rollback migration for optimization_score field fixes

BEGIN;

-- Drop indexes
DROP INDEX IF EXISTS idx_parameter_application_history_strategy_id;
DROP INDEX IF EXISTS idx_parameter_application_history_score;
DROP INDEX IF EXISTS idx_parameter_updates_status;
DROP INDEX IF EXISTS idx_parameter_updates_strategy_id;
DROP INDEX IF EXISTS idx_parameter_updates_score;
DROP INDEX IF EXISTS idx_optimization_history_applied;
DROP INDEX IF EXISTS idx_optimization_history_strategy_id;
DROP INDEX IF EXISTS idx_optimization_history_score;
DROP INDEX IF EXISTS idx_optimization_results_status;
DROP INDEX IF EXISTS idx_optimization_results_strategy_id;
DROP INDEX IF EXISTS idx_optimization_results_task_id;
DROP INDEX IF EXISTS idx_optimization_results_score;

-- Note: We don't remove the fields as they might be needed for compatibility
-- This is a conservative rollback that only removes the new indexes

COMMIT;

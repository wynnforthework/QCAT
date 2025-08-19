-- Migration 000011 rollback: Remove strategy onboarding tables

BEGIN;

-- Drop indexes
DROP INDEX IF EXISTS idx_deployment_history_status;
DROP INDEX IF EXISTS idx_deployment_history_environment;
DROP INDEX IF EXISTS idx_deployment_history_deployment_id;
DROP INDEX IF EXISTS idx_deployment_history_strategy_id;

DROP INDEX IF EXISTS idx_risk_models_is_active;
DROP INDEX IF EXISTS idx_risk_models_model_type;

DROP INDEX IF EXISTS idx_validation_rules_is_active;
DROP INDEX IF EXISTS idx_validation_rules_rule_type;

DROP INDEX IF EXISTS idx_onboarding_stages_status;
DROP INDEX IF EXISTS idx_onboarding_stages_stage_name;
DROP INDEX IF EXISTS idx_onboarding_stages_onboarding_id;

DROP INDEX IF EXISTS idx_strategy_onboarding_created_at;
DROP INDEX IF EXISTS idx_strategy_onboarding_risk_level;
DROP INDEX IF EXISTS idx_strategy_onboarding_status;
DROP INDEX IF EXISTS idx_strategy_onboarding_strategy_id;

-- Drop tables in reverse order (respecting foreign key constraints)
DROP TABLE IF EXISTS deployment_history;
DROP TABLE IF EXISTS risk_models;
DROP TABLE IF EXISTS validation_rules;
DROP TABLE IF EXISTS onboarding_stages;
DROP TABLE IF EXISTS strategy_onboarding;

COMMIT;

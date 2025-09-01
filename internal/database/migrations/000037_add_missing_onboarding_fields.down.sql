-- Rollback migration for adding missing fields to strategy_onboarding table

BEGIN;

-- Drop indexes for new fields
DROP INDEX IF EXISTS idx_strategy_onboarding_request_id;
DROP INDEX IF EXISTS idx_strategy_onboarding_current_stage;
DROP INDEX IF EXISTS idx_strategy_onboarding_start_time;

-- Remove added fields from strategy_onboarding table
ALTER TABLE strategy_onboarding DROP COLUMN IF EXISTS request_id;
ALTER TABLE strategy_onboarding DROP COLUMN IF EXISTS progress;
ALTER TABLE strategy_onboarding DROP COLUMN IF EXISTS current_stage;
ALTER TABLE strategy_onboarding DROP COLUMN IF EXISTS generated_strategies;
ALTER TABLE strategy_onboarding DROP COLUMN IF EXISTS test_results;
ALTER TABLE strategy_onboarding DROP COLUMN IF EXISTS deployed_strategies;
ALTER TABLE strategy_onboarding DROP COLUMN IF EXISTS errors;
ALTER TABLE strategy_onboarding DROP COLUMN IF EXISTS warnings;
ALTER TABLE strategy_onboarding DROP COLUMN IF EXISTS start_time;
ALTER TABLE strategy_onboarding DROP COLUMN IF EXISTS end_time;
ALTER TABLE strategy_onboarding DROP COLUMN IF EXISTS duration;

COMMIT;

-- Migration 000028 Down: Remove transfer_count and test_duration fields
-- This migration removes the fields added in the up migration

BEGIN;

-- Remove indexes first
DROP INDEX IF EXISTS idx_fund_protection_history_transfer_count;
DROP INDEX IF EXISTS idx_strategy_onboarding_test_duration;
DROP INDEX IF EXISTS idx_strategy_onboarding_progress;
DROP INDEX IF EXISTS idx_strategy_onboarding_current_stage;

-- Remove fields from fund_protection_history table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'fund_protection_history' AND column_name = 'transfer_count') THEN
        ALTER TABLE fund_protection_history DROP COLUMN transfer_count;
        RAISE NOTICE 'Removed transfer_count column from fund_protection_history table';
    END IF;
END $$;

DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'fund_protection_history' AND column_name = 'success_rate') THEN
        ALTER TABLE fund_protection_history DROP COLUMN success_rate;
        RAISE NOTICE 'Removed success_rate column from fund_protection_history table';
    END IF;
END $$;

DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'fund_protection_history' AND column_name = 'expected_risk_reduction') THEN
        ALTER TABLE fund_protection_history DROP COLUMN expected_risk_reduction;
        RAISE NOTICE 'Removed expected_risk_reduction column from fund_protection_history table';
    END IF;
END $$;

-- Remove fields from strategy_onboarding table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'test_duration') THEN
        ALTER TABLE strategy_onboarding DROP COLUMN test_duration;
        RAISE NOTICE 'Removed test_duration column from strategy_onboarding table';
    END IF;
END $$;

DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'deploy_threshold') THEN
        ALTER TABLE strategy_onboarding DROP COLUMN deploy_threshold;
        RAISE NOTICE 'Removed deploy_threshold column from strategy_onboarding table';
    END IF;
END $$;

DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'parameters') THEN
        ALTER TABLE strategy_onboarding DROP COLUMN parameters;
        RAISE NOTICE 'Removed parameters column from strategy_onboarding table';
    END IF;
END $$;

DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'progress') THEN
        ALTER TABLE strategy_onboarding DROP COLUMN progress;
        RAISE NOTICE 'Removed progress column from strategy_onboarding table';
    END IF;
END $$;

DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'current_stage') THEN
        ALTER TABLE strategy_onboarding DROP COLUMN current_stage;
        RAISE NOTICE 'Removed current_stage column from strategy_onboarding table';
    END IF;
END $$;

COMMIT;

-- Migration 000030: Fix missing transfer_count and test_duration fields
-- This migration adds the missing fields that are causing database errors

BEGIN;

-- 1. Add missing 'transfer_count' field to fund_protection_history table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'fund_protection_history') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'fund_protection_history' AND column_name = 'transfer_count') THEN
            ALTER TABLE fund_protection_history ADD COLUMN transfer_count INTEGER DEFAULT 0;
            RAISE NOTICE 'Added transfer_count column to fund_protection_history table';
        END IF;
    END IF;
END $$;

-- 2. Add missing 'success_rate' field to fund_protection_history table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'fund_protection_history') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'fund_protection_history' AND column_name = 'success_rate') THEN
            ALTER TABLE fund_protection_history ADD COLUMN success_rate DECIMAL(5,4) DEFAULT 0.0000;
            RAISE NOTICE 'Added success_rate column to fund_protection_history table';
        END IF;
    END IF;
END $$;

-- 3. Add missing 'expected_risk_reduction' field to fund_protection_history table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'fund_protection_history') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'fund_protection_history' AND column_name = 'expected_risk_reduction') THEN
            ALTER TABLE fund_protection_history ADD COLUMN expected_risk_reduction DECIMAL(10,6) DEFAULT 0.000000;
            RAISE NOTICE 'Added expected_risk_reduction column to fund_protection_history table';
        END IF;
    END IF;
END $$;

-- 4. Add missing 'test_duration' field to strategy_onboarding table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'strategy_onboarding') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'test_duration') THEN
            ALTER TABLE strategy_onboarding ADD COLUMN test_duration INTERVAL DEFAULT '7 days';
            RAISE NOTICE 'Added test_duration column to strategy_onboarding table';
        END IF;
    END IF;
END $$;

-- 5. Add missing 'deploy_threshold' field to strategy_onboarding table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'strategy_onboarding') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'deploy_threshold') THEN
            ALTER TABLE strategy_onboarding ADD COLUMN deploy_threshold DECIMAL(5,4) DEFAULT 0.0200;
            RAISE NOTICE 'Added deploy_threshold column to strategy_onboarding table';
        END IF;
    END IF;
END $$;

-- 6. Add missing 'parameters' field to strategy_onboarding table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'strategy_onboarding') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'parameters') THEN
            ALTER TABLE strategy_onboarding ADD COLUMN parameters JSONB DEFAULT '{}';
            RAISE NOTICE 'Added parameters column to strategy_onboarding table';
        END IF;
    END IF;
END $$;

-- 7. Add missing 'progress' field to strategy_onboarding table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'strategy_onboarding') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'progress') THEN
            ALTER TABLE strategy_onboarding ADD COLUMN progress DECIMAL(5,2) DEFAULT 0.00;
            RAISE NOTICE 'Added progress column to strategy_onboarding table';
        END IF;
    END IF;
END $$;

-- 8. Add missing 'current_stage' field to strategy_onboarding table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'strategy_onboarding') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'current_stage') THEN
            ALTER TABLE strategy_onboarding ADD COLUMN current_stage VARCHAR(100) DEFAULT 'pending';
            RAISE NOTICE 'Added current_stage column to strategy_onboarding table';
        END IF;
    END IF;
END $$;

-- Add indexes for better performance
CREATE INDEX IF NOT EXISTS idx_fund_protection_history_transfer_count ON fund_protection_history(transfer_count);
CREATE INDEX IF NOT EXISTS idx_strategy_onboarding_test_duration ON strategy_onboarding(test_duration);
CREATE INDEX IF NOT EXISTS idx_strategy_onboarding_progress ON strategy_onboarding(progress);
CREATE INDEX IF NOT EXISTS idx_strategy_onboarding_current_stage ON strategy_onboarding(current_stage);

COMMIT;

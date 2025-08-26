-- Migration 000025: Fix missing database fields (DOWN)
-- This migration removes the added fields

BEGIN;

-- Remove helper functions
DROP FUNCTION IF EXISTS jsonb_search_text(JSONB, TEXT);
DROP FUNCTION IF EXISTS uuid_to_text(UUID);

-- Drop tickers table
DROP TABLE IF EXISTS tickers;

-- Remove fields from strategy_onboarding table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'request_id') THEN
        ALTER TABLE strategy_onboarding DROP COLUMN request_id;
    END IF;
END $$;

-- Remove fields from strategies table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategies' AND column_name = 'status') THEN
        ALTER TABLE strategies DROP COLUMN status;
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategies' AND column_name = 'stop_reason') THEN
        ALTER TABLE strategies DROP COLUMN stop_reason;
    END IF;
END $$;

COMMIT;

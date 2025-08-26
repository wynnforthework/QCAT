-- Migration 000025: Fix missing database fields
-- This migration adds missing fields to various tables

BEGIN;

-- Add missing fields to strategies table
DO $$ 
BEGIN
    -- Add stop_reason field if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategies' AND column_name = 'stop_reason') THEN
        ALTER TABLE strategies ADD COLUMN stop_reason TEXT;
        RAISE NOTICE 'Added stop_reason column to strategies table';
    END IF;
    
    -- Add status field if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategies' AND column_name = 'status') THEN
        ALTER TABLE strategies ADD COLUMN status VARCHAR(50) DEFAULT 'active';
        RAISE NOTICE 'Added status column to strategies table';
    END IF;
END $$;

-- Add missing fields to strategy_onboarding table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'strategy_onboarding') THEN
        -- Add request_id field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'request_id') THEN
            ALTER TABLE strategy_onboarding ADD COLUMN request_id UUID DEFAULT uuid_generate_v4();
            RAISE NOTICE 'Added request_id column to strategy_onboarding table';
        END IF;
    END IF;
END $$;

-- Create tickers table if it doesn't exist
CREATE TABLE IF NOT EXISTS tickers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    symbol VARCHAR(20) NOT NULL,
    price DECIMAL(20,8) NOT NULL,
    volume DECIMAL(20,8) NOT NULL,
    change_24h DECIMAL(10,4),
    high_24h DECIMAL(20,8),
    low_24h DECIMAL(20,8),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(symbol, timestamp)
);

-- Create indexes for tickers table
CREATE INDEX IF NOT EXISTS idx_tickers_symbol ON tickers(symbol);
CREATE INDEX IF NOT EXISTS idx_tickers_timestamp ON tickers(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_tickers_symbol_timestamp ON tickers(symbol, timestamp DESC);

-- Fix SQL query issues by adding proper type casting
-- This will be handled in the application code, but we can add some helper functions

-- Create function to safely cast UUID to text for comparisons
CREATE OR REPLACE FUNCTION uuid_to_text(uuid_val UUID)
RETURNS TEXT AS $$
BEGIN
    RETURN uuid_val::TEXT;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Create function to safely search in JSONB
CREATE OR REPLACE FUNCTION jsonb_search_text(jsonb_val JSONB, search_text TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN jsonb_val::TEXT ILIKE '%' || search_text || '%';
END;
$$ LANGUAGE plpgsql IMMUTABLE;

COMMIT;

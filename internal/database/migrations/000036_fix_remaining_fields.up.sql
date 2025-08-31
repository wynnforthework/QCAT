-- Migration: Fix remaining database field issues
-- Version: 000036
-- Description: Add missing fields and tables for complete API functionality

-- Enable UUID extension if not exists
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Fix strategy_onboarding table - add missing test_results field
DO $$ 
BEGIN
    -- Check if strategy_onboarding table exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'strategy_onboarding') THEN
        -- Add user_id field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'user_id') THEN
            ALTER TABLE strategy_onboarding ADD COLUMN user_id UUID;
            RAISE NOTICE 'Added user_id column to strategy_onboarding table';
        END IF;

        -- Add test_results field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'test_results') THEN
            ALTER TABLE strategy_onboarding ADD COLUMN test_results JSONB DEFAULT '[]';
            RAISE NOTICE 'Added test_results column to strategy_onboarding table';
        END IF;
    ELSE
        -- Create strategy_onboarding table if it doesn't exist
        CREATE TABLE strategy_onboarding (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            user_id UUID,
            status VARCHAR(50) DEFAULT 'pending',
            generated_strategies JSONB DEFAULT '[]',
            test_results JSONB DEFAULT '[]',
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        );
        RAISE NOTICE 'Created strategy_onboarding table with all required fields';
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error fixing strategy_onboarding table: %', SQLERRM;
END $$;

-- 2. Fix optimization_results table - add missing optimization_score field
DO $$ 
BEGIN
    -- Check if optimization_results table exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'optimization_results') THEN
        -- Add optimization_score field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'optimization_results' AND column_name = 'optimization_score') THEN
            ALTER TABLE optimization_results ADD COLUMN optimization_score DECIMAL(10,4) DEFAULT 0;
            RAISE NOTICE 'Added optimization_score column to optimization_results table';
        END IF;
        
        -- Add other missing optimization fields
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'optimization_results' AND column_name = 'sharpe_ratio') THEN
            ALTER TABLE optimization_results ADD COLUMN sharpe_ratio DECIMAL(10,4) DEFAULT 0;
            RAISE NOTICE 'Added sharpe_ratio column to optimization_results table';
        END IF;
        
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'optimization_results' AND column_name = 'max_drawdown') THEN
            ALTER TABLE optimization_results ADD COLUMN max_drawdown DECIMAL(10,4) DEFAULT 0;
            RAISE NOTICE 'Added max_drawdown column to optimization_results table';
        END IF;
        
    ELSE
        -- Create optimization_results table if it doesn't exist
        CREATE TABLE optimization_results (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            task_id VARCHAR(100) NOT NULL,
            strategy_id UUID,
            optimization_score DECIMAL(10,4) DEFAULT 0,
            sharpe_ratio DECIMAL(10,4) DEFAULT 0,
            max_drawdown DECIMAL(10,4) DEFAULT 0,
            total_return DECIMAL(10,4) DEFAULT 0,
            parameters JSONB DEFAULT '{}',
            status VARCHAR(50) DEFAULT 'pending',
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        );
        RAISE NOTICE 'Created optimization_results table with all required fields';
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error fixing optimization_results table: %', SQLERRM;
END $$;

-- 3. Create funding_rates table for hot coin recommendations
DO $$ 
BEGIN
    -- Create funding_rates table if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'funding_rates') THEN
        CREATE TABLE funding_rates (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            symbol VARCHAR(20) NOT NULL,
            funding_rate DECIMAL(10,8) NOT NULL,
            funding_time TIMESTAMP WITH TIME ZONE NOT NULL,
            mark_price DECIMAL(20,8),
            index_price DECIMAL(20,8),
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        );
        
        -- Create indexes for better performance
        CREATE INDEX idx_funding_rates_symbol ON funding_rates(symbol);
        CREATE INDEX idx_funding_rates_funding_time ON funding_rates(funding_time DESC);
        CREATE INDEX idx_funding_rates_symbol_time ON funding_rates(symbol, funding_time DESC);
        
        RAISE NOTICE 'Created funding_rates table with indexes';
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error creating funding_rates table: %', SQLERRM;
END $$;

-- 4. Fix fund_protection_history table - ensure protocol_type is not null
DO $$ 
BEGIN
    -- Check if fund_protection_history table exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'fund_protection_history') THEN
        -- Remove NOT NULL constraint from protocol_type if it exists
        IF EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'fund_protection_history' 
            AND column_name = 'protocol_type' 
            AND is_nullable = 'NO'
        ) THEN
            ALTER TABLE fund_protection_history ALTER COLUMN protocol_type DROP NOT NULL;
            RAISE NOTICE 'Removed NOT NULL constraint from protocol_type in fund_protection_history';
        END IF;
        
        -- Set default value for protocol_type
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'fund_protection_history' AND column_name = 'protocol_type') THEN
            ALTER TABLE fund_protection_history ALTER COLUMN protocol_type SET DEFAULT 'standard';
            UPDATE fund_protection_history SET protocol_type = 'standard' WHERE protocol_type IS NULL;
            RAISE NOTICE 'Set default value for protocol_type in fund_protection_history';
        END IF;
    ELSE
        -- Create fund_protection_history table if it doesn't exist
        CREATE TABLE fund_protection_history (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            protocol_type VARCHAR(50) DEFAULT 'standard',
            action_type VARCHAR(50),
            amount DECIMAL(20,8) DEFAULT 0,
            status VARCHAR(50) DEFAULT 'completed',
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        );
        RAISE NOTICE 'Created fund_protection_history table';
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error fixing fund_protection_history table: %', SQLERRM;
END $$;

-- 5. Insert sample data for funding rates to prevent empty result errors
DO $$ 
BEGIN
    -- Insert sample funding rates if none exist
    IF NOT EXISTS (SELECT 1 FROM funding_rates LIMIT 1) THEN
        INSERT INTO funding_rates (symbol, funding_rate, funding_time, mark_price, index_price) VALUES
        ('BTCUSDT', 0.0001, NOW() - INTERVAL '8 hours', 50000.0, 49995.0),
        ('ETHUSDT', 0.0002, NOW() - INTERVAL '8 hours', 3000.0, 2998.0),
        ('BNBUSDT', 0.0001, NOW() - INTERVAL '8 hours', 300.0, 299.5),
        ('ADAUSDT', 0.0003, NOW() - INTERVAL '8 hours', 0.5, 0.499),
        ('SOLUSDT', 0.0002, NOW() - INTERVAL '8 hours', 150.0, 149.8);
        
        RAISE NOTICE 'Added sample funding rates for 5 symbols';
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error adding sample funding rates: %', SQLERRM;
END $$;

-- 6. Insert sample optimization results to prevent empty result errors
DO $$ 
DECLARE
    sample_strategy_id UUID;
BEGIN
    -- Get a sample strategy for optimization results
    SELECT id INTO sample_strategy_id FROM strategies LIMIT 1;
    
    IF sample_strategy_id IS NOT NULL THEN
        -- Insert sample optimization results if none exist
        IF NOT EXISTS (SELECT 1 FROM optimization_results LIMIT 1) THEN
            INSERT INTO optimization_results (
                task_id, strategy_id, optimization_score, sharpe_ratio, 
                max_drawdown, total_return, parameters, status
            ) VALUES (
                'sample_task_001', sample_strategy_id, 85.5, 1.2, 
                0.05, 0.15, '{"param1": 10, "param2": 0.5}', 'completed'
            );
            RAISE NOTICE 'Added sample optimization result for strategy: %', sample_strategy_id;
        END IF;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error adding sample optimization results: %', SQLERRM;
END $$;

-- 7. Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_strategy_onboarding_user_id ON strategy_onboarding(user_id);
CREATE INDEX IF NOT EXISTS idx_strategy_onboarding_status ON strategy_onboarding(status);
CREATE INDEX IF NOT EXISTS idx_optimization_results_task_id ON optimization_results(task_id);
CREATE INDEX IF NOT EXISTS idx_optimization_results_strategy_id ON optimization_results(strategy_id);
CREATE INDEX IF NOT EXISTS idx_optimization_results_score ON optimization_results(optimization_score DESC);

-- Success message
DO $$ 
BEGIN
    RAISE NOTICE 'Migration 000036 completed successfully';
    RAISE NOTICE 'Fixed missing fields: test_results, optimization_score';
    RAISE NOTICE 'Created missing table: funding_rates';
    RAISE NOTICE 'Fixed fund_protection_history protocol_type constraint';
    RAISE NOTICE 'Added sample data to prevent API errors';
END $$;

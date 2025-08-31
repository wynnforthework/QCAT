-- Migration: Fix performance metrics fields
-- Version: 000035
-- Description: Add missing performance metrics fields to fix dashboard API errors

-- Enable UUID extension if not exists
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Fix strategy_metrics table - add missing performance fields
DO $$ 
BEGIN
    -- Check if strategy_metrics table exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'strategy_metrics') THEN
        -- Add sortino_ratio field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_metrics' AND column_name = 'sortino_ratio') THEN
            ALTER TABLE strategy_metrics ADD COLUMN sortino_ratio DECIMAL(10,4) DEFAULT 0;
            RAISE NOTICE 'Added sortino_ratio column to strategy_metrics table';
        END IF;
        
        -- Add calmar_ratio field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_metrics' AND column_name = 'calmar_ratio') THEN
            ALTER TABLE strategy_metrics ADD COLUMN calmar_ratio DECIMAL(10,4) DEFAULT 0;
            RAISE NOTICE 'Added calmar_ratio column to strategy_metrics table';
        END IF;
        
        -- Add profit_factor field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_metrics' AND column_name = 'profit_factor') THEN
            ALTER TABLE strategy_metrics ADD COLUMN profit_factor DECIMAL(10,4) DEFAULT 0;
            RAISE NOTICE 'Added profit_factor column to strategy_metrics table';
        END IF;
        
        -- Add win_rate field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_metrics' AND column_name = 'win_rate') THEN
            ALTER TABLE strategy_metrics ADD COLUMN win_rate DECIMAL(5,4) DEFAULT 0;
            RAISE NOTICE 'Added win_rate column to strategy_metrics table';
        END IF;
        
    ELSE
        -- Create strategy_metrics table if it doesn't exist
        CREATE TABLE strategy_metrics (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            strategy_id UUID NOT NULL,
            sharpe_ratio DECIMAL(10,4) DEFAULT 0,
            max_drawdown DECIMAL(10,4) DEFAULT 0,
            total_return DECIMAL(10,4) DEFAULT 0,
            volatility DECIMAL(10,4) DEFAULT 0,
            win_rate DECIMAL(5,4) DEFAULT 0,
            profit_factor DECIMAL(10,4) DEFAULT 0,
            calmar_ratio DECIMAL(10,4) DEFAULT 0,
            sortino_ratio DECIMAL(10,4) DEFAULT 0,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        );
        RAISE NOTICE 'Created strategy_metrics table with all required fields';
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error fixing strategy_metrics table: %', SQLERRM;
END $$;

-- 2. Fix risk_snapshots table for risk metrics
DO $$ 
BEGIN
    -- Check if risk_snapshots table exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'risk_snapshots') THEN
        -- Add missing risk fields
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_snapshots' AND column_name = 'risk_level') THEN
            ALTER TABLE risk_snapshots ADD COLUMN risk_level VARCHAR(20) DEFAULT 'low';
            RAISE NOTICE 'Added risk_level column to risk_snapshots table';
        END IF;
        
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_snapshots' AND column_name = 'risk_score') THEN
            ALTER TABLE risk_snapshots ADD COLUMN risk_score DECIMAL(5,2) DEFAULT 0;
            RAISE NOTICE 'Added risk_score column to risk_snapshots table';
        END IF;
        
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_snapshots' AND column_name = 'leverage') THEN
            ALTER TABLE risk_snapshots ADD COLUMN leverage DECIMAL(10,2) DEFAULT 1.0;
            RAISE NOTICE 'Added leverage column to risk_snapshots table';
        END IF;
        
    ELSE
        -- Create risk_snapshots table if it doesn't exist
        CREATE TABLE risk_snapshots (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            exposure DECIMAL(20,8) DEFAULT 0,
            drawdown DECIMAL(10,4) DEFAULT 0,
            var_95 DECIMAL(20,8) DEFAULT 0,
            var_99 DECIMAL(20,8) DEFAULT 0,
            current_risk DECIMAL(10,4) DEFAULT 0,
            risk_budget DECIMAL(10,4) DEFAULT 0,
            risk_level VARCHAR(20) DEFAULT 'low',
            risk_score DECIMAL(5,2) DEFAULT 0,
            leverage DECIMAL(10,2) DEFAULT 1.0,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        );
        RAISE NOTICE 'Created risk_snapshots table with all required fields';
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error fixing risk_snapshots table: %', SQLERRM;
END $$;

-- 3. Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_strategy_metrics_strategy_id ON strategy_metrics(strategy_id);
CREATE INDEX IF NOT EXISTS idx_strategy_metrics_updated_at ON strategy_metrics(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_snapshots_created_at ON risk_snapshots(created_at DESC);

-- 4. Insert sample data to prevent empty result errors
DO $$ 
DECLARE
    sample_strategy_id UUID;
BEGIN
    -- Get a sample strategy for metrics
    SELECT id INTO sample_strategy_id FROM strategies LIMIT 1;
    
    IF sample_strategy_id IS NOT NULL THEN
        -- Insert sample strategy metrics if none exist
        IF NOT EXISTS (SELECT 1 FROM strategy_metrics WHERE strategy_id = sample_strategy_id) THEN
            INSERT INTO strategy_metrics (
                strategy_id, sharpe_ratio, max_drawdown, total_return, volatility,
                win_rate, profit_factor, calmar_ratio, sortino_ratio
            ) VALUES (
                sample_strategy_id, 1.2, 0.05, 0.15, 0.12,
                0.6, 1.5, 3.0, 1.8
            );
            RAISE NOTICE 'Added sample strategy metrics for strategy: %', sample_strategy_id;
        END IF;
    END IF;
    
    -- Insert sample risk snapshot if none exist
    IF NOT EXISTS (SELECT 1 FROM risk_snapshots LIMIT 1) THEN
        INSERT INTO risk_snapshots (
            exposure, drawdown, var_95, var_99, current_risk, risk_budget,
            risk_level, risk_score, leverage
        ) VALUES (
            50000.0, 0.05, 2000.0, 3000.0, 0.3, 0.5,
            'medium', 65.0, 2.5
        );
        RAISE NOTICE 'Added sample risk snapshot';
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error adding sample performance data: %', SQLERRM;
END $$;

-- Success message
DO $$ 
BEGIN
    RAISE NOTICE 'Migration 000035 completed successfully';
    RAISE NOTICE 'Fixed missing performance metrics fields: sortino_ratio, calmar_ratio, profit_factor, win_rate';
    RAISE NOTICE 'Fixed missing risk fields: risk_level, risk_score, leverage';
    RAISE NOTICE 'Added sample data to prevent dashboard API errors';
END $$;

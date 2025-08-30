-- Migration 000025: Fix missing tables and fields
-- This migration creates missing tables and adds missing fields

BEGIN;

-- 1. Create optimization_results table if it doesn't exist
CREATE TABLE IF NOT EXISTS optimization_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id VARCHAR(100) NOT NULL UNIQUE,
    strategy_id UUID,
    parameters JSONB DEFAULT '{}',
    performance_metrics JSONB DEFAULT '{}',
    backtest_result JSONB DEFAULT '{}',
    optimization_score DECIMAL(10,6) DEFAULT 0,  -- 添加缺失的字段
    improvement_score DECIMAL(10,6) DEFAULT 0,
    objective_value DECIMAL(10,6) DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 2. Add optimization_score field to existing optimization_results table if missing
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'optimization_results' AND column_name = 'optimization_score') THEN
        ALTER TABLE optimization_results ADD COLUMN optimization_score DECIMAL(10,6) DEFAULT 0;
    END IF;
END $$;

-- 3. Create open_interest table if it doesn't exist (fix PostgreSQL syntax)
CREATE TABLE IF NOT EXISTS open_interest (
    id BIGSERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    value DECIMAL(20,8) NOT NULL,
    notional DECIMAL(20,8) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 4. Create indexes for optimization_results
CREATE INDEX IF NOT EXISTS idx_optimization_results_task_id ON optimization_results(task_id);
CREATE INDEX IF NOT EXISTS idx_optimization_results_strategy_id ON optimization_results(strategy_id);
CREATE INDEX IF NOT EXISTS idx_optimization_results_status ON optimization_results(status);
CREATE INDEX IF NOT EXISTS idx_optimization_results_score ON optimization_results(optimization_score DESC);
CREATE INDEX IF NOT EXISTS idx_optimization_results_created_at ON optimization_results(created_at DESC);

-- 5. Create indexes for open_interest (PostgreSQL syntax)
CREATE INDEX IF NOT EXISTS idx_open_interest_symbol ON open_interest(symbol);
CREATE INDEX IF NOT EXISTS idx_open_interest_timestamp ON open_interest(timestamp);
CREATE INDEX IF NOT EXISTS idx_open_interest_symbol_timestamp ON open_interest(symbol, timestamp);

-- 6. Create trigger to automatically update updated_at for optimization_results
CREATE OR REPLACE FUNCTION update_optimization_results_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_optimization_results_updated_at
    BEFORE UPDATE ON optimization_results
    FOR EACH ROW
    EXECUTE FUNCTION update_optimization_results_updated_at();

-- 7. Add comments for documentation
COMMENT ON TABLE optimization_results IS 'Stores optimization task results and performance metrics';
COMMENT ON TABLE open_interest IS 'Stores open interest data for futures contracts';

COMMIT;

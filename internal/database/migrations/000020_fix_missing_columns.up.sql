-- Migration: Fix missing columns and database schema issues
-- Version: 000020
-- Description: Adds missing columns and fixes database schema issues

-- Add missing success_rate column to hedge_history table
ALTER TABLE hedge_history ADD COLUMN IF NOT EXISTS success_rate DECIMAL(5,4) DEFAULT 0.0000;

-- Add missing symbol column to strategy_performance table if it doesn't exist
ALTER TABLE strategy_performance ADD COLUMN IF NOT EXISTS symbol VARCHAR(20);

-- Create optimization_history table if it doesn't exist
CREATE TABLE IF NOT EXISTS optimization_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    optimization_id VARCHAR(100) NOT NULL UNIQUE,
    optimization_type VARCHAR(50) NOT NULL,
    strategy_id UUID,
    parameters_before JSONB DEFAULT '{}',
    parameters_after JSONB DEFAULT '{}',
    performance_before JSONB DEFAULT '{}',
    performance_after JSONB DEFAULT '{}',
    improvement_score DECIMAL(10,6) DEFAULT 0,
    objective_value DECIMAL(10,6) DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for optimization_history
CREATE INDEX IF NOT EXISTS idx_optimization_history_strategy_id ON optimization_history(strategy_id);
CREATE INDEX IF NOT EXISTS idx_optimization_history_status ON optimization_history(status);
CREATE INDEX IF NOT EXISTS idx_optimization_history_started_at ON optimization_history(started_at);

-- Add comments for documentation
COMMENT ON COLUMN hedge_history.success_rate IS 'Success rate of the hedge strategy (0.0 to 1.0)';
COMMENT ON TABLE optimization_history IS 'History of all optimization runs and their results';

-- Create trigger for updated_at on optimization_history
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_optimization_history_updated_at 
    BEFORE UPDATE ON optimization_history 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

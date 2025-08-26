-- Migration 000024: Add backtest_results table
-- This migration creates the missing backtest_results table that is referenced in the code

BEGIN;

-- Create backtest_results table
CREATE TABLE IF NOT EXISTS backtest_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
    total_return DECIMAL(10,6) DEFAULT 0,
    sharpe_ratio DECIMAL(10,6) DEFAULT 0,
    max_drawdown DECIMAL(10,6) DEFAULT 0,
    win_rate DECIMAL(5,4) DEFAULT 0,
    total_trades INTEGER DEFAULT 0,
    backtest_days INTEGER DEFAULT 0,
    is_valid BOOLEAN DEFAULT false,
    failure_reasons TEXT,
    backtest_date TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(strategy_id)
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_backtest_results_strategy_id ON backtest_results(strategy_id);
CREATE INDEX IF NOT EXISTS idx_backtest_results_is_valid ON backtest_results(is_valid);
CREATE INDEX IF NOT EXISTS idx_backtest_results_sharpe_ratio ON backtest_results(sharpe_ratio DESC);
CREATE INDEX IF NOT EXISTS idx_backtest_results_created_at ON backtest_results(created_at DESC);

-- Create trigger to automatically update updated_at
CREATE OR REPLACE FUNCTION update_backtest_results_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Drop trigger if exists and recreate
DROP TRIGGER IF EXISTS update_backtest_results_updated_at_trigger ON backtest_results;
CREATE TRIGGER update_backtest_results_updated_at_trigger
    BEFORE UPDATE ON backtest_results
    FOR EACH ROW EXECUTE FUNCTION update_backtest_results_updated_at();

-- Create backtest_tasks table if it doesn't exist (referenced in backtest_scheduler.go)
CREATE TABLE IF NOT EXISTS backtest_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'running', 'completed', 'failed'
    priority INTEGER DEFAULT 0,
    scheduled_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    result_data JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for backtest_tasks
CREATE INDEX IF NOT EXISTS idx_backtest_tasks_strategy_id ON backtest_tasks(strategy_id);
CREATE INDEX IF NOT EXISTS idx_backtest_tasks_status ON backtest_tasks(status);
CREATE INDEX IF NOT EXISTS idx_backtest_tasks_scheduled_at ON backtest_tasks(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_backtest_tasks_priority ON backtest_tasks(priority DESC);

-- Create trigger for backtest_tasks updated_at
DROP TRIGGER IF EXISTS update_backtest_tasks_updated_at_trigger ON backtest_tasks;
CREATE TRIGGER update_backtest_tasks_updated_at_trigger
    BEFORE UPDATE ON backtest_tasks
    FOR EACH ROW EXECUTE FUNCTION update_backtest_results_updated_at();

COMMIT;

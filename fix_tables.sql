-- Fix missing database tables and fields

-- Create optimization_results table if it doesn't exist
CREATE TABLE IF NOT EXISTS optimization_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id VARCHAR(100) NOT NULL UNIQUE,
    strategy_id UUID,
    parameters JSONB DEFAULT '{}',
    performance_metrics JSONB DEFAULT '{}',
    backtest_result JSONB DEFAULT '{}',
    optimization_score DECIMAL(10,6) DEFAULT 0,
    improvement_score DECIMAL(10,6) DEFAULT 0,
    objective_value DECIMAL(10,6) DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for optimization_results
CREATE INDEX IF NOT EXISTS idx_optimization_results_task_id ON optimization_results(task_id);
CREATE INDEX IF NOT EXISTS idx_optimization_results_strategy_id ON optimization_results(strategy_id);
CREATE INDEX IF NOT EXISTS idx_optimization_results_status ON optimization_results(status);
CREATE INDEX IF NOT EXISTS idx_optimization_results_score ON optimization_results(optimization_score DESC);
CREATE INDEX IF NOT EXISTS idx_optimization_results_created_at ON optimization_results(created_at DESC);

-- Create open_interest table if it doesn't exist (already created, but just in case)
CREATE TABLE IF NOT EXISTS open_interest (
    id BIGSERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    value DECIMAL(20,8) NOT NULL,
    notional DECIMAL(20,8) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for open_interest
CREATE INDEX IF NOT EXISTS idx_open_interest_symbol ON open_interest(symbol);
CREATE INDEX IF NOT EXISTS idx_open_interest_timestamp ON open_interest(timestamp);
CREATE INDEX IF NOT EXISTS idx_open_interest_symbol_timestamp ON open_interest(symbol, timestamp);

-- Add comments for documentation
COMMENT ON TABLE optimization_results IS 'Stores optimization task results and performance metrics';
COMMENT ON TABLE open_interest IS 'Stores open interest data for futures contracts';

-- Show created tables
SELECT 'optimization_results' as table_name, 
       CASE WHEN EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'optimization_results') 
            THEN 'EXISTS' ELSE 'NOT EXISTS' END as status
UNION ALL
SELECT 'open_interest' as table_name,
       CASE WHEN EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'open_interest') 
            THEN 'EXISTS' ELSE 'NOT EXISTS' END as status;

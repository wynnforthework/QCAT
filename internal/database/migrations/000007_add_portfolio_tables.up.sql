-- Create portfolio_allocations table
CREATE TABLE portfolio_allocations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id UUID NOT NULL REFERENCES strategies(id),
    weight DECIMAL(10,6) NOT NULL DEFAULT 0,
    target_weight DECIMAL(10,6) NOT NULL DEFAULT 0,
    pnl DECIMAL(30,10) NOT NULL DEFAULT 0,
    exposure DECIMAL(30,10) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(strategy_id)
);

-- Create rebalance_tasks table
CREATE TABLE rebalance_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    mode VARCHAR(50) NOT NULL DEFAULT 'bandit',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    config JSONB,
    result JSONB,
    error TEXT,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX idx_portfolio_allocations_strategy_id ON portfolio_allocations(strategy_id);
CREATE INDEX idx_portfolio_allocations_weight ON portfolio_allocations(weight DESC);
CREATE INDEX idx_rebalance_tasks_status ON rebalance_tasks(status);
CREATE INDEX idx_rebalance_tasks_mode ON rebalance_tasks(mode);
CREATE INDEX idx_rebalance_tasks_created_at ON rebalance_tasks(created_at);

-- Insert some sample data for testing
INSERT INTO portfolio_allocations (strategy_id, weight, target_weight, pnl, exposure) 
SELECT 
    id,
    RANDOM() * 0.5,  -- Random weight between 0 and 0.5
    RANDOM() * 0.6,  -- Random target weight between 0 and 0.6
    (RANDOM() - 0.5) * 10000,  -- Random PnL between -5000 and 5000
    RANDOM() * 50000  -- Random exposure between 0 and 50000
FROM strategies 
WHERE status = 'active'
LIMIT 5;

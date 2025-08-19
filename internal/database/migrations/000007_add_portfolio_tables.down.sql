-- Drop indexes
DROP INDEX IF EXISTS idx_rebalance_tasks_created_at;
DROP INDEX IF EXISTS idx_rebalance_tasks_mode;
DROP INDEX IF EXISTS idx_rebalance_tasks_status;
DROP INDEX IF EXISTS idx_portfolio_allocations_weight;
DROP INDEX IF EXISTS idx_portfolio_allocations_strategy_id;

-- Drop tables
DROP TABLE IF EXISTS rebalance_tasks;
DROP TABLE IF EXISTS portfolio_allocations;

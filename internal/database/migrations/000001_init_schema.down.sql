-- Drop indexes
DROP INDEX IF EXISTS idx_performance_metrics_strategy_id;
DROP INDEX IF EXISTS idx_market_data_symbol_timestamp;
DROP INDEX IF EXISTS idx_audit_logs_entity_id;
DROP INDEX IF EXISTS idx_hotlist_score;
DROP INDEX IF EXISTS idx_risk_limits_strategy_id;
DROP INDEX IF EXISTS idx_trades_strategy_id;
DROP INDEX IF EXISTS idx_trades_position_id;
DROP INDEX IF EXISTS idx_trades_order_id;
DROP INDEX IF EXISTS idx_orders_position_id;
DROP INDEX IF EXISTS idx_orders_strategy_id;
DROP INDEX IF EXISTS idx_positions_symbol;
DROP INDEX IF EXISTS idx_positions_strategy_id;
DROP INDEX IF EXISTS idx_optimizer_tasks_strategy_id;
DROP INDEX IF EXISTS idx_strategy_params_strategy_id;
DROP INDEX IF EXISTS idx_strategy_versions_strategy_id;
DROP INDEX IF EXISTS idx_strategies_status;

-- Drop tables
DROP TABLE IF EXISTS performance_metrics;
DROP TABLE IF EXISTS market_data;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS hotlist;
DROP TABLE IF EXISTS risk_limits;
DROP TABLE IF EXISTS trades;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS positions;
DROP TABLE IF EXISTS optimizer_tasks;
DROP TABLE IF EXISTS strategy_params;
DROP TABLE IF EXISTS strategy_versions;
DROP TABLE IF EXISTS strategies;

-- Drop extensions
DROP EXTENSION IF EXISTS "uuid-ossp";

-- Migration: Fix additional missing fields
-- Version: 000018
-- Description: Adds remaining missing fields that were causing errors

-- Add missing config field to strategies table
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS config JSONB DEFAULT '{}';

-- Add missing objective_value field to profit_optimization_history table
ALTER TABLE profit_optimization_history ADD COLUMN IF NOT EXISTS objective_value DECIMAL(15,8) DEFAULT 0;

-- Add missing total_operations field to hedge_history table
ALTER TABLE hedge_history ADD COLUMN IF NOT EXISTS total_operations INTEGER DEFAULT 0;

-- Add missing balance field to wallet_balances table (if it doesn't exist)
ALTER TABLE wallet_balances ADD COLUMN IF NOT EXISTS balance DECIMAL(30,10) DEFAULT 0;

-- Add comments for documentation
COMMENT ON COLUMN strategies.config IS 'Strategy configuration parameters in JSON format';
COMMENT ON COLUMN profit_optimization_history.objective_value IS 'Objective function value from optimization';
COMMENT ON COLUMN hedge_history.total_operations IS 'Total number of hedge operations performed';
COMMENT ON COLUMN wallet_balances.balance IS 'Current balance amount in the wallet';

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_strategies_config ON strategies USING GIN(config);
CREATE INDEX IF NOT EXISTS idx_profit_optimization_history_objective_value ON profit_optimization_history(objective_value DESC);
CREATE INDEX IF NOT EXISTS idx_hedge_history_total_operations ON hedge_history(total_operations);
CREATE INDEX IF NOT EXISTS idx_wallet_balances_balance ON wallet_balances(balance DESC);

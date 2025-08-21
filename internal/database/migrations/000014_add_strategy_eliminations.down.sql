-- Migration: Rollback strategy_eliminations table
-- Version: 000014
-- Description: Removes strategy_eliminations, hedge_history, and wallet_balances tables

-- Drop indexes
DROP INDEX IF EXISTS idx_wallet_balances_updated_at;
DROP INDEX IF EXISTS idx_wallet_balances_asset;
DROP INDEX IF EXISTS idx_wallet_balances_exchange;
DROP INDEX IF EXISTS idx_wallet_balances_wallet_id;

DROP INDEX IF EXISTS idx_hedge_history_strategy_ids;
DROP INDEX IF EXISTS idx_hedge_history_start_time;
DROP INDEX IF EXISTS idx_hedge_history_status;
DROP INDEX IF EXISTS idx_hedge_history_hedge_id;

DROP INDEX IF EXISTS idx_strategy_eliminations_disabled_until;
DROP INDEX IF EXISTS idx_strategy_eliminations_eliminated_at;
DROP INDEX IF EXISTS idx_strategy_eliminations_status;
DROP INDEX IF EXISTS idx_strategy_eliminations_strategy_id;

-- Drop tables
DROP TABLE IF EXISTS wallet_balances;
DROP TABLE IF EXISTS hedge_history;
DROP TABLE IF EXISTS strategy_eliminations;

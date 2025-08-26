-- Migration 000024: Drop backtest_results table (DOWN)
-- This migration drops the backtest_results and backtest_tasks tables

BEGIN;

-- Drop triggers first
DROP TRIGGER IF EXISTS update_backtest_tasks_updated_at_trigger ON backtest_tasks;
DROP TRIGGER IF EXISTS update_backtest_results_updated_at_trigger ON backtest_results;

-- Drop tables
DROP TABLE IF EXISTS backtest_tasks;
DROP TABLE IF EXISTS backtest_results;

COMMIT;

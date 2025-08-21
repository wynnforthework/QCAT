-- Migration: Rollback strategy fields
-- Version: 000019
-- Description: Rollback strategy field additions

-- Drop indexes
DROP INDEX IF EXISTS idx_strategies_risk_level;
DROP INDEX IF EXISTS idx_strategies_status_enabled;
DROP INDEX IF EXISTS idx_strategies_enabled;
DROP INDEX IF EXISTS idx_strategies_is_running;

-- Remove added columns
ALTER TABLE strategies DROP COLUMN IF EXISTS take_profit_pct;
ALTER TABLE strategies DROP COLUMN IF EXISTS stop_loss_pct;
ALTER TABLE strategies DROP COLUMN IF EXISTS max_position_size;
ALTER TABLE strategies DROP COLUMN IF EXISTS risk_level;
ALTER TABLE strategies DROP COLUMN IF EXISTS parameters;
ALTER TABLE strategies DROP COLUMN IF EXISTS enabled;
ALTER TABLE strategies DROP COLUMN IF EXISTS is_running;

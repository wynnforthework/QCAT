-- Migration: Rollback market_data table fields
-- Version: 000015
-- Description: Removes added fields from market_data table

-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_update_market_data_updated_at ON market_data;
DROP FUNCTION IF EXISTS update_market_data_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_market_data_volume_24h;
DROP INDEX IF EXISTS idx_market_data_symbol_updated_at;
DROP INDEX IF EXISTS idx_market_data_updated_at;

-- Remove added columns
ALTER TABLE market_data DROP COLUMN IF EXISTS oi_change_24h;
ALTER TABLE market_data DROP COLUMN IF EXISTS open_interest;
ALTER TABLE market_data DROP COLUMN IF EXISTS funding_rate;
ALTER TABLE market_data DROP COLUMN IF EXISTS volume_change_24h;
ALTER TABLE market_data DROP COLUMN IF EXISTS volatility;
ALTER TABLE market_data DROP COLUMN IF EXISTS change_24h;
ALTER TABLE market_data DROP COLUMN IF EXISTS price_change_24h;
ALTER TABLE market_data DROP COLUMN IF EXISTS volume_24h;
ALTER TABLE market_data DROP COLUMN IF EXISTS price;
ALTER TABLE market_data DROP COLUMN IF EXISTS updated_at;

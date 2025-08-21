-- Migration: Fix market_data table fields
-- Version: 000015
-- Description: Adds missing fields to market_data table for compatibility

-- Add missing fields to market_data table
ALTER TABLE market_data 
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE market_data 
ADD COLUMN IF NOT EXISTS price DECIMAL(30,10);

ALTER TABLE market_data 
ADD COLUMN IF NOT EXISTS volume_24h DECIMAL(30,10) DEFAULT 0;

ALTER TABLE market_data 
ADD COLUMN IF NOT EXISTS price_change_24h DECIMAL(30,10) DEFAULT 0;

ALTER TABLE market_data 
ADD COLUMN IF NOT EXISTS change_24h DECIMAL(30,10) DEFAULT 0;

ALTER TABLE market_data 
ADD COLUMN IF NOT EXISTS volatility DECIMAL(10,6) DEFAULT 0;

ALTER TABLE market_data 
ADD COLUMN IF NOT EXISTS volume_change_24h DECIMAL(30,10) DEFAULT 0;

ALTER TABLE market_data 
ADD COLUMN IF NOT EXISTS funding_rate DECIMAL(10,6) DEFAULT 0;

ALTER TABLE market_data 
ADD COLUMN IF NOT EXISTS open_interest DECIMAL(30,10) DEFAULT 0;

ALTER TABLE market_data 
ADD COLUMN IF NOT EXISTS oi_change_24h DECIMAL(30,10) DEFAULT 0;

-- Update price field with close price for existing records
UPDATE market_data 
SET price = close 
WHERE price IS NULL;

-- Update volume_24h with volume for existing records
UPDATE market_data 
SET volume_24h = volume 
WHERE volume_24h = 0 AND volume > 0;

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_market_data_updated_at ON market_data(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_market_data_symbol_updated_at ON market_data(symbol, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_market_data_volume_24h ON market_data(volume_24h DESC);

-- Add trigger to automatically update updated_at field
CREATE OR REPLACE FUNCTION update_market_data_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_market_data_updated_at ON market_data;
CREATE TRIGGER trigger_update_market_data_updated_at
    BEFORE UPDATE ON market_data
    FOR EACH ROW
    EXECUTE FUNCTION update_market_data_updated_at();

-- Add comments for documentation
COMMENT ON COLUMN market_data.price IS 'Current price (usually same as close price)';
COMMENT ON COLUMN market_data.volume_24h IS '24-hour trading volume';
COMMENT ON COLUMN market_data.price_change_24h IS '24-hour price change amount';
COMMENT ON COLUMN market_data.change_24h IS 'Alias for price_change_24h';
COMMENT ON COLUMN market_data.volatility IS 'Price volatility measure';
COMMENT ON COLUMN market_data.funding_rate IS 'Funding rate for futures';
COMMENT ON COLUMN market_data.open_interest IS 'Open interest for futures';
COMMENT ON COLUMN market_data.updated_at IS 'Last update timestamp';

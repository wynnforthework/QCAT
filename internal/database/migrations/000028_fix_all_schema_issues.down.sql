-- Migration: Rollback schema fixes
-- Version: 000028
-- Description: Rollback comprehensive schema fixes

-- Remove sample data
DELETE FROM tickers WHERE symbol IN ('BTCUSDT', 'ETHUSDT', 'BNBUSDT');
DELETE FROM market_data WHERE symbol IN ('BTCUSDT', 'ETHUSDT', 'BNBUSDT');

-- Remove indexes
DROP INDEX IF EXISTS idx_orders_status;
DROP INDEX IF EXISTS idx_orders_symbol;
DROP INDEX IF EXISTS idx_trades_symbol;
DROP INDEX IF EXISTS idx_market_data_symbol_timestamp;
DROP INDEX IF EXISTS idx_tickers_symbol;
DROP INDEX IF EXISTS idx_tickers_updated_at;

-- Rollback market_data table changes
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'market_data') THEN
        ALTER TABLE market_data DROP COLUMN IF EXISTS updated_at;
        ALTER TABLE market_data DROP COLUMN IF EXISTS price;
        ALTER TABLE market_data DROP COLUMN IF EXISTS volume_24h;
        ALTER TABLE market_data DROP COLUMN IF EXISTS price_change_24h;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        NULL;
END $$;

-- Rollback trades table changes
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'trades') THEN
        -- If we renamed size to quantity, rename it back
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'trades' AND column_name = 'quantity') THEN
            ALTER TABLE trades RENAME COLUMN quantity TO size;
        END IF;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        NULL;
END $$;

-- Rollback tickers table changes
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'tickers') THEN
        ALTER TABLE tickers DROP COLUMN IF EXISTS updated_at;
        ALTER TABLE tickers DROP COLUMN IF EXISTS volume_24h;
        ALTER TABLE tickers DROP COLUMN IF EXISTS price_change_24h;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        NULL;
END $$;

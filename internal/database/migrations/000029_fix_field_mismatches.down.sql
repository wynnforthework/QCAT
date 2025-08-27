-- Migration 000029 Down: Revert field name mismatch fixes
-- This migration reverts the changes made in the up migration

-- Drop triggers first
DROP TRIGGER IF EXISTS sync_tickers_trigger ON tickers;
DROP TRIGGER IF EXISTS sync_trades_trigger ON trades;

-- Drop trigger functions
DROP FUNCTION IF EXISTS sync_tickers_fields();
DROP FUNCTION IF EXISTS sync_trades_fields();

-- Remove added fields from tickers table (keep only essential ones)
DO $$ 
BEGIN
    -- Remove compatibility fields that were added
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'price') THEN
        ALTER TABLE tickers DROP COLUMN price;
        RAISE NOTICE 'Removed price column from tickers table';
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'price_change') THEN
        ALTER TABLE tickers DROP COLUMN price_change;
        RAISE NOTICE 'Removed price_change column from tickers table';
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'price_change_percent') THEN
        ALTER TABLE tickers DROP COLUMN price_change_percent;
        RAISE NOTICE 'Removed price_change_percent column from tickers table';
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'weighted_avg_price') THEN
        ALTER TABLE tickers DROP COLUMN weighted_avg_price;
        RAISE NOTICE 'Removed weighted_avg_price column from tickers table';
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'prev_close_price') THEN
        ALTER TABLE tickers DROP COLUMN prev_close_price;
        RAISE NOTICE 'Removed prev_close_price column from tickers table';
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'last_qty') THEN
        ALTER TABLE tickers DROP COLUMN last_qty;
        RAISE NOTICE 'Removed last_qty column from tickers table';
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'bid_price') THEN
        ALTER TABLE tickers DROP COLUMN bid_price;
        RAISE NOTICE 'Removed bid_price column from tickers table';
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'bid_qty') THEN
        ALTER TABLE tickers DROP COLUMN bid_qty;
        RAISE NOTICE 'Removed bid_qty column from tickers table';
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'ask_price') THEN
        ALTER TABLE tickers DROP COLUMN ask_price;
        RAISE NOTICE 'Removed ask_price column from tickers table';
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'ask_qty') THEN
        ALTER TABLE tickers DROP COLUMN ask_qty;
        RAISE NOTICE 'Removed ask_qty column from tickers table';
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'open_price') THEN
        ALTER TABLE tickers DROP COLUMN open_price;
        RAISE NOTICE 'Removed open_price column from tickers table';
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'high_price') THEN
        ALTER TABLE tickers DROP COLUMN high_price;
        RAISE NOTICE 'Removed high_price column from tickers table';
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'low_price') THEN
        ALTER TABLE tickers DROP COLUMN low_price;
        RAISE NOTICE 'Removed low_price column from tickers table';
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'quote_volume') THEN
        ALTER TABLE tickers DROP COLUMN quote_volume;
        RAISE NOTICE 'Removed quote_volume column from tickers table';
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'open_time') THEN
        ALTER TABLE tickers DROP COLUMN open_time;
        RAISE NOTICE 'Removed open_time column from tickers table';
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'close_time') THEN
        ALTER TABLE tickers DROP COLUMN close_time;
        RAISE NOTICE 'Removed close_time column from tickers table';
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'count') THEN
        ALTER TABLE tickers DROP COLUMN count;
        RAISE NOTICE 'Removed count column from tickers table';
    END IF;
    
    RAISE NOTICE 'Reverted tickers table field additions';
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error reverting tickers table: %', SQLERRM;
END $$;

-- Remove compatibility field from trades table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'trades' AND column_name = 'size') THEN
        ALTER TABLE trades DROP COLUMN size;
        RAISE NOTICE 'Removed size column from trades table';
    END IF;
    
    RAISE NOTICE 'Reverted trades table field additions';
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error reverting trades table: %', SQLERRM;
END $$;

-- Success message
DO $$ 
BEGIN
    RAISE NOTICE 'Migration 000029 rollback completed successfully';
    RAISE NOTICE 'Reverted field name mismatch fixes';
END $$;

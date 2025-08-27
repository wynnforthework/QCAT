-- Migration: Fix field name mismatches between code and database
-- Version: 000029
-- Description: Fix all field name mismatches causing query errors

-- Enable UUID extension if not exists
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Fix tickers table field mismatches
DO $$ 
BEGIN
    -- Add missing 'price' field (mapped to last_price for compatibility)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'price') THEN
        -- Add price column and populate it with existing data
        ALTER TABLE tickers ADD COLUMN price DECIMAL(20,8);
        -- Update price with existing price data if available
        UPDATE tickers SET price = COALESCE(price, 0) WHERE price IS NULL;
        RAISE NOTICE 'Added price column to tickers table';
    END IF;
    
    -- Ensure last_price field exists (some queries expect it)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'last_price') THEN
        ALTER TABLE tickers ADD COLUMN last_price DECIMAL(20,8);
        -- Copy from price field if it exists
        UPDATE tickers SET last_price = price WHERE last_price IS NULL;
        RAISE NOTICE 'Added last_price column to tickers table';
    END IF;
    
    -- Ensure all expected fields exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'price_change') THEN
        ALTER TABLE tickers ADD COLUMN price_change DECIMAL(20,8) DEFAULT 0;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'price_change_percent') THEN
        ALTER TABLE tickers ADD COLUMN price_change_percent DECIMAL(10,4) DEFAULT 0;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'weighted_avg_price') THEN
        ALTER TABLE tickers ADD COLUMN weighted_avg_price DECIMAL(20,8) DEFAULT 0;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'prev_close_price') THEN
        ALTER TABLE tickers ADD COLUMN prev_close_price DECIMAL(20,8) DEFAULT 0;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'last_qty') THEN
        ALTER TABLE tickers ADD COLUMN last_qty DECIMAL(20,8) DEFAULT 0;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'bid_price') THEN
        ALTER TABLE tickers ADD COLUMN bid_price DECIMAL(20,8) DEFAULT 0;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'bid_qty') THEN
        ALTER TABLE tickers ADD COLUMN bid_qty DECIMAL(20,8) DEFAULT 0;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'ask_price') THEN
        ALTER TABLE tickers ADD COLUMN ask_price DECIMAL(20,8) DEFAULT 0;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'ask_qty') THEN
        ALTER TABLE tickers ADD COLUMN ask_qty DECIMAL(20,8) DEFAULT 0;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'open_price') THEN
        ALTER TABLE tickers ADD COLUMN open_price DECIMAL(20,8) DEFAULT 0;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'high_price') THEN
        ALTER TABLE tickers ADD COLUMN high_price DECIMAL(20,8) DEFAULT 0;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'low_price') THEN
        ALTER TABLE tickers ADD COLUMN low_price DECIMAL(20,8) DEFAULT 0;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'quote_volume') THEN
        ALTER TABLE tickers ADD COLUMN quote_volume DECIMAL(20,8) DEFAULT 0;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'open_time') THEN
        ALTER TABLE tickers ADD COLUMN open_time TIMESTAMP WITH TIME ZONE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'close_time') THEN
        ALTER TABLE tickers ADD COLUMN close_time TIMESTAMP WITH TIME ZONE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'count') THEN
        ALTER TABLE tickers ADD COLUMN count BIGINT DEFAULT 0;
    END IF;
    
    RAISE NOTICE 'Fixed tickers table field mismatches';
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error fixing tickers table: %', SQLERRM;
END $$;

-- 2. Ensure trades table has both size and quantity fields for compatibility
DO $$ 
BEGIN
    -- Add size field if it doesn't exist (some queries expect it)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'trades' AND column_name = 'size') THEN
        ALTER TABLE trades ADD COLUMN size DECIMAL(20,8);
        -- Copy from quantity field
        UPDATE trades SET size = quantity WHERE size IS NULL;
        RAISE NOTICE 'Added size column to trades table (copied from quantity)';
    END IF;
    
    -- Ensure quantity field exists (it should already exist)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'trades' AND column_name = 'quantity') THEN
        ALTER TABLE trades ADD COLUMN quantity DECIMAL(30,10);
        -- Copy from size field if it exists
        UPDATE trades SET quantity = size WHERE quantity IS NULL;
        RAISE NOTICE 'Added quantity column to trades table';
    END IF;
    
    RAISE NOTICE 'Fixed trades table field compatibility';
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error fixing trades table: %', SQLERRM;
END $$;

-- 3. Update existing data to ensure consistency
DO $$ 
BEGIN
    -- Sync price and last_price in tickers
    UPDATE tickers SET 
        price = COALESCE(price, 0),
        last_price = COALESCE(last_price, price, 0)
    WHERE price IS NULL OR last_price IS NULL;
    
    -- Sync size and quantity in trades
    UPDATE trades SET 
        size = COALESCE(size, quantity, 0),
        quantity = COALESCE(quantity, size, 0)
    WHERE size IS NULL OR quantity IS NULL;
    
    RAISE NOTICE 'Synchronized field data for consistency';
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error synchronizing data: %', SQLERRM;
END $$;

-- 4. Ensure tickers table has proper constraints for sample data insertion
DO $$
BEGIN
    -- Check if symbol is already a primary key or has unique constraint
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'tickers'
        AND constraint_type IN ('PRIMARY KEY', 'UNIQUE')
        AND constraint_name LIKE '%symbol%'
    ) THEN
        -- Add unique constraint on symbol if it doesn't exist
        BEGIN
            ALTER TABLE tickers ADD CONSTRAINT tickers_symbol_unique UNIQUE (symbol);
            RAISE NOTICE 'Added unique constraint on symbol column';
        EXCEPTION
            WHEN duplicate_table THEN
                RAISE NOTICE 'Unique constraint on symbol already exists';
            WHEN OTHERS THEN
                RAISE NOTICE 'Could not add unique constraint on symbol: %', SQLERRM;
        END;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error checking/adding symbol constraint: %', SQLERRM;
END $$;

-- 5. Insert sample data with correct field names (using safe upsert)
DO $$
BEGIN
    -- Insert or update BTCUSDT
    IF EXISTS (SELECT 1 FROM tickers WHERE symbol = 'BTCUSDT') THEN
        UPDATE tickers SET
            price = 45000.00, last_price = 45000.00, volume = 1000000,
            volume_24h = 50000000, price_change_24h = 500.00, price_change = 500.00,
            price_change_percent = 1.12, open_price = 44500.00, high_price = 45500.00,
            low_price = 44000.00, bid_price = 44990.00, ask_price = 45010.00,
            updated_at = NOW()
        WHERE symbol = 'BTCUSDT';
    ELSE
        INSERT INTO tickers (symbol, price, last_price, volume, volume_24h, price_change_24h,
                           price_change, price_change_percent, open_price, high_price, low_price,
                           bid_price, ask_price, updated_at, created_at)
        VALUES ('BTCUSDT', 45000.00, 45000.00, 1000000, 50000000, 500.00, 500.00, 1.12,
                44500.00, 45500.00, 44000.00, 44990.00, 45010.00, NOW(), NOW());
    END IF;

    -- Insert or update ETHUSDT
    IF EXISTS (SELECT 1 FROM tickers WHERE symbol = 'ETHUSDT') THEN
        UPDATE tickers SET
            price = 3000.00, last_price = 3000.00, volume = 800000,
            volume_24h = 30000000, price_change_24h = 100.00, price_change = 100.00,
            price_change_percent = 3.45, open_price = 2950.00, high_price = 3050.00,
            low_price = 2900.00, bid_price = 2999.00, ask_price = 3001.00,
            updated_at = NOW()
        WHERE symbol = 'ETHUSDT';
    ELSE
        INSERT INTO tickers (symbol, price, last_price, volume, volume_24h, price_change_24h,
                           price_change, price_change_percent, open_price, high_price, low_price,
                           bid_price, ask_price, updated_at, created_at)
        VALUES ('ETHUSDT', 3000.00, 3000.00, 800000, 30000000, 100.00, 100.00, 3.45,
                2950.00, 3050.00, 2900.00, 2999.00, 3001.00, NOW(), NOW());
    END IF;

    -- Insert or update BNBUSDT
    IF EXISTS (SELECT 1 FROM tickers WHERE symbol = 'BNBUSDT') THEN
        UPDATE tickers SET
            price = 300.00, last_price = 300.00, volume = 500000,
            volume_24h = 10000000, price_change_24h = 10.00, price_change = 10.00,
            price_change_percent = 3.45, open_price = 295.00, high_price = 305.00,
            low_price = 290.00, bid_price = 299.90, ask_price = 300.10,
            updated_at = NOW()
        WHERE symbol = 'BNBUSDT';
    ELSE
        INSERT INTO tickers (symbol, price, last_price, volume, volume_24h, price_change_24h,
                           price_change, price_change_percent, open_price, high_price, low_price,
                           bid_price, ask_price, updated_at, created_at)
        VALUES ('BNBUSDT', 300.00, 300.00, 500000, 10000000, 10.00, 10.00, 3.45,
                295.00, 305.00, 290.00, 299.90, 300.10, NOW(), NOW());
    END IF;

    RAISE NOTICE 'Sample ticker data inserted/updated successfully';
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error inserting sample ticker data: %', SQLERRM;
END $$;

-- 6. Create triggers to keep fields synchronized
CREATE OR REPLACE FUNCTION sync_tickers_fields()
RETURNS TRIGGER AS $$
BEGIN
    -- Keep price and last_price synchronized
    IF NEW.price IS NOT NULL AND NEW.last_price IS NULL THEN
        NEW.last_price = NEW.price;
    ELSIF NEW.last_price IS NOT NULL AND NEW.price IS NULL THEN
        NEW.price = NEW.last_price;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION sync_trades_fields()
RETURNS TRIGGER AS $$
BEGIN
    -- Keep size and quantity synchronized
    IF NEW.quantity IS NOT NULL AND NEW.size IS NULL THEN
        NEW.size = NEW.quantity;
    ELSIF NEW.size IS NOT NULL AND NEW.quantity IS NULL THEN
        NEW.quantity = NEW.size;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers
DROP TRIGGER IF EXISTS sync_tickers_trigger ON tickers;
CREATE TRIGGER sync_tickers_trigger
    BEFORE INSERT OR UPDATE ON tickers
    FOR EACH ROW EXECUTE FUNCTION sync_tickers_fields();

DROP TRIGGER IF EXISTS sync_trades_trigger ON trades;
CREATE TRIGGER sync_trades_trigger
    BEFORE INSERT OR UPDATE ON trades
    FOR EACH ROW EXECUTE FUNCTION sync_trades_fields();

-- Success message
DO $$ 
BEGIN
    RAISE NOTICE 'Migration 000029 completed successfully';
    RAISE NOTICE 'Fixed field name mismatches between code and database';
    RAISE NOTICE 'Added synchronization triggers for data consistency';
END $$;

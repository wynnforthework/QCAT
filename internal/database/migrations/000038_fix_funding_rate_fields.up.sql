-- Migration to fix funding rate field inconsistencies
-- This fixes the "rate" field not found error in funding rate queries

BEGIN;

-- Check if funding_rates table exists and standardize field names
DO $$
BEGIN
    -- Check if the table exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'funding_rates') THEN
        
        -- Add 'rate' field if it doesn't exist (for compatibility with older queries)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'funding_rates' AND column_name = 'rate') THEN
            -- If funding_rate exists, copy it to rate
            IF EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'funding_rates' AND column_name = 'funding_rate') THEN
                ALTER TABLE funding_rates ADD COLUMN rate DECIMAL(10,8);
                UPDATE funding_rates SET rate = funding_rate WHERE funding_rate IS NOT NULL;
                RAISE NOTICE 'Added rate column and copied data from funding_rate column';
            ELSE
                -- If neither exists, add rate column
                ALTER TABLE funding_rates ADD COLUMN rate DECIMAL(10,8) NOT NULL DEFAULT 0;
                RAISE NOTICE 'Added rate column to funding_rates table';
            END IF;
        END IF;

        -- Add 'funding_rate' field if it doesn't exist (for compatibility with newer queries)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'funding_rates' AND column_name = 'funding_rate') THEN
            -- If rate exists, copy it to funding_rate
            IF EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'funding_rates' AND column_name = 'rate') THEN
                ALTER TABLE funding_rates ADD COLUMN funding_rate DECIMAL(10,8);
                UPDATE funding_rates SET funding_rate = rate WHERE rate IS NOT NULL;
                RAISE NOTICE 'Added funding_rate column and copied data from rate column';
            ELSE
                -- If neither exists, add funding_rate column
                ALTER TABLE funding_rates ADD COLUMN funding_rate DECIMAL(10,8) NOT NULL DEFAULT 0;
                RAISE NOTICE 'Added funding_rate column to funding_rates table';
            END IF;
        END IF;

        -- Ensure both fields are synchronized
        UPDATE funding_rates SET 
            rate = COALESCE(rate, funding_rate, 0),
            funding_rate = COALESCE(funding_rate, rate, 0)
        WHERE rate IS NULL OR funding_rate IS NULL;

        -- Add other missing fields for compatibility
        
        -- Add next_rate field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'funding_rates' AND column_name = 'next_rate') THEN
            ALTER TABLE funding_rates ADD COLUMN next_rate DECIMAL(10,8) DEFAULT 0;
            RAISE NOTICE 'Added next_rate column to funding_rates table';
        END IF;

        -- Add next_time field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'funding_rates' AND column_name = 'next_time') THEN
            ALTER TABLE funding_rates ADD COLUMN next_time TIMESTAMP WITH TIME ZONE;
            RAISE NOTICE 'Added next_time column to funding_rates table';
        END IF;

        -- Add last_updated field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'funding_rates' AND column_name = 'last_updated') THEN
            ALTER TABLE funding_rates ADD COLUMN last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW();
            UPDATE funding_rates SET last_updated = created_at WHERE last_updated IS NULL;
            RAISE NOTICE 'Added last_updated column to funding_rates table';
        END IF;

        -- Add funding_time field if it doesn't exist (alias for last_updated)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'funding_rates' AND column_name = 'funding_time') THEN
            ALTER TABLE funding_rates ADD COLUMN funding_time TIMESTAMP WITH TIME ZONE;
            UPDATE funding_rates SET funding_time = COALESCE(last_updated, created_at);
            RAISE NOTICE 'Added funding_time column to funding_rates table';
        END IF;

        -- Add mark_price field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'funding_rates' AND column_name = 'mark_price') THEN
            ALTER TABLE funding_rates ADD COLUMN mark_price DECIMAL(20,8);
            RAISE NOTICE 'Added mark_price column to funding_rates table';
        END IF;

        -- Add index_price field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'funding_rates' AND column_name = 'index_price') THEN
            ALTER TABLE funding_rates ADD COLUMN index_price DECIMAL(20,8);
            RAISE NOTICE 'Added index_price column to funding_rates table';
        END IF;

        -- Add timestamp field if it doesn't exist (alias for created_at)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name = 'funding_rates' AND column_name = 'timestamp') THEN
            ALTER TABLE funding_rates ADD COLUMN timestamp TIMESTAMP WITH TIME ZONE;
            UPDATE funding_rates SET timestamp = created_at WHERE timestamp IS NULL;
            RAISE NOTICE 'Added timestamp column to funding_rates table';
        END IF;

    ELSE
        -- Create the table if it doesn't exist with all necessary fields
        CREATE TABLE funding_rates (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            symbol VARCHAR(20) NOT NULL,
            rate DECIMAL(10,8) NOT NULL DEFAULT 0,
            funding_rate DECIMAL(10,8) NOT NULL DEFAULT 0,
            next_rate DECIMAL(10,8) DEFAULT 0,
            next_time TIMESTAMP WITH TIME ZONE,
            last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            funding_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            mark_price DECIMAL(20,8),
            index_price DECIMAL(20,8),
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );
        RAISE NOTICE 'Created funding_rates table with all necessary fields';
    END IF;
END $$;

-- Create or update indexes for better performance
CREATE INDEX IF NOT EXISTS idx_funding_rates_symbol ON funding_rates(symbol);
CREATE INDEX IF NOT EXISTS idx_funding_rates_created_at ON funding_rates(created_at);
CREATE INDEX IF NOT EXISTS idx_funding_rates_last_updated ON funding_rates(last_updated);
CREATE INDEX IF NOT EXISTS idx_funding_rates_funding_time ON funding_rates(funding_time DESC);
CREATE INDEX IF NOT EXISTS idx_funding_rates_timestamp ON funding_rates(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_funding_rates_symbol_time ON funding_rates(symbol, last_updated DESC);
CREATE INDEX IF NOT EXISTS idx_funding_rates_symbol_funding_time ON funding_rates(symbol, funding_time DESC);

-- Insert sample data if the table is empty
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM funding_rates LIMIT 1) THEN
        INSERT INTO funding_rates (symbol, rate, funding_rate, next_rate, next_time, last_updated, funding_time, timestamp, mark_price, index_price) VALUES
        ('BTCUSDT', 0.0001, 0.0001, 0.0001, NOW() + INTERVAL '8 hours', NOW(), NOW(), NOW(), 50000.0, 49995.0),
        ('ETHUSDT', 0.0002, 0.0002, 0.0002, NOW() + INTERVAL '8 hours', NOW(), NOW(), NOW(), 3000.0, 2998.0),
        ('BNBUSDT', 0.0001, 0.0001, 0.0001, NOW() + INTERVAL '8 hours', NOW(), NOW(), NOW(), 300.0, 299.5),
        ('ADAUSDT', 0.0003, 0.0003, 0.0003, NOW() + INTERVAL '8 hours', NOW(), NOW(), NOW(), 0.5, 0.499),
        ('SOLUSDT', 0.0002, 0.0002, 0.0002, NOW() + INTERVAL '8 hours', NOW(), NOW(), NOW(), 150.0, 149.8);
        
        RAISE NOTICE 'Added sample funding rates for 5 symbols';
    END IF;
END $$;

COMMIT;

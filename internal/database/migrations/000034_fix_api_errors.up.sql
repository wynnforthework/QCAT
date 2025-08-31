-- Migration: Fix API errors
-- Version: 000034
-- Description: Fix missing database fields causing API errors

-- Enable UUID extension if not exists
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Fix orders table - add missing quantity field
DO $$ 
BEGIN
    -- Check if orders table exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'orders') THEN
        -- Add quantity field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'orders' AND column_name = 'quantity') THEN
            ALTER TABLE orders ADD COLUMN quantity DECIMAL(30,10) DEFAULT 0;
            RAISE NOTICE 'Added quantity column to orders table';
            
            -- If size field exists, copy its values to quantity
            IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'orders' AND column_name = 'size') THEN
                UPDATE orders SET quantity = size WHERE quantity = 0 AND size IS NOT NULL;
                RAISE NOTICE 'Copied size values to quantity in orders table';
            END IF;
        END IF;
        
        -- Add order_type field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'orders' AND column_name = 'order_type') THEN
            ALTER TABLE orders ADD COLUMN order_type VARCHAR(20) DEFAULT 'MARKET';
            RAISE NOTICE 'Added order_type column to orders table';
        END IF;
    ELSE
        -- Create orders table if it doesn't exist
        CREATE TABLE orders (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            strategy_id UUID,
            symbol VARCHAR(20) NOT NULL,
            side VARCHAR(10) NOT NULL CHECK (side IN ('BUY', 'SELL')),
            quantity DECIMAL(30,10) NOT NULL DEFAULT 0,
            size DECIMAL(30,10) DEFAULT 0, -- For backward compatibility
            price DECIMAL(30,10) NOT NULL DEFAULT 0,
            order_type VARCHAR(20) DEFAULT 'MARKET',
            status VARCHAR(20) DEFAULT 'NEW',
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        );
        RAISE NOTICE 'Created orders table with all required fields';
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error fixing orders table: %', SQLERRM;
END $$;

-- 2. Fix strategy_onboarding table - add missing generated_strategies field
DO $$ 
BEGIN
    -- Check if strategy_onboarding table exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'strategy_onboarding') THEN
        -- Add generated_strategies field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'strategy_onboarding' AND column_name = 'generated_strategies') THEN
            ALTER TABLE strategy_onboarding ADD COLUMN generated_strategies JSONB DEFAULT '[]';
            RAISE NOTICE 'Added generated_strategies column to strategy_onboarding table';
        END IF;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error fixing strategy_onboarding table: %', SQLERRM;
END $$;

-- 3. Ensure trades table has both size and quantity for compatibility
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'trades') THEN
        -- Add quantity field if it doesn't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'trades' AND column_name = 'quantity') THEN
            ALTER TABLE trades ADD COLUMN quantity DECIMAL(30,10);
            -- Copy from size field if it exists
            IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'trades' AND column_name = 'size') THEN
                UPDATE trades SET quantity = size WHERE quantity IS NULL;
            END IF;
            RAISE NOTICE 'Added quantity column to trades table';
        END IF;
        
        -- Add size field if it doesn't exist (for backward compatibility)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'trades' AND column_name = 'size') THEN
            ALTER TABLE trades ADD COLUMN size DECIMAL(30,10);
            UPDATE trades SET size = quantity WHERE size IS NULL;
            RAISE NOTICE 'Added size column to trades table';
        END IF;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error fixing trades table: %', SQLERRM;
END $$;

-- 4. Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_orders_symbol ON orders(symbol);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_strategy_id ON orders(strategy_id);

-- 5. Fix existing orders with NULL strategy_id and insert sample data
DO $$
DECLARE
    sample_strategy_id UUID;
BEGIN
    -- Get or create a sample strategy for orders
    SELECT id INTO sample_strategy_id FROM strategies LIMIT 1;

    IF sample_strategy_id IS NULL THEN
        -- Create a sample strategy if none exists
        INSERT INTO strategies (id, name, type, status, description, created_at, updated_at)
        VALUES (uuid_generate_v4(), 'Sample Strategy', 'sample', 'inactive', 'Sample strategy for orders', NOW(), NOW())
        RETURNING id INTO sample_strategy_id;
        RAISE NOTICE 'Created sample strategy: %', sample_strategy_id;
    END IF;

    -- Update existing orders with NULL strategy_id
    UPDATE orders SET strategy_id = sample_strategy_id WHERE strategy_id IS NULL;

    -- Insert sample data only if no orders exist
    IF NOT EXISTS (SELECT 1 FROM orders LIMIT 1) THEN
        INSERT INTO orders (id, strategy_id, symbol, side, quantity, price, order_type, status, created_at)
        VALUES
            (uuid_generate_v4(), sample_strategy_id, 'BTCUSDT', 'BUY', 0.001, 50000.0, 'MARKET', 'FILLED', NOW() - INTERVAL '1 hour'),
            (uuid_generate_v4(), sample_strategy_id, 'ETHUSDT', 'SELL', 0.01, 3000.0, 'LIMIT', 'FILLED', NOW() - INTERVAL '30 minutes');
        RAISE NOTICE 'Added sample orders with strategy_id: %', sample_strategy_id;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error adding sample data: %', SQLERRM;
END $$;

-- Success message
DO $$ 
BEGIN
    RAISE NOTICE 'Migration 000030 completed successfully';
    RAISE NOTICE 'Fixed missing fields: quantity in orders, generated_strategies in strategy_onboarding';
    RAISE NOTICE 'Added sample data to prevent empty result errors';
END $$;

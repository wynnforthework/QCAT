-- 创建示例交易数据
-- 这个脚本会为已存在的策略创建一些示例交易记录

-- 首先确保我们有策略数据
DO $$
DECLARE
    strategy_count INTEGER;
    strategy_record RECORD;
    trade_id UUID;
    i INTEGER;
BEGIN
    -- 检查策略数量
    SELECT COUNT(*) INTO strategy_count FROM strategies;
    
    -- 如果没有策略，先创建一些
    IF strategy_count = 0 THEN
        INSERT INTO strategies (id, name, type, status, description, is_running, enabled, created_at, updated_at)
        VALUES 
            (uuid_generate_v4(), 'BTC动量策略', 'momentum', 'active', '基于移动平均线和RSI的BTC动量交易策略', true, true, NOW(), NOW()),
            (uuid_generate_v4(), 'ETH均值回归策略', 'mean_reversion', 'inactive', '基于布林带的ETH均值回归策略', false, true, NOW(), NOW()),
            (uuid_generate_v4(), 'SOL趋势跟踪策略', 'trend_following', 'inactive', '基于MACD的SOL趋势跟踪策略', false, true, NOW(), NOW());
    END IF;
    
    -- 为每个策略创建示例交易记录
    FOR strategy_record IN SELECT id, name FROM strategies LIMIT 3 LOOP
        -- 为每个策略创建20条交易记录
        FOR i IN 1..20 LOOP
            trade_id := uuid_generate_v4();
            
            -- 根据策略类型选择不同的交易对
            IF strategy_record.name LIKE '%BTC%' THEN
                INSERT INTO trades (
                    id, strategy_id, symbol, side, size, price, fee, fee_currency, created_at
                ) VALUES (
                    trade_id,
                    strategy_record.id,
                    'BTCUSDT',
                    CASE WHEN (i % 2) = 0 THEN 'BUY' ELSE 'SELL' END,
                    0.001 + (random() * 0.01), -- 0.001-0.011 BTC
                    45000 + (random() * 10000), -- 45000-55000 USDT
                    (45000 + (random() * 10000)) * (0.001 + (random() * 0.01)) * 0.001, -- 0.1% fee
                    'USDT',
                    NOW() - INTERVAL '1 hour' * i
                );
            ELSIF strategy_record.name LIKE '%ETH%' THEN
                INSERT INTO trades (
                    id, strategy_id, symbol, side, size, price, fee, fee_currency, created_at
                ) VALUES (
                    trade_id,
                    strategy_record.id,
                    'ETHUSDT',
                    CASE WHEN (i % 2) = 0 THEN 'BUY' ELSE 'SELL' END,
                    0.01 + (random() * 0.1), -- 0.01-0.11 ETH
                    2800 + (random() * 400), -- 2800-3200 USDT
                    (2800 + (random() * 400)) * (0.01 + (random() * 0.1)) * 0.001, -- 0.1% fee
                    'USDT',
                    NOW() - INTERVAL '1 hour' * i
                );
            ELSIF strategy_record.name LIKE '%SOL%' THEN
                INSERT INTO trades (
                    id, strategy_id, symbol, side, size, price, fee, fee_currency, created_at
                ) VALUES (
                    trade_id,
                    strategy_record.id,
                    'SOLUSDT',
                    CASE WHEN (i % 2) = 0 THEN 'BUY' ELSE 'SELL' END,
                    1 + (random() * 10), -- 1-11 SOL
                    90 + (random() * 20), -- 90-110 USDT
                    (90 + (random() * 20)) * (1 + (random() * 10)) * 0.001, -- 0.1% fee
                    'USDT',
                    NOW() - INTERVAL '1 hour' * i
                );
            ELSE
                -- 默认创建ADA交易
                INSERT INTO trades (
                    id, strategy_id, symbol, side, size, price, fee, fee_currency, created_at
                ) VALUES (
                    trade_id,
                    strategy_record.id,
                    'ADAUSDT',
                    CASE WHEN (i % 2) = 0 THEN 'BUY' ELSE 'SELL' END,
                    100 + (random() * 900), -- 100-1000 ADA
                    0.4 + (random() * 0.2), -- 0.4-0.6 USDT
                    (0.4 + (random() * 0.2)) * (100 + (random() * 900)) * 0.001, -- 0.1% fee
                    'USDT',
                    NOW() - INTERVAL '1 hour' * i
                );
            END IF;
        END LOOP;
    END LOOP;
    
    RAISE NOTICE '已为 % 个策略创建示例交易数据', strategy_count;
END $$;

-- 创建一些订单数据（如果表存在的话）
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'orders') THEN
        -- 为每个策略创建一些订单记录
        INSERT INTO orders (id, strategy_id, symbol, side, order_type, size, price, status, created_at, updated_at)
        SELECT 
            uuid_generate_v4(),
            s.id,
            CASE 
                WHEN s.name LIKE '%BTC%' THEN 'BTCUSDT'
                WHEN s.name LIKE '%ETH%' THEN 'ETHUSDT'
                WHEN s.name LIKE '%SOL%' THEN 'SOLUSDT'
                ELSE 'ADAUSDT'
            END,
            CASE WHEN (random() > 0.5) THEN 'BUY' ELSE 'SELL' END,
            CASE WHEN (random() > 0.7) THEN 'LIMIT' ELSE 'MARKET' END,
            CASE 
                WHEN s.name LIKE '%BTC%' THEN 0.001 + (random() * 0.01)
                WHEN s.name LIKE '%ETH%' THEN 0.01 + (random() * 0.1)
                WHEN s.name LIKE '%SOL%' THEN 1 + (random() * 10)
                ELSE 100 + (random() * 900)
            END,
            CASE 
                WHEN s.name LIKE '%BTC%' THEN 45000 + (random() * 10000)
                WHEN s.name LIKE '%ETH%' THEN 2800 + (random() * 400)
                WHEN s.name LIKE '%SOL%' THEN 90 + (random() * 20)
                ELSE 0.4 + (random() * 0.2)
            END,
            CASE 
                WHEN (random() > 0.8) THEN 'pending'
                WHEN (random() > 0.6) THEN 'filled'
                WHEN (random() > 0.4) THEN 'partially_filled'
                ELSE 'cancelled'
            END,
            NOW() - INTERVAL '1 hour' * (random() * 24),
            NOW() - INTERVAL '1 hour' * (random() * 24)
        FROM strategies s, generate_series(1, 5) -- 每个策略5个订单
        LIMIT 15; -- 总共最多15个订单
        
        RAISE NOTICE '已创建示例订单数据';
    END IF;
END $$;

-- 显示创建的数据统计
SELECT 
    'strategies' as table_name,
    COUNT(*) as record_count
FROM strategies
UNION ALL
SELECT 
    'trades' as table_name,
    COUNT(*) as record_count
FROM trades
UNION ALL
SELECT 
    'orders' as table_name,
    COUNT(*) as record_count
FROM orders
WHERE EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'orders');

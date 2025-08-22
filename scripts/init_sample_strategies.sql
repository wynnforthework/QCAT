-- 初始化示例策略数据
-- 这个脚本会创建一些示例策略，用于演示和测试

-- 确保strategies表有必要的字段
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS is_running BOOLEAN DEFAULT false;
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS enabled BOOLEAN DEFAULT true;
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS parameters JSONB DEFAULT '{}';

-- 插入示例策略（包含新字段）
INSERT INTO strategies (id, name, description, type, status, is_running, enabled, parameters, created_at, updated_at) VALUES
(uuid_generate_v4(), 'BTC动量策略', '基于比特币价格动量的交易策略，使用移动平均线和RSI指标', 'momentum', 'active', true, true, '{"symbol": "BTCUSDT", "timeframe": "1h"}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
(uuid_generate_v4(), 'ETH均值回归策略', '以太坊均值回归策略，利用价格偏离均值时的回归特性', 'mean_reversion', 'active', false, true, '{"symbol": "ETHUSDT", "timeframe": "4h"}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
(uuid_generate_v4(), 'SOL趋势跟踪策略', 'Solana趋势跟踪策略，使用布林带和MACD指标', 'trend_following', 'inactive', false, false, '{"symbol": "SOLUSDT", "timeframe": "1h"}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
(uuid_generate_v4(), 'ADA网格交易策略', 'Cardano网格交易策略，在震荡市场中获取收益', 'grid_trading', 'active', false, true, '{"symbol": "ADAUSDT", "timeframe": "15m"}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
(uuid_generate_v4(), 'BNB套利策略', 'Binance Coin套利策略，利用不同交易对之间的价差', 'arbitrage', 'inactive', false, true, '{"symbol": "BNBUSDT", "timeframe": "5m"}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
(uuid_generate_v4(), 'DOGE高频交易策略', 'Dogecoin高频交易策略，基于微小价格波动', 'high_frequency', 'testing', false, true, '{"symbol": "DOGEUSDT", "timeframe": "1m"}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
(uuid_generate_v4(), 'MATIC波段交易策略', 'Polygon波段交易策略，捕捉中期价格波动', 'swing_trading', 'active', true, true, '{"symbol": "MATICUSDT", "timeframe": "1d"}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
(uuid_generate_v4(), 'AVAX突破策略', 'Avalanche突破策略，在价格突破关键阻力位时入场', 'breakout', 'inactive', false, true, '{"symbol": "AVAXUSDT", "timeframe": "4h"}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;

-- 为策略创建版本信息
WITH strategy_ids AS (
    SELECT id, name FROM strategies WHERE name IN (
        'BTC动量策略', 'ETH均值回归策略', 'SOL趋势跟踪策略', 'ADA网格交易策略'
    )
)
INSERT INTO strategy_versions (id, strategy_id, version, parameters, performance_metrics, created_at)
SELECT 
    uuid_generate_v4(),
    s.id,
    'v1.0.0',
    CASE 
        WHEN s.name = 'BTC动量策略' THEN '{"ma_short": 10, "ma_long": 30, "rsi_period": 14, "rsi_oversold": 30, "rsi_overbought": 70}'::jsonb
        WHEN s.name = 'ETH均值回归策略' THEN '{"lookback_period": 20, "std_dev_multiplier": 2, "entry_threshold": 1.5, "exit_threshold": 0.5}'::jsonb
        WHEN s.name = 'SOL趋势跟踪策略' THEN '{"bb_period": 20, "bb_std": 2, "macd_fast": 12, "macd_slow": 26, "macd_signal": 9}'::jsonb
        WHEN s.name = 'ADA网格交易策略' THEN '{"grid_levels": 10, "grid_spacing": 0.02, "base_order_size": 100, "profit_target": 0.01}'::jsonb
    END,
    CASE 
        WHEN s.name = 'BTC动量策略' THEN '{"total_return": 0.15, "sharpe_ratio": 1.8, "max_drawdown": 0.08, "win_rate": 0.65}'::jsonb
        WHEN s.name = 'ETH均值回归策略' THEN '{"total_return": 0.12, "sharpe_ratio": 1.5, "max_drawdown": 0.06, "win_rate": 0.58}'::jsonb
        WHEN s.name = 'SOL趋势跟踪策略' THEN '{"total_return": 0.22, "sharpe_ratio": 2.1, "max_drawdown": 0.12, "win_rate": 0.72}'::jsonb
        WHEN s.name = 'ADA网格交易策略' THEN '{"total_return": 0.08, "sharpe_ratio": 1.2, "max_drawdown": 0.04, "win_rate": 0.78}'::jsonb
    END,
    CURRENT_TIMESTAMP
FROM strategy_ids s
ON CONFLICT (id) DO NOTHING;

-- 为策略创建参数配置
WITH strategy_ids AS (
    SELECT id, name FROM strategies WHERE name IN (
        'BTC动量策略', 'ETH均值回归策略', 'SOL趋势跟踪策略', 'ADA网格交易策略'
    )
)
INSERT INTO strategy_params (id, strategy_id, param_name, param_value, param_type, created_at)
SELECT 
    uuid_generate_v4(),
    s.id,
    param.name,
    param.value,
    param.type,
    CURRENT_TIMESTAMP
FROM strategy_ids s
CROSS JOIN (
    VALUES 
        ('symbol', 'BTCUSDT', 'string'),
        ('timeframe', '1h', 'string'),
        ('position_size', '0.1', 'float'),
        ('stop_loss', '0.02', 'float'),
        ('take_profit', '0.04', 'float')
) AS param(name, value, type)
WHERE s.name = 'BTC动量策略'

UNION ALL

SELECT 
    uuid_generate_v4(),
    s.id,
    param.name,
    param.value,
    param.type,
    CURRENT_TIMESTAMP
FROM strategy_ids s
CROSS JOIN (
    VALUES 
        ('symbol', 'ETHUSDT', 'string'),
        ('timeframe', '4h', 'string'),
        ('position_size', '0.15', 'float'),
        ('stop_loss', '0.03', 'float'),
        ('take_profit', '0.05', 'float')
) AS param(name, value, type)
WHERE s.name = 'ETH均值回归策略'

UNION ALL

SELECT 
    uuid_generate_v4(),
    s.id,
    param.name,
    param.value,
    param.type,
    CURRENT_TIMESTAMP
FROM strategy_ids s
CROSS JOIN (
    VALUES 
        ('symbol', 'SOLUSDT', 'string'),
        ('timeframe', '1h', 'string'),
        ('position_size', '0.2', 'float'),
        ('stop_loss', '0.04', 'float'),
        ('take_profit', '0.06', 'float')
) AS param(name, value, type)
WHERE s.name = 'SOL趋势跟踪策略'

UNION ALL

SELECT 
    uuid_generate_v4(),
    s.id,
    param.name,
    param.value,
    param.type,
    CURRENT_TIMESTAMP
FROM strategy_ids s
CROSS JOIN (
    VALUES 
        ('symbol', 'ADAUSDT', 'string'),
        ('timeframe', '15m', 'string'),
        ('position_size', '0.05', 'float'),
        ('stop_loss', '0.01', 'float'),
        ('take_profit', '0.02', 'float')
) AS param(name, value, type)
WHERE s.name = 'ADA网格交易策略'
ON CONFLICT (id) DO NOTHING;

-- 输出结果
DO $$
DECLARE
    strategy_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO strategy_count FROM strategies;
    RAISE NOTICE '✅ 示例策略初始化完成！当前策略总数: %', strategy_count;
    RAISE NOTICE '📊 策略状态分布:';
    
    FOR rec IN 
        SELECT status, COUNT(*) as count 
        FROM strategies 
        GROUP BY status 
        ORDER BY status
    LOOP
        RAISE NOTICE '   %: % 个策略', rec.status, rec.count;
    END LOOP;
END $$;

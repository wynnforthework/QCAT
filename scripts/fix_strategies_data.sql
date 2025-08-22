-- 修复策略数据脚本
-- 用于解决分享结果页面策略选择问题

-- 1. 确保strategies表有必要的字段
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS is_running BOOLEAN DEFAULT false;
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS enabled BOOLEAN DEFAULT true;
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS parameters JSONB DEFAULT '{}';

-- 2. 清理现有数据（如果存在）
DELETE FROM strategies WHERE name IN (
    'BTC动量策略', 'ETH均值回归策略', 'SOL趋势跟踪策略', 
    'ADA网格交易策略', 'MATIC波段交易策略'
);

-- 3. 插入测试策略数据
INSERT INTO strategies (id, name, description, type, status, is_running, enabled, parameters, created_at, updated_at) VALUES
(uuid_generate_v4(), 'BTC动量策略', '基于比特币价格动量的交易策略，使用移动平均线和RSI指标', 'momentum', 'active', true, true, '{"symbol": "BTCUSDT", "timeframe": "1h", "ma_short": 10, "ma_long": 30}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
(uuid_generate_v4(), 'ETH均值回归策略', '以太坊均值回归策略，利用价格偏离均值时的回归特性', 'mean_reversion', 'active', false, true, '{"symbol": "ETHUSDT", "timeframe": "4h", "lookback": 20}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
(uuid_generate_v4(), 'SOL趋势跟踪策略', 'Solana趋势跟踪策略，使用布林带和MACD指标', 'trend_following', 'inactive', false, false, '{"symbol": "SOLUSDT", "timeframe": "1h", "bb_period": 20}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
(uuid_generate_v4(), 'ADA网格交易策略', 'Cardano网格交易策略，在震荡市场中获取收益', 'grid_trading', 'active', false, true, '{"symbol": "ADAUSDT", "timeframe": "15m", "grid_levels": 10}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
(uuid_generate_v4(), 'MATIC波段交易策略', 'Polygon波段交易策略，捕捉中期价格波动', 'swing_trading', 'active', true, true, '{"symbol": "MATICUSDT", "timeframe": "1d", "swing_period": 14}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

-- 4. 验证数据插入
SELECT 
    name, 
    type, 
    status, 
    is_running, 
    enabled,
    CASE 
        WHEN is_running AND enabled THEN 'running'
        WHEN enabled THEN 'stopped'
        ELSE 'disabled'
    END as runtime_status
FROM strategies 
WHERE name IN (
    'BTC动量策略', 'ETH均值回归策略', 'SOL趋势跟踪策略', 
    'ADA网格交易策略', 'MATIC波段交易策略'
)
ORDER BY name;

-- 5. 显示统计信息
SELECT 
    'Total strategies' as metric,
    COUNT(*) as count
FROM strategies
UNION ALL
SELECT 
    'Enabled strategies' as metric,
    COUNT(*) as count
FROM strategies WHERE enabled = true
UNION ALL
SELECT 
    'Running strategies' as metric,
    COUNT(*) as count
FROM strategies WHERE is_running = true AND enabled = true;

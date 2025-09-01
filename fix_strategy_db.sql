-- 修复策略API数据库问题
-- 确保strategies表存在并创建示例数据

-- 创建UUID扩展（如果不存在）
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 创建strategies表（如果不存在）
CREATE TABLE IF NOT EXISTS strategies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(20) DEFAULT 'inactive',
    is_running BOOLEAN DEFAULT false,
    enabled BOOLEAN DEFAULT true,
    performance JSONB,
    sharpe_ratio DECIMAL(10,4),
    max_drawdown DECIMAL(10,4),
    optimization_config JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引（如果不存在）
CREATE INDEX IF NOT EXISTS idx_strategies_name ON strategies(name);
CREATE INDEX IF NOT EXISTS idx_strategies_type ON strategies(type);
CREATE INDEX IF NOT EXISTS idx_strategies_status ON strategies(status);
CREATE INDEX IF NOT EXISTS idx_strategies_enabled ON strategies(enabled);
CREATE INDEX IF NOT EXISTS idx_strategies_created_at ON strategies(created_at);

-- 插入示例策略数据（如果表为空）
DO $$ 
DECLARE
    strategy_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO strategy_count FROM strategies;
    
    IF strategy_count = 0 THEN
        -- 插入示例策略
        INSERT INTO strategies (name, description, type, status, is_running, enabled, sharpe_ratio, max_drawdown, created_at, updated_at) VALUES
        ('动量突破策略', '基于价格动量的突破交易策略，适用于趋势明显的市场环境', 'momentum', 'active', true, true, 1.25, 0.08, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
        ('均值回归策略', '利用价格偏离均值后的回归特性进行交易', 'mean_reversion', 'inactive', false, true, 0.95, 0.12, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
        ('网格交易策略', '在震荡市场中通过网格交易获取收益', 'grid', 'active', true, true, 1.15, 0.06, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
        ('套利策略', '利用不同交易所或合约间的价差进行套利', 'arbitrage', 'inactive', false, true, 2.10, 0.03, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
        ('马丁格尔策略', '基于马丁格尔理论的加仓策略', 'martingale', 'inactive', false, false, 0.75, 0.25, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
        
        RAISE NOTICE 'Inserted % sample strategies', 5;
    ELSE
        RAISE NOTICE 'Strategies table already contains % records', strategy_count;
    END IF;
END $$;

-- 创建其他相关表（如果不存在）

-- 策略性能表
CREATE TABLE IF NOT EXISTS strategy_performance (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    pnl DECIMAL(20,8) NOT NULL,
    return_rate DECIMAL(10,6) NOT NULL,
    sharpe_ratio DECIMAL(10,4),
    max_drawdown DECIMAL(10,4),
    win_rate DECIMAL(5,4),
    total_trades INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(strategy_id, date)
);

-- 策略参数表
CREATE TABLE IF NOT EXISTS strategy_parameters (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
    parameter_name VARCHAR(100) NOT NULL,
    parameter_value TEXT NOT NULL,
    parameter_type VARCHAR(50) DEFAULT 'string',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(strategy_id, parameter_name)
);

-- 策略执行日志表
CREATE TABLE IF NOT EXISTS strategy_execution_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
    log_level VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建相关索引
CREATE INDEX IF NOT EXISTS idx_strategy_performance_strategy_id ON strategy_performance(strategy_id);
CREATE INDEX IF NOT EXISTS idx_strategy_performance_date ON strategy_performance(date);
CREATE INDEX IF NOT EXISTS idx_strategy_parameters_strategy_id ON strategy_parameters(strategy_id);
CREATE INDEX IF NOT EXISTS idx_strategy_execution_logs_strategy_id ON strategy_execution_logs(strategy_id);
CREATE INDEX IF NOT EXISTS idx_strategy_execution_logs_created_at ON strategy_execution_logs(created_at);

-- 验证表创建
DO $$ 
DECLARE
    table_count INTEGER;
    strategy_count INTEGER;
BEGIN
    -- 检查strategies表
    SELECT COUNT(*) INTO table_count FROM information_schema.tables 
    WHERE table_name IN ('strategies', 'strategy_performance', 'strategy_parameters', 'strategy_execution_logs');
    
    SELECT COUNT(*) INTO strategy_count FROM strategies;
    
    RAISE NOTICE 'Created % strategy-related tables', table_count;
    RAISE NOTICE 'Strategies table contains % records', strategy_count;
    
    IF table_count >= 4 AND strategy_count > 0 THEN
        RAISE NOTICE 'Strategy database schema successfully initialized';
    ELSE
        RAISE WARNING 'Strategy database schema initialization may be incomplete';
    END IF;
END $$;

-- 显示所有策略
SELECT id, name, type, status, is_running, enabled, created_at FROM strategies ORDER BY created_at;

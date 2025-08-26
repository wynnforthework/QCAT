-- 创建策略黑名单表
-- Strategy Blacklist Table Creation Script

-- 删除已存在的表（如果存在）
DROP TABLE IF EXISTS strategy_blacklist;

-- 创建策略黑名单表
CREATE TABLE strategy_blacklist (
    id SERIAL PRIMARY KEY,
    strategy_id VARCHAR(255) NOT NULL UNIQUE,
    reason TEXT NOT NULL,
    blocked_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    blocked_by VARCHAR(100) NOT NULL DEFAULT 'system',
    permanent BOOLEAN NOT NULL DEFAULT true,
    expires_at TIMESTAMP WITH TIME ZONE NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 创建索引
CREATE INDEX idx_strategy_blacklist_strategy_id ON strategy_blacklist(strategy_id);
CREATE INDEX idx_strategy_blacklist_blocked_at ON strategy_blacklist(blocked_at);
CREATE INDEX idx_strategy_blacklist_expires_at ON strategy_blacklist(expires_at);
CREATE INDEX idx_strategy_blacklist_permanent ON strategy_blacklist(permanent);

-- 添加注释
COMMENT ON TABLE strategy_blacklist IS '策略黑名单表，用于记录被禁用的策略';
COMMENT ON COLUMN strategy_blacklist.strategy_id IS '策略ID';
COMMENT ON COLUMN strategy_blacklist.reason IS '禁用原因';
COMMENT ON COLUMN strategy_blacklist.blocked_at IS '禁用时间';
COMMENT ON COLUMN strategy_blacklist.blocked_by IS '禁用者（系统/用户）';
COMMENT ON COLUMN strategy_blacklist.permanent IS '是否永久禁用';
COMMENT ON COLUMN strategy_blacklist.expires_at IS '过期时间（仅当permanent=false时有效）';

-- 创建触发器函数来自动更新 updated_at 字段
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 创建触发器
CREATE TRIGGER update_strategy_blacklist_updated_at 
    BEFORE UPDATE ON strategy_blacklist 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- 插入一些示例数据（可选）
-- INSERT INTO strategy_blacklist (strategy_id, reason, blocked_by) 
-- VALUES ('test_strategy_001', '风控测试：持仓过多', 'risk_control');

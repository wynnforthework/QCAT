-- 创建紧急停止相关表
-- Emergency Stop Tables Creation Script

-- 删除已存在的表（如果存在）
DROP TABLE IF EXISTS emergency_stop_reset_events;
DROP TABLE IF EXISTS emergency_stop_events;

-- 创建紧急停止事件表
CREATE TABLE emergency_stop_events (
    id SERIAL PRIMARY KEY,
    reason TEXT NOT NULL,
    total_strategies INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    stopped_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 创建紧急停止重置事件表
CREATE TABLE emergency_stop_reset_events (
    id SERIAL PRIMARY KEY,
    reason TEXT NOT NULL,
    reset_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 创建索引
CREATE INDEX idx_emergency_stop_events_stopped_at ON emergency_stop_events(stopped_at);
CREATE INDEX idx_emergency_stop_events_created_at ON emergency_stop_events(created_at);
CREATE INDEX idx_emergency_stop_reset_events_reset_at ON emergency_stop_reset_events(reset_at);
CREATE INDEX idx_emergency_stop_reset_events_created_at ON emergency_stop_reset_events(created_at);

-- 添加注释
COMMENT ON TABLE emergency_stop_events IS '紧急停止事件记录表';
COMMENT ON COLUMN emergency_stop_events.reason IS '紧急停止原因';
COMMENT ON COLUMN emergency_stop_events.total_strategies IS '总策略数量';
COMMENT ON COLUMN emergency_stop_events.failed_count IS '停止失败的策略数量';
COMMENT ON COLUMN emergency_stop_events.stopped_at IS '紧急停止时间';

COMMENT ON TABLE emergency_stop_reset_events IS '紧急停止重置事件记录表';
COMMENT ON COLUMN emergency_stop_reset_events.reason IS '重置原因';
COMMENT ON COLUMN emergency_stop_reset_events.reset_at IS '重置时间';

-- 创建触发器函数来自动更新 updated_at 字段
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 创建触发器
CREATE TRIGGER update_emergency_stop_events_updated_at 
    BEFORE UPDATE ON emergency_stop_events 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

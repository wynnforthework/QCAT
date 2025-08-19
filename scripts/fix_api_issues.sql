-- 修复API接口问题的SQL脚本
-- 创建缺失的数据库表和测试数据

-- 1. 创建hotlist_scores表 (如果不存在)
CREATE TABLE IF NOT EXISTS hotlist_scores (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    symbol VARCHAR(20) NOT NULL,
    vol_jump_score DECIMAL(10,6) DEFAULT 0,
    turnover_score DECIMAL(10,6) DEFAULT 0,
    oi_change_score DECIMAL(10,6) DEFAULT 0,
    funding_z_score DECIMAL(10,6) DEFAULT 0,
    regime_shift_score DECIMAL(10,6) DEFAULT 0,
    total_score DECIMAL(10,6) DEFAULT 0,
    risk_level VARCHAR(20) DEFAULT 'medium',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(symbol)
);

-- 2. 创建optimizer_tasks表 (如果不存在)
CREATE TABLE IF NOT EXISTS optimizer_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id UUID,
    task_type VARCHAR(50) NOT NULL DEFAULT 'optimization',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    parameters JSONB,
    results JSONB,
    error_message TEXT,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. 创建audit_logs表 (如果不存在)
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50),
    resource_id VARCHAR(100),
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    success BOOLEAN DEFAULT true
);

-- 4. 插入测试数据

-- 热门符号测试数据
INSERT INTO hotlist_scores (symbol, vol_jump_score, turnover_score, oi_change_score, funding_z_score, regime_shift_score, total_score, risk_level) 
VALUES 
    ('BTCUSDT', 0.85, 0.92, 0.78, 0.65, 0.88, 0.82, 'high'),
    ('ETHUSDT', 0.75, 0.88, 0.82, 0.70, 0.85, 0.80, 'high'),
    ('ADAUSDT', 0.65, 0.75, 0.68, 0.55, 0.72, 0.67, 'medium'),
    ('SOLUSDT', 0.88, 0.85, 0.90, 0.75, 0.82, 0.84, 'high'),
    ('DOTUSDT', 0.55, 0.68, 0.62, 0.48, 0.65, 0.60, 'medium')
ON CONFLICT (symbol) DO UPDATE SET
    vol_jump_score = EXCLUDED.vol_jump_score,
    turnover_score = EXCLUDED.turnover_score,
    oi_change_score = EXCLUDED.oi_change_score,
    funding_z_score = EXCLUDED.funding_z_score,
    regime_shift_score = EXCLUDED.regime_shift_score,
    total_score = EXCLUDED.total_score,
    risk_level = EXCLUDED.risk_level,
    updated_at = CURRENT_TIMESTAMP;

-- 优化任务测试数据
INSERT INTO optimizer_tasks (task_type, status, parameters, results) 
VALUES 
    ('grid_search', 'completed', 
     '{"param_ranges": {"lookback": [10, 20, 30], "threshold": [0.01, 0.02, 0.03]}}',
     '{"best_params": {"lookback": 20, "threshold": 0.02}, "best_score": 0.85}'),
    ('bayesian', 'running', 
     '{"param_ranges": {"alpha": [0.1, 1.0], "beta": [0.1, 1.0]}}',
     NULL),
    ('random_search', 'pending', 
     '{"param_ranges": {"window": [5, 15], "multiplier": [1.5, 3.0]}}',
     NULL);

-- 审计日志测试数据
INSERT INTO audit_logs (action, resource_type, resource_id, details, ip_address, user_agent, success) 
VALUES 
    ('login', 'user', 'admin', '{"login_method": "password"}', '127.0.0.1', 'Mozilla/5.0', true),
    ('create_strategy', 'strategy', 'strategy-001', '{"name": "Test Strategy", "type": "momentum"}', '127.0.0.1', 'Mozilla/5.0', true),
    ('update_portfolio', 'portfolio', 'portfolio-001', '{"action": "rebalance", "mode": "bandit"}', '127.0.0.1', 'Mozilla/5.0', true),
    ('delete_strategy', 'strategy', 'strategy-002', '{"name": "Old Strategy"}', '127.0.0.1', 'Mozilla/5.0', false);

-- 5. 创建索引以提高性能
CREATE INDEX IF NOT EXISTS idx_hotlist_scores_total_score ON hotlist_scores(total_score DESC);
CREATE INDEX IF NOT EXISTS idx_hotlist_scores_symbol ON hotlist_scores(symbol);
CREATE INDEX IF NOT EXISTS idx_optimizer_tasks_status ON optimizer_tasks(status);
CREATE INDEX IF NOT EXISTS idx_optimizer_tasks_created_at ON optimizer_tasks(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type ON audit_logs(resource_type);

-- 6. 更新统计信息
ANALYZE hotlist_scores;
ANALYZE optimizer_tasks;
ANALYZE audit_logs;

-- 修复admin用户登录问题的SQL脚本
-- 这个脚本将创建或更新admin用户，使用正确的密码哈希

-- 首先确保UUID扩展存在
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 创建users表（如果不存在）
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    last_login TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建user_sessions表（如果不存在）
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token VARCHAR(500) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_refresh_token ON user_sessions(refresh_token);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);

-- 删除现有的admin用户（如果存在）
DELETE FROM users WHERE username = 'admin';

-- 插入新的admin用户，使用正确的密码哈希 (admin123)
INSERT INTO users (username, email, password_hash, role, status, created_at, updated_at) 
VALUES (
    'admin', 
    'admin@qcat.local', 
    '$2a$10$N9qo8uLOickgx2ZMRZoMye.IjPeOXe.2p5l/q/FQcre8HdkL6Q262', 
    'admin', 
    'active', 
    CURRENT_TIMESTAMP, 
    CURRENT_TIMESTAMP
);

-- 插入测试用户
INSERT INTO users (username, email, password_hash, role, status, created_at, updated_at) 
VALUES (
    'testuser', 
    'test@qcat.local', 
    '$2a$10$N9qo8uLOickgx2ZMRZoMye.IjPeOXe.2p5l/q/FQcre8HdkL6Q262', 
    'user', 
    'active', 
    CURRENT_TIMESTAMP, 
    CURRENT_TIMESTAMP
) ON CONFLICT (username) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    updated_at = CURRENT_TIMESTAMP;

-- 插入demo用户 (密码: demo123)
INSERT INTO users (username, email, password_hash, role, status, created_at, updated_at) 
VALUES (
    'demo', 
    'demo@qcat.local', 
    '$2a$10$8K1p/a0dhrxiH8Tf4di1HuP4lxvlmOyqjLxYiMyIlSaw1uYwy55jG', 
    'user', 
    'active', 
    CURRENT_TIMESTAMP, 
    CURRENT_TIMESTAMP
) ON CONFLICT (username) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    updated_at = CURRENT_TIMESTAMP;

-- 验证用户是否创建成功
SELECT 
    username, 
    email, 
    role, 
    status, 
    created_at,
    CASE 
        WHEN password_hash = '$2a$10$N9qo8uLOickgx2ZMRZoMye.IjPeOXe.2p5l/q/FQcre8HdkL6Q262' THEN 'admin123'
        WHEN password_hash = '$2a$10$8K1p/a0dhrxiH8Tf4di1HuP4lxvlmOyqjLxYiMyIlSaw1uYwy55jG' THEN 'demo123'
        ELSE 'unknown'
    END as password_hint
FROM users 
WHERE username IN ('admin', 'testuser', 'demo')
ORDER BY username;

-- 修复JWT认证数据库问题
-- 确保用户表存在并创建admin用户

-- 创建UUID扩展（如果不存在）
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 创建用户表（如果不存在）
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

-- 创建用户会话表（如果不存在）
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token VARCHAR(500) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引（如果不存在）
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_refresh_token ON user_sessions(refresh_token);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);

-- 插入或更新admin用户
-- 密码: admin123
-- 哈希: $2a$10$N9qo8uLOickgx2ZMRZoMye.IjPeOXe.2p5l/q/FQcre8HdkL6Q262
INSERT INTO users (username, email, password_hash, role, status, created_at, updated_at) 
VALUES ('admin', 'admin@qcat.local', '$2a$10$N9qo8uLOickgx2ZMRZoMye.IjPeOXe.2p5l/q/FQcre8HdkL6Q262', 'admin', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (username) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    email = EXCLUDED.email,
    role = EXCLUDED.role,
    status = EXCLUDED.status,
    updated_at = CURRENT_TIMESTAMP;

-- 插入测试用户
-- 密码: admin123 (相同密码用于测试)
INSERT INTO users (username, email, password_hash, role, status, created_at, updated_at) 
VALUES ('testuser', 'test@qcat.local', '$2a$10$N9qo8uLOickgx2ZMRZoMye.IjPeOXe.2p5l/q/FQcre8HdkL6Q262', 'user', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (username) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    updated_at = CURRENT_TIMESTAMP;

-- 插入demo用户
-- 密码: demo123
INSERT INTO users (username, email, password_hash, role, status, created_at, updated_at) 
VALUES ('demo', 'demo@qcat.local', '$2a$10$8K1p/a0dhrxiH8Tf4di1HuP4lxvlmOyqjLxYiMyIlSaw1uYwy55jG', 'user', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (username) DO NOTHING;

-- 验证用户创建
DO $$ 
DECLARE
    user_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO user_count FROM users;
    RAISE NOTICE 'Total users in database: %', user_count;
    
    IF user_count > 0 THEN
        RAISE NOTICE 'Users table successfully initialized with % users', user_count;
    ELSE
        RAISE WARNING 'No users found in database after initialization';
    END IF;
END $$;

-- 显示所有用户
SELECT username, email, role, status, created_at FROM users ORDER BY created_at;

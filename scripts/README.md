# 数据库迁移和用户管理脚本

这个目录包含了用于修复默认账户登录问题的脚本和迁移文件。

## 问题描述

API服务器返回401未授权错误，因为默认的admin用户密码哈希不正确，导致无法登录。

## 解决方案

### 方法1：运行数据库迁移（推荐）

如果你有现有的迁移系统，运行新的迁移文件：

```bash
# 运行迁移（如果你有migrate工具）
migrate -path internal/database/migrations -database "postgres://user:password@localhost/dbname?sslmode=disable" up

# 或者如果你有自定义的迁移命令
go run cmd/migrate/main.go
```

### 方法2：运行用户迁移脚本

使用提供的Go脚本直接修复用户数据：

```bash
# 设置数据库连接环境变量（可选，有默认值）
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=qcat_user
export DB_PASSWORD=qcat_password
export DB_NAME=qcat

# 运行用户迁移脚本
go run scripts/run_user_migration.go
```

### 方法3：手动SQL修复

直接在数据库中执行以下SQL：

```sql
-- 更新admin用户密码哈希
UPDATE users 
SET password_hash = '$2a$10$N9qo8uLOickgx2ZMRZoMye.IjPeOXe.2p5l/q/FQcre8HdkL6Q262',
    updated_at = CURRENT_TIMESTAMP
WHERE username = 'admin';

-- 如果admin用户不存在，创建它
INSERT INTO users (username, email, password_hash, role, status, created_at, updated_at) 
VALUES ('admin', 'admin@qcat.local', '$2a$10$N9qo8uLOickgx2ZMRZoMye.IjPeOXe.2p5l/q/FQcre8HdkL6Q262', 'admin', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (username) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    updated_at = CURRENT_TIMESTAMP;
```

## 默认用户账户

修复后，以下用户账户将可用：

| 用户名 | 密码 | 角色 | 邮箱 |
|--------|------|------|------|
| admin | admin123 | admin | admin@qcat.local |
| testuser | admin123 | user | test@qcat.local |
| demo | demo123 | user | demo@qcat.local |

## 测试登录

修复后，你可以使用以下方式测试登录：

### 1. 使用curl测试

```bash
# 登录获取JWT token
curl -X POST http://localhost:8082/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }'

# 使用token访问dashboard
curl -X GET http://localhost:8082/api/v1/dashboard \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN_HERE"
```

### 2. 使用浏览器测试

1. 访问 http://localhost:8082/api/v1/auth/login
2. 使用POST请求发送登录数据
3. 获取返回的access_token
4. 在请求头中添加 `Authorization: Bearer <token>` 访问其他API

## 工具脚本

- `generate_password_hash.go`: 生成bcrypt密码哈希
- `verify_password.go`: 验证密码哈希是否正确
- `run_user_migration.go`: 完整的用户迁移脚本

## 故障排除

### 如果仍然无法登录：

1. 检查数据库连接是否正常
2. 确认users表存在且有数据
3. 验证密码哈希是否正确：
   ```bash
   go run scripts/verify_password.go
   ```
4. 检查API服务器日志中的错误信息
5. 确认JWT密钥配置正确

### 常见错误：

- **数据库连接失败**: 检查数据库服务是否运行，连接参数是否正确
- **表不存在**: 运行完整的数据库迁移
- **密码哈希不匹配**: 使用提供的脚本重新生成密码哈希
- **JWT验证失败**: 检查JWT_SECRET环境变量是否设置

## 安全注意事项

- 在生产环境中，请更改默认密码
- 确保JWT_SECRET使用强随机字符串
- 定期轮换密码和JWT密钥
- 考虑启用双因素认证

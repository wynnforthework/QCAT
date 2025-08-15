# QCAT系统运维手册

## 1. 系统架构

### 1.1 组件说明
- **QCAT应用**: 主应用服务，提供API和WebSocket接口
- **PostgreSQL**: 主数据库，存储策略、交易、风控等数据
- **Redis**: 缓存服务，存储实时行情、会话等数据
- **Prometheus**: 监控系统，收集和存储指标数据
- **Grafana**: 监控面板，可视化展示系统状态
- **AlertManager**: 告警系统，处理告警通知
- **Nginx**: 反向代理，提供SSL终止和负载均衡

### 1.2 网络架构
```
Internet
    ↓
Nginx (80/443)
    ↓
QCAT App (8082)
    ↓
PostgreSQL (5432) + Redis (6379)
```

## 2. 部署指南

### 2.1 环境准备
1. 安装Docker和Docker Compose
2. 准备SSL证书（生产环境必需）
3. 配置环境变量

### 2.2 部署步骤
```bash
# 1. 克隆代码
git clone <repository-url>
cd qcat

# 2. 配置环境变量
cp deploy/env.example .env
# 编辑.env文件，设置正确的密码和密钥

# 3. 创建SSL证书目录
mkdir -p deploy/ssl

# 4. 启动服务
docker-compose -f deploy/docker-compose.prod.yml up -d

# 5. 检查服务状态
docker-compose -f deploy/docker-compose.prod.yml ps
```

### 2.3 验证部署
```bash
# 检查应用健康状态
curl http://localhost:8082/health

# 检查数据库连接
docker exec qcat_postgres psql -U qcat_user -d qcat -c "SELECT 1;"

# 检查Redis连接
docker exec qcat_redis redis-cli -a $REDIS_PASSWORD ping
```

## 3. 监控告警

### 3.1 访问监控面板
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **AlertManager**: http://localhost:9093

### 3.2 关键指标
- 系统健康状态: `up{job="qcat"}`
- HTTP请求率: `rate(http_requests_total[5m])`
- 响应时间: `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))`
- 数据库连接数: `pg_stat_database_numbackends`
- Redis内存使用: `redis_memory_used_bytes`

### 3.3 告警规则
- 应用宕机告警
- 高错误率告警
- 响应时间超时告警
- 数据库连接数告警
- 内存使用告警

## 4. 备份恢复

### 4.1 自动备份
```bash
# 设置定时备份（每天凌晨2点）
0 2 * * * /app/backup/backup.sh backup
```

### 4.2 手动备份
```bash
# 备份数据库
docker exec qcat_app /app/backup/backup.sh backup

# 恢复数据库
docker exec qcat_app /app/backup/backup.sh restore qcat_backup_20241201_020000.sql.gz

# 列出备份文件
docker exec qcat_app /app/backup/backup.sh list
```

### 4.3 备份验证
```bash
# 验证备份文件完整性
gunzip -t backup_file.sql.gz

# 测试恢复（在测试环境）
docker exec qcat_app /app/backup/backup.sh restore backup_file.sql.gz
```

## 5. 日志管理

### 5.1 日志位置
- 应用日志: `/app/logs/qcat.log`
- Nginx日志: `/var/log/nginx/`
- 数据库日志: Docker容器内部

### 5.2 日志轮转
```bash
# 配置logrotate
cat > /etc/logrotate.d/qcat << EOF
/app/logs/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 qcat qcat
    postrotate
        docker exec qcat_app kill -USR1 1
    endscript
}
EOF
```

### 5.3 日志分析
```bash
# 查看错误日志
tail -f /app/logs/qcat.log | grep ERROR

# 查看慢查询
tail -f /app/logs/qcat.log | grep "slow query"

# 统计API调用
grep "API call" /app/logs/qcat.log | wc -l
```

## 6. 故障处理

### 6.1 常见问题

#### 应用无法启动
```bash
# 检查日志
docker logs qcat_app

# 检查配置
docker exec qcat_app cat /app/configs/config.yaml

# 检查数据库连接
docker exec qcat_app /app/qcat migrate
```

#### 数据库连接失败
```bash
# 检查数据库状态
docker exec qcat_postgres pg_isready -U qcat_user -d qcat

# 检查网络连接
docker exec qcat_app ping postgres

# 检查环境变量
docker exec qcat_app env | grep DB_
```

#### Redis连接失败
```bash
# 检查Redis状态
docker exec qcat_redis redis-cli -a $REDIS_PASSWORD ping

# 检查内存使用
docker exec qcat_redis redis-cli -a $REDIS_PASSWORD info memory
```

### 6.2 性能调优

#### 数据库优化
```sql
-- 检查慢查询
SELECT query, mean_time, calls 
FROM pg_stat_statements 
ORDER BY mean_time DESC 
LIMIT 10;

-- 检查索引使用情况
SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read, idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;
```

#### Redis优化
```bash
# 检查内存使用
docker exec qcat_redis redis-cli -a $REDIS_PASSWORD info memory

# 清理过期键
docker exec qcat_redis redis-cli -a $REDIS_PASSWORD FLUSHDB

# 监控键空间
docker exec qcat_redis redis-cli -a $REDIS_PASSWORD monitor
```

## 7. 安全维护

### 7.1 密钥管理
```bash
# 轮换JWT密钥
docker exec qcat_app /app/qcat rotate-jwt

# 轮换加密密钥
docker exec qcat_app /app/qcat rotate-keys

# 更新SSL证书
cp new_cert.crt deploy/ssl/server.crt
cp new_key.key deploy/ssl/server.key
docker restart qcat_nginx
```

### 7.2 访问控制
```bash
# 检查用户权限
docker exec qcat_app /app/qcat list-users

# 审计日志查询
docker exec qcat_app /app/qcat audit-logs --days 7

# 检查API密钥
docker exec qcat_app /app/qcat list-apikeys
```

## 8. 升级维护

### 8.1 应用升级
```bash
# 1. 备份当前版本
docker exec qcat_app /app/backup/backup.sh backup

# 2. 拉取新代码
git pull origin main

# 3. 重新构建镜像
docker-compose -f deploy/docker-compose.prod.yml build qcat

# 4. 滚动更新
docker-compose -f deploy/docker-compose.prod.yml up -d --no-deps qcat

# 5. 验证升级
curl http://localhost:8082/health
```

### 8.2 数据库迁移
```bash
# 运行数据库迁移
docker exec qcat_app /app/qcat migrate

# 检查迁移状态
docker exec qcat_app /app/qcat migrate status
```

## 9. 应急响应

### 9.1 服务重启
```bash
# 重启单个服务
docker-compose -f deploy/docker-compose.prod.yml restart qcat

# 重启所有服务
docker-compose -f deploy/docker-compose.prod.yml restart

# 强制重启
docker-compose -f deploy/docker-compose.prod.yml down
docker-compose -f deploy/docker-compose.prod.yml up -d
```

### 9.2 故障转移
```bash
# 切换到备用数据库
# 1. 停止主应用
docker-compose -f deploy/docker-compose.prod.yml stop qcat

# 2. 修改数据库配置
# 编辑.env文件，指向备用数据库

# 3. 启动应用
docker-compose -f deploy/docker-compose.prod.yml start qcat
```

### 9.3 数据恢复
```bash
# 1. 停止应用
docker-compose -f deploy/docker-compose.prod.yml stop qcat

# 2. 恢复数据库
docker exec qcat_postgres /app/backup/backup.sh restore backup_file.sql.gz

# 3. 启动应用
docker-compose -f deploy/docker-compose.prod.yml start qcat
```

## 10. 联系信息

### 10.1 技术支持
- **运维团队**: ops@qcat.com
- **开发团队**: dev@qcat.com
- **紧急联系**: +86-xxx-xxxx-xxxx

### 10.2 文档资源
- **API文档**: http://localhost:8082/swagger
- **监控面板**: http://localhost:3000
- **系统日志**: /app/logs/

---

**文档版本**: v1.0  
**最后更新**: 2024年12月  
**维护团队**: QCAT运维团队

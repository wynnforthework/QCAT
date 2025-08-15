# QCAT 故障处理手册

## 概述

本手册提供QCAT量化交易系统的故障诊断和处理指南，帮助运维人员快速定位和解决系统问题。

## 故障分类

### 按严重程度分类

- **P0 (紧急)**: 系统完全不可用，影响交易
- **P1 (高)**: 核心功能不可用，部分影响交易
- **P2 (中)**: 非核心功能异常，不影响交易
- **P3 (低)**: 功能异常，用户体验受影响

### 按组件分类

- **应用服务故障**
- **数据库故障**
- **缓存故障**
- **网络故障**
- **监控故障**

## 故障诊断流程

### 1. 问题确认

```bash
# 检查服务状态
sudo systemctl status qcat

# 检查健康状态
curl -f http://localhost:8082/health

# 检查端口监听
sudo netstat -tlnp | grep 8082

# 检查进程
ps aux | grep qcat
```

### 2. 日志分析

```bash
# 查看实时日志
sudo journalctl -u qcat -f

# 查看应用日志
tail -f /var/log/qcat/qcat.log

# 查看错误日志
grep "ERROR" /var/log/qcat/qcat.log | tail -20

# 查看特定时间段的日志
sed -n '/2024-01-01 10:00/,/2024-01-01 11:00/p' /var/log/qcat/qcat.log
```

### 3. 系统资源检查

```bash
# 检查CPU使用率
top -p $(pgrep qcat)

# 检查内存使用
free -h
cat /proc/meminfo

# 检查磁盘空间
df -h
du -sh /var/log/qcat/

# 检查网络连接
ss -tuln | grep 8082
```

### 4. 依赖服务检查

```bash
# 检查PostgreSQL
sudo systemctl status postgresql
psql -U qcat_user -h localhost -d qcat -c "SELECT 1;"

# 检查Redis
sudo systemctl status redis-server
redis-cli -a your_password ping

# 检查网络连通性
ping -c 3 localhost
telnet localhost 5432
telnet localhost 6379
```

## 常见故障及解决方案

### P0级故障

#### 1. 应用服务完全不可用

**症状**:
- 服务无法启动
- 端口无监听
- 健康检查失败

**诊断步骤**:
```bash
# 1. 检查服务状态
sudo systemctl status qcat

# 2. 检查配置文件
qcat --config /etc/qcat/config.yaml --validate

# 3. 检查端口占用
sudo lsof -i :8082

# 4. 检查文件权限
ls -la /opt/qcat/bin/qcat
ls -la /etc/qcat/config.yaml

# 5. 检查依赖服务
sudo systemctl status postgresql redis-server
```

**解决方案**:
```bash
# 1. 修复配置文件
sudo vim /etc/qcat/config.yaml

# 2. 重启依赖服务
sudo systemctl restart postgresql redis-server

# 3. 重启应用服务
sudo systemctl restart qcat

# 4. 验证服务状态
curl http://localhost:8082/health
```

#### 2. 数据库连接失败

**症状**:
- 应用启动失败
- 数据库连接超时
- 查询失败

**诊断步骤**:
```bash
# 1. 检查PostgreSQL服务
sudo systemctl status postgresql

# 2. 检查数据库连接
psql -U qcat_user -h localhost -d qcat

# 3. 检查数据库配置
sudo vim /etc/postgresql/13/main/postgresql.conf

# 4. 检查用户权限
sudo -u postgres psql -c "\du qcat_user"

# 5. 检查防火墙
sudo ufw status
```

**解决方案**:
```bash
# 1. 重启PostgreSQL
sudo systemctl restart postgresql

# 2. 修复用户权限
sudo -u postgres psql
GRANT ALL PRIVILEGES ON DATABASE qcat TO qcat_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO qcat_user;
\q

# 3. 配置防火墙
sudo ufw allow 5432/tcp

# 4. 重启应用
sudo systemctl restart qcat
```

#### 3. 内存不足导致服务崩溃

**症状**:
- 服务频繁重启
- 内存使用率过高
- 系统响应缓慢

**诊断步骤**:
```bash
# 1. 检查内存使用
free -h
cat /proc/meminfo | grep MemAvailable

# 2. 检查进程内存
ps aux | grep qcat
top -p $(pgrep qcat)

# 3. 检查系统日志
dmesg | grep -i "out of memory"
journalctl -k | grep -i "killed process"

# 4. 检查Go程序内存
curl http://localhost:8082/metrics | grep go_memstats
```

**解决方案**:
```bash
# 1. 增加系统内存（如果可能）
# 2. 优化应用内存使用
sudo vim /etc/qcat/config.yaml
# 调整内存相关配置

# 3. 重启服务
sudo systemctl restart qcat

# 4. 监控内存使用
watch -n 5 'free -h'
```

### P1级故障

#### 1. API响应缓慢

**症状**:
- API响应时间超过5秒
- 客户端超时
- 用户体验差

**诊断步骤**:
```bash
# 1. 检查CPU使用率
top -p $(pgrep qcat)

# 2. 检查数据库性能
psql -U qcat_user -d qcat -c "
SELECT query, mean_time, calls 
FROM pg_stat_statements 
ORDER BY mean_time DESC 
LIMIT 10;"

# 3. 检查慢查询日志
tail -f /var/log/postgresql/postgresql-13-main.log | grep "duration:"

# 4. 检查网络延迟
ping -c 10 localhost

# 5. 检查磁盘IO
iostat -x 1
```

**解决方案**:
```bash
# 1. 优化数据库查询
# 添加索引
psql -U qcat_user -d qcat -c "
CREATE INDEX CONCURRENTLY idx_orders_strategy_id ON orders(strategy_id);"

# 2. 调整连接池配置
sudo vim /etc/qcat/config.yaml
# 增加max_open_conns

# 3. 重启服务
sudo systemctl restart qcat

# 4. 监控性能
curl http://localhost:8082/metrics | grep request_duration
```

#### 2. 数据库连接池耗尽

**症状**:
- 数据库连接错误
- 查询超时
- 应用日志显示连接池满

**诊断步骤**:
```bash
# 1. 检查当前连接数
psql -U qcat_user -d qcat -c "
SELECT count(*) as active_connections 
FROM pg_stat_activity 
WHERE datname = 'qcat';"

# 2. 检查连接池配置
grep -A 10 "database:" /etc/qcat/config.yaml

# 3. 检查长时间运行的查询
psql -U qcat_user -d qcat -c "
SELECT pid, now() - pg_stat_activity.query_start AS duration, query 
FROM pg_stat_activity 
WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes';"
```

**解决方案**:
```bash
# 1. 终止长时间运行的查询
psql -U qcat_user -d qcat -c "
SELECT pg_terminate_backend(pid) 
FROM pg_stat_activity 
WHERE (now() - pg_stat_activity.query_start) > interval '10 minutes';"

# 2. 调整连接池配置
sudo vim /etc/qcat/config.yaml
database:
  max_open_conns: 100
  max_idle_conns: 20
  conn_max_lifetime: 10m

# 3. 重启服务
sudo systemctl restart qcat
```

#### 3. Redis缓存故障

**症状**:
- 缓存访问失败
- 应用性能下降
- Redis连接错误

**诊断步骤**:
```bash
# 1. 检查Redis服务状态
sudo systemctl status redis-server

# 2. 检查Redis连接
redis-cli -a your_password ping

# 3. 检查Redis内存使用
redis-cli -a your_password info memory

# 4. 检查Redis配置
sudo vim /etc/redis/redis.conf

# 5. 检查Redis日志
sudo tail -f /var/log/redis/redis-server.log
```

**解决方案**:
```bash
# 1. 重启Redis服务
sudo systemctl restart redis-server

# 2. 清理Redis内存
redis-cli -a your_password FLUSHDB

# 3. 调整Redis配置
sudo vim /etc/redis/redis.conf
maxmemory 2gb
maxmemory-policy allkeys-lru

# 4. 重启Redis
sudo systemctl restart redis-server

# 5. 重启应用
sudo systemctl restart qcat
```

### P2级故障

#### 1. 监控系统异常

**症状**:
- Prometheus无法访问
- Grafana面板无数据
- 告警不工作

**诊断步骤**:
```bash
# 1. 检查Prometheus状态
sudo systemctl status prometheus

# 2. 检查Grafana状态
sudo systemctl status grafana-server

# 3. 检查监控端口
sudo netstat -tlnp | grep -E "(9090|3000)"

# 4. 检查监控配置
sudo vim /opt/prometheus/prometheus.yml

# 5. 检查监控日志
sudo tail -f /var/log/prometheus/prometheus.log
```

**解决方案**:
```bash
# 1. 重启监控服务
sudo systemctl restart prometheus
sudo systemctl restart grafana-server

# 2. 检查监控目标
curl http://localhost:9090/api/v1/targets

# 3. 验证指标收集
curl http://localhost:8082/metrics

# 4. 检查告警规则
sudo vim /opt/prometheus/alerts.yml
```

#### 2. 日志系统异常

**症状**:
- 日志文件过大
- 日志轮转失败
- 日志丢失

**诊断步骤**:
```bash
# 1. 检查日志文件大小
du -sh /var/log/qcat/*

# 2. 检查磁盘空间
df -h /var/log

# 3. 检查logrotate配置
sudo vim /etc/logrotate.d/qcat

# 4. 检查logrotate状态
sudo logrotate -d /etc/logrotate.d/qcat

# 5. 检查日志权限
ls -la /var/log/qcat/
```

**解决方案**:
```bash
# 1. 清理旧日志
sudo find /var/log/qcat -name "*.log.*" -mtime +7 -delete

# 2. 手动执行日志轮转
sudo logrotate -f /etc/logrotate.d/qcat

# 3. 调整日志配置
sudo vim /etc/logrotate.d/qcat
/var/log/qcat/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 qcat qcat
    postrotate
        systemctl reload qcat
    endscript
}

# 4. 重启应用
sudo systemctl restart qcat
```

#### 3. 备份系统故障

**症状**:
- 自动备份失败
- 备份文件损坏
- 备份空间不足

**诊断步骤**:
```bash
# 1. 检查备份目录
ls -la /var/backups/qcat/

# 2. 检查备份脚本
cat /opt/qcat/scripts/backup.sh

# 3. 检查crontab
crontab -l

# 4. 检查备份文件完整性
gunzip -t /var/backups/qcat/qcat_db_20240101_020000.sql.gz

# 5. 检查磁盘空间
df -h /var/backups
```

**解决方案**:
```bash
# 1. 手动执行备份
/opt/qcat/scripts/backup.sh

# 2. 清理旧备份
find /var/backups/qcat -name "*.sql.gz" -mtime +30 -delete

# 3. 修复备份脚本权限
chmod +x /opt/qcat/scripts/backup.sh

# 4. 重新配置crontab
crontab -e
# 添加: 0 2 * * * /opt/qcat/scripts/backup.sh
```

### P3级故障

#### 1. 性能指标异常

**症状**:
- 某些指标显示异常值
- 监控面板显示错误
- 性能基线偏离

**诊断步骤**:
```bash
# 1. 检查Prometheus指标
curl http://localhost:8082/metrics | grep -E "(cpu|memory|disk)"

# 2. 检查应用性能
curl http://localhost:8082/api/v1/performance/baselines

# 3. 检查系统资源
htop
iostat -x 1

# 4. 检查应用日志
grep "performance" /var/log/qcat/qcat.log
```

**解决方案**:
```bash
# 1. 重启性能监控
curl -X POST http://localhost:8082/api/v1/performance/reset

# 2. 调整性能阈值
sudo vim /etc/qcat/config.yaml
# 调整performance相关配置

# 3. 重启服务
sudo systemctl restart qcat
```

#### 2. 告警系统异常

**症状**:
- 告警不发送
- 告警重复发送
- 告警通道失败

**诊断步骤**:
```bash
# 1. 检查告警配置
grep -A 20 "alerting:" /etc/qcat/config.yaml

# 2. 检查告警通道
curl http://localhost:8082/api/v1/alerts/channels

# 3. 测试告警发送
curl -X POST http://localhost:8082/api/v1/alerts \
  -H "Content-Type: application/json" \
  -d '{"level":"info","title":"Test","message":"Test alert"}'

# 4. 检查告警日志
grep "alert" /var/log/qcat/qcat.log
```

**解决方案**:
```bash
# 1. 重启告警服务
sudo systemctl restart qcat

# 2. 修复告警配置
sudo vim /etc/qcat/config.yaml
# 修复alerting配置

# 3. 测试告警通道
# 手动测试邮件、短信、钉钉等通道
```

## 紧急处理流程

### 1. 服务完全不可用

**立即行动**:
```bash
# 1. 停止服务
sudo systemctl stop qcat

# 2. 检查系统资源
htop
df -h
free -h

# 3. 检查关键日志
tail -n 100 /var/log/qcat/qcat.log
journalctl -u qcat --no-pager -n 50

# 4. 重启依赖服务
sudo systemctl restart postgresql redis-server

# 5. 启动服务
sudo systemctl start qcat

# 6. 验证服务
curl http://localhost:8082/health
```

### 2. 数据库故障

**立即行动**:
```bash
# 1. 停止应用
sudo systemctl stop qcat

# 2. 检查数据库状态
sudo systemctl status postgresql

# 3. 尝试重启数据库
sudo systemctl restart postgresql

# 4. 检查数据库连接
psql -U qcat_user -h localhost -d qcat -c "SELECT 1;"

# 5. 如果数据库无法恢复，从备份恢复
gunzip -c /var/backups/qcat/qcat_db_$(date -d "1 day ago" +%Y%m%d)_020000.sql.gz | psql -U qcat_user -h localhost qcat

# 6. 启动应用
sudo systemctl start qcat
```

### 3. 数据丢失

**立即行动**:
```bash
# 1. 立即停止所有服务
sudo systemctl stop qcat postgresql

# 2. 评估数据丢失范围
# 检查最近的备份文件
ls -la /var/backups/qcat/

# 3. 从最新备份恢复
gunzip -c /var/backups/qcat/qcat_db_$(date -d "1 day ago" +%Y%m%d)_020000.sql.gz | psql -U qcat_user -h localhost qcat

# 4. 启动服务
sudo systemctl start postgresql
sudo systemctl start qcat

# 5. 验证数据完整性
curl http://localhost:8082/health
psql -U qcat_user -d qcat -c "SELECT COUNT(*) FROM strategies;"
```

## 预防措施

### 1. 监控告警

- 设置关键指标告警
- 配置多渠道告警
- 定期测试告警系统

### 2. 备份策略

- 每日自动备份
- 定期备份验证
- 异地备份存储

### 3. 性能优化

- 定期性能分析
- 数据库优化
- 缓存策略优化

### 4. 安全加固

- 定期安全更新
- 访问控制审计
- 密钥轮换

## 联系信息

### 紧急联系

- **运维团队**: ops@qcat.com
- **紧急电话**: +86-xxx-xxxx-xxxx
- **值班邮箱**: oncall@qcat.com

### 技术支持

- **开发团队**: dev@qcat.com
- **文档**: https://docs.qcat.com
- **GitHub**: https://github.com/qcat/qcat

### 升级流程

1. **P0故障**: 立即联系值班人员
2. **P1故障**: 1小时内响应
3. **P2故障**: 4小时内响应
4. **P3故障**: 24小时内响应

---

**文档版本**: v1.0  
**最后更新**: 2024年1月  
**维护团队**: QCAT运维团队

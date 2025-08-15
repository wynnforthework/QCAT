# QCAT 运维手册

## 概述

本手册提供QCAT量化交易系统的完整运维指南，包括部署、配置、监控、维护和故障处理等内容。

## 系统架构

### 组件说明

- **API服务器**: 提供RESTful API接口
- **数据库**: PostgreSQL存储业务数据
- **缓存**: Redis提供缓存服务
- **监控**: Prometheus + Grafana监控系统
- **日志**: 结构化日志系统
- **备份**: 自动备份和恢复系统

### 部署架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Load Balancer │    │   API Server 1  │    │   API Server 2  │
│   (Nginx)       │────│   (Port 8082)   │    │   (Port 8082)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                       │
                                └───────────────────────┘
                                        │
                                ┌─────────────────┐
                                │   PostgreSQL    │
                                │   (Port 5432)   │
                                └─────────────────┘
                                        │
                                ┌─────────────────┐
                                │     Redis       │
                                │   (Port 6379)   │
                                └─────────────────┘
```

## 部署指南

### 环境要求

#### 硬件要求

- **CPU**: 4核心以上
- **内存**: 8GB以上
- **存储**: 100GB以上SSD
- **网络**: 千兆网络连接

#### 软件要求

- **操作系统**: Ubuntu 20.04+ / CentOS 8+
- **Go版本**: 1.21+
- **PostgreSQL**: 13+
- **Redis**: 6.0+
- **Docker**: 20.10+ (可选)

### 安装步骤

#### 1. 系统准备

```bash
# 更新系统
sudo apt update && sudo apt upgrade -y

# 安装基础工具
sudo apt install -y curl wget git vim htop

# 安装Go
wget https://golang.org/dl/go1.21.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

#### 2. 安装PostgreSQL

```bash
# 安装PostgreSQL
sudo apt install -y postgresql postgresql-contrib

# 启动服务
sudo systemctl start postgresql
sudo systemctl enable postgresql

# 创建数据库和用户
sudo -u postgres psql
CREATE DATABASE qcat;
CREATE USER qcat_user WITH PASSWORD 'secure_password';
GRANT ALL PRIVILEGES ON DATABASE qcat TO qcat_user;
\q
```

#### 3. 安装Redis

```bash
# 安装Redis
sudo apt install -y redis-server

# 配置Redis
sudo vim /etc/redis/redis.conf
# 修改以下配置：
# bind 127.0.0.1
# requirepass your_redis_password
# maxmemory 2gb
# maxmemory-policy allkeys-lru

# 启动服务
sudo systemctl start redis-server
sudo systemctl enable redis-server
```

#### 4. 部署QCAT

```bash
# 克隆代码
git clone https://github.com/qcat/qcat.git
cd qcat

# 编译
go build -o bin/qcat cmd/qcat/main.go

# 创建配置目录
sudo mkdir -p /etc/qcat
sudo cp configs/config.yaml /etc/qcat/

# 创建日志目录
sudo mkdir -p /var/log/qcat
sudo chown -R $USER:$USER /var/log/qcat

# 创建数据目录
sudo mkdir -p /var/lib/qcat
sudo chown -R $USER:$USER /var/lib/qcat
```

#### 5. 配置系统服务

```bash
# 创建systemd服务文件
sudo vim /etc/systemd/system/qcat.service
```

服务文件内容：
```ini
[Unit]
Description=QCAT Quant Trading System
After=network.target postgresql.service redis-server.service

[Service]
Type=simple
User=qcat
Group=qcat
WorkingDirectory=/opt/qcat
ExecStart=/opt/qcat/bin/qcat
Restart=always
RestartSec=5
Environment=GIN_MODE=release
Environment=QCAT_CONFIG=/etc/qcat/config.yaml

[Install]
WantedBy=multi-user.target
```

```bash
# 重新加载systemd
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start qcat
sudo systemctl enable qcat

# 检查状态
sudo systemctl status qcat
```

### 配置说明

#### 主配置文件

配置文件位置：`/etc/qcat/config.yaml`

```yaml
# 应用配置
app:
  name: "QCAT"
  version: "1.0.0"
  environment: "production"

# 服务器配置
server:
  port: 8082
  read_timeout: 30s
  write_timeout: 30s

# 数据库配置
database:
  host: "localhost"
  port: 5432
  user: "qcat_user"
  password: "ENC:encrypted_password"
  name: "qcat"
  ssl_mode: "disable"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m
  conn_max_idle_time: 1m

# Redis配置
redis:
  addr: "localhost:6379"
  password: "ENC:encrypted_password"
  db: 0
  pool_size: 10
  min_idle_conns: 5

# JWT配置
jwt:
  secret_key: "ENC:encrypted_secret_key"
  expiration_hours: 24

# 日志配置
logging:
  level: "info"
  format: "json"
  output: "file"
  max_size: 100
  max_backups: 10
  max_age: 30
  compress: true
  log_dir: "/var/log/qcat"

# 监控配置
monitoring:
  prometheus_enabled: true
  prometheus_port: 9090
  metrics_path: "/metrics"

# 健康检查配置
health:
  check_interval: 30s
  timeout: 10s
  retry_count: 3
  retry_interval: 5s
  degraded_threshold: 0.8
  unhealthy_threshold: 0.5
  alert_threshold: 3
  alert_cooldown: 5m

# 优雅关闭配置
shutdown:
  shutdown_timeout: 30s
  component_timeout: 10s
  signal_timeout: 5s
  enable_signal_handling: true
  force_shutdown_after: 60s
  log_shutdown_progress: true
  shutdown_order:
    - websocket_connections
    - strategy_runners
    - market_data_streams
    - order_managers
    - position_managers
    - risk_engine
    - optimizer
    - health_checker
    - network_manager
    - memory_manager
    - redis_cache
    - database
    - http_server
```

#### 环境变量配置

创建环境变量文件：`/etc/qcat/.env`

```bash
# 应用配置
QCAT_APP_NAME=QCAT
QCAT_APP_VERSION=1.0.0
QCAT_APP_ENVIRONMENT=production

# 服务器配置
QCAT_SERVER_PORT=8082
QCAT_SERVER_READ_TIMEOUT=30s
QCAT_SERVER_WRITE_TIMEOUT=30s

# 数据库配置
QCAT_DATABASE_HOST=localhost
QCAT_DATABASE_PORT=5432
QCAT_DATABASE_USER=qcat_user
QCAT_DATABASE_PASSWORD=ENC:your_encrypted_password
QCAT_DATABASE_NAME=qcat
QCAT_DATABASE_SSL_MODE=disable

# Redis配置
QCAT_REDIS_ADDR=localhost:6379
QCAT_REDIS_PASSWORD=ENC:your_encrypted_password
QCAT_REDIS_DB=0

# JWT配置
QCAT_JWT_SECRET_KEY=ENC:your_encrypted_jwt_secret
QCAT_JWT_EXPIRATION_HOURS=24

# 加密密钥
QCAT_ENCRYPTION_KEY=your_encryption_key_here
```

## 监控指南

### Prometheus监控

#### 安装Prometheus

```bash
# 下载Prometheus
wget https://github.com/prometheus/prometheus/releases/download/v2.45.0/prometheus-2.45.0.linux-amd64.tar.gz
tar xvf prometheus-2.45.0.linux-amd64.tar.gz
sudo mv prometheus-2.45.0.linux-amd64 /opt/prometheus

# 创建配置文件
sudo vim /opt/prometheus/prometheus.yml
```

Prometheus配置：
```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  # - "first_rules.yml"
  # - "second_rules.yml"

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'qcat'
    static_configs:
      - targets: ['localhost:9090']
    metrics_path: '/metrics'
    scrape_interval: 5s
```

#### 安装Grafana

```bash
# 安装Grafana
sudo apt install -y grafana

# 启动服务
sudo systemctl start grafana-server
sudo systemctl enable grafana-server

# 访问Grafana
# http://localhost:3000
# 默认用户名/密码: admin/admin
```

### 关键监控指标

#### 系统指标

- **CPU使用率**: `cpu_usage_percent`
- **内存使用率**: `memory_usage_percent`
- **磁盘使用率**: `disk_usage_percent`
- **网络IO**: `network_io_bytes`

#### 应用指标

- **请求率**: `http_requests_total`
- **响应时间**: `http_request_duration_seconds`
- **错误率**: `http_errors_total`
- **活跃连接数**: `active_connections`

#### 数据库指标

- **连接数**: `db_connections`
- **查询时间**: `db_query_duration_seconds`
- **错误数**: `db_errors_total`

#### 业务指标

- **策略数量**: `strategies_total`
- **订单数量**: `orders_total`
- **交易量**: `trades_volume`
- **盈亏**: `pnl_total`

### 告警配置

#### 告警规则

创建告警规则文件：`/opt/prometheus/alerts.yml`

```yaml
groups:
  - name: qcat_alerts
    rules:
      - alert: HighCPUUsage
        expr: cpu_usage_percent > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High CPU usage detected"
          description: "CPU usage is {{ $value }}%"

      - alert: HighMemoryUsage
        expr: memory_usage_percent > 85
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage detected"
          description: "Memory usage is {{ $value }}%"

      - alert: DatabaseConnectionHigh
        expr: db_connections > 20
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High database connections"
          description: "Database connections: {{ $value }}"

      - alert: HighErrorRate
        expr: rate(http_errors_total[5m]) > 0.1
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate: {{ $value }} errors/second"
```

## 备份和恢复

### 数据库备份

#### 自动备份脚本

创建备份脚本：`/opt/qcat/scripts/backup.sh`

```bash
#!/bin/bash

# 配置
DB_NAME="qcat"
DB_USER="qcat_user"
BACKUP_DIR="/var/backups/qcat"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="qcat_db_${DATE}.sql"

# 创建备份目录
mkdir -p $BACKUP_DIR

# 执行备份
pg_dump -U $DB_USER -h localhost $DB_NAME > $BACKUP_DIR/$BACKUP_FILE

# 压缩备份文件
gzip $BACKUP_DIR/$BACKUP_FILE

# 删除7天前的备份
find $BACKUP_DIR -name "*.sql.gz" -mtime +7 -delete

echo "Backup completed: $BACKUP_FILE.gz"
```

#### 设置定时备份

```bash
# 编辑crontab
crontab -e

# 添加定时任务（每天凌晨2点备份）
0 2 * * * /opt/qcat/scripts/backup.sh
```

### 配置文件备份

```bash
# 备份配置文件
sudo tar -czf /var/backups/qcat/config_$(date +%Y%m%d_%H%M%S).tar.gz /etc/qcat/

# 设置定时备份
# 0 3 * * * sudo tar -czf /var/backups/qcat/config_$(date +%Y%m%d_%H%M%S).tar.gz /etc/qcat/
```

### 恢复流程

#### 数据库恢复

```bash
# 停止服务
sudo systemctl stop qcat

# 恢复数据库
gunzip -c /var/backups/qcat/qcat_db_20240101_020000.sql.gz | psql -U qcat_user -h localhost qcat

# 启动服务
sudo systemctl start qcat
```

#### 配置文件恢复

```bash
# 停止服务
sudo systemctl stop qcat

# 恢复配置文件
sudo tar -xzf /var/backups/qcat/config_20240101_030000.tar.gz -C /

# 启动服务
sudo systemctl start qcat
```

## 日志管理

### 日志配置

日志文件位置：`/var/log/qcat/`

- **应用日志**: `qcat.log`
- **访问日志**: `access.log`
- **错误日志**: `error.log`

### 日志轮转

配置logrotate：`/etc/logrotate.d/qcat`

```
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
```

### 日志分析

#### 使用grep分析错误

```bash
# 查看错误日志
grep "ERROR" /var/log/qcat/qcat.log

# 查看特定时间段的日志
grep "2024-01-01" /var/log/qcat/qcat.log

# 统计错误数量
grep -c "ERROR" /var/log/qcat/qcat.log
```

#### 使用awk分析性能

```bash
# 分析响应时间
awk '/response_time/ {sum+=$NF; count++} END {print "Average response time:", sum/count}' /var/log/qcat/qcat.log
```

## 性能调优

### 系统调优

#### 内核参数调优

编辑：`/etc/sysctl.conf`

```bash
# 网络调优
net.core.somaxconn = 65535
net.core.netdev_max_backlog = 5000
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.tcp_fin_timeout = 30
net.ipv4.tcp_keepalive_time = 1200
net.ipv4.tcp_max_tw_buckets = 5000

# 文件描述符
fs.file-max = 1000000
fs.nr_open = 1000000

# 内存调优
vm.swappiness = 10
vm.dirty_ratio = 15
vm.dirty_background_ratio = 5
```

应用配置：
```bash
sudo sysctl -p
```

#### 用户限制调优

编辑：`/etc/security/limits.conf`

```
qcat soft nofile 65535
qcat hard nofile 65535
qcat soft nproc 65535
qcat hard nproc 65535
```

### 应用调优

#### 数据库连接池调优

```yaml
database:
  max_open_conns: 50      # 根据CPU核心数调整
  max_idle_conns: 10      # max_open_conns的20%
  conn_max_lifetime: 10m  # 连接最大生命周期
  conn_max_idle_time: 5m  # 空闲连接最大时间
```

#### Redis连接池调优

```yaml
redis:
  pool_size: 20           # 连接池大小
  min_idle_conns: 5       # 最小空闲连接数
  max_retries: 3          # 最大重试次数
  dial_timeout: 5s        # 连接超时
  read_timeout: 3s        # 读取超时
  write_timeout: 3s       # 写入超时
```

## 安全配置

### 防火墙配置

```bash
# 安装ufw
sudo apt install -y ufw

# 配置防火墙规则
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow 8082/tcp  # QCAT API
sudo ufw allow 9090/tcp  # Prometheus
sudo ufw allow 3000/tcp  # Grafana

# 启用防火墙
sudo ufw enable
```

### SSL/TLS配置

#### 使用Let's Encrypt

```bash
# 安装certbot
sudo apt install -y certbot

# 获取证书
sudo certbot certonly --standalone -d your-domain.com

# 配置Nginx
sudo vim /etc/nginx/sites-available/qcat
```

Nginx配置：
```nginx
server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl;
    server_name your-domain.com;

    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;

    location / {
        proxy_pass http://localhost:8082;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 访问控制

#### API访问控制

```yaml
security:
  rate_limit_enabled: true
  rate_limit_requests: 100
  rate_limit_window: 1m
  cors_enabled: true
  cors_origins:
    - "https://your-domain.com"
    - "https://admin.your-domain.com"
```

## 故障处理

### 常见问题

#### 1. 服务无法启动

**症状**: `systemctl status qcat` 显示失败

**排查步骤**:
```bash
# 检查配置文件
qcat --config /etc/qcat/config.yaml --validate

# 检查日志
journalctl -u qcat -f

# 检查端口占用
sudo netstat -tlnp | grep 8082

# 检查数据库连接
psql -U qcat_user -h localhost -d qcat -c "SELECT 1;"
```

**解决方案**:
- 修复配置文件语法错误
- 检查数据库连接配置
- 确保端口未被占用
- 检查文件权限

#### 2. 数据库连接失败

**症状**: 日志显示数据库连接错误

**排查步骤**:
```bash
# 检查PostgreSQL服务状态
sudo systemctl status postgresql

# 检查数据库连接
psql -U qcat_user -h localhost -d qcat

# 检查防火墙
sudo ufw status

# 检查PostgreSQL配置
sudo vim /etc/postgresql/13/main/postgresql.conf
```

**解决方案**:
- 启动PostgreSQL服务
- 检查用户权限
- 配置防火墙规则
- 修改PostgreSQL配置

#### 3. 内存使用过高

**症状**: 系统内存使用率超过90%

**排查步骤**:
```bash
# 检查内存使用
free -h
top -p $(pgrep qcat)

# 检查Go程序内存
curl http://localhost:8082/metrics | grep go_memstats

# 检查系统日志
dmesg | grep -i "out of memory"
```

**解决方案**:
- 增加系统内存
- 优化应用内存使用
- 调整垃圾回收参数
- 重启服务

#### 4. 响应时间过长

**症状**: API响应时间超过5秒

**排查步骤**:
```bash
# 检查CPU使用率
top -p $(pgrep qcat)

# 检查数据库性能
psql -U qcat_user -d qcat -c "SELECT * FROM pg_stat_activity;"

# 检查网络延迟
ping localhost

# 检查磁盘IO
iostat -x 1
```

**解决方案**:
- 优化数据库查询
- 增加数据库连接池
- 使用缓存
- 优化代码逻辑

### 紧急处理流程

#### 1. 服务完全不可用

```bash
# 立即停止服务
sudo systemctl stop qcat

# 检查系统资源
htop
df -h
free -h

# 检查日志
tail -n 100 /var/log/qcat/qcat.log

# 重启服务
sudo systemctl start qcat

# 监控服务状态
watch -n 5 'systemctl status qcat'
```

#### 2. 数据库故障

```bash
# 停止应用服务
sudo systemctl stop qcat

# 检查数据库状态
sudo systemctl status postgresql

# 尝试重启数据库
sudo systemctl restart postgresql

# 检查数据库完整性
sudo -u postgres pg_checkdb qcat

# 恢复服务
sudo systemctl start qcat
```

#### 3. 数据丢失

```bash
# 立即停止所有服务
sudo systemctl stop qcat postgresql

# 从备份恢复
gunzip -c /var/backups/qcat/qcat_db_$(date -d "1 day ago" +%Y%m%d)_020000.sql.gz | psql -U qcat_user -h localhost qcat

# 启动服务
sudo systemctl start postgresql
sudo systemctl start qcat

# 验证数据完整性
curl http://localhost:8082/health
```

## 维护计划

### 日常维护

#### 每日任务

- [ ] 检查服务状态
- [ ] 检查系统资源使用
- [ ] 检查错误日志
- [ ] 验证备份状态

#### 每周任务

- [ ] 分析性能指标
- [ ] 检查磁盘空间
- [ ] 更新安全补丁
- [ ] 优化数据库

#### 每月任务

- [ ] 完整系统检查
- [ ] 性能基准测试
- [ ] 安全审计
- [ ] 文档更新

### 更新流程

#### 1. 准备更新

```bash
# 创建备份
/opt/qcat/scripts/backup.sh

# 停止服务
sudo systemctl stop qcat

# 备份当前版本
sudo cp /opt/qcat/bin/qcat /opt/qcat/bin/qcat.backup
```

#### 2. 执行更新

```bash
# 下载新版本
cd /tmp
wget https://github.com/qcat/qcat/releases/download/v1.1.0/qcat-v1.1.0-linux-amd64.tar.gz
tar xzf qcat-v1.1.0-linux-amd64.tar.gz

# 替换二进制文件
sudo cp qcat /opt/qcat/bin/qcat

# 更新配置文件（如果需要）
sudo cp config.yaml /etc/qcat/config.yaml.new
sudo diff /etc/qcat/config.yaml /etc/qcat/config.yaml.new
```

#### 3. 验证更新

```bash
# 启动服务
sudo systemctl start qcat

# 检查服务状态
sudo systemctl status qcat

# 验证功能
curl http://localhost:8082/health

# 检查版本
curl http://localhost:8082/api/v1/version
```

#### 4. 回滚计划

```bash
# 如果更新失败，立即回滚
sudo systemctl stop qcat
sudo cp /opt/qcat/bin/qcat.backup /opt/qcat/bin/qcat
sudo systemctl start qcat
```

## 联系信息

### 技术支持

- **邮箱**: ops@qcat.com
- **电话**: +86-xxx-xxxx-xxxx
- **工作时间**: 24/7

### 紧急联系

- **紧急电话**: +86-xxx-xxxx-xxxx
- **值班邮箱**: oncall@qcat.com

### 文档资源

- **在线文档**: https://docs.qcat.com
- **GitHub**: https://github.com/qcat/qcat
- **问题反馈**: https://github.com/qcat/qcat/issues

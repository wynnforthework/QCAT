# QCAT 部署指南

## 概述

本文档详细介绍了QCAT量化合约自动化交易系统的部署方法，支持多种部署方式：

- **Docker Compose**: 适用于单机部署
- **Kubernetes**: 适用于集群部署
- **传统服务器**: 适用于现有基础设施

## 目录

1. [环境要求](#环境要求)
2. [快速开始](#快速开始)
3. [Docker Compose部署](#docker-compose部署)
4. [Kubernetes部署](#kubernetes部署)
5. [传统服务器部署](#传统服务器部署)
6. [CI/CD自动化部署](#cicd自动化部署)
7. [监控和告警](#监控和告警)
8. [备份和恢复](#备份和恢复)
9. [故障排除](#故障排除)

## 环境要求

### 系统要求

- **操作系统**: Linux (Ubuntu 20.04+, CentOS 8+, RHEL 8+)
- **CPU**: 4核心以上
- **内存**: 8GB以上
- **存储**: 100GB以上可用空间
- **网络**: 稳定的互联网连接

### 软件依赖

#### Docker Compose部署
- Docker 20.10+
- Docker Compose 2.0+
- Git

#### Kubernetes部署
- Kubernetes 1.24+
- kubectl 1.24+
- Helm 3.0+ (可选)

#### 传统服务器部署
- Go 1.24+
- Node.js 20+
- PostgreSQL 14+
- Redis 7+
- Nginx 1.20+

## 快速开始

### 1. 克隆代码库

```bash
git clone https://github.com/your-org/qcat.git
cd qcat
```

### 2. 配置环境变量

```bash
# 复制环境变量模板
cp deploy/env.example .env.production

# 编辑环境变量
vim .env.production
```

### 3. 使用Docker Compose快速部署

```bash
# 启动所有服务
docker-compose -f deploy/docker-compose.prod.yml up -d

# 查看服务状态
docker-compose -f deploy/docker-compose.prod.yml ps

# 查看日志
docker-compose -f deploy/docker-compose.prod.yml logs -f
```

## Docker Compose部署

### 1. 准备环境

```bash
# 安装Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# 安装Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

### 2. 配置环境变量

```bash
# 创建环境变量文件
cat > .env.production << EOF
POSTGRES_PASSWORD=your_secure_password
REDIS_PASSWORD=your_redis_password
JWT_SECRET=your_jwt_secret
API_KEY=your_binance_api_key
API_SECRET=your_binance_api_secret
GRAFANA_PASSWORD=admin_password
EOF
```

### 3. 启动服务

```bash
# 启动所有服务
docker-compose -f deploy/docker-compose.prod.yml up -d

# 等待服务启动
sleep 60

# 检查服务状态
docker-compose -f deploy/docker-compose.prod.yml ps
```

### 4. 验证部署

```bash
# 检查后端API
curl http://localhost:8080/health

# 检查前端
curl http://localhost:80

# 检查监控
curl http://localhost:8082/-/healthy  # Prometheus
curl http://localhost:3000/api/health # Grafana
```

### 5. 访问服务

- **前端界面**: http://localhost
- **API文档**: http://localhost:8080/docs
- **Grafana监控**: http://localhost:3000 (admin/admin_password)
- **Prometheus**: http://localhost:8082

## Kubernetes部署

### 1. 准备Kubernetes集群

```bash
# 检查集群状态
kubectl cluster-info

# 创建命名空间
kubectl create namespace qcat-production
```

### 2. 创建ConfigMap和Secret

```bash
# 创建ConfigMap
kubectl create configmap qcat-config \
  --from-file=configs/config.yaml \
  -n qcat-production

# 创建Secret
kubectl create secret generic qcat-secrets \
  --from-literal=postgres-password=your_password \
  --from-literal=redis-password=your_redis_password \
  --from-literal=jwt-secret=your_jwt_secret \
  --from-literal=api-key=your_api_key \
  --from-literal=api-secret=your_api_secret \
  -n qcat-production
```

### 3. 部署应用

```bash
# 部署数据库
kubectl apply -f k8s/production/postgres.yaml

# 部署Redis
kubectl apply -f k8s/production/redis.yaml

# 部署后端
kubectl apply -f k8s/production/backend.yaml

# 部署前端
kubectl apply -f k8s/production/frontend.yaml

# 部署监控
kubectl apply -f k8s/production/monitoring.yaml
```

### 4. 配置Ingress

```bash
# 部署Ingress控制器
kubectl apply -f k8s/production/ingress.yaml

# 配置SSL证书
kubectl apply -f k8s/production/cert-manager.yaml
```

### 5. 验证部署

```bash
# 检查Pod状态
kubectl get pods -n qcat-production

# 检查服务状态
kubectl get svc -n qcat-production

# 检查Ingress
kubectl get ingress -n qcat-production
```

## 传统服务器部署

### 1. 准备服务器环境

```bash
# 更新系统
sudo apt update && sudo apt upgrade -y

# 安装基础软件
sudo apt install -y curl wget git build-essential

# 安装Go
wget https://go.dev/dl/go1.24.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# 安装Node.js
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs

# 安装PostgreSQL
sudo apt install -y postgresql postgresql-contrib

# 安装Redis
sudo apt install -y redis-server

# 安装Nginx
sudo apt install -y nginx
```

### 2. 配置数据库

```bash
# 创建数据库用户
sudo -u postgres createuser --interactive qcat

# 创建数据库
sudo -u postgres createdb qcat

# 设置密码
sudo -u postgres psql -c "ALTER USER qcat PASSWORD 'your_password';"
```

### 3. 配置Redis

```bash
# 编辑Redis配置
sudo vim /etc/redis/redis.conf

# 添加密码
requirepass your_redis_password

# 重启Redis
sudo systemctl restart redis
sudo systemctl enable redis
```

### 4. 构建和部署应用

```bash
# 克隆代码
git clone https://github.com/your-org/qcat.git
cd qcat

# 构建后端
go build -o qcat ./cmd/qcat

# 构建前端
cd frontend
npm ci
npm run build
cd ..

# 创建部署目录
sudo mkdir -p /opt/qcat
sudo cp qcat /opt/qcat/
sudo cp -r frontend/.next /opt/qcat/frontend
sudo cp -r configs /opt/qcat/
```

### 5. 配置系统服务

```bash
# 创建后端服务
sudo tee /etc/systemd/system/qcat-backend.service > /dev/null << EOF
[Unit]
Description=QCAT Backend Service
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=qcat
Group=qcat
WorkingDirectory=/opt/qcat
ExecStart=/opt/qcat/qcat
Restart=always
RestartSec=5
Environment=NODE_ENV=production
Environment=POSTGRES_HOST=localhost
Environment=POSTGRES_PORT=5432
Environment=POSTGRES_USER=qcat
Environment=POSTGRES_PASSWORD=your_password
Environment=POSTGRES_DB=qcat
Environment=REDIS_HOST=localhost
Environment=REDIS_PORT=6379
Environment=REDIS_PASSWORD=your_redis_password

[Install]
WantedBy=multi-user.target
EOF

# 创建用户
sudo useradd -r -s /bin/false qcat
sudo chown -R qcat:qcat /opt/qcat

# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable qcat-backend
sudo systemctl start qcat-backend
```

### 6. 配置Nginx

```bash
# 创建Nginx配置
sudo tee /etc/nginx/sites-available/qcat > /dev/null << EOF
server {
    listen 80;
    server_name qcat.local;
    root /opt/qcat/frontend;
    index index.html;

    # API代理
    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_cache_bypass \$http_upgrade;
    }

    # WebSocket代理
    location /ws/ {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    # 静态文件
    location / {
        try_files \$uri \$uri/ /index.html;
    }
}
EOF

# 启用站点
sudo ln -sf /etc/nginx/sites-available/qcat /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

## CI/CD自动化部署

### 1. GitHub Actions配置

项目已配置完整的CI/CD流水线：

- **后端CI/CD**: `.github/workflows/backend.yml`
- **前端CI/CD**: `.github/workflows/frontend.yml`

### 2. 配置Secrets

在GitHub仓库设置中添加以下Secrets：

```bash
# 服务器配置
STAGING_HOST=staging.qcat.local
STAGING_USER=qcat
STAGING_SSH_KEY=-----BEGIN OPENSSH PRIVATE KEY-----

PRODUCTION_HOST=production.qcat.local
PRODUCTION_USER=qcat
PRODUCTION_SSH_KEY=-----BEGIN OPENSSH PRIVATE KEY-----

# 通知配置
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK
```

### 3. 自动部署流程

1. **代码推送**: 推送到`main`分支触发生产部署
2. **代码推送**: 推送到`dev`分支触发测试部署
3. **构建**: 自动构建Docker镜像
4. **测试**: 运行单元测试和集成测试
5. **部署**: 自动部署到对应环境
6. **健康检查**: 验证部署成功
7. **通知**: 发送部署结果通知

### 4. 手动部署

```bash
# 使用部署脚本
./deploy/deploy.sh -e production -m docker
./deploy/deploy.sh -e staging -m k8s
./deploy/deploy.sh -e production -m server --force
```

## 监控和告警

### 1. Prometheus监控

```bash
# 访问Prometheus
http://localhost:8082

# 查看指标
http://localhost:8082/metrics
```

### 2. Grafana仪表板

```bash
# 访问Grafana
http://localhost:3000

# 默认凭据
用户名: admin
密码: admin_password
```

### 3. 告警配置

```bash
# 配置告警规则
vim deploy/alertmanager.yml

# 重启AlertManager
docker-compose -f deploy/docker-compose.prod.yml restart alertmanager
```

### 4. 日志监控

```bash
# 查看应用日志
docker-compose -f deploy/docker-compose.prod.yml logs -f backend

# 查看系统日志
journalctl -u qcat-backend -f
```

## 备份和恢复

### 1. 数据库备份

```bash
# 创建备份脚本
cat > backup.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/opt/qcat/backups"
DATE=$(date +%Y%m%d_%H%M%S)

# 创建备份目录
mkdir -p $BACKUP_DIR

# 备份PostgreSQL
pg_dump -h localhost -U qcat qcat > $BACKUP_DIR/qcat_$DATE.sql

# 备份Redis
redis-cli --rdb $BACKUP_DIR/redis_$DATE.rdb

# 压缩备份
tar -czf $BACKUP_DIR/backup_$DATE.tar.gz $BACKUP_DIR/qcat_$DATE.sql $BACKUP_DIR/redis_$DATE.rdb

# 清理临时文件
rm $BACKUP_DIR/qcat_$DATE.sql $BACKUP_DIR/redis_$DATE.rdb

# 删除旧备份（保留30天）
find $BACKUP_DIR -name "backup_*.tar.gz" -mtime +30 -delete

echo "备份完成: $BACKUP_DIR/backup_$DATE.tar.gz"
EOF

chmod +x backup.sh
```

### 2. 自动备份

```bash
# 添加到crontab
crontab -e

# 每天凌晨2点执行备份
0 2 * * * /opt/qcat/backup.sh
```

### 3. 数据恢复

```bash
# 恢复PostgreSQL
psql -h localhost -U qcat qcat < backup_20240101_020000.sql

# 恢复Redis
redis-cli flushall
redis-cli --pipe < redis_20240101_020000.rdb
```

## 故障排除

### 1. 常见问题

#### 服务无法启动

```bash
# 检查服务状态
docker-compose -f deploy/docker-compose.prod.yml ps

# 查看详细日志
docker-compose -f deploy/docker-compose.prod.yml logs backend

# 检查端口占用
netstat -tlnp | grep :8080
```

#### 数据库连接失败

```bash
# 检查PostgreSQL状态
sudo systemctl status postgresql

# 检查连接
psql -h localhost -U qcat -d qcat

# 检查防火墙
sudo ufw status
```

#### Redis连接失败

```bash
# 检查Redis状态
sudo systemctl status redis

# 测试连接
redis-cli ping

# 检查配置
sudo cat /etc/redis/redis.conf | grep requirepass
```

### 2. 性能优化

#### 数据库优化

```sql
-- 创建索引
CREATE INDEX idx_trades_symbol_time ON trades(symbol, timestamp);
CREATE INDEX idx_positions_strategy ON positions(strategy_id);

-- 分析表
ANALYZE trades;
ANALYZE positions;
```

#### Redis优化

```bash
# 编辑Redis配置
sudo vim /etc/redis/redis.conf

# 优化内存使用
maxmemory 1gb
maxmemory-policy allkeys-lru

# 启用持久化
save 900 1
save 300 10
save 60 10000
```

#### 应用优化

```bash
# 调整Go运行时参数
export GOMAXPROCS=4
export GOGC=100

# 调整系统参数
echo 'net.core.somaxconn = 65535' | sudo tee -a /etc/sysctl.conf
echo 'net.ipv4.tcp_max_syn_backlog = 65535' | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

### 3. 安全加固

#### 防火墙配置

```bash
# 配置UFW
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 8080/tcp
sudo ufw enable
```

#### SSL证书配置

```bash
# 使用Let's Encrypt
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d qcat.local

# 自动续期
sudo crontab -e
0 12 * * * /usr/bin/certbot renew --quiet
```

#### 安全头配置

```nginx
# 在Nginx配置中添加安全头
add_header X-Frame-Options "SAMEORIGIN" always;
add_header X-XSS-Protection "1; mode=block" always;
add_header X-Content-Type-Options "nosniff" always;
add_header Referrer-Policy "no-referrer-when-downgrade" always;
add_header Content-Security-Policy "default-src 'self' http: https: data: blob: 'unsafe-inline'" always;
```

## 维护和更新

### 1. 版本更新

```bash
# 拉取最新代码
git pull origin main

# 重新构建镜像
docker-compose -f deploy/docker-compose.prod.yml build

# 更新服务
docker-compose -f deploy/docker-compose.prod.yml up -d
```

### 2. 系统维护

```bash
# 清理Docker资源
docker system prune -f

# 清理日志
sudo journalctl --vacuum-time=7d

# 更新系统包
sudo apt update && sudo apt upgrade -y
```

### 3. 性能监控

```bash
# 监控系统资源
htop
iotop
nethogs

# 监控应用性能
docker stats
kubectl top pods -n qcat-production
```

## 联系支持

如果在部署过程中遇到问题，请：

1. 查看本文档的故障排除部分
2. 检查GitHub Issues
3. 提交详细的错误报告
4. 联系技术支持团队

---

**注意**: 本部署指南适用于QCAT v1.0.0版本，请根据实际版本调整配置参数。

# Fund Protector Deployment Guide

## 🚀 部署准备

### 1. 环境要求
- Go 1.19+
- PostgreSQL 13+
- Redis (可选，用于缓存)
- 稳定的网络连接到交易所API

### 2. 依赖服务
- 交易所API访问权限（Binance, OKX等）
- SMTP服务器（用于邮件通知）
- SMS服务提供商（Twilio或AWS SNS）
- 区块链节点访问（用于钱包操作）

## 📋 部署步骤

### Step 1: 环境变量配置

创建 `.env` 文件：
```bash
# 交易所API配置
BINANCE_API_KEY=your_binance_api_key
BINANCE_SECRET_KEY=your_binance_secret_key
OKX_API_KEY=your_okx_api_key
OKX_SECRET_KEY=your_okx_secret_key

# 通知服务配置
SMTP_USERNAME=your_smtp_username
SMTP_PASSWORD=your_smtp_password
SMTP_FROM_ADDRESS=alerts@yourcompany.com
TWILIO_API_KEY=your_twilio_api_key
TWILIO_API_SECRET=your_twilio_api_secret
TWILIO_PHONE_NUMBER=+1234567890
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...
WEBHOOK_URL=https://your-webhook-endpoint.com/alerts
WEBHOOK_TOKEN=your_webhook_token

# 钱包配置
HOT_WALLET_ADDRESS=0x1234567890123456789012345678901234567890
COLD_WALLET_ADDRESS=0x0987654321098765432109876543210987654321
ETH_NETWORK_URL=https://mainnet.infura.io/v3/your_project_id
WALLET_PRIVATE_KEY=your_encrypted_private_key

# 联系人信息
RISK_MANAGER_EMAIL=risk@yourcompany.com
RISK_MANAGER_PHONE=+1234567890
TRADING_DESK_EMAIL=trading@yourcompany.com
TRADING_DESK_PHONE=+1234567891
FUND_MANAGER_EMAIL=fund@yourcompany.com
FUND_MANAGER_PHONE=+1234567892

# 数据库配置
DATABASE_URL=postgres://user:password@localhost:5432/qcat_db
REDIS_URL=redis://localhost:6379
```

### Step 2: 数据库初始化

运行数据库迁移：
```bash
# 创建数据库表
go run cmd/migrate/main.go up

# 或者使用SQL脚本
psql -d qcat_db -f migrations/001_create_fund_protector_tables.sql
```

### Step 3: 配置文件设置

复制并修改配置文件：
```bash
cp internal/security/protector/config_example.yaml config/fund_protector.yaml
# 根据实际环境修改配置参数
```

### Step 4: 编译和部署

```bash
# 编译应用
go build -o bin/qcat cmd/main.go

# 运行应用
./bin/qcat --config config/fund_protector.yaml
```

## 🔧 配置调优

### 风险参数调优
```yaml
fund_protector:
  # 根据资金规模调整
  profit_threshold: 0.08        # 大资金可以降低阈值
  max_daily_loss: 0.03          # 保守策略可以降低限制
  
  # 根据市场波动调整
  check_interval: 3m            # 高波动期间缩短检查间隔
  
  risk_calculation:
    historical_days: 60         # 高波动期间减少历史数据天数
    var_confidence: 0.99        # 保守策略提高置信度
```

### 性能优化
```yaml
exchange:
  cache_ttl: 15s               # 高频交易降低缓存时间
  rate_limit_requests: 120     # 根据API限制调整

database:
  connection_pool_size: 20     # 高并发增加连接池
  query_timeout: 60s           # 复杂查询增加超时
```

## 🔍 监控设置

### 1. 系统监控
```bash
# 使用Prometheus监控指标
curl http://localhost:8080/metrics

# 关键指标：
# - fund_protector_risk_score
# - fund_protector_circuit_breaker_triggers
# - fund_protector_emergency_activations
# - fund_protector_auto_transfers
# - fund_protector_response_time
```

### 2. 健康检查
```bash
# 检查系统状态
curl http://localhost:8080/api/v1/fund-protector/status

# 检查资金状态
curl http://localhost:8080/api/v1/fund-protector/fund-status

# 检查保护指标
curl http://localhost:8080/api/v1/fund-protector/metrics
```

### 3. 日志监控
```bash
# 查看实时日志
tail -f logs/fund_protector.log

# 搜索紧急事件
grep "Emergency triggered" logs/fund_protector.log

# 搜索转账记录
grep "Transfer completed" logs/fund_protector.log
```

## 🚨 应急程序

### 1. 紧急停止
```bash
# 停止资金保护器
curl -X POST http://localhost:8080/api/v1/fund-protector/stop

# 或者发送SIGTERM信号
kill -TERM $(pgrep qcat)
```

### 2. 手动触发紧急协议
```bash
# 手动触发紧急事件
curl -X POST http://localhost:8080/api/v1/fund-protector/emergency \
  -H "Content-Type: application/json" \
  -d '{"type": "MANUAL_INTERVENTION", "severity": "HIGH"}'
```

### 3. 重置熔断器
```bash
# 手动重置熔断器
curl -X POST http://localhost:8080/api/v1/fund-protector/circuit-breaker/reset
```

## 🔐 安全检查清单

### 部署前检查
- [ ] 所有API密钥已安全存储
- [ ] 钱包私钥已加密存储
- [ ] 数据库连接使用SSL
- [ ] 网络访问已限制到必要的端口
- [ ] 日志不包含敏感信息
- [ ] 备份和恢复程序已测试

### 运行时监控
- [ ] API调用成功率 > 95%
- [ ] 数据库查询响应时间 < 1s
- [ ] 风险计算响应时间 < 5s
- [ ] 通知发送成功率 > 98%
- [ ] 系统内存使用 < 80%
- [ ] 磁盘空间使用 < 80%

## 🛠️ 故障排除

### 常见问题

#### 1. 交易所API连接失败
```bash
# 检查API密钥
curl -H "X-MBX-APIKEY: $BINANCE_API_KEY" https://api.binance.com/api/v3/account

# 检查网络连接
ping api.binance.com
```

#### 2. 数据库连接问题
```bash
# 测试数据库连接
psql $DATABASE_URL -c "SELECT 1"

# 检查连接池状态
curl http://localhost:8080/api/v1/health/database
```

#### 3. 通知发送失败
```bash
# 测试SMTP连接
telnet smtp.gmail.com 587

# 测试Webhook端点
curl -X POST $WEBHOOK_URL -H "Content-Type: application/json" -d '{"test": true}'
```

#### 4. 钱包操作失败
```bash
# 检查区块链网络连接
curl -X POST $ETH_NETWORK_URL \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

### 日志分析

#### 错误模式识别
```bash
# API错误
grep "exchange.*error" logs/fund_protector.log

# 数据库错误
grep "database.*error" logs/fund_protector.log

# 通知错误
grep "notification.*failed" logs/fund_protector.log

# 转账错误
grep "transfer.*failed" logs/fund_protector.log
```

## 📊 性能调优

### 1. 内存优化
- 调整历史数据缓存大小
- 优化风险计算算法
- 定期清理过期数据

### 2. 网络优化
- 使用连接池
- 启用HTTP/2
- 配置适当的超时时间

### 3. 数据库优化
- 创建适当的索引
- 分区大表
- 定期分析查询性能

## 🔄 升级程序

### 1. 准备升级
```bash
# 备份当前配置
cp config/fund_protector.yaml config/fund_protector.yaml.backup

# 备份数据库
pg_dump qcat_db > backup_$(date +%Y%m%d).sql
```

### 2. 执行升级
```bash
# 停止服务
systemctl stop qcat-fund-protector

# 更新二进制文件
cp bin/qcat bin/qcat.backup
cp new_qcat bin/qcat

# 运行数据库迁移
go run cmd/migrate/main.go up

# 启动服务
systemctl start qcat-fund-protector
```

### 3. 验证升级
```bash
# 检查服务状态
systemctl status qcat-fund-protector

# 检查API响应
curl http://localhost:8080/api/v1/health

# 检查日志
tail -f logs/fund_protector.log
```

## 📞 支持联系

如果在部署过程中遇到问题：

1. 检查日志文件中的错误信息
2. 验证所有环境变量已正确设置
3. 确认所有依赖服务正常运行
4. 参考故障排除部分的指导
5. 查看测试文件中的使用示例

## 🎯 部署验证

部署完成后，执行以下验证步骤：

1. **功能验证**
   ```bash
   # 检查系统状态
   curl http://localhost:8080/api/v1/fund-protector/status
   
   # 检查风险计算
   curl http://localhost:8080/api/v1/fund-protector/risk-assessment
   
   # 测试通知系统
   curl -X POST http://localhost:8080/api/v1/fund-protector/test-notifications
   ```

2. **性能验证**
   ```bash
   # 运行性能测试
   go test -bench=. ./internal/security/protector/
   
   # 检查内存使用
   curl http://localhost:8080/debug/pprof/heap
   ```

3. **安全验证**
   ```bash
   # 检查敏感信息泄露
   grep -r "password\|secret\|key" logs/ || echo "No sensitive data found"
   
   # 验证HTTPS配置
   curl -I https://your-domain.com/api/v1/health
   ```

部署成功后，Fund Protector将自动开始监控您的交易账户，提供全面的资金保护服务。
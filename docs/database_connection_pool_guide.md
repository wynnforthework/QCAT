# 数据库连接池配置指南

## 概述

本指南介绍QCAT系统中数据库连接池的配置和优化策略，确保系统在高并发场景下的稳定性和性能。

## 连接池参数说明

### 核心参数

- **MaxOpen**: 最大打开连接数
  - 默认值: 50
  - 建议值: CPU核心数 * 2 + 有效磁盘数
  - 生产环境: 根据并发用户数和数据库性能调整

- **MaxIdle**: 最大空闲连接数
  - 默认值: 10
  - 建议值: MaxOpen的20-30%
  - 作用: 减少连接创建/销毁的开销

- **ConnMaxLifetime**: 连接最大生命周期
  - 默认值: 1小时
  - 建议值: 1-4小时
  - 作用: 防止连接长时间占用，避免数据库端连接超时

- **ConnMaxIdleTime**: 空闲连接最大时间
  - 默认值: 15分钟
  - 建议值: 10-30分钟
  - 作用: 及时释放长时间空闲的连接

- **Timeout**: 连接超时时间
  - 默认值: 5秒
  - 建议值: 3-10秒
  - 作用: 控制连接建立的超时时间

## 配置建议

### 开发环境
```yaml
database:
  max_open: 25
  max_idle: 5
  timeout: 5s
  conn_max_lifetime: 1h
  conn_max_idle_time: 15m
```

### 测试环境
```yaml
database:
  max_open: 50
  max_idle: 10
  timeout: 5s
  conn_max_lifetime: 1h
  conn_max_idle_time: 15m
```

### 生产环境
```yaml
database:
  max_open: 100
  max_idle: 20
  timeout: 3s
  conn_max_lifetime: 2h
  conn_max_idle_time: 10m
```

### 高并发环境
```yaml
database:
  max_open: 200
  max_idle: 50
  timeout: 2s
  conn_max_lifetime: 4h
  conn_max_idle_time: 5m
```

## 监控指标

### Prometheus指标

系统提供以下数据库连接池监控指标：

- `db_pool_max_open_connections`: 最大连接数
- `db_pool_open_connections`: 当前打开连接数
- `db_pool_in_use_connections`: 正在使用的连接数
- `db_pool_idle_connections`: 空闲连接数
- `db_pool_wait_count_total`: 等待连接的总次数
- `db_pool_wait_duration_seconds`: 等待连接的时长分布
- `db_pool_max_idle_closed_total`: 因MaxIdle关闭的连接数
- `db_pool_max_lifetime_closed_total`: 因MaxLifetime关闭的连接数

### 健康检查

系统提供数据库连接池健康检查API：

```bash
GET /health
```

返回示例：
```json
{
  "status": "ok",
  "time": "2024-01-01T00:00:00Z",
  "services": {
    "database": "ok",
    "redis": "ok"
  }
}
```

### 详细状态查询

```bash
GET /api/v1/metrics/system
```

返回数据库连接池详细状态：
```json
{
  "database_pool": {
    "healthy": true,
    "max_open_connections": 50,
    "open_connections": 15,
    "in_use": 8,
    "idle": 7,
    "wait_count": 0,
    "wait_duration": "0s",
    "utilization_percent": 16.0
  }
}
```

## 性能优化建议

### 1. 连接数计算

**公式**: `MaxOpen = (CPU核心数 * 2) + 有效磁盘数`

**示例**:
- 4核CPU + 1个SSD: MaxOpen = (4 * 2) + 1 = 9
- 8核CPU + 2个SSD: MaxOpen = (8 * 2) + 2 = 18

### 2. 空闲连接优化

- **MaxIdle**: 设置为MaxOpen的20-30%
- **ConnMaxIdleTime**: 设置为10-30分钟
- 避免设置过大的MaxIdle，防止资源浪费

### 3. 连接生命周期

- **ConnMaxLifetime**: 设置为1-4小时
- 考虑数据库端的连接超时设置
- 避免频繁的连接创建/销毁

### 4. 超时设置

- **Timeout**: 设置为3-10秒
- 根据网络延迟调整
- 避免设置过短导致连接失败

## 故障排查

### 常见问题

1. **连接池耗尽**
   - 症状: 大量wait_count
   - 解决: 增加MaxOpen或优化SQL查询

2. **连接泄漏**
   - 症状: 持续增长的open_connections
   - 解决: 检查是否有未关闭的数据库连接

3. **连接超时**
   - 症状: 频繁的连接创建/销毁
   - 解决: 调整ConnMaxLifetime和ConnMaxIdleTime

4. **性能下降**
   - 症状: 高wait_duration
   - 解决: 优化数据库查询或增加连接数

### 监控告警

建议设置以下告警规则：

```yaml
# 连接池利用率过高
- alert: DatabasePoolHighUtilization
  expr: db_pool_in_use_connections / db_pool_max_open_connections > 0.8
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Database connection pool utilization is high"

# 连接等待时间过长
- alert: DatabasePoolHighWaitTime
  expr: rate(db_pool_wait_duration_seconds_sum[5m]) > 0.1
  for: 2m
  labels:
    severity: warning
  annotations:
    summary: "Database connection wait time is high"

# 连接池耗尽
- alert: DatabasePoolExhausted
  expr: db_pool_wait_count_total > 100
  for: 1m
  labels:
    severity: critical
  annotations:
    summary: "Database connection pool is exhausted"
```

## 最佳实践

1. **定期监控**: 使用Prometheus监控连接池状态
2. **压力测试**: 在生产环境部署前进行压力测试
3. **渐进调整**: 根据实际负载逐步调整参数
4. **文档记录**: 记录配置变更和性能影响
5. **备份配置**: 保存已知良好的配置参数

## 总结

合理的数据库连接池配置是系统稳定性的重要保障。通过监控、调优和最佳实践，可以确保系统在高并发场景下的稳定运行。

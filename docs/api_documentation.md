# QCAT API 文档

## 概述

QCAT (Quantitative Cat) 是一个量化交易系统，提供完整的API接口用于策略管理、交易执行、风险控制等功能。

## 基础信息

- **基础URL**: `http://localhost:8082`
- **API版本**: `v1`
- **认证方式**: JWT Bearer Token
- **数据格式**: JSON

## 认证

### 获取访问令牌

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "your_username",
  "password": "your_password"
}
```

**响应示例**:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 3600,
  "user": {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "role": "admin"
  }
}
```

### 使用访问令牌

在请求头中添加：
```
Authorization: Bearer <your_token>
```

## 用户管理

### 用户注册

```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "new_user",
  "email": "user@example.com",
  "password": "secure_password"
}
```

### 获取用户信息

```http
GET /api/v1/users/profile
Authorization: Bearer <token>
```

### 更新用户信息

```http
PUT /api/v1/users/profile
Authorization: Bearer <token>
Content-Type: application/json

{
  "email": "new_email@example.com",
  "full_name": "New Full Name"
}
```

## 策略管理

### 创建策略

```http
POST /api/v1/strategies
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "MA Cross Strategy",
  "description": "Moving Average Crossover Strategy",
  "symbols": ["BTCUSDT", "ETHUSDT"],
  "parameters": {
    "fast_period": 10,
    "slow_period": 20,
    "position_size": 0.1
  },
  "risk_limits": {
    "max_position_size": 0.2,
    "max_drawdown": 0.1,
    "stop_loss": 0.05
  }
}
```

### 获取策略列表

```http
GET /api/v1/strategies
Authorization: Bearer <token>
```

**查询参数**:
- `status`: 策略状态 (active, inactive, paused)
- `page`: 页码 (默认: 1)
- `limit`: 每页数量 (默认: 20)

### 获取策略详情

```http
GET /api/v1/strategies/{strategy_id}
Authorization: Bearer <token>
```

### 更新策略

```http
PUT /api/v1/strategies/{strategy_id}
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Updated Strategy Name",
  "parameters": {
    "fast_period": 15,
    "slow_period": 25
  }
}
```

### 启动策略

```http
POST /api/v1/strategies/{strategy_id}/start
Authorization: Bearer <token>
```

### 停止策略

```http
POST /api/v1/strategies/{strategy_id}/stop
Authorization: Bearer <token>
```

### 删除策略

```http
DELETE /api/v1/strategies/{strategy_id}
Authorization: Bearer <token>
```

## 交易管理

### 获取订单列表

```http
GET /api/v1/orders
Authorization: Bearer <token>
```

**查询参数**:
- `strategy_id`: 策略ID
- `symbol`: 交易对
- `status`: 订单状态 (pending, filled, cancelled, rejected)
- `start_date`: 开始日期 (YYYY-MM-DD)
- `end_date`: 结束日期 (YYYY-MM-DD)
- `page`: 页码
- `limit`: 每页数量

### 获取订单详情

```http
GET /api/v1/orders/{order_id}
Authorization: Bearer <token>
```

### 取消订单

```http
POST /api/v1/orders/{order_id}/cancel
Authorization: Bearer <token>
```

### 获取持仓信息

```http
GET /api/v1/positions
Authorization: Bearer <token>
```

**查询参数**:
- `strategy_id`: 策略ID
- `symbol`: 交易对

### 获取交易历史

```http
GET /api/v1/trades
Authorization: Bearer <token>
```

**查询参数**:
- `strategy_id`: 策略ID
- `symbol`: 交易对
- `start_date`: 开始日期
- `end_date`: 结束日期
- `page`: 页码
- `limit`: 每页数量

## 风险管理

### 获取风险指标

```http
GET /api/v1/risk/metrics
Authorization: Bearer <token>
```

**响应示例**:
```json
{
  "total_pnl": 1250.50,
  "daily_pnl": 45.20,
  "max_drawdown": 0.08,
  "sharpe_ratio": 1.25,
  "var_95": 0.05,
  "position_concentration": 0.15
}
```

### 设置风险限制

```http
POST /api/v1/risk/limits
Authorization: Bearer <token>
Content-Type: application/json

{
  "max_position_size": 0.2,
  "max_drawdown": 0.1,
  "daily_loss_limit": 1000,
  "position_concentration_limit": 0.3
}
```

### 获取风险事件

```http
GET /api/v1/risk/events
Authorization: Bearer <token>
```

## 市场数据

### 获取实时价格

```http
GET /api/v1/market/price/{symbol}
Authorization: Bearer <token>
```

### 获取K线数据

```http
GET /api/v1/market/klines/{symbol}
Authorization: Bearer <token>
```

**查询参数**:
- `interval`: 时间间隔 (1m, 5m, 15m, 1h, 4h, 1d)
- `limit`: 数据条数 (默认: 100, 最大: 1000)
- `start_time`: 开始时间戳
- `end_time`: 结束时间戳

### 获取深度数据

```http
GET /api/v1/market/depth/{symbol}
Authorization: Bearer <token>
```

**查询参数**:
- `limit`: 深度条数 (默认: 20, 最大: 100)

## 系统监控

### 系统健康检查

```http
GET /api/v1/health
```

**响应示例**:
```json
{
  "status": "ok",
  "time": "2024-01-01T12:00:00Z",
  "services": {
    "database": "ok",
    "redis": "ok",
    "memory": "ok",
    "network": "ok",
    "health": "ok"
  }
}
```

### 获取系统指标

```http
GET /api/v1/metrics
Authorization: Bearer <token>
```

### 获取性能指标

```http
GET /api/v1/performance/baselines
Authorization: Bearer <token>
```

### 获取监控大屏数据

```http
GET /api/v1/dashboard
Authorization: Bearer <token>
```

## 备份管理

### 创建备份

```http
POST /api/v1/backup
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "manual_backup",
  "type": "full",
  "description": "Manual backup before deployment"
}
```

### 获取备份列表

```http
GET /api/v1/backup
Authorization: Bearer <token>
```

### 恢复备份

```http
POST /api/v1/backup/{backup_id}/restore
Authorization: Bearer <token>
Content-Type: application/json

{
  "target": "/path/to/restore"
}
```

### 删除备份

```http
DELETE /api/v1/backup/{backup_id}
Authorization: Bearer <token>
```

## 告警管理

### 发送告警

```http
POST /api/v1/alerts
Authorization: Bearer <token>
Content-Type: application/json

{
  "level": "warning",
  "title": "High CPU Usage",
  "message": "CPU usage is above 80%",
  "source": "system_monitor",
  "channels": ["email", "dingtalk"]
}
```

### 获取告警历史

```http
GET /api/v1/alerts
Authorization: Bearer <token>
```

## 错误码

| 错误码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 403 | 禁止访问 |
| 404 | 资源不存在 |
| 422 | 请求格式正确但语义错误 |
| 429 | 请求过于频繁 |
| 500 | 服务器内部错误 |
| 503 | 服务不可用 |

## 错误响应格式

```json
{
  "error": {
    "code": 400,
    "message": "Invalid request parameters",
    "details": {
      "field": "username",
      "reason": "Username is required"
    }
  }
}
```

## 分页响应格式

```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 100,
    "pages": 5
  }
}
```

## 速率限制

- **认证接口**: 5次/分钟
- **普通接口**: 100次/分钟
- **数据查询接口**: 1000次/分钟

超过限制时返回429状态码。

## WebSocket接口

### 连接

```
ws://localhost:8082/ws
```

### 认证

连接后发送认证消息：
```json
{
  "type": "auth",
  "token": "your_jwt_token"
}
```

### 订阅市场数据

```json
{
  "type": "subscribe",
  "channel": "market_data",
  "symbols": ["BTCUSDT", "ETHUSDT"]
}
```

### 订阅策略更新

```json
{
  "type": "subscribe",
  "channel": "strategy_updates",
  "strategy_id": "strategy_123"
}
```

### 消息格式

```json
{
  "type": "market_data",
  "symbol": "BTCUSDT",
  "data": {
    "price": 50000.0,
    "volume": 100.5,
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

## 更新日志

### v1.0.0 (2024-01-01)
- 初始版本发布
- 支持基础策略管理
- 支持交易执行
- 支持风险管理
- 支持系统监控

## 支持

如有问题，请联系：
- 邮箱: support@qcat.com
- 文档: https://docs.qcat.com
- GitHub: https://github.com/qcat/qcat

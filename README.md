# QCAT - 量化合约自动化交易系统

QCAT是一个全面的加密货币合约自动化交易系统，具有先进的量化策略、风险管理和投资组合优化功能。

## 功能特性

- 完全自动化交易，具备10大核心自动化能力
- 先进的风险管理和仓位控制
- 策略优化和回测分析
- 实时市场数据处理
- 投资组合管理和优化
- 热门市场检测和分析
- 基于React和shadcn/ui的现代化Web界面
- RESTful API和WebSocket支持

## 技术栈

- 后端: Go + Gin + WebSocket
- 前端: React + Next.js + shadcn/ui
- 数据库: PostgreSQL
- 缓存: Redis
- CI/CD: GitHub Actions

## 项目结构

```
.
├── cmd/                    # 应用程序入口点
├── internal/              # 私有应用程序代码
│   ├── api/              # API服务器和处理器
│   ├── config/           # 配置处理
│   ├── market/           # 市场数据处理
│   ├── exchange/         # 交易所连接
│   ├── strategy/         # 交易策略
│   ├── risk/             # 风险管理
│   ├── portfolio/        # 投资组合管理
│   ├── optimizer/        # 策略优化
│   ├── backtest/         # 回测引擎
│   ├── hotlist/          # 热门市场分析
│   └── monitor/          # 系统监控
├── pkg/                   # 公共库
├── api/                   # API定义
├── frontend/             # React前端应用
├── configs/              # 配置文件
├── scripts/              # 工具脚本
├── docs/                 # 文档
└── test/                 # 测试文件
```

## 快速开始

### 前置要求

- Go 1.21 或更高版本
- Node.js 20 或更高版本
- PostgreSQL 15 或更高版本
- Redis 7 或更高版本

### 安装步骤

1. 克隆仓库:
   ```bash
   git clone <repository-url>
   cd QCAT
   ```

2. 安装Go依赖:
   ```bash
   go mod download
   ```

3. 安装Node.js依赖:
   ```bash
   cd frontend
   npm install
   ```

4. 配置应用程序:
   ```bash
   cp configs/config.yaml.example configs/config.yaml
   # 编辑 configs/config.yaml 文件，填入您的设置
   ```

5. 启动后端服务器:
   ```bash
   go run cmd/qcat/main.go
   ```

6. 启动前端开发服务器:
   ```bash
   cd frontend
   npm run dev
   ```

## API文档

### REST API端点

#### 优化器
- `POST /api/v1/optimizer/run` - 启动优化任务
- `GET /api/v1/optimizer/tasks` - 列出优化任务
- `GET /api/v1/optimizer/tasks/:id` - 获取优化任务详情
- `GET /api/v1/optimizer/results/:id` - 获取优化结果

#### 策略
- `GET /api/v1/strategy/` - 列出策略
- `GET /api/v1/strategy/:id` - 获取策略详情
- `POST /api/v1/strategy/` - 创建新策略
- `PUT /api/v1/strategy/:id` - 更新策略
- `DELETE /api/v1/strategy/:id` - 删除策略
- `POST /api/v1/strategy/:id/promote` - 升级策略版本
- `POST /api/v1/strategy/:id/start` - 启动策略
- `POST /api/v1/strategy/:id/stop` - 停止策略
- `POST /api/v1/strategy/:id/backtest` - 运行回测

#### 投资组合
- `GET /api/v1/portfolio/overview` - 获取投资组合概览
- `GET /api/v1/portfolio/allocations` - 获取投资组合配置
- `POST /api/v1/portfolio/rebalance` - 触发再平衡
- `GET /api/v1/portfolio/history` - 获取投资组合历史

#### 风险控制
- `GET /api/v1/risk/overview` - 获取风险概览
- `GET /api/v1/risk/limits` - 获取风险限额
- `POST /api/v1/risk/limits` - 设置风险限额
- `GET /api/v1/risk/circuit-breakers` - 获取熔断器
- `POST /api/v1/risk/circuit-breakers` - 设置熔断器
- `GET /api/v1/risk/violations` - 获取风险违规记录

#### 热门币种
- `GET /api/v1/hotlist/symbols` - 获取热门币种
- `POST /api/v1/hotlist/approve` - 批准币种交易
- `GET /api/v1/hotlist/whitelist` - 获取白名单
- `POST /api/v1/hotlist/whitelist` - 添加到白名单
- `DELETE /api/v1/hotlist/whitelist/:symbol` - 从白名单移除

#### 指标监控
- `GET /api/v1/metrics/strategy/:id` - 获取策略指标
- `GET /api/v1/metrics/system` - 获取系统指标
- `GET /api/v1/metrics/performance` - 获取性能指标

#### 审计日志
- `GET /api/v1/audit/logs` - 获取审计日志
- `GET /api/v1/audit/decisions` - 获取决策链
- `GET /api/v1/audit/performance` - 获取性能指标
- `POST /api/v1/audit/export` - 导出审计报告

### WebSocket端点

- `ws://localhost:8082/ws/market/:symbol` - 实时市场数据
- `ws://localhost:8082/ws/strategy/:id` - 策略状态更新
- `ws://localhost:8082/ws/alerts` - 告警通知

### 健康检查

- `GET /health` - 服务器健康状态

## 配置说明

应用程序配置存储在 `configs/config.yaml` 文件中:

```yaml
app:
  name: qcat
  version: 1.0.0
  env: development

server:
  host: localhost
  port: 8082

database:
  driver: postgres
  host: localhost
  port: 5432
  name: qcat
  user: postgres
  password: ""
  sslmode: disable

redis:
  enabled: true
  host: localhost
  port: 6379
  password: ""
  db: 0

exchange:
  binance:
    api_key: ""
    api_secret: ""
    testnet: true
    rate_limit: 1200

risk:
  max_leverage: 10
  max_position_size: 100000
  max_drawdown: 0.1
  circuit_breaker_threshold: 0.05
```

## 开发指南

### 运行测试

```bash
go test ./...
```

### 构建应用

```bash
go build -o bin/qcat cmd/qcat/main.go
```

### Docker部署

```bash
docker build -t qcat .
docker run -p 8080:8080 qcat
```

## 贡献指南

1. Fork 本仓库
2. 创建功能分支
3. 进行您的修改
4. 添加测试
5. 提交 Pull Request

## 许可证

本项目采用 MIT 许可证 - 详情请参阅 LICENSE 文件。

## 支持

如需支持和问题咨询，请在 GitHub 上提交 Issue 或联系开发团队。
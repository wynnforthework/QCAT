## 📡 API文档

### 🧠 智能化API端点

#### Intelligence Layer - 智能决策控制
- `GET /api/v1/intelligence/status` - 获取智能化系统状态
- `POST /api/v1/intelligence/optimize` - 触发智能优化
- `GET /api/v1/intelligence/metrics` - 获取智能化指标
- `POST /api/v1/intelligence/config` - 更新智能化配置

#### Position Management - 智能仓位管理
- `GET /api/v1/position/optimizer/status` - 动态优化器状态
- `POST /api/v1/position/rebalance` - 触发智能再平衡
- `GET /api/v1/position/allocation` - 获取智能分配结果
- `GET /api/v1/position/performance` - 获取仓位性能分析

#### Trading Execution - 智能交易执行
- `POST /api/v1/trading/execute` - 智能订单执行
- `GET /api/v1/trading/algorithms` - 获取执行算法
- `GET /api/v1/trading/performance` - 获取执行性能
- `POST /api/v1/trading/optimize` - 优化执行参数

#### Fund Management - 资金管理自动化
- `GET /api/v1/fund/layers` - 获取分层管理状态
- `POST /api/v1/fund/hedge` - 创建智能对冲
- `GET /api/v1/fund/hedge/performance` - 对冲性能分析
- `POST /api/v1/fund/protection/enable` - 启用资金保护

#### Security & Protection - 安全防护系统
- `GET /api/v1/security/guardian/status` - 安全监控状态
- `GET /api/v1/security/threats` - 获取威胁检测结果
- `POST /api/v1/security/protect` - 触发保护机制
- `GET /api/v1/security/alerts` - 获取安全告警

#### Analysis Automation - 分析自动化
- `POST /api/v1/analysis/backtest/auto` - 启动自动回测
- `GET /api/v1/analysis/factors/discovered` - 获取发现的因子
- `POST /api/v1/analysis/factors/discover` - 启动因子挖掘
- `GET /api/v1/analysis/performance` - 获取分析性能

#### Operations & Healing - 运维自愈
- `GET /api/v1/operations/routing/status` - 智能路由状态
- `POST /api/v1/operations/failover` - 触发故障转移
- `GET /api/v1/operations/healing/history` - 自愈历史记录
- `POST /api/v1/operations/healing/trigger` - 手动触发自愈

#### AutoML & Learning - 学习进化
- `GET /api/v1/automl/status` - AutoML引擎状态
- `POST /api/v1/automl/train` - 启动模型训练
- `GET /api/v1/automl/models` - 获取训练模型列表
- `POST /api/v1/automl/deploy` - 部署模型到生产

### 📊 传统API端点

#### 策略管理
- `GET /api/v1/strategy/` - 列出策略
- `POST /api/v1/strategy/` - 创建新策略
- `GET /api/v1/strategy/:id` - 获取策略详情
- `PUT /api/v1/strategy/:id` - 更新策略
- `POST /api/v1/strategy/:id/start` - 启动策略
- `POST /api/v1/strategy/:id/stop` - 停止策略

#### 投资组合
- `GET /api/v1/portfolio/overview` - 获取投资组合概览
- `GET /api/v1/portfolio/allocations` - 获取投资组合配置
- `GET /api/v1/portfolio/history` - 获取投资组合历史

#### 风险控制
- `GET /api/v1/risk/overview` - 获取风险概览
- `GET /api/v1/risk/limits` - 获取风险限额
- `POST /api/v1/risk/limits` - 设置风险限额
- `GET /api/v1/risk/circuit-breakers` - 获取熔断器状态

#### 系统监控
- `GET /api/v1/metrics/system` - 获取系统指标
- `GET /api/v1/metrics/performance` - 获取性能指标
- `GET /api/v1/audit/logs` - 获取审计日志

### 🔗 WebSocket端点

#### 实时数据流
- `ws://localhost:8082/ws/market/:symbol` - 实时市场数据
- `ws://localhost:8082/ws/strategy/:id` - 策略状态更新
- `ws://localhost:8082/ws/alerts` - 系统告警通知

#### 智能化实时监控
- `ws://localhost:8082/ws/intelligence/status` - 智能化系统状态
- `ws://localhost:8082/ws/automl/training` - ML训练进度
- `ws://localhost:8082/ws/healing/events` - 自愈事件流
- `ws://localhost:8082/ws/routing/decisions` - 路由决策流

### 🏥 系统健康检查

- `GET /health` - 服务器基础健康状态
- `GET /health/deep` - 深度健康检查（包括所有智能化组件）
- `GET /health/intelligence` - 智能化系统专项检查
- `GET /health/dependencies` - 外部依赖健康状态

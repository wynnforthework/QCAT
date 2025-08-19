## 📡 API文档

### ✅ 已实现的API端点

#### 认证管理
- `POST /api/v1/auth/login` - 用户登录
- `POST /api/v1/auth/register` - 用户注册
- `POST /api/v1/auth/refresh` - 刷新访问令牌

#### 仪表板
- `GET /api/v1/dashboard` - 获取仪表板概览数据

#### 市场数据
- `GET /api/v1/market/data` - 获取市场数据

#### 交易活动
- `GET /api/v1/trading/activity` - 获取交易活动记录

#### 策略管理
- `GET /api/v1/strategy/` - 列出策略
- `POST /api/v1/strategy/` - 创建新策略
- `GET /api/v1/strategy/:id` - 获取策略详情
- `PUT /api/v1/strategy/:id` - 更新策略
- `DELETE /api/v1/strategy/:id` - 删除策略
- `POST /api/v1/strategy/:id/promote` - 推广策略
- `POST /api/v1/strategy/:id/start` - 启动策略
- `POST /api/v1/strategy/:id/stop` - 停止策略
- `POST /api/v1/strategy/:id/backtest` - 运行策略回测

#### 优化器
- `POST /api/v1/optimizer/run` - 运行优化
- `GET /api/v1/optimizer/tasks` - 获取优化任务列表
- `GET /api/v1/optimizer/tasks/:id` - 获取优化任务详情
- `GET /api/v1/optimizer/results/:id` - 获取优化结果

#### 投资组合
- `GET /api/v1/portfolio/overview` - 获取投资组合概览
- `GET /api/v1/portfolio/allocations` - 获取投资组合配置
- `POST /api/v1/portfolio/rebalance` - 触发投资组合再平衡
- `GET /api/v1/portfolio/history` - 获取投资组合历史

#### 风险控制
- `GET /api/v1/risk/overview` - 获取风险概览
- `GET /api/v1/risk/limits` - 获取风险限额
- `POST /api/v1/risk/limits` - 设置风险限额
- `GET /api/v1/risk/circuit-breakers` - 获取熔断器状态
- `POST /api/v1/risk/circuit-breakers` - 设置熔断器
- `GET /api/v1/risk/violations` - 获取风险违规记录

#### 热门列表
- `GET /api/v1/hotlist/symbols` - 获取热门符号
- `POST /api/v1/hotlist/approve` - 批准符号
- `GET /api/v1/hotlist/whitelist` - 获取白名单
- `POST /api/v1/hotlist/whitelist` - 添加到白名单
- `DELETE /api/v1/hotlist/whitelist/:symbol` - 从白名单移除

#### 系统监控
- `GET /api/v1/metrics/system` - 获取系统指标
- `GET /api/v1/metrics/strategy/:id` - 获取策略指标
- `GET /api/v1/metrics/performance` - 获取性能指标

#### 内存管理
- `GET /api/v1/memory/stats` - 获取内存统计
- `POST /api/v1/memory/gc` - 强制垃圾回收

#### 网络管理
- `GET /api/v1/network/connections` - 获取网络连接
- `GET /api/v1/network/connections/:id` - 获取单个连接详情
- `POST /api/v1/network/connections/:id/reconnect` - 重新连接

#### 健康检查
- `GET /health` - 基础健康检查
- `GET /api/v1/health/status` - 详细健康状态
- `GET /api/v1/health/checks` - 所有健康检查
- `GET /api/v1/health/checks/:name` - 单个健康检查
- `POST /api/v1/health/checks/:name/force` - 强制健康检查

#### 系统管理
- `GET /api/v1/shutdown/status` - 获取关闭状态
- `POST /api/v1/shutdown/graceful` - 优雅关闭
- `POST /api/v1/shutdown/force` - 强制关闭

#### 审计日志
- `GET /api/v1/audit/logs` - 获取审计日志
- `GET /api/v1/audit/decisions` - 获取决策链
- `GET /api/v1/audit/performance` - 获取审计性能
- `POST /api/v1/audit/export` - 导出审计报告

#### 缓存管理
- `GET /api/v1/cache/status` - 缓存状态
- `GET /api/v1/cache/health` - 缓存健康
- `GET /api/v1/cache/metrics` - 缓存指标
- `GET /api/v1/cache/events` - 缓存事件
- `GET /api/v1/cache/config` - 缓存配置
- `POST /api/v1/cache/test` - 测试缓存
- `POST /api/v1/cache/fallback/force` - 强制降级
- `POST /api/v1/cache/counters/reset` - 重置计数器

#### 安全管理
- `GET /api/v1/security/keys/` - API密钥列表
- `POST /api/v1/security/keys/` - 创建API密钥
- `GET /api/v1/security/keys/:keyId` - 获取API密钥
- `POST /api/v1/security/keys/:keyId/rotate` - 轮换密钥
- `POST /api/v1/security/keys/:keyId/revoke` - 撤销密钥
- `GET /api/v1/security/keys/:keyId/usage` - 密钥使用情况
- `GET /api/v1/security/audit/logs` - 安全审计日志
- `GET /api/v1/security/audit/integrity` - 完整性验证

#### 编排器管理
- `GET /api/v1/orchestrator/status` - 编排器状态
- `GET /api/v1/orchestrator/services` - 服务列表
- `POST /api/v1/orchestrator/services/start` - 启动服务
- `POST /api/v1/orchestrator/services/stop` - 停止服务
- `POST /api/v1/orchestrator/services/restart` - 重启服务
- `POST /api/v1/orchestrator/optimize` - 编排器优化
- `GET /api/v1/orchestrator/health` - 编排器健康

### 🔗 WebSocket端点 (已实现)

#### 实时数据流
- `ws://localhost:8082/ws/market/:symbol` - 实时市场数据
- `ws://localhost:8082/ws/strategy/:id` - 策略状态更新
- `ws://localhost:8082/ws/alerts` - 系统告警通知

### 🚧 计划中的智能化API端点 (未实现)

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

#### 智能化实时监控 (WebSocket - 计划中)
- `ws://localhost:8082/ws/intelligence/status` - 智能化系统状态
- `ws://localhost:8082/ws/automl/training` - ML训练进度
- `ws://localhost:8082/ws/healing/events` - 自愈事件流
- `ws://localhost:8082/ws/routing/decisions` - 路由决策流

### 📊 API统计

- **已实现接口**: 约 70+ 个
- **计划中接口**: 约 30+ 个
- **WebSocket接口**: 3个已实现，4个计划中
- **总计**: 约 100+ 个接口

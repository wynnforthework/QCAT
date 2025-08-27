# 策略API整合方案

## 🎯 目标
消除策略管理和策略池管理的功能重复，提供统一的策略管理体验。

## 📊 当前问题
1. **功能重复**：双闭环系统和常规策略管理都提供策略列表和状态查询
2. **数据不一致**：两套API可能返回不同的策略状态信息
3. **用户困惑**：前端需要调用多个API获取完整的策略信息

## 🚀 整合方案

### 方案A：统一策略API (推荐)

#### 新的API结构
```
/api/v1/strategy/
├── GET    /                     # 策略列表 (包含池状态)
├── POST   /                     # 创建策略
├── GET    /:id                  # 策略详情 (包含执行状态)
├── PUT    /:id                  # 更新策略
├── DELETE /:id                  # 删除策略
├── POST   /:id/start            # 启动策略
├── POST   /:id/stop             # 停止策略
├── POST   /:id/backtest         # 策略回测
├── GET    /pool/overview        # 策略池概览
├── GET    /pool/distribution    # 策略分布统计
├── GET    /execution/overview   # 执行系统概览
├── GET    /execution/realtime   # 实时执行状态
└── GET    /workflow/status      # 工作流状态
```

#### 统一的响应格式
```json
{
  "id": "strategy_001",
  "name": "动量策略Alpha",
  "type": "momentum",
  "status": "active",
  "lifecycle": {
    "stage": "production",
    "version": "1.2.0",
    "createdAt": "2024-01-15T10:00:00Z",
    "updatedAt": "2024-01-20T15:30:00Z"
  },
  "execution": {
    "isRunning": true,
    "lastExecution": "2024-01-20T16:45:00Z",
    "executionCount": 1234,
    "successRate": 98.7,
    "avgLatency": 8.5
  },
  "performance": {
    "pnl": 15420.50,
    "sharpeRatio": 2.34,
    "maxDrawdown": 0.08,
    "winRate": 0.67,
    "totalReturn": 0.156
  },
  "pool": {
    "poolStatus": "enabled",
    "priority": "high",
    "resourceAllocation": {
      "cpu": 1.2,
      "memory": 2.1
    }
  }
}
```

### 方案B：保持分离但明确职责

#### 常规策略管理 (/api/v1/strategy/*)
- **职责**：策略生命周期管理
- **功能**：CRUD操作、配置管理、版本控制、回测

#### 双闭环系统 (/api/v1/dual-loop/*)
- **职责**：运行时监控和协调
- **功能**：实时状态、性能监控、资源分配、系统协调

## 🔧 实施步骤

### 第一阶段：API整合
1. **创建统一的策略服务层**
   ```go
   type UnifiedStrategyService struct {
       strategyManager    *StrategyManager
       executionMonitor   *ExecutionMonitor
       poolManager        *PoolManager
       workflowManager    *WorkflowManager
   }
   ```

2. **重构API处理器**
   ```go
   type UnifiedStrategyHandler struct {
       service *UnifiedStrategyService
   }
   
   func (h *UnifiedStrategyHandler) GetStrategy(c *gin.Context) {
       // 合并策略基本信息 + 执行状态 + 池状态
   }
   ```

3. **更新路由配置**
   ```go
   strategy := protected.Group("/strategy")
   {
       strategy.GET("/", handler.ListStrategies)           // 包含池视图
       strategy.GET("/:id", handler.GetStrategy)           // 完整信息
       strategy.GET("/pool/overview", handler.GetPoolOverview)
       strategy.GET("/execution/realtime", handler.GetExecutionStatus)
   }
   ```

### 第二阶段：前端适配
1. **统一数据模型**
   ```typescript
   interface UnifiedStrategy {
       id: string;
       name: string;
       lifecycle: LifecycleInfo;
       execution: ExecutionInfo;
       performance: PerformanceInfo;
       pool: PoolInfo;
   }
   ```

2. **合并页面组件**
   - 将策略池管理功能集成到策略管理页面
   - 添加视图切换：列表视图 / 池视图 / 执行视图

### 第三阶段：数据一致性
1. **统一数据源**
   - 确保所有策略状态从同一数据源获取
   - 实现实时数据同步机制

2. **缓存策略**
   - 实现统一的缓存层
   - 避免重复查询和数据不一致

## 📈 预期效果

### 用户体验改进
- ✅ 单一入口获取完整策略信息
- ✅ 减少API调用次数
- ✅ 统一的数据格式和状态定义
- ✅ 更好的实时性和一致性

### 开发维护改进
- ✅ 减少代码重复
- ✅ 统一的业务逻辑
- ✅ 更容易的功能扩展
- ✅ 简化的测试和调试

## 🎯 推荐实施方案A

**理由**：
1. **用户友好**：提供统一的策略管理体验
2. **数据一致**：避免多个API返回不同信息
3. **维护简单**：减少重复代码和逻辑
4. **扩展性好**：便于后续功能添加

**实施优先级**：
1. 🔴 **高优先级**：合并策略列表和详情API
2. 🟡 **中优先级**：整合执行状态和性能监控
3. 🟢 **低优先级**：优化缓存和实时同步
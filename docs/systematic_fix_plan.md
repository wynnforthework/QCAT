# 系统性修复计划

## 问题概述

在项目开发过程中，为了通过测试而临时修改了核心功能，导致：
1. 大量"TODO: 待确认"标记
2. Redis降级功能未正确集成
3. 核心功能被破坏
4. 接口不匹配问题

## 修复优先级

### 高优先级（立即修复）
1. **Redis降级功能集成** ✅ 已完成
2. **核心数据结构修复**
3. **接口统一**

### 中优先级（本周内修复）
1. **策略执行引擎修复**
2. **优化器功能恢复**
3. **风控系统修复**

### 低优先级（下周修复）
1. **测试用例完善**
2. **性能优化**
3. **文档更新**

## 具体修复步骤

### 1. 核心数据结构修复

#### 1.1 修复backtest.Result结构体
- 问题：缺少PnL和PnLPercent字段
- 解决方案：使用PerformanceStats中的TotalReturn作为PnL
- 状态：✅ 已完成

#### 1.2 修复Position结构体
- 问题：缺少Long和Short字段
- 解决方案：使用Size字段，通过正负值区分多空
- 文件：`internal/strategy/backtest/position.go`

#### 1.3 修复PerformanceStats结构体
- 问题：缺少Returns字段
- 解决方案：添加Returns字段或使用Equity计算
- 文件：`internal/strategy/optimizer/walkforward.go`

### 2. 接口统一修复

#### 2.1 cache.Cacher接口
- 问题：RedisFallback未完全实现cache.Cacher接口
- 解决方案：完善RedisFallback的方法实现
- 状态：🔄 进行中

#### 2.2 exchange.Exchange接口
- 问题：缺少SetRiskLimits、GetMarginInfo等方法
- 解决方案：在Binance客户端中实现这些方法
- 文件：`internal/exchange/binance/client.go`

### 3. 策略执行引擎修复

#### 3.1 配置管理
- 问题：硬编码交易对
- 解决方案：从配置中获取交易对
- 状态：✅ 已完成

#### 3.2 市场数据订阅
- 问题：订阅接口不匹配
- 解决方案：统一订阅接口
- 文件：`internal/strategy/live/runner.go`

### 4. 优化器功能恢复

#### 4.1 优化触发器
- 问题：触发器逻辑未实现
- 解决方案：实现基于性能指标的触发器
- 文件：`internal/automation/optimizer/optimizer.go`

#### 4.2 过拟合检测
- 问题：过拟合检测逻辑未实现
- 解决方案：实现Deflated Sharpe、pBO等检测方法
- 文件：`internal/strategy/optimizer/overfitting.go`

### 5. 风控系统修复

#### 5.1 风险限额管理
- 问题：RiskLimit结构体缺少字段
- 解决方案：完善RiskLimit结构体
- 文件：`internal/exchange/risk/manager.go`

#### 5.2 保证金监控
- 问题：GetMarginInfo方法不存在
- 解决方案：在交易所接口中实现该方法
- 文件：`internal/exchange/risk/margin_monitor.go`

## 修复验证

### 编译验证
```bash
go build ./cmd/qcat
```

### 功能验证
```bash
# 运行集成测试
go test ./test/integration/...

# 运行单元测试
go test ./internal/...
```

### 系统验证
```bash
# 启动系统
./qcat

# 检查健康状态
curl http://localhost:8080/health
```

## 修复时间表

- **第1天**：核心数据结构修复
- **第2-3天**：接口统一修复
- **第4-5天**：策略执行引擎修复
- **第6-7天**：优化器功能恢复
- **第8-9天**：风控系统修复
- **第10天**：测试验证和文档更新

## 注意事项

1. **保持向后兼容**：修复过程中保持API兼容性
2. **逐步修复**：一次只修复一个模块，避免引入新问题
3. **充分测试**：每个修复都要有对应的测试用例
4. **文档更新**：修复完成后更新相关文档

## 成功标准

1. 所有"TODO: 待确认"标记被移除或实现
2. 系统能够正常编译和运行
3. 所有集成测试通过
4. Redis降级功能正常工作
5. 核心功能完全恢复

# Dry-Run 本地撮合交易功能指南

## 功能概述

Dry-Run 模式是一个安全的交易模拟功能，允许用户在不产生真实交易的情况下测试策略和系统功能。所有交易订单都将在本地模拟执行，提供完整的交易体验而无需承担实际的市场风险。

## 主要特性

### 🔒 安全模拟
- 所有交易订单在本地模拟执行
- 不会产生真实的交易费用
- 不会影响实际的资金和持仓
- 完全隔离的测试环境

### 📊 完整功能
- 模拟订单撮合和成交
- 实时 PnL 计算
- 持仓管理和风险控制
- 完整的交易历史记录

### 🎯 易于使用
- 一键开关设计
- 实时状态指示器
- 清晰的模式提醒
- 无缝切换体验

## 使用方法

### 1. 启用 Dry-Run 模式

#### 通过前端界面：
1. 访问 **系统设置** 页面
2. 切换到 **高级设置** 标签
3. 在 **交易设置** 部分找到 **Dry-Run 模式** 开关
4. 启用开关并保存设置

#### 通过 API：
```bash
curl -X PUT http://localhost:8080/api/v1/settings \
  -H "Content-Type: application/json" \
  -d '{
    "trading": {
      "dryRunMode": true,
      "riskControl": true,
      "maxPositionRatio": 50,
      "defaultStopLoss": 5
    },
    "system": {
      "logLevel": "INFO",
      "cacheSize": "1GB",
      "debugMode": false
    }
  }'
```

### 2. 模式指示器

启用 Dry-Run 模式后，系统会显示：

- **右上角浮动指示器**：显示当前处于 Dry-Run 模式
- **页面横幅提醒**：在交易相关页面显示警告横幅
- **设置页面警告**：在设置页面显示当前模式状态

### 3. 功能验证

在 Dry-Run 模式下，您可以：

- ✅ 创建和管理策略
- ✅ 执行交易订单（模拟）
- ✅ 查看持仓和 PnL（模拟）
- ✅ 测试风险控制机制
- ✅ 验证自动化功能
- ❌ 产生真实交易
- ❌ 影响实际资金

## 技术实现

### 后端组件

#### 1. 设置管理器 (`internal/api/settings_handler.go`)
```go
type TradingSettings struct {
    DryRunMode        bool `json:"dryRunMode"`
    RiskControl       bool `json:"riskControl"`
    MaxPositionRatio  int  `json:"maxPositionRatio"`
    DefaultStopLoss   int  `json:"defaultStopLoss"`
}
```

#### 2. 交易模拟器 (`internal/trading/dryrun/simulator.go`)
```go
type TradingSimulator struct {
    config *SimulatorConfig
    // ... 模拟交易逻辑
}
```

#### 3. PnL 执行器集成 (`internal/exchange/pnl/executor.go`)
```go
func (e *Executor) SetDryRun(dryRun bool) {
    e.dryRun = dryRun
    log.Printf("PnL Executor dry run mode: %v", dryRun)
}
```

### 前端组件

#### 1. 设置管理 Hook (`frontend/hooks/useSettings.ts`)
```typescript
export function useSettings() {
  const updateTradingSettings = async (tradingSettings: Partial<TradingSettings>) => {
    return saveSettings({
      trading: {
        ...settings.trading,
        ...tradingSettings
      }
    })
  }
}
```

#### 2. 状态指示器 (`frontend/components/ui/dry-run-indicator.tsx`)
```typescript
export function DryRunIndicator() {
  const { settings } = useSettings()
  
  if (!settings.trading.dryRunMode) return null
  
  return <Alert>Dry-Run 模式激活</Alert>
}
```

## API 接口

### 获取设置
```http
GET /api/v1/settings
```

### 更新设置
```http
PUT /api/v1/settings
Content-Type: application/json

{
  "trading": {
    "dryRunMode": true,
    "riskControl": true,
    "maxPositionRatio": 50,
    "defaultStopLoss": 5
  },
  "system": {
    "logLevel": "INFO",
    "cacheSize": "1GB",
    "debugMode": false
  }
}
```

## 测试验证

运行测试脚本验证功能：

```bash
go run test_dry_run_settings.go
```

测试内容包括：
- 获取当前设置
- 启用 Dry-Run 模式
- 验证设置更新
- 关闭 Dry-Run 模式
- 测试系统设置

## 注意事项

### ⚠️ 重要提醒

1. **数据隔离**：Dry-Run 模式下的数据与真实交易数据完全隔离
2. **性能影响**：模拟交易可能会消耗一定的系统资源
3. **市场数据**：使用真实市场数据进行模拟，但执行结果为模拟
4. **切换影响**：模式切换会立即生效，请谨慎操作

### 🔧 故障排除

1. **设置不生效**：检查后端 API 连接状态
2. **指示器不显示**：确认前端组件正确加载
3. **模拟数据异常**：检查交易模拟器配置
4. **API 错误**：查看服务器日志获取详细信息

## 未来扩展

- [ ] 支持多种模拟场景配置
- [ ] 添加模拟市场数据回放功能
- [ ] 集成更详细的性能分析
- [ ] 支持模拟数据导出和分析
- [ ] 添加 A/B 测试功能

## 相关文件

### 后端文件
- `internal/api/settings_handler.go` - 设置 API 处理器
- `internal/trading/dryrun/simulator.go` - 交易模拟器
- `internal/exchange/pnl/executor.go` - PnL 执行器
- `examples/dryrun_trading_demo.go` - 演示示例

### 前端文件
- `frontend/hooks/useSettings.ts` - 设置管理 Hook
- `frontend/app/api/settings/route.ts` - 前端 API 路由
- `frontend/components/ui/dry-run-indicator.tsx` - 状态指示器
- `frontend/app/settings/page.tsx` - 设置页面

### 测试文件
- `test_dry_run_settings.go` - 功能测试脚本

---

**版本**: v1.0.0  
**更新时间**: 2025-01-27  
**维护者**: QCAT 开发团队
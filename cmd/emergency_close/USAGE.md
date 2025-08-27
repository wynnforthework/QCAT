# 紧急平仓程序使用指南

## 问题背景

由于之前策略和风控的错误，导致持仓接近10万，需要紧急平掉所有仓位。

## 解决方案

已创建紧急平仓程序 `emergency_close.exe`，该程序会：

1. **取消所有挂单** - 使用 `DELETE /fapi/v1/allOpenOrders` 接口
2. **平掉所有仓位** - 使用市价单平掉所有持仓

## 使用步骤

### 1. 准备环境变量

在项目根目录创建或编辑 `.env` 文件：

```bash
# Binance API 配置
EXCHANGE_API_KEY=your_binance_api_key
EXCHANGE_API_SECRET=your_binance_api_secret
```

### 2. 确认配置文件

确保 `configs/config.yaml` 存在并配置正确：

```yaml
exchange:
  name: "binance"
  api_key: "${EXCHANGE_API_KEY}"
  api_secret: "${EXCHANGE_API_SECRET}"
  test_net: false  # 生产环境设为 false
  base_url: "https://fapi.binance.com"
```

### 3. 运行程序

```bash
cd cmd/emergency_close
./emergency_close.exe
```

### 4. 确认操作

程序会显示警告并要求确认：

```
WARNING: This will close ALL positions and cancel ALL orders. Are you sure? (yes/no): 
```

输入 `yes` 继续执行。

## 执行过程

程序会按以下顺序执行：

1. **加载配置** - 从配置文件和环境变量加载API密钥
2. **验证凭据** - 确保API密钥已配置
3. **用户确认** - 要求用户确认操作
4. **取消挂单** - 获取所有有挂单的交易对，逐个取消
5. **获取持仓** - 获取所有当前持仓
6. **执行平仓** - 对每个持仓创建市价平仓单

## 安全特性

- ✅ **用户确认** - 必须输入 `yes` 才能执行
- ✅ **只平仓模式** - 所有订单设置 `reduceOnly=true`
- ✅ **详细日志** - 记录每个操作的详细信息
- ✅ **错误处理** - 部分失败不影响其他操作
- ✅ **超时保护** - 5分钟超时防止卡死

## 预期输出

```
2024/01/01 12:00:00 Emergency Position Closer - Starting...
WARNING: This will close ALL positions and cancel ALL orders. Are you sure? (yes/no): yes
2024/01/01 12:00:01 === EMERGENCY POSITION CLOSURE STARTED ===
2024/01/01 12:00:01 Step 1: Canceling all open orders...
2024/01/01 12:00:01 Found open orders for 5 symbols
2024/01/01 12:00:01 Canceling all orders for BTCUSDT...
2024/01/01 12:00:02 Successfully canceled all orders for BTCUSDT
...
2024/01/01 12:00:05 Step 2: Getting current positions...
2024/01/01 12:00:06 Found 8 positions to close
2024/01/01 12:00:06 Step 3: Closing all positions...
2024/01/01 12:00:06 Closing position: ETHUSDT, Size: 15.500000, Side: LONG
2024/01/01 12:00:07 Close order placed: OrderID=12345, Symbol=ETHUSDT, Side=SELL, Quantity=15.50000000
2024/01/01 12:00:07 Successfully closed position for ETHUSDT
...
2024/01/01 12:00:15 === EMERGENCY CLOSURE COMPLETED ===
2024/01/01 12:00:15 Positions closed: 8/8
2024/01/01 12:00:15 Emergency closure completed successfully!
```

## 故障排除

### 1. API权限错误
```
Failed to get positions: API-key format invalid
```
**解决方案**: 检查API密钥是否正确配置

### 2. 网络连接错误
```
Failed to cancel orders for BTCUSDT: context deadline exceeded
```
**解决方案**: 检查网络连接，程序会继续执行其他操作

### 3. 余额不足错误
```
Order rejected: Insufficient balance
```
**解决方案**: 这通常不会发生，因为是平仓操作

## 注意事项

⚠️ **重要警告**:
- 这个程序会关闭**所有**持仓
- 确保在正确的账户上运行
- 建议先在测试网络测试
- 执行前请备份重要数据

## 仓位管理问题修复

同时已修复了导致仓位超标的根本问题：

### 问题原因
在 `internal/exchange/portfolio/manager.go` 中，仓位计算逻辑存在问题：
- 每次都基于总权益计算目标仓位
- 没有考虑当前已有持仓
- 导致重复累加仓位

### 修复方案
- ✅ 获取当前持仓大小
- ✅ 计算目标仓位与当前仓位的差异
- ✅ 只有差异超过阈值才调整仓位
- ✅ 避免频繁小幅调整

这样可以防止未来再次出现仓位超标的问题。

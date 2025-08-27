# Emergency Position Closer

这是一个紧急平仓工具，用于在紧急情况下快速关闭所有持仓并取消所有挂单。

## 功能

1. **取消所有挂单**: 使用 `DELETE /fapi/v1/allOpenOrders` 接口取消所有交易对的挂单
2. **平掉所有仓位**: 使用市价单平掉所有当前持仓
3. **安全确认**: 执行前需要用户确认
4. **详细日志**: 提供详细的执行日志和错误报告

## 使用方法

### 1. 设置环境变量

```bash
export BINANCE_API_KEY="your_api_key"
export BINANCE_SECRET_KEY="your_secret_key"
```

### 2. 运行程序

```bash
cd cmd/emergency_close
go run main.go
```

### 3. 确认操作

程序会提示确认，输入 `yes` 继续执行：

```
WARNING: This will close ALL positions and cancel ALL orders. Are you sure? (yes/no): yes
```

## 执行流程

1. **加载配置**: 从环境变量加载API密钥
2. **初始化客户端**: 创建Binance API客户端
3. **取消挂单**: 
   - 获取所有有挂单的交易对
   - 逐个调用 `DELETE /fapi/v1/allOpenOrders` 取消挂单
4. **获取持仓**: 调用 `GET /fapi/v2/positionRisk` 获取所有持仓
5. **平仓操作**:
   - 对每个非零持仓创建市价平仓单
   - 多头仓位使用SELL单平仓
   - 空头仓位使用BUY单平仓
   - 设置 `reduceOnly=true` 确保只平仓不开新仓

## 安全特性

- **用户确认**: 执行前必须输入 `yes` 确认
- **只平仓模式**: 所有订单都设置 `reduceOnly=true`
- **错误处理**: 即使部分操作失败也会继续执行其他操作
- **详细日志**: 记录每个操作的结果
- **超时保护**: 设置5分钟超时防止程序卡死

## 注意事项

⚠️ **警告**: 这个工具会关闭所有持仓，请谨慎使用！

- 确保API密钥有足够的权限（交易权限）
- 在测试网络上先测试
- 执行前确认当前持仓情况
- 网络问题可能导致部分操作失败，请检查执行结果

## 错误处理

程序会尽力执行所有操作，即使遇到错误也会继续：

- 取消挂单失败不会阻止平仓操作
- 单个仓位平仓失败不会影响其他仓位
- 最终会报告成功和失败的操作数量

## 日志示例

```
2024/01/01 12:00:00 Emergency Position Closer - Starting...
2024/01/01 12:00:01 === EMERGENCY POSITION CLOSURE STARTED ===
2024/01/01 12:00:01 Step 1: Canceling all open orders...
2024/01/01 12:00:01 Found open orders for 3 symbols
2024/01/01 12:00:01 Canceling all orders for BTCUSDT...
2024/01/01 12:00:02 Successfully canceled all orders for BTCUSDT
2024/01/01 12:00:02 Step 2: Getting current positions...
2024/01/01 12:00:03 Found 2 positions to close
2024/01/01 12:00:03 Step 3: Closing all positions...
2024/01/01 12:00:03 Closing position: ETHUSDT, Size: 1.500000, Side: LONG
2024/01/01 12:00:04 Close order placed: OrderID=12345, Symbol=ETHUSDT, Side=SELL, Quantity=1.50000000
2024/01/01 12:00:04 Successfully closed position for ETHUSDT
2024/01/01 12:00:05 === EMERGENCY CLOSURE COMPLETED ===
2024/01/01 12:00:05 Positions closed: 2/2
2024/01/01 12:00:05 Emergency closure completed successfully!
```

# 历史数据回填功能

本文档介绍如何使用QCAT项目中的历史数据回填功能，从Binance API获取历史K线数据并存储到数据库中。

## 功能特性

### 🔄 智能数据回填
- **自动检测数据缺失**: 检查数据库中的数据完整性
- **增量回填**: 只获取缺失的数据，避免重复
- **批量处理**: 支持大时间范围的数据回填
- **API限制处理**: 自动处理Binance API的速率限制

### 📊 数据完整性检查
- **数据统计**: 统计期望数据点vs实际数据点
- **间隙检测**: 识别数据中的时间间隙
- **完整度评估**: 计算数据完整度百分比

### 🛠️ 灵活配置
- **多交易对支持**: 同时处理多个交易对
- **多时间间隔**: 支持1m, 5m, 15m, 1h, 1d等间隔
- **自定义时间范围**: 指定开始和结束日期

## 使用方法

### 1. 命令行工具

#### 基本用法
```bash
# 回填BTCUSDT最近30天的1小时K线数据
go run cmd/backfill/main.go -symbol=BTCUSDT -interval=1h -days=30

# 回填指定日期范围的数据
go run cmd/backfill/main.go -symbol=ETHUSDT -interval=15m -start=2024-01-01 -end=2024-01-31

# 同时回填多个交易对
go run cmd/backfill/main.go -symbols="BTCUSDT,ETHUSDT,ADAUSDT" -interval=1h -days=7

# 只检查数据完整性，不回填
go run cmd/backfill/main.go -symbol=BTCUSDT -interval=1h -days=30 -check
```

#### 参数说明
- `-config`: 配置文件路径 (默认: configs/config.yaml)
- `-symbol`: 单个交易对符号 (默认: BTCUSDT)
- `-symbols`: 多个交易对，用逗号分隔
- `-interval`: K线间隔 (1m, 5m, 15m, 30m, 1h, 4h, 1d等)
- `-start`: 开始日期 (YYYY-MM-DD格式)
- `-end`: 结束日期 (YYYY-MM-DD格式)
- `-days`: 回填天数，从今天往前计算 (默认: 30)
- `-check`: 只检查数据完整性，不执行回填

### 2. 编程接口

#### 基本使用
```go
package main

import (
    "context"
    "log"
    "time"
    
    "qcat/internal/database"
    "qcat/internal/exchange/binance"
    "qcat/internal/market/kline"
)

func main() {
    // 创建数据库连接
    db, err := database.NewConnection(dbConfig)
    if err != nil {
        log.Fatal(err)
    }
    
    // 创建Binance客户端
    binanceClient := binance.NewClient(exchangeConfig, nil)
    
    // 创建K线管理器
    manager := kline.NewManagerWithBinance(db.DB, binanceClient)
    
    ctx := context.Background()
    symbol := "BTCUSDT"
    interval := kline.Interval1h
    startTime := time.Now().AddDate(0, 0, -30) // 30天前
    endTime := time.Now()
    
    // 执行历史数据回填
    err = manager.BackfillHistoricalData(ctx, symbol, interval, startTime, endTime)
    if err != nil {
        log.Printf("Backfill failed: %v", err)
    }
}
```

#### 智能获取历史数据
```go
// 获取历史数据，如果数据库中没有则自动回填
klines, err := manager.GetHistoryWithBackfill(ctx, "BTCUSDT", kline.Interval1h, startTime, endTime)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("获取到 %d 条K线数据\n", len(klines))
```

#### 数据完整性检查
```go
// 检查数据完整性
report, err := manager.CheckDataIntegrity(ctx, "BTCUSDT", kline.Interval1h, startTime, endTime)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("数据完整度: %.2f%%\n", report.Completeness)
fmt.Printf("数据间隙: %d个\n", len(report.Gaps))

if report.HasGaps {
    for _, gap := range report.Gaps {
        fmt.Printf("间隙: %s 到 %s\n", 
            gap.Start.Format("2006-01-02 15:04"), 
            gap.End.Format("2006-01-02 15:04"))
    }
}
```

### 3. 在MarketAnalyzer中使用

现在MarketAnalyzer已经集成了历史数据回填功能：

```go
// 创建带有历史数据回填功能的MarketAnalyzer
db, _ := database.NewConnection(dbConfig)
binanceClient := binance.NewClient(exchangeConfig, nil)
klineManager := kline.NewManagerWithBinance(db.DB, binanceClient)

analyzer := generator.NewMarketAnalyzer(db, binanceClient, klineManager)

// 分析市场数据 - 如果数据库中没有足够的历史数据，会自动从API获取
analysis, err := analyzer.AnalyzeMarket(ctx, "BTCUSDT", 30*24*time.Hour)
```

## 配置要求

### 数据库配置
确保PostgreSQL数据库中有market_data表：

```sql
CREATE TABLE market_data (
    symbol VARCHAR(20) NOT NULL,
    interval VARCHAR(10) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    open DECIMAL(20,8) NOT NULL,
    high DECIMAL(20,8) NOT NULL,
    low DECIMAL(20,8) NOT NULL,
    close DECIMAL(20,8) NOT NULL,
    volume DECIMAL(20,8) NOT NULL,
    complete BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (symbol, timestamp, interval)
);

CREATE INDEX idx_market_data_symbol_interval_timestamp 
ON market_data (symbol, interval, timestamp);
```

### Binance API配置
在config.yaml中配置Binance API：

```yaml
exchange:
  name: "binance"
  api_key: "${EXCHANGE_API_KEY}"
  api_secret: "${EXCHANGE_API_SECRET}"
  test_net: true  # 测试网络
  base_url: "https://api.binance.com"
  rate_limit:
    enabled: true
    requests_per_minute: 1200
    burst: 100
```

## 性能优化

### 批量处理
- 每次API调用最多获取1000条K线数据
- 自动分批处理大时间范围的请求
- 使用事务批量插入数据库

### API限制处理
- 自动添加请求间隔（100ms）
- 支持速率限制器
- 错误重试机制

### 数据去重
- 使用数据库UPSERT操作避免重复数据
- 内存中缓存已存在的时间戳

## 监控和日志

### 日志输出
```
2024-01-15 10:30:00 Starting historical data backfill for BTCUSDT 1h from 2023-12-16 to 2024-01-15
2024-01-15 10:30:01 Saved 24 new klines for BTCUSDT 1h (batch: 2023-12-16 00:00 to 2023-12-17 00:00)
2024-01-15 10:30:02 Saved 24 new klines for BTCUSDT 1h (batch: 2023-12-17 00:00 to 2023-12-18 00:00)
...
2024-01-15 10:35:00 Historical data backfill completed for BTCUSDT 1h: fetched 720, saved 720 new records
```

### 数据完整性报告
```
数据完整性报告 - BTCUSDT 1h:
  期望数据点: 720
  实际数据点: 718
  完整度: 99.72%
  数据间隙: 1个
  间隙详情:
    2024-01-10 15:00 到 2024-01-10 17:00
```

## 常见问题

### Q: 如何处理API限制？
A: 工具自动处理API限制，包括请求间隔和重试机制。如果遇到限制，会自动等待并重试。

### Q: 数据回填会覆盖现有数据吗？
A: 不会。工具使用UPSERT操作，只会更新已存在的数据或插入新数据。

### Q: 支持哪些时间间隔？
A: 支持Binance API的所有间隔：1m, 3m, 5m, 15m, 30m, 1h, 2h, 4h, 6h, 8h, 12h, 1d, 3d, 1w, 1M

### Q: 如何处理网络错误？
A: 工具会记录错误并继续处理下一批数据，确保部分失败不会影响整个回填过程。

## 最佳实践

1. **分批回填**: 对于大时间范围，建议分批进行回填
2. **定期检查**: 定期运行数据完整性检查
3. **监控日志**: 关注回填过程中的错误和警告
4. **测试环境**: 先在测试环境验证配置和功能
5. **备份数据**: 在大规模回填前备份数据库

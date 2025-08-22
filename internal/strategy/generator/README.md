# MarketAnalyzer - 市场分析器

MarketAnalyzer 是一个强大的市场数据分析工具，能够从真实数据源获取历史价格数据并计算各种技术指标和市场特征。

## 功能特性

### 数据获取
- **真实数据源**: 优先从数据库和交易所API获取真实历史数据
- **多层次回退**: 数据库 → 交易所API → 模拟数据
- **多时间框架**: 支持15分钟、1小时、1天等多种时间间隔

### 市场分析指标
- **波动率计算**: 基于历史收益率的年化波动率
- **趋势分析**: 使用线性回归计算趋势强度（-1到1）
- **夏普比率**: 风险调整后的收益指标
- **最大回撤**: 历史最大损失幅度
- **市场周期**: 基于峰谷检测的周期估算
- **流动性指标**: 成交量与价格波动的比率
- **市场状态**: trending/ranging/volatile 三种状态
- **分析置信度**: 基于数据质量的置信度评估

### 技术指标
- RSI (相对强弱指数)
- MACD (移动平均收敛散度)
- 布林带 (上轨、中轨、下轨、宽度)
- 移动平均线 (SMA20, SMA50, EMA12, EMA26)
- ATR (平均真实波幅)
- 成交量指标

### 相关性分析
- 计算与其他资产的价格相关性
- 支持多种加密货币对的相关性矩阵

## 使用方法

### 基本用法（使用模拟数据）
```go
// 创建不带依赖项的分析器（将使用模拟数据）
analyzer := &MarketAnalyzer{}

ctx := context.Background()
symbol := "BTCUSDT"
timeRange := 24 * time.Hour

// 执行市场分析
analysis, err := analyzer.AnalyzeMarket(ctx, symbol, timeRange)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("波动率: %.4f\n", analysis.Volatility)
fmt.Printf("趋势强度: %.4f\n", analysis.Trend)
fmt.Printf("市场状态: %s\n", analysis.MarketRegime)
```

### 高级用法（使用真实数据源）

**推荐方式：使用生产环境服务**
```go
// 创建生产环境服务（自动创建所有依赖项）
service, err := generator.CreateProductionService(dbConfig, exchangeConfig)
if err != nil {
    log.Fatal(err)
}

// 生成策略（使用真实数据）
result, err := service.GenerateStrategy(ctx, req)
```

**手动创建依赖项**
```go
// 创建带依赖项的分析器（将使用真实数据）
db := database.NewConnection(dbConfig)
binanceClient := binance.NewClient(exchangeConfig, rateLimiter)
klineManager := kline.NewManagerWithBinance(db.DB, binanceClient)

analyzer := NewMarketAnalyzer(db, binanceClient, klineManager)

// 执行分析
analysis, err := analyzer.AnalyzeMarket(ctx, "BTCUSDT", 7*24*time.Hour)
```

**自动批量生成策略**
```go
// 创建自动生成服务
autoService, err := generator.CreateProductionAutoService(dbConfig, exchangeConfig)
if err != nil {
    log.Fatal(err)
}

// 批量生成策略
symbols := []string{"BTCUSDT", "ETHUSDT", "ADAUSDT"}
results, err := autoService.AutoGenerateStrategies(ctx, symbols, 5)
```

### 策略表现分析
```go
performance, err := analyzer.AnalyzePerformance(ctx, "strategy_id", timeRange)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("总收益: %.2f%%\n", performance.TotalReturn*100)
fmt.Printf("夏普比率: %.4f\n", performance.SharpeRatio)
fmt.Printf("胜率: %.1f%%\n", performance.WinRate*100)
```

### 市场状态检测
```go
regime := analyzer.DetectMarketRegime(analysis)
fmt.Printf("市场状态: %s\n", regime) // "trending", "ranging", "volatile"

optimalTimeframe := analyzer.CalculateOptimalTimeframe(analysis)
fmt.Printf("建议时间框架: %v\n", optimalTimeframe)
```

## 数据源配置

### 数据库配置
MarketAnalyzer 支持从PostgreSQL数据库获取历史K线数据：
```sql
-- 市场数据表结构
CREATE TABLE market_data (
    symbol VARCHAR(20),
    interval VARCHAR(10),
    timestamp TIMESTAMP,
    open DECIMAL(20,8),
    high DECIMAL(20,8),
    low DECIMAL(20,8),
    close DECIMAL(20,8),
    volume DECIMAL(20,8),
    complete BOOLEAN,
    PRIMARY KEY (symbol, timestamp, interval)
);
```

### 交易所API配置
支持通过Binance API获取实时和历史数据：
```go
config := &exchange.ExchangeConfig{
    APIKey:    "your_api_key",
    APISecret: "your_api_secret",
    TestNet:   true,
}
```

## 输出结果

### MarketAnalysis 结构
```go
type MarketAnalysis struct {
    Symbol              string              // 交易对符号
    TimeRange           time.Duration       // 分析时间范围
    Volatility          float64             // 年化波动率
    Trend               float64             // 趋势强度 (-1到1)
    SharpeRatio         float64             // 夏普比率
    MaxDrawdown         float64             // 最大回撤 (0到1)
    MarketCycle         float64             // 市场周期(天)
    Liquidity           float64             // 流动性指标
    Correlation         map[string]float64  // 相关性矩阵
    TechnicalIndicators TechnicalIndicators // 技术指标
    MarketRegime        string              // 市场状态
    Confidence          float64             // 分析置信度
}
```

### PerformanceAnalysis 结构
```go
type PerformanceAnalysis struct {
    StrategyID   string        // 策略ID
    TimeRange    time.Duration // 分析时间范围
    TotalReturn  float64       // 总收益率
    SharpeRatio  float64       // 夏普比率
    MaxDrawdown  float64       // 最大回撤
    WinRate      float64       // 胜率
    ProfitFactor float64       // 盈利因子
    TotalTrades  int           // 总交易次数
    AvgTrade     float64       // 平均交易收益
    Volatility   float64       // 策略波动率
    Confidence   float64       // 分析置信度
}
```

## 示例

查看 `examples/market_analyzer_example.go` 获取完整的使用示例。

## 数据源说明

### 真实数据 vs 模拟数据

**真实数据源（推荐用于生产）：**
- 从PostgreSQL数据库获取历史K线数据
- 从Binance API实时获取市场数据
- 从数据库获取策略历史表现数据
- 自动数据回填和质量验证

**模拟数据（仅用于开发测试）：**
- 使用数学函数生成的模拟价格数据
- 硬编码的策略表现指标
- 不反映真实市场条件

### 如何确保使用真实数据

1. **使用推荐的构造函数：**
   ```go
   // ✅ 推荐：使用真实数据源
   service, err := generator.CreateProductionService(dbConfig, exchangeConfig)

   // ❌ 不推荐：可能使用模拟数据
   service := generator.NewService()
   ```

2. **检查日志输出：**
   - 真实数据：`"Successfully retrieved X klines from database"`
   - 模拟数据：`"Warning: using mock data"`

3. **验证数据质量：**
   ```go
   // 检查分析结果的置信度
   if analysis.Confidence < 0.5 {
       log.Printf("Warning: Low confidence analysis, may be using insufficient data")
   }
   ```

### 数据源优先级

系统按以下优先级获取数据：
1. **数据库（KlineManager）** - 最快，支持自动回填
2. **直接数据库查询** - 备用方案
3. **Binance API** - 实时数据获取
4. **错误返回** - 不再使用模拟数据

## 注意事项

1. **数据质量**: 分析结果的准确性依赖于输入数据的质量
2. **时间范围**: 建议使用至少24小时的数据进行分析
3. **市场条件**: 不同市场条件下的指标含义可能不同
4. **数据源**: 生产环境请务必使用真实数据源，避免使用模拟数据
5. **风险提示**: 所有分析结果仅供参考，不构成投资建议

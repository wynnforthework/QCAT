# QCAT 策略结果分享系统

## 系统概述

QCAT 策略结果分享系统是一个专为量化交易策略开发者设计的平台，允许用户分享、发现和评估交易策略结果。系统提供了完整的性能指标展示、可复现性保证和智能筛选功能。

## 核心功能

### 1. 策略结果分享
- **完整的性能指标**：总收益率、年化收益率、最大回撤、夏普比率、胜率等
- **可复现性数据**：随机种子、数据哈希、代码版本、运行环境等
- **策略支持信息**：支持的交易品种、时间框架、资金要求等
- **回测信息**：回测时间范围、市场环境、手续费设置等
- **风险评估**：VaR、Beta、Alpha、信息比率等高级指标
- **市场适应性**：不同市场环境下的表现分析

### 2. 智能筛选和评估
- **多维度筛选**：按收益率、回撤、夏普比率等指标筛选
- **标签系统**：策略类型、风险等级、市场环境等标签
- **评分算法**：基于多个指标的加权评分系统
- **排序功能**：按评分、时间、收益率等排序

### 3. 社区功能
- **用户贡献**：分享者信息、贡献统计
- **评分评论**：用户评分和评论系统
- **标签分类**：策略分类和标签管理

## 技术架构

### 后端 (Go)
- **主服务**：`cmd/optimizer/main.go` - 优化器服务，集成结果分享功能
- **结果分享管理**：`internal/learning/automl/result_sharing.go` - 核心分享逻辑
- **配置管理**：`configs/result_sharing.yaml` - 系统配置

### 前端 (Next.js + TypeScript)
- **分享页面**：`frontend/app/share-result/page.tsx` - 策略结果分享表单
- **结果展示**：`frontend/app/shared-results/page.tsx` - 共享结果列表和详情
- **API路由**：`frontend/app/api/share-result/route.ts` - 分享API
- **导航组件**：`frontend/components/navigation.tsx` - 系统导航

## 快速开始

### 1. 启动后端服务

```bash
# 进入项目目录
cd QCAT

# 启动优化器服务（包含结果分享功能）
go run cmd/optimizer/main.go
```

服务将在 `http://localhost:8080` 启动，提供以下API端点：
- `POST /share-result` - 分享策略结果
- `GET /shared-results` - 获取共享结果列表

### 2. 启动前端服务

```bash
# 进入前端目录
cd frontend

# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

前端将在 `http://localhost:3000` 启动，提供以下页面：
- `/` - 系统首页
- `/share-result` - 分享策略结果
- `/shared-results` - 浏览共享结果

### 3. 配置系统

编辑 `configs/result_sharing.yaml` 文件来配置系统参数：

```yaml
result_sharing:
  enabled: true
  file_storage:
    directory: "./data/shared_results"
    file_extension: ".json"
    max_file_size: 10485760 # 10MB
    retention_days: 365
  performance_threshold:
    min_total_return: 5.0
    min_sharpe_ratio: 0.5
    max_drawdown: 20.0
    min_win_rate: 0.4
    min_profit_factor: 1.2
  scoring_weights:
    total_return: 0.25
    sharpe_ratio: 0.20
    max_drawdown: 0.15
    win_rate: 0.10
    profit_factor: 0.10
    live_performance: 0.15
    risk_assessment: 0.05
```

## 使用指南

### 分享策略结果

1. **访问分享页面**
   - 打开浏览器访问 `http://localhost:3000/share-result`

2. **填写基本信息**
   - 任务ID：唯一标识符
   - 策略名称：策略的显示名称
   - 版本号：策略版本
   - 分享者：您的姓名或ID

3. **输入性能指标**
   - 总收益率：策略的总收益百分比
   - 最大回撤：最大回撤百分比
   - 夏普比率：风险调整后收益
   - 胜率：盈利交易的比例
   - 总交易次数：完成的交易数量

4. **提供可复现性数据**
   - 随机种子：用于重现结果的种子值
   - 数据哈希：数据集的哈希值
   - 代码版本：策略代码的版本
   - 运行环境：Python版本、依赖库等
   - 数据时间范围：回测数据的时间范围
   - 数据源：数据来源列表

5. **描述策略支持**
   - 支持的交易品种：BTC、ETH等
   - 支持的时间框架：1m、5m、1h、1d等
   - 资金要求：最小和最大资金要求
   - 杠杆支持：是否支持杠杆交易

6. **填写回测信息**
   - 回测时间范围：开始和结束时间
   - 初始和最终资金
   - 手续费和滑点设置
   - 市场环境描述

7. **设置分享信息**
   - 分享描述：策略特点和优势
   - 标签：策略类型、风险等级等标签

8. **提交分享**
   - 点击"分享结果"按钮
   - 系统会验证数据并保存结果

### 浏览共享结果

1. **访问结果页面**
   - 打开浏览器访问 `http://localhost:3000/shared-results`

2. **筛选结果**
   - 使用搜索框搜索策略名称或描述
   - 设置最小收益率、最大回撤等筛选条件
   - 按标签筛选特定类型的策略

3. **查看详情**
   - 点击结果卡片查看详细信息
   - 查看完整的性能指标和可复现性数据
   - 评估策略的风险和适应性

4. **导入结果**
   - 对于满意的结果，可以导入到本地系统
   - 使用提供的随机种子和参数重现结果

## 数据格式

### 共享结果数据结构

```typescript
interface SharedResultV2 {
  id: string;
  task_id: string;
  strategy_name: string;
  version: string;
  created_at: string;
  shared_by: string;
  
  // 策略参数
  parameters: Record<string, any>;
  
  // 性能指标
  performance: {
    total_return: number;
    annual_return: number;
    max_drawdown: number;
    sharpe_ratio: number;
    win_rate: number;
    total_trades: number;
    // ... 更多指标
  };
  
  // 可复现性数据
  reproducibility: {
    random_seed: number;
    data_hash: string;
    code_version: string;
    environment: string;
    data_range: string;
    data_sources: string[];
    // ... 更多信息
  };
  
  // 策略支持信息
  strategy_support: {
    supported_markets: string[];
    supported_timeframes: string[];
    min_capital: number;
    max_capital: number;
    // ... 更多信息
  };
  
  // 回测信息
  backtest_info: {
    start_date: string;
    end_date: string;
    initial_capital: number;
    final_capital: number;
    // ... 更多信息
  };
  
  // 风险评估
  risk_assessment: {
    var_95: number;
    beta: number;
    alpha: number;
    // ... 更多指标
  };
  
  // 市场适应性
  market_adaptation: {
    bull_market_return: number;
    bear_market_return: number;
    // ... 更多指标
  };
  
  // 分享信息
  share_info: {
    share_method: string;
    share_platform: string;
    share_description: string;
    tags: string[];
    rating: number;
  };
}
```

## API 文档

### POST /share-result

分享新的策略结果。

**请求体**：
```json
{
  "task_id": "task_001",
  "strategy_name": "MA交叉策略",
  "version": "1.0.0",
  "shared_by": "trader001",
  "performance": {
    "total_return": 25.6,
    "max_drawdown": 8.2,
    "sharpe_ratio": 1.8
  },
  "reproducibility": {
    "random_seed": 12345,
    "data_hash": "abc123..."
  }
  // ... 其他字段
}
```

**响应**：
```json
{
  "success": true,
  "message": "策略结果分享成功",
  "data": {
    "id": "result_1234567890",
    "score": 85.6
  }
}
```

### GET /shared-results

获取共享结果列表。

**查询参数**：
- `query`: 搜索关键词
- `limit`: 返回结果数量限制
- `offset`: 偏移量
- `min_total_return`: 最小总收益率
- `max_drawdown`: 最大回撤
- `min_sharpe_ratio`: 最小夏普比率
- `strategy_name`: 策略名称

**响应**：
```json
{
  "success": true,
  "data": [
    {
      "id": "result_1234567890",
      "strategy_name": "MA交叉策略",
      "performance": {
        "total_return": 25.6,
        "max_drawdown": 8.2,
        "sharpe_ratio": 1.8
      },
      "score": 85.6
    }
  ],
  "total": 100,
  "limit": 20,
  "offset": 0
}
```

## 最佳实践

### 分享策略结果

1. **提供完整数据**
   - 确保所有性能指标都准确填写
   - 提供详细的可复现性信息
   - 描述策略的适用范围和限制

2. **使用合适的标签**
   - 选择准确的策略类型标签
   - 标注风险等级和市场环境
   - 使用描述性的标签帮助他人理解

3. **编写清晰的描述**
   - 说明策略的核心逻辑
   - 描述策略的优势和特点
   - 提供使用建议和注意事项

### 评估共享结果

1. **全面分析性能指标**
   - 不仅看总收益率，还要关注回撤和风险指标
   - 评估夏普比率和胜率
   - 考虑交易频率和成本

2. **验证可复现性**
   - 检查是否有完整的可复现性数据
   - 确认数据源和代码版本
   - 评估环境依赖

3. **评估适用性**
   - 检查策略是否适合您的交易品种
   - 确认资金要求是否匹配
   - 评估市场环境适应性

## 故障排除

### 常见问题

1. **分享失败**
   - 检查必需字段是否填写完整
   - 确认性能指标数据格式正确
   - 验证后端服务是否正常运行

2. **无法获取共享结果**
   - 检查网络连接
   - 确认API端点配置正确
   - 查看后端服务日志

3. **前端页面显示异常**
   - 检查浏览器控制台错误
   - 确认前端依赖安装完整
   - 验证API响应格式

### 日志查看

后端日志位置：
```bash
# 查看优化器服务日志
tail -f logs/optimizer.log

# 查看结果分享相关日志
grep "result_sharing" logs/optimizer.log
```

## 贡献指南

欢迎贡献代码和改进建议！

1. **代码贡献**
   - Fork 项目仓库
   - 创建功能分支
   - 提交 Pull Request

2. **问题报告**
   - 使用 GitHub Issues 报告问题
   - 提供详细的错误信息和复现步骤

3. **功能建议**
   - 在 Issues 中提出新功能建议
   - 描述功能需求和预期效果

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

## 联系方式

如有问题或建议，请通过以下方式联系：

- GitHub Issues: [项目仓库](https://github.com/your-repo/qcat)
- 邮箱: your-email@example.com

---

感谢使用 QCAT 策略结果分享系统！

# 前端测试指南

## 环境配置

1. 确保后端服务运行在 `http://localhost:8082`
2. 在 `frontend/.env.local` 文件中配置：
```
NEXT_PUBLIC_API_URL=http://localhost:8082
NEXT_PUBLIC_WS_URL=ws://localhost:8082
```

## 启动前端

```bash
cd frontend
npm install
npm run dev
```

## 测试功能

### 1. 仪表板页面 (/)
- 显示系统概览数据
- 账户资产、策略状态、风险指标
- 实时监控组件

### 2. 策略管理页面 (/strategies)
- 策略列表和状态
- 启动/停止策略
- 参数设置和历史记录
- 回测功能

### 3. 投资组合页面 (/portfolio)
- 资产分配概览
- 策略权重管理
- 再平衡功能
- 风险分析

### 4. 风险控制页面 (/risk)
- 风险指标监控
- 持仓风险分析
- 风险告警管理
- 压力测试

### 5. 热门币种页面 (/hotlist)
- 热门币种推荐
- 白名单管理
- 市场分析

### 6. 审计日志页面 (/audit)
- 操作日志记录
- 决策链追踪
- 合规检查

## API 接口测试

可以通过浏览器开发者工具的网络面板查看 API 调用情况：

- GET `/api/v1/dashboard` - 仪表板数据
- GET `/api/v1/strategy/` - 策略列表
- GET `/api/v1/portfolio/overview` - 投资组合概览
- GET `/api/v1/risk/overview` - 风险概览
- GET `/api/v1/hotlist/symbols` - 热门币种
- GET `/api/v1/audit/logs` - 审计日志

## 故障排除

1. **API 连接失败**
   - 检查后端服务是否运行
   - 确认 API URL 配置正确
   - 查看浏览器控制台错误信息

2. **页面显示模拟数据**
   - 正常现象，API 调用失败时会fallback到模拟数据
   - 确保后端API返回正确格式的数据

3. **WebSocket 连接问题**
   - 检查 WebSocket URL 配置
   - 确认后端支持 WebSocket 连接

## 开发建议

1. 使用浏览器开发者工具调试
2. 检查 Console 输出的错误信息
3. 验证 API 响应格式是否符合前端期望
4. 测试不同的用户操作场景

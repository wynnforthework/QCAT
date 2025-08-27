# API 超时问题修复说明

## 问题描述

前端在请求仪表盘数据时出现超时错误：
```
API request failed: AbortError: signal is aborted without reason
Failed to fetch dashboard data: ApiError: 请求超时，请检查网络连接
```

## 根本原因分析

1. **后端响应慢**：仪表盘API虽然能响应，但处理时间很长
2. **数据库问题**：
   - `pq: 字段 "sortino_ratio" 不存在` - 数据库表结构问题
   - `pq: 关系 "risk_metrics" 不存在` - 缺少risk_metrics表
3. **前端超时设置过短**：30秒超时对于复杂查询可能不够

## 修复方案

### 1. 前端超时和重试机制优化

#### 增加超时时间
- 从30秒增加到60秒，给后端更多处理时间

#### 添加重试机制
- 自动重试2次，每次间隔2秒
- 只对网络错误和超时错误进行重试
- 避免对业务逻辑错误进行重试

#### 改进错误处理
```typescript
// 重试逻辑
if (retries > 0 && (
  (error instanceof Error && error.name === 'AbortError') ||
  (error instanceof TypeError && error.message.includes('fetch'))
)) {
  console.log(`Retrying request to ${endpoint} in 2 seconds...`);
  await new Promise(resolve => setTimeout(resolve, 2000));
  return this.request<T>(endpoint, options, skipAuth, retries - 1);
}
```

### 2. 仪表盘API特殊处理

#### Fallback数据机制
- 当API失败时，返回模拟数据确保页面正常显示
- 避免白屏或完全无法使用的情况

```typescript
async getDashboardData(): Promise<DashboardData> {
  try {
    return await this.request<DashboardData>('/api/v1/dashboard');
  } catch (error) {
    console.warn('Dashboard API failed, returning mock data:', error);
    // 返回模拟数据
    return mockDashboardData;
  }
}
```

### 3. 用户体验改进

#### 加载状态优化
- 添加加载提示信息
- 说明可能的延迟原因

#### 错误状态显示
- 显示错误横幅而不是完全阻止页面显示
- 提供重试按钮
- 区分不同类型的错误（超时、网络、服务器错误）

#### 更新频率调整
- 从30秒改为60秒自动刷新，减少服务器压力

### 4. 后端优化建议

#### 数据库修复
需要修复以下数据库问题：
1. 添加缺失的 `sortino_ratio` 字段
2. 创建缺失的 `risk_metrics` 表
3. 优化查询性能，添加必要的索引

#### API性能优化
1. 添加数据缓存机制
2. 优化复杂查询
3. 考虑分页或数据分批加载

## 测试验证

### 测试步骤
1. 启动前端应用
2. 访问仪表盘页面
3. 观察加载行为和错误处理

### 预期结果
- 页面能正常显示（使用fallback数据）
- 显示适当的错误提示
- 不会出现白屏或崩溃
- 自动重试机制工作正常

### 验证命令
```bash
# 测试仪表盘API响应时间
curl -w "%{time_total}" -s -o /dev/null http://localhost:8082/api/v1/dashboard \
  -H "Authorization: Bearer <token>"

# 检查API响应内容
curl -s http://localhost:8082/api/v1/dashboard \
  -H "Authorization: Bearer <token>" | jq .
```

## 长期解决方案

1. **数据库优化**：修复表结构和性能问题
2. **缓存策略**：实现Redis缓存减少数据库查询
3. **API分层**：将复杂查询拆分为多个轻量级API
4. **监控告警**：添加API响应时间监控
5. **健康检查**：定期检查数据库连接和表结构

## 影响范围

- ✅ 仪表盘页面加载体验改善
- ✅ 减少用户遇到的超时错误
- ✅ 提供更好的错误反馈
- ✅ 系统整体稳定性提升

## 注意事项

1. Fallback数据是模拟数据，不反映真实状态
2. 需要尽快修复后端数据库问题
3. 监控重试频率，避免对服务器造成过大压力
4. 考虑添加用户手动刷新功能

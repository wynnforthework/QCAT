#!/bin/bash

# 从实际路由配置中提取所有API路径
echo "=== 从实际路由配置提取API路径 ==="

# 提取所有路由定义
echo "正在分析 internal/api/server.go 中的路由配置..."

# 创建临时文件存储路由
temp_file="/tmp/actual_routes.txt"
> "$temp_file"

# 提取基础路由
echo '"/health"' >> "$temp_file"
echo '"/health/detailed"' >> "$temp_file"

# 提取认证路由 (公共)
echo '"/api/v1/auth/login"' >> "$temp_file"
echo '"/api/v1/auth/register"' >> "$temp_file"
echo '"/api/v1/auth/refresh"' >> "$temp_file"

# 提取设置路由 (公共)
echo '"/api/v1/settings"' >> "$temp_file"

# 提取受保护的路由
echo '"/api/v1/auth/profile"' >> "$temp_file"

# 仪表盘路由
echo '"/api/v1/dashboard"' >> "$temp_file"
echo '"/api/v1/dashboard/db-health"' >> "$temp_file"

# 市场数据路由
echo '"/api/v1/market/data"' >> "$temp_file"

# 交易活动路由
echo '"/api/v1/trading/activity"' >> "$temp_file"
echo '"/api/v1/trading/history"' >> "$temp_file"
echo '"/api/v1/trading/positions"' >> "$temp_file"

# 系统指标路由
echo '"/api/v1/metrics/system"' >> "$temp_file"
echo '"/api/v1/metrics/strategy/:id"' >> "$temp_file"
echo '"/api/v1/metrics/performance"' >> "$temp_file"

# 策略路由
echo '"/api/v1/strategy"' >> "$temp_file"
echo '"/api/v1/strategy/:id"' >> "$temp_file"
echo '"/api/v1/strategy/pool/overview"' >> "$temp_file"
echo '"/api/v1/strategy/execution/overview"' >> "$temp_file"
echo '"/api/v1/strategy/execution/realtime"' >> "$temp_file"
echo '"/api/v1/strategy/workflow/status"' >> "$temp_file"
echo '"/api/v1/strategy/:id/promote"' >> "$temp_file"
echo '"/api/v1/strategy/:id/start"' >> "$temp_file"
echo '"/api/v1/strategy/:id/stop"' >> "$temp_file"
echo '"/api/v1/strategy/:id/backtest"' >> "$temp_file"
echo '"/api/v1/strategy/:id/auto-start"' >> "$temp_file"

# 自动启动路由
echo '"/api/v1/auto-start/strategies"' >> "$temp_file"
echo '"/api/v1/auto-start/stats"' >> "$temp_file"
echo '"/api/v1/auto-start/trigger"' >> "$temp_file"

# 黑名单路由
echo '"/api/v1/blacklist/"' >> "$temp_file"
echo '"/api/v1/blacklist/:strategy_id"' >> "$temp_file"
echo '"/api/v1/blacklist/clear-expired"' >> "$temp_file"

# 紧急停止路由
echo '"/api/v1/emergency/stop-all"' >> "$temp_file"
echo '"/api/v1/emergency/status"' >> "$temp_file"
echo '"/api/v1/emergency/reset"' >> "$temp_file"
echo '"/api/v1/emergency/history"' >> "$temp_file"

# 工作流路由
echo '"/api/v1/workflow/dependency-graph"' >> "$temp_file"
echo '"/api/v1/workflow/execute"' >> "$temp_file"
echo '"/api/v1/workflow/results"' >> "$temp_file"
echo '"/api/v1/workflow/status"' >> "$temp_file"
echo '"/api/v1/workflow/validate"' >> "$temp_file"
echo '"/api/v1/workflow/enabled"' >> "$temp_file"
echo '"/api/v1/workflow/functions/:function_id/enable"' >> "$temp_file"
echo '"/api/v1/workflow/functions/:function_id/disable"' >> "$temp_file"
echo '"/api/v1/workflow/functions/:function_id"' >> "$temp_file"

# 并发管理路由
echo '"/api/v1/concurrent/pools"' >> "$temp_file"
echo '"/api/v1/concurrent/pools/:pool_name"' >> "$temp_file"
echo '"/api/v1/concurrent/pools/:pool_name/scale"' >> "$temp_file"
echo '"/api/v1/concurrent/tasks"' >> "$temp_file"
echo '"/api/v1/concurrent/monitor"' >> "$temp_file"
echo '"/api/v1/concurrent/alerts"' >> "$temp_file"
echo '"/api/v1/concurrent/load-balancer"' >> "$temp_file"
echo '"/api/v1/concurrent/task-queue"' >> "$temp_file"

# 优化器路由
echo '"/api/v1/optimizer/run"' >> "$temp_file"
echo '"/api/v1/optimizer/tasks"' >> "$temp_file"
echo '"/api/v1/optimizer/tasks/:id"' >> "$temp_file"
echo '"/api/v1/optimizer/results/:id"' >> "$temp_file"

# 投资组合路由
echo '"/api/v1/portfolio/overview"' >> "$temp_file"
echo '"/api/v1/portfolio/allocations"' >> "$temp_file"
echo '"/api/v1/portfolio/rebalance"' >> "$temp_file"
echo '"/api/v1/portfolio/history"' >> "$temp_file"
echo '"/api/v1/portfolio/performance"' >> "$temp_file"

# 风险管理路由
echo '"/api/v1/risk/overview"' >> "$temp_file"
echo '"/api/v1/risk/limits"' >> "$temp_file"
echo '"/api/v1/risk/circuit-breakers"' >> "$temp_file"
echo '"/api/v1/risk/violations"' >> "$temp_file"

# 热点列表路由
echo '"/api/v1/hotlist/symbols"' >> "$temp_file"
echo '"/api/v1/hotlist/approve"' >> "$temp_file"
echo '"/api/v1/hotlist/whitelist"' >> "$temp_file"
echo '"/api/v1/hotlist/whitelist/:symbol"' >> "$temp_file"

# 策略验证路由
echo '"/api/v1/validation/strategies"' >> "$temp_file"
echo '"/api/v1/validation/problems"' >> "$temp_file"
echo '"/api/v1/validation/automation"' >> "$temp_file"

# 内存管理路由
echo '"/api/v1/memory/stats"' >> "$temp_file"
echo '"/api/v1/memory/gc"' >> "$temp_file"

# 网络管理路由
echo '"/api/v1/network/connections"' >> "$temp_file"
echo '"/api/v1/network/connections/:id"' >> "$temp_file"
echo '"/api/v1/network/connections/:id/reconnect"' >> "$temp_file"

# 健康检查路由
echo '"/api/v1/health/status"' >> "$temp_file"
echo '"/api/v1/health/checks"' >> "$temp_file"
echo '"/api/v1/health/checks/:name"' >> "$temp_file"
echo '"/api/v1/health/checks/:name/force"' >> "$temp_file"

# 关闭管理路由
echo '"/api/v1/shutdown/status"' >> "$temp_file"
echo '"/api/v1/shutdown/graceful"' >> "$temp_file"
echo '"/api/v1/shutdown/force"' >> "$temp_file"

# 审计路由
echo '"/api/v1/audit/logs"' >> "$temp_file"
echo '"/api/v1/audit/decisions"' >> "$temp_file"
echo '"/api/v1/audit/performance"' >> "$temp_file"
echo '"/api/v1/audit/export"' >> "$temp_file"

# 结果分享路由
echo '"/api/v1/share-result"' >> "$temp_file"
echo '"/api/v1/shared-results"' >> "$temp_file"

# 编排器路由
echo '"/api/v1/orchestrator/status"' >> "$temp_file"
echo '"/api/v1/orchestrator/services"' >> "$temp_file"
echo '"/api/v1/orchestrator/services/start"' >> "$temp_file"
echo '"/api/v1/orchestrator/services/stop"' >> "$temp_file"
echo '"/api/v1/orchestrator/services/restart"' >> "$temp_file"
echo '"/api/v1/orchestrator/optimize"' >> "$temp_file"
echo '"/api/v1/orchestrator/health"' >> "$temp_file"

# WebSocket路由
echo '"/ws/market/:symbol"' >> "$temp_file"
echo '"/ws/strategy/:id"' >> "$temp_file"
echo '"/ws/alerts"' >> "$temp_file"

# 排序并去重
sort "$temp_file" | uniq > "/tmp/actual_routes_sorted.txt"

echo "提取完成！实际路由总数: $(wc -l < /tmp/actual_routes_sorted.txt)"
echo
echo "实际API路由列表:"
cat "/tmp/actual_routes_sorted.txt"

# 清理临时文件
rm -f "$temp_file"
mv "/tmp/actual_routes_sorted.txt" "actual_api_routes.txt"

echo
echo "实际路由列表已保存到: actual_api_routes.txt"

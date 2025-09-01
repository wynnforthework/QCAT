#!/bin/bash

# QCAT API状态检查脚本
# 检查所有API接口的可用性，区分已实现和未实现的接口

API_BASE="http://localhost:8082"

echo "=== QCAT API接口状态检查 ==="
echo

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
GRAY='\033[0;37m'
NC='\033[0m' # No Color

# 统计变量
TOTAL_APIS=0
AVAILABLE_APIS=0
UNAVAILABLE_APIS=0
NOT_IMPLEMENTED_APIS=0

# 检查API函数
check_api() {
    local name="$1"
    local method="$2"
    local url="$3"
    local auth_header="$4"
    local category="$5"
    
    TOTAL_APIS=$((TOTAL_APIS + 1))
    
    if [ "$method" = "GET" ]; then
        if [ -n "$auth_header" ]; then
            response=$(curl -s -w "\n%{http_code}" -H "$auth_header" "$url" 2>/dev/null)
        else
            response=$(curl -s -w "\n%{http_code}" "$url" 2>/dev/null)
        fi
    else
        if [ -n "$auth_header" ]; then
            response=$(curl -s -w "\n%{http_code}" -X "$method" -H "Content-Type: application/json" -H "$auth_header" -d '{}' "$url" 2>/dev/null)
        else
            response=$(curl -s -w "\n%{http_code}" -X "$method" -H "Content-Type: application/json" -d '{}' "$url" 2>/dev/null)
        fi
    fi
    
    http_code=$(echo "$response" | tail -n1)
    
    case $http_code in
        200|201)
            echo -e "${GREEN}✅ $name${NC} - $category"
            AVAILABLE_APIS=$((AVAILABLE_APIS + 1))
            ;;
        401)
            echo -e "${YELLOW}🔒 $name${NC} - $category (需要认证)"
            AVAILABLE_APIS=$((AVAILABLE_APIS + 1))
            ;;
        404)
            echo -e "${RED}❌ $name${NC} - $category (404 未找到)"
            UNAVAILABLE_APIS=$((UNAVAILABLE_APIS + 1))
            ;;
        500)
            echo -e "${RED}💥 $name${NC} - $category (500 服务器错误)"
            UNAVAILABLE_APIS=$((UNAVAILABLE_APIS + 1))
            ;;
        "")
            echo -e "${GRAY}🔌 $name${NC} - $category (连接失败)"
            NOT_IMPLEMENTED_APIS=$((NOT_IMPLEMENTED_APIS + 1))
            ;;
        *)
            echo -e "${YELLOW}⚠️ $name${NC} - $category (HTTP $http_code)"
            UNAVAILABLE_APIS=$((UNAVAILABLE_APIS + 1))
            ;;
    esac
}

# 首先检查API服务是否运行
echo -e "${BLUE}=== 基础连接检查 ===${NC}"
health_response=$(curl -s -w "\n%{http_code}" "$API_BASE/health" 2>/dev/null)
health_code=$(echo "$health_response" | tail -n1)

if [ "$health_code" != "200" ]; then
    echo -e "${RED}❌ API服务未运行或不可访问${NC}"
    echo "请先启动API服务: mage simple 或 go run cmd/api/main.go"
    exit 1
fi

echo -e "${GREEN}✅ API服务正常运行${NC}"
echo

# 尝试获取访问令牌
echo -e "${BLUE}=== 获取访问令牌 ===${NC}"
login_response=$(curl -s -w "\n%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}' \
    "$API_BASE/api/v1/auth/login" 2>/dev/null)

login_code=$(echo "$login_response" | tail -n1)
login_body=$(echo "$login_response" | head -n -1)

if [ "$login_code" = "200" ]; then
    access_token=$(echo "$login_body" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
    if [ -n "$access_token" ]; then
        echo -e "${GREEN}✅ 成功获取访问令牌${NC}"
        auth_header="Authorization: Bearer $access_token"
    else
        echo -e "${YELLOW}⚠️ 登录成功但无法提取令牌${NC}"
        auth_header=""
    fi
else
    echo -e "${YELLOW}⚠️ 无法登录，将测试无认证接口${NC}"
    auth_header=""
fi

echo

# 检查所有API接口
echo -e "${BLUE}=== API接口状态检查 ===${NC}"

# 认证相关API - 使用特殊的测试方法
echo -e "${BLUE}测试认证API...${NC}"

# 测试用户登录
login_test_response=$(curl -s -w "\n%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}' \
    "$API_BASE/api/v1/auth/login" 2>/dev/null)
login_test_code=$(echo "$login_test_response" | tail -n1)
TOTAL_APIS=$((TOTAL_APIS + 1))
if [ "$login_test_code" = "200" ]; then
    echo -e "${GREEN}✅ 用户登录${NC} - 认证管理"
    AVAILABLE_APIS=$((AVAILABLE_APIS + 1))
else
    echo -e "${YELLOW}⚠️ 用户登录${NC} - 认证管理 (HTTP $login_test_code)"
    UNAVAILABLE_APIS=$((UNAVAILABLE_APIS + 1))
fi

# 测试用户注册
register_test_response=$(curl -s -w "\n%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -d '{"username":"testuser_'$(date +%s)'","email":"test'$(date +%s)'@example.com","password":"testpass123"}' \
    "$API_BASE/api/v1/auth/register" 2>/dev/null)
register_test_code=$(echo "$register_test_response" | tail -n1)
TOTAL_APIS=$((TOTAL_APIS + 1))
if [ "$register_test_code" = "200" ] || [ "$register_test_code" = "201" ]; then
    echo -e "${GREEN}✅ 用户注册${NC} - 认证管理"
    AVAILABLE_APIS=$((AVAILABLE_APIS + 1))
elif [ "$register_test_code" = "409" ]; then
    echo -e "${GREEN}✅ 用户注册${NC} - 认证管理 (用户已存在，功能正常)"
    AVAILABLE_APIS=$((AVAILABLE_APIS + 1))
else
    echo -e "${YELLOW}⚠️ 用户注册${NC} - 认证管理 (HTTP $register_test_code)"
    UNAVAILABLE_APIS=$((UNAVAILABLE_APIS + 1))
fi

check_api "获取用户信息" "GET" "$API_BASE/api/v1/auth/profile" "$auth_header" "认证管理"

# 测试刷新令牌
if [ -n "$access_token" ]; then
    refresh_test_response=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d "{\"refresh_token\":\"$access_token\"}" \
        "$API_BASE/api/v1/auth/refresh" 2>/dev/null)
    refresh_test_code=$(echo "$refresh_test_response" | tail -n1)
    TOTAL_APIS=$((TOTAL_APIS + 1))
    if [ "$refresh_test_code" = "200" ]; then
        echo -e "${GREEN}✅ 刷新令牌${NC} - 认证管理"
        AVAILABLE_APIS=$((AVAILABLE_APIS + 1))
    else
        echo -e "${YELLOW}⚠️ 刷新令牌${NC} - 认证管理 (HTTP $refresh_test_code)"
        UNAVAILABLE_APIS=$((UNAVAILABLE_APIS + 1))
    fi
else
    echo -e "${YELLOW}⚠️ 刷新令牌${NC} - 认证管理 (无法获取令牌进行测试)"
    TOTAL_APIS=$((TOTAL_APIS + 1))
    UNAVAILABLE_APIS=$((UNAVAILABLE_APIS + 1))
fi

# 仪表盘API
check_api "仪表盘概览" "GET" "$API_BASE/api/v1/dashboard" "$auth_header" "仪表盘"
check_api "数据库健康检查" "GET" "$API_BASE/api/v1/dashboard/db-health" "$auth_header" "仪表盘"

# 策略管理API
check_api "策略列表" "GET" "$API_BASE/api/v1/strategy" "$auth_header" "策略管理"

check_api "策略池概览" "GET" "$API_BASE/api/v1/strategy/pool/overview" "$auth_header" "策略管理"
check_api "策略执行概览" "GET" "$API_BASE/api/v1/strategy/execution/overview" "$auth_header" "策略管理"
check_api "策略实时状态" "GET" "$API_BASE/api/v1/strategy/execution/realtime" "$auth_header" "策略管理"
check_api "策略工作流状态" "GET" "$API_BASE/api/v1/strategy/workflow/status" "$auth_header" "策略管理"

# 投资组合API
check_api "投资组合概览" "GET" "$API_BASE/api/v1/portfolio/overview" "$auth_header" "投资组合"
check_api "投资组合分配" "GET" "$API_BASE/api/v1/portfolio/allocations" "$auth_header" "投资组合"
check_api "投资组合历史" "GET" "$API_BASE/api/v1/portfolio/history" "$auth_header" "投资组合"
check_api "投资组合性能" "GET" "$API_BASE/api/v1/portfolio/performance" "$auth_header" "投资组合"

# 风险管理API
check_api "风险概览" "GET" "$API_BASE/api/v1/risk/overview" "$auth_header" "风险管理"
check_api "风险限制" "GET" "$API_BASE/api/v1/risk/limits" "$auth_header" "风险管理"
check_api "熔断器状态" "GET" "$API_BASE/api/v1/risk/circuit-breakers" "$auth_header" "风险管理"
check_api "风险违规记录" "GET" "$API_BASE/api/v1/risk/violations" "$auth_header" "风险管理"

# 市场数据API
check_api "市场数据" "GET" "$API_BASE/api/v1/market/data" "$auth_header" "市场数据"

# 交易活动API
check_api "交易活动" "GET" "$API_BASE/api/v1/trading/activity" "$auth_header" "交易活动"
check_api "交易历史" "GET" "$API_BASE/api/v1/trading/history" "$auth_header" "交易活动"
check_api "交易持仓" "GET" "$API_BASE/api/v1/trading/positions" "$auth_header" "交易活动"

# 系统监控API
check_api "系统指标" "GET" "$API_BASE/api/v1/metrics/system" "$auth_header" "系统监控"

# 审计日志API
check_api "审计日志" "GET" "$API_BASE/api/v1/audit/logs" "$auth_header" "审计日志"
check_api "决策链" "GET" "$API_BASE/api/v1/audit/decisions" "$auth_header" "审计日志"
check_api "性能指标" "GET" "$API_BASE/api/v1/audit/performance" "$auth_header" "审计日志"

# 审计导出API
check_api "导出审计报告" "POST" "$API_BASE/api/v1/audit/export" "$auth_header" "审计日志"

# 编排器API
check_api "编排器状态" "GET" "$API_BASE/api/v1/orchestrator/status" "$auth_header" "系统编排"
check_api "服务列表" "GET" "$API_BASE/api/v1/orchestrator/services" "$auth_header" "系统编排"
check_api "启动服务" "POST" "$API_BASE/api/v1/orchestrator/services/start" "$auth_header" "系统编排"
check_api "停止服务" "POST" "$API_BASE/api/v1/orchestrator/services/stop" "$auth_header" "系统编排"
check_api "重启服务" "POST" "$API_BASE/api/v1/orchestrator/services/restart" "$auth_header" "系统编排"
check_api "系统优化" "POST" "$API_BASE/api/v1/orchestrator/optimize" "$auth_header" "系统编排"
check_api "编排器健康检查" "GET" "$API_BASE/api/v1/orchestrator/health" "$auth_header" "系统编排"

# 热点管理API
check_api "热点符号" "GET" "$API_BASE/api/v1/hotlist/symbols" "$auth_header" "热点管理"
check_api "白名单" "GET" "$API_BASE/api/v1/hotlist/whitelist" "$auth_header" "热点管理"
check_api "添加白名单" "POST" "$API_BASE/api/v1/hotlist/whitelist/:symbol" "$auth_header" "热点管理"
check_api "批准热点" "POST" "$API_BASE/api/v1/hotlist/approve" "$auth_header" "热点管理"

# 仪表盘API (补充)
check_api "数据库健康" "GET" "$API_BASE/api/v1/dashboard/db-health" "$auth_header" "仪表盘"

# 自动启动管理API
check_api "自动启动策略" "GET" "$API_BASE/api/v1/auto-start/strategies" "$auth_header" "自动启动"
check_api "自动启动统计" "GET" "$API_BASE/api/v1/auto-start/stats" "$auth_header" "自动启动"

# 黑名单API
check_api "黑名单列表" "GET" "$API_BASE/api/v1/blacklist/" "$auth_header" "黑名单管理"

# 并发管理API
check_api "线程池状态" "GET" "$API_BASE/api/v1/concurrent/pools" "$auth_header" "并发管理"
check_api "监控统计" "GET" "$API_BASE/api/v1/concurrent/monitor" "$auth_header" "并发管理"
check_api "并发告警" "GET" "$API_BASE/api/v1/concurrent/alerts" "$auth_header" "并发管理"
check_api "负载均衡器状态" "GET" "$API_BASE/api/v1/concurrent/load-balancer" "$auth_header" "并发管理"
check_api "任务队列状态" "GET" "$API_BASE/api/v1/concurrent/task-queue" "$auth_header" "并发管理"

# 优化器API
check_api "优化任务列表" "GET" "$API_BASE/api/v1/optimizer/tasks" "$auth_header" "优化器"

# 工作流API
check_api "依赖图" "GET" "$API_BASE/api/v1/workflow/dependency-graph" "$auth_header" "工作流"
check_api "执行结果" "GET" "$API_BASE/api/v1/workflow/results" "$auth_header" "工作流"
check_api "工作流状态" "GET" "$API_BASE/api/v1/workflow/status" "$auth_header" "工作流"
check_api "工作流验证" "GET" "$API_BASE/api/v1/workflow/validate" "$auth_header" "工作流"
check_api "启用的功能" "GET" "$API_BASE/api/v1/workflow/enabled" "$auth_header" "工作流"

# 紧急停止API
check_api "紧急停止状态" "GET" "$API_BASE/api/v1/emergency/status" "$auth_header" "紧急停止"
check_api "紧急停止历史" "GET" "$API_BASE/api/v1/emergency/history" "$auth_header" "紧急停止"
check_api "紧急停止重置" "POST" "$API_BASE/api/v1/emergency/reset" "$auth_header" "紧急停止"
check_api "紧急停止所有" "POST" "$API_BASE/api/v1/emergency/stop-all" "$auth_header" "紧急停止"

# 自动启动触发API
check_api "自动启动触发" "POST" "$API_BASE/api/v1/auto-start/trigger" "$auth_header" "自动启动"

# 黑名单管理API (补充)
check_api "删除黑名单条目" "DELETE" "$API_BASE/api/v1/blacklist/:strategy_id" "$auth_header" "黑名单管理"
check_api "清理过期条目" "POST" "$API_BASE/api/v1/blacklist/clear-expired" "$auth_header" "黑名单管理"

# 并发管理API (补充)
check_api "指定线程池状态" "GET" "$API_BASE/api/v1/concurrent/pools/:pool_name" "$auth_header" "并发管理"
check_api "线程池扩缩容" "POST" "$API_BASE/api/v1/concurrent/pools/:pool_name/scale" "$auth_header" "并发管理"
check_api "任务列表" "GET" "$API_BASE/api/v1/concurrent/tasks" "$auth_header" "并发管理"

# 策略验证API
check_api "策略验证状态" "GET" "$API_BASE/api/v1/validation/strategies" "$auth_header" "策略验证"
check_api "策略问题" "GET" "$API_BASE/api/v1/validation/problems" "$auth_header" "策略验证"
check_api "自动化状态" "GET" "$API_BASE/api/v1/validation/automation" "$auth_header" "策略验证"

# 系统稳定性API
check_api "内存统计" "GET" "$API_BASE/api/v1/memory/stats" "$auth_header" "系统稳定性"
check_api "内存垃圾回收" "POST" "$API_BASE/api/v1/memory/gc" "$auth_header" "系统稳定性"
check_api "网络连接" "GET" "$API_BASE/api/v1/network/connections" "$auth_header" "系统稳定性"
check_api "指定网络连接" "GET" "$API_BASE/api/v1/network/connections/:id" "$auth_header" "系统稳定性"
check_api "网络重连" "POST" "$API_BASE/api/v1/network/connections/:id/reconnect" "$auth_header" "系统稳定性"
check_api "健康状态" "GET" "$API_BASE/api/v1/health/status" "$auth_header" "系统稳定性"
check_api "健康检查" "GET" "$API_BASE/api/v1/health/checks" "$auth_header" "系统稳定性"
check_api "指定健康检查" "GET" "$API_BASE/api/v1/health/checks/:name" "$auth_header" "系统稳定性"
check_api "强制健康检查" "POST" "$API_BASE/api/v1/health/checks/:name/force" "$auth_header" "系统稳定性"
check_api "关闭状态" "GET" "$API_BASE/api/v1/shutdown/status" "$auth_header" "系统稳定性"
check_api "优雅关闭" "POST" "$API_BASE/api/v1/shutdown/graceful" "$auth_header" "系统稳定性"
check_api "强制关闭" "POST" "$API_BASE/api/v1/shutdown/force" "$auth_header" "系统稳定性"

# 优化器API
check_api "运行优化" "POST" "$API_BASE/api/v1/optimizer/run" "$auth_header" "策略优化"
check_api "优化任务列表" "GET" "$API_BASE/api/v1/optimizer/tasks" "$auth_header" "策略优化"
check_api "优化任务详情" "GET" "$API_BASE/api/v1/optimizer/tasks/:id" "$auth_header" "策略优化"
check_api "优化结果" "GET" "$API_BASE/api/v1/optimizer/results/:id" "$auth_header" "策略优化"

# 策略操作API (补充)
check_api "策略回测" "POST" "$API_BASE/api/v1/strategy/:id/backtest" "$auth_header" "策略管理"
check_api "策略提升" "POST" "$API_BASE/api/v1/strategy/:id/promote" "$auth_header" "策略管理"
check_api "策略自动启动" "POST" "$API_BASE/api/v1/strategy/:id/auto-start" "$auth_header" "策略管理"
check_api "策略执行概览" "GET" "$API_BASE/api/v1/strategy/execution/overview" "$auth_header" "策略管理"
check_api "策略实时状态" "GET" "$API_BASE/api/v1/strategy/execution/realtime" "$auth_header" "策略管理"
check_api "策略池概览" "GET" "$API_BASE/api/v1/strategy/pool/overview" "$auth_header" "策略管理"
check_api "策略工作流状态" "GET" "$API_BASE/api/v1/strategy/workflow/status" "$auth_header" "策略管理"

# 工作流API
check_api "依赖图" "GET" "$API_BASE/api/v1/workflow/dependency-graph" "$auth_header" "工作流"
check_api "执行结果" "GET" "$API_BASE/api/v1/workflow/results" "$auth_header" "工作流"
check_api "工作流状态" "GET" "$API_BASE/api/v1/workflow/status" "$auth_header" "工作流"
check_api "工作流验证" "GET" "$API_BASE/api/v1/workflow/validate" "$auth_header" "工作流"
check_api "启用的功能" "GET" "$API_BASE/api/v1/workflow/enabled" "$auth_header" "工作流"
check_api "执行工作流" "POST" "$API_BASE/api/v1/workflow/execute" "$auth_header" "工作流"
check_api "工作流函数" "GET" "$API_BASE/api/v1/workflow/functions/:function_id" "$auth_header" "工作流"
check_api "启用函数" "POST" "$API_BASE/api/v1/workflow/functions/:function_id/enable" "$auth_header" "工作流"
check_api "禁用函数" "POST" "$API_BASE/api/v1/workflow/functions/:function_id/disable" "$auth_header" "工作流"

# 结果分享API
check_api "分享结果" "GET" "$API_BASE/api/v1/shared-results" "$auth_header" "结果分享"
check_api "创建分享" "POST" "$API_BASE/api/v1/share-result" "$auth_header" "结果分享"

# 指标API (补充)
check_api "性能指标" "GET" "$API_BASE/api/v1/metrics/performance" "$auth_header" "系统指标"
check_api "策略指标" "GET" "$API_BASE/api/v1/metrics/strategy/:id" "$auth_header" "系统指标"

# 投资组合API (补充)
check_api "投资组合重平衡" "POST" "$API_BASE/api/v1/portfolio/rebalance" "$auth_header" "投资组合"

# 策略基础API (补充)
check_api "策略详情" "GET" "$API_BASE/api/v1/strategy/:id" "$auth_header" "策略管理"
check_api "启动策略" "POST" "$API_BASE/api/v1/strategy/:id/start" "$auth_header" "策略管理"
check_api "停止策略" "POST" "$API_BASE/api/v1/strategy/:id/stop" "$auth_header" "策略管理"

# 基础健康检查API
check_api "基础健康检查" "GET" "$API_BASE/health" "" "基础API"
check_api "详细健康检查" "GET" "$API_BASE/health/detailed" "" "基础API"

# WebSocket API
check_api "WebSocket告警" "GET" "$API_BASE/ws/alerts" "" "WebSocket"
check_api "WebSocket市场数据" "GET" "$API_BASE/ws/market/:symbol" "" "WebSocket"
check_api "WebSocket策略数据" "GET" "$API_BASE/ws/strategy/:id" "" "WebSocket"

# 系统设置API
check_api "系统设置" "GET" "$API_BASE/api/v1/settings" "" "系统设置"

# WebSocket API
check_api "WebSocket告警" "GET" "$API_BASE/ws/alerts" "" "WebSocket"
check_api "WebSocket市场数据" "GET" "$API_BASE/ws/market/:symbol" "" "WebSocket"
check_api "WebSocket策略数据" "GET" "$API_BASE/ws/strategy/:id" "" "WebSocket"

# Swagger文档
check_api "Swagger UI" "GET" "$API_BASE/swagger/index.html" "" "API文档"
check_api "Swagger JSON" "GET" "$API_BASE/swagger/doc.json" "" "API文档"

echo
echo -e "${BLUE}=== 检查结果统计 ===${NC}"
echo "总接口数: $TOTAL_APIS"
echo -e "可用接口: ${GREEN}$AVAILABLE_APIS${NC}"
echo -e "不可用接口: ${RED}$UNAVAILABLE_APIS${NC}"
echo -e "连接失败: ${GRAY}$NOT_IMPLEMENTED_APIS${NC}"

echo
if [ $UNAVAILABLE_APIS -gt 0 ]; then
    echo -e "${YELLOW}⚠️ 发现 $UNAVAILABLE_APIS 个不可用的接口，建议检查实现状态${NC}"
else
    echo -e "${GREEN}🎉 所有接口都可以正常访问！${NC}"
fi

echo
echo "📝 建议："
echo "1. 404错误的接口需要检查路由注册和handler实现"
echo "2. 500错误的接口需要检查业务逻辑和数据库连接"
echo "3. 需要认证的接口是正常的，表示安全机制工作正常"
echo "4. 更新API文档，移除未实现的接口或标记其状态"

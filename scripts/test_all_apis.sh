#!/bin/bash

# QCAT API完整测试脚本
BASE_URL="http://localhost:8082"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 计数器
TOTAL=0
SUCCESS=0
AUTH_REQUIRED=0
CLIENT_ERROR=0
SERVER_ERROR=0
NETWORK_ERROR=0

# 测试函数
test_endpoint() {
    local method=$1
    local path=$2
    local needs_body=$3
    local description=$4
    
    TOTAL=$((TOTAL + 1))
    printf "[$TOTAL] Testing %s %s" "$method" "$path"
    if [ -n "$description" ]; then
        printf " (%s)" "$description"
    fi
    echo
    
    # 构建curl命令
    local curl_cmd="curl -s -w '%{http_code}' -o /dev/null"
    curl_cmd="$curl_cmd -X $method"
    curl_cmd="$curl_cmd -H 'Content-Type: application/json'"
    curl_cmd="$curl_cmd --connect-timeout 5"
    
    # 如果需要请求体，添加测试数据
    if [ "$needs_body" = "true" ]; then
        curl_cmd="$curl_cmd -d '{\"test\":\"data\"}'"
    fi
    
    curl_cmd="$curl_cmd $BASE_URL$path"
    
    # 执行请求
    local status_code
    status_code=$(eval "$curl_cmd" 2>/dev/null)
    
    # 分析结果
    case $status_code in
        200|201|204)
            printf "  ${GREEN}SUCCESS ($status_code)${NC}\n"
            SUCCESS=$((SUCCESS + 1))
            ;;
        401)
            printf "  ${YELLOW}AUTH REQUIRED (401)${NC}\n"
            AUTH_REQUIRED=$((AUTH_REQUIRED + 1))
            ;;
        400|404|409)
            printf "  ${YELLOW}CLIENT ERROR ($status_code)${NC}\n"
            CLIENT_ERROR=$((CLIENT_ERROR + 1))
            ;;
        500|502|503)
            printf "  ${RED}SERVER ERROR ($status_code)${NC}\n"
            SERVER_ERROR=$((SERVER_ERROR + 1))
            ;;
        000|"")
            printf "  ${RED}NETWORK ERROR${NC}\n"
            NETWORK_ERROR=$((NETWORK_ERROR + 1))
            ;;
        *)
            printf "  ${RED}UNKNOWN ERROR ($status_code)${NC}\n"
            SERVER_ERROR=$((SERVER_ERROR + 1))
            ;;
    esac
    
    sleep 0.1
}

echo -e "${CYAN}=== QCAT API Complete Test Suite ===${NC}"
echo -e "${YELLOW}Testing all API endpoints to identify fixable issues${NC}"
echo

# 基础健康检查
echo -e "${BLUE}--- Basic Health Checks ---${NC}"
test_endpoint "GET" "/health" "false" "Basic health"

# 认证接口
echo -e "${BLUE}--- Authentication APIs ---${NC}"
test_endpoint "POST" "/api/v1/auth/login" "true" "User login"
test_endpoint "POST" "/api/v1/auth/register" "true" "User registration"
test_endpoint "POST" "/api/v1/auth/refresh" "true" "Token refresh"

# 核心业务接口
echo -e "${BLUE}--- Core Business APIs ---${NC}"
test_endpoint "GET" "/api/v1/dashboard" "false" "Dashboard data"
test_endpoint "GET" "/api/v1/market/data" "false" "Market data"
test_endpoint "GET" "/api/v1/trading/activity" "false" "Trading activity"

# 系统指标
echo -e "${BLUE}--- System Metrics ---${NC}"
test_endpoint "GET" "/api/v1/metrics/system" "false" "System metrics"
test_endpoint "GET" "/api/v1/metrics/performance" "false" "Performance metrics"
test_endpoint "GET" "/api/v1/metrics/strategy/test-id" "false" "Strategy metrics"

# 策略管理
echo -e "${BLUE}--- Strategy Management ---${NC}"
test_endpoint "GET" "/api/v1/strategy/" "false" "List strategies"
test_endpoint "POST" "/api/v1/strategy/" "true" "Create strategy"
test_endpoint "GET" "/api/v1/strategy/test-id" "false" "Get strategy"
test_endpoint "PUT" "/api/v1/strategy/test-id" "true" "Update strategy"
test_endpoint "DELETE" "/api/v1/strategy/test-id" "false" "Delete strategy"
test_endpoint "POST" "/api/v1/strategy/test-id/promote" "true" "Promote strategy"
test_endpoint "POST" "/api/v1/strategy/test-id/start" "false" "Start strategy"
test_endpoint "POST" "/api/v1/strategy/test-id/stop" "false" "Stop strategy"
test_endpoint "POST" "/api/v1/strategy/test-id/backtest" "true" "Run backtest"

# 优化器
echo -e "${BLUE}--- Optimizer APIs ---${NC}"
test_endpoint "POST" "/api/v1/optimizer/run" "true" "Run optimization"
test_endpoint "GET" "/api/v1/optimizer/tasks" "false" "Get optimizer tasks"
test_endpoint "GET" "/api/v1/optimizer/tasks/test-id" "false" "Get optimizer task"
test_endpoint "GET" "/api/v1/optimizer/results/test-id" "false" "Get optimizer results"

# 投资组合
echo -e "${BLUE}--- Portfolio APIs ---${NC}"
test_endpoint "GET" "/api/v1/portfolio/overview" "false" "Portfolio overview"
test_endpoint "GET" "/api/v1/portfolio/allocations" "false" "Portfolio allocations"
test_endpoint "POST" "/api/v1/portfolio/rebalance" "true" "Portfolio rebalance"
test_endpoint "GET" "/api/v1/portfolio/history" "false" "Portfolio history"

# 风险管理
echo -e "${BLUE}--- Risk Management ---${NC}"
test_endpoint "GET" "/api/v1/risk/overview" "false" "Risk overview"
test_endpoint "GET" "/api/v1/risk/limits" "false" "Risk limits"
test_endpoint "POST" "/api/v1/risk/limits" "true" "Set risk limits"
test_endpoint "GET" "/api/v1/risk/circuit-breakers" "false" "Circuit breakers"
test_endpoint "POST" "/api/v1/risk/circuit-breakers" "true" "Set circuit breakers"
test_endpoint "GET" "/api/v1/risk/violations" "false" "Risk violations"

# 热门列表
echo -e "${BLUE}--- Hotlist APIs ---${NC}"
test_endpoint "GET" "/api/v1/hotlist/symbols" "false" "Hot symbols"
test_endpoint "POST" "/api/v1/hotlist/approve" "true" "Approve symbol"
test_endpoint "GET" "/api/v1/hotlist/whitelist" "false" "Get whitelist"
test_endpoint "POST" "/api/v1/hotlist/whitelist" "true" "Add to whitelist"
test_endpoint "DELETE" "/api/v1/hotlist/whitelist/BTCUSDT" "false" "Remove from whitelist"

# 系统管理
echo -e "${BLUE}--- System Management ---${NC}"
test_endpoint "GET" "/api/v1/memory/stats" "false" "Memory stats"
test_endpoint "POST" "/api/v1/memory/gc" "false" "Force GC"
test_endpoint "GET" "/api/v1/network/connections" "false" "Network connections"
test_endpoint "GET" "/api/v1/network/connections/test-id" "false" "Get connection"
test_endpoint "POST" "/api/v1/network/connections/test-id/reconnect" "false" "Reconnect"

# 健康检查
echo -e "${BLUE}--- Health Check APIs ---${NC}"
test_endpoint "GET" "/api/v1/health/status" "false" "Health status"
test_endpoint "GET" "/api/v1/health/checks" "false" "All health checks"
test_endpoint "GET" "/api/v1/health/checks/database" "false" "Database health"
test_endpoint "POST" "/api/v1/health/checks/database/force" "false" "Force health check"

# 审计
echo -e "${BLUE}--- Audit APIs ---${NC}"
test_endpoint "GET" "/api/v1/audit/logs" "false" "Audit logs"
test_endpoint "GET" "/api/v1/audit/decisions" "false" "Audit decisions"
test_endpoint "GET" "/api/v1/audit/performance" "false" "Audit performance"
test_endpoint "POST" "/api/v1/audit/export" "true" "Export audit"

# 缓存管理
echo -e "${BLUE}--- Cache Management ---${NC}"
test_endpoint "GET" "/api/v1/cache/status" "false" "Cache status"
test_endpoint "GET" "/api/v1/cache/health" "false" "Cache health"
test_endpoint "GET" "/api/v1/cache/metrics" "false" "Cache metrics"
test_endpoint "GET" "/api/v1/cache/events" "false" "Cache events"
test_endpoint "GET" "/api/v1/cache/config" "false" "Cache config"
test_endpoint "POST" "/api/v1/cache/test" "false" "Test cache"
test_endpoint "POST" "/api/v1/cache/fallback/force" "false" "Force fallback"
test_endpoint "POST" "/api/v1/cache/counters/reset" "false" "Reset counters"

# 安全管理
echo -e "${BLUE}--- Security Management ---${NC}"
test_endpoint "POST" "/api/v1/security/keys/" "true" "Create API key"
test_endpoint "GET" "/api/v1/security/keys/" "false" "List API keys"
test_endpoint "GET" "/api/v1/security/keys/test-id" "false" "Get API key"
test_endpoint "POST" "/api/v1/security/keys/test-id/rotate" "false" "Rotate API key"
test_endpoint "POST" "/api/v1/security/keys/test-id/revoke" "false" "Revoke API key"
test_endpoint "GET" "/api/v1/security/keys/test-id/usage" "false" "Key usage"
test_endpoint "GET" "/api/v1/security/audit/logs" "false" "Security audit logs"
test_endpoint "GET" "/api/v1/security/audit/integrity" "false" "Audit integrity"

# 编排器
echo -e "${BLUE}--- Orchestrator APIs ---${NC}"
test_endpoint "GET" "/api/v1/orchestrator/status" "false" "Orchestrator status"
test_endpoint "GET" "/api/v1/orchestrator/services" "false" "Orchestrator services"
test_endpoint "POST" "/api/v1/orchestrator/services/start" "true" "Start service"
test_endpoint "POST" "/api/v1/orchestrator/services/stop" "true" "Stop service"
test_endpoint "POST" "/api/v1/orchestrator/services/restart" "true" "Restart service"
test_endpoint "POST" "/api/v1/orchestrator/optimize" "true" "Optimize"
test_endpoint "GET" "/api/v1/orchestrator/health" "false" "Orchestrator health"

# 关机管理
echo -e "${BLUE}--- Shutdown Management ---${NC}"
test_endpoint "GET" "/api/v1/shutdown/status" "false" "Shutdown status"
# 注意：不测试实际的关机接口，避免关闭服务器

echo
echo -e "${CYAN}=== TEST RESULTS SUMMARY ===${NC}"
echo -e "Total endpoints tested: ${TOTAL}"
echo -e "${GREEN}Success (2xx): ${SUCCESS}${NC}"
echo -e "${YELLOW}Auth required (401): ${AUTH_REQUIRED}${NC}"
echo -e "${YELLOW}Client errors (4xx): ${CLIENT_ERROR}${NC}"
echo -e "${RED}Server errors (5xx): ${SERVER_ERROR}${NC}"
echo -e "${RED}Network errors: ${NETWORK_ERROR}${NC}"

HEALTHY=$((SUCCESS + AUTH_REQUIRED))
HEALTHY_PERCENT=$((HEALTHY * 100 / TOTAL))

echo
if [ $HEALTHY_PERCENT -gt 90 ]; then
    echo -e "${GREEN}Healthy endpoints: ${HEALTHY}/${TOTAL} (${HEALTHY_PERCENT}%)${NC}"
elif [ $HEALTHY_PERCENT -gt 70 ]; then
    echo -e "${YELLOW}Healthy endpoints: ${HEALTHY}/${TOTAL} (${HEALTHY_PERCENT}%)${NC}"
else
    echo -e "${RED}Healthy endpoints: ${HEALTHY}/${TOTAL} (${HEALTHY_PERCENT}%)${NC}"
fi

echo
echo -e "${CYAN}=== RECOMMENDATIONS ===${NC}"

if [ $SERVER_ERROR -gt 0 ]; then
    echo -e "${RED}🔥 HIGH PRIORITY: Fix ${SERVER_ERROR} server errors (500/502/503)${NC}"
fi

if [ $CLIENT_ERROR -gt 0 ]; then
    echo -e "${YELLOW}📋 MEDIUM PRIORITY: Review ${CLIENT_ERROR} client errors (400/404/409)${NC}"
fi

if [ $AUTH_REQUIRED -gt 0 ]; then
    echo -e "${BLUE}ℹ️  INFO: ${AUTH_REQUIRED} endpoints require authentication (expected)${NC}"
fi

if [ $NETWORK_ERROR -gt 0 ]; then
    echo -e "${RED}🌐 NETWORK: ${NETWORK_ERROR} endpoints have network issues${NC}"
fi

echo
echo -e "${CYAN}Test completed. Check server logs for detailed error information.${NC}"

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
check_api "策略工作流状态" "GET" "$API_BASE/api/v1/strategy/workflow/status" "$auth_header" "策略管理"

# 投资组合API
check_api "投资组合概览" "GET" "$API_BASE/api/v1/portfolio/overview" "$auth_header" "投资组合"
check_api "投资组合持仓" "GET" "$API_BASE/api/v1/portfolio/positions" "$auth_header" "投资组合"
check_api "投资组合性能" "GET" "$API_BASE/api/v1/portfolio/performance" "$auth_header" "投资组合"

# 风险管理API
check_api "风险概览" "GET" "$API_BASE/api/v1/risk/overview" "$auth_header" "风险管理"
check_api "风险限制" "GET" "$API_BASE/api/v1/risk/limits" "$auth_header" "风险管理"
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

# 缓存管理API
check_api "缓存状态" "GET" "$API_BASE/api/v1/cache/status" "$auth_header" "缓存管理"
check_api "缓存健康检查" "GET" "$API_BASE/api/v1/cache/health" "$auth_header" "缓存管理"
check_api "缓存指标" "GET" "$API_BASE/api/v1/cache/metrics" "$auth_header" "缓存管理"

# 自动化系统API
check_api "自动化状态" "GET" "$API_BASE/api/v1/automation/status" "$auth_header" "自动化系统"
check_api "自动化健康检查" "GET" "$API_BASE/api/v1/automation/health" "$auth_header" "自动化系统"
check_api "执行统计" "GET" "$API_BASE/api/v1/automation/stats" "$auth_header" "自动化系统"

# 编排器API
check_api "编排器状态" "GET" "$API_BASE/api/v1/orchestrator/status" "$auth_header" "系统编排"
check_api "服务列表" "GET" "$API_BASE/api/v1/orchestrator/services" "$auth_header" "系统编排"
check_api "编排器健康检查" "GET" "$API_BASE/api/v1/orchestrator/health" "$auth_header" "系统编排"

# 热点管理API
check_api "热点符号" "GET" "$API_BASE/api/v1/hotlist/symbols" "$auth_header" "热点管理"
check_api "白名单" "GET" "$API_BASE/api/v1/hotlist/whitelist" "$auth_header" "热点管理"

# 黑名单API
check_api "黑名单列表" "GET" "$API_BASE/api/v1/blacklist" "$auth_header" "黑名单管理"

# 并发管理API
check_api "线程池状态" "GET" "$API_BASE/api/v1/concurrent/pools" "$auth_header" "并发管理"
check_api "监控统计" "GET" "$API_BASE/api/v1/concurrent/monitor" "$auth_header" "并发管理"

# 安全管理API
check_api "API密钥" "GET" "$API_BASE/api/v1/security/keys" "$auth_header" "安全管理"
check_api "安全审计日志" "GET" "$API_BASE/api/v1/security/audit/logs" "$auth_header" "安全管理"

# 系统设置API
check_api "系统设置" "GET" "$API_BASE/api/v1/settings" "" "系统设置"

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

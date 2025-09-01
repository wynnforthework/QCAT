#!/bin/bash

# QCAT API修复验证脚本
# 测试所有四个已修复的问题

API_BASE="http://localhost:8082"

echo "=== QCAT API修复验证测试 ==="
echo

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试结果统计
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 测试函数
test_endpoint() {
    local name="$1"
    local method="$2"
    local url="$3"
    local headers="$4"
    local data="$5"
    local expected_code="$6"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "${BLUE}测试 $TOTAL_TESTS: $name${NC}"
    
    if [ "$method" = "GET" ]; then
        if [ -n "$headers" ]; then
            response=$(curl -s -w "\n%{http_code}" -H "$headers" "$url")
        else
            response=$(curl -s -w "\n%{http_code}" "$url")
        fi
    else
        if [ -n "$headers" ]; then
            response=$(curl -s -w "\n%{http_code}" -X "$method" -H "Content-Type: application/json" -H "$headers" -d "$data" "$url")
        else
            response=$(curl -s -w "\n%{http_code}" -X "$method" -H "Content-Type: application/json" -d "$data" "$url")
        fi
    fi
    
    http_code=$(echo "$response" | tail -n1)
    response_body=$(echo "$response" | head -n -1)
    
    if [ "$http_code" = "$expected_code" ]; then
        echo -e "${GREEN}✅ 通过 (HTTP $http_code)${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        if [ ${#response_body} -lt 200 ]; then
            echo "响应: $response_body"
        else
            echo "响应: ${response_body:0:100}..."
        fi
    else
        echo -e "${RED}❌ 失败 (期望 HTTP $expected_code, 实际 HTTP $http_code)${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        echo "响应: $response_body"
    fi
    echo
}

# 1. 测试API服务是否运行
echo -e "${BLUE}=== 1. 基础连接测试 ===${NC}"
test_endpoint "健康检查" "GET" "$API_BASE/health" "" "" "200"

# 2. 测试用户登录 (修复问题1的前置条件)
echo -e "${BLUE}=== 2. 用户认证测试 ===${NC}"
login_response=$(curl -s -w "\n%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}' \
    "$API_BASE/api/v1/auth/login")

login_code=$(echo "$login_response" | tail -n1)
login_body=$(echo "$login_response" | head -n -1)

if [ "$login_code" = "200" ]; then
    echo -e "${GREEN}✅ 用户登录成功${NC}"
    access_token=$(echo "$login_body" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
    
    if [ -n "$access_token" ]; then
        echo "获取到访问令牌: ${access_token:0:30}..."
        
        # 3. 测试修复问题1: /api/v1/auth/profile 401错误
        echo -e "${BLUE}=== 3. 修复问题1测试: JWT认证 ===${NC}"
        test_endpoint "获取用户信息" "GET" "$API_BASE/api/v1/auth/profile" "Authorization: Bearer $access_token" "" "200"
        
    else
        echo -e "${RED}❌ 无法提取访问令牌${NC}"
        access_token=""
    fi
else
    echo -e "${RED}❌ 用户登录失败 (HTTP $login_code)${NC}"
    echo "响应: $login_body"
    access_token=""
fi

# 4. 测试修复问题2: /api/v1/strategy 404错误
echo -e "${BLUE}=== 4. 修复问题2测试: 策略列表API ===${NC}"
if [ -n "$access_token" ]; then
    test_endpoint "获取策略列表" "GET" "$API_BASE/api/v1/strategy" "Authorization: Bearer $access_token" "" "200"
else
    test_endpoint "获取策略列表(无认证)" "GET" "$API_BASE/api/v1/strategy" "" "" "401"
fi

# 5. 测试修复问题3: /api/v1/strategy/pool/overview 404错误
echo -e "${BLUE}=== 5. 修复问题3测试: 策略池概览API ===${NC}"
if [ -n "$access_token" ]; then
    test_endpoint "获取策略池概览" "GET" "$API_BASE/api/v1/strategy/pool/overview" "Authorization: Bearer $access_token" "" "200"
else
    test_endpoint "获取策略池概览(无认证)" "GET" "$API_BASE/api/v1/strategy/pool/overview" "" "" "401"
fi

# 6. 测试修复问题4: Swagger文档生成
echo -e "${BLUE}=== 6. 修复问题4测试: Swagger文档 ===${NC}"
test_endpoint "Swagger UI" "GET" "$API_BASE/swagger/index.html" "" "" "200"
test_endpoint "Swagger JSON" "GET" "$API_BASE/swagger/doc.json" "" "" "200"

# 7. 额外测试：无效token处理
echo -e "${BLUE}=== 7. 额外测试: 错误处理 ===${NC}"
test_endpoint "无效token测试" "GET" "$API_BASE/api/v1/auth/profile" "Authorization: Bearer invalid_token" "" "401"
test_endpoint "缺少Authorization header" "GET" "$API_BASE/api/v1/auth/profile" "" "" "401"

# 测试结果总结
echo -e "${BLUE}=== 测试结果总结 ===${NC}"
echo "总测试数: $TOTAL_TESTS"
echo -e "通过: ${GREEN}$PASSED_TESTS${NC}"
echo -e "失败: ${RED}$FAILED_TESTS${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}🎉 所有测试通过！API修复成功！${NC}"
    exit 0
else
    echo -e "${RED}⚠️ 有 $FAILED_TESTS 个测试失败，请检查相关问题${NC}"
    exit 1
fi

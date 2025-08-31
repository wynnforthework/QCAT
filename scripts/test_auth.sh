#!/bin/bash

# 测试认证功能的脚本
# 用于验证登录、获取用户信息等API是否正常工作

API_BASE="http://localhost:8082/api/v1"

echo "=== QCAT 认证功能测试 ==="
echo

# 测试用户凭据
TEST_USERS=(
    "admin:admin123"
    "demo:demo123"
    "testuser:test123"
)

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试登录功能
test_login() {
    local username=$1
    local password=$2
    
    echo -e "${YELLOW}测试登录: $username${NC}"
    
    # 发送登录请求
    response=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$username\",\"password\":\"$password\"}" \
        "$API_BASE/auth/login")
    
    # 分离响应体和状态码
    http_code=$(echo "$response" | tail -n1)
    response_body=$(echo "$response" | head -n -1)
    
    if [ "$http_code" = "200" ]; then
        echo -e "${GREEN}✓ 登录成功${NC}"
        
        # 提取access_token
        access_token=$(echo "$response_body" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
        
        if [ -n "$access_token" ]; then
            echo "  Access Token: ${access_token:0:20}..."
            
            # 测试获取用户信息
            test_profile "$access_token"
        else
            echo -e "${RED}✗ 无法提取access_token${NC}"
        fi
    else
        echo -e "${RED}✗ 登录失败 (HTTP $http_code)${NC}"
        echo "  响应: $response_body"
    fi
    
    echo
}

# 测试获取用户信息
test_profile() {
    local token=$1
    
    echo -e "${YELLOW}  测试获取用户信息${NC}"
    
    response=$(curl -s -w "\n%{http_code}" \
        -H "Authorization: Bearer $token" \
        "$API_BASE/auth/profile")
    
    http_code=$(echo "$response" | tail -n1)
    response_body=$(echo "$response" | head -n -1)
    
    if [ "$http_code" = "200" ]; then
        echo -e "${GREEN}  ✓ 获取用户信息成功${NC}"
        echo "    响应: $response_body"
    else
        echo -e "${RED}  ✗ 获取用户信息失败 (HTTP $http_code)${NC}"
        echo "    响应: $response_body"
    fi
}

# 测试受保护的API接口
test_protected_apis() {
    local token=$1
    
    echo -e "${YELLOW}  测试受保护的API接口${NC}"
    
    # 测试dashboard接口
    echo "    测试 /dashboard..."
    response=$(curl -s -w "\n%{http_code}" \
        -H "Authorization: Bearer $token" \
        "$API_BASE/dashboard")
    
    http_code=$(echo "$response" | tail -n1)
    
    if [ "$http_code" = "200" ]; then
        echo -e "${GREEN}    ✓ Dashboard API 正常${NC}"
    else
        echo -e "${RED}    ✗ Dashboard API 失败 (HTTP $http_code)${NC}"
    fi
    
    # 测试market data接口
    echo "    测试 /market/data..."
    response=$(curl -s -w "\n%{http_code}" \
        -H "Authorization: Bearer $token" \
        "$API_BASE/market/data")
    
    http_code=$(echo "$response" | tail -n1)
    
    if [ "$http_code" = "200" ]; then
        echo -e "${GREEN}    ✓ Market Data API 正常${NC}"
    else
        echo -e "${RED}    ✗ Market Data API 失败 (HTTP $http_code)${NC}"
    fi
}

# 主测试流程
main() {
    echo "开始测试认证功能..."
    echo
    
    # 测试服务器连接
    echo -e "${YELLOW}检查服务器连接...${NC}"
    if curl -s --connect-timeout 5 "$API_BASE/auth/login" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ 服务器连接正常${NC}"
    else
        echo -e "${RED}✗ 无法连接到服务器 ($API_BASE)${NC}"
        echo "请确保QCAT服务正在运行"
        exit 1
    fi
    echo
    
    # 测试每个用户
    for user_cred in "${TEST_USERS[@]}"; do
        IFS=':' read -r username password <<< "$user_cred"
        test_login "$username" "$password"
    done
    
    echo "=== 认证测试完成 ==="
}

# 运行主函数
main "$@"

#!/bin/bash

# JWT认证测试脚本
# 用于测试修复后的JWT认证功能

API_BASE="http://localhost:8082/api/v1"

echo "=== QCAT JWT认证修复测试 ==="
echo

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试1: 健康检查
echo -e "${BLUE}1. 测试API健康检查${NC}"
health_response=$(curl -s -w "\n%{http_code}" "http://localhost:8082/health")
health_code=$(echo "$health_response" | tail -n1)
health_body=$(echo "$health_response" | head -n -1)

if [ "$health_code" = "200" ]; then
    echo -e "${GREEN}✅ API服务正常运行${NC}"
    echo "响应: $health_body"
else
    echo -e "${RED}❌ API服务不可用 (HTTP $health_code)${NC}"
    echo "请先启动API服务: mage simple"
    exit 1
fi

echo

# 测试2: 用户登录
echo -e "${BLUE}2. 测试用户登录${NC}"
login_response=$(curl -s -w "\n%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}' \
    "$API_BASE/auth/login")

login_code=$(echo "$login_response" | tail -n1)
login_body=$(echo "$login_response" | head -n -1)

if [ "$login_code" = "200" ]; then
    echo -e "${GREEN}✅ 用户登录成功${NC}"
    
    # 提取access token
    access_token=$(echo "$login_body" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
    
    if [ -n "$access_token" ]; then
        echo "Access Token: ${access_token:0:50}..."
    else
        echo -e "${RED}❌ 无法提取access token${NC}"
        echo "登录响应: $login_body"
        exit 1
    fi
else
    echo -e "${RED}❌ 用户登录失败 (HTTP $login_code)${NC}"
    echo "响应: $login_body"
    exit 1
fi

echo

# 测试3: 获取用户信息（使用JWT token）
echo -e "${BLUE}3. 测试获取用户信息 (JWT认证)${NC}"
profile_response=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: Bearer $access_token" \
    "$API_BASE/auth/profile")

profile_code=$(echo "$profile_response" | tail -n1)
profile_body=$(echo "$profile_response" | head -n -1)

if [ "$profile_code" = "200" ]; then
    echo -e "${GREEN}✅ JWT认证成功，获取用户信息成功${NC}"
    echo "用户信息: $profile_body"
else
    echo -e "${RED}❌ JWT认证失败 (HTTP $profile_code)${NC}"
    echo "响应: $profile_body"
    
    # 分析可能的错误原因
    if [[ "$profile_body" == *"Authorization header required"* ]]; then
        echo -e "${YELLOW}⚠️ 可能原因: Authorization header未发送${NC}"
    elif [[ "$profile_body" == *"Bearer token required"* ]]; then
        echo -e "${YELLOW}⚠️ 可能原因: Bearer token格式错误${NC}"
    elif [[ "$profile_body" == *"Invalid token"* ]]; then
        echo -e "${YELLOW}⚠️ 可能原因: JWT token无效或过期${NC}"
    elif [[ "$profile_body" == *"User not found"* ]]; then
        echo -e "${YELLOW}⚠️ 可能原因: 用户不存在${NC}"
    fi
    
    exit 1
fi

echo

# 测试4: 测试无效token
echo -e "${BLUE}4. 测试无效token处理${NC}"
invalid_response=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: Bearer invalid_token_123" \
    "$API_BASE/auth/profile")

invalid_code=$(echo "$invalid_response" | tail -n1)
invalid_body=$(echo "$invalid_response" | head -n -1)

if [ "$invalid_code" = "401" ]; then
    echo -e "${GREEN}✅ 无效token正确被拒绝${NC}"
    echo "响应: $invalid_body"
else
    echo -e "${YELLOW}⚠️ 无效token处理异常 (HTTP $invalid_code)${NC}"
    echo "响应: $invalid_body"
fi

echo

# 测试5: 测试缺少Authorization header
echo -e "${BLUE}5. 测试缺少Authorization header${NC}"
no_auth_response=$(curl -s -w "\n%{http_code}" \
    "$API_BASE/auth/profile")

no_auth_code=$(echo "$no_auth_response" | tail -n1)
no_auth_body=$(echo "$no_auth_response" | head -n -1)

if [ "$no_auth_code" = "401" ]; then
    echo -e "${GREEN}✅ 缺少Authorization header正确被拒绝${NC}"
    echo "响应: $no_auth_body"
else
    echo -e "${YELLOW}⚠️ 缺少Authorization header处理异常 (HTTP $no_auth_code)${NC}"
    echo "响应: $no_auth_body"
fi

echo
echo -e "${GREEN}=== JWT认证测试完成 ===${NC}"
echo
echo "如果所有测试都通过，JWT认证问题已修复！"
echo "如果仍有问题，请检查："
echo "1. 数据库连接是否正常"
echo "2. 用户表是否存在admin用户"
echo "3. JWT secret key是否正确配置"
echo "4. API服务是否正常运行"

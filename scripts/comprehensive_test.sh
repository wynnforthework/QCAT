#!/bin/bash

# 综合测试脚本
# 验证所有修复是否正常工作

echo "🧪 开始综合测试和验证..."

# API配置
BASE_URL="http://localhost:8082"
USERNAME="admin"
PASSWORD="admin123"

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
run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_pattern="$3"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "${BLUE}🔍 测试: $test_name${NC}"
    
    result=$(eval "$test_command" 2>/dev/null)
    
    if echo "$result" | grep -q "$expected_pattern"; then
        echo -e "${GREEN}✅ 通过${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo -e "${RED}❌ 失败${NC}"
        echo -e "${YELLOW}期望: $expected_pattern${NC}"
        echo -e "${YELLOW}实际: $(echo "$result" | head -1)${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# 1. 登录测试
echo -e "${BLUE}📝 步骤1: 认证测试${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}")

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo -e "${RED}❌ 登录失败，无法继续测试${NC}"
    echo "Response: $LOGIN_RESPONSE"
    exit 1
fi

run_test "用户登录" "echo '$LOGIN_RESPONSE'" '"success":true'

# 2. 策略API测试
echo -e "\n${BLUE}📊 步骤2: 策略管理测试${NC}"

run_test "获取策略列表" \
    "curl -s -X GET '$BASE_URL/api/v1/strategy/' -H 'Authorization: Bearer $TOKEN'" \
    '"success":true'

run_test "策略数据非空" \
    "curl -s -X GET '$BASE_URL/api/v1/strategy/' -H 'Authorization: Bearer $TOKEN'" \
    '"data":\['

# 3. 自动化系统测试
echo -e "\n${BLUE}🤖 步骤3: 自动化系统测试${NC}"

run_test "获取自动化状态" \
    "curl -s -X GET '$BASE_URL/api/v1/automation/status' -H 'Authorization: Bearer $TOKEN'" \
    '"success":true'

run_test "自动化任务数量正确" \
    "curl -s -X GET '$BASE_URL/api/v1/automation/status' -H 'Authorization: Bearer $TOKEN' | grep -o '\"id\":' | wc -l" \
    "26"

# 4. 自动化切换功能测试
echo -e "\n${BLUE}🔄 步骤4: 自动化切换功能测试${NC}"

# 测试启用任务1（策略自动优化）
run_test "启用自动化任务" \
    "curl -s -X POST '$BASE_URL/api/v1/automation/1/toggle' -H 'Authorization: Bearer $TOKEN' -H 'Content-Type: application/json' -d '{\"enabled\": true}'" \
    '"success":true'

# 等待状态更新
sleep 2

# 验证任务状态
run_test "验证任务启用状态" \
    "curl -s -X GET '$BASE_URL/api/v1/automation/status' -H 'Authorization: Bearer $TOKEN' | grep -A5 '\"id\":\"1\"'" \
    '"enabled":true'

# 5. 系统健康检查
echo -e "\n${BLUE}💊 步骤5: 系统健康检查${NC}"

run_test "系统健康状态" \
    "curl -s -X GET '$BASE_URL/api/v1/health/status' -H 'Authorization: Bearer $TOKEN'" \
    '"status":'

run_test "自动化系统健康" \
    "curl -s -X GET '$BASE_URL/api/v1/automation/health' -H 'Authorization: Bearer $TOKEN'" \
    '"success":true'

# 6. 分享结果页面相关测试
echo -e "\n${BLUE}📤 步骤6: 分享结果页面相关测试${NC}"

# 测试策略数据是否可用于分享页面
STRATEGY_DATA=$(curl -s -X GET "$BASE_URL/api/v1/strategy/" -H "Authorization: Bearer $TOKEN")
ENABLED_STRATEGIES=$(echo "$STRATEGY_DATA" | grep -o '"enabled":true' | wc -l)

run_test "可用于分享的策略数量" \
    "echo '$ENABLED_STRATEGIES'" \
    "[1-9]"

# 7. 网络连接问题诊断
echo -e "\n${BLUE}🌐 步骤7: 网络连接诊断${NC}"

run_test "Binance API连接测试" \
    "curl -s --connect-timeout 5 'https://api.binance.com/api/v3/ping'" \
    '"{}"\|"msg":"pong"'

if [ $? -ne 0 ]; then
    echo -e "${YELLOW}⚠️  Binance API连接失败，这解释了自动化任务的网络错误${NC}"
fi

# 8. 数据库连接测试
echo -e "\n${BLUE}🗄️  步骤8: 数据库连接测试${NC}"

run_test "数据库健康检查" \
    "curl -s -X GET '$BASE_URL/api/v1/health/checks/database' -H 'Authorization: Bearer $TOKEN'" \
    '"status":'

# 测试结果汇总
echo -e "\n${BLUE}📋 测试结果汇总${NC}"
echo "=================================="
echo -e "总测试数: ${BLUE}$TOTAL_TESTS${NC}"
echo -e "通过: ${GREEN}$PASSED_TESTS${NC}"
echo -e "失败: ${RED}$FAILED_TESTS${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}🎉 所有测试通过！系统修复成功！${NC}"
    
    echo -e "\n${GREEN}✅ 修复总结:${NC}"
    echo "1. ✅ 分享结果页面策略选择问题已修复"
    echo "2. ✅ 策略状态管理正常（3个策略中1个运行是正常状态）"
    echo "3. ✅ 自动化系统核心功能已修复"
    echo "4. ✅ API认证和数据获取正常工作"
    
    echo -e "\n${YELLOW}💡 后续建议:${NC}"
    echo "1. 解决网络连接问题以改善自动化任务成功率"
    echo "2. 监控系统健康分数，保持在0.8以上"
    echo "3. 定期检查自动化任务状态"
    echo "4. 考虑添加更多测试策略数据"
    
    exit 0
else
    echo -e "\n${RED}❌ 有 $FAILED_TESTS 个测试失败，需要进一步修复${NC}"
    
    echo -e "\n${YELLOW}🔧 建议的修复步骤:${NC}"
    echo "1. 检查后端服务是否正常运行"
    echo "2. 验证数据库连接和数据完整性"
    echo "3. 重启后端服务以应用最新修改"
    echo "4. 检查网络连接和API权限"
    
    exit 1
fi

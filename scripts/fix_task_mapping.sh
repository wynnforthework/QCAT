#!/bin/bash

# 修复自动化任务ID映射问题
# 解决数字ID与字符串ID不匹配的问题

echo "🔧 开始修复自动化任务ID映射问题..."

# API配置
BASE_URL="http://localhost:8082"
USERNAME="admin"
PASSWORD="admin123"

# 1. 登录获取token
echo "📝 正在登录..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}")

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "❌ 登录失败"
    echo "Response: $LOGIN_RESPONSE"
    exit 1
fi

echo "✅ 登录成功"

# 2. 获取自动化状态
echo "📊 获取自动化任务状态..."
AUTOMATION_RESPONSE=$(curl -s -X GET "$BASE_URL/api/v1/automation/status" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json")

echo "📋 当前自动化系统状态（前5个任务）:"
echo "$AUTOMATION_RESPONSE" | grep -o '"name":"[^"]*"' | head -5

# 3. 尝试使用字符串ID启用任务
echo ""
echo "🛠️  尝试使用字符串ID启用关键自动化任务..."

# 定义字符串任务ID（从后端日志中获取）
STRING_TASK_IDS=("strategy_optimization" "risk_monitoring" "system_health" "multi_exchange_redundancy" "data_cleaning")
TASK_NAMES=("策略参数自动优化" "风险监控" "系统健康监控" "多交易所冗余" "数据清洗与校正")

for i in "${!STRING_TASK_IDS[@]}"; do
    TASK_ID="${STRING_TASK_IDS[$i]}"
    TASK_NAME="${TASK_NAMES[$i]}"
    
    echo "🔄 启用任务: $TASK_NAME (ID: $TASK_ID)"
    
    TOGGLE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/automation/$TASK_ID/toggle" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d '{"enabled": true}')
    
    if echo "$TOGGLE_RESPONSE" | grep -q '"success":true'; then
        echo "✅ 启用成功"
    else
        echo "❌ 启用失败: $TOGGLE_RESPONSE"
    fi
    
    sleep 0.5  # 避免请求过快
done

# 4. 验证结果
echo ""
echo "📊 验证修复结果..."
sleep 2

UPDATED_RESPONSE=$(curl -s -X GET "$BASE_URL/api/v1/automation/status" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json")

ENABLED_COUNT=$(echo "$UPDATED_RESPONSE" | grep -o '"enabled":true' | wc -l)
TOTAL_COUNT=$(echo "$UPDATED_RESPONSE" | grep -o '"id":"[^"]*"' | wc -l)

echo "✅ 修复完成!"
echo "📊 已启用任务数量: $ENABLED_COUNT/$TOTAL_COUNT"

# 5. 显示任务状态详情
echo ""
echo "📋 任务状态详情:"
echo "$UPDATED_RESPONSE" | grep -E '"name"|"enabled"|"status"' | head -15

echo ""
echo "💡 修复建议:"
echo "   1. 如果仍然显示0个启用任务，说明需要修复ID映射逻辑"
echo "   2. 检查后端日志中的实际任务ID"
echo "   3. 确保API路由正确处理任务ID"
echo "   4. 考虑统一使用字符串ID或数字ID"

echo ""
echo "🎯 任务ID映射修复尝试完成!"

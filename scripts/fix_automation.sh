#!/bin/bash

# 修复自动化系统脚本
# 用于启用关键的自动化任务

echo "🔧 开始修复自动化系统状态..."

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

echo "📋 当前自动化系统状态:"
echo "$AUTOMATION_RESPONSE" | grep -o '"name":"[^"]*"' | head -5

# 3. 启用关键任务
echo ""
echo "🛠️  开始启用关键自动化任务..."

# 定义关键任务ID
CRITICAL_TASKS=("1" "7" "11" "19" "16")
TASK_NAMES=("策略自动优化" "风险实时监控" "熔断机制" "系统健康检查" "市场数据采集")

for i in "${!CRITICAL_TASKS[@]}"; do
    TASK_ID="${CRITICAL_TASKS[$i]}"
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

echo ""
echo "💡 修复建议:"
echo "   1. 检查网络连接，确保能访问Binance API"
echo "   2. 监控系统健康分数，保持在0.8以上"
echo "   3. 定期检查自动化任务状态"
echo "   4. 对于持续失败的任务，检查配置和权限"

echo ""
echo "🎯 修复完成! 关键自动化任务已启用"

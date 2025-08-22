#!/bin/bash

# 修复策略数据脚本
# 用于解决分享结果页面策略选择问题

echo "🔧 开始修复策略数据..."

# 检查PostgreSQL是否运行
if ! pgrep -x "postgres" > /dev/null; then
    echo "❌ PostgreSQL 未运行，请先启动 PostgreSQL"
    exit 1
fi

# 数据库连接参数
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=${DB_NAME:-qcat}
DB_USER=${DB_USER:-postgres}

echo "📊 连接数据库: $DB_HOST:$DB_PORT/$DB_NAME"

# 运行修复脚本
if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f scripts/fix_strategies_data.sql; then
    echo "✅ 策略数据修复完成！"
    echo ""
    echo "📋 修复内容："
    echo "   - 添加了 is_running 和 enabled 字段"
    echo "   - 插入了 5 个测试策略"
    echo "   - 其中 2 个策略处于运行状态"
    echo "   - 3 个策略处于已启用但停止状态"
    echo "   - 1 个策略处于禁用状态"
    echo ""
    echo "🎯 现在可以测试分享结果页面的策略选择功能了"
else
    echo "❌ 策略数据修复失败"
    echo "💡 请检查数据库连接和权限"
    exit 1
fi

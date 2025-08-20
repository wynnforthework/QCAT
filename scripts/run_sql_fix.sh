#!/bin/bash

echo "🔧 运行SQL修复脚本"
echo "=================="

# 设置数据库连接参数
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_NAME=${DB_NAME:-qcat}

# 提示用户输入密码（如果没有设置环境变量）
if [ -z "$DB_PASSWORD" ]; then
    echo "请输入PostgreSQL密码 (通常是postgres或空):"
    read -s DB_PASSWORD
    export PGPASSWORD=$DB_PASSWORD
fi

echo ""
echo "正在连接数据库并运行修复脚本..."
echo ""

# 运行SQL脚本
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f scripts/fix_admin_user.sql

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ SQL修复脚本执行成功！"
    echo ""
    echo "默认用户账户已创建/更新："
    echo "- 用户名: admin, 密码: admin123, 角色: admin"
    echo "- 用户名: testuser, 密码: admin123, 角色: user"
    echo "- 用户名: demo, 密码: demo123, 角色: user"
    echo ""
    echo "现在可以测试登录："
    echo "curl -X POST http://localhost:8082/api/v1/auth/login \\"
    echo "  -H \"Content-Type: application/json\" \\"
    echo "  -d '{\"username\": \"admin\", \"password\": \"admin123\"}'"
else
    echo ""
    echo "❌ SQL脚本执行失败"
    echo "请检查："
    echo "1. PostgreSQL服务是否运行"
    echo "2. 数据库连接参数是否正确"
    echo "3. 用户是否有足够的权限"
fi

echo ""

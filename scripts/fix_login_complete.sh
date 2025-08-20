#!/bin/bash

echo "🔧 QCAT 登录问题完整修复工具"
echo "=============================="

# 设置数据库密码环境变量
echo "📝 设置环境变量..."
export DATABASE_PASSWORD=${DATABASE_PASSWORD:-postgres}

echo "✅ 数据库密码环境变量已设置: $DATABASE_PASSWORD"

# 检查是否在项目根目录
if [ ! -f "go.mod" ]; then
    echo "❌ 错误: 请在项目根目录运行此脚本"
    exit 1
fi

# 检查配置文件是否存在
if [ ! -f "configs/config.yaml" ]; then
    echo "❌ 错误: 配置文件 configs/config.yaml 不存在"
    exit 1
fi

echo "✅ 找到配置文件"

# 运行Go修复脚本
echo ""
echo "🔧 运行用户修复脚本..."
go run scripts/fix_user_via_app.go

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ 用户修复完成！"
    echo ""
    echo "📋 默认用户账户:"
    echo "- 用户名: admin, 密码: admin123, 角色: admin"
    echo "- 用户名: testuser, 密码: admin123, 角色: user"
    echo "- 用户名: demo, 密码: demo123, 角色: user"
    echo ""
    echo "🧪 现在测试登录..."
    echo ""
    
    # 测试登录
    echo "发送登录请求..."
    curl -X POST http://localhost:8082/api/v1/auth/login \
         -H "Content-Type: application/json" \
         -d '{"username": "admin", "password": "admin123"}'
    
    echo ""
    echo ""
    echo "🎉 修复完成！"
    echo ""
    echo "📝 下一步:"
    echo "1. 如果API服务器未运行，启动它: go run cmd/qcat/main.go"
    echo "2. 如果登录成功，你会看到包含access_token的JSON响应"
    echo "3. 使用access_token访问dashboard:"
    echo "   curl -H \"Authorization: Bearer YOUR_TOKEN\" http://localhost:8082/api/v1/dashboard"
    
else
    echo ""
    echo "❌ 用户修复失败"
    echo ""
    echo "🔍 可能的问题:"
    echo "1. 数据库服务未运行"
    echo "2. 数据库密码不正确"
    echo "3. 数据库连接配置错误"
    echo ""
    echo "💡 尝试以下解决方案:"
    echo "1. 检查PostgreSQL服务是否运行"
    echo "2. 尝试不同的数据库密码:"
    echo "   export DATABASE_PASSWORD=postgres"
    echo "   export DATABASE_PASSWORD=password"
    echo "   export DATABASE_PASSWORD=123456"
    echo "   export DATABASE_PASSWORD=''"
    echo "3. 检查configs/config.yaml中的数据库配置"
fi

echo ""

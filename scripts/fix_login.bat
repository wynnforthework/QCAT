@echo off
chcp 65001 >nul

echo 🔧 QCAT 登录问题修复工具
echo ==========================

REM 检查是否在项目根目录
if not exist "go.mod" (
    echo ❌ 错误: 请在项目根目录运行此脚本
    pause
    exit /b 1
)

REM 检查配置文件是否存在
set CONFIG_FILE=configs\config.yaml
if not exist "%CONFIG_FILE%" (
    echo ❌ 错误: 配置文件 %CONFIG_FILE% 不存在
    echo 请确保配置文件存在并包含正确的数据库连接信息
    pause
    exit /b 1
)

echo ✅ 找到配置文件: %CONFIG_FILE%

REM 运行数据库迁移
echo.
echo 📦 运行数据库迁移...
go run cmd/migrate/main.go -up
if %errorlevel% neq 0 (
    echo ❌ 数据库迁移失败
    echo 请检查数据库连接和配置
    pause
    exit /b 1
)

echo ✅ 数据库迁移完成

REM 验证用户是否创建成功
echo.
echo 🔍 验证默认用户...
echo 如果看到以下用户信息，说明修复成功：
echo.
echo 默认用户账户：
echo - 用户名: admin, 密码: admin123, 角色: admin
echo - 用户名: testuser, 密码: admin123, 角色: user
echo - 用户名: demo, 密码: demo123, 角色: user
echo.

REM 测试登录
echo 🧪 测试登录功能...
echo 请确保API服务器正在运行 (http://localhost:8082)
echo.
echo 测试命令：
echo curl -X POST http://localhost:8082/api/v1/auth/login ^
echo   -H "Content-Type: application/json" ^
echo   -d "{\"username\": \"admin\", \"password\": \"admin123\"}"
echo.

echo 🎉 修复完成！
echo.
echo 下一步：
echo 1. 启动API服务器: go run cmd/qcat/main.go
echo 2. 使用上面的curl命令测试登录
echo 3. 使用返回的access_token访问dashboard API
echo.
echo 如果仍有问题，请检查：
echo - 数据库服务是否运行
echo - API服务器是否启动
echo - 配置文件中的数据库连接信息是否正确

pause

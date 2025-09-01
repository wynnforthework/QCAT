@echo off
echo 🚀 启动 QCAT 开发环境...

echo 🔍 检查依赖...
go version >nul 2>&1
if errorlevel 1 (
    echo ❌ Go 未安装
    exit /b 1
)

node --version >nul 2>&1
if errorlevel 1 (
    echo ❌ Node.js 未安装
    exit /b 1
)

echo ✅ 依赖检查完成

echo 🔧 启动优化器服务...
start "QCAT Optimizer" cmd /k "go run ./cmd/optimizer"

echo 🌐 启动 API 服务...
start "QCAT API" cmd /k "go run ./cmd/qcat"

echo 💻 启动前端服务...
start "QCAT Frontend" cmd /k "cd frontend && npm run dev"

echo ✅ 所有服务已启动
echo 📝 服务地址:
echo    - API: http://localhost:8082
echo    - 前端: http://localhost:3000
echo    - 优化器: http://localhost:8081
echo.
echo 按任意键关闭此窗口...
pause >nul

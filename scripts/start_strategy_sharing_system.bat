@echo off
chcp 65001 >nul
echo ========================================
echo QCAT 策略结果分享系统启动脚本
echo ========================================
echo.

:: 检查Go是否安装
echo [1/4] 检查Go环境...
go version >nul 2>&1
if errorlevel 1 (
    echo ❌ Go未安装或未添加到PATH
    echo 请先安装Go: https://golang.org/dl/
    pause
    exit /b 1
)
echo ✅ Go环境正常

:: 检查Node.js是否安装
echo [2/4] 检查Node.js环境...
node --version >nul 2>&1
if errorlevel 1 (
    echo ❌ Node.js未安装或未添加到PATH
    echo 请先安装Node.js: https://nodejs.org/
    pause
    exit /b 1
)
echo ✅ Node.js环境正常

:: 检查Python是否安装
echo [3/4] 检查Python环境...
python --version >nul 2>&1
if errorlevel 1 (
    echo ❌ Python未安装或未添加到PATH
    echo 请先安装Python: https://python.org/
    pause
    exit /b 1
)
echo ✅ Python环境正常

:: 创建必要的目录
echo [4/4] 创建必要目录...
if not exist "data" mkdir data
if not exist "data\shared_results" mkdir data\shared_results
if not exist "logs" mkdir logs
echo ✅ 目录创建完成

echo.
echo ========================================
echo 启动系统组件...
echo ========================================

:: 启动后端服务
echo 🚀 启动后端服务 (端口: 8080)...
start "QCAT Backend" cmd /k "cd /d %~dp0.. && go run cmd/optimizer/main.go"

:: 等待后端启动
echo ⏳ 等待后端服务启动...
timeout /t 5 /nobreak >nul

:: 检查后端是否启动成功
echo 🔍 检查后端服务状态...
curl -s http://localhost:8080/health >nul 2>&1
if errorlevel 1 (
    echo ⚠️  后端服务可能未完全启动，请稍等...
    timeout /t 3 /nobreak >nul
)

:: 启动前端服务
echo 🚀 启动前端服务 (端口: 3000)...
cd /d "%~dp0..\frontend"
if not exist "node_modules" (
    echo 📦 安装前端依赖...
    npm install
)
start "QCAT Frontend" cmd /k "npm run dev"

:: 等待前端启动
echo ⏳ 等待前端服务启动...
timeout /t 10 /nobreak >nul

echo.
echo ========================================
echo 🎉 系统启动完成！
echo ========================================
echo.
echo 📱 前端地址: http://localhost:3000
echo 🔧 后端地址: http://localhost:8080
echo.
echo 📋 可用页面:
echo    - 首页: http://localhost:3000
echo    - 分享结果: http://localhost:3000/share-result
echo    - 浏览结果: http://localhost:3000/shared-results
echo.
echo 🧪 运行测试:
echo    python scripts/test_strategy_sharing.py
echo.
echo ⚠️  按任意键关闭所有服务...
pause >nul

:: 关闭所有相关进程
echo 🛑 关闭服务...
taskkill /f /im "go.exe" >nul 2>&1
taskkill /f /im "node.exe" >nul 2>&1
taskkill /f /im "cmd.exe" >nul 2>&1

echo ✅ 服务已关闭
pause

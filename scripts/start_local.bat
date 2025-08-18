@echo off
REM QCAT 本地开发环境一键启动脚本 (Windows版本)
REM 支持Windows 10/11

setlocal enabledelayedexpansion

REM 设置颜色代码
set "RED=[91m"
set "GREEN=[92m"
set "YELLOW=[93m"
set "BLUE=[94m"
set "CYAN=[96m"
set "NC=[0m"

REM 配置
set "DB_HOST=localhost"
set "DB_PORT=5432"
set "DB_USER=postgres"
set "DB_PASSWORD=123"
set "DB_NAME=qcat"
set "REDIS_HOST=localhost"
set "REDIS_PORT=6379"
set "JWT_SECRET=f31e8818003142e8ad518726cda4af31"

echo %CYAN%==========================================%NC%
echo %CYAN%    QCAT 本地开发环境一键启动脚本%NC%
echo %CYAN%==========================================%NC%

REM 检查依赖
echo %BLUE%[INFO]%NC% 检查系统依赖...

REM 检查Go
go version >nul 2>&1
if errorlevel 1 (
    echo %RED%[ERROR]%NC% Go 未安装，请先安装 Go 1.23+
    echo 请访问 https://golang.org/dl/ 下载安装Go
    pause
    exit /b 1
)

REM 检查Node.js
node --version >nul 2>&1
if errorlevel 1 (
    echo %RED%[ERROR]%NC% Node.js 未安装，请先安装 Node.js 20+
    echo 请访问 https://nodejs.org/ 下载安装Node.js
    pause
    exit /b 1
)

REM 检查npm
npm --version >nul 2>&1
if errorlevel 1 (
    echo %RED%[ERROR]%NC% npm 未安装
    pause
    exit /b 1
)

echo %GREEN%[SUCCESS]%NC% 依赖检查完成

REM 安装依赖
echo %BLUE%[INFO]%NC% 安装项目依赖...

REM 安装Go依赖
echo 正在下载Go模块依赖...
go mod download
go mod tidy

REM 安装前端依赖
echo 正在安装前端依赖...
cd frontend
call npm install
cd ..

echo %GREEN%[SUCCESS]%NC% 依赖安装完成

REM 配置环境
echo %BLUE%[INFO]%NC% 配置环境...

REM 复制配置文件
if not exist "configs\config.yaml" (
    if exist "configs\config.yaml.example" (
        copy "configs\config.yaml.example" "configs\config.yaml" >nul
        echo 已复制配置文件
    )
)

REM 创建日志目录
if not exist "logs" mkdir logs

echo %GREEN%[SUCCESS]%NC% 环境配置完成

REM 启动数据库服务
echo %BLUE%[INFO]%NC% 启动数据库服务...

REM 检查Docker Compose
docker-compose --version >nul 2>&1
if errorlevel 1 (
    echo %YELLOW%[WARNING]%NC% Docker Compose不可用，请手动启动PostgreSQL和Redis
    set /p "confirm=数据库服务已启动? (y/N): "
    if /i not "!confirm!"=="y" (
        echo %RED%[ERROR]%NC% 请先启动数据库服务
        pause
        exit /b 1
    )
) else (
    if exist "deploy\docker-compose.prod.yml" (
        docker-compose -f deploy\docker-compose.prod.yml up -d postgres redis
        timeout /t 10 /nobreak >nul
        echo %GREEN%[SUCCESS]%NC% 数据库服务启动完成
    ) else (
        echo %YELLOW%[WARNING]%NC% docker-compose.prod.yml 文件不存在
        set /p "confirm=数据库服务已启动? (y/N): "
        if /i not "!confirm!"=="y" (
            echo %RED%[ERROR]%NC% 请先启动数据库服务
            pause
            exit /b 1
        )
    )
)

REM 初始化数据库
echo %BLUE%[INFO]%NC% 初始化数据库...
go run cmd\qcat\main.go -migrate
echo %GREEN%[SUCCESS]%NC% 数据库初始化完成

REM 启动服务
echo %BLUE%[INFO]%NC% 启动服务...

REM 启动后端
echo 启动后端服务 (端口: 8082)...
start "QCAT Backend" cmd /k "go run cmd\qcat\main.go"

REM 启动优化器
echo 启动优化器服务 (端口: 8081)...
start "QCAT Optimizer" cmd /k "go run cmd\optimizer\main.go"

REM 启动前端
echo 启动前端服务 (端口: 3000)...
start "QCAT Frontend" cmd /k "cd frontend && npm run dev"

timeout /t 10 /nobreak >nul
echo %GREEN%[SUCCESS]%NC% 所有服务启动完成

REM 显示状态
echo.
echo ==========================================
echo            QCAT 服务状态
echo ==========================================

REM 检查后端
curl -s -f http://localhost:8082/health >nul 2>&1
if errorlevel 1 (
    echo ❌ 后端API服务 (端口: 8082) - %RED%未运行%NC%
) else (
    echo ✅ 后端API服务 (端口: 8082) - %GREEN%运行中%NC%
)

REM 检查优化器
curl -s -f http://localhost:8081/health >nul 2>&1
if errorlevel 1 (
    echo ⚠️  优化器服务 (端口: 8081) - %YELLOW%状态未知%NC%
) else (
    echo ✅ 优化器服务 (端口: 8081) - %GREEN%运行中%NC%
)

REM 检查前端
curl -s -f http://localhost:3000 >nul 2>&1
if errorlevel 1 (
    echo ⚠️  前端服务 (端口: 3000) - %YELLOW%状态未知%NC%
) else (
    echo ✅ 前端服务 (端口: 3000) - %GREEN%运行中%NC%
)

echo ==========================================
echo.
echo 🌐 访问地址:
echo    前端界面: http://localhost:3000
echo    后端API:  http://localhost:8082
echo    优化器:   http://localhost:8081
echo.
echo 🛑 停止服务: 关闭对应的命令行窗口
echo.

echo %GREEN%[SUCCESS]%NC% 所有服务已启动，按任意键退出...
pause >nul

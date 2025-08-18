@echo off
chcp 65001 >nul
REM QCAT 本地开发环境一键启动脚本 (Windows版本)
REM 支持Windows 10/11

setlocal enabledelayedexpansion

echo ==========================================
echo    QCAT 本地开发环境一键启动脚本
echo ==========================================

REM 检查依赖
echo [INFO] 检查系统依赖...

REM 检查Go
echo 检查Go环境...
go version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Go 未安装，请先安装 Go 1.23+
    echo.
    echo 安装步骤:
    echo 1. 访问 https://golang.org/dl/
    echo 2. 下载Windows版本的Go安装包
    echo 3. 运行安装程序并按照提示完成安装
    echo 4. 重启命令行窗口
    echo.
    pause
    exit /b 1
) else (
    go version
    echo [SUCCESS] Go环境检查通过
)

REM 检查Node.js
echo.
echo 检查Node.js环境...
node --version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Node.js 未安装，请先安装 Node.js 20+
    echo.
    echo 安装步骤:
    echo 1. 访问 https://nodejs.org/
    echo 2. 下载LTS版本的Node.js安装包
    echo 3. 运行安装程序并按照提示完成安装
    echo 4. 重启命令行窗口
    echo.
    pause
    exit /b 1
) else (
    node --version
    echo [SUCCESS] Node.js环境检查通过
)

REM 检查npm
echo.
echo 检查npm环境...
npm --version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] npm 未安装或配置有问题
    echo 请重新安装Node.js，npm会随Node.js一起安装
    pause
    exit /b 1
) else (
    npm --version
    echo [SUCCESS] npm环境检查通过
)

echo.
echo [SUCCESS] 所有依赖检查完成
echo.

REM 询问是否继续
set /p "continue=是否继续安装项目依赖并启动服务? (Y/n): "
if /i "!continue!"=="n" (
    echo 已取消操作
    pause
    exit /b 0
)

REM 安装依赖
echo.
echo [INFO] 安装项目依赖...

REM 安装Go依赖
echo 正在下载Go模块依赖...
go mod download
if errorlevel 1 (
    echo [ERROR] Go模块下载失败
    pause
    exit /b 1
)

go mod tidy
if errorlevel 1 (
    echo [ERROR] Go模块整理失败
    pause
    exit /b 1
)

echo [SUCCESS] Go依赖安装完成

REM 安装前端依赖
echo.
echo 正在安装前端依赖...
if not exist "frontend" (
    echo [ERROR] frontend目录不存在
    pause
    exit /b 1
)

cd frontend
call npm install
if errorlevel 1 (
    echo [ERROR] 前端依赖安装失败
    cd ..
    pause
    exit /b 1
)
cd ..

echo [SUCCESS] 前端依赖安装完成

REM 配置环境
echo.
echo [INFO] 配置环境...

REM 复制配置文件
if not exist "configs\config.yaml" (
    if exist "configs\config.yaml.example" (
        copy "configs\config.yaml.example" "configs\config.yaml" >nul
        echo 已复制配置文件 configs\config.yaml
    ) else (
        echo [WARNING] 未找到配置文件模板
    )
)

REM 创建日志目录
if not exist "logs" mkdir logs

echo [SUCCESS] 环境配置完成

REM 启动数据库服务
echo.
echo [INFO] 启动数据库服务...

REM 检查Docker Compose
docker-compose --version >nul 2>&1
if errorlevel 1 (
    echo [WARNING] Docker Compose不可用
    echo.
    echo 请手动启动PostgreSQL和Redis服务:
    echo 1. 安装Docker Desktop: https://www.docker.com/products/docker-desktop/
    echo 2. 或者手动安装PostgreSQL和Redis
    echo.
    set /p "confirm=数据库服务已启动? (y/N): "
    if /i not "!confirm!"=="y" (
        echo [ERROR] 请先启动数据库服务
        pause
        exit /b 1
    )
) else (
    if exist "deploy\docker-compose.prod.yml" (
        echo 使用Docker Compose启动数据库服务...
        docker-compose -f deploy\docker-compose.prod.yml up -d postgres redis
        timeout /t 10 /nobreak >nul
        echo [SUCCESS] 数据库服务启动完成
    ) else (
        echo [WARNING] docker-compose.prod.yml 文件不存在
        set /p "confirm=数据库服务已启动? (y/N): "
        if /i not "!confirm!"=="y" (
            echo [ERROR] 请先启动数据库服务
            pause
            exit /b 1
        )
    )
)

REM 初始化数据库
echo.
echo [INFO] 初始化数据库...
if not exist "cmd\qcat\main.go" (
    echo [ERROR] 主程序文件不存在: cmd\qcat\main.go
    pause
    exit /b 1
)

go run cmd\qcat\main.go -migrate
if errorlevel 1 (
    echo [ERROR] 数据库初始化失败
    pause
    exit /b 1
)

echo [SUCCESS] 数据库初始化完成

REM 启动服务
echo.
echo [INFO] 启动服务...

REM 启动后端
echo 启动后端服务 (端口: 8082)...
start "QCAT Backend" cmd /k "go run cmd\qcat\main.go"

REM 启动优化器
echo 启动优化器服务 (端口: 8081)...
if exist "cmd\optimizer\main.go" (
    start "QCAT Optimizer" cmd /k "go run cmd\optimizer\main.go"
) else (
    echo [WARNING] 优化器主程序不存在，跳过优化器启动
)

REM 启动前端
echo 启动前端服务 (端口: 3000)...
start "QCAT Frontend" cmd /k "cd frontend && npm run dev"

timeout /t 10 /nobreak >nul
echo [SUCCESS] 所有服务启动完成

REM 显示状态
echo.
echo ==========================================
echo            QCAT 服务状态
echo ==========================================

REM 检查后端
curl -s -f http://localhost:8082/health >nul 2>&1
if errorlevel 1 (
    echo ❌ 后端API服务 (端口: 8082) - 未运行
) else (
    echo ✅ 后端API服务 (端口: 8082) - 运行中
)

REM 检查优化器
curl -s -f http://localhost:8081/health >nul 2>&1
if errorlevel 1 (
    echo ⚠️  优化器服务 (端口: 8081) - 状态未知
) else (
    echo ✅ 优化器服务 (端口: 8081) - 运行中
)

REM 检查前端
curl -s -f http://localhost:3000 >nul 2>&1
if errorlevel 1 (
    echo ⚠️  前端服务 (端口: 3000) - 状态未知
) else (
    echo ✅ 前端服务 (端口: 3000) - 运行中
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

echo [SUCCESS] 所有服务已启动，按任意键退出...
pause >nul

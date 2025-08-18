@echo off
REM QCAT Local Development Environment Startup Script (Windows)
REM Simple version without encoding issues

echo ==========================================
echo    QCAT Local Development Environment
echo ==========================================

REM Check dependencies
echo [INFO] Checking system dependencies...

REM Check Go
echo Checking Go environment...
go version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Go is not installed. Please install Go 1.23+
    echo Visit: https://golang.org/dl/
    pause
    exit /b 1
) else (
    go version
    echo [SUCCESS] Go environment check passed
)

REM Check Node.js
echo.
echo Checking Node.js environment...
node --version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Node.js is not installed. Please install Node.js 20+
    echo Visit: https://nodejs.org/
    pause
    exit /b 1
) else (
    node --version
    echo [SUCCESS] Node.js environment check passed
)

REM Check npm
echo.
echo Checking npm environment...
npm --version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] npm is not installed or configured
    echo Please reinstall Node.js, npm comes with Node.js
    pause
    exit /b 1
) else (
    npm --version
    echo [SUCCESS] npm environment check passed
)

echo.
echo [SUCCESS] All dependency checks completed
echo.

REM Ask to continue
set /p "continue=Continue to install project dependencies and start services? (Y/n): "
if /i "!continue!"=="n" (
    echo Operation cancelled
    pause
    exit /b 0
)

REM Install dependencies
echo.
echo [INFO] Installing project dependencies...

REM Install Go dependencies
echo Downloading Go module dependencies...
go mod download
if errorlevel 1 (
    echo [ERROR] Go module download failed
    pause
    exit /b 1
)

go mod tidy
if errorlevel 1 (
    echo [ERROR] Go module tidy failed
    pause
    exit /b 1
)

echo [SUCCESS] Go dependencies installed

REM Install frontend dependencies
echo.
echo Installing frontend dependencies...
if not exist "frontend" (
    echo [ERROR] frontend directory does not exist
    pause
    exit /b 1
)

cd frontend
call npm install
if errorlevel 1 (
    echo [ERROR] Frontend dependency installation failed
    cd ..
    pause
    exit /b 1
)
cd ..

echo [SUCCESS] Frontend dependencies installed

REM Configure environment
echo.
echo [INFO] Configuring environment...

REM Copy config file
if not exist "configs\config.yaml" (
    if exist "configs\config.yaml.example" (
        copy "configs\config.yaml.example" "configs\config.yaml" >nul
        echo Copied config file: configs\config.yaml
    ) else (
        echo [WARNING] Config template not found
    )
)

REM Create logs directory
if not exist "logs" mkdir logs

echo [SUCCESS] Environment configuration completed

REM Start database services
echo.
echo [INFO] Starting database services...

REM Check Docker Compose
docker-compose --version >nul 2>&1
if errorlevel 1 (
    echo [WARNING] Docker Compose not available
    echo.
    echo Please manually start PostgreSQL and Redis services:
    echo 1. Install Docker Desktop: https://www.docker.com/products/docker-desktop/
    echo 2. Or manually install PostgreSQL and Redis
    echo.
    set /p "confirm=Database services started? (y/N): "
    if /i not "!confirm!"=="y" (
        echo [ERROR] Please start database services first
        pause
        exit /b 1
    )
) else (
    if exist "deploy\docker-compose.prod.yml" (
        echo Starting database services with Docker Compose...
        docker-compose -f deploy\docker-compose.prod.yml up -d postgres redis
        timeout /t 10 /nobreak >nul
        echo [SUCCESS] Database services started
    ) else (
        echo [WARNING] docker-compose.prod.yml file not found
        set /p "confirm=Database services started? (y/N): "
        if /i not "!confirm!"=="y" (
            echo [ERROR] Please start database services first
            pause
            exit /b 1
        )
    )
)

REM Initialize database
echo.
echo [INFO] Initializing database...
if not exist "cmd\qcat\main.go" (
    echo [ERROR] Main program file not found: cmd\qcat\main.go
    pause
    exit /b 1
)

go run cmd\qcat\main.go -migrate
if errorlevel 1 (
    echo [ERROR] Database initialization failed
    pause
    exit /b 1
)

echo [SUCCESS] Database initialization completed

REM Start services
echo.
echo [INFO] Starting services...

REM Start backend
echo Starting backend service (port: 8082)...
start "QCAT Backend" cmd /k "go run cmd\qcat\main.go"

REM Start optimizer
echo Starting optimizer service (port: 8081)...
if exist "cmd\optimizer\main.go" (
    start "QCAT Optimizer" cmd /k "go run cmd\optimizer\main.go"
) else (
    echo [WARNING] Optimizer main program not found, skipping optimizer startup
)

REM Start frontend
echo Starting frontend service (port: 3000)...
start "QCAT Frontend" cmd /k "cd frontend && npm run dev"

timeout /t 10 /nobreak >nul
echo [SUCCESS] All services started

REM Show status
echo.
echo ==========================================
echo            QCAT Service Status
echo ==========================================

REM Check backend
curl -s -f http://localhost:8082/health >nul 2>&1
if errorlevel 1 (
    echo ❌ Backend API Service (port: 8082) - Not running
) else (
    echo ✅ Backend API Service (port: 8082) - Running
)

REM Check optimizer
curl -s -f http://localhost:8081/health >nul 2>&1
if errorlevel 1 (
    echo ⚠️  Optimizer Service (port: 8081) - Status unknown
) else (
    echo ✅ Optimizer Service (port: 8081) - Running
)

REM Check frontend
curl -s -f http://localhost:3000 >nul 2>&1
if errorlevel 1 (
    echo ⚠️  Frontend Service (port: 3000) - Status unknown
) else (
    echo ✅ Frontend Service (port: 3000) - Running
)

echo ==========================================
echo.
echo 🌐 Access URLs:
echo    Frontend: http://localhost:3000
echo    Backend API: http://localhost:8082
echo    Optimizer: http://localhost:8081
echo.
echo 🛑 Stop services: Close the corresponding command windows
echo.

echo [SUCCESS] All services started, press any key to exit...
pause >nul

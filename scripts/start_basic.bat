@echo off
echo ==========================================
echo    QCAT Basic Startup Script
echo ==========================================

echo [INFO] Checking dependencies...

echo Go version:
go version

echo.
echo Node.js version:
node --version

echo.
echo npm version:
npm --version

echo.
echo [INFO] Installing dependencies...

echo Installing Go dependencies...
go mod download
go mod tidy

echo Installing frontend dependencies...
cd frontend
call npm install
cd ..

echo.
echo [INFO] Starting services...

echo Starting backend service (port: 8082)...
start "QCAT Backend" cmd /k "go run cmd\qcat\main.go"

echo Starting frontend service (port: 3000)...
start "QCAT Frontend" cmd /k "cd frontend && npm run dev"

echo.
echo [SUCCESS] Services started!
echo.
echo Access URLs:
echo   Frontend: http://localhost:3000
echo   Backend API: http://localhost:8082
echo.
echo Press any key to exit...
pause >nul

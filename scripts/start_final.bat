@echo off
echo ==========================================
echo    QCAT Startup Script
echo ==========================================

echo Checking dependencies...
go version
node --version
npm --version

echo.
echo Installing Go dependencies...
go mod download
go mod tidy

echo Installing frontend dependencies...
cd frontend
call npm install
cd ..

echo.
echo Starting services...

echo Starting backend service...
start "QCAT Backend" cmd /k "go run cmd\qcat\main.go"

echo Starting frontend service...
start "QCAT Frontend" cmd /k "cd frontend && npm run dev"

echo.
echo Services started!
echo Frontend: http://localhost:3000
echo Backend: http://localhost:8082
echo.
echo Press any key to exit...
pause >nul

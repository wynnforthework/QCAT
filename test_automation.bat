@echo off
echo 🚀 Starting QCAT Automation System Test...

REM Check if server is running
echo 📡 Checking if QCAT server is running...
curl -s http://localhost:8082/health >nul 2>&1
if %errorlevel% equ 0 (
    echo ✅ Server is running
) else (
    echo ❌ Server is not running. Please start the server first:
    echo    go run cmd/qcat/main.go
    exit /b 1
)

REM Wait a moment for system to initialize
echo ⏳ Waiting for system initialization...
timeout /t 3 /nobreak >nul

REM Run the test
echo 🧪 Running automation system tests...
go run test_automation.go

echo ✅ Test completed!
pause
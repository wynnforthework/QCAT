@echo off
echo Building Emergency Position Closer...

REM Set build variables
set BINARY_NAME=emergency_close.exe
set BUILD_DIR=bin

REM Create build directory if it doesn't exist
if not exist %BUILD_DIR% mkdir %BUILD_DIR%

REM Build the binary
echo Building %BINARY_NAME%...
go build -o %BUILD_DIR%\%BINARY_NAME% main.go

if %ERRORLEVEL% EQU 0 (
    echo Build successful! Binary created at %BUILD_DIR%\%BINARY_NAME%
    echo.
    echo To run:
    echo   1. Set environment variables:
    echo      set BINANCE_API_KEY=your_api_key
    echo      set BINANCE_SECRET_KEY=your_secret_key
    echo   2. Run: %BUILD_DIR%\%BINARY_NAME%
) else (
    echo Build failed!
    exit /b 1
)

pause

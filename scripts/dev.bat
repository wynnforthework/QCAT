@echo off
REM QCAT Development Helper Script for Windows
REM Provides convenient shortcuts for common development tasks

setlocal enabledelayedexpansion

if "%1"=="" goto help
if "%1"=="help" goto help
if "%1"=="-h" goto help
if "%1"=="--help" goto help

REM Make scripts executable
chmod +x scripts\start_local_improved.sh 2>nul

if "%1"=="dev" goto dev
if "%1"=="dev-api" goto dev-api
if "%1"=="dev-frontend" goto dev-frontend
if "%1"=="dev-optimizer" goto dev-optimizer
if "%1"=="prod" goto prod
if "%1"=="quick-start" goto quick-start
if "%1"=="debug" goto debug
if "%1"=="start-local" goto start-local
if "%1"=="build" goto build
if "%1"=="test" goto test
if "%1"=="clean" goto clean

echo Unknown command: %1
echo.
goto help

:dev
echo Starting development mode...
bash scripts/start_local_improved.sh --dev
goto end

:dev-api
echo Starting API service only (dev mode)...
bash scripts/start_local_improved.sh --services api --dev
goto end

:dev-frontend
echo Starting frontend service only (dev mode)...
bash scripts/start_local_improved.sh --services frontend --dev
goto end

:dev-optimizer
echo Starting optimizer service only (dev mode)...
bash scripts/start_local_improved.sh --services optimizer --dev
goto end

:prod
echo Starting production mode...
bash scripts/start_local_improved.sh --production
goto end

:quick-start
echo Quick start (skip deps and build)...
bash scripts/start_local_improved.sh --skip-deps --skip-build
goto end

:debug
echo Starting with debug output...
bash scripts/start_local_improved.sh --debug
goto end

:start-local
echo Starting local development environment...
bash scripts/start_local_improved.sh
goto end

:build
echo Building Go applications...
go build -o qcat.exe ./cmd/qcat/main.go
go build -o optimizer.exe ./cmd/optimizer/main.go
echo Build complete.
goto end

:test
echo Running tests...
go test -v ./internal/...
goto end

:clean
echo Cleaning build artifacts...
del /f /q qcat.exe optimizer.exe 2>nul
echo Clean complete.
goto end

:help
echo QCAT Development Helper v1.0
echo.
echo Usage: %0 [command]
echo.
echo Available commands:
echo.
echo   Development Environment:
echo     start-local     - Start local development environment (all services)
echo     dev             - Start development mode (hot reload, debug)
echo     dev-api         - Start API service only (dev mode)
echo     dev-frontend    - Start frontend service only (dev mode)
echo     dev-optimizer   - Start optimizer service only (dev mode)
echo     prod            - Start production mode
echo     quick-start     - Quick start (skip deps and build)
echo     debug           - Start with debug output
echo.
echo   Build ^& Test:
echo     build           - Build Go applications
echo     test            - Run tests
echo     clean           - Clean build artifacts
echo.
echo   Help:
echo     help            - Show this help
echo.
echo Examples:
echo   %0 dev              # Start development mode
echo   %0 dev-frontend     # Start only frontend in dev mode
echo   %0 quick-start      # Quick start without deps/build
echo   %0 debug            # Start with debug output
echo.
goto end

:end

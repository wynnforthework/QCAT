@echo off
echo =========================================
echo  Killing processes on ports 8081, 8082, 3000
echo =========================================

for %%p in (8081 8082 3000 7897) do (
    echo Checking port %%p ...
    for /f "tokens=5" %%a in ('netstat -ano ^| findstr :%%p') do (
        echo Killing PID %%a on port %%p ...
        taskkill /PID %%a /F >nul 2>&1
    )
)

echo =========================================
echo   Done. All target ports have been cleared.
echo =========================================
pause

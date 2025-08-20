@echo off

echo QCAT Login Fix Tool
echo ===================

REM Set database password environment variable
set DATABASE_PASSWORD=postgres

echo Database password set to: %DATABASE_PASSWORD%

REM Check if we're in the project root
if not exist "go.mod" (
    echo Error: Please run this script from the project root directory
    pause
    exit /b 1
)

echo Running user fix script...
go run scripts/fix_user_via_app.go

if %errorlevel% equ 0 (
    echo.
    echo User fix completed successfully!
    echo.
    echo Default user accounts:
    echo - Username: admin, Password: admin123, Role: admin
    echo - Username: testuser, Password: admin123, Role: user
    echo - Username: demo, Password: demo123, Role: user
    echo.
    echo Testing login...
    curl -X POST http://localhost:8082/api/v1/auth/login -H "Content-Type: application/json" -d "{\"username\": \"admin\", \"password\": \"admin123\"}"
    echo.
    echo.
    echo Fix completed! If you see an access_token above, login is working.
) else (
    echo.
    echo User fix failed. Please check:
    echo 1. PostgreSQL service is running
    echo 2. Database password is correct
    echo 3. Database configuration in configs/config.yaml
)

pause

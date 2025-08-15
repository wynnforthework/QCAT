@echo off
REM Integration Test Script for QCAT
REM Tests all integrated features including auth, database, Redis, monitoring

set BASE_URL=http://localhost:8082
set API_BASE=%BASE_URL%/api/v1

echo === QCAT Integration Test Script ===
echo Base URL: %BASE_URL%
echo API Base: %API_BASE%
echo.

REM Test health check with service status
echo 1. Testing Health Check with Service Status...
curl -s "%BASE_URL%/health"
echo.
echo.

REM Test authentication endpoints
echo 2. Testing Authentication endpoints...
echo   - Login with valid credentials:
curl -s -X POST "%API_BASE%/auth/login" -H "Content-Type: application/json" -d "{\"username\":\"admin\",\"password\":\"password\"}"
echo.
echo.

echo   - Login with invalid credentials:
curl -s -X POST "%API_BASE%/auth/login" -H "Content-Type: application/json" -d "{\"username\":\"invalid\",\"password\":\"wrong\"}"
echo.
echo.

echo   - Register new user:
curl -s -X POST "%API_BASE%/auth/register" -H "Content-Type: application/json" -d "{\"username\":\"testuser\",\"password\":\"testpass\",\"email\":\"test@example.com\"}"
echo.
echo.

REM Get token for authenticated requests
echo 3. Getting authentication token...
for /f "tokens=2 delims=," %%a in ('curl -s -X POST "%API_BASE%/auth/login" -H "Content-Type: application/json" -d "{\"username\":\"admin\",\"password\":\"password\"}" ^| findstr "access_token"') do (
    set TOKEN=%%a
)
set TOKEN=%TOKEN:"=%
set TOKEN=%TOKEN: =%
echo Token: %TOKEN%
echo.

REM Test protected endpoints with authentication
echo 4. Testing Protected Endpoints with Authentication...
echo   - Strategy list (authenticated):
curl -s -H "Authorization: Bearer %TOKEN%" "%API_BASE%/strategy/"
echo.
echo.

echo   - Portfolio overview (authenticated):
curl -s -H "Authorization: Bearer %TOKEN%" "%API_BASE%/portfolio/overview"
echo.
echo.

echo   - Risk overview (authenticated):
curl -s -H "Authorization: Bearer %TOKEN%" "%API_BASE%/risk/overview"
echo.
echo.

echo   - Hotlist symbols (authenticated):
curl -s -H "Authorization: Bearer %TOKEN%" "%API_BASE%/hotlist/symbols"
echo.
echo.

echo   - System metrics (authenticated):
curl -s -H "Authorization: Bearer %TOKEN%" "%API_BASE%/metrics/system"
echo.
echo.

echo   - Audit logs (authenticated):
curl -s -H "Authorization: Bearer %TOKEN%" "%API_BASE%/audit/logs"
echo.
echo.

REM Test Prometheus metrics
echo 5. Testing Prometheus Metrics...
curl -s "%BASE_URL%/metrics"
echo.
echo.

REM Test Swagger documentation (if in development mode)
echo 6. Testing Swagger Documentation...
curl -s "%BASE_URL%/swagger/index.html" | findstr "title"
echo.
echo.

REM Test WebSocket connections
echo 7. Testing WebSocket Connections...
echo   - Market data stream (using wscat if available):
echo     wscat -c "ws://localhost:8082/ws/market/BTCUSDT"
echo.
echo   - Strategy status stream:
echo     wscat -c "ws://localhost:8082/ws/strategy/strategy_001"
echo.
echo   - Alerts stream:
echo     wscat -c "ws://localhost:8082/ws/alerts"
echo.

echo === Integration Test Complete ===
echo.
echo Note: WebSocket tests require wscat or similar WebSocket client
echo       Database and Redis tests require actual services running
echo.

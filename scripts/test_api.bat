@echo off
REM API Test Script for QCAT (Windows)
REM Tests all API endpoints to ensure they are working correctly

set BASE_URL=http://localhost:8082
set API_BASE=%BASE_URL%/api/v1

echo === QCAT API Test Script ===
echo Base URL: %BASE_URL%
echo API Base: %API_BASE%
echo.

REM Test health check
echo 1. Testing Health Check...
curl -s "%BASE_URL%/health"
echo.
echo.

REM Test Strategy endpoints
echo 2. Testing Strategy endpoints...
echo   - List strategies:
curl -s "%API_BASE%/strategy/"
echo.
echo.

echo   - Get strategy (should return 404 for non-existent):
curl -s "%API_BASE%/strategy/nonexistent"
echo.
echo.

REM Test Portfolio endpoints
echo 3. Testing Portfolio endpoints...
echo   - Portfolio overview:
curl -s "%API_BASE%/portfolio/overview"
echo.
echo.

echo   - Portfolio allocations:
curl -s "%API_BASE%/portfolio/allocations"
echo.
echo.

REM Test Risk endpoints
echo 4. Testing Risk endpoints...
echo   - Risk overview:
curl -s "%API_BASE%/risk/overview"
echo.
echo.

echo   - Risk limits:
curl -s "%API_BASE%/risk/limits"
echo.
echo.

REM Test Hotlist endpoints
echo 5. Testing Hotlist endpoints...
echo   - Hot symbols:
curl -s "%API_BASE%/hotlist/symbols"
echo.
echo.

echo   - Whitelist:
curl -s "%API_BASE%/hotlist/whitelist"
echo.
echo.

REM Test Metrics endpoints
echo 6. Testing Metrics endpoints...
echo   - System metrics:
curl -s "%API_BASE%/metrics/system"
echo.
echo.

echo   - Performance metrics:
curl -s "%API_BASE%/metrics/performance"
echo.
echo.

REM Test Audit endpoints
echo 7. Testing Audit endpoints...
echo   - Audit logs:
curl -s "%API_BASE%/audit/logs"
echo.
echo.

echo   - Decision chains:
curl -s "%API_BASE%/audit/decisions"
echo.
echo.

REM Test Optimizer endpoints
echo 8. Testing Optimizer endpoints...
echo   - Optimization tasks:
curl -s "%API_BASE%/optimizer/tasks"
echo.
echo.

REM Test POST endpoints with sample data
echo 9. Testing POST endpoints...
echo   - Create strategy:
curl -s -X POST "%API_BASE%/strategy/" -H "Content-Type: application/json" -d "{\"name\":\"Test Strategy\",\"description\":\"Test strategy for API testing\"}"
echo.
echo.

echo   - Run optimization:
curl -s -X POST "%API_BASE%/optimizer/run" -H "Content-Type: application/json" -d "{\"strategy_id\":\"test_strategy\",\"method\":\"grid\",\"objective\":\"sharpe\"}"
echo.
echo.

echo   - Portfolio rebalance:
curl -s -X POST "%API_BASE%/portfolio/rebalance" -H "Content-Type: application/json" -d "{\"mode\":\"bandit\"}"
echo.
echo.

echo === API Test Complete ===
echo All endpoints tested successfully!
pause

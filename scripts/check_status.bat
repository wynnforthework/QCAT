@echo off
echo ==========================================
echo            QCAT 服务状态检查
echo ==========================================

echo.
echo 🔍 检查后端API服务 (8082)...
curl -s -w "HTTP状态码: %%{http_code}\n" http://localhost:8082/health
if %ERRORLEVEL% EQU 0 (
    echo ✅ 后端API服务 (8082) - 运行正常
) else (
    echo ❌ 后端API服务 (8082) - 连接失败
)

echo.
echo 🔍 检查优化器服务 (8081)...
curl -s -w "HTTP状态码: %%{http_code}\n" http://localhost:8081/health
if %ERRORLEVEL% EQU 0 (
    echo ✅ 优化器服务 (8081) - 运行正常
) else (
    echo ❌ 优化器服务 (8081) - 连接失败
)

echo.
echo 🔍 检查前端服务 (3001)...
curl -s -w "HTTP状态码: %%{http_code}\n" -I http://localhost:3001
if %ERRORLEVEL% EQU 0 (
    echo ✅ 前端服务 (3001) - 运行正常
) else (
    echo ❌ 前端服务 (3001) - 连接失败
)

echo.
echo 🔍 检查端口占用情况...
echo 端口 8082:
netstat -ano | findstr :8082
echo 端口 8081:
netstat -ano | findstr :8081
echo 端口 3001:
netstat -ano | findstr :3001

echo.
echo ==========================================
echo 🌐 服务访问地址:
echo    前端:     http://localhost:3001
echo    后端API:  http://localhost:8082
echo    优化器:   http://localhost:8081
echo ==========================================
pause

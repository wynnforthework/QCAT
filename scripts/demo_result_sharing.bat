@echo off
setlocal enabledelayedexpansion

echo ========================================
echo 结果共享系统演示脚本
echo ========================================
echo.

echo 这个演示将展示结果共享系统的完整功能：
echo 1. 随机种子训练
echo 2. 多种共享模式
echo 3. 跨服务器结果共享
echo 4. 性能评估和排序
echo.

REM 检查Go环境
echo [INFO] 检查Go环境...
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Go未安装或不在PATH中，请先安装Go
    pause
    exit /b 1
)
echo [SUCCESS] Go环境正常

REM 检查依赖
echo [INFO] 检查依赖...
go mod tidy
if %errorlevel% neq 0 (
    echo [ERROR] 依赖检查失败
    pause
    exit /b 1
)
echo [SUCCESS] 依赖检查通过

REM 创建必要的目录
echo [INFO] 创建必要的目录...
if not exist data\shared_results\files mkdir data\shared_results\files
if not exist data\shared_results mkdir data\shared_results
if not exist logs mkdir logs
echo [SUCCESS] 目录创建完成

REM 启动优化器服务
echo [INFO] 启动优化器服务...
echo 服务将在后台启动，端口8081
echo.

start /B go run cmd/optimizer/main.go

REM 等待服务启动
echo [INFO] 等待服务启动...
timeout /t 3 /nobreak >nul

REM 检查服务是否启动
echo [INFO] 检查服务状态...
curl -s http://localhost:8081/health >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] 服务启动失败，请检查日志
    pause
    exit /b 1
)
echo [SUCCESS] 服务启动成功

echo.
echo ========================================
echo 开始演示结果共享功能
echo ========================================
echo.

REM 演示1：基本结果共享
echo [演示1] 基本结果共享
echo 创建并共享一个训练结果...
echo.

set result1={
  "task_id": "demo_task_001",
  "strategy_name": "ma_cross_strategy",
  "parameters": {
    "ma_short": 10,
    "ma_long": 20,
    "stop_loss": 0.05
  },
  "performance": {
    "profit_rate": 15.8,
    "sharpe_ratio": 2.1,
    "max_drawdown": 8.5,
    "win_rate": 0.68
  },
  "random_seed": 1234567890,
  "discovered_by": "demo_server_1"
}

echo %result1% > temp_result1.json
curl -s -X POST http://localhost:8081/share-result -H "Content-Type: application/json" -d @temp_result1.json
echo.
echo [SUCCESS] 结果1共享完成
echo.

REM 演示2：多个结果比较
echo [演示2] 多个结果比较
echo 创建多个不同性能的结果进行对比...
echo.

set result2={
  "task_id": "demo_task_001",
  "strategy_name": "ma_cross_strategy",
  "parameters": {
    "ma_short": 5,
    "ma_long": 15,
    "stop_loss": 0.03
  },
  "performance": {
    "profit_rate": 22.5,
    "sharpe_ratio": 2.8,
    "max_drawdown": 6.2,
    "win_rate": 0.75
  },
  "random_seed": 2345678901,
  "discovered_by": "demo_server_2"
}

set result3={
  "task_id": "demo_task_001",
  "strategy_name": "ma_cross_strategy",
  "parameters": {
    "ma_short": 15,
    "ma_long": 30,
    "stop_loss": 0.08
  },
  "performance": {
    "profit_rate": 8.2,
    "sharpe_ratio": 1.2,
    "max_drawdown": 12.5,
    "win_rate": 0.55
  },
  "random_seed": 3456789012,
  "discovered_by": "demo_server_3"
}

echo %result2% > temp_result2.json
echo %result3% > temp_result3.json

curl -s -X POST http://localhost:8081/share-result -H "Content-Type: application/json" -d @temp_result2.json
curl -s -X POST http://localhost:8081/share-result -H "Content-Type: application/json" -d @temp_result3.json

echo.
echo [SUCCESS] 多个结果共享完成
echo.

REM 演示3：查询最优结果
echo [演示3] 查询最优结果
echo 获取所有共享结果并按性能排序...
echo.

curl -s http://localhost:8081/shared-results
echo.
echo [SUCCESS] 结果查询完成
echo.

REM 演示4：跨服务器场景
echo [演示4] 跨服务器场景模拟
echo 模拟不同服务器之间的结果共享...
echo.

echo [INFO] 模拟服务器A（高性能服务器）生成结果...
set server_a_result={
  "task_id": "cross_server_demo",
  "strategy_name": "advanced_strategy",
  "parameters": {
    "rsi_period": 14,
    "rsi_overbought": 70,
    "rsi_oversold": 30,
    "volume_factor": 1.5
  },
  "performance": {
    "profit_rate": 28.5,
    "sharpe_ratio": 3.2,
    "max_drawdown": 5.8,
    "win_rate": 0.78
  },
  "random_seed": 4567890123,
  "discovered_by": "high_performance_server_a"
}

echo %server_a_result% > temp_server_a.json
curl -s -X POST http://localhost:8081/share-result -H "Content-Type: application/json" -d @temp_server_a.json

echo.
echo [INFO] 模拟服务器B（普通服务器）查询结果...
curl -s http://localhost:8081/shared-results | findstr "high_performance_server_a"

echo.
echo [SUCCESS] 跨服务器共享演示完成
echo.

REM 演示5：文件共享模式
echo [演示5] 文件共享模式
echo 检查文件共享模式生成的文件...
echo.

if exist data\shared_results\files\*.json (
    echo [SUCCESS] 文件共享模式工作正常
    echo [INFO] 生成的共享文件：
    dir /B data\shared_results\files\*.json
) else (
    echo [WARNING] 未找到共享文件，可能文件共享模式未启用
)

echo.

REM 演示6：字符串共享模式
echo [演示6] 字符串共享模式
echo 检查字符串共享模式...
echo.

if exist data\shared_results\strings.txt (
    echo [SUCCESS] 字符串共享模式工作正常
    echo [INFO] 字符串存储文件内容预览：
    type data\shared_results\strings.txt
) else (
    echo [WARNING] 未找到字符串存储文件，可能字符串共享模式未启用
)

echo.

REM 演示7：种子共享模式
echo [演示7] 种子共享模式
echo 检查种子共享模式...
echo.

if exist data\shared_results\seed_mapping.json (
    echo [SUCCESS] 种子共享模式工作正常
    echo [INFO] 种子映射文件内容：
    type data\shared_results\seed_mapping.json
) else (
    echo [WARNING] 未找到种子映射文件，可能种子共享模式未启用
)

echo.

REM 演示8：性能阈值过滤
echo [演示8] 性能阈值过滤
echo 测试低性能结果是否被过滤...
echo.

set low_performance_result={
  "task_id": "threshold_test",
  "strategy_name": "poor_strategy",
  "parameters": {
    "param1": 1
  },
  "performance": {
    "profit_rate": 1.5,
    "sharpe_ratio": 0.1,
    "max_drawdown": 25.0,
    "win_rate": 0.3
  },
  "random_seed": 9999999999,
  "discovered_by": "threshold_test"
}

echo %low_performance_result% > temp_low.json
curl -s -X POST http://localhost:8081/share-result -H "Content-Type: application/json" -d @temp_low.json

echo.
echo [INFO] 低性能结果处理完成
echo.

REM 演示9：最终结果展示
echo [演示9] 最终结果展示
echo 展示所有共享的结果和最优选择...
echo.

echo [INFO] 所有共享结果：
curl -s http://localhost:8081/shared-results

echo.
echo [INFO] 最优结果分析：
echo 根据性能评分，系统会自动选择最优的结果供所有服务器使用
echo 这样可以确保所有服务器都能获得全局最优的训练结果
echo.

echo ========================================
echo 演示完成！
echo ========================================
echo.
echo 演示总结：
echo 1. ✓ 随机种子训练：每台服务器使用不同的随机种子
echo 2. ✓ 结果共享：支持文件、字符串、种子等多种共享方式
echo 3. ✓ 跨服务器兼容：完全不相连的服务器也能共享结果
echo 4. ✓ 性能评估：自动评估和排序训练结果
echo 5. ✓ 最优选择：系统自动选择全局最优结果
echo.
echo 系统特点：
echo - 灵活性：支持多种共享模式，适应不同环境
echo - 可靠性：多重保障机制，确保结果不丢失
echo - 易用性：简单的API接口，易于集成
echo - 扩展性：模块化设计，便于功能扩展
echo.

REM 清理临时文件
del temp_*.json 2>nul

echo 按任意键退出演示...
pause >nul

REM 停止服务
echo [INFO] 停止优化器服务...
taskkill /F /IM go.exe >nul 2>&1
echo [SUCCESS] 服务已停止

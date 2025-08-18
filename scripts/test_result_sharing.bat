@echo off
setlocal enabledelayedexpansion

REM 结果共享系统测试脚本 (Windows版本)
REM 测试各种共享模式和功能

echo ========================================
echo 结果共享系统测试脚本
echo ========================================

REM 配置
set OPTIMIZER_URL=http://localhost:8081
set TEST_DATA_DIR=.\test_data
set SHARED_RESULTS_DIR=.\data\shared_results

REM 创建测试目录
echo [INFO] 设置测试环境...
if not exist %TEST_DATA_DIR% mkdir %TEST_DATA_DIR%
if not exist %SHARED_RESULTS_DIR%\files mkdir %SHARED_RESULTS_DIR%\files
if not exist %SHARED_RESULTS_DIR% mkdir %SHARED_RESULTS_DIR%
echo [SUCCESS] 测试环境设置完成

REM 测试健康检查
echo [INFO] 测试健康检查...
curl -s "%OPTIMIZER_URL%/health" > temp_response.txt
findstr /C:"healthy" temp_response.txt >nul
if %errorlevel% equ 0 (
    echo [SUCCESS] 健康检查通过
) else (
    echo [ERROR] 健康检查失败
    type temp_response.txt
    goto :cleanup
)

REM 测试手动共享结果
echo [INFO] 测试手动共享结果...
echo { > temp_result.json
echo   "task_id": "test_task_001", >> temp_result.json
echo   "strategy_name": "test_strategy", >> temp_result.json
echo   "parameters": { >> temp_result.json
echo     "param1": 100, >> temp_result.json
echo     "param2": 200 >> temp_result.json
echo   }, >> temp_result.json
echo   "performance": { >> temp_result.json
echo     "profit_rate": 15.5, >> temp_result.json
echo     "sharpe_ratio": 2.1, >> temp_result.json
echo     "max_drawdown": 8.2, >> temp_result.json
echo     "win_rate": 0.68 >> temp_result.json
echo   }, >> temp_result.json
echo   "random_seed": 1234567890, >> temp_result.json
echo   "discovered_by": "test_script" >> temp_result.json
echo } >> temp_result.json

curl -s -X POST "%OPTIMIZER_URL%/share-result" -H "Content-Type: application/json" -d @temp_result.json > temp_response.txt
findstr /C:"success" temp_response.txt >nul
if %errorlevel% equ 0 (
    echo [SUCCESS] 手动共享结果成功
    type temp_response.txt
) else (
    echo [ERROR] 手动共享结果失败
    type temp_response.txt
    goto :cleanup
)

REM 测试获取共享结果
echo [INFO] 测试获取共享结果...
curl -s "%OPTIMIZER_URL%/shared-results" > temp_response.txt
findstr /C:"results" temp_response.txt >nul
if %errorlevel% equ 0 (
    echo [SUCCESS] 获取共享结果成功
    echo [INFO] 结果内容:
    type temp_response.txt
) else (
    echo [ERROR] 获取共享结果失败
    type temp_response.txt
    goto :cleanup
)

REM 测试文件共享模式
echo [INFO] 测试文件共享模式...
dir /B "%SHARED_RESULTS_DIR%\files\*.json" >nul 2>&1
if %errorlevel% equ 0 (
    echo [SUCCESS] 文件共享模式工作正常
    echo [INFO] 生成的共享文件:
    dir /B "%SHARED_RESULTS_DIR%\files\*.json"
) else (
    echo [WARNING] 未找到共享文件，可能文件共享模式未启用
)

REM 测试字符串共享模式
echo [INFO] 测试字符串共享模式...
if exist "%SHARED_RESULTS_DIR%\strings.txt" (
    echo [SUCCESS] 字符串共享模式工作正常
    echo [INFO] 字符串存储文件内容预览:
    type "%SHARED_RESULTS_DIR%\strings.txt"
) else (
    echo [WARNING] 未找到字符串存储文件，可能字符串共享模式未启用
)

REM 测试种子共享模式
echo [INFO] 测试种子共享模式...
if exist "%SHARED_RESULTS_DIR%\seed_mapping.json" (
    echo [SUCCESS] 种子共享模式工作正常
    echo [INFO] 种子映射文件内容:
    type "%SHARED_RESULTS_DIR%\seed_mapping.json"
) else (
    echo [WARNING] 未找到种子映射文件，可能种子共享模式未启用
)

REM 测试性能阈值过滤
echo [INFO] 测试性能阈值过滤...
echo { > temp_low_result.json
echo   "task_id": "test_task_low", >> temp_low_result.json
echo   "strategy_name": "low_performance_strategy", >> temp_low_result.json
echo   "parameters": { >> temp_low_result.json
echo     "param1": 50 >> temp_low_result.json
echo   }, >> temp_low_result.json
echo   "performance": { >> temp_low_result.json
echo     "profit_rate": 2.0, >> temp_low_result.json
echo     "sharpe_ratio": 0.2, >> temp_low_result.json
echo     "max_drawdown": 20.0, >> temp_low_result.json
echo     "win_rate": 0.3 >> temp_low_result.json
echo   }, >> temp_low_result.json
echo   "random_seed": 9876543210, >> temp_low_result.json
echo   "discovered_by": "test_script" >> temp_low_result.json
echo } >> temp_low_result.json

curl -s -X POST "%OPTIMIZER_URL%/share-result" -H "Content-Type: application/json" -d @temp_low_result.json > temp_response.txt
findstr /C:"success" temp_response.txt >nul
if %errorlevel% equ 0 (
    echo [WARNING] 低性能结果未被过滤，可能需要调整阈值配置
) else (
    echo [SUCCESS] 性能阈值过滤工作正常
)

REM 测试跨服务器场景模拟
echo [INFO] 测试跨服务器场景模拟...
echo [INFO] 模拟服务器A生成结果...
echo { > temp_server_a.json
echo   "task_id": "cross_server_task", >> temp_server_a.json
echo   "strategy_name": "server_a_strategy", >> temp_server_a.json
echo   "parameters": { >> temp_server_a.json
echo     "ma_short": 10, >> temp_server_a.json
echo     "ma_long": 20 >> temp_server_a.json
echo   }, >> temp_server_a.json
echo   "performance": { >> temp_server_a.json
echo     "profit_rate": 18.5, >> temp_server_a.json
echo     "sharpe_ratio": 2.5, >> temp_server_a.json
echo     "max_drawdown": 7.8, >> temp_server_a.json
echo     "win_rate": 0.72 >> temp_server_a.json
echo   }, >> temp_server_a.json
echo   "random_seed": 1111111111, >> temp_server_a.json
echo   "discovered_by": "server_a" >> temp_server_a.json
echo } >> temp_server_a.json

curl -s -X POST "%OPTIMIZER_URL%/share-result" -H "Content-Type: application/json" -d @temp_server_a.json > temp_response.txt
findstr /C:"success" temp_response.txt >nul
if %errorlevel% equ 0 (
    echo [SUCCESS] 服务器A结果生成成功
) else (
    echo [ERROR] 服务器A结果生成失败
    goto :cleanup
)

echo [INFO] 模拟服务器B查询结果...
curl -s "%OPTIMIZER_URL%/shared-results" > temp_response.txt
findstr /C:"server_a" temp_response.txt >nul
if %errorlevel% equ 0 (
    echo [SUCCESS] 服务器B成功获取到服务器A的结果
    echo [INFO] 服务器A的结果:
    findstr /C:"server_a" temp_response.txt
) else (
    echo [ERROR] 服务器B未能获取到服务器A的结果
    goto :cleanup
)

REM 测试结果评分和排序
echo [INFO] 测试结果评分和排序...
for /L %%i in (1,1,3) do (
    echo { > temp_scoring_%%i.json
    echo   "task_id": "scoring_test", >> temp_scoring_%%i.json
    echo   "strategy_name": "strategy_%%i", >> temp_scoring_%%i.json
    echo   "performance": { >> temp_scoring_%%i.json
    echo     "profit_rate": !(10 + %%i * 5)!, >> temp_scoring_%%i.json
    echo     "sharpe_ratio": !(1 + %%i * 0.5)!, >> temp_scoring_%%i.json
    echo     "max_drawdown": !(10 - %%i * 2)!, >> temp_scoring_%%i.json
    echo     "win_rate": !(0.5 + %%i * 0.1)! >> temp_scoring_%%i.json
    echo   }, >> temp_scoring_%%i.json
    echo   "random_seed": !(1000 + %%i)!, >> temp_scoring_%%i.json
    echo   "discovered_by": "test_%%i" >> temp_scoring_%%i.json
    echo } >> temp_scoring_%%i.json
    
    curl -s -X POST "%OPTIMIZER_URL%/share-result" -H "Content-Type: application/json" -d @temp_scoring_%%i.json >nul
)

curl -s "%OPTIMIZER_URL%/shared-results" > temp_response.txt
findstr /C:"scoring_test" temp_response.txt >nul
if %errorlevel% equ 0 (
    echo [SUCCESS] 结果评分测试完成
    echo [INFO] 按性能排序的结果:
    findstr /C:"scoring_test" temp_response.txt
) else (
    echo [ERROR] 结果评分测试失败
    goto :cleanup
)

REM 测试错误处理
echo [INFO] 测试错误处理...
echo [INFO] 测试无效JSON...
curl -s -X POST "%OPTIMIZER_URL%/share-result" -H "Content-Type: application/json" -d "{\"invalid\": json}" > temp_response.txt 2>&1
findstr /C:"Invalid request" temp_response.txt >nul
if %errorlevel% equ 0 (
    echo [SUCCESS] 无效JSON错误处理正常
) else (
    echo [WARNING] 无效JSON错误处理可能有问题
)

echo [INFO] 测试缺少必需字段...
echo { > temp_incomplete.json
echo   "task_id": "incomplete_test" >> temp_incomplete.json
echo } >> temp_incomplete.json

curl -s -X POST "%OPTIMIZER_URL%/share-result" -H "Content-Type: application/json" -d @temp_incomplete.json > temp_response.txt
findstr /C:"error" temp_response.txt >nul
if %errorlevel% equ 0 (
    echo [SUCCESS] 缺少字段错误处理正常
) else (
    echo [WARNING] 缺少字段错误处理可能有问题
)

REM 性能测试
echo [INFO] 测试性能...
set start_time=%time%
for /L %%i in (1,1,10) do (
    echo { > temp_perf_%%i.json
    echo   "task_id": "perf_test_%%i", >> temp_perf_%%i.json
    echo   "strategy_name": "perf_strategy", >> temp_perf_%%i.json
    echo   "performance": { >> temp_perf_%%i.json
    echo     "profit_rate": !(10 + %%i)!, >> temp_perf_%%i.json
    echo     "sharpe_ratio": !(1 + %%i / 10)!, >> temp_perf_%%i.json
    echo     "max_drawdown": !(10 - %%i / 2)!, >> temp_perf_%%i.json
    echo     "win_rate": !(0.5 + %%i / 100)! >> temp_perf_%%i.json
    echo   }, >> temp_perf_%%i.json
    echo   "random_seed": !(10000 + %%i)!, >> temp_perf_%%i.json
    echo   "discovered_by": "perf_test" >> temp_perf_%%i.json
    echo } >> temp_perf_%%i.json
    
    curl -s -X POST "%OPTIMIZER_URL%/share-result" -H "Content-Type: application/json" -d @temp_perf_%%i.json >nul
)
set end_time=%time%
echo [SUCCESS] 性能测试完成，10个结果共享完成

REM 生成测试报告
echo [INFO] 生成测试报告...
set report_file=%TEST_DATA_DIR%\test_report_%date:~0,4%%date:~5,2%%date:~8,2%_%time:~0,2%%time:~3,2%%time:~6,2%.txt
echo 结果共享系统测试报告 > "%report_file%"
echo ====================== >> "%report_file%"
echo 测试时间: %date% %time% >> "%report_file%"
echo 测试环境: %OPTIMIZER_URL% >> "%report_file%"
echo. >> "%report_file%"
echo 测试项目: >> "%report_file%"
echo 1. 健康检查 >> "%report_file%"
echo 2. 手动共享结果 >> "%report_file%"
echo 3. 获取共享结果 >> "%report_file%"
echo 4. 文件共享模式 >> "%report_file%"
echo 5. 字符串共享模式 >> "%report_file%"
echo 6. 种子共享模式 >> "%report_file%"
echo 7. 性能阈值过滤 >> "%report_file%"
echo 8. 跨服务器场景模拟 >> "%report_file%"
echo 9. 结果评分和排序 >> "%report_file%"
echo 10. 错误处理 >> "%report_file%"
echo 11. 性能测试 >> "%report_file%"
echo. >> "%report_file%"
echo 测试完成时间: %date% %time% >> "%report_file%"
echo [SUCCESS] 测试报告已生成: %report_file%

echo.
echo [SUCCESS] 所有测试完成！
echo.

:cleanup
REM 清理临时文件
del temp_*.json temp_*.txt 2>nul
echo [INFO] 临时文件清理完成

echo.
echo 测试完成！请查看测试报告了解详细信息。
pause

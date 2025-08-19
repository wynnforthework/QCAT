#!/bin/bash

# QCAT 自动化系统停止脚本
# 优雅地停止所有自动化功能

set -e

echo "🛑 Stopping QCAT Automation System..."
echo "=================================================="

# 检查PID文件
if [ ! -f "qcat.pid" ]; then
    echo "⚠️  Warning: qcat.pid file not found"
    echo "🔍 Searching for QCAT processes..."
    
    QCAT_PIDS=$(pgrep -f "qcat" || true)
    if [ -z "$QCAT_PIDS" ]; then
        echo "✅ No QCAT processes found"
        exit 0
    else
        echo "📋 Found QCAT processes: $QCAT_PIDS"
    fi
else
    QCAT_PID=$(cat qcat.pid)
    echo "📋 Found PID file: $QCAT_PID"
    
    # 检查进程是否存在
    if ! kill -0 $QCAT_PID 2>/dev/null; then
        echo "⚠️  Warning: Process $QCAT_PID not running"
        rm -f qcat.pid
        exit 0
    fi
fi

# 发送优雅停止信号
echo "📤 Sending graceful shutdown signal..."
if [ -n "$QCAT_PID" ]; then
    kill -TERM $QCAT_PID
    echo "✅ SIGTERM sent to process $QCAT_PID"
else
    # 如果没有PID文件，停止所有QCAT进程
    for pid in $QCAT_PIDS; do
        kill -TERM $pid
        echo "✅ SIGTERM sent to process $pid"
    done
fi

# 等待优雅停止
echo "⏳ Waiting for graceful shutdown..."
WAIT_TIME=0
MAX_WAIT=30

while [ $WAIT_TIME -lt $MAX_WAIT ]; do
    if [ -n "$QCAT_PID" ]; then
        if ! kill -0 $QCAT_PID 2>/dev/null; then
            echo "✅ Process $QCAT_PID stopped gracefully"
            break
        fi
    else
        RUNNING_PIDS=$(pgrep -f "qcat" || true)
        if [ -z "$RUNNING_PIDS" ]; then
            echo "✅ All QCAT processes stopped gracefully"
            break
        fi
    fi
    
    sleep 1
    WAIT_TIME=$((WAIT_TIME + 1))
    echo -n "."
done

echo ""

# 检查是否需要强制停止
if [ $WAIT_TIME -ge $MAX_WAIT ]; then
    echo "⚠️  Graceful shutdown timeout, forcing stop..."
    
    if [ -n "$QCAT_PID" ]; then
        if kill -0 $QCAT_PID 2>/dev/null; then
            kill -KILL $QCAT_PID
            echo "🔨 Process $QCAT_PID force killed"
        fi
    else
        RUNNING_PIDS=$(pgrep -f "qcat" || true)
        for pid in $RUNNING_PIDS; do
            kill -KILL $pid
            echo "🔨 Process $pid force killed"
        done
    fi
fi

# 清理PID文件
if [ -f "qcat.pid" ]; then
    rm -f qcat.pid
    echo "🗑️  Removed PID file"
fi

# 显示停止的自动化功能
echo "=================================================="
echo "🛑 QCAT Automation System stopped successfully!"
echo "=================================================="
echo ""
echo "📊 Stopped Automation Features:"
echo "   🛑 Strategy Parameter Optimization"
echo "   🛑 Best Parameter Application"
echo "   🛑 Dynamic Position Optimization"
echo "   🛑 Smart Position Management"
echo "   🛑 Automatic Stop Loss/Take Profit"
echo "   🛑 Periodic Strategy Optimization"
echo "   🛑 Strategy Elimination System"
echo "   🛑 New Strategy Introduction"
echo "   🛑 Dynamic Stop Loss Adjustment"
echo "   🛑 Hot Coin Recommendation"
echo "   🛑 Profit Maximization Engine"
echo "   🛑 Abnormal Market Response"
echo "   🛑 Account Security Monitoring"
echo "   🛑 Fund Diversification & Transfer"
echo "   🛑 Dynamic Fund Allocation"
echo "   🛑 Layered Position Management"
echo "   🛑 Multi-Strategy Hedging"
echo "   🛑 Data Cleaning & Correction"
echo "   🛑 Automatic Backtesting"
echo "   🛑 Dynamic Factor Library"
echo "   🛑 System Health Monitoring"
echo "   🛑 Multi-Exchange Redundancy"
echo "   🛑 Audit & Logging"
echo "   🛑 Strategy Self-Learning"
echo "   🛑 Genetic Evolution System"
echo "   🛑 Market Pattern Recognition"
echo ""
echo "💤 All automation features have been safely stopped."
echo "🔧 To restart: scripts/start_automation.sh"

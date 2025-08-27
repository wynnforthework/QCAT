#!/bin/bash

# 测试状态检查函数
# 设置端口变量
QCAT_API_PORT=8082
QCAT_OPTIMIZER_PORT=8081
FRONTEND_DEV_PORT=3001

# 显示状态函数（从start_local_improved.sh复制）
show_status() {
    echo "=========================================="
    echo "           QCAT 服务状态"
    echo "=========================================="
    
    # 检查后端API服务
    echo "🔍 检查后端API服务 (端口: $QCAT_API_PORT)..."
    if curl -s --connect-timeout 3 --max-time 5 http://localhost:$QCAT_API_PORT/health >/dev/null 2>&1; then
        echo -e "✅ 后端API服务 ($QCAT_API_PORT) - 运行中"
    else
        echo -e "❌ 后端API服务 ($QCAT_API_PORT) - 未运行"
        # 显示详细错误信息
        echo "   调试信息: $(curl -s --connect-timeout 3 --max-time 5 http://localhost:$QCAT_API_PORT/health 2>&1 | head -1 || echo '连接失败')"
    fi
    
    # 检查优化器服务
    echo "🔍 检查优化器服务 (端口: $QCAT_OPTIMIZER_PORT)..."
    if curl -s --connect-timeout 3 --max-time 5 http://localhost:$QCAT_OPTIMIZER_PORT/health >/dev/null 2>&1; then
        echo -e "✅ 优化器服务 ($QCAT_OPTIMIZER_PORT) - 运行中"
    else
        echo -e "⚠️  优化器服务 ($QCAT_OPTIMIZER_PORT) - 状态未知"
        echo "   调试信息: $(curl -s --connect-timeout 3 --max-time 5 http://localhost:$QCAT_OPTIMIZER_PORT/health 2>&1 | head -1 || echo '连接失败')"
    fi
    
    # 检查前端服务
    echo "🔍 检查前端服务 (端口: $FRONTEND_DEV_PORT)..."
    if curl -s --connect-timeout 3 --max-time 5 -I http://localhost:$FRONTEND_DEV_PORT >/dev/null 2>&1; then
        echo -e "✅ 前端服务 ($FRONTEND_DEV_PORT) - 运行中"
    else
        echo -e "⚠️  前端服务 ($FRONTEND_DEV_PORT) - 状态未知"
        echo "   调试信息: $(curl -s --connect-timeout 3 --max-time 5 -I http://localhost:$FRONTEND_DEV_PORT 2>&1 | head -1 || echo '连接失败')"
    fi
    
    # 显示端口占用情况
    echo ""
    echo "📊 端口占用情况:"
    if command -v netstat >/dev/null 2>&1; then
        echo "   端口 $QCAT_API_PORT: $(netstat -ano 2>/dev/null | grep :$QCAT_API_PORT | wc -l) 个连接"
        echo "   端口 $QCAT_OPTIMIZER_PORT: $(netstat -ano 2>/dev/null | grep :$QCAT_OPTIMIZER_PORT | wc -l) 个连接"
        echo "   端口 $FRONTEND_DEV_PORT: $(netstat -ano 2>/dev/null | grep :$FRONTEND_DEV_PORT | wc -l) 个连接"
    fi
    
    echo "=========================================="
    echo "🌐 前端:   http://localhost:$FRONTEND_DEV_PORT"
    echo "   后端API: http://localhost:$QCAT_API_PORT"
    echo "   优化器:  http://localhost:$QCAT_OPTIMIZER_PORT"
    echo "🛑 停止服务: Ctrl+C"
}

# 调用状态检查函数
show_status

#!/bin/bash

# QCAT 简化启动脚本 - 跳过数据库初始化

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

echo "=========================================="
echo "    QCAT 简化启动脚本"
echo "=========================================="

# 检查依赖
log_info "检查系统依赖..."

# 检查Go
if ! command -v go &> /dev/null; then
    log_error "Go 未安装，请先安装 Go 1.23+"
    exit 1
fi

# 检查Node.js
if ! command -v node &> /dev/null; then
    log_error "Node.js 未安装，请先安装 Node.js 20+"
    exit 1
fi

# 检查npm
if ! command -v npm &> /dev/null; then
    log_error "npm 未安装"
    exit 1
fi

log_success "依赖检查完成"

# 安装依赖
log_info "安装项目依赖..."

# 安装Go依赖
go mod download
go mod tidy

# 安装前端依赖
cd frontend
npm install
cd ..

log_success "依赖安装完成"

# 配置环境
log_info "配置环境..."

# 复制配置文件
if [ ! -f "configs/config.yaml" ] && [ -f "configs/config.yaml.example" ]; then
    cp configs/config.yaml.example configs/config.yaml
    log_info "已复制配置文件"
fi

# 创建日志目录
mkdir -p logs

log_success "环境配置完成"

# 启动服务
log_info "启动服务..."

# 启动后端
log_info "启动后端服务 (端口: 8082)..."
go run cmd/qcat/main.go &
BACKEND_PID=$!

# 启动前端
log_info "启动前端服务 (端口: 3000)..."
cd frontend
npm run dev &
FRONTEND_PID=$!
cd ..

sleep 10
log_success "所有服务启动完成"

# 显示状态
echo
echo "=========================================="
echo "           QCAT 服务状态"
echo "=========================================="

if curl -f http://localhost:8082/health >/dev/null 2>&1; then
    echo -e "✅ 后端API服务 (端口: 8082) - ${GREEN}运行中${NC}"
else
    echo -e "❌ 后端API服务 (端口: 8082) - ${RED}未运行${NC}"
fi

if curl -f http://localhost:3000 >/dev/null 2>&1; then
    echo -e "✅ 前端服务 (端口: 3000) - ${GREEN}运行中${NC}"
else
    echo -e "⚠️  前端服务 (端口: 3000) - ${YELLOW}状态未知${NC}"
fi

echo "=========================================="
echo
echo "🌐 访问地址:"
echo "   前端界面: http://localhost:3000"
echo "   后端API:  http://localhost:8082"
echo
echo "🛑 停止服务: 按 Ctrl+C"
echo

# 清理函数
cleanup() {
    log_info "正在停止服务..."
    
    if [ ! -z "$FRONTEND_PID" ]; then
        kill $FRONTEND_PID 2>/dev/null || true
    fi
    
    if [ ! -z "$BACKEND_PID" ]; then
        kill $BACKEND_PID 2>/dev/null || true
    fi
    
    log_success "服务已停止"
    exit 0
}

# 设置信号处理
trap cleanup SIGINT SIGTERM

# 等待用户中断
log_info "所有服务已启动，按 Ctrl+C 停止服务"
wait

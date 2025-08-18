#!/bin/bash

# QCAT 前端启动脚本

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
echo "    QCAT 前端启动脚本"
echo "=========================================="

# 检查依赖
log_info "检查系统依赖..."

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

# 安装前端依赖
log_info "安装前端依赖..."
cd frontend
npm install
cd ..

log_success "依赖安装完成"

# 启动前端服务
log_info "启动前端服务 (端口: 3000)..."
cd frontend
npm run dev

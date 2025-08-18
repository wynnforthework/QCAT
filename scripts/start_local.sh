#!/bin/bash

# QCAT 本地开发环境一键启动脚本

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

# 检查依赖
check_dependencies() {
    log_info "检查系统依赖..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装，请先安装 Go 1.23+"
        exit 1
    fi
    
    if ! command -v node &> /dev/null; then
        log_error "Node.js 未安装，请先安装 Node.js 20+"
        exit 1
    fi
    
    if ! command -v npm &> /dev/null; then
        log_error "npm 未安装"
        exit 1
    fi
    
    log_success "依赖检查完成"
}

# 安装依赖
install_dependencies() {
    log_info "安装项目依赖..."
    
    # 安装Go依赖
    go mod download
    go mod tidy
    
    # 安装前端依赖
    cd frontend
    npm install
    cd ..
    
    log_success "依赖安装完成"
}

# 配置环境
setup_config() {
    log_info "配置环境..."
    
    # 复制配置文件
    if [ ! -f "configs/config.yaml" ] && [ -f "configs/config.yaml.example" ]; then
        cp configs/config.yaml.example configs/config.yaml
        log_info "已复制配置文件"
    fi
    
    # 创建日志目录
    mkdir -p logs
    
    # 检查是否存在.env文件
    if [ ! -f ".env" ]; then
        if [ -f "deploy/env.example" ]; then
            log_warning "未找到.env文件，请复制deploy/env.example为.env并配置环境变量"
            log_info "或者使用以下默认环境变量："
            echo "export QCAT_DATABASE_PASSWORD=123"
            echo "export QCAT_REDIS_PASSWORD="
            echo "export QCAT_JWT_SECRET_KEY=f31e8818003142e8ad518726cda4af31"
            echo "export QCAT_EXCHANGE_API_KEY=your_api_key"
            echo "export QCAT_EXCHANGE_API_SECRET=your_api_secret"
            echo "export QCAT_ENCRYPTION_KEY=your_encryption_key"
        fi
    else
        log_info "找到.env文件，正在加载环境变量..."
        export $(grep -v '^#' .env | xargs)
    fi
    
    log_success "环境配置完成"
}

# 启动数据库服务
start_database() {
    log_info "启动数据库服务..."
    
    if command -v docker-compose &> /dev/null && [ -f "deploy/docker-compose.prod.yml" ]; then
        # 检查Redis是否启用
        if [ "$QCAT_REDIS_ENABLED" = "true" ]; then
            log_info "启动PostgreSQL和Redis..."
            docker-compose -f deploy/docker-compose.prod.yml up -d postgres redis
        else
            log_info "Redis已禁用，仅启动PostgreSQL..."
            docker-compose -f deploy/docker-compose.prod.yml up -d postgres
        fi
        sleep 10
        log_success "数据库服务启动完成"
    else
        log_warning "Docker Compose不可用，请手动启动PostgreSQL"
        if [ "$QCAT_REDIS_ENABLED" = "true" ]; then
            log_warning "Redis已启用，请确保Redis服务正在运行"
        else
            log_info "Redis已禁用，无需启动Redis服务"
        fi
        read -p "数据库服务已启动? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_error "请先启动数据库服务"
            exit 1
        fi
    fi
}

# 初始化数据库
init_database() {
    log_info "初始化数据库..."
    go run cmd/qcat/main.go -migrate
    log_success "数据库初始化完成"
}

# 启动服务
start_services() {
    log_info "启动服务..."
    
    # 启动后端
    log_info "启动后端服务 (端口: 8082)..."
    go run cmd/qcat/main.go &
    BACKEND_PID=$!
    
    # 启动优化器
    log_info "启动优化器服务 (端口: 8081)..."
    go run cmd/optimizer/main.go &
    OPTIMIZER_PID=$!
    
    # 启动前端
    log_info "启动前端服务 (端口: 3000)..."
    cd frontend
    npm run dev &
    FRONTEND_PID=$!
    cd ..
    
    sleep 10
    log_success "所有服务启动完成"
}

# 显示状态
show_status() {
    echo
    echo "=========================================="
    echo "           QCAT 服务状态"
    echo "=========================================="
    
    if curl -f http://localhost:8082/health >/dev/null 2>&1; then
        echo -e "✅ 后端API服务 (端口: 8082) - ${GREEN}运行中${NC}"
    else
        echo -e "❌ 后端API服务 (端口: 8082) - ${RED}未运行${NC}"
    fi
    
    if curl -f http://localhost:8081/health >/dev/null 2>&1; then
        echo -e "✅ 优化器服务 (端口: 8081) - ${GREEN}运行中${NC}"
    else
        echo -e "⚠️  优化器服务 (端口: 8081) - ${YELLOW}状态未知${NC}"
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
    echo "   优化器:   http://localhost:8081"
    echo
    echo "📊 配置信息:"
    echo "   环境: $QCAT_APP_ENVIRONMENT"
    echo "   数据库: $QCAT_DATABASE_HOST:$QCAT_DATABASE_PORT/$QCAT_DATABASE_NAME"
    echo "   Redis: $([ "$QCAT_REDIS_ENABLED" = "true" ] && echo "启用" || echo "禁用")"
    echo "   交易所: $QCAT_EXCHANGE_NAME ($([ "$QCAT_EXCHANGE_TEST_NET" = "true" ] && echo "测试网" || echo "主网"))"
    echo
    echo "🛑 停止服务: 按 Ctrl+C"
    echo
}

# 清理函数
cleanup() {
    log_info "正在停止服务..."
    
    if [ ! -z "$FRONTEND_PID" ]; then
        kill $FRONTEND_PID 2>/dev/null || true
    fi
    
    if [ ! -z "$OPTIMIZER_PID" ]; then
        kill $OPTIMIZER_PID 2>/dev/null || true
    fi
    
    if [ ! -z "$BACKEND_PID" ]; then
        kill $BACKEND_PID 2>/dev/null || true
    fi
    
    log_success "服务已停止"
    exit 0
}

# 主函数
main() {
    echo "=========================================="
    echo "    QCAT 本地开发环境一键启动脚本"
    echo "=========================================="
    
    # 设置信号处理
    trap cleanup SIGINT SIGTERM
    
    # 检查依赖
    check_dependencies
    
    # 安装依赖
    install_dependencies
    
    # 配置环境
    setup_config
    
    # 启动数据库
    start_database
    
    # 初始化数据库
    init_database
    
    # 启动服务
    start_services
    
    # 显示状态
    show_status
    
    # 等待用户中断
    log_info "所有服务已启动，按 Ctrl+C 停止服务"
    wait
}

# 显示帮助
if [[ "$1" == "-h" || "$1" == "--help" ]]; then
    echo "用法: $0"
    echo "启动QCAT本地开发环境"
    echo ""
    echo "环境变量配置:"
    echo "  复制 deploy/env.example 为 .env 并修改配置"
    echo "  或设置以下环境变量:"
    echo "    QCAT_DATABASE_PASSWORD - 数据库密码"
    echo "    QCAT_REDIS_ENABLED - 是否启用Redis (true/false)"
    echo "    QCAT_REDIS_PASSWORD - Redis密码"
    echo "    QCAT_JWT_SECRET_KEY - JWT密钥"
    echo "    QCAT_EXCHANGE_API_KEY - 交易所API密钥"
    echo "    QCAT_EXCHANGE_API_SECRET - 交易所API密钥"
    echo "    QCAT_ENCRYPTION_KEY - 加密密钥"
    exit 0
fi

# 执行主函数
main

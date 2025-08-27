#!/usr/bin/env bash
set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $1"; }

# 检测操作系统
detect_os() {
    case "$(uname -s)" in
        Linux*)     OS_TYPE="linux";;
        Darwin*)    OS_TYPE="mac";;
        CYGWIN*|MINGW*|MSYS*|Windows_NT) OS_TYPE="windows";;
        *)          OS_TYPE="unknown";;
    esac
    log_info "检测到操作系统: $OS_TYPE"
}

# 检查端口是否被占用
check_port() {
    local port=$1
    local service_name=$2
    
    if [ "$OS_TYPE" = "windows" ]; then
        if netstat -ano | findstr ":$port " > /dev/null 2>&1; then
            log_warning "端口 $port 已被占用，正在停止 $service_name 服务..."
            # 查找占用端口的进程并停止
            for pid in $(netstat -ano | findstr ":$port " | awk '{print $5}'); do
                taskkill /F /PID $pid > /dev/null 2>&1 || true
            done
            sleep 2
        fi
    else
        if lsof -i :$port > /dev/null 2>&1; then
            log_warning "端口 $port 已被占用，正在停止 $service_name 服务..."
            # 查找占用端口的进程并停止
            lsof -ti :$port | xargs kill -9 > /dev/null 2>&1 || true
            sleep 2
        fi
    fi
}

# 读取配置文件中的端口信息
read_port_config() {
    # 默认端口
    QCAT_API_PORT=8082
    QCAT_OPTIMIZER_PORT=8081
    FRONTEND_DEV_PORT=3001  # 改为3001避免冲突

    # 尝试从config.yaml读取端口配置
    if [ -f "configs/config.yaml" ]; then
        if command -v yq &> /dev/null; then
            QCAT_API_PORT=$(yq eval '.ports.qcat_api // 8082' configs/config.yaml 2>/dev/null || echo 8082)
            QCAT_OPTIMIZER_PORT=$(yq eval '.ports.qcat_optimizer // 8081' configs/config.yaml 2>/dev/null || echo 8081)
            FRONTEND_DEV_PORT=$(yq eval '.ports.frontend_dev // 3001' configs/config.yaml 2>/dev/null || echo 3001)
        else
            QCAT_API_PORT=$(grep -A 20 "^ports:" configs/config.yaml | grep "qcat_api:" | sed 's/.*qcat_api: *\([0-9]*\).*/\1/' | head -1)
            QCAT_OPTIMIZER_PORT=$(grep -A 20 "^ports:" configs/config.yaml | grep "qcat_optimizer:" | sed 's/.*qcat_optimizer: *\([0-9]*\).*/\1/' | head -1)
            FRONTEND_DEV_PORT=$(grep -A 20 "^ports:" configs/config.yaml | grep "frontend_dev:" | sed 's/.*frontend_dev: *\([0-9]*\).*/\1/' | head -1)

            [ -z "$QCAT_API_PORT" ] && QCAT_API_PORT=8082
            [ -z "$QCAT_OPTIMIZER_PORT" ] && QCAT_OPTIMIZER_PORT=8081
            [ -z "$FRONTEND_DEV_PORT" ] && FRONTEND_DEV_PORT=3001
        fi
    fi

    # 从环境变量覆盖（如果设置了）
    [ ! -z "$QCAT_PORTS_QCAT_API" ] && QCAT_API_PORT=$QCAT_PORTS_QCAT_API
    [ ! -z "$QCAT_PORTS_QCAT_OPTIMIZER" ] && QCAT_OPTIMIZER_PORT=$QCAT_PORTS_QCAT_OPTIMIZER
    [ ! -z "$QCAT_PORTS_FRONTEND_DEV" ] && FRONTEND_DEV_PORT=$QCAT_PORTS_FRONTEND_DEV

    log_info "端口配置: API=$QCAT_API_PORT, 优化器=$QCAT_OPTIMIZER_PORT, 前端=$FRONTEND_DEV_PORT"
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
    go mod download
    go mod tidy
    cd frontend && npm install && cd ..
    log_success "依赖安装完成"
}

# 配置环境
setup_config() {
    log_info "配置环境..."
    if [ ! -f "configs/config.yaml" ] && [ -f "configs/config.yaml.example" ]; then
        cp configs/config.yaml.example configs/config.yaml
        log_info "已复制配置文件"
    fi
    mkdir -p logs
    if [ ! -f ".env" ]; then
        if [ -f "deploy/env.example" ]; then
            log_warning "未找到.env文件，请复制 deploy/env.example 为 .env 并配置环境变量"
        fi
    else
        log_info "加载 .env 环境变量..."
        export $(grep -v '^#' .env | xargs)
    fi
    log_success "环境配置完成"
}

# 启动数据库服务
start_database() {
    log_info "启动数据库服务..."

    if command -v docker-compose &> /dev/null && [ -f "deploy/docker-compose.prod.yml" ]; then
        if [ "$QCAT_REDIS_ENABLED" = "true" ]; then
            log_info "使用 Docker 启动 PostgreSQL 和 Redis..."
            docker-compose -f deploy/docker-compose.prod.yml up -d postgres redis
        else
            log_info "使用 Docker 启动 PostgreSQL..."
            docker-compose -f deploy/docker-compose.prod.yml up -d postgres
        fi
        sleep 10
        log_success "数据库服务启动完成 (Docker)"
    else
        log_warning "未检测到 Docker Compose，将尝试手动启动数据库服务"
        log_info "⚠️ 请确保 PostgreSQL 已经在本地运行 (端口: $QCAT_DATABASE_PORT)"
        if [ "$QCAT_REDIS_ENABLED" = "true" ]; then
            log_info "⚠️ 请确保 Redis 已经在本地运行 (端口: $QCAT_REDIS_PORT)"
        else
            log_info "Redis 已禁用，无需启动"
        fi
    fi
}

# 初始化数据库
init_database() {
    log_info "初始化数据库..."
    go run cmd/migrate/main.go -up
    log_success "数据库初始化完成"
}

# 编译 Go 项目
build_binaries() {
    log_info "编译 Go 项目..."
    if [ "$OS_TYPE" = "windows" ]; then
        go build -o qcat.exe ./cmd/qcat/main.go
        go build -o optimizer.exe ./cmd/optimizer/main.go
    else
        go build -o qcat ./cmd/qcat/main.go
        go build -o optimizer ./cmd/optimizer/main.go
    fi
    log_success "Go 项目编译完成"
}

# 启动服务
start_services() {
    log_info "启动服务..."

    # 检查并清理端口占用
    check_port $QCAT_API_PORT "QCAT API"
    check_port $QCAT_OPTIMIZER_PORT "QCAT Optimizer"
    check_port $FRONTEND_DEV_PORT "Frontend"

    # 启动后端服务
    if [ "$OS_TYPE" = "windows" ]; then
        ./qcat.exe &
        BACKEND_PID=$!
        sleep 3
        ./optimizer.exe --port=$QCAT_OPTIMIZER_PORT &
        OPTIMIZER_PID=$!
    else
        ./qcat &
        BACKEND_PID=$!
        sleep 3
        ./optimizer --port=$QCAT_OPTIMIZER_PORT &
        OPTIMIZER_PID=$!
    fi

    # 为前端设置环境变量并启动
    log_info "设置前端环境变量: NEXT_PUBLIC_API_URL=http://localhost:$QCAT_API_PORT"
    cd frontend

    # 创建或更新 .env.local 文件
    echo "NEXT_PUBLIC_API_URL=http://localhost:$QCAT_API_PORT" > .env.local
    echo "NEXT_PUBLIC_APP_NAME=QCAT" >> .env.local
    echo "NEXT_PUBLIC_APP_VERSION=2.0.0" >> .env.local

    # 启动前端开发服务器（不使用turbopack）
    npx next dev --port $FRONTEND_DEV_PORT & FRONTEND_PID=$!
    cd ..

    sleep 10
    log_success "所有服务启动完成"
}

# 显示状态
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

# 清理函数
cleanup() {
    log_info "正在停止服务..."
    kill $FRONTEND_PID 2>/dev/null || true
    kill $OPTIMIZER_PID 2>/dev/null || true
    kill $BACKEND_PID 2>/dev/null || true
    log_success "服务已停止"
    exit 0
}

main() {
    trap cleanup SIGINT SIGTERM
    detect_os
    read_port_config
    check_dependencies
    install_dependencies
    setup_config
    start_database
    init_database
    build_binaries
    start_services
    show_status
    log_info "所有服务已启动，按 Ctrl+C 停止服务"
    wait
}

main

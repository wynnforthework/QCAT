#!/usr/bin/env bash
set -euo pipefail

# 脚本版本和信息
SCRIPT_VERSION="2.1.0"
SCRIPT_NAME="QCAT Local Development Starter"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# 日志函数增强
log_info()    { echo -e "${BLUE}[INFO]${NC} $(date '+%H:%M:%S') $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $(date '+%H:%M:%S') $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $(date '+%H:%M:%S') $1"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $(date '+%H:%M:%S') $1"; }
log_debug()   { [ "$DEBUG_MODE" = "true" ] && echo -e "${PURPLE}[DEBUG]${NC} $(date '+%H:%M:%S') $1" || true; }

# 全局变量
DEBUG_MODE=false
DEV_MODE=false
PRODUCTION_MODE=false
SKIP_DEPS=false
SKIP_BUILD=false
SERVICES_TO_START="all"
MAX_RETRIES=3
RETRY_DELAY=5

# 显示帮助信息
show_help() {
    cat << EOF
$SCRIPT_NAME v$SCRIPT_VERSION

用法: $0 [选项]

选项:
  -h, --help              显示此帮助信息
  -v, --version           显示版本信息
  -d, --dev               开发模式（启用热重载和调试）
  -p, --production        生产模式
  --debug                 启用调试输出
  --skip-deps             跳过依赖安装
  --skip-build            跳过编译步骤
  --services SERVICE      指定要启动的服务 (api,optimizer,frontend,all)
  --max-retries N         设置最大重试次数 (默认: 3)
  --retry-delay N         设置重试延迟秒数 (默认: 5)

示例:
  $0                      # 启动所有服务
  $0 --dev                # 开发模式启动
  $0 --services api       # 只启动API服务
  $0 --skip-deps --debug  # 跳过依赖安装并启用调试

EOF
}

# 解析命令行参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -v|--version)
                echo "$SCRIPT_NAME v$SCRIPT_VERSION"
                exit 0
                ;;
            -d|--dev)
                DEV_MODE=true
                log_info "启用开发模式"
                shift
                ;;
            -p|--production)
                PRODUCTION_MODE=true
                log_info "启用生产模式"
                shift
                ;;
            --debug)
                DEBUG_MODE=true
                log_info "启用调试模式"
                shift
                ;;
            --skip-deps)
                SKIP_DEPS=true
                log_info "跳过依赖安装"
                shift
                ;;
            --skip-build)
                SKIP_BUILD=true
                log_info "跳过编译步骤"
                shift
                ;;
            --services)
                SERVICES_TO_START="$2"
                log_info "指定启动服务: $SERVICES_TO_START"
                shift 2
                ;;
            --max-retries)
                MAX_RETRIES="$2"
                log_info "设置最大重试次数: $MAX_RETRIES"
                shift 2
                ;;
            --retry-delay)
                RETRY_DELAY="$2"
                log_info "设置重试延迟: $RETRY_DELAY 秒"
                shift 2
                ;;
            *)
                log_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# 检测操作系统
detect_os() {
    case "$(uname -s)" in
        Linux*)     OS_TYPE="linux";;
        Darwin*)    OS_TYPE="mac";;
        CYGWIN*|MINGW*|MSYS*|Windows_NT) OS_TYPE="windows";;
        *)          OS_TYPE="unknown";;
    esac
    log_info "检测到操作系统: $OS_TYPE"
    log_debug "系统详细信息: $(uname -a)" || true
    return 0
}

# 重试执行函数
retry_command() {
    local cmd="$1"
    local description="$2"
    local retries=0

    while [ $retries -lt $MAX_RETRIES ]; do
        log_debug "执行命令 (尝试 $((retries + 1))/$MAX_RETRIES): $cmd"
        if eval "$cmd"; then
            log_success "$description 成功"
            return 0
        else
            retries=$((retries + 1))
            if [ $retries -lt $MAX_RETRIES ]; then
                log_warning "$description 失败，$RETRY_DELAY 秒后重试 ($retries/$MAX_RETRIES)"
                sleep $RETRY_DELAY
            else
                log_error "$description 失败，已达到最大重试次数"
                return 1
            fi
        fi
    done
}

# 检查端口是否被占用（增强版）
check_port() {
    local port=$1
    local service_name=$2
    local force_kill=${3:-false}

    log_debug "检查端口 $port 是否被占用"

    if [ "$OS_TYPE" = "windows" ]; then
        if netstat -ano | findstr ":$port " > /dev/null 2>&1; then
            local pids=$(netstat -ano | findstr ":$port " | awk '{print $5}' | sort -u)
            log_warning "端口 $port 已被占用，进程ID: $pids"

            if [ "$force_kill" = "true" ]; then
                log_info "正在停止占用端口 $port 的 $service_name 服务..."
                for pid in $pids; do
                    if [ "$pid" != "0" ] && [ -n "$pid" ]; then
                        log_debug "终止进程 $pid"
                        taskkill /F /PID $pid > /dev/null 2>&1 || true
                    fi
                done
                sleep 3
                # 验证端口是否已释放
                if netstat -ano | findstr ":$port " > /dev/null 2>&1; then
                    log_error "端口 $port 仍被占用，请手动检查"
                    return 1
                else
                    log_success "端口 $port 已释放"
                fi
            else
                return 1
            fi
        fi
    else
        if lsof -i :$port > /dev/null 2>&1; then
            local pids=$(lsof -ti :$port)
            log_warning "端口 $port 已被占用，进程ID: $pids"

            if [ "$force_kill" = "true" ]; then
                log_info "正在停止占用端口 $port 的 $service_name 服务..."
                # 先尝试优雅关闭
                echo "$pids" | xargs -r kill -TERM 2>/dev/null || true
                sleep 2
                # 检查是否还在运行，强制关闭
                if lsof -i :$port > /dev/null 2>&1; then
                    log_debug "优雅关闭失败，强制终止进程"
                    echo "$pids" | xargs -r kill -9 2>/dev/null || true
                    sleep 1
                fi
                # 验证端口是否已释放
                if lsof -i :$port > /dev/null 2>&1; then
                    log_error "端口 $port 仍被占用，请手动检查"
                    return 1
                else
                    log_success "端口 $port 已释放"
                fi
            else
                return 1
            fi
        fi
    fi
    return 0
}

# 验证配置文件
validate_config() {
    local config_file="configs/config.yaml"
    local env_file=".env"

    log_info "验证配置文件..."

    # 检查配置文件是否存在
    if [ ! -f "$config_file" ]; then
        if [ -f "configs/config.yaml.example" ]; then
            log_warning "配置文件不存在，从示例文件创建"
            cp "configs/config.yaml.example" "$config_file"
        else
            log_error "配置文件和示例文件都不存在"
            return 1
        fi
    fi

    # 检查环境文件
    if [ ! -f "$env_file" ]; then
        if [ -f "deploy/env.example" ]; then
            log_warning "环境文件不存在，从示例文件创建"
            cp "deploy/env.example" "$env_file"
            log_warning "请编辑 .env 文件配置必要的环境变量"
        fi
    fi

    # 验证配置文件语法
    if command -v yq &> /dev/null; then
        if ! yq eval '.' "$config_file" > /dev/null 2>&1; then
            log_error "配置文件 $config_file 语法错误"
            return 1
        fi
        log_debug "配置文件语法验证通过"
    fi

    log_success "配置文件验证完成"
}

# 读取配置文件中的端口信息（增强版）
read_port_config() {
    log_info "读取端口配置..."

    # 默认端口配置
    QCAT_API_PORT=8082
    QCAT_OPTIMIZER_PORT=8081
    FRONTEND_DEV_PORT=3001

    # 根据模式调整默认端口
    if [ "$DEV_MODE" = "true" ]; then
        FRONTEND_DEV_PORT=3001  # 开发模式使用3001
    elif [ "$PRODUCTION_MODE" = "true" ]; then
        FRONTEND_DEV_PORT=3000  # 生产模式使用3000
    fi

    # 尝试从config.yaml读取端口配置
    if [ -f "configs/config.yaml" ]; then
        log_debug "从 configs/config.yaml 读取端口配置"
        if command -v yq &> /dev/null; then
            QCAT_API_PORT=$(yq eval '.ports.qcat_api // 8082' configs/config.yaml 2>/dev/null || echo 8082)
            QCAT_OPTIMIZER_PORT=$(yq eval '.ports.qcat_optimizer // 8081' configs/config.yaml 2>/dev/null || echo 8081)
            FRONTEND_DEV_PORT=$(yq eval '.ports.frontend_dev // 3001' configs/config.yaml 2>/dev/null || echo 3001)
        else
            log_debug "yq 未安装，使用 grep 解析配置"
            QCAT_API_PORT=$(grep -A 20 "^ports:" configs/config.yaml | grep "qcat_api:" | sed 's/.*qcat_api: *\([0-9]*\).*/\1/' | head -1)
            QCAT_OPTIMIZER_PORT=$(grep -A 20 "^ports:" configs/config.yaml | grep "qcat_optimizer:" | sed 's/.*qcat_optimizer: *\([0-9]*\).*/\1/' | head -1)
            FRONTEND_DEV_PORT=$(grep -A 20 "^ports:" configs/config.yaml | grep "frontend_dev:" | sed 's/.*frontend_dev: *\([0-9]*\).*/\1/' | head -1)

            [ -z "$QCAT_API_PORT" ] && QCAT_API_PORT=8082
            [ -z "$QCAT_OPTIMIZER_PORT" ] && QCAT_OPTIMIZER_PORT=8081
            [ -z "$FRONTEND_DEV_PORT" ] && FRONTEND_DEV_PORT=3001
        fi
    else
        log_warning "配置文件 configs/config.yaml 不存在，使用默认端口"
    fi

    # 从环境变量覆盖（优先级最高）
    [ -n "${QCAT_PORTS_QCAT_API:-}" ] && QCAT_API_PORT=$QCAT_PORTS_QCAT_API
    [ -n "${QCAT_PORTS_QCAT_OPTIMIZER:-}" ] && QCAT_OPTIMIZER_PORT=$QCAT_PORTS_QCAT_OPTIMIZER
    [ -n "${QCAT_PORTS_FRONTEND_DEV:-}" ] && FRONTEND_DEV_PORT=$QCAT_PORTS_FRONTEND_DEV

    # 验证端口号有效性
    for port in "$QCAT_API_PORT" "$QCAT_OPTIMIZER_PORT" "$FRONTEND_DEV_PORT"; do
        # 检查是否为数字
        if ! echo "$port" | grep -q '^[0-9]\+$'; then
            log_error "端口号必须是数字: $port"
            return 1
        fi
        # 检查端口范围
        if [ "$port" -lt 1024 ] || [ "$port" -gt 65535 ]; then
            log_error "端口号超出有效范围 (1024-65535): $port"
            return 1
        fi
    done

    log_info "端口配置: API=$QCAT_API_PORT, 优化器=$QCAT_OPTIMIZER_PORT, 前端=$FRONTEND_DEV_PORT"
    log_debug "运行模式: DEV=$DEV_MODE, PROD=$PRODUCTION_MODE"
}

# 检查系统依赖（增强版）
check_dependencies() {
    log_info "检查系统依赖..."
    local missing_deps=()

    # 检查 Go
    if ! command -v go &> /dev/null; then
        missing_deps+=("Go 1.23+")
        log_error "Go 未安装"
    else
        local go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
        log_debug "检测到 Go 版本: $go_version"
        # 简单的版本检查（可以更精确）
        if [[ $(echo "$go_version" | cut -d. -f1) -lt 1 ]] || [[ $(echo "$go_version" | cut -d. -f1) -eq 1 && $(echo "$go_version" | cut -d. -f2) -lt 21 ]]; then
            log_warning "Go 版本可能过低，建议使用 1.21+"
        fi
    fi

    # 检查 Node.js
    if ! command -v node &> /dev/null; then
        missing_deps+=("Node.js 20+")
        log_error "Node.js 未安装"
    else
        local node_version=$(node --version | sed 's/v//')
        log_debug "检测到 Node.js 版本: $node_version"
        local major_version=$(echo "$node_version" | cut -d. -f1)
        if [ "$major_version" -lt 18 ]; then
            log_warning "Node.js 版本可能过低，建议使用 18+"
        fi
    fi

    # 检查 npm
    if ! command -v npm &> /dev/null; then
        missing_deps+=("npm")
        log_error "npm 未安装"
    else
        local npm_version=$(npm --version)
        log_debug "检测到 npm 版本: $npm_version"
    fi

    # 检查可选依赖
    if ! command -v yq &> /dev/null; then
        log_warning "yq 未安装，将使用 grep 解析 YAML 配置"
    fi

    if ! command -v docker &> /dev/null; then
        log_warning "Docker 未安装，将无法使用容器化数据库"
    fi

    if ! command -v docker-compose &> /dev/null; then
        log_warning "Docker Compose 未安装，将无法使用容器化服务"
    fi

    # 如果有缺失的关键依赖，退出
    if [ ${#missing_deps[@]} -gt 0 ]; then
        log_error "缺少以下关键依赖:"
        for dep in "${missing_deps[@]}"; do
            log_error "  - $dep"
        done
        log_error "请安装缺失的依赖后重试"
        exit 1
    fi

    log_success "系统依赖检查完成"
}

# 安装项目依赖（增强版）
install_dependencies() {
    if [ "$SKIP_DEPS" = "true" ]; then
        log_info "跳过依赖安装"
        return 0
    fi

    log_info "安装项目依赖..."

    # 安装 Go 依赖
    log_info "安装 Go 依赖..."
    if ! retry_command "go mod download" "Go 依赖下载"; then
        log_error "Go 依赖下载失败"
        return 1
    fi

    if ! retry_command "go mod tidy" "Go 依赖整理"; then
        log_error "Go 依赖整理失败"
        return 1
    fi

    # 安装前端依赖
    if [ -d "frontend" ]; then
        log_info "安装前端依赖..."
        cd frontend

        # 检查 package.json 是否存在
        if [ ! -f "package.json" ]; then
            log_error "frontend/package.json 不存在"
            cd ..
            return 1
        fi

        # 清理可能的缓存问题
        if [ "$DEV_MODE" = "true" ]; then
            log_debug "开发模式：清理 npm 缓存"
            npm cache clean --force 2>/dev/null || true
        fi

        # 安装依赖
        if ! retry_command "npm install" "前端依赖安装"; then
            log_error "前端依赖安装失败"
            cd ..
            return 1
        fi

        # 检查是否需要安装开发依赖
        if [ "$DEV_MODE" = "true" ] && [ -f "package-lock.json" ]; then
            log_debug "验证依赖完整性"
            npm audit fix --audit-level moderate 2>/dev/null || true
        fi

        cd ..
    else
        log_warning "frontend 目录不存在，跳过前端依赖安装"
    fi

    log_success "项目依赖安装完成"
}

# 配置环境（增强版）
setup_config() {
    log_info "配置环境..."

    # 验证配置文件
    if ! validate_config; then
        log_error "配置验证失败"
        return 1
    fi

    # 创建必要的目录
    log_debug "创建必要的目录"
    mkdir -p logs data/shared_results audit_logs

    # 设置日志文件权限
    if [ "$OS_TYPE" != "windows" ]; then
        chmod 755 logs 2>/dev/null || true
    fi

    # 加载环境变量
    if [ -f ".env" ]; then
        log_info "加载 .env 环境变量..."
        # 安全地加载环境变量，避免包含空格或特殊字符的问题
        while IFS= read -r line; do
            # 跳过注释和空行
            [[ $line =~ ^[[:space:]]*# ]] && continue
            [[ -z "${line// }" ]] && continue

            # 导出变量
            if [[ $line =~ ^[A-Za-z_][A-Za-z0-9_]*= ]]; then
                export "$line"
                log_debug "加载环境变量: ${line%%=*}"
            fi
        done < .env
    else
        log_warning "未找到 .env 文件"
        if [ -f "deploy/env.example" ]; then
            log_info "可以复制 deploy/env.example 为 .env 并配置环境变量"
        fi
    fi

    # 根据运行模式设置特定配置
    if [ "$DEV_MODE" = "true" ]; then
        log_info "配置开发模式环境"
        export QCAT_APP_ENVIRONMENT=development
        export QCAT_LOGGING_LEVEL=debug
    elif [ "$PRODUCTION_MODE" = "true" ]; then
        log_info "配置生产模式环境"
        export QCAT_APP_ENVIRONMENT=production
        export QCAT_LOGGING_LEVEL=info
    fi

    log_success "环境配置完成"
}

# 启动数据库服务（增强版）
start_database() {
    if [[ "$SERVICES_TO_START" == *"frontend"* ]] && [[ "$SERVICES_TO_START" != "all" ]]; then
        log_info "仅启动前端服务，跳过数据库启动"
        return 0
    fi

    log_info "启动数据库服务..."

    # 检查是否有 Docker 环境
    if command -v docker-compose &> /dev/null && [ -f "deploy/docker-compose.prod.yml" ]; then
        log_info "检测到 Docker Compose，使用容器化数据库"

        # 检查 Docker 是否运行
        if ! docker info > /dev/null 2>&1; then
            log_error "Docker 未运行，请启动 Docker 后重试"
            return 1
        fi

        # 启动数据库服务
        local services_to_start="postgres"
        if [ "${QCAT_REDIS_ENABLED:-true}" = "true" ]; then
            services_to_start="$services_to_start redis"
            log_info "启动 PostgreSQL 和 Redis 服务..."
        else
            log_info "启动 PostgreSQL 服务..."
        fi

        if ! retry_command "docker-compose -f deploy/docker-compose.prod.yml up -d $services_to_start" "数据库服务启动"; then
            log_error "数据库服务启动失败"
            return 1
        fi

        # 等待数据库服务就绪
        log_info "等待数据库服务就绪..."
        local wait_time=0
        local max_wait=60

        while [ $wait_time -lt $max_wait ]; do
            if docker-compose -f deploy/docker-compose.prod.yml ps postgres | grep -q "Up"; then
                log_success "PostgreSQL 服务已就绪"
                break
            fi
            sleep 2
            wait_time=$((wait_time + 2))
            if [ $((wait_time % 10)) -eq 0 ]; then
                log_debug "等待 PostgreSQL 启动... ($wait_time/$max_wait 秒)"
            fi
        done

        if [ $wait_time -ge $max_wait ]; then
            log_error "PostgreSQL 启动超时"
            return 1
        fi

        # 检查 Redis（如果启用）
        if [ "${QCAT_REDIS_ENABLED:-true}" = "true" ]; then
            if docker-compose -f deploy/docker-compose.prod.yml ps redis | grep -q "Up"; then
                log_success "Redis 服务已就绪"
            else
                log_warning "Redis 服务状态异常"
            fi
        fi

        log_success "数据库服务启动完成 (Docker)"
    else
        log_warning "未检测到 Docker Compose 或配置文件，假设使用本地数据库"

        # 检查本地数据库连接
        local db_host="${QCAT_DATABASE_HOST:-localhost}"
        local db_port="${QCAT_DATABASE_PORT:-5432}"

        log_info "检查本地 PostgreSQL 连接 ($db_host:$db_port)"

        # 简单的端口检查
        if [ "$OS_TYPE" = "windows" ]; then
            if ! netstat -ano | findstr ":$db_port " > /dev/null 2>&1; then
                log_error "PostgreSQL 未在端口 $db_port 运行"
                log_info "请确保 PostgreSQL 已启动并监听端口 $db_port"
                return 1
            fi
        else
            if ! nc -z "$db_host" "$db_port" 2>/dev/null; then
                log_error "无法连接到 PostgreSQL ($db_host:$db_port)"
                log_info "请确保 PostgreSQL 已启动并可访问"
                return 1
            fi
        fi

        log_success "本地 PostgreSQL 连接正常"

        # 检查 Redis（如果启用）
        if [ "${QCAT_REDIS_ENABLED:-true}" = "true" ]; then
            local redis_addr="${QCAT_REDIS_ADDR:-localhost:6379}"
            local redis_host="${redis_addr%%:*}"
            local redis_port="${redis_addr##*:}"

            log_info "检查本地 Redis 连接 ($redis_addr)"

            if [ "$OS_TYPE" = "windows" ]; then
                if ! netstat -ano | findstr ":$redis_port " > /dev/null 2>&1; then
                    log_warning "Redis 未在端口 $redis_port 运行"
                fi
            else
                if ! nc -z "$redis_host" "$redis_port" 2>/dev/null; then
                    log_warning "无法连接到 Redis ($redis_addr)"
                fi
            fi
        else
            log_info "Redis 已禁用，无需检查"
        fi
    fi
}

# 初始化数据库（增强版）
init_database() {
    if [[ "$SERVICES_TO_START" == *"frontend"* ]] && [[ "$SERVICES_TO_START" != "all" ]]; then
        log_info "仅启动前端服务，跳过数据库初始化"
        return 0
    fi

    log_info "初始化数据库..."

    # 检查迁移工具是否存在
    local migrate_cmd="go run cmd/migrate/main.go"
    if [ ! -f "cmd/migrate/main.go" ]; then
        log_error "数据库迁移工具不存在: cmd/migrate/main.go"
        return 1
    fi

    # 执行数据库迁移
    if ! retry_command "$migrate_cmd -up" "数据库迁移"; then
        log_error "数据库迁移失败"
        log_info "请检查数据库连接和迁移文件"
        return 1
    fi

    log_success "数据库初始化完成"
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

# 编译 Go 项目（增强版）
build_binaries() {
    if [ "$SKIP_BUILD" = "true" ]; then
        log_info "跳过编译步骤"
        return 0
    fi

    log_info "编译 Go 项目..."

    # 设置编译标志
    local build_flags="-v"
    if [ "$PRODUCTION_MODE" = "true" ]; then
        build_flags="$build_flags -ldflags='-w -s'"  # 生产模式：减小二进制文件大小
    elif [ "$DEV_MODE" = "true" ]; then
        # 开发模式：启用竞态检测（仅在支持的平台上）
        if [ "$OS_TYPE" != "windows" ] && command -v gcc >/dev/null 2>&1; then
            build_flags="$build_flags -race"
            log_debug "启用竞态检测"
        else
            log_debug "Windows 环境或缺少 CGO，跳过竞态检测"
        fi
    fi

    # 编译主应用
    log_info "编译 QCAT 主应用..."
    if [ "$OS_TYPE" = "windows" ]; then
        if ! retry_command "go build $build_flags -o qcat.exe ./cmd/qcat/main.go" "QCAT 主应用编译"; then
            log_error "QCAT 主应用编译失败"
            return 1
        fi
    else
        if ! retry_command "go build $build_flags -o qcat ./cmd/qcat/main.go" "QCAT 主应用编译"; then
            log_error "QCAT 主应用编译失败"
            return 1
        fi
    fi

    # 编译优化器（如果需要启动）
    if [[ "$SERVICES_TO_START" == "all" || "$SERVICES_TO_START" == *"optimizer"* ]]; then
        log_info "编译 QCAT 优化器..."
        if [ "$OS_TYPE" = "windows" ]; then
            if ! retry_command "go build $build_flags -o optimizer.exe ./cmd/optimizer/main.go" "QCAT 优化器编译"; then
                log_error "QCAT 优化器编译失败"
                return 1
            fi
        else
            if ! retry_command "go build $build_flags -o optimizer ./cmd/optimizer/main.go" "QCAT 优化器编译"; then
                log_error "QCAT 优化器编译失败"
                return 1
            fi
        fi
    fi

    # 验证编译结果
    local binaries=()
    if [ "$OS_TYPE" = "windows" ]; then
        [ -f "qcat.exe" ] && binaries+=("qcat.exe")
        [[ "$SERVICES_TO_START" == "all" || "$SERVICES_TO_START" == *"optimizer"* ]] && [ -f "optimizer.exe" ] && binaries+=("optimizer.exe")
    else
        [ -f "qcat" ] && binaries+=("qcat")
        [[ "$SERVICES_TO_START" == "all" || "$SERVICES_TO_START" == *"optimizer"* ]] && [ -f "optimizer" ] && binaries+=("optimizer")
    fi

    log_info "编译完成的二进制文件:"
    for binary in "${binaries[@]}"; do
        local size=$(ls -lh "$binary" | awk '{print $5}')
        log_info "  - $binary ($size)"
    done

    log_success "Go 项目编译完成"
}

# 启动单个服务
start_service() {
    local service_name="$1"
    local service_port="$2"
    local service_cmd="$3"
    local service_args="$4"

    log_info "启动 $service_name 服务 (端口: $service_port)..."

    # 检查并清理端口占用
    if ! check_port "$service_port" "$service_name" true; then
        log_error "无法清理端口 $service_port，$service_name 启动失败"
        echo ""
        return 1
    fi

    # 启动服务（确保真正后台运行）
    log_debug "执行命令: $service_cmd $service_args"
    local log_name=$(echo "${service_name,,}" | tr ' ' '_')
    if [ -n "$service_args" ]; then
        nohup $service_cmd $service_args > "logs/${log_name}_$(date +%Y%m%d_%H%M%S).log" 2>&1 &
    else
        nohup $service_cmd > "logs/${log_name}_$(date +%Y%m%d_%H%M%S).log" 2>&1 &
    fi

    local pid=$!
    log_debug "$service_name PID: $pid"

    # 简化启动检查，避免复杂的端口检查导致问题
    sleep 3  # 给服务一些时间启动

    # 检查进程是否还在运行
    if kill -0 $pid 2>/dev/null; then
        log_success "$service_name 启动成功 (PID: $pid, 端口: $service_port)"
        echo $pid
    else
        log_error "$service_name 进程启动失败或已退出"
        echo ""
        return 1
    fi
}

# 确保日志目录存在
ensure_log_directory() {
    if [ ! -d "logs" ]; then
        mkdir -p logs
        log_debug "创建日志目录: logs"
    fi
}

# 启动服务（增强版）
start_services() {
    log_info "启动服务..."
    log_info "服务启动配置: $SERVICES_TO_START"

    # 确保日志目录存在
    ensure_log_directory

    # 全局变量存储 PID
    BACKEND_PID=""
    OPTIMIZER_PID=""
    FRONTEND_PID=""

    # 启动 API 服务
    if [[ "$SERVICES_TO_START" == "all" || "$SERVICES_TO_START" == *"api"* ]]; then
        local api_cmd
        if [ "$OS_TYPE" = "windows" ]; then
            api_cmd="./qcat.exe"
        else
            api_cmd="./qcat"
        fi

        if [ -f "$api_cmd" ]; then
            BACKEND_PID=$(start_service "QCAT API" "$QCAT_API_PORT" "$api_cmd" "" || echo "")
            if [ -z "$BACKEND_PID" ]; then
                log_error "QCAT API 启动失败"
                BACKEND_PID=""
            else
                sleep 2  # 等待 API 服务完全启动
            fi
        else
            log_error "QCAT API 二进制文件不存在: $api_cmd"
            BACKEND_PID=""
        fi
    fi

    # 启动优化器服务
    if [[ "$SERVICES_TO_START" == "all" || "$SERVICES_TO_START" == *"optimizer"* ]]; then
        local optimizer_cmd
        if [ "$OS_TYPE" = "windows" ]; then
            optimizer_cmd="./optimizer.exe"
        else
            optimizer_cmd="./optimizer"
        fi

        if [ -f "$optimizer_cmd" ]; then
            OPTIMIZER_PID=$(start_service "QCAT Optimizer" "$QCAT_OPTIMIZER_PORT" "$optimizer_cmd" "--port=$QCAT_OPTIMIZER_PORT" || echo "")
            if [ -z "$OPTIMIZER_PID" ]; then
                log_warning "QCAT Optimizer 启动失败，但继续启动其他服务"
            fi
        else
            log_warning "QCAT Optimizer 二进制文件不存在: $optimizer_cmd"
        fi
    fi

    # 启动前端服务
    if [[ "$SERVICES_TO_START" == "all" || "$SERVICES_TO_START" == *"frontend"* ]]; then
        if [ -d "frontend" ]; then
            log_info "配置前端环境变量"

            # 创建或更新 .env.local 文件
            cat > frontend/.env.local << EOF
NEXT_PUBLIC_API_URL=http://localhost:$QCAT_API_PORT
NEXT_PUBLIC_APP_NAME=QCAT
NEXT_PUBLIC_APP_VERSION=2.0.0
EOF

            if [ "$DEV_MODE" = "true" ]; then
                echo "NEXT_PUBLIC_ENV=development" >> frontend/.env.local
            elif [ "$PRODUCTION_MODE" = "true" ]; then
                echo "NEXT_PUBLIC_ENV=production" >> frontend/.env.local
            else
                echo "NEXT_PUBLIC_ENV=development" >> frontend/.env.local
            fi

            log_debug "前端环境变量配置完成"

            # 检查并清理端口占用
            if ! check_port "$FRONTEND_DEV_PORT" "Frontend" true; then
                log_error "无法清理端口 $FRONTEND_DEV_PORT，Frontend 启动失败"
                FRONTEND_PID=""
            else
                # 启动前端开发服务器
                log_info "启动 Frontend 服务 (端口: $FRONTEND_DEV_PORT)..."

                # 构建启动命令
                local frontend_cmd="npx next dev --port $FRONTEND_DEV_PORT"
                if [ "$DEV_MODE" = "true" ]; then
                    frontend_cmd="$frontend_cmd --turbo"  # 开发模式启用 turbopack
                fi

                log_debug "执行命令: cd frontend && $frontend_cmd"

                # 启动前端服务（确保真正后台运行）
                cd frontend
                nohup $frontend_cmd > "../logs/frontend_$(date +%Y%m%d_%H%M%S).log" 2>&1 &
                FRONTEND_PID=$!
                cd ..

                log_debug "Frontend PID: $FRONTEND_PID"

                # 等待前端服务启动
                local wait_time=0
                local max_wait=30
                local frontend_ready=false

                while [ $wait_time -lt $max_wait ]; do
                    # 检查进程是否还在运行
                    if ! kill -0 "$FRONTEND_PID" 2>/dev/null; then
                        log_error "Frontend 进程意外退出"
                        FRONTEND_PID=""
                        break
                    fi

                    # 检查端口是否开始监听
                    if [ "$OS_TYPE" = "windows" ]; then
                        if netstat -ano | findstr ":$FRONTEND_DEV_PORT " > /dev/null 2>&1; then
                            frontend_ready=true
                            break
                        fi
                    else
                        if lsof -i :$FRONTEND_DEV_PORT > /dev/null 2>&1; then
                            frontend_ready=true
                            break
                        fi
                    fi

                    sleep 1
                    wait_time=$((wait_time + 1))

                    if [ $((wait_time % 10)) -eq 0 ]; then
                        log_debug "等待 Frontend 启动... ($wait_time/$max_wait 秒)"
                    fi
                done

                if [ "$frontend_ready" = true ]; then
                    log_success "Frontend 启动成功 (PID: $FRONTEND_PID, 端口: $FRONTEND_DEV_PORT)"
                elif [ -n "$FRONTEND_PID" ]; then
                    log_warning "Frontend 启动超时，但进程仍在运行 (PID: $FRONTEND_PID)"
                    log_info "请检查日志文件: logs/frontend_$(date +%Y%m%d_%H%M%S).log"
                fi
            fi
        else
            log_warning "frontend 目录不存在，跳过前端服务启动"
            FRONTEND_PID=""
        fi
    fi

    # 等待所有服务稳定
    log_info "等待服务稳定..."
    sleep 3

    # 验证服务状态
    log_info "验证服务状态..."
    local services_ok=true

    if [[ "$SERVICES_TO_START" == "all" || "$SERVICES_TO_START" == *"api"* ]]; then
        if [ -n "$BACKEND_PID" ] && kill -0 "$BACKEND_PID" 2>/dev/null; then
            log_success "✅ API服务运行正常 (PID: $BACKEND_PID)"
        else
            log_error "❌ API服务启动失败"
            services_ok=false
        fi
    fi

    if [[ "$SERVICES_TO_START" == "all" || "$SERVICES_TO_START" == *"optimizer"* ]]; then
        if [ -n "$OPTIMIZER_PID" ] && kill -0 "$OPTIMIZER_PID" 2>/dev/null; then
            log_success "✅ 优化器服务运行正常 (PID: $OPTIMIZER_PID)"
        else
            log_warning "⚠️ 优化器服务启动失败或未启动"
        fi
    fi

    if [[ "$SERVICES_TO_START" == "all" || "$SERVICES_TO_START" == *"frontend"* ]]; then
        if [ -n "$FRONTEND_PID" ] && kill -0 "$FRONTEND_PID" 2>/dev/null; then
            log_success "✅ 前端服务运行正常 (PID: $FRONTEND_PID)"
        else
            log_error "❌ 前端服务启动失败"
            services_ok=false
        fi
    fi

    if [ "$services_ok" = true ]; then
        log_success "🎉 所有服务启动完成！"
    else
        log_warning "⚠️ 部分服务启动失败，请检查日志"
    fi

    log_info "启动的服务 PID: API=$BACKEND_PID, Optimizer=$OPTIMIZER_PID, Frontend=$FRONTEND_PID"
}

# 检查单个服务状态
check_service_status() {
    local service_name="$1"
    local service_port="$2"
    local health_endpoint="$3"
    local pid="$4"

    echo "🔍 检查 $service_name (端口: $service_port)..."

    # 检查进程是否存在
    local process_status="❌"
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
        process_status="✅"
    fi

    # 检查端口是否监听
    local port_status="❌"
    if [ "$OS_TYPE" = "windows" ]; then
        if netstat -ano | findstr ":$service_port " > /dev/null 2>&1; then
            port_status="✅"
        fi
    else
        if lsof -i :$service_port > /dev/null 2>&1; then
            port_status="✅"
        fi
    fi

    # 检查健康端点（如果提供）
    local health_status="⚠️"
    local health_info=""
    if [ -n "$health_endpoint" ]; then
        if curl -s --connect-timeout 2 --max-time 3 "$health_endpoint" >/dev/null 2>&1; then
            health_status="✅"
            health_info="健康检查通过"
        else
            health_info="健康检查失败"
        fi
    else
        health_info="无健康检查端点"
    fi

    # 显示状态
    echo "   进程: $process_status | 端口: $port_status | 健康: $health_status"
    if [ "$DEBUG_MODE" = "true" ]; then
        echo "   详情: $health_info"
        [ -n "$pid" ] && echo "   PID: $pid"
    fi
}

# 显示状态（增强版）
show_status() {
    echo ""
    echo "=========================================="
    echo "           QCAT 服务状态检查"
    echo "=========================================="
    echo "启动时间: $(date '+%Y-%m-%d %H:%M:%S')"
    echo "运行模式: $([ "$DEV_MODE" = "true" ] && echo "开发模式" || ([ "$PRODUCTION_MODE" = "true" ] && echo "生产模式" || echo "标准模式"))"
    echo "启动服务: $SERVICES_TO_START"
    echo ""

    # 检查各个服务状态
    if [[ "$SERVICES_TO_START" == "all" || "$SERVICES_TO_START" == *"api"* ]]; then
        check_service_status "QCAT API" "$QCAT_API_PORT" "http://localhost:$QCAT_API_PORT/health" "$BACKEND_PID"
    fi

    if [[ "$SERVICES_TO_START" == "all" || "$SERVICES_TO_START" == *"optimizer"* ]]; then
        check_service_status "QCAT Optimizer" "$QCAT_OPTIMIZER_PORT" "http://localhost:$QCAT_OPTIMIZER_PORT/health" "$OPTIMIZER_PID"
    fi

    if [[ "$SERVICES_TO_START" == "all" || "$SERVICES_TO_START" == *"frontend"* ]]; then
        check_service_status "Frontend" "$FRONTEND_DEV_PORT" "" "$FRONTEND_PID"
    fi

    # 显示系统资源信息
    if [ "$DEBUG_MODE" = "true" ]; then
        echo ""
        echo "📊 系统资源信息:"
        if command -v free >/dev/null 2>&1; then
            echo "   内存使用: $(free -h | awk 'NR==2{printf "%.1f%%", $3*100/$2 }')"
        fi
        if command -v df >/dev/null 2>&1; then
            echo "   磁盘使用: $(df -h . | awk 'NR==2{print $5}')"
        fi

        echo ""
        echo "📊 端口连接统计:"
        if command -v netstat >/dev/null 2>&1; then
            [ -n "$QCAT_API_PORT" ] && echo "   API端口 $QCAT_API_PORT: $(netstat -ano 2>/dev/null | grep :$QCAT_API_PORT | wc -l) 个连接"
            [ -n "$QCAT_OPTIMIZER_PORT" ] && echo "   优化器端口 $QCAT_OPTIMIZER_PORT: $(netstat -ano 2>/dev/null | grep :$QCAT_OPTIMIZER_PORT | wc -l) 个连接"
            [ -n "$FRONTEND_DEV_PORT" ] && echo "   前端端口 $FRONTEND_DEV_PORT: $(netstat -ano 2>/dev/null | grep :$FRONTEND_DEV_PORT | wc -l) 个连接"
        fi
    fi

    echo ""
    echo "=========================================="
    echo "🌐 访问地址:"
    [[ "$SERVICES_TO_START" == "all" || "$SERVICES_TO_START" == *"frontend"* ]] && echo "   前端应用: http://localhost:$FRONTEND_DEV_PORT"
    [[ "$SERVICES_TO_START" == "all" || "$SERVICES_TO_START" == *"api"* ]] && echo "   后端API:  http://localhost:$QCAT_API_PORT"
    [[ "$SERVICES_TO_START" == "all" || "$SERVICES_TO_START" == *"optimizer"* ]] && echo "   优化器:   http://localhost:$QCAT_OPTIMIZER_PORT"
    echo ""
    echo "🛑 停止服务: Ctrl+C"
    echo "🔧 调试模式: 使用 --debug 参数重新启动"
    echo "=========================================="
}

# 清理函数（增强版）
cleanup() {
    log_info "正在停止服务..."

    local pids_to_kill=()
    [ -n "$FRONTEND_PID" ] && pids_to_kill+=("$FRONTEND_PID:Frontend")
    [ -n "$OPTIMIZER_PID" ] && pids_to_kill+=("$OPTIMIZER_PID:Optimizer")
    [ -n "$BACKEND_PID" ] && pids_to_kill+=("$BACKEND_PID:API")

    # 优雅关闭
    for pid_info in "${pids_to_kill[@]}"; do
        local pid="${pid_info%%:*}"
        local name="${pid_info##*:}"

        if kill -0 "$pid" 2>/dev/null; then
            log_info "优雅关闭 $name (PID: $pid)"
            kill -TERM "$pid" 2>/dev/null || true
        fi
    done

    # 等待进程关闭
    sleep 3

    # 强制关闭仍在运行的进程
    for pid_info in "${pids_to_kill[@]}"; do
        local pid="${pid_info%%:*}"
        local name="${pid_info##*:}"

        if kill -0 "$pid" 2>/dev/null; then
            log_warning "强制关闭 $name (PID: $pid)"
            kill -9 "$pid" 2>/dev/null || true
        fi
    done

    # 清理端口（如果进程仍占用）
    if [ -n "$QCAT_API_PORT" ]; then
        check_port "$QCAT_API_PORT" "API" true >/dev/null 2>&1 || true
    fi
    if [ -n "$QCAT_OPTIMIZER_PORT" ]; then
        check_port "$QCAT_OPTIMIZER_PORT" "Optimizer" true >/dev/null 2>&1 || true
    fi
    if [ -n "$FRONTEND_DEV_PORT" ]; then
        check_port "$FRONTEND_DEV_PORT" "Frontend" true >/dev/null 2>&1 || true
    fi

    log_success "🛑 所有服务已停止"
    exit 0
}

# 主函数（增强版）
main() {
    # 显示脚本信息
    echo "=========================================="
    echo "  $SCRIPT_NAME v$SCRIPT_VERSION"
    echo "=========================================="

    # 解析命令行参数
    parse_args "$@"

    # 设置信号处理
    trap cleanup SIGINT SIGTERM

    # 执行启动流程
    log_info "开始启动流程..."

    detect_os || { log_error "操作系统检测失败"; exit 1; }
    read_port_config || { log_error "端口配置读取失败"; exit 1; }
    check_dependencies || { log_error "依赖检查失败"; exit 1; }

    if [[ "$SERVICES_TO_START" == "all" || "$SERVICES_TO_START" == *"api"* || "$SERVICES_TO_START" == *"optimizer"* ]]; then
        install_dependencies
        setup_config
        start_database
        init_database
        build_binaries
    elif [[ "$SERVICES_TO_START" == *"frontend"* ]]; then
        # 只启动前端时，仍需要安装前端依赖
        install_dependencies
        setup_config
    fi

    start_services
    show_status

    echo ""
    echo "=========================================="
    log_success "QCAT 开发环境启动完成！"
    echo "=========================================="
    log_info "按 Ctrl+C 停止所有服务"
    echo ""

    # 等待信号
    wait
}

# 启动主函数
main "$@"

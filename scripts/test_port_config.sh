#!/usr/bin/env bash
# 端口配置测试脚本

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

# 读取配置文件中的端口信息
read_port_config() {
    # 默认端口
    QCAT_API_PORT=8082
    QCAT_OPTIMIZER_PORT=8081
    FRONTEND_DEV_PORT=3000
    
    # 尝试从config.yaml读取端口配置
    if [ -f "configs/config.yaml" ]; then
        # 使用grep和sed来解析
        QCAT_API_PORT=$(grep -A 20 "^ports:" configs/config.yaml | grep "qcat_api:" | sed 's/.*qcat_api: *\([0-9]*\).*/\1/' | head -1)
        QCAT_OPTIMIZER_PORT=$(grep -A 20 "^ports:" configs/config.yaml | grep "qcat_optimizer:" | sed 's/.*qcat_optimizer: *\([0-9]*\).*/\1/' | head -1)
        FRONTEND_DEV_PORT=$(grep -A 20 "^ports:" configs/config.yaml | grep "frontend_dev:" | sed 's/.*frontend_dev: *\([0-9]*\).*/\1/' | head -1)
        
        # 如果解析失败，使用默认值
        [ -z "$QCAT_API_PORT" ] && QCAT_API_PORT=8082
        [ -z "$QCAT_OPTIMIZER_PORT" ] && QCAT_OPTIMIZER_PORT=8081
        [ -z "$FRONTEND_DEV_PORT" ] && FRONTEND_DEV_PORT=3000
    fi
    
    # 从环境变量覆盖（如果设置了）
    [ ! -z "$QCAT_PORTS_QCAT_API" ] && QCAT_API_PORT=$QCAT_PORTS_QCAT_API
    [ ! -z "$QCAT_PORTS_QCAT_OPTIMIZER" ] && QCAT_OPTIMIZER_PORT=$QCAT_PORTS_QCAT_OPTIMIZER
    [ ! -z "$QCAT_PORTS_FRONTEND_DEV" ] && FRONTEND_DEV_PORT=$QCAT_PORTS_FRONTEND_DEV
    
    log_info "端口配置: API=$QCAT_API_PORT, 优化器=$QCAT_OPTIMIZER_PORT, 前端=$FRONTEND_DEV_PORT"
}

# 测试配置文件解析
test_config_parsing() {
    log_info "测试配置文件解析..."
    
    if [ ! -f "configs/config.yaml" ]; then
        log_error "配置文件 configs/config.yaml 不存在"
        return 1
    fi
    
    # 检查配置文件中是否包含端口配置
    if grep -q "^ports:" configs/config.yaml; then
        log_success "找到端口配置段"
    else
        log_error "配置文件中未找到端口配置段"
        return 1
    fi
    
    # 检查各个端口配置
    local ports=("qcat_api" "qcat_optimizer" "postgres" "redis" "prometheus" "grafana" "alertmanager" "nginx_http" "nginx_https" "frontend_dev")
    
    for port in "${ports[@]}"; do
        if grep -A 20 "^ports:" configs/config.yaml | grep -q "$port:"; then
            local port_value=$(grep -A 20 "^ports:" configs/config.yaml | grep "$port:" | sed "s/.*$port: *\([0-9]*\).*/\1/" | head -1)
            log_success "端口配置 $port: $port_value"
        else
            log_warning "未找到端口配置: $port"
        fi
    done
}

# 测试环境变量覆盖
test_env_override() {
    log_info "测试环境变量覆盖..."
    
    # 设置测试环境变量
    export QCAT_PORTS_QCAT_API=9082
    export QCAT_PORTS_QCAT_OPTIMIZER=9081
    
    read_port_config
    
    if [ "$QCAT_API_PORT" = "9082" ]; then
        log_success "环境变量覆盖 API 端口成功: $QCAT_API_PORT"
    else
        log_error "环境变量覆盖 API 端口失败: 期望 9082, 实际 $QCAT_API_PORT"
    fi
    
    if [ "$QCAT_OPTIMIZER_PORT" = "9081" ]; then
        log_success "环境变量覆盖优化器端口成功: $QCAT_OPTIMIZER_PORT"
    else
        log_error "环境变量覆盖优化器端口失败: 期望 9081, 实际 $QCAT_OPTIMIZER_PORT"
    fi
    
    # 清理测试环境变量
    unset QCAT_PORTS_QCAT_API
    unset QCAT_PORTS_QCAT_OPTIMIZER
}

# 测试前端配置
test_frontend_config() {
    log_info "测试前端配置..."
    
    if [ -f "frontend/.env.local" ]; then
        if grep -q "NEXT_PUBLIC_API_URL" frontend/.env.local; then
            local api_url=$(grep "NEXT_PUBLIC_API_URL" frontend/.env.local | cut -d'=' -f2)
            log_success "前端 API URL 配置: $api_url"
        else
            log_warning "前端配置文件中未找到 NEXT_PUBLIC_API_URL"
        fi
    else
        log_warning "前端配置文件 frontend/.env.local 不存在"
    fi
    
    if [ -f "frontend/.env.example" ]; then
        log_success "前端配置模板文件存在"
    else
        log_warning "前端配置模板文件 frontend/.env.example 不存在"
    fi
}

# 测试Docker配置
test_docker_config() {
    log_info "测试Docker配置..."
    
    if [ -f "deploy/docker-compose.prod.yml" ]; then
        if grep -q "QCAT_PORTS_" deploy/docker-compose.prod.yml; then
            log_success "Docker配置文件包含端口环境变量"
        else
            log_warning "Docker配置文件中未找到端口环境变量"
        fi
    else
        log_warning "Docker配置文件 deploy/docker-compose.prod.yml 不存在"
    fi
}

# 主测试函数
main() {
    echo "=========================================="
    echo "           QCAT 端口配置测试"
    echo "=========================================="
    
    test_config_parsing
    echo
    
    read_port_config
    echo
    
    test_env_override
    echo
    
    test_frontend_config
    echo
    
    test_docker_config
    echo
    
    log_success "端口配置测试完成"
}

main "$@"

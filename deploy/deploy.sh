#!/bin/bash

# QCAT 自动化部署脚本
# 支持多种部署方式：Docker Compose、Kubernetes、传统服务器

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
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

# 显示帮助信息
show_help() {
    cat << EOF
QCAT 自动化部署脚本

用法: $0 [选项] [环境]

选项:
    -h, --help          显示此帮助信息
    -e, --env ENV       指定部署环境 (staging|production)
    -m, --method METHOD 指定部署方式 (docker|k8s|server)
    -f, --force         强制部署，跳过确认
    -r, --rollback      回滚到上一个版本
    -v, --version       显示版本信息

环境:
    staging     部署到测试环境
    production  部署到生产环境

部署方式:
    docker      使用Docker Compose部署
    k8s         使用Kubernetes部署
    server      部署到传统服务器

示例:
    $0 -e staging -m docker
    $0 -e production -m k8s --force
    $0 -e production -m server --rollback

EOF
}

# 检查依赖
check_dependencies() {
    log_info "检查部署依赖..."
    
    case $DEPLOY_METHOD in
        docker)
            if ! command -v docker &> /dev/null; then
                log_error "Docker 未安装"
                exit 1
            fi
            if ! command -v docker-compose &> /dev/null; then
                log_error "Docker Compose 未安装"
                exit 1
            fi
            ;;
        k8s)
            if ! command -v kubectl &> /dev/null; then
                log_error "kubectl 未安装"
                exit 1
            fi
            ;;
        server)
            if ! command -v ssh &> /dev/null; then
                log_error "SSH 未安装"
                exit 1
            fi
            ;;
    esac
    
    log_success "依赖检查通过"
}

# 加载环境变量
load_env() {
    local env_file=".env.${DEPLOY_ENV}"
    if [ -f "$env_file" ]; then
        log_info "加载环境变量: $env_file"
        export $(cat "$env_file" | grep -v '^#' | xargs)
    else
        log_warning "环境变量文件不存在: $env_file"
    fi
}

# Docker Compose 部署
deploy_docker() {
    log_info "使用 Docker Compose 部署到 $DEPLOY_ENV 环境"
    
    # 检查Docker Compose文件
    local compose_file="docker-compose.${DEPLOY_ENV}.yml"
    if [ ! -f "$compose_file" ]; then
        log_error "Docker Compose 文件不存在: $compose_file"
        exit 1
    fi
    
    # 停止现有服务
    log_info "停止现有服务..."
    docker-compose -f "$compose_file" down
    
    # 拉取最新镜像
    log_info "拉取最新镜像..."
    docker-compose -f "$compose_file" pull
    
    # 启动服务
    log_info "启动服务..."
    docker-compose -f "$compose_file" up -d
    
    # 等待服务启动
    log_info "等待服务启动..."
    sleep 30
    
    # 健康检查
    health_check_docker
}

# Docker 健康检查
health_check_docker() {
    log_info "执行健康检查..."
    
    local services=("backend" "frontend" "postgres" "redis")
    local failed_services=()
    
    for service in "${services[@]}"; do
        if docker-compose -f "docker-compose.${DEPLOY_ENV}.yml" ps "$service" | grep -q "Up"; then
            log_success "服务 $service 运行正常"
        else
            log_error "服务 $service 运行异常"
            failed_services+=("$service")
        fi
    done
    
    if [ ${#failed_services[@]} -gt 0 ]; then
        log_error "以下服务运行异常: ${failed_services[*]}"
        return 1
    fi
    
    log_success "所有服务健康检查通过"
}

# Kubernetes 部署
deploy_k8s() {
    log_info "使用 Kubernetes 部署到 $DEPLOY_ENV 环境"
    
    # 检查kubectl配置
    if ! kubectl cluster-info &> /dev/null; then
        log_error "无法连接到 Kubernetes 集群"
        exit 1
    fi
    
    # 创建命名空间
    kubectl create namespace qcat-${DEPLOY_ENV} --dry-run=client -o yaml | kubectl apply -f -
    
    # 应用配置
    log_info "应用 Kubernetes 配置..."
    kubectl apply -f k8s/${DEPLOY_ENV}/ -n qcat-${DEPLOY_ENV}
    
    # 等待部署完成
    log_info "等待部署完成..."
    kubectl rollout status deployment/qcat-backend -n qcat-${DEPLOY_ENV} --timeout=300s
    kubectl rollout status deployment/qcat-frontend -n qcat-${DEPLOY_ENV} --timeout=300s
    
    # 健康检查
    health_check_k8s
}

# Kubernetes 健康检查
health_check_k8s() {
    log_info "执行 Kubernetes 健康检查..."
    
    local namespace="qcat-${DEPLOY_ENV}"
    
    # 检查Pod状态
    local pods=$(kubectl get pods -n "$namespace" -l app=qcat -o jsonpath='{.items[*].metadata.name}')
    for pod in $pods; do
        local status=$(kubectl get pod "$pod" -n "$namespace" -o jsonpath='{.status.phase}')
        if [ "$status" = "Running" ]; then
            log_success "Pod $pod 运行正常"
        else
            log_error "Pod $pod 状态异常: $status"
            return 1
        fi
    done
    
    # 检查服务端点
    local services=("qcat-backend" "qcat-frontend")
    for service in "${services[@]}"; do
        if kubectl get endpoints "$service" -n "$namespace" | grep -q "ENDPOINTS"; then
            log_success "服务 $service 端点正常"
        else
            log_error "服务 $service 端点异常"
            return 1
        fi
    done
    
    log_success "Kubernetes 健康检查通过"
}

# 传统服务器部署
deploy_server() {
    log_info "部署到传统服务器 ($DEPLOY_ENV 环境)"
    
    # 获取服务器配置
    local host_var="${DEPLOY_ENV^^}_HOST"
    local user_var="${DEPLOY_ENV^^}_USER"
    local ssh_key_var="${DEPLOY_ENV^^}_SSH_KEY"
    
    local host="${!host_var}"
    local user="${!user_var}"
    local ssh_key="${!ssh_key_var}"
    
    if [ -z "$host" ] || [ -z "$user" ]; then
        log_error "服务器配置不完整"
        exit 1
    fi
    
    # 上传文件
    log_info "上传部署文件到服务器..."
    scp -i "$ssh_key" -r deploy/ "$user@$host:/tmp/qcat-deploy/"
    
    # 执行部署脚本
    log_info "在服务器上执行部署..."
    ssh -i "$ssh_key" "$user@$host" << EOF
        cd /tmp/qcat-deploy
        chmod +x deploy.sh
        ./deploy.sh --env $DEPLOY_ENV --method server
EOF
    
    # 健康检查
    health_check_server
}

# 服务器健康检查
health_check_server() {
    log_info "执行服务器健康检查..."
    
    local host_var="${DEPLOY_ENV^^}_HOST"
    local host="${!host_var}"
    
    # 检查服务状态
    local services=("qcat-backend" "qcat-frontend" "nginx")
    for service in "${services[@]}"; do
        if ssh "$user@$host" "systemctl is-active --quiet $service"; then
            log_success "服务 $service 运行正常"
        else
            log_error "服务 $service 运行异常"
            return 1
        fi
    done
    
    # 检查端口
    local ports=(80 443 8080)
    for port in "${ports[@]}"; do
        if ssh "$user@$host" "netstat -tlnp | grep :$port"; then
            log_success "端口 $port 监听正常"
        else
            log_error "端口 $port 监听异常"
            return 1
        fi
    done
    
    log_success "服务器健康检查通过"
}

# 回滚部署
rollback() {
    log_info "执行回滚操作..."
    
    case $DEPLOY_METHOD in
        docker)
            docker-compose -f "docker-compose.${DEPLOY_ENV}.yml" down
            docker-compose -f "docker-compose.${DEPLOY_ENV}.yml" up -d --force-recreate
            ;;
        k8s)
            kubectl rollout undo deployment/qcat-backend -n qcat-${DEPLOY_ENV}
            kubectl rollout undo deployment/qcat-frontend -n qcat-${DEPLOY_ENV}
            ;;
        server)
            local host_var="${DEPLOY_ENV^^}_HOST"
            local user_var="${DEPLOY_ENV^^}_USER"
            local ssh_key_var="${DEPLOY_ENV^^}_SSH_KEY"
            
            local host="${!host_var}"
            local user="${!user_var}"
            local ssh_key="${!ssh_key_var}"
            
            ssh -i "$ssh_key" "$user@$host" << EOF
                # 恢复备份
                sudo systemctl stop qcat-backend qcat-frontend
                sudo cp /opt/qcat-backup/* /opt/qcat/
                sudo systemctl start qcat-backend qcat-frontend
EOF
            ;;
    esac
    
    log_success "回滚完成"
}

# 主函数
main() {
    # 解析命令行参数
    DEPLOY_ENV=""
    DEPLOY_METHOD="docker"
    FORCE_DEPLOY=false
    ROLLBACK=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -e|--env)
                DEPLOY_ENV="$2"
                shift 2
                ;;
            -m|--method)
                DEPLOY_METHOD="$2"
                shift 2
                ;;
            -f|--force)
                FORCE_DEPLOY=true
                shift
                ;;
            -r|--rollback)
                ROLLBACK=true
                shift
                ;;
            -v|--version)
                echo "QCAT 部署脚本 v1.0.0"
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 验证参数
    if [ -z "$DEPLOY_ENV" ]; then
        log_error "请指定部署环境"
        show_help
        exit 1
    fi
    
    if [[ ! "$DEPLOY_ENV" =~ ^(staging|production)$ ]]; then
        log_error "无效的部署环境: $DEPLOY_ENV"
        exit 1
    fi
    
    if [[ ! "$DEPLOY_METHOD" =~ ^(docker|k8s|server)$ ]]; then
        log_error "无效的部署方式: $DEPLOY_METHOD"
        exit 1
    fi
    
    # 显示部署信息
    log_info "部署环境: $DEPLOY_ENV"
    log_info "部署方式: $DEPLOY_METHOD"
    
    # 确认部署
    if [ "$FORCE_DEPLOY" = false ] && [ "$ROLLBACK" = false ]; then
        read -p "确认部署到 $DEPLOY_ENV 环境? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "部署已取消"
            exit 0
        fi
    fi
    
    # 检查依赖
    check_dependencies
    
    # 加载环境变量
    load_env
    
    # 执行部署或回滚
    if [ "$ROLLBACK" = true ]; then
        rollback
    else
        case $DEPLOY_METHOD in
            docker)
                deploy_docker
                ;;
            k8s)
                deploy_k8s
                ;;
            server)
                deploy_server
                ;;
        esac
    fi
    
    log_success "部署完成!"
}

# 执行主函数
main "$@"

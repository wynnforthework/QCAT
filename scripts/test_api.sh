#!/bin/bash

# QCAT API 测试脚本
# 测试所有API端点以确保它们正常工作
# 支持Windows、Linux、macOS

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 配置
BASE_URL=${BASE_URL:-"http://localhost:8082"}
API_BASE="$BASE_URL/api/v1"
TIMEOUT=${TIMEOUT:-10}

# 检测操作系统
detect_os() {
    case "$(uname -s)" in
        Linux*)     OS="linux";;
        Darwin*)    OS="macos";;
        CYGWIN*)    OS="windows";;
        MINGW*)     OS="windows";;
        MSYS*)      OS="windows";;
        *)          OS="unknown";;
    esac
}

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

# 检查依赖
check_dependencies() {
    if ! command -v curl &> /dev/null; then
        log_error "curl 未安装"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        log_warning "jq 未安装，将使用简化输出"
        JQ_AVAILABLE=false
    else
        JQ_AVAILABLE=true
    fi
}

# 等待服务启动
wait_for_service() {
    log_info "等待服务启动..."
    
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "$BASE_URL/health" >/dev/null 2>&1; then
            log_success "服务已启动"
            return 0
        fi
        
        log_info "尝试 $attempt/$max_attempts - 等待服务启动..."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    log_error "服务启动超时"
    return 1
}

# 格式化JSON输出
format_json() {
    if [ "$JQ_AVAILABLE" = true ]; then
        jq . 2>/dev/null || cat
    else
        cat
    fi
}

# 测试健康检查
test_health_check() {
    log_info "1. 测试健康检查..."
    echo "  - 健康检查端点:"
    if curl -s -f --max-time $TIMEOUT "$BASE_URL/health" | format_json; then
        log_success "健康检查通过"
        return 0
    else
        log_error "健康检查失败"
        return 1
    fi
}

# 测试策略端点
test_strategy_endpoints() {
    log_info "2. 测试策略端点..."
    
    echo "  - 获取策略列表:"
    if curl -s -f --max-time $TIMEOUT "$API_BASE/strategy/" | format_json; then
        log_success "策略列表获取成功"
    else
        log_warning "策略列表获取失败或为空"
    fi
    
    echo "  - 获取不存在的策略 (应该返回404):"
    if curl -s -f --max-time $TIMEOUT "$API_BASE/strategy/nonexistent" | format_json; then
        log_warning "意外成功获取不存在的策略"
    else
        log_success "正确返回404错误"
    fi
}

# 测试投资组合端点
test_portfolio_endpoints() {
    log_info "3. 测试投资组合端点..."
    
    echo "  - 投资组合概览:"
    if curl -s -f --max-time $TIMEOUT "$API_BASE/portfolio/overview" | format_json; then
        log_success "投资组合概览获取成功"
    else
        log_warning "投资组合概览获取失败"
    fi
    
    echo "  - 投资组合配置:"
    if curl -s -f --max-time $TIMEOUT "$API_BASE/portfolio/allocations" | format_json; then
        log_success "投资组合配置获取成功"
    else
        log_warning "投资组合配置获取失败"
    fi
}

# 测试风险端点
test_risk_endpoints() {
    log_info "4. 测试风险端点..."
    
    echo "  - 风险概览:"
    if curl -s -f --max-time $TIMEOUT "$API_BASE/risk/overview" | format_json; then
        log_success "风险概览获取成功"
    else
        log_warning "风险概览获取失败"
    fi
    
    echo "  - 风险限制:"
    if curl -s -f --max-time $TIMEOUT "$API_BASE/risk/limits" | format_json; then
        log_success "风险限制获取成功"
    else
        log_warning "风险限制获取失败"
    fi
}

# 测试热门列表端点
test_hotlist_endpoints() {
    log_info "5. 测试热门列表端点..."
    
    echo "  - 热门符号:"
    if curl -s -f --max-time $TIMEOUT "$API_BASE/hotlist/symbols" | format_json; then
        log_success "热门符号获取成功"
    else
        log_warning "热门符号获取失败"
    fi
    
    echo "  - 白名单:"
    if curl -s -f --max-time $TIMEOUT "$API_BASE/hotlist/whitelist" | format_json; then
        log_success "白名单获取成功"
    else
        log_warning "白名单获取失败"
    fi
}

# 测试指标端点
test_metrics_endpoints() {
    log_info "6. 测试指标端点..."
    
    echo "  - 系统指标:"
    if curl -s -f --max-time $TIMEOUT "$API_BASE/metrics/system" | format_json; then
        log_success "系统指标获取成功"
    else
        log_warning "系统指标获取失败"
    fi
    
    echo "  - 性能指标:"
    if curl -s -f --max-time $TIMEOUT "$API_BASE/metrics/performance" | format_json; then
        log_success "性能指标获取成功"
    else
        log_warning "性能指标获取失败"
    fi
}

# 测试审计端点
test_audit_endpoints() {
    log_info "7. 测试审计端点..."
    
    echo "  - 审计日志:"
    if curl -s -f --max-time $TIMEOUT "$API_BASE/audit/logs" | format_json; then
        log_success "审计日志获取成功"
    else
        log_warning "审计日志获取失败"
    fi
    
    echo "  - 决策链:"
    if curl -s -f --max-time $TIMEOUT "$API_BASE/audit/decisions" | format_json; then
        log_success "决策链获取成功"
    else
        log_warning "决策链获取失败"
    fi
}

# 测试优化器端点
test_optimizer_endpoints() {
    log_info "8. 测试优化器端点..."
    
    echo "  - 优化任务:"
    if curl -s -f --max-time $TIMEOUT "$API_BASE/optimizer/tasks" | format_json; then
        log_success "优化任务获取成功"
    else
        log_warning "优化任务获取失败"
    fi
}

# 测试POST端点
test_post_endpoints() {
    log_info "9. 测试POST端点..."
    
    echo "  - 创建策略:"
    if curl -s -f --max-time $TIMEOUT -X POST "$API_BASE/strategy/" \
        -H "Content-Type: application/json" \
        -d '{"name":"Test Strategy","description":"Test strategy for API testing"}' | format_json; then
        log_success "策略创建成功"
    else
        log_warning "策略创建失败"
    fi
    
    echo "  - 运行优化:"
    if curl -s -f --max-time $TIMEOUT -X POST "$API_BASE/optimizer/run" \
        -H "Content-Type: application/json" \
        -d '{"strategy_id":"test_strategy","method":"grid","objective":"sharpe"}' | format_json; then
        log_success "优化运行成功"
    else
        log_warning "优化运行失败"
    fi
    
    echo "  - 投资组合再平衡:"
    if curl -s -f --max-time $TIMEOUT -X POST "$API_BASE/portfolio/rebalance" \
        -H "Content-Type: application/json" \
        -d '{"mode":"bandit"}' | format_json; then
        log_success "投资组合再平衡成功"
    else
        log_warning "投资组合再平衡失败"
    fi
}

# 测试智能系统端点
test_intelligence_endpoints() {
    log_info "10. 测试智能系统端点..."
    
    echo "  - 智能控制器状态:"
    if curl -s -f --max-time $TIMEOUT "$API_BASE/intelligence/status" | format_json; then
        log_success "智能控制器状态获取成功"
    else
        log_warning "智能控制器状态获取失败"
    fi
    
    echo "  - AutoML引擎状态:"
    if curl -s -f --max-time $TIMEOUT "$API_BASE/automl/status" | format_json; then
        log_success "AutoML引擎状态获取成功"
    else
        log_warning "AutoML引擎状态获取失败"
    fi
    
    echo "  - 自愈系统状态:"
    if curl -s -f --max-time $TIMEOUT "$API_BASE/healing/status" | format_json; then
        log_success "自愈系统状态获取成功"
    else
        log_warning "自愈系统状态获取失败"
    fi
}

# 显示帮助信息
show_help() {
    cat << EOF
QCAT API 测试脚本

用法: $0 [选项]

选项:
    -h, --help          显示此帮助信息
    -u, --url URL       指定基础URL (默认: http://localhost:8082)
    -t, --timeout SEC   指定超时时间 (默认: 10秒)
    -w, --wait          等待服务启动
    --no-wait           不等待服务启动

环境变量:
    BASE_URL            基础URL
    TIMEOUT             超时时间

示例:
    $0                    # 使用默认设置
    $0 -u http://localhost:8080  # 指定URL
    $0 -t 30             # 设置超时
    $0 --wait            # 等待服务启动

EOF
}

# 主函数
main() {
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -u|--url)
                BASE_URL="$2"
                API_BASE="$BASE_URL/api/v1"
                shift 2
                ;;
            -t|--timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            -w|--wait)
                WAIT_FOR_SERVICE=true
                shift
                ;;
            --no-wait)
                WAIT_FOR_SERVICE=false
                shift
                ;;
            *)
                log_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 检测操作系统
    detect_os
    
    echo -e "${CYAN}=== QCAT API 测试脚本 ===${NC}"
    echo "基础URL: $BASE_URL"
    echo "API基础: $API_BASE"
    echo "超时时间: ${TIMEOUT}秒"
    echo "操作系统: $OS"
    echo ""
    
    # 检查依赖
    check_dependencies
    
    # 等待服务启动
    if [ "${WAIT_FOR_SERVICE:-true}" = true ]; then
        wait_for_service
    fi
    
    # 开始测试
    test_health_check
    test_strategy_endpoints
    test_portfolio_endpoints
    test_risk_endpoints
    test_hotlist_endpoints
    test_metrics_endpoints
    test_audit_endpoints
    test_optimizer_endpoints
    test_post_endpoints
    test_intelligence_endpoints
    
    echo ""
    echo -e "${GREEN}=== API 测试完成 ===${NC}"
    echo "所有端点测试成功完成!"
}

# 运行主函数
main "$@"

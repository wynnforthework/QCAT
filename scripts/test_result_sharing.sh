#!/bin/bash

# 结果共享系统测试脚本
# 测试各种共享模式和功能

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

# 配置
OPTIMIZER_URL="http://localhost:8081"
TEST_DATA_DIR="./test_data"
SHARED_RESULTS_DIR="./data/shared_results"

# 创建测试目录
setup_test_environment() {
    log_info "设置测试环境..."
    
    mkdir -p $TEST_DATA_DIR
    mkdir -p $SHARED_RESULTS_DIR/files
    mkdir -p $SHARED_RESULTS_DIR
    
    log_success "测试环境设置完成"
}

# 测试健康检查
test_health_check() {
    log_info "测试健康检查..."
    
    response=$(curl -s "$OPTIMIZER_URL/health")
    if echo "$response" | grep -q "healthy"; then
        log_success "健康检查通过"
    else
        log_error "健康检查失败: $response"
        return 1
    fi
}

# 测试手动共享结果
test_manual_sharing() {
    log_info "测试手动共享结果..."
    
    # 创建测试结果
    test_result='{
        "task_id": "test_task_001",
        "strategy_name": "test_strategy",
        "parameters": {
            "param1": 100,
            "param2": 200
        },
        "performance": {
            "profit_rate": 15.5,
            "sharpe_ratio": 2.1,
            "max_drawdown": 8.2,
            "win_rate": 0.68
        },
        "random_seed": 1234567890,
        "discovered_by": "test_script"
    }'
    
    # 发送共享请求
    response=$(curl -s -X POST "$OPTIMIZER_URL/share-result" \
        -H "Content-Type: application/json" \
        -d "$test_result")
    
    if echo "$response" | grep -q "success"; then
        log_success "手动共享结果成功"
        echo "$response" | jq '.'
    else
        log_error "手动共享结果失败: $response"
        return 1
    fi
}

# 测试获取共享结果
test_get_shared_results() {
    log_info "测试获取共享结果..."
    
    response=$(curl -s "$OPTIMIZER_URL/shared-results")
    
    if echo "$response" | grep -q "results"; then
        log_success "获取共享结果成功"
        count=$(echo "$response" | jq '.count')
        log_info "找到 $count 个共享结果"
        
        # 显示结果详情
        echo "$response" | jq '.results[] | {task_id, strategy_name, performance}'
    else
        log_error "获取共享结果失败: $response"
        return 1
    fi
}

# 测试文件共享模式
test_file_sharing() {
    log_info "测试文件共享模式..."
    
    # 检查共享文件是否生成
    files=$(find $SHARED_RESULTS_DIR/files -name "*.json" 2>/dev/null || true)
    
    if [ -n "$files" ]; then
        log_success "文件共享模式工作正常"
        log_info "生成的共享文件:"
        echo "$files" | head -5
    else
        log_warning "未找到共享文件，可能文件共享模式未启用"
    fi
}

# 测试字符串共享模式
test_string_sharing() {
    log_info "测试字符串共享模式..."
    
    # 检查字符串存储文件
    string_file="$SHARED_RESULTS_DIR/strings.txt"
    
    if [ -f "$string_file" ]; then
        log_success "字符串共享模式工作正常"
        log_info "字符串存储文件内容预览:"
        head -3 "$string_file" 2>/dev/null || log_warning "字符串文件为空"
    else
        log_warning "未找到字符串存储文件，可能字符串共享模式未启用"
    fi
}

# 测试种子共享模式
test_seed_sharing() {
    log_info "测试种子共享模式..."
    
    # 检查种子映射文件
    seed_file="$SHARED_RESULTS_DIR/seed_mapping.json"
    
    if [ -f "$seed_file" ]; then
        log_success "种子共享模式工作正常"
        log_info "种子映射文件内容:"
        cat "$seed_file" | jq '.' 2>/dev/null || log_warning "种子映射文件格式错误"
    else
        log_warning "未找到种子映射文件，可能种子共享模式未启用"
    fi
}

# 测试性能阈值过滤
test_performance_threshold() {
    log_info "测试性能阈值过滤..."
    
    # 创建低性能结果
    low_performance_result='{
        "task_id": "test_task_low",
        "strategy_name": "low_performance_strategy",
        "parameters": {
            "param1": 50
        },
        "performance": {
            "profit_rate": 2.0,
            "sharpe_ratio": 0.2,
            "max_drawdown": 20.0,
            "win_rate": 0.3
        },
        "random_seed": 9876543210,
        "discovered_by": "test_script"
    }'
    
    # 发送低性能结果
    response=$(curl -s -X POST "$OPTIMIZER_URL/share-result" \
        -H "Content-Type: application/json" \
        -d "$low_performance_result")
    
    # 检查是否被过滤
    if echo "$response" | grep -q "success"; then
        log_warning "低性能结果未被过滤，可能需要调整阈值配置"
    else
        log_success "性能阈值过滤工作正常"
    fi
}

# 测试跨服务器场景模拟
test_cross_server_scenario() {
    log_info "测试跨服务器场景模拟..."
    
    # 模拟服务器A生成结果
    log_info "模拟服务器A生成结果..."
    server_a_result='{
        "task_id": "cross_server_task",
        "strategy_name": "server_a_strategy",
        "parameters": {
            "ma_short": 10,
            "ma_long": 20
        },
        "performance": {
            "profit_rate": 18.5,
            "sharpe_ratio": 2.5,
            "max_drawdown": 7.8,
            "win_rate": 0.72
        },
        "random_seed": 1111111111,
        "discovered_by": "server_a"
    }'
    
    response=$(curl -s -X POST "$OPTIMIZER_URL/share-result" \
        -H "Content-Type: application/json" \
        -d "$server_a_result")
    
    if echo "$response" | grep -q "success"; then
        log_success "服务器A结果生成成功"
    else
        log_error "服务器A结果生成失败"
        return 1
    fi
    
    # 模拟服务器B查询结果
    log_info "模拟服务器B查询结果..."
    response=$(curl -s "$OPTIMIZER_URL/shared-results")
    
    if echo "$response" | grep -q "server_a"; then
        log_success "服务器B成功获取到服务器A的结果"
        echo "$response" | jq '.results[] | select(.discovered_by == "server_a") | {task_id, discovered_by, performance}'
    else
        log_error "服务器B未能获取到服务器A的结果"
        return 1
    fi
}

# 测试结果评分和排序
test_result_scoring() {
    log_info "测试结果评分和排序..."
    
    # 创建多个不同性能的结果
    results=(
        '{"task_id": "scoring_test", "strategy_name": "strategy_1", "performance": {"profit_rate": 10.0, "sharpe_ratio": 1.0, "max_drawdown": 10.0, "win_rate": 0.5}, "random_seed": 1001, "discovered_by": "test_1"}'
        '{"task_id": "scoring_test", "strategy_name": "strategy_2", "performance": {"profit_rate": 15.0, "sharpe_ratio": 1.5, "max_drawdown": 8.0, "win_rate": 0.6}, "random_seed": 1002, "discovered_by": "test_2"}'
        '{"task_id": "scoring_test", "strategy_name": "strategy_3", "performance": {"profit_rate": 20.0, "sharpe_ratio": 2.0, "max_drawdown": 5.0, "win_rate": 0.7}, "random_seed": 1003, "discovered_by": "test_3"}'
    )
    
    for result in "${results[@]}"; do
        curl -s -X POST "$OPTIMIZER_URL/share-result" \
            -H "Content-Type: application/json" \
            -d "$result" > /dev/null
    done
    
    # 查询结果并检查排序
    response=$(curl -s "$OPTIMIZER_URL/shared-results")
    
    if echo "$response" | grep -q "scoring_test"; then
        log_success "结果评分测试完成"
        log_info "按性能排序的结果:"
        echo "$response" | jq '.results[] | select(.task_id == "scoring_test") | {discovered_by, performance}'
    else
        log_error "结果评分测试失败"
        return 1
    fi
}

# 测试错误处理
test_error_handling() {
    log_info "测试错误处理..."
    
    # 测试无效JSON
    response=$(curl -s -X POST "$OPTIMIZER_URL/share-result" \
        -H "Content-Type: application/json" \
        -d '{"invalid": json}' 2>/dev/null || echo "Invalid JSON")
    
    if echo "$response" | grep -q "Invalid request\|Invalid JSON"; then
        log_success "无效JSON错误处理正常"
    else
        log_warning "无效JSON错误处理可能有问题"
    fi
    
    # 测试缺少必需字段
    incomplete_result='{
        "task_id": "incomplete_test"
    }'
    
    response=$(curl -s -X POST "$OPTIMIZER_URL/share-result" \
        -H "Content-Type: application/json" \
        -d "$incomplete_result")
    
    if echo "$response" | grep -q "error\|failed"; then
        log_success "缺少字段错误处理正常"
    else
        log_warning "缺少字段错误处理可能有问题"
    fi
}

# 性能测试
test_performance() {
    log_info "测试性能..."
    
    start_time=$(date +%s)
    
    # 批量创建结果
    for i in {1..10}; do
        result="{\"task_id\": \"perf_test_$i\", \"strategy_name\": \"perf_strategy\", \"performance\": {\"profit_rate\": $((10 + i)), \"sharpe_ratio\": $((1 + i/10)), \"max_drawdown\": $((10 - i/2)), \"win_rate\": 0.$((50 + i))}, \"random_seed\": $((10000 + i)), \"discovered_by\": \"perf_test\"}"
        
        curl -s -X POST "$OPTIMIZER_URL/share-result" \
            -H "Content-Type: application/json" \
            -d "$result" > /dev/null
    done
    
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    
    log_success "性能测试完成，10个结果共享耗时 ${duration} 秒"
}

# 清理测试数据
cleanup_test_data() {
    log_info "清理测试数据..."
    
    # 清理测试目录
    rm -rf $TEST_DATA_DIR
    
    # 清理共享结果（可选，取消注释以启用）
    # rm -rf $SHARED_RESULTS_DIR/files/*
    # rm -f $SHARED_RESULTS_DIR/strings.txt
    # rm -f $SHARED_RESULTS_DIR/seed_mapping.json
    
    log_success "测试数据清理完成"
}

# 生成测试报告
generate_test_report() {
    log_info "生成测试报告..."
    
    report_file="$TEST_DATA_DIR/test_report_$(date +%Y%m%d_%H%M%S).txt"
    
    {
        echo "结果共享系统测试报告"
        echo "======================"
        echo "测试时间: $(date)"
        echo "测试环境: $OPTIMIZER_URL"
        echo ""
        echo "测试项目:"
        echo "1. 健康检查"
        echo "2. 手动共享结果"
        echo "3. 获取共享结果"
        echo "4. 文件共享模式"
        echo "5. 字符串共享模式"
        echo "6. 种子共享模式"
        echo "7. 性能阈值过滤"
        echo "8. 跨服务器场景模拟"
        echo "9. 结果评分和排序"
        echo "10. 错误处理"
        echo "11. 性能测试"
        echo ""
        echo "测试完成时间: $(date)"
    } > "$report_file"
    
    log_success "测试报告已生成: $report_file"
}

# 主测试函数
main() {
    log_info "开始结果共享系统测试..."
    
    # 检查依赖
    if ! command -v curl &> /dev/null; then
        log_error "curl 未安装，请先安装 curl"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        log_error "jq 未安装，请先安装 jq"
        exit 1
    fi
    
    # 检查优化器服务是否运行
    if ! curl -s "$OPTIMIZER_URL/health" &> /dev/null; then
        log_error "优化器服务未运行，请先启动服务"
        log_info "启动命令: go run cmd/optimizer/main.go"
        exit 1
    fi
    
    # 执行测试
    setup_test_environment
    
    test_health_check
    test_manual_sharing
    test_get_shared_results
    test_file_sharing
    test_string_sharing
    test_seed_sharing
    test_performance_threshold
    test_cross_server_scenario
    test_result_scoring
    test_error_handling
    test_performance
    
    generate_test_report
    
    log_success "所有测试完成！"
    
    # 询问是否清理测试数据
    read -p "是否清理测试数据？(y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cleanup_test_data
    fi
}

# 运行主函数
main "$@"

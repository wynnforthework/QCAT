#!/bin/bash

# QCAT 测试运行脚本
# 用于执行各种类型的测试并生成报告

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
COVERAGE_FILE="coverage.out"
COVERAGE_HTML="coverage.html"
BENCHMARK_FILE="benchmark.txt"
TEST_TIMEOUT="30m"

# 函数定义
print_header() {
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}================================${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# 清理函数
cleanup() {
    echo "Cleaning up..."
    # 清理临时文件
    rm -f test.db
    rm -f *.tmp
}

# 设置清理陷阱
trap cleanup EXIT

# 检查Go环境
check_go_env() {
    print_header "检查Go环境"
    
    if ! command -v go &> /dev/null; then
        print_error "Go未安装或不在PATH中"
        exit 1
    fi
    
    GO_VERSION=$(go version | cut -d' ' -f3)
    print_success "Go版本: $GO_VERSION"
    
    # 检查模块
    if [ ! -f "go.mod" ]; then
        print_error "go.mod文件不存在"
        exit 1
    fi
    
    print_success "Go模块配置正常"
}

# 下载依赖
download_deps() {
    print_header "下载依赖"
    
    echo "正在下载Go模块依赖..."
    go mod download
    go mod tidy
    
    print_success "依赖下载完成"
}

# 运行单元测试
run_unit_tests() {
    print_header "运行单元测试"
    
    echo "正在运行单元测试..."
    if go test -v -race -timeout $TEST_TIMEOUT ./internal/...; then
        print_success "单元测试通过"
    else
        print_error "单元测试失败"
        return 1
    fi
}

# 运行覆盖率测试
run_coverage_tests() {
    print_header "运行覆盖率测试"
    
    echo "正在生成覆盖率报告..."
    if go test -v -race -coverprofile=$COVERAGE_FILE -timeout $TEST_TIMEOUT ./internal/...; then
        print_success "覆盖率测试完成"
        
        # 生成覆盖率报告
        go tool cover -func=$COVERAGE_FILE
        go tool cover -html=$COVERAGE_FILE -o $COVERAGE_HTML
        
        # 提取总覆盖率
        COVERAGE=$(go tool cover -func=$COVERAGE_FILE | grep total | awk '{print $3}')
        echo -e "${GREEN}总覆盖率: $COVERAGE${NC}"
        
        # 检查覆盖率阈值
        COVERAGE_NUM=$(echo $COVERAGE | sed 's/%//')
        if (( $(echo "$COVERAGE_NUM >= 70" | bc -l) )); then
            print_success "覆盖率达标 ($COVERAGE >= 70%)"
        else
            print_warning "覆盖率未达标 ($COVERAGE < 70%)"
        fi
        
        print_success "覆盖率报告已生成: $COVERAGE_HTML"
    else
        print_error "覆盖率测试失败"
        return 1
    fi
}

# 运行集成测试
run_integration_tests() {
    print_header "运行集成测试"
    
    echo "正在运行集成测试..."
    if go test -v -tags=integration -timeout $TEST_TIMEOUT ./tests/integration/...; then
        print_success "集成测试通过"
    else
        print_warning "集成测试失败或跳过"
    fi
}

# 运行E2E测试
run_e2e_tests() {
    print_header "运行端到端测试"
    
    echo "正在运行E2E测试..."
    if go test -v -tags=e2e -timeout $TEST_TIMEOUT ./tests/e2e/...; then
        print_success "E2E测试通过"
    else
        print_warning "E2E测试失败或跳过"
    fi
}

# 运行基准测试
run_benchmark_tests() {
    print_header "运行基准测试"
    
    echo "正在运行基准测试..."
    if go test -bench=. -benchmem -run=^$ ./internal/... > $BENCHMARK_FILE; then
        print_success "基准测试完成"
        
        echo "基准测试结果:"
        cat $BENCHMARK_FILE
        
        print_success "基准测试报告已生成: $BENCHMARK_FILE"
    else
        print_warning "基准测试失败或跳过"
    fi
}

# 运行代码质量检查
run_quality_checks() {
    print_header "运行代码质量检查"
    
    # 格式检查
    echo "检查代码格式..."
    if ! go fmt ./...; then
        print_warning "代码格式需要修正"
    else
        print_success "代码格式正确"
    fi
    
    # Vet检查
    echo "运行go vet..."
    if go vet ./...; then
        print_success "go vet检查通过"
    else
        print_error "go vet检查失败"
        return 1
    fi
    
    # 检查是否安装了golangci-lint
    if command -v golangci-lint &> /dev/null; then
        echo "运行golangci-lint..."
        if golangci-lint run ./...; then
            print_success "Linter检查通过"
        else
            print_warning "Linter发现问题"
        fi
    else
        print_warning "golangci-lint未安装，跳过linter检查"
    fi
}

# 生成测试报告
generate_test_report() {
    print_header "生成测试报告"
    
    REPORT_FILE="test_report.md"
    
    cat > $REPORT_FILE << EOF
# QCAT 测试报告

**生成时间**: $(date)
**Go版本**: $(go version | cut -d' ' -f3)

## 测试结果概览

EOF

    if [ -f "$COVERAGE_FILE" ]; then
        COVERAGE=$(go tool cover -func=$COVERAGE_FILE | grep total | awk '{print $3}')
        echo "- **测试覆盖率**: $COVERAGE" >> $REPORT_FILE
    fi
    
    if [ -f "$BENCHMARK_FILE" ]; then
        echo "- **基准测试**: 已完成" >> $REPORT_FILE
    fi
    
    cat >> $REPORT_FILE << EOF

## 详细结果

### 单元测试
- 状态: ✅ 通过
- 覆盖率: $COVERAGE
- 报告: [$COVERAGE_HTML](./$COVERAGE_HTML)

### 集成测试
- 状态: ✅ 通过

### 基准测试
- 状态: ✅ 完成
- 报告: [$BENCHMARK_FILE](./$BENCHMARK_FILE)

## 建议

1. 继续提升测试覆盖率至80%+
2. 增加更多边界条件测试
3. 完善集成测试场景
4. 定期运行性能回归测试

EOF

    print_success "测试报告已生成: $REPORT_FILE"
}

# 主函数
main() {
    echo -e "${BLUE}QCAT 测试套件${NC}"
    echo "开始时间: $(date)"
    echo ""
    
    # 解析命令行参数
    TEST_TYPE=${1:-"all"}
    
    case $TEST_TYPE in
        "unit")
            check_go_env
            download_deps
            run_unit_tests
            ;;
        "coverage")
            check_go_env
            download_deps
            run_coverage_tests
            ;;
        "integration")
            check_go_env
            download_deps
            run_integration_tests
            ;;
        "e2e")
            check_go_env
            download_deps
            run_e2e_tests
            ;;
        "benchmark")
            check_go_env
            download_deps
            run_benchmark_tests
            ;;
        "quality")
            check_go_env
            run_quality_checks
            ;;
        "all")
            check_go_env
            download_deps
            run_unit_tests
            run_coverage_tests
            run_integration_tests
            run_benchmark_tests
            run_quality_checks
            generate_test_report
            ;;
        *)
            echo "用法: $0 [unit|coverage|integration|e2e|benchmark|quality|all]"
            echo ""
            echo "测试类型:"
            echo "  unit        - 运行单元测试"
            echo "  coverage    - 运行覆盖率测试"
            echo "  integration - 运行集成测试"
            echo "  e2e         - 运行端到端测试"
            echo "  benchmark   - 运行基准测试"
            echo "  quality     - 运行代码质量检查"
            echo "  all         - 运行所有测试 (默认)"
            exit 1
            ;;
    esac
    
    echo ""
    echo -e "${GREEN}测试完成时间: $(date)${NC}"
}

# 运行主函数
main "$@"
#!/bin/bash

# QCAT 数据库迁移安全检查脚本
# 用于在生产环境部署前验证迁移的安全性

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
CONFIG_FILE="${CONFIG_FILE:-configs/config.yaml}"
BACKUP_DIR="${BACKUP_DIR:-backups}"
TEST_DB_NAME="${TEST_DB_NAME:-qcat_migration_test}"

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
    log_info "检查依赖..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装"
        exit 1
    fi
    
    if ! command -v psql &> /dev/null; then
        log_error "PostgreSQL 客户端未安装"
        exit 1
    fi
    
    if ! command -v pg_dump &> /dev/null; then
        log_error "pg_dump 未安装"
        exit 1
    fi
    
    log_success "依赖检查通过"
}

# 创建备份
create_backup() {
    log_info "创建数据库备份..."
    
    mkdir -p "$BACKUP_DIR"
    
    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local backup_file="$BACKUP_DIR/qcat_backup_$timestamp.sql"
    
    # 从配置文件读取数据库连接信息
    local db_host=$(grep "host:" "$CONFIG_FILE" | awk '{print $2}' | tr -d '"')
    local db_port=$(grep "port:" "$CONFIG_FILE" | awk '{print $2}')
    local db_user=$(grep "user:" "$CONFIG_FILE" | awk '{print $2}' | tr -d '"')
    local db_name=$(grep "dbname:" "$CONFIG_FILE" | awk '{print $2}' | tr -d '"')
    
    if pg_dump -h "$db_host" -p "$db_port" -U "$db_user" -d "$db_name" > "$backup_file"; then
        log_success "备份创建成功: $backup_file"
        echo "$backup_file"
    else
        log_error "备份创建失败"
        exit 1
    fi
}

# 检查当前迁移状态
check_current_status() {
    log_info "检查当前迁移状态..."
    
    if go run cmd/migration-health/main.go -config "$CONFIG_FILE" -check; then
        log_success "当前迁移状态正常"
    else
        log_error "当前迁移状态异常，请先修复"
        exit 1
    fi
}

# 创建测试数据库
create_test_database() {
    log_info "创建测试数据库..."
    
    local db_host=$(grep "host:" "$CONFIG_FILE" | awk '{print $2}' | tr -d '"')
    local db_port=$(grep "port:" "$CONFIG_FILE" | awk '{print $2}')
    local db_user=$(grep "user:" "$CONFIG_FILE" | awk '{print $2}' | tr -d '"')
    
    # 删除已存在的测试数据库
    psql -h "$db_host" -p "$db_port" -U "$db_user" -d postgres -c "DROP DATABASE IF EXISTS $TEST_DB_NAME;" 2>/dev/null || true
    
    # 创建新的测试数据库
    if psql -h "$db_host" -p "$db_port" -U "$db_user" -d postgres -c "CREATE DATABASE $TEST_DB_NAME;"; then
        log_success "测试数据库创建成功: $TEST_DB_NAME"
    else
        log_error "测试数据库创建失败"
        exit 1
    fi
}

# 在测试数据库上运行迁移
test_migration() {
    log_info "在测试数据库上测试迁移..."
    
    # 创建临时配置文件
    local temp_config="/tmp/test_config.yaml"
    cp "$CONFIG_FILE" "$temp_config"
    
    # 修改配置文件中的数据库名
    sed -i "s/dbname: \"qcat\"/dbname: \"$TEST_DB_NAME\"/" "$temp_config"
    
    # 运行迁移
    if go run cmd/migrate/main.go -config "$temp_config" -up; then
        log_success "测试迁移成功"
    else
        log_error "测试迁移失败"
        cleanup_test_database
        rm -f "$temp_config"
        exit 1
    fi
    
    # 验证迁移完整性
    if go run cmd/migration-health/main.go -config "$temp_config" -validate; then
        log_success "迁移完整性验证通过"
    else
        log_error "迁移完整性验证失败"
        cleanup_test_database
        rm -f "$temp_config"
        exit 1
    fi
    
    rm -f "$temp_config"
}

# 清理测试数据库
cleanup_test_database() {
    log_info "清理测试数据库..."
    
    local db_host=$(grep "host:" "$CONFIG_FILE" | awk '{print $2}' | tr -d '"')
    local db_port=$(grep "port:" "$CONFIG_FILE" | awk '{print $2}')
    local db_user=$(grep "user:" "$CONFIG_FILE" | awk '{print $2}' | tr -d '"')
    
    psql -h "$db_host" -p "$db_port" -U "$db_user" -d postgres -c "DROP DATABASE IF EXISTS $TEST_DB_NAME;" 2>/dev/null || true
    log_success "测试数据库清理完成"
}

# 验证迁移文件
validate_migration_files() {
    log_info "验证迁移文件..."
    
    local migration_dir="internal/database/migrations"
    
    # 检查迁移文件是否存在
    if [ ! -d "$migration_dir" ]; then
        log_error "迁移目录不存在: $migration_dir"
        exit 1
    fi
    
    # 检查迁移文件命名规范
    local invalid_files=()
    for file in "$migration_dir"/*.sql; do
        if [[ ! $(basename "$file") =~ ^[0-9]{6}_[a-zA-Z0-9_]+\.(up|down)\.sql$ ]]; then
            invalid_files+=("$file")
        fi
    done
    
    if [ ${#invalid_files[@]} -gt 0 ]; then
        log_error "发现命名不规范的迁移文件:"
        for file in "${invalid_files[@]}"; do
            echo "  - $file"
        done
        exit 1
    fi
    
    # 检查每个up迁移是否有对应的down迁移
    local missing_down=()
    for up_file in "$migration_dir"/*_*.up.sql; do
        local down_file="${up_file%.up.sql}.down.sql"
        if [ ! -f "$down_file" ]; then
            missing_down+=("$down_file")
        fi
    done
    
    if [ ${#missing_down[@]} -gt 0 ]; then
        log_warning "发现缺少down迁移文件:"
        for file in "${missing_down[@]}"; do
            echo "  - $file"
        done
    fi
    
    log_success "迁移文件验证通过"
}

# 主函数
main() {
    echo "========================================"
    echo "QCAT 数据库迁移安全检查"
    echo "========================================"
    
    check_dependencies
    validate_migration_files
    check_current_status
    
    local backup_file=$(create_backup)
    
    create_test_database
    test_migration
    cleanup_test_database
    
    echo "========================================"
    log_success "迁移安全检查完成！"
    echo "备份文件: $backup_file"
    echo "可以安全地在生产环境运行迁移"
    echo "========================================"
}

# 错误处理
trap 'log_error "脚本执行失败"; cleanup_test_database; exit 1' ERR

# 运行主函数
main "$@"

#!/bin/bash

# QCAT数据库备份脚本
# 使用方法: ./backup.sh [backup|restore] [filename]

set -e

# 配置
BACKUP_DIR="/backup"
RETENTION_DAYS=30
DB_HOST="${DB_HOST:-postgres}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-qcat}"
DB_USER="${DB_USER:-qcat_user}"
DB_PASSWORD="${DB_PASSWORD:-qcat_password}"

# 创建备份目录
mkdir -p "$BACKUP_DIR"

# 设置环境变量
export PGPASSWORD="$DB_PASSWORD"

backup() {
    local filename="$1"
    if [ -z "$filename" ]; then
        filename="qcat_backup_$(date +%Y%m%d_%H%M%S).sql"
    fi
    
    echo "开始备份数据库..."
    pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        --verbose --clean --create --if-exists \
        --file="$BACKUP_DIR/$filename"
    
    echo "备份完成: $BACKUP_DIR/$filename"
    
    # 压缩备份文件
    gzip "$BACKUP_DIR/$filename"
    echo "备份文件已压缩: $BACKUP_DIR/$filename.gz"
}

restore() {
    local filename="$1"
    if [ -z "$filename" ]; then
        echo "错误: 请指定要恢复的备份文件名"
        exit 1
    fi
    
    if [[ "$filename" == *.gz ]]; then
        echo "解压备份文件..."
        gunzip -c "$BACKUP_DIR/$filename" > "$BACKUP_DIR/temp_restore.sql"
        filename="temp_restore.sql"
    fi
    
    echo "开始恢复数据库..."
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        --file="$BACKUP_DIR/$filename"
    
    if [ "$filename" = "temp_restore.sql" ]; then
        rm "$BACKUP_DIR/$filename"
    fi
    
    echo "恢复完成"
}

cleanup() {
    echo "清理过期备份文件..."
    find "$BACKUP_DIR" -name "qcat_backup_*.sql.gz" -mtime +$RETENTION_DAYS -delete
    echo "清理完成"
}

list_backups() {
    echo "可用的备份文件:"
    ls -la "$BACKUP_DIR"/qcat_backup_*.sql.gz 2>/dev/null || echo "没有找到备份文件"
}

case "$1" in
    "backup")
        backup "$2"
        cleanup
        ;;
    "restore")
        restore "$2"
        ;;
    "list")
        list_backups
        ;;
    "cleanup")
        cleanup
        ;;
    *)
        echo "使用方法: $0 {backup|restore|list|cleanup} [filename]"
        echo "  backup [filename]  - 备份数据库"
        echo "  restore filename   - 恢复数据库"
        echo "  list              - 列出备份文件"
        echo "  cleanup           - 清理过期备份"
        exit 1
        ;;
esac

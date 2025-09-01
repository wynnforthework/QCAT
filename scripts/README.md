# QCAT 启动脚本使用指南

## 概述

`start_local_improved.sh` 是 QCAT 项目的增强版本地开发环境启动脚本，提供了丰富的功能和灵活的配置选项。

## 主要特性

### ✨ 新增功能

- **命令行参数支持**: 灵活控制启动行为
- **多种运行模式**: 开发模式、生产模式、调试模式
- **选择性服务启动**: 可以只启动需要的服务
- **增强的错误处理**: 自动重试和详细的错误信息
- **智能端口管理**: 自动检测和清理端口占用
- **配置验证**: 自动验证配置文件完整性
- **健康检查**: 详细的服务状态监控
- **优雅关闭**: 安全地停止所有服务

### 🔧 改进的功能

- **依赖检查**: 更详细的系统依赖验证
- **编译优化**: 根据模式优化编译参数
- **日志增强**: 带时间戳的彩色日志输出
- **跨平台兼容**: 改进的 Windows/Linux/macOS 支持

## 使用方法

### 基本用法

```bash
# 启动所有服务（默认）
./scripts/start_local_improved.sh

# 显示帮助信息
./scripts/start_local_improved.sh --help

# 显示版本信息
./scripts/start_local_improved.sh --version
```

### 运行模式

```bash
# 开发模式（启用热重载、调试信息、竞态检测）
./scripts/start_local_improved.sh --dev

# 生产模式（优化编译、减小二进制大小）
./scripts/start_local_improved.sh --production

# 调试模式（详细输出）
./scripts/start_local_improved.sh --debug
```

### 选择性启动服务

```bash
# 只启动 API 服务
./scripts/start_local_improved.sh --services api

# 只启动前端服务
./scripts/start_local_improved.sh --services frontend

# 启动 API 和优化器
./scripts/start_local_improved.sh --services api,optimizer

# 启动所有服务
./scripts/start_local_improved.sh --services all
```

### 跳过特定步骤

```bash
# 跳过依赖安装（适用于依赖已安装的情况）
./scripts/start_local_improved.sh --skip-deps

# 跳过编译步骤（适用于二进制文件已存在的情况）
./scripts/start_local_improved.sh --skip-build

# 组合使用
./scripts/start_local_improved.sh --skip-deps --skip-build --services frontend
```

### 自定义重试参数

```bash
# 设置最大重试次数为 5 次
./scripts/start_local_improved.sh --max-retries 5

# 设置重试延迟为 10 秒
./scripts/start_local_improved.sh --retry-delay 10
```

## 配置说明

### 端口配置优先级

1. **环境变量** (最高优先级)
   - `QCAT_PORTS_QCAT_API`
   - `QCAT_PORTS_QCAT_OPTIMIZER`
   - `QCAT_PORTS_FRONTEND_DEV`

2. **配置文件** (`configs/config.yaml`)
   ```yaml
   ports:
     qcat_api: 8082
     qcat_optimizer: 8081
     frontend_dev: 3001
   ```

3. **默认值** (最低优先级)
   - API: 8082
   - 优化器: 8081
   - 前端: 3001 (开发模式) / 3000 (生产模式)

### 环境文件

脚本会自动查找并加载 `.env` 文件。如果不存在，会提示从 `deploy/env.example` 复制。

## 服务状态监控

脚本启动后会显示详细的服务状态信息：

- ✅ 服务正常运行
- ⚠️ 服务状态未知
- ❌ 服务未运行

### 状态检查项目

- **进程状态**: 检查进程是否存在
- **端口监听**: 检查端口是否被监听
- **健康检查**: 调用健康检查端点（如果可用）

## 故障排除

### 常见问题

1. **端口被占用**
   ```bash
   # 脚本会自动尝试清理端口，如果失败可以手动检查
   netstat -ano | findstr :8082  # Windows
   lsof -i :8082                 # Linux/macOS
   ```

2. **依赖缺失**
   ```bash
   # 脚本会检查并提示缺失的依赖
   # 请根据提示安装相应的依赖
   ```

3. **配置文件错误**
   ```bash
   # 使用调试模式查看详细信息
   ./scripts/start_local_improved.sh --debug
   ```

### 调试技巧

1. **启用调试模式**
   ```bash
   ./scripts/start_local_improved.sh --debug
   ```

2. **查看详细日志**
   - 脚本日志会显示时间戳和详细信息
   - 服务日志位于 `logs/` 目录

3. **分步启动**
   ```bash
   # 先只启动 API
   ./scripts/start_local_improved.sh --services api --debug
   
   # 确认 API 正常后再启动其他服务
   ./scripts/start_local_improved.sh --services frontend --skip-deps --skip-build
   ```

## 停止服务

按 `Ctrl+C` 停止所有服务。脚本会：

1. 优雅关闭所有服务 (SIGTERM)
2. 等待 3 秒
3. 强制关闭仍在运行的服务 (SIGKILL)
4. 清理占用的端口

## 与构建工具集成

### Makefile 集成 (Linux/macOS)

项目已集成到 `Makefile` 中，提供便捷的开发命令：

```bash
# 开发环境快捷命令
make dev              # 开发模式启动
make dev-api          # 只启动 API 服务
make dev-frontend     # 只启动前端服务
make dev-optimizer    # 只启动优化器服务
make prod             # 生产模式启动
make quick-start      # 快速启动（跳过依赖和编译）
make debug            # 调试模式启动

# 传统命令
make start-local      # 启动本地开发环境
make build            # 编译应用
make test             # 运行测试
make clean            # 清理构建文件
```

### Windows 批处理脚本

对于 Windows 用户，提供了批处理脚本：

```cmd
REM 使用 dev.bat
scripts\dev.bat dev              # 开发模式启动
scripts\dev.bat dev-frontend     # 只启动前端服务
scripts\dev.bat quick-start      # 快速启动
scripts\dev.bat help             # 显示帮助
```

### PowerShell 脚本

Windows PowerShell 用户可以使用：

```powershell
# 使用 dev.ps1
.\scripts\dev.ps1 dev              # 开发模式启动
.\scripts\dev.ps1 dev-frontend     # 只启动前端服务
.\scripts\dev.ps1 status           # 检查服务状态
.\scripts\dev.ps1 help             # 显示帮助
```

### 跨平台使用建议

- **Linux/macOS**: 推荐使用 `make` 命令
- **Windows (Git Bash)**: 可以使用 `make` 或直接调用脚本
- **Windows (CMD)**: 使用 `scripts\dev.bat`
- **Windows (PowerShell)**: 使用 `.\scripts\dev.ps1`

## 更新日志

### v2.1.0 (当前版本)

- ✨ 新增命令行参数支持
- ✨ 新增多种运行模式
- ✨ 新增选择性服务启动
- 🔧 改进错误处理和重试机制
- 🔧 增强配置验证和端口管理
- 🔧 优化服务状态监控
- 🔧 改进跨平台兼容性

### v2.0.0 (原版本)

- 基本的服务启动功能
- 简单的端口检查
- 基础的配置管理

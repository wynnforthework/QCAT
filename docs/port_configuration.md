# QCAT 端口配置指南

## 概述

QCAT系统采用统一的端口配置管理方案，所有服务端口都可以通过配置文件和环境变量进行统一管理。这确保了不同环境下的端口配置一致性，避免了硬编码端口带来的问题。

## 端口列表

### 默认端口配置

| 服务 | 端口 | 描述 | 配置键 |
|------|------|------|--------|
| QCAT API | 8082 | 主应用API服务 | `qcat_api` |
| QCAT优化器 | 8081 | 优化器服务 | `qcat_optimizer` |
| PostgreSQL | 5432 | 数据库服务 | `postgres` |
| Redis | 6379 | 缓存服务 | `redis` |
| Prometheus | 9090 | 监控服务 | `prometheus` |
| Grafana | 3000 | 监控面板 | `grafana` |
| AlertManager | 9093 | 告警服务 | `alertmanager` |
| Nginx HTTP | 80 | Web服务器HTTP | `nginx_http` |
| Nginx HTTPS | 443 | Web服务器HTTPS | `nginx_https` |
| 前端开发 | 3000 | 前端开发服务器 | `frontend_dev` |

## 配置方法

### 1. 主配置文件

在 `configs/config.yaml` 中配置：

```yaml
ports:
  qcat_api: 8082          # QCAT主应用API服务
  qcat_optimizer: 8081    # QCAT优化器服务
  postgres: 5432          # PostgreSQL数据库
  redis: 6379            # Redis缓存
  prometheus: 9090       # Prometheus监控
  grafana: 3000         # Grafana监控面板
  alertmanager: 9093    # AlertManager告警
  nginx_http: 80        # Nginx HTTP
  nginx_https: 443      # Nginx HTTPS
  frontend_dev: 3000    # 前端开发服务器
```

### 2. 环境变量

在 `.env` 文件或系统环境变量中配置：

```bash
# 端口配置
QCAT_PORTS_QCAT_API=8082
QCAT_PORTS_QCAT_OPTIMIZER=8081
QCAT_PORTS_POSTGRES=5432
QCAT_PORTS_REDIS=6379
QCAT_PORTS_PROMETHEUS=9090
QCAT_PORTS_GRAFANA=3000
QCAT_PORTS_ALERTMANAGER=9093
QCAT_PORTS_NGINX_HTTP=80
QCAT_PORTS_NGINX_HTTPS=443
QCAT_PORTS_FRONTEND_DEV=3000
```

### 3. 前端环境变量

在 `frontend/.env.local` 中配置：

```bash
NEXT_PUBLIC_API_URL=http://localhost:8082
```

## 配置优先级

端口配置按以下优先级生效（从高到低）：

1. **环境变量** - 最高优先级
2. **配置文件** - 中等优先级  
3. **默认值** - 最低优先级

## 使用场景

### 开发环境

1. 修改 `configs/config.yaml` 中的端口配置
2. 运行 `scripts/start_local.sh` 启动服务
3. 脚本会自动读取配置并使用正确的端口

### 测试环境

通过环境变量覆盖默认端口：

```bash
export QCAT_PORTS_QCAT_API=8092
export QCAT_PORTS_POSTGRES=5442
./scripts/start_local.sh
```

### 生产环境

使用Docker Compose部署：

```bash
# 设置环境变量
export QCAT_PORTS_QCAT_API=8082
export QCAT_PORTS_POSTGRES=5432

# 启动服务
docker-compose -f deploy/docker-compose.prod.yml up -d
```

## 故障排除

### 端口冲突

如果遇到端口冲突，可以：

1. 检查端口占用：`netstat -tulpn | grep :8082`
2. 修改配置文件中的端口
3. 或通过环境变量临时修改端口

### 前端无法连接后端

1. 检查前端的 `NEXT_PUBLIC_API_URL` 配置
2. 确保后端API端口与前端配置一致
3. 检查防火墙设置

### 配置验证

使用配置验证工具：

```bash
go run cmd/config/main.go -validate
```

## 最佳实践

1. **统一管理**: 所有端口配置都通过配置文件管理
2. **环境隔离**: 不同环境使用不同的端口配置
3. **文档同步**: 及时更新文档中的端口信息
4. **安全考虑**: 生产环境避免使用默认端口
5. **监控检查**: 定期检查端口配置的一致性

## 相关文件

- `configs/config.yaml` - 主配置文件
- `configs/config.yaml.example` - 配置模板
- `.env` - 环境变量文件
- `deploy/env.example` - 环境变量模板
- `frontend/.env.local` - 前端环境变量
- `frontend/.env.example` - 前端环境变量模板
- `deploy/docker-compose.prod.yml` - Docker部署配置
- `scripts/start_local.sh` - 本地启动脚本

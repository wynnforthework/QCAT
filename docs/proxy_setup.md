# Binance API 代理配置指南

由于部分地区访问 Binance API 需要代理，本项目已经内置了代理支持。以下是配置方法：

## 方法1：环境变量配置（推荐）

### Windows PowerShell
```powershell
# 设置代理环境变量
$env:HTTP_PROXY="http://127.0.0.1:1080"
$env:HTTPS_PROXY="http://127.0.0.1:1080"

# 如果代理需要认证
$env:HTTP_PROXY="http://username:password@127.0.0.1:1080"
$env:HTTPS_PROXY="http://username:password@127.0.0.1:1080"

# 运行程序
go run main.go
```

### Windows CMD
```cmd
set HTTP_PROXY=http://127.0.0.1:1080
set HTTPS_PROXY=http://127.0.0.1:1080
go run main.go
```

### Linux/macOS
```bash
export HTTP_PROXY=http://127.0.0.1:1080
export HTTPS_PROXY=http://127.0.0.1:1080
go run main.go
```

## 方法2：配置文件配置

在 `configs/config.yaml` 中添加代理配置：

```yaml
exchange:
  name: "binance"
  api_key: "${EXCHANGE_API_KEY}"
  api_secret: "${EXCHANGE_API_SECRET}"
  test_net: true
  base_url: "https://api.binance.com"
  futures_base_url: "https://fapi.binance.com"
  proxy_url: "http://127.0.0.1:1080"  # 你的代理地址
```

## 方法3：环境变量设置代理

也可以通过环境变量设置代理：

```bash
export EXCHANGE_PROXY_URL="http://127.0.0.1:1080"
```

## 支持的代理类型

- **HTTP代理**: `http://127.0.0.1:1080`
- **HTTPS代理**: `https://127.0.0.1:1080`
- **SOCKS5代理**: `socks5://127.0.0.1:1080`
- **带认证的代理**: `http://username:password@127.0.0.1:1080`

## 常见VPN软件的代理端口

- **Clash**: 通常是 `http://127.0.0.1:7890`
- **V2Ray**: 通常是 `http://127.0.0.1:1080` 或 `socks5://127.0.0.1:1080`
- **Shadowsocks**: 通常是 `socks5://127.0.0.1:1080`
- **Proxifier**: 根据配置而定

## 测试代理连接

使用项目中的测试工具验证代理配置：

```bash
# 设置代理环境变量
export HTTPS_PROXY="http://127.0.0.1:1080"

# 运行连接测试
go run cmd/test-connection/main.go
```

## 故障排除

1. **确认代理软件正在运行**
2. **检查代理端口是否正确**
3. **确认代理软件允许本地连接**
4. **查看程序日志中的代理使用信息**

程序会在日志中显示：
```
Using configured proxy: http://127.0.0.1:1080
```

如果看到这条日志，说明代理配置成功。

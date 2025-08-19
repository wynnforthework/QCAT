# 全面API接口状态检测脚本
param(
    [string]$BaseUrl = "http://localhost:8082",
    [int]$TimeoutSeconds = 10
)

Write-Host "🔍 QCAT API接口全面状态检测" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "基础URL: $BaseUrl" -ForegroundColor Yellow
Write-Host "超时时间: $TimeoutSeconds 秒" -ForegroundColor Yellow
Write-Host ""

# 登录获取token
Write-Host "🔐 正在获取认证token..." -ForegroundColor Yellow
try {
    $loginResponse = Invoke-WebRequest -Uri "$BaseUrl/api/v1/auth/login" -Method POST -Body (@{username="admin"; password="admin123"} | ConvertTo-Json) -ContentType "application/json" -TimeoutSec $TimeoutSeconds
    $loginData = ($loginResponse.Content | ConvertFrom-Json)
    $token = $loginData.data.access_token
    Write-Host "✅ 认证成功" -ForegroundColor Green
} catch {
    Write-Host "❌ 认证失败: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# 定义测试接口列表
$testEndpoints = @(
    # 公共接口
    @{ name = "健康检查"; method = "GET"; path = "/health"; auth = $false; category = "公共接口" },
    
    # 认证接口
    @{ name = "用户注册"; method = "POST"; path = "/api/v1/auth/register"; auth = $false; category = "认证"; 
       body = @{username="testuser$(Get-Random)"; password="testpass123"; email="test@example.com"} },
    
    # 仪表盘
    @{ name = "仪表盘数据"; method = "GET"; path = "/api/v1/dashboard"; auth = $true; category = "仪表盘" },
    
    # 市场数据
    @{ name = "市场数据"; method = "GET"; path = "/api/v1/market/data"; auth = $true; category = "市场数据" },
    
    # 交易活动
    @{ name = "交易活动"; method = "GET"; path = "/api/v1/trading/activity"; auth = $true; category = "交易" },
    
    # 系统指标
    @{ name = "系统指标"; method = "GET"; path = "/api/v1/metrics/system"; auth = $true; category = "系统指标" },
    @{ name = "性能指标"; method = "GET"; path = "/api/v1/metrics/performance"; auth = $true; category = "系统指标" },
    
    # 策略管理
    @{ name = "策略列表"; method = "GET"; path = "/api/v1/strategy/"; auth = $true; category = "策略管理" },
    @{ name = "创建策略"; method = "POST"; path = "/api/v1/strategy/"; auth = $true; category = "策略管理";
       body = @{name="Test Strategy $(Get-Random)"; type="momentum"; description="Test strategy"} },
    
    # 优化器
    @{ name = "优化任务列表"; method = "GET"; path = "/api/v1/optimizer/tasks"; auth = $true; category = "优化器" },
    @{ name = "运行优化"; method = "POST"; path = "/api/v1/optimizer/run"; auth = $true; category = "优化器";
       body = @{strategy_id="test-strategy"; method="grid"; objective="sharpe"} },
    
    # 投资组合
    @{ name = "投资组合概览"; method = "GET"; path = "/api/v1/portfolio/overview"; auth = $true; category = "投资组合" },
    @{ name = "投资组合配置"; method = "GET"; path = "/api/v1/portfolio/allocations"; auth = $true; category = "投资组合" },
    @{ name = "投资组合再平衡"; method = "POST"; path = "/api/v1/portfolio/rebalance"; auth = $true; category = "投资组合";
       body = @{mode="bandit"} },
    @{ name = "投资组合历史"; method = "GET"; path = "/api/v1/portfolio/history"; auth = $true; category = "投资组合" },
    
    # 风险管理
    @{ name = "风险概览"; method = "GET"; path = "/api/v1/risk/overview"; auth = $true; category = "风险管理" },
    @{ name = "风险限额"; method = "GET"; path = "/api/v1/risk/limits"; auth = $true; category = "风险管理" },
    @{ name = "熔断器状态"; method = "GET"; path = "/api/v1/risk/circuit-breakers"; auth = $true; category = "风险管理" },
    @{ name = "风险违规"; method = "GET"; path = "/api/v1/risk/violations"; auth = $true; category = "风险管理" },
    
    # 热门列表
    @{ name = "热门符号"; method = "GET"; path = "/api/v1/hotlist/symbols"; auth = $true; category = "热门列表" },
    @{ name = "白名单"; method = "GET"; path = "/api/v1/hotlist/whitelist"; auth = $true; category = "热门列表" },
    
    # 健康检查
    @{ name = "健康状态"; method = "GET"; path = "/api/v1/health/status"; auth = $true; category = "健康检查" },
    @{ name = "所有健康检查"; method = "GET"; path = "/api/v1/health/checks"; auth = $true; category = "健康检查" },
    
    # 审计
    @{ name = "审计日志"; method = "GET"; path = "/api/v1/audit/logs"; auth = $true; category = "审计" },
    @{ name = "决策链"; method = "GET"; path = "/api/v1/audit/decisions"; auth = $true; category = "审计" },
    @{ name = "审计性能"; method = "GET"; path = "/api/v1/audit/performance"; auth = $true; category = "审计" },
    
    # 缓存管理
    @{ name = "缓存状态"; method = "GET"; path = "/api/v1/cache/status"; auth = $true; category = "缓存管理" },
    @{ name = "缓存健康"; method = "GET"; path = "/api/v1/cache/health"; auth = $true; category = "缓存管理" },
    @{ name = "缓存指标"; method = "GET"; path = "/api/v1/cache/metrics"; auth = $true; category = "缓存管理" },
    @{ name = "缓存配置"; method = "GET"; path = "/api/v1/cache/config"; auth = $true; category = "缓存管理" },
    
    # 安全管理
    @{ name = "API密钥列表"; method = "GET"; path = "/api/v1/security/keys/"; auth = $true; category = "安全管理" },
    @{ name = "安全审计日志"; method = "GET"; path = "/api/v1/security/audit/logs"; auth = $true; category = "安全管理" },
    @{ name = "完整性验证"; method = "GET"; path = "/api/v1/security/audit/integrity"; auth = $true; category = "安全管理" },
    
    # 编排器
    @{ name = "编排器状态"; method = "GET"; path = "/api/v1/orchestrator/status"; auth = $true; category = "编排器" },
    @{ name = "服务列表"; method = "GET"; path = "/api/v1/orchestrator/services"; auth = $true; category = "编排器" },
    @{ name = "编排器健康"; method = "GET"; path = "/api/v1/orchestrator/health"; auth = $true; category = "编排器" }
)

# 测试结果统计
$results = @()
$successCount = 0
$failCount = 0
$categoryStats = @{}

Write-Host "🧪 开始测试 $($testEndpoints.Count) 个API接口..." -ForegroundColor Yellow
Write-Host ""

foreach ($endpoint in $testEndpoints) {
    $headers = @{}
    if ($endpoint.auth) {
        $headers["Authorization"] = "Bearer $token"
    }
    
    $body = $null
    if ($endpoint.body) {
        $body = $endpoint.body | ConvertTo-Json
        $headers["Content-Type"] = "application/json"
    }
    
    try {
        $startTime = Get-Date
        if ($endpoint.method -eq "GET") {
            $response = Invoke-WebRequest -Uri "$BaseUrl$($endpoint.path)" -Method $endpoint.method -Headers $headers -TimeoutSec $TimeoutSeconds
        } else {
            $response = Invoke-WebRequest -Uri "$BaseUrl$($endpoint.path)" -Method $endpoint.method -Headers $headers -Body $body -TimeoutSec $TimeoutSeconds
        }
        $endTime = Get-Date
        $responseTime = ($endTime - $startTime).TotalMilliseconds
        
        $status = if ($response.StatusCode -eq 200) { "SUCCESS" } else { "STATUS: $($response.StatusCode)" }
        $successCount++
        
        Write-Host "✅ $($endpoint.name): $status (${responseTime}ms)" -ForegroundColor Green
        
        $results += @{
            name = $endpoint.name
            category = $endpoint.category
            path = $endpoint.path
            method = $endpoint.method
            status = "success"
            statusCode = $response.StatusCode
            responseTime = $responseTime
        }
    } catch {
        $failCount++
        $errorMsg = $_.Exception.Message
        
        # 解析HTTP状态码
        $statusCode = 0
        if ($_.Exception.Response) {
            $statusCode = $_.Exception.Response.StatusCode.value__
        }
        
        Write-Host "❌ $($endpoint.name): FAILED" -ForegroundColor Red
        if ($statusCode -gt 0) {
            Write-Host "   HTTP $statusCode" -ForegroundColor Red
        }
        Write-Host "   $errorMsg" -ForegroundColor Red
        
        $results += @{
            name = $endpoint.name
            category = $endpoint.category
            path = $endpoint.path
            method = $endpoint.method
            status = "failed"
            statusCode = $statusCode
            error = $errorMsg
        }
    }
    
    # 统计分类结果
    if (-not $categoryStats.ContainsKey($endpoint.category)) {
        $categoryStats[$endpoint.category] = @{success = 0; failed = 0}
    }
    if ($results[-1].status -eq "success") {
        $categoryStats[$endpoint.category].success++
    } else {
        $categoryStats[$endpoint.category].failed++
    }
}

Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "📊 测试结果汇总" -ForegroundColor Cyan
Write-Host "总接口数: $($testEndpoints.Count)" -ForegroundColor White
Write-Host "成功: $successCount ($(($successCount / $testEndpoints.Count * 100).ToString('F1'))%)" -ForegroundColor Green
Write-Host "失败: $failCount ($(($failCount / $testEndpoints.Count * 100).ToString('F1'))%)" -ForegroundColor Red
Write-Host ""

Write-Host "📋 按分类统计:" -ForegroundColor Yellow
foreach ($category in $categoryStats.Keys | Sort-Object) {
    $stats = $categoryStats[$category]
    $total = $stats.success + $stats.failed
    $successRate = if ($total -gt 0) { ($stats.success / $total * 100).ToString('F1') } else { "0.0" }
    Write-Host "  $category`: $($stats.success)/$total 成功 ($successRate%)" -ForegroundColor White
}

Write-Host ""
if ($failCount -gt 0) {
    Write-Host "❌ 失败的接口:" -ForegroundColor Red
    foreach ($result in $results | Where-Object { $_.status -eq "failed" }) {
        Write-Host "  - $($result.name) ($($result.method) $($result.path))" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "✅ 测试完成!" -ForegroundColor Green

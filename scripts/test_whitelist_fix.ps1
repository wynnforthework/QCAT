# 测试限流白名单修复脚本
$baseUrl = "http://localhost:8082"

# 之前受限流影响的API接口
$endpoints = @(
    "/api/v1/health/status",
    "/api/v1/health/checks",
    "/api/v1/cache/status",
    "/api/v1/orchestrator/status",
    "/api/v1/orchestrator/health"
)

Write-Host "测试限流白名单修复功能" -ForegroundColor Cyan
Write-Host ""

$successCount = 0
$totalCount = $endpoints.Count

foreach ($endpoint in $endpoints) {
    Write-Host "测试: $endpoint" -ForegroundColor White

    try {
        $response = Invoke-WebRequest -Uri "$baseUrl$endpoint" -Method GET -TimeoutSec 5

        if ($response.StatusCode -eq 200) {
            Write-Host "  成功 (200)" -ForegroundColor Green
            $successCount++
        } elseif ($response.StatusCode -eq 401) {
            Write-Host "  需要认证 (401) - 正常" -ForegroundColor Yellow
            $successCount++
        } else {
            Write-Host "  状态码: $($response.StatusCode)" -ForegroundColor Yellow
        }
    } catch {
        $errorMsg = $_.Exception.Message
        if ($errorMsg -like "*429*") {
            Write-Host "  仍然被限流 (429)" -ForegroundColor Red
        } elseif ($errorMsg -like "*401*") {
            Write-Host "  需要认证 (401) - 正常" -ForegroundColor Yellow
            $successCount++
        } else {
            Write-Host "  错误: $errorMsg" -ForegroundColor Yellow
        }
    }

    Start-Sleep -Milliseconds 200
}

Write-Host ""
Write-Host "测试结果: $successCount/$totalCount 接口可访问" -ForegroundColor Cyan

if ($successCount -eq $totalCount) {
    Write-Host "限流白名单修复成功！" -ForegroundColor Green
} else {
    Write-Host "部分接口仍有问题" -ForegroundColor Yellow
}

# 测试认证接口的400错误
$baseUrl = "http://localhost:8082"

Write-Host "Testing authentication endpoints with 400 errors" -ForegroundColor Cyan
Write-Host ""

# 测试登录接口
Write-Host "1. Testing login with invalid data:" -ForegroundColor White
try {
    $response = Invoke-WebRequest -Uri "$baseUrl/api/v1/auth/login" -Method POST -Body '{"test":"data"}' -ContentType "application/json"
    Write-Host "  Unexpected success: $($response.StatusCode)" -ForegroundColor Green
} catch {
    $errorResponse = $_.Exception.Response
    if ($errorResponse) {
        $statusCode = $errorResponse.StatusCode.value__
        Write-Host "  Status: $statusCode" -ForegroundColor Yellow
        
        # 读取错误响应内容
        $stream = $errorResponse.GetResponseStream()
        $reader = New-Object System.IO.StreamReader($stream)
        $errorContent = $reader.ReadToEnd()
        Write-Host "  Error: $errorContent" -ForegroundColor Red
    }
}

Write-Host ""

# 测试注册接口
Write-Host "2. Testing register with invalid data:" -ForegroundColor White
try {
    $response = Invoke-WebRequest -Uri "$baseUrl/api/v1/auth/register" -Method POST -Body '{"test":"data"}' -ContentType "application/json"
    Write-Host "  Unexpected success: $($response.StatusCode)" -ForegroundColor Green
} catch {
    $errorResponse = $_.Exception.Response
    if ($errorResponse) {
        $statusCode = $errorResponse.StatusCode.value__
        Write-Host "  Status: $statusCode" -ForegroundColor Yellow
        
        # 读取错误响应内容
        $stream = $errorResponse.GetResponseStream()
        $reader = New-Object System.IO.StreamReader($stream)
        $errorContent = $reader.ReadToEnd()
        Write-Host "  Error: $errorContent" -ForegroundColor Red
    }
}

Write-Host ""

# 测试刷新令牌接口
Write-Host "3. Testing refresh with invalid data:" -ForegroundColor White
try {
    $response = Invoke-WebRequest -Uri "$baseUrl/api/v1/auth/refresh" -Method POST -Body '{"test":"data"}' -ContentType "application/json"
    Write-Host "  Unexpected success: $($response.StatusCode)" -ForegroundColor Green
} catch {
    $errorResponse = $_.Exception.Response
    if ($errorResponse) {
        $statusCode = $errorResponse.StatusCode.value__
        Write-Host "  Status: $statusCode" -ForegroundColor Yellow
        
        # 读取错误响应内容
        $stream = $errorResponse.GetResponseStream()
        $reader = New-Object System.IO.StreamReader($stream)
        $errorContent = $reader.ReadToEnd()
        Write-Host "  Error: $errorContent" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "Testing completed" -ForegroundColor Cyan

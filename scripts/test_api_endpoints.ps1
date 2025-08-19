# API Endpoint Test Script
$baseUrl = "http://localhost:8082"

# Login to get token
Write-Host "Logging in..." -ForegroundColor Yellow
try {
    $loginResponse = Invoke-WebRequest -Uri "$baseUrl/api/v1/auth/login" -Method POST -Body (@{username="admin"; password="admin123"} | ConvertTo-Json) -ContentType "application/json"
    $loginData = ($loginResponse.Content | ConvertFrom-Json)
    $token = $loginData.data.access_token
    Write-Host "Login successful" -ForegroundColor Green
} catch {
    Write-Host "Login failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test endpoints
$endpoints = @(
    @{ name = "Health Check"; method = "GET"; path = "/health"; auth = $false },
    @{ name = "Dashboard"; method = "GET"; path = "/api/v1/dashboard"; auth = $true },
    @{ name = "Strategy List"; method = "GET"; path = "/api/v1/strategy/"; auth = $true },
    @{ name = "Market Data"; method = "GET"; path = "/api/v1/market/data"; auth = $true },
    @{ name = "Trading Activity"; method = "GET"; path = "/api/v1/trading/activity"; auth = $true },
    @{ name = "Portfolio Overview"; method = "GET"; path = "/api/v1/portfolio/overview"; auth = $true },
    @{ name = "Portfolio Allocations"; method = "GET"; path = "/api/v1/portfolio/allocations"; auth = $true },
    @{ name = "Risk Overview"; method = "GET"; path = "/api/v1/risk/overview"; auth = $true },
    @{ name = "Risk Limits"; method = "GET"; path = "/api/v1/risk/limits"; auth = $true },
    @{ name = "Hot Symbols"; method = "GET"; path = "/api/v1/hotlist/symbols"; auth = $true },
    @{ name = "System Metrics"; method = "GET"; path = "/api/v1/metrics/system"; auth = $true },
    @{ name = "Performance Metrics"; method = "GET"; path = "/api/v1/metrics/performance"; auth = $true },
    @{ name = "Optimizer Tasks"; method = "GET"; path = "/api/v1/optimizer/tasks"; auth = $true },
    @{ name = "Health Status"; method = "GET"; path = "/api/v1/health/status"; auth = $true },
    @{ name = "Health Checks"; method = "GET"; path = "/api/v1/health/checks"; auth = $true },
    @{ name = "Audit Logs"; method = "GET"; path = "/api/v1/audit/logs"; auth = $true },
    @{ name = "Cache Status"; method = "GET"; path = "/api/v1/cache/status"; auth = $true },
    @{ name = "Security Keys"; method = "GET"; path = "/api/v1/security/keys/"; auth = $true },
    @{ name = "Orchestrator Status"; method = "GET"; path = "/api/v1/orchestrator/status"; auth = $true }
)

$successCount = 0
$failCount = 0
$results = @()

Write-Host "`nTesting API endpoints..." -ForegroundColor Yellow
Write-Host "=" * 60

foreach ($endpoint in $endpoints) {
    $headers = @{}
    if ($endpoint.auth) {
        $headers["Authorization"] = "Bearer $token"
    }

    try {
        $response = Invoke-WebRequest -Uri "$baseUrl$($endpoint.path)" -Method $endpoint.method -Headers $headers -TimeoutSec 10
        $status = if ($response.StatusCode -eq 200) { "SUCCESS" } else { "STATUS: $($response.StatusCode)" }
        $successCount++

        Write-Host "$($endpoint.name): $status" -ForegroundColor Green
        $results += @{
            name = $endpoint.name
            path = $endpoint.path
            status = "success"
            statusCode = $response.StatusCode
        }
    } catch {
        $failCount++
        $errorMsg = $_.Exception.Message
        if ($_.Exception.Response) {
            $statusCode = $_.Exception.Response.StatusCode.value__
            Write-Host "$($endpoint.name): FAILED (HTTP $statusCode)" -ForegroundColor Red
        } else {
            Write-Host "$($endpoint.name): FAILED ($errorMsg)" -ForegroundColor Red
        }

        $results += @{
            name = $endpoint.name
            path = $endpoint.path
            status = "failed"
            error = $errorMsg
        }
    }
}

Write-Host "`n" + "=" * 60
Write-Host "Test completed!" -ForegroundColor Yellow
Write-Host "Success: $successCount" -ForegroundColor Green
Write-Host "Failed: $failCount" -ForegroundColor Red
Write-Host "Total: $($endpoints.Count)" -ForegroundColor Blue

# Output failed endpoints
if ($failCount -gt 0) {
    Write-Host "`nFailed endpoints:" -ForegroundColor Red
    foreach ($result in $results) {
        if ($result.status -eq "failed") {
            Write-Host "  - $($result.name) ($($result.path))" -ForegroundColor Red
        }
    }
}

Write-Host "`nRecommendations:" -ForegroundColor Yellow
Write-Host "1. Check if failed endpoints are implemented"
Write-Host "2. Mark expected failures in frontend test page"
Write-Host "3. Ensure necessary test data exists in database"

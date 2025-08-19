# Comprehensive API Status Test
param(
    [string]$BaseUrl = "http://localhost:8082",
    [int]$TimeoutSeconds = 10
)

Write-Host "QCAT API Comprehensive Status Test" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "Base URL: $BaseUrl" -ForegroundColor Yellow
Write-Host "Timeout: $TimeoutSeconds seconds" -ForegroundColor Yellow
Write-Host ""

# Login to get token
Write-Host "Getting authentication token..." -ForegroundColor Yellow
try {
    $loginResponse = Invoke-WebRequest -Uri "$BaseUrl/api/v1/auth/login" -Method POST -Body (@{username="admin"; password="admin123"} | ConvertTo-Json) -ContentType "application/json" -TimeoutSec $TimeoutSeconds
    $loginData = ($loginResponse.Content | ConvertFrom-Json)
    $token = $loginData.data.access_token
    Write-Host "Authentication successful" -ForegroundColor Green
} catch {
    Write-Host "Authentication failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Define test endpoints
$testEndpoints = @(
    # Public endpoints
    @{ name = "Health Check"; method = "GET"; path = "/health"; auth = $false; category = "Public" },
    
    # Authentication
    @{ name = "User Register"; method = "POST"; path = "/api/v1/auth/register"; auth = $false; category = "Auth"; 
       body = @{username="testuser$(Get-Random)"; password="testpass123"; email="test@example.com"} },
    
    # Dashboard
    @{ name = "Dashboard Data"; method = "GET"; path = "/api/v1/dashboard"; auth = $true; category = "Dashboard" },
    
    # Market Data
    @{ name = "Market Data"; method = "GET"; path = "/api/v1/market/data"; auth = $true; category = "Market" },
    
    # Trading
    @{ name = "Trading Activity"; method = "GET"; path = "/api/v1/trading/activity"; auth = $true; category = "Trading" },
    
    # System Metrics
    @{ name = "System Metrics"; method = "GET"; path = "/api/v1/metrics/system"; auth = $true; category = "Metrics" },
    @{ name = "Performance Metrics"; method = "GET"; path = "/api/v1/metrics/performance"; auth = $true; category = "Metrics" },
    
    # Strategy Management
    @{ name = "Strategy List"; method = "GET"; path = "/api/v1/strategy/"; auth = $true; category = "Strategy" },
    @{ name = "Create Strategy"; method = "POST"; path = "/api/v1/strategy/"; auth = $true; category = "Strategy";
       body = @{name="Test Strategy $(Get-Random)"; type="momentum"; description="Test strategy"} },
    
    # Optimizer
    @{ name = "Optimizer Tasks"; method = "GET"; path = "/api/v1/optimizer/tasks"; auth = $true; category = "Optimizer" },
    @{ name = "Run Optimization"; method = "POST"; path = "/api/v1/optimizer/run"; auth = $true; category = "Optimizer";
       body = @{strategy_id="test-strategy"; method="grid"; objective="sharpe"} },
    
    # Portfolio
    @{ name = "Portfolio Overview"; method = "GET"; path = "/api/v1/portfolio/overview"; auth = $true; category = "Portfolio" },
    @{ name = "Portfolio Allocations"; method = "GET"; path = "/api/v1/portfolio/allocations"; auth = $true; category = "Portfolio" },
    @{ name = "Portfolio Rebalance"; method = "POST"; path = "/api/v1/portfolio/rebalance"; auth = $true; category = "Portfolio";
       body = @{mode="bandit"} },
    @{ name = "Portfolio History"; method = "GET"; path = "/api/v1/portfolio/history"; auth = $true; category = "Portfolio" },
    
    # Risk Management
    @{ name = "Risk Overview"; method = "GET"; path = "/api/v1/risk/overview"; auth = $true; category = "Risk" },
    @{ name = "Risk Limits"; method = "GET"; path = "/api/v1/risk/limits"; auth = $true; category = "Risk" },
    @{ name = "Circuit Breakers"; method = "GET"; path = "/api/v1/risk/circuit-breakers"; auth = $true; category = "Risk" },
    @{ name = "Risk Violations"; method = "GET"; path = "/api/v1/risk/violations"; auth = $true; category = "Risk" },
    
    # Hotlist
    @{ name = "Hot Symbols"; method = "GET"; path = "/api/v1/hotlist/symbols"; auth = $true; category = "Hotlist" },
    @{ name = "Whitelist"; method = "GET"; path = "/api/v1/hotlist/whitelist"; auth = $true; category = "Hotlist" },
    
    # Health Checks
    @{ name = "Health Status"; method = "GET"; path = "/api/v1/health/status"; auth = $true; category = "Health" },
    @{ name = "All Health Checks"; method = "GET"; path = "/api/v1/health/checks"; auth = $true; category = "Health" },
    
    # Audit
    @{ name = "Audit Logs"; method = "GET"; path = "/api/v1/audit/logs"; auth = $true; category = "Audit" },
    @{ name = "Decision Chains"; method = "GET"; path = "/api/v1/audit/decisions"; auth = $true; category = "Audit" },
    @{ name = "Audit Performance"; method = "GET"; path = "/api/v1/audit/performance"; auth = $true; category = "Audit" },
    
    # Cache Management
    @{ name = "Cache Status"; method = "GET"; path = "/api/v1/cache/status"; auth = $true; category = "Cache" },
    @{ name = "Cache Health"; method = "GET"; path = "/api/v1/cache/health"; auth = $true; category = "Cache" },
    @{ name = "Cache Metrics"; method = "GET"; path = "/api/v1/cache/metrics"; auth = $true; category = "Cache" },
    @{ name = "Cache Config"; method = "GET"; path = "/api/v1/cache/config"; auth = $true; category = "Cache" },
    
    # Security Management
    @{ name = "API Keys List"; method = "GET"; path = "/api/v1/security/keys/"; auth = $true; category = "Security" },
    @{ name = "Security Audit Logs"; method = "GET"; path = "/api/v1/security/audit/logs"; auth = $true; category = "Security" },
    @{ name = "Integrity Check"; method = "GET"; path = "/api/v1/security/audit/integrity"; auth = $true; category = "Security" },
    
    # Orchestrator
    @{ name = "Orchestrator Status"; method = "GET"; path = "/api/v1/orchestrator/status"; auth = $true; category = "Orchestrator" },
    @{ name = "Services List"; method = "GET"; path = "/api/v1/orchestrator/services"; auth = $true; category = "Orchestrator" },
    @{ name = "Orchestrator Health"; method = "GET"; path = "/api/v1/orchestrator/health"; auth = $true; category = "Orchestrator" }
)

# Test results
$results = @()
$successCount = 0
$failCount = 0
$categoryStats = @{}

Write-Host "Testing $($testEndpoints.Count) API endpoints..." -ForegroundColor Yellow
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
        
        Write-Host "SUCCESS $($endpoint.name): $status (${responseTime}ms)" -ForegroundColor Green
        
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
        
        # Parse HTTP status code
        $statusCode = 0
        if ($_.Exception.Response) {
            $statusCode = $_.Exception.Response.StatusCode.value__
        }
        
        Write-Host "FAILED $($endpoint.name): FAILED" -ForegroundColor Red
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
    
    # Category statistics
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
Write-Host "Test Results Summary" -ForegroundColor Cyan
Write-Host "Total endpoints: $($testEndpoints.Count)" -ForegroundColor White
Write-Host "Success: $successCount ($(($successCount / $testEndpoints.Count * 100).ToString('F1'))%)" -ForegroundColor Green
Write-Host "Failed: $failCount ($(($failCount / $testEndpoints.Count * 100).ToString('F1'))%)" -ForegroundColor Red
Write-Host ""

Write-Host "By Category:" -ForegroundColor Yellow
foreach ($category in $categoryStats.Keys | Sort-Object) {
    $stats = $categoryStats[$category]
    $total = $stats.success + $stats.failed
    $successRate = if ($total -gt 0) { ($stats.success / $total * 100).ToString('F1') } else { "0.0" }
    Write-Host "  $category`: $($stats.success)/$total success ($successRate%)" -ForegroundColor White
}

Write-Host ""
if ($failCount -gt 0) {
    Write-Host "Failed endpoints:" -ForegroundColor Red
    foreach ($result in $results | Where-Object { $_.status -eq "failed" }) {
        Write-Host "  - $($result.name) ($($result.method) $($result.path))" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "Test completed!" -ForegroundColor Green

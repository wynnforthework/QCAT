# Test registration API
$baseUrl = "http://localhost:8082"

# Test data
$testUsers = @(
    @{
        username = "testuser1"
        password = "test123"
        email = "test1@example.com"
    },
    @{
        username = "admin"
        password = "admin123"
        email = "admin@qcat.local"
    },
    @{
        username = "testuser2"
        password = "test456"
        email = "test2@example.com"
    }
)

Write-Host "Testing user registration API" -ForegroundColor Cyan
Write-Host ""

foreach ($user in $testUsers) {
    Write-Host "Testing registration for user: $($user.username)" -ForegroundColor White

    $body = @{
        username = $user.username
        password = $user.password
        email = $user.email
    } | ConvertTo-Json

    try {
        $response = Invoke-WebRequest -Uri "$baseUrl/api/v1/auth/register" -Method POST -Body $body -ContentType "application/json"

        if ($response.StatusCode -eq 201) {
            Write-Host "  Registration successful (201)" -ForegroundColor Green
        } else {
            Write-Host "  Status code: $($response.StatusCode)" -ForegroundColor Yellow
        }
    } catch {
        $errorResponse = $_.Exception.Response
        if ($errorResponse) {
            $statusCode = $errorResponse.StatusCode.value__
            if ($statusCode -eq 409) {
                Write-Host "  User already exists (409) - Expected" -ForegroundColor Yellow
            } elseif ($statusCode -eq 400) {
                Write-Host "  Bad request (400)" -ForegroundColor Red
            } else {
                Write-Host "  Error status code: $statusCode" -ForegroundColor Red
            }
        } else {
            Write-Host "  Network error: $($_.Exception.Message)" -ForegroundColor Red
        }
    }

    Start-Sleep -Milliseconds 500
}

Write-Host ""
Write-Host "Testing completed" -ForegroundColor Cyan

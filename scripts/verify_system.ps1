param(
    [string]$BaseUrl = "http://localhost:8082",
    [string]$Token
)

$uri = "$BaseUrl/api/v1/operations/matrix?refresh=true"
$headers = @{}
if ($Token) {
    $headers["Authorization"] = "Bearer $Token"
} else {
    Write-Warning "JWT token not supplied; request is likely to be rejected."
}

try {
    $response = Invoke-RestMethod -Method Get -Uri $uri -Headers $headers -ErrorAction Stop
} catch {
    Write-Error ("Failed to query operations matrix: {0}" -f $_)
    exit 1
}

if (-not $response.success) {
    $errorMessage = if ($response.error) { $response.error } else { 'unknown server error' }
    Write-Error ("Request failed: {0}" -f $errorMessage)
    exit 1
}

$data = $response.data
if (-not $data) {
    Write-Output "No data returned"
    exit 0
}

Write-Host ("Mode          : {0}" -f $data.mode)
Write-Host ("Generated At  : {0}" -f $data.generated_at)

if ($data.domains) {
    foreach ($domainName in $data.domains.Keys) {
        $domain = $data.domains[$domainName]
        Write-Host ""
        Write-Host ("[{0}] status={1}" -f $domainName, $domain.status)
        if ($domain.checks) {
            foreach ($check in $domain.checks) {
                Write-Host (" - {0}: {1}" -f $check.name, $check.summary)
            }
        }
    }
} else {
    Write-Host "No domain information available."
}
